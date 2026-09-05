package cmd

import (
	"errors"

	"github.com/spf13/cobra"
)

// newMigrateCommand 执行 PostgreSQL 版本化迁移（迁移文件位于 apps/server/migrations/）。
// MVP 骨架阶段仅注册命令形态，未接入 infra/database。
func newMigrateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "migrate",
		Short: "执行数据库迁移（暂未实现）",
		Long:  "执行 apps/server/migrations/ 下的版本化 SQL 迁移，见 docs/backend/structure.md §3。",
	}
	cmd.AddCommand(
		&cobra.Command{
			Use:   "up",
			Short: "应用全部未执行的迁移",
			RunE:  notImplemented("migrate up"),
		},
		&cobra.Command{
			Use:   "down",
			Short: "回退指定步数的迁移（默认 1 步）",
			Args:  cobra.MaximumNArgs(1),
			RunE:  notImplemented("migrate down"),
		},
	)
	return cmd
}

// notImplemented 返回报告「尚未实现」的 RunE，用于骨架占位命令。
func notImplemented(name string) func(*cobra.Command, []string) error {
	return func(_ *cobra.Command, _ []string) error {
		return errors.New(name + " 尚未实现（MVP 骨架占位）")
	}
}
