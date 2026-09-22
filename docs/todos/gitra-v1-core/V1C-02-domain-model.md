# V1C-02: Domain Model & Invariants

- **Todo ID**: V1C-02
- **Execution Map**: execution-map.json
- **Todo maturity**: DECOMPOSED
- **Task detail format**: v1
- **Source baseline**: docs/plans/gitra-v1-core-plan.md（WP-02）；docs/gitra-v1-implementation-baseline.md；docs/gitra-architecture-development-design.md

## Outcome

实现与框架无关的领域模型与校验不变式：Account/Provider/Identity/Transport、Repository/Remote、RepositoryBinding、BindingHealth 与错误分类。

## Contract

- **Requirements**: `REQ-WP-02`, `REQ-IF-03`, `REQ-IF-08`, `REQ-TEST-01`
- **Produces**: `artifact:domain-model` — 可被 app/adapters 依赖的 domain 包（类型、不变式、typed errors）
- **Gates**: `go test ./internal/domain/...`、`go test ./...`、`go vet ./...`
- **Boundaries**: domain 不 import 任何 adapter/presentation/os/exec；不做 I/O、不读环境变量。

## Tasks

- [ ] `T01` 定义 Account 聚合与校验不变式（AccountID、Alias、ProviderRef/Endpoint、CommitIdentity、TransportConfig）。
  - **Test and RED**: 新增 `internal/domain/account_test.go` 表驱动用例（ID 前缀、alias 空白/超长/非法字符、host 空值、SSH 端口 0/1/65535/65536、transport strategy 白名单）；此刻类型不存在导致编译失败，即预期 RED。
  - **Execution logic**: 定义值类型；`Validate()` 顺序为 ID → Alias → Provider(type/host/username) → Endpoint(host/ssh user/port) → Identity(name/email 基本格式) → Transport(strategy 白名单 + 必填 config)；第一个失败即返回对应 typed error，不聚合多错。
  - **Files and responsibilities**: `internal/domain/account.go` 承担类型与校验；`internal/domain/errors.go` 承担 `ErrAccountNotFound`/`ErrAccountExists`/`ErrAuthInvalid` 等 sentinel error；`account_test.go` 承担正常/边界/非法矩阵。
  - **Verification**: 正常：合法账号通过；边界：端口 1 与 65535 合法、0/65536 非法；非法：空 alias、非法字符、未知 strategy；并发不适用（纯函数）；无 I/O 副作用。
  - **Artifact handoff**: `artifact:domain-model` 提供 Account 及其校验；WP-05 存储层与 WP-06 Strategy 直接复用，不得复制校验逻辑。
- [ ] `T02` 定义 Repository、Remote、RepositoryBinding 与 BindingHealth 不变式。
  - **Test and RED**: 新增 `internal/domain/binding_test.go` 覆盖 root/gitdir 必需、remote 名唯一、binding 必含 AccountID 与 canonical path、health 取值枚举；编译失败即为 RED。
  - **Execution logic**: Repository 必须同时持有 RootPath 与 GitDir（禁止假设 `.git` 为目录）；Binding 只引用 AccountID 与 canonical path；Health 为 `ok|drift|missing|broken` 四值枚举并提供 `String()`；非法枚举值返回 typed error。
  - **Files and responsibilities**: `internal/domain/repository.go`（Repository/Remote）、`internal/domain/binding.go`（Binding/Health）、对应测试文件；错误复用 `errors.go`。
  - **Verification**: 正常：合法 binding 通过；边界：Remote 列表为空允许、名称重复拒绝；非法：health 非法值、缺 AccountID；状态冲突：同一 path 出现两个 binding 时由上层（WP-05/WP-07）拒绝，domain 提供 `SamePath` 判定辅助。
  - **Artifact handoff**: 供 WP-05 持久化、WP-07 生命周期与 WP-08 健康度直接消费。
- [ ] `T03` 定义 ProviderEndpoint 默认值与 port 语义辅助（含 `-p` 规则所需的端口判定）。
  - **Test and RED**: `internal/domain/provider_test.go` 断言 GitHub/GitLab/Gitea 默认 endpoint 与 `IsDefaultSSHPort()`；未实现时 RED。
  - **Execution logic**: 提供默认值构造器与 `SSHPortRequiresFlag(remoteHasExplicitPort bool)` 之类的纯函数，把基线 §14 修订规则编码为可测语义：URL 自带端口时不追加 `-p`。
  - **Files and responsibilities**: `internal/domain/provider.go` 与测试；不得依赖网络或硬编码用户路径。
  - **Verification**: 正常：github.com/git/22；边界：自建 host + 2222；非法：空 host；不可变：默认值函数不修改入参。
  - **Artifact handoff**: WP-06 生成 `core.sshCommand` 时消费该语义，避免端口规则在 adapter 层重复实现。


## Tests

- 正常/边界/非法三类表驱动覆盖 Account 与 Binding 校验。
- 状态冲突：health 枚举、重复 remote 名。
- **RED**: 类型与校验函数不存在时的编译失败。


## Done

- [ ] Every task has RED/GREEN evidence and the node Gates pass.
- [ ] Artifact/evidence/status are updated in the execution map.
- [ ] Graph and semantic checks pass.

## STOP conditions

- 领域校验规则与基线 §4.1/§4.3/§13 冲突。
- 需要引入 domain 不应感知的实现细节（os/exec、文件路径）。


## Evidence / Handoff

- RED：三个测试文件先行，`go test ./internal/domain/...` 因 `undefined: Account` 等类型缺失而 build failed。
- GREEN：新增 `errors.go`（sentinel + ValidationError）、`account.go`、`provider.go`（含 §14 端口纯函数）、`repository.go`、`binding.go`；18 个 Account 用例、6 个 Repository 用例、6 个 Binding 用例、健康度枚举与默认 endpoint 全部通过。
- Gates：`go test ./internal/domain/...` PASS、`go test ./...` PASS、`go vet ./...` PASS；gofmt 干净。
- Handoff：`artifact:domain-model` 提供 Account/Provider/Endpoint/Identity/Transport/Repository/Remote/RepositoryRef/Binding/Health 与 typed errors；V1C-03 在此基础上定义 ports 接口。
