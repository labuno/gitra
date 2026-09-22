# gitra V1.1 登录与新手引导增补设计

> 增补范围：浏览器登录、HTTPS 凭据托管、引导式克隆、错误白话化
> 状态：V1.1 / V1.2 正式开发输入（2026-09-22 起生效）
> 上游文档：`gitra-architecture-development-design.md`、`gitra-v1-implementation-baseline.md`
> 优先级：本增补与上游文档冲突时，以本增补为准；基线文档已同步「修订 1」
> 宗旨：**把用户当傻瓜——用户不需要获取任何东西（不需要 key、不需要 token、不需要知道 host/端口/协议）**

---

# 1. 背景与目标

V1.0 基线的假设用户是「懂 Git 的多账号使用者」：自己生成 SSH key、自己在平台注册、自己理解 SSH/HTTPS 差异。

本增补把目标用户改为**完全不懂 Git 的人**：

> 安装 gitra → 登录（浏览器点一次授权）→ 拿到一个能用的仓库 → 之后在 IDE 里直接同步/上传，自始至终不需要理解 Git 概念。

判定标准（V1.1 + V1.2 完成时）：

```text
从零开始（机器上没有任何 Git 账号配置）
  -> 安装 gitra
  -> 双击打开（TUI）
  -> 点「登录 GitHub」，浏览器点一次「授权」
  -> 粘贴一个仓库链接（或选一个本地文件夹）
  -> 完成
全程：不创建 key、不复制 token、不填表单、不输命令、不改 remote URL。
```

---

# 2. 产品原则（新增硬约束）

## 2.1 零输入原则

用户永远不需要：

- 创建 / 复制 / 粘贴 SSH key
- 创建 / 复制 / 粘贴 token（PAT 兜底除外，且仅一次，见 §4.4）
- 填写 host / 端口 / 协议 / 用户名 / Git 邮箱
- 修改 remote URL
- 理解 SSH、HTTPS、credential、OAuth 等概念

## 2.2 唯一必需的用户动作

**浏览器里的一次「授权」确认。**

这是安全与合规边界，不可省略、不可代用户授权。除此之外的一切输入由 gitra 自动完成。

## 2.3 UI 词汇表（界面禁用词）

| 禁用（内部术语） | 用户可见说法 |
|---|---|
| SSH / SSH key / 密钥 | （不出现） |
| token / PAT / OAuth | （不出现，兜底时叫「访问码」并附截图指引） |
| remote / 远端 | 仓库地址 |
| host / 端口 | （不出现，自动） |
| credential / 凭据 | 登录 |
| repo / repository | 仓库 |
| fetch / pull / push | 同步 / 上传 |
| commit identity | 提交身份（用「你在 Git 里显示的名字」） |

## 2.4 失败必须可自愈

任何登录 / 凭据 / 网络失败，界面必须给出「一键修复」动作：重新登录、重试、打开帮助页。
禁止出现裸错误码或 Git 原始报错（见 §10）。

## 2.5 与 V1.0 原则的关系

以下原则**不变**：

- gitra 不代理 Git 命令（不提供 `gitra push`）
- 绑定后原生 Git / IDE / Codex / 脚本正常工作
- 仓库自包含、可搬迁；状态可恢复（Snapshot / Rollback / Reconciler 全部沿用）
- 不做常驻进程、不做 daemon、不做 Web 服务

---

# 3. 认证路线决策

## 3.1 结论

| | HTTPS + 浏览器登录（主路径，V1.1） | SSH（高级选项，保留 V1.0 能力） |
|---|---|---|
| 用户操作 | 浏览器点一次授权 | 手工指定已有 key（V1.0 方式） |
| 需要用户拥有的东西 | 无 | 已有 SSH key |
| Git 侧机制 | repo-local `credential.*` 投影 | repo-local `core.sshCommand` 投影 |
| 克隆的仓库 | 直接用（HTTPS remote 原样） | 需 remote 为 SSH 形式 |
| 企业网络 | 443 端口，通行 | 常被封锁 22 端口 |
| 推理 | **新手默认** | 面向进阶用户，需显式选择 |

## 3.2 理由

