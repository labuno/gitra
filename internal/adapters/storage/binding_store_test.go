package storage

import (
	"context"
	"errors"
	"testing"

	"github.com/zhanhd/gitra/internal/domain"
)

func testBinding(id domain.BindingID, account domain.AccountID, path string) domain.RepositoryBinding {
	return domain.RepositoryBinding{
		ID:         id,
		AccountID:  account,
		Repository: domain.RepositoryRef{Path: path},
		Strategy:   domain.StrategyRepoLocal,
		Revision:   1,
	}
}

func TestBindingStoreCRUD(t *testing.T) {
	dir := t.TempDir()
	store := NewBindingStore(dir)
	ctx := context.Background()
	a := testBinding("bnd_001", "acc_001", "/Users/luna/Projects/luna-site")
	b := testBinding("bnd_002", "acc_002", "/Users/luna/Projects/company-api")

	if err := store.Save(ctx, a); err != nil {
		t.Fatal(err)
	}
	if err := store.Save(ctx, b); err != nil {
		t.Fatal(err)
	}
	got, found, err := store.FindByRepository(ctx, a.Repository.Path)
	if err != nil || !found || got.ID != a.ID {
		t.Fatalf("FindByRepository = (%+v, %v, %v)", got, found, err)
	}
	if _, found, err := store.FindByRepository(ctx, "/tmp/not-bound"); err != nil || found {
		t.Fatalf("unknown path = (found %v, err %v)", found, err)
	}
	list, err := store.ListByAccount(ctx, "acc_001")
	if err != nil || len(list) != 1 || list[0].ID != a.ID {
		t.Fatalf("ListByAccount = (%+v, %v)", list, err)
	}
	if err := store.Delete(ctx, a.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Get(ctx, a.ID); !errors.Is(err, domain.ErrBindingNotFound) {
		t.Fatalf("Get after delete = %v, want ErrBindingNotFound", err)
	}
	if _, _, err := store.FindByRepository(ctx, a.Repository.Path); err != nil {
		t.Fatal(err)
	}
}
