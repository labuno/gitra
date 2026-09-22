# CLIC-04: Bind / Unbind / Status Commands

- **Todo ID**: CLIC-04
- **Execution Map**: execution-map.json
- **Todo maturity**: DECOMPOSED
- **Task detail format**: v1
- **Source baseline**: docs/plans/gitra-v1-cli-plan.md（WP-CLI-04）；docs/gitra-v1-implementation-baseline.md；docs/gitra-v1-login-onboarding-design.md

## Outcome

提供 bind/unbind/status 三个命令，status 输出增补设计 A.1 冻结的 JSON 形状并与健康度联动。

## Contract

- **Requirements**: `REQ-IF-03`, `REQ-IF-04`, `REQ-IF-06`, `REQ-STEP-04`, `REQ-WP-04`, `REQ-TEST-02`
- **Produces**: `artifact:cli-binding` — bind/unbind/status 命令与 A.1 JSON
- **Gates**: `go test ./internal/presentation/...`、`go test ./...`、`go vet ./...`
- **Boundaries**: 不修改 remote URL；不自动 clone；status 不联网。

## Tasks

- [ ] `T01` 实现 `bind` 与 `unbind`。
  - **Test and RED**: 集成测试断言 `bind <alias>` 默认绑定当前目录、`bind <alias> <path>` 指定目录、重复绑定 exit 6、HTTPS remote exit 7 且带可操作提示、`unbind` 恢复绑定前状态；实现前 RED。
  - **Execution logic**: alias 解析优先 `GetByAlias`，失败再按 `acc_` ID `Get`；path 缺省 `os.Getwd()`；错误信息保留服务层措辞（含「改用 SSH」指引）并附 alias/path 上下文。
  - **Files and responsibilities**: `internal/presentation/cli/bind.go`（bind/unbind 命令）。
  - **Verification**: 正常/冲突/非法（未知 alias → 3；非仓库路径 → 4）/安全（不改 remote）。
  - **Artifact handoff**: 供 CLIC-05 回归与后续 TUI 复用同一服务调用方式。
- [ ] `T02` 实现 `status`（文本 + `--json`）与健康度退出码。
  - **Test and RED**: 断言未绑定输出 `{"schema_version":1,"repository":"/abs","bound":false}` 且 exit 0；已绑定输出含 binding/account/auth/health 字段；注入 drift 后 exit 6；实现前 RED。
  - **Execution logic**: 未绑定 → exit 0；bound 且 health=ok → 0；health∈{drift,broken,missing} → 6 并在 stderr 说明修复动作（`gitra bind` 重新绑定或排查路径）；JSON 字段顺序固定，`auth.state` 取 `configured/needs_login/invalid` 本地判定。
  - **Files and responsibilities**: `internal/presentation/cli/status.go` 与 JSON DTO。
  - **Verification**: 正常/边界（未绑定、health=missing）/非法（非仓库 → 4）；JSON 字段与 A.1 一致。
  - **Artifact handoff**: 脚本/Agent 可直接消费 status --json。


## Tests

- CLI 集成测试覆盖 bind/unbind/status 的正常、冲突与非法路径。
- **RED**: 命令缺失时断言失败。


## Done

- [ ] Every task has RED/GREEN evidence and the node Gates pass.
- [ ] Artifact/evidence/status are updated in the execution map.
- [ ] Graph and semantic checks pass.

## STOP conditions

- 需要网络或真实远端才能通过测试。
- JSON 形状需要偏离 A.1 冻结契约。


## Evidence / Handoff

- RED：`bind`/`unbind`/`status` 未注册（unknown command）。
- GREEN：`bind <alias> [path]`（默认 cwd、重复绑定 exit 6、HTTPS remote exit 7 且含 SSH 指引、未知 alias exit 3、非仓库 exit 4）；`unbind` 恢复绑定前状态；`status` 文本 + A.1 冻结 JSON（未绑定 `bound:false` exit 0；已绑定含 binding/account/auth/health；drift 时 JSON 仍输出且 exit 6）。
- 测试修正：断言使用 `git rev-parse --show-toplevel` 的 canonical 路径（与 §4.9 一致），非测试实现缺陷。
- Gates：`go test ./internal/presentation/...`、`go test ./...`、`go vet ./...` 全 PASS。
- Handoff：`artifact:cli-binding`；脚本可用 `status --json` 直接消费。
