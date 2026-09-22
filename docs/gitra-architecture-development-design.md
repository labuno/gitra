# gitra 设计文档

> 极简 Git 多账号身份管理器（CLI + TUI）
>
> 版本：V1.0 Design Draft  
> 技术栈：Go + Native Git/SSH + Bubble Tea + Lip Gloss  
> 核心原则：**Bind once, Git forever**

> **增补说明（2026-09-22）**：项目统一更名为 `gitra`。登录、HTTPS 凭据与新手引导的增补设计见
> `gitra-v1-login-onboarding-design.md`（V1.1 / V1.2）；与之冲突处以增补文档为准。
> 本文 §2.2、§11、§28、§32、§45.2、§48、§53、§55 已同步修订。

---

## 1. 项目定位

`gitra` 是一个极简的 Git 多账号身份管理器。

它解决的问题只有一个：

> 在同一台电脑上管理多个 Git 托管账号，并让每个本地 Git Repository 永久绑定到正确账号，使 `git pull / push / fetch`、Codex、Cursor、VS Code、Shell 脚本等在不切换账号的情况下自动使用正确身份。

它不是 Git GUI，不负责 Git 历史、分支、PR、Issue，也不代理 Git 命令。

---

## 2. 产品目标

### 2.1 核心目标

1. 管理多个 Git 托管账号。
2. 支持 GitHub、GitLab、Gitea 等 Provider。
3. 每个账号保存独立：
   - Provider
   - Host
   - Username
   - Git commit identity
   - Git transport authentication
4. 将本地 Repository 绑定到账号。
5. 绑定后不需要 `gitra` 常驻。
6. 原生 `git pull / push / fetch` 自动使用正确账号。
7. 提供：
   - CLI：脚本、Agent、自动化使用
   - TUI：人工可视化管理
8. 核心业务逻辑 CLI/TUI 共享。

### 2.2 非目标

V1 明确不做：

- Git commit history
- Diff viewer
- Staging
- Branch graph
- Branch manager
- Merge/Rebase UI
- Pull Request
- Issue
- Repository 浏览器
- GitHub Dashboard
- Terminal
- Workspace
- Editor
- AI 功能
- Web UI
- Electron/Tauri
- Daemon
- 后台常驻服务
- OAuth Provider API 登录（V1.0；V1.1 起立项为浏览器登录，见增补设计）
- Git remote 自动同步
- Repository cloning assistant（V1.0；V1.2 起立项为引导式克隆，见增补设计）

---

# 3. 核心产品原则

## 3.1 Repository 决定 Account

系统中不应该存在：

```text
Current Active Account
```

正确模型是：

```text
luna-site      -> lunafoundry
company-api    -> work
novel-studio   -> personal
```

即：

```text
Repository -> Account
```

而不是：

```text
Global Current Account -> Repository
```

---

## 3.2 Bind Once, Git Forever

`gitra` 只参与：

```text
账号管理
+
Repository 绑定
+
配置同步
```

绑定完成之后：

```bash
git pull
git push
git fetch
```

直接运行原生 Git。

禁止设计：

```bash
gitra push
gitra pull
gitra fetch
```

---

## 3.3 Native Git 是 Runtime

`gitra` 不实现 Git 协议。

实际 Git 行为永远交给用户机器上的：

```text
git
ssh
```

因此：

```text
gitra 负责配置
git 负责运行
```

---

## 3.4 Desired State 与 Applied State 分离

业务真实状态：

```text
Account
Binding
```

实际机器状态：

```text
.git/config
```

关系：

```text
Desired State
    |
    v
Reconciler
    |
    v
Applied State
```

`.git/config` 只是 Desired State 的投影。

---

## 3.5 架构可扩展，功能不提前扩展

架构必须允许未来支持：

- GitLab
- GitHub Enterprise
- Gitea / Forgejo
- HTTPS Token
- SSH Agent
- Keychain
- Directory Routing
- Remote Owner Routing
- GUI/Web

但 V1 不提前实现没有真实需求的功能。

---

# 4. V1 产品形态

## 4.1 默认运行

```bash
gitra
```

启动 TUI。

---

## 4.2 CLI 模式

```bash
gitra account add
gitra account list
gitra account show lunafoundry
gitra account edit lunafoundry
gitra account remove lunafoundry
gitra account test lunafoundry

gitra bind lunafoundry
gitra bind lunafoundry ~/Projects/luna-site

gitra unbind
gitra unbind ~/Projects/luna-site

gitra status
gitra status ~/Projects/luna-site
```

机器使用：

```bash
gitra status --json
gitra account list --json
```

---

# 5. TUI 设计

## 5.1 Accounts 首页

