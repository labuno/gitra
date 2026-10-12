# Todo: 每次发布使用中英更新说明

- **Status**: 验证成功
- **Scope**: Release 使用对应标签的中英发布文档，描述新增功能、修复、限制和下载方式。
- **Baseline**: eec0bb5 / v0.1.4-rc1

## Behavior

- Input: 标签和 `docs/releases/<tag>.md`。
- Output: GitHub Release 使用该版本说明，英文在前、中文在后。
- Constraints/errors: 缺少说明时明确失败，不再使用固定下载模板掩盖缺失更新记录；已有 Release 不重复创建。

## Test first

- mock gh 执行真实发布步骤，断言使用对应标签文档；没有文档时不创建 Release。
- 保留 RC/正式版及重复发布测试。
- 预期 RED: 创建命令仍引用固定 release-notes.md。

## Implementation

- prepare 检出仓库，读取标签文档；新增 rc1 更新说明，README 说明维护约定。
- 本次线上 rc1 说明直接更新，不移动标签或重构建。

## Verification

- `python3 scripts/test-release-workflow.py`；中英内容、相对链接、`git diff --check`。
- 提交推送，核对 GitHub Release 正文与文档一致。

## Done

- [x] RED observed before implementation.
- [x] Workflow tests and document consistency pass.
- [x] Versioned release notes ready.
- [x] Status is 验证成功.

## Evidence

- RED: RC 和正式版测试观察到旧命令只使用固定 `release-notes.md`，不使用版本文档。
- GREEN: 四个工作流执行测试通过，覆盖版本正文、缺失文档报错、RC 参数和已有发布跳过创建。
- Documentation: 中英文 README 均说明版本文档约定，rc1 文档覆盖相对于 v0.1.2 的全部四个提交。
- Source/test mapping: 工作流 prepare 步骤由 mock gh 的实际 shell 执行测试覆盖；不改动构建和应用行为。
- Delivery: 本次在线正文可通过 `gh release edit --notes-file docs/releases/v0.1.4-rc1.md` 更新，已有标签和产物无需变更。
- Asset choice: 用户尚未选择下载产物范围，继续保留现有平台文件。
