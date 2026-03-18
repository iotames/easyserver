package httpsvr

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"
)

// SessionStore 定义 Session 存储接口，方便替换为 Redis 等外部存储
type SessionStore interface {
	// Get 根据 sessionID 获取存储的数据，返回 nil 表示不存在
	Get(sessionID string) (map[string]interface{}, error)
	// Set 存储 sessionID 对应的数据，并指定过期时间（秒）
	Set(sessionID string, data map[string]interface{}, maxAge int) error
	// Delete 删除 sessionID 对应的数据
	Delete(sessionID string) error
}

// memorySessionStore 基于内存的 Session 存储实现（仅供示例，生产环境建议使用 Redis）
type memorySessionStore struct {
	mu   sync.RWMutex
	data map[string]sessionItem
}

type sessionItem struct {
	data      map[string]interface{}
	expiresAt int64 // Unix 时间戳（秒），0 表示永不过期
}

// Get 实现 SessionStore 接口（带双重检查锁定防止误删）
func (s *memorySessionStore) Get(sessionID string) (map[string]interface{}, error) {
	s.mu.RLock()
	item, ok := s.data[sessionID]
	s.mu.RUnlock()
	if !ok {
		return nil, nil
	}
	if item.expiresAt > 0 && item.expiresAt < time.Now().Unix() {
		s.mu.Lock()
		if item2, ok := s.data[sessionID]; ok && item2.expiresAt == item.expiresAt {
			delete(s.data, sessionID)
		}
		s.mu.Unlock()
		return nil, nil
	}
	// 深拷贝
	return deepCopyMap(item.data), nil
}

// Set 实现 SessionStore 接口
func (s *memorySessionStore) Set(sessionID string, data map[string]interface{}, maxAge int) error {
	var expiresAt int64
	if maxAge > 0 {
		expiresAt = time.Now().Unix() + int64(maxAge)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.data == nil {
		s.data = make(map[string]sessionItem)
	}
	s.data[sessionID] = sessionItem{
		data:      deepCopyMap(data),
		expiresAt: expiresAt,
	}
	return nil
}

// deepCopyMap 通过 JSON 序列化实现深拷贝（要求 data 可 JSON 序列化）
func deepCopyMap(src map[string]interface{}) map[string]interface{} {
	if src == nil {
		return nil
	}
	b, _ := json.Marshal(src)
	var dst map[string]interface{}
	_ = json.Unmarshal(b, &dst)
	return dst
}

// Delete 实现 SessionStore 接口
func (s *memorySessionStore) Delete(sessionID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.data, sessionID)
	return nil
}

// 默认的内存 Session 存储实例
var defaultSessionStore SessionStore = &memorySessionStore{}

// SessionConfig Session 配置，用于 SetSession、GetSession 等函数
type SessionConfig struct {
	// CookieName 存储 sessionID 的 Cookie 名称，默认为 "session_id"
	CookieName string
	// MaxAge Cookie 和 Session 的有效期（秒），默认为 86400（24小时）
	MaxAge int
	// Path Cookie 路径，默认为 "/"
	Path string
	// Domain Cookie 域名，默认为空
	Domain string
	// Secure 是否仅 HTTPS 传输，默认为 false
	Secure bool
	// HTTPOnly 是否禁止 JavaScript 访问，默认为 true
	HTTPOnly bool
	// SameSite SameSite 属性，默认为 http.SameSiteDefaultMode
	SameSite http.SameSite
	// Store 自定义 Session 存储，默认使用内存存储
	Store SessionStore
}

// SessionOption 定义 Session 的可选参数配置函数
type SessionOption func(*SessionConfig)

// WithSessionCookieName 设置存储 sessionID 的 Cookie 名称
func WithSessionCookieName(name string) SessionOption {
	return func(c *SessionConfig) { c.CookieName = name }
}

// WithSessionMaxAge 设置 Session 和 Cookie 的有效期（秒）
func WithSessionMaxAge(maxAge int) SessionOption {
	return func(c *SessionConfig) { c.MaxAge = maxAge }
}

// WithSessionPath 设置 Cookie 的路径
func WithSessionPath(path string) SessionOption {
	return func(c *SessionConfig) { c.Path = path }
}

// WithSessionDomain 设置 Cookie 的域名
func WithSessionDomain(domain string) SessionOption {
	return func(c *SessionConfig) { c.Domain = domain }
}

