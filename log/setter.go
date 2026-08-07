package log

import (
	"errors"
	"io"
	"log/slog"
	"os"
	"sync"
	"time"
)

var (
	logWriter  io.Writer
	opts       *slog.HandlerOptions
	isSetLevel bool

	// 包级可变状态用 mutex 保护：SetLogWriterByFile/SetMaxSize 写、SetFileWriter/GetInfo 读，
	// watcher 回调 goroutine 与 admin handler 并发时不得有数据竞争。
	stateMu      sync.Mutex                      // 保护 filePath/maxSize（模板文件名启动时定死，不需锁）
	templateFile            = "log.tpl"          // 模板文件名（经 ScriptDir 兜底加载）
	filePath                = "logs/rocksys.log" // 文件存档路径（写时持 stateMu）
	maxSize      int64      = 50 * 1024 * 1024   // 默认 50MB（写时持 stateMu）
)

// SetLevel 设置级别；级别真正变化时触发钩子。
// ★ 基座约束：级别最高只能设到 error——若传入高于 error 的级别，钳制为 error。
//
//	（即使钳制失效，templateHandler.Enabled 仍对 >=error 无条件放行，双保险）
func SetLevel(level slog.Level) {
	if level > slog.LevelError {
		level = slog.LevelError
	}
	old := lgLevel.Level()
	lgLevel.Set(level)
	if old != level && onLevelChange != nil {
		onLevelChange(level.String())
	}
}

// SetOnLevelChange 注册级别变更钩子（rocksys 注入持久化回调）。
// 仅装配期单线程调用一次，onLevelChange 无锁可接受。
func SetOnLevelChange(fn func(string)) {
	onLevelChange = fn
}

// SetFileWriter 开启/关闭文件存档通道（E1）。
// ★ 内部实现 setFileWriterUnlocked 不持 stateMu，供 SetLogWriterByFile 复用（避免死锁）。
func SetFileWriter(on bool) error {
	ensureFanout()
	stateMu.Lock()
	defer stateMu.Unlock()
	return setFileWriterUnlocked(on)
}

// setFileWriterUnlocked 开启/关闭文件通道（调用方须已持 stateMu）。
// 禁止在本函数内再次 stateMu.Lock()——sync.Mutex 不可重入，否则与 SetLogWriterByFile 嵌套调用时死锁。
func setFileWriterUnlocked(on bool) error {
	if on {
		if fanout.File() != nil {
			return nil // 已开
		}
		fw, err := newFileWriter(filePath, maxSize)
		if err != nil {
			return err
		}
		fanout.SetFile(fw)
	} else {
		fanout.SetFile(nil)
	}
	return nil
}

// SetLogWriterByFile 保留：创建文件并开启文件通道。
// ★ 不直接调用 SetFileWriter（两者都持 stateMu），改为调无锁内部实现，避免自死锁。
// ★ 返回值仅为「兼容旧语义」保留：返回的 *os.File 归 log 包托管，调用方**不应** Close 它
//
//	（否则会关闭 fanout 正在使用的句柄）。
func SetLogWriterByFile(path string) (*os.File, error) {
	ensureFanout()
	stateMu.Lock()
	defer stateMu.Unlock()
	if path == "" {
		return nil, errors.New("log: empty file path")
	}
	filePath = path
	if err := setFileWriterUnlocked(true); err != nil {
		return nil, err
	}
	return fanout.File().f, nil // 返回底层句柄（兼容旧语义）
}

// SetMaxSize 设置文件大小上限（E2，整数 MB，0=不限制；负数按 0 处理=不限制）。
func SetMaxSize(mb int64) {
	ensureFanout() // 兜底初始化
	stateMu.Lock()
	defer stateMu.Unlock()
	if mb < 0 {
		mb = 0
	}
	maxSize = mb * 1024 * 1024
	if fanout.File() != nil {
		fanout.File().SetMaxSize(maxSize)
	}
}

