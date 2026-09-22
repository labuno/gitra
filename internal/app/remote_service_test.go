package app

import (
	"context"
	"errors"
	"testing"

	"github.com/zhanhd/gitra/internal/domain"
	"github.com/zhanhd/gitra/internal/ports"
)

type stubRepos struct {
	exists  bool
	created bool
	private bool
	owner   string
	name    string
}

func (s *stubRepos) LookupRepo(context.Context, domain.ProviderType, string, string, string, string) (ports.RemoteStatus, error) {
	return ports.RemoteStatus{Exists: s.exists, CloneURL: "https://github.com/lunafoundry/luna-site.git"}, nil
}

func (s *stubRepos) CreateRepo(_ context.Context, _ domain.ProviderType, _ string, _ string, owner, name string, private bool) (ports.RemoteStatus, error) {
	s.created, s.private, s.owner, s.name = true, private, owner, name
	return ports.RemoteStatus{Exists: true, CloneURL: "https://github.com/" + owner + "/" + name + ".git"}, nil
}

func TestRemoteCheckReportsExistence(t *testing.T) {
	env := newHTTPSEnv(t, "https://github.com/lunafoundry/luna-site.git")
	repos := &stubRepos{exists: true}
	service := NewRemoteService(env.deps, repos, env.service)
	ctx := context.Background()

	check, err := service.Check(ctx, env.account.ID, "/repo")
	if err != nil {
		t.Fatalf("Check() error = %v", err)
	}
	if !check.Exists || check.Owner != "lunafoundry" || check.Name != "luna-site" || check.Host != "github.com" {
		t.Fatalf("check = %+v", check)
	}

	repos.exists = false
	check, err = service.Check(ctx, env.account.ID, "/repo")
	if err != nil {
		t.Fatal(err)
	}
	if check.Exists {
		t.Fatal("missing repository must be reported as not existing")
	}
}

func TestRemoteCreateAndBindSetsOriginAndBinds(t *testing.T) {
	env := newHTTPSEnv(t, "")
	repos := &stubRepos{}
	service := NewRemoteService(env.deps, repos, env.service)
	ctx := context.Background()

	// Flow: the picker stores the address the user confirmed, then gitra asks
	// the provider to create the repository and binds the folder.
	if err := env.service.EnsureOriginRemote(ctx, "/repo", "https://github.com/lunafoundry/luna-site.git"); err != nil {
		t.Fatalf("EnsureOriginRemote() error = %v", err)
	}
	if _, err := service.CreateAndBind(ctx, env.account.ID, "/repo", true); err != nil {
		t.Fatalf("CreateAndBind() error = %v", err)
	}
	if !repos.created || !repos.private || repos.name != "luna-site" {
		t.Fatalf("create call = %+v", repos)
	}
	if env.git.values["remote.origin.url"] != "https://github.com/lunafoundry/luna-site.git" {
		t.Fatalf("origin = %q", env.git.values["remote.origin.url"])
	}
	if len(env.bindings.saved) != 1 {
		t.Fatalf("binding not saved: %+v", env.bindings.saved)
	}
}

func TestRemoteOperationsRefuseUnsupportedCases(t *testing.T) {
	ctx := context.Background()

	// A folder with no address at all cannot be verified online.
	emptyEnv := newHTTPSEnv(t, "")
	empty := NewRemoteService(emptyEnv.deps, &stubRepos{}, emptyEnv.service)
	if _, err := empty.Check(ctx, emptyEnv.account.ID, "/repo"); !errors.Is(err, domain.ErrUnsupportedRemote) {
		t.Fatalf("missing remote error = %v, want ErrUnsupportedRemote", err)
	}

	// Non-HTTPS remote: the API cannot verify or create it.
	sshEnv := newHTTPSEnv(t, "git@github.com:lunafoundry/luna-site.git")
	service := NewRemoteService(sshEnv.deps, &stubRepos{}, sshEnv.service)
	if _, err := service.Check(ctx, sshEnv.account.ID, "/repo"); !errors.Is(err, domain.ErrUnsupportedRemote) {
		t.Fatalf("ssh remote error = %v, want ErrUnsupportedRemote", err)
	}

	// Repository owned by somebody else: gitra only creates inside the account.
	foreignEnv := newHTTPSEnv(t, "https://github.com/someone-else/luna-site.git")
	foreign := NewRemoteService(foreignEnv.deps, &stubRepos{}, foreignEnv.service)
	if _, err := foreign.CreateAndBind(ctx, foreignEnv.account.ID, "/repo", true); !errors.Is(err, domain.ErrInvalid) {
		t.Fatalf("foreign owner error = %v, want ErrInvalid", err)
	}
}
