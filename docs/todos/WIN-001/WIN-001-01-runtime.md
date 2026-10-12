# WIN-001-01: Windows 运行适配

- **Todo ID**: WIN-001-01
- **Execution Map**: execution-map.json
- **Todo maturity**: DECOMPOSED
- **Task detail format**: v1
- **Source baseline**: docs/plans/windows-portable-plan.md / main 6c10584

## Outcome

Windows runner 证明现有帐号和绑定流程在 Windows 路径与凭据环境下可运行。

## Contract

- Produces: artifact:windows-runtime。
- Gates: 本机完整 Go 回归；Windows 核心行为测试。
- Boundaries: 仅增加 Windows 窄适配，不增加通用基础设施；真实浏览器交互留给人工试用。

## Tasks

- [x] `T01` Windows 锁支持串行写入、取消等待和执行失败后释放。
  - **Test and RED**: 锁合同测试增加错误释放；Windows runner 运行既有串行/取消测试，旧占位实现返回 unsupported。
  - **Execution logic**: 打开锁文件，非阻塞 LockFileEx，争用时复用现有等待间隔并检查 context，其他错误返回；退出释放锁并关闭文件。
  - **Files and responsibilities**: storage/locker_windows.go 与 locker_other.go 负责平台边界，locker_test.go 负责行为。
  - **Verification**: 串行写入、等待超时、回调错误和释放，不新增调度或锁状态平台。
  - **Artifact handoff**: 为 Windows runtime 提供和 Unix 相同的应用写事务合同。
- [x] `T02` Windows 路径、SSH 指令与 Git helper 支持真实 Git 和帐号隔离。
  - **Test and RED**: 新增含空格/中文路径、SSH shell 参数、配置/默认 helper、Windows 凭据存取和绑定/解绑/本地首次上传场景；缺失规范化和默认 helper 的断言失败。
  - **Execution logic**: 原生 Git 输出在发现边界规范化；shell 指令引用经过路径处理；沿用配置 helper，Windows 检测 manager，未安装明确报错；凭据按用户名区分。
  - **Files and responsibilities**: gitcli/git.go、secretstore/helper.go、sshkey/sshkey.go 的窄边界；新增对应测试与 Windows e2e。
  - **Verification**: 空格/中文目录、两帐号身份/凭据不混用、初始化无自动提交、上传到本地裸远端、解绑还原；Windows runner 测试。
  - **Artifact handoff**: package 节点可依赖已验证 Windows runtime 编译和运行便携包。

## Done

- [x] RED/GREEN 与本机/Windows Gate 证据已记录。
- [x] 执行图更新并通过结构校验。

## STOP conditions

需引入安装器、内嵌 Git 或真实用户凭据才能验收时重新确认范围。

## Evidence / Handoff

RED: Windows run 38222813392 confirmed unsupported locking and incorrect paths/quoting. GREEN: run 38224078961 passed Windows core tests, Linux full regression and macOS bundle smoke. Local `go test ./...`, `go vet ./...` and Windows cross-compilation passed. Credentials were tested against real Git Credential Manager / wincredman with two synthetic accounts; browser authorization and desktop TUI interaction remain manual checks. Package node can consume this runtime.
