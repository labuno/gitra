package repolocal

import (
	"context"
	"testing"

	"github.com/zhanhd/gitra/internal/domain"
	"github.com/zhanhd/gitra/internal/ports"
	"github.com/zhanhd/gitra/internal/strategies/routing"
)

// fakeGit is an in-memory ports.Git for strategy tests.
type fakeGit struct {
	values map[string]string
	writes []string
}

func newFakeGit() *fakeGit { return &fakeGit{values: map[string]string{}} }

func (f *fakeGit) DiscoverRepository(context.Context, string) (domain.Repository, error) {
	return domain.Repository{RootPath: "/repo", GitDir: "/repo/.git"}, nil
}
func (f *fakeGit) GetLocalConfig(_ context.Context, _ domain.Repository, key string) (string, bool, error) {
	value, ok := f.values[key]
	return value, ok, nil
}
func (f *fakeGit) GetGlobalConfig(context.Context, string) (string, bool, error) {
	return "", false, nil
}
func (f *fakeGit) SetLocalConfig(_ context.Context, _ domain.Repository, key, value string) error {
	f.values[key] = value
	f.writes = append(f.writes, "set "+key)
	return nil
}
func (f *fakeGit) UnsetLocalConfig(_ context.Context, _ domain.Repository, key string) error {
	delete(f.values, key)
	f.writes = append(f.writes, "unset "+key)
	return nil
}
func (f *fakeGit) Remotes(context.Context, domain.Repository) ([]domain.Remote, error) {
	return nil, nil
}

func repo() domain.Repository {
	return domain.Repository{RootPath: "/repo", GitDir: "/repo/.git"}
}

func TestApplyWritesEntriesInOrder(t *testing.T) {
	git := newFakeGit()
	entries := []ports.GitConfigEntry{
		{Key: "user.name", Value: "Luna"},
		{Key: "user.email", Value: "luna@example.com"},
	}
	if err := New().Apply(context.Background(), git, repo(), entries); err != nil {
		t.Fatal(err)
	}
	if git.values["user.email"] != "luna@example.com" {
		t.Fatalf("values = %+v", git.values)
	}
	got := git.writes
	want := []string{"set user.name", "set user.email"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("write order = %v, want %v", got, want)
		}
	}
}

func TestInspectDetectsDrift(t *testing.T) {
	git := newFakeGit()
	entries := []ports.GitConfigEntry{{Key: "user.email", Value: "luna@example.com"}}
	status, err := New().Inspect(context.Background(), git, repo(), entries)
	if err != nil {
		t.Fatal(err)
	}
	if len(status.Drift) != 1 || status.Drift[0] != "user.email" {
		t.Fatalf("drift = %v", status.Drift)
	}
	if err := New().Apply(context.Background(), git, repo(), entries); err != nil {
		t.Fatal(err)
	}
	status, err = New().Inspect(context.Background(), git, repo(), entries)
	if err != nil || len(status.Drift) != 0 {
		t.Fatalf("status = %+v, err = %v", status, err)
	}
}

func TestRestoreSetsAndUnsets(t *testing.T) {
	git := newFakeGit()
	git.values["user.email"] = "luna@example.com"
	previous := []routing.PreviousEntry{
		{Key: "user.name", Exists: true, Value: "Old Name"},
		{Key: "user.email", Exists: false},
	}
	if err := New().Restore(context.Background(), git, repo(), previous); err != nil {
		t.Fatal(err)
	}
	if git.values["user.name"] != "Old Name" {
		t.Fatalf("user.name = %q", git.values["user.name"])
	}
	if _, ok := git.values["user.email"]; ok {
		t.Fatal("user.email must be unset")
	}
}
