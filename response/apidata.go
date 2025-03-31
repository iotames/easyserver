package response

import (
	"encoding/json"
	"net/http"
)

type JsonObject map[string]interface{}

type ResponseApiData struct {
	Code int        `json:"code"`
	Msg  string     `json:"msg"`
	Data JsonObject `json:"data"`
}

func NewApiData(data JsonObject, msg string, code int) *ResponseApiData {
	return &ResponseApiData{Data: data, Msg: msg, Code: code}
}

func (a ResponseApiData) Bytes() []byte {
	b, _ := json.Marshal(a)
	return b
}

func (a ResponseApiData) String() string {
	return string(a.Bytes())
}

func NewApiDataOk(msg string) *ResponseApiData {
	return NewApiData(JsonObject{}, msg, http.StatusOK)
}

func NewApiDataFail(msg string, code int) *ResponseApiData {
	return NewApiData(JsonObject{}, msg, code)
}

func NewApiDataNotFound() *ResponseApiData {
	return NewApiDataFail("NotFound.无法找到请求对象", http.StatusNotFound)
}

func NewApiDataUnauthorized() *ResponseApiData {
	return NewApiDataFail("Unauthorized.您没有权限访问此页面", http.StatusUnauthorized)
}

func NewApiDataMethodNotAllowed() *ResponseApiData {
	return NewApiDataFail("MethodNotAllowed.不允许的请求方法", http.StatusMethodNotAllowed)
}

func NewApiDataServerError(msg string) *ResponseApiData {
	return NewApiDataFail("ServerError.服务器内部错误:"+msg, http.StatusInternalServerError)
}

func NewApiDataQueryArgsError(msg string) *ResponseApiData {
	return NewApiDataFail("QueryArgsError.请求参数错误:"+msg, http.StatusBadRequest)
}
