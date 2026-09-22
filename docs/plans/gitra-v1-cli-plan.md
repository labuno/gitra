# gitra V1.0 CLI 实施计划

> Plan ID：`gitra-v1-cli`
> 范围：基线 Milestone 2（CLI），不含 `account test`（Milestone 3）
> 上游约束：`docs/gitra-v1-implementation-baseline.md`（修订 1）、`docs/gitra-architecture-development-design.md`、`docs/gitra-v1-login-onboarding-design.md`

---

## Objective

让 gitra 在没有 TUI 的情况下完整可用：CLI 覆盖账号生命周期与仓库绑定，支持 `--json` / `--verbose`，退出码遵循基线 §35，并按 §47 的验收流程走通本地全流程。

## Scope

- 新增 Application 层 `AccountService`（Create/Update/Delete/Get/GetByAlias/List）与账号变更后的 Reconcile。
- bootstrap 组合根：把 storage / gitcli / runner / strategies 装配为可复用对象。
- Cobra 命令树；CLI 只调用 Application 层，不直接碰 storage / git / ssh。
- `--json` 稳定输出（形状见增补设计附录 A.1）；`--verbose` 只影响日志级别。
- 退出码映射：0 success / 1 internal / 2 参数与校验 / 3 account / 4 repository / 5 认证 / 6 冲突 / 7 不支持。

## Non-goals

- `account test` 与 Provider 联网验证（Milestone 3）。
- TUI（Milestone 4）、登录与 HTTPS 凭据（V1.1 增补）、引导式克隆。
- 交互式提问：M2 全部非交互，缺参即报错（exit 2）。
- 需要网络或真实账号的自动化测试。

## Steps

1. bootstrap + AccountService。
2. CLI foundation（root、错误映射、--verbose、--json）。
3. account add/list/show/edit/remove。
4. bind/unbind/status。
5. 脚本化 E2E 回归。

## Interfaces

- AccountService 签名按基线 §25.1；CLI 只依赖 app 层。
- 命令形态：`gitra account add --alias A --provider github [--host H] [--ssh-port P] [--ssh-user U] --username X --name N --email E --auth ssh-key --key PATH`；`gitra account edit <alias> [同名可选 flags]`；`gitra bind <alias> [path]`；`gitra unbind [path]`；`gitra status [path]`。
- Provider 默认值来自 `domain.DefaultEndpoint`（host/ssh user/port 可省略）。
- JSON 契约：`schema_version: 1`，字段只增不改。

## Work Packages

### CLI-01 Bootstrap & AccountService

bootstrap 组合根 + AccountService（含 ErrAccountInUse 守卫与 update 后 reconcile 传播）。

### CLI-02 CLI Foundation

Cobra root、全局 flags、错误到退出码映射、JSON 写出器、文本渲染基座。

### CLI-03 Account Commands

account add / list / show / edit / remove 五个子命令及各自的 JSON 输出。

### CLI-04 Bind / Unbind / Status Commands

bind / unbind / status 三个命令，status 输出 A.1 冻结的 JSON 形状。

### CLI-05 CLI End-to-End Regression

在隔离配置目录上的脚本化全流程回归（§47 验收流程）与退出码矩阵。

## Test Matrix

- app 单测（AccountService：create/update/delete 守卫/reconcile 传播）。
- CLI 集成测试（临时 GITRA_CONFIG_DIR；退出码与 JSON）。
- 脚本化 E2E（§47 验收流程，去掉需要真实远端的 fetch/push）。

## Verification

- 每节点 go test ./... + go vet ./...。
- CLI 输出与文档示例一致，JSON 通过固定断言。

## Done

- 全部 Work Package 完成、节点验证成功、证据入图。
- §47 验收流程在本地可完整走通（fetch/push 仅需真实账号时可跳过）。

## STOP Conditions

- 需要修改 Application 层公共语义才能实现 CLI。
- 需要网络或真实账号才能通过自动化测试。
- 源文档 digest 漂移或语义复核失败。

## External Sources

- Cobra（基线 §19.1）：github.com/spf13/cobra。
- Go 1.25 与系统 git（沿用 V1.0 core 已验证环境）。
