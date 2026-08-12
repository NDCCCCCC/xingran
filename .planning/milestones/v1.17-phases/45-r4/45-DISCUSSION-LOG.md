# Phase 45: 工位详情整合 + 资产详情摘要 (R4) - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-06-28
**Phase:** 45-工位详情整合 + 资产详情摘要 (R4)
**Areas discussed:** 整合落点（工位/资产/权限降级）、健康度得分口径 + 时间窗口、抽屉历史变更 Tab 数据源、后端聚合 API 形态 + 实时性

---

## 整合落点（工位/资产/权限降级）

### Q1: 工位健康度整合的落点

| Option | Description | Selected |
|--------|-------------|----------|
| 现有展开区顶部+HealthCard (推荐) | CardView/FloorPlanView 列表已有 expand 渲染 WorkstationDeviceTable, 在 expand 顶部塞 HealthCard | ✓ |
| 新建 /ops/workstation/:id 详情页路由 | 按 ROADMAP SC1 字面新建独立详情页路由+布局+菜单 | |
| 点卡片弹 ReconciliationDrawer | CardView 点工作站卡片直接弹 ReconciliationDrawer | |
| Claude 决定 | 信任 Claude 根据 strategy/ROADMAP/project memory 的最佳判断 | |

**User's choice:** 现有展开区顶部+HealthCard (推荐)
**Notes:** 与 strategy §6.1 "工位→子表" 一致；WorkstationDeviceTable 已是 expand 子组件，HealthCard 自然落在它之前。SC1 "详情页" 语义由 expand 区承担，不新建路由。

### Q2: SC4 资产详情摘要的落点

| Option | Description | Selected |
|--------|-------------|----------|
| 列表行内抽屉(推荐) | ops/assets/index.tsx 列表行内加 HealthBadge, 点击弹 ReconciliationDrawer | ✓ |
| 新建 /asset/card/:id 详情页 | 按 ROADMAP SC4 字面新建独立详情页 | |
| 列表顶部摘要 + 抽屉 | 列表顶部加 "对账状态总览" 卡片 + 行内抽屉 | |
| Claude 决定 | Claude 最佳判断 | |

**User's choice:** 列表行内抽屉(推荐, 复用 ReconciliationDrawer)
**Notes:** 与工位侧对称（都用同一抽屉组件 + 三 Tab 同契约）；SC4 "顶部摘要" 语义由 drawer 顶部承担；避免与工位顶部 HealthCard 信息冗余。

### Q3: 权限降级范围

| Option | Description | Selected |
|--------|-------------|----------|
| 双侧都降级(推荐) | ops/workstation + asset 双侧用 ReconciliationVisible 静默隐藏 | ✓ |
| 只工位侧降级 | 仅工位侧交叉场景降级，资产侧不额外隐藏 | |
| 全部 403 不降级 | 不降级, 无权限 403 | |
| Claude 决定 | Claude 锁定 | |

**User's choice:** 双侧都降级(推荐, 与 strategy 一致)
**Notes:** 与 strategy §7.4 + cross-module-permission.md §2.3 锁定决策一致；避免读写路径碎片化（参 `xingran-perm-namespace-split-readonly-page` project memory）。

---

## 健康度得分口径 + 时间窗口

### Q4: HealthScore 得分范围（资产集合）

| Option | Description | Selected |
|--------|-------------|----------|
| 工位下所有资产 (推荐) | 走 reconciliation_normalized 物化视图 workstation_id 列 | ✓ |
| 工位负责人UserID 关联资产 | 仅 w.user_id 关联的 ops_asset | |
| 合并二者 | 合集去重 | |
| Claude 决定 | Claude 锁定 | |

**User's choice:** 工位下所有资产 (推荐, 与 strategy §6.5 一致)

### Q5: HealthCard 时间窗口

| Option | Description | Selected |
|--------|-------------|----------|
| 固定本周 (推荐) | 默认最近 7 天检出异常 | ✓ |
| 可切换(今日/本周/本月) | 顶部加 Select 切换 | |
| 不取时间段, 用当前态 | 取 resolved_at IS NULL 作计算源 | |
| Claude 决定 | Claude 锁定 | |

