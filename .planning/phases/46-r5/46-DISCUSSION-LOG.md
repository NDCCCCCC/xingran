# Phase 46: 半自动修复（可选） (R5) - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-07-03
**Phase:** 46-半自动修复（可选） (R5)
**Areas discussed:** A. 修复字段范围与置信度门槛; B. 建议-确认-应用-回滚数据模型; C. 回滚机制 + 误修复监控 + 缓存/静默期联动; D. 人工确认 UI 形态 + 建议展示密度

---

## A. 修复字段范围与置信度门槛

| Option | Description | Selected |
|--------|-------------|----------|
| 仅修复 user_id (推荐) | 物理链路已锁定责任人 user_id，dept_id 由 user_id JOIN 推导，与 R1 RECON-02 物理链路语义对齐，风险最小 | ✓ |
| 同步修复 user_id + dept_id | 同时写 ops_asset.user_id 和 dept_id；需 asset model 新增 DeptID 严格取值逻辑，风险中 | |
| 修复 user_id + NowUserName + DeptName | 同时写 3 个字段联动；兼容性最高但误修复率高 | |

**User's choice:** 仅修复 user_id

| Option | Description | Selected |
|--------|-------------|----------|
| 不动 Type D (推荐) | Type D 修复需补充 machine_uptime/machine_ip 反验逻辑复杂、误修复高；R5 仅修 Type B，Type D 不进入修复范围 | ✓ |
| 仅在 confidence ≥0.95 时修 Type D | 高 confidence + 全满分时补充 machine_uptime = NOW()；需追加物理校验逻辑 | |
| Type D 转手动修复 | 在确认页只展示、不提供"一键接受"按钮，需手动"标记已解决"流程 | |

**User's choice:** 不动 Type D

| Option | Description | Selected |
|--------|-------------|----------|
| 固定 ≥0.9 (与 ROADMAP 对齐) | ROADMAP SC1 + strategy §14 明确'confidence ≥0.9' | |
| 固定 ≥0.95 | 提高门槛，误修复概率更低但实际可生成建议更少 | |
| 0.9 默认 + sys_config 可配 (推荐) | 默认 0.9 但加 sys_config 动态可调；增加 INFRA-02 配置项 + 热加载逻辑 | ✓ |

**User's choice:** 0.9 默认 + sys_config 可配

| Option | Description | Selected |
|--------|-------------|----------|
| 限定 Type B 且未关工单 (推荐) | 仅当 (conflict_type='B' AND workorder_id IS NULL AND ...) 才生成建议；R2 已转单 B 类排除 | ✓ |
| 不限工单状态 | workorder 是否存在不影响建议生成；需求更多 UI 判断逻辑 | |
| 额外限定 critical/high severity | 仅对 severity=critical/high 生成建议 | |

**User's choice:** 限定 Type B 且未关工单

---

## B. 建议-确认-应用-回滚数据模型

| Option | Description | Selected |
|--------|-------------|----------|
| 独立 sys_reconciliation_fix_suggestion 表 (推荐) | 状态机独立、可追踪多轮建议、审计链完整、与 R1-R4 架构解耦 | ✓ |
| 扩展 sys_data_reconciliation 加 suggestion 字段 | 微改动但 reconciliation 表变重、复查难度上升 | |
| 复用 workorder 模块 + workorder_suggestion 子表 | 不适用于 R5 独立启动场景（workorder_id 为空才进 R5） | |

**User's choice:** 独立 sys_reconciliation_fix_suggestion 表

| Option | Description | Selected |
|--------|-------------|----------|
| fix_status: 6 状态 (推荐) | pending/accepted/rejected/applied/rolled_back/failed；生命周期状态全；与 ROADMAP SC6 一致 | ✓ |
| status + sub_status 两字段 | 状态间不能严格互斥，查询复杂度上升 | |
| 仅 4 状态 | 不区分 applied 状态，误修复审计能力下降 | |

**User's choice:** fix_status: 6 状态

