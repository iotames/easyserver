package httpsvr

import (
	"net/http"
	"strings"
)

// middleRouter 路由中间件。处理业务逻辑
type middleRouter struct {
	routingList []Routing
}

func NewMiddleRouter(routingList []Routing) *middleRouter {
	return &middleRouter{routingList: routingList}
}

// Handler 路由中间件。处理业务逻辑
func (m middleRouter) Handler(w http.ResponseWriter, r *http.Request, dataFlow *DataFlow) (subNext bool) {
	routings := m.routingList
	isMatch := false
	rpath := r.URL.Path
	rmethod := r.Method

	for _, rt := range routings {
		if rt.Path == rpath {
			// UrlPath匹配成功
			if len(rt.Methods) == 0 {
				// 匹配任意的Request Mothod请求方法
				isMatch = true
			} else {
				for _, m := range rt.Methods {
					// strings.ToUpper(m) == strings.ToUpper(rmethod)
					if strings.EqualFold(m, rmethod) {
						// 匹配指定的Request Mothod请求方法。如GET, POST, PUT, DELETE
						isMatch = true
						break
					}
				}
			}
			if isMatch {
				// 匹配UrlPath和RequestMethod，执行处理函数
				rt.handler(w, r, dataFlow)
				break
			}
		}
	}
	if !isMatch {
		// 匹配不到UrlPath和RequestMethod
		ResponseNotFound(w, r)
	}
	return true
}
