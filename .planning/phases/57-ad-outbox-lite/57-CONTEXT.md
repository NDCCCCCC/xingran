# Phase 57: AD 推送队列基建（outbox-lite 同事务入队） - Context

**Gathered:** 2026-10-16
**Status:** Ready for planning

<domain>
## Phase Boundary

落地 `ad_push_queue` 表（**migration_215**，roadmap 中"migration_200"编号已失效——214 已被 ad_login_history 占用）+ `user_service` 三条写路径（Update / Delete / BatchDelete）同事务入队 + 部分唯一索引按实体合并 dedupe + sqlite 测试环境补建表（HYGIENE-06）。

**不含：** 排水器与 compare-and-move（Phase 58）、登录守卫（Phase 59）、对账 VIEW（Phase 59）、handler 旧推送路径删除（移入 Phase 58 范围，见 D-03）。
</domain>

<decisions>
## Implementation Decisions

### 入队触发与降级语义（区 1，用户选定 diff 入队）
- **D-01: diff 入队** — 对比事务前已加载的旧值（`user_service.go:148-154` 的 `First` 即旧值），nickname/email/phone/deptId 四字段实际发生变更才入队；纯角色/岗位/备注/状态编辑不产生队列行。nil 指针语义：req 字段 nil = 无变更，跳过该字段。diff 逻辑做成纯函数 helper，可单测覆盖 nil/空串边界。
- **D-02: 降级语义（两层，研究定论随选项生效）**
  - **前置条件**：入队前查 `sys_ad_config` 无 `sync_enabled=true AND status=enabled` 行 → 不入队 + WARN 日志放行用户编辑（真实信号；`user_handler.go:194` 的 `userADSyncService != nil` 判断只防测试构造，`user_router.go:35` 无条件构造）。否则"AD 未部署"环境积累永 pending 行，污染 Phase 59 GUARD-01 信号守卫。
  - **队列 INSERT 真失败** → 同事务回滚，用户编辑整体失败（"跳过+WARN"会静默架空 QUEUE-01 不丢语义；同库同事务下 INSERT 失败≈DB 级故障）。

### 旧推送路径切换时序（区 2，用户选定保留至 58）
- **D-03: handler 旧推送路径（`user_handler.go:194-238`，含 `adSyncInFlight` dedupe、`buildADSyncMap`、goroutine + 3 次重试）保留至 Phase 58，随排水器同一二进制删除**。理由：57→58 存在真实部署间隙（手动上传二进制 + 部署漂移前科），保留使间隙期行为=现状，无推送真空；58 排水靠 compare-and-move 幂等（DN 归一化相等跳 MoveUser、属性 Modify 幂等）消化双推积压，并补推旧路径 dedupe 曾丢弃的编辑。
- **⚠ Roadmap 修正项：handler 旧路径删除必须写进 Phase 58 范围**——当前 roadmap 无任何 phase 调度此删除点（Phase 62 HYGIENE-03/F-22 的"旧路径"是 `user_ou_service.go` 等另一组死代码，依赖 Phase 60 SAFE-01，与本 handler 推送路径无关）。
- QUEUE-02 验收固定在 service 层断言队列表状态（handler 测试已支持 `userADSyncService=nil` 隔离旧路径）；间隙期 UAT 需知悉 AD 端仍由旧路径推送。
- 否决"即时排水钩子"（选项 c）：要么把 58 核心（compare-and-move/按 ad_dn 路由配置/code 32 终态/DN 转义）提前拖进 57，要么退化为同步调 `SyncUserUpdateToAD` 的旧路径变体且把 LDAP 延迟耦合进 HTTP 请求。

