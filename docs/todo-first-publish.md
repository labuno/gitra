# Todo: 首次上传（first publish）

- **Status**: 待执行
- **Scope**: 一个行为边界——把「已绑定、远端为空、本地有提交」的仓库做一次首次推送并建立跟踪
- **Baseline**: `main`（提交 84b8d55 之后）

## Why

新手流程走到"创建仓库 + 绑定"后仍卡住：`git pull` 在空仓库上必然报
`There is no tracking information for the current branch`，而正确动作是首次 `push`。
baseline §7 禁止 gitra 代理 git 命令，本 Todo 记录一个**窄例外**：仅用于"远端空 + 本地有提交"的一次性发布，不提供通用 push/pull/fetch 代理。

## Behavior

- **Input**: 仓库路径（TUI 的 `U`，或 CLI `gitra publish [path]`）。
- **Output**:
  - 状态：当前分支、本地是否有提交、远端是否已有该分支、能否上传及原因；
  - 动作：`git push --set-upstream <remote> <branch>`，随后用 `ls-remote` 复核远端已出现该分支。
- **Constraints/errors**:
  - 未绑定 → `ErrBindingNotFound`；
  - 无 origin → `ErrUnsupportedRemote`；
  - 无本地提交 → 明确提示"还没有提交"；
  - 远端已有该分支 → 明确提示"远端已有分支，直接同步即可"（不做覆盖式推送）；
  - 推送失败（无权限/网络/凭据缺失）→ 原样分类为可读错误，并提示重新登录或检查网络。

## Test first

- Normal: 本地仓库 + 空 bare 远端 → publish 成功，远端出现分支，且本地设置了 upstream。
- Edge: 远端已有分支 → 拒绝并给出提示；无提交 → 拒绝并提示。
- Error: 未绑定 → `ErrBindingNotFound`；无 origin → `ErrUnsupportedRemote`。
- Expected RED: `ports.Publisher`/`PublishService` 不存在时编译失败。

## Implementation

- `ports.Publisher`（状态 + 推送）与 `gitcli` 实现（复用 CommandRunner，60s 上限）。
- `app.PublishService`（状态判定、首次上传编排、成功后复核）。
- TUI：详情页 `U 首次上传`；创建远端并绑定成功后主动询问是否立即上传。
- CLI：`gitra publish [path]`（脚本/Agent 用，输出同样走退出码契约）。

非目标：通用 push/pull/fetch、强制推送、分支管理、冲突处理。

## Verification

- Focused: `go test ./internal/app/... ./internal/adapters/gitcli/... ./internal/presentation/tui/...`
- Regression: `go test ./...`、`go vet ./...`、gofmt。

## Evidence

- RED:
- GREEN:
- Changed files:
- Remaining limitation:

## Done

- [ ] Test was written first and RED observed, or exception recorded.
- [ ] Implementation stays in scope.
- [ ] Focused and relevant regression checks pass.
- [ ] Status is 验证成功.
