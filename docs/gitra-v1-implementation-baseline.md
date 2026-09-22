# gitra V1 Implementation Baseline

> 开发规划、实现限定与行为契约  
> 状态：V1 开发基线  
> 技术栈：Go + Native Git/SSH + Cobra + Bubble Tea + Lip Gloss  
> 核心原则：**Bind once, Git forever**  
> 开发原则：**功能验证优先，不做防御性编程，不提前实现无真实需求的能力**

> **修订记录**
>
> **修订 1（2026-09-22）**
>
> - 项目统一更名为 `gitra`（原 `gacct`）。
> - 新增增补设计 `gitra-v1-login-onboarding-design.md`：登录 / HTTPS 凭据 / 新手引导（V1.1、V1.2）。
> - 本文随之修订：§3.1、§3.2、§4.2、§9、§14、§37、§40、§44、§53.8（新增）、§55、§60、§62。
> - 与增补设计冲突时以增补设计为准；未修订条款继续有效。

---

## 1. 文档目的

本文用于把 `gitra` V1 从“架构设计”推进到“可以直接开发”的状态。

主架构文档负责回答：

- 系统为什么这样设计
- 模块如何划分
- Adapter / Strategy / Registry / Reconciler 如何组织
- CLI / TUI 如何共享 Application Core

本文负责进一步锁定：

- V1 到底支持什么、不支持什么
- 开工前必须确定的行为契约
- Git / SSH / Repository 的实现边界
- 开发顺序和里程碑
- 每个里程碑的验收条件
- Go 依赖与 Package 划分
- 错误处理、回滚、存储、兼容性与测试基线
- 开发过程中禁止“顺手加”的功能

开发过程中，如果实现与本文冲突，应先修改本文形成新的基线，再修改代码。

---

# 2. V1 一句话目标

`gitra` V1 只解决：

> 在一台机器上管理多个 Git 托管账号，并将每个本地 Repository 永久绑定到指定账号，使后续原生 `git pull / push / fetch` 自动使用正确身份。

理想体验：

```text
添加账号
   |
绑定项目
   |
以后直接 git push / git pull / git fetch
```

`gitra` 不参与正常 Git Runtime。

---

# 3. V1 总体边界

## 3.1 V1 必须支持

- 多个 Git 账号
- GitHub / GitLab / Gitea Provider 基础适配
- 每个账号独立 Commit Identity
- 每个账号独立 SSH Key
- 本地 Repository -> Account 绑定
- Repo-local Git config 投影
- 原生 Git 无感工作
- CLI
- 卡片式 TUI
- Account 编辑
- Project 添加 / 移除
- Test Connection
- Snapshot / Rollback
- Reconcile
- JSON 输出
- 本地 JSON 配置持久化
- 双账号双仓库 E2E 验证

> 修订 1 追加（V1.1 / V1.2）：浏览器登录、HTTPS 凭据托管、引导式克隆、错误白话化。
> 范围与验收见 `gitra-v1-login-onboarding-design.md`。

## 3.2 V1 明确不做

- Git history
- diff
- staging
- branch graph
- branch manager
- merge / rebase UI
- Pull Request
- Issue
- Repository Browser
- Provider Dashboard
- GitHub API repository list
- SSH key 上传
- SSH key 自动生成向导
- 自动修改 remote URL
- daemon
- background service
- tray app
- Web UI
- Electron / Tauri
- telemetry
- plugin marketplace
- cloud sync

> 修订 1：`OAuth`、`HTTPS Token`、`Credential Helper 管理`、`clone manager` 已从「V1 不做」调整为 V1.1 / V1.2 立项能力，见 `gitra-v1-login-onboarding-design.md`。

---

# 4. 开工前必须锁定的 9 条实现约束

以下 9 条属于 V1 的硬约束，不允许开发过程中自行改变。

---

## 4.1 Account ID 与 Alias 必须分离

### 规则

Account 必须拥有稳定的内部 ID：

```text
id = acc_01H...
```

同时拥有用户可读、可修改的 Alias：

```text
alias = lunafoundry
```

### 数据模型

```go
type Account struct {
    ID        AccountID
    Alias     string
    Provider  ProviderRef
    Identity  CommitIdentity
    Transport TransportConfig
    Revision  int
}
```

### Binding 必须引用 AccountID

正确：

```text
RepositoryBinding.AccountID = acc_01H...
```

错误：

```text
RepositoryBinding.Account = "lunafoundry"
```

### 原因

Alias 未来可能修改：

```text
lunafoundry -> personal-github
```

如果 Binding 直接引用 Alias，会导致：

- Alias 改名需要级联更新全部 Binding
- 更容易出现悬挂引用
- 不利于迁移
- 不利于恢复
- 不利于未来多语言或显示名称扩展

因此：

> AccountID 是持久身份；Alias 只是人类可读标识。

### V1 约束

- AccountID 创建后不可修改
- Alias 在本地唯一
- Alias 可以修改
- CLI 可以使用 Alias 作为用户输入
- Application 层收到 Alias 后应尽早解析成 AccountID

---

## 4.2 认证路径：HTTPS 为主，SSH 为高级选项

> **修订 1**：本节原标题为「V1 只支持 SSH Remote」。V1.0 core 仍以 SSH 验证核心投影机制；V1.1 起主路径改为 HTTPS + 浏览器登录（见增补设计）。

### V1.0 core（本文件 M0–M5 主路径）

支持 SSH Remote：

```text
git@github.com:lunafoundry/luna-site.git
ssh://git@git.example.com:2222/luna/repo.git
```

