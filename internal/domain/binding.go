package domain

import "regexp"

// BindingHealth follows baseline §13.
type BindingHealth string

const (
	HealthOK      BindingHealth = "ok"
	HealthDrift   BindingHealth = "drift"
	HealthMissing BindingHealth = "missing"
	HealthBroken  BindingHealth = "broken"
)

// Valid reports whether the value is one of the four documented states.
func (h BindingHealth) Valid() bool {
	switch h {
	case HealthOK, HealthDrift, HealthMissing, HealthBroken:
		return true
	default:
		return false
	}
}

// Routing strategy identifiers (baseline §19: repo-local only in V1).
const StrategyRepoLocal = "repo-local"

var bindingIDPattern = regexp.MustCompile(`^bnd_[A-Za-z0-9]{2,40}$`)

// RepositoryBinding binds one repository to exactly one account
// (baseline §10: one repository has at most one account in V1).
type RepositoryBinding struct {
	ID         BindingID
	AccountID  AccountID
	Repository RepositoryRef
	Strategy   string
	Revision   int
}

func (b RepositoryBinding) Validate() error {
	if !bindingIDPattern.MatchString(string(b.ID)) {
		return invalid("id", "must match bnd_<alphanumeric>")
	}
	if !accountIDPattern.MatchString(string(b.AccountID)) {
		return invalid("account_id", "must match acc_<alphanumeric>")
	}
	if err := b.Repository.Validate(); err != nil {
		return err
	}
	if b.Strategy != StrategyRepoLocal {
		return invalid("strategy", "must be repo-local in V1")
	}
	if b.Revision < 0 {
		return invalid("revision", "must not be negative")
	}
	return nil
}
