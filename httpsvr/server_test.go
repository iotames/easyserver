package httpsvr_test

import (
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/iotames/easyserver/httpsvr"
)

// newCountServer 构造一个带原子计数器的测试服务器，GET /ping 返回 "pong"。
func newCountServer(t *testing.T, addr string, count *atomic.Int64) *httpsvr.EasyServer {
	t.Helper()
	s := httpsvr.NewEasyServer(addr)
	s.AddHandler("GET", "/ping", func(ctx httpsvr.Context) {
		count.Add(1)
		ctx.Text("pong", http.StatusOK)
	})
	return s
}

// checkSingleBody 断言单个请求的响应体恰好为 expected（无拼接重复）。
func checkSingleBody(t *testing.T, body string, expected string) {
	t.Helper()
	if body != expected {
		t.Fatalf("响应体应为 %q, 实际得到 %q (可能被重复写入)", expected, body)
	}
}

// TestListenPrepareIdempotent 回归测试：ListenAndServe 之后第一个请求的 handler
// 必须只执行一次，响应体不得双写（修复 listenPrepare 非幂等导致的重复组装）。
func TestListenPrepareIdempotent(t *testing.T) {
	var count atomic.Int64

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("监听端口失败: %v", err)
	}
	addr := ln.Addr().String()
	ln.Close()

	// 通过真实端口走 ListenAndServe 路径（此路径先组装一次，
	// 第一个请求触发 initOnce 再组装一次，正是双执行 bug 的根因场景）。
	srv := newCountServer(t, addr, &count)
	go srv.ListenAndServe()
	defer srv.Close()

	// 等待服务器就绪
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", addr, 200*time.Millisecond)
		if err == nil {
			conn.Close()
			break
		}
		time.Sleep(20 * time.Millisecond)
	}

	for i := 0; i < 3; i++ {
		resp, err := http.Get("http://" + addr + "/ping")
		if err != nil {
			t.Fatalf("第 %d 个请求失败: %v", i+1, err)
		}
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			t.Fatalf("第 %d 个请求读取响应失败: %v", i+1, err)
		}
		checkSingleBody(t, string(body), "pong")
	}

	if got := count.Load(); got != 3 {
		t.Fatalf("handler 应执行 3 次(每个请求恰好一次), 实际执行 %d 次", got)
	}
}

// TestServeHTTPOnceViaInitOnce 回归测试：不调用 ListenAndServe，
// 直接以 httptest 触发 ServeHTTP(initOnce 路径)，handler 只执行一次。
func TestServeHTTPOnceViaInitOnce(t *testing.T) {
	var count atomic.Int64
	s := newCountServer(t, ":0", &count)

	ts := httptest.NewServer(s)
	defer ts.Close()

	for i := 0; i < 3; i++ {
		resp, err := http.Get(ts.URL + "/ping")
		if err != nil {
			t.Fatalf("第 %d 个请求失败: %v", i+1, err)
		}
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			t.Fatalf("第 %d 个请求读取响应失败: %v", i+1, err)
		}
		checkSingleBody(t, string(body), "pong")
	}

	if got := count.Load(); got != 3 {
		t.Fatalf("handler 应执行 3 次(每个请求恰好一次), 实际执行 %d 次", got)
	}
}

// TestServeHTTPDirectRecorder 直接调用 s.ServeHTTP(Recorder)，不走 ListenAndServe。
func TestServeHTTPDirectRecorder(t *testing.T) {
	var count atomic.Int64
	s := newCountServer(t, ":0", &count)

	req := httptest.NewRequest("GET", "/ping", nil)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)

	checkSingleBody(t, w.Body.String(), "pong")
	if got := count.Load(); got != 1 {
		t.Fatalf("handler 应执行 1 次, 实际执行 %d 次", got)
	}
}
