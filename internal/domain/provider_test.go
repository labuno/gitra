package domain

import "testing"

func TestDefaultEndpoint(t *testing.T) {
	tests := []struct {
		provider ProviderType
		host     string
		user     string
		port     int
		ok       bool
	}{
		{ProviderGitHub, "github.com", "git", 22, true},
		{ProviderGitLab, "gitlab.com", "git", 22, true},
		{ProviderGitea, "gitea.com", "git", 22, true},
		{"bitbucket", "", "", 0, false},
	}
	for _, tt := range tests {
		got, ok := DefaultEndpoint(tt.provider)
		if ok != tt.ok {
			t.Fatalf("DefaultEndpoint(%q) ok = %v, want %v", tt.provider, ok, tt.ok)
		}
		if !ok {
			continue
		}
		if got.Host != tt.host || got.SSHUser != tt.user || got.SSHPort != tt.port {
			t.Fatalf("DefaultEndpoint(%q) = %+v, want host=%s user=%s port=%d", tt.provider, got, tt.host, tt.user, tt.port)
		}
	}
}

func TestSSHCommandNeedsPort(t *testing.T) {
	tests := []struct {
		name                  string
		port                  int
		remoteHasExplicitPort bool
		want                  bool
	}{
		{"default port no flag", 22, false, false},
		{"default port with url port", 22, true, false},
		{"custom port scp-like", 2222, false, true},
		{"custom port url carries port", 2222, true, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := ProviderEndpoint{Host: "git.example.com", SSHUser: "git", SSHPort: tt.port}
			if got := e.SSHCommandNeedsPort(tt.remoteHasExplicitPort); got != tt.want {
				t.Fatalf("SSHCommandNeedsPort(%v) = %v, want %v", tt.remoteHasExplicitPort, got, tt.want)
			}
		})
	}
}
