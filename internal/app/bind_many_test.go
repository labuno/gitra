package app

import (
	"context"
	"errors"
	"testing"

	"github.com/zhanhd/gitra/internal/domain"
)

func TestBindManyHandlesMixedFolders(t *testing.T) {
	ctx := context.Background()
	env := newHTTPSEnv(t, "https://github.com/lunafoundry/luna-site.git")
	service := env.service

	// One bindable folder, the same folder again, and one path that is not a
	// repository. (The missing-remote case is covered below.)
	env.git.missing["/missing"] = true
	results, err := service.BindMany(ctx, env.account.ID, []string{"/repo", "/repo", "/missing"})
	if err != nil {
		t.Fatalf("BindMany() error = %v", err)
	}
	if len(results) != 3 {
		t.Fatalf("results = %+v", results)
	}
	if results[0].Status != "bound" {
		t.Fatalf("first result = %+v, want bound", results[0])
	}
	byStatus := map[string]int{}
	for _, result := range results {
		byStatus[result.Status]++
	}
	// /repo binds once, the second attempt reports "already", /missing is skipped.
	if byStatus["bound"] != 1 || byStatus["already"] != 1 || byStatus["skipped"] != 1 {
		t.Fatalf("status counts = %v (results=%+v)", byStatus, results)
	}
	if len(env.bindings.saved) != 1 {
		t.Fatalf("bindings = %+v", env.bindings.saved)
	}
}

func TestBindManyReportsMissingRemote(t *testing.T) {
	ctx := context.Background()
	env := newHTTPSEnv(t, "") // repository without any remote
	env.git.remote = ""
	results, err := env.service.BindMany(ctx, env.account.ID, []string{"/repo"})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0].Status != "skipped" {
		t.Fatalf("results = %+v", results)
	}
	if !errors.Is(results[0].Err, domain.ErrUnsupportedRemote) {
		t.Fatalf("reason = %v, want ErrUnsupportedRemote", results[0].Err)
	}
}
