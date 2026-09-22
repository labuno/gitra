package runner

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestRunCapturesStreamsAndExitCode(t *testing.T) {
	result, err := New().Run(context.Background(), "sh", "-c", "echo out; echo err >&2; exit 3")
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if result.ExitCode != 3 {
		t.Fatalf("ExitCode = %d, want 3", result.ExitCode)
	}
	if strings.TrimSpace(result.Stdout) != "out" {
		t.Fatalf("Stdout = %q", result.Stdout)
	}
	if strings.TrimSpace(result.Stderr) != "err" {
		t.Fatalf("Stderr = %q", result.Stderr)
	}
}

func TestRunMissingBinaryIsExecutionError(t *testing.T) {
	if _, err := New().Run(context.Background(), "gitra-missing-binary-for-test"); err == nil {
		t.Fatal("Run() error = nil, want execution error for missing binary")
	}
}

func TestRunContextCancelIsExecutionError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	if _, err := New().Run(ctx, "sleep", "5"); err == nil {
		t.Fatal("Run() error = nil, want cancellation error")
	}
}

func TestRunPassesArgumentsWithoutShell(t *testing.T) {
	result, err := New().Run(context.Background(), "printf", "%s", "a b; rm -rf /")
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if result.Stdout != "a b; rm -rf /" {
		t.Fatalf("Stdout = %q, want literal argument", result.Stdout)
	}
}
