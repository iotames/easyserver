package conf

import (
	"fmt"
	"os"
	"strings"
)

// 是否使用嵌入到可执行程序里的静态资源文件。默认false
func UseEmbedFile() bool {
	val, ok := os.LookupEnv("USE_EMBED_FILE")
	if ok {
		if strings.EqualFold(val, "true") || val == "1" {
			return true
		}
	}
	return false
}

func GetStaticDir() string {
	dirpath, ok := os.LookupEnv("STATIC_DIR")
	if ok {
		return dirpath
	}
	// 获取当前工作目录
	wd, err := os.Getwd()
	if err != nil {
		fmt.Println("获取当前工作目录失败:", err)
		return "./"
	}
	// fmt.Println("当前工作目录:", wd)
	return wd
}
