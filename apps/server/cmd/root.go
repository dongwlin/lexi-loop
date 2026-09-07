// Package cmd 是 LexiLoop 的 CLI 路由层：只负责命令注册、参数绑定、上下文
// 与退出码处理，不承载业务逻辑（docs/backend/structure.md §3）。
package cmd

import (
	"os"

	"github.com/rs/zerolog"
	"github.com/spf13/cobra"

	"github.com/dongwlin/lexi-loop/apps/server/internal/infra/logger"
)

// appLogger 由根命令 PersistentPreRun 经 logger.Init 构造，供各子命令注入
// 给需要显式依赖日志的组件（如 app.Run）。
var appLogger zerolog.Logger

// Execute 执行根命令；出错时以非零退出码结束进程。
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:          "lexi-loop",
		Short:        "LexiLoop 后端服务",
		Long:         "LexiLoop（词环）英语生词复习系统后端。运行 lexi-loop serve 启动 HTTP API。",
		SilenceUsage: true,
		// 各子命令启动前用 ConsoleWriter 接管全局日志，使 migrations 等
		// 直接使用全局 log 的包输出一致的控制台格式；返回值保留给
		// 需要显式注入 logger 的子命令。
		PersistentPreRun: func(*cobra.Command, []string) {
			appLogger = logger.Init(logger.Options{Level: logger.DefaultLevel})
		},
	}
	root.AddCommand(newServeCommand())
	root.AddCommand(newMigrateCommand())
	root.AddCommand(newImportECDictCommand())
	root.AddCommand(newOpenAPICommand())
	root.AddCommand(newVersionCommand())
	return root
}

var rootCmd = newRootCmd()
