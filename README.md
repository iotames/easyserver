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

入口文件 `main.go`

```
package main

import (
	"github.com/iotames/easyserver/httpsvr"
	"github.com/iotames/easyserver/response"
)

func main() {
	s := httpsvr.NewEasyServer(":1598")
	s.AddHandler("GET", "/hello", func(ctx httpsvr.Context) {
		ctx.Writer.Write(response.NewApiDataOk("hello api").Bytes())
	})
	s.ListenAndServe()
}
```
