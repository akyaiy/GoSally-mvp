package cmd

import (
	"log"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "node",
	Short: "Go Sally node",
	Long:  "Main node runner for Go Sally",
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

func Execute() {
	log.SetOutput(os.Stdout)
	log.SetPrefix("\033[34m[INIT]\033[0m ")
	log.SetFlags(log.Ldate | log.Ltime)
	if err := rootCmd.Execute(); err != nil {
		log.Fatalf("Unexpected error: %s", err.Error())
	}
}
