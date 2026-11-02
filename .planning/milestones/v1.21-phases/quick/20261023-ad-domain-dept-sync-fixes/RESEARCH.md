# RESEARCH — 域控部门向同步修复（P0×3 + P1×7 + P2×6）

> 调查来源：subagent 7e374610（AD→影子与组管理）、subagent 75d39dbb（系统→AD）、subagent 62f1f529（修复细节补充）、生产探针 scripts/diag/sync_audit_probe（v3）。
> 全程只读。所有事实带 `文件:行号`。

## 一、生产现状证据（探针 v3，2026-10-23）
- sys_job 生产任务：`dept_to_ad_sync` cron=0 0 2 * * ?（每天 02:00）；`dept_member_to_ad_group_sync` cron=0 */15 * * * ?（每 15 分钟）——两条任务由用户手工建（migration 137 只 seed 了后者，见 D-1）。
- 5 条 OU-组映射：4 条 last_sync_at=18:15 整点（任务在跑）；「三农中心 承保理赔部」last_sync_at=NULL（5/28 创建至今从未成功，OU DN 疑似 AD 端不存在）。
- `sys_ou_group_mapping_sync_log` 0 条——活跃映射从不写日志（日志链路从未接线）。
- 幻影表残留 0 → migration_222 已生效。

## 二、精确代码事实

### A. sys_job 执行器（P0-3 修复点）
- `internal/scheduler/cron.go:26-31` JobExecutor{job, db, logger, scheduler}；`Execute` :34-77；`executeTask` :80-98。
- **裸注册** `cron.go:314-319`：`s.cron.AddFunc(job.CronExpression, func(){ ctx := context.Background(); executor.Execute(ctx) })` —— 无 Recover、无防重叠、无超时；panic 击穿 goroutine。
- Logger：`cron.go:142-147` 接口；`cronLogger()` 适配器先例 `internal/scheduler/ad_sync_tasks.go:270`。
- **chain 先例** `ad_sync_tasks.go:203-216`：`s.cron.AddJob(expr, cron.NewChain(cron.Recover(lg), cron.SkipIfStillRunning(lg)).Then(cron.FuncJob(func(){...})))`。
- Job 模型 `internal/models/log.go:76-88`：`Concurrent bool gorm:"default:false"`；JobStatus 0=正常 1=暂停。**executor 从不读 Concurrent**（job_service.go:145/:205 仅透传存库）。
- 超时常量存放先例：`pkg/constants/timeouts.go`。

### B. 组成员同步（P0-1 / P1-5 / P1-6 / P2×3 修复点）
`internal/scheduler/dept_sync_tasks.go:104-267` executeDeptMemberToADGroupSyncTask：
- :131-137 查映射 `ad_config_id + mapping_status=active + sync_enabled=true`；:151-152 FailoverClient；operation 边界=整批（:157-256 闭包）。
- :166-173 查 AD 用户：`Where("ou_dn LIKE ? AND is_enabled = ?", "%"+mapping.OUDN, true)` —— **无 ad_config_id 过滤、LIKE 无转义**。
- :199-227 逐用户 AddGroupMember（N+1）；"Entry Already Exists"/"68"、"No Such Object"/"32" 靠 `strings.Contains(errMsg, ...)`。
- :243-246 裸 `db.Model(&mapping).Update("last_sync_at", ...)`（不复用 UpdateSyncStatus）；**只有 Add，没有 Remove**（调岗/离职残留组权限）；`OUGroupMappingSyncLog`/`CreateSyncLog`(ou_group_mapping_service.go:232) 零生产调用。
- LDAP 能力：`ldap_client.go:374 AddGroupMember` / `:382 RemoveGroupMember` / `:416 AddGroupMembers(空切片直返nil)` / `:428 RemoveGroupMembers`；**无 GetGroupMembers**——读组当前 member 需经 `:223 SearchWithRequest` 新增封装；接口 `ldap_iface.go:18-55`，mock `ldap_client_mock_test.go:12-181` 手写全 22 方法（**新增接口方法必须同步 mock**，iface 断言 :59）。
- `getGlobalADAccountPool` :21-27：nil 时 Errorf 返 nil → 调用方 panic。

### C. 部门→OU 同步（P1-7 / P1-8 / P2 修复点）
`internal/services/addomain/dept_sync_service.go`：
- `SyncDeptStructureToAD` :31-110：:81-87 **单树失败 `result.Status="failed"` + `break`，闭包恒 return nil**（外层永远"成功"，且中断其余树）；:102-104 running→completed。
- `syncDeptTree` :117-173：CreateOU 失败 return error 传播；UpsertMapping 冲突不计失败；递归子失败仅 log。
- `getRootDepartments` :176-184：`Preload("Children.Children.Children")` **固定 3 层截断**——应改为读全表按 ancestors 内存建树。
- `BuildOUDN` `dn_util.go:63-74`（ldap.EscapeDN，全仓唯一拼接点）。
- `department_service.go`：Delete :167-196 只软删 sys_dept，**无映射/OU 联动**（P0-2）；Update :73-152 唯一 AD 联动 = leader 变更 EnqueueUpdateManager(:140)。

