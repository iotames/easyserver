package httpsvr

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type Context struct {
	Writer   http.ResponseWriter
	Request  *http.Request
	Server   *EasyServer
	DataFlow *DataFlow
}

// GetPostJson 获取POST请求中的JSON数据
//
//	var requestData map[string]interface{}
//	err := GetPostJson(&requestData)
func (ctx Context) GetPostJson(v any) error {
	// b1, err := ctx.Request.Body() runtime error: invalid memory address or nil pointer dereference
	// 读取请求体中的数据
	var err error
	var b []byte
	b, err = io.ReadAll(ctx.Request.Body)
	if err != nil {
		return fmt.Errorf("读取请求体失败io.ReadAll error: %w", err)
	}
	// 解析JSON数据
	err = json.Unmarshal(b, v)
	if err != nil {
		return fmt.Errorf("解析JSON失败json.Unmarshal error: %w", err)
	}
	return err
}

// GetQueryValue 获取URL参数
func (ctx Context) GetQueryValue(k string, defauleValue string) string {
	v := ctx.Request.URL.Query().Get(k)
	if v == "" {
		return defauleValue
	}
	return v
}
