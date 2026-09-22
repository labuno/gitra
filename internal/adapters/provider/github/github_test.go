package github

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/zhanhd/gitra/internal/domain"
)

func server(t *testing.T, user, emails string, userStatus, emailStatus int) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/user", func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer tok" {
			t.Errorf("authorization = %q", got)
		}
		w.WriteHeader(userStatus)
		_, _ = w.Write([]byte(user))
	})
	mux.HandleFunc("/user/emails", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(emailStatus)
		_, _ = w.Write([]byte(emails))
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func TestProfilePrimaryEmail(t *testing.T) {
	srv := server(t,
		`{"login":"lunafoundry","name":"Luna"}`,
		`[{"email":"old@example.com","primary":false,"verified":true},{"email":"luna@example.com","primary":true,"verified":true}]`,
		http.StatusOK, http.StatusOK)
	client := New(srv.URL, srv.Client())
	profile, err := client.Profile(context.Background(), "tok", "github.com")
	if err != nil {
		t.Fatal(err)
	}
	if profile.Username != "lunafoundry" || profile.Name != "Luna" || profile.Email != "luna@example.com" || profile.EmailFallback {
		t.Fatalf("profile = %+v", profile)
	}
}

func TestProfileFallsBackToNoreplyEmail(t *testing.T) {
	srv := server(t, `{"login":"lunafoundry","name":""}`, `[]`, http.StatusOK, http.StatusOK)
	client := New(srv.URL, srv.Client())
	profile, err := client.Profile(context.Background(), "tok", "github.com")
	if err != nil {
		t.Fatal(err)
	}
	if !profile.EmailFallback || profile.Email != "lunafoundry@users.noreply.github.com" || profile.Name != "lunafoundry" {
		t.Fatalf("profile = %+v", profile)
	}
}

func TestProfileClassifiesErrors(t *testing.T) {
	srv := server(t, `{"message":"Bad credentials"}`, `[]`, http.StatusUnauthorized, http.StatusUnauthorized)
	client := New(srv.URL, srv.Client())
	if _, err := client.Profile(context.Background(), "tok", "github.com"); !errors.Is(err, domain.ErrAuthInvalid) {
		t.Fatalf("401 error = %v, want ErrAuthInvalid", err)
	}

	srv500 := server(t, `{}`, `[]`, http.StatusInternalServerError, http.StatusInternalServerError)
	if _, err := New(srv500.URL, srv500.Client()).Profile(context.Background(), "tok", "github.com"); errors.Is(err, domain.ErrAuthInvalid) {
		t.Fatalf("500 must not be classified as auth invalid: %v", err)
	}
}
