# Phase 42: 资产对账观测底座 (R1) - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-06-27
**Phase:** 42-资产对账观测底座 (R1)
**Areas discussed:** 物化视图刷新策略, Dashboard 形态与位置, Layer 3 引擎 R1 边界, 健康度评分函数位置, 测试策略, operlog 与 R1 写操作边界

---

## 物化视图刷新策略

| Option | Description | Selected |
|--------|-------------|----------|
| 5min 定时 CONCURRENTLY 刷新 | 每 5min 执行 REFRESH MATERIALIZED VIEW CONCURRENTLY reconciliation_normalized | ✓ |
| 事件触发增量 | sys_port_mac/sys_workstation 变更时 trigger 增量 | |
| 双轨: 定时兜底+触发器加速 | 5min 定时 + 变更 trigger | |
| 5min 定时全量刷新(非 CONCURRENTLY) | 不用 CONCURRENTLY, 允许 refresh 期间读锁 | |

**User's choice:** 5min 定时 CONCURRENTLY 刷新
**Notes:** PG 原生支持 CONCURRENTLY（需 unique index，v0.3 schema 已用 `asset_id` 作 unique index ✓）。D-01/D-02 关联决策：失败仅日志不告警，启动后立即 refresh 一次，IP 字段 R1 最小（仅取 `ops_asset.machine_ip` 单值）。

---

## Dashboard 形态与位置

| Option | Description | Selected |
|--------|-------------|----------|
| 父路由 302 → dashboard | 访问 /asset/reconciliation 自动 redirect | ✓ |
| 父路由内嵌 dashboard 内容 | URL 短但子菜单 click 不跳页 | |

**User's choice:** 父路由 302 → dashboard
**Notes:** D-04。

| Option | Description | Selected |
|--------|-------------|----------|
| 双向打通(点击跳转) | Dashboard 图表点击跳异常列表 + 预填筛选 | ✓ |
| Dashboard 只读(不跳) | 仅展示，需手动到异常列表页输条件 | |

**User's choice:** 双向打通(点击跳转)
**Notes:** D-05。URL 模式：`/asset/reconciliation/exceptions?type=C&severity=critical&from=2026-06-20&to=2026-06-27`。

| Option | Description | Selected |
|--------|-------------|----------|
| 全量资产数 / 未解决异常数 / critical 数 / 7d 新增 / Top1 冲突类型 | 5 个都能用 SELECT COUNT(*) 拿 | ✓ |
| 健康度总评分(0-100) + 四个细分 | 需 HealthScore 函数 | |
| 运维 5 个 KPI | 含采集率/填充率/AD 同步率，R1 复杂 | |

**User's choice:** 全量资产数 / 未解决异常数 / critical 数 / 7d 新增 / Top1 冲突类型
**Notes:** D-06。**严禁**用 `list.length`，必须独立 COUNT 端点（`stat-cards-from-list-length-capped-at-100` 项目记忆）。

---

## Layer 3 引擎 R1 边界

| Option | Description | Selected |
|--------|-------------|----------|
| R1 同步做完整 Layer 3 | R1 不告警不转工单但写 sys_data_reconciliation | ✓ |
| R1 只跑 ETL, R2 才上引擎 | R1 admin 异常列表 0 条数据 | |
| R1 全量版本, R2 才增量 | 全量重算简单但慢 | |

**User's choice:** R1 同步做完整 Layer 3
**Notes:** D-07。满足 ROADMAP success criteria 7 异常列表展示 Type A-F 分布。

| Option | Description | Selected |
|--------|-------------|----------|
| MAC1/MAC2 → port_mac → info_point → workstation.user_id | 链路与 v0.3 §4 一致，mac1 优先 | ✓ |
| 只走 mac1 | 简单但漏无线 MAC | |
| mac1 与 mac2 并行查 UNION | 最准确但物化视图 2 倍行数 | |

**User's choice:** MAC1/MAC2 → port_mac → info_point → workstation.user_id
**Notes:** D-08。mac1 优先 + mac2 备选（`COALESCE`），避免物化视图单资产 2 行。

| Option | Description | Selected |
|--------|-------------|----------|
| Type A 仅作 dashboard 统计, 不进异常表 | 避免异常表膨胀 | ✓ |
| Type A 也写但默认隐藏 | 全量审计但 R1 异常表大 | |
| Type A 走专门表 | R1 多一张表 | |

**User's choice:** Type A 仅作 dashboard 统计, 不进异常表
**Notes:** D-09。

| Option | Description | Selected |
|--------|-------------|----------|
| 独立cron, MV 0/5/10min + 检测 1/6/11min | MV 刷新后 1min 跑检测 | |
| 合并成一个 job | 顺序 refresh→detection，原子性可控 | |
| MV 触发器驱动 detection | 需 pg_listen/notify | |

**User's choice:** 走 sys_job 表（api/v1/scheduler 现有页面管理）
**Notes:** D-10。运维在 UI 改 cron 表达式，无需发版。sys_job 表新增 4 个 job records：MV 刷新 / Layer 3 检测 / 静默期到期重检测 / 临时例外清理。

| Option | Description | Selected |
|--------|-------------|----------|
| 依赖 unique index(并发安全) | catch unique violation 静默处理 | ✓ |
| 先 SELECT 再 INSERT | TOCTOU 风险 | |
| UPSERT 语义 | PG 原生 ON CONFLICT | |

