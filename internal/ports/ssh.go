package ports

import "context"

// SSHTestRequest describes one explicit connectivity/authentication probe.
type SSHTestRequest struct {
	Host           string
	User           string
	Port           int
	PrivateKeyPath string
}

// SSHTestResult is the raw ssh outcome. Provider adapters decide what
// "authenticated" means (baseline §4.6: exit code alone is not enough).
type SSHTestResult struct {
	ExitCode int
	Stdout   string
	Stderr   string
}

// SSH is the outbound port to the system ssh client.
type SSH interface {
	Test(ctx context.Context, req SSHTestRequest) (SSHTestResult, error)
}
