package httpsvr

// UserAgent 获取UserAgent请求头
func (ctx Context) UserAgent() string {
	return ctx.Request.UserAgent()
}
