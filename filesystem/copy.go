package filesystem

import (
	"fmt"
	"os"
	"os/exec"
)

func Copy(src, dst string) error {
	sourceFileStat, err := os.Stat(src)
	if err != nil {
		return err
	}

	if !sourceFileStat.Mode().IsRegular() {
		return fmt.Errorf("path '%s' does not denote a file", src)
	}

	// Use system "cp" to preserve timestamps.
	cpCmd := exec.Command("cp", "--preserve=timestamps", src, dst)
	return cpCmd.Run()
}