遇到 HTTPS remote 时返回明确错误：

```text
Repository remote uses HTTPS.
V1.0 core supports SSH remotes only.
HTTPS support is planned in V1.1 (see login/onboarding addendum).
```

### V1.1 起（增补设计）

- 主路径：HTTPS + 浏览器授权；SSH 降级为高级选项
- HTTPS remote 直接绑定，由 `https-token` Auth Strategy 投影 repo-local `credential.*` 作用域配置
- 凭据存入系统 SecretStore，取回由 git 自带 helper 完成：**git 运行时零依赖 gitra 二进制**
- 用户不需要创建 key、粘贴 token、修改 remote URL

### 两条路径共同的禁止行为

不允许自动把 SSH remote 改写成 HTTPS，反之亦然：

> `gitra bind` 不修改用户的 remote URL。

### 原因

自动改 remote 会引入额外副作用：

- 破坏现有凭据方案
- 影响企业代理
- 影响 CI / 脚本
- 增加 rollback 复杂度
- 超出“身份绑定”的职责范围

---

## 4.3 Provider Endpoint 必须显式建模

Provider 不能只保存 Host。

推荐模型：

```go
type ProviderEndpoint struct {
    Host    string
    SSHUser string
    SSHPort int
}
```

ProviderRef：

```go
type ProviderRef struct {
    Type     ProviderType
    Username string
    Endpoint ProviderEndpoint
}
```

### 默认值

GitHub：

```text
Host    github.com
SSHUser git
SSHPort 22
```

GitLab.com：

```text
Host    gitlab.com
SSHUser git
SSHPort 22
```

自建 Gitea 示例：

```text
Host    git.example.com
SSHUser git
SSHPort 2222
```

### 原因

自建：

- GitLab
- Gitea
- Forgejo
- GitHub Enterprise

很可能使用非 22 SSH 端口。

因此禁止在 Adapter 里写死：

```go
port := 22
```

---

## 4.4 Repository 类型范围必须明确

### V1 正式支持

V1 只正式保证：

> 普通 non-bare Git Repository。

典型目录：

```text
/Users/luna/Projects/luna-site
```

### V1 暂不承诺完整支持

- bare repository
- linked worktree
- 特殊 shared git-dir 布局
- submodule 特殊绑定语义

### Worktree 策略

V1 可以：

```text
识别 -> 明确提示暂不支持
```

不要求：

```text
自动兼容所有 worktree config 语义
```

因为 worktree 涉及：

- common git dir
- worktree-specific config
- shared config
- working tree root 与 git-dir 分离

没有真实需求前不引入该复杂度。

---

## 4.5 禁止假设 `.git` 一定是目录

### 错误实现

禁止：

```go
statePath := filepath.Join(repoRoot, ".git", "gitra", "state.json")
```

因为 `.git` 可能是文件。

### 正确方式

通过 Git 获取实际 Git Dir：

```bash
git -C <path> rev-parse --show-toplevel
git -C <path> rev-parse --absolute-git-dir
```

Repository Runtime Model：

```go
type Repository struct {
    RootPath string
    GitDir   string
    Remotes  []Remote
}
```

Repository metadata 应写入：

```text
<GitDir>/gitra/state.json
```

而不是硬编码：

```text
<Root>/.git/gitra/state.json
```

---

## 4.6 `ssh -T` 成功不能只看 Exit Code

### 问题

GitHub 等 Provider 在 SSH 认证成功后，可能仍返回非 0 exit status。

因此下面逻辑是错误的：

```go
if err != nil {
    return ErrAuthInvalid
}
```

### SSH Adapter 必须返回原始执行结果

```go
type ProcessResult struct {
    ExitCode int
    Stdout   string
    Stderr   string
}
```

SSH：

```go
type SSHTestResult struct {
    ExitCode int
    Stdout   string
    Stderr   string
}
```

### Provider 决定“认证成功”的语义

```text
SSHCLIAdapter
     |
     v
raw stdout / stderr / exit code
     |
     v
ProviderAdapter
     |
     v
Provider-specific parser
     |
     v
authenticated identity
```

建议接口：

```go
type ProviderAdapter interface {
    ParseSSHVerification(
        result SSHTestResult,
    ) (ProviderIdentity, error)
}
```

最后比较：

```text
expected username
actual username
```

---

## 4.7 Remote Owner 不等于 Account Username

合法情况：

```text
Account username: lunafoundry
Remote owner:     some-company
```

只要该账号有 Repository 权限，就完全合法。

因此：

```text
remote owner != provider username
```

不是错误。

### Bind 应验证

- remote 是否为 SSH
- Provider Host 是否兼容
- Account 是否配置完整
- Repository 是否有效

### Bind 不应验证

```text
remote owner == account username
```

Owner 只属于 Repository Metadata，不属于 Account Identity 判断逻辑。

---

## 4.8 TUI 状态禁止依赖实时网络

打开 TUI 时禁止：

```text
account1 -> ssh test
account2 -> ssh test
account3 -> ssh test
```

否则会导致：

- 启动慢
- 网络差时 UI 卡顿
- 离线时本地管理不可用
- 每次打开都产生不必要连接

### TUI 首页只显示本地状态

建议：

```text
Configured
Invalid
```

Configured 判定仅依据：

- Provider 配置完整
- SSH key path 已配置
- SSH key 文件存在
- Commit Identity 完整

### 只有显式操作才联网

TUI：

```text
T -> Test Connection
```

CLI：

```bash
gitra account test lunafoundry
```

