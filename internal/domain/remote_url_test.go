package domain

import (
	"errors"
	"testing"
)

func TestParseRemoteURL(t *testing.T) {
	tests := []struct {
		name         string
		url          string
		host         string
		port         int
		explicitPort bool
		wantErr      error
	}{
		{"scp-like", "git@github.com:lunafoundry/luna-site.git", "github.com", 22, false, nil},
		{"scp-like no suffix", "git@github.com:openai/example", "github.com", 22, false, nil},
		{"ssh url default port", "ssh://git@github.com/owner/repo.git", "github.com", 22, false, nil},
		{"ssh url custom port", "ssh://git@git.example.com:2222/luna/repo.git", "git.example.com", 2222, true, nil},
		{"https rejected", "https://github.com/lunafoundry/luna-site.git", "", 0, false, ErrUnsupportedRemote},
		{"empty rejected", "", "", 0, false, ErrUnsupportedRemote},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseRemoteURL(tt.url)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("ParseRemoteURL(%q) error = %v, want %v", tt.url, err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseRemoteURL(%q) error = %v", tt.url, err)
			}
			if got.Host != tt.host || got.Port != tt.port || got.ExplicitPort != tt.explicitPort {
				t.Fatalf("ParseRemoteURL(%q) = %+v", tt.url, got)
			}
		})
	}
}
