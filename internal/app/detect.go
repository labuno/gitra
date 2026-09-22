package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/zhanhd/gitra/internal/domain"
	"github.com/zhanhd/gitra/internal/ports"
)

// Candidate is one login that already exists on this machine, offered to the
// user as a single click (nothing to fetch, nothing to paste).
type Candidate struct {
	Kind     string // "cli" | "ssh-key"
	Source   string // gh | glab | ssh
	Label    string
	Provider domain.ProviderType
	Username string
	Host     string
	KeyPath  string
}

// Detector discovers reusable logins: an authenticated gh/glab session and
// private keys under ~/.ssh that a provider already accepts.
type Detector struct {
	deps         Deps
	tokens       ports.TokenProvider
	parsers      ports.SSHIdentityParser
	homeDir      func() (string, error)
	probeTimeout time.Duration
	maxKeys      int
}

// NewDetector builds the detector.
func NewDetector(deps Deps, tokens ports.TokenProvider, parsers ports.SSHIdentityParser) *Detector {
	return &Detector{
		deps: deps, tokens: tokens, parsers: parsers,
		homeDir: os.UserHomeDir, probeTimeout: 4 * time.Second, maxKeys: 3,
	}
}

// Detect returns the candidates for one provider, ready to render.
func (d *Detector) Detect(ctx context.Context, providerType domain.ProviderType, host string) []Candidate {
	var candidates []Candidate
	if d.tokens != nil {
		if providerType == domain.ProviderGitHub || providerType == domain.ProviderGitLab {
			if _, err := d.tokens.Acquire(ctx, providerType, "", false, true); err == nil {
				source := "gh"
				if providerType == domain.ProviderGitLab {
					source = "glab"
				}
				candidates = append(candidates, Candidate{
					Kind: "cli", Source: source, Host: host, Provider: providerType,
					Label: fmt.Sprintf("使用本机已登录的 %s（无需输入）", providerLabel(providerType)),
				})
			}
		}
	}
	candidates = append(candidates, d.detectSSHKeys(ctx, providerType, host)...)
	return candidates
}

func (d *Detector) detectSSHKeys(ctx context.Context, providerType domain.ProviderType, host string) []Candidate {
	if d.deps.SSH == nil || d.parsers == nil {
		return nil
	}
	endpoint, ok := domain.DefaultEndpoint(providerType)
	if !ok {
		return nil
	}
	if host != "" {
		endpoint.Host = host
	}

	home, err := d.homeDir()
	if err != nil {
		return nil
	}
	sshDir := filepath.Join(home, ".ssh")
	entries, err := os.ReadDir(sshDir)
	if err != nil {
		return nil
	}
	var paths []string
	for _, entry := range entries {
		if entry.IsDir() || !looksLikePrivateKey(entry.Name()) {
			continue
		}
		paths = append(paths, filepath.Join(sshDir, entry.Name()))
	}
	sort.Strings(paths)
	if len(paths) > d.maxKeys {
		paths = paths[:d.maxKeys]
	}

	// Probe the candidate keys in parallel with one overall deadline so the
	// login screen never waits on several slow handshakes in a row.
	type probeResult struct {
		candidate Candidate
		ok        bool
	}
	results := make(chan probeResult, len(paths))
	var wait sync.WaitGroup
	for _, path := range paths {
		wait.Add(1)
		go func(keyPath string) {
			defer wait.Done()
			probeCtx, cancel := context.WithTimeout(ctx, d.probeTimeout)
			defer cancel()
			result, err := d.deps.SSH.Test(probeCtx, ports.SSHTestRequest{
				Host: endpoint.Host, User: endpoint.SSHUser, Port: endpoint.SSHPort, PrivateKeyPath: keyPath,
			})
			if err != nil {
				results <- probeResult{}
				return
			}
			username, ok := d.parsers.ParseSSHIdentity(ctx, providerType, result.Stdout, result.Stderr)
			if !ok || username == "" {
				results <- probeResult{}
				return
			}
			results <- probeResult{ok: true, candidate: Candidate{
				Kind: "ssh-key", Source: "ssh", Host: endpoint.Host, Provider: providerType,
				Username: username, KeyPath: keyPath,
				Label: fmt.Sprintf("使用已有密钥 %s（已验证属于 %s）", filepath.Base(keyPath), username),
			}}
		}(path)
	}
	wait.Wait()
	close(results)

	var candidates []Candidate
	for result := range results {
		if result.ok {
			candidates = append(candidates, result.candidate)
		}
	}
	sort.Slice(candidates, func(i, j int) bool { return candidates[i].KeyPath < candidates[j].KeyPath })
	return candidates
}

