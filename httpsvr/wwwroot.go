package httpsvr

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"mime"
	"net"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// WWWROOT 兜底：URL 在路由表与静态前缀都未命中时，ServeHTTP 链尾尝试从
// SetWWWRoot 设置的目录中按请求路径直接返回文件；仍未命中才走 404。

// SetWWWRoot 设置全站兜底静态目录。请求路径在路由表找不到匹配、且链尾尚未写响应时，
// 尝试从 dir 目录中按请求路径定位文件并直接返回（目录请求自动尝试其下 index.html）。
// 传空字符串关闭兜底。随时调用即刻生效，无需重启。
//
//	s.SetWWWRoot("/var/wwwroot")  // 外部配置（如 easyconf）读出后传入即可
func (s *EasyServer) SetWWWRoot(dir string) {
	s.fieldLock.Lock()
	defer s.fieldLock.Unlock()
	s.wwwroot = dir
}

// SetNotFoundHandler 自定义最终 404 响应。仅当路由、静态资源、WWWROOT 兜底全部
// 未命中时调用。不设置则使用默认的 ResponseNotFound（JSON 响应体）。
// 随时调用即刻生效。
//
//	s.SetNotFoundHandler(func(w http.ResponseWriter, r *http.Request) {
//	    http.Error(w, "404 page not found", http.StatusNotFound)
//	})
func (s *EasyServer) SetNotFoundHandler(fn func(w http.ResponseWriter, r *http.Request)) {
	s.fieldLock.Lock()
	defer s.fieldLock.Unlock()
	s.notFoundHandler = fn
}

// getWWWRoot 读取当前兜底目录。
func (s *EasyServer) getWWWRoot() string {
	s.fieldLock.RLock()
	defer s.fieldLock.RUnlock()
	return s.wwwroot
}

// notFound 调用自定义 404 处理器；未设置则使用默认 ResponseNotFound。
func (s *EasyServer) notFound(w http.ResponseWriter, r *http.Request) {
	s.fieldLock.RLock()
	fn := s.notFoundHandler
	s.fieldLock.RUnlock()
	if fn != nil {
		fn(w, r)
		return
	}
	ResponseNotFound(w, r)
}

// tryServeWWWRoot 尝试按请求路径在兜底目录中定位并返回文件。
// 返回 true 表示已写响应（无论成败）；false 表示文件不存在，需继续走 404。
func (s *EasyServer) tryServeWWWRoot(w http.ResponseWriter, r *http.Request) bool {
	// 静态文件只应答 GET/HEAD，其他方法视为未命中走 404
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		return false
	}
	dir := s.getWWWRoot()
	if dir == "" {
		return false
	}
	// 安全拼接：path.Clean 按POSIX语义折叠掉 .. 等相对成分，防止目录遍历攻击。
	// 注意两点：
	//  1. 不能用 filepath.Clean：Windows 下 URL 以/开头拼接成 // 时会被当作
	//     UNC 前缀保留，导致 .. 不被折叠而逃逸目录。
	//  2. 须先把 \ 归一化为 /：path.Clean 不把 \ 当分隔符，Windows 下请求
	//     /..\..\x 中的 .. 不会被折叠，随后 filepath.Join 又按 \ 折叠，可逃逸目录。
	fpath := filepath.Join(dir, filepath.FromSlash(path.Clean("/"+strings.ReplaceAll(r.URL.Path, `\`, "/"))))
	fileInfo, err := os.Stat(fpath)
	if err != nil {
		if os.IsNotExist(err) {
			return false
		}
		errWrite(w, err.Error(), http.StatusInternalServerError)
		return true
	}
	if fileInfo.IsDir() {
		// 目录请求尝试其下 index.html。stat 失败的错误分类与首层保持一致：
		// 不存在视为未命中，其他错误（如权限）写 500。
		fpath = filepath.Join(fpath, "index.html")
		fileInfo, err = os.Stat(fpath)
		if err != nil {
			if os.IsNotExist(err) {
				return false
			}
			errWrite(w, err.Error(), http.StatusInternalServerError)
			return true
		}
	}
	serveFileObj(w, r, fpath, fileInfo)
	return true
}

// serveFileObj 打开文件并按扩展名设置 Content-Type 后输出文件内容。
// 打开/读取失败时写 500。
func serveFileObj(w http.ResponseWriter, r *http.Request, fpath string, fileInfo fs.FileInfo) {
	var file fs.File
	var err error
	file, err = os.Open(fpath)
	if err != nil {
		errWrite(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer file.Close()

	switch ext := filepath.Ext(fpath); ext {
	case ".css":
		w.Header().Set("Content-Type", "text/css")
	case ".js":
		w.Header().Set("Content-Type", "application/javascript")
	case ".json":
		w.Header().Set("Content-Type", "application/json")
	case ".png", ".jpg", ".jpeg", ".gif", ".webp":
		w.Header().Set("Content-Type", "image/"+ext[1:])
	case ".svg":
		w.Header().Set("Content-Type", "image/svg+xml")
	case ".html", ".htm":
		w.Header().Set("Content-Type", "text/html")
	default:
		// 未显式列出的扩展名（.pdf/.woff2/.mp4 等）查系统 MIME 表，
		// 查不到再退回 text/plain，避免常见类型被误标为纯文本
		if ct := mime.TypeByExtension(ext); ct != "" {
			w.Header().Set("Content-Type", ct)
		} else {
			w.Header().Set("Content-Type", "text/plain")
		}
	}

	if readSeeker, ok := file.(io.ReadSeeker); ok {
		http.ServeContent(w, r, fileInfo.Name(), fileInfo.ModTime(), readSeeker)
		return
	}
	// 回退方案：非 ReadSeeker 无法使用 http.ServeContent 缓存读取
	b, err := io.ReadAll(file)
	if err != nil {
		errWrite(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(b)))
	w.Write(b)
}

// writtenWriter 包装 http.ResponseWriter，跟踪响应是否已被写出。
// 用于 ServeHTTP 链尾判断是否需要走 WWWROOT 兜底 / 404。
type writtenWriter struct {
	http.ResponseWriter
	written bool
}

func (ww *writtenWriter) WriteHeader(code int) {
	ww.written = true
	ww.ResponseWriter.WriteHeader(code)
}

func (ww *writtenWriter) Write(b []byte) (int, error) {
	ww.written = true
	return ww.ResponseWriter.Write(b)
}

// Flush 转发底层 http.Flusher（存在时）。实际刷新意味着响应头已提交（隐式 200），
// 故同步标记 written，避免链尾在流式响应之后再追加兜底/404 内容。
func (ww *writtenWriter) Flush() {
	if f, ok := ww.ResponseWriter.(http.Flusher); ok {
		ww.written = true
		f.Flush()
	}
}

// Unwrap 暴露底层 ResponseWriter，使 http.ResponseController 的能力探测能透传到底层实现。
func (ww *writtenWriter) Unwrap() http.ResponseWriter {
	return ww.ResponseWriter
}

// Hijack 转发底层 http.Hijacker（存在时）。中间件包装 writer 后，调用方
// （如 WebSocket 代理）仍可能直接断言 http.Hijacker，必须显式透传，
// 否则断言失败导致连接劫持类功能不可用。
func (ww *writtenWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	ww.written = true
	if hj, ok := ww.ResponseWriter.(http.Hijacker); ok {
		return hj.Hijack()
	}
	return nil, nil, errors.New("httpsvr: underlying ResponseWriter does not implement http.Hijacker")
}
