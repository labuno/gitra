# CLIC-05: CLI End-to-End Regression

- **Todo ID**: CLIC-05
- **Execution Map**: execution-map.json
- **Todo maturity**: DECOMPOSED
- **Task detail format**: v1
- **Source baseline**: docs/plans/gitra-v1-cli-plan.md（WP-CLI-05）；docs/gitra-v1-implementation-baseline.md；docs/gitra-v1-login-onboarding-design.md

## Outcome

以脚本化回归证明 CLI 可完成基线 §47 的验收流程，并锁定退出码矩阵，作为长期回归用例。

## Contract

- **Requirements**: `REQ-TEST-03`, `REQ-VER-02`, `REQ-DONE-01`, `REQ-DONE-02`, `REQ-STOP-02`, `REQ-WP-05`, `REQ-OBJ-01`, `REQ-VER-01`
- **Produces**: `artifact:cli-e2e` — CLI 全流程回归与退出码矩阵
- **Gates**: `go test ./...`、`go vet ./...`
- **Boundaries**: 全部离线、隔离配置目录；不做真实 fetch/push。

## Tasks

- [ ] `T01` 实现全流程 E2E：add → bind → status → edit → status → unbind → remove。
  - **Test and RED**: `internal/e2e/cli_flow_test.go` 依次执行 CLI 并断言：edit 后仓库内 `user.email` 变化、unbind 后恢复原值、remove 后账号消失、每步 exit code；实现前 RED。
  - **Execution logic**: 测试内构造 `bootstrap.App`（`GITRA_CONFIG_DIR` 指向 t.TempDir()）与临时仓库，通过 `cli.New(...).Execute` 驱动，全部断言基于 stdout JSON 与真实 git config。
  - **Files and responsibilities**: 该测试文件承担端到端断言；不重复单元测试细节。
  - **Verification**: 正常全流程；边界：未绑定仓库 status；安全：断言未改动全局 gitconfig，且未写入用户真实配置目录。
  - **Artifact handoff**: 长期回归用例（对应基线 §47 验收流程）。
- [ ] `T02` 实现退出码矩阵回归。
  - **Test and RED**: 表驱动执行：成功 0、缺参 2、未知 alias 3、非仓库 4、无效 key 5、已绑定 6、HTTPS remote 7；实现前 RED。
  - **Execution logic**: 每个用例独立临时环境，断言 exit code 与 stderr 关键短语（如「supports SSH remotes only」）。
  - **Files and responsibilities**: 同文件内独立 TestExitCodeMatrix。
  - **Verification**: 七类退出码逐一命中；不依赖网络。
  - **Artifact handoff**: 供后续 M3/M4/M2.5 复用同一隔离测试基础设施。


## Tests

- 脚本化 E2E 与退出码矩阵（离线）。
- **RED**: 端到端流程无法完成或退出码不符。


## Done

- [ ] Every task has RED/GREEN evidence and the node Gates pass.
- [ ] Artifact/evidence/status are updated in the execution map.
- [ ] Graph and semantic checks pass.

## STOP conditions

- 需要真实远端/账号才能通过。
- 五轮修复仍失败。


## Evidence / Handoff

- RED：E2E 文件首次运行即失败于 CLI 命令缺失（与 CLIC-03/04 RED 同源）。
- GREEN：`internal/e2e/cli_flow_test.go` 两个用例通过——全流程（add → bind → status --json → edit（reconcile 生效）→ status → unbind（恢复）→ remove → list 为空）与退出码矩阵（0/2/3/4/5/6/7 逐一命中）。
- 补充改造：`AccountService.Create/Update` 增加离线本地认证校验（调用 auth strategy 的 Validate），使 `--key` 指向不存在文件时报 `ErrAuthInvalid`（exit 5）。该行为符合基线 §5.1「validate」与 §46 的本地配置无效分类；已记入 CLIC-01 Todo 的修订说明。
- Gates：`go test ./...`、`go vet ./...` 全 PASS；gofmt 干净。
- Handoff：`artifact:cli-e2e` 永久保留；M2.5/M3/M4 复用同一隔离测试基础设施。
