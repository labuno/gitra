package tui

import (
	"context"
	"errors"
	"fmt"

	"github.com/zhanhd/gitra/internal/domain"
)

func providerLabel(providerType domain.ProviderType) string {
	switch providerType {
	case domain.ProviderGitHub:
		return "GitHub"
	case domain.ProviderGitLab:
		return "GitLab"
	case domain.ProviderGitea:
		return "Gitea"
	default:
		return string(providerType)
	}
}

func authLabel(account domain.Account) string {
	switch account.Transport.Strategy {
	case domain.StrategyHTTPSToken:
		return "安全登录（HTTPS）"
	case domain.StrategySSHKey:
		return "SSH 密钥"
	default:
		return account.Transport.Strategy
	}
}

func stateLabel(state string) (string, string) {
	switch state {
	case "configured", "ok":
		return "● 可用", "ok"
	case "needs_login":
		return "● 需要重新登录", "warn"
	case "drift":
		return "● 配置被改动", "warn"
	case "missing":
		return "● 文件夹不存在", "warn"
	default:
		return "● 有问题", "err"
	}
}

func healthLabel(state string) string {
	text, _ := stateLabel(state)
	return text
}

// plainError translates internal errors into user-facing guidance.
func plainError(err error) string {
	if err == nil {
		return ""
	}
	switch {
	case errors.Is(err, context.DeadlineExceeded), errors.Is(err, context.Canceled):
		return "操作超时：网络或系统凭据库没有响应，请稍后重试。"
	case errors.Is(err, domain.ErrAuthInvalid):
		return "登录已失效：请在账号页按 T 重新登录。"
	case errors.Is(err, domain.ErrAlreadyBound):
		return "这个文件夹已经绑定过账号了。"
	case errors.Is(err, domain.ErrBindingNotFound):
		return "没有找到这个文件夹的绑定记录。"
	case errors.Is(err, domain.ErrNotGitRepository):
		return "这个文件夹不是 Git 仓库（没有 .git）。请选择克隆下来的项目文件夹。"
	case errors.Is(err, domain.ErrUnsupportedRepo):
		return "暂不支持这种仓库类型（例如裸仓库或 worktree）。"
	case errors.Is(err, domain.ErrUnsupportedRemote):
		return "仓库地址不受支持：请在 Git 里把远程地址改成 https://… 或 git@… 形式。"
	case errors.Is(err, domain.ErrProviderMismatch):
		return "仓库地址与账号不是同一个平台，请检查是否选错了账号。"
	case errors.Is(err, domain.ErrAccountNotFound):
		return "没有找到这个账号。"
	case errors.Is(err, domain.ErrAccountExists):
		return "同名账号已存在。"
	case errors.Is(err, domain.ErrAccountInUse):
		return "账号还有绑定的项目，需要先解除绑定。"
	case errors.Is(err, domain.ErrInvalid):
		return "输入有误：" + err.Error()
	default:
		return fmt.Sprintf("%v", err)
	}
}
