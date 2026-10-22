# Phase 58: 排水器与 compare-and-move 推送 - Context

**Gathered:** 2026-10-17
**Status:** Ready for planning

<domain>
## Phase Boundary

`ad_push_queue` 排水器落地：in-process cron 拾取（禁 sys_job）→ 按用户 `ad_dn` 路由 AD 配置（消灭 `First()` 歧义）→ 闭包内重读最新值 compare-and-move 幂等移动 OU → 失败指数退避重试至终态可查 → LDAP code 32 / 配置无匹配转 dead 终态；HYGIENE-01 双份配置 helper 合并 + HYGIENE-02 OU DN 转义 helper；两条遗留直推路径处置（单用户编辑直推删 + 导入批量直推换批量入队）；F-07~F-16 按分档复核。

**不含：** 登录守卫（Phase 59 GUARD-01/02）、对账 VIEW（Phase 59）、`UpdateStatus` 双向推送（57 Deferred）、`sys_user.ad_config_id` 用户级归属（本 phase Deferred）、PG 测试基座（v1.22 候选）。
</domain>

<decisions>
## Implementation Decisions

### 导入直推路径处置（区 1，用户选定删直推+批量入队）
- **D-01: `triggerADSyncAfterImport`（`user_import_handler.go:114-148`）删除直推 goroutine，替换为批量入队 `ad_push_queue`**。新增无 diff 的批量 `EnqueueUpdate` 变体（复用 `EnqueueDisable` 批量模式：`ad_dn` 非空预过滤 + `IsADSyncEnabled` 前置门控 + 批量上限）。与 D-03（单用户直推删）合并达成 ROADMAP SC6：**两条直推路径均删除，无互斥共存终态**。
  - 依据：`BatchSyncUsersToAD` 只做属性 Modify、明确无 OU 移动（`user_ad_sync_service.go:679-683` 注释承认无旧值快照做不了）——它是 compare-and-move 排水的**严格子集**；选项 B（纯删除不补）被否决：导入更新存量 AD 用户部门后 5 分钟全量同步窗口内被 AD 登录回写冲回（F1 乒乓土壤，Phase 59 守卫前无防护）；选项 C（保留+互斥）被否决：互斥防并发不防跨路径乱序。
  - 已知并接受的代价：入队在 ExcelService 事务**外**（crash 窗口丢推送，outbox 同生共死不变量此路径打折）；队列洪峰按排水节奏消化（分钟级 vs 现 ~10 秒，同数量级可接受）。

### 配置路由（区 2，用户选定排水时解析）
- **D-02: 队列行不加 `ad_config_id` 列——排水时按 `sys_user.ad_dn` 归属解析配置**（ROADMAP SC3 假设成立，零迁移、57 五条已上线路径零返工）。**显式契约：最长后缀唯一匹配；无匹配或多匹配 → 队列行转 dead 不推**。契约必须写进设计文档并由单测锁定（反例：父子域共享后缀 `DC=na,DC=corp,DC=com` 字符串级 ends-with `DC=corp,DC=com`）。"匹配不到置 dead"与 QUEUE-05（code 32→dead）同一终态机制。
  - 否决加列：`sys_user` 无权威归属源，入队填列仍靠 DN 推断——推断未消除只是提前冻结且跨域移动后过期；migration_216 + 回填 + 五路径返工全动在 Phase 57 刚上线的同事务路径上。
  - `sys_user.ad_config_id` 用户级归属（登录/同步时持久化，根因修复）记 Deferred——触碰登录热路径，超 58 交付边界。
- **D-03 (HYGIENE-01): `getDefaultADConfigID`（`internal/scheduler/ad_sync_tasks.go:306`）与 `getDefaultADConfigIDForDept`（`internal/scheduler/dept_sync_tasks.go:47`，逐字拷贝）合并为公共 helper**（放 addomain 包）；`ou_mapping_handler.go:117` 的 `sync_enabled=true AND status=enabled .First()` 是第三处同类歧义，顺带收敛（或显式保留并注释，planner 定）。

