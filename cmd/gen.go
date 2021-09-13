package cmd

import (
	"github.com/spf13/cobra"
)

// genCmd represents the gen command
var genCmd = &cobra.Command{
	Use:   "gen",
	Short: "Generate entries, preview images, etc.",
	Long:  `Generate collects procedures to ease the authoring process.`,
}

func init() {
	rootCmd.AddCommand(genCmd)
}
