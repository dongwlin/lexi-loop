package cmd

import (
	"errors"

	"github.com/spf13/cobra"
)

// newImportECDictCommand 将 ECDICT CSV 导入 dictionary_entries。
// MVP 骨架阶段仅注册命令形态，未接入 internal/importer/ecdict。
func newImportECDictCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "import-ecdict <file>",
		Short: "导入 ECDICT CSV 到本地词典库（暂未实现）",
		Args:  cobra.ExactArgs(1),
		RunE:  notImplemented("import-ecdict"),
	}
}

// notImplemented 返回报告「尚未实现」的 RunE，用于骨架占位命令。
func notImplemented(name string) func(*cobra.Command, []string) error {
	return func(_ *cobra.Command, _ []string) error {
		return errors.New(name + " 尚未实现（MVP 骨架占位）")
	}
}
