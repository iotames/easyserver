package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/iotames/easyserver/httpsvr"
	"github.com/iotames/easyserver/response"
)

var svr *httpsvr.EasyServer

func main() {
	svr = httpsvr.NewEasyServer(":1212")
	svr.SetData("key1", "mysitename, copyright")
	svr.AddMiddleware(UserAuthMiddle{})
	svr.AddHandler("GET", "/", func(ctx httpsvr.Context) {
		ctx.Writer.Write([]byte("hello world"))
	})
	svr.AddHandler("GET", "/hello", hello)
	svr.ListenAndServe()
}

func hello(ctx httpsvr.Context) {
	df := ctx.DataFlow                                   // 获取从上游中间件传递下来的数据
	username := df.GetData("username").Value.(string)    // 获取用户鉴权中间件传递下来的数据
	data := map[string]interface{}{"username": username} // API返回的主数据
	result := response.NewApiData(data, "success", 200)  // 封装API返回的整体数据
	ctx.Writer.Write(result.Bytes())
	dtkeys := df.GetDataKeys()                            // 获取所有数据的key
	costime := time.Since(df.GetStartAt()).Microseconds() // 获取本次请求的耗时
	globalData1 := svr.GetData("key1").Value.(string)     // 获取全局数据，比如网站名，版权信息等
	fmt.Printf("---hello--GetDataKeys(%+v)--globalData1(ke1=%s)--cost(%v ms)----\n", dtkeys, globalData1, costime)
}

type UserAuthMiddle struct{}

// 自定义用户中间件：比如进行用户认证，并往下游传递数据
func (h UserAuthMiddle) Handler(w http.ResponseWriter, r *http.Request, dataFlow *httpsvr.DataFlow) (next bool) {
	dataFlow.SetDataReadonly("username", "iotames")
	return true
}
