Gitra Windows x64 便携版

1. 安装 Git for Windows：https://gitforwindows.org/ 。保留默认的 Git Credential
   Manager 和命令行 PATH 选项，安装后重新打开终端。不需要安装 Go 或 WSL。
2. 完整解压 ZIP，再双击 Gitra.cmd。界面在终端窗口中运行，当前使用中文标签。
3. 方向键选择，Enter 确认，Esc 返回，Q 退出。
4. GitHub 浏览器授权需要 GitHub CLI（gh）：https://cli.github.com/ 。
   也可以直接粘贴访问码，此方式不需要 gh。
5. 选择项目文件夹并绑定账号。普通文件夹可以在绑定时初始化 Git；
   首次上传前先创建本地提交，然后按 U 上传。

HTTPS 凭据沿用已配置的 Git helper，通常为 Git Credential Manager，
使用 Windows Credential Manager 保存。带口令的 SSH 密钥需要 SSH agent。
gitra.exe 同时支持命令行，例如 gitra.exe --version。

出现错误时，启动脚本保留窗口方便阅读。此版本是终端应用，
浏览器授权和桌面交互显示仍需要实际试用确认。
