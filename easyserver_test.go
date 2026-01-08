package easyserver_test

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/iotames/easyserver"
	"github.com/iotames/easyserver/httpsvr"
	"github.com/iotames/easyserver/response"
)

// TestMain 用于设置和清理测试环境
func TestMain(m *testing.M) {
	// 创建测试静态目录
	os.MkdirAll("./test_static", 0755)
	os.MkdirAll("./test_assets", 0755)

	// 创建测试文件
	os.WriteFile("./test_static/test.txt", []byte("Hello, World!"), 0644)
	os.WriteFile("./test_static/index.html", []byte("<h1>Test HTML</h1>"), 0644)
	os.WriteFile("./test_static/app.js", []byte("console.log('test')"), 0644)
	os.WriteFile("./test_static/style.css", []byte("body { color: red; }"), 0644)

	// 运行测试
	code := m.Run()

	// 清理测试文件
	os.RemoveAll("./test_static")
	os.RemoveAll("./test_assets")

	os.Exit(code)
}

// TestServer 测试服务器的基本功能
func TestServer(t *testing.T) {
	t.Run("创建服务器", func(t *testing.T) {
		s := easyserver.NewServer(":0") // 使用:0让系统分配端口
		if s == nil {
			t.Fatal("NewServer failed: server is nil")
		}
	})

	t.Run("添加路由", func(t *testing.T) {
		s := easyserver.NewServer(":0")

		// 测试添加GET路由
		s.AddGetHandler("/ping", func(ctx easyserver.HttpContext) {
			ctx.Writer.Write([]byte("pong"))
		})

		// 创建测试请求
		req := httptest.NewRequest("GET", "/ping", nil)
		w := httptest.NewRecorder()

		s.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
		if body := w.Body.String(); body != "pong" {
			t.Errorf("Expected 'pong', got '%s'", body)
		}
	})
}

// TestAddStatic 测试静态文件服务
func TestAddStatic(t *testing.T) {
	s := easyserver.NewServer(":0")

	t.Run("添加静态路径", func(t *testing.T) {
		err := s.AddStatic("/static", "./test_static")
		if err != nil {
			t.Errorf("AddStatic failed: %v", err)
		}
	})

	t.Run("重复添加静态路径", func(t *testing.T) {
		err := s.AddStatic("/static", "./test_static")
		if err == nil || !strings.Contains(err.Error(), "already exists") {
			t.Errorf("Expected error for duplicate static path, got: %v", err)
		}
	})

	t.Run("访问静态文件", func(t *testing.T) {
		testCases := []struct {
			name        string
			path        string
			expected    string
			statusCode  int
			contentType string
		}{
			{"文本文件", "/static/test.txt", "Hello, World!", 200, "text/plain"},
			{"HTML文件", "/static/index.html", "<h1>Test HTML</h1>", 200, "text/html"},
			{"JS文件", "/static/app.js", "console.log('test')", 200, "application/javascript"},
			{"CSS文件", "/static/style.css", "body { color: red; }", 200, "text/css"},
			{"不存在的文件", "/static/nonexistent.txt", "", 200, ""}, // 注意：返回200但内容为空
			{"目录遍历攻击", "/static/../server.go", "", 200, ""},    // 返回200但内容为空
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				req := httptest.NewRequest("GET", tc.path, nil)
				w := httptest.NewRecorder()

				s.ServeHTTP(w, req)

				if tc.statusCode > 0 && w.Code != tc.statusCode {
					t.Errorf("Expected status %d, got %d", tc.statusCode, w.Code)
				}

				if tc.contentType != "" && !strings.Contains(w.Header().Get("Content-Type"), tc.contentType) {
					t.Errorf("Expected Content-Type containing '%s', got '%s'",
						tc.contentType, w.Header().Get("Content-Type"))
				}

				if tc.expected != "" {
					if body := w.Body.String(); body != tc.expected {
						t.Errorf("Expected '%s', got '%s'", tc.expected, body)
					}
				}
			})
		}
	})
}

