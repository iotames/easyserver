package httpsvr

import (
	"net/http"
)

// middleCORS CORS跨域设置中间件
type middleCORS struct {
	allowOrigin string
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
	w.Header().Set("Access-Control-Allow-Origin", m.allowOrigin)
	w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE, UPDATE")
	w.Header().Set("Access-Control-Allow-Headers", "Origin, Content-Length, Content-Type, Accept, Token, Auth-Token, X-Requested-With")
	w.Header().Set("Access-Control-Expose-Headers", "Content-Length, Access-Control-Allow-Origin, Access-Control-Allow-Headers, Cache-Control, Content-Language, Content-Type")
	w.Header().Set("Access-Control-Allow-Credentials", "true")
	dataFlow.SetDataReadonly("CorsAllowOrigin", m.allowOrigin)
	return r.Method != "OPTIONS"
}
