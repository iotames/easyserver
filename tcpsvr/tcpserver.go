package tcpsvr

import (
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"github.com/iotames/easyserver/log"
)

type Server struct {
	addr      string
	DropAfter int
	// filterDataLen int
	usermap map[string]*User
	// 操作字典时，要加锁
	lock sync.RWMutex
}

var svr *Server

func GetServer() *Server {
	return svr
}

// 创建一个server的接口
func NewServer(addr string, dropAfterSec int) *Server {
	if dropAfterSec == 0 {
		dropAfterSec = 300
	}
	server := &Server{
		addr: addr,
		// filterDataLen: filterDataLen,
		DropAfter: dropAfterSec,
		usermap:   make(map[string]*User, 10),
	}
	svr = server
	return server
}

func (s *Server) GetConns() []net.Conn {
	var conns []net.Conn
	s.Lock()
	for _, u := range s.usermap {
		conns = append(conns, u.GetConn())
	}
	s.Unlock()
	return conns
}

// GetOutputWriters 返回所有当前连接的 writer，WebSocket 连接自动包装，普通 TCP 也统一通过 channel 发送
func (s *Server) GetOutputWriters() []io.Writer {
	s.Lock()
	defer s.Unlock()

	var writers []io.Writer
	for _, u := range s.usermap {
		if u.IsWebSocket() {
			writers = append(writers, &webSocketWriter{user: u})
		} else {
			writers = append(writers, &rawTCPWriter{user: u})
		}
	}
	return writers
}

func (s *Server) Lock() {
	s.lock.Lock()
}

func (s *Server) Unlock() {
	s.lock.Unlock()
}

// 启动服务器的接口
func (s *Server) Run() error {
	//socket listen
	fmt.Printf("[START] TCP Server. listenner at Addr: %s, Starting\n", s.addr)
	listener, err := net.Listen("tcp", s.addr)
	if err != nil {
		return fmt.Errorf("net.Listen err(%v)", err)
	}
	//close listen socket
	defer listener.Close()

	closeListener := false
	for {
		go func() {
			<-StopSignChan
			fmt.Println("程序将在5秒后退出")
			time.Sleep(time.Second * 5)
			closeListener = true
		}()
		if closeListener {
			break
		}
		//accept
		conn, err := listener.Accept()
		if err != nil {
			// TODO error
			fmt.Println("listener accept err:", err)
			continue
		}

		//do handler
		go Handler(s, conn)
	}
	fmt.Println("程序已退出")
	return nil
}

func (s *Server) SendMsg(u IClient, data []byte) error {
	data = WebSocketPack(data)
	u.ReceiveDataToSend(data)
	return nil
}

// FirstMsg 处理首条消息(心跳消息)
func (s *Server) FirstMsg(u IClient, data []byte) error {
	// 建立连接后发送的第一条消息必须为心跳事件消息
	// 处理首次心跳消息，上线用户
	log.Debug("--FirstMsg--", "MsgCount", u.MsgCount())
	return s.UserOnline(u, data)
}

func (s *Server) HandlerMsg(u IClient, data []byte) error {

	// 在线调试 http://www.websocket-test.com/, https://websocketking.com/
	var err error
	msgCount := u.MsgCount()

	log.Debug("-TCP---Server.HandlerMsg-", "IsWebSocket", u.IsWebSocket(), "MsgCount", u.MsgCount())

	if (u.IsWebSocket() && msgCount == 2) || (!u.IsWebSocket() && msgCount == 1) {
		// 连接建立后客户端主动发送一个心跳事件消息
		return s.FirstMsg(u, data)
	}

	// 解包 WebSocket 帧，获取 payload 和 opcode
	payload, opcode, err := WebSocketUnpack(data)
	if err != nil {
		log.Warn("unpackWebsocket err", "err", err)
		return err
	}

	addr := u.GetConn().RemoteAddr().String()
	fmt.Printf("----Server.HandlerMsg--addr(%s)--msg(%s)--\n", addr, string(data))

	// uu := s.addrToUser[addr]
	// // 根据access_token进行用户身份鉴权
	// b, err := s.checkToken(uu, u, &msg)
	// if !b {
	// 	return err
	// }

	// // TODO 对未上线的发送对象，保存离线消息，下次上线时发送

	switch opcode {
	case 0x1: // 文本帧
		fmt.Printf("----Server.HandlerMsg--addr(%s)--text msg(%s)--\n", addr, string(payload))
		newmsg := fmt.Sprintf("client(%s) say: %s", addr, string(payload))
		// 回显文本帧
		return s.SendMsg(u, []byte(newmsg))

	case 0x2: // 二进制帧
		fmt.Printf("----Server.HandlerMsg--addr(%s)--binary msg len=%d--\n", addr, len(payload))
		// 回映二进制数据，使用二进制帧
		return s.SendBinaryMsg(u, payload)

	default:
		// 其他 opcode（如 ping/pong）可忽略或关闭连接
		log.Warn("unsupported", "opcode", opcode)
		return nil
	}
}

// SendBinaryMsg 发送二进制帧给客户端
func (s *Server) SendBinaryMsg(u IClient, data []byte) error {
	framed := WebSocketPackBinary(data) // 使用二进制封帧
	u.ReceiveDataToSend(framed)
	return nil
}

func (s *Server) UserOnline(u IClient, data []byte) error {
	addr := u.GetConn().RemoteAddr().String()
	log.Debug("UserOnline", "addr", addr)
	// uu := NewUser(msg.FromUserId)
	// b, err := s.checkToken(uu, u, msg)
	// if !b {
	// 	return err
	// }

	semddata := "SUCCESS"
	// msg.Content = "SUCCESS"
	// msg.ToUserId = msg.FromUserId
	// msg.FromUserId = MSG_KEEP_ALIVE
	// TODO 读取离线消息，有则发送
	return s.SendMsg(u, []byte(semddata))
}

// func (s *Server) UserOffline(addr string) {
// 	s.Lock()
// 	fmt.Println("用户已离线", addr)
// 	s.Unlock()
// }

var StopSignChan chan string = make(chan string)

// // HTTP 请求处理
// //
// //	POST /api/local/stop
// func closeListener(req *Request, resp *Response) Response {
// 	remoteAddr := req.RemoteAddr().String()
// 	if strings.Contains(remoteAddr, "127.0.0.1") || strings.Contains(remoteAddr, "::1") {
// 		go func() {
// 			StopSignChan <- "stop"
// 		}()
// 		return resp.Json(ResponseOk("操作成功"))
// 	}
// 	return resp.Json(ResponseFail("仅限内网访问", 400))
// }
