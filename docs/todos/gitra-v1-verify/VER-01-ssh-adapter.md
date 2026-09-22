# VER-01: SSH Adapter & Provider SSH Parsing

- **Todo ID**: VER-01
- **Execution Map**: execution-map.json
- **Todo maturity**: DECOMPOSED
- **Task detail format**: v1
- **Source baseline**: docs/plans/gitra-v1-verify-plan.md（WP-VERIFY-01）；docs/gitra-v1-implementation-baseline.md §4.6/§16.2/§48

## Outcome

提供真实 ssh 调用（原始结果透传）与三家 Provider 的 `ssh -T` 响应解析，使「认证成功」语义由 Provider 决定。

## Contract

- **Requirements**: `REQ-OBJ-01`, `REQ-SCOPE-01`, `REQ-SCOPE-02`, `REQ-STEP-01`, `REQ-IF-01`, `REQ-IF-02`, `REQ-IF-04`, `REQ-WP-01`, `REQ-TEST-01`, `REQ-TEST-02`, `REQ-STOP-01`, `REQ-STOP-02`, `REQ-EXT-01`
- **Produces**: `artifact:ssh-adapter` — sshcli adapter + 三家 Provider 的 ssh 响应解析
- **Gates**: `go test ./internal/adapters/...`、`go test ./...`、`go vet ./...`
- **Boundaries**: 不判断「认证成功」于 adapter 层；不打印私钥内容；不修改 ssh 配置。

## Tasks

- [ ] `T01` 实现 sshcli adapter：按基线 §17.2 构造 `ssh -T -i <key> -o IdentitiesOnly=yes [-p port] <user>@<host>` 并透传原始 exit code/stdout/stderr。
  - **Test and RED**: `internal/adapters/sshcli/ssh_test.go` 用 fake runner 断言 argv（默认端口不加 `-p`、非 22 加 `-p`）与原始结果透传；实现前 RED。
  - **Execution logic**: 组装参数列表；`-p` 仅在 Port 非 0 且非 22 时追加；调用 ports.CommandRunner；超时/缺二进制等执行错误原样返回（不吞掉）；退出码非 0 视为数据而非错误。
  - **Files and responsibilities**: `internal/adapters/sshcli/ssh.go`（实现 ports.SSH）与测试；不得包含用户名解析。
  - **Verification**: 正常/边界（自定义端口）/非法（缺 key 路径）；安全：argv 只含路径，不含私钥内容。
  - **Artifact handoff**: VER-02 通过 ports.SSH 调用。
- [ ] `T02` 实现三家 Provider 的 SSH 响应解析（fixture 驱动）。
  - **Test and RED**: 各 provider 包新增 `sshparse_test.go`，fixture 覆盖成功（含 GitHub/GitLab/Gitea 真实文案）、`Permission denied (publickey).`、空输出与未知文案；实现前 RED。
  - **Execution logic**: 使用锚定正则从 stdout+stderr 提取用户名；仅在匹配成功时返回 ok=true；未知响应返回 ("", false) 且不 panic。
  - **Files and responsibilities**: `internal/adapters/provider/{github,gitlab,gitea}/sshparse.go` 与测试；router 暴露统一入口 `ParseSSHIdentity`。
  - **Verification**: 成功/失败/未知三类 fixture；大小写与 `@` 前缀差异；不误判 `Permission denied` 为成功。
  - **Artifact handoff**: VER-02 依据解析结果比对期望用户名。


## Tests

- sshcli argv 与原始结果透传单测；Provider fixture 表驱动。
- **RED**: 实现缺失时的编译失败。


## Done

- [ ] Every task has RED/GREEN evidence and the node Gates pass.
- [ ] Artifact/evidence/status are updated in the execution map.
- [ ] Graph and semantic checks pass.

## STOP conditions

- 需要真实网络/真实 GPU 之外的账号才能测试。
- 需要把 Provider 语义判断放进 adapter 之外的层。


## Evidence / Handoff

- RED：`go test ./internal/adapters/...` build failed（undefined: New / ParseSSHVerification）。
- GREEN：`sshcli`（`-T -i <key> -o IdentitiesOnly=yes [-p port] user@host`，原始 exit code/stdout/stderr 透传，执行失败才是 error）；GitHub/GitLab/Gitea 三个 `ParseSSHVerification`（锚定正则，Permission denied 与未知文案判为不成功）；`ports.SSHIdentityParser` 与 router 统一入口。
- Gates：`go test ./internal/adapters/...`、`go test ./...`、`go vet ./...` 全 PASS。
- Handoff：`artifact:ssh-adapter` 供 VER-02 使用。
