package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/nomoyu/nomoyu/internal/scaffold"
)

var (
	modulePath string
	contexts   string // 可选：初始化时就生成一些领域
)

var initCmd = &cobra.Command{
	Use:   "init <project>",
	Short: "初始化 DDD 项目骨架",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		project := args[0]
		var ctxs []string
		if strings.TrimSpace(contexts) != "" {
			ctxs = scaffold.SplitCSV(contexts)
		}
		if err := scaffold.GenerateDDDProject(project, modulePath, ctxs, frameworkPath); err != nil {
			return fmt.Errorf("初始化失败: %w", err)
		}
		fmt.Println("✅ DDD 项目创建成功：", project)
		return nil
	},
}

var frameworkPath string

func init() {
	rootCmd.AddCommand(initCmd)
	initCmd.Flags().StringVar(&modulePath, "module", "", "Go module 路径（如 github.com/yourname/yourproj）")
	initCmd.Flags().StringVar(&contexts, "contexts", "", "初始化时一并创建的领域（逗号分隔），如：user,billing")
	initCmd.Flags().StringVar(&frameworkPath, "framework", "", "go-gin-framework 本地路径（本地开发用）")
}
