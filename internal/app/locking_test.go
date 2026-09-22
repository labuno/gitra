package app

import (
	"context"
	"testing"
)

// TestWritePathsTakeTheProcessLock documents that every write transaction runs
// inside the inter-process write lock (baseline §31).
func TestWritePathsTakeTheProcessLock(t *testing.T) {
	env := newHTTPSEnv(t, "https://github.com/lunafoundry/luna-site.git")
	locker := &countingLocker{}
	env.deps.Locker = locker
	service := NewBindingService(env.deps)
	accounts := NewAccountService(env.deps, service)
	reconciler := NewReconciler(env.deps)
	ctx := context.Background()

	binding, err := service.Bind(ctx, BindRequest{AccountID: env.account.ID, Path: "/repo"})
	if err != nil {
		t.Fatal(err)
	}
	afterBind := locker.acquired
	if afterBind == 0 {
		t.Fatal("Bind must run inside the write lock")
	}

	if _, err := reconciler.ReconcileBinding(ctx, binding.ID); err != nil {
		t.Fatal(err)
	}
	if locker.acquired <= afterBind {
		t.Fatal("ReconcileBinding must run inside the write lock")
	}
	afterReconcile := locker.acquired

	if err := accounts.Delete(ctx, DeleteAccountRequest{ID: env.account.ID, UnbindAll: true}); err != nil {
		t.Fatal(err)
	}
	if locker.acquired <= afterReconcile {
		t.Fatal("account deletion must run inside the write lock")
	}
}
