package gitlab

import "regexp"

// sshSuccess matches "Welcome to GitLab, @<user>!"
var sshSuccess = regexp.MustCompile(`Welcome to GitLab, @([A-Za-z0-9][A-Za-z0-9._-]*)!`)

// ParseSSHVerification extracts the authenticated username from `ssh -T`
// output.
func ParseSSHVerification(stdout, stderr string) (string, bool) {
	if match := sshSuccess.FindStringSubmatch(stdout + "\n" + stderr); match != nil {
		return match[1], true
	}
	return "", false
}
