# Phase 57 代码审查修复报告（F-01 ~ F-06）

**来源**：[57-REVIEW.md](57-REVIEW.md)
**日期**：2026-10-16
**范围**：F-01 ~ F-06（审查报告标注"建议在本 phase 内修掉"的全部 4 条高危 + 2 条中危）

---

## 汇总

| Finding | 级别 | 主题 | 提交 | 状态 |
|---|---|---|---|---|
| F-01 | 高（破坏性） | disable 入队压制同实体活跃 `update_user` 行 | `0e61c384` | 已修 |
| F-02 | 高（安全） | 编辑表单停用用户时同事务入队 disable | `3c57bfa6` | 已修 |
| F-03 | 高（部署） | `SKIP_AUTOMIGRATE=true` 旁路补建 `ad_push_queue` | `11efec8f` | 已修 |
| F-04 | 高（部署） | 队列表/索引创建失败阻断启动 | `9bdecab2` | 已修 |
| F-05 | 中（测试有效性） | disable id 去重抽成可单测纯函数 | `a76feda4` | 已修 |
| F-06 | 中（数据完整性） | Update 写路径收窄为显式字段，与 diff 语义对齐 | `fbf9ba3b` | 已修 |

每条修复均**独立原子提交**，且都在提交前实测"回退生产代码 → 新测试转红 → 恢复"（RED 证据见各条）。

**未处理**：F-07 ~ F-16（审查报告判定为 Phase 58 输入或低危，不在本轮 fix_scope 内）。

---

## F-01 disable 压制 update_user 行（`0e61c384`）

**问题**：`action_type` 是部分唯一索引的一维，同一用户可同时存在 `pending update_user` 与 `pending disable`。用户软删后排水器重读 `sys_user` 会落空 → 推送空属性进 AD（破坏性）或变僵尸行。

**改动**
- `EnqueueDisable` 在入队 disable 的**同一事务**内，把这些实体所有 `status IN ('pending','failed') AND action_type='update_user'` 的行打终态 `dead`，`last_error='superseded by disable'`，清空 `next_retry_at`/`failed_at`。
- 放在 D-02 配置门控**之前**：压制是"已失效队列行"的安全性不变量，与入队降级策略无关（AD 重新启用后仍须安全）。
- 终态用既有 `dead` 而非新增 `superseded`：`dead` 不在索引谓词 `status IN ('pending','failed')` 内 → 索引 DDL 与 ON CONFLICT TargetWhere 的谓词**一行未改**。
- 顺带把谓词提为单一事实源 `models.ADPushQueueActivePredicate`，migration_215 DDL 与 service TargetWhere 复用同一常量（结构上消除漂移）。

**偏离建议处**：审查给的片段用 `models.ADPushStatusPending` 等自定义类型直接进 `[]interface{}` 切片参数。实际改为 `[]string{string(...)}` —— GORM 对自定义 string 类型切片参数会展开为参数列表而非字面量 IN，写法必须显式转 string，否则 sqlite/PG 行为不一致。

**守护测试**：`TestPhase57_EnqueueDisable_SupersedesPendingUpdateUser`（删除压制逻辑后转红）、`..._SupersedesFailedUpdateUser`、`..._SupersedeScopedToTargets`、`..._NoADConfig_StillSupersedes`、`TestPhase57_ActiveRowPredicate_MatchesIndexDDL`（谓词被改成含 `dead` 后转红）。

---

## F-02 停用经编辑表单入队 disable（`3c57bfa6`；GAP-01 补全见下）

**问题**：停用有 `UpdateStatus()` 与 `Update()` 两个入口，Phase 57 只接了前者 → 系统停用而 AD 账号仍启用，域内可登录。

> **⚠ 本节描述的两处事实错误已于 GAP-01 修复时更正（保留原文以存真）：**
> ① 原文"Phase 57 只接了前者（`UpdateStatus`）"与代码事实**相反** —— `3c57bfa6` 实际接的是
> `Update()`（编辑表单），`UpdateStatus()`（列表页 Switch，`POST /system/users/:id/status`）
> 当时**未接线**。`ad_push_queue_service.go` 上同源的错误注释已一并改正。
> ② "Phase 57 只接了前者"这一措辞本身也把缺口说反了：只接 `Update` 而不接 `UpdateStatus`
> 同样留下安全洞，且 `UpdateStatus` 才是列表页停用开关（最高频入口）所走的路径。
>
> **更正后的真实状态**：`Update()`（`3c57bfa6`）与 `UpdateStatus()`（GAP-01 修复）**两个入口
> 现均已同事务入队 disable**，见下方 "GAP-01 补全" 一节及其守护测试。

**改动**
- 新增 `addomain.EnqueueDisableOnStatusTransition(tx, oldStatus, newStatus, userID)`：仅 0→1 入队；D-02 降级与 `EnqueueDisable` 一致。
- 不做 F-01 压制：用户行仍在（`status=1`），残留 `update_user` 行仍可正常推送，两条动作按 `created_at` 先后处理语义正确。
- 启用（1→0）刻意不入队：`ADPushActionEnable` 在 v1.21 不存在。
- `user_service.Update` 在同事务内（同 tx、共用一个 enqueuer）调用。

