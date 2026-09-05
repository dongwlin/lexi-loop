package migrations

// 集成测试：共享 PostgreSQL 容器由 main_test.go 的包级 TestMain 启动，
// Docker 不可用或传 -short 时集成用例跳过。
// 覆盖 Go 测试规范 §2 的迁移基线：空库可完整 up；并验证版本簿记、
// 重复 up 的 ErrNoChange 幂等、down 按步回退与最低版本边界。

import (
	"context"
	"io/fs"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// migrationVersion 连接容器数据库读取 schema_migrations 当前版本；
// 尚无任何已应用版本（最低版本）时返回 (0, false)。
func migrationVersion(t *testing.T, connStr string) (uint, bool) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	conn, err := pgx.Connect(ctx, connStr)
	require.NoError(t, err, "应能连接容器数据库")
	defer func() { _ = conn.Close(ctx) }()

	var version uint
	var dirty bool
	err = conn.QueryRow(ctx, "SELECT version, dirty FROM schema_migrations").Scan(&version, &dirty)
	if err != nil {
		require.ErrorIs(t, err, pgx.ErrNoRows, "schema_migrations 查询失败应为空表")
		return 0, false
	}
	return version, dirty
}

// expectedVersion 统计内嵌迁移的 up 文件数作为最新版本号（版本号按 000001 起连续编号），
// 避免后续追加 000003 等迁移时用例失效。
func expectedVersion(t *testing.T) uint {
	t.Helper()

	matches, err := fs.Glob(migrationFS, "*.up.sql")
	require.NoError(t, err)
	require.NotEmpty(t, matches, "本目录应存在至少一个 up 迁移文件")
	return uint(len(matches))
}

// TestIntegration_MigrateUp 验证空库可完整 up，且重复 up 幂等（ErrNoChange 视为成功）。
func TestIntegration_MigrateUp(t *testing.T) {
	connStr := requireTestConnStr(t)
	want := expectedVersion(t)

	require.NoError(t, Migrate(connStr, DirectionUp))

	version, dirty := migrationVersion(t, connStr)
	assert.Equal(t, want, version, "up 应应用到全部迁移")
	assert.False(t, dirty)

	// 重复 up：无可执行迁移，migrate.ErrNoChange 被视为成功（返回 nil），版本不变。
	require.NoError(t, Migrate(connStr, DirectionUp))
	version, _ = migrationVersion(t, connStr)
	assert.Equal(t, want, version)
}

// TestIntegration_MigrateDown 验证 down 按步回退、缺省步数为 1、
// 最低版本的 ErrNoChange 幂等，以及 down/up 可逆；结束时恢复到最新版本。
func TestIntegration_MigrateDown(t *testing.T) {
	connStr := requireTestConnStr(t)
	want := expectedVersion(t)

	// 前置：迁移到最新版本。
	require.NoError(t, Migrate(connStr, DirectionUp))

	// 回退步数超过可用迁移数：截断为全部可用步数，版本归零，schema_migrations 为空表。
	require.NoError(t, Migrate(connStr, DirectionDown, int(want)+3))
	version, dirty := migrationVersion(t, connStr)
	assert.Zero(t, version)
	assert.False(t, dirty)

	// 已在最低版本再 down：ErrNoChange 视为成功。
	require.NoError(t, Migrate(connStr, DirectionDown, 1))
	version, _ = migrationVersion(t, connStr)
	assert.Zero(t, version)

	// 重新 up 后缺省 down：回退 1 步。
	require.NoError(t, Migrate(connStr, DirectionUp))
	require.NoError(t, Migrate(connStr, DirectionDown))
	version, _ = migrationVersion(t, connStr)
	assert.Equal(t, want-1, version)

	// 恢复到最新版本，保持共享容器状态可预期。
	require.NoError(t, Migrate(connStr, DirectionUp))
	version, _ = migrationVersion(t, connStr)
	assert.Equal(t, want, version)
}