| Option | Description | Selected |
|--------|-------------|----------|
| 1 对多版本化 (推荐) | 同一异常可生成多轮建议；旧记录加 superseded_at 字段标记 | ✓ |
| 1 对 1 最新建议 | 表小但状态转换复杂、中间态丢失 | |
| 同一异常只生成一次建议 | 重启可能性差，不符合 R5 '多次送建议直到接受' 设计意图 | |

**User's choice:** 1 对多版本化

| Option | Description | Selected |
|--------|-------------|----------|
| 乐观锁 + 部分唯一索引 (推荐) | API 层事务 + WHERE fix_status='pending' 条件 UPDATE + 部分唯一索引 (exception_id) WHERE fix_status='pending'；与 R1 uniq_recon_asset_type_open 同模式 | ✓ |
| 悲观锁 | SELECT FOR UPDATE 锁行，可靠但限并发性能 | |
| 无锁幂等 | 不满足 audit-01 '谁接受' 要求 | |

**User's choice:** 乐观锁 + 部分唯一索引

---

## C. 回滚机制 + 误修复监控 + 缓存/静默期联动

| Option | Description | Selected |
|--------|-------------|----------|
| 仅恢复 user_id (推荐) | 因 D-A1 锁定仅修复 user_id，回滚仅恢复 user_id 为 pre_fix_user_id | ✓ |
| 恢复 raw_snapshot 全字段 | 复杂度高、不适用于只修改 user_id 的 R5 | |
| 手动选择回滚字段 | 与 R5 '一键回滚' 语义偏离 | |

**User's choice:** 仅恢复 user_id

| Option | Description | Selected |
|--------|-------------|----------|
| 固定 7 天 (推荐) | 与 R2 7d 静默期语义一致；suggestion 表加 rollback_window_until = applied_at + 7d | ✓ |
| 固定 30 天 | 反应时间更充足，但 raw_snapshot 保留期变长、误修复修正踩中概率上升 | |
| 永久可回滚 | 满足 audit-02 最强但表增长无上限 | |

**User's choice:** 固定 7 天

| Option | Description | Selected |
|--------|-------------|----------|
| 回滚强写 operlog (推荐) | rollback 动作强写 operlog（OperTypeReset=11 已存于 25 常量集）；与 CLAUDE.md '操作日志记录约定 强制' 一致 | ✓ |
| 仅写 sys_reconciliation_fix_suggestion 不写 operlog | 审计检索能力下降 | |
| operlog + sys_oper_log 联动 | 增加联表查询与页面映射复杂度 | |

**User's choice:** 回滚强写 operlog

| Option | Description | Selected |
|--------|-------------|----------|
| 启用 7d 静默期 + MV 缓存失效 (推荐) | applied 后同 (asset, type) 自动进入 R2 7d 静默期（复用 MV 扩展机制）、手动 invalidate_workstation_health 缓存；下个 cron 周期自然检 | ✓ |
| 只联缓存不联静默期 | 下一轮 DetectLayer3 可能重新检出为 Type B | |
| 修复后立即触发 DetectLayer3 | 与 R2 D-A4-04 '不联动 resolve' 语义不符 | |

**User's choice:** 启用 7d 静默期 + MV 缓存失效

| Option | Description | Selected |
|--------|-------------|----------|
| 回滚/应用 比率 + 滑动窗口 (推荐) | 新增端点 GET /asset/reconciliation/fix-suggestion/stats 返回滑动窗口统计；误修复率 = rolled_back/applied；超过 sys_config 阈值产出 SysNotice 告警 | ✓ (用户自定义为 7d 滑动窗口) |
| 仅记录不计告警 | 仅 sys_reconciliation_fix_suggestion 记录、不计算比率、不告警 | |
| 不计率只告警 | 只记录，critical 误修复即时告警 | |

**User's choice:** 回滚/应用 比率 + 7d 滑动窗口

---

## D. 人工确认 UI 形态 + 建议展示密度