**偏离建议处**：审查建议"在 `EnqueueUpdateIfChanged` 里显式处理 status 跃迁"。改为独立函数，理由：`EnqueueUpdateIfChanged` 的第一行是四字段 diff 短路（无四字段变更直接 return），status 跃迁会被这条短路吃掉，塞进同一函数必然要重排短路顺序；独立函数也让"停用方向不在 v1.21"的边界单独可读。

**守护测试**：`TestPhase57_UserService_Update_Disable_EnqueuesDisable`（删除接线后转红）、`..._Disable_RollsBackWhenQueueInsertFails`（删除接线后转红）、`..._NoStatusTransition_NoDisable`、`..._Disable_NoADConfig_NoEnqueue`。

**改写的既有断言**：`TestPhase57_UserService_Update_NoADFieldChange_NoEnqueue` 原断言"纯状态编辑产生 0 条队列行"编码的正是本缺口，改为按动作类型收窄（`update_user` = 0），并在注释里注明修正来源。

### GAP-01 补全：`UpdateStatus` 入口（列表页 Switch）

**问题**：F-02 只接了编辑表单，`user_service.UpdateStatus`（`POST /system/users/:id/status`，
前端用户列表的状态开关走这条）仍然裸写 `status`、零入队 → 系统已停用而 AD 账号仍启用，
且**没有任何队列行**可供 Phase 59 GUARD-01 信号守卫识别（守卫以 pending/failed 行判据，
"无行"会被读成"无待推送 → 一致"，比不接线更危险）。

**改动**
- `UpdateStatus` 改为 `s.db.Transaction(...)` 包裹：状态写与 `EnqueueDisableOnStatusTransition(tx, ...)`
  在**同一事务**、用 `tx` 而非外层 `db`，入队失败即回滚 —— 与 `Update`/`Delete`/`BatchDelete`
  同一形状（D-02 立场）。事务外的 `First` 只为取旧状态做 0→1 diff，不参与写。
- 1→0（重新启用）/ 0→0 / 1→1 三种情况仍为 no-op（v1.21 无 `ADPushActionEnable`）。
- 缓存层无需调整：`userCacheService.UpdateStatus` 已失效 user 列表与 user 单键缓存，
  入队不引入新的可见性面。
- 同步改正 `ad_push_queue_service.go` 中 `EnqueueDisableOnStatusTransition` 的错误注释
  （原文"Phase 57 只接了前者(UpdateStatus)"与代码相反）。

**守护测试**：`TestPhase57_UserService_UpdateStatus_Disable_EnqueuesDisable`、
`..._Disable_RollsBackWhenQueueInsertFails`、`..._NoTransition_NoDisable`、
`..._Disable_NoADConfig_NoEnqueue`。
**RED 证据**：revert 生产改动后前两条转红（`git stash push -- user_service.go` 实测）。

---

## F-03 旁路补建队列表（`11efec8f`）

**问题**：`SKIP_AUTOMIGRATE=true && postgres` 时只跑 `BootstrapMissingTables()`，队列表不存在 → 用户编辑/删除 500 并回滚。

**改动**
- 新增 `(*Database).ensureADPushQueue()` 调 `Migrate215CreateADPushQueue`（幂等），`BootstrapMissingTables()` 调用它 → 与 AutoMigrate 路径**复用同一 DDL 事实源**，不另写一份。
- 不用 `Migrator().CreateTable`：模型刻意不声明 `uniqueIndex` tag（W-1：AutoMigrate 的 DROP 无 IF EXISTS 会与手写索引名冲突致启动 FATA），裸 CreateTable 会留下没有部分唯一索引的表 → 运行期首触即 42P10。

**sqlite 分支核查**：该旁路只在 `d.Type == "postgres"` 时触发（`core.go:286` + `database.go:788` 双 guard），sqlite 走完整 `AutoMigrate` 并已包含迁移 215 调用 → 无缺口。但为防旁路语义未来放宽，helper 与断言写成双方言通用。

**守护测试**：`TestBootstrapMissingTables_CreatesADPushQueue`（删除旁路接线后转红：表缺失）、`TestEnsureADPushQueue_SharedByBothStartupPaths`（建表 + 唯一索引生效 + DDL 谓词取自共享常量 + 旁路建出的表可写）。

---

## F-04 索引失败阻断启动（`9bdecab2`）

**问题**：PG 分支下索引创建失败只打一行 ERROR 就继续，运行期第一次用户编辑撞 42P10 并回滚，真实原因埋在几小时前的启动日志。

**改动**：PG 与 sqlite 两个启动分支改为调 `d.ensureADPushQueue()`，失败直接 `return error`。选择"阻断启动"而非"事后断言"，依据本仓既有先例：`backfillSysAPIKeysKeyHash` / `dropSQLiteReconciliationViews` 等硬前置失败同样是 `return error` 上抛，由 `core.go initDBAndData` 终止启动（与 P0 #16 的 fail-fast 策略一致）。error 里点名 `uq_ad_push_queue_active`，排障无需回看源码。PG 分支 advisory lock 由既有 `defer releaseMigrationAdvisoryLock()` 正常释放。

