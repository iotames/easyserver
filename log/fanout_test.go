package log

import (
	"bytes"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestFanoutBase SetFileWriter(false) 后 Info/Error 仍写 console+ring。
func TestFanoutBase(t *testing.T) {
	// 直接构造 fanout，验证 console + ring 恒开分发。
	console := &bytes.Buffer{}
	ring := NewRingBuffer()
	f := &fanoutWriter{console: console, ring: ring}
	n, err := f.Write([]byte("hello\n"))
	if err != nil || n != 6 {
		t.Fatal("fanout Write 异常")
	}
	if console.String() != "hello\n" {
		t.Fatal("console 通道未收到")
	}
	res := ring.Tail(0, -1)
	if len(res.Lines) != 1 || res.Lines[0] != "hello\n" {
		t.Fatal("ring 通道未收到")
	}

	// 关闭文件通道后，Info/Error 仍写 console+ring（经全局 API 验证 ring 侧）。
	setLevelForTest(t, slog.LevelDebug)
	if err := SetFileWriter(false); err != nil {
		t.Fatal(err)
	}
	Info("fanout-base-info")
	Error("fanout-base-error")
	res2 := Tail(100, -1)
	var gotInfo, gotErr bool
	for _, l := range res2.Lines {
		if strings.Contains(l, "fanout-base-info") {
			gotInfo = true
		}
		if strings.Contains(l, "fanout-base-error") {
			gotErr = true
		}
	}
	if !gotInfo || !gotErr {
		t.Fatal("SetFileWriter(false) 后 Info/Error 未写入 ring（基座通道应恒开）")
	}
	info := GetInfo()
	if info.FileOn {
		t.Fatal("关闭文件后 FileOn 应为 false")
	}
}

// TestFanoutFile 开文件后 console+ring 立即输出、file 异步落盘；
// 开文件后 GetInfo/Tail 仍可用（无死锁）。
func TestFanoutFile(t *testing.T) {
	setLevelForTest(t, slog.LevelDebug)
	// 清理可能的残留文件通道。
	_ = SetFileWriter(false)

	dir := t.TempDir()
	path := filepath.Join(dir, "logs", "rocksys-test.log") // 父目录不存在，验证 MkdirAll
	f, err := SetLogWriterByFile(path)
	if err != nil {
		t.Fatal(err)
	}
	_ = f // 返回句柄归 log 包托管，调用方不应 Close。

	// 开文件后 FileOn 立即生效。
	info := GetInfo()
	if !info.FileOn {
		t.Fatal("开文件后 FileOn 应为 true")
	}
	if info.FilePath != path {
		t.Fatalf("FilePath 异常: %s", info.FilePath)
	}

	// console+ring 立即输出。
	Info("fanout-file-msg")
	res := Tail(100, -1)
	found := false
	for _, l := range res.Lines {
		if strings.Contains(l, "fanout-file-msg") {
			found = true
		}
	}
	if !found {
		t.Fatal("开文件后 Info 未写入 ring")
	}

	// file 异步落盘（后台 goroutine）。
	deadline := time.Now().Add(3 * time.Second)
	for {
		b, rerr := os.ReadFile(path)
		if rerr == nil && strings.Contains(string(b), "fanout-file-msg") {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("file 未异步落盘")
		}
		time.Sleep(10 * time.Millisecond)
	}

	// 开文件后 GetInfo/Tail 仍可用（无死锁）。
	if GetInfo().RingTotal <= 0 {
		t.Fatal("GetInfo 在文件通道开启时应可用")
	}

	// 关闭文件通道。
	if err := SetFileWriter(false); err != nil {
		t.Fatal(err)
	}
	if GetInfo().FileOn {
		t.Fatal("关闭后 FileOn 应为 false")
	}
}
