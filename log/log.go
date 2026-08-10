package log

import (
	"log/slog"
	"os"
	"sync"
)

var (
	lg            *slog.Logger   // 当前 logger（保留声明）
	lgLevel       *slog.LevelVar = &slog.LevelVar{}
	fanout        *fanoutWriter
	once          sync.Once
	onLevelChange func(string) // 级别变更钩子（入参为级别字符串）
)

// Debug 记录调试级别日志
func Debug(msg string, args ...any) {
	getLogger().Debug(msg, args...)
}

// Info 记录信息级别日志
func Info(msg string, args ...any) {
	getLogger().Info(msg, args...)
}

// Warn 记录警告级别日志
func Warn(msg string, args ...any) {
	getLogger().Warn(msg, args...)
}

// Error 记录错误级别日志
func Error(msg string, args ...any) {
	getLogger().Error(msg, args...)
}

// getLogger 惰性初始化：console + ring 恒开，输出格式由内置常量模板决定。
func getLogger() *slog.Logger {
	ensureFanout()
	return lg
}

// ensureFanout 保证 fanout/lg 已初始化（sync.Once 幂等）。
// 所有可能被「首次日志调用之前」触发的 API（SetFileWriter/GetInfo/Tail/SetMaxSize）
// 必须先调用 ensureFanout()，避免 fanout==nil 时 nil 解引用 panic。
func ensureFanout() {
	once.Do(func() {
		fanout = &fanoutWriter{
			console: os.Stdout,
			ring:    NewRingBuffer(),
		}
		// 模板启动时加载，运行期定死（见 template.go）。
		lg = buildLogger()
	})
}

// buildLogger 按内置常量模板构建 slog.Logger（启动时定死，运行期不重建）。
func buildLogger() *slog.Logger {
	th, err := newTemplateHandler(fanout) // 解析内置常量模板，模板渲染输出
	if err != nil {
		// 模板加载失败 → 回退 slog 默认 text handler（保证可用）。
		// ⚠️ L1 落定：NewOptions() 内部 `if !isSetLevel { lgLevel.Set(LevelInfo) }` 会把级别重置为 info。
		//   修复：回退前保存 lgLevel.Level()，构建后再恢复（NewOptions 可能把它重置为 info）。
		saved := lgLevel.Level()
		fallback := NewOptions()                               // 含时间格式化的 HandlerOptions（其 Level 已绑定 lgLevel 指针）
		lgLevel.Set(saved)                                     // 恢复级别（NewOptions 可能把它重置为 info）
		return slog.New(slog.NewTextHandler(fanout, fallback)) // fallback.Level 无需再赋（同一指针）
	}
	return slog.New(th)
}

// ResetLogger 重置日志 似乎无法即刻生效。
// 已废弃：fanout 接管输出后重建 handler 会切断 console + ring 通道、破坏基座，故改为 no-op（仅打警告）。
func ResetLogger() {
	Warn("ResetLogger 已废弃，调用无效（输出由 fanout 通道接管）")
}
