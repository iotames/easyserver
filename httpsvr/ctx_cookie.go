package httpsvr

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// CookieOption 定义 Cookie 的可选参数配置函数
type CookieOption func(*cookieOptions)

// cookieOptions 内部存储 Cookie 的所有可选字段
type cookieOptions struct {
	path     string
	domain   string
	maxAge   int
	secure   bool
	httpOnly bool
	sameSite http.SameSite
}

// WithPath 设置 Cookie 的路径，默认为 "/"
func WithPath(path string) CookieOption {
	return func(o *cookieOptions) { o.path = path }
}

// WithDomain 设置 Cookie 的域名，默认为空（当前域名）
func WithDomain(domain string) CookieOption {
	return func(o *cookieOptions) { o.domain = domain }
}

// WithMaxAge 设置 Cookie 的有效期（秒），默认为 0（会话 Cookie）
func WithMaxAge(maxAge int) CookieOption {
	return func(o *cookieOptions) { o.maxAge = maxAge }
}

// WithSecure 设置是否仅通过 HTTPS 传输，默认为 false
func WithSecure(secure bool) CookieOption {
	return func(o *cookieOptions) { o.secure = secure }
}

// WithHTTPOnly 设置是否禁止 JavaScript 访问，默认为 true（增强安全性）
func WithHTTPOnly(httpOnly bool) CookieOption {
	return func(o *cookieOptions) { o.httpOnly = httpOnly }
}

// WithSameSite 设置 SameSite 属性，默认为 http.SameSiteDefaultMode
func WithSameSite(sameSite http.SameSite) CookieOption {
	return func(o *cookieOptions) { o.sameSite = sameSite }
}

// SetCookie 设置 Cookie
//
// 必需参数：
//   - name:  Cookie 名称
//   - value: Cookie 值
//
// 可选参数通过传入 CookieOption 函数进行配置，例如：
//
//	// 设置一个会话 Cookie（默认 HttpOnly=true）
//	err := ctx.SetCookie("user_preference", "dark_mode")
//
//	// 设置一个 24 小时有效期的 Cookie，并指定路径
//	err := ctx.SetCookie("session_id", "abc123", WithMaxAge(86400), WithPath("/admin"))
//
//	// 允许 JavaScript 访问该 Cookie（不推荐）
//	err := ctx.SetCookie("tracking_id", "xyz", WithHTTPOnly(false))
func (ctx Context) SetCookie(name, value string, opts ...CookieOption) error {
	if name == "" {
		return fmt.Errorf("cookie 名称不能为空")
	}

	// 初始化默认选项
	options := &cookieOptions{
		path:     "/",
		httpOnly: true, // 默认启用 HttpOnly 提高安全性
	}

	// 应用用户传入的可选配置
	for _, opt := range opts {
		opt(options)
	}

	cookie := &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     options.path,
		Domain:   options.domain,
		MaxAge:   options.maxAge,
		Secure:   options.secure,
		HttpOnly: options.httpOnly,
		SameSite: options.sameSite,
	}

	// 如果设置了 MaxAge，计算过期时间（用于兼容旧浏览器）
	if options.maxAge > 0 {
		cookie.Expires = time.Now().Add(time.Duration(options.maxAge) * time.Second)
	}

	http.SetCookie(ctx.Writer, cookie)
	return nil
}

// GetCookie 获取 Cookie
//
//	value, err := ctx.GetCookie("user_preference", "")
//	if err == nil {
//		fmt.Printf("用户偏好：%s\n", value)
//	}
func (ctx Context) GetCookie(cookieName string, defaultValue string) (string, error) {
	if cookieName == "" {
		return defaultValue, fmt.Errorf("cookie 名称不能为空")
	}

	cookie, err := ctx.Request.Cookie(cookieName)
	if err != nil {
		return defaultValue, fmt.Errorf("未找到 cookie [%s]: %w", cookieName, err)
	}

	return cookie.Value, nil
}

