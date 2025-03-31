package main

import (
	"github.com/iotames/easyserver/httpsvr"
)

func main() {
	s := httpsvr.NewEasyServer(":1123")
	// TODO
	s.AddMiddleware(httpsvr.NewMiddleStatic("static", []string{"/static/"}))
	s.ListenAndServe()
}