### UI 文案

推荐：

```text
● Configured
```

不推荐：

```text
● Online
● Logged in
```

因为本地 SSH 配置有效并不等于实时网络在线，也不等于 Provider OAuth 登录。

---

## 4.9 Path Canonicalization 必须统一

所有输入：

```text
.
./foo
~/Projects/foo
/Users/luna/Projects/foo
repo 的任意子目录
```

最终都必须归一成 Git 认可的 Repository Root。

### Canonicalization 来源

不能只做：

```go
filepath.Abs(path)
```

最终以：

```bash
git -C <path> rev-parse --show-toplevel
```

结果为准。

再执行：

- clean
- absolute normalization

### BindingStore 只保存 Canonical Root

禁止同一个 repo 同时存在：

```text
~/Projects/luna-site
./luna-site
/Users/luna/Projects/luna-site
```

多个路径表示。

---

# 5. V1 行为契约

---

## 5.1 Account Create

输入：

- Alias
- Provider Type
- Provider Endpoint
- Provider Username
- Git name
- Git email
- SSH private key path

流程：

```text
validate
  |
generate AccountID
  |
normalize config
  |
save account
```

默认不进行网络认证。

只有用户显式 Test 时才执行 SSH Test。

---

## 5.2 Account Edit

允许修改：

- Alias
- Provider fields
- Username
- Git name
- Git email
- SSH key path

禁止修改：

```text
AccountID
```

保存后：

```text
Revision + 1
```

然后：

```text
BindingStore.ListByAccount(AccountID)
        |
        v
Reconcile bound repositories
```

---

## 5.3 Account Remove

无 Binding：

```text
直接删除
```

有 Binding：

```text
ErrAccountInUse
```

TUI 可以提供：

```text
Unbind all and remove
```

但必须显式确认。

---

# 6. Bind 行为契约

CLI：

```bash
gitra bind <account-alias> [path]
```

固定流程：

```text
1. Resolve Alias -> AccountID
2. Load Account
3. Discover Repository
4. Canonicalize Repository Root
5. Resolve GitDir
6. Detect Repository type
7. Read remotes
8. Validate SSH transport
9. Validate Provider compatibility
10. Validate Account local config
11. Check existing binding
12. Snapshot managed Git config
13. Build Desired State
14. Apply repo-local Git config
15. Write gitra repository metadata
16. Save central Binding
17. Read back actual state
18. Verify desired == applied
19. Commit success
```

任何一步失败：

```text
rollback
```

---

# 7. Bind 明确不做的事

V1 Bind 禁止：

- 修改 remote URL
- 创建 SSH key
- 上传 SSH public key
- 登录 GitHub / GitLab / Gitea
- 调用 Provider API
- 自动 clone
- 自动 commit
- 自动 push
- 修改 `~/.ssh/config`
- 修改 `~/.gitconfig`
- 自动判断并接管 credential helper

---

# 8. Unbind 行为契约

CLI：

```bash
gitra unbind [path]
```

流程：

```text
1. Discover Repository
2. Resolve canonical root / git dir
3. Locate binding
4. Load original snapshot
5. Restore previous local Git config
6. Remove repository gitra metadata
7. Remove central binding
8. Verify restore result
```

关键原则：

> Unbind 恢复“绑定前状态”，而不是简单删除配置。

---

# 9. Managed State Ownership

V1.0 core（`ssh-key` 路径）只管理以下 Git Config：

```text
user.name
user.email
core.sshCommand
gitra.bindingId
gitra.accountId
gitra.strategy
gitra.version
```

V1.1 起（`https-token` 路径）追加管理，全部限定 repo-local 作用域：

```text
credential.helper                     # 仅当仓库需要时写入，指向系统原生 helper（osxkeychain / manager / libsecret）
credential.<https-url>.username       # URL 作用域内的账号名（多账号隔离的关键）
credential.<https-url>.useHttpPath    # 仅在需要路径维度区分时写入
```

以上全部纳入 Snapshot / Rollback 与 DRIFT 检测范围。

其他配置一律不碰：

```text
remote.*
branch.*
pull.*
merge.*
fetch.*
用户的全局 credential 配置（只读不改）
```

如果绑定期间用户手工修改 gitra 管理字段：

```text
BindingHealth = DRIFT
```

---

# 10. Repository Metadata

建议写入 local Git config：

```ini
[gitra]
    bindingId = bnd_xxx
    accountId = acc_xxx
    strategy = repo-local
    version = 1
```

注意：

> 存 AccountID，不存 Alias 作为主引用。

Alias 可以随时修改。

---

# 11. Snapshot 设计

Bind 前必须记录原有 local config。

位置：

```text
<GitDir>/gitra/state.json
```

示例：

```json
{
  "schema_version": 1,
  "binding_id": "bnd_xxx",
  "previous": {
    "user.name": {
      "exists": true,
      "value": "Old Name"
    },
    "user.email": {
      "exists": true,
      "value": "old@example.com"
    },
    "core.sshCommand": {
      "exists": false
    }
  }
}
```

Unbind 必须根据该 Snapshot 恢复。

---

# 12. Reconcile 行为契约

Reconciler 处理：

```text
Desired State -> Applied State
```

输入：

```text
Account + Binding
```

输出目标：

```text
repository local git config
```

流程：

```text
Build Desired
    |
Read Actual
    |
Compare
    |
+--- same -> OK
|
+--- diff -> DRIFT -> Apply -> Verify
```

Reconcile 只能修改 gitra ownership 范围内的 key。

---

