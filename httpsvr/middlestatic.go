package httpsvr

import (
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/iotames/easyserver/conf"
)

// middleStatic 静态资源中间件
type middleStatic struct {
	wwwrootDir    string
	staticUrlPath string
}

// NewMiddleStatic 静态资源中间件。例: NewMiddleStatic("/static", "./static"), NewMiddleStatic("/", "")
// urlPathBegin 启用静态资源的URL路径。必须以正斜杠/开头。如 "/static/" 或 "/static"
// wwwroot 网站根目录。默认值为当前工作目录 或 "./"
func NewMiddleStatic(urlPathBegin string, wwwroot string) *middleStatic {
	if wwwroot == "" {
		// 获取当前工作目录
		wwwroot = conf.GetStaticDir()
	}
	return &middleStatic{wwwrootDir: wwwroot, staticUrlPath: urlPathBegin}
}

// matchStaticUrl 匹配命中URL静态资源
func (m middleStatic) matchStaticUrl(w http.ResponseWriter, r *http.Request, fpath, staticUrlPath string) bool {
	var err error
	var fileInfo fs.FileInfo
	if conf.UseEmbedFile() {
		// fileInfo, err = fs.Stat(resource.ResourceFs, fpath)
		panic("not support UseEmbedFile")
	} else {
		fileInfo, err = os.Stat(fpath)
	}
	fmt.Printf("---staticUrlPath(%s)--fpath(%s)--os.Stat.err(%v)--\n", staticUrlPath, fpath, err)

	// 1. 先检查文件是否存在（不实际打开文件）
	if err != nil {
		if os.IsNotExist(err) {
			// 文件不存在，继续后续中间件处理
			// errWrite(w, "file IsNotExist ", 400)
			return true
		}
		// 其他错误
		errWrite(w, err.Error(), http.StatusInternalServerError)
		return false
	}

	// 2. 检查是否是目录
	if fileInfo.IsDir() {
		errWrite(w, "directory access not allowed", http.StatusForbidden)
		return false
	}

	// 3. 只有在前面的检查都通过后，才实际打开文件
	var file fs.File
	var b []byte
	if conf.UseEmbedFile() {
		// file, err = resource.ResourceFs.Open(fpath)
		panic("not support UseEmbedFile")
	} else {
		file, err = os.Open(fpath)
	}
	if err != nil {
		errWrite(w, err.Error(), http.StatusInternalServerError)
		return false
	}
	defer file.Close()

	// 4. 设置Content-Type
	ext := filepath.Ext(fpath)
	switch ext {
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
		w.Header().Set("Content-Type", "text/plain")
	}

	// 5. 提供文件内容
	if readSeeker, ok := file.(io.ReadSeeker); ok {
		http.ServeContent(w, r, fileInfo.Name(), fileInfo.ModTime(), readSeeker)
	} else {
		// 回退方案
		fmt.Printf("----not-readSeeker--file(%s)---can not use cache to read file by http.ServeContent---\n", fpath)
		b, err = io.ReadAll(file)
		if err != nil {
			errWrite(w, err.Error(), http.StatusInternalServerError)
			return false
		}
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(b)))
		w.Write(b)
	}

	return false
}

// middleStatic 定义静态资源
func (m middleStatic) Handler(w http.ResponseWriter, r *http.Request, dataFlow *DataFlow) (subNext bool) {
	rpath := r.URL.Path

	// 检查请求路径是否以静态URL前缀开头
	// if strings.Index(rpath, m.staticUrlPath) != 0
	if !strings.HasPrefix(rpath, m.staticUrlPath) {
		return true // 不处理，交给下一个中间件
	}

	var fpath string
	if conf.UseEmbedFile() {
		fpath = strings.TrimPrefix(rpath, "/")
	} else {
		// 处理普通文件系统的情况
		// 1. 移除静态URL前缀
		relativePath := strings.TrimPrefix(rpath, m.staticUrlPath)
		// 2. 安全拼接路径，防止目录遍历攻击
		fpath = filepath.Join(m.wwwrootDir, filepath.Clean("/"+relativePath))
	}
	fmt.Printf("---[Static] Request Path:(%s)---File Path:(%s)---staticUrlPath(%s)---\n", rpath, fpath, m.staticUrlPath)
	return m.matchStaticUrl(w, r, fpath, m.staticUrlPath)
}
