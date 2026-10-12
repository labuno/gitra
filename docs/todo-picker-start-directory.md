# Todo: 保存绑定选择器起始目录

- **Status**: 验证成功
- **Scope**: 选择器按 S 保存当前目录为起始目录，跨帐号和重启生效。
- **Baseline**: main / 5edf39b

## Behavior

- Input: 选择器当前目录与 S 按键。
- Output: 持久化在 gitra 配置目录；之后打开选择器优先使用保存的目录。
- Constraints/errors: 未设置时沿用现有默认与会话记忆；设置读写失败或目录已删除时显示错误。

## Test first

- 保存目录后新建应用实例仍使用它，优先于会话临时目录。
- 未设置、设置文件损坏、持久化失败以及已删除目录。
- 预期 RED: S 无作用，新实例不记忆目录。

## Implementation

- 窄偏好接口、JSON 存储 Adapter、bootstrap 注入；TUI 按键及帮助文案。

## Verification

- Focused: go test ./internal/adapters/storage ./internal/presentation/tui
- Regression/lint/build: go test ./...; go vet ./...; scripts/build-app.sh

## Done

- [x] Test first and RED observed.
- [x] Implementation stays in scope.
- [x] Checks pass.
- [x] Status is 验证成功.

## Evidence

- RED: `TestPickerStartDirectorySurvivesRestart` 按 S 后没有保存反馈；存储测试因尚无偏好 Adapter 失败。
- GREEN: 存储读写/损坏/失败测试以及 TUI 重启、优先级、删除目录、保存失败场景通过。
- Regression: `go test ./...`、`go vet ./...`、`git diff --check` 通过。
- Build: 命令行二进制和 macOS 通用应用包已更新；签名校验通过。
- Mapping: 偏好接口与 bootstrap 注入由 TUI 跨应用实例测试覆盖；持久化由 `picker_preferences_test.go` 覆盖；视图与按键由 TUI 测试覆盖。
- Limitation: 保存目录已删除或设置文件损坏时显示错误，需修正或删除配置目录的 `picker.json`。
