# V1C-07: Binding Lifecycle (Snapshot / Bind / Unbind / Status)

- **Todo ID**: V1C-07
- **Execution Map**: execution-map.json
- **Todo maturity**: DECOMPOSED
- **Task detail format**: v1
- **Source baseline**: docs/plans/gitra-v1-core-plan.md（WP-07）；docs/gitra-v1-implementation-baseline.md；docs/gitra-architecture-development-design.md

## Outcome

实现具备原子语义的绑定生命周期：快照、受管配置投影、失败回滚、解绑恢复与状态查询。

## Contract

- **Requirements**: `REQ-WP-07`, `REQ-STEP-02`, `REQ-STEP-03`, `REQ-SCOPE-02`, `REQ-SCOPE-05`, `REQ-SCOPE-06`, `REQ-TEST-02`, `REQ-TEST-04`, `REQ-DONE-01`, `REQ-STOP-01`, `REQ-IF-07`
- **Produces**: `artifact:binding-service` — BindingService（bind/unbind/status）+ Snapshot/Rollback
- **Gates**: `go test ./internal/app/...`、`go test ./...`、`go vet ./...`
- **Boundaries**: 不修改 remote URL；不 clone；不做网络访问；所有写路径在文件锁内。

## Tasks

- [ ] `T01` 实现 Snapshot：记录受管 key 的绑定前状态到 `<GitDir>/gitra/state.json`。
  - **Test and RED**: `internal/app/snapshot_test.go` 断言 exists/value 语义、schema_version、重复绑定拒绝；实现前 RED。
  - **Execution logic**: 读取受管白名单元数据（存在性与值），序列化为 `{schema_version, binding_id, previous}`；写入使用原子写；若已存在 snapshot 且 bindingId 不同则返回 `ErrAlreadyBound`。
  - **Files and responsibilities**: `internal/app/snapshot.go` 与测试；文件格式归 WP-07 所有（基线 §11）。
  - **Verification**: 正常：三个 key 的存在/缺失矩阵；边界：空仓库（全不存在）；非法：GitDir 不可写；安全：不记录密钥内容。
  - **Artifact handoff**: Unbind 与 Bootstrapping 恢复均消费该快照。
- [ ] `T02` 实现 Bind 流程与回滚：从校验到提交的固定顺序，任何一步失败恢复现场。
  - **Test and RED**: `binding_service_test.go` 用 FakeGitStore 注入失败点（第 5 步写配置失败、第 7 步 Store 失败）断言最终 repo 配置恢复原状且 central store 无残留；实现前 RED。
  - **Execution logic**: 顺序：解析 Alias→加载账号→发现仓库→canonical 化→读 remotes→校验 SSH/Provider 兼容→校验账号完整性→检查既有绑定→snapshot→构建 desired→写 local config→写 repo 元数据→写 central binding→读回校验→提交；任一步失败执行 rollback（恢复 snapshot、删除部分元数据与部分 binding）。
  - **Files and responsibilities**: `internal/app/binding_service.go`（编排）、`internal/app/rollback.go`（补偿动作）、测试用故障注入。
  - **Verification**: 正常：端到端 bind；非法：HTTPS remote（V1.0 core 报错）、非仓库、账号不存在；冲突：已绑定报 `ErrAlreadyBound`；回滚：每个注入点都必须恢复；安全：不修改 remote。
  - **Artifact handoff**: `artifact:binding-service` 提供 Bind；WP-08 与 WP-09 消费。
- [ ] `T03` 实现 Unbind 与 Status。
  - **Test and RED**: 断言 Unbind 后 user.name/email/core.sshCommand 恢复绑定前值、`gitra.*` 键消失、central binding 删除；Status 对未绑定仓库返回 bound=false 且 exit 0；实现前 RED。
  - **Execution logic**: Unbind：发现仓库→定位 binding→读 snapshot→恢复→删 repo 元数据→删 central→校验恢复；Status：读取 central 与 repo 元数据并计算健康度（缺失/漂移/损坏）。
  - **Files and responsibilities**: `internal/app/unbind.go`、`internal/app/status.go` 与测试；健康度判定逻辑与 WP-08 共用函数。
  - **Verification**: 正常：绑定后解绑；边界：未绑定仓库 Status；非法：bucket 元数据缺失；状态：快照丢失时返回 `broken` 并保留 central 记录不删（避免数据丢失）。
  - **Artifact handoff**: Status 输出结构供后续 CLI `--json` 直接映射。


## Tests

- 回滚：至少三个注入点（基线 §53.6）。
- 正常/边界/非法/冲突矩阵覆盖 Bind 与 Unbind。
- **RED**: 生命周期未实现时的失败断言。


## Done

- [ ] Every task has RED/GREEN evidence and the node Gates pass.
- [ ] Artifact/evidence/status are updated in the execution map.
- [ ] Graph and semantic checks pass.

## STOP conditions

- 实现与源文档冲突（例如需要修改 remote URL 才能完成绑定）。
- 回滚无法保证时不得宣称完成。


## Evidence / Handoff

- RED：`go test ./internal/app/...` build failed（undefined: ports.Snapshot / BindingService / Deps / BindRequest …）。
- GREEN：`ports.Snapshot`/`SnapshotStore`、`storage.SnapshotStore`（<GitDir>/gitra/state.json 原子写）、`app.health`（A.4 真值表）、`app.ids`、`app.binding_service`（19 步固定流程：校验→快照→投影→元数据→central→读回校验；任一步失败按序回滚；Unbind 恢复快照并清理残留受管键；Status 输出 bound/health/drift）。
- 分层调整：`ParseRemoteURL` 由 gitcli 迁至 `domain`，以维持 app 不得 import adapter 的依赖方向（V1C-04 handoff 相应调整，行为不变、测试保留）。
- Gates：`go test ./internal/app/...`、`go test ./...`、`go vet ./...` 全 PASS。
- Handoff：`artifact:binding-service` 提供 Bind/Unbind/Status，供 V1C-08 与后续 CLI 直接消费。
