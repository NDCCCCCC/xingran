---
title: 资产对账（多源数据 Reconciliation）v1.17 阶段种子
trigger_condition: v1.16 技术债清理 100% 完成 + 用户/团队确认 observe-only 策略与资产管理归属
planted_date: 2026-06-27
origin: gsd:explore session — asset-reconciliation-strategy v0.3
related_note: .planning/notes/asset-reconciliation-strategy.md
related_todo: .planning/todos/pending/v1.17-reconciliation-decisions.md
priority: high
status: pending (等待 promote 到 ROADMAP)
estimated_phases: 5 (R1-R5)
estimated_duration: 8-12 周
---

# 资产对账 v1.17 阶段种子

## 触发条件（满足任一即可启动）

1. v1.16 技术债清理 100% 完成（✅ 2026-06-26 已达成：11/11 plans，5/5 code gaps fixed）
2. 用户/团队对 observe-only 策略与资产管理归属达成共识（✅ 2026-06-27 已确认）
3. 端口采集覆盖率达到 ≥80%（R1 启动前置）

## 范围摘要

**核心目标**：建立"实物层 vs 声明层"对账引擎，使资产系统数据保持与实际情况相符。

**关键决策**：
- 策略：Observe-only（不修改任何业务表，仅记录 + 告警 + 人工修复）
- 例外机制：IP 段级别 + 冲突类型 + 范围三维
- 菜单归属：资产管理 / 数据质量
- 权限命名：`asset:reconciliation:*`
- API 前缀：`/asset/reconciliation/*`

## 阶段拆分

### R1：观测底座（2-3 周）— **首个实施 phase**
- 物化视图 `reconciliation_normalized`（5min 增量刷新）
- `sys_data_reconciliation` 主表
- admin 异常列表（只读）
- 跑覆盖率报告（端口数 vs 信息点数 vs 工位数）

### R2：告警 + 工单闭环（2-3 周）
- critical/high 自动转 workorder
- WebSocket 实时推送
- SysNotice 通知
- 工单模板 6 个（按 Type A~F）

### R3：置信度评分 + 降噪（1-2 周）
- 置信度评分函数（physical +0.5 / declared +0.3 / ad +0.2）
- 24h 节流规则
- `sys_reconciliation_exception` 表
- 例外规则管理 UI
- 命中测试工具

### R4：工位详情整合（1-2 周）
- 顶部健康度卡片
- 行内徽标 + 抽屉详情
- 跨模块 service 层调用 + 权限降级

### R5：半自动修复（2-4 周，可选）
- 高置信度建议修复（confidence ≥0.9）
- 人工确认 UI
- 一键回滚机制

## 前置依赖

| 依赖 | 状态 | 来源 |
|------|------|------|
| 网络设备端口采集 | ✅ 已就绪 | sys_network_device, sys_port_mac 表 |
| 工位-用户链路 | ✅ 已就绪 | sys_workstation + user_id 字段（Phase 34 已修复 user_id/status 联动） |
| 工位-信息点-端口链路 | ✅ 已就绪 | sys_info_point, sys_workstation_info_point |
| 资产模块 MAC 字段 | ✅ 已就绪 | sys_asset.mac 字段 |
| 资产模块 IP 字段 | ⚠️ 需 migration | sys_asset 缺 ip 列（待 R1 加） |
| AD 同步 | ✅ 已就绪 | sys_user.ad_dn + sys_user_ad_attrs |
| operlog 体系 | ✅ 已就绪 | Phase 34 全模块覆盖 |
| workorder 模块 | ✅ 已就绪 | internal/services/workorder/ |

## 关键风险

| 风险 | 缓解 |
|------|------|
| 端口采集覆盖不全 | R1 阶段先跑覆盖率报告 |
| AD managed_by 不可靠 | 物理链路优先，AD 仅作 fallback |
| 菜单权限命名空间割裂 | 显式声明 + RequirePermissionsWithQuery |
| 现有 Phase 13 技术债回归 | 复用 constraint 命名规范 + operlog 强制约束 |
| 修复回写触发循环 | 7d 静默期 + 异步通知引擎 |

## 启动检查清单

启动 `/gsd-discuss-phase` → `/gsd-plan-phase` 前必须确认：

- [ ] 端口采集覆盖率 ≥ 80%
- [ ] 资产管理模块 owner 已确认（一般由后端核心组 + 资产业务 owner 共同评审）
- [ ] workorder 模块已支持自定义模板与自动创建
- [ ] operlog 模块对 asset 模块有 audit 权限
- [ ] 跨模块权限边界（ops/workstation 需具备 asset:reconciliation:list）已与权限 owner 对齐
- [ ] 不在 v1.16 收尾期（避免与 in-flight 的 13-07/08/09/10 冲突）

## Promote 触发

满足以下任一条件即可 promote 到 ROADMAP.md 候选：

1. v1.16 收尾 ≥ 7 天无回归
2. 用户主动发起 `/gsd-progress --next` 且 current phase 完结
3. 业务侧提出具体数据质量诉求（资产盘点对不上、离职未移交等）

## 关联文档

- 决策记录：[`.planning/notes/asset-reconciliation-strategy.md`](../notes/asset-reconciliation-strategy.md)
- 决策追踪：[`.planning/todos/pending/v1.17-reconciliation-decisions.md`](../todos/pending/v1.17-reconciliation-decisions.md)
- 当前进度：[`.planning/STATE.md`](../STATE.md)
- 路线图：[`.planning/ROADMAP.md`](../ROADMAP.md)

---

**注**：本 seed 不承诺立即执行，仅在触发条件满足后由用户/团队决定 promote。v0.3 架构决策记录已完整沉淀为 Note，决策点状态已记录到 Todo。