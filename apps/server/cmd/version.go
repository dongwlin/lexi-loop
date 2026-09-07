package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/dongwlin/lexi-loop/apps/server/internal/infra/buildinfo"
)

// newVersionCommand 输出版本信息（lexi-loop version）。默认只输出版本号，
// 便于脚本取值；-b 携带完整构建信息（构建时间 / Go 版本）。输出直接写
// stdout，不走 logger。
func newVersionCommand() *cobra.Command {
	var withBuild bool
	cmd := &cobra.Command{
		Use:   "version",
		Short: "输出版本信息（-b 携带构建信息）",
		RunE: func(cmd *cobra.Command, _ []string) error {
			out := cmd.OutOrStdout()
			if !withBuild {
				fmt.Fprintln(out, buildinfo.Version)
				return nil
			}
			buildTime := buildinfo.BuildTime
			if buildTime == "" {
				buildTime = "-"
			}
			fmt.Fprintf(out, "version: %s\nbuild_time: %s\ngo_version: %s\n",
				buildinfo.Version, buildTime, buildinfo.GoVersion())
			return nil
		},
	}
	cmd.Flags().BoolVarP(&withBuild, "build", "b", false, "携带构建信息（构建时间 / Go 版本）")
	return cmd
}
