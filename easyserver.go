package easyserver

import (
	"github.com/iotames/easyserver/httpsvr"
)

type Server = httpsvr.EasyServer
type HttpContext = httpsvr.Context
type HttpDataFlow = httpsvr.DataFlow

func NewServer(addr string) *Server {
	return httpsvr.NewEasyServer(addr)
}
