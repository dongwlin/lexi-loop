package handler

// 集成测试公共设施：包级 TestMain 启动一次共享的真实 PostgreSQL
//（postgres:18-alpine，testcontainers），应用全部迁移后构造真实
// Service + Handler 的完整 HTTP 栈；Docker 不可用或传 -short 时集成
// 用例跳过（docs/specs/backend/Go 测试规范.md：Handler 组件测试使用
// httptest + 真实 Service / Test DB）。

import (
	"context"
	"encoding/json"
	"flag"
	"log"
	"math/rand/v2"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	"github.com/uptrace/bun"

	v1 "github.com/dongwlin/lexi-loop/apps/server/internal/handler/v1"
	"github.com/dongwlin/lexi-loop/apps/server/internal/infra/database"
	"github.com/dongwlin/lexi-loop/apps/server/internal/service"
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

// TestMain 为整个包启动一个共享的 PostgreSQL 容器，应用全部迁移后跑用例。
func TestMain(m *testing.M) {
	flag.Parse()

	gin.SetMode(gin.TestMode)

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

// requireTestDB 返回共享数据库；Docker 不可用或 -short 时跳过集成用例。
func requireTestDB(t *testing.T) *bun.DB {
	t.Helper()
	if testDB == nil {
		t.Skip("testcontainers 不可用，跳过集成测试")
	}
	return testDB
}

// resetTables 清空全部业务表；用例共享数据库且涉及全局不变量
// （全库至多一个 active session），不做并行。
func resetTables(t *testing.T) {
	t.Helper()
	if testDB == nil {
		t.Skip("testcontainers 不可用，跳过集成测试")
	}
	_, err := testDB.ExecContext(context.Background(),
		"TRUNCATE review_items, review_sessions, user_words, dictionary_entries")
	require.NoError(t, err, "清空测试表失败")
}

// newTestServer 构造挂载真实 Service 的完整路由（固定随机源保证可复现）。
// 组件测试关注端点契约：日志静默（Nop），不配置 CORS 白名单（同源直连）。
func newTestServer(t *testing.T) *gin.Engine {
	t.Helper()
	db := requireTestDB(t)
	dictionarySvc := service.NewDictionary(db)
	wordSvc := service.NewWord(db, dictionarySvc)
	reviewSvc := service.NewReview(db, service.NewWeightedSampler(rand.NewPCG(7, 8)))

	engine := gin.New()
	RegisterRoutes(engine, Options{
		Log: zerolog.Nop(),
	}, v1.NewWordHandler(wordSvc), v1.NewReviewHandler(reviewSvc), v1.NewVersionHandler())
	return engine
}

// doJSON 执行一次 JSON 请求并返回响应记录器；body 为 nil 时不带请求体。
func doJSON(t *testing.T, engine *gin.Engine, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, encodeBody(t, body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)
	return rec
}

func encodeBody(t *testing.T, body any) *strings.Reader {
	t.Helper()
	if body == nil {
		return strings.NewReader("")
	}
	raw, err := json.Marshal(body)
	require.NoError(t, err)
	return strings.NewReader(string(raw))
}