### F-07~F-16 复核分档（区 3，用户选定契约+顺带清扫）
- **D-04: 三档分组。**
  - **纳入 58 正式范围**：
    - **F-11（dead 判定契约，必答项）**：合并子句每次编辑清零 `attempts`（`ad_push_queue_service.go:161-169`）——**排水器禁止裸 `attempts` 判 dead**，否则"每次编辑后都失败一次"的属性永不终态。具体方向（独立 merge-proof 标记 / failed 且 `updated_at` 超窗 / 显式接受"编辑即续命"写明后果）属 58 设计文档决策，但禁令本身进验收。
    - **F-07（孤儿 disable 行，双层）**：排水器侧 not-found → dead **必选**（与 code 32 同一条分支，边际成本≈0，保终止性）；入队侧 `BatchDelete` 事务内 `Pluck` existing 只对存在行入队为顺带加固（一行查询掐根 + "入队数==软删数"断言口径 + 堵无限刷行）。
    - **F-10**：补 `Delete`/`BatchDelete` 回滚路径 2 个用例（复用 `ad_push_queue_test.go` 既有 RollsBack 模式；断言 `deleted_at IS NULL` 且 `sys_user_role` 行数不变）。
  - **顺带一行修**：F-12（排水器是 `pushed_at`/`failed_at` 首个写入方——写时统一 `.UTC()`，顺带修 `ad_push_queue_service.go:182,273` 与 `models/ad_push_queue.go:92,104` 四处本地时区）；F-14（model 加 `default:'pending'` tag 与 DDL 对齐）；F-15（两处 fixture 补"故意不 seed 配置行"注释）；F-16（`BareInsertWithoutID` 加 `t.Skip`）。
  - **留档不修**：F-13（F-06 selective Updates 已从根上消解别名风险，核对归档即可）；F-16 升级 PG 契约断言（v1.22 候选）。
  - 触碰 57 已合入代码（BatchDelete Pluck）需回归 42 个 Phase 57 既有用例（sqlite 快测）。

### 排水节奏与退避（区 4，用户选定 30s + sys_config）
- **D-05: 排水间隔默认 30s，sys_config 单旋钮 `ad_push_queue.drain_interval_seconds`**（启动读取、改后重启生效——复制 `network.connection_pool.*` 先例；不实时生效）。管理员编辑→AD 生效感知 ≤1min；给"域控吃不消"留放宽到 5min 的逃生门。这是**唯一**进 sys_config 的参数。
- **实现不变量（研究定论随选项生效，均为硬编码 discretion）**：
  - 单轮批量上限 100 行（LDAP 单写 10-100ms → 单轮 1-10s 天然有界）；
  - 退避曲线 60s→2min→4min→8min→16min 封顶 30min，**attempts 达 5 转 dead**——总重试窗 ~61min 刻意桥接账号池 30min 熔断周期（避免熔断窗内密集重试继续喂 MarkFailure，AD Modify 失败放大熔断有前科）；
  - **`next_retry_at IS NULL` = 立即可拾取**（谓词 `status IN ('pending','failed') AND (next_retry_at IS NULL OR next_retry_at <= now())`）——与 Phase 57 合并路径显式写 NULL 自洽的唯一解，反向语义会制造永不可拾取的僵尸行；
  - 不可恢复错误（code 32 / 配置无匹配类）短路直 dead 不烧 attempts（沿用 D-04 既有语义）；
  - `cron.SkipIfStillRunning` 防轮次重叠（30s 短间隔下单轮可能超 30s）；手动触发路径复用同一互斥。

### Claude's Discretion
- 批量 `EnqueueUpdate` 变体具体签名/命名；ExcelService 写用户后批量入队的挂接点与切批（`pushQueueBatchSize=500` 先例）
- F-11 dead 判定具体方向（独立标记 vs updated_at 超窗 vs 显式接受续命）——禁令进验收即可
- 排水器代码位置（addomain 包 vs scheduler 包，按依赖方向定；参照 D-03 helper 放 addomain）
- 审计输入清单技术项：`Unscoped()` 读 `sys_user.ad_dn`、`pushed_at`/`failed_at` 终态一致性、update_user/disable 处理顺序（F-01 压制已缓解，按 created_at 序即可）
- HYGIENE-02 OU DN 转义 helper 实现细节（LDAP DN 特殊字符转义语义，与 NormalizeDN 同放单一真实源）
- NormalizeDN 归一化比较的单一真实源位置（三方审计要求放 addomain 包）

