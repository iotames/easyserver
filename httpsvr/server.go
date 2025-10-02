package httpsvr

import (
	"crypto/tls"
	"fmt"
	"net/http"
)

type EasyServer struct {
	httpServer  *http.Server
	routingList []Routing
	headMiddles []MiddleHandle
	middles     []MiddleHandle
	tailMiddles []MiddleHandle
	data        map[string]GlobalData
}

// NewEasyServer addr like: ":1598", "127.0.0.1:1598"
// You Can SET ENV: USE_EMBED_FILE=true To UseEmbedFile
func NewEasyServer(addr string) *EasyServer {
	fmt.Printf(`
	欢迎使用 EasyServer v1.0.1
	运行地址: %s
`, addr)
	return &EasyServer{httpServer: newServer(addr)}
}

func (s *EasyServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// 初始化dataflow。每个请求的生命周期中，只存在一个dataflow对象。
	// TODO 可以取出RemoteIP, UserAgent 等信息，作为dataflow的一部分
	dataFlow := NewDataFlow()
	// 按顺序依次执行中间件。业务处理逻辑包含在路由中间件里
	for _, m := range s.middles {
		if !m.Handler(w, r, dataFlow) {
			break
		}
	}
}

func (s *EasyServer) ListenAndServe() error {
	s.listenPrepare()
	return s.httpServer.ListenAndServe()
}

func (s *EasyServer) ListenAndServeTLS(certFile, keyFile string) error {
	s.listenPrepare()
	return s.httpServer.ListenAndServeTLS(certFile, keyFile)
}

// ConfTls 自定义TLS配置。
//
//	import "crypto/tls"
//
//	tlsConfig := &tls.Config{
//		MinVersion:               tls.VersionTLS12,
//	}
//	s.ConfTls(tlsConfig)
func (s *EasyServer) ConfTls(tlsConf *tls.Config) {
	s.httpServer.TLSConfig = tlsConf
}

func (s *EasyServer) listenPrepare() {
	// if len(s.middles) == 0 {
	// 	s.middles = GetDefaultMiddlewareList()
	// }
	if len(s.routingList) == 0 {
		fmt.Printf("----routingList不能为空。已启用默认路由设置。请使用AddRouting或AddHandler方法添加路由-----\n")
		s.routingList = GetDefaultRoutingList()
	}
	// 前置中间件：包含静态资源设置，CORS跨域，处理用户验证等前置组件
	s.addMiddleware(s.headMiddles...)
	// 路由中间件。处理业务主逻辑。
	s.addMiddleware(NewMiddleRouter(s.routingList))
	// 后置中间件：包含耗时统计等一些收尾工作。
	s.addMiddleware(s.tailMiddles...)

	for i, m := range s.middles {
		fmt.Printf("---[%d]--EnableMiddleware(%#v)--\n", i, m)
	}
	for i, r := range s.routingList {
		fmt.Printf("---[%d]--RoutePath(%s)---Methods(%+s)--\n", i, r.Path, r.Methods)
	}
	s.httpServer.Handler = s
}

func newServer(addr string) *http.Server {
	server := http.Server{
		Addr: addr,
		// Handler: http.HandlerFunc(httpHandler),
		// MaxHeaderBytes: 1 << 20, // 1048576
	}
	return &server
}
