// ABOUTME: Tests for CLI argument handling and help output.
// ABOUTME: Uses subprocess execution to test main() behavior including exit codes.
package main

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestMain_HelpFlag(t *testing.T) {
	bin := t.TempDir() + "/sprint-ledger-test"
	build := exec.Command("go", "build", "-o", bin, ".")
	build.Stderr = os.Stderr
	if err := build.Run(); err != nil {
		t.Fatalf("failed to build: %v", err)
	}

	for _, flag := range []string{"--help", "-h"} {
		t.Run(flag, func(t *testing.T) {
			cmd := exec.Command(bin, flag)
			out, err := cmd.CombinedOutput()
			output := string(out)

			if err != nil {
				t.Errorf("expected exit 0 for %s, got error: %v\nOutput: %s", flag, err, output)
			}

			if !strings.Contains(output, "sprint-ledger") {
				t.Errorf("help output should mention 'sprint-ledger', got: %s", output)
			}
			if !strings.Contains(output, "<project-root>") {
				t.Errorf("help output should mention '<project-root>', got: %s", output)
			}
		})
	}
}

func TestMain_NoArgs(t *testing.T) {
	bin := t.TempDir() + "/sprint-ledger-test"
	build := exec.Command("go", "build", "-o", bin, ".")
	build.Stderr = os.Stderr
	if err := build.Run(); err != nil {
		t.Fatalf("failed to build: %v", err)
	}

	cmd := exec.Command(bin)
	err := cmd.Run()
	if err == nil {
		t.Fatal("expected non-zero exit for no args")
	}
	exitErr, ok := err.(*exec.ExitError)
	if !ok {
		t.Fatalf("expected ExitError, got %T", err)
	}
	if exitErr.ExitCode() != 1 {
		t.Errorf("expected exit code 1, got %d", exitErr.ExitCode())
	}
}
