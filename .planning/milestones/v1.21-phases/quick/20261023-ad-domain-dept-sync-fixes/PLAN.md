# PLAN — 域控部门向同步修复

> 依据：同目录 RESEARCH.md。GSD quick 流程：本计划确认后由 Lead 串行实施（T1→T5），每步 `go build ./...` 验证。
> 全局约束：遵循 Handler-Service 模式；不碰 ADPushQueueActivePredicate 红线；不使用裸 c.JSON；asset_columns_schema.json 漂移需 git checkout 还原；go build 慢（>120s，用后台 job）；提交前需用户确认。

## 任务分解

### T1 执行器加固（P0-3）— 作用域：internal/scheduler/cron.go、pkg/constants/timeouts.go
- `pkg/constants/timeouts.go` 新增 `JobExecutionTimeout = 15 * time.Minute`。
- `cron.go` addJob(:314-319)：改 `s.cron.AddJob(job.CronExpression, chain.Then(cron.FuncJob(...)))`；chain = `cron.NewChain(cron.Recover(cronLogger()), 可选 cron.SkipIfStillRunning(cronLogger()))`——`job.Concurrent==false`（默认）才加 SkipIfStillRunning；闭包内 `ctx, cancel := context.WithTimeout(context.Background(), constants.JobExecutionTimeout)`。
- 效果：所有 sys_job（含生产 2 条）获得 panic 恢复、防重叠、超时。
- 验收：`cron_80_01_test.go` 相关测试过；新增测试：Concurrent=false 时 chain 生效（可用注册后 executors 长度/无 panic 冒烟验证）。

### T2 组成员双向同步 + 查询修正（P0-1 / P1-5 / P1-6 / P2-1 / P2-6）— 作用域：ldap_iface.go、ldap_client.go、ldap_client_mock_test.go、dept_sync_tasks.go
- iface+client 新增 `GetGroupMembers(groupDN string) ([]string, error)`：SearchWithRequest(`(&(objectClass=group)(distinguishedName=groupDN))`，attr=member) → 收集 DN 切片；同步补 mockLDAPClient。
- executeDeptMemberToADGroupSyncTask 重构每个 mapping 分支：
  - 查询加 `ad_config_id = ?`；LIKE 值经 `escapeLike(mapping.OUDN)`（转义 %/_）。
  - LDAP 读当前 member → diff：expectedSet(影子表) vs actualSet → 缺失走 `AddGroupMembers` 批量（仍需逐条 68 容错则降级逐个）、多余逐个 `RemoveGroupMember`（"32"/No Such Object 容错跳过）。
  - 错误分类改 `ldap.IsErrorWithCode(err, 68)` / `(err, 32)`，字符串匹配仅作兜底。
  - 每映射写 `OUGroupMappingSyncLog{SyncType:"member_sync", MembersAdded, MembersRemoved, TotalMembers, Status, ErrorMsg, DurationMs}`（CreateSyncLog）；last_sync_at 改 `UpdateSyncStatus`。
  - totalProcessed 改按用户次数统计（P2-1）；mapping 级失败 continue 下一个 mapping。
- 验收：新增/更新测试（mock 注入 clientFactory 模式）覆盖：只加缺失、只删多余、68 容错、日志行生成；`dept_sync_tasks_80_02_test.go` 全过。

### T3 部门级联 + 部门树同步健壮化（P0-2 / P1-7 / P1-8 + 用户补充两项）— 作用域：internal/services/system/department_service.go、internal/services/addomain/dept_sync_service.go、ldap_iface.go、ldap_client.go、ldap_client_mock_test.go、dept_ou_mapper.go
- department Delete(:167-196)：软删部门的事务内，按 dept_id 软删 `sys_dept_ou_mapping`、按其 ou_dn 删除 `sys_ou_group_mapping`（硬删）；事务提交后（用户已确认的新语义）：若该部门有 OU 映射 → 检查 AD OU 下无任何 AD 用户（查 sys_ad_user 该 ou_dn 后缀行数=0，部门本身无用户由现有 Delete 前置校验保证）→ 为空则调新增的 `DeleteOU(ouDN)` 删除 AD OU；非空则跳过删除并 WARN。AD 调用失败 WARN 不阻断本地删除。
- department Update(:73-152)：检测 deptName 变更且存在 OU 映射 → 事务提交后 `RenameOU(oldDN, newName)`（新增 iface 方法，LDAP ModifyDN deleteoldrdn=true，子树随动）→ 成功后更新 `sys_dept_ou_mapping.ad_dn`、`sys_ou_group_mapping.ou_dn`（前缀替换 old→new）、`sys_ad_user.ou_dn/ad_dn` 前缀替换；失败 WARN（漂移可由后续重新生成映射修复）。
- SyncDeptStructureToAD(:81-87)：单树失败 `result.Errors` 记录 + **continue**；全部完成后 `result.Status` 按失败数取 completed/partial/failed，不再恒 return nil 的"假成功"。
- getRootDepartments(:176-184)：去掉固定 3 层 Preload，改为一次查询全部 `status=normal` 部门（id/parent_id/ancestors/dept_name），按 parent_id 内存建树后从根遍历（无层数上限）。
- 验收：dept_sync_conflict_test.go / dept_sync_service_test.go 更新与全过；新增"深层（4+ 层）部门包含在树中"测试。
- getRootDepartments(:176-184)：去掉固定 3 层 Preload，改为一次查询全部 `status=normal` 部门（id/parent_id/ancestors/dept_name），按 parent_id 内存建树后从根遍历（无层数上限）。
- 验收：dept_sync_conflict_test.go / dept_sync_service_test.go 更新与全过；新增"深层（4+ 层）部门包含在树中"测试。

