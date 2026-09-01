package httpsvr

import (
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/iotames/easyserver/conf"
)

// middleStatic 静态资源中间件
type middleStatic struct {
	wwwrootDir    string
	staticUrlPath string
}

// NewMiddleStatic 静态资源中间件。
// urlPathBegin 启用静态资源的URL路径。必须以正斜杠/开头。如 "/static/" 或 "/static"
// wwwroot 网站根目录。默认值为当前工作目录 或 "./"
//
//	NewMiddleStatic("/static", "./static") NewMiddleStatic("/", "")
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
		panic("not support UseEmbedFile")
	}
	fileInfo, err = os.Stat(fpath)
	fmt.Printf("---staticUrlPath(%s)--fpath(%s)--os.Stat.err(%v)--\n", staticUrlPath, fpath, err)

	// 1. 先检查文件是否存在（不实际打开文件）
	if err != nil {
		if os.IsNotExist(err) {
			// 文件不存在，继续后续中间件处理
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

	// 3. 检查都通过后，设置Content-Type并提供文件内容
	serveFileObj(w, r, fpath, fileInfo)
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
		// 2. 安全拼接路径，防止目录遍历攻击。path.Clean 按POSIX语义折叠 ..；
		// 先把 \ 归一化为 /（path.Clean 不把 \ 当分隔符，Windows 下 /..\..\x
		// 的 .. 不会被折叠，随后 filepath.Join 又按 \ 折叠，可逃逸目录）
		relativePath = strings.ReplaceAll(relativePath, `\`, "/")
		fpath = filepath.Join(m.wwwrootDir, filepath.FromSlash(path.Clean("/"+relativePath)))
	}
	fmt.Printf("---[Static] Request Path:(%s)---File Path:(%s)---staticUrlPath(%s)---\n", rpath, fpath, m.staticUrlPath)
	return m.matchStaticUrl(w, r, fpath, m.staticUrlPath)
}
