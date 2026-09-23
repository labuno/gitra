# gitra

**让每个文件夹用对账号。** —— 极简的 Git 多账号身份管理器（CLI + 卡片式 TUI）。

> Bind once, Git forever: bind a repository to an account once, then plain
> `git pull / push / fetch`, your editor and your scripts use the right
> identity — no account switching, no daemon, no Git protocol implementation.

---

## 它解决什么问题

一台电脑上经常有多个 Git 身份：公司的、个人的、开源项目的。原生 Git 的默认行为会让它们互相打架：

- **提交身份**：`user.name` / `user.email` 默认是全局的，工作仓库里可能提交成私人邮箱；
- **凭据**：同一个 `github.com` 只有一个钥匙串条目，最后一个登录的账号会"赢"，于是另一个仓库 `git pull` 报 `Repository not found`；
- **SSH 密钥**：一个 host 多把 key 时，要么配 `~/.ssh/config` Host alias 并改写 remote URL，要么反复 `ssh-add -D`。

gitra 把"哪个文件夹用哪个账号"变成一次性的绑定：绑定之后，**每个仓库自带正确的提交身份与凭据作用域**，之后你正常用 Git 就好。

## 谁需要它

- 在**同一台机器上使用 ≥ 2 个 Git 身份**（公司 + 个人、多个 GitHub 账号、GitHub + GitLab 混用）的人；
- 不想研究 `includeIf`、`core.sshCommand`、SSH Host alias、credential helper 的开发者；
- 希望**不碰命令行**也能完成登录与绑定的用户（有双击启动的 TUI）；
- 让 AI 编码工具（Codex / Cursor 等）、脚本、CI 本机任务在正确仓库里自动使用正确身份的人。

如果你只有一个 Git 账号，原生 `gh`/凭据助手已经够用，gitra 对你的价值有限——这一点我们如实说明。

## 怎么工作

```
登录（一次）                    绑定（每个项目一次）                之后
浏览器授权 / 复用 gh / 访问码 → 把文件夹绑定到账号 → 仓库内写入：
                                 user.name / user.email
                                 credential.<url>.username（或 SSH key 指令）
                                 [gitra] 元数据 + 绑定前的配置快照
                                                        ↓
                                          原生 git / 编辑器 / 脚本 直接可用
```

- 凭据只存在**系统凭据库**（macOS Keychain / Windows Credential Manager / Linux Secret Service），
  仓库与配置里**不出现 token**；
- 绑定前会保存仓库原有配置快照，`unbind` 精确还原；
- 不改写 remote URL、不改全局 `~/.gitconfig`、不改 `~/.ssh/config`、没有常驻进程。

支持的 Provider：**GitHub / GitLab / Gitea（含自建，支持自定义 host 与 SSH 端口）**。
传输方式：HTTPS 凭据（默认）与 SSH 密钥（高级）。

## 开始使用

### macOS（双击即用，不需要命令行）

1. 打开本仓库的 **Releases** 页面，下载最新版里的 `Gitra-macOS.zip`；
2. 双击解压 → 双击解压出来的 `Gitra.app`：会弹出一个「终端」窗口，里面就是 gitra 界面；
3. 首次打开若被 macOS 拦截（“无法验证开发者”/“无法检查是否包含恶意软件”）：
   打开「系统设置 → 隐私与安全性」，在“安全性”一节点「仍要打开」，再双击一次即可——
   这一步只需做一次（解压出来的文件夹里也放着同样的说明）。
4. 打开后看到欢迎菜单 → **登录 GitHub**：
   - 本机登录过官方 `gh` 时可直接授权（浏览器点一次 Authorize）；
   - 否则选「粘贴访问码」，我们会自动打开平台页面并勾好所需权限。
5. 登录后界面会问「要现在添加项目吗？」→ 选择你的项目文件夹
   → 若远端仓库还不存在，会问「要现在创建吗？」→ 创建并绑定
   → 首次上传按 `U`（仅此一次；之后在编辑器里同步即可）。

界面操作全靠键盘：`↑↓←→` 移动 · `Enter` 确认 · `Esc` 返回 · `A` 添加账号 · `B` 绑定项目 · `T` 测试连接 · `U` 首次上传 · `Q` 退出（会先确认）。