```text
 Accounts                                      [+ Add Account]

+--------------------------------+
| GitHub                         |
|                                |
| lunafoundry                    |
| ● Configured                        |
|                                |
| github.com                     |
| SSH · id_ed25519_lunafoundry  |
|                                |
| 6 projects                     |
|                                |
| Enter Open              ...    |
+--------------------------------+

+--------------------------------+
| GitLab                         |
|                                |
| work                           |
| ● Configured                        |
|                                |
| gitlab.company.com             |
| SSH · id_ed25519_work         |
|                                |
| 12 projects                    |
+--------------------------------+

↑↓←→ Navigate  Enter Open  A Add  Q Quit
```

响应式：

```text
宽终端 -> 多列卡片
中终端 -> 两列
窄终端 -> 单列
```

---

## 5.2 Account Detail

```text
 Accounts / GitHub · lunafoundry

+------------------------------------------------------+
| GitHub · lunafoundry                     ● Configured     |
|                                                      |
| Host       github.com                                |
| Username   lunafoundry                               |
| Identity   Luna <xxx@example.com>                    |
| Auth       SSH · ~/.ssh/id_ed25519_lunafoundry      |
+------------------------------------------------------+


 Projects                                      [+ Add]

 > luna-site
   ~/Projects/luna-site
   git@github.com:lunafoundry/luna-site.git

   novel-studio
   ~/Projects/novel-studio
   git@github.com:lunafoundry/novel-studio.git

E Edit   A Add Project   D Remove   T Test   Esc Back
```

---

## 5.3 Add Account

```text
+--------------- Add Account ----------------+
|                                            |
| Provider                                   |
| > GitHub                                   |
|   GitLab                                   |
|   Gitea                                    |
|                                            |
| Alias                                      |
| [ lunafoundry________________________ ]    |
|                                            |
| Username                                   |
| [ lunafoundry________________________ ]    |
|                                            |
| Host                                       |
| [ github.com_________________________ ]    |
|                                            |
| Git Name                                   |
| [ Luna_______________________________ ]    |
|                                            |
| Git Email                                  |
| [ xxx@example.com____________________ ]    |
|                                            |
| SSH Private Key                            |
| [ ~/.ssh/id_ed25519_lunafoundry______ ]   |
|                                            |
|             [ Test ] [ Save ]              |
+--------------------------------------------+
```

---

## 5.4 Add Project

使用本地目录选择 TUI：

```text
+------------ Select Project ---------------+
| ~/Projects                                |
|                                           |
|   ai-daily-digest                         |
| > luna-site                               |
|   novel-studio                            |
|                                           |
| Enter Select                   Esc Cancel |
+-------------------------------------------+
```

选择后：

```text
+------------- Bind Project ----------------+
|                                           |
| Repository                                |
| luna-site                                 |
| ~/Projects/luna-site                      |
|                                           |
| Remote                                    |
| git@github.com:lunafoundry/luna-site.git  |
|                                           |
| Account                                   |
| lunafoundry                               |
|                                           |
|                  [ Bind ]                 |
+-------------------------------------------+
```

---

# 6. 总体架构

采用：

> Hexagonal Architecture / Ports & Adapters

同时使用：

- Adapter Pattern
- Strategy Pattern
- Registry Pattern
- Reconciler Pattern
- Application Service
- Repository Pattern
- Composition Root

总体结构：

```text
                    Presentation
                         |
              +----------+----------+
              |                     |
            CLI                   TUI
              |                     |
              +----------+----------+
                         |
                         v
                 Application Layer
       +-----------------+------------------+
       |                 |                  |
 AccountService    BindingService     Reconciler
       |                 |                  |
       +-----------------+------------------+
                         |
                         v
                       Domain
       +-----------------+------------------+
       |                 |                  |
     Account          Binding          Repository
                         |
                         v
                    Strategies
       +-----------------+------------------+
       |                 |                  |
 AuthStrategy    RoutingStrategy     Policies
                         |
                         v
                       Ports
       +-----------------+------------------+
       |                 |                  |
      Git               SSH             Storage
       |                 |                  |
       v                 v                  v
   Git CLI            ssh CLI        Filesystem/JSON
```

---

# 7. 推荐目录结构

