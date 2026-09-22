# CLIC-03: Account Commands

- **Todo ID**: CLIC-03
- **Execution Map**: execution-map.json
- **Todo maturity**: DECOMPOSED
- **Task detail format**: v1
- **Source baseline**: docs/plans/gitra-v1-cli-plan.md（WP-CLI-03）；docs/gitra-v1-implementation-baseline.md；docs/gitra-v1-login-onboarding-design.md

## Outcome

提供 account add/list/show/edit/remove 五个子命令，非交互、支持 --json，覆盖基线 §47 的账号管理能力。

## Contract

- **Requirements**: `REQ-STEP-03`, `REQ-SCOPE-02`, `REQ-SCOPE-03`, `REQ-IF-02`, `REQ-WP-03`, `REQ-TEST-02`
- **Produces**: `artifact:cli-account` — account 子命令与文本/JSON 输出
- **Gates**: `go test ./internal/presentation/...`、`go test ./...`、`go vet ./...`
- **Boundaries**: 缺参即退出 2，不进入交互提问；不提供 account test（M3）。

## Tasks

- [ ] `T01` 实现 `account add`、`account list`、`account show`。
  - **Test and RED**: 集成测试断言 add 成功后 list/show 可见、省略 host 时使用 provider 默认值、缺 `--key` 退出 2；实现前 RED。
  - **Execution logic**: flags 组装 `CreateAccountRequest`；`--provider` 必填且校验枚举；`--host/--ssh-user/--ssh-port` 缺省取 `domain.DefaultEndpoint`；alias 冲突（ErrAccountExists）→ exit 3；show 接受 alias 或 `acc_` ID。
  - **Files and responsibilities**: `internal/presentation/cli/account.go`（命令 + flag 绑定）、`account_render.go`（文本与 JSON 渲染）。
  - **Verification**: 正常/非法（未知 provider、缺参、重复 alias）/边界（无账号时 list 输出 `[]`）。
  - **Artifact handoff**: 供 CLIC-05 回归调用。
- [ ] `T02` 实现 `account edit` 与 `account remove`。
  - **Test and RED**: edit 改 email 后绑定仓库内 `user.email` 同步变化；remove 在有绑定时 exit 3 且提示、带 `--unbind-all` 时成功；实现前 RED。
  - **Execution logic**: edit 仅覆盖显式提供的 flags（未提供保持原值），调用 `Update`；若返回 ErrStateDrift 包装则在 stderr 打印「已保存 + 受影响仓库」，exit 6；remove 缺省拒绝删除有绑定账号，`--unbind-all` 走服务层解绑后删除。
  - **Files and responsibilities**: 复用 `account.go`；服务调用错误原样上抛由根命令映射。
  - **Verification**: 正常/冲突/边界（edit 无 flag 时报 2；remove 不存在账号 → 3）。
  - **Artifact handoff**: 完成 §47 账号生命周期闭环。


## Tests

- CLI 集成测试（隔离 `GITRA_CONFIG_DIR`）覆盖五个子命令与 JSON。
- **RED**: 命令未实现时 Execute 返回 2 或无输出。


## Done

- [ ] Every task has RED/GREEN evidence and the node Gates pass.
- [ ] Artifact/evidence/status are updated in the execution map.
- [ ] Graph and semantic checks pass.

## STOP conditions

- 需要显示层实现业务规则（应下沉到 Application）。
- 需要交互式输入才能完成（与 M2 非交互约定冲突）。


## Evidence / Handoff

- RED：`go test ./internal/presentation/...` 报 `unknown command "account" for "gitra"`。
- GREEN：`account add/list/show/edit/remove`（非交互、缺参 exit 2、alias 冲突 exit 3、edit 无变更 exit 2、remove 有绑定 exit 3 并提示 `--unbind-all`、edit 后自动 reconcile 且仓库内身份同步）。
- Gates：`go test ./internal/presentation/...`、`go test ./...`、`go vet ./...` 全 PASS。
- Handoff：`artifact:cli-account`；CLIC-04/05 与后续 TUI 复用同一 Application 调用方式。
