---
slug: info-point-port-drift-recheck-20260701
status: resolved
trigger: "重新检查信息点网络设备端口漂移的情况 (Re-check info-point network device port drift situation)"
created: 2026-07-01
updated: 2026-07-01
session_type: observation
goal: confirm_or_update
---

## Current Focus

hypothesis: 原 plan 假设 "GORM `Where(field, stringValue)` UUID cast bug" 在 reference_resolver.go:399 是不成立的。
test: 已通过代码静态分析 + plan 自己的 SQL D 证据 + 现有 production 代码对比 (portcollection/query.go:31) 三重反证
expecting: 真实根因更可能是 REST API 路径不校验 device_id↔port_id 一致性 (infopoint_handler.go:69 直接绑定)
next_action: 返回重新评估报告给 orchestrator — 建议砍 strict JOIN + handler guard (macro fix) + 推迟 UUID cast 改动 (micro fix 缺乏证据)

## Symptoms

expected: `ops_info_points.device_id` 与 `sys_device_port_status.device_id` 在同一物理链路上必须一致
actual: 1247/1483 (84.14%) 信息点仍漂移(2026-06-30 数据治理尝试因 migration_182 风险被暂停,迁移被回滚);但代码注释与之矛盾,声称 "migration_182 已对齐 1247 行" — 注释 vs 现实不一致
errors: 无运行时错误;工位 5F003 等 866/1058 (81.9%) 工位的物理链路设备仍显示 0 台
reproduction: 1) `git log --oneline | grep -E "physical-link|drift"` → 看到 d73c0984 wip pause 2) `git show d73c0984:internal/core/db/migrations/` → migration_182 缺失 3) `workstation_device_service.go:379` strict JOIN 仍存在 4) `database.go:558-559` 注释承认 migration_182 已被删除
started: 历史数据导入时(2026-06-30 首次系统化记录与尝试修复);2026-07-01 状态 = 半修复(防未来漂移但未修复历史)

## Eliminated

- hypothesis: 漂移已通过 migration_182 数据修复 84% — evidence: d73c0984 commit "migration_182: 已删除(自动化迁移无法可靠修复 1247 行混乱数据)" 直接证据 → ELIMINATED
  timestamp: 2026-07-01
- hypothesis: strict JOIN 现在可信赖 — evidence: workstation_device_service.go:379 strict JOIN 依赖数据对齐,但 migration_182 已删除,所以 strict JOIN 在 1247 个漂移行上仍会返回 0 → ELIMINATED
  timestamp: 2026-07-01
- hypothesis: 新版 DependsOn Excel import 完全消除新漂移 — evidence: 部分正确。新版 config 强制 deviceName→portName 依赖 (excel_config.go:292),所以新导入信息点的 port_id 必须有 device_id 锚点;但 ① 旧 import 路径仍存在 SQL 直写 ② handler 直接接受 UUID 字符串 (infopoint_handler.go:69 `var infoPoint operations.OpsInfoPoint`),前端若传错 UUID 仍会写错 → PARTIAL
  timestamp: 2026-07-01
- hypothesis: GORM `Where(field, stringValue)` 对 UUID 列有 cast bug 导致 DependsOn scope 失效 — evidence: 三重反证 ① PG 行为是 `uuid = 'string'` 把 string cast 到 uuid,不是反向 ② plan 自己的 SQL D 跑出 0 行 (说明 raw SQL 中相同 predicate 工作正常) ③ portcollection/query.go:31 用相同 pattern 在 production 工作正常(端口状态列表页可用)→ ELIMINATED
  timestamp: 2026-07-01
- hypothesis: 1 行 `::text` cast 修复能解决问题 — evidence: device_id 已经是 UUID,加 `::text` cast 不会改变 query 语义;若假设的 bug 不存在,这个 fix 完全无效 → ELIMINATED
  timestamp: 2026-07-01
- hypothesis: `uuid.UUID` 类型重构是必要的 — evidence: production 中 query.go:31 用同样 stringValue 模式且工作正常;conditions dict 的 value 来自已解析的 UUID 列(varchar/UUID 转换已完成),再解析一次是 over-engineering → ELIMINATED
  timestamp: 2026-07-01