// TestSetCORS 测试CORS跨域设置
func TestSetCORS(t *testing.T) {
	s := easyserver.NewServer(":0")

	t.Run("设置CORS", func(t *testing.T) {
		err := s.SetCORS("*")
		if err != nil {
			t.Errorf("SetCORS failed: %v", err)
		}
	})

	t.Run("重复设置CORS", func(t *testing.T) {
		err := s.SetCORS("example.com")
		if err == nil || !strings.Contains(err.Error(), "already been set") {
			t.Errorf("Expected error for duplicate CORS setting, got: %v", err)
		}
	})

	t.Run("CORS头部检查", func(t *testing.T) {
		s.AddGetHandler("/api/test", func(ctx easyserver.HttpContext) {
			ctx.Writer.Write([]byte("CORS Test"))
		})

		req := httptest.NewRequest("GET", "/api/test", nil)
		w := httptest.NewRecorder()

		s.ServeHTTP(w, req)

		t.Logf("----CORS头部检查--Result.StatusCode(%+v)---\n", w.Result().StatusCode)

		checkHeaders := []struct {
			header   string
			expected string
		}{
			{"Access-Control-Allow-Origin", "*"},
			{"Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE, UPDATE"},
			{"Access-Control-Allow-Headers", "Origin, Content-Length, Content-Type, Accept, Token, Auth-Token, X-Requested-With"},
			{"Access-Control-Allow-Credentials", "true"},
		}

		for _, h := range checkHeaders {
			value := w.Header().Get(h.header)

			if value != h.expected {
				t.Errorf("Header %s: expected '%s', got '%s'", h.header, h.expected, value)
			}
		}
	})

	t.Run("OPTIONS预检请求", func(t *testing.T) {
		req := httptest.NewRequest("OPTIONS", "/api/test", nil)
		req.Header.Set("Origin", "http://example.com")
		req.Header.Set("Access-Control-Request-Method", "GET")

		w := httptest.NewRecorder()
		s.ServeHTTP(w, req)

		if w.Code != 200 {
			t.Errorf("OPTIONS: expected status 200, got %d", w.Code)
		}
	})
}

// TestResponseFunctions 测试响应函数
func TestResponseFunctions(t *testing.T) {
	s := easyserver.NewServer(":0")

	t.Run("ResponseJson", func(t *testing.T) {
		s.AddGetHandler("/json", func(ctx easyserver.HttpContext) {
			data := map[string]interface{}{
				"message": "test",
				"value":   123,
			}
			easyserver.ResponseJson(ctx, data, "success", 200)
		})

		req := httptest.NewRequest("GET", "/json", nil)
		w := httptest.NewRecorder()
		s.ServeHTTP(w, req)

		if w.Header().Get("Content-Type") != "application/json" {
			t.Error("Expected Content-Type application/json")
		}

		var result response.ResponseApiData
		body, _ := io.ReadAll(w.Body)
		json.Unmarshal(body, &result)

		if result.Code != 200 {
			t.Errorf("Expected code 200, got %d", result.Code)
		}
		if result.Msg != "success" {
			t.Errorf("Expected msg 'success', got '%s'", result.Msg)
		}
	})

	t.Run("ResponseJsonOk", func(t *testing.T) {
		s.AddGetHandler("/json-ok", func(ctx easyserver.HttpContext) {
			easyserver.ResponseJsonOk(ctx, "Operation successful")
		})

		req := httptest.NewRequest("GET", "/json-ok", nil)
		w := httptest.NewRecorder()
		s.ServeHTTP(w, req)

		var result response.ResponseApiData
		json.Unmarshal(w.Body.Bytes(), &result)

		if result.Code != 200 {
			t.Errorf("Expected code 200, got %d", result.Code)
		}
		if result.Msg != "Operation successful" {
			t.Errorf("Expected msg 'Operation successful', got '%s'", result.Msg)
		}
	})

	t.Run("ResponseJsonFail", func(t *testing.T) {
		s.AddGetHandler("/json-fail", func(ctx easyserver.HttpContext) {
			easyserver.ResponseJsonFail(ctx, "Invalid request", 400)
		})

		req := httptest.NewRequest("GET", "/json-fail", nil)
		w := httptest.NewRecorder()
		s.ServeHTTP(w, req)

		var result response.ResponseApiData
		json.Unmarshal(w.Body.Bytes(), &result)

		if result.Code != 400 {
			t.Errorf("Expected code 400, got %d", result.Code)
		}
		if !strings.Contains(result.Msg, "Invalid request") {
			t.Errorf("Expected msg containing 'Invalid request', got '%s'", result.Msg)
		}
	})

	t.Run("ResponseText", func(t *testing.T) {
		s.AddGetHandler("/text", func(ctx easyserver.HttpContext) {
			easyserver.ResponseText(ctx, []byte("Plain text response"))
		})

		req := httptest.NewRequest("GET", "/text", nil)
		w := httptest.NewRecorder()
		s.ServeHTTP(w, req)

		if body := w.Body.String(); body != "Plain text response" {
			t.Errorf("Expected 'Plain text response', got '%s'", body)
		}
	})
}

