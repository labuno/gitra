# VER-03: CLI account test & Regression

- **Todo ID**: VER-03
- **Execution Map**: execution-map.json
- **Todo maturity**: DECOMPOSED
- **Task detail format**: v1
- **Source baseline**: docs/plans/gitra-v1-verify-plan.md（WP-VERIFY-03）；docs/gitra-v1-implementation-baseline.md §4.6/§16.2/§48

## Outcome

提供 `gitra account test <alias> [--json]` 与离线回归，锁定退出码与 JSON 契约。

## Contract

- **Requirements**: `REQ-OBJ-01`, `REQ-SCOPE-04`, `REQ-STEP-03`, `REQ-IF-03`, `REQ-WP-03`, `REQ-TEST-04`, `REQ-VER-01`, `REQ-DONE-01`, `REQ-DONE-02`
- **Produces**: `artifact:verify-cli` — account test 命令与离线回归
- **Gates**: `go test ./internal/presentation/...`、`go test ./...`、`go vet ./...`
- **Boundaries**: 不新增网络能力；输出不包含 token 或私钥内容。

## Tasks

- [ ] `T01` 实现 `gitra account test <alias> [--json]`。
  - **Test and RED**: CLI 测试覆盖成功（exit 0）、失败/不匹配（exit 5，提示可操作动作）、未知 alias（exit 3）、缺参（exit 2）与 JSON 形状；实现前 RED。
  - **Execution logic**: 解析 alias → 调 VerificationService → 文本输出 `provider/host: username (ok)` 或失败原因；JSON 输出 `{schema_version, account_id, success, expected_username, actual_username, message}`；`--verbose` 追加 Raw。
  - **Files and responsibilities**: `internal/presentation/cli/account_test_cmd.go`（挂到 account 命令）与测试。
  - **Verification**: 四类退出码 + JSON 字段断言；失败输出必须给出下一步动作。
  - **Artifact handoff**: TUI 的 T 键与脚本/Agent 直接使用。
- [ ] `T02` 离线回归：httptest provider 与 ssh fixture 端到端。
  - **Test and RED**: `internal/e2e/verify_flow_test.go`：登录（httptest）后 `account test` 成功；token 被撤销（httptest 401）后 test 失败 exit 5；SSH 账号用 fake ssh runner + 真实 provider fixture 断言解析路径；实现前 RED。
  - **Execution logic**: 复用 V1.1 的隔离测试基础设施；SSH 路径注入 fake ports.SSH 与真实解析器。
  - **Files and responsibilities**: 该测试文件负责端到端断言。
  - **Verification**: 正常/失败两条端到端链路；断言输出不含 token。
  - **Artifact handoff**: 长期回归，支撑 M4 TUI 的 T 动作。


## Tests

- CLI 集成与端到端回归（全离线）。
- **RED**: 命令未实现或结果分类缺失。


## Done

- [ ] Every task has RED/GREEN evidence and the node Gates pass.
- [ ] Artifact/evidence/status are updated in the execution map.
- [ ] Graph and semantic checks pass.

## STOP conditions

- 需要真实网络/账号。
- 五轮修复仍失败。


## Evidence / Handoff

- GREEN：`gitra account test <alias> [--json] [--verbose]`（文本与 JSON：schema_version/account_id/status/success/expected_username/actual_username/message[/raw]）；退出码 0 成功、5 认证类失败、1 环境类失败、3 未知 alias、2 缺参；顺手补齐了 cobra 参数校验错误的退出码归类（`isUsageError`）。
- 回归：CLI 用例覆盖 HTTPS 四类与 SSH 两类结果；`internal/e2e/verify_flow_test.go` 覆盖「登录 → test 成功 → 凭据被撤销（401）→ test 报 authentication_failed」且输出无 token。
- Gates：`go test ./internal/presentation/...`、`go test ./...`、`go vet ./...` 全 PASS。
- Handoff：`artifact:verify-cli` 完成 M3；TUI 的 Test Connection 将复用 `VerificationService`。
