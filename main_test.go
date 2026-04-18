// ABOUTME: Tests for CLI argument handling and help output.
// ABOUTME: Uses subprocess execution to test main() behavior including exit codes.
package main

import (
	"os"
	"os/exec"
	"path/filepath"
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
			if !strings.Contains(output, "project-root") {
				t.Errorf("help output should mention 'project-root', got: %s", output)
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

	// Run from a directory without .ai/ — should exit 1
	cmd := exec.Command(bin)
	cmd.Dir = t.TempDir()
	err := cmd.Run()
	if err == nil {
		t.Fatal("expected non-zero exit for no args without .ai/")
	}
	exitErr, ok := err.(*exec.ExitError)
	if !ok {
		t.Fatalf("expected ExitError, got %T", err)
	}
	if exitErr.ExitCode() != 1 {
		t.Errorf("expected exit code 1, got %d", exitErr.ExitCode())
	}
}

func TestMain_AutoDetectDotAI(t *testing.T) {
	bin := t.TempDir() + "/sprint-ledger-test"
	build := exec.Command("go", "build", "-o", bin, ".")
	build.Stderr = os.Stderr
	if err := build.Run(); err != nil {
		t.Fatalf("failed to build: %v", err)
	}

	// Run from the testdata/valid directory which has .ai/
	cmd := exec.Command(bin)
	cmd.Dir = filepath.Join("testdata", "valid")
	// TUI needs a terminal — it will fail with an error about the terminal,
	// but NOT with "usage:" which means the auto-detect worked
	out, _ := cmd.CombinedOutput()
	output := string(out)
	if strings.Contains(output, "usage:") {
		t.Errorf("should auto-detect .ai/ in cwd, but got usage error: %s", output)
	}
}