// WithSessionSecure 设置是否仅 HTTPS 传输
func WithSessionSecure(secure bool) SessionOption {
	return func(c *SessionConfig) { c.Secure = secure }
}

// WithSessionHTTPOnly 设置是否禁止 JavaScript 访问 Cookie
func WithSessionHTTPOnly(httpOnly bool) SessionOption {
	return func(c *SessionConfig) { c.HTTPOnly = httpOnly }
}

// WithSessionSameSite 设置 SameSite 属性
func WithSessionSameSite(sameSite http.SameSite) SessionOption {
	return func(c *SessionConfig) { c.SameSite = sameSite }
}

// WithSessionStore 设置自定义的 Session 存储
func WithSessionStore(store SessionStore) SessionOption {
	return func(c *SessionConfig) { c.Store = store }
}

// SetSession 创建或更新 Session
//
// 必需参数：
//   - data:  需要存储的会话数据（必须是可 JSON 序列化的）
//
// 可选参数通过传入 SessionOption 函数进行配置（例如修改 Cookie 名称、过期时间等）
// 若不提供任何选项，将使用默认配置（Cookie 名称 "session_id"，有效期 24 小时）
//
// 行为：
//   - 如果当前请求已包含有效会话，则复用该会话 ID 更新数据
//   - 否则生成新的会话 ID 并存储数据
//
// 示例：
//
//	// 使用默认配置创建 Session
//	err := ctx.SetSession(map[string]interface{}{"user_id": 123, "role": "admin"})
//
//	// 自定义 Cookie 名称和有效期
//	err := ctx.SetSession(userData, WithSessionCookieName("my_session"), WithSessionMaxAge(3600))
func (ctx *Context) SetSession(data map[string]interface{}, opts ...SessionOption) error {
	// 1. 构建默认配置
	config := &SessionConfig{
		CookieName: "session_id",
		MaxAge:     86400, // 24小时
		Path:       "/",
		HTTPOnly:   true, // 默认启用 HttpOnly
		Store:      defaultSessionStore,
	}
	// 2. 应用选项
	for _, opt := range opts {
		opt(config)
	}
	if config.CookieName == "" {
		return fmt.Errorf("session cookie name cannot be empty")
	}

	// 3. 尝试获取当前 sessionID（如果存在且有效）
	var sessionID string
	existingID, _ := ctx.GetCookie(config.CookieName, "")
	if existingID != "" {

		existingData, err := config.Store.Get(existingID)
		if err != nil {
			// 存储错误应向上返回，避免误判
			return fmt.Errorf("检查现有 session 失败: %w", err)
		}
		if existingData != nil {
			sessionID = existingID
		}

	}

	// 4. 如果没有有效 sessionID，生成新 ID
	if sessionID == "" {
		newID, err := generateSessionID()
		if err != nil {
			return fmt.Errorf("生成 sessionID 失败: %w", err)
		}
		sessionID = newID
	}

	// 5. 深拷贝 data（防止外部修改影响存储）
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("序列化 session 数据失败: %w", err)
	}
	var storedData map[string]interface{}
	if err := json.Unmarshal(jsonData, &storedData); err != nil {
		return fmt.Errorf("反序列化 session 数据失败: %w", err)
	}

	// 6. 存入存储
	if err := config.Store.Set(sessionID, storedData, config.MaxAge); err != nil {
		return fmt.Errorf("存储 session 失败: %w", err)
	}

	// 7. 设置 Cookie
	cookieOpts := []CookieOption{
		WithPath(config.Path),
		WithDomain(config.Domain),
		WithMaxAge(config.MaxAge),
		WithSecure(config.Secure),
		WithHTTPOnly(config.HTTPOnly),
		WithSameSite(config.SameSite),
	}
	return ctx.SetCookie(config.CookieName, sessionID, cookieOpts...)
}

