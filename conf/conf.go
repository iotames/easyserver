package conf

import (
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
	return "static"
}