### T4 migration 224 + 唯一键扩展（P1-4 / P1-10 / P2-4 / P2-5）— 作用域：internal/core/db/migrations/migration_224_ad_dept_sync_job.go、database.go、internal/models/ou_group_mapping.go、internal/services/addomain/ou_group_mapping_service.go
- migration_224（Go，双方言，非阻断，HasTable 守卫）：
  ① 幂等 seed sys_job：`dept_to_ad_sync`（'部门到AD同步'，cron `0 0 2 * * *`，concurrent=false，status=0，WHERE NOT EXISTS invoke_target + deleted_at IS NULL）。
  ② 唯一索引扩展：DROP INDEX IF EXISTS uni_ou_group_mapping_ou → CREATE UNIQUE INDEX uni_ou_group_mapping_ou ON sys_ou_group_mapping(ad_config_id, ou_dn, ad_group_id)。
- models/ou_group_mapping.go：唯一索引 tag 加 ad_config_id（priority:0），OU/Group 顺延 priority。
- ou_group_mapping_service.go：isUniqueConstraintError 改 pgconn.PgError（码 23505）+ sqlite 兜底；CreateMapping/Create 相关冲突子句对齐新键（如有 OnConflict 需同步列序）。
- database.go：PG(:1135 区) 与 sqlite(:1205 区) 注册。
- 验收：go build + db 包测试；探针式 SQL 验证可后续在生产观察（非阻断迁移）。

### T5 收尾杂项 + 全量验证（P1-9 / P2-2 / P2-3）— 作用域：ou_group_mapping_service.go、dept_sync_tasks.go、dept_sync_service.go
- DeleteMapping：删前按映射查 sys_ad_user（ad_config_id+ou_dn LIKE 后缀）→ FailoverClient RemoveGroupMembers 清理 AD 组成员，失败 WARN 不阻断删除（管理面低频操作）。
- getGlobalADAccountPool(:21-27) 语义不变，但调用处（executeDeptToADSyncTask :84 / executeDeptMemberToADGroupSyncTask :151）nil 时返回明确 error 而非带 nil 池继续。
- `NewDeptToADSyncService(db, pool, nil, mapper)` 死参数 ldap：删字段或在 dept_sync_tasks.go 传入实际 client 构造（择小改动）。
- 全量验证：`go build ./...`、`go test ./internal/scheduler/... ./internal/services/addomain/ ./internal/core/db/ ./internal/services/system/`、前端无改动不跑 tsc。

## 执行顺序与提交
1. T1 → T2 → T3 → T4 → T5 串行，每步 build+test。
2. 提交拆分建议：`fix: sys_job 执行器加固（Recover/SkipIfStillRunning/超时）`（T1）、`feat: 组成员双向同步与日志落地`（T2）、`fix: 部门删除级联与部门树同步健壮化`（T3）、`feat: migration_224 部门同步 seed 与 OU 组映射唯一键扩展`（T4+T5 合并或分拆）。提交前逐个征询用户。
3. 部署后观察项：生产 sys_ou_group_mapping_sync_log 出现记录；三农承保理赔部映射若 DN 确不存在应出现在日志 failed 记录中（或需用户修正映射）。

## 明确不做（本轮范围外）
- 不做部门"移动"（parent 变更）的 AD OU 移动（仅改名 RenameOU 已覆盖用户需求）。
- 不动用户向链路（update_user/update_manager）与影子清理双闸门。
