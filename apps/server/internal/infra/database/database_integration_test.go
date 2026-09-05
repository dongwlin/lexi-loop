package database

// 集成测试：包级 TestMain 启动一次共享的真实 PostgreSQL（postgres:18-alpine，
// testcontainers），本包全部集成用例复用同一实例；Docker 不可用或传 -short 时
// 集成用例跳过、纯单元测试仍运行。
// 覆盖：真实建连与查询、config 连接串 / 池参数端到端、带池参数建连
// （docs/specs/backend/Go 测试规范.md §11：不 mock 数据库）。

import (
	"context"
	"flag"
	"log"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	"github.com/uptrace/bun"

	"github.com/dongwlin/lexi-loop/apps/server/internal/infra/config"
)

const (
	testDBName   = "lexi_loop_test"
	testDBUser   = "lexi"
	testDBPass   = "lexi"
	pgImage      = "postgres:18-alpine"
	pgConnParams = "sslmode=disable"
)

// 包级共享状态：TestMain 初始化，集成用例通过 requireTestDB 获取。
var (
	testDB        *bun.DB
	testDBConnStr string
)

// TestMain 为整个包启动一个共享的 PostgreSQL 容器，跑完全部用例后统一清理。
func TestMain(m *testing.M) {
	flag.Parse() // testing.Short 依赖已解析的 -test.short 等 flag

	if testing.Short() {
		log.Print("short 模式：跳过 testcontainers 集成测试（纯单元测试继续）")
		os.Exit(m.Run())
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	container, err := tcpostgres.Run(ctx, pgImage,
		tcpostgres.WithDatabase(testDBName),
		tcpostgres.WithUsername(testDBUser),
		tcpostgres.WithPassword(testDBPass),
		testcontainers.WithWaitStrategy(
			// postgres 首次启动会先跑 initdb 的临时实例、随后重启为正式实例；
			// 只等端口就绪会抢在 init 完成前连接、被服务端重置。
			// 等就绪日志出现两次（第二次才是正式实例）再开始跑用例。
			// 启动超时保持默认 60s：冷缓存下 initdb→重启 可能超过 5s。
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2),
		),
	)
	if err != nil {
		if container != nil {
			_ = container.Terminate(context.Background())
		}
		log.Printf("testcontainers 启动失败，集成测试将被跳过（纯单元测试继续）: %v", err)
		os.Exit(m.Run())
	}
	defer func() {
		if err := container.Terminate(context.Background()); err != nil {
			log.Printf("终止 postgres 容器失败: %v", err)
		}
	}()

	connStr, err := container.ConnectionString(ctx, pgConnParams)
	if err != nil {
		log.Fatalf("获取容器连接串失败: %v", err)
	}
	testDBConnStr = connStr

	testDB, err = New(ctx, Options{URL: connStr})
	if err != nil {
		log.Fatalf("创建 *bun.DB 失败: %v", err)
	}

	code := m.Run()

	if err := testDB.Close(); err != nil {
		log.Printf("关闭 *bun.DB 失败: %v", err)
	}
	os.Exit(code)
}

// requireTestDB 返回 TestMain 创建的共享 *bun.DB；Docker 不可用或 -short 时跳过集成用例。
func requireTestDB(t *testing.T) *bun.DB {
	t.Helper()
	if testDB == nil {
		t.Skip("testcontainers 不可用，跳过集成测试")
	}
	return testDB
}

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