### Folded Todos
（无匹配 todos——operlog exclude_paths 与 v1.17 对账待办均不在本 phase 范围）

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### 规划文档
- `.planning/REQUIREMENTS.md` — QUEUE-03/04/05、HYGIENE-01/02 原文 + 覆盖矩阵（:11-13, :30-31, :54-76）
- `.planning/ROADMAP.md` — Phase 58 条目：SC1-6、移交待决项 1/2/3、wave 顺序约束 HYGIENE-01 先于 QUEUE-04（:78-105）
- `.planning/PROJECT.md` — v1.21 关键决策段（compare-and-move / 禁 sys_job / 时间戳仲裁死刑 / 禁新旧值比较）
- `.planning/phases/57-ad-outbox-lite/57-CONTEXT.md` — D-01~D-08 入队契约（D-03 handler 旧路径删除锁定、D-05 最小动作行、D-06 部分唯一索引、D-07 退避列已建齐）
- `.planning/phases/57-ad-outbox-lite/57-REVIEW.md:147-236` — F-07~F-16 逐条全文 + "给 Phase 58 的输入清单"（:240-244：Unscoped 必读、pushed_at/failed_at 无写入方、NULL next_retry_at 语义、处理顺序）
- `.planning/phases/57-ad-outbox-lite/57-REVIEW-FIX.md` — F-01~F-06 修复细节（F-01 disable 压制、F-06 写路径收窄）+"给 Phase 58 的输入补充"节

### 队列基建（排水对象）
- `internal/models/ad_push_queue.go` — 队列模型 + `ADPushQueueActivePredicate` 单一事实源常量
- `internal/services/addomain/ad_push_queue_service.go` — 入队 service：`IsADSyncEnabled` 前置（:39-47）、合并 upsert（:129-137）、attempts 归零点（:161-169，F-11）、`EnqueueDisable` 批量模式（:220-243，D-01 新变体模板）
- `internal/core/db/migrations/migration_215_*.go` — 部分唯一索引 + `(status, next_retry_at)` 拾取索引 DDL

### AD 执行侧
- `internal/services/addomain/ldap_client.go` — `MoveUser` :350、`DisableUser` :337（幂等可重放）
- `internal/services/addomain/user_ad_sync_service.go` — `SyncUserUpdateToAD` :44-104（属性 Modify 语义）、`BatchSyncUsersToAD` :687（含 :679-683 "无 OU move"注释——D-01 依据）
- `internal/scheduler/ad_sync_tasks.go` — :306 `getDefaultADConfigID`（合并对象一）、:118-162 in-process cron 先例（`robfig/cron` WithSeconds、每 5min AddFunc）
- `internal/scheduler/dept_sync_tasks.go:47` — `getDefaultADConfigIDForDept`（合并对象二，逐字拷贝）
- `internal/api/v1/system/ou_mapping_handler.go:117` — 第三处 `First()` 配置歧义（D-03 顺带收敛对象）
- `internal/services/addomain/failover_client.go` — 账号池熔断 30min 周期（D-05 退避曲线桥接依据）

### 直推删除目标
- `internal/api/v1/system/user_handler.go:194-238` — 单用户编辑直推（57 D-03 已锁删，含 `adSyncInFlight` :28-32、`buildADSyncMap` :245-260 一并清理）
- `internal/api/v1/system/user_import_handler.go:114-148` — `triggerADSyncAfterImport`（58 D-01 删 + 换批量入队）

