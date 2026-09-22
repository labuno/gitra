# V1C-09: Dual-Account E2E & Regression

- **Todo ID**: V1C-09
- **Execution Map**: execution-map.json
- **Todo maturity**: DECOMPOSED
- **Task detail format**: v1
- **Source baseline**: docs/plans/gitra-v1-core-plan.md（WP-09）；docs/gitra-v1-implementation-baseline.md；docs/gitra-architecture-development-design.md

## Outcome

以真实 git 仓库证明核心命题：Repo A→Account A、Repo B→Account B 身份与密钥完全隔离，Unbind 可恢复，失败可回滚，且 git 操作不依赖 gitra 进程。

## Contract

- **Requirements**: `REQ-WP-09`, `REQ-TEST-05`, `REQ-VER-01`, `REQ-VER-02`, `REQ-OBJ-01`, `REQ-OBJ-02`, `REQ-SCOPE-01`, `REQ-DONE-03`, `REQ-STOP-02`, `REQ-DONE-01`
- **Produces**: `artifact:isolation-e2e` — 双账号双仓库隔离开环测试 + 全量回归 Gate
- **Gates**: `go test ./...`、`go vet ./...`
- **Boundaries**: E2E 使用临时目录与 `GITRA_CONFIG_DIR` 隔离，不触碰真实用户配置；不联网。

## Tasks

- [ ] `T01` 建立双账号双仓库 E2E：绑定后断言身份与密钥隔离。
  - **Test and RED**: `internal/e2e/isolation_test.go`（`testing.Short` 跳过或 build tag 控制）在临时目录创建两个仓库与两个账号（不同 SSH key 路径占位），执行 bind 后断言 `git -C repo-a config --local user.email` 与 `core.sshCommand` 各自正确、互不相同；实现前 RED。
  - **Execution logic**: 测试自建环境：`t.TempDir()` + `GITRA_CONFIG_DIR` 指向临时目录；通过真实 BindingService + 真实 Git adapter 执行；断言还包含：全局 gitconfig 未被改动（记录前后 digest）。
  - **Files and responsibilities**: `internal/e2e/isolation_test.go` 拥有环境搭建与断言；不得 mock git。
  - **Verification**: 正常：两仓库各自生效；边界：第三仓库未绑定（配置不出现 gitra 键）；非法：HTTPS remote 绑定失败但仓库其他配置不变；安全：不写入用户真实配置目录。
  - **Artifact handoff**: `artifact:isolation-e2e` 成为长期回归用例（基线 §53.7）。
- [ ] `T02` 回归：Rollback 与 Unbind 恢复在 E2E 层面成立。
  - **Test and RED**: 扩展 `isolation_test.go`：注入 central store 写失败断言 bind 全量回滚；unbind 后断言三个受管键恢复为绑定前值且 `gitra.*` 消失；实现前 RED。
  - **Execution logic**: 复用故障注入 Fake 包装真实 store；断言 git 配置逐 key 相等而非字符串包含。
  - **Files and responsibilities**: `internal/e2e/rollback_e2e_test.go` 与辅助断言函数。
  - **Verification**: 正常：解绑恢复；错误：注入失败点后无残留；状态：再次绑定成功（幂等恢复后）。
  - **Artifact handoff**: 满足基线 §53.6/§53.7 的回归义务。
- [ ] `T03` 全量 Gate 与交接：`go test ./...`、`go vet ./...` 证据入图，更新节点状态与后继 readiness。
  - **Test and RED**: 在实现完成前 Gate 失败（缺包/缺实现），作为 RED 记录；完成后必须全绿。
  - **Execution logic**: 顺序执行 `go build ./...`、`go test ./...`、`go vet ./...`；把命令、结果、scope、时间写入 execution-map 的 evidence 与 integration_gates；同步所有节点 status/readiness 并重新运行校验器。
  - **Files and responsibilities**: `docs/todos/gitra-v1-core/execution-map.json` 只承载状态与证据；不修改既有任务拆解。
  - **Verification**: 正常：三条 Gate 全 PASS；无效：任一失败则节点保持验证失败并记录原因；一致性：`validate_todo_graph.py` 输出 graph/trace/semantic 三项 PASS。
  - **Artifact handoff**: 完成后 Plan 级 Done 条件（WP-01..WP-09 全绿）可被评审。


## Tests

- 双账号双仓库隔离（基线 §53.7，永久保留）。
- 回滚与恢复的 E2E 版本。
- **RED**: E2E 未实现时无法绑定两个账号。


## Done

- [ ] Every task has RED/GREEN evidence and the node Gates pass.
- [ ] Artifact/evidence/status are updated in the execution map.
- [ ] Graph and semantic checks pass.

## STOP conditions

- 需要真实网络或外部账号才能完成断言（应改为纯本地断言）。
- Gate 在五轮内仍失败。


## Evidence / Handoff

- RED：E2E 首次运行失败（全局 git config 读取在未设置时返回非零被误判为致命错误），修正测试辅助函数后继续。
- GREEN：`internal/e2e/isolation_test.go` 三个用例全部通过——双账号双仓库隔离（identity 与 core.sshCommand 各自正确且互不相同）、remote URL 未被改写、未绑定仓库无受管键、全局 git config 前后一致、Unbind 逐键恢复、注入 central store 失败后仓库完整回滚且快照删除。
- Gates：`go test ./...`（全部 13 个包）与 `go vet ./...` 全 PASS；gofmt 干净。
- Handoff：`artifact:isolation-e2e` 作为永久回归用例；Milestone 1 的 DoD（双账号共存、身份不串、可恢复、可回滚、git 运行时不依赖 gitra 进程）全部有可执行证据。
