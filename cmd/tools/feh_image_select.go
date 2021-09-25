package tools

import (
	"bufio"
	"io/ioutil"
	"os/exec"
	"syscall"
)

// FehSelectImage takes a directory and starts FEH on it. When the user
// presses <ENTER> in FEH the process is killed and the selected image's path
// is returned.
func FehSelectImage(directory string) (string, error) {
	cmd := exec.Command(
		"feh",
		"-A", "echo %F",
		directory,
	)

	var resultPath string

	cmdStdout, err := cmd.StdoutPipe()
	if err != nil {
		return resultPath, err
	}

	err = cmd.Start()
	if err != nil {
		return resultPath, err
	}

	scanner := bufio.NewScanner(cmdStdout)
	if scanner.Scan() {
		resultPath = scanner.Text()

		err := cmd.Process.Signal(syscall.SIGTERM)
		if err != nil {
			return resultPath, err
		}
	}

	_, err = ioutil.ReadAll(cmdStdout)
	if err != nil {
		return resultPath, err
	}

	err = cmd.Wait()
	if err != nil {
		return resultPath, err
	}

	return resultPath, nil
}
