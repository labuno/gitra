# CLIC-01: Bootstrap & AccountService

- **Todo ID**: CLIC-01
- **Execution Map**: execution-map.json
- **Todo maturity**: DECOMPOSED
- **Task detail format**: v1
- **Source baseline**: docs/plans/gitra-v1-cli-plan.md（WP-CLI-01）；docs/gitra-v1-implementation-baseline.md；docs/gitra-v1-login-onboarding-design.md

## Outcome

提供可复用的 Composition Root 与 AccountService（含账号守卫与 update 后 Reconcile 传播），为 CLI 提供唯一业务入口。

## Contract

- **Requirements**: `REQ-OBJ-01`, `REQ-SCOPE-01`, `REQ-SCOPE-04`, `REQ-STEP-01`, `REQ-STEP-05`, `REQ-IF-01`, `REQ-IF-05`, `REQ-WP-01`, `REQ-TEST-01`, `REQ-TEST-04`, `REQ-STOP-01`
- **Produces**: `artifact:account-service` — bootstrap.New() 组合根 + AccountService（Create/Update/Delete/Get/GetByAlias/List）
- **Gates**: `go test ./internal/app/...`、`go test ./...`、`go vet ./...`
- **Boundaries**: 不新增业务语义到显示层；AccountService 只依赖 ports 与既有 Deps；不做网络访问。

## Tasks

- [ ] `T01` 实现 bootstrap 组合根：装配 config dir、stores、snapshot store、file locker、runner+git adapter、auth/routing 注册表与 App 聚合。
  - **Test and RED**: 新增 `internal/bootstrap/bootstrap_test.go`，断言 `New()` 返回非空 Deps/Accounts/Bindings/Reconciler 且 locker 可获取写锁；实现前 `undefined: New` 即预期 RED。
  - **Execution logic**: `storage.ConfigDir()` → `NewAccountStore/NewBindingStore/NewSnapshotStore/NewFileLocker` → `gitcli.New(runner.New())` → 注册 `sshkey.New()` 与 `repolocal.New()` → 组装 `app.Deps` → 构造 `app.NewBindingService/NewAccountService/NewReconciler`。
  - **Files and responsibilities**: `internal/bootstrap/app.go` 只做装配；`bootstrap_test.go` 只断言装配结果；不得包含业务分支。
  - **Verification**: 正常：`GITRA_CONFIG_DIR` 指向临时目录时 New() 成功；边界：连续两次 New() 互不共享状态；非法：配置目录不可写时返回可解释错误；安全：不读取用户真实配置目录。
  - **Artifact handoff**: 提供 `bootstrap.App{Deps, Accounts, Bindings, Reconciler}`，CLIC-02..05 全部通过它获取服务。
- [ ] `T02` 实现 AccountService：Create/Update/Delete/Get/GetByAlias/List 与守卫。
  - **Test and RED**: 新增 `internal/app/account_service_test.go`（复用现有 fakes）覆盖创建、非法输入、别名重复、Update 传播 drift、无绑定删除、有绑定删除被拒、UnbindAll 删除；实现前 RED。
  - **Execution logic**: Create 生成 `acc_` ID、Revision=1、先 Validate 再 Save；Update 固定 ID、Revision+1、Save 后调用 `Reconciler.ReconcileAccount`，存在失败时返回 `%w: ErrStateDrift` 包装（账号已保存）；Delete 先 `ListByAccount`，有绑定且未 `UnbindAll` 返回 `ErrAccountInUse`，`UnbindAll` 时逐仓库 `Unbind` 后删除；GetByAlias 遍历 List 匹配 alias。
  - **Files and responsibilities**: `internal/app/account_service.go` 承担用例编排；测试文件承担守卫矩阵；ID 生成复用 `newRandomID`。
  - **Verification**: 正常/非法/冲突/边界（0 绑定、无匹配 alias）全覆盖；并发由 Locker 在写路径保护；不引入新依赖。
  - **Artifact handoff**: CLI 的 account 子命令只调用该服务；`GetByAlias` 同时服务 bind 的 alias 解析。


## Tests

- app 单测覆盖 Create/Update/Delete 守卫与 reconcile 传播（基线 §53.2 的 Fake 手法）。
- **RED**: 服务与组合根不存在时的编译失败。


## Done

- [ ] Every task has RED/GREEN evidence and the node Gates pass.
- [ ] Artifact/evidence/status are updated in the execution map.
- [ ] Graph and semantic checks pass.

## STOP conditions

- 需要修改既有 Application 公共语义才能实现。
- bootstrap 需要读取真实用户配置目录才能通过测试。


## Evidence / Handoff

- RED：`go test ./internal/app/... ./internal/bootstrap/...` build failed（undefined: AccountService / NewAccountService / CreateAccountRequest …）。
- GREEN：`internal/bootstrap/app.go`（config dir → stores/snapshots/locker/runner/git/registries → app.Deps → BindingService/AccountService/Reconciler）；`internal/app/account_service.go`（Create 校验+锁内保存、Update 递增 Revision 后传播 reconcile 并以 ErrStateDrift 报告失败、Delete 的 ErrAccountInUse 守卫与 UnbindAll 路径、GetByAlias 支持 alias 与 acc_ ID）。
- Gates：`go test ./internal/app/...`、`go test ./internal/bootstrap/...`、`go test ./...`、`go vet ./...` 全 PASS；gofmt 干净。
- Handoff：`artifact:account-service` 提供 `bootstrap.App{Deps, Accounts, Bindings, Reconciler}`，CLIC-02 起所有命令通过它访问服务。

## Amendment (2026-09-22)

- CLIC-05 期间补充：`Create`/`Update` 现在会调用 `validateLocalAuth`（auth strategy 的离线校验），确保 `--key` 指向不存在文件时返回 `ErrAuthInvalid`（exit 5）。行为边界与原合同一致（§5.1 validate / §46 本地配置无效），仅补齐离线校验，不改变公共语义。
