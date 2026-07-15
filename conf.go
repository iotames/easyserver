package easyconf

// import "path/filepath"

import (
	"fmt"
	"os"

	"strings"
)

// Conf 配置管理器，管理配置文件列表和已注册的配置项。
// 通过 NewConf 创建实例，使用 StringVar/IntVar 等方法注册配置项，
// 最后调用 Parse 从配置文件、环境变量、命令行参数中读取值。
type Conf struct {
	files []string
	items []*ConfItem
}

// NewConf 定义配置文件。留空默认为: .env, default.env
// 配置优先级：命令行参数 > 系统环境变量 > 配置文件列表（按顺序，优先级依次降低）
// 配置文件列表：首个文件优先级最高。前面文件的配置值，会覆盖后面文件的配置值。
func NewConf(files ...string) *Conf {
	defaultFiles := []string{".env", "default.env"}
	if len(files) == 0 {
		files = defaultFiles
	}
	for _, ff := range files {
		if ff == "" {
			panic("配置文件路径不能为空")
		}
	}
	return &Conf{files: files}
}

// DefaultString 默认配置的文件内容
func (cf Conf) DefaultString() string {
	var result []string
	for _, item := range cf.items {
		result = append(result, item.DefaultString())
	}
	return strings.Join(result, "\n\n")
}

// String 生成当前配置值的完整 .env 文件内容（含注释）。
// 每个配置项以空行分隔，包含标题、默认值说明、用法注释和键值对。
func (cf Conf) String() string {
	var result []string
	for _, item := range cf.items {
		result = append(result, item.String())
	}
	return strings.Join(result, "\n\n")
}

func createEnvFile(fpath, content string) error {
	if fpath == "" {
		return fmt.Errorf("createEnvFile empty file")
	}
	f, err := os.Create(fpath)
	if err != nil {
		return fmt.Errorf("create env file(%s)err(%v)", fpath, err)
	}
	_, err = f.WriteString(content)
	if err != nil {
		return fmt.Errorf("write env file(%s)err(%v)", fpath, err)
	}
	return f.Close()
}
