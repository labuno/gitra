package ports

import "context"

// ProcessResult is the raw outcome of one process execution (baseline §18).
// ExitCode is data: a non-zero code is not an execution error.
type ProcessResult struct {
	ExitCode int
	Stdout   string
	Stderr   string
}

// CommandRunner is the single abstraction used by the git, ssh and credential
// adapters.
type CommandRunner interface {
	Run(ctx context.Context, name string, args ...string) (ProcessResult, error)
	// RunWithInput feeds input on stdin. Secrets (tokens) must travel this
	// way, never through argv.
	RunWithInput(ctx context.Context, name string, input string, args ...string) (ProcessResult, error)
}
