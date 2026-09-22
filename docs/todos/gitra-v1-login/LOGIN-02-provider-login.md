# LOGIN-02: Provider Login & Profile

- **Todo ID**: LOGIN-02
- **Execution Map**: execution-map.json
- **Todo maturity**: DECOMPOSED
- **Task detail format**: v1
- **Source baseline**: docs/plans/gitra-v1-login-plan.md（WP-LOGIN-02）；docs/gitra-v1-login-onboarding-design.md；docs/gitra-v1-implementation-baseline.md

## Outcome

提供 Provider 登录（gh/glab 复用、PAT 输入）与 profile 拉取（可注入 HTTP），并给出可读的错误分类。

## Contract

- **Requirements**: `REQ-SCOPE-01`, `REQ-SCOPE-03`, `REQ-STEP-02`, `REQ-WP-02`, `REQ-TEST-02`, `REQ-EXT-01`
- **Produces**: `artifact:provider-login` — LoginResolver 与 GitHub/GitLab/Gitea profile 适配
- **Gates**: `go test ./internal/adapters/provider/...`、`go test ./...`、`go vet ./...`
- **Boundaries**: 不实现 device flow（仅接口占位）；不打印 token；不自动修改远端。

## Tasks

- [ ] `T01` 实现 GitHub 登录源与 profile：gh 复用、`GITRA_TOKEN`/stdin、`GET /user` + `/user/emails`。
  - **Test and RED**: `internal/adapters/provider/github/github_test.go` 用 httptest 断言 profile 解析（username/name/email）与 401→`domain.ErrAuthInvalid`、403→可读错误；fake runner 提供 `gh auth token` 输出；实现前 RED。
  - **Execution logic**: 登录源顺序：显式 token → `GITRA_TOKEN` → stdin → `gh auth token`；profile 请求带 `Authorization: Bearer`；email 取 primary+verified，缺失时回退 `<username>@users.noreply.github.com` 并在返回结构中标记 `EmailFallback`。
  - **Files and responsibilities**: `internal/adapters/provider/github/github.go`（LoginSources + Profile）、`github_test.go`；HTTP 基址可注入（默认 `https://api.github.com`）。
  - **Verification**: 正常/401/403/网络错误/缺 scope 各一例；安全：错误信息不得回显 token。
  - **Artifact handoff**: 供 LOGIN-04 的 LoginService 使用。
- [ ] `T02` 实现 GitLab/Gitea profile 与统一 LoginResolver（阶梯选择）。
  - **Test and RED**: `provider_test.go` 断言三者 profile 端点与鉴权头差异（GitLab `PRIVATE-TOKEN`、Gitea `Authorization: token`）、Resolver 按可用性选择登录方式并在全不可用时返回 `ErrAuthInvalid` 且附获取指引；实现前 RED。
  - **Execution logic**: `LoginResolver.Login(ctx, provider, opts)` 顺序：env/stdin token → gh/glab → 报错（含为 GitHub 生成个人访问码的 URL 指引）；device flow 返回 `ErrUnsupported`（占位，等待 client_id 配置）。
  - **Files and responsibilities**: `internal/adapters/provider/{gitlab,gitea}/*.go`、`internal/adapters/provider/resolver.go` 与测试。
  - **Verification**: 正常：三 provider 各自 profile；边界：无任何 token 来源；非法：host 为空；安全：无 secret 日志。
  - **Artifact handoff**: LOGIN-04 通过 Resolver 一次调用完成登录。


## Tests

- httptest 固定响应 + 错误码分类；fake runner 提供 CLI token。
- **RED**: 实现缺失时的编译失败。


## Done

- [ ] Every task has RED/GREEN evidence and the node Gates pass.
- [ ] Artifact/evidence/status are updated in the execution map.
- [ ] Graph and semantic checks pass.

## STOP conditions

- 需要真实网络或真实 token 才能测试。
- 需要实现 device flow（client_id 未注册）。


## Evidence / Handoff

- RED：`go test ./internal/adapters/provider/...` build failed（undefined: TokenResolver / NewTokenResolver / TokenRequest）。
- GREEN：`TokenResolver` 阶梯（explicit → `GITRA_TOKEN` → stdin → `gh auth token`/`glab --show-token`）并在全不可用时返回带获取指引的 `ErrAuthInvalid`；GitHub/GitLab/Gitea `Profile`（可注入 base URL 与 HTTP client，httptest 覆盖）；`DoJSON` 将 401/403 归类为凭证问题、5xx/网络/坏响应归类为 `ErrProfileUnavailable`。
- Gates：`go test ./internal/adapters/provider/...`、`go test ./...`、`go vet ./...` 全 PASS。
- Handoff：`artifact:provider-login` 供 LOGIN-04 的 LoginService 与后续 device flow 扩展使用。