# 13. Binding Health

```go
type BindingHealth string

const (
    HealthOK      BindingHealth = "ok"
    HealthDrift   BindingHealth = "drift"
    HealthMissing BindingHealth = "missing"
    HealthBroken  BindingHealth = "broken"
)
```

定义：

### OK

Desired State 与实际配置一致。

### DRIFT

Repository 存在，但 gitra 管理字段被外部修改。

### MISSING

BindingStore 记录存在，但旧路径已不存在。

### BROKEN

例如：

- SSH key 不存在
- Account 缺配置
- Git config 无法读取
- Provider Endpoint 无效

---

# 14. SSH Command 生成规范

V1 `ssh-key` Strategy 输出：

```text
ssh -i "<absolute-key-path>" -o IdentitiesOnly=yes
```

端口规则（**修订 1**，实测依据）：

- remote URL 自带端口（`ssh://git@host:2222/...`）时 Git 会自行追加 `-p`：此时**不要**在 `core.sshCommand` 中再写 `-p`
- remote URL 不带端口（scp-like）且 Endpoint 端口 ≠ 22 时：追加 `-p <endpoint-port>`
- OpenSSH 对重复 `-p` 取**首次出现**的值（`ssh -G -p 1111 -p 2222` → port 1111）：写死端口会覆盖 URL 显式端口
- `core.sshCommand` 作用于该仓库所有 SSH remote：bind 时发现其他 host 的 SSH remote 必须给出警告

Private key path 在写入前应标准化成绝对路径。

---

# 15. V1 不提前支持 ssh-agent

V1 Transport Strategy 只有：

```text
ssh-key
```

暂不实现：

- ssh-agent
- hardware key
- HTTPS token
- macOS Keychain credential transport
- Windows Credential Manager transport

架构保留 Strategy 扩展点即可。

---

# 16. Provider Adapter 最低职责

V1 Provider Adapter 只负责：

1. Provider 类型与默认 Endpoint
2. SSH remote 匹配
3. SSH remote 解析
4. SSH Test 响应解析
5. authenticated username 提取

不负责：

- API 登录
- Repository API
- OAuth
- 上传 SSH key
- 用户 profile API

---

# 17. Remote Parser 输出

```go
type RemoteRepositoryRef struct {
    Host      string
    Port      int
    OwnerPath string
    RepoName  string
}
```

例如：

```text
git@github.com:openai/example.git
```

解析：

```text
Host      github.com
Port      22
OwnerPath openai
RepoName  example
```

`OwnerPath` 不参与账号合法性判断。

---

# 18. Process Execution 抽象

建议抽离统一进程执行接口：

```go
type CommandRunner interface {
    Run(
        ctx context.Context,
        name string,
        args ...string,
    ) (ProcessResult, error)
}
```

```go
type ProcessResult struct {
    ExitCode int
    Stdout   string
    Stderr   string
}
```

用途：

```text
GitCLIAdapter
SSHCLIAdapter
```

收益：

- 进程执行细节集中
- stdout / stderr / exit code 语义统一
- 容易 Fake
- 容易做 timeout / cancellation
- Provider 不直接依赖 `os/exec`

---

# 19. Go 技术选择

## 19.1 CLI

使用：

```text
github.com/spf13/cobra
```

职责：

- command tree
- flags
- help
- argument validation 的展示层部分
- exit code 交付

不承担业务逻辑。

---

## 19.2 TUI

使用：

```text
github.com/charmbracelet/bubbletea
github.com/charmbracelet/lipgloss
```

可选：

```text
github.com/charmbracelet/bubbles
```

只有确认需要以下组件时再引入：

- text input
- list
- viewport

---

## 19.3 Core

优先使用 Go Standard Library：

- context
- os
- os/exec
- path/filepath
- encoding/json
- errors
- fmt
- io
- sync
- testing
- runtime

---

# 20. V1 明确不引入的依赖

不使用：

```text
ORM
SQL Database
DI Framework
Event Bus
Config Framework
Plugin Framework
HTTP Server
Web Framework
复杂 Logging Framework
```

除非后续出现明确需求。

---

# 21. 第一批 Package 规划

优先创建：

```text
internal/domain
internal/ports
internal/app

internal/adapters/gitcli
internal/adapters/sshcli
internal/adapters/storage

internal/adapters/provider/github
internal/adapters/provider/gitlab
internal/adapters/provider/gitea

internal/strategies/auth/sshkey
internal/strategies/routing/repolocal

internal/bootstrap
```

CLI / TUI 后建：

```text
internal/presentation/cli
internal/presentation/tui
```

---

# 22. 依赖方向强约束

允许：

```text
presentation -> app
app -> domain
app -> ports
adapters -> ports/domain
strategies -> ports/domain
bootstrap -> concrete implementations
```

禁止：

```text
domain -> adapter
domain -> cobra
domain -> bubbletea
app -> gitcli
app -> sshcli
app -> JSON concrete storage
app -> GitHub concrete adapter
```

---

# 23. Domain 不允许知道的实现细节

`internal/domain` 不应 import 或感知：

- Cobra
- Bubble Tea
- Lip Gloss
- `os/exec`
- Git CLI concrete adapter
- SSH concrete process wrapper
- JSON 文件路径
- GitHub / GitLab SDK
- TUI ViewModel

Domain 只表达业务语义。

---

# 24. Bootstrap 是唯一 Composition Root

依赖组装统一在：

```text
internal/bootstrap
```

或由：

```text
cmd/gitra/main.go
```

调用 bootstrap。