**User's choice:** 固定本周 (推荐, 与 mockup 一致)
**Notes:** 与 strategy §6.2 mockup "[本周]" 字面一致；与 R2 24h 节流 + R3 7d 静默期语义对齐；趋势 mini chart 单独取历史趋势数据补足长期视角。

### Q6: HealthScore 得分公式

| Option | Description | Selected |
|--------|-------------|----------|
| 简单比(推荐) | score = (1 - 异常资产数/总资产数) × 100 | ✓ |
| 权重加权（用 seed config） | score = Σ(asset_i.weight) / Σ(总资产数 × 1.0) × 100 | |
| 资产总数 vs 异常状态分类 | score = (normal×1.0 + drift×0.5 + nodata×0.7) / 总资产数 × 100 | |
| Claude 决定 | Claude 锁定 | |

**User's choice:** 简单比(推荐, 与 SC/ROADMAP 一致)
**Notes:** 与 ROADMAP SC1 mockup "得分: 78/100" 字面一致；权重公式 conflict=0 会让单个冲突资产把 score 拉到 0 产生假报警，违反得分直观性。

### Q7: 行内徽标粒度

| Option | Description | Selected |
|--------|-------------|----------|
| 六类色点(推荐) | 6 种状态点, 复用 conflict_type 字典 list_class 颜色 | ✓ |
| 仅健康/异常二态 | 仅两态(健康/异常), 异常才出徽标 | |
| 冲突类型文本 + 色点 | 冲突类型文本(B/C/D/E/F) + 色点 | |
| Claude 决定 | Claude 锁定 | |

**User's choice:** 六类色点(推荐)
**Notes:** 与 useDict("asset_reconciliation_conflict_type") 字典颜色映射复用；列宽友好（WorkstationDeviceTable 当前列宽 ~90 像素）。

---

## 抽屉历史变更 Tab 数据源

### Q8: ReconciliationTimeline '历史变更' 数据源

| Option | Description | Selected |
|--------|-------------|----------|
| resolved 记录(推荐) | 该资产所有 resolved_at IS NOT NULL 记录 | ✓ |
| sys_oper_log 操作日志 | 查 oper_type=Approve/Update 且 module_name LIKE '资产对账%' | |
| resolved + detected_at 变更 | 同一资产 all records 按 detected_at 倒序 | |
| Claude 决定 | Claude 锁定 | |

**User's choice:** resolved 记录(推荐)
**Notes:** 表内字段齐全（resolved_at + resolution_note + raw_snapshot）；与唯一索引防风暴语义一致；满足 AUDIT-02 审计要求。

### Q9: raw_snapshot 充不充实

| Option | Description | Selected |
|--------|-------------|----------|
| 纯文本 timeline (推荐) | 仅显示冲突类型 + 检出/解决时间 + resolution_note | ✓ |
| timeline + 可展开 raw_snapshot | 点击展开项可以看到 raw_snapshot JSONB | |
| 仅看例外命中 | Timeline 仅显示例外命中记录 | |
| Claude 决定 | Claude 锁定 | |

**User's choice:** 纯文本 timeline (推荐)
**Notes:** 抽屉空间受限；raw_snapshot 留作后台溯源入口（admin 异常详情页可看，抽屉不重复）。

### Q10: 抽屉 '例外规则' Tab 展示

| Option | Description | Selected |
|--------|-------------|----------|
| 命中该资产 IP 的例外 (推荐) | 当前生效中且 IP CIDR 命中该资产 IP 的 sys_reconciliation_exception 列表 | ✓ |
| 命中 + 已停用历史 | 该资产的所有例外命中(含停用的历史记录) | |
| 全局例外快照 | 仅显示 sys_config.exception.default_expiry_days + R3 表 全局生效例外数 | |
| Claude 决定 | Claude 锁定 | |

**User's choice:** 命中该资产 IP 的例外 (推荐)
**Notes:** 与 R3 命中测试工具一致；表达 "为什么这个资产被静默"；已停用历史由 R3 admin 例外规则管理页承担。

