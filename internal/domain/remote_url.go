package domain

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// RemoteURL is the parsed form of an SSH remote URL. ExplicitPort reports
// whether the URL itself carried a port, which drives the baseline §14 rule
// for generating core.sshCommand.
type RemoteURL struct {
	Host         string
	Port         int
	ExplicitPort bool
	Path         string
}

// ParseRemoteURL accepts scp-like (git@host:owner/repo.git) and ssh:// forms.
// Anything else (notably HTTPS) is unsupported in V1.0 core.
func ParseRemoteURL(raw string) (RemoteURL, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return RemoteURL{}, fmt.Errorf("%w: empty remote url", ErrUnsupportedRemote)
	}

	if strings.HasPrefix(raw, "ssh://") {
		parsed, err := url.Parse(raw)
		if err != nil || parsed.Hostname() == "" {
			return RemoteURL{}, fmt.Errorf("%w: %s", ErrUnsupportedRemote, raw)
		}
		result := RemoteURL{Host: parsed.Hostname(), Port: 22, Path: strings.TrimPrefix(parsed.Path, "/")}
		if port := parsed.Port(); port != "" {
			number, err := strconv.Atoi(port)
			if err != nil || number < 1 || number > 65535 {
				return RemoteURL{}, fmt.Errorf("%w: bad port in %s", ErrUnsupportedRemote, raw)
			}
			result.Port = number
			result.ExplicitPort = true
		}
		return result, nil
	}

	at := strings.IndexByte(raw, '@')
	colon := strings.IndexByte(raw, ':')
	if at > 0 && colon > at+1 && colon < len(raw)-1 {
		return RemoteURL{Host: raw[at+1 : colon], Port: 22, Path: raw[colon+1:]}, nil
	}
	return RemoteURL{}, fmt.Errorf("%w: %s", ErrUnsupportedRemote, raw)
}
