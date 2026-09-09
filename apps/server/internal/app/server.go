package app

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"

	"github.com/dongwlin/lexi-loop/apps/server/internal/infra/config"
	"github.com/dongwlin/lexi-loop/apps/server/internal/infra/database"
)

// shutdownTimeout 是优雅关闭的最大等待时间。
const shutdownTimeout = 10 * time.Second

// Run 组装长生命周期组件（structure.md §3 组合根：*bun.DB、具体 Service、
// 版本化 Handler），以 HTTP 服务方式运行直到 ctx 结束，随后优雅关闭。
// Repo、Domain 不进入长生命周期依赖图，由 Service 方法内按需构造。
// log 是注入的 zerolog logger，由 cmd 根命令经 logger.Init 构造后传入，
// 并继续传给 RegisterRoutes（structure.md §4.4）。
func Run(ctx context.Context, cfg *config.Config, log zerolog.Logger) error {
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
			log.Error().Err(err).Msg("close database")
		}
	}()

	engine, dictImportSvc := newEngine(cfg, db, log)
	srv := &http.Server{
		Addr:              cfg.HTTP.Addr,
		Handler:           engine,
		ReadHeaderTimeout: 5 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		log.Info().Str("addr", cfg.HTTP.Addr).Msg("lexi-loop http server listening")
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	// serve 即时可用：导入编排作为后台 goroutine 在 HTTP 服务启动后拉起，
	// 导入期间服务正常响应其它请求（issue #3）。ctx 取消时导入在批次
	// 边界停止，已提交批次保持有效，下次启动续传。
	dictImportSvc.StartAutoImport(ctx)

	select {
	case err := <-errCh:
		return fmt.Errorf("http server: %w", err)
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("graceful shutdown: %w", err)
		}
		log.Info().Msg("lexi-loop http server stopped")
		return nil
	}
}

// handleHealthz 返回存活状态，供负载均衡 / 容器探活使用。
func handleHealthz(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
