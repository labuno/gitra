# V1C-06: Strategies (SSH Key / Repo-Local)

- **Todo ID**: V1C-06
- **Execution Map**: execution-map.json
- **Todo maturity**: DECOMPOSED
- **Task detail format**: v1
- **Source baseline**: docs/plans/gitra-v1-core-plan.md（WP-06）；docs/gitra-v1-implementation-baseline.md；docs/gitra-architecture-development-design.md

## Outcome

实现 ssh-key Auth Strategy 与 repo-local Routing Strategy 及其注册表，使账号凭据可确定性投影为 repo-local Git 配置。

## Contract

- **Requirements**: `REQ-WP-06`, `REQ-SCOPE-03`, `REQ-IF-06`, `REQ-IF-07`, `REQ-IF-01`, `REQ-VER-03`
- **Produces**: `artifact:strategies` — AuthStrategy/RoutingStrategy 注册表与两个 V1 实现
- **Gates**: `go test ./internal/strategies/...`、`go test ./...`、`go vet ./...`
- **Boundaries**: Strategy 只通过 ports 使用 Git/SSH；不直接读写文件；不生成或上传密钥。

## Tasks

- [ ] `T01` 实现 SSHKeyAuthStrategy：校验 key path 并生成受管配置条目。
  - **Test and RED**: `internal/strategies/auth/sshkey/sshkey_test.go` 断言生成 `core.sshCommand = ssh -i "<abs-path>" -o IdentitiesOnly=yes`、URL 自带端口时不追加 `-p`、scp-like 且端口非 22 时追加 `-p`；实现前 RED。
  - **Execution logic**: 输入 Account.Transport.config.private_key；路径 `~` 展开为绝对路径并校验存在与权限；端口规则消费 domain 的纯函数；输出 `[]GitConfigEntry{Key, Value}` 顺序稳定。
  - **Files and responsibilities**: `internal/strategies/auth/sshkey/sshkey.go` 只负责映射与校验；测试覆盖规则矩阵；注册表见 T03。
  - **Verification**: 正常：默认端口；边界：路径含空格（引号）、非 22 端口、URL 带端口；非法：key 不存在报 `ErrAuthInvalid`；安全：不读取私钥内容。
  - **Artifact handoff**: WP-07 通过注册表获取条目并写入 repo-local 配置。
- [ ] `T02` 实现 RepoLocalRoutingStrategy：投影、恢复与检查。
  - **Test and RED**: `internal/strategies/routing/repolocal/repolocal_test.go` 用 FakeGit 断言 Bind 写入的 key 集合恰好等于受管白名单（user.name/user.email/core.sshCommand/gitra.*）、Unbind 恢复快照、Inspect 报告 drift；实现前 RED。
  - **Execution logic**: Bind 接收 Repository + Account + 条目列表，按顺序写 local config；Unbind 依据 snapshot 恢复（快照接口由 WP-07 传入或通过 port 读取）；Inspect 读取实际值并与期望比较，返回 `RoutingStatus{state, diff}`。
  - **Files and responsibilities**: `internal/strategies/routing/repolocal/repolocal.go` 与测试；不得实现快照文件格式（由 WP-07 拥有）。
  - **Verification**: 正常：首次绑定；边界：已存在相同值时幂等；非法：写入失败向上返回并保持部分写入可回滚（由 WP-07 编排）；状态：人为改 user.email 后 Inspect=drift。
  - **Artifact handoff**: WP-08 Reconciler 复用 Inspect 判定健康度。
- [ ] `T03` 实现 Auth/Routing 注册表与 bootstrap 装配点。
  - **Test and RED**: `registry_test.go` 断言按 ID 取用、未知 ID 返回错误、注册重复报错；实现前 RED。
  - **Execution logic**: 简单 map + `Register/Get`；bootstrap 只在此装配 `ssh-key` 与 `repo-local`；业务层禁止 if/else 分支。
  - **Files and responsibilities**: `internal/strategies/auth/registry.go`、`internal/strategies/routing/registry.go`、`internal/bootstrap/strategies.go`。
  - **Verification**: 正常：取到两个实现；非法：未知 ID；边界：并发只读安全（注册发生在启动期）。
  - **Artifact handoff**: 后续 HTTPS/其他 routing 只新增实现并注册，核心零修改。


## Tests

- 规则矩阵：端口与引号（正常/边界/非法）。
- 状态：drift 检测与幂等投影。
- **RED**: 策略未实现时的断言失败。


## Done

- [ ] Every task has RED/GREEN evidence and the node Gates pass.
- [ ] Artifact/evidence/status are updated in the execution map.
- [ ] Graph and semantic checks pass.

## STOP conditions

- 端口规则与基线 §14 修订冲突。
- 需要修改 ports 签名才能实现（应先走设计变更）。


## Evidence / Handoff

- RED：`go test ./internal/strategies/...` build failed（undefined: Strategy / NewRegistry / ErrUnknownStrategy / New / BuildRequest）。
- GREEN：`auth` 与 `routing` 注册表（重复注册拒绝、未知 ID 报错）；`sshkey`（key 存在性校验、路径含空格加引号、§14 端口规则消费 domain 纯函数，只产出 core.sshCommand）；`repolocal`（按序写入、drift 检测、快照恢复置值/删除）。
- 决策：共享值类型 `ports.GitConfigEntry` 存放于 ports，避免 strategy 之间的横向依赖（增量，不改既有签名）。
- Gates：`go test ./internal/strategies/...`、`go test ./...`、`go vet ./...` 全 PASS。
- Handoff：`artifact:strategies` 供 V1C-07 生成 desired 状态并在应用层组合 user.name/user.email/gitra.* 。
