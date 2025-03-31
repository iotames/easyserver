package httpsvr

import (
	"net/http"

	"github.com/iotames/easyserver/response"
)

func ResponseNotFound(w http.ResponseWriter, r *http.Request) {
	dt := response.NewApiDataNotFound()
	w.Write(dt.Bytes())
}
