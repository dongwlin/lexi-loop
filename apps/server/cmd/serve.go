package cmd

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/dongwlin/lexi-loop/apps/server/internal/app"
	"github.com/dongwlin/lexi-loop/apps/server/internal/infra/config"
)

// newServeCommand 启动 HTTP API 服务（lexi-loop serve）。
func newServeCommand() *cobra.Command {
	var httpAddr string
	cmd := &cobra.Command{
		Use:   "serve",
		Short: "启动 HTTP API 服务",
		RunE: func(_ *cobra.Command, _ []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			if httpAddr != "" {
				cfg.HTTP.Addr = httpAddr
			}
			// 收到 SIGINT / SIGTERM 时取消 ctx，交由 app 完成优雅关闭。
			ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
			defer stop()
			return app.Run(ctx, cfg)
		},
	}
	cmd.Flags().StringVar(&httpAddr, "http-addr", "", "HTTP 监听地址；为空时取环境变量 LEXI_HTTP_ADDR，缺省为 :8080")
	return cmd
}
