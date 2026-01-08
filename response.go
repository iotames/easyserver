package easyserver

import (
	"github.com/iotames/easyserver/response"
)

func SetOkCode(code int) {
	response.SetOkCode(code)
}

// ResponseJson 响应JSON数据
func ResponseJson(ctx HttpContext, data map[string]any, msg string, code int) error {
	ctx.SetHeader("Content-Type", "application/json")
	_, err := ctx.Writer.Write(response.NewApiData(data, msg, code).Bytes())
	return err
}

// ResponseJsonOk 响应成功
func ResponseJsonOk(ctx HttpContext, msg string) error {
	ctx.SetHeader("Content-Type", "application/json")
	_, err := ctx.Writer.Write(response.NewApiDataOk(msg).Bytes())
	return err
}

// ResponseJsonFail 响应失败
func ResponseJsonFail(ctx HttpContext, msg string, code int) error {
	ctx.SetHeader("Content-Type", "application/json")
	_, err := ctx.Writer.Write(response.NewApiDataFail(msg, code).Bytes())
	return err
}

// ResponseText 响应文本数据
func ResponseText(ctx HttpContext, b []byte) error {
	_, err := ctx.Writer.Write(b)
	return err
}
