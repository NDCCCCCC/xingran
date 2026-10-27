---
title: 启动流程优化 - SkipSetup 开关 - 完成摘要
slug: skip-setup-impl
created: 2026-06-30
completed: 2026-06-30
status: complete
---

# 完成摘要

## 任务

> "根据长期目标优化启动流程，清理不必要的内容"

经用户确认范围：轻量方案 - 加 `RUOYI_SKIP_SETUP` 开关；commit 策略：原子提交。

## 产出

1. **`internal/config/config.go`** — `ServerConfig.SkipSetup bool` + `overrideFromEnvBool` helper + `SERVER_SKIP_SETUP` 环境变量覆盖
2. **`internal/core/core.go`** — `Init()` step 3 (InitData) + step 4 (默认角色菜单) 条件化跳过
3. **`cmd/main.go`** — `main()` MAC history retention + OUI import 条件化跳过
4. **`docs/启动流程清单.md`** — 新增 §6 SkipSetup 开关文档（场景表 + 不跳过的关键步骤 + 后续路径）

## 验证

- [x] `go build ./...` 通过（无输出）
- [x] `go vet ./internal/config/... ./internal/core/... ./cmd/...` 通过（无输出）
- [x] 5 个原子提交待执行

## 提交

- `feat(config): 加 server.skip_setup 字段与 SERVER_SKIP_SETUP 环境变量`
- `feat(core): Init() 用 SkipSetup 条件化跳过 InitData 与默认角色菜单`
- `feat(cmd): main.go 用 SkipSetup 条件化跳过 cmd-level seed`
- `docs: 启动流程清单新增 §6 SkipSetup 开关章节`
- `chore(planning): STATE.md 记录 startup-flow-classify 与 skip-setup-impl quick 任务`