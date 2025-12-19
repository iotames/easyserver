package httpsvr

import (
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
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

// GetUploadFile 获取上传的文件
// formKey 表单字段名。默认为file
func (ctx Context) GetUploadFile(formKey string, saveFilePath string) (file multipart.File, fileInfo *multipart.FileHeader, err error) {
	if formKey == "" {
		formKey = "file"
	}

	// 获取上传的文件
	file, fileInfo, err = ctx.Request.FormFile(formKey)
	if err != nil {
		err = fmt.Errorf("get request file error: %w", err)
		return
	}
	// fmt.Printf("----------upload-----filename(%s)----\n", header.Filename)
	defer file.Close()

	uploadDir := filepath.Dir(saveFilePath)

	if !isPathExists(uploadDir) {
		if err = os.MkdirAll(uploadDir, 0755); err != nil {
			err = fmt.Errorf("创建上传目录失败: %w", err)
			return
		}
	}
	var dst *os.File
	dst, err = os.Create(saveFilePath)
	if err != nil {
		err = fmt.Errorf("目标文件 %s 创建失败: %w", saveFilePath, err)
		return
	}
	defer dst.Close()

	// 将上传文件内容复制到目标文件
	if _, err = io.Copy(dst, file); err != nil {
		err = fmt.Errorf("目标文件 %s 保存失败: %w", saveFilePath, err)
		return
	}
	return
}

// SetHeader 设置响应头
//
//	ctx.SetHeader("Content-Type", "text/plain; charset=utf-8")
//	ctx.SetHeader("Content-Type", "application/json")
func (ctx Context) SetHeader(key, value string) {
	ctx.Writer.Header().Set(key, value)
}

// isPathExists 判断文件或文件夹是否存在
func isPathExists(path string) bool {
	_, err := os.Stat(path)
	if err == nil {
		// fmt.Println(stat.IsDir())
		return true
	}
	if os.IsNotExist(err) {
		return false
	}
	return false
}
