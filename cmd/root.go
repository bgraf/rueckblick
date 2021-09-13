package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/bgraf/rueckblick/config"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/viper"
)

var cfgFile string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "rueckblick",
	Short: "Rueckblick",
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	var err error

	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.rueckblick.yaml)")

	rootCmd.PersistentFlags().StringP("root-dir", "r", "", "Journal root directory")
	err = viper.BindPFlag(
		config.KeyJournalDirectory,
		rootCmd.PersistentFlags().Lookup("root-dir"),
	)
	if err != nil {
		panic(err)
	}
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	// TODO: set config defaults here

	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := homedir.Dir()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		// Search config in home directory with name ".rueckblick" (without extension).
		if dir, err := os.Getwd(); err == nil {
			viper.AddConfigPath(dir)
		}
		viper.AddConfigPath(home)
		viper.SetConfigName(".rueckblick")
	}

	viper.SetEnvPrefix("rueckblick")
	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}
