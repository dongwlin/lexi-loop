package cmd

import (
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
