package main

import (
	"fmt"
	"time"

	"github.com/iotames/easyserver/httpsvr"
	"github.com/iotames/easyserver/response"
)

func main() {

	s := httpsvr.NewEasyServer(":1598")
	s.SetData("dt111", "dtv126")
	s.AddHandler("GET", "/hello", func(ctx httpsvr.Context) {
		df := ctx.DataFlow
		dtkeys := df.GetDataKeys()
		err := ctx.DataFlow.SetData("startat", "hello")
		costime := time.Since(ctx.DataFlow.GetStartAt())
		fmt.Printf("-------cors(%s)--dtkeys(%+v)--cost(%.6fs)--startat(%v)--resetErr(%v)---\n", df.GetStr("CorsAllowOrigin"), dtkeys, costime.Seconds(), df.GetStartAt(), err)
		ctx.Writer.Write(response.NewApiDataOk("hello api").Bytes())
	})

	s.ListenAndServe()
}

func HelloWordd() {
	fmt.Println("HELLO GLAYUI")
}
