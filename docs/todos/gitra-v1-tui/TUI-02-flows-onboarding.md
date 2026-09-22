# TUI-02: Flows & Onboarding

- **Todo ID**: TUI-02
- **Execution Map**: execution-map.json
- **Todo maturity**: DECOMPOSED
- **Task detail format**: v1
- **Source baseline**: docs/plans/gitra-v1-tui-plan.md（WP-TUI-02）；docs/gitra-v1-implementation-baseline.md §36–§42/§49/§51；docs/gitra-v1-login-onboarding-design.md §6

## Outcome

实现全部用户流程：登录向导（掩码输入）、文件夹选择与绑定、确认对话框、首次登录引导与错误白话化。

## Contract

- **Requirements**: `REQ-OBJ-01`, `REQ-SCOPE-01`, `REQ-SCOPE-02`, `REQ-SCOPE-03`, `REQ-SCOPE-04`, `REQ-STEP-02`, `REQ-WP-02`, `REQ-TEST-02`, `REQ-TEST-03`, `REQ-TEST-04`, `REQ-DONE-02`
- **Produces**: `artifact:tui-flows` — 登录向导/文件夹选择/确认对话框/首次启动引导/白话错误
- **Gates**: `go test ./internal/presentation/tui/...`、`go test ./...`、`go vet ./...`
- **Boundaries**: 登录与测试连接只在用户显式操作时联网；token 只在内存中传递且永不回显。

## Tasks

- [ ] `T01` 实现登录向导（平台 → 方式 → 掩码输入）与登录结果处理。
  - **Test and RED**: 断言三步流转与 Esc 回退、输入以 `•` 掩码显示、视图中不出现原始 token、登录成功后刷新卡片；实现前 RED。
  - **Execution logic**: 方式一复用本机 gh/glab；方式二读取掩码输入后调用 `LoginService.Login`；结果消息决定成功提示或白话错误；busy 状态期间忽略按键。
  - **Files and responsibilities**: `app.go`（`handleLoginKey`、`loginDoneMsg` 处理）、`views.go`（`viewLogin`）、`commands.go`（`loginCommand`）。
  - **Verification**: 正常/边界（空 token 提示）/安全（掩码、不打印到视图与错误信息）。
  - **Artifact handoff**: 首次启动引导复用登录结果。
- [ ] `T02` 实现文件夹选择与绑定、确认对话框、首次登录引导与错误白话化。
  - **Test and RED**: 断言文件夹选择器列出子目录、可进入/返回、可切换手动输入；确认对话框 Y 执行、N 取消；首次登录后当前目录为未绑定仓库时弹出绑定确认；常见错误被翻译为中文文案；实现前 RED。
  - **Execution logic**: 选择器条目为「使用这个文件夹 / 上一层 / 子目录…」，绑定调用 `BindingService.Bind`；确认动作以 `func() tea.Cmd` 延迟执行，避免 Update 阻塞；`plainError` 映射 sentinel error → 用户文案与下一步动作。
  - **Files and responsibilities**: `app.go`（`handleBindKey`、`handleConfirmKey`、`offerOnboardingBind`）、`views.go`（`viewBind`、`viewConfirm`）、`labels.go`（`plainError`）。
  - **Verification**: 正常/边界（手动路径、上一层）/错误（非仓库、已绑定、凭据失效）/引导（仓库+未绑定）。
  - **Artifact handoff**: TUI-03 的启动器与文档以此流程为准。


## Tests

- 登录向导流转与掩码、文件夹选择导航、确认对话框、首次登录引导、错误翻译。
- **RED**: 流程缺失时的断言失败。


## Done

- [ ] Every task has RED/GREEN evidence and the node Gates pass.
- [ ] Artifact/evidence/status are updated in the execution map.
- [ ] Graph and semantic checks pass.

## STOP conditions

- 需要真实网络/真实账号才能通过自动化测试。
- 需要在界面回显 token 才能调试（禁止）。


## Evidence / Handoff

- GREEN：登录向导三步（平台 → 方式（gh/glab 复用或粘贴访问码）→ 掩码输入，Esc 逐级返回，视图中 token 只显示为 `•`，测试断言原始值不出现在视图）、文件夹选择器（「使用这个文件夹 / 上一层 / 子目录」+ E 手动输入路径）、确认对话框（Y 执行 / N 取消，动作以 `func() tea.Cmd` 延迟执行避免阻塞）、首次登录引导（当前目录为未绑定仓库时弹出「要绑定到 … 吗？」）、`plainError` 中文白话映射与下一步动作。
- 测试：`TestLoginWizardStepsAndMasking`、`TestBindPickerNavigation`、`TestOnboardingOfferAfterFirstLogin`、`TestConfirmDialogRunsAction`、`TestVerifyMessageRendering`、`TestPlainErrorTranslation` 全部通过。
- Gates：`go test ./internal/presentation/tui/...`、`go test ./...`、`go vet ./...` 全 PASS。
- Handoff：`artifact:tui-flows` 供 TUI-03 启动器与用户自举验证。
