package app

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/zhanhd/gitra/internal/domain"
	"github.com/zhanhd/gitra/internal/ports"
)

type failingTokens struct{}

func (failingTokens) Acquire(context.Context, ports.TokenRequest) (ports.LoginToken, error) {
	return ports.LoginToken{}, errors.New("no CLI session")
}

type stubDetectSSH struct{}

func (stubDetectSSH) Test(context.Context, ports.SSHTestRequest) (ports.SSHTestResult, error) {
	return ports.SSHTestResult{ExitCode: 1, Stderr: "Hi lunafoundry! You've successfully authenticated"}, nil
}

type stubDetectParser struct{}

func (stubDetectParser) ParseSSHIdentity(context.Context, domain.ProviderType, string, string) (string, bool) {
	return "lunafoundry", true
}

func TestDetectorOffersEveryLoggedInCLIAccount(t *testing.T) {
	deps := Deps{SSH: stubDetectSSH{}}
	detector := NewDetector(deps, failingTokens{}, stubDetectParser{})
	detector.homeDir = func() (string, error) { return t.TempDir(), nil } // no SSH keys
	detector.SetCLIAccounts(func(context.Context, domain.ProviderType, string) []string {
		return []string{"labuno", "lucas-zan"}
	})

	candidates := detector.Detect(context.Background(), domain.ProviderGitHub, "github.com")
	var names []string
	for _, candidate := range candidates {
		if candidate.Kind == "cli" {
			names = append(names, candidate.Username)
		}
	}
	if len(names) != 2 || names[0] != "labuno" || names[1] != "lucas-zan" {
		t.Fatalf("cli candidates = %v (all=%+v)", names, candidates)
	}
	for _, candidate := range candidates {
		if !strings.Contains(candidate.Label, candidate.Username) {
			t.Fatalf("label must name the account: %+v", candidate)
		}
	}
}

func TestDetectorFallsBackToActiveCLISession(t *testing.T) {
	deps := Deps{SSH: stubDetectSSH{}}
	detector := NewDetector(deps, stubTokensOK{token: ports.LoginToken{Value: "tok", Source: "gh"}}, stubDetectParser{})
	detector.homeDir = func() (string, error) { return t.TempDir(), nil }
	detector.SetCLIAccounts(func(context.Context, domain.ProviderType, string) []string { return nil })

	candidates := detector.Detect(context.Background(), domain.ProviderGitHub, "github.com")
	if len(candidates) != 1 || candidates[0].Kind != "cli" || candidates[0].Username != "" {
		t.Fatalf("candidates = %+v, want one generic CLI candidate", candidates)
	}
}

func TestDetectorOffersVerifiedSSHKeys(t *testing.T) {
	home := t.TempDir()
	sshDir := home + "/.ssh"
	if err := osMkdirAll(sshDir); err != nil {
		t.Fatal(err)
	}
	if err := osWriteFile(sshDir+"/id_ed25519", []byte("key"), 0o600); err != nil {
		t.Fatal(err)
	}

	detector := NewDetector(Deps{SSH: stubDetectSSH{}}, failingTokens{}, stubDetectParser{})
	detector.homeDir = func() (string, error) { return home, nil }
	detector.SetCLIAccounts(func(context.Context, domain.ProviderType, string) []string { return nil })

	candidates := detector.Detect(context.Background(), domain.ProviderGitHub, "github.com")
	if len(candidates) != 1 || candidates[0].Kind != "ssh-key" || candidates[0].Username != "lunafoundry" {
		t.Fatalf("candidates = %+v", candidates)
	}
	if !strings.Contains(candidates[0].Label, "id_ed25519") {
		t.Fatalf("label = %q", candidates[0].Label)
	}
}
