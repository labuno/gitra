package github

import "regexp"

// sshSuccess matches "Hi <user>! You've successfully authenticated, but GitHub
// does not provide shell access."
var sshSuccess = regexp.MustCompile(`Hi ([A-Za-z0-9][A-Za-z0-9-]*)! You've successfully authenticated`)

// ParseSSHVerification extracts the authenticated username from `ssh -T`
// output. GitHub returns a non-zero exit code on success, so callers must not
// rely on the exit code.
func ParseSSHVerification(stdout, stderr string) (string, bool) {
	if match := sshSuccess.FindStringSubmatch(stdout + "\n" + stderr); match != nil {
		return match[1], true
	}
	return "", false
}
