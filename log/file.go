package log

import (
	"io"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
)

// fileWriter 文件存档 writer（仅供存档，不参与实时监控）。异步：入队即返回。
// 红线原则：文件故障（磁盘满/权限/慢写）只影响后台落盘 goroutine，绝不阻塞 console + ring 主通道与主业务。
type fileWriter struct {
	mu      sync.Mutex // 保护 maxSize / closed / q
	f       *os.File   // 仅在后台 goroutine 内访问（落盘线程单写，无需每写加锁）
	path    string
	maxSize int64          // 字节上限，0=不限制
	q       chan []byte    // 写队列（有界，默认 1024）
	dropped atomic.Int64   // 丢弃计数（队列满时）
	wg      sync.WaitGroup // 后台 goroutine 生命周期
	closed  bool
}

// newFileWriter 打开（或创建）path 文件并启动后台落盘 goroutine；maxSize 为字节上限（0=不限）。
// ★ 若父目录不存在，先 os.MkdirAll(filepath.Dir(path))——os.OpenFile 不建父目录，全新环境开 E1 会失败。
// ★ 以 os.O_CREATE|os.O_APPEND|os.O_WRONLY 打开（O_APPEND 保证多实例/外部 logrotate 截断后仍追加到文件尾）。
func newFileWriter(path string, maxSize int64) (*fileWriter, error) {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, err
	}
	w := &fileWriter{
		f:       f,
		path:    path,
		maxSize: maxSize,
		q:       make(chan []byte, 1024),
	}
	w.wg.Add(1)
	go w.writeLoop()
	return w, nil
}

// Write 实现 io.Writer：**入队立即返回 len(p), nil**（不等待落盘）。
// 队列满时丢弃并计数（dropped），返回 len(p), nil（主通道永不阻塞）。
func (w *fileWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.closed {
		return len(p), nil
	}
	select {
	case w.q <- p:
	default:
		w.dropped.Add(1)
	}
	return len(p), nil
}

// Close 关闭：停收新写入、等待后台 goroutine 排空后关闭文件句柄。
func (w *fileWriter) Close() error {
	w.mu.Lock()
	if w.closed {
		w.mu.Unlock()
		return nil
	}
	w.closed = true
	close(w.q)
	w.mu.Unlock()
	w.wg.Wait()
	return w.f.Close()
}

// SetMaxSize 热切更新大小上限（字节）。持锁写，后台 goroutine 持锁读。
func (w *fileWriter) SetMaxSize(n int64) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.maxSize = n
}

// writeLoop 单写线程：串行处理队列，执行 stat→truncate→append。
// ⚠️ 本 goroutine 内**禁止调 log.Error**（递归风险，见 §1.3 fanout 说明）——失败仅计数/忽略。
func (w *fileWriter) writeLoop() {
	defer w.wg.Done()
	for p := range w.q {
		w.mu.Lock()
		maxSize := w.maxSize
		w.mu.Unlock()
		// 写前检查大小，超限清空再写。
		if maxSize > 0 {
			fi, err := w.f.Stat()
			if err != nil {
				// Stat 失败（外部 logrotate 已删文件）跳过截断检查直接追加——O_APPEND 对已删 inode 无害。
			} else if fi.Size() >= maxSize {
				// TODO：stat → truncate → append 非原子，多 fileWriter 实例共享同一文件时
				// 可能都截断丢数据。当前单实例 + 单写线程实际串行，跨实例竞态一期不处理。
				_ = w.f.Truncate(0)
				_, _ = w.f.Seek(0, io.SeekStart)
				_, _ = w.f.WriteString("[log] 文件超限已清空\n")
			}
		}
		_, _ = w.f.Write(p) // 落盘失败：忽略（红线——不影响主通道）。
	}
}
