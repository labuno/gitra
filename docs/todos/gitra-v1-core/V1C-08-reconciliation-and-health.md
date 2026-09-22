# V1C-08: Reconciliation & Health

- **Todo ID**: V1C-08
- **Execution Map**: execution-map.json
- **Todo maturity**: DECOMPOSED
- **Task detail format**: v1
- **Source baseline**: docs/plans/gitra-v1-core-plan.md（WP-08）；docs/gitra-v1-implementation-baseline.md；docs/gitra-architecture-development-design.md

## Outcome

实现 Desired/Applied 对比与修复：DRIFT 检测、ReconcileBinding 与 ReconcileAccount，使账号变更可传播到所有绑定仓库。

## Contract

- **Requirements**: `REQ-WP-08`, `REQ-STEP-04`, `REQ-DONE-04`
- **Produces**: `artifact:reconciler` — Reconciler（比较/修复/健康度）与 account 级传播
- **Gates**: `go test ./internal/app/...`、`go test ./...`、`go vet ./...`
- **Boundaries**: 只修改受管白名单 key；missing/broken 不自动删除记录；不联网。

## Tasks

- [ ] `T01` 实现 Desired/Applied 比较与健康度计算。
  - **Test and RED**: `reconcile_test.go` 覆盖 ok/drift/missing/broken 四态真值表；实现前 RED。
  - **Execution logic**: 依据增补设计附录 A.4 真值表：central 存在+路径存在+元数据匹配+配置一致=ok；配置被改=drift；路径不存在=missing；元数据缺失/不匹配=broken；无 central 且有可识别元数据=自恢复路径。
  - **Files and responsibilities**: `internal/app/health.go` 与其测试；纯逻辑，无 I/O（输入为读取结果）。
  - **Verification**: 正常/边界（无元数据、无 central）/冲突（元数据与 central 不一致）全覆盖；非法：health 枚举外值拒绝。
  - **Artifact handoff**: Status（WP-07）与 Reconcile 共用同一判定函数。
- [ ] `T02` 实现 ReconcileBinding：漂移修复与幂等。
  - **Test and RED**: 人为修改 user.email 后 Reconcile 恢复期望值；重复执行无变化（幂等）；missing 路径返回健康度而不报错；实现前 RED。
  - **Execution logic**: 读账号→构建 desired→比较→仅对差异 key 写入→读回验证→返回 `ReconcileResult{Changed, Health}`；写入在文件锁内；失败返回错误且不留下半写入。
  - **Files and responsibilities**: `internal/app/reconcile.go` 与测试；复用 WP-06 条目生成与 WP-07 的恢复原语。
  - **Verification**: 正常：drift 修复；边界：已一致时 Changed=false；非法：账号缺 field；状态：broken 时不尝试修复并报告原因。
  - **Artifact handoff**: ReconcileAccount 与后续 CLI `reconcile` 命令消费。
- [ ] `T03` 实现 ReconcileAccount：账号变更传播到全部绑定。
  - **Test and RED**: Fake store 返回 3 个绑定（2 个存在、1 个 missing），断言仅存在的被更新且结果汇总；实现前 RED。
  - **Execution logic**: ListByAccount→逐个 ReconcileBinding→聚合 Change 计数与健康度；单仓库失败不中断其余（记录失败并在结果中呈现）。
  - **Files and responsibilities**: `internal/app/reconcile_account.go` 与测试。
  - **Verification**: 正常：多仓库传播；边界：0 绑定；错误：单仓库失败隔离；并发：整体在写锁内串行。
  - **Artifact handoff**: 供账号编辑流程（后续 CLI Plan）在 Account.Update 后调用。


## Tests

- 健康度真值表（含 self-heal 分支）。
- 幂等与部分失败隔离。
- **RED**: 比较逻辑缺失时的失败断言。


## Done

- [ ] Every task has RED/GREEN evidence and the node Gates pass.
- [ ] Artifact/evidence/status are updated in the execution map.
- [ ] Graph and semantic checks pass.

## STOP conditions

- 真值表与增补设计附录 A.4 冲突。
- 需要删除记录或写非受管 key 才能修复（禁止）。


## Evidence / Handoff

- RED：`go test ./internal/app/...` build failed（undefined: NewReconciler）。
- GREEN：`app/reconcile.go`（ReconcileBinding：漂移修复 + 幂等 + 仓库移动时跟随新路径并递增 Revision；missing 路径报 HealthMissing 不报错；ReconcileAccount：逐绑定传播、单仓库失败隔离并汇总）。
- Gates：`go test ./internal/app/...`、`go test ./...`、`go vet ./...` 全 PASS。
- Handoff：`artifact:reconciler` 供账号编辑流程与 V1C-09 回归使用。
