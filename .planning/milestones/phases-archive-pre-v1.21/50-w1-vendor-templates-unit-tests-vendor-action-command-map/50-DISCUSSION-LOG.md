# Phase 50: W1 — Vendor Templates + Unit Tests - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-07-06
**Phase:** 50-w1-vendor-templates-unit-tests-vendor-action-command-map
**Areas discussed:** 4 个 gray area（Action 命名 / Params 类型 / 多命令策略 / Golden 策略）

---

## 用户中断 4 选项讨论，要求匹配之前处理风格

用户回复"之前是怎么处理的，保持一致"，指示采用 Phase 3 / Phase 48 / Phase 49 模式：**轻量讨论 + Claude discretion 列表 + CONTEXT.md 记录已锁定决策**。

参考依据：
- Phase 3 DISCUSSION-LOG："Areas discussed: (none — user chose to skip all, Claude discretion)"
- Phase 48 CONTEXT：14 条 D-XX 决策全部来自前期 /gsd-explore 6 轮讨论 + 36 个真机样本
- Phase 49 CONTEXT：纯 gap 诊断 + 修复顺序，无选项问答

## Gray Areas Considered (未追问)

| Area | Claude 默认选择 | 理由 |
|------|----------------|------|
| Action 命名风格 | **D-03**: PascalCase 常量 + snake_case 字符串值 | 兼顾 Go 代码可读 + 审计日志可读 |
| Params 参数类型 | **D-04**: 显式 `PortTemplateParams` struct | 强类型 + 字段文档化 + 易扩展 |
| 多命令 action 策略 | **D-05**: 统一 `[]string` 返回 | 与 `scrapli_wrapper.SendConfigs([]string)` 天然对齐 |
| 测试 golden 数据组织 | **Claude Discretion**: 倾向内联 table-driven | 15 用例不构成引入 testdata JSON 的必要性 |

## Claude's Discretion (已记入 CONTEXT.md)

- 测试用例编排（表驱动 vs t.Run 子测试）
- Golden 数据存放形式（内联 vs testdata JSON）
- Sentinel error 定义位置（同文件 vs 独立 errors.go）
- `PortAction.String()` 方法是否暴露（倾向暴露）

## Deferred Ideas

- Maipu / Cisco 厂商模板（v1.19 OUT-OF-SCOPE）
- Description 内容转义（D-06 备注）
- 接口名短/全称归一化（Phase 51 实施时核实）
- 模板数据库抽象（v1.19 init 锁定"落地为先"）
- `operlog-exclude-paths` todo（score 0.2，归 Phase 52 处理）

## 决策结果

9 条 D-XX 决策（D-01 ~ D-09）已写入 `50-CONTEXT.md` 供下游 planner / researcher 使用。
本 phase 不进入选项问答流程，匹配项目一贯的"CONTEXT 是已锁定决策的归档"模式。