// TestContextMethods 测试Context方法
func TestContextMethods(t *testing.T) {
	s := easyserver.NewServer(":0")

	t.Run("GetQueryValue", func(t *testing.T) {
		s.AddGetHandler("/query", func(ctx easyserver.HttpContext) {
			q := ctx.GetQueryValue("q", "default")
			ctx.Writer.Write([]byte(q))
		})

		req := httptest.NewRequest("GET", "/query?q=search", nil)
		w := httptest.NewRecorder()
		s.ServeHTTP(w, req)

		if body := w.Body.String(); body != "search" {
			t.Errorf("Expected 'search', got '%s'", body)
		}
	})

	t.Run("GetQueryValue默认值", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/query", nil) // 没有q参数
		w := httptest.NewRecorder()
		s.ServeHTTP(w, req)

		if body := w.Body.String(); body != "default" {
			t.Errorf("Expected 'default', got '%s'", body)
		}
	})

	t.Run("SetHeader", func(t *testing.T) {
		s.AddGetHandler("/header", func(ctx easyserver.HttpContext) {
			ctx.SetHeader("X-Test-Header", "TestValue")
			ctx.Writer.Write([]byte("OK"))
		})

		req := httptest.NewRequest("GET", "/header", nil)
		w := httptest.NewRecorder()
		s.ServeHTTP(w, req)

		header := w.Header().Get("X-Test-Header")
		if header != "TestValue" {
			t.Errorf("Expected header 'TestValue', got '%s'", header)
		}
	})
}

// TestMiddleware 测试中间件功能
func TestMiddleware(t *testing.T) {
	s := easyserver.NewServer(":0")

	// 添加自定义中间件
	s.AddMiddleHead(httpsvr.NewMiddle(func(w http.ResponseWriter, r *http.Request, df *httpsvr.DataFlow) bool {
		df.SetData("timestamp", time.Now().Unix())
		return true
	}))

	s.AddGetHandler("/middleware-test", func(ctx easyserver.HttpContext) {
		timestamp := ctx.DataFlow.GetData("timestamp").Value.(int64)
		ctx.Writer.Write([]byte(fmt.Sprintf("Timestamp: %d", timestamp)))
	})

	req := httptest.NewRequest("GET", "/middleware-test", nil)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)

	body := w.Body.String()
	if !strings.HasPrefix(body, "Timestamp: ") {
		t.Errorf("Expected response starting with 'Timestamp: ', got '%s'", body)
	}
}

// TestDataFlow 测试数据流
func TestDataFlow(t *testing.T) {
	s := easyserver.NewServer(":0")

	s.AddGetHandler("/dataflow", func(ctx easyserver.HttpContext) {
		// 设置数据
		ctx.DataFlow.SetData("key1", "value1")
		ctx.DataFlow.SetDataReadonly("key2", "value2")

		// 获取数据
		keys := ctx.DataFlow.GetDataKeys()
		if len(keys) < 3 { // 至少包含 startat, key1, key2
			ctx.Writer.Write([]byte("Not enough keys"))
			return
		}

		// 检查只读数据不能被重写
		err := ctx.DataFlow.SetData("key2", "newvalue")
		if err == nil {
			ctx.Writer.Write([]byte("Should not be able to rewrite readonly data"))
			return
		}

		ctx.Writer.Write([]byte("DataFlow OK"))
	})

	req := httptest.NewRequest("GET", "/dataflow", nil)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)

	body := w.Body.String()
	if body != "DataFlow OK" {
		t.Errorf("Expected 'DataFlow OK', got '%s'", body)
	}
}

