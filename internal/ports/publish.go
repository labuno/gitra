package ports

import (
	"context"

	"github.com/zhanhd/gitra/internal/domain"
)

// PublishStatus describes whether a first push is meaningful for a repository.
type PublishStatus struct {
	Branch          string
	HasLocalCommits bool
	RemoteHasBranch bool
}

// Publisher supports the one-time "publish this folder" step: it is NOT a
// general git proxy (baseline §7 keeps push/pull/fetch out of gitra), it only
// answers "is the remote still empty?" and performs the first push.
type Publisher interface {
	PublishStatus(ctx context.Context, repo domain.Repository) (PublishStatus, error)
	Push(ctx context.Context, repo domain.Repository, remote, branch string, setUpstream bool) error
}
