package main

import (
	"bytes"
	"errors"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

var nasaBin string

func TestMain(m *testing.M) {
	// Build the binary to a temp directory.
	tmpDir, err := os.MkdirTemp("", "nasa-cli-test")
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	binPath := filepath.Join(tmpDir, "nasa")
	cmd := exec.Command("go", "build", "-o", binPath, "./")
	cmd.Dir = "."
	if out, err := cmd.CombinedOutput(); err != nil {
		log.Fatalf("build failed: %s\n%s", err, out)
	}

	nasaBin = binPath
	os.Exit(m.Run())
}

// runCLI executes the nasa binary with args and returns stdout, stderr, and exit code.
func runCLI(t *testing.T, args ...string) (stdout, stderr string, exitCode int) {
	t.Helper()
	cmd := exec.Command(nasaBin, args...)
	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf
	err := cmd.Run()
	exitCode = 0
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			exitCode = exitErr.ExitCode()
		} else {
			t.Fatalf("exec error: %v", err)
		}
	}
	return outBuf.String(), errBuf.String(), exitCode
}

func TestCLI_Help(t *testing.T) {
	stdout, _, exitCode := runCLI(t, "--help")
	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d", exitCode)
	}
	for _, svc := range []string{"apod", "neo", "donki", "images", "ssd", "eonet"} {
		if !strings.Contains(stdout, svc) {
			t.Errorf("expected stdout to contain %q", svc)
		}
	}
}

func TestCLI_Version(t *testing.T) {
	stdout, _, exitCode := runCLI(t, "--version")
	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d", exitCode)
	}
	if !strings.Contains(stdout, "dev") {
		t.Errorf("expected stdout to contain %q, got %q", "dev", stdout)
	}
}

func TestCLI_UnknownCommand(t *testing.T) {
	_, stderr, exitCode := runCLI(t, "foobar")
	if exitCode == 0 {
		t.Fatal("expected non-zero exit code")
	}
	if !strings.Contains(stderr, "unknown command") {
		t.Errorf("expected stderr to contain %q, got %q", "unknown command", stderr)
	}
}

func TestCLI_NoArgs(t *testing.T) {
	_, stderr, exitCode := runCLI(t)
	if exitCode == 0 {
		t.Fatal("expected non-zero exit code")
	}
	if !strings.Contains(stderr, "Usage:") {
		t.Errorf("expected stderr to contain %q, got %q", "Usage:", stderr)
	}
}

func TestCLI_ServiceHelp(t *testing.T) {
	stdout, _, exitCode := runCLI(t, "donki", "--help")
	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d", exitCode)
	}
	for _, sub := range []string{"cme", "gst", "flr"} {
		if !strings.Contains(stdout, sub) {
			t.Errorf("expected stdout to contain %q", sub)
		}
	}
}