### QUEUE-06 disable 边界（区 3，用户选定 Delete+BatchDelete）
- **D-04: `Delete`（user_service.go:228）与 `BatchDelete`（:569）两条软删事务路径都在事务内入队 `disable AD 账号` 动作行**。严格按 REQUIREMENTS 字面只挂 Delete 会让批量删除成为复活链路残留（BatchDelete 是同语义软删）。
- 动作枚举预留扩展位（disable 之外未来可加 status 类动作），但 **UpdateStatus 双向推送明确排除在 v1.21 外**（停用→disable / 启用→enable 属 scope 膨胀；见 Deferred）。
- disable 复用现成 `ldap_client.go:337 DisableUser`（幂等，UAC 514 Replace 可重放）；**禁止 delete AD 账号**——addomain 全库无 DeleteUser 先例，delete 销毁 group membership 且不可逆恢复，与 Phase 60 SAFE-01 ad_dn 校验恢复不兼容。
- 复活链路代码证据（守门人 = AD bind 本身）：AD bind 成功 → `ad_authenticator.go:131` → `user_sync_service.go:271 restoreSoftDeletedUser` 清 deleted_at + `applyRestoreSoftDeleted`（:295-307）强制 enabled + 密码重置 → `auth.go` 签发 JWT。UAC 514 后 bind 即失败（data 533），故 disable 即闭环。

### 表结构与合并语义（区 4，用户选定部分唯一索引）
- **D-05: 载荷 = 最小动作行**（entity_type / entity_id / action_type），**禁用 JSON payload 快照**——快照过期正是已判死刑的时间戳仲裁土壤；compare-and-move 下排水闭包重读 DB 后全量推送是幂等超集，Phase 59 守卫只查"该用户有无 pending/failed 行"，最小动作行覆盖全部三个 Phase 查询需求。
- **D-06: 合并键 = 部分唯一索引 `UNIQUE(entity_type, entity_id, action_type) WHERE status IN ('pending','failed')`**。upsert 覆盖即 QUEUE-02 合并（新编辑重置 failed 行退避）；终态行（pushed/dead）退出唯一索引、留表可查（QUEUE-03"可查询确认"+ 排障友好）。ON CONFLICT 冲突目标必须文本级重复 WHERE 谓词（PG/sqlite 同要求，写错即 42P10）。先例：migration_199、migration_212（双方言部分唯一索引，本项目 sqlite 内核 3.35+ 已验证）。合并必须走唯一索引夺锁，禁用"先查后写"应用层合并（net_device_enrichment_task 并发窗口 + 僵尸前科）。
- **D-07: Phase 58 退避列一次建齐**（attempts / next_retry_at / last_error / failed_at，nullable 不阻断），让 (status, next_retry_at) 拾取索引与唯一索引谓词一次设计到位，避免 58 再出迁移。
- **D-08: 表杂项** — 队列表**不软删**（无 DeletedAt；ConfigRestoreTask 先例：扁平结构 + BeforeCreate 生成 UUID 防御 23502）；终态用 *time.Time 可空列；模型 tag **不得声明该唯一索引**（避免 GORM/PG 约束命名冲突 W-1 教训），只能手写 migration_215 显式 `uni_` 命名；sqlite 测试建表直调迁移函数（先例 `config_backup_service_93_02_test.go:402`）或复用 fix_suggestion sqlite 测试基建。

### Claude's Discretion
- diff helper 具体签名/命名、队列 service 包位置（addomain vs system，planner 结合依赖方向定）
- WARN 日志格式、entity_type 常量取值（如 "user"）
- sqlite 测试建表方式二选一（直调迁移 vs 复制建表 SQL）
- action_type 枚举具体字符串值

### Folded Todos
（无匹配 todos——operlog exclude_paths 与 v1.17 对账待办均不在本 phase 范围）

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### 规划文档
- `.planning/REQUIREMENTS.md` — QUEUE-01/02/06、HYGIENE-06 原文 + 覆盖矩阵（:9-14, :35, :58-76）
- `.planning/ROADMAP.md` — Phase 57/58 条目与依赖约束（:41-75）；注意 Notes 中"migration_200"编号失效，实际 migration_215
- `.planning/PROJECT.md` — v1.21 关键决策段（时间戳仲裁死刑 / outbox-lite / 信号守卫 / compare-and-move / 禁 sys_job）

### 代码锚点（改动目标）
- `internal/services/system/user_service.go` — Update :146-226（事务 :157，diff 旧值来源 :148-154）、Delete :228-259、BatchDelete :569、UpdateStatus :602
- `internal/api/v1/system/user_handler.go` — 旧推送路径 :194-238（Phase 58 删）、`adSyncInFlight` :28-32、`buildADSyncMap` :245-260（四字段契约参照）
- `internal/api/v1/system/user_router.go` — :35 无条件构造 UserADSyncService（nil 判断只防测试）

