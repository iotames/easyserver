package log

import (
	"bytes"
	"sync"
)

// ringCap 环形缓冲容量，钉死 8MB，不可配置。
const ringCap = 8 << 20 // 8MB

// RingBuffer 内存环形缓冲，作为实时监控数据源（基座，恒开）。
// 写满覆盖最旧；游标为全局单调递增字节 offset。
type RingBuffer struct {
	mu    sync.Mutex
	data  []byte // 容量 ringCap
	total int64  // 已写入总字节数（单调递增，只增不减）
}

// NewRingBuffer 创建容量为 ringCap 的环形缓冲。
func NewRingBuffer() *RingBuffer {
	return &RingBuffer{
		data: make([]byte, ringCap),
	}
}

// Write 追加 p 到缓冲；超容量覆盖最旧。返回 len(p), nil（永不失败）。
func (r *RingBuffer) Write(p []byte) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// 单条超长日志截断：只保留最后 ringCap 字节，避免覆盖整个缓冲。
	// 截断后 total 只累加截断后的长度 ringCap，保证逻辑写入起点与物理环形位置一致。
	if len(p) >= ringCap {
		p = p[len(p)-ringCap:]
	}
	// 写入环形位置 int(total % ringCap)，分两段（尾部 + 头部回绕）写入。
	pos := int(r.total % ringCap)
	written := copy(r.data[pos:], p)
	if written < len(p) {
		written += copy(r.data, p[written:])
	}
	r.total += int64(written)
	return len(p), nil
}

// TailResult 一次增量读取的结果。
type TailResult struct {
	Lines      []string // 完整行（按行对齐，无半行）
	NextOffset int64    // 下一次读取的游标（指向最后返回行的行尾）
	Reset      bool     // since 已被覆盖，需重新拉尾部
	EOF        bool     // 无新数据
}

// Tail 从 since 读取增量，最多 n 行（n<=0 表示不限制）。
// since==-1 表示「尾部首拉」：从窗口尾部向前取最后 n 个完整行，NextOffset 指向窗口最新游标。
// 其余 since 为增量游标。
func (r *RingBuffer) Tail(n int64, since int64) TailResult {
	r.mu.Lock()
	defer r.mu.Unlock()

	// 有效窗口起点。
	validStart := r.total - int64(ringCap)
	if validStart < 0 {
		validStart = 0
	}

	// 尾部首拉（since==-1）必须先拦截并 return（M1），
	// 否则字面上会落入 since < validStart 分支返回 Reset 空行，首次拉取全失效。
	if since == -1 {
		return r.tailFirst(n, validStart)
	}

	// 过期游标判定：显式有序分支，勿合并条件。
	if since > r.total {
		// 旧游标超过当前 total，视作 stale，客户端重新尾部首拉。
		return TailResult{NextOffset: r.total, Reset: true}
	}
	if since == r.total {
		// SSE 起始游标 RingTotal 与 HTTP 增量 next_offset==total 轮询依赖此分支返回 EOF 而非 Reset。
		return TailResult{NextOffset: since, EOF: true}
	}
	if since < validStart {
		return TailResult{NextOffset: r.total, Reset: true}
	}

	// 非行边界判定：仅对 validStart < since < total 成立。
	// since==validStart（含 since==0）视为行起点直接进入增量读取，不做前字节检查——
	// 否则 since==validStart 时前一个字节在窗口外（物理残留旧数据），照字面检查会误 Reset；
	// 且 since==0 时 (0-1)%ringCap==-1 会导致 data[-1] panic。
	// 守卫顺序保证 since==0 永不执行取模（避免数组越界 panic）。
	if since > validStart {
		// 逻辑字节 i 的物理位置恒为 i % ringCap，切勿再加 total。
		prevByte := r.data[(since-1)%ringCap]
		if prevByte != '\n' {
			return TailResult{NextOffset: r.total, Reset: true}
		}
	}

	// 增量读取：从 since 读到 r.total，按 \n 切分，只返回完整行。
	return r.readIncremental(n, since)
}

