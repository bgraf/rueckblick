package config

import "github.com/spf13/viper"

var (
	KeyJournalDirectory = "journal.directory"
)

func HasJournalDirectory() bool {
	return viper.IsSet(KeyJournalDirectory)
}

func JournalDirectory() string {
	return viper.GetString(KeyJournalDirectory)
}