### D. 迁移与模型
- migration 137 seed（archive/legacy-2026-06-15/137_add_ad_sync_jobs.sql）：`WHERE NOT EXISTS ... invoke_target='dept_member_to_ad_group_sync' AND deleted_at IS NULL` 幂等 INSERT sys_job；concurrent=false、status=0。**dept_to_ad_sync 无 seed**（P1-4）。
- 当前最大迁移号 **223**（database.go:1135 PG / :1205 sqlite，非阻断模式先例）；下一个 = **224**。
- `internal/models/ou_group_mapping.go:20-37`：唯一索引 `uni_ou_group_mapping_ou` = (ou_dn, ad_group_id)，**未含 ad_config_id**（P1-10，多域隐患）；ADConfigID 仅普通索引 idx_ou_group_config。硬删模型（无 DeletedAt）。
- `ou_group_mapping_service.go`：ListMappings :62 / CreateMapping :115（isUniqueConstraintError :253-261 字符串匹配，应 pgconn.PgError）/ DeleteMapping :202-213（**不清 AD 组成员**，P1-9）/ CreateSyncLog :232 / UpdateSyncStatus :240-250。
- `dept_sync_tasks.go:85-86`：`NewDeptToADSyncService(db, pool, nil, mapper)` —— ldap 参数恒 nil（P2 死字段）。

### E. 测试先例
- 手写 mock（无 gomock）：`mockLDAPClient`（ldap_client_mock_test.go）每方法 `xxxErr`/`xxxRes` + `xxxCalls` 计数；drain 测试用 `clientFactoryOverride` 注入。
- 相关测试文件：dept_sync_tasks_80_02_test.go、dept_sync_service_test.go、dept_sync_conflict_test.go、group_*_test.go、ou_group_mapping_78_06_test.go、ad_push_queue_drain_test.go。

## 三、修复设计决策（PLAN 依据）
1. **P0-3**：addJob 改 `cron.AddJob + NewChain(Recover(cronLogger()), SkipIfStillRunning(cronLogger())).Then(...)`；闭包内 `context.WithTimeout(context.Background(), constants.JobExecutionTimeout)`；`job.Concurrent==true` 时不加 SkipIfStillRunning（尊重 DB 开关）。新常量 `JobExecutionTimeout = 15 * time.Minute`（小于最小常见周期裕度 + SkipIfStillRunning 双保险）。**一处修复惠及全部 sys_job**。
2. **P0-1**：LDAPClientIface 新增 `GetGroupMembers(groupDN string) ([]string, error)`（SearchWithRequest 实现，读 member 属性，返回 DN 切片）；任务内 diff：影子表期望集 vs AD 实际集 → 多余的 RemoveGroupMember（"32"/No Such Object 容错跳过）、缺失的 AddGroupMembers 批量；每 mapping 写一条 SyncLog（CreateSyncLog）+ UpdateSyncStatus。
3. **P0-2**：department Delete 事务内级联软删 `sys_dept_ou_mapping`（按 dept_id）+ 删除 `sys_ou_group_mapping`（按该部门 ou_dn 前缀）；AD 端 OU/组成员**不自动删**（保守原则：WARN + operlog 记录需人工清理项）。
4. **P1-4**：migration_224（Go 迁移，双方言）seed `dept_to_ad_sync` sys_job 行（cron 0 0 2 * * *，幂等 WHERE NOT EXISTS）。
5. **P1-5/6**：组员同步查询加 `ad_config_id = ?`；LIKE 值转义 %/_；批量 AddGroupMembers；错误码 `ldap.IsErrorWithCode(err, 68/32)`。
6. **P1-7**：单树失败 continue 下树；最终 result.Status = partial/failed 如实上报。
7. **P1-8**：getRootDepartments 改为读全量正常部门（id/parent_id/ancestors/dept_name）内存建树（ ancestors 切分，无层数上限）。
8. **P1-9**：DeleteMapping 前（管理面同步操作）按映射查影子用户 → RemoveGroupMembers 清理 AD 组成员，失败 WARN 不阻断删除。
9. **P1-10**：migration_224 同时 DROP/重建唯一索引为 (ad_config_id, ou_dn, ad_group_id)，模型 tag 同步；pgconn.PgError 判重。
10. **P2**：totalProcessed 改按用户数、nil 池返回 error、删 DeptToADSyncService.ldap 死参数、UpsertMapping 口径、last_sync_at 收敛 UpdateSyncStatus。