1. SSH 方案要求「用户先拥有 key」，与零输入原则冲突；自动生成/上传 key 仍需先登录，复杂度只增不减。
2. HTTPS 凭据可以托管进系统凭据库，git 用自带 helper 取回，**运行时零依赖 gitra 二进制**（见 §5.2 实测）。
3. 网页端复制出来的克隆链接默认就是 HTTPS，主路径不需要任何 remote 改写。

---

# 4. 登录方案矩阵与阶梯

## 4.1 Provider 能力矩阵

| Provider | 自动登录方式 | 备注 |
|---|---|---|
| GitHub（github.com） | Device Flow（确定支持，无需 client secret）；Loopback + PKCE（待 spike） | 复用已登录的 `gh`；PAT 兜底 |
| GitLab.com | OAuth（Device Flow 支持度待 spike；Loopback + PKCE 待 spike） | 复用已登录的 `glab`；PAT 兜底 |
| 自建 GitLab / Gitea / Forgejo | 每个实例需自行注册 OAuth App，无法预置 | **默认 PAT 兜底** + 打开实例设置页的引导 |
| 其他（Bitbucket、Gerrit 等） | 不预置 | `generic` provider 能力见 V2 候选 |

## 4.2 阶梯与降级算法

```text
Login(provider):
  1. 若本机已有 gh / glab 且已登录         -> 复用其 token（零浏览器交互）
  2. 若 provider 支持 Loopback + PKCE      -> 自动打开浏览器，回环接收授权
  3. 若 provider 支持 Device Flow          -> 显示设备码（自动复制到剪贴板），打开浏览器
  4. 其余（自建实例等）                     -> 引导粘贴访问码（PAT）
```

对用户始终只显示一个按钮：`登录 GitHub` / `登录 GitLab` / `登录 Gitea`。阶梯对用户不可见。

## 4.3 gh / glab 复用（已批准）

- 检测手段：`gh auth status` / `gh auth token`；`glab auth status`
- 复用 token 前必须验证 scope 与用户名，并明确告知用户「已使用你电脑上已登录的 GitHub 账号 lunafoundry」
- 用户拒绝或校验失败则继续走阶梯 2/3/4

## 4.4 PAT 兜底（已批准）

- 适用：自建实例、企业 SSO 限制、OAuth 不可用
- 交互：给出「打开 xxx/settings/tokens」深链 + 所需权限清单 + 粘贴框；只要求一次
- 校验：粘贴后立即调用 Provider Profile API 验证并显示「已登录为 lunafoundry」
- 失败：明确指出缺哪个权限（如 `repo`）

## 4.5 OAuth App 运维要求

- 项目需要注册并维护：GitHub OAuth App（开启 Device Flow）、GitLab.com OAuth Application
- client_id 内嵌于二进制（公开信息，非机密）；GitHub Device Flow 不需要 client secret
- 自建实例：不预置 client_id，一律 PAT 兜底
- 需要准备：应用主页、隐私声明 URL、用户可读的授权说明文案

---

# 5. 凭据架构

## 5.1 存储：SecretStore（系统原生，不落明文）

| 平台 | 凭据库 | 说明 |
|---|---|---|
| macOS | Keychain（git 自带 `git-credential-osxkeychain`） | 无需额外安装 |
| Windows | Windows Credential Manager（GCM，随 Git for Windows） | 无需额外安装 |
| Linux | libsecret / Secret Service | 若系统缺失，降级见 §5.7 |

存储实现统一走 git 原生接口：

```text
git credential approve   # 写入
git credential fill      # 读取
git credential reject    # 删除
```

收益：零 cgo、零平台专属代码、与用户已有 Git 环境行为一致。

**Token 永不写入 gitra 的 JSON 配置、永不进入日志、永不写入 `.git/config`。**

## 5.2 取回：交给 git 原生 helper，运行时零依赖

仓库内只投影「用哪个账号名」，token 由系统 helper 提供：

```ini
[credential "https://github.com"]
    username = lunafoundry
```

结论：**`git push` / IDE 同步时不需要 gitra 二进制在场**，「Bind once, Git forever」继续成立。

实测证据（macOS，Git 2.49）：

