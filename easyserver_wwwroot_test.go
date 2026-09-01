package easyserver_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/iotames/easyserver"
	"github.com/iotames/easyserver/httpsvr"
)

// newWWWRootTestServer 创建带一条路由的测试服务器
func newWWWRootTestServer(t *testing.T) (*easyserver.Server, string) {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "hello.txt"), []byte("hello wwwroot"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "sub"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "sub", "index.html"), []byte("<h1>sub index</h1>"), 0644); err != nil {
		t.Fatal(err)
	}

	s := easyserver.NewServer(":0")
	s.SetQuiet(true)
	s.AddGetHandler("/api/ping", func(ctx easyserver.HttpContext) {
		ctx.Writer.Write([]byte("pong"))
	})
	return s, dir
}

func doReq(s *easyserver.Server, method, target string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, target, nil)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	return w
}

func TestWWWRootFallback(t *testing.T) {
	s, dir := newWWWRootTestServer(t)
	s.SetWWWRoot(dir)

	t.Run("路由命中时不触发兜底", func(t *testing.T) {
		w := doReq(s, "GET", "/api/ping")
		if w.Code != http.StatusOK || w.Body.String() != "pong" {
			t.Errorf("got code=%d body=%q", w.Code, w.Body.String())
		}
	})

	t.Run("路由未命中兜底返回文件", func(t *testing.T) {
		w := doReq(s, "GET", "/hello.txt")
		if w.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", w.Code)
		}
		if got := w.Body.String(); got != "hello wwwroot" {
			t.Errorf("expected file content, got %q", got)
		}
		// MIME 表依环境可能带 charset 后缀（如 text/plain; charset=utf-8），前缀匹配即可
		if ct := w.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/plain") {
			t.Errorf("expected text/plain, got %q", ct)
		}
	})

	t.Run("目录请求兜底index.html", func(t *testing.T) {
		w := doReq(s, "GET", "/sub")
		if w.Code != http.StatusOK || w.Body.String() != "<h1>sub index</h1>" {
			t.Errorf("got code=%d body=%q", w.Code, w.Body.String())
		}
	})

	t.Run("未命中走默认404", func(t *testing.T) {
		w := doReq(s, "GET", "/no-such-file.txt")
		if w.Body.String() == "" {
			t.Error("expected default ResponseNotFound JSON body, got empty")
		}
	})

	t.Run("目录遍历攻击不泄露文件", func(t *testing.T) {
		// 构造 wwwroot 之外的敏感文件，验证 .. 无法逃逸
		outside := filepath.Join(filepath.Dir(dir), "outside_secret.txt")
		if err := os.WriteFile(outside, []byte("secret"), 0644); err != nil {
			t.Fatal(err)
		}
		defer os.Remove(outside)
		w := doReq(s, "GET", "/../outside_secret.txt")
		if w.Body.String() == "secret" {
			t.Error("directory traversal escaped wwwroot")
		}
	})

	t.Run("关闭兜底", func(t *testing.T) {
		s.SetWWWRoot("")
		w := doReq(s, "GET", "/hello.txt")
		if w.Body.String() == "hello wwwroot" {
			t.Error("fallback should be disabled after SetWWWRoot(\"\")")
		}
	})
}

func TestSetNotFoundHandler(t *testing.T) {
	s, dir := newWWWRootTestServer(t)
	s.SetWWWRoot(dir)
	s.SetNotFoundHandler(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("custom 404"))
	})

	t.Run("自定义404生效", func(t *testing.T) {
		w := doReq(s, "GET", "/no-such-file.txt")
		if w.Code != http.StatusNotFound || w.Body.String() != "custom 404" {
			t.Errorf("got code=%d body=%q", w.Code, w.Body.String())
		}
	})

	t.Run("兜底命中时不走自定义404", func(t *testing.T) {
		w := doReq(s, "GET", "/hello.txt")
		if w.Code != http.StatusOK || w.Body.String() != "hello wwwroot" {
			t.Errorf("got code=%d body=%q", w.Code, w.Body.String())
		}
	})
}

