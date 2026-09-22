package e2e

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

// TestVerifyFlowAfterLoginAndAfterRevoke covers the M3 acceptance path: an
// account created through login verifies successfully, and fails with an
// authentication class once the provider rejects the stored credential.
func TestVerifyFlowAfterLoginAndAfterRevoke(t *testing.T) {
	var unauthorized atomic.Bool
	mux := http.NewServeMux()
	mux.HandleFunc("/user", func(w http.ResponseWriter, _ *http.Request) {
		if unauthorized.Load() {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"message":"Bad credentials"}`))
			return
		}
		_, _ = w.Write([]byte(`{"login":"lunafoundry","name":"Luna"}`))
	})
	mux.HandleFunc("/user/emails", func(w http.ResponseWriter, _ *http.Request) {
		if unauthorized.Load() {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`[]`))
			return
		}
		_, _ = w.Write([]byte(`[{"email":"luna@example.com","primary":true,"verified":true}]`))
	})
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	env := newLoginE2E(t, server, "gho_verify_token\n")
	if code, _, stderr := env.run(t, "login", "--provider", "github", "--token-stdin"); code != 0 {
		t.Fatalf("login exit=%d stderr=%s", code, stderr)
	}

	code, stdout, stderr := env.run(t, "account", "test", "lunafoundry", "--json")
	if code != 0 {
		t.Fatalf("verify exit=%d stderr=%s", code, stderr)
	}
	var payload struct {
		SchemaVersion int    `json:"schema_version"`
		Status        string `json:"status"`
		Success       bool   `json:"success"`
	}
	if err := json.Unmarshal([]byte(stdout), &payload); err != nil {
		t.Fatalf("verify JSON invalid: %v\n%s", err, stdout)
	}
	if payload.SchemaVersion != 1 || !payload.Success || payload.Status != "ok" {
		t.Fatalf("payload = %+v", payload)
	}

	unauthorized.Store(true)
	code, stdout, stderr = env.run(t, "account", "test", "lunafoundry", "--json")
	if code != 5 {
		t.Fatalf("revoked verify exit=%d stderr=%s, want 5", code, stderr)
	}
	if err := json.Unmarshal([]byte(stdout), &payload); err != nil {
		t.Fatal(err)
	}
	if payload.Status != "authentication_failed" {
		t.Fatalf("payload after revoke = %+v", payload)
	}
	if strings.Contains(stdout+stderr, "gho_verify_token") {
		t.Fatal("token leaked into verification output")
	}
}