```text
同机同 host（github.com）两个仓库，repo-local 配置各自指向不同账号，
git credential fill 分别返回各自凭据，互不串号。
git-credential-osxkeychain 由 Git 自带，存在性已验证。
```

## 5.3 repo-local 配置投影（受管 key 清单）

`https-token` 路径受管 key（全部 repo-local，纳入 Snapshot / Rollback / DRIFT）：

```text
user.name
user.email
credential.helper                  # 仅当系统未配置任何 helper 时写入，指向系统原生 helper
credential.<https-url>.username    # 账号选择（多账号隔离的关键）
credential.<https-url>.useHttpPath # 仅在需要路径维度区分时写入
gitra.bindingId
gitra.accountId
gitra.strategy
gitra.version
```

## 5.4 多账号隔离规则

1. 每个账号的 token 在凭据库中按 `(host, username)` 维度区分（**开工前必须 spike 验证 osxkeychain / GCM 的维度行为**，见 §12）
2. 多账号（同一 host ≥ 2 个账号）：只在被管理仓库内写 repo-local `credential.<url>.username`，**禁止**写 host 级全局选择
3. 单账号：允许在克隆等未绑定场景临时使用 host 级配置，但必须在克隆完成后收敛为 repo-local

## 5.5 克隆先于绑定的时序（关键）

```text
登录（账号凭据入库，按 host+username）
   |
克隆（gitra 使用该账号凭据；无需用户理解）
   |
克隆完成 -> 立即 bind（投影 repo-local credential 作用域）
   |
之后所有 Git 操作走 repo-local 配置，与 gitra 进程无关
```

## 5.6 登出与撤销

```bash
gitra logout <alias>
```

执行：删除凭据库条目 → 清理该账号受管的 repo-local 配置（恢复 Snapshot）→ 提示平台侧撤销页链接。
不允许「静默保留 token」。

## 5.7 降级与兜底

- Linux 无 libsecret：降级为 repo-local 凭据文件（权限 600），并在 UI 中明确告知风险与替代方案
- 凭据库不可写：退回 PAT 重输一次，不写任何明文

---

# 6. 登录交互设计（零输入剧本）

## 6.1 首次启动（Onboarding，V1.2）

```text
欢迎使用 gitra
  [ 登录 GitHub ]  [ 登录 GitLab ]  [ 登录 Gitea ]
        |
        v
浏览器自动打开 -> 用户点「授权」-> 回到 gitra -> 账号卡片自动生成
        |
        v
  [ 克隆一个仓库 ]  [ 打开本地文件夹 ]
        |
        v
完成页：告诉用户「以后在编辑器里直接同步即可」
```

## 6.2 添加账号

- TUI：选 Provider → 登录 → 自动填充 → 保存（用户零输入）
- CLI：`gitra login --provider github`（同样零输入；高级用户仍可用 `gitra account add` 手工模式）

## 6.3 登录流程状态机

```text
Idle -> OpeningBrowser -> WaitingForUser -> ExchangingToken -> FetchingProfile -> Done
                              |                   |                  |
                              +---- 超时/取消 ----+---- 失败 ---------+-> Failed(可重试/换兜底)
```

- Device Flow：展示设备码 + 自动复制 + 打开浏览器 + 轮询
- Loopback：启动本地回环监听 → 打开浏览器 → 接收 code → 换取 token

## 6.4 登录后的自动填充

- GitHub：`GET /user` + `GET /user/emails`（scope: `read:user user:email`）
- GitLab：`GET /user`（scope: `read_user`）
- Gitea：`GET /user`（scope: `user`）
- 自动生成：Alias（username，冲突时加后缀）、Git name、Git email（私有邮箱给出显式二选一提示）

## 6.5 过期 / 失效恢复

- 卡片状态增加 `NeedsLogin`；任何 Git 操作失败且诊断为「凭据失效」时，界面直接给出「重新登录」按钮
- 不要求用户理解 401 / permission denied

---

# 7. 引导式克隆（V1.2）

## 7.1 入口与流程