// GetInfo 返回当前日志状态（先 ensureFanout 兜底初始化）。
func GetInfo() LogInfo {
	ensureFanout()
	stateMu.Lock()
	defer stateMu.Unlock()
	// RingCap/RingTotal/RingUsed/RingValidStart 三个访问器各自持 r.mu，快照可能非同一时刻
	// （仅诊断字段，无碍）。FileOn 由 fanout.File()!=nil 判定，不得读 fileWriter 内部字段。
	return LogInfo{
		Level:          lgLevel.Level().String(),
		Template:       templateFile,
		FileOn:         fanout.File() != nil,
		FilePath:       filePath,
		MaxSizeMB:      maxSize / (1024 * 1024),
		RingCap:        int64(ringCap),
		RingUsed:       fanout.ring.Used(),
		RingTotal:      fanout.ring.Total(),
		RingValidStart: fanout.ring.ValidStart(),
	}
}

// Tail 读 ring buffer 增量（供 HTTP tail / SSE）。
// 客户端首次请求时 since 缺省：统一由 admin 层 parseSince 返回 -1 → Tail 内部「尾部首拉」
// （取窗口尾部最后 n 行）。切勿改用 GetInfo().RingValidStart——那会取窗口最旧 n 行，与「尾部」契约相反。
func Tail(n int64, since int64) TailResult {
	ensureFanout()
	return fanout.ring.Tail(n, since)
}

// SetLogWriter 设置日志输出
//
//	f, err = os.OpenFile("netguard.log", os.O_CREATE|os.O_APPEND, 0644)
//	if err != nil {
//		panic(err)
//	}
//	defer f.Close()
//	log.SetLogWriter(f)
//
// Deprecated: 输出由 fanout 通道接管，本 API 不再生效。
func SetLogWriter(writer io.Writer) {
	logWriter = writer
}

// SetOptions 设置日志选项
//
//	import "log/slog"
//	lgLevel = &slog.LevelVar{}
//	sopts := &slog.HandlerOptions{
//		Level: lgLevel,
//		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
//			// 如果当前属性是时间戳
//			if a.Key == slog.TimeKey && len(groups) == 0 {
//				a.Key = "time" // 键名可以保持不变或修改
//				// 将时间值转换为自定义格式
//				if t, ok := a.Value.Any().(time.Time); ok {
//					a.Value = slog.StringValue(t.Format(time.DateTime))
//				}
//			}
//			return a
//		},
//	}
//	SetOptions(sopts)
//
// Deprecated: 输出由 fanout 通道接管，本 API 不再生效。
func SetOptions(sopts *slog.HandlerOptions) {
	opts = sopts
}

func NewOptions() *slog.HandlerOptions {
	// 设置 HandlerOptions，自定义时间属性
	if !isSetLevel {
		// 设置默认日志级别为 Info
		lgLevel.Set(slog.LevelInfo)
	}
	return &slog.HandlerOptions{
		Level: lgLevel,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			// 如果当前属性是时间戳
			if a.Key == slog.TimeKey && len(groups) == 0 {
				a.Key = "time" // 键名可以保持不变或修改
				// 将时间值转换为自定义格式
				if t, ok := a.Value.Any().(time.Time); ok {
					a.Value = slog.StringValue(t.Format(time.DateTime))
				}
			}
			return a
		},
	}
}

// LogInfo 日志系统当前状态（供 GET /admin/log/info）。
// ⚠️ 命名说明：文档原型为 `Info`，但包内已存在日志函数 `Info(msg, ...)`
//
//	（17 处调用方依赖），Go 包级命名空间唯一不可共存，故类型命名为 LogInfo；
//	字段与 JSON tag 按规格原样保留，P3 通过 GetInfo() 字段访问不受影响。
type LogInfo struct {
	Level          string `json:"level"`
	Template       string `json:"template"` // 当前模板文件名（log.tpl）
	FileOn         bool   `json:"file_on"`
	FilePath       string `json:"file_path"`
	MaxSizeMB      int64  `json:"max_size_mb"`
	RingCap        int64  `json:"ring_cap"`
	RingUsed       int64  `json:"ring_used"`
	RingTotal      int64  `json:"ring_total"`
	RingValidStart int64  `json:"ring_valid_start"` // 仅诊断用（有效窗口起点）；客户端首拉**不得**使用
}