不要在业务模块里自行 `new` concrete adapter。

---

# 25. Application Interface Baseline

## 25.1 AccountService

```go
type AccountService interface {
    Create(ctx context.Context, req CreateAccountRequest) (Account, error)
    Update(ctx context.Context, req UpdateAccountRequest) (Account, error)
    Delete(ctx context.Context, req DeleteAccountRequest) error
    Get(ctx context.Context, id AccountID) (Account, error)
    GetByAlias(ctx context.Context, alias string) (Account, error)
    List(ctx context.Context) ([]Account, error)
}
```

## 25.2 BindingService

```go
type BindingService interface {
    Bind(ctx context.Context, req BindRequest) (RepositoryBinding, error)
    Unbind(ctx context.Context, req UnbindRequest) error
    Status(ctx context.Context, path string) (BindingStatus, error)
    ListByAccount(ctx context.Context, id AccountID) ([]RepositoryBinding, error)
}
```

## 25.3 VerificationService

```go
type VerificationService interface {
    VerifyAccount(
        ctx context.Context,
        id AccountID,
    ) (VerificationResult, error)
}
```

## 25.4 ReconcileService

```go
type ReconcileService interface {
    ReconcileBinding(
        ctx context.Context,
        id BindingID,
    ) (ReconcileResult, error)

    ReconcileAccount(
        ctx context.Context,
        id AccountID,
    ) (ReconcileResult, error)
}
```

---

# 26. Request DTO Baseline

```go
type CreateAccountRequest struct {
    Alias     string
    Provider  ProviderRef
    Identity  CommitIdentity
    Transport TransportConfig
}
```

```go
type BindRequest struct {
    AccountID AccountID
    Path      string
}
```

```go
type UnbindRequest struct {
    Path string
}
```

---

# 27. Git Port Baseline

```go
type Git interface {
    DiscoverRepository(
        ctx context.Context,
        path string,
    ) (Repository, error)

    GetLocalConfig(
        ctx context.Context,
        repo Repository,
        key string,
    ) (string, bool, error)

    SetLocalConfig(
        ctx context.Context,
        repo Repository,
        key string,
        value string,
    ) error

    UnsetLocalConfig(
        ctx context.Context,
        repo Repository,
        key string,
    ) error

    Remotes(
        ctx context.Context,
        repo Repository,
    ) ([]Remote, error)
}
```

---

# 28. Storage 基线

V1 不使用数据库。

配置目录：

```text
os.UserConfigDir()/gitra
```

典型 macOS：

```text
~/Library/Application Support/gitra
```

若项目最终决定强制 XDG 风格，可在实现前统一调整，但不要在代码中散落路径规则。

建议结构：

```text
gitra/
├── accounts/
│   ├── acc_xxx.json
│   └── acc_yyy.json
├── bindings.json
└── .lock
```

### Account 文件使用 ID 命名

推荐：

```text
acc_xxx.json
```

而不是：

```text
lunafoundry.json
```

避免 Alias 改名导致文件改名逻辑。

---

# 29. Atomic Write

所有持久化 JSON 写入：

```text
write temp
   |
fsync
   |
rename
```

例如：

```text
bindings.json.tmp
    -> fsync
    -> rename
    -> bindings.json
```

避免进程中断导致 JSON 半写入。

---

# 30. Schema Version

所有持久化数据必须包含：

```json
{
  "schema_version": 1
}
```

未来迁移接口：

```go
type Migration interface {
    From() int
    To() int
    Migrate([]byte) ([]byte, error)
}
```

V1 不需要复杂 migration engine，但格式必须从第一天可版本化。

---

# 31. 并发限定

V1 不处理服务端多用户并发。

只处理：

> 同一 OS 用户同时启动多个 gitra 进程。

写操作必须使用进程间文件锁。

写操作包括：

- CreateAccount
- UpdateAccount
- DeleteAccount
- Bind
- Unbind
- Reconcile

接口建议：

```go
type Locker interface {
    WithWriteLock(
        ctx context.Context,
        fn func() error,
    ) error
}
```

---

# 32. Transaction / Rollback 基线

Bind 必须具有“要么全部成功，要么恢复原状”的语义。

禁止出现：

```text
user.email 已修改
core.sshCommand 未修改
BindingStore 已写入
```

推荐流程：

```text
Snapshot
   |
Apply local config
   |
Write repo metadata
   |
Write central binding
   |
Verify
   |
Success
```

任意步骤失败：

```text
Restore Snapshot
Remove partial metadata
Remove partial central binding
```

---

# 33. Repository Path Move

BindingStore 保存：

```text
binding ID
account ID
canonical path
```

Repository 自身保存：

```text
bindingId
accountId
```

当 Repository 被移动后：

```bash
cd /new/path/repo
gitra status
```

若读取到相同 BindingID，可以更新中央 path。

这不是后台扫描，而是当前 repo 的自恢复。

---

# 34. 错误模型基线

必须区分：

```text
User Error
Environment Error
External Process Error
State Conflict
Unsupported Feature
Internal Error
```

建议 typed errors：

```go
var (
    ErrAccountNotFound   = errors.New("account not found")
    ErrAccountExists     = errors.New("account already exists")
    ErrAccountInUse      = errors.New("account has active bindings")

    ErrNotGitRepository  = errors.New("not a git repository")
    ErrUnsupportedRepo   = errors.New("repository type is unsupported")
    ErrUnsupportedRemote = errors.New("remote transport is unsupported")
    ErrAlreadyBound      = errors.New("repository already bound")
    ErrBindingNotFound   = errors.New("binding not found")

    ErrAuthInvalid       = errors.New("authentication invalid")
    ErrProviderMismatch  = errors.New("provider mismatch")
    ErrIdentityMismatch  = errors.New("identity mismatch")

    ErrStateDrift        = errors.New("managed state drift")
)
```

