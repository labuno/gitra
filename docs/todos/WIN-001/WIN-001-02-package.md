# WIN-001-02: Windows 便携包与发布

- **Todo ID**: WIN-001-02
- **Execution Map**: execution-map.json
- **Todo maturity**: DECOMPOSED
- **Task detail format**: v1
- **Source baseline**: docs/plans/windows-portable-plan.md / main 6c10584

## Outcome

Windows 用户下载一个 ZIP 后可双击启动；后续 release 同时生成 macOS 与 Windows 包。

## Contract

- Consumes: artifact:windows-runtime。
- Produces: artifact:windows-portable-package。
- Gates: 打包测试、Windows runner ZIP/CLI smoke、工作流测试及中英文档一致性。
- Boundaries: x64 便携包，不增加安装器，不发布未经请求的新标签。

## Tasks

- [ ] `T01` 生成可双击启动的 Windows x64 ZIP 并加入发布与 CI。
  - **Test and RED**: 打包测试期望 exe/cmd/中英说明以及标注版本；工作流测试期望 Windows/macOS/校验三个产物，旧流程缺少 Windows。
  - **Execution logic**: 脚本编译 Windows x64 控制台 exe，生成 UTF-8 终端启动与 Git 缺失提示脚本，复制中英说明，ZIP 打包；Windows CI 解压并执行 CLI，release 上传包再统一校验。
  - **Files and responsibilities**: scripts/build-windows.py、对应测试、Windows 启动模板、CI 与 release；README 说明安装 Git/gh 可选及测试边界。
  - **Verification**: ZIP 文件名/内容/版本、空格/中文目录启动、Git 缺失提示和失败窗口、共享校验、macOS 回归；真实浏览器和窗口视觉为人工试用。
  - **Artifact handoff**: CI 提供可下载 Windows 测试包，后续用户指定标签时发布。

## Done

- [ ] 测试先行且打包与运行 Gate 通过。
- [ ] 文档与执行图更新并通过校验。

## STOP conditions

Windows runtime Gate 未通过，或打包需要超出 x64 便携版范围时停止依赖工作。
