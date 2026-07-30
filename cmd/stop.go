package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"go-fly-muti/tools"
	"path/filepath"
)

var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "停止客服http服务",
	Run: func(cmd *cobra.Command, args []string) {
		rootPath = tools.GetRootPath()
		pidPath := filepath.Join(rootPath, pidFileName)
		if err := stopProcesses(pidPath); err != nil {
			fmt.Println("停止服务失败:", err)
			return
		}
		fmt.Println("GOFLY 服务已停止")
	},
}