**User's choice:** 依赖 unique index(并发安全)
**Notes:** D-11。`uniq_recon_asset_type_open(asset_id, conflict_type) WHERE resolved_at IS NULL AND deleted_at IS NULL`。

---

## 健康度评分函数位置

| Option | Description | Selected |
|--------|-------------|----------|
| R1 不做 HealthScore | 5 KPI 不含 0-100 分 | ✓ |
| R1 做简化版 HealthScore | Dashboard 5 KPI 增一项 | |
| R1 引入 HealthScore 接口 + 占位实现 | 后续预留扩展点 | |

**User's choice:** R1 不做 HealthScore
**Notes:** D-12。R4 工位详情页 HealthCard 需要时再补。

---

## 测试策略

| Option | Description | Selected |
|--------|-------------|----------|
| sqlmock + test DB 集成测试 | 不引入 testcontainers，集成走 dev DB | ✓ |
| testcontainers-go 拉临时 PG | 引入新依赖 | |
| 人工 SQL 验证脚本 + Go unit test | 不写自动化 | |

**User's choice:** sqlmock + 集成测试
**Notes:** D-13。CI 阶段再决定是否引入 docker。

| Option | Description | Selected |
|--------|-------------|----------|
| 依赖 dev DB seed + e2e 验证 | 6 端点全 e2e 贵 | |
| 单元测试 sqlmock 为主 | 全 sqlmock 可能与真 SQL 漂移 | |
| 混合: 单元 + 集成 | sqlmock 全覆盖 + 1-2 端点 e2e | ✓ |

**User's choice:** 混合: 单元 + 集成
**Notes:** D-14。验证不走 `list.length` 路径。

| Option | Description | Selected |
|--------|-------------|----------|
| 依赖 dev DB seed + 手工 UAT | 不引入 Vitest | ✓ |
| Vitest + React Testing Library 组件测试 | 项目未用过 | |
| 两者都做 | 全覆盖 | |

**User's choice:** 依赖 dev DB seed + 手工 UAT
**Notes:** D-15。UAT 走 ROADMAP success criteria 7 全部 5 KPI + 3 图表 + 列表分页 + 筛选。

---

## operlog 与 R1 写操作边界

| Option | Description | Selected |
|--------|-------------|----------|
| 只定义 1 个:资产对账 | R1 最小 | ✓ |
| 4 个全定义(v0.3 计划) | R1 占位 | |
| 不加新常量, 复用"资产管理" | 简单但与 v0.3 不一致 | |

**User's choice:** 只定义 1 个:资产对账
**Notes:** D-16。R2-R4 再加 `ModuleReconciliationExceptionRule` / `AutoWorkorder` / `Export`。

| Option | Description | Selected |
|--------|-------------|----------|
| R1 全部写操作都记录 | 全审计 | ✓ |
| cron 触发不记 operlog | 只记用户主动写 | |
| R1 全部不记(纯读底座) | 不上 R2 再说 | |

**User's choice:** R1 全部写操作都记录
**Notes:** D-17。Layer 3 cron (`OperTypeSync`)、reconciliation 写入 (`OperTypeCreate`/`Update`)、静默期回收 (`OperTypeSync`)、sys_job 新增 4 cron (`OperTypeCreate`)。

| Option | Description | Selected |
|--------|-------------|----------|
| R1 不上 UI, R2 再加 | 列表只读 | ✓ |
| R1 就上"标记已解决"按钮 | 提前为人工闭环铺路 | |

**User's choice:** R1 不上 UI, R2 再加
**Notes:** D-18。R1 success criteria 7 说"展示"不是"可操作"。

---

## Claude's Discretion

R1 plan-phase 时由 Claude 自决的细节（用户未明确选项）：
- 异常列表列顺序与默认排序
- TopUnresolved limit（建议 10，与 ROADMAP success criteria 7 一致）
- 6 个 Statistics 端点缓存策略（R1 不缓存建议）
- HealthTrend 时间窗口默认 7d
- Dashboard 趋势图 x 轴粒度（按天 vs 按小时）
- migration_NNN 编号（需查 `internal/core/db/migrations/` 现有最大编号 +1）
- 例外规则子菜单 R1 是否创建空壳页面（建议 R1 不创建，R3 接入时再加）

## Deferred Ideas

显式推后到 R2-R5 阶段，R1 不实现：
- **R2（Phase 43）**：critical/high 自动转工单 + WebSocket 推送 + SysNotice + 6 类工单模板 + 修复回写 7d 静默期 + 24h 节流 + 异常标记已解决 UI
- **R3（Phase 44）**：IP 段例外规则引擎（CIDR GiST 索引）+ 例外规则 CRUD admin 页 + 命中测试工具 + Excel 导入导出例外规则
- **R4（Phase 45）**：工位详情页整合 HealthCard + HealthBadge + ReconciliationDrawer + 资产详情摘要块 + HealthScore 函数
- **R5（Phase 46, 可选）**：高置信度修复建议（confidence ≥0.9）+ 人工确认 UI + 一键回滚

v0.3 决策点显式推到对应 phase：
- D13 告警分发范围 → R2
- D14 R5 半自动修复阈值 → R5
- D18 临时例外默认有效期 → R3
