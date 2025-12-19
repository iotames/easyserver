package easyserver

import (
	"github.com/iotames/easyserver/httpsvr"
	"github.com/iotames/easyserver/response"
)

type Server = httpsvr.EasyServer
type HttpContext = httpsvr.Context
type HttpDataFlow = httpsvr.DataFlow

func NewServer(addr string) *httpsvr.EasyServer {
	return httpsvr.NewEasyServer(addr)
}

// ResponseJson 响应JSON数据
func ResponseJson(ctx httpsvr.Context, data map[string]any, msg string, code int) error {
	ctx.SetHeader("Content-Type", "application/json")
	_, err := ctx.Writer.Write(response.NewApiData(data, msg, code).Bytes())
	return err
}

// ResponseJsonOk 响应成功
func ResponseJsonOk(ctx httpsvr.Context, msg string) error {
	ctx.SetHeader("Content-Type", "application/json")
	_, err := ctx.Writer.Write(response.NewApiDataOk(msg).Bytes())
	return err
}

// ResponseJsonFail 响应失败
func ResponseJsonFail(ctx httpsvr.Context, msg string, code int) error {
	ctx.SetHeader("Content-Type", "application/json")
	_, err := ctx.Writer.Write(response.NewApiDataFail(msg, code).Bytes())
	return err
}

// ResponseText 响应文本数据
func ResponseText(ctx httpsvr.Context, b []byte) error {
	_, err := ctx.Writer.Write(b)
	return err
}
