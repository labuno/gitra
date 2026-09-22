package gitea

import "regexp"

// sshSuccess matches "Hi there, <user>! You've successfully authenticated with
// the key named ..., but Gitea does not provide shell access."
var sshSuccess = regexp.MustCompile(`Hi there, ([A-Za-z0-9][A-Za-z0-9._-]*)!`)

// ParseSSHVerification extracts the authenticated username from `ssh -T`
// output.
func ParseSSHVerification(stdout, stderr string) (string, bool) {
	if match := sshSuccess.FindStringSubmatch(stdout + "\n" + stderr); match != nil {
		return match[1], true
	}
	return "", false
}
