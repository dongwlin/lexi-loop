package app

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/dongwlin/lexi-loop/apps/server/internal/handler"
	"github.com/dongwlin/lexi-loop/apps/server/internal/infra/config"
)

// shutdownTimeout 是优雅关闭的最大等待时间。
const shutdownTimeout = 10 * time.Second

// Run 构建 Gin Engine，以 HTTP 服务方式运行直到 ctx 结束，随后优雅关闭。
// MVP 骨架阶段仅提供 /healthz 与 /api/v1 空挂载点，未接入数据库等基础设施。
func Run(ctx context.Context, cfg *config.Config) error {
	srv := &http.Server{
		Addr:              cfg.HTTP.Addr,
		Handler:           newEngine(),
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

// newEngine 创建 Gin Engine：全局中间件与 /api/v1 业务路由统一经
// handler.RegisterRoutes 挂载，基础设施端点直接挂在根路径。
func newEngine() *gin.Engine {
	engine := gin.New()
	handler.RegisterRoutes(engine)
	// 健康检查属基础设施端点，不参与业务版本（docs/specs/backend/HTTP API 设计规范.md §2.4）。
	engine.GET("/healthz", handleHealthz)
	return engine
}

// handleHealthz 返回存活状态，供负载均衡 / 容器探活使用。
func handleHealthz(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