- 入口：「克隆仓库」按钮 + 粘贴框（可从剪贴板自动检测 GitHub/GitLab/Gitea 链接）
- 解析链接 → 识别 provider / owner / repo → 若无对应账号则内联引导登录 → clone 到默认目录 → 自动 bind → 提示完成
- 这是唯一允许用户手工输入的内容（一个链接）

## 7.2 失败恢复矩阵

| 情况 | 用户可见处理 |
|---|---|
| 目录已存在 | 「这个文件夹已存在，换个名字或直接打开它」 |
| 无权限 / 404 | 「这个账号看不到该仓库，换一个账号或申请权限」 |
| 凭据过期 | 内联「重新登录」 |
| 网络失败 | 「连不上 xxx，检查网络后重试」 |
| 私有仓库无账号 | 引导登录后自动继续 |

## 7.3 与 bind 的边界

`gitra bind` 本身仍然**不 clone**（V1.0 契约不变）；引导式克隆是独立用例：`clone -> bind` 组合调用。

---

# 8. 领域与接口增量

## 8.1 Domain

```go
type TransportConfig struct {
    Strategy string            // "ssh-key" | "https-token"
    Config   map[string]string // https-token: credential_ref / url_scope / username，禁止出现明文 token
}

type LoginState string

const (
    LoginConfigured LoginState = "configured"
    LoginRequired   LoginState = "needs_login"
    LoginInvalid    LoginState = "invalid"
)
```

## 8.2 Ports 新增

```go
type SecretStore interface {
    Get(ctx context.Context, ref string) (string, error)
    Set(ctx context.Context, ref string, value string) error
    Delete(ctx context.Context, ref string) error
}

type BrowserLauncher interface {
    Open(ctx context.Context, url string) error
}

type LoginMethod interface {
    ID() string // "reuse-cli" | "loopback" | "device-flow" | "pat"
    Available(ctx context.Context, provider ProviderRef) bool
    Login(ctx context.Context, req LoginRequest, obs LoginObserver) (LoginResult, error)
}
```

`LoginObserver` 用于把「等待授权 / 设备码 / 进度」推给 TUI，不泄漏 token。

## 8.3 Strategies

- `AuthStrategy: https-token`：输出 `[]GitConfigEntry`（`credential.<url>.username` 等）
- `ssh-key` 保持不变，作为高级路径

## 8.4 Application

- `LoginService`：`Login / Logout / Status / Refresh`
- `OnboardingService`（V1.2）：`CloneAndBind / SuggestActions`

## 8.5 Provider Adapter 扩展

```go
type ProviderLoginAdapter interface {
    BeginDeviceFlow(ctx context.Context) (DeviceFlow, error)
    PollDeviceFlow(ctx context.Context, flow DeviceFlow) (Token, error)
    LoopbackConfig() (LoopbackConfig, bool)
    ExchangeLoopbackCode(ctx context.Context, code string) (Token, error)
    Profile(ctx context.Context, token Token) (ProviderProfile, error)
}
```

ProviderCapabilities 中已有的 `OAuth / DeviceFlow / SSHKeyAPI` 字段正式启用（SSHKeyAPI 暂不实现）。

## 8.6 Presentation

- CLI 新增：`gitra login`、`gitra logout`、`gitra clone <url> [dir]`（V1.2）
- TUI 新增页面：`Login`、`Onboarding`
- 卡片新增状态：`Configured / NeedsLogin / Invalid`（首页仍然不主动联网）

## 8.7 目录结构增量

```text
internal/adapters/secretstore/     # 基于 git credential approve/fill/reject 的实现
internal/adapters/browser/         # 打开浏览器（可测）
internal/strategies/auth/httptoken/
internal/app/login_service.go
internal/app/onboarding_service.go
internal/adapters/provider/*/login.go
internal/presentation/tui/screens/login.go
internal/presentation/tui/screens/onboarding.go
```

---

# 9. 安全与隐私

- Token 仅存系统凭据库；gitra 的 JSON / 日志 / `.git/config` 中不出现
- `--verbose` 日志同样脱敏；日志白名单机制（只允许打印 username、host，不打印 secret）
- OAuth client_id 为公开信息；项目不托管任何用户数据，无云服务、无 telemetry
- 登出即删除本地凭据并给出平台侧撤销入口
- 明确不做：密码登录、自建账号体系、token 云同步、跨设备迁移、企业 SSO 深度适配

