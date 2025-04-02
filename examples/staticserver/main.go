package main

import (
	"github.com/iotames/easyserver/httpsvr"
)

func main() {
	s := httpsvr.NewEasyServer(":1212")
	// s.AddMiddleware(httpsvr.NewMiddleStatic("/static", "./static"))
	s.AddMiddleware(httpsvr.NewMiddleStatic("/", "./"))
	s.ListenAndServe()
}
