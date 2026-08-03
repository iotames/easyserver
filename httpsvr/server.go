package httpsvr

import (
	"context"
	"crypto/tls"
	"log"
	"net/http"
	"sync"
)

const MAIN_VERSION = "v1.4.0"

type EasyServer struct {
	httpServer           *http.Server
	responseHeaderMiddle MiddleHandle
	routingList          []Routing
	headMiddles          []MiddleHandle
	middles              []MiddleHandle
	tailMiddles          []MiddleHandle
	data                 map[string]GlobalData
	lock                 *sync.RWMutex
	initOnce             sync.Once
}

// NewEasyServer addr like: ":1598", "127.0.0.1:1598"
// You Can SET ENV: USE_EMBED_FILE=true To UseEmbedFile
func NewEasyServer(addr string) *EasyServer {
	log.Printf(`
	欢迎使用 EasyServer %s ------>>> github.com/iotames/easyserver
	运行地址: %s
`, MAIN_VERSION, addr)
	return &EasyServer{
		httpServer: newServer(addr),
		data:       make(map[string]GlobalData),
		lock:       &sync.RWMutex{},
	}
}

// SetTplDelims 设置模板的左右边界符。不设置默认为 {{ 和 }}。
//
// Example:
//
//	httpsvr.SetTplDelims("<%{", "}%>")
func (s *EasyServer) SetTplDelims(left, right string) {
	SetTplDelims(left, right)
}

func (s *EasyServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// 惰性初始化：确保中间件链（路由、CORS、静态文件等）已组装。
	// 通过 sync.Once 保证 ListenAndServe 和直接调用 ServeHTTP 都安全。
	s.initOnce.Do(func() { s.listenPrepare() })

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

// Shutdown 优雅停机：停止接收新连接，等待在途请求完成。
// 委托 http.Server.Shutdown，ctx 用于控制等待超时。
func (s *EasyServer) Shutdown(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}

// Close 立即关闭服务器，不等待在途请求完成。
func (s *EasyServer) Close() error {
	return s.httpServer.Close()
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
	// 幂等守卫：中间件链已组装过则直接返回，避免 ListenAndServe 与
	// ServeHTTP(initOnce) 双重触发导致重复组装、handler 执行两次、响应双写。
	if len(s.middles) > 0 {
		return
	}
	// if len(s.middles) == 0 {
	// 	s.middles = GetDefaultMiddlewareList()
	// }
	if len(s.routingList) == 0 {
		log.Printf("----Warn!!!--routingList未设置。请使用AppendRouting或AddHandler方法添加路由-----\n")
		// s.routingList = GetDefaultRoutingList()
	}

	// 前置中间件：包含静态资源设置，CORS跨域，处理用户验证等前置组件
	if len(s.headMiddles) > 0 {
		s.appendMiddleware(s.headMiddles...)
	}

	// 路由中间件。处理业务主逻辑。
	if len(s.routingList) > 0 {
		s.appendMiddleware(NewMiddleRouter(s.routingList))
	}

	// 后置中间件：包含耗时统计等一些收尾工作。
	if len(s.tailMiddles) > 0 {
		s.appendMiddleware(s.tailMiddles...)
	}

	for i, m := range s.middles {
		log.Printf("---[%d]--EnableMiddleware(%T)--\n", i, m)
	}
	for i, r := range s.routingList {
		log.Printf("---[%d]--RoutePath(%s)---Methods(%+s)--\n", i, r.Path, r.Methods)
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