---

# 10. 错误白话化对照表

| 内部错误 | 用户可见文案（含动作） |
|---|---|
| `ErrAuthInvalid` / 401 / permission denied | 「你的 GitHub 登录已过期」+［重新登录］ |
| `ErrNotGitRepository` | 「这个文件夹不是仓库」+［换一个］［帮我创建一个］ |
| 无权限 / 404 | 「这个账号看不到该仓库，换一个账号或申请权限」 |
| 网络超时 / DNS | 「连不上 github.com，检查网络后重试」+［重试］ |
| 凭据库不可写 | 「无法保存登录信息」+［重试］［改用访问码］ |
| 高级路径 SSH 失败 | 「仓库地址格式不支持」+［改用 HTTPS（自动）］ |

原则：不出现 `fatal:`、exit code、SSH/credential 等术语。

---

# 11. 对 V1.0 基线的修订清单（9 条硬约束逐条）

| 基线条款 | 原文 | 修订后 |
|---|---|---|
| §3.2 | OAuth / HTTPS Token / Credential Helper / clone manager 列入「V1 不做」 | 移入 V1.1 / V1.2 立项（本增补） |
| §4.2 | V1 只支持 SSH Remote | V1.0 core 用 SSH 验证机制；V1.1 起 HTTPS 为主路径，SSH 为高级选项 |
| §4.3 | Endpoint 含 SSHUser / SSHPort | 不变；HTTPS 路径增加 url_scope 概念（`https://<host>`） |
| §4.6 | `ssh -T` 不能只看 exit code | 不变，仅适用于 SSH 高级路径；HTTPS 路径验证改走 Provider API（`/user`） |
| §4.8 | TUI 状态禁止依赖实时网络 | 语义不变：首页仍不主动联网；登录是用户显式触发的唯一联网流程 |
| §4.9 | Path Canonicalization | 不变 |
| §9 | credential.* 一律不碰 | 修订为：credential.* 纳入受管清单（URL 作用域 + 纳入 Snapshot / DRIFT） |
| §14 | 非 22 端口固定追加 `-p` | 修订：URL 自带端口时不加 `-p`（OpenSSH 重复 `-p` 取首个，实测），跨 host remote 给出警告 |
| §53 | 测试基线 | 新增 §53.8 HTTPS 凭据隔离 E2E |
| §60 | 「SSH-only 边界明确」「无 OAuth」 | 修订为两条认证路径边界明确；登录仅走平台浏览器授权 |

命名与路径冻结（V1.0 决策）：

```text
模块       github.com/zhanhd/gitra
二进制     gitra
配置目录   os.UserConfigDir()/gitra        （测试/便携覆盖：GITRA_CONFIG_DIR）
账号文件   accounts/acc_xxx.json（按 ID 命名，不按 Alias）
仓库元数据 <GitDir>/gitra/state.json
Git 配置段 [gitra]
```

---

# 12. 需要 spike 验证的点（开工前完成，各 ≤ 0.5 天）

| # | 问题 | 影响 | 备选方案 |
|---|---|---|---|
| 1 | GitHub 是否支持 Loopback + PKCE（含任意 127.0.0.1 端口） | 决定是否完全零粘贴 | 退回 Device Flow（设备码自动复制） |
| 2 | GitHub `verification_uri` 是否支持预填 user_code | 零粘贴程度 | 自动复制设备码，用户 Cmd+V |
| 3 | GitLab.com Device Flow 支持度与 scope | 主路径可用性 | Loopback（若可用）→ PAT |
| 4 | osxkeychain / GCM 是否按 (host, username) 维度存取 | 多账号隔离根基 | 自研 helper 或 repo-local 凭据文件 |
| 5 | GitHub OAuth token 作为 HTTPS 凭据的 username/password 组合 | push 可用性 | 使用 `x-access-token` 或账号名两种组合实测 |
| 6 | Gitea 自建实例 PAT 最小创建步骤 | 兜底文案 | 提供带截图的分步引导 |

---

# 13. 里程碑与验收