---

# 35. CLI Exit Code 基线

```text
0 success
1 internal/general error
2 invalid CLI input
3 account error
4 repository error
5 authentication error
6 binding/state conflict
7 unsupported operation
```

JSON 模式使用同样 exit code。

---

# 36. 错误信息原则

错误必须告诉用户：

```text
发生了什么
为什么
用户应该怎么处理
```

例如 HTTPS remote：

错误：

```text
bind failed
```

正确：

```text
Repository remote uses HTTPS.
gitra V1 supports SSH remotes only.
Change the repository remote to SSH before binding.
```

SSH key 不存在：

```text
SSH private key does not exist:
~/.ssh/id_ed25519_lunafoundry
```

Git 不存在：

```text
Git executable was not found.
Install Git and retry.
```

---

# 37. CLI 基线

V1 命令：

```bash
gitra

gitra account add
gitra account list
gitra account show <alias>
gitra account edit <alias>
gitra account remove <alias>
gitra account test <alias>

gitra bind <alias> [path]
gitra unbind [path]
gitra status [path]
```

V1.1 / V1.2 增补（见增补设计）：

```bash
gitra login [--provider github|gitlab|gitea]
gitra logout <alias>
gitra clone <url> [dir]
```

查询类支持：

```text
--json
```

通用调试：

```text
--verbose
```

---

# 38. TUI Presentation Contract

TUI 只属于 Presentation Adapter。

禁止：

```text
TUI -> os/exec git
TUI -> ssh
TUI -> accounts.json
TUI -> bindings.json
TUI -> .git/config
```

正确：

```text
TUI
 -> Controller
 -> Application Service
 -> Port
 -> Adapter
```

---

# 39. TUI 首页卡片数据

```go
type AccountCardViewModel struct {
    ID           string
    Provider     string
    Alias        string
    Host         string
    AuthType     string
    LocalState   string
    ProjectCount int
}
```

首页只展示：

- Provider
- Alias
- Host
- Auth 类型
- Configured / Invalid
- Project Count

不展示：

- last commit
- branch
- dirty state
- remote repository count
- API profile
- realtime online 状态

---

# 40. TUI 页面范围

V1 只有：

## Accounts

账号卡片列表。

## Account Detail

账号配置 + 已绑定项目。

## Add/Edit Account

添加或修改账号。

## Add Project

选择本地目录并 Bind。

## Confirm Dialog

删除账号、移除项目等危险操作确认。

V1.1 / V1.2 增补（见增补设计）：

## Login

浏览器授权 / 设备码 / 等待 / 成功。

## Onboarding

首次启动引导：登录 -> 克隆或打开仓库 -> 自动绑定。

不做更多页面。

---

# 41. V1 Provider Scope

架构支持 Provider Registry。

V1 最低 Provider：

```text
GitHub
GitLab
Gitea
```

但 Provider 能力只覆盖：

```text
SSH endpoint defaults
remote parsing
SSH verification parsing
```

不实现 Provider API。

---

# 42. 跨平台基线

首要平台：

```text
macOS
Linux
```

Windows 保持代码可移植，但 V1 不作为主要验收平台。

必须使用：

```go
filepath
os.UserConfigDir
os.UserHomeDir
```

禁止散落硬编码：

```text
/Users/...
/home/...
```

---

# 43. 性能目标

这是本地小型工具，不进行过度优化。

目标：

```text
CLI 查询：本地逻辑尽量 < 100ms，不计真实 git 子进程耗时
TUI 启动：不发网络请求
几十个账号：无压力
数百个 Binding：可接受
```

V1 不需要数据库。

---

# 44. 开发里程碑总览

严格建议：

```text
Milestone 0   Skeleton
Milestone 1   Core Identity Binding
Milestone 2   CLI
Milestone 2.5 登录 MVP（PAT + SecretStore + 凭据投影）
Milestone 3   Provider Verification
Milestone 3.5 浏览器登录（device flow / loopback + 自动填充）
Milestone 4   TUI
Milestone 4.5 引导式克隆 + 错误白话化
Milestone 5   Stability
```

开发优先级：

```text
核心机制正确 > CLI 可用 > TUI 好用 > 视觉优化
```

---

# 45. Milestone 0 - Skeleton

## 目标

建立正确代码边界，不追求功能完整。

## 工作

创建：

```text
internal/domain
internal/ports
internal/app
internal/adapters
internal/strategies
internal/bootstrap
```

定义：

- Account
- AccountID
- Binding
- BindingID
- Repository
- Remote
- ProviderRef
- ProviderEndpoint
- CommitIdentity
- TransportConfig

## 验收

```bash
go test ./...
go vet ./...
```

通过。

## 非目标

此阶段不做：

- TUI
- 联网认证
- 完整 CLI
- UI 样式

---

# 46. Milestone 1 - Core Identity Binding

## 目标

先证明产品最核心的技术命题：

> 不切换账号，也能让不同 Repository 自动使用不同 Git 身份。

## 实现

完成：

- AccountStore
- BindingStore
- CommandRunner
- GitCLIAdapter
- SSHCLIAdapter 基础
- SSHKeyAuthStrategy
- RepoLocalRoutingStrategy
- Snapshot
- Rollback
- BindingService
- Reconciler 基础