// RemoveCookie 删除 Cookie
//
// 必需参数：
//   - name: Cookie 名称
//
// 可选参数（用于精确匹配要删除的 Cookie）：
//   - WithPath():   指定路径，默认为 "/"
//   - WithDomain(): 指定域名，默认为空（当前域名）
//
// 示例：
//
//	// 删除名为 "session_id" 的 Cookie（使用默认路径 "/"）
//	err := ctx.RemoveCookie("session_id")
//
//	// 删除指定路径下的 Cookie
//	err := ctx.RemoveCookie("pref", WithPath("/admin"))
func (ctx Context) RemoveCookie(name string, opts ...CookieOption) error {
	if name == "" {
		return fmt.Errorf("cookie 名称不能为空")
	}

	// 初始化默认选项（与 SetCookie 的默认值保持一致）
	options := &cookieOptions{
		path: "/",
	}
	for _, opt := range opts {
		opt(options)
	}

	// 创建一个过期的 Cookie 来删除它
	cookie := &http.Cookie{
		Name:     name,
		Value:    "",
		Path:     options.path,
		Domain:   options.domain,
		Expires:  time.Unix(0, 0), // 设置为过去的时间
		MaxAge:   -1,              // 立即过期
		HttpOnly: true,            // 删除时 HttpOnly 不影响，保留原值
		Secure:   options.secure,  // 若原 Cookie 有 Secure 属性，此处需保持一致
		SameSite: options.sameSite,
	}

	http.SetCookie(ctx.Writer, cookie)
	return nil
}

// CheckCookie 检查 Cookie 是否存在且有效
//
//	exists, value, err := ctx.CheckCookie("session_id")
//	if exists {
//		fmt.Printf("Session ID: %s\n", value)
//	} else {
//		fmt.Println("Cookie 不存在或已过期")
//	}
func (ctx Context) CheckCookie(cookieName string) (bool, string, error) {
	if cookieName == "" {
		return false, "", fmt.Errorf("cookie 名称不能为空")
	}

	cookie, err := ctx.Request.Cookie(cookieName)
	if err != nil {
		return false, "", fmt.Errorf("未找到 cookie [%s]: %w", cookieName, err)
	}

	if cookie.Value == "" {
		return false, "", fmt.Errorf("cookie [%s] 值为空", cookieName)
	}

	return true, cookie.Value, nil
}

// SetJsonCookie 设置 JSON 数据的 Cookie (自动编码)
//
//	data := map[string]interface{}{"user_id": 123, "role": "admin"}
//	err := ctx.SetJsonCookie("auth_info", data, 3600) // 有效期 1 小时
//
// 注意：Cookie 大小通常限制在 4KB 左右，请勿存储过大数据。
func (ctx Context) SetJsonCookie(cookieName string, data interface{}, maxAge int, opts ...CookieOption) error {
	if cookieName == "" {
		return fmt.Errorf("cookie 名称不能为空")
	}
	if maxAge < 0 {
		return fmt.Errorf("maxAge 不能为负数")
	}

	// 将数据编码为 JSON
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("序列化数据失败：%w", err)
	}

	// Base64 编码
	encodedData := base64.StdEncoding.EncodeToString(jsonData)

	// 使用 SetCookie 设置 Cookie（默认 HttpOnly=true，路径="/"）
	return ctx.SetCookie(cookieName, encodedData, append(opts, WithMaxAge(maxAge))...)
}

// GetJsonCookie 获取 JSON 数据的 Cookie (自动解码)
//
//	var userData map[string]interface{}
//	err := ctx.GetJsonCookie("auth_info", &userData)
//	if err == nil {
//		fmt.Printf("用户 ID: %v\n", userData["user_id"])
//	}
func (ctx Context) GetJsonCookie(cookieName string, v interface{}) error {
	if cookieName == "" {
		return fmt.Errorf("cookie 名称不能为空")
	}

	// 获取 Cookie
	cookieValue, err := ctx.GetCookie(cookieName, "")
	if err != nil {
		return fmt.Errorf("获取 cookie 失败：%w", err)
	}
	if cookieValue == "" {
		return fmt.Errorf("cookie [%s] 值为空", cookieName)
	}

	// Base64 解码
	decodedData, err := base64.StdEncoding.DecodeString(cookieValue)
	if err != nil {
		return fmt.Errorf("解码数据失败：%w", err)
	}

	// 解析 JSON 数据
	err = json.Unmarshal(decodedData, v)
	if err != nil {
		return fmt.Errorf("解析数据失败：%w", err)
	}
	return nil
}
