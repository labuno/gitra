package routing

import (
	"context"
	"errors"
	"testing"

	"github.com/zhanhd/gitra/internal/domain"
	"github.com/zhanhd/gitra/internal/ports"
)

type fakeStrategy struct{ id string }

func (f fakeStrategy) ID() string { return f.id }

func (f fakeStrategy) Apply(context.Context, ports.Git, domain.Repository, []ports.GitConfigEntry) error {
	return nil
}

func (f fakeStrategy) Inspect(context.Context, ports.Git, domain.Repository, []ports.GitConfigEntry) (Status, error) {
	return Status{}, nil
}

func (f fakeStrategy) Restore(context.Context, ports.Git, domain.Repository, []PreviousEntry) error {
	return nil
}

func TestRegistryRegisterAndGet(t *testing.T) {
	registry := NewRegistry()
	if err := registry.Register(fakeStrategy{id: "repo-local"}); err != nil {
		t.Fatal(err)
	}
	if err := registry.Register(fakeStrategy{id: "repo-local"}); err == nil {
		t.Fatal("duplicate registration must fail")
	}
	if _, err := registry.Get("repo-local"); err != nil {
		t.Fatal(err)
	}
	if _, err := registry.Get("missing"); !errors.Is(err, ErrUnknownStrategy) {
		t.Fatalf("unknown strategy error = %v", err)
	}
}