// tailFirst 尾部首拉：从窗口尾部向前取最后 n 个完整行。
// 扫描下界为 validStart（勿扫到物理 0，避免读到环形残留旧数据污染结果）。
// 末尾存在半行（末字节非 \n）时 NextOffset 指向半行起点，否则为 total。
// 非空窗口恒 EOF=false；仅 total==0 时 EOF=true。
func (r *RingBuffer) tailFirst(n int64, validStart int64) TailResult {
	if r.total == 0 {
		return TailResult{NextOffset: 0, EOF: true}
	}

	window := r.readLogical(validStart, r.total)
	windowLen := int64(len(window))

	// 末尾半行处理（M2）：末字节非 \n 时，尾部存在半行，
	// 完整行区域右边界 end 为半行起点（最后一个 \n 之后的位置）。
	end := windowLen
	if window[windowLen-1] != '\n' {
		pos := windowLen - 1
		for pos > 0 && window[pos-1] != '\n' {
			pos--
		}
		end = pos
	}

	// 从 end 向前扫描完整行边界，收集最后 n 个完整行。
	// 完整行 = 以 '\n' 结尾的段（可能为空行，如 "a\n\nb\n" 中的 "\n"）；
	// 行起点为其前面最近一个 '\n' 之后的位置。
	lines := []string{}
	curEnd := end // 当前行右边界；循环不变式：curEnd>0 时 window[curEnd-1]=='\n'
	for (n <= 0 || int64(len(lines)) < n) && curEnd > 0 {
		pos := curEnd - 2 // 跳过当前行末尾的 '\n'，向前找本行起点
		for pos >= 0 && window[pos] != '\n' {
			pos--
		}
		if pos < 0 {
			// 无更早的 '\n'：行起点为窗口起点 0。
			lines = append(lines, string(window[0:curEnd]))
			break
		}
		lines = append(lines, string(window[pos+1:curEnd]))
		curEnd = pos + 1
	}
	// 收集顺序为从新到旧，需倒转为时间正序。
	for i, j := 0, len(lines)-1; i < j; i, j = i+1, j-1 {
		lines[i], lines[j] = lines[j], lines[i]
	}

	nextOffset := r.total
	if end < windowLen {
		// 存在末尾半行，NextOffset 指向半行起点（逻辑字节 = validStart + end）。
		nextOffset = validStart + end
	}
	return TailResult{Lines: lines, NextOffset: nextOffset, EOF: false}
}

// readIncremental 增量读取逻辑字节区间 [since, total)。
// 只返回完整行；末尾半行不返回、NextOffset 停半行起点；因 n 截断时 NextOffset 指向截断处。
func (r *RingBuffer) readIncremental(n int64, since int64) TailResult {
	data := r.readLogical(since, r.total)
	lines := []string{}
	nextOffset := since
	pos := 0 // data 内偏移，data[i] 对应逻辑字节 since+i
	for pos < len(data) {
		idx := bytes.IndexByte(data[pos:], '\n')
		if idx < 0 {
			// 末尾半行：不返回，NextOffset 停半行起点（M5），客户端继续轮询。
			nextOffset = since + int64(pos)
			break
		}
		line := string(data[pos : pos+idx+1])
		lines = append(lines, line)
		nextOffset = since + int64(pos+idx+1)
		pos += idx + 1
		if n > 0 && int64(len(lines)) >= n {
			break
		}
	}
	return TailResult{Lines: lines, NextOffset: nextOffset, EOF: false}
}

// readLogical 将逻辑字节区间 [from, to) 从物理环形缓冲复制为连续切片（窗口内偏移 i 对应逻辑字节 from+i）。
// 物理位置恒为 i % ringCap，区间可能跨越环的回绕点，故分两段复制。
func (r *RingBuffer) readLogical(from, to int64) []byte {
	length := to - from
	out := make([]byte, length)
	pos := from % ringCap
	first := copy(out, r.data[pos:])
	if int64(first) < length {
		copy(out[first:], r.data[:length-int64(first)])
	}
	return out
}

// Total 已写入总字节数（单调递增，即最新游标）。内部持 r.mu。
func (r *RingBuffer) Total() int64 {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.total
}

// ValidStart 有效窗口起点（= max(0, total - ringCap)）。内部持 r.mu。
func (r *RingBuffer) ValidStart() int64 {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.total <= int64(ringCap) {
		return 0
	}
	return r.total - int64(ringCap)
}

// Used 当前窗口内实际数据字节数（= total - ValidStart）。内部持 r.mu。
func (r *RingBuffer) Used() int64 {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.total <= int64(ringCap) {
		return r.total
	}
	return int64(ringCap)
}
