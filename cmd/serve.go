package cmd

import (
	"github.com/spf13/cobra"
	"gitlab.com/begraf/rueckblick/cmd/serve"
)

// serveCmd represents the serve command
var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "A brief description of your command",
	RunE:  serve.RunServeCmd,
}

func init() {
	rootCmd.AddCommand(serveCmd)

}
