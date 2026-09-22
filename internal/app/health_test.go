package app

import (
	"testing"

	"github.com/zhanhd/gitra/internal/domain"
)

func TestComputeHealthTruthTable(t *testing.T) {
	tests := []struct {
		name  string
		in    healthInput
		want  domain.BindingHealth
		bound bool
	}{
		{"unbound without metadata", healthInput{}, "", false},
		{"ok", healthInput{HasCentralBinding: true, RepoExists: true, MetadataPresent: true, MetadataMatches: true}, domain.HealthOK, true},
		{"drift", healthInput{HasCentralBinding: true, RepoExists: true, MetadataPresent: true, MetadataMatches: true, Drift: []string{"user.email"}}, domain.HealthDrift, true},
		{"missing", healthInput{HasCentralBinding: true, RepoExists: false, MetadataPresent: true, MetadataMatches: true}, domain.HealthMissing, true},
		{"broken without metadata", healthInput{HasCentralBinding: true, RepoExists: true, MetadataPresent: false}, domain.HealthBroken, true},
		{"broken on metadata mismatch", healthInput{HasCentralBinding: true, RepoExists: true, MetadataPresent: true, MetadataMatches: false}, domain.HealthBroken, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, bound := computeHealth(tt.in)
			if got != tt.want || bound != tt.bound {
				t.Fatalf("computeHealth(%+v) = (%q, %v), want (%q, %v)", tt.in, got, bound, tt.want, tt.bound)
			}
		})
	}
}
