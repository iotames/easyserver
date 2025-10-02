package main

import (
	"github.com/iotames/easyserver/httpsvr"
)

func main() {
	s := httpsvr.NewEasyServer(":1212")
	// s.AddMiddleHead(httpsvr.NewMiddleStatic("/static", "./static"))
	s.AddMiddleHead(httpsvr.NewMiddleStatic("/", "./"))
	s.ListenAndServe()
}