**守护测试**：`TestAutoMigrate_FailsFastWhenADPushQueueIndexCannotBeCreated`（改回"非阻断打日志"后转红）、`TestAutoMigrate_SucceedsAndBuildsQueueIndexes`、`TestEnsureADPushQueue_FailsFastOnIndexFailure`。

**故障注入手法**：sqlite 中索引名与表名共享命名空间，先建一张名为 `uq_ad_push_queue_active` 的表即可让 `CREATE UNIQUE INDEX` 失败 —— 无需 PG 即可复现同一失败面。

---

## F-05 去重测试有效性（`a76feda4`）

**问题**：sqlite 对同一条 INSERT 内重复命中的 UPSERT 行不报错，把去重循环整个删掉该用例依然变绿 → 假断言。

**改动**
- 抽出 `addomain.DedupeEntityIDs([]string) []string` 纯函数（丢空串、首次出现顺序稳定、不改入参），`EnqueueDisable` 改为调用它（行为等价）。
- 新增 `TestPhase57_DedupeEntityIDs_PureFunction` 表驱动断言（去重 / 空串 / 顺序 / 纯度）—— 删除函数体后 5 个子用例转红。
- 原 `TestPhase57_EnqueueDisable_DedupesDuplicateIDs` **保留**，注释改写为"sqlite 行为记录，非契约断言"并指向新用例（不删证据）。PG 专属故障（`ON CONFLICT DO UPDATE command cannot affect row a second time`）仍需 PG 测试环境，已在注释里指向 Phase 58 验证清单。

---

## F-06 写路径与 diff 语义对齐（`fbf9ba3b`）

**问题**：`tx.Save(&user)` 全字段覆盖 → 省略字段被写 NULL/零值；而 `adFieldDiffers` 把 nil 当"无变更、不入队"。只带部分字段的请求静默清库却不入队。

**改动**
- 事务内改为 `tx.Model(&models.User{}).Where("id = ?", req.ID).Updates(字段映射)`，只写 `!= nil` 的字段 → **nil = 未提供 = 不写 = 无变更**，与 diff 一致。
- `gender` / `status` 是值类型，`ShouldBindJSON` 无法区分"未传"与"零值"，保持既有全量写语义（前端表单始终携带）；status 跃迁由 F-02 的入队逻辑显式 diff，不依赖"是否写入"。
- 部门：`nil` = 不动；**显式空串 = 清空** `dept_id` + `dept_name`（保住前端清空选择器的表达方式）；查不到部门时保持旧 `dept_name`，不把有效字段写成 NULL。
- 顺带按 F-13 的方向改写 `oldUserSnapshot` 注释的不变量表述（写路径不再改动 `user` 结构体，"原地解引用改写指针"的风险自然消除）。

**偏离建议处**：审查给了 (a) 让 diff 与写路径一致（nil 视作 `""`）与 (b) handler 层加 `binding:"required"` 两个选项；fix_scope 明确要求选"收窄写入"路线，故按上述实现。选"显式空串 = 清空"是为了在收窄写入后仍保留清空字段的能力（否则 UI 清空部门/昵称将变成 no-op）。

**守护测试**：`TestPhase57_UserService_Update_PartialRequest_PreservesOmittedColumns`（换回 `tx.Save` 后转红）、`TestPhase57_UserService_Update_ExplicitFieldsStillWrite`、`TestPhase57_UserService_Update_OmittedField_NeverEnqueues`。

**改写的既有断言**：`TestUserServiceCRUD_UpdateFullPaths` 中"DeptID 置 nil → DeptName 清空"锁定的正是本缺口，改为"省略不动 + 显式空串才清空"并注明修正来源。

`UpdateStatus()` 走 `Model().Update("status")` 单列路径，未受影响（`TestUserService_UpdateStatus_*` 全绿）。

---

## 验证

| 项 | 结果 |
|---|---|
| `go build ./...` | exit 0 |
| `go vet`（services/system、services/addomain、core/db、core/db/migrations、models） | 无输出 |
| `go test ./internal/services/system/ -run TestPhase57 -v` | 33 例全绿（既有 20 + 新增 13），0 FAIL |
| `go test ./...` | exit 0，无 FAIL 行 |

## 给 Phase 58 的输入补充

- **F-01 已消解队列语义**：delete 入队即压制该实体的活跃 `update_user` 行 → 排水器不必再处理"软删用户的属性推送"分支，但**仍需 `Unscoped()` 读取 `sys_user.ad_dn`**（软删行的 DN 仍在行内）。
- **F-02 新增一类入队来源**：`Update` 表单停用也会产生 `disable` 行，其 `entity_id` 对应的 `sys_user` 行**未软删**（status=1）→ 排水器按 `ad_dn` 路由时两种来源都必须能取到 DN。
- **F-06 的写语义**：省略字段不入队也不写库；排水器"重读 DB 全量推送"读到的就是最终值，不存在"库里被清空但队列没动"的偏差。
- **F-04 的 fail-fast**：队列表/索引缺失会在启动期暴露，不再是运行期 42P10。