## M2.5 登录 MVP（PAT + SecretStore + 凭据投影）

验收：

- 手工粘贴 PAT → 账号创建成功且自动填充 profile
- 绑定仓库后 `git push` 成功，且**移除 gitra 二进制后 push 仍成功**
- 同 host 双账号凭据隔离测试通过（§53.8）
- `gitra logout` 后凭据条目与受管配置被清理

## M3.5 浏览器登录（零输入）

验收：

- 全新用户从「零账号」到 push 成功：除浏览器点一次「授权」外无任何输入
- gh / glab 已有登录时可零浏览器完成
- 凭据过期时出现「重新登录」，一键恢复

## M4.5 引导式克隆 + 错误白话化

验收：

- 粘贴仓库链接 → 必要时内联登录 → clone → 自动 bind → push 成功
- §7.2 失败矩阵全部有对应白话提示与恢复动作

## 回归要求

- V1.0 SSH 高级路径与双账号双仓库 E2E 不得退化
- `git push/pull/fetch` 不依赖 gitra 进程（两条路径都要成立）

---

# 14. 明确不做（本增补）

- SSH key 自动生成 / 自动上传（V2 候选）
- 密码登录、自建账号体系、token 云同步、多设备迁移
- 企业 SSO / SCIM / 组织策略深度适配（仅提供提示与手动兜底）
- 为任意自建实例托管 OAuth App
- 代理 Git 命令（保持「不代理」原则）
- 仓库浏览、历史、分支、PR 等既有非目标照旧

---

# 附录 A：V1.0 决策冻结（P0 清零）

## A.1 查询命令 JSON Schema（稳定契约，`schema_version: 1`）

```json
// gitra status --json（未绑定）
{ "schema_version": 1, "repository": "/abs/path", "bound": false }

// gitra status --json（已绑定）
{
  "schema_version": 1,
  "repository": "/abs/path",
  "bound": true,
  "binding": { "id": "bnd_...", "account_id": "acc_...", "strategy": "repo-local", "health": "ok" },
  "account": { "id": "acc_...", "alias": "lunafoundry",
               "provider": { "type": "github", "host": "github.com", "username": "lunafoundry" } },
  "auth": { "strategy": "ssh-key", "state": "configured" }
}

// gitra account list --json
{
  "schema_version": 1,
  "accounts": [
    { "id": "acc_...", "alias": "lunafoundry",
      "provider": { "type": "github", "host": "github.com", "username": "lunafoundry" },
      "auth": { "strategy": "https-token", "state": "configured" },
      "project_count": 6 }
  ]
}

// gitra account test --json
{
  "schema_version": 1,
  "account_id": "acc_...",
  "success": true,
  "expected_username": "lunafoundry",
  "actual_username": "lunafoundry",
  "message": "authenticated"
}
```

字段一旦发布只增不改；删除字段视为 breaking change，需要提升 schema_version。

## A.2 Exit Code（与基线 §35 一致）

```text
0 success | 1 internal | 2 invalid input | 3 account | 4 repository
5 authentication | 6 binding/state conflict（含 drift） | 7 unsupported
```

## A.3 Remote 校验口径

- 仓库存在 `origin` 时校验 `origin`；否则校验第一个 remote；无 remote 时允许绑定但给出警告
- remote host 必须等于 Account Endpoint host（V1.0）；SSH Host Alias 场景作为后续增强：用 `ssh -G <host>` 解析 effective HostName / Port 后再比对
- 存在其他 host 的 SSH remote 时给出警告（`core.sshCommand` 作用于全仓库）

## A.4 Binding Health 真值表

| Central Binding | 路径存在 | 仓库元数据 | 受管配置 | 结果 |
|---|---|---|---|---|
| 有 | 是 | 匹配 | 一致 | `ok` |
| 有 | 是 | 匹配 | 漂移 | `drift` |
| 有 | 是 | 缺失 / 不匹配 | — | `broken` |
| 有 | 否 | — | — | `missing` |
| 无 | 是 | 存在且 bindingId 可识别 | — | 自恢复：以元数据重建 central，`broken`→修复 |
| 无 | 是 | 无 | — | 未绑定（`bound: false`） |
