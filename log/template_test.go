package log

import (
	"bytes"
	"context"
	"log/slog"
	"os"
	"strings"
	"testing"
	"time"
)

// mockLoader 模拟模板加载器（外挂 log.tpl）。
type mockLoader struct {
	scripts map[string]string
}

// GetScriptBytes 实现 TemplateLoader 接口。
func (m *mockLoader) GetScriptBytes(fpath string) ([]byte, error) {
	if b, ok := m.scripts[fpath]; ok {
		return []byte(b), nil
	}
	return nil, os.ErrNotExist
}

// TestFormatTemplate 外挂 log.tpl 生效；缺失回退内嵌；json 风格模板可用；模板解析失败回退默认。
func TestFormatTemplate(t *testing.T) {
	oldLoader := tplLoader
	defer func() { tplLoader = oldLoader }()

	// 1. 外挂 json 风格模板生效；末尾自动补换行。
	var buf bytes.Buffer
	tplLoader = &mockLoader{scripts: map[string]string{
		"log.tpl": `{"time":"{{.time}}","level":"{{.level}}","msg":"{{.msg}}"}`,
	}}
	h, err := newTemplateHandler(&buf)
	if err != nil {
		t.Fatal(err)
	}
	r := slog.NewRecord(time.Now(), slog.LevelInfo, "hello", 0)
	if err := h.Handle(context.Background(), r); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), `"msg":"hello"`) {
		t.Fatalf("json 风格模板未生效: %q", buf.String())
	}
	if !strings.HasSuffix(buf.String(), "\n") {
		t.Fatal("渲染末尾无换行时应补换行")
	}

	// 2. 缺失回退内嵌默认模板。
	buf.Reset()
	tplLoader = &mockLoader{scripts: map[string]string{}}
	h2, err := newTemplateHandler(&buf)
	if err != nil {
		t.Fatal(err)
	}
	r2 := slog.NewRecord(time.Now(), slog.LevelInfo, "hi", 0)
	if err := h2.Handle(context.Background(), r2); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(buf.String(), "time=") || !strings.Contains(buf.String(), "level=INFO") || !strings.Contains(buf.String(), "msg=hi") {
		t.Fatalf("缺失回退内嵌失败: %q", buf.String())
	}

	// 3. 模板解析失败回退默认。
	buf.Reset()
	tplLoader = &mockLoader{scripts: map[string]string{"log.tpl": "{{.bad"}}
	h3, err := newTemplateHandler(&buf)
	if err != nil {
		t.Fatal("解析失败应回退内嵌而非报错")
	}
	if err := h3.Handle(context.Background(), r2); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(buf.String(), "time=") {
		t.Fatal("解析失败回退默认失败")
	}

	// 4. 记录级 attr 覆盖 With 级；time/level/msg 同名 attr 跳过；模板未引用 attr 不输出。
	buf.Reset()
	tplLoader = &mockLoader{scripts: map[string]string{
		"log.tpl": `{{.time}} {{.level}} {{.msg}} k={{.k}}`,
	}}
	h4, err := newTemplateHandler(&buf)
	if err != nil {
		t.Fatal(err)
	}
	h4w := h4.WithAttrs([]slog.Attr{slog.String("k", "with"), slog.String("time", "fake-time")})
	r4 := slog.NewRecord(time.Now(), slog.LevelInfo, "m", 0)
	r4.AddAttrs(slog.String("k", "record"), slog.String("level", "fake-level"))
	if err := h4w.Handle(context.Background(), r4); err != nil {
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

	// 5. WithGroup 嵌套注入（group attr 以 group.key 形式存储，按 '.' 分割构造嵌套 map）。
	buf.Reset()
	tplLoader = &mockLoader{scripts: map[string]string{
		"log.tpl": `{{.req.id}} {{.req.name}}`,
	}}
	h5, err := newTemplateHandler(&buf)
	if err != nil {
		t.Fatal(err)
	}
	h5g := h5.WithGroup("req").WithAttrs([]slog.Attr{slog.String("id", "7"), slog.String("name", "n")})
	if err := h5g.Handle(context.Background(), slog.NewRecord(time.Now(), slog.LevelInfo, "m", 0)); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "7 n") {
		t.Fatalf("WithGroup 嵌套注入失败: %q", buf.String())
	}

	// 5b. 点号 key 的 attr（非 WithGroup）同样按 '.' 分割构造嵌套 map。
	buf.Reset()
	h5b := h5.WithAttrs([]slog.Attr{slog.String("req.id", "9")})
	if err := h5b.Handle(context.Background(), slog.NewRecord(time.Now(), slog.LevelInfo, "m", 0)); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "9") {
		t.Fatalf("点号 key attr 嵌套注入失败: %q", buf.String())
	}

	// 6. WithAttrs/WithGroup 返回新 handler（不共享 buf、不返回自身、不污染原 handler）。
	buf.Reset()
	tplLoader = &mockLoader{scripts: map[string]string{"log.tpl": `{{.time}} {{.k}}`}}
	h6, err := newTemplateHandler(&buf)
	if err != nil {
		t.Fatal(err)
	}
	h6a := h6.WithAttrs([]slog.Attr{slog.String("k", "XYZV")})
	h6aH := h6a.(*templateHandler)
	if h6aH == h6 {
		t.Fatal("WithAttrs 不得返回自身")
	}
	if h6aH.buf == h6.buf {
		t.Fatal("WithAttrs 不得共享原 buf")
	}
	// 原 handler 不受新 handler 影响。
	if err := h6.Handle(context.Background(), slog.NewRecord(time.Now(), slog.LevelInfo, "m", 0)); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(buf.String(), "XYZV") {
		t.Fatal("新 handler 的 attr 不得污染原 handler")
	}
}
