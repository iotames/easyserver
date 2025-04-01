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

```
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
```

静态资源服务：

```
package main

import (
	"github.com/iotames/easyserver/httpsvr"
)

func main() {
	s := httpsvr.NewEasyServer(":1212")
	s.AddMiddleware(httpsvr.NewMiddleStatic("/static", "./static"))
	s.ListenAndServe()
}
```