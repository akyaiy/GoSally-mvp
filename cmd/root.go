package cmd

import (
	"fmt"
	"log"
	"os"

	"github.com/akyaiy/GoSally-mvp/hooks"
	"github.com/akyaiy/GoSally-mvp/internal/core/corestate"
	"github.com/akyaiy/GoSally-mvp/internal/engine/logs"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "node",
	Short: "Go Sally node",
	Long: `
GoSally is an http server that handles jsonrpc-2.0 requests by calling methods as lua 
scripts in a given directory. For more information, visit: https://gosally.oblat.lv/`,
	Run: func(cmd *cobra.Command, args []string) {
		_ = cmd.Help()
	},
}

func Execute() {
	log.SetOutput(os.Stdout)
	log.SetPrefix(logs.SetBrightBlack(fmt.Sprintf("(%s) ", corestate.StageNotReady)))
	log.SetFlags(log.Ldate | log.Ltime)
	hooks.Compositor.LoadCMDLine(rootCmd)
	_ = rootCmd.Execute()
	// if err := rootCmd.Execute(); err != nil {
	// 	log.Fatalf("Unexpected error: %s", err.Error())
	// }
}
