package domain

import (
	"errors"
	"testing"
)

func TestRepositoryValidate(t *testing.T) {
	tests := []struct {
		name      string
		mutate    func(*Repository)
		wantErr   bool
		wantField string
	}{
		{"valid", func(r *Repository) {}, false, ""},
		{"missing root", func(r *Repository) { r.RootPath = "" }, true, "root_path"},
		{"missing git dir", func(r *Repository) { r.GitDir = "" }, true, "git_dir"},
		{"duplicate remote name", func(r *Repository) {
			r.Remotes = []Remote{{Name: "origin", URL: "git@github.com:a/b.git"}, {Name: "origin", URL: "git@github.com:c/d.git"}}
		}, true, "remotes"},
		{"empty remote url", func(r *Repository) { r.Remotes = []Remote{{Name: "origin", URL: ""}} }, true, "remotes"},
		{"no remotes allowed", func(r *Repository) { r.Remotes = nil }, false, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := Repository{
				RootPath: "/Users/luna/Projects/luna-site",
				GitDir:   "/Users/luna/Projects/luna-site/.git",
				Remotes:  []Remote{{Name: "origin", URL: "git@github.com:lunafoundry/luna-site.git"}},
			}
			tt.mutate(&repo)
			err := repo.Validate()
			assertValidation(t, err, tt.wantErr, tt.wantField)
		})
	}
}

func TestRepositoryBindingValidate(t *testing.T) {
	tests := []struct {
		name      string
		mutate    func(*RepositoryBinding)
		wantErr   bool
		wantField string
	}{
		{"valid", func(b *RepositoryBinding) {}, false, ""},
		{"missing id", func(b *RepositoryBinding) { b.ID = "" }, true, "id"},
		{"missing account", func(b *RepositoryBinding) { b.AccountID = "" }, true, "account_id"},
		{"relative path", func(b *RepositoryBinding) { b.Repository.Path = "luna-site" }, true, "repository.path"},
		{"unknown strategy", func(b *RepositoryBinding) { b.Strategy = "directory" }, true, "strategy"},
		{"negative revision", func(b *RepositoryBinding) { b.Revision = -1 }, true, "revision"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := RepositoryBinding{
				ID:         "bnd_01HXYZ",
				AccountID:  "acc_01HXYZ",
				Repository: RepositoryRef{Path: "/Users/luna/Projects/luna-site"},
				Strategy:   StrategyRepoLocal,
				Revision:   1,
			}
			tt.mutate(&b)
			assertValidation(t, b.Validate(), tt.wantErr, tt.wantField)
		})
	}
}

func TestBindingHealthValues(t *testing.T) {
	for _, h := range []BindingHealth{HealthOK, HealthDrift, HealthMissing, HealthBroken} {
		if !h.Valid() {
			t.Fatalf("health %q should be valid", h)
		}
	}
	if BindingHealth("online").Valid() {
		t.Fatal("health \"online\" must be invalid")
	}
}

func assertValidation(t *testing.T, err error, wantErr bool, wantField string) {
	t.Helper()
	if !wantErr {
		if err != nil {
			t.Fatalf("Validate() = %v, want nil", err)
		}
		return
	}
	if err == nil {
		t.Fatal("Validate() = nil, want error")
	}
	if !errors.Is(err, ErrInvalid) {
		t.Fatalf("error %v does not wrap ErrInvalid", err)
	}
	var verr *ValidationError
	if !errors.As(err, &verr) {
		t.Fatalf("error %v is not *ValidationError", err)
	}
	if verr.Field != wantField {
		t.Fatalf("ValidationError.Field = %q, want %q", verr.Field, wantField)
	}
}
