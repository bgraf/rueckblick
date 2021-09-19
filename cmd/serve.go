package cmd

import (
	"github.com/bgraf/rueckblick/cmd/serve"
	"github.com/spf13/cobra"
)

// serveCmd represents the serve command
var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "A brief description of your command",
	RunE:  serve.RunServeCmd,
}

func init() {
	rootCmd.AddCommand(serveCmd)

	serveCmd.Flags().StringP(
		"resource-dir",
		"R",
		"",
		"Directory containing templates and static files",
	)
}
