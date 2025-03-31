package httpsvr

import (
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"strings"

	"github.com/iotames/easyserver/conf"
	// "github.com/iotames/glayui/resource"
)

type MiddleHandle interface {
	Handler(w http.ResponseWriter, r *http.Request, dataFlow *DataFlow) (next bool)
}

// middleStatic 静态资源中间件
type middleStatic struct {
	wwwrootDir    string
	staticUrlPath []string
}

// middleCORS CORS跨域设置中间件
type middleCORS struct {
	allowOrigin string
}

// middleRouter 路由中间件。处理业务逻辑
type middleRouter struct {
	routingList []Routing
}

func NewMiddleRouter(routingList []Routing) *middleRouter {
	return &middleRouter{routingList: routingList}
}

// Handler 路由中间件。处理业务逻辑
func (m middleRouter) Handler(w http.ResponseWriter, r *http.Request, dataFlow *DataFlow) (subNext bool) {
	routings := m.routingList
	isMatch := false
	rpath := r.URL.Path
	rmethod := r.Method

	for _, rt := range routings {
		if rt.Path == rpath {
			// UrlPath匹配成功
			if len(rt.Methods) == 0 {
				// 匹配任意的Request Mothod请求方法
				isMatch = true
			} else {
				for _, m := range rt.Methods {
					// strings.ToUpper(m) == strings.ToUpper(rmethod)
					if strings.EqualFold(m, rmethod) {
						// 匹配指定的Request Mothod请求方法。如GET, POST, PUT, DELETE
						isMatch = true
						break
					}
				}
			}
			if isMatch {
				// 匹配UrlPath和RequestMethod，执行处理函数
				rt.handler(w, r, dataFlow)
				break
			}
		}
	}

	if !isMatch {
		// 匹配不到UrlPath和RequestMethod
		ResponseNotFound(w, r)
	}

	return true
}

// NewMiddleStatic 静态资源中间件。例: NewMiddleStatic("", []string{"/static/"})
// wwwroot 网站根目录。字符串末尾不用添加斜杠/。默认值为 "static"
// urlPathBegin 启用静态资源的URL路径。必须以正斜杠/开头和结尾。如 []string{"/static/"}
func NewMiddleStatic(wwwroot string, urlPathBegin []string) *middleStatic {
	if wwwroot == "" {
		wwwroot = conf.GetStaticDir()
	}
	return &middleStatic{wwwrootDir: wwwroot, staticUrlPath: urlPathBegin}
}

// middleStatic 定义静态资源
func (m middleStatic) Handler(w http.ResponseWriter, r *http.Request, dataFlow *DataFlow) (subNext bool) {
	var err error
	rpath := r.URL.Path
	fpath := m.wwwrootDir + rpath
	if conf.UseEmbedFile() {
		fpath = rpath
		if strings.Index(rpath, "/") == 0 {
			fpath = strings.Replace(fpath, "/", "", 1)
		}
	}

	for _, v := range m.staticUrlPath {
		if strings.Index(rpath, v) == 0 {
			// 匹配命中URL静态资源
			var finfo fs.FileInfo
			if conf.UseEmbedFile() {
				// finfo, err = fs.Stat(resource.ResourceFs, fpath)
				panic("not support UseEmbedFile")
			} else {
				finfo, err = os.Stat(fpath)
			}

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
			if strings.Contains(rpath, `.css`) {
				w.Header().Set(`Content-Type`, `text/css`)
			}
			if strings.Contains(rpath, `.js`) {
				w.Header().Set(`Content-Type`, `application/javascript`)
			}
			w.Header().Set(`Content-Length`, fmt.Sprintf("%d", len(b)))
			w.Write(b)
			return false
		}
	}
	return true
}

// NewMiddleCORS CORS中间件: 跨域设置。例: NewMiddleCORS("*")
// allowOrigin: 允许跨域的站点。默认值为 "*"。可将将 * 替换为指定的域名
func NewMiddleCORS(allowOrigin string) *middleCORS {
	if allowOrigin == "" {
		allowOrigin = "*"
	}
	return &middleCORS{allowOrigin: allowOrigin}
}

func (m middleCORS) Handler(w http.ResponseWriter, r *http.Request, dataFlow *DataFlow) (subNext bool) {
	w.Header().Add("Access-Control-Allow-Origin", m.allowOrigin)
	w.Header().Add("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE, UPDATE")
	w.Header().Add("Access-Control-Allow-Headers", "Origin, Content-Length, Content-Type, Accept, Token, Auth-Token, X-Requested-With")
	w.Header().Add("Access-Control-Expose-Headers", "Content-Length, Access-Control-Allow-Origin, Access-Control-Allow-Headers, Cache-Control, Content-Language, Content-Type")
	w.Header().Add("Access-Control-Allow-Credentials", "true")
	dataFlow.SetDataReadonly("CorsAllowOrigin", m.allowOrigin)
	return r.Method != "OPTIONS"
}

// GetDefaultMiddlewareList 获取中间件列表。按数组列表的顺序依次执行中间件
func GetDefaultMiddlewareList() []MiddleHandle {
	return []MiddleHandle{
		NewMiddleCORS("*"),
		NewMiddleStatic("", []string{"/static/"}),
	}
}

func errWrite(w http.ResponseWriter, msg string, code int) {
	w.WriteHeader(code)
	w.Write([]byte(msg))
}
