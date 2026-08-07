package log

import (
	"bytes"
	"testing"
)

// TestRingBufferWriteRead 写入→Tail 增量正确；NextOffset 续读无重复无遗漏。
func TestRingBufferWriteRead(t *testing.T) {
	r := NewRingBuffer()
	r.Write([]byte("a\n"))
	r.Write([]byte("b\n"))

	// 尾部首拉：返回最后 n 个完整行，NextOffset 指向窗口最新游标。
	res := r.Tail(0, -1)
	if res.Reset || res.EOF {
		t.Fatal("首拉不应 Reset/EOF")
	}
	if len(res.Lines) != 2 || res.Lines[0] != "a\n" || res.Lines[1] != "b\n" {
		t.Fatalf("首拉行异常: %v", res.Lines)
	}
	if res.NextOffset != r.Total() {
		t.Fatalf("NextOffset 应等于 total: got %d want %d", res.NextOffset, r.Total())
	}

	// 续读无新数据 → EOF。
	res2 := r.Tail(0, res.NextOffset)
	if !res2.EOF || len(res2.Lines) != 0 {
		t.Fatal("无新数据应 EOF")
	}
	if res2.Reset {
		t.Fatal("EOF 分支不应 Reset")
	}

	// 新数据增量续读，无重复无遗漏。
	r.Write([]byte("c\n"))
	res3 := r.Tail(0, res.NextOffset)
	if len(res3.Lines) != 1 || res3.Lines[0] != "c\n" {
		t.Fatalf("增量续读异常: %v", res3.Lines)
	}
	if res3.NextOffset != r.Total() {
		t.Fatalf("续读 NextOffset 应等于 total: got %d", res3.NextOffset)
	}

	// 旧游标超过当前 total → Reset（stale）。
	res4 := r.Tail(0, r.Total()+100)
	if !res4.Reset || res4.NextOffset != r.Total() {
		t.Fatal("since>total 应 Reset 且 NextOffset=total")
	}
}

// TestRingBufferOverwrite 写满覆盖最旧；旧 since → Reset=true。
func TestRingBufferOverwrite(t *testing.T) {
	r := NewRingBuffer()
	r.Write([]byte("oldline\n"))                                // 8 字节
	chunk := bytes.Repeat([]byte("0123456789\n"), ringCap/11+1) // 超 ringCap
	r.Write(chunk)
	if r.Total() != int64(ringCap)+8 {
		t.Fatalf("total 异常: got %d want %d", r.Total(), int64(ringCap)+8)
	}
	// 旧游标 since=0 已被覆盖出窗口 → Reset。
	res := r.Tail(0, 0)
	if !res.Reset || res.NextOffset != r.Total() {
		t.Fatal("旧游标应 Reset 且 NextOffset=total")
	}
	// 尾部首拉仍可用。
	res2 := r.Tail(5, -1)
	if res2.Reset || res2.EOF {
		t.Fatal("首拉不应 Reset/EOF")
	}
	if len(res2.Lines) != 5 {
		t.Fatalf("应返回 5 行: got %d", len(res2.Lines))
	}
}

// TestRingBufferSinceZero since==0 且 total>0：不 panic、视为行起点进入增量读取（M1 边界）。
func TestRingBufferSinceZero(t *testing.T) {
	r := NewRingBuffer()
	r.Write([]byte("a\nb\n"))

	res := r.Tail(0, 0)
	if res.Reset {
		t.Fatal("since==0 不应 Reset")
	}
	if len(res.Lines) != 2 || res.Lines[0] != "a\n" || res.Lines[1] != "b\n" {
		t.Fatalf("since==0 增量读取异常: %v", res.Lines)
	}
	if res.NextOffset != 4 {
		t.Fatalf("NextOffset 应等于 total: got %d", res.NextOffset)
	}
}

