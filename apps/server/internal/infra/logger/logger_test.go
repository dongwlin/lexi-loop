package logger

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNew_控制台格式与级别过滤 验证 New 输出 ConsoleWriter 人类可读格式
// （非 JSON、含级别 / 消息 / 字段 / 时间列），并按 Options.Level 过滤。
func TestNew_控制台格式与级别过滤(t *testing.T) {
	var buf bytes.Buffer
	l := New(Options{Level: zerolog.InfoLevel, Out: &buf})

	l.Info().Str("key", "value").Msg("请求完成")

	out := buf.String()
	assert.Contains(t, out, "INF")
	assert.Contains(t, out, "请求完成")
	assert.Contains(t, out, "key=value")
	assert.Regexp(t, `\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}`, out)
	assert.NotContains(t, out, `"level"`, "不应输出 JSON")

	buf.Reset()
	l.Debug().Msg("低于级别被丢弃")
	assert.Empty(t, buf.String())

	buf.Reset()
	l.Warn().Msg("达到级别保留")
	assert.Contains(t, buf.String(), "WRN")
}

// TestNew_非终端不着色 验证非终端输出目标（缓冲、普通文件）不产生 ANSI
// 转义序列——重定向到文件 / 管道时 ConsoleWriter 不应带颜色码。
func TestNew_非终端不着色(t *testing.T) {
	var buf bytes.Buffer
	l := New(Options{Out: &buf})
	l.Info().Msg("缓冲输出")
	assert.NotContains(t, buf.String(), "\x1b[")

	f, err := os.Create(filepath.Join(t.TempDir(), "log.txt"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = f.Close() })
	l = New(Options{Out: f})
	l.Info().Msg("文件输出")
	assert.NotContains(t, buf.String(), "\x1b[")
}

// TestInit_接管全局日志 验证 Init 替换 zerolog/log 全局 Logger、设置全局
// 级别，并返回可显式传递的 Logger；结束时还原全局状态避免污染其它用例。
func TestInit_接管全局日志(t *testing.T) {
	origLogger, origLevel := log.Logger, zerolog.GlobalLevel()
	t.Cleanup(func() {
		log.Logger = origLogger
		zerolog.SetGlobalLevel(origLevel)
	})

	var buf bytes.Buffer
	l := Init(Options{Out: &buf, Level: zerolog.InfoLevel})

	log.Info().Str("scope", "migrations").Msg("全局接管")
	assert.Contains(t, buf.String(), "全局接管")
	assert.Contains(t, buf.String(), "scope=migrations")
	assert.Equal(t, zerolog.InfoLevel, zerolog.GlobalLevel())

	buf.Reset()
	log.Debug().Msg("低于全局级别被丢弃")
	assert.Empty(t, buf.String())

	buf.Reset()
	l.Warn().Msg("显式传递")
	assert.Contains(t, buf.String(), "WRN")
}

// TestIsTerminal_非终端目标 验证 isTerminal 对非终端目标返回 false；
// 终端目标无法在测试环境可靠构造，不做正向断言。
func TestIsTerminal_非终端目标(t *testing.T) {
	assert.False(t, isTerminal(&bytes.Buffer{}))
	assert.False(t, isTerminal(io.Discard))

	f, err := os.Create(filepath.Join(t.TempDir(), "f.txt"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = f.Close() })
	assert.False(t, isTerminal(f))
}
