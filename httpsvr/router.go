package httpsvr

import (
	"net/http"
)

type Routing struct {
	Path    string
	Methods []string
	handler func(w http.ResponseWriter, r *http.Request, dataFlow *DataFlow)
}

func GetDefaultRoutingList() []Routing {
	return []Routing{
		{Path: "/", handler: func(w http.ResponseWriter, r *http.Request, dataFlow *DataFlow) {
			w.Write([]byte("Hello EasyServer"))
		}},
	}
}