> 想跑源码版本：`./scripts/build-app.sh` 生成 `dist/Gitra.app`；
> 或直接双击仓库根目录的 `Gitra.command`（首次会自动构建，需要本机有 Go）。

### 命令行（给脚本与自动化）

```bash
gitra login --provider github          # 浏览器/gh 授权
gitra account list                     # 账号列表（--json）
gitra bind <alias> [path]              # 绑定当前或指定仓库
gitra status [path]                    # 绑定状态与健康度（--json）
gitra account test <alias>             # 显式联网验证身份
gitra publish [path]                   # 一次性首次上传
gitra unbind [path]                    # 还原绑定前状态
```

退出码：`0` 成功 · `2` 参数/校验 · `3` 账号 · `4` 仓库 · `5` 认证 · `6` 冲突/漂移 · `7` 不支持 · `1` 其它。

## 安全模型

- **不收集、不上传**任何数据：没有账号系统、没有遥测、没有云端；
- 凭据仅存本机系统凭据库，gitra 的 JSON 配置、日志、仓库配置里都不出现明文 token；
- 需要凭据时由 **git 自己**从系统凭据库取用，因此卸载 gitra 后仓库仍可正常 push；
- `gitra logout <alias>` 删除凭据并清理仓库内的凭据作用域，并给出平台侧撤销入口。

## 明确不做

不做 Git GUI、不展示历史/分支/PR/Issue、**不代理 Git 命令**（没有 `gitra push/pull/fetch`；
唯一的例外是一次性的 `gitra publish`，用于把空远端与本地首次关联）、不做守护进程、不修改远端地址。

## 当前状态与限制

- 已实现：账号管理、HTTPS/SSH 双路径绑定与还原、登录（gh 复用 / 访问码 / 官方 gh 浏览器授权）、
  显式身份验证、首次上传、卡片式 TUI、`--json` 与退出码契约；测试 140+ 用例、全量 `go test ./...` 与 `go vet` 通过。
- 待实现：内置 device flow（需先注册 OAuth App 的 client_id）、TUI 内的账号字段编辑页、
  TUI 内克隆仓库向导、Windows 作为一等平台。
- SSH 密钥若带口令，需要先 `ssh-add` 到 agent（探测过程不会弹口令提示）。

## 发布与自动化（GitHub Actions）

- **CI**（`.github/workflows/ci.yml`）：`main` 分支推送与 Pull Request 会跑 gofmt/vet/`go test`/构建，
  并在 macOS 上构建一次应用包做冒烟（防止平台相关退化）。
- **Release**（`.github/workflows/release.yml`）：推送 **以 `v` 开头的 tag** 即自动构建并发布：

  ```bash
  git tag v0.1.0          # 例如 v0.1.0、v1.2.3-rc1
  git push origin v0.1.0  # 触发 Release workflow
  ```

  产物包括：六个平台的 `gitra-{darwin,linux,windows}-{arm64,amd64}.zip`、
  macOS 应用包 `Gitra-macOS.zip`（内含首次使用说明）、以及 `checksums.txt`；
  随后自动创建（或更新）同名的 GitHub Release 并附上下载说明。
  （也可以在 GitHub 网页上「Draft a new release」创建同名 tag，工作流会把产物补齐到该 Release。）

版本号会写进二进制：`gitra --version` 显示 tag 名（本地构建显示 `dev`）。

## 首次推送（只有这一次需要指定分支）

新仓库第一次推送必须建立上游关系，之后就不需要了：

```bash
git push -u origin main      # 一次性；也可以直接用 gitra 的「U 首次上传」
```

## 开发

```bash
go build ./... && go test ./... && go vet ./...
./scripts/build-app.sh          # 生成 macOS 应用包 dist/Gitra.app
```

架构与契约文档见 `docs/`：`gitra-architecture-development-design.md`（系统设计）、
`gitra-v1-implementation-baseline.md`（V1 行为基线）、`gitra-v1-login-onboarding-design.md`（登录与新手引导），
以及 `docs/plans/`、`docs/todos/` 下的实施计划与执行证据。
