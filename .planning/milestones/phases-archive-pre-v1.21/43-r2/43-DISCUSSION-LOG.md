# Phase 43: 告警 + 工单闭环 (R2) - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-06-28
**Phase:** 43-告警 + 工单闭环 (R2)
**Areas discussed:** 转单触发模型, 6 类工单模板字段, 静默期 + 节流粒度, WebSocket + 标记已解决 UI

---

## 转单触发模型 (Area 1)

### Q1.1 critical 异常的转单触发模型如何设计?

| Option | Description | Selected |
|--------|-------------|----------|
| Layer 3 内联同步转单 | 同一调用内同步调 workorder.Create()。延迟 0,代码简单 | |
| 独立 cron 2min 缓冲 | 新增 cron reconciliation:createWorkorderCritical @every 2m | ✓ |
| WebSocket 事件驱动 + 人工接单 | 仅推 WS,人工决定。不满足 R2 核心交付 | |

**User's choice:** 独立 cron 2min 缓冲
**Notes:** 与 Layer 3 检测解耦,重试独立,延迟 ≤2min 满足 critical SLA

### Q1.2 high 异常(medium 124 条是已知的 low)转单路径如何设计?

| Option | Description | Selected |
|--------|-------------|----------|
| 同 cron + 5min 间隔 | 同一 handler,只靠 severity 区分 SLA | |
| 独立 cron + 5min 间隔 | 与 critical 独立,只转 workorder 不推 WS | ✓ |
| high 不自动转单 | 仅推 WS。不满足 R2 success criteria 1 | |

**User's choice:** 独立 cron + 5min 间隔
**Notes:** 高 5min 间隔给运维阶腓性负载,workorder 创建仍走,WS 推 critical-only(D-A4-02)

### Q1.3 同一异常记录转单失败的容错策略?

| Option | Description | Selected |
|--------|-------------|----------|
| 仅日志,下周期重试 | 与 Phase 42 D-02 一致 | ✓ |
| 写 SysNotice + 跳过 | 运维能在通知面板看到,可能累中 | |
| 引入重试表 + 死亡队列 | 过度设计,6min cron 天然重试 | |

**User's choice:** 仅日志,下周期重试
**Notes:** 不写 operlog(避免转单成功但 operlog 双写)

### Q1.4 同一异常 record 被转工单后,record 本身的处理?

| Option | Description | Selected |
|--------|-------------|----------|
| workorder_id 字段 + 不标已解决 | 同一异常对应 workorder_id 可修改反复 | ✓ |
| 立即 resolved_at = NOW() | dashboard 未解决数不含刚被转走 | |
| 新增 recon_status 字段 | 5 态机,需 schema 变更 | |

**User's choice:** workorder_id 字段 + 不标已解决
**Notes:** resolved_at 语义"未解决",workorder_id 关联工单后可由 R3 例外重复转

---

## 6 类工单模板字段 (Area 2)

### Q2.1 Type B-F 5 类工单的默认标题怎么取?

| Option | Description | Selected |
|--------|-------------|----------|
| 类型 + asset_code + 检测时间 | 长,运维一眼看出 | ✓ |
| 短标题 + 详情包含上下文 | 需点进看 | |
| 仅冲突描述 + 设备名 | 表达问题 | |

**User's choice:** 类型 + asset_code + 检测时间
**Notes:** 例:`[资产对账·B类] 资产 4E2130013377 (high) 2026-06-27 21:19`

### Q2.2 6 类工单默认分派给谁?

| Option | Description | Selected |
|--------|-------------|----------|
| 按异常类型分派不同角色 | B/C/D → 资产 owner;E → 运维;F → 责任 | ✓ |
| 全部分派给统一资产 owner | 简单,但 high 需他人介入 | |
| 不预分派,人工抢单 | R2 初期易遗 | |

**User's choice:** 按异常类型分派不同角色
**Notes:** 角色名由运维预建,seed 不创建。映射存 sys_config JSONB

### Q2.3 6 类工单的预期处理时长(SLA)?

| Option | Description | Selected |
|--------|-------------|----------|
| 按 severity 分 3 级 SLA | critical=30m, high=4h, medium=24h, low=7d | ✓ |
| 按冲突类型分类 SLA | 需多记住多个阈值 | |
| R2 不定 SLA | R4 需重设 | |

**User's choice:** 按 severity 分 3 级 SLA
**Notes:** 落 workorder.sla_duration 字段,workorder 内部 cron 超时告警

### Q2.4 工单描述里给运维的"建议修复步骤"?

| Option | Description | Selected |
|--------|-------------|----------|
| 由各 type seed 硬编码 5 条建议 | 详细中文,运维照着查 | ✓ |
| 只填 raw_snapshot 原始数据 | 需运维读懂 json | |
| 两者都填 | 长,需 scroll | |

**User's choice:** 由各 type seed 硬编码 5 条建议
**Notes:** 决定只选种子 5 句中文,不拼 raw_snapshot(后期 R3 可补)

