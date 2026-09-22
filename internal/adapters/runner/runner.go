// Package runner implements ports.CommandRunner on top of os/exec.
//
// Process exit codes are returned as data (ProcessResult.ExitCode); error is
// reserved for execution failures such as a missing binary or cancellation.
package runner

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"

	"github.com/zhanhd/gitra/internal/ports"
)

// Runner executes external processes.
type Runner struct{}

// New returns a Runner.
func New() *Runner { return &Runner{} }

// Run implements ports.CommandRunner.
func (r Runner) Run(ctx context.Context, name string, args ...string) (ports.ProcessResult, error) {
	return r.RunWithInput(ctx, name, "", args...)
}

// RunWithInput implements ports.CommandRunner with stdin support.
func (Runner) RunWithInput(ctx context.Context, name string, input string, args ...string) (ports.ProcessResult, error) {
	var stdout, stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, name, args...)
	if input != "" {
		cmd.Stdin = strings.NewReader(input)
	}
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	result := ports.ProcessResult{Stdout: stdout.String(), Stderr: stderr.String()}
	if err == nil {
		return result, nil
	}

	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return result, fmt.Errorf("run %s: %w", name, ctxErr)
		}
		result.ExitCode = exitErr.ExitCode()
		return result, nil
	}
	return result, fmt.Errorf("run %s: %w", name, err)
}
