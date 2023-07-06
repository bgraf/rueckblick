package tools

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/viper"
)

// RunEditor runs a user configured editor on the given path.
// User editor is retrieved via configuration `tools.editor`. If the configuration
// is not set, the environment variable EDITOR is used.
// An error is returned in case that neither is set.
func RunEditor(file string) error {
	editorName, err := lookupEditor()
	if err != nil {
		return err
	}

	cmd := exec.Command(editorName, file)

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

const keyEditorConfig = "tools.editor"

func lookupEditor() (string, error) {
	if viper.IsSet(keyEditorConfig) {
		str := viper.GetString(keyEditorConfig)
		return exec.LookPath(str)
	}

	editor, ok := os.LookupEnv("EDITOR")
	if !ok {
		return "", fmt.Errorf("Environment variable EDITOR not set")
	}

	return exec.LookPath(editor)
}
