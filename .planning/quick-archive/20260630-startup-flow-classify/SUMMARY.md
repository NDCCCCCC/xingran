---
title: 启动流程分类文档化 - 完成摘要
slug: startup-flow-classify
created: 2026-06-30
completed: 2026-06-30
status: complete
commit: - # 纯文档化产出,无 git commit(用户未要求 commit)
---

# 启动流程分类 - 完成摘要

## 任务

> "当前系统处于开发阶段，没有上线，因此对于数据库迁移的需求不大，不需要每次更新后进行生产环境的数据库迁移，上显示只要一次性的根据现有数据结构生成数据库就行了。因此，请帮我优化项目启动流程，到底哪些是必须要每次运行的，哪些是只需要运行一次的。"

**用户确认范围**：仅梳理分类 + 文档化（不改代码）

## 产出

1. **`docs/启动流程清单.md`** — 主要交付物
   - 7 阶段 × 3 分类（🔴 每次必跑 / 🟢 一次性 / 🟡 混合幂等）
   - §3 详细分类表覆盖 `main.go` → `Core.Init()` → `core.Init()` 全部 60+ 步骤
   - §4 风险清单：MV 阻塞、SM4 默认 key、GORM 命名冲突、AD 池空、Cache 预热等
   - §5 未来 setup/start 拆分建议（含 6 个可拆分步骤 + dev 环境兼容方案）

2. **`.planning/quick/20260630-startup-flow-classify/PLAN.md`** — 任务规划
3. **`.planning/STATE.md`** 更新 — Quick Tasks Completed 表 +1 行

## 关键发现摘要

| 阶段 | 每次必跑 | 一次性 | 混合/幂等 |
|------|---------|--------|----------|
| `init()` / `main()` 配置 | 5 | 0 | 0 |
| `core.New()` 构造 | 3 | 0 | 0 |
| `core.Init()` 核心 | 11 | 3 (DB 创建/InitData/Permission) | 5 (AutoMigrate/RefreshView/MV drop/分区/同步周期 job) |
| cmd 末尾 seed | 0 | 0 | 2 (MAC history retention / OUI) |
| HTTP 引擎 + 路由 | 7（含 2 个 debug-only） | 0 | 0 |
| 服务 + 关闭 | 2 | 0 | 0 |
| **AutoMigrate 内嵌** | 1 (审计) | 0 | 36+ 显式 migrations + 1 cleanup + 1 MV drop |
| **总计** | **29** | **3** | **44+** |

## 后续建议（未实施）

- 短期：dev 环境保留现状
- 中期：新增 `RUOYI_SKIP_SETUP=1` 环境变量 + `xingran-backend setup` CLI
- 长期：CI/CD 拆分 `setup --db --seed` 与 `serve` 两阶段

## 验证

- [x] 文档涵盖 `main.go` → `Core.Init()` 全部 17+ 步骤
- [x] 36+ 显式 migrations 在表中以 group 形式列出（详见 §3.4 第 2e-2q 行）
- [x] seed 函数全部标注了 count 检查路径
- [x] 风险清单引用具体行号（如 `database.go:91-93`、`core.go:178-181`）

## 验证前 commit

未提交。文档化产出 + STATE.md 更新均未 commit，等待用户审阅。