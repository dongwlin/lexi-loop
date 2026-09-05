// Package cmd 是 LexiLoop 的 CLI 路由层：只负责命令注册、参数绑定、上下文
// 与退出码处理，不承载业务逻辑（docs/backend/structure.md §3）。
package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

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
	}
	root.AddCommand(newServeCommand())
	root.AddCommand(newMigrateCommand())
	root.AddCommand(newImportECDictCommand())
	return root
}

var rootCmd = newRootCmd()
