# Phase 43: 告警 + 工单闭环 (R2) - Context

**Gathered:** 2026-06-28
**Status:** Ready for planning

<domain>
## Phase Boundary

将 Phase 42 R1 的"观测底座"提升为"可行动闭环"——critical/high 异常自动转工单,WebSocket + SysNotice 实时推送,异常列表加"标记已解决" UI(兑现 Phase 42 D-18 推迟的 UI),并实现 7d 静默期 + 24h 节流防告警风暴。

**已锁定的高层决策**（沿用 v0.3 / Phase 42 / ROADMAP）:
- 策略:Observe-only + 告警驱动人工修复(D1)
- 告警分发范围:先 WebSocket + SysNotice,钉钉/邮件下个 phase(D13 锁定)
- API 前缀:`/asset/reconciliation/*`
- 跨模块调用走 service 层(无权限时降级隐藏)
- Owner 合并:运维 + 资产 + 权限 同一人(无双签)
- 6 类工单对应 Type A-F 冲突类型(A 不入主表,实际只针对 B-F 5 类)
- Phase 42 已有 6 个 Statistics 端点 + dashboard + 异常列表 admin 页 + 4 个 cron 任务
- workorder.BaseService.Create() 已存在(Phase 23)
- WebSocket 框架:gorilla/websocket v1.5.3,internal/websocket/notice_hub.go

**与 Phase 42 R1 边界差异**:
- R2 新增:2 个 cron(转单 critical / 转单 high)+ WebSocket 推送 + SysNotice 写入 + resolve API + MV 扩展(7d 静默期)
- R2 不实现:IP 段例外规则(R3)、置信度评分(R4)、半自动修复(R5)

</domain>

<decisions>
## Implementation Decisions

### 转单触发模型 (Area 1)

- **D-A1-01:** **critical 异常走独立 cron `reconciliation:createWorkorderCritical`,`@every 2m`**
  - 扫 `sys_data_reconciliation` WHERE `severity='critical' AND deleted_at IS NULL AND resolved_at IS NULL AND workorder_id IS NULL`
  - 命中即调 `workorder.BaseService.Create()`,符合 ROADMAP success criteria 1 "critical → 2min"

- **D-A1-02:** **high 异常走独立 cron `reconciliation:createWorkorderHigh`,`@every 5m`**
  - 同样 SELECT 但 `severity='high'`
  - 符合 ROADMAP success criteria 1 "high → 5min"

- **D-A1-03:** **转单失败仅 logrus.Errorf,下个 cron 周期重试,不写 SysNotice**
  - 与 Phase 42 D-02 风格一致
  - 避免重试期产生重复告警
  - workorder 服务挂掉 / 模板缺失 → 2-5min 自动重试

- **D-A1-04:** **转单成功后 UPDATE `workorder_id`,resolved_at 仍 NULL**
  - `sys_data_reconciliation.workorder_id` 已存在 schema(Phase 42 migration_168)
  - resolved_at NULL 表示"未解决",等运维标记已解决
  - 同一异常可被多个 workorder 关联(应对 R3 例外场景)

### 6 类工单模板字段 (Area 2)

- **D-A2-01:** **工单默认标题模板**:`[资产对账·{Type}] 资产 {asset_code} ({severity}) {detected_at}`
  - 例:`[资产对账·B类] 资产 4E2130013377 (high) 2026-06-27 21:19`
  - 标题偏长但运维一眼看出冲突类型/资产/时间,符合 ROADMAP success criteria 3 模板差异化要求

- **D-A2-02:** **默认分派按 Type 映射不同角色**:
  - B/C/D → 资产 owner role
  - E → 运维责任 owner role
  - F → 责任 owner role
  - 角色名由运维预建,migration 不创建(避免硬编码)
  - 类型→role 映射存 `sys_config:asset.reconciliation.workorder.assignee_role_map`(JSONB 字符串)