```text
gitra/
├── cmd/
│   └── gitra/
│       └── main.go
│
├── internal/
│   ├── domain/
│   │   ├── account.go
│   │   ├── binding.go
│   │   ├── repository.go
│   │   ├── identity.go
│   │   ├── provider.go
│   │   ├── auth.go
│   │   ├── health.go
│   │   └── errors.go
│   │
│   ├── app/
│   │   ├── account_service.go
│   │   ├── binding_service.go
│   │   ├── status_service.go
│   │   ├── verification_service.go
│   │   └── reconcile_service.go
│   │
│   ├── ports/
│   │   ├── git.go
│   │   ├── ssh.go
│   │   ├── account_store.go
│   │   ├── binding_store.go
│   │   ├── provider.go
│   │   ├── secret_store.go
│   │   ├── filesystem.go
│   │   └── clock.go
│   │
│   ├── strategies/
│   │   ├── auth/
│   │   │   ├── registry.go
│   │   │   └── ssh_key.go
│   │   │
│   │   ├── routing/
│   │   │   ├── registry.go
│   │   │   └── repo_local.go
│   │   │
│   │   └── conflict/
│   │       └── managed.go
│   │
│   ├── adapters/
│   │   ├── gitcli/
│   │   │   └── git.go
│   │   │
│   │   ├── sshcli/
│   │   │   └── ssh.go
│   │   │
│   │   ├── provider/
│   │   │   ├── github/
│   │   │   ├── gitlab/
│   │   │   └── gitea/
│   │   │
│   │   └── storage/
│   │       ├── account_json.go
│   │       └── binding_json.go
│   │
│   ├── presentation/
│   │   ├── cli/
│   │   │   ├── root.go
│   │   │   ├── account.go
│   │   │   ├── bind.go
│   │   │   └── status.go
│   │   │
│   │   └── tui/
│   │       ├── app.go
│   │       ├── model.go
│   │       ├── messages.go
│   │       ├── controller.go
│   │       ├── styles.go
│   │       ├── components/
│   │       └── screens/
│   │
│   └── bootstrap/
│       └── app.go
│
├── test/
│   ├── integration/
│   └── e2e/
│
├── go.mod
└── README.md
```

---

# 8. 领域模型

## 8.1 Account

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

## 8.2 ProviderRef

```go
type ProviderRef struct {
    Type     ProviderType
    Host     string
    Username string
}
```

示例：

```json
{
  "type": "github",
  "host": "github.com",
  "username": "lunafoundry"
}
```

## 8.3 CommitIdentity

```go
type CommitIdentity struct {
    Name  string
    Email string
}
```

负责：

```text
git config user.name
git config user.email
```

注意：

> Commit Identity 与 Authentication Identity 必须分离。

## 8.4 TransportConfig

```go
type TransportConfig struct {
    Strategy string
    Config   map[string]string
}
```

V1：

```json
{
  "strategy": "ssh-key",
  "config": {
    "private_key": "~/.ssh/id_ed25519_lunafoundry"
  }
}
```

## 8.5 Repository

```go
type Repository struct {
    RootPath string
    Remotes  []Remote
}
```

## 8.6 Remote

```go
type Remote struct {
    Name string
    URL  string
}
```

## 8.7 RepositoryBinding

```go
type RepositoryBinding struct {
    ID         BindingID
    AccountID  AccountID
    Repository RepositoryRef
    Strategy   StrategyID
    Revision   int
}
```

## 8.8 RepositoryRef

```go
type RepositoryRef struct {
    Path string
}
```

---

# 9. Account Alias 设计

`Alias` 是 gitra 本地身份。

示例：

```text
company
```

Provider Username：

```text
luna-work
```

因此：

```text
company -> github.com/luna-work
```

CLI 使用：

```bash
gitra bind company
```

不要要求 Alias 等于 Provider Username。

---

# 10. Binding 模型

核心关系：

```text
Account
   |
   +---- Binding ---- Repository A
   |
   +---- Binding ---- Repository B
   |
   +---- Binding ---- Repository C
```

V1 约束：

```text
一个 Repository 同时只能绑定一个 Account。
```

V1 不支持：

```text
origin -> Account A
upstream -> Account B
```

如果以后有真实需求，通过新 Routing Strategy 扩展。

---

# 11. Git Binding 实现

V1 使用 Repo Local Git Config。

最终写入：

```ini
[user]
    name = Luna
    email = xxx@example.com

[core]
    sshCommand = ssh -i "/Users/luna/.ssh/id_ed25519_lunafoundry" -o IdentitiesOnly=yes

[gitra]
    bindingId = bnd_01
    accountId = acc_01H...
    strategy = repo-local
    version = 1
```

Remote 保持标准形式：

```text
git@github.com:lunafoundry/luna-site.git
```

不要求使用 SSH Host Alias。

---

# 12. 为什么使用 core.sshCommand

优点：

1. Repository 自包含。
2. 不污染 `~/.ssh/config`。
3. 不依赖 SSH Host Alias。
4. 不依赖“当前账号”。
5. Codex、Cursor、VS Code、Shell 都能自动使用。
6. 行为确定。
7. 容易恢复。
8. 容易测试。

---

# 13. Managed Git State

定义 gitra 管理的 Git 配置：

```go
type ManagedGitState struct {
    UserName   string
    UserEmail  string
    SSHCommand string
}
```

V1 gitra ownership：

```text
user.name
user.email
core.sshCommand
gitra.*
```

如果 Repository 已绑定，以上字段视为 gitra managed state。