// TestRingBufferSinceValidStart since==validStart：不误 Reset、正常增量读取（M1 边界）。
func TestRingBufferSinceValidStart(t *testing.T) {
	r := NewRingBuffer()
	r.Write([]byte("line-ab\n")) // 8 字节，制造 validStart>0
	chunk := bytes.Repeat([]byte("world\n"), ringCap/6)
	r.Write(chunk)

	vs := r.ValidStart()
	if vs <= 0 {
		t.Fatal("本用例需要 validStart>0 的场景")
	}
	res := r.Tail(5, vs)
	if res.Reset {
		t.Fatal("since==validStart 不应误 Reset（M1 边界）")
	}
	if res.EOF {
		t.Fatal("窗口非空不应 EOF")
	}
	if len(res.Lines) == 0 {
		t.Fatal("应正常增量读取到行")
	}
	// 从返回的最后一行之后续读仍正常（无 Reset）。
	if len(res.Lines) == 5 {
		res2 := r.Tail(5, res.NextOffset)
		if res2.Reset {
			t.Fatal("续读不应 Reset")
		}
	}
}

// TestRingBufferLineAlign 半行不返回；NextOffset 指向完整行边界（行尾或半行起点，非半行中间）；
// 续读无「半行+新行」拼接脏行。
func TestRingBufferLineAlign(t *testing.T) {
	r := NewRingBuffer()
	r.Write([]byte("a\nb\nc")) // 末尾半行 "c"
	res := r.Tail(0, 0)
	if len(res.Lines) != 2 || res.Lines[0] != "a\n" || res.Lines[1] != "b\n" {
		t.Fatalf("半行不应返回: %v", res.Lines)
	}
	if res.NextOffset != 4 {
		t.Fatalf("NextOffset 应停在半行起点 4: got %d", res.NextOffset)
	}
	if res.EOF {
		t.Fatal("存在半行未就绪，不应 EOF")
	}

	// 半行补全后续读，无「半行+新行」拼接脏行。
	r.Write([]byte("d\n"))
	res2 := r.Tail(0, res.NextOffset)
	if len(res2.Lines) != 1 || res2.Lines[0] != "cd\n" {
		t.Fatalf("续读应拼接为完整行 cd\\n: %v", res2.Lines)
	}
	if res2.NextOffset != 7 {
		t.Fatalf("续读 NextOffset 应为 7: got %d", res2.NextOffset)
	}

	// 尾部首拉同样遵循半行语义：NextOffset 指向半行起点。
	r2 := NewRingBuffer()
	r2.Write([]byte("x\ny\nz")) // 末尾半行 "z"
	res3 := r2.Tail(0, -1)
	if len(res3.Lines) != 2 || res3.Lines[0] != "x\n" || res3.Lines[1] != "y\n" {
		t.Fatalf("尾部首拉半行不应返回: %v", res3.Lines)
	}
	if res3.NextOffset != 4 {
		t.Fatalf("尾部首拉 NextOffset 应停在半行起点 4: got %d", res3.NextOffset)
	}
	if res3.EOF {
		t.Fatal("尾部首拉存在半行不应 EOF")
	}
}

// TestRingBufferHugeLine 单条超 ringCap 截断为最后 ringCap 字节，total 只累加 ringCap。
func TestRingBufferHugeLine(t *testing.T) {
	r := NewRingBuffer()
	big := make([]byte, ringCap+100)
	for i := range big {
		big[i] = 'a'
	}
	big[len(big)-1] = '\n' // 截断段以 \n 结尾
	r.Write(big)

	if r.Total() != int64(ringCap) {
		t.Fatalf("超长截断后 total 应等于 ringCap: got %d", r.Total())
	}
	// 截断段（最后 ringCap 字节）以 \n 结尾，尾部首拉应返回 1 行，内容为原日志后半段。
	res := r.Tail(0, -1)
	if len(res.Lines) != 1 {
		t.Fatalf("截断段应以完整行返回: got %d 行", len(res.Lines))
	}
	expected := string(big[len(big)-ringCap:])
	if res.Lines[0] != expected {
		t.Fatal("截断保留的最后 ringCap 字节不一致")
	}
}
