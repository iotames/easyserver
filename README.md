<div align="center">
   <span style="font-size:100px">🧰</span>
  <br><a href="https://github.com/iotames/easyserver">Github</a> | <a href="https://gitee.com/catmes/easyserver">Gitee</a>
  <br>简单易用的HttpServer<br>助你成功转职Golang工程师！
</div>


## 简介

[![GoDoc](https://badgen.net/badge/Go/referenct)](https://pkg.go.dev/github.com/iotames/easyserver)
[![License](https://badgen.net/badge/License/MIT/green)](https://github.com/iotames/easyserver/blob/main/LICENSE)

简单的HTTP服务器功能实现，简易的API接口调用。


## 快速开始


API接口服务：

```go
package main

import (
	"github.com/iotames/easyserver/httpsvr"
	"github.com/iotames/easyserver/response"
)

func main() {
	s := httpsvr.NewEasyServer(":1212")
	s.AddMiddleHead(httpsvr.NewMiddleCORS("*"))
	// 默认状态码code=200，可自定义code=0
	response.SetOkCode(0)
	s.AddHandler("GET", "/hello", func(ctx httpsvr.Context) {
		ctx.Writer.Write(response.NewApiDataOk("hello api").Bytes())
	})
	s.ListenAndServe()
}
```

静态资源服务：

```go
package main

import (
	"github.com/iotames/easyserver/httpsvr"
)

func main() {
	s := httpsvr.NewEasyServer(":1212")
	s.AddMiddleHead(httpsvr.NewMiddleStatic("/static", "./static"))
	s.ListenAndServe()
}
```

自定义中间件，全部配置功能，上下文数据流传递：

```go
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
	svr.AddMiddleHead(UserAuthMiddle{})
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

```


## 测试

```bash
# 运行所有测试（-count=1 禁用缓存，确保每次重新运行）
go test -count=1 -timeout 30s .

# 仅运行 HTTP 相关测试
go test -v -count=1 -timeout 30s -run "^TestServer$|^TestAddStatic|^TestSetCORS|^TestResponse|^TestContext|^TestMiddleware|^TestDataFlow|^TestCustomOkCode|^TestNotFound|^TestPostJson|^TestMultipleHandlers" .

# 仅运行 TCP/WebSocket 相关测试
go test -v -count=1 -timeout 30s -run "^TestWebSocket|^TestTCP|^TestServerSend" .

# 单个测试函数
go test -v -count=1 -run TestWebSocketPack .
```