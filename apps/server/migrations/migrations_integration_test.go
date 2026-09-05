package migrations

// 集成测试：包级 TestMain 启动一次共享的真实 PostgreSQL（postgres:18-alpine，
// testcontainers），本包全部集成用例复用同一实例；Docker 不可用或传 -short 时
// 集成用例跳过、纯单元测试仍运行。
// 覆盖 Go 测试规范 §2 的迁移基线：空库可完整 up；并验证版本簿记、
// 重复 up 的 ErrNoChange 幂等、down 按步回退与最低版本边界。

import (
	"context"
	"flag"
	"io/fs"
	"log"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

const (
	testDBName   = "lexi_loop_test"
	testDBUser   = "lexi"
	testDBPass   = "lexi"
	pgImage      = "postgres:18-alpine"
	pgConnParams = "sslmode=disable"
)

// 包级共享状态：TestMain 初始化，集成用例通过 requireTestConnStr 获取。
var testDBConnStr string

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

	os.Exit(m.Run())
}

// requireTestConnStr 返回 TestMain 创建的容器连接串；Docker 不可用或 -short 时跳过集成用例。
func requireTestConnStr(t *testing.T) string {
	t.Helper()
	if testDBConnStr == "" {
		t.Skip("testcontainers 不可用，跳过集成测试")
	}
	return testDBConnStr
}

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
