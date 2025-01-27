package tester

import (
	"os"
	"os/exec"
)

func StartTest() error {
	cmd := exec.Command("go", "test", "-v", "./...")
	cmd.Dir = "./../"
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		return err
	}

	return nil
}