### 先例
- sys_config 动态配置先例：`network.connection_pool.*`（migration_203 seed + 启动读取 + 改后重启生效，D-05 单旋钮照抄此模式）
- 双方言迁移先例：migration_199 / 212 / 215（PG/sqlite 部分唯一索引与建表）
- 回滚测试模板：`internal/services/addomain/ad_push_queue_test.go` 既有 4 个 RollsBack 用例（F-10 两个新用例复制源）

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `cron.SkipIfStillRunning`（robfig/cron v3 chain wrapper）— 30s 短间隔防轮次重叠的原生解；手动触发路径需自行复用同一互斥
- `EnqueueDisable` 批量入队模式（预过滤 + 前置门控 + `DedupeEntityIDs` + `pushQueueBatchSize=500`）— D-01 批量 `EnqueueUpdate` 变体的直接模板
- `ADPushQueueActivePredicate` 常量 — 排水器拾取谓词扩展时与索引 DDL 保持单一事实源
- 既有 RollsBack 测试用例 — F-10 补测的复制模板

### Established Patterns
- Handler-Service 分层：排水器为 service 层组件，handler 不碰队列；入队只能在 service 事务内（ExcelService 导入路径除外——D-01 已接受的打折）
- in-process cron（`StartADSyncScheduler` 先例）+ 禁注册 sys_job（里程碑锁定）
- sys_config 动态配置：启动读取、改后重启生效（`network.connection_pool.*` 先例）
- 双方言（PG/sqlite）迁移与测试：sqlite 单测覆盖约束语义，PG 专属故障（ON CONFLICT 二次命中等）指向验证清单
- LDAP 写经 `FailoverClient`（账号池 + 熔断）——排水器所有 LDAP 操作走同一执行面

### Integration Points
- `(status, next_retry_at)` 拾取索引 — 排水器 WHERE 谓词的物理基础
- Phase 59 GUARD-01 信号守卫查 pending/failed 行 — 排水器终态转换（pushed/dead）直接决定守卫信号质量；dead 行退出唯一索引但留表可查（QUEUE-03）
- `user_service.go` 五条入队路径 — F-07 入队侧 Pluck 加固点（BatchDelete :569 附近）
- `user_sync_service.go` AD 登录回写 — 5 分钟全量同步窗口语义（D-01 否决 B 方案的依据；Phase 59 守卫前的已知风险窗）
- `getDefaultADConfigID` 双份 + `ou_mapping_handler.go:117` — D-03 helper 合并后的三处调用点收敛

</code_context>

<specifics>
## Specific Ideas

- 用户 4 区全选研究推荐项，无个性化偏离（delegator 风格延续 Phase 57）
- 退避曲线数值刻意桥接账号池 30min 熔断周期——用户采纳该研究定论
- SC6 终态 = 两条直推路径均删除（单用户 D-03 + 导入 D-01），明确拒绝"互斥共存"路线
- D-02 契约措辞升级为验收级："最长后缀唯一匹配，多匹配/无匹配置 dead" 必须有单测锁定（含父子域共享后缀反例）
- 三方审计否决项持续有效：禁时间戳仲裁、禁新旧值比较、禁 sys_job、禁 JSON 快照、禁 delete AD 账号

</specifics>

<deferred>
## Deferred Ideas

- **`sys_user.ad_config_id` 用户级归属**（登录/同步时持久化，消除 DN 推断的根因修复）— 触碰登录热路径，超 58 交付边界；多域真实排期时独立立项
- **F-16 升级 PG 契约断言 + PG 测试基座** — 项目级基础设施，v1.22 候选（PG 专属 UPSERT 语义当前无真覆盖）
- **F-07 入队侧 Pluck 降档** — 若 B 档执行超载可单独留档（排水器侧容错已中和用户可见危害）
- **UpdateStatus 双向推送**（停用→disable / 启用→enable）— 57 遗留，v1.21 范围外，动作枚举扩展位已预留
- **auth.go AD 登录路径不查 user.Status** — 57 遗留独立 bug，候选后续 phase 或并入 GUARD 工作

</deferred>

---

*Phase: 58-排水器与 compare-and-move 推送*
*Context gathered: 2026-10-17*
