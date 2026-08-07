package log

import (
	"io"
	"sync"
)

// fanoutWriter 多路分发：console + ring（恒开）+ file（可选）。
type fanoutWriter struct {
	mu      sync.RWMutex
	console io.Writer   // 恒开（stdout）
	ring    *RingBuffer // 恒开（监控通道）
	file    *fileWriter // nil = 未开文件
}

// Write 分发到所有通道；任一通道失败不影响其他（红线：文件故障不影响 console + ring 与主业务）。
func (f *fanoutWriter) Write(p []byte) (int, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	_, _ = f.console.Write(p)
	_, _ = f.ring.Write(p)
	if f.file != nil {
		_, _ = f.file.Write(p)
	}
	return len(p), nil
}

// SetFile 开启/关闭文件通道（nil 关闭）。
// 旧 f.file 若非 nil：Close()（确定性关闭，无在途写入——因持写锁，无并发写）。
func (f *fanoutWriter) SetFile(w *fileWriter) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.file != nil {
		_ = f.file.Close()
	}
	f.file = w
}

// File 返回当前文件通道（nil 表示未开）。持读锁，避免与 SetFile 写竞争。
func (f *fanoutWriter) File() *fileWriter {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.file
}
