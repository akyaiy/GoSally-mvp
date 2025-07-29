package cmd

import (
	"fmt"
	"log"
	"os"

	"github.com/akyaiy/GoSally-mvp/internal/core/corestate"
	"github.com/akyaiy/GoSally-mvp/internal/engine/config"
	"github.com/akyaiy/GoSally-mvp/internal/engine/logs"
	"github.com/spf13/cobra"
)

var compositor *config.Compositor = config.NewCompositor()

var rootCmd = &cobra.Command{
	Use:   "node",
	Short: "Go Sally node",
	Long:  "Main node runner for Go Sally",
	Run: func(cmd *cobra.Command, args []string) {
		_ = cmd.Help()
	},
}

func Execute() {
	log.SetOutput(os.Stdout)
	log.SetPrefix(logs.SetBrightBlack(fmt.Sprintf("(%s) ", corestate.StageNotReady)))
	log.SetFlags(log.Ldate | log.Ltime)
	compositor.LoadCMDLine(rootCmd)
	_ = rootCmd.Execute()
	// if err := rootCmd.Execute(); err != nil {
	// 	log.Fatalf("Unexpected error: %s", err.Error())
	// }
}
