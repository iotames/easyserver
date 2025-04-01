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

// 自己以前写的代码注释掉，AI写的好像更简洁
func (m middleStatic) matchStaticUrl(w http.ResponseWriter, fpath, staticUrlPath string) bool {
	var err error
	// 匹配命中URL静态资源
	var finfo fs.FileInfo
	if conf.UseEmbedFile() {
		// finfo, err = fs.Stat(resource.ResourceFs, fpath)
		panic("not support UseEmbedFile")
	} else {
		finfo, err = os.Stat(fpath)
	}
	fmt.Printf("----in-range-staticUrlPath--staticUrlPath(%s)--err(%v)--\n", staticUrlPath, err)

	if err != nil {
		if os.IsNotExist(err) {
			// 文件不存在
			// errWrite(w, "file IsNotExist ", 400)
			return true
		}
		// 其他错误
		errWrite(w, err.Error(), 500)
		return false
	}

	if finfo.IsDir() {
		errWrite(w, "not allow visit dir path", 400)
		return false
	}

	var b []byte
	if conf.UseEmbedFile() {
		// b, err = resource.ResourceFs.ReadFile(fpath)
		panic("not support UseEmbedFile")
	} else {
		var f *os.File
		f, err = os.Open(fpath)
		if err != nil {
			errWrite(w, err.Error(), 500)
			return false
		}
		b, err = io.ReadAll(f)
	}

	if err != nil {
		errWrite(w, err.Error(), 500)
		return false
	}

	// 设置正确的Content-Type
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

	w.Header().Set(`Content-Length`, fmt.Sprintf("%d", len(b)))
	w.Write(b)
	// http.ServeContent(w, r, fileInfo.Name(), fileInfo.ModTime(), file)
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
		// fpath = rpath
		// if strings.Index(rpath, "/") == 0 {
		// 	fpath = strings.Replace(fpath, "/", "", 1)
		// }
		fpath = strings.TrimPrefix(rpath, "/")
	} else {
		// 处理普通文件系统的情况
		// 1. 移除静态URL前缀
		relativePath := strings.TrimPrefix(rpath, m.staticUrlPath)
		// 2. 安全拼接路径，防止目录遍历攻击
		fpath = filepath.Join(m.wwwrootDir, filepath.Clean("/"+relativePath))
	}
	fmt.Printf("---[Static] Request Path:(%s)---File Path:(%s)---staticUrlPath(%s)---\n", rpath, fpath, m.staticUrlPath)
	return m.matchStaticUrl(w, fpath, m.staticUrlPath)
}
