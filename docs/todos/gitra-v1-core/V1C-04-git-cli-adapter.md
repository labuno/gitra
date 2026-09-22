# V1C-04: Git CLI Adapter

- **Todo ID**: V1C-04
- **Execution Map**: execution-map.json
- **Todo maturity**: DECOMPOSED
- **Task detail format**: v1
- **Source baseline**: docs/plans/gitra-v1-core-plan.md（WP-04）；docs/gitra-v1-implementation-baseline.md；docs/gitra-architecture-development-design.md

## Outcome

以统一 CommandRunner 实现 GitCLIAdapter：仓库发现（root/gitdir/bare/worktree）、local config 读写删、remotes 解析，并保证 canonical path 语义。

## Contract

- **Requirements**: `REQ-WP-04`, `REQ-IF-01`, `REQ-IF-05`, `REQ-IF-09`, `REQ-TEST-03`, `REQ-EXT-02`
- **Produces**: `artifact:git-adapter` — GitCLIAdapter（真实 git 进程）+ CommandRunner 统一执行器
- **Gates**: `go test ./internal/adapters/...`、`go test ./...`、`go vet ./...`
- **Boundaries**: 不自行解析 `.git/config` 文件；不硬编码 `.git` 目录假设；不做任何写操作之外的 Git 变更。

## Tasks

- [ ] `T01` 实现 CommandRunner（os/exec）并覆盖超时、取消、退出码语义。
  - **Test and RED**: `internal/adapters/runner/runner_test.go` 覆盖 stdout/stderr/退出码/超时；实现前 RED。
  - **Execution logic**: `exec.CommandContext`；退出码非零时返回 `ProcessResult{ExitCode: n}` 且 error 为 nil（除进程未启动/被杀）；超时通过 ctx 取消并映射为可识别错误；不使用 shell 拼接。
  - **Files and responsibilities**: `internal/adapters/runner/runner.go` 只承担进程执行；`runner_test.go` 覆盖语义；实现 ports.CommandRunner。
  - **Verification**: 正常：`echo` 类命令；边界：退出码 1/127、空 stderr；错误：不存在的可执行文件；安全：参数以 argv 传递，不经过 shell。
  - **Artifact handoff**: WP-04 T02/T03 与 WP-06 SSH 调用共用该 runner。
- [ ] `T02` 实现 DiscoverRepository：root、gitdir、bare 与 worktree 判定。
  - **Test and RED**: `internal/adapters/gitcli/git_test.go` 在临时目录 `git init` 后断言 RootPath/GitDir；实现前 RED。
  - **Execution logic**: 依次调用 `git -C <path> rev-parse --show-toplevel`、`--absolute-git-dir`、`--is-bare-repository`、`--git-common-dir`；bare 或 git-common-dir != absolute-git-dir（linked worktree）返回 `ErrUnsupportedRepo`；非仓库返回 `ErrNotGitRepository`；输入路径先 `~` 展开再做 git 判定，最终路径以 git 输出为准。
  - **Files and responsibilities**: `internal/adapters/gitcli/discover.go`（发现与分类）、`internal/adapters/gitcli/git.go`（adapter 装配）；测试使用真实 git。
  - **Verification**: 正常：普通仓库（含从子目录调用）；边界：路径带符号链接时返回物理路径；非法：非仓库目录；不支持：bare 与 linked worktree 必须明确报错而不是静默接受。
  - **Artifact handoff**: 为 WP-07 提供 canonical root/gitdir，为 WP-05 元数据落点提供 GitDir。
- [ ] `T03` 实现 LocalConfig 读/写/删与 Remotes 解析（含端口语义输出）。
  - **Test and RED**: 断言 `GetLocalConfig` 未设置时返回 `("", false, nil)`、设置后可读回、Unset 后消失；`Remotes` 解析 `git@host:owner/repo.git` 与 `ssh://git@host:2222/owner/repo.git`；实现前 RED。
  - **Execution logic**: 通过 `git config --local --get/--add`、`git remote` 或用 `git config --get-regexp ^remote\..*\.url$`；禁止直接读文件；解析结果包含 host 与显式端口（供 §14 规则判断）。
  - **Files and responsibilities**: `internal/adapters/gitcli/config.go`、`internal/adapters/gitcli/remotes.go` 与各自测试。
  - **Verification**: 正常：单/多 remote；边界：无 remote、URL 带端口、大小写与尾随 `.git`；非法：空 key；安全：写操作只允许白名单 key（本任务只做通用能力，白名单由 WP-07 强制）。
  - **Artifact handoff**: WP-06 routing 与 WP-07 binding 直接消费配置与 remote 信息。


## Tests

- 集成：临时仓库上的 root/gitdir/config/remotes（基线 §53.3）。
- 边界：子目录调用、符号链接、无 remote、端口 URL。
- **RED**: adapter 未实现时的失败断言。


## Done

- [ ] Every task has RED/GREEN evidence and the node Gates pass.
- [ ] Artifact/evidence/status are updated in the execution map.
- [ ] Graph and semantic checks pass.

## STOP conditions

- 环境缺少 git 可执行文件（基线 STOP：测试基础设施无法建立）。
- 发现需要解析配置文件才能满足语义（违背设计约束）。


## Evidence / Handoff

- RED：`go test ./internal/adapters/...` build failed（undefined: New / NewRunner / ParseRemoteURL）。
- GREEN：`runner`（os/exec，退出码为数据）+ `gitcli`（Discovery 顺序：is-bare → show-toplevel → absolute-git-dir → git-common-dir 判 worktree；config get/set/unset 幂等；remotes 排序稳定；scp-like 与 ssh:// 解析含端口显式性）。
- Gates：`go test ./internal/adapters/...`、`go test ./...`、`go vet ./...` 全 PASS。
- Handoff：`artifact:git-adapter` 提供 canonical root/gitdir 与 RemoteURL.ExplicitPort，供 V1C-07 校验与 §14 端口规则使用。
