package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/dongwlin/lexi-loop/apps/server/internal/importer/ecdict"
	"github.com/dongwlin/lexi-loop/apps/server/internal/infra/database"
)

// newImportECDictCommand 将 ECDICT CSV 导入 dictionary_entries
// （lexi-loop import-ecdict <file>）；CSV 解析与批处理编排都在
// internal/importer/ecdict，本命令只做参数绑定与生命周期处理。
func newImportECDictCommand() *cobra.Command {
	var batchSize int
	cmd := &cobra.Command{
		Use:   "import-ecdict <file>",
		Short: "将 ECDICT CSV 导入本地词典库（dictionary_entries）",
		Long: `将 ECDICT CSV（https://github.com/skywind3000/ECDICT）批量导入 dictionary_entries。

导入前必须先执行 lexi-loop migrate up 建表。导入按固定批次分事务提交，
失败只回滚当前批次并中止；写入按 headword 唯一键幂等，中断或失败后
重跑同一文件即可续传。连接串经 LEXI_DATABASE_URL 提供（与 serve /
migrate 相同）。`,
		Args: cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			url, err := databaseURL()
			if err != nil {
				return err
			}

			// 收到 SIGINT / SIGTERM 时取消 ctx：批次边界停止导入，
			// 已提交批次保持有效，重跑续传。
			ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
			defer stop()

			db, err := database.New(ctx, database.Options{URL: url})
			if err != nil {
				return fmt.Errorf("连接数据库失败: %w", err)
			}
			defer func() {
				_ = db.Close()
			}()

			file, err := os.Open(args[0])
			if err != nil {
				return fmt.Errorf("打开 CSV 文件失败: %w", err)
			}
			defer func() {
				_ = file.Close()
			}()

			importer := ecdict.NewImporter(db, batchSize)
			result, err := importer.Import(ctx, file, func(p ecdict.Progress) {
				fmt.Fprintf(os.Stderr, "\r已提交 %d 批，处理 %d 行，跳过 %d 行，写入 %d 词条",
					p.BatchesCommitted, p.RowsProcessed, p.RowsSkipped, p.EntriesWritten)
			})
			fmt.Fprintln(os.Stderr)
			if err != nil {
				return err
			}
			fmt.Printf("导入完成：%d 批事务，处理 %d 行，跳过 %d 行，写入 / 刷新 %d 词条\n",
				result.BatchesCommitted, result.RowsProcessed, result.RowsSkipped, result.EntriesWritten)
			return nil
		},
	}
	cmd.Flags().IntVar(&batchSize, "batch-size", ecdict.DefaultBatchSize,
		"单个事务提交的行数；批次失败只回滚当前批")
	return cmd
}
