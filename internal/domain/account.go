package domain

import (
	"regexp"
	"strings"
)

// AccountID is the stable, immutable identity of an account (baseline §4.1).
type AccountID string

// BindingID is the stable identity of a repository binding.
type BindingID string

var (
	accountIDPattern = regexp.MustCompile(`^acc_[A-Za-z0-9]{2,40}$`)
	// Alias is a local, human-readable identifier: lowercase, 1..32 chars.
	aliasPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{0,31}$`)
)

// CommitIdentity maps to git config user.name / user.email.
type CommitIdentity struct {
	Name  string
	Email string
}

func (c CommitIdentity) Validate() error {
	if strings.TrimSpace(c.Name) == "" {
		return invalid("identity.name", "must not be empty")
	}
	email := strings.TrimSpace(c.Email)
	if !strings.Contains(email, "@") || strings.ContainsAny(email, " \t\n") {
		return invalid("identity.email", "must look like an email address")
	}
	return nil
}

// TransportConfig selects how git authenticates; secrets never live here,
// only references such as a private key path.
type TransportConfig struct {
	Strategy string
	Config   map[string]string
}

func (t TransportConfig) Validate() error {
	switch t.Strategy {
	case StrategySSHKey:
		if strings.TrimSpace(t.Config["private_key"]) == "" {
			return invalid("transport.config.private_key", "must not be empty")
		}
		return nil
	default:
		return invalid("transport.strategy", "must be ssh-key in V1")
	}
}

// Account is the desired state for one Git hosting account.
type Account struct {
	ID        AccountID
	Alias     string
	Provider  ProviderRef
	Identity  CommitIdentity
	Transport TransportConfig
	Revision  int
}

// Validate enforces the account invariants in a deterministic order:
// id → alias → provider → identity → transport → revision.
func (a Account) Validate() error {
	if !accountIDPattern.MatchString(string(a.ID)) {
		return invalid("id", "must match acc_<alphanumeric>")
	}
	if !aliasPattern.MatchString(a.Alias) {
		return invalid("alias", "must be 1..32 lowercase characters [a-z0-9._-] starting alphanumeric")
	}
	if err := a.Provider.Validate(); err != nil {
		return err
	}
	if err := a.Identity.Validate(); err != nil {
		return err
	}
	if err := a.Transport.Validate(); err != nil {
		return err
	}
	if a.Revision < 0 {
		return invalid("revision", "must not be negative")
	}
	return nil
}
