package httpsvr

import "fmt"

func (s *EasyServer) AddMiddleHead(middle MiddleHandle) {
	s.headMiddles = append(s.headMiddles, middle)
}

func (s *EasyServer) AddMiddleTail(middle MiddleHandle) {
	s.tailMiddles = append(s.tailMiddles, middle)
}

func (s *EasyServer) appendMiddleware(middles ...MiddleHandle) {
	s.middles = append(s.middles, middles...)
}

var staticMap map[string]string

// AddStaticPath 添加静态资源路径映射
// Example:
//
//	AddStatic("/static", "./static")
func (s *EasyServer) AddStatic(urlPathBegin string, wwwroot string) error {
	if staticMap == nil {
		staticMap = make(map[string]string)
		s.AddMiddleHead(NewMiddleStatic(urlPathBegin, wwwroot))
		staticMap[urlPathBegin] = wwwroot
		return nil
	}
	_, ok := staticMap[urlPathBegin]
	if ok {
		return fmt.Errorf("urlPathBegin (%s)->(%s) already exists", urlPathBegin, wwwroot)
	}
	staticMap[urlPathBegin] = wwwroot
	s.AddMiddleHead(NewMiddleStatic(urlPathBegin, wwwroot))
	return nil
}

var hasSetCORS bool

// SetCORS 设置CORS跨域访问
//
// allowOrigin: 允许的域名，如 "*" 表示所有域名都可以访问
//
//	SetCORS("*")
func (s *EasyServer) SetCORS(allowOrigin string) error {
	if hasSetCORS {
		return fmt.Errorf("CORS has already been set: %s", allowOrigin)
	}
	s.AddMiddleHead(NewMiddleCORS(allowOrigin))
	hasSetCORS = true
	return nil
}
