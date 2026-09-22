# V1C-05: Persistence & Locking

- **Todo ID**: V1C-05
- **Execution Map**: execution-map.json
- **Todo maturity**: DECOMPOSED
- **Task detail format**: v1
- **Source baseline**: docs/plans/gitra-v1-core-plan.md（WP-05）；docs/gitra-v1-implementation-baseline.md；docs/gitra-architecture-development-design.md

## Outcome

实现 AccountStore 与 BindingStore 的 JSON 持久化、原子写与进程间文件锁，保证配置目录与 schema_version 冻结值生效。

## Contract

- **Requirements**: `REQ-WP-05`, `REQ-SCOPE-04`, `REQ-IF-02`, `REQ-TEST-02`
- **Produces**: `artifact:stores` — accounts/<acc_id>.json + bindings.json 的原子读写实现与 FileLocker
- **Gates**: `go test ./internal/adapters/storage/...`、`go test ./...`、`go vet ./...`
- **Boundaries**: 写入路径只来自配置目录解析；不得把数据写入仓库目录；不得在存储层执行业务校验之外的逻辑。

## Tasks

- [ ] `T01` 实现配置目录解析与原子写原语。
  - **Test and RED**: `internal/adapters/storage/fs_test.go` 断言 `os.UserConfigDir()/gitra`、`GITRA_CONFIG_DIR` 覆盖、temp+fsync+rename 后内容完整；实现前 RED。
  - **Execution logic**: 目录解析优先级：`GITRA_CONFIG_DIR` → `os.UserConfigDir()/gitra`；原子写：同目录 temp 文件 → fsync → rename 覆盖；失败时清理 temp；权限 0600。
  - **Files and responsibilities**: `internal/adapters/storage/fs.go`（路径解析 + 原子写）、`fs_test.go`；不含账号/绑定语义。
  - **Verification**: 正常：写入后可读；边界：目录不存在时创建；非法：目标目录不可写时返回可解释错误；安全：文件权限 0600，临时文件不残留。
  - **Artifact handoff**: 供 T02/T03 复用；WP-07 的 snapshot 也可复用原子写原语。
- [ ] `T02` 实现 AccountStore：按 ID 命名的 JSON 文件、List/Get/Delete、别名唯一性检查。
  - **Test and RED**: `account_store_test.go` 断言 `acc_xxx.json` 命名、schema_version=1、按 alias 查找唯一、重复 alias 报 `ErrAccountExists`；实现前 RED。
  - **Execution logic**: Save 前校验 alias 唯一（扫描目录）；文件内容包含 schema_version 与完整账号 JSON；Delete 幂等返回 `ErrAccountNotFound`；List 按 ID 排序保证稳定输出。
  - **Files and responsibilities**: `internal/adapters/storage/account_store.go` 与其测试；错误使用 domain sentinel。
  - **Verification**: 正常：增删查改；边界：空目录；非法：损坏 JSON 报错且不删原文件；冲突：并发写由 T03 的锁保护。
  - **Artifact handoff**: `artifact:stores` 提供账号持久化；WP-07 与后续 CLI Plan 直接消费。
- [ ] `T03` 实现 BindingStore 与 FileLocker（跨进程写锁）。
  - **Test and RED**: `binding_store_test.go` 断言 bindings.json 读写、FindByRepository 按 canonical path、Delete；`locker_test.go` 用两个 goroutine/子进程验证互斥与超时；实现前 RED。
  - **Execution logic**: BindingStore 单文件数组 + schema_version；FileLocker 基于 `flock`/`LOCK_EX`（unix）封装为 `WithWriteLock(ctx, fn)`，支持 ctx 超时；锁文件位于配置目录 `.lock`。
  - **Files and responsibilities**: `internal/adapters/storage/binding_store.go`、`internal/adapters/storage/locker_unix.go`（+ 后续平台文件）、对应测试。
  - **Verification**: 正常：写锁内完成写操作；边界：锁超时返回可解释错误；非法：锁文件不可创建；并发：两进程同时写 bindings.json 不丢失记录（子进程测试）。
  - **Artifact handoff**: WP-07 的 Bind/Unbind/Reconcile 全部在 `WithWriteLock` 内执行。


## Tests

- 应用/存储单测与 Fake 对照（基线 §53.2）。
- 并发：锁互斥与超时；原子写中断不产生半文件。
- **RED**: 原子写与锁实现前的失败断言。


## Done

- [ ] Every task has RED/GREEN evidence and the node Gates pass.
- [ ] Artifact/evidence/status are updated in the execution map.
- [ ] Graph and semantic checks pass.

## STOP conditions

- 需要引入数据库或外部锁服务（超出基线 §20 依赖约束）。
- 配置目录策略与增补设计附录 A 冲突。


## Evidence / Handoff

- RED：`go test ./internal/adapters/storage/...` build failed（undefined: ConfigDir / WriteFileAtomic / NewAccountStore / NewBindingStore / NewFileLocker）。
- GREEN：`fs.go`（GITRA_CONFIG_DIR 覆盖、temp+fsync+rename 原子写、0600）、`account_store.go`（按 ID 命名、alias 唯一、损坏文件不删除）、`binding_store.go`（bindings.json + schema_version + 同 path 拒绝）、`locker_unix.go`（flock 排他 + ctx 超时）+ `locker_other.go`（非 unix 明确失败）。
- Gates：`go test ./internal/adapters/storage/...`、`go test ./...`、`go vet ./...` 全 PASS。
- Handoff：`artifact:stores` 提供 accounts/bindings 持久化与跨进程写锁，V1C-07 全部写路径在 `WithWriteLock` 内执行。
