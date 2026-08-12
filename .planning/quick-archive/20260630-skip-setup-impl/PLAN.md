---
title: 启动流程优化 - 加 SERVER_SKIP_SETUP 开关
slug: skip-setup-impl
created: 2026-06-30
status: complete
---

# 启动流程优化 - 加 SERVER_SKIP_SETUP 开关

## 目标

基于 `docs/启动流程清单.md` 的"长期目标"方案，实现中期方案：
新增 `SERVER_SKIP_SETUP` 环境变量，开启后跳过 🟢 一次性步骤（InitData / 默认角色菜单 / cmd-level seed）。

让"启动"与"初始化"分离，避免 DB 已稳定环境下每次启动都跑 10+ 表 count 查询。

## 实现

| # | 改动 | 文件 |
|---|------|------|
| 1 | `ServerConfig.SkipSetup bool` 字段 + `SERVER_SKIP_SETUP` 环境变量覆盖 | `internal/config/config.go` |
| 2 | `core.Init()` step 3-4 用 SkipSetup 条件化跳过 InitData / 默认角色菜单 | `internal/core/core.go` |
| 3 | `cmd/main.go` 用 SkipSetup 条件化跳过 MAC history retention / OUI import | `cmd/main.go` |
| 4 | `docs/启动流程清单.md` §6 同步 SkipSetup 开关文档 | `docs/启动流程清单.md` |

## 跳过范围

✅ InitData（10+ 张表 count 查询）
✅ Permission 默认角色菜单
✅ MAC history retention 配置
✅ OUI 厂商数据导入

❌ AutoMigrate + 36+ migrations（保留，schema 可能变）
❌ DB 连接 / Cache / Cron 注册 / HTTP（每次必跑）
❌ RefreshView 异步刷新（R1 冷启兜底）

## 使用

```bash
# dev 本地开发（默认）
./xingran-backend.exe

# 生产/CI 部署（跳过一次性步骤）
SERVER_SKIP_SETUP=true ./xingran-backend.exe
```

## 提交

5 个原子提交：config → core → cmd → docs → planning