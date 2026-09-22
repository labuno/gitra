# gitra V1.0 Provider 验证实施计划

> Plan ID：`gitra-v1-verify`
> 范围：基线 Milestone 3（显式账号连接验证 `account test`，HTTPS 与 SSH 两条路径）
> 上游约束：`docs/gitra-v1-implementation-baseline.md`（修订 1）、`docs/gitra-v1-login-onboarding-design.md`、`docs/gitra-architecture-development-design.md`

---

## Objective

提供显式、可脚本化的账号验证：HTTPS 账号通过 Provider API 校验 token 与身份，SSH 账号通过 `ssh -T` 原始输出由 Provider 解析认证用户名，二者都用「期望用户名 vs 实际用户名」判定。

## Scope

- `ports.SSH` 的真实实现（sshcli）：`ssh -T -i <key> -o IdentitiesOnly=yes [-p port] <user>@<host>`，返回原始 exit code/stdout/stderr。
- Provider SSH 响应解析（基线 §4.6）：GitHub / GitLab / Gitea 各自的「认证成功 + 用户名」判定，绝不只看 exit code。
- `app.VerificationService`：按账号策略分流 HTTPS（API profile）与 SSH（ssh -T）路径，返回 `VerificationResult{Success, ExpectedUsername, ActualUsername, Message}`。
- CLI `gitra account test <alias> [--json]`，退出码遵循基线 §35（认证失败/身份不匹配 → 5）。
- TUI 的 `T` 动作复用同一服务（本 Plan 只保证 Application 层接口稳定）。

## Non-goals

- 后台定时验证、TUI 首页自动联网（基线 §4.8 仍然禁止）。
- OAuth/device flow、SSH key 上传、Provider 其它 API 能力。

## Steps

1. sshcli adapter 与 Provider SSH 解析。
2. VerificationService（双路径 + 错误分类）。
3. CLI `account test` 与离线回归（fixture + httptest）。

## Interfaces

- `ports.SSH.Test(ctx, SSHTestRequest{Host, User, Port, PrivateKeyPath}) (SSHTestResult, error)`（基线 §16.2/§4.6）。
- Provider 解析入口：`ParseSSHVerification(stdout, stderr string) (username string, ok bool)`。
- CLI 输出：`{"schema_version":1,"account_id":"acc_...","success":true,"expected_username":"...","actual_username":"...","message":"authenticated"}`。

## Work Packages

### VERIFY-01 SSH Adapter & Provider SSH Parsing

sshcli 实现 + GitHub/GitLab/Gitea 的 `ssh -T` 响应解析与 fixture 测试。

### VERIFY-02 VerificationService

按策略分流的验证用例：HTTPS 走 Profile API，SSH 走 ssh -T + 解析，输出统一结果与错误分类。

### VERIFY-03 CLI account test & Regression

`gitra account test <alias> [--json]` 命令与离线回归（httptest + fixture），退出码矩阵。

## Test Matrix

- sshcli 单测（argv 构造、原始结果透传、超时/缺失二进制）。
- Provider 解析 fixture（成功/失败/未知响应）。
- VerificationService 单测（成功、用户名不匹配、token 失效、解析失败）。
- CLI 集成测试（exit 0/3/5 与 JSON 形状）。

## Verification

- 每节点 `go test ./...` + `go vet ./...`；全流程离线可跑。

## Done

- 全部 Work Package 完成、节点验证成功、证据入图。
- `gitra account test` 能区分：本地配置无效、认证成功、用户名不匹配、认证失败、网络/进程失败、无法识别的响应。

## STOP Conditions

- 需要真实网络或真实账号才能通过自动化测试。
- 需要把 Provider 特有判断下沉到业务层（违反基线 §21）。
- 源文档 digest 漂移或语义复核失败。

## External Sources

- OpenSSH 客户端（开发机 9.9p2）；GitHub/GitLab/Gitea 的 `ssh -T` 实际响应文本。
- Provider REST API（HTTPS 路径复用 V1.1 已实现 client）。
