package httpsvr

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"time"
)

type Context struct {
	Writer   http.ResponseWriter
	Request  *http.Request
	Server   *EasyServer
	DataFlow *DataFlow
}

type GlobalData struct {
	Key        string
	Rewritable bool
	Value      interface{}
	CreatedAt  time.Time
}

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

func (s *EasyServer) SetData(k string, v interface{}) error {
	if s.data == nil {
		s.data = make(map[string]GlobalData)
	}
	vv, ok := s.data[k]
	if ok && !vv.Rewritable {
		// 已存在数据不可被重写覆盖
		return fmt.Errorf("the data with key:%s could not rewritable", k)
	}
	s.data[k] = GlobalData{Key: k, Value: v, CreatedAt: time.Now(), Rewritable: true}
	return nil
}

func (s *EasyServer) SetDataReadonly(k string, v interface{}) error {
	if s.data == nil {
		s.data = make(map[string]GlobalData)
	}
	vv, ok := s.data[k]
	if ok && !vv.Rewritable {
		// 已存在数据不可被重写覆盖
		return fmt.Errorf("the data with key:%s could not rewritable", k)
	}
	s.data[k] = GlobalData{Key: k, Value: v, CreatedAt: time.Now(), Rewritable: false}
	return nil
}

func (s *EasyServer) GetData(k string) GlobalData {
	if s.data == nil {
		return GlobalData{}
	}
	v, ok := s.data[k]
	if ok {
		return v
	}
	return GlobalData{}
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

func (s *EasyServer) AddMiddleHead(middle MiddleHandle) {
	s.headMiddles = append(s.headMiddles, middle)
}

func (s *EasyServer) AddMiddleTail(middle MiddleHandle) {
	s.tailMiddles = append(s.tailMiddles, middle)
}

func (s *EasyServer) addMiddleware(middles ...MiddleHandle) {
	s.middles = append(s.middles, middles...)
}

func (s *EasyServer) AddRouting(routing Routing) {
	s.routingList = append(s.routingList, routing)
}

func (s *EasyServer) AddHandler(method, urlpath string, ctxfunc func(ctx Context)) {
	handler := func(w http.ResponseWriter, r *http.Request, dataFlow *DataFlow) {
		hctx := Context{Writer: w, Request: r, Server: s, DataFlow: dataFlow}
		ctxfunc(hctx)
	}
	routing := Routing{Methods: []string{method}, Path: urlpath, handler: handler}
	s.AddRouting(routing)
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
