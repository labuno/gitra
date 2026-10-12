# WIN-001: Windows x64 便携版

## Runtime (runtime)

目标：安装 Git for Windows 后，Windows x64 用户可运行现有 CLI/TUI，完成帐号登录、普通文件夹初始化、绑定、首次上传与解绑。浏览器登录使用 gh，访问码登录不要求 gh。源码无需改成 GUI，不依赖 WSL 或 Go 工具链。

实现边界：Windows 文件锁 Adapter 复用现有取消和资源释放合同；Git 发现结果及 SSH/helper 命令在边界处理原生路径、空格和中文；凭据优先沿用配置的 helper，默认 Windows 可用的 Git Credential Manager，不增加凭据体系。

验证：本机单元与 Unix 回归、Windows runner 的锁/真实 Git/TUI/两帐号凭据隔离/本地首次推送测试。Windows Credential Manager 使用测试凭据验证存取并清理，不调用真实用户帐号。真实浏览器授权和桌面手工操作保留为用户试用事项。

## Package (package)

目标：提供 `Gitra-Windows-x64.zip`，包含 `gitra.exe`、`Gitra.cmd` 和中英使用说明。双击启动脚本检查 Git，开启 UTF-8 后运行终端界面，错误保留窗口。不增加安装器或内嵌 Git/gh。

发布保留 macOS 应用包，增加一个 Windows x64 包和共享 checksums.txt；不恢复多架构矩阵。Windows CI 构建包，验证 ZIP 内容、版本号和可执行文件在 Windows 环境的 CLI 行为，并提供测试 Artifact。本轮不移动旧标签或自动发布新标签。

## Done

Runtime 与 Package 的执行图 Gate 全部通过；中英 README 准确说明依赖、启动和功能限制；提供 Windows CI 与打包验证证据。