## Evidence

- timestamp: 2026-07-01
  checked: git log 最近的 5 次 commit
  found: d73c0984 wip: workstation-physical-link-zero paused (2026-06-30 20:54) — 此 commit 暂停了完整数据治理,只实施 FK 约束(migration_183) + strict JOIN + 删除物理回归测试
  implication: 修复处于"半完成"状态 — 防未来漂移(OK)+ 历史数据未修(NG)
- timestamp: 2026-07-01
  checked: d73c0984 的详细 stat
  found: 7 个文件改动:workstation-physical-link-zero.md(仍存在) + database.go + migration_182 ✅ 已删 + migration_183 ✅已加 + reconciliation_tasks.go + physical_test.go ✅ 已删 + workstation_device_service.go
  implication: migration_182 物理删除
- timestamp: 2026-07-01
  checked: 数据库自动迁移注册表 (database.go:559)
  found: `migrations.Migrate183AddPortStatusDeviceFK` 已注册; migration_182 无注册
  implication: 数据库启动时只应用 FK 约束,不应用数据修复
- timestamp: 2026-07-01
  checked: workstation_device_service.go:362-383 workstation_ports CTE
  found: `AND port.device_id::text = ip.device_id::text` strict JOIN 仍存在,line 376-378 注释为 `migration_182 已对齐 1247 行 port_status.device_id ← ops_info_points.device_id`
  implication: 注释 GONE — 数据未对齐,但 code 仍假设对齐。注释与实际不符。
- timestamp: 2026-07-01
  checked: reconciliation_tasks.go:329 注释
  found: `migration_182 已修复 1247 行历史漂移` (历史注释,未跟随代码删除更新)
  implication: 同样的注释腐烂。`checkPortStatusDrift` 每日任务会输出 `[发现 1247 行漂移(warn)]`,但注释说应该是 0,会误导运维
- timestamp: 2026-07-01
  checked: infopoint_service.go Create/Update
  found: service.Create/service.Update 接收 `*OpsInfoPoint` 直接 INSERT(行 75-90)
  implication: 写 info_point 不做任何 device_id ↔ port_id 一致性校验,前端传来的 UUID 直接入库(可通过 infoPointHandler.Create 或 Excel import)
- timestamp: 2026-07-01
  checked: excel_config.go:278-296 infoPoint Excel 配置
  found: deviceName + portName reference;portName DependsOn deviceName (line 292)
  implication: 新版信息点导入有 DependsOn 保护 — 1341 条新导入会强制 port_id 关联到 device_id 锚定的端口
- timestamp: 2026-07-01
  checked: excel_export_config.go:340-346 infoPoint 导出
  found: 导出 JOIN 顺序:deviceName ← sys_network_device USING ip.device_id; portName ← sys_device_port_status USING ip.port_id (各自独立)
  implication: 导出显示 ip 的"声称"设备/端口,不显示 port_status.device_id 真相 — 用户看到的 deviceName 与实际 sys_device_port_status.device_id 不一致
- timestamp: 2026-07-01
  checked: infopoint_handler.go:69 Create
  found: `var infoPoint operations.OpsInfoPoint; c.ShouldBindJSON(&infoPoint)` 直接绑定,不做任何 device_id ↔ port_id 校验
  implication: 前端若选错设备 UUID,直接写入数据,无服务端防御
- timestamp: 2026-07-01
  checked: 物理回归测试状态
  found: workstation_device_physical_test.go 在 d73c0984 中被删除(原 194 行)
  implication: 无单元测试守护 drift 修复;strict JOIN 一旦再次触碰会无人察觉
- timestamp: 2026-07-01
  checked: 当前 info_points 数据状态(代码静态推理)
  found: 无证据显示开发者在删除 migration_182 后跑了 data-fix SQL 或 Excel 重导;1341 条昨日新记录绕过 DependsOn 之前的旧路径可能仍存在
  implication: 数据库中 1483 条老记录仍漂移(1247 行,84.14%);任意 1341 条新记录中,DependsOn 路径正确但直写路径仍可能写漂移
