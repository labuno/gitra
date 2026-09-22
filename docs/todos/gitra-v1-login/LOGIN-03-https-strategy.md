# LOGIN-03: HTTPS Token Strategy & Binding Branch

- **Todo ID**: LOGIN-03
- **Execution Map**: execution-map.json
- **Todo maturity**: DECOMPOSED
- **Task detail format**: v1
- **Source baseline**: docs/plans/gitra-v1-login-plan.md（WP-LOGIN-03）；docs/gitra-v1-login-onboarding-design.md；docs/gitra-v1-implementation-baseline.md

## Outcome

新增 `https-token` Auth Strategy 并让 bind/status 按账号策略校验 remote 类型与 host，受管键扩展到 credential.*。

## Contract

- **Requirements**: `REQ-SCOPE-02`, `REQ-SCOPE-04`, `REQ-STEP-03`, `REQ-IF-04`, `REQ-IF-05`, `REQ-IF-06`, `REQ-IF-07`, `REQ-WP-03`, `REQ-TEST-03`
- **Produces**: `artifact:https-strategy` — https-token 策略 + 按策略分流的绑定校验
- **Gates**: `go test ./internal/strategies/...`、`go test ./internal/app/...`、`go test ./...`、`go vet ./...`
- **Boundaries**: 不把 token 写入任何仓库/配置文件；不改 remote URL。

## Tasks

- [ ] `T01` 实现 `https-token` Strategy：Validate 与 BuildGitConfig。
  - **Test and RED**: `internal/strategies/auth/httptoken/httptoken_test.go` 断言投影键为 `credential.<https-url>.username`、需要时补 `credential.helper`、缺少凭据时 Validate 返回 `ErrAuthInvalid`；实现前 RED。
  - **Execution logic**: Validate 通过注入的 `ports.SecretStore` 检查 ref 是否存在；BuildGitConfig 接收 `BuildRequest{Account, RemoteHost, RemoteExplicitPort}`，输出 URL 作用域 username 与（当 helper 解析结果需要时）helper 条目；不输出任何 secret。
  - **Files and responsibilities**: `internal/strategies/auth/httptoken/httptoken.go` + 测试；注册沿用 auth registry。
  - **Verification**: 正常：单账号投影；边界：多账号时仅写 repo-local；非法：凭据缺失；安全：条目中不含 token 字符串。
  - **Artifact handoff**: 供 LOGIN-04 建号（strategy=https-token）与 bind 使用。
- [ ] `T02` 实现绑定分流：remote 类型必须匹配账号策略，白名单扩展到 credential.*。
  - **Test and RED**: `internal/app/binding_service_test.go` 增补：https-token 账号 + HTTPS remote → 成功；https-token + SSH remote → `ErrUnsupportedRemote`；ssh-key + HTTPS remote → `ErrUnsupportedRemote`（保持既有行为）；drift/status 对 HTTPS 仓库成立；实现前 RED。
  - **Execution logic**: `validateRemote` 接收账号策略：ssh-key 走既有解析与 host 比对；https-token 解析 `https://host/owner/repo.git` 并比对 host；`managedKeys()` 在 https-token 时返回 `credential.<url>.username` 与（必要时）`credential.helper`；snapshot/restore/drift 复用同一列表。
  - **Files and responsibilities**: 修改 `internal/app/binding_service.go`（分流与白名单）、`internal/domain/remote_url.go`（新增 HTTPS 解析，保留 SSH 行为）；测试扩展。
  - **Verification**: 四种策略×协议组合矩阵；status --json 的 health 在两种策略下均正确。
  - **Artifact handoff**: CLI/TUI 按策略绑定；M4 复用同一服务。


## Tests

- 策略投影单测 + 绑定分流矩阵（含既有 SSH 回归）。
- **RED**: 策略与分流缺失时的断言失败。


## Done

- [ ] Every task has RED/GREEN evidence and the node Gates pass.
- [ ] Artifact/evidence/status are updated in the execution map.
- [ ] Graph and semantic checks pass.

## STOP conditions

- 需要修改 remote URL 才能通过测试。
- 需要偏离增补设计 §5.3/§5.4 的受管键约定。


## Evidence / Handoff

- RED：`go test ./internal/strategies/auth/httptoken/...` build failed（undefined: domain.StrategyHTTPSToken / New / BuildRequest.RemoteHost）。
- GREEN：`domain` 新增 `RemoteTransport`、`StrategyHTTPSToken`、`ParseHTTPSRemoteURL` 与 `Account.CredentialRef`；`ports.SecretStore`/`ports.CredentialHelperResolver`；auth 接口增加 `Transport()` 与 BuildRequest 的 `RemoteHost`/`CredentialHelper`；`httptoken` 策略（URL 作用域 username + 可选 helper，token 绝不进入配置条目）；`secretstore.HelperResolver`；binding 按策略分流校验 remote 类型与 host，白名单按账号策略生成；status/reconcile 在凭据缺失时报告 `NeedsLogin`/broken 而不报错。
- Gates：`go test ./internal/strategies/...`、`go test ./internal/app/...`、`go test ./...`、`go vet ./...` 全 PASS。
- Handoff：`artifact:https-strategy` 供 LOGIN-04 建号（strategy=https-token）与绑定使用。
