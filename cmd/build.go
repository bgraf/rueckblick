package cmd

import (
	"github.com/bgraf/rueckblick/cmd/building"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// buildCmd represents the build command
var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "",
	Long:  "",
	RunE:  building.RunBuildCmd,
}

func init() {
	rootCmd.AddCommand(buildCmd)

	buildCmd.Flags().StringP("output", "O", "", "Build directory")

	viper.BindPFlag("build.directory", buildCmd.Flags().Lookup("builddir"))
}
