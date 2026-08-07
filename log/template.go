package log

import (
	"bytes"
	"context"
	_ "embed"
	"io"
	"log/slog"
	"strings"
	"sync"
	"text/template"
	"time"
)

//go:embed log.tpl
var embedLogTpl string

// TemplateLoader 模板加载接口（rocksys 装配时注入 hotswap.ScriptDir 适配器，接口解耦不反向依赖）。
type TemplateLoader interface {
	GetScriptBytes(fpath string) ([]byte, error)
}

// tplLoader 模板加载器（外挂优先，内嵌兜底）。启动装配期注入，运行期定死。
var tplLoader TemplateLoader

// SetTemplateLoader 注入模板加载器（rocksys 装配时传入）。
// 若 once 已执行（日志已初始化），外挂模板不生效，仅打警告并拒绝静默忽略。
func SetTemplateLoader(l TemplateLoader) {
	if onceDone.Load() {
		Warn("模板加载器注入过晚，外挂 log.tpl 不生效")
		return
	}
	tplLoader = l
}

// templateHandler 按 log.tpl 渲染每条日志。
type templateHandler struct {
	mu     sync.Mutex
	w      io.Writer
	tpl    *template.Template // text/template，解析后的模板
	buf    *bytes.Buffer      // 复用缓冲（避免每行分配）
	attrs  []slog.Attr        // WithAttrs 累积的附加 attrs（group attr 以 group.key 形式存储）
	prefix string             // WithGroup 累积的 group 前缀（如 "req."），供后续 WithAttrs 加前缀
}

// newTemplateHandler 加载 log.tpl 并解析。
// ★ 回退层级：内部先经 tplLoader（外部）加载 → 失败/为空回退内嵌 log.tpl → 仍失败才返回 error
//
//	（调用方 buildLogger 收到 error 才回退 slog text handler）。
func newTemplateHandler(w io.Writer) (*templateHandler, error) {
	text := embedLogTpl
	if tplLoader != nil {
		if b, err := tplLoader.GetScriptBytes(templateFile); err == nil && len(bytes.TrimSpace(b)) > 0 {
			text = string(b)
		}
	}
	tpl, err := template.New(templateFile).Parse(text)
	if err != nil {
		// 外挂模板解析失败 → 回退内嵌默认模板。
		if text == embedLogTpl {
			// 内嵌模板都失败，属异常，返回 error 让调用方回退 slog text handler。
			return nil, err
		}
		tpl, err = template.New(templateFile).Parse(embedLogTpl)
		if err != nil {
			return nil, err
		}
	}
	return &templateHandler{
		w:   w,
		tpl: tpl,
		buf: &bytes.Buffer{},
	}, nil
}

// Handle 实现 slog.Handler：渲染模板 → 写 w。
func (h *templateHandler) Handle(_ context.Context, r slog.Record) error {
	// 组装数据：time/level/msg 恒为注入的标准值。
	data := map[string]any{
		"time":  r.Time.Format(time.DateTime),
		"level": r.Level.String(),
		"msg":   r.Message,
	}
	// 先展开 h.attrs（WithAttrs/WithGroup 累积的），再展开记录级 attr——记录级后写覆盖 With 级。
	for _, a := range h.attrs {
		injectAttr(data, a)
	}
	r.Attrs(func(a slog.Attr) bool {
		injectAttr(data, a)
		return true
	})

	// h.mu 保护下复用 h.buf 渲染，末尾无 \n 补写。
	h.mu.Lock()
	defer h.mu.Unlock()
	h.buf.Reset()
	if err := h.tpl.Execute(h.buf, data); err != nil {
		return err
	}
	if h.buf.Len() == 0 || h.buf.Bytes()[h.buf.Len()-1] != '\n' {
		h.buf.WriteByte('\n')
	}
	_, err := h.w.Write(h.buf.Bytes())
	return err
}

// injectAttr 将 attr 注入 data。
// time/level/msg 同名 attr 跳过（保留键防御）；group attr 的 key 按 '.' 分割逐级构造嵌套 map。
func injectAttr(data map[string]any, a slog.Attr) {
	if a.Key == "" {
		return
	}
	switch a.Key {
	case "time", "level", "msg":
		// 保留键防御：同名跳过，保证模板的 time/level/msg 恒为标准注入值。
		return
	}
	parts := strings.Split(a.Key, ".")
	m := data
	for i, part := range parts {
		if i == len(parts)-1 {
			m[part] = a.Value.Any()
			return
		}
		child, ok := m[part].(map[string]any)
		if !ok {
			child = map[string]any{}
			m[part] = child
		}
		m = child
	}
}

// Enabled 实现 slog.Handler：级别过滤。
// l >= slog.LevelError 无条件放行（Error 恒输出基座约束）；其余按 lgLevel 过滤。
func (h *templateHandler) Enabled(_ context.Context, l slog.Level) bool {
	if l >= slog.LevelError {
		return true
	}
	return l >= lgLevel.Level()
}

// WithAttrs 实现 slog.Handler。
// ⚠️ 返回**新 handler**：复制本 handler、追加 attrs 到 h.attrs、**新建独立的 buf**（不得共享原 buf），
//
//	不得返回自身（否则 logger.With() 的 attr 静默丢弃）。
func (h *templateHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	newAttrs := make([]slog.Attr, 0, len(h.attrs)+len(attrs))
	newAttrs = append(newAttrs, h.attrs...)
	for _, a := range attrs {
		// 若带 group 前缀，则为 attr key 加前缀（如 "req.id"）。
		if h.prefix != "" {
			newAttrs = append(newAttrs, slog.Any(h.prefix+a.Key, a.Value.Any()))
		} else {
			newAttrs = append(newAttrs, a)
		}
	}
	return &templateHandler{
		w:      h.w,
		tpl:    h.tpl,
		buf:    &bytes.Buffer{},
		attrs:  newAttrs,
		prefix: h.prefix,
	}
}

// WithGroup 实现 slog.Handler。
// ⚠️ 同理返回新 handler（attrs 带 group 前缀，新建 buf）。group attr 以 group.key 形式存储，
//
//	Handle 组装时按 '.' 分割逐级构造嵌套 map[string]any（模板作者按 {{.req.id}} 引用）。
func (h *templateHandler) WithGroup(name string) slog.Handler {
	if name == "" {
		// slog 约定：空 group 名不改变。
		return h
	}
	prefix := name + "."
	if h.prefix != "" {
		prefix = h.prefix + name + "."
	}
	return &templateHandler{
		w:      h.w,
		tpl:    h.tpl,
		buf:    &bytes.Buffer{},
		attrs:  h.attrs,
		prefix: prefix,
	}
}
