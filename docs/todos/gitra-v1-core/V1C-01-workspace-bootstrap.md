# V1C-01: Toolchain & Repository Bootstrap

- **Todo ID**: V1C-01
- **Execution Map**: execution-map.json
- **Todo maturity**: DECOMPOSED
- **Task detail format**: v1
- **Source baseline**: docs/plans/gitra-v1-core-plan.md（WP-01）；docs/gitra-v1-implementation-baseline.md；docs/gitra-architecture-development-design.md

## Outcome

建立可构建、可测试的 Go module 骨架，使命名冻结（module `github.com/zhanhd/gitra`）与目录边界立即生效，后续节点可直接落实现。

## Contract

- **Requirements**: `REQ-WP-01`, `REQ-IF-04`, `REQ-EXT-01`, `REQ-VER-02`, `REQ-OBJ-01`, `REQ-SCOPE-01`, `REQ-SCOPE-02`, `REQ-STEP-01`
- **Produces**: `artifact:workspace` — 可构建的 Go module + cmd/gitra 入口 + internal 包骨架 + Gate 命令基线
- **Gates**: `go build ./...`、`go test ./...`、`go vet ./...`
- **Boundaries**: 不实现任何业务行为；不创建 CLI 命令树、不创建 TUI、不联网。

## Tasks

- [ ] `T01` 建立 Go module 与进程入口，使 `go build ./...` 成功并产出可执行的 `gitra`。
  - **Test and RED**: 运行 `go build ./...`；当前目录没有 go.mod，预期报 `go.mod file not found` 失败并记录输出。
  - **Execution logic**: 执行 `go mod init github.com/zhanhd/gitra`；`cmd/gitra/main.go` 仅解析 `--version` 占位并退出 0；不在此处装配具体 adapter（Composition Root 属于后续节点）。
  - **Files and responsibilities**: `go.mod` 固定 module 名；`cmd/gitra/main.go` 只承担进程入口与退出码；`internal/bootstrap/doc.go` 标注未来 Composition Root 位置。
  - **Verification**: 正常：`go build ./...` 与 `go run ./cmd/gitra --version` 成功；边界：重复执行 bootstrap 幂等，不重写已有 go.mod；非法：module 名与冻结值不一致时人工纠正后重跑 Gate。
  - **Artifact handoff**: 为 `artifact:workspace` 提供 module 路径与入口；WP-02 起所有包在此 module 内新增。
- [ ] `T02` 建立 internal 包骨架与依赖方向占位，使后续每个 Work Package 有唯一落点。
  - **Test and RED**: 骨架建立前 `go vet ./...` 无包可检查（或报错）；记录实际输出作为 RED 证据。
  - **Execution logic**: 为 `internal/domain`、`internal/ports`、`internal/app`、`internal/adapters/gitcli`、`internal/adapters/sshcli`、`internal/adapters/storage`、`internal/strategies/auth`、`internal/strategies/routing` 各创建 `doc.go`（仅 package 声明 + 一行职责注释）。
  - **Files and responsibilities**: 每个 `doc.go` 只声明包与边界注释；不写业务类型（由 WP-02/WP-03 负责）。
  - **Verification**: 正常：`go build ./...`、`go vet ./...`、`go test ./...` 通过；非法：grep 确认 `internal/domain` 未 import `internal/adapters` 或 `internal/app`（依赖方向）；边界：空包不得包含未使用 import。
  - **Artifact handoff**: 固化包路径清单，供 WP-02..WP-09 直接使用。
- [ ] `T03` 初始化仓库并建立卫生基线（.gitignore、README 导航）。
  - **Test and RED**: `git status` 当前报 `not a git repository`（预期 RED）；初始化后必须成功返回。
  - **Execution logic**: `git init`；写 `.gitignore`（`/bin/`、`dist/`、`*.test`、`.DS_Store`）；写 `README.md` 一句话定位与 `docs/` 导航；不做任何提交（提交由用户决定）。
  - **Files and responsibilities**: `.gitignore` 只忽略构建产物与系统文件；`README.md` 只做导航，不复制设计内容。
  - **Verification**: 正常：`git status` 可运行且工作区可见；边界：`git check-ignore bin/gitra` 返回忽略；安全：README 与 .gitignore 不得包含密钥、token 或本机绝对路径。
  - **Artifact handoff**: 仓库可支撑后续节点的 diff 与证据采集。


## Tests

- 正常：module 构建通过，`cmd/gitra` 可执行。
- 边界：bootstrap 重复执行幂等；空包 vet 通过。
- 非法：module 名偏离冻结值时由人工检查拦截。
- **RED**: 无 go.mod 时 `go build ./...` 必然失败。


## Done

- [ ] Every task has RED/GREEN evidence and the node Gates pass.
- [ ] Artifact/evidence/status are updated in the execution map.
- [ ] Graph and semantic checks pass.

## STOP conditions

- 环境缺少 Go 工具链或版本低于 1.25。
- module 名或目录冻结值与增补设计 §11 冲突。


## Evidence / Handoff

- RED（2026-09-22）：`go build ./...` → `pattern ./...: directory prefix . does not contain main module or its selected dependencies`；`git status` → `fatal: not a git repository`。
- GREEN：`go mod init github.com/zhanhd/gitra`；新增 `cmd/gitra/main.go`（`--version` 输出 `gitra dev`，无参数退出码 2）与 `internal/...` 九个 `doc.go` 骨架；`git init` + `.gitignore` + `README.md`。
- Gates：`go build ./...` PASS、`go test ./...` PASS（10 个包，无测试文件）、`go vet ./...` PASS。
- Handoff：module 路径 `github.com/zhanhd/gitra` 与包路径清单已稳定，V1C-02 可直接在 `internal/domain` 落实现。