// TestWWWRootFallbackGating 兜底触发条件的回归测试：
// 链被中间件 return false 主动中断、或请求方法不是 GET/HEAD 时，不得进入 wwwroot 兜底。
func TestWWWRootFallbackGating(t *testing.T) {
	t.Run("CORS预检OPTIONS保持空响应", func(t *testing.T) {
		s, dir := newWWWRootTestServer(t)
		s.SetWWWRoot(dir)
		s.AddMiddleHead(httpsvr.NewMiddleCORS("*"))

		w := doReq(s, "OPTIONS", "/api/ping")
		if w.Body.Len() != 0 {
			t.Errorf("预检应保持空响应体，got %q", w.Body.String())
		}
		if w.Header().Get("Access-Control-Allow-Origin") != "*" {
			t.Error("CORS 响应头丢失")
		}

		// 预检路径恰好命中 wwwroot 文件时也不能把文件内容当响应体
		w2 := doReq(s, "OPTIONS", "/hello.txt")
		if w2.Body.String() == "hello wwwroot" {
			t.Error("OPTIONS 不应返回 wwwroot 文件内容")
		}
		if w2.Header().Get("Access-Control-Allow-Origin") != "*" {
			t.Error("CORS 响应头丢失")
		}
	})

	t.Run("中间件拦截未写响应时不泄漏兜底文件", func(t *testing.T) {
		s, dir := newWWWRootTestServer(t)
		s.SetWWWRoot(dir)
		s.AddMiddleHead(httpsvr.NewMiddle(func(w http.ResponseWriter, r *http.Request, df *httpsvr.DataFlow) bool {
			return false // 模拟「拦截了请求但忘了写响应」的中间件
		}))
		w := doReq(s, "GET", "/hello.txt")
		if w.Body.String() == "hello wwwroot" {
			t.Fatal("被拦截的请求拿到了 wwwroot 文件")
		}
		if w.Body.Len() != 0 {
			t.Errorf("应维持历史空 200 行为，got %q", w.Body.String())
		}
	})

	t.Run("中间件拦截且已写响应时保持原响应", func(t *testing.T) {
		s, dir := newWWWRootTestServer(t)
		s.SetWWWRoot(dir)
		s.AddMiddleHead(httpsvr.NewMiddle(func(w http.ResponseWriter, r *http.Request, df *httpsvr.DataFlow) bool {
			w.WriteHeader(http.StatusForbidden)
			w.Write([]byte("forbidden"))
			return false
		}))
		w := doReq(s, "GET", "/hello.txt")
		if w.Code != http.StatusForbidden || w.Body.String() != "forbidden" {
			t.Errorf("got code=%d body=%q", w.Code, w.Body.String())
		}
	})

	t.Run("POST不返回静态文件", func(t *testing.T) {
		s, dir := newWWWRootTestServer(t)
		s.SetWWWRoot(dir)
		w := doReq(s, "POST", "/hello.txt")
		if w.Body.String() == "hello wwwroot" {
			t.Error("POST 不应返回静态文件内容")
		}
	})

	t.Run("HEAD命中兜底文件但无响应体", func(t *testing.T) {
		s, dir := newWWWRootTestServer(t)
		s.SetWWWRoot(dir)
		w := doReq(s, "HEAD", "/hello.txt")
		if w.Code != http.StatusOK {
			t.Errorf("HEAD 应命中兜底文件，got %d", w.Code)
		}
		if w.Body.Len() != 0 {
			t.Errorf("HEAD 不应有响应体，got %q", w.Body.String())
		}
	})

	t.Run("Flusher经包装透传且链尾不追加内容", func(t *testing.T) {
		s, dir := newWWWRootTestServer(t)
		s.SetWWWRoot(dir)
		s.AddMiddleHead(httpsvr.NewMiddle(func(w http.ResponseWriter, r *http.Request, df *httpsvr.DataFlow) bool {
			flusher, ok := w.(http.Flusher)
			if !ok {
				t.Error("包装后的 ResponseWriter 应支持 http.Flusher 断言")
				return true
			}
			flusher.Flush()
			return true // 链继续走完，但响应头已随 Flush 提交，链尾不得再写兜底内容
		}))
		w := doReq(s, "GET", "/hello.txt")
		if !w.Flushed {
			t.Error("Flush 应透传到底层 ResponseWriter")
		}
		if w.Body.String() == "hello wwwroot" {
			t.Error("Flush 提交响应后链尾不应再追加兜底文件内容")
		}
	})
}
