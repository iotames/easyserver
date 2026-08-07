package log

import (
	"log/slog"
	"strings"
	"testing"
	"time"
)

// setLevelForTest 设置级别并在测试结束后恢复原级别（避免污染后续用例）。
func setLevelForTest(t *testing.T, l slog.Level) {
	prev := lgLevel.Level()
	SetLevel(l)
	t.Cleanup(func() {
		SetLevel(prev)
	})
}

// TestLevelHook SetLevel 值变化触发钩子；相同值不触发；超过 error 钳制为 error。
func TestLevelHook(t *testing.T) {
	var changed []string
	SetOnLevelChange(func(level string) { changed = append(changed, level) })
	defer SetOnLevelChange(nil)
	setLevelForTest(t, slog.LevelInfo)

	SetLevel(slog.LevelDebug)
	SetLevel(slog.LevelDebug) // 相同值不触发
	SetLevel(slog.LevelWarn)
	if len(changed) != 2 || changed[0] != "DEBUG" || changed[1] != "WARN" {
		t.Fatalf("钩子触发异常: %v", changed)
	}

	// 钳制：超过 error 钳制为 error，并触发钩子（基座约束）。
	SetLevel(slog.LevelError + 100)
	if lgLevel.Level() != slog.LevelError {
		t.Fatal("应钳制为 error")
	}
	if len(changed) != 3 || changed[2] != "ERROR" {
		t.Fatalf("钳制应触发钩子: %v", changed)
	}
}

// TestCompatRegression 默认路径（无增强）输出关键字段与行结构贴近现状
// （time=... level=... msg=...，time 用 time.DateTime 格式）；
// 模板未引用的 attr 不输出（默认模板即丢弃全部 attr，属既定取舍）；15 处调用方行为不变。
func TestCompatRegression(t *testing.T) {
	setLevelForTest(t, slog.LevelDebug)
	// 确保默认路径：无文件增强。
	_ = SetFileWriter(false)

	Info("compat-regression", "attr1", "v1", "attr2", "v2")
	res := Tail(100, -1)
	var found string
	for _, l := range res.Lines {
		if strings.Contains(l, "compat-regression") {
			found = l
			break
		}
	}
	if found == "" {
		t.Fatal("未找到 compat-regression 日志")
	}
	// 关键字段：time=... level=... msg=...。
	if !strings.HasPrefix(found, "time=") {
		t.Fatalf("应以 time= 开头: %q", found)
	}
	if !strings.Contains(found, "level=INFO") {
		t.Fatalf("应含 level=INFO: %q", found)
	}
	if !strings.Contains(found, "msg=compat-regression") {
		t.Fatalf("应含 msg=compat-regression: %q", found)
	}
	// time 为 time.DateTime 格式（YYYY-MM-DD HH:MM:SS）。
	timePart := strings.TrimPrefix(found, "time=")
	idx := strings.Index(timePart, " level=")
	if idx != len("2006-01-02 15:04:05") {
		t.Fatalf("time 字段长度异常: %q", timePart)
	}
	timeStr := timePart[:idx]
	if _, err := time.Parse(time.DateTime, timeStr); err != nil {
		t.Fatalf("time 字段非 time.DateTime 格式: %q", timeStr)
	}
	// 模板未引用 attr 不输出（默认模板即丢弃全部 attr）。
	if strings.Contains(found, "attr1") || strings.Contains(found, "attr2") {
		t.Fatal("默认模板不应输出未引用的 attr")
	}
}
