# Todo: 仅发布 macOS 应用包

- **Status**: 验证成功
- **Scope**: 用户选择后续 Release 只生成 macOS 应用包和校验文件。
- **Baseline**: d72f0af

## Behavior

- Output: `Gitra-macOS.zip` 和 `checksums.txt` 两个上传产物；GitHub 自动源码包不受控制。
- Constraints: 不删除历史发布的 CLI 文件；CLI 仍可从源码构建；macOS 保留通用架构与签名验证。

## Test first

- 验证工作流上传文件仅为两个指定产物，校验任务仅依赖 macOS 构建。
- 预期 RED: 仍包含多平台 CLI 上传及 build 依赖。

## Implementation

- 移除 release 的 CLI build 矩阵，同步 README 下载说明；保持历史 rc1 文档准确。

## Verification

- `python3 scripts/test-release-workflow.py`；文档一致性；`git diff --check`。

## Done

- [x] RED observed before implementation.
- [x] Tests and document consistency pass.
- [x] macOS-only release workflow ready.
- [x] Status is 验证成功.

## Evidence

- RED: 上传目标包含六平台 CLI ZIP 模板，超过约定的两个上传文件。
- GREEN: 工作流仅上传 macOS 应用包和校验文件；校验任务仅依赖 macOS 构建。
- Documentation: 中英 README 同步新产物策略，rc1 更新说明保留当次历史下载描述。
- Scope: 未删除现有 Release 文件，未修改标签、应用构建脚本或产品代码。
