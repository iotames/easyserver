package main

import (
	"github.com/iotames/easyserver"
)

func main() {
	// s := httpsvr.NewEasyServer(":1212")
	// // 默认状态码code=200，可自定义code=0
	// response.SetOkCode(0)
	// s.AddMiddleHead(httpsvr.NewMiddleCORS("*"))
	// s.AddHandler("GET", "/hello", func(ctx httpsvr.Context) {
	// 	ctx.Writer.Write(response.NewApiDataOk("hello api").Bytes())
	// })

	s := easyserver.NewServer(":1212")
	easyserver.SetOkCode(0)
	err := s.SetCORS("*")
	if err != nil {
		panic(err)
	}
	s.AddGetHandler("/hello", func(ctx easyserver.HttpContext) {
		easyserver.ResponseJsonOk(ctx, "hello api")
	})

	if err := s.ListenAndServe(); err != nil {
		panic(err)
	}

}