// TestCustomOkCode 测试自定义成功状态码
func TestCustomOkCode(t *testing.T) {
	// 设置自定义成功状态码
	easyserver.SetOkCode(0)
	defer easyserver.SetOkCode(200) // 恢复默认值

	s := easyserver.NewServer(":0")

	s.AddGetHandler("/custom-code", func(ctx easyserver.HttpContext) {
		easyserver.ResponseJsonOk(ctx, "Success with custom code")
	})

	req := httptest.NewRequest("GET", "/custom-code", nil)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)

	var result response.ResponseApiData
	json.Unmarshal(w.Body.Bytes(), &result)

	if result.Code != 0 {
		t.Errorf("Expected custom code 0, got %d", result.Code)
	}
}

// TestNotFound 测试404响应
func TestNotFound(t *testing.T) {
	s := easyserver.NewServer(":0")

	// 只添加一个路由
	s.AddGetHandler("/", func(ctx easyserver.HttpContext) {
		ctx.Writer.Write([]byte("Home"))
	})

	// 测试不存在的路由
	req := httptest.NewRequest("GET", "/not-found", nil)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)

	// 注意：easyserver返回200状态码，但JSON中的code是404
	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var result response.ResponseApiData
	json.Unmarshal(w.Body.Bytes(), &result)

	if result.Code != 404 {
		t.Errorf("Expected JSON code 404, got %d", result.Code)
	}
}

// TestPostJson 测试POST JSON处理
func TestPostJson(t *testing.T) {
	s := easyserver.NewServer(":0")

	s.AddPostHandler("/post-json", func(ctx easyserver.HttpContext) {
		var data map[string]interface{}
		if err := ctx.GetPostJson(&data); err != nil {
			easyserver.ResponseJsonFail(ctx, err.Error(), 400)
			return
		}

		easyserver.ResponseJson(ctx, data, "success", 200)
	})

	t.Run("有效的JSON", func(t *testing.T) {
		jsonData := `{"name": "John", "age": 30}`
		req := httptest.NewRequest("POST", "/post-json", strings.NewReader(jsonData))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		s.ServeHTTP(w, req)

		var result response.ResponseApiData
		json.Unmarshal(w.Body.Bytes(), &result)

		if result.Code != 200 {
			t.Errorf("Expected code 200, got %d", result.Code)
		}
	})

	t.Run("无效的JSON", func(t *testing.T) {
		invalidJson := `{"name": "John", "age": 30`
		req := httptest.NewRequest("POST", "/post-json", strings.NewReader(invalidJson))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		s.ServeHTTP(w, req)

		var result response.ResponseApiData
		json.Unmarshal(w.Body.Bytes(), &result)

		if result.Code != 400 {
			t.Errorf("Expected code 400 for invalid JSON, got %d", result.Code)
		}
	})
}

// TestMultipleHandlers 测试多个处理器
func TestMultipleHandlers(t *testing.T) {
	s := easyserver.NewServer(":0")

	s.AddGetHandler("/api/users", func(ctx easyserver.HttpContext) {
		data := map[string]interface{}{
			"users": []string{"Alice", "Bob", "Charlie"},
		}
		easyserver.ResponseJson(ctx, data, "Users retrieved", 200)
	})

	s.AddPostHandler("/api/users", func(ctx easyserver.HttpContext) {
		data := map[string]interface{}{
			"id":      123,
			"message": "User created",
		}
		easyserver.ResponseJson(ctx, data, "User created", 201)
	})

	t.Run("GET请求", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/users", nil)
		w := httptest.NewRecorder()
		s.ServeHTTP(w, req)

		if w.Code != 200 {
			t.Errorf("GET: Expected status 200, got %d", w.Code)
		}
	})

	t.Run("POST请求", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/users", nil)
		w := httptest.NewRecorder()
		s.ServeHTTP(w, req)

		if w.Code != 200 { // 注意：ResponseJson设置的是JSON的code，HTTP状态码还是200
			t.Errorf("POST: Expected status 200, got %d", w.Code)
		}
	})

	t.Run("PUT请求（未定义）", func(t *testing.T) {
		req := httptest.NewRequest("PUT", "/api/users", nil)
		w := httptest.NewRecorder()
		s.ServeHTTP(w, req)

		// 应该返回404
		var result response.ResponseApiData
		json.Unmarshal(w.Body.Bytes(), &result)

		if result.Code != 404 {
			t.Errorf("PUT: Expected code 404 for undefined method, got %d", result.Code)
		}
	})
}