> V1.1 起（HTTPS 路径）追加 `credential.<url>.username` 等 URL 作用域配置，见 `gitra-v1-login-onboarding-design.md`。

---

# 14. Snapshot 与恢复

绑定前必须保存原有 local config。

存放：

```text
.git/gitra/state.json
```

示例：

```json
{
  "schema_version": 1,
  "binding_id": "bnd_01",
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

执行：

```bash
gitra unbind
```

恢复绑定前状态。

不是简单：

```bash
git config --unset ...
```

---

# 15. Application Layer

## 15.1 AccountService

职责：

- CreateAccount
- UpdateAccount
- DeleteAccount
- GetAccount
- ListAccounts

接口：

```go
type AccountService interface {
    Create(ctx context.Context, req CreateAccountRequest) (Account, error)
    Update(ctx context.Context, req UpdateAccountRequest) (Account, error)
    Delete(ctx context.Context, id AccountID) error
    Get(ctx context.Context, id AccountID) (Account, error)
    List(ctx context.Context) ([]Account, error)
}
```

## 15.2 VerificationService

```go
type VerificationService interface {
    VerifyAccount(
        ctx context.Context,
        id AccountID,
    ) (VerificationResult, error)
}
```

```go
type VerificationResult struct {
    Success          bool
    ExpectedUsername string
    ActualUsername   string
    Message          string
}
```

## 15.3 BindingService

```go
type BindingService interface {
    Bind(ctx context.Context, req BindRequest) (RepositoryBinding, error)
    Unbind(ctx context.Context, req UnbindRequest) error
    GetStatus(ctx context.Context, path string) (BindingStatus, error)
    ListByAccount(ctx context.Context, id AccountID) ([]RepositoryBinding, error)
}
```

## 15.4 ReconcileService

```go
type ReconcileService interface {
    ReconcileBinding(ctx context.Context, id BindingID) error
    ReconcileAccount(ctx context.Context, id AccountID) (ReconcileResult, error)
}
```

---

# 16. Ports

## 16.1 Git Port

```go
type Git interface {
    DiscoverRepository(ctx context.Context, path string) (Repository, error)

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

    GetEffectiveConfig(
        ctx context.Context,
        repo Repository,
        key string,
    ) (ConfigValue, error)

    Remotes(
        ctx context.Context,
        repo Repository,
    ) ([]Remote, error)
}
```

## 16.2 SSH Port

```go
type SSH interface {
    Test(
        ctx context.Context,
        req SSHTestRequest,
    ) (SSHTestResult, error)
}
```

## 16.3 AccountStore

```go
type AccountStore interface {
    Save(ctx context.Context, account Account) error
    Get(ctx context.Context, id AccountID) (Account, error)
    List(ctx context.Context) ([]Account, error)
    Delete(ctx context.Context, id AccountID) error
}
```

## 16.4 BindingStore

```go
type BindingStore interface {
    Save(ctx context.Context, binding RepositoryBinding) error
    Get(ctx context.Context, id BindingID) (RepositoryBinding, error)
    FindByRepository(ctx context.Context, path string) (*RepositoryBinding, error)
    ListByAccount(ctx context.Context, id AccountID) ([]RepositoryBinding, error)
    Delete(ctx context.Context, id BindingID) error
}
```

---

# 17. Adapter 设计

## 17.1 GitCLIAdapter

```text
Git Port
   |
   v
GitCLIAdapter
   |
   v
