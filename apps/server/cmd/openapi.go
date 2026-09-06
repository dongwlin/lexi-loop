package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cobra"

	"github.com/dongwlin/lexi-loop/apps/server/internal/app"
)

// newOpenAPICommand 离线生成 /api/v1 的 OpenAPI 3.1 spec：产物为
// openapi.json 与 openapi.yaml，落盘 docs/openapi/ 后不手改，更新一律
// 重跑本命令（docs/specs/backend/Go 技术栈.md「API 文档」）。spec 生成
// 不需要数据库。
func newOpenAPICommand() *cobra.Command {
	var out string
	cmd := &cobra.Command{
		Use:   "openapi",
		Short: "离线生成 /api/v1 的 OpenAPI 3.1 spec",
		Long:  "生成 /api/v1 的 OpenAPI 3.1 spec（openapi.json 与 openapi.yaml）到指定目录；产物由生成器维护，请勿手改。",
		RunE: func(cmd *cobra.Command, _ []string) error {
			// 离线构建不输出 gin 调试路由日志，保持命令输出干净。
			gin.SetMode(gin.ReleaseMode)
			jsonSpec, yamlSpec, err := app.BuildOpenAPISpec()
			if err != nil {
				return err
			}
			if err := os.MkdirAll(out, 0o755); err != nil {
				return fmt.Errorf("创建输出目录: %w", err)
			}
			for _, artifact := range []struct {
				name    string
				content []byte
			}{
				{"openapi.json", jsonSpec},
				{"openapi.yaml", yamlSpec},
			} {
				path := filepath.Join(out, artifact.name)
				if err := os.WriteFile(path, artifact.content, 0o644); err != nil {
					return fmt.Errorf("写入 %s: %w", path, err)
				}
				fmt.Fprintf(cmd.OutOrStdout(), "written %s\n", path)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&out, "out", "../../docs/openapi",
		"输出目录（相对当前工作目录；默认为仓库根的 docs/openapi，在 apps/server 下运行时无需调整）")
	return cmd
}