### 复活链路与 AD 侧能力
- `internal/services/system/user_sync_service.go` — restoreSoftDeletedUser :271、applyRestoreSoftDeleted :295-307（强制 enabled + 密码重置）
- `internal/core/security/ad_authenticator.go` — :131 AD bind → SyncADUser 入口
- `internal/api/v1/auth.go` — :177-268 AD 登录路径不查 user.Status（Deferred 相关）
- `internal/services/addomain/ldap_client.go` — EnableUser :332、**DisableUser :337（QUEUE-06 执行侧复用）**、group DelRequest :401（全库唯一 Delete 类先例）
- `internal/services/addomain/user_ad_sync_service.go` — SyncUserUpdateToAD :44-104、配置检查 :59-61（`sync_enabled=true AND status=enabled` 判定参照）
- `internal/services/addomain/failover_client.go` — MarkFailure 熔断（no-op LDAP 写代价的现实依据）

### 表结构先例
- `internal/models/config_restore_task.go` — 队列表模型先例：无 DeletedAt、BeforeCreate UUID、终态 *time.Time
- `internal/core/db/migrations/migration_199_fix_suggestion_unique_index.go` — 部分唯一索引独立迁移 + 显式 uni_ 命名
- `internal/core/db/migrations/migration_212_create_restore_task_active_unique_index.go` — 双方言（PG/sqlite）部分唯一索引注册
- `internal/core/db/migrations/migration_214_create_ad_login_history.go` — 当前最新编号（215 起顺延）
- `internal/core/db/database.go` — MigrateModelList :646、AutoMigrate :755、migration 调用点 :992-1087

### 测试基建
- `internal/services/asset/fix_suggestion_service_test.go` — :36-77 sqlite 测试建表基建
- `internal/services/system/config_backup_service_93_02_test.go` — :402 直调迁移函数建表先例

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `addomain/ldap_client.go DisableUser`（:337）— QUEUE-06 disable 执行原语，幂等可重放
- 部分唯一索引双先例（migration_199/212）— ON CONFLICT + WHERE 谓词双方言写法直接照抄
- `ConfigRestoreTask` 模型 — 队列型表的全部惯例（无软删 / UUID BeforeCreate / 可空终态时间列）

### Established Patterns
- Handler-Service 分层：入队逻辑放 service 层（`user_service.go` 事务内），handler 不碰队列
- 手写 migration_NNN 函数 + database.go 注册；GORM AutoMigrate 仅作加列安全网
- sqlite 单测双方言约束：ON CONFLICT 冲突目标须文本匹配部分索引谓词（sqlite 3.35+ 行为，modernc v1.40.1 满足）

### Integration Points
- `user_service.go:157` Update 事务块 — diff + 入队插入点
- `user_service.go:240` / `:569` Delete / BatchDelete 事务块 — disable 入队插入点
- `sys_ad_config` 启用行检查 — 入队前置条件（D-02）
- Phase 58 排水器（下游）、Phase 59 登录守卫查 pending/failed 行（下游消费者）

</code_context>

<specifics>
## Specific Ideas

- 用户在区 1-4 全部选定研究推荐项，无个性化偏离
- 降级语义（D-02 两层）虽未单独设问，随区 1 推荐项一并生效——研究定论：无 AD 配置不入队是前置条件而非降级；INSERT 失败回滚是"同事务入队"的唯一诚实实现
- 三方审计否决项约束持续有效：禁 JSON 快照（D-05）、禁应用层先查后写合并（D-06）、禁 delete AD 账号（D-04）

</specifics>

<deferred>
## Deferred Ideas

- **UpdateStatus 双向推送**（停用→disable / 启用→enable）— v1.21 范围外；动作枚举已预留扩展位（D-04），后续 milestone 可低成本接入
- **auth.go AD 登录路径不查 user.Status**（停用用户仍可 AD bind 登录拿 JWT，local 路径有检查 :39）— 独立 bug，修法应为 auth 补 ~3 行 status 检查而非队列推送；候选后续 phase 或并入 GUARD 相关工作
- **BatchDelete 之外的批量路径排查**（如 Excel 删除类入口若有）— Phase 57 只覆盖已确认的两条软删事务路径

</deferred>

---

*Phase: 57-AD 推送队列基建（outbox-lite 同事务入队）*
*Context gathered: 2026-10-16*
