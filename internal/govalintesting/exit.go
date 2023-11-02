package govalintesting

import (
	"errors"
	"os"
	"os/exec"
	"testing"
)

const testExitEnvVariable = "GOVALIN_EXITER"

func TestExit(t *testing.T, testFunc func()) (int, string) {
	if os.Getenv(testExitEnvVariable) == "1" {
		testFunc()
		return 0, ""
	}

	cmd := exec.Command(os.Args[0], "-test.run="+t.Name()) //nolint:gosec // simply used for testing

	cmd.Env = append(os.Environ(), testExitEnvVariable+"=1")

	out, err := cmd.CombinedOutput()
	outputString := string(out)

	if err == nil {
		return 0, outputString
	}

	var exitError *exec.ExitError
	if errors.As(err, &exitError) {
		return exitError.ExitCode(), outputString
	}

	return 0, outputString
}
