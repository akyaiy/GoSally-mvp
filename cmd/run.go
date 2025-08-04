package cmd

import (
	"github.com/akyaiy/GoSally-mvp/hooks"
	"github.com/spf13/cobra"
)

var runCmd = &cobra.Command{
	Use:     "run",
	Aliases: []string{"r"},
	Short:   "Run node normally",
	Long: `
"run" starts the node with settings depending on the configuration file`,
	Run: hooks.Run,
}

func init() {
	rootCmd.AddCommand(runCmd)
}
