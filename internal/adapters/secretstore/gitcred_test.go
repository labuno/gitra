package secretstore

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/zhanhd/gitra/internal/adapters/runner"
	"github.com/zhanhd/gitra/internal/domain"
	"github.com/zhanhd/gitra/internal/ports"
)

type recordedCall struct {
	name  string
	args  []string
	stdin string
}

type fakeRunner struct {
	calls   []recordedCall
	results []ports.ProcessResult
	index   int
}

func (f *fakeRunner) Run(_ context.Context, name string, args ...string) (ports.ProcessResult, error) {
	return f.RunWithInput(context.Background(), name, "", args...)
}

func (f *fakeRunner) RunWithInput(_ context.Context, name string, input string, args ...string) (ports.ProcessResult, error) {
	call := recordedCall{name: name, args: append([]string(nil), args...), stdin: input}
	f.calls = append(f.calls, call)
	if f.index >= len(f.results) {
		f.index++
		return ports.ProcessResult{}, nil
	}
	result := f.results[f.index]
	f.index++
	return result, nil
}

func TestSetApproveCommandShape(t *testing.T) {
	fr := &fakeRunner{}
	store := NewGitCredentialStore(fr, "osxkeychain")
	if err := store.Set(context.Background(), "github.com/lunafoundry", "gho_token"); err != nil {
		t.Fatalf("Set() error = %v", err)
	}
	if len(fr.calls) != 1 {
		t.Fatalf("calls = %+v", fr.calls)
	}
	call := fr.calls[0]
	want := []string{"-c", "credential.helper=osxkeychain", "credential", "approve"}
	for i := range want {
		if call.args[i] != want[i] {
			t.Fatalf("argv = %v, want %v", call.args, want)
		}
	}
	for _, arg := range call.args {
		if strings.Contains(arg, "gho_token") {
			t.Fatal("token must never appear in argv")
		}
	}
}

func TestRefParsingAndErrors(t *testing.T) {
	fr := &fakeRunner{}
	store := NewGitCredentialStore(fr, "")
	if _, err := store.Get(context.Background(), "no-username"); !errors.Is(err, domain.ErrAuthInvalid) {
		t.Fatalf("bad ref error = %v, want ErrAuthInvalid", err)
	}
	if err := store.Set(context.Background(), "bad-ref", "x"); !errors.Is(err, domain.ErrAuthInvalid) {
		t.Fatalf("bad ref set = %v, want ErrAuthInvalid", err)
	}
}

func TestGetParsesPassword(t *testing.T) {
	fr := &fakeRunner{results: []ports.ProcessResult{{
		ExitCode: 0,
		Stdout:   "protocol=https\nhost=github.com\nusername=lunafoundry\npassword=gho_secret\n",
	}}}
	store := NewGitCredentialStore(fr, "")
	secret, err := store.Get(context.Background(), "github.com/lunafoundry")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if secret != "gho_secret" {
		t.Fatalf("secret = %q", secret)
	}
}

func TestGetMissingSecretIsAuthInvalid(t *testing.T) {
	fr := &fakeRunner{results: []ports.ProcessResult{{ExitCode: 1}}}
	store := NewGitCredentialStore(fr, "")
	if _, err := store.Get(context.Background(), "github.com/ghost"); !errors.Is(err, domain.ErrAuthInvalid) {
		t.Fatalf("missing secret error = %v, want ErrAuthInvalid", err)
	}
}

func TestDeleteRejectsAndIsIdempotent(t *testing.T) {
	fr := &fakeRunner{results: []ports.ProcessResult{{ExitCode: 0}, {ExitCode: 1}}}
	store := NewGitCredentialStore(fr, "")
	if err := store.Delete(context.Background(), "github.com/lunafoundry"); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if err := store.Delete(context.Background(), "github.com/lunafoundry"); err != nil {
		t.Fatalf("second Delete() error = %v", err)
	}
}

// Integration: real git with the built-in store helper on an isolated file.
func TestStoreRoundTripWithRealGit(t *testing.T) {
	dir := t.TempDir()
	helper := "store --file=" + filepath.Join(dir, "credentials")
	store := NewGitCredentialStore(runner.New(), helper)
	ctx := context.Background()

	if err := store.Set(ctx, "github.com/lunafoundry", "tok_round_trip"); err != nil {
		t.Fatalf("Set() error = %v", err)
	}
	secret, err := store.Get(ctx, "github.com/lunafoundry")
	if err != nil || secret != "tok_round_trip" {
		t.Fatalf("Get() = (%q, %v)", secret, err)
	}
	if err := store.Delete(ctx, "github.com/lunafoundry"); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if _, err := store.Get(ctx, "github.com/lunafoundry"); !errors.Is(err, domain.ErrAuthInvalid) {
		t.Fatalf("after delete error = %v, want ErrAuthInvalid", err)
	}
}

type blockingRunner struct{}

func (blockingRunner) Run(ctx context.Context, _ string, _ ...string) (ports.ProcessResult, error) {
	<-ctx.Done()
	return ports.ProcessResult{}, ctx.Err()
}
func (b blockingRunner) RunWithInput(ctx context.Context, name, _ string, args ...string) (ports.ProcessResult, error) {
	return b.Run(ctx, name, args...)
}

func TestCredentialOperationsAreBounded(t *testing.T) {
	store := NewGitCredentialStore(blockingRunner{}, "")
	start := time.Now()
	if err := store.Set(context.Background(), "github.com/lunafoundry", "tok"); err == nil {
		t.Fatal("a blocked credential store must return an error")
	}
	if elapsed := time.Since(start); elapsed > 30*time.Second {
		t.Fatalf("Set took %s; credential operations must be bounded", elapsed)
	}
}
