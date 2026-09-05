package cmd

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/dongwlin/lexi-loop/apps/server/internal/infra/config"
	"github.com/dongwlin/lexi-loop/apps/server/migrations"
)

// newMigrateCommand 执行 PostgreSQL 版本化迁移（迁移文件位于 apps/server/migrations/，
// 由 migrations 包的 Migrate 显式执行；应用启动不隐式迁移）。
func newMigrateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "migrate",
		Short: "执行数据库迁移",
		Long:  "执行 apps/server/migrations/ 下的版本化 SQL 迁移，见 docs/backend/structure.md §2。",
	}
	cmd.AddCommand(
		&cobra.Command{
			Use:   "up",
			Short: "应用全部未执行的迁移",
			RunE: func(_ *cobra.Command, _ []string) error {
				url, err := databaseURL()
				if err != nil {
					return err
				}
				return migrations.Migrate(url, migrations.DirectionUp)
			},
		},
		&cobra.Command{
			Use:   "down [steps]",
			Short: "回退指定步数的迁移（默认 1 步）",
			Args:  cobra.MaximumNArgs(1),
			RunE: func(_ *cobra.Command, args []string) error {
				url, err := databaseURL()
				if err != nil {
					return err
				}
				steps := 1
				if len(args) == 1 {
					n, err := strconv.Atoi(args[0])
					if err != nil || n < 1 {
						return fmt.Errorf("回退步数必须是正整数，收到 %q", args[0])
					}
					steps = n
				}
				return migrations.Migrate(url, migrations.DirectionDown, steps)
			},
		},
	)
	return cmd
}

// databaseURL 加载迁移目标数据库连接串（LEXI_DATABASE_URL，必填）。
func databaseURL() (string, error) {
	cfg, err := config.Load()
	if err != nil {
		return "", err
	}
	if cfg.Database.URL == "" {
		return "", errors.New("数据库连接串为空：请设置 LEXI_DATABASE_URL")
	}
	return cfg.Database.URL, nil
}