// GetSession 获取当前请求的 Session 数据
//
// 可选参数通过传入 SessionOption 函数进行配置（用于指定非默认的 Cookie 名称等）
//
// 返回值：
//   - data: 会话数据，如果不存在或已过期返回 nil
//   - err:  错误信息
//
// 示例：
//
//	data, err := ctx.GetSession()
//	if err != nil {
//		// 处理错误
//	}
//	if data != nil {
//		userID := data["user_id"]
//	}
func (ctx *Context) GetSession(opts ...SessionOption) (map[string]interface{}, error) {
	// 构建默认配置
	config := &SessionConfig{
		CookieName: "session_id",
		Path:       "/",
		Store:      defaultSessionStore,
	}
	for _, opt := range opts {
		opt(config)
	}

	if config.CookieName == "" {
		return nil, fmt.Errorf("session cookie name cannot be empty")
	}

	// 从 Cookie 中获取 sessionID
	sessionID, err := ctx.GetCookie(config.CookieName, "")
	if err != nil || sessionID == "" {
		return nil, nil // Cookie 不存在视为无会话，不返回错误
	}

	// 从存储中获取数据
	data, err := config.Store.Get(sessionID)
	if err != nil {
		return nil, fmt.Errorf("获取 session 失败: %w", err)
	}
	if data == nil {
		// 数据不存在或已过期，尝试清理客户端 Cookie（记录日志便于排查）
		if err := ctx.RemoveCookie(config.CookieName,
			WithPath(config.Path),
			WithDomain(config.Domain),
			WithSecure(config.Secure),
			WithSameSite(config.SameSite),
		); err != nil {
			// 删除失败不影响主流程，仅记录
			log.Printf("清理过期 session cookie 失败: %v", err)
		}
	}
	return data, nil
}

// ClearSession 清除当前请求的 Session（删除存储中的数据并清除 Cookie）
//
// 可选参数通过传入 SessionOption 函数进行配置（用于指定非默认的 Cookie 名称等）
//
// 示例：
//
//	err := ctx.ClearSession() // 登出
func (ctx *Context) ClearSession(opts ...SessionOption) error {
	config := &SessionConfig{
		CookieName: "session_id",
		Path:       "/",
		Store:      defaultSessionStore,
	}
	for _, opt := range opts {
		opt(config)
	}
	if config.CookieName == "" {
		return fmt.Errorf("session cookie name cannot be empty")
	}

	sessionID, _ := ctx.GetCookie(config.CookieName, "")
	var storeErr error
	if sessionID != "" {
		storeErr = config.Store.Delete(sessionID)
	}

	cookieErr := ctx.RemoveCookie(config.CookieName,
		WithPath(config.Path),
		WithDomain(config.Domain),
		WithSecure(config.Secure),
		WithSameSite(config.SameSite),
	)

	if storeErr != nil {
		return fmt.Errorf("failed to delete session from store: %w", storeErr)
	}
	return cookieErr
}

// generateSessionID 生成一个安全的随机字符串作为 sessionID
func generateSessionID() (string, error) {
	b := make([]byte, 32) // 256 位
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// ==================== 使用示例（注释中） ====================
//
// 以下代码仅为演示用法，实际应放在处理器函数中：
//
// func LoginHandler(ctx *httpsvr.Context) {
//     var creds struct{ Username string `json:"username"` }
//     if err := ctx.GetPostJson(&creds); err != nil {
//         ctx.Json(map[string]any{"code": 1, "msg": "无效请求"}, http.StatusBadRequest)
//         return
//     }
//     // 假设验证通过，用户ID为 123
//     userData := map[string]interface{}{
//         "user_id": 123,
//         "name":    "张三",
//         "role":    "admin",
//     }
//     // 创建 Session（使用默认配置）
//     if err := ctx.SetSession(userData); err != nil {
//         ctx.Json(map[string]any{"code": 1, "msg": "登录失败"}, http.StatusInternalServerError)
//         return
//     }
//     ctx.Json(map[string]any{"code": 0, "msg": "登录成功"}, http.StatusOK)
// }
//
// func ProfileHandler(ctx *httpsvr.Context) {
//     // 获取 Session 数据
//     data, err := ctx.GetSession()
//     if err != nil {
//         ctx.Json(map[string]any{"code": 1, "msg": "服务器错误"}, http.StatusInternalServerError)
//         return
//     }
//     if data == nil {
//         ctx.Json(map[string]any{"code": 2, "msg": "未登录"}, http.StatusUnauthorized)
//         return
//     }
//     userID := data["user_id"]
//     ctx.Json(map[string]any{"code": 0, "user_id": userID}, http.StatusOK)
// }
//
// func LogoutHandler(ctx *httpsvr.Context) {
//     if err := ctx.ClearSession(); err != nil {
//         ctx.Json(map[string]any{"code": 1, "msg": "登出失败"}, http.StatusInternalServerError)
//         return
//     }
//     ctx.Json(map[string]any{"code": 0, "msg": "已登出"}, http.StatusOK)
// }
