package log

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestFileTruncate maxSize=1 连写超限 → 文件清空并留标记（后台异步）；maxSize=0 不限制。
func TestFileTruncate(t *testing.T) {
	dir := t.TempDir()

	// maxSize=1：超限清空并留标记。
	path := filepath.Join(dir, "trunc.log")
	w, err := newFileWriter(path, 1)
	if err != nil {
		t.Fatal(err)
	}
	w.Write([]byte("first-line\n"))
	w.Write([]byte("second-line\n"))
	w.Write([]byte("third-line\n"))
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	if !strings.Contains(s, "[log] 文件超限已清空") {
		t.Fatal("超限清空应留标记")
	}
	if strings.Contains(s, "first-line") {
		t.Fatal("超限后旧内容应被清空")
	}
	if !strings.Contains(s, "third-line") {
		t.Fatal("最后一条应保留")
	}

	// maxSize=0：不限制。
	path2 := filepath.Join(dir, "nolimit.log")
	w2, err := newFileWriter(path2, 0)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 100; i++ {
		w2.Write([]byte("keep-line\n"))
	}
	w2.Close()
	b2, err := os.ReadFile(path2)
	if err != nil {
		t.Fatal(err)
	}
	s2 := string(b2)
	if !strings.Contains(s2, "keep-line") {
		t.Fatal("maxSize=0 不应截断")
	}
	if strings.Count(s2, "keep-line") != 100 {
		t.Fatalf("maxSize=0 应保留全部 100 条: got %d", strings.Count(s2, "keep-line"))
	}
}

// TestFileAsync 文件队列满/写入失败时 console+ring 不受影响；dropped 计数递增；Close 排空后文件完整。
func TestFileAsync(t *testing.T) {
	// 1. 队列满丢弃 + dropped 计数（未启动 writeLoop，q 不会被排空，确定性验证）。
	pr, pw, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	w := &fileWriter{f: pw, maxSize: 0, q: make(chan []byte, 4)}
	for i := 0; i < 4; i++ {
		if _, err := w.Write([]byte("item\n")); err != nil {
			t.Fatal(err)
		}
	}
	before := w.dropped.Load()
	w.Write([]byte("drop\n"))
	if w.dropped.Load()-before != 1 {
		t.Fatal("队列满应丢弃并计数")
	}

	// 2. 启动 writeLoop 并 Close：排空后文件完整（经 pipe 读端验证）。
	w.wg.Add(1)
	go w.writeLoop()
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	tmp := make([]byte, 512)
	for {
		n, rerr := pr.Read(tmp)
		buf.Write(tmp[:n])
		if rerr != nil {
			break
		}
	}
	if buf.String() != "item\nitem\nitem\nitem\n" {
		t.Fatalf("Close 排空后文件不完整: %q", buf.String())
	}

	// 3. 文件写入失败不影响 console + ring（红线：异步隔离）。
	console := &bytes.Buffer{}
	ring := NewRingBuffer()
	devFull, ferr := os.OpenFile("/dev/full", os.O_WRONLY, 0)
	if ferr != nil {
		t.Fatal(ferr)
	}
	fw := &fileWriter{f: devFull, maxSize: 0, q: make(chan []byte, 4)}
	fw.wg.Add(1)
	go fw.writeLoop()
	f := &fanoutWriter{console: console, ring: ring, file: fw}
	n, werr := f.Write([]byte("red-line\n"))
	if werr != nil || n != 9 {
		t.Fatal("fanout Write 不应受文件写入失败影响")
	}
	if console.String() != "red-line\n" {
		t.Fatal("console 应正常输出")
	}
	res := ring.Tail(0, -1)
	if len(res.Lines) != 1 || res.Lines[0] != "red-line\n" {
		t.Fatal("ring 应正常接收")
	}
	fw.Close()
}
