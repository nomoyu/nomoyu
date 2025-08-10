package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "nomoyu",
	Short: "Nomoyu 脚手架",
	Long:  "Nomoyu 脚手架：快速初始化 DDD 项目与领域模块",
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println("❌ Error:", err)
		os.Exit(1)
	}
}
