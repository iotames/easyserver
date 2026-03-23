package tcpsvr

import (
	"fmt"
	"net"
	"time"

	"github.com/iotames/easyserver/log"
)

// Handler 当前链接的业务
func Handler(s *Server, conn net.Conn) {
	u := NewUser(conn)
	remoteAddr := conn.RemoteAddr().String()

	u.SetOnConnectStart(func(uu User) {
		s.Lock()
		log.Debug("TCP连接建立成功", "remoteAddr", remoteAddr)
		_, ok := s.usermap[remoteAddr]
		if !ok {
			s.usermap[remoteAddr] = u
			log.Debug("用户已上线", "remoteAddr", remoteAddr)
		}
		s.Unlock()
	})
	u.SetOnConnectLost(func(uu User) {
		s.Lock()
		_, ok := s.usermap[remoteAddr]
		if ok {
			delete(s.usermap, remoteAddr)
			log.Debug("用户已离线", "remoteAddr", remoteAddr)
		}
		log.Debug("TCP连接断开", "remoteAddr", remoteAddr)
		s.Unlock()
	})
	u.ConnectStart()

	//接受客户端发送的消息
	go func() {
		for {
			err := MainHandler(s, u)
			if err != nil {
				if err.Error() == ERR_CONNECT_LOST {
					return
				}
				if !u.IsClosed {
					u.Close()
					fmt.Println("--conn-Closed--After--MainHandler--error:", err)
				}
				return
			}
			if !u.IsClosed {
				//用户的任意消息，代表当前用户是一个活跃的
				u.KeepActive()
			}
		}
	}()

	//当前handler阻塞
	for {
		select {
		case <-u.GetActiveChannel():
			//当前用户是活跃的，应该重置定时器
			//不做任何事情，为了激活select，更新下面的定时器

		case <-time.After(time.Second * time.Duration(s.DropAfter)):
			//已经超时
			//将当前的User强制的关闭
			if !u.IsClosed {
				u.Close()
			}
			//退出当前Handler
			return //runtime.Goexit()
		}
	}
}

// 用户处理消息的业务 Request
func MainHandler(s *Server, u *User) error {
	// 通过命令行读取的消息data, 有换行符，转为字符串值为: string(data[:len(data)-1])

	data, err := u.GetConnData()
	if err != nil {
		return err
	}

	// // 数据过滤
	// lendata := len(data)
	// if lendata < s.filterDataLen {
	// 	err = fmt.Errorf("req data too small")
	// 	log.Warn("handler.MainHandler--error", "error", err)
	// 	return err
	// }
	isHttp := u.IsHttp(data)
	msgCount := u.MsgCount()

	// dp := GetDataPack()
	if isHttp && msgCount == 1 {
		// HTTP API 接口业务处理。不支持HTTP 的 Keep-Alive
		req := NewRequest(u.GetConn())
		req.SetRawData(data)
		err = req.ParseHttp()
		if err != nil {
			log.Warn("ParseHttpError", "error", err, "RequestRAW", string(data))
			return err
		}
		if req.IsWebSocket() {
			// websocket 握手
			// dp.SetProtocol(PROTOCOL_WEBSOCKET)
			u.SetProtocol(PROTOCOL_WEBSOCKET)
			return req.ResponseWebSocket()
		}
		// err = HttpHandler(req)
		// if err != nil {
		// 	logger.Debug("---handler.MainHandler--HttpHandler--error:", err)
		// 	return err
		// }
		// HTTP 一次请求响应后，立即关闭连接。不支持HTTP 的 Keep-Alive
		return u.Close()
	}
	log.Debug("---TCP---ReceivedMessage--SUCCESS-", "msgCount", u.MsgCount())

	return s.HandlerMsg(u, data)
}
