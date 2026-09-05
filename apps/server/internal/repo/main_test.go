package repo

// 集成测试公共设施：包级 TestMain 启动一次共享的真实 PostgreSQL
// （postgres:18-alpine，testcontainers），应用 migrations 里的全部迁移后，
// 本包全部集成用例复用同一实例与同一 *bun.DB；Docker 不可用或传 -short
// 时集成用例跳过、纯单元测试仍运行（docs/specs/backend/Go 测试规范.md）。

import (
	"context"
	"flag"
	"log"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	"github.com/uptrace/bun"

	"github.com/dongwlin/lexi-loop/apps/server/internal/infra/database"
	"github.com/dongwlin/lexi-loop/apps/server/migrations"
)

const (
	testDBName   = "lexi_loop_test"
	testDBUser   = "lexi"
	testDBPass   = "lexi"
	pgImage      = "postgres:18-alpine"
	pgConnParams = "sslmode=disable"
)

// 包级共享状态：TestMain 初始化，集成用例通过 requireTestDB 获取。
var testDB *bun.DB

// TestMain 为整个包启动一个共享的 PostgreSQL 容器，应用全部迁移后跑用例，
// 跑完统一清理。Repo 层测试依赖真实 schema，因此先执行 migrations.Migrate。
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
			// 等就绪日志出现两次再开始（与 migrations / infra/database 相同）。
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

	// Repo 集成测试依赖迁移后的真实 schema（迁移是数据库结构的唯一落点）。
	if err := migrations.Migrate(connStr, migrations.DirectionUp); err != nil {
		log.Fatalf("应用迁移失败: %v", err)
	}

	db, err := database.New(ctx, database.Options{URL: connStr})
	if err != nil {
		log.Fatalf("构造测试数据库失败: %v", err)
	}
	testDB = db

	code := m.Run()

	_ = db.Close()
	os.Exit(code)
}

// requireTestDB 返回 TestMain 创建的共享数据库；Docker 不可用或 -short 时跳过集成用例。
func requireTestDB(t *testing.T) *bun.DB {
	t.Helper()
	if testDB == nil {
		t.Skip("testcontainers 不可用，跳过集成测试")
	}
	return testDB
}

// resetTables 清空全部业务表。集成用例共享一个数据库，表级用例不并行，
// 各用例开始时清空以保证互相独立（Go 测试规范：共享数据库状态不做并行）。
func resetTables(t *testing.T) {
	t.Helper()
	_, err := testDB.ExecContext(context.Background(),
		"TRUNCATE review_items, review_sessions, user_words, dictionary_entries")
	require.NoError(t, err, "清空测试表失败")
}
