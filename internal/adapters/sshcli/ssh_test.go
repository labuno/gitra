package sshcli

import (
	"context"
	"strings"
	"testing"

	"github.com/zhanhd/gitra/internal/ports"
)

type fakeRunner struct {
	calls []string
	out   ports.ProcessResult
	err   error
}

func (f *fakeRunner) Run(_ context.Context, name string, args ...string) (ports.ProcessResult, error) {
	f.calls = append(f.calls, strings.Join(append([]string{name}, args...), " "))
	return f.out, f.err
}
func (f *fakeRunner) RunWithInput(ctx context.Context, name, _ string, args ...string) (ports.ProcessResult, error) {
	return f.Run(ctx, name, args...)
}

func TestSSHCommandShape(t *testing.T) {
	runner := &fakeRunner{out: ports.ProcessResult{ExitCode: 1, Stderr: "Hi lunafoundry! You've successfully authenticated"}}
	adapter := New(runner)
	result, err := adapter.Test(context.Background(), ports.SSHTestRequest{
		Host: "github.com", User: "git", Port: 22, PrivateKeyPath: "/keys/id_ed25519",
	})
	if err != nil {
		t.Fatal(err)
	}
	want := `ssh -T -i /keys/id_ed25519 -o IdentitiesOnly=yes git@github.com`
	if runner.calls[0] != want {
		t.Fatalf("argv = %q, want %q", runner.calls[0], want)
	}
	if result.ExitCode != 1 || !strings.Contains(result.Stderr, "successfully authenticated") {
		t.Fatalf("raw result not preserved: %+v", result)
	}
}

func TestSSHCommandCustomPort(t *testing.T) {
	runner := &fakeRunner{}
	adapter := New(runner)
	if _, err := adapter.Test(context.Background(), ports.SSHTestRequest{
		Host: "git.example.com", User: "git", Port: 2222, PrivateKeyPath: "/keys/k",
	}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(runner.calls[0], "-p 2222") {
		t.Fatalf("argv = %q, want -p 2222", runner.calls[0])
	}
	if runner.calls[0] != `ssh -T -i /keys/k -o IdentitiesOnly=yes -p 2222 git@git.example.com` {
		t.Fatalf("argv = %q", runner.calls[0])
	}
}