- **D-A2-03:** **SLA 按 severity 分级**:
  - critical=30min / high=4h / medium=24h / low=7d
  - 落 workorder 表 `sla_duration` 字段(Phase 23 workorder schema 已有)
  - 触发后由 workorder 内部 cron 监控超时

- **D-A2-04:** **工单 description 由 2 部分拼接**:
  - ① type seed 硬编码的 5 句中文建议(B/C/D/E/F 各 5 条)
  - ② raw_snapshot 原始三路 + signals JSON
  - 便于运维照着查又保留完整上下文

### 静默期 + 节流粒度 (Area 3)

- **D-A3-01:** **静默期固定 7d,ROADMAP success criteria 7 默认值**
  - 扩展 `reconciliation_normalized` MV 加 3 字段:
    - `last_resolved_at` (来自 `sys_data_reconciliation` 最近一条 `resolved_at IS NOT NULL` 记录)
    - `last_resolved_by` (最近 resolved_by user_id)
    - `last_conflict_type` (最近已解决的 conflict_type)
  - LEFT JOIN LATERAL `(SELECT resolved_at, resolved_by, conflict_type FROM sys_data_reconciliation WHERE asset_id = a.id AND resolved_at IS NOT NULL AND deleted_at IS NULL ORDER BY resolved_at DESC LIMIT 1)`
  - Layer 3 检测时:`NOW() - last_resolved_at < 7d AND last_conflict_type = current_type` → 跳过

- **D-A3-02:** **24h 节流维度仅 (asset_id, conflict_type)**
  - ROADMAP success criteria 8 默认值
  - Layer 3 检测前查 `sys_data_reconciliation WHERE detected_at > NOW() - INTERVAL '24h' AND asset_id=? AND conflict_type=? AND deleted_at IS NULL HAVING COUNT > 0` → 跳过

- **D-A3-03:** **拦截位置在 Layer 3 检测循环内 INSERT 前**
  - ClassifyType 输出后,先查两个 guard:
    - ① MV.last_resolved_at + 7d 静默(D-A3-01)
    - ② sys_data_reconciliation 24h 内同 (asset, type) 计数(D-A3-02)
  - 任一命中则 `skipped++ continue`,不报、不入表、不写 operlog
  - 计数计入 DetectLayer3 返回值(incremented skipped_silence / skipped_throttle 区分)

### WebSocket + 标记已解决 UI (Area 4)

- **D-A4-01:** **WebSocket 全量 broadcast**
  - 所有连接 `/asset/reconciliation/dashboard` 的 client 都收到 critical 异常事件
  - push 由 `reconciliation:createWorkorderCritical` cron 触发后调用 `internal/websocket/notice_hub.go` Broadcast
  - 不按角色/用户订阅(简化为所有人收到,dashboard 列表有则关注,无则忽略)

- **D-A4-02:** **WS 推送事件仅 2 类**:
  - `critical_exception_detected` (新 critical 异常出现)
  - `critical_workorder_created` (critical 已转工单)
  - high/medium/low 不推,避免 dashboard 事件过载(6min 一次)

- **D-A4-03:** **WS + SysNotice 双通道同时推**
  - WS 给在线 dashboard,SysNotice 给未在线的运维
  - 写 `sys_notice(notice_type='asset_reconciliation_critical', asset_id, conflict_type, workorder_id, link_url)`
  - 与 ROADMAP success criteria 4/5 一致

- **D-A4-04:** **标记已解决 UI 调用链**(兑现 Phase 42 D-18):
  - 点按钮 → 弹 Modal "是否已修复?" 填 `resolution_note` (可选) → 调 `POST /asset/reconciliation/exception/{id}/resolve`
  - 后端 SET `resolved_at=NOW(), resolved_by=current_user_id, resolution_note`
  - 写 operlog: `OperTypeUpdate` (状态变更,Phase 34 约定)
  - **不**联动 workorder 关闭(workorder 单独在 workorder UI 关闭,避免 2 边状态不一致)
  - **不**触发重检(7d 静默期已兜底,人工重检反而打破静默设计)
  - 权限粒度:`asset:reconciliation:resolve` (新增,只有该权限的 user 看得到按钮)

