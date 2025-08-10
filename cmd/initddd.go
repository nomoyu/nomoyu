package cmd

import (
	"fmt"

	"github.com/nomoyu/nomoyu/internal/scaffold"
	"github.com/spf13/cobra"
)

var initDDDCmd = &cobra.Command{
	Use:   "init-ddd <ctx1,ctx2,...>",
	Short: "在当前 DDD 项目中追加一个或多个领域上下文",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctxs := scaffold.SplitCSV(args[0])
		if err := scaffold.AddDDDContexts(".", ctxs); err != nil {
			return fmt.Errorf("追加领域失败: %w", err)
		}
		fmt.Println("✅ 追加领域成功：", ctxs)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(initDDDCmd)
}
