package cmd

import (
	"fmt"
	"runtime"

	"github.com/akyaiy/GoSally-mvp/src/internal/engine/config"
	"github.com/spf13/cobra"
)

var verCmd = &cobra.Command{
	Use:     "version",
	Aliases: []string{"ver", "v"},
	Short:   "Return node version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("GoSally node: %s\n", config.NodeVersion)
		fmt.Printf("Go version: %s\n", runtime.Version())
		fmt.Printf("Go OS/Arch: %s/%s\n", runtime.GOOS, runtime.GOARCH)
	},
}

func init() {
	rootCmd.AddCommand(verCmd)
}
