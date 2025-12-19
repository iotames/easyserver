package httpsvr

import (
	"net/http"
)

type MiddleHandle interface {
	Handler(w http.ResponseWriter, r *http.Request, dataFlow *DataFlow) (next bool)
}

// // GetDefaultMiddlewareList 获取中间件列表。按数组列表的顺序依次执行中间件
// func GetDefaultMiddlewareList() []MiddleHandle {
// 	return []MiddleHandle{
// 		NewMiddleCORS("*"),
// 	}
// }

func errWrite(w http.ResponseWriter, msg string, code int) {
	w.WriteHeader(code)
	w.Write([]byte(msg))
}

// middleCommon 通用中间件
type middleCommon struct {
	handlerFunc func(w http.ResponseWriter, r *http.Request, dataFlow *DataFlow) (subNext bool)
}

// NewMiddle 创建通用中间件
func NewMiddle(handlerFunc func(w http.ResponseWriter, r *http.Request, dataFlow *DataFlow) (subNext bool)) *middleCommon {
	return &middleCommon{handlerFunc: handlerFunc}
}

func (m middleCommon) Handler(w http.ResponseWriter, r *http.Request, dataFlow *DataFlow) (subNext bool) {
	return m.handlerFunc(w, r, dataFlow)
}
