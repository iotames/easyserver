package easyserver_test

import (
	"encoding/binary"
	"net"
	"strconv"
	"strings"
	"testing"

	"github.com/iotames/easyserver/tcpsvr"
)

// 测试 WebSocket 帧封包（文本帧、二进制帧、自定义 opcode）
func TestWebSocketPack(t *testing.T) {
	t.Run("文本帧-短载荷", func(t *testing.T) {
		data := []byte("Hello")
		frame := tcpsvr.WebSocketPack(data)

		// FIN=1, opcode=0x1(文本)
		if frame[0] != 0x81 {
			t.Errorf("首字节应为 0x81(文本帧), 实际 0x%02x", frame[0])
		}
		// 长度 5 ≤ 125
		if frame[1] != 5 {
			t.Errorf("载荷长度应为 5, 实际 %d", frame[1])
		}
		if string(frame[2:]) != "Hello" {
			t.Errorf("载荷应为 'Hello', 实际 '%s'", string(frame[2:]))
		}
	})

	t.Run("文本帧-中等载荷(126~65535)", func(t *testing.T) {
		data := make([]byte, 200)
		for i := range data {
			data[i] = byte(i)
		}
		frame := tcpsvr.WebSocketPack(data)

		if frame[0] != 0x81 {
			t.Errorf("首字节应为 0x81, 实际 0x%02x", frame[0])
		}
		if frame[1] != 126 {
			t.Errorf("第2字节应为 126(扩展长度), 实际 %d", frame[1])
		}
		wantLen := uint16(len(data))
		gotLen := binary.BigEndian.Uint16(frame[2:4])
		if gotLen != wantLen {
			t.Errorf("扩展长度应为 %d, 实际 %d", wantLen, gotLen)
		}
	})

	t.Run("文本帧-大载荷(>65535)", func(t *testing.T) {
		data := make([]byte, 70000)
		frame := tcpsvr.WebSocketPack(data)

		if frame[0] != 0x81 {
			t.Errorf("首字节应为 0x81, 实际 0x%02x", frame[0])
		}
		if frame[1] != 127 {
			t.Errorf("第2字节应为 127(8字节扩展长度), 实际 %d", frame[1])
		}
		wantLen := uint64(len(data))
		gotLen := binary.BigEndian.Uint64(frame[2:10])
		if gotLen != wantLen {
			t.Errorf("扩展长度应为 %d, 实际 %d", wantLen, gotLen)
		}
	})

	t.Run("二进制帧", func(t *testing.T) {
		data := []byte{0x00, 0x01, 0x02}
		frame := tcpsvr.WebSocketPackBinary(data)

		if frame[0] != 0x82 {
			t.Errorf("首字节应为 0x82(二进制帧), 实际 0x%02x", frame[0])
		}
		if frame[1] != 3 {
			t.Errorf("载荷长度应3, 实际 %d", frame[1])
		}
	})

	t.Run("自定义 opcode", func(t *testing.T) {
		data := []byte("ping")
		frame := tcpsvr.WebSocketPackWithOpcode(data, 0x9) // ping(0x9)

		if frame[0] != 0x89 {
			t.Errorf("首字节应为 0x89(ping), 实际 0x%02x", frame[0])
		}
	})
}

// 模拟客户端掩码：构造带掩码的 WebSocket 帧
func maskClientFrame(payload []byte, opcode byte) []byte {
	maskKey := []byte{0x12, 0x34, 0x56, 0x78}
	masked := make([]byte, len(payload))
	for i, b := range payload {
		masked[i] = b ^ maskKey[i%4]
	}

	length := len(payload)
	var header []byte
	if length <= 125 {
		header = []byte{0x80 | opcode, byte(length) | 0x80}
		header = append(header, maskKey...)
		header = append(header, masked...)
	} else if length <= 65535 {
		header = []byte{0x80 | opcode, 126 | 0x80}
		ext := make([]byte, 2)
		binary.BigEndian.PutUint16(ext, uint16(length))
		header = append(header, ext...)
		header = append(header, maskKey...)
		header = append(header, masked...)
	} else {
		header = []byte{0x80 | opcode, 127 | 0x80}
		ext := make([]byte, 8)
		binary.BigEndian.PutUint64(ext, uint64(length))
		header = append(header, ext...)
		header = append(header, maskKey...)
		header = append(header, masked...)
	}
	return header
}

