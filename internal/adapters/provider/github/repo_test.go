package github

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/zhanhd/gitra/internal/adapters/provider"
	"github.com/zhanhd/gitra/internal/domain"
)

func TestLookupRepo(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/labuno/gitra", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer tok" {
			t.Errorf("authorization = %q", r.Header.Get("Authorization"))
		}
		_, _ = w.Write([]byte(`{"private":true,"clone_url":"https://github.com/labuno/gitra.git"}`))
	})
	mux.HandleFunc("/repos/labuno/missing", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"message":"Not Found"}`))
	})
	mux.HandleFunc("/repos/labuno/denied", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	})
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	client := New(server.URL, server.Client())

	status, err := client.LookupRepo(context.Background(), "tok", "labuno", "gitra")
	if err != nil || !status.Exists || !status.Private || status.CloneURL == "" {
		t.Fatalf("lookup = (%+v, %v)", status, err)
	}

	status, err = client.LookupRepo(context.Background(), "tok", "labuno", "missing")
	if err != nil {
		t.Fatalf("missing repository must not be an error: %v", err)
	}
	if status.Exists {
		t.Fatal("missing repository reported as existing")
	}

	if _, err := client.LookupRepo(context.Background(), "tok", "labuno", "denied"); !errors.Is(err, domain.ErrAuthInvalid) {
		t.Fatalf("401 error = %v, want ErrAuthInvalid", err)
	}
}

func TestCreateRepo(t *testing.T) {
	var received map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/user/repos" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
			t.Errorf("body decode: %v", err)
		}
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"private":true,"clone_url":"https://github.com/labuno/gitra.git"}`))
	}))
	t.Cleanup(server.Close)

	status, err := New(server.URL, server.Client()).CreateRepo(context.Background(), "tok", "labuno", "gitra", true)
	if err != nil {
		t.Fatalf("CreateRepo() error = %v", err)
	}
	if !status.Exists || status.CloneURL == "" {
		t.Fatalf("status = %+v", status)
	}
	if received["name"] != "gitra" || received["private"] != true {
		t.Fatalf("request body = %+v", received)
	}
}

var _ = provider.ErrNotFound
