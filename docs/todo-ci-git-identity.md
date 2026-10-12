# Todo: CI 测试提交身份独立于本机配置

- **Status**: 验证成功
- **Scope**: 修复保留既有仓库历史的测试，显式使用测试身份。
- **Baseline**: e191a51 / v0.1.3

## Behavior

- Input: 不存在全局 Git 配置的测试环境。
- Output: 测试仍可创建既有提交并证明初始化不改变历史和远端。
- Constraints: 不修改用户或 CI 的全局 Git 配置，不修改产品逻辑。

## Test first

- 先禁用该测试的系统及全局 Git 配置，观察 Author identity unknown 的 RED。
- 对创建测试提交的命令提供局部身份参数，再验证 GREEN。

## Implementation

- 修改 `internal/presentation/tui/folder_setup_test.go`；保留原有行为断言。

## Verification

- Focused: 无全局 Git 配置下运行失败的测试。
- Regression: 无全局 Git 配置下 `go test ./...`；`go vet ./...`；`git diff --check`。

## Done

- [x] RED observed before fix.
- [x] Focused and regression checks pass.
- [x] Fix and delivery evidence prepared.
- [x] Status is 验证成功.

## Evidence

- RED: 禁用系统和全局配置，并启用 `user.useConfigOnly=true` 后，原测试提交失败（Author identity unknown）。该选项同时排除了 macOS 根据系统用户名自动猜测身份的行为。
- GREEN: 测试提交显式使用 `git -c user.name=Test User -c user.email=test@example.com commit ...`，原有 HEAD/origin 保留断言通过。
- Regression: `GIT_CONFIG_GLOBAL=/dev/null GIT_CONFIG_NOSYSTEM=1 go test ./... -count=1`、`go vet ./...`、`git diff --check` 均通过。
- Release handoff: v0.1.3 已存在并指向修复前提交；重新运行旧标签流程不会包含修复，需发布新版本或在明确确认后移动旧标签。
