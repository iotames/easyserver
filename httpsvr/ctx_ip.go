package httpsvr

import (
	"net"
	"net/http"
	"strings"
)

// GetOriginalDirectIP 获取直连源IP（TCP层真实来源，不读取任何HTTP头）。
// 无论请求是否经过代理，此函数始终返回 TCP 连接的对端 IP。
// 不可信代理使用本方法获取客户端IP
func GetOriginalDirectIP(r *http.Request) string {
	if r == nil {
		return ""
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// GetClientIP 获取客户端IP（优先读取上级代理设置的转发头）。
//
// 优先级：X-Real-IP → X-Forwarded-For 第一个 → 直连源IP
//
// X-Real-IP 由 Nginx 通过 proxy_set_header X-Real-IP $remote_addr 设置，
// 是覆写语义（客户端伪造值会被丢弃），在"上级代理为自有 Nginx"时可信。
//
// X-Forwarded-For 为追加语义，ips[0] 是客户端声明的原始 IP，可能被伪造。
// 当 X-Real-IP 缺失时退回此字段，此时已不可信，取 ips[0] 作为最佳推测。
// 既然到这一步已经不可信，取末尾取开头没有本质区别，取开头符合 XFF 规范
// 对"原始客户端"的定义，且在合法多级代理场景下正是期望值。
//
// 注意：本函数不做可信代理校验。如需防止公网直连时的头伪造攻击，
// 应在上层（如 rocksys 网络层）先判断 TCP 源 IP 是否为可信代理，
// 可信代理使用本方法获取客户端IP。不可信时调用 GetOriginalDirectIP。
func GetClientIP(r *http.Request) string {
	if r == nil {
		return ""
	}
	// 1. X-Real-IP：自主配置的Nginx 覆写，不可被客户端伪造
	if realIP := strings.TrimSpace(r.Header.Get("X-Real-IP")); realIP != "" {
		return realIP
	}
	// 2. X-Forwarded-For：追加语义，取第一个（客户端声明的原始 IP）
	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
		ips := strings.Split(fwd, ",")
		if ip := strings.TrimSpace(ips[0]); ip != "" {
			return ip
		}
	}
	// 3. 兜底：直连源 IP
	return GetOriginalDirectIP(r)
}

// GetOriginalDirectIP 获取直连源IP。不可信代理使用本方法获取客户端IP
func (ctx Context) GetOriginalDirectIP() string {
	return GetOriginalDirectIP(ctx.Request)
}

// GetClientIP 获取客户端IP。可信代理使用本方法获取客户端IP。不可信时调用 GetOriginalDirectIP。
func (ctx Context) GetClientIP() string {
	return GetClientIP(ctx.Request)
}
