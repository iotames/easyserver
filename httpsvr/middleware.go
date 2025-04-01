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
