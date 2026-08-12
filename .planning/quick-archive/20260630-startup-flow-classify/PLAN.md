---
title: 启动流程分类与文档化
slug: startup-flow-classify
created: 2026-06-30
status: in-progress
---

# 启动流程分类与文档化

## 目标

梳理 XingRan-Next 后端启动流程，区分：
- **每次必跑**：每次进程启动都必须执行
- **一次性**：仅在首次部署或 schema 变更时执行
- **混合/幂等**：每次都执行但内部有"已应用则跳过"判断

输出 `docs/启动流程清单.md`，便于：
- 开发环境调试快速定位启动卡点
- 评估每次启动的耗时与风险
- 未来如果需要拆分"setup"和"start"两个步骤，提供分类基线

## 范围

**纳入分析**：
- `cmd/main.go` 的 `init()` / `main()` 及其调用链
- `internal/core/core.go` 的 `New()` / `Init()` / `Close()`
- `internal/core/db/database.go` 的 `NewDatabase()` / `AutoMigrate()` / `InitData()`
- `cmd/main.go` 末尾的 seed 函数（`initMACHistoryRetentionConfig`、`importOUIData`）

**不在本次范围**（仅梳理不改动）：
- 实际代码重构（本次仅文档化，不改代码）
- 拆分 setup/start 脚本（用户已确认仅梳理分类）
- 加新的环境变量开关

## 文档结构

`docs/启动流程清单.md` 包含：
1. 总览：5 个启动阶段 × 3 种分类
2. 详细表格：每个步骤的 [阶段 / 操作 / 分类 / 关键代码位置 / 跳过机制 / 风险]
3. 风险清单：每次启动可能引发的问题（如 AutoMigrate 阻塞 MV、SM4 默认 key 警告）
4. 优化建议：哪些可以挪到 setup 脚本或一次性迁移工具

## 验证

- [ ] 文档涵盖 `main.go` → `Core.Init()` 全部 17+ 步骤
- [ ] 每条 migration 至少有一行（group 形式列出亦可）
- [ ] seed 函数都标注了 count 检查路径
- [ ] 风险清单引用具体行号

## 提交策略

单一提交 `docs: 添加启动流程分类清单文档`（仅新增 markdown，无代码变更）。