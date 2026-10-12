# Todo: 普通文件夹初始化与绑定

- **Status**: 验证成功
- **Scope**: TUI 选择现有普通文件夹后初始化 main 分支，进入已有的远端关联/创建流程；不自动提交或上传。
- **Baseline**: main / 5edf39b

## Behavior

- Input: 所选帐号与现有文件夹。
- Output: 已有仓库保持历史和远端；普通文件夹初始化后按照帐号 HTTPS/SSH 类型建议远端。
- Constraints/errors: 不创建不存在的目录；不重新初始化裸仓库/worktree；帐号或初始化失败明确报错。

## Test first

- TUI 普通文件夹进入远端页，可关联远端并绑定，main 无提交。
- SSH 地址建议匹配帐号；既有仓库不改变 HEAD/origin；不存在目录失败。
- Adapter 的真实 Git 初始化测试，预期 RED 为缺失初始化接口/仍拒绝普通目录。

## Implementation

- Git Adapter 增加窄初始化端口；应用服务负责准备仓库；TUI 复用远端引导。
- 批量绑定和 CLI 保持现有合同，本轮不增加自动提交功能。

## Verification

- Focused: go test ./internal/adapters/gitcli ./internal/presentation/tui
- Regression/lint/build: go test ./...; go vet ./...; scripts/build-app.sh

## Done

- [x] Test first and RED observed.
- [x] Implementation stays in scope.
- [x] Checks pass.
- [x] Status is 验证成功.

## Evidence

- RED: `TestPlainFolderCanInitializeAssociateAndBind` 和 SSH 场景返回“不是 Git 仓库”；初始化端口测试因接口尚不存在失败。
- GREEN: `go test ./internal/adapters/gitcli ./internal/adapters/storage ./internal/presentation/tui` 通过。
- Regression: `go test ./...`、`go vet ./...`、`git diff --check` 通过。
- Build: `go build -o bin/gitra ./cmd/gitra` 与 `bash scripts/build-app.sh` 通过；通用架构应用包签名校验通过。
- Mapping: 初始化端口、Git Adapter、应用服务及 bootstrap 由 `initialize_test.go` 和 TUI `folder_setup_test.go` 的真实 Git 集成场景验证；既有 TUI 回归验证远端创建流程。
- Verified: 既有仓库 HEAD/origin 保留，普通目录无自动提交，缺失路径/错误帐号/裸仓库/初始化不可用明确失败，取消关联后保持未绑定。
- Limitation: HTTPS 可在线校验/创建；SSH 关联已有地址；CLI 和批量绑定沿用原合同。
