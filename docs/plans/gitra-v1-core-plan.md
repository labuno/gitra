# gitra V1.0 Core 实施计划

> Plan ID：`gitra-v1-core`
> 范围：基线 Milestone 0（Skeleton）+ Milestone 1（Core Identity Binding）
> 目标产物：可验证的「多账号 + 每仓库绑定 + 原生 Git 身份隔离」核心
> 上游约束：`docs/gitra-v1-implementation-baseline.md`（修订 1）、`docs/gitra-architecture-development-design.md`、`docs/gitra-v1-login-onboarding-design.md`

---

## Objective

交付 gitra V1.0 core：Domain、Ports、Storage、Adapters、Strategies、BindingService、Reconciler 的可运行实现，并以「双账号双仓库」E2E 证明不同 Repository 在无 gitra 进程介入时使用互不串扰的 Git 身份与 SSH 凭据。

## Scope

- 覆盖基线 §45 Milestone 0 与 §46 Milestone 1 的全部范围。
- 只实现 `ssh-key` Auth Strategy 与 `repo-local` Routing Strategy 的 repo-local Git 配置投影。
- CLI、TUI、Provider 联网验证、登录、引导式克隆不在本 Plan，属于后续独立 Plan。
- 语言与工具链：Go（基线 §19）；git 与 ssh 一律通过统一 CommandRunner 调用。

## Non-goals

- 任何 CLI / TUI 呈现层（基线 Milestone 2 / 4）。
- Provider 联网验证（基线 Milestone 3 的 `account test`）。
- HTTPS 凭据、登录与新手引导（V1.1 / V1.2 增补设计，另行立项）。
- 数据库、daemon、Web、plugin、telemetry。
- 自动修改 remote URL、自动 clone、SSH key 生成或上传。

## Steps

1. Skeleton：模块、目录边界、Domain、Ports。
2. Storage：AccountStore / BindingStore + 原子写 + 进程间文件锁。
3. CommandRunner → GitCLIAdapter（DiscoverRepository / LocalConfig / Remotes）。
4. Strategies：ssh-key 与 repo-local 及其注册表。
5. Snapshot 与 Rollback。
6. BindingService：bind / unbind / status。
7. Reconciler 与 BindingHealth。
8. 双账号双仓库 E2E 与回归 Gate。

## Interfaces

- 端口签名以基线 §27（Git Port）、§16.3 / §16.4（Stores）为准，实现不得擅改签名。
- 错误模型以基线 §34 为准；exit code 冻结见基线 §35。
- 命名冻结（增补设计 §11）：module `github.com/zhanhd/gitra`、二进制 `gitra`、配置目录 `os.UserConfigDir()/gitra`、仓库元数据 `<GitDir>/gitra/state.json`、Git 配置段 `[gitra]`。

## Work Packages

### WP-01 Toolchain & Repository Bootstrap

go.mod、cmd/gitra、internal 目录骨架、.gitignore、构建与 vet/test Gate 基线。

### WP-02 Domain Model & Invariants

Account、AccountID、ProviderRef、ProviderEndpoint、CommitIdentity、TransportConfig、Repository、Remote、RepositoryBinding、BindingHealth、错误分类与校验不变式。

### WP-03 Ports & Boundaries

Git、SSH、CommandRunner、AccountStore、BindingStore、Locker、Clock 等接口；依赖方向约束（domain 不依赖 adapter/presentation）。

### WP-04 Git CLI Adapter

DiscoverRepository（RootPath / GitDir / bare / worktree 判定）、LocalConfig 读/写/删、Remotes 解析与端口语义。

### WP-05 Persistence & Locking

AccountStore / BindingStore 的 JSON 实现、原子写（temp+fsync+rename）、文件锁、schema_version。

### WP-06 Strategies (SSH Key / Repo-Local)

SSHKeyAuthStrategy（`core.sshCommand` 生成与端口规则）、RepoLocalRoutingStrategy（投影与恢复）、Auth / Routing 注册表。

### WP-07 Binding Lifecycle

Snapshot、Rollback、Bind 固定流程、Unbind 恢复、Status 查询。

### WP-08 Reconciliation & Health

Desired / Applied 对比、DRIFT 检测与修复、ReconcileBinding / ReconcileAccount。

### WP-09 Dual-Account E2E & Regression

双账号双仓库隔离用例、Rollback 回归、全量 `go test ./...` 与 `go vet ./...` Gate。

## Test Matrix

- Domain 单测（基线 §53.1）。
- Application 单测与 Fakes（基线 §53.2）。
- Git adapter 集成测试（基线 §53.3）。
- Rollback 测试（基线 §53.6）。
- 双账号双仓库 E2E（基线 §53.7）。

## Verification

- 每个节点：聚焦测试 + `go test ./...` + `go vet ./...`。
- M1 验收：双账号共存、双仓库绑定成功、identity 与 key 不串、Unbind 可恢复、失败可 rollback。

## Done

- 全部 Work Package 完成，执行图节点均为 `验证成功`，Artifact 与 Gate 证据入图。
- 无跨层依赖污染（domain 不 import adapter / presentation）。
- `git fetch / pull / push` 不需要 gitra 进程即可使用正确身份。

## STOP Conditions

- 基线文档与实现事实冲突，或需要未批准的公共语义变更。
- 源文档 digest 漂移、Requirement 缺失或语义复核失败。
- 测试基础设施无法安全建立（例如系统缺少 git 可执行文件）。
- 单节点五轮修复仍失败。

## External Sources

- Go 1.25（开发机 go1.25.1 darwin/arm64）。
- 系统 git（开发机 2.49.0）与 OpenSSH（开发机 9.9p2）。
- 平台：macOS 优先，Linux 兼容，Windows 保持可移植。
