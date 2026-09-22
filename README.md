# gitra

## 如何打开（macOS，无需命令行）

1. **双击 `dist/Gitra.app`**（推荐，可拖进「应用程序」文件夹）
   —— 会打开一个终端窗口并显示 gitra 界面。
   如尚未生成：双击 `scripts/build-app.sh` 无法直接运行，可先双击仓库里的
   `Gitra.command`（首次会自动构建）。
2. **双击 `Gitra.command`**：效果相同，界面直接出现在终端里。
3. **给喜欢命令的人**：`bin/gitra` 就是那个命令行程序；`gitra --help` 查看全部子命令。

退出：主页按 `Q`，任何界面按 `Ctrl+C`。

### 第一次使用（全程键盘，不需要敲命令）

1. 按 `A` → 选平台（GitHub / GitLab / Gitea）→ 回车
2. 选登录方式：
   - 本机已登录过 `gh` / `glab` → 直接回车即可；
   - 否则选「粘贴访问码」，在网页创建一个 token（GitHub 权限勾选
     `repo`、`read:user`、`user:email`），回到界面按 `⌘V` 粘贴 → 回车
3. 登录成功后，如果当前文件夹是一个 Git 仓库，界面会问「要把它绑定到 … 吗？」
   按 `Y`；如果仓库还没有地址，会请你确认/填写仓库地址（已有地址绝不被改写）
4. 之后在 Cursor / VS Code / GitHub Desktop 里直接同步/推送即可

### 构建 macOS 应用包

```bash
./scripts/build-app.sh    # 生成 dist/Gitra.app
```


极简 Git 多账号身份管理器：给每个本地仓库绑定一个账号，之后原生
`git pull / push / fetch`、IDE 与脚本自动使用正确身份，无需切换账号、
无需常驻进程。

## 文档

- `docs/plans/gitra-v1-core-plan.md` — 当前实施计划（Milestone 0 + 1）
- `docs/gitra-v1-implementation-baseline.md` — V1 开发基线（行为契约）
- `docs/gitra-architecture-development-design.md` — 架构设计
- `docs/gitra-v1-login-onboarding-design.md` — 登录与新手引导增补（V1.1/V1.2）
- `docs/todos/gitra-v1-core/execution-map.json` — 执行图（唯一状态事实源）

## 开发

```bash
go build ./...
go test ./...
go vet ./...
```
