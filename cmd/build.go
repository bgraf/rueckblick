package cmd

import (
	"fmt"
	"log"

	"github.com/bgraf/rueckblick/building"
	"github.com/bgraf/rueckblick/config"
	"github.com/bgraf/rueckblick/filesystem"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// buildCmd represents the build command
var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "",
	Long:  "",
	RunE:  runBuildCmd,
}

func init() {
	rootCmd.AddCommand(buildCmd)

	buildCmd.PersistentFlags().StringP("output", "O", "", "Build directory")
	err := viper.BindPFlag("build.directory", buildCmd.PersistentFlags().Lookup("output"))
	if err != nil {
		panic(err)
	}

	buildCmd.Flags().BoolP("clean", "C", false, "Clean build everything")
}

func runBuildCmd(cmd *cobra.Command, args []string) error {
	buildOpts, err := initBuildOptions(cmd)
	if err != nil {
		return err
	}

	log.Printf("journal directory: %s", buildOpts.JournalDirectory)
	log.Printf("build directory:   %s", buildOpts.BuildDirectory)

	if err := building.Build(buildOpts); err != nil {
		return err
	}

	return nil
}

func initBuildOptions(cmd *cobra.Command) (building.Options, error) {
	b := building.Options{}

	if !config.HasJournalDirectory() {
		return b, fmt.Errorf("no journal directory configured")
	}

	if !config.HasBuildDirectory() {
		return b, fmt.Errorf("no build directory configured")
	}

	var err error
	b.Clean, err = cmd.Flags().GetBool("clean")
	if err != nil {
		return b, err
	}

	b.JournalDirectory = filesystem.Abs(config.JournalDirectory())
	b.BuildDirectory = filesystem.Abs(config.BuildDirectory())

	return b, nil
}