## 核心 E2E

准备：

```text
Account A
SSH Key A

Account B
SSH Key B

Repo A
Repo B
```

绑定：

```text
Repo A -> Account A
Repo B -> Account B
```

验证：

```bash
git -C repo-a config --local user.email
git -C repo-b config --local user.email

git -C repo-a config --local core.sshCommand
git -C repo-b config --local core.sshCommand
```

必须完全隔离。

如果有真实可访问 remote，再验证：

```bash
git -C repo-a fetch
git -C repo-b fetch
```

分别使用正确身份。

## Definition of Done

- 双账号共存
- 双仓库绑定成功
- Git identity 不串
- SSH key 不串
- Unbind 可恢复
- 失败可 rollback

---

# 47. Milestone 2 - CLI

## 目标

没有 TUI 的情况下，产品已经完整可用。

## 实现

```text
account add
account list
account show
account edit
account remove
account test
bind
unbind
status
--json
--verbose
```

## 验收流程

```text
Create Account
 -> Bind Repository
 -> git fetch/pull/push
 -> Edit Account
 -> Reconcile
 -> Status
 -> Unbind
 -> Delete Account
```

全流程可完成。

---

# 48. Milestone 3 - Provider Verification

## 目标

支持显式账号连接验证。

CLI：

```bash
gitra account test <alias>
```

Provider：

- GitHub
- GitLab
- Gitea

## 必须能区分

```text
local config invalid
authentication success
username mismatch
authentication failure
network/process failure
unexpected provider response
```

TUI 首页仍然不主动联网。

---

# 49. Milestone 4 - TUI

## 目标

实现类 GUI 的卡片式终端体验。

## Accounts 页面

卡片显示：

```text
Provider
Alias
Host
Local Config State
Project Count
```

## Account Detail

显示：

```text
Provider
Endpoint
Username
Commit Identity
SSH Key
Bound Projects
```

## 操作

```text
Add Account
Edit Account
Remove Account
Test Connection
Add Project
Remove Project
```

## 非目标

不显示：

- history
- branch
- commit
- diff
- PR
- Issue
- working tree status

---

# 50. Milestone 5 - Stability

完成：

- Atomic file write
- file lock
- drift detection
- path canonicalization
- repository move recovery
- schema version / migration baseline
- typed errors
- stable JSON output
- cross-platform path handling
- regression tests

---

# 51. Definition of Ready

一个功能进入开发前必须满足：

1. 业务目标明确
2. 属于 V1 Scope
3. 输入明确
4. 输出明确
5. 错误行为明确
6. 是否修改外部状态明确
7. rollback 行为明确
8. 所属模块明确
9. Adapter / Strategy / Application 边界明确
10. 有明确验收测试

不满足则不直接编码。

---

# 52. Definition of Done

一个 Application Use Case 完成必须满足：

```text
功能正确
+
Unit Test
+
必要 Integration Test
+
错误可解释
+
无跨层依赖污染
+
无未定义副作用
```

Binding 类功能额外要求：

```text
Rollback Test 通过
```

---

# 53. 测试基线

## 53.1 Domain Unit Test

必须覆盖：

- AccountID
- Alias validation
- Alias uniqueness
- Provider endpoint validation
- SSH port validation
- Transport config validation
- Binding invariant
- BindingHealth
- Strategy registry

---

## 53.2 Application Unit Test

使用 Fake：

```text
FakeAccountStore
FakeBindingStore
FakeGit
FakeSSH
FakeAuthStrategy
FakeRoutingStrategy
FakeProvider
FakeLocker
```

覆盖：

- CreateAccount
- EditAccount
- DeleteAccount
- AccountInUse
- Bind
- AlreadyBound
- UnsupportedRemote
- Unbind
- Reconcile
- Drift
- Rollback

---

## 53.3 Git Adapter Integration Test

临时目录：

```bash
git init
```

验证：

- DiscoverRepository
- RootPath
- GitDir
- GetLocalConfig
- SetLocalConfig
- UnsetLocalConfig
- Remotes

---

## 53.4 SSH Adapter Test

Adapter 层只关心：

```text
command construction
stdout
stderr
exit code
```

Provider-specific 成功判断不放 SSH Adapter。

---

## 53.5 Provider Adapter Test

使用 fixture：

```text
GitHub success response
GitHub failure response
GitLab success response
Gitea success response
unexpected response
```

验证：

- authenticated username parsing
- failure classification

---

## 53.6 Rollback Test

至少模拟：

```text
config write 1 success
config write 2 success
binding store write failure
```

最终 Repository 必须恢复原状。

---

## 53.7 双账号双仓库 E2E

这是 V1 最重要的长期 regression test。

必须永久保留。

## 53.8 HTTPS 凭据隔离 E2E（修订 1，V1.1 起）

同一 host（例如 github.com）两个仓库绑定两个账号：

```text
Repo A -> Account A（凭据 A）
Repo B -> Account B（凭据 B）
```

验证：

- 在 Repo A / Repo B 内执行凭据查询，各自返回正确的 username / secret
- 多账号场景不写入 host 级全局凭据选择
- 移除 gitra 二进制后，凭据取回仍由系统 helper 完成
- `gitra logout` 后凭据条目被删除、repo-local 受管配置被清理

必须永久保留。

---

# 54. 变更控制

开发过程中任何新需求必须先判断：

### A. 是否直接服务核心目标？

```text
Account -> Repository Binding -> Native Git identity
```

如果不是，默认不进入 V1。

