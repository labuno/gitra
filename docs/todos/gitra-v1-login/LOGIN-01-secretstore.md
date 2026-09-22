# LOGIN-01: SecretStore & Git Credential Adapter

- **Todo ID**: LOGIN-01
- **Execution Map**: execution-map.json
- **Todo maturity**: DECOMPOSED
- **Task detail format**: v1
- **Source baseline**: docs/plans/gitra-v1-login-plan.md（WP-LOGIN-01）；docs/gitra-v1-login-onboarding-design.md；docs/gitra-v1-implementation-baseline.md

## Outcome

提供基于 git 原生 credential 机制的凭据存取端口实现与 helper 解析，保证 token 只进系统凭据库。

## Contract

- **Requirements**: `REQ-OBJ-01`, `REQ-SCOPE-05`, `REQ-STEP-01`, `REQ-IF-01`, `REQ-IF-02`, `REQ-WP-01`, `REQ-TEST-01`, `REQ-STOP-01`, `REQ-EXT-02`
- **Produces**: `artifact:secretstore` — ports.SecretStore 的 git-credential 实现与 helper 解析
- **Gates**: `go test ./internal/adapters/secretstore/...`、`go test ./...`、`go vet ./...`
- **Boundaries**: 不写入 gitra JSON 配置；不打印 secret；不修改用户全局 git 配置。

## Tasks

- [ ] `T01` 实现 git credential 读写：Set→`git credential approve`、Get→`git credential fill`、Delete→`git credential reject`。
  - **Test and RED**: `internal/adapters/secretstore/gitcred_test.go` 用 fake runner 断言 argv、stdin（protocol/host/username/password）与 ref 解析；实现前 RED。
  - **Execution logic**: ref 形如 `<host>/<username>`；Set 以 `protocol=https
host=...
username=...
password=...

` 写入 stdin 并调用 approve；Get 用同样协议头调用 fill 并解析输出，未命中返回 `domain.ErrAuthInvalid`；Delete 调用 reject 且幂等。
  - **Files and responsibilities**: `internal/adapters/secretstore/gitcred.go`（封装 + ref 解析）、`gitcred_test.go`（命令序列与解析）。
  - **Verification**: 正常：approve/fill/reject 往返；边界：Get 未命中；非法：ref 缺少 host；安全：token 只经 stdin，不出现在 argv。
  - **Artifact handoff**: 供 LOGIN-03 的策略校验与 LOGIN-04 的登录写入。
- [ ] `T02` 实现 helper 解析与降级链：全局 helper → 平台原生 helper（macOS osxkeychain）→ 受管文件库。
  - **Test and RED**: `helper_test.go` 三分支表驱动（fake runner 返回不同 `git config --get-all credential.helper` / `git --exec-path`）；实现前 RED。
  - **Execution logic**: 优先沿用用户既有全局 helper；否则若 `git --exec-path` 下存在 `git-credential-osxkeychain` 则使用 `osxkeychain`；否则使用 `store --file=<configDir>/credentials` 并把文件权限设为 0600（目录 0700）；解析结果只用于「需要写入 repo-local helper 时」与本站读写。
  - **Files and responsibilities**: `internal/adapters/secretstore/helper.go`、`helper_test.go`；`NewGitCredentialStore(runner, helper)` 支持显式注入以便测试。
  - **Verification**: 三分支命中；集成测试用真实 git + `store --file=<tmp>` 完成 approve→fill→reject 往返；断言未写全局配置、未写 JSON。
  - **Artifact handoff**: LOGIN-03 依据该解析决定是否写 repo-local `credential.helper`。


## Tests

- fake runner 命令序列 + 真实 git 文件库往返（隔离路径）。
- **RED**: 实现缺失时的编译失败。


## Done

- [ ] Every task has RED/GREEN evidence and the node Gates pass.
- [ ] Artifact/evidence/status are updated in the execution map.
- [ ] Graph and semantic checks pass.

## STOP conditions

- 测试必须写真实系统凭据库或联网。
- 需要修改用户全局 git 配置才能工作。


## Evidence / Handoff

- RED：`go test ./internal/adapters/secretstore/...` build failed（undefined: NewGitCredentialStore / ResolveHelper）。
- GREEN：`gitcred.go`（approve/fill/reject，token 只走 stdin；ref=`host/username`；未命中→ErrAuthInvalid；reject 幂等）+ `helper.go`（全局 helper → 平台 osxkeychain → 受管 `store --file=<configDir>/credentials`，文件 0600、目录 0700）。
- Port 增量：`ports.CommandRunner` 新增 `RunWithInput`（token 不得出现在 argv）；runner adapter、ports fake 与既有测试同步更新。
- Gates：`go test ./internal/adapters/secretstore/...`、`go test ./...`、`go vet ./...` 全 PASS（含真实 git + `store --file=<tmp>` 的往返集成测试）。
- Handoff：`artifact:secretstore` 供 LOGIN-03/04 与 TUI 使用。
