package provider

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/zhanhd/gitra/internal/domain"
	"github.com/zhanhd/gitra/internal/ports"
)

type fakeRunner struct {
	outputs map[string]ports.ProcessResult
}

func (f *fakeRunner) Run(ctx context.Context, name string, args ...string) (ports.ProcessResult, error) {
	return f.RunWithInput(ctx, name, "", args...)
}
func (f *fakeRunner) RunWithInput(_ context.Context, name string, _ string, args ...string) (ports.ProcessResult, error) {
	key := name + " " + strings.Join(args, " ")
	if result, ok := f.outputs[key]; ok {
		return result, nil
	}
	return ports.ProcessResult{ExitCode: 127}, nil
}

func resolver(runner ports.CommandRunner, env map[string]string, stdin string) *TokenResolver {
	r := NewTokenResolver(runner, strings.NewReader(stdin))
	r.Getenv = func(key string) string { return env[key] }
	return r
}

func TestTokenResolverLadder(t *testing.T) {
	gh := &fakeRunner{outputs: map[string]ports.ProcessResult{
		"gh auth token": {ExitCode: 0, Stdout: "gho_from_cli\n"},
	}}
	ctx := context.Background()

	token, err := resolver(gh, nil, "").Resolve(ctx, TokenRequest{Provider: domain.ProviderGitHub, ExplicitToken: "explicit_tok"})
	if err != nil || token.Value != "explicit_tok" || token.Source != "explicit" {
		t.Fatalf("explicit token = %+v, %v", token, err)
	}
	token, err = resolver(gh, map[string]string{"GITRA_TOKEN": "env_tok"}, "").Resolve(ctx, TokenRequest{Provider: domain.ProviderGitHub})
	if err != nil || token.Value != "env_tok" || token.Source != "env" {
		t.Fatalf("env token = %+v, %v", token, err)
	}
	token, err = resolver(gh, nil, "stdin_tok\n").Resolve(ctx, TokenRequest{Provider: domain.ProviderGitHub, AllowStdin: true})
	if err != nil || token.Value != "stdin_tok" || token.Source != "stdin" {
		t.Fatalf("stdin token = %+v, %v", token, err)
	}
	token, err = resolver(gh, nil, "").Resolve(ctx, TokenRequest{Provider: domain.ProviderGitHub, AllowCLIReuse: true})
	if err != nil || token.Value != "gho_from_cli" || token.Source != "gh" {
		t.Fatalf("gh token = %+v, %v", token, err)
	}
}

func TestTokenResolverExplainsHowToGetToken(t *testing.T) {
	empty := &fakeRunner{outputs: map[string]ports.ProcessResult{}}
	_, err := resolver(empty, nil, "").Resolve(context.Background(), TokenRequest{Provider: domain.ProviderGitHub, AllowCLIReuse: true})
	if !errors.Is(err, domain.ErrAuthInvalid) {
		t.Fatalf("error = %v, want ErrAuthInvalid", err)
	}
	for _, want := range []string{"--token-stdin", "GITRA_TOKEN", "personal access token"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error must mention %q: %v", want, err)
		}
	}
}