Native git executable
```

命令示例：

```bash
git -C <path> rev-parse --show-toplevel
git -C <path> config --local --get user.name
git -C <path> config --local user.name "Luna"
git -C <path> remote get-url origin
```

原则：

> 不自行解析 `.git/config` 作为主实现。

原因：gitra 应看到和用户 Git 完全一致的行为。

## 17.2 SSHCLIAdapter

调用系统 `ssh`。

账号测试：

```bash
ssh   -T   -i ~/.ssh/id_ed25519_lunafoundry   -o IdentitiesOnly=yes   git@github.com
```

---

# 18. Strategy 设计

## 18.1 AuthStrategy

```go
type AuthStrategy interface {
    ID() string

    Validate(
        ctx context.Context,
        account Account,
    ) error

    BuildGitConfig(
        account Account,
    ) ([]GitConfigEntry, error)

    Verify(
        ctx context.Context,
        account Account,
        provider ProviderAdapter,
    ) (VerificationResult, error)
}
```

## 18.2 SSHKeyAuthStrategy

V1 唯一 Auth Strategy。

输入：

```text
private_key path
```

输出：

```text
core.sshCommand
```

示例：

```text
ssh -i "/Users/luna/.ssh/id_ed25519_lunafoundry" -o IdentitiesOnly=yes
```

## 18.3 AuthStrategyRegistry

```go
type AuthStrategyRegistry struct {
    strategies map[string]AuthStrategy
}
```

```go
func (r *AuthStrategyRegistry) Register(s AuthStrategy)
func (r *AuthStrategyRegistry) Get(id string) (AuthStrategy, error)
```

禁止将 provider/auth 判断散落为大量 `if/else`。

---

# 19. Routing Strategy

```go
type RoutingStrategy interface {
    ID() string

    Bind(
        ctx context.Context,
        repo Repository,
        account Account,
    ) error

    Unbind(
        ctx context.Context,
        repo Repository,
    ) error

    Inspect(
        ctx context.Context,
        repo Repository,
    ) (RoutingStatus, error)
}
```

V1：

```text
RepoLocalRoutingStrategy
```

未来可实现：

```text
DirectoryRoutingStrategy
GitIncludeRoutingStrategy
RemoteOwnerRoutingStrategy
```

---

# 20. Provider Adapter

```go
type ProviderAdapter interface {
    ID() string

    Capabilities() ProviderCapabilities

    MatchRemote(remote Remote) bool

    ParseRemote(
        remote Remote,
    ) (RemoteRepositoryRef, error)

    VerifyIdentity(
        ctx context.Context,
        account Account,
        auth AuthStrategy,
    ) (VerificationResult, error)
}
```

## 20.1 ProviderCapabilities

```go
type ProviderCapabilities struct {
    SSH           bool
    HTTPS         bool
    OAuth         bool
    DeviceFlow    bool
    RepositoryAPI bool
    SSHKeyAPI     bool
}
```

V1 UI 实际只使用 SSH。

OAuth 字段只是未来扩展能力，不在 V1 实现。

---

# 21. Provider Registry

```go
type ProviderRegistry struct {
    providers map[string]ProviderAdapter
}
```

Bootstrap：

```go
providers.Register(github.New(...))
providers.Register(gitlab.New(...))
providers.Register(gitea.New(...))
```

业务层禁止：

```go
if provider == "github" { ... }
```

---

# 22. Reconciler

Reconciler 是整个系统的重要组件。

```go
type Reconciler struct {
    accounts AccountStore
    bindings BindingStore
    git      Git

    auth    *AuthStrategyRegistry
    routing *RoutingStrategyRegistry
}
```

流程：

```text
Account + Binding
       |
       v
Build Desired Git State
       |
       v
Read Actual Git State
       |
       v
Compare
       |
       +--- same ---> OK
       |
       +--- diff ---> DRIFT
                        |
                        v
                      Apply
```

---

# 23. Binding Health

```go
type BindingHealth string

const (
    HealthOK      BindingHealth = "ok"
    HealthDrift   BindingHealth = "drift"
    HealthMissing BindingHealth = "missing"
    HealthBroken  BindingHealth = "broken"
)
```

- `OK`：Desired == Applied
- `DRIFT`：用户手工修改了 gitra 管理配置
- `MISSING`：记录存在但 Repository 路径不存在
- `BROKEN`：配置无法生成、SSH Key 不存在、Git config 无效等

---

# 24. Transaction / Rollback

Bind 必须具有原子语义。

绑定流程：

```text
1. Discover Repository
2. Read Account
3. Validate Account
4. Snapshot Current Config
5. Apply Git Config
6. Save gitra metadata
7. Save Binding
8. Verify Applied State
9. Commit
```

失败：

```text
restore snapshot
remove partial metadata
remove partial binding
```

伪代码：

```go
snapshot, err := gitSnapshot.Create(...)
if err != nil {
    return err
}

if err := apply(); err != nil {
    _ = snapshot.Restore(...)
    return err
}

if err := verify(); err != nil {
    _ = snapshot.Restore(...)
    return err
}
```

---

# 25. Account Update 流程

例如修改 SSH key：

```text
Update Account
   |
   v
Revision + 1
   |
   v
BindingStore.ListByAccount()
   |
   v
Reconcile each binding
   |
   v
Update repo local configs
```

因此 Account 是 Desired State。

---

# 26. Account 删除

如果存在绑定，禁止直接删除。

Application 返回：

```go
ErrAccountInUse
```

TUI 提示：

```text
Remove "lunafoundry"?

6 projects are currently bound.