### Claude's Discretion

- 15 个 D-A1/D-A2/D-A3/D-A4 由用户与 Claude 共同决策,无 "you decide" 兜底
- 未讨论的 R2 细节由 plan-phase 决定:
  - critical/high cron 内部 SQL 的 ORDER BY / LIMIT 写法
  - sys_data_reconciliation.workorder_id 字段类型(string uuid,Phase 42 已建)
  - WS 事件的 JSON payload 结构
  - 异常列表 resolve Modal 的表单验证规则
  - 6 类工单模板的中文建议具体文案(B-F 各 5 条)
  - 4 个新增 cron 的 task handler 在 internal/scheduler/reconciliation_tasks.go 续写(Phase 42 已有文件)

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents (researcher / planner) MUST read these before planning or implementing.**

### Phase 42 R1 上下文(必读)
- `.planning/phases/42-r1/42-CONTEXT.md` — R1 全部 18 个决策(D-01~D-18)
- `.planning/phases/42-r1/42-VERIFICATION.md` — R1 验收报告
- `.planning/phases/42-r1/42-REVIEW.md` — R1 code review
- `.planning/phases/42-r1/42-HUMAN-UAT.md` — R1 人工 UAT
- `.planning/phases/42-r1/42-01-SUMMARY.md` ~ `42-06-SUMMARY.md` — 6 个 plan 实施总结

### 架构与策略
- `.planning/notes/asset-reconciliation-strategy.md` — v0.3 架构 + v0.4 复用审计 + v0.5 字段名调整
- `.planning/seeds/asset-reconciliation-v1.17.md` — v1.17 阶段种子 + R1-R5 启动门槛
- `.planning/todos/pending/v1.17-reconciliation-decisions.md` — D1-D18 决策点状态 + T1-T30 待办

### 复用与跨模块
- `.planning/notes/260627-reconciliation-reuse-audit.md` — F1-F7 必补项 + P1-P4 部分复用项
- `.planning/notes/260627-cross-module-permission.md` — ops/workstation ↔ asset/reconciliation 跨模块权限边界

### Roadmap 与 Requirements
- `.planning/ROADMAP.md` Phase 43 段 — 3 plans 拆分(43-01/02/03) + 9 条 success criteria
- `.planning/REQUIREMENTS.md` v1.17 — WORKORDER-01~02 / MONITOR-02~03 R2 范围

### 项目级 CLAUDE.md(强约束)
- `CLAUDE.md` "操作日志记录约定 (operlog convention) — 强制" — 11 关键词 + 25 OperType 常量
- `CLAUDE.md` "Status Value Convention" — 0=启用 1=停用
- `CLAUDE.md` "Cache Key Prefix Handling" — Redis `xingran:` 前缀处理
- `CLAUDE.md` "跨模块调用" — service 层调用 + 权限降级

