# gitra

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