---

## 后端聚合 API 形态 + 实时性

### Q11: 后端聚合 API 路由形态

| Option | Description | Selected |
|--------|-------------|----------|
| POST + body(推荐) | POST /asset/reconciliation/by-workstation body: {workstationId, window} | ✓ |
| GET /:ws_id (strict RESTful) | GET /asset/reconciliation/by-workstation/:ws_id | |
| POST /list 拓展 | 拓展现有 POST /exceptions/list, 加 mode: "by_workstation" | |
| Claude 决定 | Claude 锁定 | |

**User's choice:** POST + body(推荐, 与项目惯例一致)
**Notes:** 与 CLAUDE.md "POST /list 模式" + opsApi.ts factory 模式一致；body 易拓展；strategy §6.5 原始 GET 写法不采用。

### Q12: API 响应结构

| Option | Description | Selected |
|--------|-------------|----------|
| health + list (推荐) | {workstation, healthScore, assets[], visible} | ✓ |
| 仅 health + 资产 ID | {workstation, healthScore, assetIds[]} | |
| 仅 health, 资产子表另调 | 仅返 healthScore, 资产另调 list exceptions | |
| Claude 决定 | Claude 锁定 | |

**User's choice:** health + list (推荐)
**Notes:** 一次拿完顶部卡片 + 资产子表徽标 + 详情跳转锚点；与 SC1/SC2 一次拿完；避免 N+1（ROADMAP SC7）。

### Q13: 实时性策略

| Option | Description | Selected |
|--------|-------------|----------|
| 打开拉一次 + 缓存(推荐) | 调一次 API + CacheProvider 缓存, TTL 5min | ✓ |
| 与 R2 WS 复用 | 工位页订阅 R2 notice_hub WS 接收推送 | |
| 另建工位 WS 通道 | 新建专属 WS 频道 reconciliation:workstation:<ws_id> | |
| Claude 决定 | Claude 锁定 | |

**User's choice:** 打开拉一次 + 缓存(推荐)
**Notes:** 工位详情页是低频访问页面，5 分钟 TTL 足够；SC5 ≤200ms 用 service 层 + 缓存达成；不重复造 WS 通道。

### Q14: 缓存主动失效

| Option | Description | Selected |
|--------|-------------|----------|
| R2 转单/resolve 后 invalidate (推荐) | R2 createWorkorder* + resolve API 完成后调 invalidate_workstation_health(wsID) | ✓ |
| 仅靠 TTL | 仅靠 5min TTL 自然过期 | |
| 不需要 invalidate | 不同步缓存 | |
| Claude 决定 | Claude 锁定 | |

**User's choice:** R2 转单/resolve 后 invalidate (推荐)
**Notes:** 业务闭环——修复后用户重看页面立即看到变化；R3 7d 静默期生效期间不重 invalidate（避免资源重撞）。调用顺序：**invalidate → operlog.Record → response.Success**。

---

## Claude's Discretion

无（用户在 14 个问题中均未选择 "Claude 决定"，所有决策显式锁定）。

## Deferred Ideas

- **R5（Phase 46，可选）** — 半自动修复（高置信度建议修复 + 人工确认 UI + 一键回滚 + 误修复监控）。R4 健康度 UI 是 R5 修复建议的基础。
- **R4 显式不做**：钉钉/邮件告警通道、WS 推送增量到工位详情页、抽屉内展开 raw_snapshot、工位/资产详情页路由新建、健康度得分公式用权重加权、例外规则批量启用/停用、版本历史/审计回溯、工位设备子表加新列以外的功能（如设备编辑、对账修复按钮直改）。

## Reviewed Todos (not folded)

`cross_reference_todos` 命中 2 项（与 Phase 44 审阅结论一致）：

- `v1.17-reconciliation-decisions.md` — R4 相关项（T27 HealthScore + T4 跨模块权限文档）已被 CONTEXT D-A2-03 + D-A1-03 锁定。
- `operlog-exclude-paths.md` — Phase 35 范围，与 R4 无关。