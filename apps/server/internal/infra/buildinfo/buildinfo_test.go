package buildinfo

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// 未注入 ldflags 时的默认值即「开发构建」契约：版本以 dev 占位、构建时间
// 为空（docs/api/meta.md §2）；测试环境不做 ldflags 注入，正好锁定该形态。
func TestDefaults(t *testing.T) {
	assert.Equal(t, "dev", Version)
	assert.Empty(t, BuildTime)
	assert.True(t, strings.HasPrefix(GoVersion(), "go"), "GoVersion 返回 runtime.Version() 形如 go1.26.7")
}
