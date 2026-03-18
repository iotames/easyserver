package httpsvr

import (
	"os"
)

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