// 测试 WebSocket 帧解包（客户端掩码帧）
func TestWebSocketUnpack(t *testing.T) {
	t.Run("文本帧-短载荷", func(t *testing.T) {
		payload := []byte("Hello EasyServer")
		frame := maskClientFrame(payload, 0x1)

		got, opcode, err := tcpsvr.WebSocketUnpack(frame)
		if err != nil {
			t.Fatalf("解包失败: %v", err)
		}
		if opcode != 0x1 {
			t.Errorf("opcode应为 0x1, 实际 0x%02x", opcode)
		}
		if string(got) != string(payload) {
			t.Errorf("载荷应为 '%s', 实际 '%s'", string(payload), string(got))
		}
	})

	t.Run("二进制帧-中等载荷", func(t *testing.T) {
		payload := make([]byte, 300)
		for i := range payload {
			payload[i] = byte(i & 0xFF)
		}
		frame := maskClientFrame(payload, 0x2)

		got, opcode, err := tcpsvr.WebSocketUnpack(frame)
		if err != nil {
			t.Fatalf("解包失败: %v", err)
		}
		if opcode != 0x2 {
			t.Errorf("opcode应为 0x2, 实际 0x%02x", opcode)
		}
		if len(got) != len(payload) {
			t.Fatalf("载荷长度应为 %d, 实际 %d", len(payload), len(got))
		}
		for i := range payload {
			if got[i] != payload[i] {
				t.Errorf("byte[%d] 应为 %d, 实际 %d", i, payload[i], got[i])
				break
			}
		}
	})

	t.Run("大载荷(>65535)", func(t *testing.T) {
		payload := make([]byte, 70000)
		for i := range payload {
			payload[i] = byte(i & 0xFF)
		}
		frame := maskClientFrame(payload, 0x2)

		got, opcode, err := tcpsvr.WebSocketUnpack(frame)
		if err != nil {
			t.Fatalf("解包失败: %v", err)
		}
		if opcode != 0x2 {
			t.Errorf("opcode应为 0x2, 实际 0x%02x", opcode)
		}
		if len(got) != len(payload) {
			t.Fatalf("载荷长度应为 %d, 实际 %d", len(payload), len(got))
		}
	})

	t.Run("帧过短返回错误", func(t *testing.T) {
		_, _, err := tcpsvr.WebSocketUnpack([]byte{0x80})
		if err == nil {
			t.Fatal("期望帧过短错误, 实际无错误")
		}
	})

	t.Run("未掩码的客户端帧返回错误", func(t *testing.T) {
		// 服务端帧：FIN=1, opcode=0x1, 无掩码
		frame := []byte{0x81, 0x05, 'H', 'e', 'l', 'l', 'o'}
		_, _, err := tcpsvr.WebSocketUnpack(frame)
		if err == nil {
			t.Fatal("期望客户端帧必须带掩码错误, 实际无错误")
		}
	})
}

