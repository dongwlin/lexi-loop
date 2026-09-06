// Package logger 基于 zerolog 初始化应用日志：以 ConsoleWriter 输出人类可读
// 的控制台格式（时间 / 级别 / 消息 / 字段），并提供接管 zerolog 全局默认
// logger 的入口，使 migrations 等直接使用全局 log 的包获得一致的输出
// （docs/backend/structure.md §4.5：infra 只包含纯技术组件，由组合根组装）。
package logger

import (
	"io"
	"os"
	"time"

	"github.com/mattn/go-isatty"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// DefaultLevel 是推荐的最低日志级别：cmd 组装 Options 时显式传入；需要
// 排查问题时改为 zerolog.DebugLevel。
//
// 注意 zerolog.Level 的零值是 DebugLevel，Options 零值不会改写它——
// 不希望 Debug 输出时不要传零值 Level。
const DefaultLevel = zerolog.InfoLevel

// consoleTimeFormat 只控制 ConsoleWriter 的时间列展示；JSON 事件内部仍用
// zerolog 默认的 TimeFieldFormat（RFC3339）。
const consoleTimeFormat = time.DateTime

// Options 控制 Logger 的构造。
type Options struct {
	// Level 是最低输出级别，低于该级别的日志被丢弃。
	Level zerolog.Level

	// Out 是日志输出目标；nil 时为 os.Stderr。
	Out io.Writer

	// NoColor 强制关闭 ANSI 颜色。false 时按输出目标自动判断：仅当 Out
	// 是终端时着色，重定向到文件 / 管道不产生转义序列（zerolog 的
	// ConsoleWriter 自身不做终端检测）。
	NoColor bool
}

// New 构造使用 ConsoleWriter 的 zerolog.Logger，自带时间戳上下文，
// 时间格式为 consoleTimeFormat；级别过滤见 Options.Level。
func New(opts Options) zerolog.Logger {
	if opts.Out == nil {
		opts.Out = os.Stderr
	}
	w := zerolog.ConsoleWriter{
		Out:        opts.Out,
		NoColor:    opts.NoColor || !isTerminal(opts.Out),
		TimeFormat: consoleTimeFormat,
	}
	return zerolog.New(w).With().Timestamp().Logger().Level(opts.Level)
}

// Init 在 New 的基础上接管全局默认日志：替换 zerolog/log 的全局 Logger 并
// 调用 zerolog.SetGlobalLevel，使 migrations 等直接使用全局 log 的包输出
// 一致的 ConsoleWriter 格式。返回构造的 Logger，供组合根进一步传递给需要
// 显式依赖日志的组件（如未来的请求日志中间件）。
func Init(opts Options) zerolog.Logger {
	zerolog.SetGlobalLevel(opts.Level)
	log.Logger = New(opts)
	return log.Logger
}

// isTerminal 报告 w 是否为终端；非 *os.File（测试缓冲等）视为非终端。
func isTerminal(w io.Writer) bool {
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	fd := f.Fd()
	return isatty.IsTerminal(fd) || isatty.IsCygwinTerminal(fd)
}