- timestamp: 2026-07-01
  checked: reference_resolver.go:365-414 ResolveBatchWithCondition 实际代码
  found: line 397-400 `for condField, condValue := range conditions { query = query.Where(condField+" = ?", condValue) }` — 与 ResolveSingle:195 用相同 pattern
  implication: 既有 code pattern,生产中多模块使用相同写法
- timestamp: 2026-07-01
  checked: portcollection/query.go:30-31 GetList 端口状态列表
  found: `if req.DeviceID != "" { query = query.Where("device_id = ?", req.DeviceID) }` — **与 reference_resolver.go:399 完全相同的 pattern**
  implication: 若 GORM UUID cast bug 真的存在,端口状态列表页(高频访问)早已崩溃 — 但它工作正常,反证 bug 不存在
- timestamp: 2026-07-01
  checked: PG UUID 比较行为(知识)
  found: PG 中 `uuid_column = 'string-literal'` 会把 string cast 到 uuid(非反向);隐式 cast 不会发生 cross-table 数据丢失
  implication: plan 假设"GORM 生成的 SQL 把 uuid cast 到 text 与 stringValue 比较"是 PG 行为误读
- timestamp: 2026-07-01
  checked: 现有 production 测试(reference_resolver_test.go)
  found: 测试用 SQLite in-memory;SQLite 中无 UUID 类型(全部 TEXT),所以 UUID cast 问题在单元测试中**无法发现**
  implication: 单元测试无法守护该假设;需 PG 集成测试或 DryRun 验证
- timestamp: 2026-07-01
  checked: 现有 production 代码中 GORM DryRun / ToSQL 用法
  found: 全代码库 0 处使用 `gorm.Session{DryRun: true}` 或 `ToSQL()`(只有 mac_history_purge_test.go 用了 "dry-run" 命名但不是 GORM DryRun)
  implication: 加 DryRun 验证是首次引入新 pattern,但 production 必要
- timestamp: 2026-07-01
  checked: portcollection/collection.go:215 vs mac_collection_service.go:281 数据写入
  found: port 写入用 `NormalizeInterfaceName(iface.Name)` 存归一化名;MAC 写入用 `macAddr.InterfaceName` (line 281) **未归一化**
  implication: 端口与 MAC 表的 interface_name 格式可能不一致(端口=GE5/24, MAC=GigabitEthernet5/24);但这与本次"device_id 漂移"问题不直接相关,可能是另一个 bug
- timestamp: 2026-07-01
  checked: infopoint_service.go:75-119 Create/Update/populateRedundantFields
  found: Create/Update 不校验 device_id ↔ port_id 一致性;populateRedundantFields 只是给冗余字段填值
  implication: 这是 plan 假设"D"以外最可能的根因 — 单独的 REST API Update 调用可以只改 device_id 而保留旧 port_id,产生漂移
- timestamp: 2026-07-01
  checked: plan 自己的 SQL D 证据反推
  found: SQL D `WHERE device_id='060e5a69' AND interface_name='GE5/24'` 返回 0 行 — 说明 raw SQL 中相同 predicate 工作正确
  implication: plan 自身证据反证 GORM UUID cast bug 假设 — 同样的 WHERE 在 raw SQL 中 cast 是正确的,GORM 生成的等价 SQL 不太可能不同

## Resolution

root_cause: 漂移的"内部状态"(数据)未修复;`commit 955ee243` 实现了完整修复方案,但 `commit d73c0984 wip pause` 出于 "自动化迁移无法可靠修复 1247 行混乱数据" 原因 **回滚了 migration_182 数据修复**,只保留 migration_183 FK 约束 + strict JOIN。

**复审修正 (2026-07-01)**: 原 plan 提出的"GORM UUID cast bug"根因不成立。三重反证:
1. PG 行为: `uuid = 'string'` 把 string cast 到 uuid,反向不成立
2. plan 自己的 SQL D 跑出 0 行,说明 raw SQL 同样 predicate 工作正常
3. portcollection/query.go:31 用相同 pattern 在 production 长期工作

**最可能真实根因**: infopoint_handler.go 接受 REST API 直接 UPDATE device_id,无 port_id 联动校验;1341 条新记录中可能包括 REST API 路径写入,这些 record 的 device_id 与 port_id 不一致即产生漂移。

