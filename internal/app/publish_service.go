package app

import (
	"context"
	"fmt"
	"time"

	"github.com/zhanhd/gitra/internal/domain"
	"github.com/zhanhd/gitra/internal/ports"
)

// publishTimeout bounds one first push.
const publishTimeout = 90 * time.Second

// PublishView is the read model behind the "first upload" action.
type PublishView struct {
	Bound           bool
	Remote          string
	Branch          string
	HasLocalCommits bool
	RemoteHasBranch bool
	CanPublish      bool
	Reason          string
}

// PublishResult reports a completed first upload.
type PublishResult struct {
	Branch string
	Remote string
}

// PublishService performs the one-time publish step: pushing a local branch to
// an empty remote and setting its upstream. It deliberately does not offer
// general push/pull/fetch (baseline §7).
type PublishService struct {
	deps      Deps
	publisher ports.Publisher
}

// NewPublishService wires the publish use case.
func NewPublishService(deps Deps) *PublishService {
	return &PublishService{deps: deps, publisher: deps.Publisher}
}

// Status explains whether a first upload makes sense right now.
func (s *PublishService) Status(ctx context.Context, path string) (PublishView, error) {
	repo, err := s.deps.Git.DiscoverRepository(ctx, path)
	if err != nil {
		return PublishView{}, err
	}
	view := PublishView{}

	_, found, err := s.deps.Bindings.FindByRepository(ctx, repo.RootPath)
	if err != nil {
		return PublishView{}, err
	}
	view.Bound = found

	origin, hasOrigin := repo.RemoteByName("origin")
	if hasOrigin {
		view.Remote = origin.URL
	}

	if s.publisher == nil {
		view.Reason = "上传功能不可用（缺少 git 发布通道）"
		return view, nil
	}
	status, err := s.publisher.PublishStatus(ctx, repo)
	if err != nil {
		return view, err
	}
	view.Branch = status.Branch
	view.HasLocalCommits = status.HasLocalCommits
	view.RemoteHasBranch = status.RemoteHasBranch

	switch {
	case !found:
		view.Reason = "这个文件夹还没有绑定账号"
	case !hasOrigin:
		view.Reason = "这个文件夹还没有仓库地址"
	case !status.HasLocalCommits:
		view.Reason = "还没有任何提交：先提交一次代码再上传"
	case status.RemoteHasBranch:
		view.Reason = fmt.Sprintf("远端已有 %s 分支：直接在编辑器里同步即可", status.Branch)
	default:
		view.CanPublish = true
		view.Reason = fmt.Sprintf("可以上传：%s → origin/%s", status.Branch, status.Branch)
	}
	return view, nil
}

// FirstPublish pushes the current branch for the first time and verifies that
// the remote actually received it.
func (s *PublishService) FirstPublish(ctx context.Context, path string) (PublishResult, error) {
	view, err := s.Status(ctx, path)
	if err != nil {
		return PublishResult{}, err
	}
	if !view.Bound {
		return PublishResult{}, fmt.Errorf("%w: %s", domain.ErrBindingNotFound, path)
	}
	if !view.CanPublish {
		return PublishResult{}, fmt.Errorf("%w: %s", domain.ErrInvalid, view.Reason)
	}
	repo, err := s.deps.Git.DiscoverRepository(ctx, path)
	if err != nil {
		return PublishResult{}, err
	}

	pushCtx, cancel := context.WithTimeout(ctx, publishTimeout)
	defer cancel()
	if err := s.publisher.Push(pushCtx, repo, "origin", view.Branch, true); err != nil {
		return PublishResult{}, fmt.Errorf("上传失败：%w", err)
	}

	status, err := s.publisher.PublishStatus(ctx, repo)
	if err != nil {
		return PublishResult{}, err
	}
	if !status.RemoteHasBranch {
		return PublishResult{}, fmt.Errorf("%w: 上传后没有在远端看到 %s 分支", domain.ErrStateDrift, view.Branch)
	}
	return PublishResult{Branch: view.Branch, Remote: "origin"}, nil
}
