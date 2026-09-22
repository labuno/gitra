package provider_test

import (
	"testing"

	"github.com/zhanhd/gitra/internal/adapters/provider/gitea"
	"github.com/zhanhd/gitra/internal/adapters/provider/github"
	"github.com/zhanhd/gitra/internal/adapters/provider/gitlab"
)

func TestSSHVerificationFixtures(t *testing.T) {
	tests := []struct {
		name   string
		parse  func(string, string) (string, bool)
		stdout string
		stderr string
		want   string
		wantOK bool
	}{
		{
			name: "github success", parse: github.ParseSSHVerification,
			stderr: "Hi lunafoundry! You've successfully authenticated, but GitHub does not provide shell access.",
			want:   "lunafoundry", wantOK: true,
		},
		{
			name: "gitlab success", parse: gitlab.ParseSSHVerification,
			stderr: "Welcome to GitLab, @luna-work!",
			want:   "luna-work", wantOK: true,
		},
		{
			name: "gitea success", parse: gitea.ParseSSHVerification,
			stderr: "Hi there, luna! You've successfully authenticated with the key named mac, but Gitea does not provide shell access.",
			want:   "luna", wantOK: true,
		},
		{
			name: "github permission denied", parse: github.ParseSSHVerification,
			stderr: "git@github.com: Permission denied (publickey).",
			wantOK: false,
		},
		{
			name: "gitlab permission denied", parse: gitlab.ParseSSHVerification,
			stderr: "git@gitlab.com: Permission denied (publickey).",
			wantOK: false,
		},
		{
			name: "empty output", parse: github.ParseSSHVerification, wantOK: false,
		},
		{
			name: "unexpected text", parse: gitea.ParseSSHVerification,
			stdout: "some other greeting", wantOK: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := tt.parse(tt.stdout, tt.stderr)
			if ok != tt.wantOK || got != tt.want {
				t.Fatalf("parse = (%q, %v), want (%q, %v)", got, ok, tt.want, tt.wantOK)
			}
		})
	}
}
