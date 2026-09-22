package router

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/zhanhd/gitra/internal/domain"
	"github.com/zhanhd/gitra/internal/ports"
)

type noopRunner struct{}

func (noopRunner) Run(context.Context, string, ...string) (ports.ProcessResult, error) {
	return ports.ProcessResult{ExitCode: 1}, nil
}
func (r noopRunner) RunWithInput(ctx context.Context, name, _ string, args ...string) (ports.ProcessResult, error) {
	return r.Run(ctx, name, args...)
}

func TestNewUsesBoundedHTTPClient(t *testing.T) {
	adapter := New(noopRunner{}, strings.NewReader(""))
	client, ok := adapter.http.(*http.Client)
	if !ok {
		t.Fatalf("adapter.http = %T, want *http.Client so it can carry a timeout", adapter.http)
	}
	if client.Timeout <= 0 {
		t.Fatalf("HTTP client timeout = %v, want a positive bound", client.Timeout)
	}
	if client.Timeout > time.Minute {
		t.Fatalf("HTTP client timeout = %v, too long for an interactive tool", client.Timeout)
	}
}

func TestProfileTimesOutInsteadOfHanging(t *testing.T) {
	blocked := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		<-blocked
	}))
	t.Cleanup(func() { close(blocked); server.Close() })

	adapter := New(noopRunner{}, strings.NewReader(""))
	adapter.BaseURL = func(domain.ProviderType, string) string { return server.URL }
	adapter.http = &http.Client{Timeout: 300 * time.Millisecond}

	start := time.Now()
	if _, err := adapter.Profile(context.Background(), domain.ProviderGitHub, "github.com", "tok"); err == nil {
		t.Fatal("a stalled provider must produce an error")
	}
	if elapsed := time.Since(start); elapsed > 5*time.Second {
		t.Fatalf("profile call took %s, must be bounded", elapsed)
	}
}