### B. 是否真实遇到？

如果只是：

```text
以后可能有用
```

只保留扩展点，不实现。

---

# 55. 明确禁止“顺手加”的功能

V1 开发过程中不要顺手实现：

- repository browser
- history
- branch status
- GitHub repository API list
- remote auto conversion
- SSH key generator wizard
- SSH key upload
- automatic repo scanning
- tray app
- daemon
- auto updater
- telemetry
- plugin system
- cloud sync

> 修订 1：`clone manager`、`OAuth`、`HTTPS token` 已在增补设计中立项（V1.1 / V1.2），不属于「顺手加」；
> 但仍须按增补设计的验收标准实施，不得提前夹带。

---

# 56. “功能验证优先”在本项目中的含义

开发顺序必须是：

```text
先验证核心身份绑定机制
        |
再实现 CLI
        |
最后实现 TUI
```

禁止：

```text
先把卡片界面做漂亮
再验证 git 是否真的走正确账号
```

V1 第一技术风险是：

> 不同 Repository 是否能稳定、可恢复地使用不同 Git Identity。

必须优先消除这个风险。

---

# 57. “不做防御性编程”在本项目中的含义

不是“不处理错误”。

而是：

> 不为没有明确需求的异常世界提前构造复杂系统。

必须处理：

- Git command failure
- SSH key missing
- repository invalid
- JSON write failure
- rollback
- duplicate binding
- account missing

不提前处理：

- 所有奇怪 git-dir 布局
- 自动修复损坏 Git Repository
- 自动判断所有 credential helper
- 任意版本旧配置迁移
- 所有 Provider 私有部署差异

---

# 58. 推荐最终开发顺序

严格建议：

```text
1. Domain
2. Ports
3. Storage
4. CommandRunner
5. Git Adapter
6. SSH Adapter
7. SSH Auth Strategy
8. Routing Strategy
9. Binding Service
10. 双账号双仓库 E2E
11. Account Service
12. Reconciler
13. CLI
14. Provider Verification
15. TUI
16. Stability
```

---

# 59. 核心验收场景

## Scenario A - 两个账号

```text
GitHub Account A
GitHub Account B
```

分别使用不同 SSH Key。

## Scenario B - 两个 Repo

```text
Repo A -> Account A
Repo B -> Account B
```

## Scenario C - 无感 Git

连续运行：

```bash
cd repo-a
git fetch

cd ../repo-b
git fetch
```

过程中不执行：

```text
gh auth switch
ssh config switch
environment switch
gitra switch
```

两者都使用正确身份。

## Scenario D - Account Update

修改 Account A 邮箱：

```text
old@example.com
->
new@example.com
```

Reconcile 后：

```text
Repo A 更新
Repo B 不受影响
```

## Scenario E - Unbind

Unbind Repo A 后恢复 bind 前：

```text
user.name
user.email
core.sshCommand
```

## Scenario F - gitra 不常驻

退出 TUI，甚至完全没有 `gitra` 进程。

再次：

```bash
git push
```

仍正常。

这证明：

> gitra 不是 Git Runtime dependency。

---

# 60. V1 最终 Definition of Done

只有全部满足才认为 V1 完成：

- 多账号可管理
- AccountID 与 Alias 分离
- GitHub / GitLab / Gitea Provider 基础适配
- SSH 与 HTTPS 两条认证路径边界明确（HTTPS 为 V1.1 主路径）
- 不自动修改 remote URL
- Provider Endpoint 支持 SSH Port
- Repo-local binding 成功
- 不假设 `.git` 为目录
- 双账号双仓库完全隔离
- Unbind 正确恢复
- Account Update 可 Reconcile
- SSH Test 不依赖单一 exit code
- Remote Owner 不作为 Account Username 校验
- TUI 启动不依赖网络
- Path Canonicalization 统一
- Rollback 可用
- Atomic persistence
- File lock
- JSON schema version
- CLI 完整
- `--json` 完整
- TUI 卡片式管理完整
- `git push/pull/fetch` 不依赖 gitra 进程
- 无 Web
- 无 daemon
- 无自建账号体系（登录仅走平台浏览器授权）
- 无 Git history 等无关功能

V1.1 / V1.2 追加 DoD（见增补设计）：

- 新手从零到 push 成功，除浏览器授权外零输入
- Token 仅存系统 SecretStore，不落明文、不进日志
- 同 host 多账号凭据隔离 E2E（§53.8）通过
- 登出后受管凭据与配置被完整清理

---

# 61. 最终原则

开发过程中始终判断：

> 这个功能是否直接让“多账号 Repository 无感使用正确 Git 身份”变得更可靠、更简单？

如果答案是否定的：

> V1 不做。

最终产品保持：

```text
极简
确定
透明
可恢复
易扩展
不侵入 Git 正常工作流
```

---

# 62. 与主架构文档的关系

主架构文档：

```text
gitra-architecture-development-design.md
```

回答：

```text
系统为什么这样设计
整体架构是什么
模块怎么拆分
接口怎么组织
```

本文：

```text
gitra-v1-implementation-baseline.md
```

回答：

```text
第一版到底怎么实现
哪些行为已经锁定
哪些不能做
按什么顺序开发
做到什么算完成
```

三份文档共同构成 `gitra` 的正式开发输入：

```text
gitra-architecture-development-design.md   系统设计、扩展点
gitra-v1-implementation-baseline.md        V1.0 开发基线（本文件）
gitra-v1-login-onboarding-design.md        登录与新手引导增补设计（V1.1 / V1.2）
```
