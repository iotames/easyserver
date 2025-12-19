package httpsvr

import (
	"net/http"
)

var initHeaders map[string]string

// SetHeader 设置响应头
func (s *EasyServer) SetHeader(k string, v string) {
	if initHeaders == nil {
		initHeaders = make(map[string]string, 20)
	}
	initHeaders[k] = v
	if s.responseHeaderMiddle == nil {
		s.responseHeaderMiddle = NewMiddle(func(w http.ResponseWriter, r *http.Request, dataFlow *DataFlow) (subNext bool) {
			for k, v := range initHeaders {
				w.Header().Set(k, v)
			}
			return true
		})
		s.AddMiddleHead(s.responseHeaderMiddle)
	}
}