// 测试 User 的基本行为（使用 net.Pipe）
func TestTCPUser(t *testing.T) {
	t.Run("新建用户", func(t *testing.T) {
		pipe1, _ := net.Pipe()
		defer pipe1.Close()

		u := tcpsvr.NewUser(pipe1)
		if u == nil {
			t.Fatal("NewUser 返回 nil")
		}
	})

	t.Run("IsHttp 检测-HTTP请求", func(t *testing.T) {
		pipe1, pipe2 := net.Pipe()
		defer pipe1.Close()
		defer pipe2.Close()

		u := tcpsvr.NewUser(pipe1)

		httpData := []byte("GET / HTTP/1.1\r\nHost: test\r\n\r\n")
		if !u.IsHttp(httpData) {
			t.Error("HTTP GET 应该被识别为 HTTP")
		}
	})

	t.Run("IsHttp 检测-非HTTP数据", func(t *testing.T) {
		pipe1, pipe2 := net.Pipe()
		defer pipe1.Close()
		defer pipe2.Close()

		u := tcpsvr.NewUser(pipe1)

		nonHttp := []byte{0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06}
		if u.IsHttp(nonHttp) {
			t.Error("二进制数据不应该被识别为 HTTP")
		}
	})

	t.Run("IsHttp 检测-POST请求", func(t *testing.T) {
		pipe1, pipe2 := net.Pipe()
		defer pipe1.Close()
		defer pipe2.Close()

		u := tcpsvr.NewUser(pipe1)

		httpData := []byte("POST /api/data HTTP/1.1\r\nHost: test\r\n\r\nbody")
		if !u.IsHttp(httpData) {
			t.Error("HTTP POST 应该被识别为 HTTP")
		}
	})

	t.Run("WebSocket 协议标记", func(t *testing.T) {
		pipe1, _ := net.Pipe()
		defer pipe1.Close()

		u := tcpsvr.NewUser(pipe1)
		if u.IsWebSocket() {
			t.Error("新用户默认不应该为 WebSocket 协议")
		}

		u.SetProtocol(tcpsvr.PROTOCOL_WEBSOCKET)
		if !u.IsWebSocket() {
			t.Error("设置协议后应该识别为 WebSocket")
		}
	})

	t.Run("消息计数", func(t *testing.T) {
		pipe1, _ := net.Pipe()
		defer pipe1.Close()

		u := tcpsvr.NewUser(pipe1)
		if u.MsgCount() != 0 {
			t.Errorf("新用户消息计数应为 0, 实际 %d", u.MsgCount())
		}
	})

	t.Run("SendData 和 ReceiveDataToSend 双向通信", func(t *testing.T) {
		pipe1, pipe2 := net.Pipe()
		defer pipe1.Close()
		defer pipe2.Close()

		u := tcpsvr.NewUser(pipe1)

		// 通过 ReceiveDataToSend 发送数据 -> channel -> ListenMessage -> SendData -> pipe1.Write -> pipe2 可读
		sendMsg := []byte("Hello from server")
		u.ReceiveDataToSend(sendMsg)

		// 从 pipe2 读取数据
		buf := make([]byte, 1024)
		n, err := pipe2.Read(buf)
		if err != nil {
			t.Fatalf("读取数据失败: %v", err)
		}
		got := string(buf[:n])
		if got != string(sendMsg) {
			t.Errorf("期望 '%s', 实际 '%s'", string(sendMsg), got)
		}
	})

	t.Run("关闭连接", func(t *testing.T) {
		pipe1, _ := net.Pipe()
		u := tcpsvr.NewUser(pipe1)

		err := u.Close()
		if err != nil {
			t.Errorf("关闭连接失败: %v", err)
		}
		if !u.IsClosed {
			t.Error("关闭后 IsClosed 应为 true")
		}
	})
}

