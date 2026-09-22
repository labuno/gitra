// Package sshcli implements the SSH port by invoking the system ssh client.
// It returns raw results; deciding what "authenticated" means belongs to the
// provider adapters (baseline §4.6).
package sshcli

import (
	"context"
	"strconv"

	"github.com/zhanhd/gitra/internal/ports"
)

var _ ports.SSH = (*Adapter)(nil)

// Adapter implements ports.SSH.
type Adapter struct {
	runner ports.CommandRunner
}

// New builds the adapter over a process runner.
func New(runner ports.CommandRunner) *Adapter { return &Adapter{runner: runner} }

// Test runs `ssh -T -i <key> -o IdentitiesOnly=yes [-p port] <user>@<host>`.
func (a *Adapter) Test(ctx context.Context, req ports.SSHTestRequest) (ports.SSHTestResult, error) {
	args := []string{"-T"}
	if req.PrivateKeyPath != "" {
		args = append(args, "-i", req.PrivateKeyPath, "-o", "IdentitiesOnly=yes")
	}
	if req.Port != 0 && req.Port != 22 {
		args = append(args, "-p", strconv.Itoa(req.Port))
	}
	args = append(args, req.User+"@"+req.Host)

	result, err := a.runner.Run(ctx, "ssh", args...)
	if err != nil {
		return ports.SSHTestResult{}, err
	}
	return ports.SSHTestResult{ExitCode: result.ExitCode, Stdout: result.Stdout, Stderr: result.Stderr}, nil
}