[ Cancel ]
[ Unbind all and remove ]
```

---

# 27. Storage

V1 不需要数据库。

推荐：

```text
~/.config/gitra/
├── accounts/
│   ├── lunafoundry.json
│   ├── work.json
│   └── personal.json
│
└── bindings.json
```

## 27.1 Account JSON

```json
{
  "schema_version": 1,
  "id": "lunafoundry",
  "alias": "lunafoundry",
  "provider": {
    "type": "github",
    "host": "github.com",
    "username": "lunafoundry"
  },
  "identity": {
    "name": "Luna",
    "email": "xxx@example.com"
  },
  "transport": {
    "strategy": "ssh-key",
    "config": {
      "private_key": "~/.ssh/id_ed25519_lunafoundry"
    }
  },
  "revision": 3
}
```

## 27.2 Bindings JSON

```json
{
  "schema_version": 1,
  "bindings": [
    {
      "id": "bnd_01",
      "repository": {
        "path": "/Users/luna/Projects/luna-site"
      },
      "account_id": "lunafoundry",
      "strategy": "repo-local",
      "revision": 1
    }
  ]
}
```

---

# 28. Repository Metadata

Repository 内：

```ini
[gitra]
    bindingId = bnd_01
    accountId = acc_01H...
    strategy = repo-local
    version = 1
```

Repository 移动后仍可依赖 `bindingId` 识别，并由 `gitra status` 修正 BindingStore 路径。

---

# 29. 并发控制

防止两个 `gitra` 进程同时写配置。

推荐：

```text
~/.config/gitra/.lock
```

写操作：

- CreateAccount
- UpdateAccount
- DeleteAccount
- Bind
- Unbind
- Reconcile

都必须获取写锁。

接口：

```go
type Locker interface {
    WithWriteLock(
        ctx context.Context,
        fn func() error,
    ) error
}
```

---

# 30. 文件写入原子性

禁止直接覆盖 `bindings.json`。

应：

```text
write temp
fsync
rename
```

例如：

```text
bindings.json.tmp
      |
      v
fsync
      |
      v
rename
      |
      v
bindings.json
```

---

# 31. 错误模型

```go
var (
    ErrAccountNotFound  = errors.New("account not found")
    ErrAccountExists    = errors.New("account already exists")
    ErrAccountInUse     = errors.New("account has active bindings")

    ErrNotGitRepository = errors.New("not a git repository")
    ErrAlreadyBound     = errors.New("repository already bound")
    ErrBindingNotFound  = errors.New("binding not found")

    ErrAuthInvalid      = errors.New("authentication invalid")
    ErrProviderMismatch = errors.New("provider mismatch")
    ErrIdentityMismatch = errors.New("identity mismatch")

    ErrStateDrift       = errors.New("managed state drift")
)
```

---

# 32. CLI Exit Code

建议：

```text
0   success
1   internal / general error
2   invalid CLI input
3   account error
4   repository error
5   authentication error
6   binding / state conflict（含 drift）
7   unsupported operation
```

---

# 33. CLI 接口设计

## 33.1 Account

```bash
gitra account add
```

非交互模式：

```bash
gitra account add   --alias lunafoundry   --provider github   --host github.com   --username lunafoundry   --name Luna   --email xxx@example.com   --auth ssh-key   --key ~/.ssh/id_ed25519_lunafoundry
```

其他：

```bash
gitra account list
gitra account show lunafoundry
gitra account edit lunafoundry
gitra account remove lunafoundry
gitra account test lunafoundry
```

---

# 34. Binding CLI

```bash
gitra bind lunafoundry
gitra bind lunafoundry ~/Projects/luna-site

gitra unbind
gitra unbind ~/Projects/luna-site

gitra status
gitra status ~/Projects/luna-site
```

---

# 35. JSON 输出

查询命令 V1 必须支持：

```text
--json
```

示例：

```bash
gitra status --json
```

输出：

```json
{
  "repository": "/Users/luna/Projects/luna-site",
  "binding": {
    "id": "bnd_01",
    "account": "lunafoundry",
    "state": "ok"
  },
  "provider": {
    "type": "github",
    "host": "github.com",
    "username": "lunafoundry"
  }
}
```

---

# 36. TUI 架构

TUI 是 Presentation Adapter。

```text
TUI
 |
 v
Controller
 |
 v
Application Service
```

禁止 TUI：

- 直接执行 Git
- 读取 storage JSON
- 调用 SSH
- 直接修改 `.git/config`

---

# 37. TUI Model

```go
type Screen int

const (
    ScreenAccounts Screen = iota
    ScreenAccountDetail
    ScreenAddAccount
    ScreenEditAccount
    ScreenAddProject
    ScreenConfirm
)
```

```go
type Model struct {
    Screen Screen

    Width  int
    Height int

    Accounts []AccountCardViewModel
    Selected int

    Detail *AccountDetailViewModel

    Dialog DialogState
}
```

---

# 38. TUI ViewModel

```go
type AccountCardViewModel struct {
    ID           string
    Provider     string
    Alias        string
    Host         string
    AuthLabel    string
    Health       string
    ProjectCount int
}
```

```go
type AccountDetailViewModel struct {
    ID       string
    Provider string
    Alias    string
    Host     string
    Username string

    Identity string
    Auth     string
    Health   string

    Projects []ProjectViewModel
}
```

---

# 39. TUI Controller

```go
type AccountController struct {
    Accounts     AccountService
    Bindings     BindingService
    Verification VerificationService
}
```

Controller 负责：

```text
Domain Model -> ViewModel
```

---

# 40. Bubble Tea Message 设计

```go
type AccountsLoadedMsg struct {
    Accounts []AccountCardViewModel
}

