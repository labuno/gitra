package app

import "github.com/zhanhd/gitra/internal/domain"

// healthInput captures the observed facts used by the A.4 truth table.
type healthInput struct {
	HasCentralBinding bool
	RepoExists        bool
	MetadataPresent   bool
	MetadataMatches   bool
	Drift             []string
}

// computeHealth maps observed facts to BindingHealth. bound is false only when
// neither a central binding nor repository metadata exists.
func computeHealth(in healthInput) (domain.BindingHealth, bool) {
	if !in.HasCentralBinding && !in.MetadataPresent {
		return "", false
	}
	if !in.RepoExists {
		return domain.HealthMissing, true
	}
	if !in.MetadataPresent || !in.MetadataMatches || !in.HasCentralBinding {
		return domain.HealthBroken, true
	}
	if len(in.Drift) > 0 {
		return domain.HealthDrift, true
	}
	return domain.HealthOK, true
}
