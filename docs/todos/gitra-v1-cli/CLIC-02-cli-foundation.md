# CLIC-02: CLI Foundation

- **Todo ID**: CLIC-02
- **Execution Map**: execution-map.json
- **Todo maturity**: DECOMPOSED
- **Task detail format**: v1
- **Source baseline**: docs/plans/gitra-v1-cli-plan.md（WP-CLI-02）；docs/gitra-v1-implementation-baseline.md；docs/gitra-v1-login-onboarding-design.md

## Outcome

建立 Cobra 命令树基座：root 命令、全局 flags、错误→退出码映射与 JSON 写出器，使后续命令只关注用例。

## Contract

- **Requirements**: `REQ-STEP-02`, `REQ-SCOPE-03`, `REQ-IF-01`, `REQ-IF-04`, `REQ-IF-07`, `REQ-WP-02`, `REQ-EXT-01`, `REQ-EXT-02`, `REQ-VER-01`
- **Produces**: `artifact:cli-foundation` — cli.New().Execute 基座、退出码映射、JSON/文本输出基元
- **Gates**: `go test ./internal/presentation/...`、`go test ./...`、`go vet ./...`
- **Boundaries**: CLI 只调用 Application 层；不直接访问 storage/git/ssh；不打印密钥/token。

## Tasks

- [ ] `T01` 实现 root 命令与退出码映射。
  - **Test and RED**: `internal/presentation/cli/root_test.go` 表驱动断言 `ExitCodeFor`（ErrInvalid→2、ErrAccountNotFound→3、ErrNotGitRepository→4、ErrAuthInvalid→5、ErrAlreadyBound→6、ErrUnsupportedRemote→7、其他→1）与 `Execute(ctx, []string{"nope"})`→2；实现前 RED。
  - **Execution logic**: 使用 cobra（SilenceUsage/SilenceErrors），`Execute` 返回 int；未知命令/未知 flag 归一为 `errUsage`（exit 2）；业务错误输出到 stderr（单行 + 可操作提示）；`--verbose` 为持久 flag。
  - **Files and responsibilities**: `internal/presentation/cli/root.go`（命令树与 Execute）、`exitcode.go`（映射表）、`errors.go`（errUsage）；测试只断言契约。
  - **Verification**: 正常：`--help`→0；非法：未知命令→2、缺参→2；不产生 stdout 噪音；不 panic。
  - **Artifact handoff**: 供 CLIC-03/04 挂载子命令。
- [ ] `T02` 实现 JSON 基座与 verbose 诊断。
  - **Test and RED**: 断言 `--json` 输出可解析且含 `schema_version: 1`，verbose 关闭时 stderr 安静、开启时有诊断行；实现前 RED。
  - **Execution logic**: 定义 `schemaVersion=1` 的 DTO 基座与 `writeJSON(w, v)`（缩进 2、尾随换行）；`--json` 为各查询子命令的局部 flag；verbose 通过 `log.New(stderr)` 仅打印操作名与路径（白名单，不含 secret）。
  - **Files and responsibilities**: `internal/presentation/cli/json.go`、`internal/presentation/cli/output.go`。
  - **Verification**: 正常：JSON 可被 encoding/json 解析；安全：输出中不出现 private_key 内容（只允许路径）；边界：空集合输出 `[]` 而非 null。
  - **Artifact handoff**: CLIC-03/04 的命令直接复用写出器。


## Tests

- 退出码矩阵与 JSON 契约断言。
- **RED**: 基座缺失时的编译/断言失败。


## Done

- [ ] Every task has RED/GREEN evidence and the node Gates pass.
- [ ] Artifact/evidence/status are updated in the execution map.
- [ ] Graph and semantic checks pass.

## STOP conditions

- 需要引入 Cobra 以外的 CLI 框架（与基线 §19.1 冲突）。
- 退出码映射需要偏离基线 §35。


## Evidence / Handoff

- RED：`go test ./internal/presentation/...` build failed（undefined: errUsage / ExitCodeFor / App / New / schemaVersion / WriteJSON）。
- GREEN：Cobra root（`gitra --help`、`gitra version`、`gitra --version`、未知命令/未知 flag→2）、`ExitCodeFor` 覆盖 0/1/2/3/4/5/6/7 全表、`WriteJSON`（缩进 2、禁用 HTML 转义、尾随换行、空切片输出 `[]`）、`--verbose` 惰性日志（默认静默）；`cmd/gitra/main.go` 改为装配 bootstrap + cli。
- Gates：`go test ./internal/presentation/...`、`go test ./...`、`go vet ./...` 全 PASS；gofmt 干净。
- Handoff：`artifact:cli-foundation` 提供命令挂载点、退出码映射与 JSON 写出器，CLIC-03/04 直接复用。
