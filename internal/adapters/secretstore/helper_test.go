package secretstore

import (
	"context"
	"strings"
	"testing"

	"github.com/zhanhd/gitra/internal/ports"
)

type scriptedRunner struct {
	responses map[string]ports.ProcessResult
}

func (s *scriptedRunner) Run(ctx context.Context, name string, args ...string) (ports.ProcessResult, error) {
	return s.RunWithInput(ctx, name, "", args...)
}

func (s *scriptedRunner) RunWithInput(_ context.Context, name string, _ string, args ...string) (ports.ProcessResult, error) {
	key := strings.Join(args, " ")
	if result, ok := s.responses[key]; ok {
		return result, nil
	}
	return ports.ProcessResult{ExitCode: 1}, nil
}

func TestResolveHelperPrefersUserConfigured(t *testing.T) {
	r := &scriptedRunner{responses: map[string]ports.ProcessResult{
		"config --get-all credential.helper": {ExitCode: 0, Stdout: "osxkeychain\n"},
	}}
	helper, err := ResolveHelper(context.Background(), r, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if helper != "" {
		t.Fatalf("helper = %q, want empty (reuse user configured helper)", helper)
	}
}

func TestResolveHelperFallsBackToManagedFile(t *testing.T) {
	r := &scriptedRunner{responses: map[string]ports.ProcessResult{
		"config --get-all credential.helper": {ExitCode: 1},
		"--exec-path":                        {ExitCode: 0, Stdout: "/nonexistent/exec-path\n"},
	}}
	dir := t.TempDir()
	helper, err := ResolveHelper(context.Background(), r, dir)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(helper, "store --file=") || !strings.Contains(helper, dir) {
		t.Fatalf("helper = %q, want managed store --file under %s", helper, dir)
	}
}

func TestResolveHelperUsesPlatformHelperWhenAvailable(t *testing.T) {
	dir := t.TempDir()
	// Simulate git's exec-path containing the osxkeychain helper.
	r := &scriptedRunner{responses: map[string]ports.ProcessResult{
		"config --get-all credential.helper": {ExitCode: 1},
		"--exec-path":                        {ExitCode: 0, Stdout: dir + "\n"},
	}}
	touch(t, dir+"/git-credential-osxkeychain")
	helper, err := ResolveHelper(context.Background(), r, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if helper != "osxkeychain" {
		t.Fatalf("helper = %q, want osxkeychain", helper)
	}
}