// 测试 TCP Server 的创建和单例行为
func TestTCPServer(t *testing.T) {
	t.Run("NewServer 基本参数", func(t *testing.T) {
		s := tcpsvr.NewServer("127.0.0.1:9999", 60)
		if s == nil {
			t.Fatal("NewServer 返回 nil")
		}
		if s.DropAfter != 60 {
			t.Errorf("DropAfter 应为 60, 实际 %d", s.DropAfter)
		}
	})

	t.Run("DropAfter 默认值", func(t *testing.T) {
		s := tcpsvr.NewServer(":0", 0)
		if s.DropAfter != 300 {
			t.Errorf("DropAfter 应为 300(默认值), 实际 %d", s.DropAfter)
		}
	})

	t.Run("GetServer 单例", func(t *testing.T) {
		s1 := tcpsvr.NewServer("127.0.0.1:9998", 30)
		s2 := tcpsvr.GetServer()
		if s1 != s2 {
			t.Error("GetServer 应返回最后一次 NewServer 创建的实例")
		}
	})

	t.Run("空连接时 GetConns 为空", func(t *testing.T) {
		s := tcpsvr.NewServer("127.0.0.1:9997", 30)
		conns := s.GetConns()
		if len(conns) != 0 {
			t.Errorf("无连接时 GetConns 应为空, 实际 %d", len(conns))
		}
	})

	t.Run("空连接时 GetOutputWriters 为空", func(t *testing.T) {
		s := tcpsvr.NewServer("127.0.0.1:9996", 30)
		writers := s.GetOutputWriters()
		if len(writers) != 0 {
			t.Errorf("无连接时 GetOutputWriters 应为空, 实际 %d", len(writers))
		}
	})
}

// 测试 HTTP 请求解析和 WebSocket 握手检测
func TestTCPRequest(t *testing.T) {
	t.Run("解析普通 HTTP GET", func(t *testing.T) {
		pipe1, _ := net.Pipe()
		defer pipe1.Close()

		r := tcpsvr.NewRequest(pipe1)
		raw := []byte("GET /hello HTTP/1.1\r\nHost: example.com\r\n\r\n")
		r.SetRawData(raw)

		err := r.ParseHttp()
		if err != nil {
			t.Fatalf("ParseHttp 失败: %v", err)
		}

		req := r.GetHttpRequest()
		if req == nil {
			t.Fatal("httpRequest 为 nil")
		}
		if req.Method != "GET" {
			t.Errorf("Method 应为 GET, 实际 '%s'", req.Method)
		}
		body := r.GetHttpBody()
		if len(body) != 0 {
			t.Errorf("GET 请求的 body 应为空, 实际长度 %d", len(body))
		}
	})

	t.Run("解析 HTTP POST 带 body", func(t *testing.T) {
		pipe1, _ := net.Pipe()
		defer pipe1.Close()

		r := tcpsvr.NewRequest(pipe1)
		bodyStr := `{"name":"test"}`
		raw := []byte("POST /api/data HTTP/1.1\r\nHost: example.com\r\nContent-Type: application/json\r\nContent-Length: " +
			strconv.Itoa(len(bodyStr)) + "\r\n\r\n" + bodyStr)
		r.SetRawData(raw)

		err := r.ParseHttp()
		if err != nil {
			t.Fatalf("ParseHttp 失败: %v", err)
		}

		if r.GetHttpRequest().Method != "POST" {
			t.Errorf("Method 应为 POST, 实际 '%s'", r.GetHttpRequest().Method)
		}
		body := r.GetHttpBody()
		if string(body) != bodyStr {
			t.Errorf("Body 应为 '%s', 实际 '%s'", bodyStr, string(body))
		}
	})

	t.Run("检测非 WebSocket 请求", func(t *testing.T) {
		pipe1, _ := net.Pipe()
		defer pipe1.Close()

		r := tcpsvr.NewRequest(pipe1)
		raw := []byte("GET /hello HTTP/1.1\r\nHost: example.com\r\n\r\n")
		r.SetRawData(raw)
		r.ParseHttp()

		if r.IsWebSocket() {
			t.Error("普通 GET 请求不应被识别为 WebSocket")
		}
	})

	t.Run("检测 WebSocket 升级请求", func(t *testing.T) {
		pipe1, _ := net.Pipe()
		defer pipe1.Close()

		r := tcpsvr.NewRequest(pipe1)
		raw := []byte("GET /ws HTTP/1.1\r\nHost: example.com\r\nConnection: Upgrade\r\nUpgrade: websocket\r\nSec-WebSocket-Key: dGhlIHNhbXBsZSBub25jZQ==\r\nSec-WebSocket-Version: 13\r\n\r\n")
		r.SetRawData(raw)
		r.ParseHttp()

		if !r.IsWebSocket() {
			t.Error("应识别为 WebSocket 升级请求")
		}
	})

	t.Run("WebSocket 握手响应", func(t *testing.T) {
		pipe1, pipe2 := net.Pipe()
		defer pipe1.Close()
		defer pipe2.Close()

		r := tcpsvr.NewRequest(pipe1)
		raw := []byte("GET /ws HTTP/1.1\r\nHost: example.com\r\nConnection: Upgrade\r\nUpgrade: websocket\r\nSec-WebSocket-Key: dGhlIHNhbXBsZSBub25jZQ==\r\nSec-WebSocket-Version: 13\r\n\r\n")
		r.SetRawData(raw)
		r.ParseHttp()

		// 先启动读 goroutine，避免 net.Pipe 同步阻塞
		var resp string
		done := make(chan error, 1)
		go func() {
			buf := make([]byte, 1024)
			n, err := pipe2.Read(buf)
			if err != nil {
				done <- err
				return
			}
			resp = string(buf[:n])
			done <- nil
		}()

		err := r.ResponseWebSocket()
		if err != nil {
			t.Fatalf("ResponseWebSocket 失败: %v", err)
		}

		if err := <-done; err != nil {
			t.Fatalf("读取握手响应失败: %v", err)
		}
		if !strings.Contains(resp, "101") {
			t.Errorf("响应应包含状态码 101, 实际 '%s'", resp)
		}
		if !strings.Contains(resp, "Upgrade: websocket") {
			t.Errorf("响应应包含 Upgrade: websocket, 实际 '%s'", resp)
		}
		if !strings.Contains(resp, "Sec-WebSocket-Accept:") {
			t.Errorf("响应应包含 Sec-WebSocket-Accept, 实际 '%s'", resp)
		}
	})
}

