package auth

import (
	"context"
	"errors"
	"testing"

	"github.com/zhanhd/gitra/internal/domain"
	"github.com/zhanhd/gitra/internal/ports"
)

type fakeStrategy struct{ id string }

func (f fakeStrategy) ID() string { return f.id }

func (f fakeStrategy) Validate(context.Context, domain.Account) error { return nil }

func (f fakeStrategy) BuildGitConfig(BuildRequest) ([]ports.GitConfigEntry, error) { return nil, nil }

func TestRegistryRegisterAndGet(t *testing.T) {
	registry := NewRegistry()
	if err := registry.Register(fakeStrategy{id: "ssh-key"}); err != nil {
		t.Fatal(err)
	}
	if err := registry.Register(fakeStrategy{id: "ssh-key"}); err == nil {
		t.Fatal("duplicate registration must fail")
	}
	got, err := registry.Get("ssh-key")
	if err != nil || got.ID() != "ssh-key" {
		t.Fatalf("Get() = (%v, %v)", got, err)
	}
	if _, err := registry.Get("missing"); !errors.Is(err, ErrUnknownStrategy) {
		t.Fatalf("unknown strategy error = %v", err)
	}
}
