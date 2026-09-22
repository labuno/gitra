package app

import (
	"context"
	"errors"
	"testing"

	"github.com/zhanhd/gitra/internal/domain"
)

func TestEnsureOriginRemoteAddsMissingRemote(t *testing.T) {
	env := newHTTPSEnv(t, "") // repository without any remote
	ctx := context.Background()

	if err := env.service.EnsureOriginRemote(ctx, "/repo", "https://github.com/lunafoundry/luna-site.git"); err != nil {
		t.Fatalf("EnsureOriginRemote() error = %v", err)
	}
	if env.git.values["remote.origin.url"] != "https://github.com/lunafoundry/luna-site.git" {
		t.Fatalf("origin url = %q", env.git.values["remote.origin.url"])
	}
	if env.git.values["remote.origin.fetch"] == "" {
		t.Fatal("fetch refspec must be set together with the url")
	}
}

func TestEnsureOriginRemoteNeverRewrites(t *testing.T) {
	env := newHTTPSEnv(t, "https://github.com/lunafoundry/luna-site.git")
	ctx := context.Background()

	if err := env.service.EnsureOriginRemote(ctx, "/repo", "https://github.com/someone-else/other.git"); err != nil {
		t.Fatalf("EnsureOriginRemote() error = %v", err)
	}
	if _, present := env.git.values["remote.origin.url"]; present {
		t.Fatalf("existing origin must not be rewritten: %+v", env.git.values)
	}
}

func TestEnsureOriginRemoteRejectsGarbage(t *testing.T) {
	env := newHTTPSEnv(t, "")
	if err := env.service.EnsureOriginRemote(context.Background(), "/repo", "not a url"); !errors.Is(err, domain.ErrUnsupportedRemote) {
		t.Fatalf("error = %v, want ErrUnsupportedRemote", err)
	}
}

func TestStatusReportsOrigin(t *testing.T) {
	withOrigin := newHTTPSEnv(t, "https://github.com/lunafoundry/luna-site.git")
	status, err := withOrigin.service.Status(context.Background(), "/repo")
	if err != nil {
		t.Fatal(err)
	}
	if !status.HasOrigin || status.OriginURL == "" {
		t.Fatalf("status must report the origin remote: %+v", status)
	}

	withoutOrigin := newHTTPSEnv(t, "")
	status, err = withoutOrigin.service.Status(context.Background(), "/repo")
	if err != nil {
		t.Fatal(err)
	}
	if status.HasOrigin || status.OriginURL != "" {
		t.Fatalf("status = %+v, want no origin", status)
	}
}
