# Phase 58: 排水器与 compare-and-move 推送 - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-10-17
**Phase:** 58-排水器与 compare-and-move 推送
**Areas discussed:** 导入直推路径处置 / ad_config_id 取舍 / F-07~F-16 复核范围 / 排水节奏与退避参数
**Mode:** advisor（4 区全选，每区 1 个 gsd-advisor-researcher 并行研究，table-first 讨论）

---

## 导入直推路径处置（移交待决项 1 剩余半边）

| Option | Description | Selected |
|--------|-------------|----------|
| A: 删直推+批量入队 | triggerADSyncAfterImport 换成批量入队 ad_push_queue（新 EnqueueUpdate 变体，保留 ad_dn 预过滤）。与 D-03 精神一致的唯一终态，SC6 严格达标 | ✓ |
| B: 纯删除不补 | 只删 goroutine 块，不接入队。导入更新存量 AD 用户的四字段变更不再推 AD（靠全量同步兜底，有回写窗口） | |
| C: 保留+互斥 | 保留 BatchSyncUsersToAD，加进程内互斥防并发。乱序风险仅部分缓解，双路径永续 | |

**User's choice:** A — 删直推+批量入队（研究推荐项）
**Notes:** 研究关键事实：BatchSyncUsersToAD 只做属性 Modify、明确无 OU 移动（:679-683 注释）——是 compare-and-move 排水的严格子集。B 的致命伤：5 分钟全量同步窗口内 AD 登录回写冲回导入变更（F1 乒乓土壤，Phase 59 守卫前无防护）。C 互斥防并发不防跨路径乱序。A 的已知代价（入队在 ExcelService 事务外 crash 窗口丢推送、队列洪峰分钟级消化）被用户接受。

---

## ad_config_id 取舍（移交待决项 2）

| Option | Description | Selected |
|--------|-------------|----------|
| 排水时解析+契约 | 零迁移。排水器按 ad_dn 最长后缀唯一匹配 config；多匹配/无匹配 → dead（不推）。契约写进设计文档+单测锁定。SC3 假设即此 | ✓ |
| 加列 migration_216 | migration_216 双方言加列 + 回填 + 五条入队路径返工。多域已排期才值得；且入队时刻仍靠 DN 推断填列 | |
| 用户级归属（超范围） | sys_user 加归属列，登录/同步时持久化。根因修复但触碰登录热路径，超出 58 范围 → 记 deferred | |

**User's choice:** 排水时解析+契约（研究推荐项，即 ROADMAP SC3 假设方案）
**Notes:** 决定性事实：sys_user 无权威归属源，加列只是把 DN 推断提前冻结且跨域移动后过期。研究顺带定位 HYGIENE-01 双份实现：ad_sync_tasks.go:306 + dept_sync_tasks.go:47（逐字拷贝），ou_mapping_handler.go:117 为第三处同类歧义。契约反例（父子域共享后缀）须单测锁定。

---

## F-07~F-16 复核范围（移交待决项 3）

| Option | Description | Selected |
|--------|-------------|----------|
| B: 契约+顺带清扫 | F-11+F-07+F-10 进 58 正式范围；F-12/14/15/16 顺带一行修；F-13 归档、F-16 PG 升级留档。新增正式范围最小 | ✓ |
| A: 契约最小 | 仅 F-11 + F-07 排水器侧容错。F-10 回滚不变量继续无守护，队列留孤儿行 | |
| C: 全量升级 | 含 F-16 升级 PG 契约断言（需新建 PG 测试基座，基础设施级投入，超 58 授权） | |

**User's choice:** B — 契约修复+顺带清扫（研究推荐项）
**Notes:** 研究逐条核对当前代码后确认实际待决集合比 review 报告小：F-01/02/05/06/13 已在 57 收口修掉。F-11 为必答契约缺口（合并子句编辑清零 attempts，裸 attempts 判 dead 上线即缺陷）；F-07 双层修（排水器侧 not-found→dead 必选 + 入队侧 BatchDelete Pluck 顺带加固）；F-10 补 2 个回滚用例。

---

## 排水节奏与退避参数

| Option | Description | Selected |
|--------|-------------|----------|
| 30s + sys_config | 改完→AD 生效 ≤1min；sys_config 单旋钮 ad_push_queue.drain_interval_seconds 可调，域控敏感时可自行放宽。复制 network.connection_pool.* 先例 | ✓ |
| 5min 硬编码 | 与 ADSyncScheduler 每 5min 先例一致，硬编码零旋钮。感知延迟最坏 ~5min | |
| 1min 硬编码 | 折中：感知 ≤2min，硬编码不进 sys_config | |

**User's choice:** 30s + sys_config 单旋钮（研究推荐项）
**Notes:** 唯一运维感知参数。其余三项随选项生效为硬编码实现不变量：批量 100 行、退避 60s→2min→4min→8min→16min 封顶 30min + 5 次转 dead（总窗 ~61min 刻意桥接账号池 30min 熔断周期）、NULL next_retry_at = 立即可拾取。cron.SkipIfStillRunning 防轮次重叠。

---

## Claude's Discretion

- 批量 EnqueueUpdate 变体签名/命名、ExcelService 挂接点与切批
- F-11 dead 判定具体方向（独立标记 vs updated_at 超窗 vs 显式接受续命）——禁令进验收即可
- 排水器代码位置（addomain vs scheduler 包，按依赖方向定）
- 审计输入清单技术项：Unscoped() 读、pushed_at/failed_at 一致性、update_user/disable 处理顺序
- HYGIENE-02 OU DN 转义 helper 实现细节、NormalizeDN 单一真实源位置（addomain 包）
- D-03 ou_mapping_handler.go:117 收敛或注释保留

## Deferred Ideas

- sys_user.ad_config_id 用户级归属（登录/同步时持久化）——多域真实排期时独立立项
- F-16 升级 PG 契约断言 + PG 测试基座——v1.22 候选
- F-07 入队侧 Pluck 降档留档（若 B 档执行超载）
- UpdateStatus 双向推送（57 遗留）
- auth.go AD 登录路径不查 user.Status（57 遗留）
