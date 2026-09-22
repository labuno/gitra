package app

import (
	"context"
	"fmt"
	"strings"

	"github.com/zhanhd/gitra/internal/domain"
	"github.com/zhanhd/gitra/internal/ports"
)

// RemoteCheck is the provider-side state of the repository a folder points at.
type RemoteCheck struct {
	Exists   bool
	Host     string
	Owner    string
	Name     string
	CloneURL string
	Private  bool
}

// RemoteService verifies and creates the provider-side repository, which is
// what a first-time publisher needs before git can push anywhere.
type RemoteService struct {
	deps     Deps
	repos    ports.RepoService
	bindings *BindingService
}

// NewRemoteService wires the provider repository operations.
func NewRemoteService(deps Deps, repos ports.RepoService, bindings *BindingService) *RemoteService {
	return &RemoteService{deps: deps, repos: repos, bindings: bindings}
}

// Check reports whether the folder's origin repository exists on the provider.
func (s *RemoteService) Check(ctx context.Context, accountID domain.AccountID, path string) (RemoteCheck, error) {
	account, secret, owner, name, host, err := s.contextFor(ctx, accountID, path)
	if err != nil {
		return RemoteCheck{}, err
	}
	status, err := s.repos.LookupRepo(ctx, account.Provider.Type, host, secret, owner, name)
	if err != nil {
		return RemoteCheck{}, err
	}
	return RemoteCheck{
		Exists: status.Exists, Host: host, Owner: owner, Name: name,
		CloneURL: status.CloneURL, Private: status.Private,
	}, nil
}

// CreateAndBind creates the provider-side repository (under the account's own
// namespace) and then binds the folder to the account.
func (s *RemoteService) CreateAndBind(ctx context.Context, accountID domain.AccountID, path string, private bool) (domain.RepositoryBinding, error) {
	account, secret, owner, name, host, err := s.contextFor(ctx, accountID, path)
	if err != nil {
		return domain.RepositoryBinding{}, err
	}
	if !strings.EqualFold(owner, account.Provider.Username) {
		return domain.RepositoryBinding{}, fmt.Errorf(
			"%w: 只能在你自己的账号（%s）下创建仓库；若要使用组织仓库，请先在网页创建后再回来绑定",
			domain.ErrInvalid, account.Provider.Username)
	}

	status, err := s.repos.CreateRepo(ctx, account.Provider.Type, host, secret, owner, name, private)
	if err != nil {
		return domain.RepositoryBinding{}, err
	}
	cloneURL := strings.TrimSpace(status.CloneURL)
	if cloneURL == "" {
		cloneURL = fmt.Sprintf("https://%s/%s/%s.git", host, owner, name)
	}
	if err := s.bindings.EnsureOriginRemote(ctx, path, cloneURL); err != nil {
		return domain.RepositoryBinding{}, err
	}
	return s.bindings.Bind(ctx, BindRequest{AccountID: accountID, Path: path})
}

// contextFor resolves everything the provider calls need.
func (s *RemoteService) contextFor(ctx context.Context, accountID domain.AccountID, path string) (domain.Account, string, string, string, string, error) {
	account, err := s.deps.Accounts.Get(ctx, accountID)
	if err != nil {
		return domain.Account{}, "", "", "", "", err
	}
	if account.Transport.Strategy != domain.StrategyHTTPSToken {
		return domain.Account{}, "", "", "", "", fmt.Errorf(
			"%w: 只有通过登录（HTTPS 凭据）接入的账号可以在线校验仓库地址", domain.ErrUnsupportedRepo)
	}
	if s.repos == nil {
		return domain.Account{}, "", "", "", "", fmt.Errorf("provider API is not configured in this build")
	}
	secret, err := s.deps.Secrets.Get(ctx, account.CredentialRef())
	if err != nil {
		return domain.Account{}, "", "", "", "", fmt.Errorf("%w: 登录已失效，请重新登录", domain.ErrAuthInvalid)
	}
	repo, err := s.deps.Git.DiscoverRepository(ctx, path)
	if err != nil {
		return domain.Account{}, "", "", "", "", err
	}
	origin, ok := repo.RemoteByName("origin")
	if !ok {
		return domain.Account{}, "", "", "", "", fmt.Errorf("%w: 这个文件夹还没有仓库地址", domain.ErrUnsupportedRemote)
	}
	parsed, err := domain.ParseHTTPSRemoteURL(origin.URL)
	if err != nil {
		return domain.Account{}, "", "", "", "", fmt.Errorf(
			"%w: 只有 https:// 形式的地址可以在线校验或创建", domain.ErrUnsupportedRemote)
	}
	owner, name := splitRepoPath(parsed.Path)
	if owner == "" || name == "" {
		return domain.Account{}, "", "", "", "", fmt.Errorf("%w: 无法从地址中解析出仓库名", domain.ErrUnsupportedRemote)
	}
	return account, secret, owner, name, parsed.Host, nil
}

// splitRepoPath turns "group/sub/repo.git" into ("group/sub", "repo").
func splitRepoPath(path string) (owner string, name string) {
	path = strings.TrimSuffix(strings.Trim(path, "/"), ".git")
	index := strings.LastIndex(path, "/")
	if index <= 0 {
		return "", ""
	}
	return path[:index], path[index+1:]
}
