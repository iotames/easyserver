package httpsvr

import (
	"net/http"

	"github.com/iotames/easyserver/response"
)

// AddHandler 添加路由
// 底层使用 strings.EqualFold 函数判断method，故大小写不敏感
func (s *EasyServer) AddHandler(method, urlpath string, ctxfunc func(ctx Context)) {
	handler := func(w http.ResponseWriter, r *http.Request, dataFlow *DataFlow) {
		hctx := Context{Writer: w, Request: r, Server: s, DataFlow: dataFlow}
		ctxfunc(hctx)
	}
	routing := Routing{Methods: []string{method}, Path: urlpath, handler: handler}
	s.AppendRouting(routing)
}

// AddPostHandler 添加POST路由
func (s *EasyServer) AddPostHandler(urlpath string, ctxfunc func(ctx Context)) {
	s.AddHandler("POST", urlpath, ctxfunc)
}

// AddGetHandler 添加GET路由
func (s *EasyServer) AddGetHandler(urlpath string, ctxfunc func(ctx Context)) {
	s.AddHandler("GET", urlpath, ctxfunc)
}

func ResponseNotFound(w http.ResponseWriter, r *http.Request) {
	dt := response.NewApiDataNotFound()
	w.Write(dt.Bytes())
}