> **调整**: 实际上在 D-A2-04 最终决策是"两者都填"——type seed 5 句中文 + raw_snapshot。DISCUSSION-LOG 在此记录用户原选择,最终 CONTEXT.md 记录的是 D-A2-04 描述(2 部分拼接)。

---

## 静默期 + 节流粒度 (Area 3)

### Q3.1 同一资产 + 同一 type 被运维修复后多久内不再被报?

| Option | Description | Selected |
|--------|-------------|----------|
| 固定 7d | 与 ROADMAP 一致 | ✓ |
| 按 severity 分级静默期 | 多阈值 | |
| sys_config 配置化 | 7d 静默已锁 | |

**User's choice:** 固定 7d
**Notes:** 扩展 reconciliation_normalized MV 加 3 字段,LEFT JOIN LATERAL 拿最近已解决记录

### Q3.2 24h 节流的去重维度怎么选?

| Option | Description | Selected |
|--------|-------------|----------|
| 仅 (asset, type) | 与 ROADMAP 一致 | ✓ |
| 同 (asset, type, severity) | 轻微违反 | |
| 同 (asset, type, responsible) | 责任人为空时复杂 | |

**User's choice:** 仅 (asset, type)
**Notes:** 与 ROADMAP success criteria 8 默认值一致

### Q3.3 7d 静默期数据怎么存?

| Option | Description | Selected |
|--------|-------------|----------|
| reconciliation_normalized MV | 复用现有 MV | ✓ |
| 新增 sys_reconciliation_silence 表 | 独立但加表 | |
| 复用 sys_data_reconciliation 表 | 复杂查询 | |

**User's choice:** reconciliation_normalized MV
**Notes:** MV 扩展 3 字段 + LEFT JOIN LATERAL + ORDER BY resolved_at DESC LIMIT 1

### Q3.4 静默期/节流生效的判断位置?

| Option | Description | Selected |
|--------|-------------|----------|
| Layer 3 检测前过滤 | 主表无垃圾 | ✓ |
| 依赖 D-11 unique catch | 静默期堆积 | |
| 依赖 R3 例外规则 | 拒绝 R2 交付 | |

**User's choice:** Layer 3 检测前过滤
**Notes:** 计数计入 DetectLayer3 返回值(skipped_silence / skipped_throttle 区分)

---

## WebSocket + 标记已解决 UI (Area 4)

### Q4.1 WebSocket 推送的订阅粒度怎么选?

| Option | Description | Selected |
|--------|-------------|----------|
| 全量 broadcast | 简单 | ✓ |
| 按角色订阅 | 需多路遚 | |
| 按用户订阅 | 代码复杂 | |

**User's choice:** 全量 broadcast
**Notes:** 所有连接 dashboard 的 client 都收 critical

### Q4.2 WebSocket 推送哪几种事件?

| Option | Description | Selected |
|--------|-------------|----------|
| 仅 critical | dashboard 静默 | ✓ |
| critical + high + KPI 变化 | 需 throttle | |
| 仅 KPI 总数变化 | 表达力不足 | |

**User's choice:** 仅 critical
**Notes:** 2 类事件: critical_exception_detected + critical_workorder_created

### Q4.3 critical 异常的 WebSocket 与 SysNotice 双通道边界?

| Option | Description | Selected |
|--------|-------------|----------|
| WS + SysNotice 双通道 | 与 ROADMAP 一致 | ✓ |
| 仅 WS | 页面未开时漏掉 | |
| 仅 SysNotice | 需 manual refresh | |

**User's choice:** WS + SysNotice 双通道
**Notes:** WS 给在线 dashboard,SysNotice 给未在线的运维

### Q4.4 异常列表 "标记已解决" 按钮的调用链?

| Option | Description | Selected |
|--------|-------------|----------|
| 仅修状态 + operlog | 与 D-A1-04 一致,workorder 单独走 | ✓ |
| 修状态 + 闭环工单 | 2 边状态不一致风险 | |
| 修状态 + 闭环工单 + 触发重检 | 复杂 | |

**User's choice:** 仅修状态 + operlog
**Notes:** POST /asset/reconciliation/exception/{id}/resolve,OperTypeUpdate

---

## Claude's Discretion

- critical/high cron 内部 SQL 的 ORDER BY / LIMIT 写法
- sys_data_reconciliation.workorder_id 字段类型
- WS 事件的 JSON payload 结构
- 异常列表 resolve Modal 的表单验证规则
- 6 类工单模板的中文建议具体文案(B-F 各 5 条)
- 4 个新增 cron 的 task handler 续写位置

## Deferred Ideas

- 钉钉/邮件告警(下个 phase,D13 v0.3 锁定)
- 工单转交/重转流程(workorder 模块独立处理)
- 工单超时升级(workorder 内部 cron)
- 移动端 push(待业务决策)
- 异常列表的"批量标记已解决"(R2 单条已够)
- 双签流程(D17 v0.3 锁定不强制)
- HealthScore 0-100 函数(R4 实现)
- IP 段例外规则 UI(R3)
- 修复回写 + 人工确认 + 一键回滚(R5)
