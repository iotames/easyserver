package main

import (
	"github.com/iotames/easyserver/httpsvr"
	"github.com/iotames/easyserver/response"
)

func main() {
	s := httpsvr.NewEasyServer(":1212")
	// 默认状态码code=200，可自定义code=0
	response.SetOkCode(0)
	s.AddMiddleHead(httpsvr.NewMiddleCORS("*"))
	s.AddHandler("GET", "/hello", func(ctx httpsvr.Context) {
		ctx.Writer.Write(response.NewApiDataOk("hello api").Bytes())
	})
	if err := s.ListenAndServe(); err != nil {
		panic(err)
	}
}
