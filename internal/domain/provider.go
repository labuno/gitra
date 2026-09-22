package domain

import "strings"

// ProviderType enumerates the hosting providers modelled by V1.
type ProviderType string

const (
	ProviderGitHub ProviderType = "github"
	ProviderGitLab ProviderType = "gitlab"
	ProviderGitea  ProviderType = "gitea"
)

// Auth strategy identifiers (baseline §15: ssh-key only in V1).
const StrategySSHKey = "ssh-key"

// ProviderEndpoint models the SSH endpoint explicitly (baseline §4.3):
// self-hosted instances may use custom hosts and non-22 ports.
type ProviderEndpoint struct {
	Host    string
	SSHUser string
	SSHPort int
}

// Validate checks the endpoint invariants. SSHPort must be concrete (1..65535);
// default constructors provide 22 where the provider uses its default.
func (e ProviderEndpoint) Validate() error {
	if strings.TrimSpace(e.Host) == "" {
		return invalid("provider.endpoint.host", "must not be empty")
	}
	if strings.TrimSpace(e.SSHUser) == "" {
		return invalid("provider.endpoint.ssh_user", "must not be empty")
	}
	if e.SSHPort < 1 || e.SSHPort > 65535 {
		return invalid("provider.endpoint.ssh_port", "must be within 1..65535")
	}
	return nil
}

// IsDefaultSSHPort reports whether the endpoint uses the standard SSH port.
func (e ProviderEndpoint) IsDefaultSSHPort() bool { return e.SSHPort == 22 }

// SSHCommandNeedsPort encodes the baseline §14 revision: the generated
// core.sshCommand must only append -p when the remote URL does not already
// carry an explicit port and the endpoint is not on 22.
func (e ProviderEndpoint) SSHCommandNeedsPort(remoteHasExplicitPort bool) bool {
	return !remoteHasExplicitPort && !e.IsDefaultSSHPort()
}

// ProviderRef identifies the provider account behind a local alias
// (baseline §4.1: AccountID is the durable identity, the alias is not).
type ProviderRef struct {
	Type     ProviderType
	Username string
	Endpoint ProviderEndpoint
}

// Valid reports whether the provider type is supported by V1.
func (t ProviderType) Valid() bool {
	switch t {
	case ProviderGitHub, ProviderGitLab, ProviderGitea:
		return true
	default:
		return false
	}
}

func (p ProviderRef) Validate() error {
	if !p.Type.Valid() {
		return invalid("provider.type", "must be github, gitlab or gitea")
	}
	if strings.TrimSpace(p.Username) == "" {
		return invalid("provider.username", "must not be empty")
	}
	if err := p.Endpoint.Validate(); err != nil {
		return err
	}
	return nil
}

// DefaultEndpoint returns the documented defaults for the hosted providers
// (baseline §4.3). The second result is false for unknown provider types.
func DefaultEndpoint(t ProviderType) (ProviderEndpoint, bool) {
	switch t {
	case ProviderGitHub:
		return ProviderEndpoint{Host: "github.com", SSHUser: "git", SSHPort: 22}, true
	case ProviderGitLab:
		return ProviderEndpoint{Host: "gitlab.com", SSHUser: "git", SSHPort: 22}, true
	case ProviderGitea:
		return ProviderEndpoint{Host: "gitea.com", SSHUser: "git", SSHPort: 22}, true
	default:
		return ProviderEndpoint{}, false
	}
}
