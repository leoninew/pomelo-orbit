package deploymentrunner

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestShellRunnerRunIncludesCommandOutputInError(t *testing.T) {
	dir := t.TempDir()
	var logBuf bytes.Buffer
	var err error
	if runtime.GOOS == "windows" {
		script := filepath.Join(dir, "fail.cmd")
		if writeErr := os.WriteFile(script, []byte("@echo unable to get image 1>&2\r\n@exit /b 1\r\n"), 0o644); writeErr != nil {
			t.Fatal(writeErr)
		}
		err = ShellRunner{}.Run(context.Background(), dir, &logBuf, script)
	} else {
		script := filepath.Join(dir, "fail.sh")
		if writeErr := os.WriteFile(script, []byte("#!/bin/sh\necho unable to get image 1>&2\nexit 1\n"), 0o755); writeErr != nil {
			t.Fatal(writeErr)
		}
		err = ShellRunner{}.Run(context.Background(), dir, &logBuf, script)
	}
	if err == nil {
		t.Fatal("expected command failure")
	}
	if !strings.Contains(err.Error(), "unable to get image") {
		t.Fatalf("error should include command output, got %v", err)
	}
	if !strings.Contains(err.Error(), "exit status") {
		t.Fatalf("error should keep exit status, got %v", err)
	}
	if !strings.Contains(logBuf.String(), "unable to get image") {
		t.Fatalf("log should include command output, got %q", logBuf.String())
	}
	if strings.Contains(logBuf.String(), "Working directory:") {
		t.Fatalf("runner must not duplicate the working directory, got %q", logBuf.String())
	}
	if !strings.Contains(logBuf.String(), "Running: ") {
		t.Fatalf("log should identify the command being run, got %q", logBuf.String())
	}
}

func TestCommandDisplayTextUsesMinimalQuoting(t *testing.T) {
	got := commandDisplayText("docker", "compose", "-p", "demo-default", "two words", `contains"quote`)
	want := `docker compose -p demo-default "two words" "contains\"quote"`
	if got != want {
		t.Fatalf("unexpected command display: got %q, want %q", got, want)
	}
}

func TestCommandErrorOutputKeepsTail(t *testing.T) {
	body := strings.Repeat("line\n", 2000) + "final failure reason"
	got := commandErrorOutput(body)
	if !strings.HasPrefix(got, "...\n") {
		t.Fatalf("expected truncated prefix, got %q", got[:min(20, len(got))])
	}
	if !strings.Contains(got, "final failure reason") {
		t.Fatalf("expected tail content, got %q", got[len(got)-80:])
	}
	if len(got) > maxErrorOutputBytes+10 {
		t.Fatalf("output longer than expected bound: %d", len(got))
	}
}

func TestCommandErrorOutputEmpty(t *testing.T) {
	if got := commandErrorOutput("  \n  "); got != "" {
		t.Fatalf("expected empty, got %q", got)
	}
}
