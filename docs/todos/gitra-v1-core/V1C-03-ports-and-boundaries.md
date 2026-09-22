# V1C-03: Ports & Boundaries

- **Todo ID**: V1C-03
- **Execution Map**: execution-map.json
- **Todo maturity**: DECOMPOSED
- **Task detail format**: v1
- **Source baseline**: docs/plans/gitra-v1-core-plan.md（WP-03）；docs/gitra-v1-implementation-baseline.md；docs/gitra-architecture-development-design.md

## Outcome

以接口形式固化跨层契约：Git、SSH、CommandRunner、AccountStore、BindingStore、Locker、Clock，并给出依赖方向的可执行约束。

## Contract

- **Requirements**: `REQ-WP-03`, `REQ-IF-01`, `REQ-IF-02`, `REQ-DONE-02`
- **Produces**: `artifact:ports` — ports 包（接口集 + ProcessResult 值类型）与依赖方向检查
- **Gates**: `go test ./internal/ports/...`、`go test ./...`、`go vet ./...`
- **Boundaries**: ports 只依赖 domain 与标准库；不得 import adapters/app/presentation。

## Tasks

- [ ] `T01` 定义 Git、SSH、CommandRunner 三个出站端口与 ProcessResult。
  - **Test and RED**: `internal/ports/git_test.go` 用 fake 实现断言接口可满足；接口不存在时编译失败即 RED。
  - **Execution logic**: 方法签名逐字对齐基线 §27（DiscoverRepository/GetLocalConfig/SetLocalConfig/UnsetLocalConfig/Remotes）；SSH 端口只暴露 `Test(ctx, req) (SSHTestResult, error)`；CommandRunner 返回 `ProcessResult{ExitCode, Stdout, Stderr}`，错误语义为「进程未能执行」而非「退出码非零」。
  - **Files and responsibilities**: `internal/ports/git.go`、`internal/ports/ssh.go`、`internal/ports/runner.go`、`internal/ports/ports_test.go`（编译期 fake 断言）。
  - **Verification**: 正常：fake 满足接口；边界：退出码非零仍返回结果 + nil error（由调用方判定）；非法：缺 ctx 不编译；安全：接口不得传递明文密钥内容（只传路径）。
  - **Artifact handoff**: WP-04 实现 runner+git，WP-06 实现 ssh 相关策略消费方。
- [ ] `T02` 定义 AccountStore、BindingStore、Locker、Clock 端口。
  - **Test and RED**: `internal/ports/stores_test.go` 编译期断言；缺接口时 RED。
  - **Execution logic**: Store 读方法返回 `(*T, error)` 或 `(T, error)`，Find 类方法用 `bool` 或 sentinel error 表达未找到（与基线 §34 一致：`ErrBindingNotFound`）；Locker 暴露 `WithWriteLock(ctx, fn)`；Clock 仅暴露 `Now()` 便于测试。
  - **Files and responsibilities**: `internal/ports/account_store.go`、`binding_store.go`、`locker.go`、`clock.go` 与断言测试。
  - **Verification**: 正常：fake 全满足；边界：未找到语义明确（不返回 nil,nil）；非法：方法签名与基线不一致即视为失败；并发：接口文档注明调用方在写路径外层持锁。
  - **Artifact handoff**: WP-05 提供 JSON 实现，WP-07 通过接口编排。
- [ ] `T03` 建立依赖方向检查：domain/app/ports 不得 import adapters 或 presentation。
  - **Test and RED**: 新增 `internal/ports/boundary_test.go`（或 `tools` 下的静态检查测试）扫描 import 图；当前无实现时 RED。
  - **Execution logic**: 用 `go list -deps` 或解析源码 import 的方式，断言 `internal/domain` 与 `internal/ports` 的依赖闭包不含 `internal/adapters`、`internal/presentation`；失败即 t.Fatal 并列出违规包。
  - **Files and responsibilities**: 该测试是唯一职责（边界守护），不得复制业务断言。
  - **Verification**: 正常：当前包通过；非法：人为加入违规 import 后测试必须失败（用手工临时验证后回滚）；安全：不修改源码。
  - **Artifact handoff**: 成为 WP-09 全量回归的一部分，防止后续节点引入跨层污染。


## Tests

- 正常：编译期 fake 断言全部通过。
- 非法：退出码语义错误（把非零退出码映射为 error）必须被评审拒绝。
- **RED**: 接口缺失导致的编译失败。


## Done

- [ ] Every task has RED/GREEN evidence and the node Gates pass.
- [ ] Artifact/evidence/status are updated in the execution map.
- [ ] Graph and semantic checks pass.

## STOP conditions

- 基线端口签名与现有 domain 类型不兼容。
- 依赖方向检查无法在无外部依赖的前提下实现。


## Evidence / Handoff

- RED：`go test ./internal/ports/...` 因 `undefined: Git / SSH / CommandRunner / AccountStore / BindingStore / Locker / Clock / ProcessResult` 而 build failed。
- GREEN：新增 `git.go`、`ssh.go`、`runner.go`、`stores.go`、`locker.go`、`clock.go`；签名逐字对齐基线 §27 / §16.3 / §16.4；`boundary_test.go` 用 `go list` 强制 domain/ports 不得依赖 adapters/presentation/bootstrap。
- Gates：`go test ./internal/ports/...` PASS、`go test ./...` PASS、`go vet ./...` PASS；gofmt 干净。
- Handoff：`artifact:ports` 已冻结；V1C-04（Git adapter）、V1C-05（Storage）、V1C-06（Strategies）可并行启动，均只依赖本节点接口。
