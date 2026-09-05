package migrations

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMigrate_UnknownDirection 验证未知方向报错；方向校验先于建连，
// 用空连接串即可证明错误不依赖数据库。
func TestMigrate_UnknownDirection(t *testing.T) {
	err := Migrate("", "sideways")
	require.Error(t, err)
	assert.Contains(t, err.Error(), `unknown migration direction "sideways"`)
}

// TestMigrate_DownRequiresPositiveSteps 验证 DirectionDown 的显式步数必须为正整数。
func TestMigrate_DownRequiresPositiveSteps(t *testing.T) {
	for _, steps := range []int{0, -1} {
		err := Migrate("", DirectionDown, steps)
		require.Error(t, err, "steps=%d 应报错", steps)
		assert.Contains(t, err.Error(), "down steps must be a positive integer")
	}
}

// TestMigrate_UnknownURLScheme 验证未注册的数据库 scheme 在驱动查找阶段即失败
// （golang-migrate 按连接串 scheme 查注册表，不发起网络连接）。
func TestMigrate_UnknownURLScheme(t *testing.T) {
	err := Migrate("mysql://localhost/db", DirectionUp)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown driver")
}

// TestMigrate_UnreachableServer 验证 postgres scheme 走真实驱动建连：
// 不可达端口在迁移执行前返回错误（连接被拒，无需 Docker）。
func TestMigrate_UnreachableServer(t *testing.T) {
	err := Migrate("postgres://lexi:lexi@127.0.0.1:1/lexi_loop?sslmode=disable&connect_timeout=1", DirectionUp)
	require.Error(t, err)
}
