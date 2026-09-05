package database

// 集成测试公共设施：包级 TestMain 启动一次共享的真实 PostgreSQL
// （postgres:18-alpine，testcontainers），本包全部集成用例复用同一实例；
// Docker 不可用或传 -short 时集成用例跳过、纯单元测试仍运行。

import (
	"context"
	"flag"
	"log"
	"os"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	"github.com/uptrace/bun"
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
