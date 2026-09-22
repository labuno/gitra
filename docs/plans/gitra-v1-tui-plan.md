# gitra V1.0 TUI 实施计划

> Plan ID：`gitra-v1-tui`
> 范围：基线 Milestone 4（卡片式 TUI）+ 增补设计 §6 的首次启动引导（M4.5 首段）
> 上游约束：`docs/gitra-v1-implementation-baseline.md`（修订 1）、`docs/gitra-v1-login-onboarding-design.md`、`docs/gitra-architecture-development-design.md`

---

## Objective

让完全不使用命令行的用户也能完成全部操作：双击启动 gitra，在界面里登录、选择文件夹绑定、检查状态；界面只调用 Application 层，启动时不联网。

## Scope

- Bubble Tea 模型与屏幕：账号卡片页（响应式多列）、账号详情页、登录向导、文件夹选择与绑定、确认对话框。
- 登录向导：选择平台 → 选择方式（复用 gh/glab / 粘贴访问码）→ 掩码输入 → 执行登录；token 在界面中永不回显。
- 首次启动引导：无账号时欢迎页直接引导登录；首次登录成功后若当前文件夹是未绑定仓库，弹出「是否绑定」确认。
- 错误白话化：所有失败都翻译为用户可读文字与下一步动作。
- 启动方式：无参数 + 终端 → TUI；非终端或带子命令 → CLI 不变；提供可双击启动的 `Gitra.command`。
- 边界：TUI 不直接访问 git/ssh/storage（基线 §38）；首页不主动联网（基线 §4.8）。

## Non-goals

- 浏览器 device flow（等待 OAuth App）、目录树图形浏览（提供列表 + 手动输入路径）、账号字段的完整编辑表单（先用重新登录刷新凭据）、主题/配色定制。

## Steps

1. 模型、屏幕切换、账号卡片与详情页导航。
2. 登录向导、文件夹选择与绑定、确认对话框、首次启动引导。
3. 启动路径（TUI/CLI 分流）与双击启动器；TUI 测试。

## Interfaces

- `tui.Run(app *bootstrap.App) error` 为唯一入口；模型只持有 `*bootstrap.App`。
- 视图模型：`AccountCard{ID, Provider, Alias, Host, AuthLabel, LocalState, ProjectCount}`、`Project{ID, Alias, Path, State}`。
- 响应式：`columnsFor(width)`；卡片内容宽 34、含边框 38、间距 2。

## Work Packages

### TUI-01 Shell & Navigation

Bubble Tea 模型、账号卡片页与详情页、响应式列数、键盘导航与本地数据加载。

### TUI-02 Flows & Onboarding

登录向导、文件夹选择/绑定、确认对话框、首次登录后的绑定引导与错误白话化。

### TUI-03 Launch & Packaging

无参数启动 TUI（TTY 检测与回退）、`Gitra.command` 双击启动器、TUI 测试与文档。

## Test Matrix

- 布局：不同终端宽度下的列数。
- 交互：登录向导三步骤与返回、账号导航与详情、确认对话框的确认/取消、文件夹选择器导航。
- 引导：首次登录后对未绑定仓库弹出绑定确认。
- 安全：访问码在界面中掩码、任何视图不出现 token。
- 渲染：卡片包含别名/站点/状态；失败信息可见。

## Verification

- 每节点 `go test ./...` + `go vet ./...`；`go test ./internal/presentation/tui/...` 覆盖交互与渲染。

## Done

- 全部 Work Package 完成、节点验证成功、证据入图。
- 用户双击 `Gitra.command` 后，只用键盘即可完成：登录 → 选择文件夹 → 绑定；无需输入任何命令。

## STOP Conditions

- 需要 TUI 直接访问 git/ssh/storage 才能实现（违反基线 §38）。
- 需要启动时联网或需要真实网络才能通过自动化测试。
- 源文档 digest 漂移或语义复核失败。

## External Sources

- Bubble Tea / Lip Gloss（基线 §19.2）。
- macOS 双击 `.command` 启动行为。
