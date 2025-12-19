package main

import (
	"fmt"
	"net/http"
	"time"

	e "github.com/iotames/easyserver"
)

var svr *e.Server

func main() {
	svr = e.NewServer(":1212")

	// 设置全局响应头
	svr.SetHeader("Server", "EasyServer")
	// 全局生效，慎用！建议按路由分组进行配置
	// svr.SetHeader("Content-Type", "text/plain; charset=utf-8")
	// svr.SetHeader("Content-Type", "application/json")

	svr.SetData("key1", "mysitename, copyright")
	svr.AddMiddleHead(UserAuthMiddle{})
	svr.AddHandler("GET", "/", func(ctx e.HttpContext) {
		ctx.Writer.Write([]byte("hello world"))
	})
	svr.AddHandler("GET", "/hello", hello)
	svr.AddGetHandler("/setheader", func(ctx e.HttpContext) {
		k := ctx.GetQueryValue("k", "k1")
		v := ctx.GetQueryValue("v", "v1212356")

		ctx.SetHeader(k, v)
	})
	svr.ListenAndServe()
}

func hello(ctx e.HttpContext) {
	// 设置本次HTTP请求的响应头
	ctx.SetHeader("Content-Type", "application/json")

	df := ctx.DataFlow                                // 获取从上游中间件传递下来的数据
	username := df.GetData("username").Value.(string) // 获取用户鉴权中间件传递下来的数据
	data := map[string]any{"username": username}      // API返回的主数据
	// 返回响应的内容
	e.ResponseJson(ctx, data, "success", 200) // 封装API返回的整体数据

	// 处理后续事务
	dtkeys := df.GetDataKeys()                            // 获取所有数据的key
	costime := time.Since(df.GetStartAt()).Microseconds() // 获取本次请求的耗时
	globalData1 := svr.GetData("key1").Value.(string)     // 获取全局数据，比如网站名，版权信息等
	fmt.Printf("---hello--GetDataKeys(%+v)--globalData1(ke1=%s)--cost(%v ms)----\n", dtkeys, globalData1, costime)
}

type UserAuthMiddle struct{}

// 自定义用户中间件：比如进行用户认证，并往下游传递数据
func (h UserAuthMiddle) Handler(w http.ResponseWriter, r *http.Request, dataFlow *e.HttpDataFlow) (next bool) {
	dataFlow.SetDataReadonly("username", "iotames")
	return true
}
