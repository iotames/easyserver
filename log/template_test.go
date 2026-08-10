package log

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"
	"text/template"
	"time"
)

// TestFormatTemplate 纯渲染逻辑：默认常量模板格式、attr 覆盖、time/level/msg 同名跳过、
// WithGroup 嵌套、点号 key、WithAttrs/WithGroup 返回新 handler（不共享 buf、不污染原 handler）。
func TestFormatTemplate(t *testing.T) {
	// 1. 内置常量模板 defaultLogTpl 渲染输出 time=... level=... msg=... 格式；末尾自动补换行。
	var buf bytes.Buffer
	h, err := newTemplateHandler(&buf)
	if err != nil {
		t.Fatal(err)
	}
	r := slog.NewRecord(time.Now(), slog.LevelInfo, "hello", 0)
	if err := h.Handle(context.Background(), r); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(buf.String(), "time=") || !strings.Contains(buf.String(), "level=INFO") || !strings.Contains(buf.String(), "msg=hello") {
		t.Fatalf("内置常量模板渲染失败: %q", buf.String())
	}
	if !strings.HasSuffix(buf.String(), "\n") {
		t.Fatal("渲染末尾无换行时应补换行")
	}

	// 2. 记录级 attr 覆盖 With 级；time/level/msg 同名 attr 跳过；模板未引用 attr 不输出。
	buf.Reset()
	tpl, err := template.New("test").Parse(`{{.time}} {{.level}} {{.msg}} k={{.k}}`)
	if err != nil {
		t.Fatal(err)
	}
	h2 := &templateHandler{w: &buf, tpl: tpl, buf: &bytes.Buffer{}}
	h2w := h2.WithAttrs([]slog.Attr{slog.String("k", "with"), slog.String("time", "fake-time")})
	r2 := slog.NewRecord(time.Now(), slog.LevelInfo, "m", 0)
	r2.AddAttrs(slog.String("k", "record"), slog.String("level", "fake-level"))
	if err := h2w.Handle(context.Background(), r2); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "k=record") {
		t.Fatal("记录级 attr 应覆盖 With 级")
	}
	if strings.Contains(out, "fake-time") {
		t.Fatal("time 同名 attr 应跳过")
	}
	if strings.Contains(out, "fake-level") {
		t.Fatal("level 同名 attr 应跳过")
	}

	// 3. WithGroup 嵌套注入（group attr 以 group.key 形式存储，按 '.' 分割构造嵌套 map）。
	buf.Reset()
	tpl3, err := template.New("test").Parse(`{{.req.id}} {{.req.name}}`)
	if err != nil {
		t.Fatal(err)
	}
	h3 := &templateHandler{w: &buf, tpl: tpl3, buf: &bytes.Buffer{}}
	h3g := h3.WithGroup("req").WithAttrs([]slog.Attr{slog.String("id", "7"), slog.String("name", "n")})
	if err := h3g.Handle(context.Background(), slog.NewRecord(time.Now(), slog.LevelInfo, "m", 0)); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "7 n") {
		t.Fatalf("WithGroup 嵌套注入失败: %q", buf.String())
	}

	// 3b. 点号 key 的 attr（非 WithGroup）同样按 '.' 分割构造嵌套 map。
	buf.Reset()
	h3b := h3.WithAttrs([]slog.Attr{slog.String("req.id", "9")})
	if err := h3b.Handle(context.Background(), slog.NewRecord(time.Now(), slog.LevelInfo, "m", 0)); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "9") {
		t.Fatalf("点号 key attr 嵌套注入失败: %q", buf.String())
	}

	// 4. WithAttrs/WithGroup 返回新 handler（不共享 buf、不返回自身、不污染原 handler）。
	buf.Reset()
	tpl4, err := template.New("test").Parse(`{{.time}} {{.k}}`)
	if err != nil {
		t.Fatal(err)
	}
	h4 := &templateHandler{w: &buf, tpl: tpl4, buf: &bytes.Buffer{}}
	h4a := h4.WithAttrs([]slog.Attr{slog.String("k", "XYZV")})
	h4aH := h4a.(*templateHandler)
	if h4aH == h4 {
		t.Fatal("WithAttrs 不得返回自身")
	}
	if h4aH.buf == h4.buf {
		t.Fatal("WithAttrs 不得共享原 buf")
	}
	// 原 handler 不受新 handler 影响。
	if err := h4.Handle(context.Background(), slog.NewRecord(time.Now(), slog.LevelInfo, "m", 0)); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(buf.String(), "XYZV") {
		t.Fatal("新 handler 的 attr 不得污染原 handler")
	}
}
