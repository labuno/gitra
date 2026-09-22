# VER-02: VerificationService

- **Todo ID**: VER-02
- **Execution Map**: execution-map.json
- **Todo maturity**: DECOMPOSED
- **Task detail format**: v1
- **Source baseline**: docs/plans/gitra-v1-verify-plan.md（WP-VERIFY-02）；docs/gitra-v1-implementation-baseline.md §4.6/§16.2/§48

## Outcome

提供统一验证用例：HTTPS 走 Provider API、SSH 走 ssh -T + Provider 解析，输出六类可区分的结果。

## Contract

- **Requirements**: `REQ-OBJ-01`, `REQ-SCOPE-03`, `REQ-STEP-02`, `REQ-WP-02`, `REQ-TEST-03`, `REQ-DONE-01`, `REQ-EXT-02`
- **Produces**: `artifact:verification-service` — VerificationService 与 VerificationResult
- **Gates**: `go test ./internal/app/...`、`go test ./...`、`go vet ./...`
- **Boundaries**: 不联网除显式调用；不在首页自动触发；不打印 token。

## Tasks

- [ ] `T01` 实现 HTTPS 验证路径：读取凭据 → Profile API → 比对用户名。
  - **Test and RED**: `internal/app/verification_test.go` 用 fake ProfileProvider/fake SecretStore 覆盖成功、用户名不匹配、token 失效（401）、Provider 不可用（5xx/网络）四类；实现前 RED。
  - **Execution logic**: 凭据缺失 → `Success=false, Message="no stored credential; run gitra login"`；profile 调用错误按 `ErrAuthInvalid` 与其他错误分类到 Message；用户名比较用 EqualFold；结果包含 Expected/Actual。
  - **Files and responsibilities**: `internal/app/verification_service.go`（两路径统一入口）与测试。
  - **Verification**: 四类 HTTPS 结果 + 本地配置无效（key 缺失走 SSH 分支）各一例。
  - **Artifact handoff**: CLI 与 TUI 的 Test Connection 复用。
- [ ] `T02` 实现 SSH 验证路径：ssh -T → Provider 解析 → 比对。
  - **Test and RED**: 断言成功（解析出用户名且匹配）、解析失败（未知响应 → 明确提示并附原始输出）、用户名不匹配、进程失败（超时/缺 ssh）四类；实现前 RED。
  - **Execution logic**: 由账号策略取 key 路径与 endpoint；构造 SSHTestRequest 调用 ports.SSH；用 ports.SSHIdentityParser 解析；解析失败时保留原始 stdout/stderr 到 `Raw` 供 --verbose 展示；不把 exit code 直接当失败。
  - **Files and responsibilities**: 同文件 SSH 分支 + 测试；接口定义在 ports。
  - **Verification**: 四类 SSH 结果；`Raw` 不含私钥材料。
  - **Artifact handoff**: VER-03 命令与 TUI 的 T 动作消费。


## Tests

- 服务单测覆盖六类结果（本地无效/成功/不匹配/认证失败/网络进程失败/无法识别）。
- **RED**: 服务缺失时的编译失败。


## Done

- [ ] Every task has RED/GREEN evidence and the node Gates pass.
- [ ] Artifact/evidence/status are updated in the execution map.
- [ ] Graph and semantic checks pass.

## STOP conditions

- 需要真实账号或网络。
- 需要修改既有 Application 公共语义。


## Evidence / Handoff

- GREEN：`app.VerificationService`（按策略分流；六类状态 `ok / local_config_invalid / authentication_failed / username_mismatch / provider_unavailable / unrecognized_response`；HTTPS 走 Profile API、SSH 走 ssh -T + Provider 解析；`Raw` 保留原始 ssh 输出供 --verbose）；`Deps.SSH` 端口接入。
- 测试：HTTPS 五类（成功/不匹配/401/5xx/无凭据）与 SSH 五类（成功/不识别/不匹配/进程失败/本地 key 缺失）全覆盖。
- Gates：`go test ./internal/app/...`、`go test ./...`、`go vet ./...` 全 PASS。
- Handoff：`artifact:verification-service` 供 VER-03 与 TUI 的 T 动作使用。
