# Phase 69 Deferred Items (out-of-scope discoveries)

## 2026-08-19 · 69-01 executor

| # | Item | Discovered during | Why not fixed here |
|---|------|-------------------|--------------------|
| 1 | `internal/services/system/settings_service_test.go:58` 编译失败：`unknown field configService in struct literal of type settingsService`。工作区未提交的 default-theme 删除遗留改动把 `settingsService` 构造函数从 2 参改为 1 参（`git diff internal/services/system/settings_service.go` 可见 `configService` 字段被移除），但已提交的测试文件仍引用该字段 → 整个 `internal/services/system` 包的 test 构建在**主工作树**失败。 | 69-01 T3 验证（`go test ./internal/services/system/`） | 属于前一会话 in-flight 遗留改动（settings/default-theme 重构），该测试本身测的正是被删除的 default-theme 逻辑（`sys.theme.default` 回退），预计随该会话提交一并更新；executor 明确禁止触碰遗留改动文件。已在干净隔离 worktree（`git worktree add --detach` @ da5d0a0）验证：`go test ./internal/services/system/` 与 `go test ./internal/models/` 全绿，证明批 1 替换零回归。 |
| 2 | `internal/services/system/` 与 `internal/models/` 若干文件存在 gofmt 偏差（如 role_service.go 尾部多余空行、user_service.go 双空行、widget_data_fetcher.go 结构体 tag 对齐、以及 apikey_service.go / cache_keys.go / config_service.go 等未触碰文件）——本仓 gofmt 漂移为存量状态。 | 69-01 T3 gofmt 检查 | 非本 plan 改动引入（偏差均位于未编辑区域）；按 scope-constrainment 原则不顺手重构无关格式。若后续需要可在 Phase 63 工具链 phase 统一收口。 |
