package database

// 集成测试：共享 PostgreSQL 容器由 main_test.go 的包级 TestMain 启动，
// Docker 不可用或传 -short 时集成用例跳过。
// 覆盖：真实建连与查询、config 连接串 / 池参数端到端、带池参数建连
// （docs/specs/backend/Go 测试规范.md §11：不 mock 数据库）。

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dongwlin/lexi-loop/apps/server/internal/infra/config"
)

// TestIntegration_NewConnectsAndExecutesQuery 验证共享连接池能执行真实查询
// （server_version 由 postgres:18 返回，非 0 即建连成功）。
func TestIntegration_NewConnectsAndExecutesQuery(t *testing.T) {
	db := requireTestDB(t)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var version string
	err := db.NewRaw("SELECT version()").Scan(ctx, &version)
	require.NoError(t, err, "bun 上下文查询应成功")
	assert.NotEmpty(t, version, "version() 应返回 PostgreSQL 版本字符串")
}

// TestIntegration_ConfigToOptionsRoundTrip 验证从 config（环境变量）加载的连接串与
// 连接池参数可真实建连：等价于用 LEXI_DATABASE_* 配置数据库后的组装路径。
func TestIntegration_ConfigToOptionsRoundTrip(t *testing.T) {
	requireTestDB(t) // 前置：容器已就绪

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	t.Setenv("LEXI_DATABASE_URL", testDBConnStr)
	t.Setenv("LEXI_DATABASE_MAX_CONNS", "5")
	t.Setenv("LEXI_DATABASE_MIN_CONNS", "1")

	cfg, err := config.Load()
	require.NoError(t, err)
	assert.Equal(t, testDBConnStr, cfg.Database.URL)
	assert.Equal(t, 5, cfg.Database.MaxConns)
	assert.Equal(t, 1, cfg.Database.MinConns)

	// 按 config 取值走一遍 New：验证配置映射出的 Options 能真实建连。
	db, err := New(ctx, Options{
		URL:      cfg.Database.URL,
		MaxConns: cfg.Database.MaxConns,
		MinConns: cfg.Database.MinConns,
	})
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	var n int
	err = db.NewRaw("SELECT 1").Scan(ctx, &n)
	require.NoError(t, err, "按 config 组装应能执行真实查询")
	assert.Equal(t, 1, n)
}

// TestIntegration_PoolOptionsWorkEndToEnd 验证带连接池参数的 Options 经 New 后
// 仍能正常建连与查询（参数语义本身由 TestMergePoolOptions_* 单元测试保证；
// 这里验证真实 Postgres 上整条路径可用）。
func TestIntegration_PoolOptionsWorkEndToEnd(t *testing.T) {
	requireTestDB(t) // 前置：容器已就绪

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	db, err := New(ctx, Options{
		URL:             testDBConnStr,
		MaxConns:        4,
		MinConns:        1,
		MaxConnLifetime: 10 * time.Minute,
		MaxConnIdleTime: 30 * time.Second,
	})
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	var n int
	err = db.NewRaw("SELECT 1").Scan(ctx, &n)
	require.NoError(t, err, "带池参数连接后应能执行真实查询")
	assert.Equal(t, 1, n)
}
