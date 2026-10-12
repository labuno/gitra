# Todo: RC 标签自动预发布

- **Status**: 验证成功
- **Scope**: `v*-rcN` 标签触发现有构建并创建 GitHub 预发布，正式版本沿用原行为。
- **Baseline**: main / 2b6fc79

## Behavior

- Input: `v0.1.4-rc1` 或正式标签，现有发布工作流。
- Output: RC 标记 prerelease 且不设为 Latest；正式版本使用原发布选项。已有 Release 不重复创建。
- Constraints: 复用原构建、产物和校验机制，不改写旧标签。

## Test first

- 用 mock gh 执行工作流的实际创建步骤，断言 RC 参数、正式参数和已存在 Release 的跳过行为。
- 预期 RED: RC 创建命令缺少 `--prerelease` 和 `--latest=false`。

## Implementation

- 发布工作流按标签设置创建参数；CI/Release 运行该测试；中英 README 说明 RC 流程。

## Verification

- Focused: `python3 scripts/test-release-workflow.py`。
- Regression: `go test ./...`、`go vet ./...`、`git diff --check`。
- Delivery: 修复提交推送后验证远端 SHA，再创建并推送 `v0.1.4-rc1`，确认预发布和产物。

## Done

- [x] RED observed before implementation.
- [x] Tests and documentation checks pass.
- [x] Release workflow ready for RC tags.
- [x] Status is 验证成功.

## Evidence

- RED: mock gh 收到 RC 的创建命令，但缺少 `--prerelease`。
- GREEN: `python3 scripts/test-release-workflow.py` 三个执行测试通过，覆盖候选预发布、不设置 Latest、正式版参数和已有 Release 跳过创建。
- Regression: `go test ./...`、`go vet ./...` 通过；工作流测试已加入 CI 与 Release 验证步骤。
- Documentation: 两种语言同步说明 rc1/rc2/正式版流程以及不移动旧标签的约定。
- Source/test mapping: 两份工作流修改由实际发布创建步骤执行测试覆盖；未更改产品源码。
- Delivery handoff: 本轮使用新标签 `v0.1.4-rc1`，从已推送的提交发布，保留原标签。
