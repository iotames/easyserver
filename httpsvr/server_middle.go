package httpsvr

import (
	"net/http"
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

func (s *EasyServer) AppendRouting(routings ...Routing) {
	s.routingList = append(s.routingList, routings...)
}

func (s *EasyServer) AddMiddleHead(middle MiddleHandle) {
	s.headMiddles = append(s.headMiddles, middle)
}

func (s *EasyServer) AddMiddleTail(middle MiddleHandle) {
	s.tailMiddles = append(s.tailMiddles, middle)
}

func (s *EasyServer) appendMiddleware(middles ...MiddleHandle) {
	s.middles = append(s.middles, middles...)
}
