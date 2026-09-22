# LOGIN-04: Login / Logout CLI

- **Todo ID**: LOGIN-04
- **Execution Map**: execution-map.json
- **Todo maturity**: DECOMPOSED
- **Task detail format**: v1
- **Source baseline**: docs/plans/gitra-v1-login-plan.md（WP-LOGIN-04）；docs/gitra-v1-login-onboarding-design.md；docs/gitra-v1-implementation-baseline.md

## Outcome

提供 `gitra login`（自动建号/更新 + 凭据落库）与 `gitra logout`（删除凭据 + 撤销指引），输出绝不包含 token。

## Contract

- **Requirements**: `REQ-STEP-04`, `REQ-IF-03`, `REQ-WP-04`, `REQ-OBJ-01`, `REQ-DONE-01`
- **Produces**: `artifact:login-cli` — LoginService 与 login/logout 命令
- **Gates**: `go test ./internal/presentation/...`、`go test ./...`、`go vet ./...`
- **Boundaries**: login 不绑定任何仓库；logout 不删除账号（只删凭据并提示）；不打印 token。

## Tasks

- [ ] `T01` 实现 LoginService 与 `gitra login`。
  - **Test and RED**: CLI 测试：`login --provider github --token-stdin`（stdin 提供 token，httptest profile）→ 账号自动创建（username/name/email 来自 profile、strategy=https-token）、凭据入库、stdout 无 token；重复 login 更新既有账号（按 username 匹配）；实现前 RED。
  - **Execution logic**: LoginService 步骤：Resolver 取 token → Provider.Profile → 查找同 host+username 账号（存在则更新，不存在则创建）→ SecretStore.Set(ref) → 返回摘要（alias、username、email、host）；`--json` 输出 A.1 风格账号对象但不含凭据。
  - **Files and responsibilities**: `internal/app/login_service.go`、`internal/presentation/cli/login.go`、`internal/adapters/provider/resolver.go` 接线、测试。
  - **Verification**: 正常/重复登录/自定义 alias；非法：无 token 来源（exit 5 + 指引）；安全：stdout/stderr 与配置目录均无 token。
  - **Artifact handoff**: LOGIN-05 与 TUI 引导复用。
- [ ] `T02` 实现 `gitra logout <alias>` 与受管凭据清理。
  - **Test and RED**: 断言 logout 后 SecretStore 中 ref 消失、账号仍在、已绑定 HTTPS 仓库的 `credential.<url>.username` 被清理/健康度变为 needs_login；实现前 RED。
  - **Execution logic**: logout 删除凭据 ref，然后对账号的每个绑定执行 reconcile（凭据缺失会让策略校验失败 → 状态呈现 `needs_login`）；输出平台撤销页链接（GitHub/GitLab/Gitea 各自的 token 设置页）。
  - **Files and responsibilities**: `internal/presentation/cli/logout.go`、LoginService.Logout、测试。
  - **Verification**: 正常/边界（无绑定账号）/非法（未知 alias → exit 3）；安全：仅删除凭据，不动用户其它 git 配置。
  - **Artifact handoff**: 完成后「登录 → 绑定 → 登出」闭环可供 E2E 使用。


## Tests

- CLI 级登录/登出测试（httptest + stdin token + 隔离 config dir）。
- **RED**: 命令未注册（unknown command）。


## Done

- [ ] Every task has RED/GREEN evidence and the node Gates pass.
- [ ] Artifact/evidence/status are updated in the execution map.
- [ ] Graph and semantic checks pass.

## STOP conditions

- 需要真实网络/token。
- 需要在输出中回显 token 才能调试（禁止）。


## Evidence / Handoff

- GREEN：`ports.TokenProvider`/`ProfileProvider`；`app.LoginService`（阶梯取 token → profile → 先存凭据再建号/更新，失败回滚已存凭据；Logout 删除凭据并清理 repo-local `credential.*` 但保留账号与 gitra.* 元数据）；`adapters/provider/router`（按 provider 路由 + 可注入 BaseURL/HTTP）；`bootstrap.NewFromDeps`（测试注入点）；CLI `gitra login`/`gitra logout`（`--token-stdin`/`GITRA_TOKEN`/gh 复用；输出与日志绝不含 token；邮箱回退时给出提示；logout 附撤销链接）。
- 测试：登录建号、二次登录更新同一账号（revision=2、别名改名）、无凭据时 exit 5 且给出两种获取方式、logout 后 repo 内 credential 键清空且 `status --json` 报 `auth.state=needs_login`（exit 6）。
- Gates：`go test ./internal/presentation/...`、`go test ./...`、`go vet ./...` 全 PASS。
- Handoff：`artifact:login-cli` 供 LOGIN-05 与 TUI 引导复用。
