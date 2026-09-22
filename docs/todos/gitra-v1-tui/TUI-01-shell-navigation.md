# TUI-01: Shell & Navigation

- **Todo ID**: TUI-01
- **Execution Map**: execution-map.json
- **Todo maturity**: DECOMPOSED
- **Task detail format**: v1
- **Source baseline**: docs/plans/gitra-v1-tui-plan.md（WP-TUI-01）；docs/gitra-v1-implementation-baseline.md §36–§42/§49/§51；docs/gitra-v1-login-onboarding-design.md §6

## Outcome

建立 Bubble Tea 外壳：账号卡片页（响应式多列）、账号详情页、键盘导航与本地数据加载，启动不联网。

## Contract

- **Requirements**: `REQ-OBJ-01`, `REQ-SCOPE-01`, `REQ-SCOPE-05`, `REQ-STEP-01`, `REQ-IF-02`, `REQ-IF-03`, `REQ-IF-04`, `REQ-WP-01`, `REQ-TEST-01`, `REQ-STOP-01`, `REQ-EXT-01`
- **Produces**: `artifact:tui-shell` — 模型/屏幕/卡片页/详情页/响应式布局
- **Gates**: `go test ./internal/presentation/tui/...`、`go test ./...`、`go vet ./...`
- **Boundaries**: TUI 只调用 Application 层；启动只做本地读取；不直接 import adapters/git/ssh/storage。

## Tasks

- [ ] `T01` 实现模型、屏幕切换与账号卡片页（含响应式列数与空状态引导）。
  - **Test and RED**: `internal/presentation/tui/tui_test.go` 断言 `columnsFor` 在各宽度下的列数、无账号时出现欢迎与「按 A 登录」、卡片视图包含别名/站点/状态；实现前 RED。
  - **Execution logic**: `New(app)` 加载本地账号与项目数（`Accounts.List` + `Bindings.ListByAccount`），本状态由策略 `Validate` 离线判定（configured / needs_login / invalid）；`View()` 按屏幕渲染；卡片内容宽 34、总宽 38、间距 2。
  - **Files and responsibilities**: `model.go`（状态与视图模型）、`views.go`（渲染与 `columnsFor`）、`commands.go`（本地加载与操作命令）、`labels.go`（状态与文案）、`styles.go`。
  - **Verification**: 正常：两个账号显示两张卡片；边界：窄终端 1 列、宽终端多列；空状态引导可见；安全：不联网（测试环境无网络亦可跑）。
  - **Artifact handoff**: TUI-02 的流程界面复用同一模型与渲染基座。
- [ ] `T02` 实现账号详情页与项目列表导航（含健康度展示与返回）。
  - **Test and RED**: 断言 Enter 打开详情、详情包含「提交身份 / 已绑定的项目」、Esc 返回、↑↓ 切换项目；实现前 RED。
  - **Execution logic**: 打开详情时加载账号与绑定列表，逐项调用 `Bindings.Status`（本地）计算健康度（ok/drift/missing/needs_login）；选中项目高亮。
  - **Files and responsibilities**: `app.go`（`handleDetailKey`、`openDetail`、`loadDetailProjects`）与测试。
  - **Verification**: 正常：打开/返回/选择；边界：无项目时提示按 B 绑定；状态：drift 与 needs_login 显示为警告文案。
  - **Artifact handoff**: TUI-02 在详情页挂载绑定/解绑/测试/删除动作。


## Tests

- 列数、空状态、卡片渲染、详情导航。
- **RED**: 模型与渲染缺失时的编译/断言失败。


## Done

- [ ] Every task has RED/GREEN evidence and the node Gates pass.
- [ ] Artifact/evidence/status are updated in the execution map.
- [ ] Graph and semantic checks pass.

## STOP conditions

- 需要 TUI 直接访问 git/ssh/storage。
- 启动需要联网。


## Evidence / Handoff

- GREEN：`internal/presentation/tui/{model,styles,views,commands,labels,app}.go`——Bubble Tea 模型、账号卡片页（响应式 1–4 列，卡片含平台/站点/别名/状态/登录方式/项目数）、账号详情页（提交身份、项目列表与健康度）、键盘导航（↑↓←→/jk/Enter/Esc/Q）；启动仅本地读取（`Accounts.List` + `Bindings.ListByAccount` + 策略离线 Validate）。
- 真实终端冒烟：`./bin/gitra`（隔离 GITRA_CONFIG_DIR）渲染欢迎页；`A` 进入登录向导显示三个平台；回车进入方式选择；`Ctrl+C` 干净退出（alt-screen 正常恢复）。
- Gates：`go test ./internal/presentation/tui/...`、`go test ./...`、`go vet ./...` 全 PASS；gofmt 干净。
- Handoff：`artifact:tui-shell` 供 TUI-02 挂载流程。
