package main

import (
	"github.com/iotames/easyserver/httpsvr"
	"github.com/iotames/easyserver/response"
)

func main() {
	s := httpsvr.NewEasyServer(":1212")
	s.AddMiddleware(httpsvr.NewMiddleCORS("*"))
	s.AddHandler("GET", "/hello", func(ctx httpsvr.Context) {
		ctx.Writer.Write(response.NewApiDataOk("hello api").Bytes())
	})
	s.ListenAndServe()
}