type AccountOpenedMsg struct {
    Account AccountDetailViewModel
}

type AccountSavedMsg struct {
    Account AccountCardViewModel
}

type BindingCreatedMsg struct {
    Project ProjectViewModel
}

type OperationFailedMsg struct {
    Err error
}
```

---

# 41. TUI 事件流

```text
User presses Enter
      |
      v
OpenAccount
      |
      v
Controller.LoadAccount()
      |
      v
Application Services
      |
      v
AccountOpenedMsg
      |
      v
Update Model
      |
      v
Render Detail Screen
```

---

# 42. Responsive Card Layout

建议：

```text
card min width: 32
card ideal width: 38-42
gap: 2
```

公式：

```text
columns = max(1, usableWidth / (cardWidth + gap))
```

---

# 43. Bootstrap / Dependency Injection

不用 DI Framework。

```go
git := gitcli.New(...)
ssh := sshcli.New(...)

accountStore := storage.NewAccountJSONStore(...)
bindingStore := storage.NewBindingJSONStore(...)

providers := provider.NewRegistry()
providers.Register(github.New(...))
providers.Register(gitlab.New(...))
providers.Register(gitea.New(...))

auth := authstrategy.NewRegistry()
auth.Register(sshkey.New(ssh))

routing := routingstrategy.NewRegistry()
routing.Register(repolocal.New(git))

accountService := app.NewAccountService(...)
bindingService := app.NewBindingService(...)
verifyService := app.NewVerificationService(...)
reconciler := app.NewReconciler(...)
```

这里就是 Composition Root。

---

# 44. 依赖方向

必须保持：

```text
Presentation
     |
Application
     |
Domain / Ports
     ^
Adapters
```

禁止：

```text
Domain -> Adapter
Application -> Bubble Tea
Application -> Git CLI concrete implementation
```

---

# 45. 安全设计

## 45.1 SSH 私钥

不复制。

Account 只存：

```text
private key path
```

禁止把私钥内容写入 gitra 自己的配置目录。

## 45.2 Token

V1.0 不实现；V1.1 立项（登录 token），见 `gitra-v1-login-onboarding-design.md`。

Token 必须交给系统 SecretStore：

```text
macOS Keychain
Windows Credential Manager
Linux Secret Service
```

## 45.3 ~/.ssh/config

V1 不修改。

## 45.4 ~/.gitconfig

V1 不修改。

作用范围只限：

```text
~/.config/gitra
+
repo/.git/config
```

---

# 46. Test Connection

```text
Account
  |
  v
Auth Strategy
  |
  v
Provider Adapter
  |
  v
SSH Test
```

验证 expected username 与 remote authenticated username 是否一致。

---

# 47. Provider 与 Auth 分离

必须区分：

```text
Commit Identity
Git Transport Authentication
Provider API Authentication
```

V1 只有前两者。

UI 推荐：

```text
Git Auth: Ready
```

而不是笼统：

```text
Logged in
```

---

# 48. Provider API Authentication

V1.0 不实现；V1.1 立项（浏览器登录、账号资料自动填充），见增补设计。

后续出现明确需求：

- 自动读取用户名
- 获取邮箱
- 获取远端仓库
- 自动上传 SSH Key

再增加：

```text
ProviderAPIAuthStrategy
```

---

# 49. 日志

支持：

```bash
gitra --verbose ...
```

建议级别：

```text
DEBUG
INFO
WARN
ERROR
```

默认只展示 WARN+。

严禁日志记录：

- private key 内容
- token
- secret
- credential helper 输出

---

# 50. 测试体系

## 50.1 Domain Unit Test

测试：

- Account validation
- Provider validation
- Alias validation
- Binding invariant
- Health transitions
- Strategy registry

## 50.2 Application Unit Test

使用：

```text
FakeAccountStore
FakeBindingStore
FakeGit
FakeSSH
FakeAuthStrategy
FakeRoutingStrategy
```

测试：

- CreateAccount
- UpdateAccount
- DeleteAccount
- Bind
- Unbind
- Rollback
- Reconcile
- Account in use

## 50.3 Adapter Integration Test

真实临时目录：

```bash
git init /tmp/test-repo
```

测试：

- repo detection
- git config
- remote parsing
- core.sshCommand
- restore

## 50.4 关键 E2E

```text
Account A
Account B

Repo A -> Account A
Repo B -> Account B
```

验证：

```bash
git -C repo-a config user.email
git -C repo-b config user.email

