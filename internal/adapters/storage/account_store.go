package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/zhanhd/gitra/internal/domain"
)

// AccountStore implements ports.AccountStore with one JSON file per account,
// named by AccountID (baseline §28).
type AccountStore struct {
	dir string
}

// NewAccountStore roots the store at the gitra config directory.
func NewAccountStore(dir string) *AccountStore { return &AccountStore{dir: dir} }

func (s *AccountStore) accountsDir() string { return filepath.Join(s.dir, "accounts") }

type endpointFile struct {
	Host    string `json:"host"`
	SSHUser string `json:"ssh_user"`
	SSHPort int    `json:"ssh_port"`
}

type providerFile struct {
	Type     string       `json:"type"`
	Username string       `json:"username"`
	Endpoint endpointFile `json:"endpoint"`
}

type identityFile struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

type transportFile struct {
	Strategy string            `json:"strategy"`
	Config   map[string]string `json:"config"`
}

type accountFile struct {
	SchemaVersion int           `json:"schema_version"`
	ID            string        `json:"id"`
	Alias         string        `json:"alias"`
	Provider      providerFile  `json:"provider"`
	Identity      identityFile  `json:"identity"`
	Transport     transportFile `json:"transport"`
	Revision      int           `json:"revision"`
}

func toAccountFile(a domain.Account) accountFile {
	return accountFile{
		SchemaVersion: schemaVersion,
		ID:            string(a.ID),
		Alias:         a.Alias,
		Provider: providerFile{
			Type:     string(a.Provider.Type),
			Username: a.Provider.Username,
			Endpoint: endpointFile{
				Host:    a.Provider.Endpoint.Host,
				SSHUser: a.Provider.Endpoint.SSHUser,
				SSHPort: a.Provider.Endpoint.SSHPort,
			},
		},
		Identity: identityFile{Name: a.Identity.Name, Email: a.Identity.Email},
		Transport: transportFile{
			Strategy: a.Transport.Strategy,
			Config:   a.Transport.Config,
		},
		Revision: a.Revision,
	}
}

func (f accountFile) toDomain() (domain.Account, error) {
	if f.SchemaVersion != schemaVersion {
		return domain.Account{}, fmt.Errorf("unsupported account schema_version %d", f.SchemaVersion)
	}
	account := domain.Account{
		ID:    domain.AccountID(f.ID),
		Alias: f.Alias,
		Provider: domain.ProviderRef{
			Type:     domain.ProviderType(f.Provider.Type),
			Username: f.Provider.Username,
			Endpoint: domain.ProviderEndpoint{
				Host:    f.Provider.Endpoint.Host,
				SSHUser: f.Provider.Endpoint.SSHUser,
				SSHPort: f.Provider.Endpoint.SSHPort,
			},
		},
		Identity:  domain.CommitIdentity{Name: f.Identity.Name, Email: f.Identity.Email},
		Transport: domain.TransportConfig{Strategy: f.Transport.Strategy, Config: f.Transport.Config},
		Revision:  f.Revision,
	}
	if err := account.Validate(); err != nil {
		return domain.Account{}, fmt.Errorf("account %s is invalid: %w", f.ID, err)
	}
	return account, nil
}

// Save validates the account and enforces local alias uniqueness.
func (s *AccountStore) Save(ctx context.Context, account domain.Account) error {
	if err := account.Validate(); err != nil {
		return err
	}
	existing, err := s.List(ctx)
	if err != nil {
		return err
	}
	for _, other := range existing {
		if other.Alias == account.Alias && other.ID != account.ID {
			return fmt.Errorf("%w: alias %q", domain.ErrAccountExists, account.Alias)
		}
	}
	data, err := json.MarshalIndent(toAccountFile(account), "", "  ")
	if err != nil {
		return err
	}
	return WriteFileAtomic(s.path(account.ID), append(data, '\n'), 0o600)
}

// Get loads one account by ID.
func (s *AccountStore) Get(ctx context.Context, id domain.AccountID) (domain.Account, error) {
	data, err := os.ReadFile(s.path(id))
	if err != nil {
		if os.IsNotExist(err) {
			return domain.Account{}, fmt.Errorf("%w: %s", domain.ErrAccountNotFound, id)
		}
		return domain.Account{}, err
	}
	var file accountFile
	if err := json.Unmarshal(data, &file); err != nil {
		return domain.Account{}, fmt.Errorf("account %s: %w", id, err)
	}
	return file.toDomain()
}

// List returns all accounts sorted by AccountID.
func (s *AccountStore) List(ctx context.Context) ([]domain.Account, error) {
	entries, err := os.ReadDir(s.accountsDir())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var accounts []domain.Account
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasPrefix(name, "acc_") || !strings.HasSuffix(name, ".json") {
			continue
		}
		id := domain.AccountID(strings.TrimSuffix(name, ".json"))
		account, err := s.Get(ctx, id)
		if err != nil {
			return nil, err
		}
		accounts = append(accounts, account)
	}
	sort.Slice(accounts, func(i, j int) bool { return accounts[i].ID < accounts[j].ID })
	return accounts, nil
}

// Delete removes an account file; deleting a missing account is an error.
func (s *AccountStore) Delete(ctx context.Context, id domain.AccountID) error {
	if err := os.Remove(s.path(id)); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("%w: %s", domain.ErrAccountNotFound, id)
		}
		return err
	}
	return nil
}

func (s *AccountStore) path(id domain.AccountID) string {
	return filepath.Join(s.accountsDir(), string(id)+".json")
}
