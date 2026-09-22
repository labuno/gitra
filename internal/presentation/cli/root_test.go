package cli

import (
	"bytes"
	"errors"
	"testing"

	"github.com/zhanhd/gitra/internal/bootstrap"
	"github.com/zhanhd/gitra/internal/domain"
)

func TestExitCodeFor(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want int
	}{
		{"success", nil, 0},
		{"usage", errUsage, 2},
		{"domain invalid", domain.ErrInvalid, 2},
		{"account missing", domain.ErrAccountNotFound, 3},
		{"account exists", domain.ErrAccountExists, 3},
		{"account in use", domain.ErrAccountInUse, 3},
		{"not a repo", domain.ErrNotGitRepository, 4},
		{"auth invalid", domain.ErrAuthInvalid, 5},
		{"identity mismatch", domain.ErrIdentityMismatch, 5},
		{"provider mismatch", domain.ErrProviderMismatch, 5},
		{"already bound", domain.ErrAlreadyBound, 6},
		{"binding missing", domain.ErrBindingNotFound, 6},
		{"drift", domain.ErrStateDrift, 6},
		{"unsupported repo", domain.ErrUnsupportedRepo, 7},
		{"unsupported remote", domain.ErrUnsupportedRemote, 7},
		{"internal", errors.New("boom"), 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ExitCodeFor(tt.err); got != tt.want {
				t.Fatalf("ExitCodeFor(%v) = %d, want %d", tt.err, got, tt.want)
			}
		})
	}
}

func TestExitCodeForWrapped(t *testing.T) {
	wrapped := errors.Join(errors.New("context"), domain.ErrAlreadyBound)
	if got := ExitCodeFor(wrapped); got != 6 {
		t.Fatalf("wrapped error code = %d, want 6", got)
	}
}

func newTestApp(t *testing.T) *App {
	t.Helper()
	t.Setenv("GITRA_CONFIG_DIR", t.TempDir())
	application, err := bootstrap.New()
	if err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	return New(application, &stdout, &stderr)
}

func TestExecuteHelpAndUnknownCommand(t *testing.T) {
	app := newTestApp(t)
	if code := app.Execute(nil, []string{"--help"}); code != 0 {
		t.Fatalf("--help exit code = %d, want 0", code)
	}
	if code := app.Execute(nil, []string{"--version"}); code != 0 {
		t.Fatalf("--version exit code = %d, want 0", code)
	}
	if code := app.Execute(nil, []string{"nope"}); code != 2 {
		t.Fatalf("unknown command exit code = %d, want 2", code)
	}
	if code := app.Execute(nil, []string{"--unknown-flag"}); code != 2 {
		t.Fatalf("unknown flag exit code = %d, want 2", code)
	}
}

func TestVerboseJournal(t *testing.T) {
	app := newTestApp(t)
	var buf bytes.Buffer
	app.Stderr = &buf
	app.verbose = false
	app.logf("should not appear")
	if buf.Len() != 0 {
		t.Fatalf("quiet mode wrote: %q", buf.String())
	}
	app.verbose = true
	app.journal = nil
	app.ensureJournal()
	app.logf("account listed")
	if !bytes.Contains(buf.Bytes(), []byte("account listed")) {
		t.Fatalf("verbose mode did not write: %q", buf.String())
	}
}
