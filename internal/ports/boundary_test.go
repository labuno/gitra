package ports

import (
	"os/exec"
	"strings"
	"testing"
)

// TestLayerBoundaries enforces the dependency direction from the baseline §22:
// domain and ports must never import adapters or presentation packages.
func TestLayerBoundaries(t *testing.T) {
	cmd := exec.Command("go", "list", "-f", "{{.ImportPath}} {{join .Imports \" \"}}", "./...")
	cmd.Dir = moduleRoot(t)
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("go list failed: %v", err)
	}
	const module = "github.com/zhanhd/gitra/"
	forbidden := []string{module + "internal/adapters", module + "internal/presentation", module + "internal/bootstrap"}
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		pkg := fields[0]
		if !strings.HasPrefix(pkg, module+"internal/domain") && !strings.HasPrefix(pkg, module+"internal/ports") {
			continue
		}
		for _, dep := range fields[1:] {
			for _, banned := range forbidden {
				if strings.HasPrefix(dep, banned) {
					t.Fatalf("%s must not import %s (layer boundary)", pkg, dep)
				}
			}
		}
	}
}

func moduleRoot(t *testing.T) string {
	t.Helper()
	cmd := exec.Command("go", "env", "GOMOD")
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("go env GOMOD failed: %v", err)
	}
	gomod := strings.TrimSpace(string(out))
	if gomod == "" || gomod == "/dev/null" {
		t.Fatal("no go.mod found for module root")
	}
	return strings.TrimSuffix(gomod, "/go.mod")
}