func looksLikePrivateKey(name string) bool {
	if strings.HasSuffix(name, ".pub") || strings.HasPrefix(name, "known_hosts") {
		return false
	}
	switch name {
	case "config", "authorized_keys", "agent.sock":
		return false
	}
	lower := strings.ToLower(name)
	return strings.HasPrefix(name, "id_") ||
		strings.Contains(lower, "github") || strings.Contains(lower, "gitlab") ||
		strings.Contains(lower, "gitea") || strings.Contains(lower, "git")
}

func providerLabel(providerType domain.ProviderType) string {
	switch providerType {
	case domain.ProviderGitHub:
		return "GitHub"
	case domain.ProviderGitLab:
		return "GitLab"
	case domain.ProviderGitea:
		return "Gitea"
	default:
		return string(providerType)
	}
}

// SSHLoginRequest registers an account backed by an existing SSH key.
type SSHLoginRequest struct {
	Provider domain.ProviderType
	Host     string
	KeyPath  string
	Username string
	Alias    string
}

// LoginWithSSHKey creates or updates an ssh-key account for a key that the
// provider already accepts. The commit identity reuses the machine's global
// git identity when present, otherwise a provider noreply address.
func (s *LoginService) LoginWithSSHKey(ctx context.Context, req SSHLoginRequest) (LoginResult, error) {
	if strings.TrimSpace(req.KeyPath) == "" || strings.TrimSpace(req.Username) == "" {
		return LoginResult{}, fmt.Errorf("%w: key path and username are required", domain.ErrInvalid)
	}
	endpoint, ok := domain.DefaultEndpoint(req.Provider)
	if !ok {
		return LoginResult{}, fmt.Errorf("%w: unknown provider %q", domain.ErrInvalid, req.Provider)
	}
	if req.Host != "" {
		endpoint.Host = req.Host
	}

	identity := machineIdentity(ctx, s.deps, req.Provider, endpoint.Host, req.Username)
	alias := req.Alias
	if alias == "" {
		alias = req.Username
	}

	existing, found, err := s.findAccount(ctx, req.Provider, endpoint.Host, req.Username)
	if err != nil {
		return LoginResult{}, err
	}
	if found {
		next := existing
		next.Provider.Endpoint = endpoint
		next.Transport = domain.TransportConfig{Strategy: domain.StrategySSHKey, Config: map[string]string{"private_key": req.KeyPath}}
		if req.Alias != "" {
			next.Alias = req.Alias
		}
		if next.Identity.Name == "" {
			next.Identity.Name = identity.Name
		}
		if next.Identity.Email == "" {
			next.Identity.Email = identity.Email
		}
		updated, err := s.accounts.Update(ctx, UpdateAccountRequest{Account: next})
		if err != nil {
			return LoginResult{Account: updated}, err
		}
		return LoginResult{Account: updated, Profile: profileFor(req, endpoint.Host), Created: false, TokenSource: "ssh-key"}, nil
	}

	account, err := s.accounts.Create(ctx, CreateAccountRequest{
		Alias: alias,
		Provider: domain.ProviderRef{
			Type: req.Provider, Username: req.Username, Endpoint: endpoint,
		},
		Identity:  identity,
		Transport: domain.TransportConfig{Strategy: domain.StrategySSHKey, Config: map[string]string{"private_key": req.KeyPath}},
	})
	if err != nil {
		return LoginResult{}, err
	}
	return LoginResult{Account: account, Profile: profileFor(req, endpoint.Host), Created: true, TokenSource: "ssh-key"}, nil
}

func profileFor(req SSHLoginRequest, host string) ports.ProviderProfile {
	return ports.ProviderProfile{Provider: req.Provider, Host: host, Username: req.Username}
}

// machineIdentity reuses what the machine already uses for commits.
func machineIdentity(ctx context.Context, deps Deps, providerType domain.ProviderType, host, username string) domain.CommitIdentity {
	identity := domain.CommitIdentity{Name: username, Email: noreplyEmail(providerType, host, username)}
	if deps.Git == nil {
		return identity
	}
	if value, found, err := deps.Git.GetGlobalConfig(ctx, "user.name"); err == nil && found && strings.TrimSpace(value) != "" {
		identity.Name = strings.TrimSpace(value)
	}
	if value, found, err := deps.Git.GetGlobalConfig(ctx, "user.email"); err == nil && found && strings.TrimSpace(value) != "" {
		identity.Email = strings.TrimSpace(value)
	}
	return identity
}

func noreplyEmail(providerType domain.ProviderType, host, username string) string {
	switch providerType {
	case domain.ProviderGitHub:
		return username + "@users.noreply.github.com"
	case domain.ProviderGitLab:
		return username + "@users.noreply.gitlab.com"
	default:
		return username + "@" + host
	}
}
