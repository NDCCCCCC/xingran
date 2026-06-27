---
type: requirements
project: RuoYi-Next
milestone: v1.17
status: planning
last_updated: 2026-06-27
---

# Requirements: v1.17 资产对账 (Asset Reconciliation)

**Project:** RuoYi-Next 网络运维管理系统
**Milestone:** v1.17 资产对账 (Asset Reconciliation)
**Status:** Planning — 2026-06-27 milestone started
**Last Updated:** 2026-06-27
**Previous:** v1.16 完结于 Phase 13 gap closure (8 atomic commits, score 18/18)

**关键决策 (locked, 来自 v0.3):**
- 策略：Observe-only（不修改业务表）
- 菜单归属：资产管理 / 数据质量
- 权限命名：asset:reconciliation:*
- API 路由：/asset/reconciliation/*
- Owner：运维 + 资产 + 权限 同一人

---

## Milestone v1.17 Requirements

### RECON — 对账引擎核心

- [ ] **RECON-01**: 系统能建立三路对账（物理链路 vs 系统责任人 vs AD managed_by），并按 Type A-F 分类（A 健康 / B 无主 / C 不一致 / D 未上线 / E 幽灵 / F AD 不一致）
- [ ] **RECON-02**: 系统能基于 `ops_asset.mac1/mac2` + `sys_port_mac` + `sys_info_point` + `sys_workstation.user_id` 反向推导物理链路 user_id
- [ ] **RECON-03**: 系统能计算置信度评分（physical +0.5 / declared +0.3 / ad +0.2），低于阈值进异常池
- [ ] **RECON-04**: 系统能维护 `reconciliation_normalized` 物化视图，5 分钟增量刷新
- [ ] **RECON-05**: 系统能在冲突检出时立即写入 `sys_data_reconciliation`，包含 raw_snapshot (JSONB) 完整快照
- [ ] **RECON-06**: 系统能在修复回写后保留 7 天静默期（避免循环触发）
- [ ] **RECON-07**: 系统能基于 `sys_config` 的 `asset.reconciliation.score.*` 与 `view.refresh_interval` 参数化运行行为

### EXCEPTION — 例外规则

- [ ] **EXCEPTION-01**: 系统能按 CIDR 格式（如 `192.168.0.0/16`）配置 IP 段级别例外规则
- [ ] **EXCEPTION-02**: 系统能配置 5 种 actions 组合（no_alert / no_notice / no_workorder / skip_severity / silence），多条规则取并集
- [ ] **EXCEPTION-03**: 系统能在例外规则中设定有效期（expires_at），到期自动停用
- [ ] **EXCEPTION-04**: 系统能提供"命中测试"工具，输入 IP/CIDR 实时返回命中规则与合并效果

### WORKORDER — 工单闭环

- [x] **WORKORDER-01**: 系统能在 critical / high 异常出现时，自动调用 `workorder.BaseService.Create()` 生成工单（带 6 类冲突分类）
- [ ] **WORKORDER-02**: 系统能在工单状态变更时通过 `operlog.Record()` 写入审计日志

### MONITOR — 看板 + 告警

- [ ] **MONITOR-01**: 系统能提供 6 个 Statistics 专用 COUNT 端点（Summary / ByConflictType / BySeverity / HealthTrend / TopUnresolved / ExceptionRuleStats），不依赖 `list.length` 防 MaxPageSize 钳制
- [ ] **MONITOR-02**: 系统能通过 WebSocket 实时推送 critical 异常到前端
- [ ] **MONITOR-03**: 系统能将高危异常写入 `sys_notice` 系统通知

### INTEGRATE — 前端整合

- [ ] **INTEGRATE-01**: 系统能在工位详情页（`/ops/workstation/:id`）顶部显示健康度卡片（5 KPI + 趋势 mini chart），无权限时静默隐藏
- [ ] **INTEGRATE-02**: 系统能在工位详情页的资产设备子表 + 域控设备子表中加入"对账健康"列（行内徽标）
- [ ] **INTEGRATE-03**: 系统能在点击徽标时打开 `ReconciliationDrawer`（冲突摘要 / 历史变更 / 例外规则三 Tab）

### AUDIT — 审计 + 血缘

- [ ] **AUDIT-01**: 系统能在所有对账相关写操作（标记已解决 / 例外 CRUD / 自动转工单）中调用 `operlog.Record()` 或 `operlog.RecordWithBody()`
- [ ] **AUDIT-02**: 系统能在 `raw_snapshot` JSONB 中冻结触发时刻的三路原始值，支持事后溯源

### INFRA — 基础设施复用

- [ ] **INFRA-01**: 系统能在 `sys_dict_type` / `sys_dict_data` 中 seed 4 个字典（`asset_reconciliation_conflict_type` / `asset_reconciliation_severity` / `asset_reconciliation_exception_action` / `asset_reconciliation_status`）
- [ ] **INFRA-02**: 系统能在 `sys_config` 中 seed 8 个 config 项（refresh_interval / score.physical / score.declared / score.ad / default_expiry_days / critical_threshold / silence_after_resolved_hours / health.score_weights）
- [ ] **INFRA-03**: 系统能在 `sys_workorder_category` 中 seed 6 个对账冲突分类（A-F）
- [ ] **INFRA-04**: 系统能在 `internal/services/asset/cache_keys.go` 定义 8 个 cache key 常量与 helper 函数
- [ ] **INFRA-05**: 系统能在 `src/lib/queryKeys.ts` 注册 9 个 reconciliation queryKey

### Future / Out of Scope

| 项 | 原因 |
|----|------|
| 自动覆盖业务表 | 用户明确选择 observe-only；自动覆盖需后续 R5 演进 |
| AD 域控实时同步 | 已知不可靠（code 32 / lockout），仅作 fallback 信号 |
| 机器学习异常检测 | 当前置信度评分够用；ML 是远期方向 |
| VDI 虚拟机对账 | VDI 模块独立 domain，关联用户通过 AD，反向推导与现有链路重合 |
| 工单自动关闭 | 工单状态由人工闭环，不应系统自动关单 |

### Traceability

| Requirement | Phase | Plan |
|-------------|-------|------|
| RECON-01~07 | Phase 42 (R1) | 42-01 / 42-02 / 42-03 |
| EXCEPTION-01~04 | Phase 44 (R3) | 44-01 / 44-02 |
| WORKORDER-01~02 | Phase 43 (R2) | 43-01 / 43-02 |
| MONITOR-01 | Phase 42 (R1) | 42-04 |
| MONITOR-02~03 | Phase 43 (R2) | 43-03 |
| INTEGRATE-01~03 | Phase 45 (R4) | 45-01 / 45-02 |
| AUDIT-01~02 | 跨 phase | 各 plan 内 |
| INFRA-01~03 | Phase 42 (R1) | 42-05（基础 seed） |
| INFRA-04~05 | Phase 42 (R1) | 42-06（基础设施） |

### 关联决策记录

- [.planning/notes/asset-reconciliation-strategy.md](notes/asset-reconciliation-strategy.md) — v0.3 架构 + v0.4 复用审计
- [.planning/notes/260627-reconciliation-reuse-audit.md](notes/260627-reconciliation-reuse-audit.md) — 完整复用审计
- [.planning/notes/260627-cross-module-permission.md](notes/260627-cross-module-permission.md) — 跨模块权限边界
- [.planning/notes/260627-port-coverage-audit.md](notes/260627-port-coverage-audit.md) — 端口采集覆盖率审计模板
- [.planning/seeds/asset-reconciliation-v1.17.md](seeds/asset-reconciliation-v1.17.md) — 阶段种子
- [.planning/todos/pending/v1.17-reconciliation-decisions.md](todos/pending/v1.17-reconciliation-decisions.md) — T1-T30 待办