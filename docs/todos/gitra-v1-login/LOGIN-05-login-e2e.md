# LOGIN-05: Login End-to-End Regression

- **Todo ID**: LOGIN-05
- **Execution Map**: execution-map.json
- **Todo maturity**: DECOMPOSED
- **Task detail format**: v1
- **Source baseline**: docs/plans/gitra-v1-login-plan.md（WP-LOGIN-05）；docs/gitra-v1-login-onboarding-design.md；docs/gitra-v1-implementation-baseline.md

## Outcome

以端到端回归证明：登录 → 绑定 HTTPS 仓库 → 原生 git 可取回凭据 → 解绑/登出清理，且 token 永不落盘。

## Contract

- **Requirements**: `REQ-STEP-05`, `REQ-WP-05`, `REQ-TEST-04`, `REQ-VER-01`, `REQ-VER-02`, `REQ-OBJ-01`, `REQ-DONE-02`
- **Produces**: `artifact:login-e2e` — 登录全流程回归 + 凭据泄漏扫描
- **Gates**: `go test ./...`、`go vet ./...`
- **Boundaries**: 全程离线（httptest + 文件型凭据库）；不触碰真实系统凭据。

## Tasks

- [ ] `T01` 实现登录全流程 E2E（httptest provider + `store --file` 凭据库）。
  - **Test and RED**: `internal/e2e/login_flow_test.go`：login（stdin token）→ 断言账号字段来自 profile → bind HTTPS remote 仓库 → 在仓库内执行 `git credential fill`（注入同 helper 配置）返回该 token → unbind → 断言受管键清空；实现前 RED。
  - **Execution logic**: 测试内以临时 config dir、临时凭据文件与 httptest server 组装 bootstrap 变体（provider base URL 可注入）；对仓库断言使用真实 git 命令。
  - **Files and responsibilities**: 该测试文件负责端到端断言；如需注入 base URL，在 bootstrap 增加可选配置。
  - **Verification**: 正常全流程；边界：解绑后无残留；安全：token 不出现在仓库/配置目录任何文件中。
  - **Artifact handoff**: 长期回归用例。
- [ ] `T02` 实现错误路径与安全检查回归。
  - **Test and RED**: 覆盖：无 token 来源 → exit 5 且提示两种获取方式；401 → exit 5；logout 后 status 呈 `needs_login`（或 health 异常）；扫描 config dir、repo、stdout/stderr 确认无 token；实现前 RED。
  - **Execution logic**: 复用 T01 基础设施，逐用例独立临时环境。
  - **Files and responsibilities**: 与 T01 同文件不同 Test。
  - **Verification**: 各错误码命中且信息可操作；无 secret 泄漏。
  - **Artifact handoff**: 完成 M2.5 的验收证据，支撑用户自举验证。


## Tests

- 端到端登录/绑定/凭据/登出 + 泄漏扫描。
- **RED**: 流程不可完成或凭据不可取回。


## Done

- [ ] Every task has RED/GREEN evidence and the node Gates pass.
- [ ] Artifact/evidence/status are updated in the execution map.
- [ ] Graph and semantic checks pass.

## STOP conditions

- 需要真实网络、真实 token 或真实系统凭据库。
- 五轮修复仍失败。


## Evidence / Handoff

- GREEN：`internal/e2e/login_flow_test.go`——httptest 模拟 GitHub API + 真实 git + 受管凭据文件：login（stdin token）→ 账号自动填充（alias/username/name/email、strategy=https-token）→ 绑定 HTTPS remote 仓库 → **在仓库内用 `git credential fill` 取回 token**（等价于 push 时的取回路径）→ unbind 清空 credential 作用域并恢复身份 → logout 删除凭据。
- 错误路径与卫生：无凭据来源 exit 5 且提示 `--token-stdin`/`GITRA_TOKEN`；401 exit 5；登录后扫描配置目录断言 token 仅存在于 0600 的凭据文件，其余文件与命令输出均无泄漏。
- Gates：`go test ./...`、`go vet ./...` 全 PASS。
- Handoff：`artifact:login-e2e` 长期回归；M2.5 完成，用户可用 `gitra login` 后直接绑定并 push。
