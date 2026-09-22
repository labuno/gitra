# gitra V1.1 登录与 HTTPS 凭据实施计划

> Plan ID：`gitra-v1-login`
> 范围：增补设计 §4–§6 的登录 MVP（PAT + gh 复用 + 系统凭据存储 + repo-local credential 投影 + logout），device flow 仅保留接口
> 上游约束：`docs/gitra-v1-login-onboarding-design.md`、`docs/gitra-v1-implementation-baseline.md`（修订 1）、`docs/gitra-architecture-development-design.md`

---

## Objective

让用户无需创建或粘贴 SSH key 即可完成身份接入：一条命令登录 Provider，凭据安全落库，绑定后原生 `git push` 直接使用正确账号。

## Scope

- 登录阶梯（可用性降级）：复用已登录的 `gh`/`glab` → `GITRA_TOKEN` 环境变量或标准输入提供访问码（PAT）→ device flow（接口预留，client_id 未注册前不启用）。
- SecretStore：基于 git 原生 `credential approve/fill/reject` 的系统凭据读写；helper 解析顺序为「用户全局 helper → 平台原生 helper → 受管文件库（0600）」。
- Profile 自动填充：GitHub `GET /user` + `/user/emails`；GitLab `/api/v4/user`；Gitea `/api/v1/user`。
- 新增 `https-token` Auth Strategy：投影 `credential.<url>.username`（必要时补 helper），不写 token 到任何文本文件。
- bind 校验按账号策略分流：`ssh-key` 要求 SSH remote；`https-token` 要求 HTTPS remote。
- `gitra login` / `gitra logout` 命令与 `--json`；logout 删除凭据并提示平台侧撤销。

## Non-goals

- 浏览器一键授权（device flow 的真实实现）与 OAuth App 注册；本 Plan 只留接口与配置位。
- TUI 页面（Milestone 4）、引导式克隆（M4.5）、token 云同步、企业 SSO 深度适配。
- 自动修改 remote URL。

## Steps

1. SecretStore 端口与 git-credential adapter。
2. Provider 登录与 profile（gh 复用 + PAT + 可注入 HTTP）。
3. `https-token` Strategy 与 bind 分流。
4. `login` / `logout` CLI 与账号自动创建。
5. 端到端回归（登录 → 绑定 HTTPS 仓库 → 凭据可取回 → 解绑/登出清理）。

## Interfaces

- `ports.SecretStore`：`Get/Set/Delete(ctx, ref string)`；ref 形如 `github.com/lunafoundry`。
- `ports.Profile`：`{Provider, Host, Username, Name, Email}`；登录后用于自动建号。
- `internal/adapters/secretstore`：git-credential 实现；`internal/adapters/provider/{github,gitlab,gitea}` 各带 `Profile(ctx, token)`。
- CLI：`gitra login --provider github [--host H] [--alias A] [--token-stdin] [--json]`；`gitra logout <alias>`。

## Work Packages

### LOGIN-01 SecretStore & Git Credential Adapter

git credential approve/fill/reject 封装、helper 解析与降级、ref 约定。

### LOGIN-02 Provider Login & Profile

gh/glab 复用、PAT 输入（env/stdin）、可注入 HTTP 的 profile 拉取与错误分类。

### LOGIN-03 HTTPS Token Strategy & Binding Branch

`https-token` Auth Strategy；bind/status 按策略校验 remote 类型与 host；managed key 白名单扩展。

### LOGIN-04 Login / Logout CLI

`login`（自动建号或更新、--json）与 `logout`（删除凭据、提示撤销）命令。

### LOGIN-05 Login End-to-End Regression

隔离环境下的登录 → 绑定 → 凭据取回 → 解绑 → 登出全流程与非交互错误路径。

## Test Matrix

- secretstore 单测（fake runner 断言命令序列）+ 用 `store --file=<tmp>` 的真实 git 集成测试。
- provider profile 单测（httptest 固定响应与错误码分类）。
- strategy 单测（投影键与 helper 决策）。
- CLI E2E（隔离 GITRA_CONFIG_DIR + 临时仓库 + httptest provider）。

## Verification

- 每节点 `go test ./...` + `go vet ./...`；E2E 断言 token 不出现在任何文本文件与日志中。

## Done

- 全部 Work Package 完成、节点验证成功、证据入图。
- 用户可用一条 `gitra login`（PAT 或 gh 复用）完成接入，并在绑定仓库中由原生 git 取回凭据。

## STOP Conditions

- 需要在测试中写入真实系统凭据库或真实网络。
- 需要修改既有 Application 公共语义才能实现。
- 源文档 digest 漂移或语义复核失败。

## External Sources

- gh CLI（可选，`gh auth token`）；glab CLI（可选）。
- GitHub/GitLab/Gitea REST API（profile 与 token 语义）。
- 系统 git 的 credential helper 机制。