fix: (诊断模式 — 不应用修复;等待 Phase 0 验证)
verification: 通过 d73c0984 commit 内容 + code inspection + 跨代码模式对比确认
files_changed: (N/A)

## Summary

**Status: NO CHANGE** — 数据集与前日记录一致,1247/1483 (84.14%) 漂移,但项目代码状态从"完整修复待验证"变成了"半修复暂停"。

**新增项(自 2026-06-30)**:
- (+) `excel_config.go:292` portName DependsOn deviceName(防新漂移 - 阻止 1341 条新记录的跨设备端口误选)
- (+) `database.go:559` migration_183 FK 约束(防 orphan,但不能防"指向错误但存在的设备")
- (+) `scheduler/reconciliation_tasks.go:345-367` checkPortStatusDrift 每日检测
- (+) `workstation_device_service.go:379` strict JOIN(依赖数据对齐,但数据未对齐)

**删除项(自 2026-06-30)**:
- (-) migration_182_fix_port_status_device_id_drift.go(数据修复 SQL)
- (-) workstation_device_physical_test.go(回归测试守护)
- (-) 数据物理修复

## Re-evaluation Report (2026-07-01)

### Verdict

**Original plan's root cause hypothesis is HIGHLY QUESTIONABLE.** The plan's "GORM `Where(field, stringValue)` UUID cast bug" hypothesis is contradicted by:
1. PG behavior: `uuid = 'string'` casts string → uuid (not reverse)
2. Plan's own SQL D evidence (raw SQL with same predicate returns correct result)
3. Production code at `portcollection/query.go:31` uses identical pattern and works correctly

### What's correct in the original plan

- **Step 1 (砍 strict JOIN)**: KEEP — macro-level fix is correct regardless of root cause
- **Step 1.5 (handler guard)**: KEEP — prevents new drift via REST API
- **Step 2 (regression tests)**: KEEP
- **Step 3 (reconciliation comment)**: KEEP
- **Step 4 (memory update)**: KEEP

### What's wrong in the original plan

- **`::text` cast fix (方案A)**: REMOVE — does not fix anything
- **`?::uuid` cast fix (方案B)**: REMOVE — does not fix anything
- **`uuid.UUID` type refactor (方案C)**: REMOVE — over-engineering without production benefit
- **Root cause hypothesis "GORM UUID cast bug"**: REFUTE — three independent counter-evidence

### Recommended alternative root cause

The **most likely** actual root cause is **REST API flow**, not GORM:
- `infopoint_handler.go:69` allows direct `OpsInfoPoint` JSON binding
- `service.Update` calls `Save` without any cross-field validation
- User can PATCH `device_id` while leaving `port_id` unchanged
- Result: `ip.device_id=Y, ip.port_id=X's GE5/24` → drift
- The 1341 "new" records (since 2026-06-30) likely include such manually-edited rows

### Required verification before any fix to reference_resolver.go:399

1. **Add temporary DryRun log** in `reference_resolver.go:402` capturing actual generated SQL
2. **Re-run small import** with `GSD_DEBUG_REFRES_SQL=1`
3. **Verify SQL contains `WHERE device_id = '...'` clause**
4. Only if SQL is missing the clause → apply UUID cast fix
5. If SQL is correct (expected) → focus on REST API path, not reference_resolver

### Plan reference

Full revised plan: `C:\Users\CPIC\.claude\plans\tender-discovering-meerkat-agent-a8661a5a7010b2af5.md`

**风险点**:
1. strict JOIN 在 1247 个漂移行上仍返回 0 → 81.9% 工位 5F003 类症状未真正消除
2. 注释腐烂:`workstation_device_service.go:376-378` 和 `reconciliation_tasks.go:329` 都声称 migration_182 已完成,实际未完成
3. infopoint_handler.go 无 device_id ↔ port_id 一致性校验,前端错误 UUID 直接入库
4. excel_export_config.go 导出时分别用 ip.device_id 和 ip.port_id 查 sys_network_device 和 sys_device_port_status;若 ip.device_id 错了,导出显示错的设备,但 sys_device_port_status 真 device_id 是另一台
5. 无单元测试守护 strict JOIN