### 项目记忆(规划时必查)
- `stat-cards-from-list-length-capped-at-100` — Statistics 必须用 COUNT 端点
- `xingran-server-side-sort-infra` — BaseListRequest + ApplySort 白名单
- `xingran-migrations-no-sql-autoloader` — migration_NNN_*.go 显式调用
- `xingran-gorm-sql-constraint-naming-conflict` — uniqueIndex 命名规范
- `migration-sql-name-must-match-model` — 字段名以实际 DB schema 为准
- `GORM AutoMigrate 被 PG 物化视图阻塞` — MV DROP+RECREATE 流程
- `workstation-ad-device-managedby-vs-description` — AD 反查路径
- `Excel 导入路由冲突陷阱` — 不预注册 /asset/reconciliation/*

### 现有代码参考(实施时查)
- `internal/services/workorder/base.go` — `workorder.BaseService.Create()` 签名(CreateRequest, submitterID) → *models.WorkOrder, error
- `internal/websocket/notice_hub.go` — WebSocket broadcast helper
- `internal/services/system/config_service.go` — `ConfigService.GetByKey()` 模式(D-A2-02 sys_config 读取)
- `internal/services/system/dict_cache_impl.go` — 字典缓存实现(D-A2-04 type seed 5 句建议存储)
- `internal/services/asset/reconciliation_service.go` — ListExceptions JOIN 模式
- `internal/services/asset/reconciliation_detection.go` — DetectLayer3 循环 (D-A3-03 拦截位置)
- `internal/scheduler/reconciliation_tasks.go` — Phase 42 R1 已建 4 个 cron handler,D-A1-01/02 续写 2 个
- `internal/core/db/migrations/migration_168_reconciliation_tables.go` — sys_data_reconciliation + MV 创建(D-A3-01 MV 扩展参考)
- `internal/core/db/migrations/migration_170_fix_asset_list_menu_path.go` — migration 模板参考(D-A3-01 新 migration 模式)

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `internal/services/workorder/base.go:Create(ctx, req, submitterID)` — R2 核心调用,接受 CreateRequest 结构体
- `internal/websocket/notice_hub.go:Broadcast(event, payload)` — WebSocket 推送 helper(D-A4-01)
- `internal/services/system/config_service.go:GetByKey(key)` — sys_config 读取(D-A2-02 type→role 映射)
- `internal/services/system/dict_cache_impl.go:useDict` — 字典 hook(D-A2-04 type 5 句建议)
- `internal/services/asset/reconciliation_detection.go:DetectLayer3` 循环 — D-A3-03 拦截位置(在 ClassifyType 后 INSERT 前)
- `internal/scheduler/reconciliation_tasks.go:RegisterReconciliationTasks` — 单 taskType "reconciliation" + params["param"] 分发(D-A1-01/02 续写 2 个 case)
- `internal/services/asset/reconciliation_service.go:ListExceptions` — 异常列表 SQL(给 R2 resolve API 参考)

### Established Patterns
- **Handler-Service Pattern**:interface + 私有 impl + 构造函数(CLAUDE.md 范例)
- **cron 走 sys_job 表 + 单 taskType 分发** — Phase 42 R1 D-10 模式,D-A1-01/02 续写
- **MV CONCURRENTLY 刷新** — Phase 42 R1 D-01 模式,D-A3-01 扩展字段
- **operlog.Record 强制约定** — CLAUDE.md 强约束(D-A1-04 / D-A4-04)
- **Cache Key Helper**:`internal/services/cache_keys.go` 常量 + 函数模式
- **MV 依赖降级**:`mvAvailable()` 探测 + fallback(Phase 42 R1)

### Integration Points
- `internal/api/router.go` — 注册 `SetupReconciliationRouter` (Phase 42 R1 已加,R2 加 resolve API)
- `internal/api/v1/asset/reconciliation_handler.go` — R2 加 `ResolveException(c)` handler
- `internal/core/db/migrations/migration_NNN_reconciliation_silence.go` — R2 新增,扩展 MV(D-A3-01)
- `internal/scheduler/reconciliation_tasks.go` — R2 续写 2 个 case(createWorkorderCritical / createWorkorderHigh)
- `xingran-react-frontend/src/pages/asset/reconciliation/exceptions/index.tsx` — R2 加"标记已解决"按钮(D-A4-04)
- `xingran-react-frontend/src/pages/asset/reconciliation/dashboard/index.tsx` — R2 加 WebSocket client + critical 实时刷新
- `src/utils/websocket.ts` (待建) — 封装 WebSocket 订阅 hook(D-A4-01/02)

</code_context>

<specifics>
## Specific Ideas

- **D-A2-01 标题示例**:`[资产对账·B类] 资产 4E2130013377 (high) 2026-06-27 21:19`
- **D-A2-04 描述结构**:
  ```
  【建议修复】
  1. 在 ops_asset 上补充责任人(检查 sys_user.deleted_at)
  2. 检查 sys_user.status 是否被禁用
  3. ...

  【原始数据】
  physical: {"user_id":"...","username":"..."}
  declared: {"user_id":"...","username":"..."}
  ad: {"ad_id":"...","ad_username":"...","is_enabled":true}
  signals: {HasPhysical:true, HasDeclared:false, ...}
  ```
- **D-A3-01 MV 扩展 SQL**:
  ```sql
  LEFT JOIN LATERAL (
    SELECT resolved_at, resolved_by, conflict_type
    FROM sys_data_reconciliation r
    WHERE r.asset_id = a.id
      AND r.resolved_at IS NOT NULL
      AND r.deleted_at IS NULL
    ORDER BY r.resolved_at DESC
    LIMIT 1
  ) last_resolved ON true
  ```
- **D-A4-01 WS 事件 payload**:
  ```json
  {
    "event": "critical_exception_detected",
    "data": {
      "exception_id": "uuid",
      "asset_code": "4E2130013377",
      "conflict_type": "B",
      "detected_at": "2026-06-27T21:19:00+08:00"
    }
  }
  ```
- **D-A4-04 resolve API 路径**:`POST /asset/reconciliation/exception/{id}/resolve`
- **D-A4-04 operlog 写入**:`operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "资产对账", operlog.OperTypeUpdate)`

</specifics>

<deferred>
## Deferred Ideas

下列决策**显式推后**到后续 R3-R5 阶段,R2 不实现:

### R3 (Phase 44) — IP 段例外规则
- 例外规则 CRUD admin 页 + 命中测试工具
- Excel 导入导出例外规则
- 多 actions 组合(no_alert / no_notice / no_workorder / skip_severity / silence)
- `sys_reconciliation_exception` 表已建,Phase 44 补 CRUD + UI
- R2 的 7d 静默期不依赖 R3(基于 last_resolved_at 单维度)

### R4 (Phase 45) — 工位详情整合
- HealthCard / HealthBadge / ReconciliationDrawer 组件
- 工位详情页 `/ops/workstation/:id` 顶部嵌入对账健康度
- 资产详情页 `/asset/card/:id` 摘要块
- HealthScore 函数(0-100 分)R4 实现,R2 不依赖
- N+1 优化、跨模块调用性能 ≤ 200ms

### R5 (Phase 46, 可选) — 半自动修复
- 高置信度修复建议(confidence ≥0.9)
- 人工确认 UI、一键回滚
- 误修复率 < 1% 监控
- R2 标记已解决仅修改 resolved_at,R5 才做"修复回写"逻辑

### R2 显式不做
- 钉钉/邮件告警(下个 phase 范围,D13 v0.3 锁定)
- 工单转交/重转流程(由 workorder 模块独立处理,R2 不耦合)
- 工单超时升级(R2 由 workorder 内部 cron 处理,R2 不主动接管)
- 移动端 push(待业务决策)
- 异常列表的"批量标记已解决"(单条设计已够 R2 验证)
- 双签流程(D17 v0.3 锁定不强制)

### Claude's Discretion 范围(R2 plan-phase 自决)
- critical/high cron 内部 SQL 的 ORDER BY / LIMIT 写法
- sys_data_reconciliation.workorder_id 字段类型(Phase 42 已建 string uuid)
- WS 事件的 JSON payload 字段细节
- 异常列表 resolve Modal 的表单验证规则(resolution_note 长度/必填)
- 6 类工单模板的中文建议具体文案(B-F 各 5 条)
- 4 个新增 cron 的 task handler 在 internal/scheduler/reconciliation_tasks.go 续写模式

</deferred>

---

*Phase: 43-告警 + 工单闭环 (R2)*
*Context gathered: 2026-06-28*
