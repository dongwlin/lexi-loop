package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dongwlin/lexi-loop/apps/server/internal/infra/buildinfo"
)

// executeVersion 以给定参数运行 version 子命令并返回 stdout 输出。
func executeVersion(t *testing.T, args ...string) string {
	t.Helper()
	var out bytes.Buffer
	root := newRootCmd()
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs(append([]string{"version"}, args...))
	require.NoError(t, root.Execute())
	return out.String()
}

func TestVersionCommand(t *testing.T) {
	t.Run("默认只输出版本号", func(t *testing.T) {
		assert.Equal(t, buildinfo.Version+"\n", executeVersion(t))
	})

	t.Run("b 标志携带构建信息", func(t *testing.T) {
		// 临时注入构建时间验证输出格式，结束后恢复。
		origTime := buildinfo.BuildTime
		buildinfo.BuildTime = "2026-09-07T00:00:00Z"
		t.Cleanup(func() { buildinfo.BuildTime = origTime })

		lines := strings.Split(strings.TrimSuffix(executeVersion(t, "-b"), "\n"), "\n")
		require.Len(t, lines, 3)
		assert.Equal(t, "version: "+buildinfo.Version, lines[0])
		assert.Equal(t, "build_time: 2026-09-07T00:00:00Z", lines[1])
		assert.Equal(t, "go_version: "+buildinfo.GoVersion(), lines[2])
	})

	t.Run("未注入构建时间时以 - 占位", func(t *testing.T) {
		origTime := buildinfo.BuildTime
		buildinfo.BuildTime = ""
		t.Cleanup(func() { buildinfo.BuildTime = origTime })

		out := executeVersion(t, "--build")
		assert.Contains(t, out, "build_time: -")
	})
}