| Option | Description | Selected |
|--------|-------------|----------|
| 独立 /fix-suggestion 页面 (推荐) | 新建独立页面显示全 6 状态；与 ROADMAP SC2 '一键接受/拒绝/修改建议' 独立交互面需求匹配 | ✓ |
| 在 ReconciliationDrawer 加 修复建议 Tab | Drawer 模式为资产详情开不同 Tab，但页面装接受/拒绝按钮易冲突 | |
| 独立页面 + R4 抽屉快捷入口 | 复杂度高、不必要的二次跳转 | |

**User's choice:** 独立 /fix-suggestion 页面

| Option | Description | Selected |
|--------|-------------|----------|
| 紧凑行 + 点击展开详情 (推荐) | 列: asset_code / 现 ops_asset.user_id / 建议 user_id / confidence_score / conflict_type / created_at / fix_status；点击行 → Drawer 展开 raw_snapshot 三路信息与冲突原因 | ✓ |
| 默认全字段表 | 默认同时展示 raw_snapshot 三路数据；页面变重、带宽高 | |
| 卡片列表 (反设计) | 默认仅 asset_code + user_id + 接受/拒绝按钮；看不到任何 context | |

**User's choice:** 紧凑行 + 点击展开详情

| Option | Description | Selected |
|--------|-------------|----------|
| 仅单条接受 (推荐) | 仅提供单条接受 / 拒绝 / 回滚；误修复率低、个体决策；与 R5 '需人工确认' 语义一致 | ✓ |
| 支持批量接受 | 运维高效，但误修复概率上升 | |
| 仅批量接受同 user_id | 偏运维手动调节，与 R5 独立意图不符 | |

**User's choice:** 仅单条接受

| Option | Description | Selected |
|--------|-------------|----------|
| 默认按 confidence desc + 部门/状态筛选 (推荐) | 列表默认 confidence_score DESC + created_at DESC；可按部门/状态筛选；复用 Phase 13 BaseListRequest + ApplySort 白名单 | ✓ |
| 默认按 created_at desc | 与 R1 异常列表一致 | |
| 仅状态筛选 | 不提供排序/筛选 | |

**User's choice:** 默认按 confidence desc + 部门/状态筛选

---

## Claude's Discretion

下列实现细节由 planner/researcher 在 plan-phase 自决（已在 CONTEXT.md "Claude's Discretion" 部分详述）：
- 修复建议生成的触发时机（DetectLayer3 同步生成 vs 列表查询时 lazy 生成 vs 独立 cron）
- `sys_reconciliation_fix_suggestion` 表的精确字段列表
- 拒绝建议时是否必填 `rejection_reason`
- 修改建议功能的 UI 形态
- sys_reconciliation_fix_suggestion migration 命名/编号
- raw_snapshot 是否需补字段记录"修复前完整 ops_asset 状态"
- 修复建议是否在 R4 ReconciliationDrawer 加跳转链接
- INFRA-02 中各 sys_config 项的默认值 + config seed 命名

## Deferred Ideas

下列决策**显式推后**到后续 phase，R5 不实现：

### 显式不做（scope creep）
- Type D / Type C / Type E / Type F 修复
- 修复 dept_id / NowUserName / DeptName / machine_uptime / machine_ip
- 批量接受
- 自动触发 DetectLayer3 同步重检
- 联动 workorder 自动关闭
- DB TRIGGER 路线
- AD managed_by 作为修复建议源
- "全字段修复"自动覆盖

### 后续 phase 候选
- v1.18 R5+: 其他类型的修复
- v1.18 R5+: 修改建议功能 UI
- v1.18 R5+: 修复建议的 Excel 导出
- v1.18 R5+: 自动接受（critical + 部门白名单）

### Reviewed Todos (not folded)
- `v1.17-reconciliation-decisions.md` (T1-T30) — R5 相关项已被本 CONTEXT 锁定
- `operlog-exclude-paths.md` — Phase 35 范围，与 R5 无关