// 测试 SendMsg 和 SendBinaryMsg
func TestServerSendMsg(t *testing.T) {
	t.Run("SendMsg 发送文本帧", func(t *testing.T) {
		pipe1, pipe2 := net.Pipe()
		defer pipe1.Close()
		defer pipe2.Close()

		s := tcpsvr.NewServer(":0", 30)
		u := tcpsvr.NewUser(pipe1)

		err := s.SendMsg(u, []byte("Hello"))
		if err != nil {
			t.Fatalf("SendMsg 失败: %v", err)
		}

		buf := make([]byte, 1024)
		n, err := pipe2.Read(buf)
		if err != nil {
			t.Fatalf("读取发送数据失败: %v", err)
		}

		// 验证是 WebSocket 帧（首字节 0x81=文本帧）
		if buf[0] != 0x81 {
			t.Errorf("首字节应为 0x81(文本帧), 实际 0x%02x", buf[0])
		}
		t.Logf("收到 SendMsg 数据: %x", buf[:n])
	})

	t.Run("SendBinaryMsg 发送二进制帧", func(t *testing.T) {
		pipe1, pipe2 := net.Pipe()
		defer pipe1.Close()
		defer pipe2.Close()

		s := tcpsvr.NewServer(":0", 30)
		u := tcpsvr.NewUser(pipe1)

		data := []byte{0xDE, 0xAD, 0xBE, 0xEF}
		err := s.SendBinaryMsg(u, data)
		if err != nil {
			t.Fatalf("SendBinaryMsg 失败: %v", err)
		}

		buf := make([]byte, 1024)
		n, err := pipe2.Read(buf)
		if err != nil {
			t.Fatalf("读取二进制数据失败: %v", err)
		}

		if buf[0] != 0x82 {
			t.Errorf("首字节应为 0x82(二进制帧), 实际 0x%02x", buf[0])
		}
		t.Logf("收到 SendBinaryMsg 数据: %x", buf[:n])
	})
}