git -C repo-a config core.sshCommand
git -C repo-b config core.sshCommand
```

必须严格隔离。

---

# 51. TUI Test

测试：

- 不同终端宽度列数
- keyboard navigation
- card selection
- modal state
- account list rendering
- error rendering

业务测试不放 TUI 层。

---

# 52. Version / Schema Migration

所有持久化 JSON 必须包含：

```json
{
  "schema_version": 1
}
```

未来：

```go
type Migration interface {
    From() int
    To() int
    Migrate([]byte) ([]byte, error)
}
```

---

# 53. Future Extension Matrix

| 需求 | 扩展位置 | 核心修改 |
|---|---|---|
| GitHub Enterprise | Provider Adapter | 否 |
| GitLab | Provider Adapter | 否 |
| Gitea | Provider Adapter | 否 |
| Forgejo | Provider Adapter | 否 |
| HTTPS Token / 浏览器登录（V1.1 立项） | Auth Strategy + Provider API Auth Strategy | 否 |
| SSH Agent | Auth Strategy | 否 |
| macOS Keychain | SecretStore Adapter | 否 |
| Directory Routing | Routing Strategy | 否 |
| Remote Owner Routing | Routing Strategy | 否 |
| Web UI | Presentation Adapter | 否 |
| Desktop GUI | Presentation Adapter | 否 |
| VS Code Extension | Presentation/API Adapter | 否 |
| OAuth / 设备码 / Loopback（V1.1 立项） | Provider API Auth Strategy | 小 |
| 引导式克隆（V1.2 立项） | Application Use Case + Presentation | 小 |
| 新手引导 Onboarding（V1.2 立项） | Presentation Adapter | 小 |

---

# 54. 不做动态 Plugin System

V1 不做：

```text
.so
WASM
RPC plugin
plugin marketplace
```

采用：

```go
registry.Register(...)
```

足够。

---

# 55. Clone 功能

V1.0 不做；V1.2 立项为「引导式克隆」，见增补设计。

命令形态（修订 1）：

```bash
gitra clone <url> [dir]
```

账号由 URL + 已登录账号自动解析（不再要求先给 alias）。

内部：

```text
temporary account auth
       |
       v
git clone
       |
       v
bind repository
```

---

# 56. V1 实现优先级

## Phase 1 - Core

完成：

- Domain Model
- Ports
- AccountStore
- BindingStore
- GitCLIAdapter
- SSHCLIAdapter
- SSHKeyAuthStrategy
- RepoLocalRoutingStrategy
- AccountService
- BindingService
- Reconciler

验收：

```text
Repo A -> Account A
Repo B -> Account B
git identity isolated
```

## Phase 2 - CLI

完成：

```text
account add/list/show/edit/remove/test
bind
unbind
status
--json
```

## Phase 3 - TUI

完成：

```text
Accounts card page
Account detail
Add/Edit Account
Add/Remove Project
Test Connection
Confirm dialog
Responsive layout
```

## Phase 4 - Stability

完成：

- rollback
- locks
- atomic persistence
- drift detection
- path move recovery
- schema migration
- E2E tests

---

# 57. Definition of Done

## Account

- 可创建多个 GitHub/GitLab/Gitea 账号
- 每个账号独立 SSH Key
- 可 Test Connection

## Binding

- Repository 绑定账号
- Repository unbind 恢复之前 Git config
- Account 修改后可 Reconcile

## Git

以下全部不需要 gitra 介入：

```bash
git fetch
git pull
git push
```

并自动使用正确账号。

## Multi Repository

```text
Repo A -> Account A
Repo B -> Account B
Repo C -> Account A
```

同时存在无冲突。

## CLI

可被脚本/Agent 使用。

## TUI

可完成所有核心管理操作。

## Runtime

不需要：

```text
daemon
browser
node
database server
background service
```

---

# 58. 最终架构原则

## 58.1 高内聚

```text
AccountService -> account lifecycle
BindingService -> repository binding lifecycle
Reconciler     -> desired/applied synchronization
GitAdapter     -> Git interaction
SSHAdapter     -> SSH interaction
Provider       -> provider-specific behavior
Strategy       -> replaceable behavior
```

## 58.2 低耦合

业务代码只依赖 interfaces / ports。

## 58.3 易扩展

新功能优先通过：

```text
Adapter
Strategy
Registry
Presentation Adapter
```

扩展。

## 58.4 高复用

同一个 Application Core 可以被：

```text
CLI
TUI
未来 Web
未来 Desktop
未来 IDE plugin
未来 automation
```

共同调用。

---

# 59. 一句话架构总结

```text
gitra = Account Desired State
      + Repository Binding
      + Strategy-based Git Identity Projection
      + Native Git Runtime
```

产品体验：

```text
添加账号
   |
绑定项目
   |
以后永远直接 git push
```

这就是 gitra V1 应该解决的全部问题。
