package app

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/uptrace/bun"

	"github.com/dongwlin/lexi-loop/apps/server/internal/handler"
	"github.com/dongwlin/lexi-loop/apps/server/internal/handler/v1"
	"github.com/dongwlin/lexi-loop/apps/server/internal/infra/config"
	"github.com/dongwlin/lexi-loop/apps/server/internal/infra/database"
	"github.com/dongwlin/lexi-loop/apps/server/internal/service"
)

// shutdownTimeout 是优雅关闭的最大等待时间。
const shutdownTimeout = 10 * time.Second

// Run 组装长生命周期组件（structure.md §3 组合根：*bun.DB、具体 Service、
// 版本化 Handler），以 HTTP 服务方式运行直到 ctx 结束，随后优雅关闭。
// Repo、Domain 不进入长生命周期依赖图，由 Service 方法内按需构造。
func Run(ctx context.Context, cfg *config.Config) error {
	if cfg.Database.URL == "" {
		return fmt.Errorf("missing database config: set LEXI_DATABASE_URL")
	}
	db, err := database.New(ctx, database.Options{
		URL:             cfg.Database.URL,
		MaxConns:        cfg.Database.MaxConns,
		MinConns:        cfg.Database.MinConns,
		MaxConnLifetime: cfg.Database.MaxConnLifetime,
		MaxConnIdleTime: cfg.Database.MaxConnIdleTime,
	})
	if err != nil {
		return fmt.Errorf("connect database: %w", err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			log.Printf("close database: %v", err)
		}
	}()

	srv := &http.Server{
		Addr:              cfg.HTTP.Addr,
		Handler:           newEngine(db),
		ReadHeaderTimeout: 5 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		log.Printf("lexi-loop http server listening on %s", cfg.HTTP.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	select {
	case err := <-errCh:
		return fmt.Errorf("http server: %w", err)
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("graceful shutdown: %w", err)
		}
		log.Print("lexi-loop http server stopped")
		return nil
	}
}

// newEngine 创建 Gin Engine：组合根显式构造具体 Service 与版本化 Handler
// （不创建接口），全局中间件与 /api/v1 业务路由统一经 handler.RegisterRoutes
// 挂载，基础设施端点直接挂在根路径。
func newEngine(db *bun.DB) *gin.Engine {
	dictionarySvc := service.NewDictionary(db)
	wordSvc := service.NewWord(db, dictionarySvc)
	reviewSvc := service.NewReview(db, service.NewWeightedSampler(service.NewTimeSeededSource()))

	wordHandler := v1.NewWordHandler(wordSvc)
	reviewHandler := v1.NewReviewHandler(reviewSvc)

	engine := gin.New()
	handler.RegisterRoutes(engine, wordHandler, reviewHandler)
	// 健康检查属基础设施端点，不参与业务版本（docs/specs/backend/HTTP API 设计规范.md §2.4）。
	engine.GET("/healthz", handleHealthz)
	return engine
}

// handleHealthz 返回存活状态，供负载均衡 / 容器探活使用。
func handleHealthz(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
