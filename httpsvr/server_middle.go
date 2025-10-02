package httpsvr

import (
	"net/http"
)

type Context struct {
	Writer   http.ResponseWriter
	Request  *http.Request
	Server   *EasyServer
	DataFlow *DataFlow
}

func (s *EasyServer) AddHandler(method, urlpath string, ctxfunc func(ctx Context)) {
	handler := func(w http.ResponseWriter, r *http.Request, dataFlow *DataFlow) {
		hctx := Context{Writer: w, Request: r, Server: s, DataFlow: dataFlow}
		ctxfunc(hctx)
	}
	routing := Routing{Methods: []string{method}, Path: urlpath, handler: handler}
	s.AddRouting(routing)
}

func (s *EasyServer) AddRouting(routing Routing) {
	s.routingList = append(s.routingList, routing)
}

func (s *EasyServer) AddMiddleHead(middle MiddleHandle) {
	s.headMiddles = append(s.headMiddles, middle)
}

func (s *EasyServer) AddMiddleTail(middle MiddleHandle) {
	s.tailMiddles = append(s.tailMiddles, middle)
}

func (s *EasyServer) addMiddleware(middles ...MiddleHandle) {
	s.middles = append(s.middles, middles...)
}
