package custom

import (
	"embed"
)

//go:embed *.json
var jsonFS embed.FS

func GetFs() embed.FS {
	return jsonFS
}
