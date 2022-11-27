package govalintesting

import (
	"errors"
	"os"
	"os/exec"
	"testing"
)

const testExitEnvVariable = "GOVALIN_EXITER"

func TestExit(t *testing.T, testFunc func()) int {
	if os.Getenv(testExitEnvVariable) == "1" {
		testFunc()
		return 0
	}

	cmd := exec.Command(os.Args[0], "-test.run="+t.Name()) //nolint:gosec // simply used for testing

	cmd.Env = append(cmd.Env, testExitEnvVariable+"=1")
	err := cmd.Run()

	if err == nil {
		return 0
	}

	var exitError *exec.ExitError
	if errors.As(err, &exitError); !exitError.Success() {
		return exitError.ExitCode()
	}

	return 0
}
