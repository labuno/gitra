package app

import (
	"context"
	"errors"
	"testing"

	"github.com/zhanhd/gitra/internal/domain"
	"github.com/zhanhd/gitra/internal/ports"
)

type stubPublisher struct {
	status    ports.PublishStatus
	pushed    bool
	afterPush bool
}

func (s *stubPublisher) PublishStatus(context.Context, domain.Repository) (ports.PublishStatus, error) {
	if s.pushed {
		return ports.PublishStatus{Branch: s.status.Branch, HasLocalCommits: true, RemoteHasBranch: s.afterPush}, nil
	}
	return s.status, nil
}

func (s *stubPublisher) Push(context.Context, domain.Repository, string, string, bool) error {
	s.pushed = true
	return nil
}

func publishEnv(t *testing.T) (*PublishService, *stubPublisher, *httpsEnv) {
	t.Helper()
	env := newHTTPSEnv(t, "https://github.com/lunafoundry/luna-site.git")
	publisher := &stubPublisher{status: ports.PublishStatus{Branch: "main", HasLocalCommits: true}}
	env.deps.Publisher = publisher
	return NewPublishService(env.deps), publisher, env
}

func TestPublishStatusExplainsEveryState(t *testing.T) {
	ctx := context.Background()

	// Not bound yet.
	service, _, env := publishEnv(t)
	view, err := service.Status(ctx, "/repo")
	if err != nil {
		t.Fatal(err)
	}
	if view.CanPublish || view.Reason == "" {
		t.Fatalf("unbound view = %+v", view)
	}

	// Bound, but nothing committed yet.
	if _, err := env.service.Bind(ctx, BindRequest{AccountID: env.account.ID, Path: "/repo"}); err != nil {
		t.Fatal(err)
	}
	noCommits := &stubPublisher{status: ports.PublishStatus{Branch: "main", HasLocalCommits: false}}
	env.deps.Publisher = noCommits
	view, err = NewPublishService(env.deps).Status(ctx, "/repo")
	if err != nil {
		t.Fatal(err)
	}
	if view.CanPublish || view.Reason != "还没有任何提交：先提交一次代码再上传" {
		t.Fatalf("no-commit view = %+v", view)
	}

	// Remote already has the branch.
	existing := &stubPublisher{status: ports.PublishStatus{Branch: "main", HasLocalCommits: true, RemoteHasBranch: true}}
	env.deps.Publisher = existing
	view, _ = NewPublishService(env.deps).Status(ctx, "/repo")
	if view.CanPublish {
		t.Fatalf("existing remote branch must not be publishable: %+v", view)
	}

	// Ready to publish.
	env.deps.Publisher = &stubPublisher{status: ports.PublishStatus{Branch: "main", HasLocalCommits: true}}
	view, _ = NewPublishService(env.deps).Status(ctx, "/repo")
	if !view.CanPublish || view.Branch != "main" {
		t.Fatalf("ready view = %+v", view)
	}
}

func TestFirstPublishPushesAndVerifies(t *testing.T) {
	ctx := context.Background()
	service, publisher, env := publishEnv(t)
	if _, err := env.service.Bind(ctx, BindRequest{AccountID: env.account.ID, Path: "/repo"}); err != nil {
		t.Fatal(err)
	}
	service = NewPublishService(env.deps)
	publisher.afterPush = true

	result, err := service.FirstPublish(ctx, "/repo")
	if err != nil {
		t.Fatalf("FirstPublish() error = %v", err)
	}
	if !publisher.pushed || result.Branch != "main" || result.Remote != "origin" {
		t.Fatalf("result = %+v, pushed=%v", result, publisher.pushed)
	}

	// A remote that did not actually receive the branch must be reported.
	env.deps.Publisher = &stubPublisher{status: ports.PublishStatus{Branch: "main", HasLocalCommits: true}}
	service = NewPublishService(env.deps)
	if _, err := service.FirstPublish(ctx, "/repo"); !errors.Is(err, domain.ErrStateDrift) {
		t.Fatalf("unverified push error = %v, want ErrStateDrift", err)
	}
}

func TestFirstPublishRefusesUnboundRepository(t *testing.T) {
	service, _, _ := publishEnv(t)
	if _, err := service.FirstPublish(context.Background(), "/repo"); !errors.Is(err, domain.ErrBindingNotFound) {
		t.Fatalf("error = %v, want ErrBindingNotFound", err)
	}
}
