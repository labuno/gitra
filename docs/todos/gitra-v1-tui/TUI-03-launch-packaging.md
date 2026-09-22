# TUI-03: Launch & Packaging

- **Todo ID**: TUI-03
- **Execution Map**: execution-map.json
- **Todo maturity**: DECOMPOSED
- **Task detail format**: v1
- **Source baseline**: docs/plans/gitra-v1-tui-plan.md（WP-TUI-03）；docs/gitra-v1-implementation-baseline.md §36–§42/§49/§51；docs/gitra-v1-login-onboarding-design.md §6

## Outcome

把 TUI 变成默认入口：无参数启动 TUI（非终端回退帮助），提供双击启动器并完成全套测试与文档。

## Contract

- **Requirements**: `REQ-OBJ-01`, `REQ-STEP-03`, `REQ-IF-01`, `REQ-WP-03`, `REQ-VER-01`, `REQ-DONE-01`, `REQ-EXT-02`
- **Produces**: `artifact:tui-launch` — 无参数启动分流与 Gitra.command
- **Gates**: `go test ./...`、`go vet ./...`
- **Boundaries**: 不改动 CLI 子命令行为；非终端场景不得挂起。

## Tasks

- [ ] `T01` 实现启动分流与双击启动器。
  - **Test and RED**: 构造性验证：无参数 + 非 TTY 时输出帮助并退出 0（可脚本断言），带子命令时仍走 CLI；`Gitra.command` 存在且可执行、内容在缺少二进制时先构建；实现前 RED（无参数时无 TUI/无帮助）。
  - **Execution logic**: `main` 判断 `len(args)==0 && isTerminal(stdin) && isTerminal(stdout)` → `tui.Run(app)`；否则走 CLI；`isTerminal` 用 `ModeCharDevice`；启动器用 bash 检查并构建后 `exec bin/gitra`。
  - **Files and responsibilities**: `cmd/gitra/main.go`（分流）、`Gitra.command`（双击启动）、`.gitignore`（忽略 `bin/`）。
  - **Verification**: 无参数非 TTY → 帮助且 exit 0；`--version`/子命令不受影响；启动器权限 0755。
  - **Artifact handoff**: 用户验证入口（双击 → 登录 → 绑定）。
- [ ] `T02` 运行全量回归并把证据写入执行图。
  - **Test and RED**: 全量 `go test ./...`、`go vet ./...`；实现前（新代码未接入时）失败作为 RED。
  - **Execution logic**: 顺序执行构建、测试、vet、gofmt 检查；把命令/结果/时间写入 execution-map 的 evidence 与 integration_gates；同步节点状态。
  - **Files and responsibilities**: `docs/todos/gitra-v1-tui/execution-map.json` 只承载状态与证据。
  - **Verification**: 三条 Gate 全 PASS；`validate_todo_graph.py` 三项 PASS。
  - **Artifact handoff**: Plan 级 Done 可评审；用户可据此自举验证。


## Tests

- 启动分流冒烟（非 TTY 回退）、启动器可执行性、全量回归。
- **RED**: 分流与启动器缺失时失败。


## Done

- [ ] Every task has RED/GREEN evidence and the node Gates pass.
- [ ] Artifact/evidence/status are updated in the execution map.
- [ ] Graph and semantic checks pass.

## STOP conditions

- 需要破坏 CLI 既有行为才能实现。
- 五轮修复仍失败。


## Evidence / Handoff

- GREEN：`cmd/gitra/main.go` 启动分流（无参数 + stdin/stdout 为终端 → `tui.Run`；其余走 CLI，未改动任何子命令行为）；`Gitra.command` 双击启动器（自动构建 `bin/gitra` 后执行，权限 0755）；`bin/` 已在 .gitignore。
- 验证：`./bin/gitra </dev/null`（非 TTY、无参数）输出帮助并退出 0；真实终端下 TUI 可启动、可导航、可干净退出；全量 Gate 通过。
- Gates：`go test ./...`、`go vet ./...` 全 PASS。
- Handoff：`artifact:tui-launch` 即用户入口：双击 `Gitra.command` → A 登录 → B 选文件夹绑定。

## Amendment (2026-09-22) — 真实终端联调发现并修复的两个问题

- **卡死修复**：TUI 的「复用 gh/glab」路径曾把 stdin 当作 token 来源，而终端被 Bubble Tea 接管，导致界面挂起。现在 `loginRequest()` 固定 `AllowStdin=false`（TUI 只用掩码输入框取访问码），并有单测锁定。
- **外部命令超时**：gh/glab 探测增加 5 秒上限（`cliReuseTimeout`），避免网络/钥匙串阻塞界面；单测用「挂起 runner」验证有界返回。
- **文案修复**：登录失败给出界面内可执行的动作（「请选择粘贴访问码」/「确认 token 权限」），不再出现任何终端命令。
- 真实终端验证：`./bin/gitra` → `A` → 回车（gh 路径）→ 1 秒内返回可操作提示；`Ctrl+C` 干净退出。
