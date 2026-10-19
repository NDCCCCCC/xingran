---
phase: 58-compare-and-move
plan: 04
subsystem: addomain / ad_push_queue
tags: [go, ldap, drainer, compare-and-move, f-a, f-b, ou-migration, phase-58]

# Dependency graph
requires:
  - phase: 58-compare-and-move
    plan: 03
    provides: "ADPushDrainer 排水执行面（drainRow 无返回值 + cfgID 已入签名 + `:370` 的 compare-and-move 接口位 + newDrainTestDrainer 注入 helper）"
  - phase: 58-compare-and-move
    plan: 02
    provides: "NormalizeDN（attr 名与值双侧 ToLower）/ MatchADConfigByDN（D-02 最长后缀唯一匹配）"
  - phase: 58-compare-and-move
    plan: 01
    provides: "排水测试基座（setupDrainTestDB / seedDrainUser / seedDrainQueueRow / seedDrainADConfig / mockLDAPClient 计数）"
  - phase: 57-ad-outbox-lite
    provides: "ADPushQueue 模型 + syncUserAttributes（DNExists 预检 + code 32 清 ad_dn）+ clearUserADRefs 语义"
provides:
  - "addomain.DeptOUmapper.FindOUDNByDeptIDForConfig —— 配置作用域 OU 查询（F-B）"
  - "addomain.ADPushDrainer.compareAndMoveOU —— QUEUE-04 幂等 OU 移动执行面，返回 (terminal bool, err error)"
  - "mockLDAPClient.updateUserAttrLastAttrs —— 属性载荷快照（闭包内重读最新值的行为证据）"
  - "F-A 三处载体的 ad_dn 回写修复（drainer / moveUserToNewOU / moveSingleUserToOU）"
affects: [58-05, 58-06, 58-07]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "compare-and-move：闭包内重读最新值 → NormalizeDN 双侧归一比较 → 一致跳过 LDAP 写（幂等点）→ 不一致才移动 + 双字段回写"
    - "terminal 返回值契约：写终态的分支返回 (true, nil)，调用方必须短路 markPushed —— 防「终态被成功路径覆盖」"
    - "F-A 双写模式：ModifyDN 成功后 Updates(map){ad_dn: extractRDNFromDN(旧 DN)+\",\"+新 OU, ad_ou_dn: 新 OU} 且必须 Unscoped()"
    - "配置作用域查询：凡(OUI 映射)按 (dept_id, ad_config_id) 唯一键过滤，禁跨配置 First() 取首行"
    - "有意收紧相对旧路径的宽容语义时必须写进注释 + 独立用例断言「不是 pushed」"

key-files:
  created: []
  modified:
    - internal/services/addomain/dept_ou_mapper.go
    - internal/services/addomain/dept_ou_mapper_test.go
    - internal/services/addomain/ad_push_queue_drain_service.go
    - internal/services/addomain/ad_push_queue_drain_test.go
    - internal/services/addomain/user_ad_sync_service.go
    - internal/services/addomain/ldap_client_mock_test.go

key-decisions:
  - "Plan 03 的 TestPhase58_Drain_UpdateUserSuccessWritesPushedAt fixture 补一行 OU 映射铺底（叠加 compare-and-move 后有 dept_id 而无映射按 Q2 转 dead，原 fixture 语义失效）；补的是「与用户当前 ad_ou_dn 归一化相等」的映射，断言强度未降低"
  - "Task 1 的三个 FindOUDNByDeptIDForConfig 用例放在 dept_ou_mapper_test.go（不带 skip_db_tests tag）—— 排水基座的 setupDrainTestDB / drainNow 都在带 !skip_db_tests 的文件里，mapper 用例引用它们会让 -tags skip_db_tests 构建红"
  - "compareAndMoveOU 的签名与返回值严格按 plan 终稿（含 terminal 返回值）；未退回单 error 返回 —— 退回会让 TestPhase58_Drain_NoOUMappingGoesDead 必红"
  - "旧 FindOUDNByDeptID 保留不动（调用方本轮只修 F-A 回写、不改触发条件）；排水器内裸 FindOUDNByDeptID(ctx 命中数为 0（F-B 防线）"
  - "不标记 QUEUE-04 / QUEUE-05 为 complete —— 沿用 58-01/58-03 分档口径（cron 接线在 58-05、端到端证据在 58-07），最终回填归 58-07"
  - "plan 的「sed 区间内必须含 return 且不含 markPushed」断言文字自相矛盾（区间终点即 markPushed 那一行）；以语义等价区间 [if terminal 行, markPushed 行) 核对，见 Deviations #1"

requirements-completed: []

# Metrics
duration: 30min
completed: 2026-10-19
---

# Phase 58 Plan 04: compare-and-move + F-A 双写 + F-B 配置收敛 Summary

**QUEUE-04 的 compare-and-move 执行面落地：闭包内重读最新值 → NormalizeDN 双侧归一化比较目标 OU → 一致跳过 `MoveUser`（幂等点）→ 不一致才移动并**同时**回写 `ad_dn` 与 `ad_ou_dn`（F-A 阻断级，三处载体一并修复）；目标 OU 改走配置作用域的 `FindOUDNByDeptIDForConfig`（F-B）；补齐 OU 路径 code 32 转 dead 且终态在 `markPushed` 之前短路；28 个 Phase 58 用例全绿、全仓 `go test ./...` exit 0、Phase 57 sentinel 绿**

## Performance

- **Duration:** ~30 min
- **Started:** 2026-10-19T10:20Z（近似）
- **Completed:** 2026-10-19T10:45Z（近似）
- **Tasks:** 3/3
- **Files modified:** 6（0 新建 + 6 增量修改）

## 交付契约（下游 58-05/58-06/58-07 直接消费）

```go
// internal/services/addomain/dept_ou_mapper.go
func (m *DeptOUmapper) FindOUDNByDeptIDForConfig(ctx context.Context, deptID, adConfigID string) (string, error)

// internal/services/addomain/ad_push_queue_drain_service.go
func (d *ADPushDrainer) compareAndMoveOU(ctx context.Context, c LDAPClientIface, row models.ADPushQueue,
    user *models.User, cfgID string, now time.Time) (terminal bool, err error)
```

### `compareAndMoveOU` 全文（源码逐字，`ad_push_queue_drain_service.go:394-468`）

```go
// compareAndMoveOU OU 移动的幂等执行面（Phase 58 QUEUE-04）。
//
// 三步，全部基于**闭包内重读到的最新值**（不是入队时刻的旧值 —— 禁新旧值比较，
// 那会激活 MoveUser 的未验证重试分支）：
//  1. 取目标 OU：FindOUDNByDeptIDForConfig(ctx, *user.DeptID, cfgID)（F-B）
//     无映射 → 不可恢复，dead（Q2：需运营动作「跑部门同步」，重试无意义）
//  2. 归一化比较：当前 ad_ou_dn 与目标 OU **双侧**过 NormalizeDN 后相等 → 直接返回
//     （幂等点。对同一 DN 重复 MoveUser 会 code 32，见 ldap_client.go:350-361，
//      ModifyDN 的 deleteOldRDN=true 会先删掉旧 RDN）
//  3. 不一致 → MoveUser(*user.AdDn, targetOUDN)；成功后**同时**回写
//     ad_dn = extractRDNFromDN(*user.AdDn) + "," + targetOUDN 与 ad_ou_dn = targetOUDN
//     （F-A 阻断级：ModifyDN 后 AD 侧 DN 已变，ad_dn 不回写会让 D-02 下一轮按陈旧 DN
//     解析配置，跨域移动后落到另一个 base_dn → 误判「配置无匹配」→ 误转 dead；
//     属性 Modify 也会打到旧 DN → 二次 code 32）
//
// ⚠ 与 SyncUserUpdateToAD 的语义差异（有意收紧）：后者 OU 移动失败仅 ERROR 日志并继续推属性；
// 本函数把 OU 失败上抛给 handlePushError → 行转 failed 并退避重试。理由：否则"OU 永远移不动"
// 的行会被属性成功掩盖成 pushed，SC2「重复排水幂等、失败可查」的可观测性就没了。
//
// ⚠ 返回值契约（terminal 是本函数存在的理由，**不得**退回单 error 返回）：
//
//	(false, nil) = 本函数未对行做任何终态写 → 调用方**可以** markPushed
//	(true,  nil) = 本函数已把行写成终态（当前仅 no-ou-mapping → dead）→ 调用方**必须**跳过 markPushed
//	(false, err) = 本函数失败 → 调用方交给 handlePushError 分流
//
// 把 (true, nil) 当成 (false, nil) 处理会让 dead 被 markPushed 覆盖成 pushed，
// TestPhase58_Drain_NoOUMappingGoesDead 必红，且生产上"需要重跑部门同步"被误报为成功。
func (d *ADPushDrainer) compareAndMoveOU(ctx context.Context, c LDAPClientIface, row models.ADPushQueue, user *models.User, cfgID string, now time.Time) (terminal bool, err error) {
	// 1) 用户无部门 → 无 OU 概念：既不是失败也不是终态，调用方照常 markPushed。
	if user.DeptID == nil || *user.DeptID == "" {
		return false, nil
	}

	// 2) 目标 OU 按 AD 配置收敛（F-B）。查不到 → 不可恢复：映射不会自己出现，
	//    重试只会无限堆积 pending；转 dead 并由 last_error 前缀引导管理员跑部门同步。
	targetOUDN, mapErr := d.mapper.FindOUDNByDeptIDForConfig(ctx, *user.DeptID, cfgID)
	if mapErr != nil {
		// ⚠ terminal=true：本函数已经写了终态，调用方不得再 markPushed。
		// ⚠ 第三个实参是 row.Attempts（不可恢复不烧 attempts，与 markDead 契约一致）。
		d.markDead(ctx, row.ID, now, row.Attempts, "unrecoverable:no-ou-mapping")
		return true, nil
	}

	// 3) 幂等点：当前 OU 与目标 OU 归一化后相等 → 不调 MoveUser。
	//    ⚠ 必须走 NormalizeDN（attr 名与值双侧 ToLower + 去空格），裸比较会把
	//    `OU=IT,DC=corp,DC=com` 与 `ou=it , dc=corp , dc=com` 判为不等 →
	//    对已搬移的源 DN 再调一次 ModifyDN → code 32。
	if user.AdOuDn != nil && NormalizeDN(*user.AdOuDn) == NormalizeDN(targetOUDN) {
		return false, nil
	}

	// 防御：drainRow 已判过 ad_dn 非空；此处再兜一层，避免 nil 解引用 panic（P1：闭包内不得 panic）。
	if user.AdDn == nil || *user.AdDn == "" {
		return false, nil
	}

	// 4) 不一致才移动。%w 包装让 handlePushError 的 ldap.IsErrorWithCode 能穿透判定 code 32。
	if moveErr := c.MoveUser(*user.AdDn, targetOUDN); moveErr != nil {
		return false, fmt.Errorf("移动用户到新 OU 失败 [%s → %s]: %w", *user.AdDn, targetOUDN, moveErr)
	}

	// 5) F-A：同时回写 ad_dn 与 ad_ou_dn。
	//    RDN 用同包 extractRDNFromDN 取（MoveUser 内部用的也是它，保证拼接一致）。
	//    ⚠ 必须 Unscoped()（disable 场景用户已软删，作用域 Updated 会命中 0 行且不报错）。
	//    ⚠ 用 Updates(map[...]) 而非 Save（Phase 57 F-06 收窄写路径的同款决策：显式字段）。
	//    回写失败仅 Warn 不上抛 —— MoveUser 已成功、AD 侧状态正确；本地回写失败下轮会重算。
	newAdDn := fmt.Sprintf("%s,%s", extractRDNFromDN(*user.AdDn), targetOUDN)
	if err := d.db.WithContext(ctx).Unscoped().Model(&models.User{}).Where("id = ?", user.ID).
		Updates(map[string]interface{}{"ad_dn": newAdDn, "ad_ou_dn": targetOUDN}).Error; err != nil {
		applogger.Warnf("[ad_push_queue] 回写用户 %s 的 ad_dn/ad_ou_dn 失败 [%s]: %v", user.ID, newAdDn, err)
	}

	// 终态由调用方的 markPushed 写，本函数不重复写。
	return false, nil
}
```

### `drainRow` 的 update_user 分支终稿（`:354-387`）

```go
	case models.ADPushActionUpdateUser:
		// 闭包内重读到的最新值（不依赖入队时刻的旧值 —— compare-and-move 的前提）。
		updateReq := buildDrainUpdateReq(user)
		// 属性推送（复用 syncUserAttributes：DNExists 预检 + code 32 清 ad_dn 已封装）。
		if err := d.adSync.syncUserAttributes(ctx, c, user, updateReq); err != nil {
			d.handlePushError(ctx, row, now, err, "update_user")
			return
		}
		// ⚠ syncUserAttributes 在 DNExists=false 时清空 ad_dn 后 **return nil**（不报错）。
		//   故必须重新 loadUserUnscoped 判空：目标在 AD 端不存在时终态是 dead 而不是
		//   pushed（QUEUE-05）。缺这一步 → DNExists=false 用例会把"AD 端不存在"误报成功。
		if refreshed, err := d.loadUserUnscoped(ctx, row.EntityID); err == nil {
			if refreshed.AdDn == nil || *refreshed.AdDn == "" {
				d.markDead(ctx, row.ID, now, row.Attempts, "unrecoverable:target-not-found") // 不烧 attempts
				return
			}
			user = refreshed
		}

		// compare-and-move（QUEUE-04）：目标 OU 归一化一致则跳过 MoveUser，重复排水幂等。
		terminal, err := d.compareAndMoveOU(ctx, c, row, user, cfgID, now)
		if err != nil {
			// ⚠ 有意相对 SyncUserUpdateToAD 收紧：后者 OU 移动失败仅 ERROR 日志并继续，
			//   这里上抛 → failed + 退避。否则"OU 永远移不动"的行会被属性成功掩盖成 pushed。
			d.handlePushError(ctx, row, now, err, "move_ou")
			return
		}
		// ⚠ 终态不覆盖不变量：行已被 compareAndMoveOU 写成终态（no-ou-mapping → dead）时
		//   **必须**在此处短路返回，绝不能落到下面的 markPushed —— 那会把 dead 覆盖成
		//   pushed（生产上"需要重跑部门同步"被误报成功 + 终态不可查）。
		if terminal {
			return
		}
		d.markPushed(ctx, row.ID, now)
```

## Q2 / Q3 决策的最终落地位置与可见后果

| 决策 | 落地点 | 可见后果（必须让运维知悉） |
|------|--------|--------------------------|
| **Q2：目标 OU 映射缺失 → dead（非重试）** | `compareAndMoveOU:430-435` → `markDead(..., row.Attempts, "unrecoverable:no-ou-mapping")`，返回 `(true, nil)` | **部门同步未跑过的环境下，`update_user` 行会批量转 dead**（`last_error = unrecoverable:no-ou-mapping`）。需管理员跑一次部门同步（生成 `sys_dept_ou_mapping`）后**重新编辑用户**触发新入队。理由：映射不会自己出现，留 pending 会无限堆积。排障 SQL：`SELECT entity_id FROM ad_push_queue WHERE status='dead' AND last_error='unrecoverable:no-ou-mapping'` |
| **Q3(i)：全局零启用配置 → defer** | `groupByResolvedConfig`（58-03 既有，本 plan 未动） | 不动任何行，下轮仍可拾取 |
| **Q3(ii)：单用户 `ad_dn` 匹配不到配置 → dead** | `groupByResolvedConfig`（58-03 既有） | `unrecoverable:no-ad-config` |
| **用户无 `dept_id`** | `compareAndMoveOU:423-425` → `return false, nil` | 不是失败也不是终态：跳过移动、照常 `pushed`（`TestPhase58_Drain_NoDeptSkipsMove`） |

`last_error` 前缀在本 plan 新增一类（累计可 grep 分类）：

| 前缀 | 语义 | 产出点 |
|------|------|--------|
| `unrecoverable:no-ou-mapping` | 有 `dept_id` 但该配置下无有效 OU 映射（Q2） | `compareAndMoveOU` |
| `retry:move_ou:<err>` | OU 移动失败但属性已推成功 → 可重试退避（有意收紧） | `drainRow` → `handlePushError` |
| `unrecoverable:target-not-found` | OU 路径 code 32（与属性路径同款终态） | `handlePushError` 第 2 分支 |

## 相对 `SyncUserUpdateToAD` 的语义收紧差异（有意，须运维知悉）

| 维度 | `SyncUserUpdateToAD`（旧单用户编辑路径） | 本 plan 的排水器 |
|------|----------------------------------------|-----------------|
| OU 移动失败 | `applogger.Errorf` 后**继续推属性**（`user_ad_sync_service.go:77-81`），整次调用仍视为成功 | 上抛 → `handlePushError(..., "move_ou")` → 行转 `failed` + 退避重试 |
| 后果 | "OU 永远移不动"的行被属性成功掩盖成成功 | 行状态可观测（`failed` + `last_error` 前缀），不会被掩盖成 `pushed` |
| 理由 | —— | SC2「重复排水幂等、失败可查」的可观测性前提 |

## F-A 三处修复的行号

| # | 文件 | 位置 | 修复内容 |
|---|------|------|----------|
| 1 | `internal/services/addomain/ad_push_queue_drain_service.go` | `:460-464`（`compareAndMoveOU` 第 5 步） | `Updates(map){ad_dn: extractRDNFromDN(*user.AdDn)+","+targetOUDN, ad_ou_dn: targetOUDN}` + `Unscoped()` |
| 2 | `internal/services/addomain/user_ad_sync_service.go` | `:129-136`（`moveUserToNewOU`） | 由 `Update("ad_ou_dn", ouDN)` 改为同款双字段 `Updates(map)` |
| 3 | `internal/services/addomain/user_ad_sync_service.go` | `:325-331`（`moveSingleUserToOU`） | 同上 |

三处载体语义一致；`grep -c 'Update("ad_ou_dn", ouDN)' internal/services/addomain/*.go` == 0（无残留单字段回写）。

## Plan 06 删除旧路径时需要一并清理的调用点清单

Plan 06（D-01/D-03，SC6）删除两条直推路径时，除 CONTEXT 已列的删除目标外，本 plan 额外确认以下由旧路径带出的调用点：

| 调用点 | 文件:行 | 与旧路径的关系 |
|--------|---------|---------------|
| `h.userADSyncService.SyncUserUpdateToAD(ctx, id, adSyncReq)` | `internal/api/v1/system/user_handler.go:215` | **D-03 删除目标本身**（单用户编辑直推） |
| `s.moveUserToNewOU(ctx, ldapClient, userID, newDeptIDStr)` | `internal/services/addomain/user_ad_sync_service.go:77` | `SyncUserUpdateToAD` 闭包内调用；D-03 删 `SyncUserUpdateToAD` 后成为死代码（本 plan 已修 F-A，但**不要**在 06 之前删） |
| `s.moveSingleUserToOU(ctx, ldapClient, userID, ouDN)` | `internal/services/addomain/user_ad_sync_service.go:281` | 仅被 `BatchMoveUsersToNewOU` 调用；`BatchMoveUsersToNewOU` 当前**无生产调用方**（grep 仅命中定义处）→ Plan 06 可一并评估删除 |
| `h.userADSyncService.BatchSyncUsersToAD(...)` / `SyncManagersToAD(...)` | `internal/services/addomain/user_ad_sync_service.go:694,762` 附近注释指向 | 与 `SyncUserUpdateToAD` 的"OU 移动交给单用户编辑路径"注释耦合；删旧路径时须同步改注释，否则留下指向已删函数的说明 |
| `adSyncInFlight` / `buildADSyncMap` | `internal/api/v1/system/user_handler.go:28-32 / :245-260` | CONTEXT 已列，58-01 F-G 已删 `TestUserHandler_BuildADSyncMap`（测试侧就绪） |

⚠ **`moveUserToNewOU` / `moveSingleUserToOU` 本 plan 必须保留**：Plan 06 之前它们仍是旧路径的组成部分，删除属 06 的范围（本 plan 只修 F-A 回写缺陷，不动触发条件）。排水器侧**不依赖**这两个函数（`compareAndMoveOU` 是独立实现）。

## Task Commits

Each task was committed atomically:

1. **Task 1: F-B 配置作用域 OU 查询（FindOUDNByDeptIDForConfig）** — `44b4b958` (feat)
2. **Task 2: compare-and-move OU 分支 + F-A 双写 + OU 路径终态** — `f00f41b1` (feat)
3. **Task 3: user_ad_sync_service.go 两处 F-A 缺陷修复** — `c89b4835` (fix)

## 验收断言实测（全部通过）

### Task 1

| 断言 | 结果 |
|------|------|
| `go build ./...` | exit 0 |
| `go test ./internal/services/addomain/ -run "TestPhase58_FindOUDNByDeptIDForConfig" -v -count=1` | 3/3 PASS |
| `grep -c "func (m \*DeptOUmapper) FindOUDNByDeptIDForConfig(ctx context.Context, deptID, adConfigID string) (string, error)" dept_ou_mapper.go` | 1 |
| `grep -A3 "FindOUDNByDeptIDForConfig" dept_ou_mapper.go \| grep -c "ad_config_id = ?"` | 1（≥1） |
| `grep -c "func (m \*DeptOUmapper) FindOUDNByDeptID(" dept_ou_mapper.go`（旧方法保留） | 1 |
| `go test ./internal/services/addomain/ -count=1` | ok |

### Task 2

| 断言 | 结果 |
|------|------|
| `go test ./internal/services/addomain/ -run "TestPhase58_Drain_" -v -count=1` | 23/23 PASS（Plan 03 的 13 + 本 plan 10） |
| `grep -c "func (d \*ADPushDrainer) compareAndMoveOU(" ad_push_queue_drain_service.go` | 1 |
| `grep -c "...compareAndMoveOU(ctx context.Context, c LDAPClientIface, row models.ADPushQueue, user *models.User, cfgID string, now time.Time) (terminal bool, err error)"` | 1（签名含 terminal） |
| 行序：`if terminal {` = :384 < `d.markPushed(ctx, row.ID, now)`（update_user 分支）= :387，且区间 [384,386] 含 `return`、不含 `markPushed` | ✅（见 Deviations #1 对断言文字的处置） |
| `grep -c "d.markDead(ctx, row.ID, now, row.Attempts," ad_push_queue_drain_service.go` | 9（≥2） |
| `grep -c "newDrainTestDrainer(" ad_push_queue_drain_test.go` | 23（≥10） |
| `grep -c "extractRDNFromDN(" ad_push_queue_drain_service.go` | 2（≥1，1 处为注释引用）；`grep -A6` 邻域含 `ad_ou_dn` = 3（≥1） |
| `grep -c "NormalizeDN(\*user.AdOuDn) == NormalizeDN(targetOUDN)" ad_push_queue_drain_service.go` | 1 |
| `grep -c "FindOUDNByDeptIDForConfig(ctx" ad_push_queue_drain_service.go` | 2（≥1）；`grep -c "FindOUDNByDeptID(ctx"` = **0**（F-B 防线） |
| P2 防线：回写 `ad_dn` 的 `Updates` 之前有 `Unscoped()`（人工核对 `:461`） | ✅ |
| `grep -c "ad_dn.*updated_at\|updated_at.*ad_dn" ad_push_queue_drain_service.go` | 0（禁时间戳仲裁） |
| `go test ./internal/services/addomain/ -count=1` | ok |

### Task 3

| 断言 | 结果 |
|------|------|
| `go test ./internal/services/addomain/ -run "TestPhase58_Move" -v -count=1` | 2/2 PASS |
| `grep -c "extractRDNFromDN(" user_ad_sync_service.go` | 2（≥2） |
| `grep -c 'Update("ad_ou_dn", ouDN)' user_ad_sync_service.go` | **0** |
| `grep -c 'Updates(map\[string\]interface{}{\"ad_dn\": newAdDn, \"ad_ou_dn\": ouDN})' user_ad_sync_service.go` | 2（≥1） |
| 人工核对：两处均未改动 `MoveUser` 调用条件；`grep -n NormalizeDN user_ad_sync_service.go` 仅 1 处且为注释 | ✅ |
| `go test ./internal/services/addomain/ -count=1`（含 `manager_sync_test.go` 既有 BatchMove 用例） | ok |

### 采样命令（phase 约束）

```
go test ./internal/services/addomain/ -run "TestPhase58_Drain_|TestPhase58_Move|TestPhase58_FindOUDNByDeptIDForConfig" -count=1
  → ok（28 个用例全 PASS）
go test ./internal/services/system/ -run TestPhase57 -count=1（sentinel）
  → ok
```

## Test Results

```
go build ./...                                                              → exit 0
go test ./internal/services/addomain/ -count=1                              → ok (7.7s)
go test ./internal/services/system/ -run TestPhase57 -count=1（sentinel）    → ok
go test ./...                                                               → exit 0（全仓，0 FAIL）
go vet ./internal/services/addomain/                                        → 无输出
```

## Decisions Made

1. **Plan 03 的 `TestPhase58_Drain_UpdateUserSuccessWritesPushedAt` fixture 补 OU 映射铺底** —— 见 Deviations #2。
2. **Task 1 的三个 mapper 用例放在 `dept_ou_mapper_test.go`** —— 见 Deviations #3。
3. **不标记 QUEUE-04 / QUEUE-05 为 complete** —— 沿用 58-01/58-03 分档口径（cron 接线在 58-05、端到端证据在 58-07），由 58-07 回填。
4. **`compareAndMoveOU` 中新增一层 `ad_dn` 判空防御**（`plan` 未显式要求）—— `drainRow` 已判过一次，此处是闭包内 panic 的最后防线（P1「闭包内不得 panic」）。

## Deviations from Plan

**1. [Rule 1 - 计划内部矛盾] 「sed 区间内必须含 return 且不含 markPushed」的断言文字自相矛盾**

- **Found during:** Task 2
- **Issue:** plan 的验收要求 `sed -n '<if terminal 行>,<markPushed 行>p'` 打印区间，并断言"区间内必须含 `return` 且**不含** `markPushed`" —— 但该区间的**终点行就是 `markPushed` 那一行**，任何满足前者的代码都不可能满足后者（与 58-03 的 F-12 grep 超串矛盾同款）。
- **Fix:** 按 plan 的 `<action>` 实施（`if terminal { return }` 紧随 `d.markPushed(ctx, row.ID, now)` 之前），并给出**语义等价的成立口径**：取半开区间 `[384, 386]`（`sed -n '384,386p'`），实测输出为 `if terminal {` / `return` / `}` —— 含 `return`、不含 `markPushed`；且 `if terminal {`(:384) < update_user 分支的 `markPushed`(:387)。断言实质未降低。
- **Files modified:** `internal/services/addomain/ad_push_queue_drain_service.go`
- **Committed in:** `f00f41b1`

**2. [Rule 1 - 语义变更导致既有 fixture 失效] Plan 03 的 UpdateUserSuccess 用例补 OU 映射铺底**

- **Found during:** Task 2（首次 `go test -run TestPhase58_Drain_` 红）
- **Issue:** `TestPhase58_Drain_UpdateUserSuccessWritesPushedAt` 给用户设了 `dept_id = dept-1` 但**没有** `sys_dept_ou_mapping` 行；叠加 compare-and-move 后该组合按 Q2 决策转 `dead`，与用例断言的 `pushed` 冲突。
- **Fix:** 补一行 `seedDrainOUMapping(dept-1, cfg-1, "OU=研发部,DC=corp,DC=com")`（与该用户当前 `ad_ou_dn` **归一化相等**）→ 目标一致跳过 `MoveUser` → 照常 `pushed`。用例语义（"属性路径成功写 pushed_at"）完整保留，断言一条未删（`updateUserAttrCalls == 1` / `status == pushed` / `PushedAt 非 nil`）。**不是**为变绿弱化断言，而是让 fixture 与 Plan 04 明确引入的新语义对齐；新旧语义的边界另有 `TestPhase58_Drain_NoOUMappingGoesDead` 专门锁定。
- **Files modified:** `internal/services/addomain/ad_push_queue_drain_test.go`
- **Committed in:** `f00f41b1`

**3. [Rule 3 - Blocking] Task 1 的用例落点从「复用 setupDrainTestDB」改为 `dept_ou_mapper_test.go` 自带 fixture**

- **Found during:** Task 1
- **Issue:** plan 建议"若 `dept_ou_mapper_test.go` 无 sqlite fixture 则追加到 `ad_push_queue_drain_test.go` 并复用 `setupDrainTestDB`"。但 `ad_push_queue_drain_test.go` 与 `drainNow()` 都带 `//go:build !skip_db_tests` tag（58-01 的编译面决策），而 `dept_ou_mapper_test.go` **无** tag —— 无 tag 文件引用带 tag 符号会让 `go vet -tags skip_db_tests ./internal/services/addomain/` 报 `undefined`（正是 58-01 记录要避免的回归）。且 `dept_ou_mapper_test.go` **已有** sqlite fixture（`setupOUMapperDB`）。
- **Fix:** 在 `dept_ou_mapper_test.go` 内新增 `setupOUMapperDBWithHandle`（返回 `*gorm.DB` 以便按 `ad_config_id` 铺多行）+ `seedOUMappingRow`，复用同包无 tag 的 `createDeptOUMappingTable` const。3 个用例与 plan 要求逐条对应，断言强度不变。
- **Files modified:** `internal/services/addomain/dept_ou_mapper_test.go`
- **Committed in:** `44b4b958`

---

**Total deviations:** 3 auto-fixed（2 处 Rule 1：1 处计划内部矛盾 + 1 处语义变更致 fixture 失效；1 处 Rule 3 blocking）
**Impact on plan:** 均未降低任何验收断言的强度；矛盾项以"plan 自己写的可证伪断言"为准裁决（与 58-02/58-03 同款处置口径）。

## Issues Encountered

- **`gsd-sdk` CLI 在本执行环境不可用**（沿用 58-01/58-02/58-03 结论，`NO_GSD_SDK`）。STATE.md / ROADMAP.md / VALIDATION.md 的进度回填按现有结构手工完成，未调用 `state.advance-plan` 等 SDK 动词。
- 工作区存在大量**与本 plan 无关**的未暂存 `.planning/phases/*` 删除（执行前既有状态）。全程只用 `git add <具体文件>` 逐文件暂存。

## Known Stubs

无。本 plan 交付的全部符号均被生产代码或测试真实引用：`FindOUDNByDeptIDForConfig` 被 `compareAndMoveOU` 调用（生产路径）并被 3 个用例锁定；`compareAndMoveOU` 被 `drainRow` 的 update_user 分支调用并被 10 个用例覆盖；`updateUserAttrLastAttrs` 被 `TestPhase58_Drain_UsesLatestDBValues` 消费。

⚠ 一处**预期内但须下游知悉**的边界：`compareAndMoveOU` 第 1 步（`user.DeptID == nil`）在当前调用链下**不可达**（用户在 `drainRow` 已通过；无部门用户仍会走到该分支并返回 `(false, nil)` —— 实际可达，见 `TestPhase58_Drain_NoDeptSkipsMove`）。非 stub，仅为文档澄清。

## Threat Flags

无新增安全相关表面。本 plan 的威胁登记项（T-58-04-01 ~ T-58-04-09）全部以**代码 + 可证伪测试**落实：

| Threat | 落实位置 |
|--------|----------|
| T-58-04-01 Tampering（跨域写入，F-B） | 排水器走 `FindOUDNByDeptIDForConfig`；`grep -c "FindOUDNByDeptID(ctx" ad_push_queue_drain_service.go` == 0；`TestPhase58_Drain_UsesConfigScopedOUMapping`（双配置同部门，cfg-b 行 id 排序在前仍必须取 cfg-a）+ `TestPhase58_FindOUDNByDeptIDForConfig_*` 三例 |
| T-58-04-02 Tampering（MoveUser 后不回写 ad_dn，F-A） | 三处载体全部 `Updates(map){ad_dn, ad_ou_dn}`；`TestPhase58_Drain_MoveUpdatesAdDnAndAdOuDn` / `TestPhase58_MoveUserToNewOU_UpdatesAdDn` / `TestPhase58_MoveSingleUserToOU_UpdatesAdDn` 各自断言 DB 落值；后果链（陈旧 DN → 配置误解析 → 误转 dead → 属性打旧 DN → 二次 code 32）在 `compareAndMoveOU` 注释完整记录 |
| T-58-04-03 Tampering（幂等性失效，未归一化） | `TestPhase58_Drain_NormalizedComparisonSkipsMove`（attr 名大小写 + 值大小写 + 空格三类差异）→ `moveUserCalls == 0`；未过 `NormalizeDN` 时必红 |
| T-58-04-04 Tampering（新旧值比较） | `TestPhase58_Drain_UsesLatestDBValues`：入队后改 nickname + dept_id，断言 AD 侧 `displayName`/`department` 与移动目标均为最新值；代码中不存在"入队旧值"数据源 |
| T-58-04-05 Info Disclosure（OU 缺失批量 dead） | **accept**（Q2）—— 见上文「Q2/Q3 落地位置与可见后果」的排障 SQL 与人工作业说明 |
| T-58-04-06 Elevation（deleteOldRDN=true） | **accept** —— 既有 `LDAPClient.MoveUser` 行为未改；RDN 由 `extractRDNFromDN` 从已存储的 `ad_dn` 取出，无外部输入 → 无注入面 |
| T-58-04-07 Repudiation（移动失败被属性成功掩盖） | `handlePushError(..., "move_ou")` → `failed` + 退避；`TestPhase58_Drain_MoveFailureIsRetryable` 断言 `status == failed` 且 `last_error` 前缀 `retry:move_ou:`（**不是** pushed） |
| T-58-04-08 Tampering（终态被覆盖） | 返回值契约 `(terminal, err)` + 调用方 `if terminal { return }`（:384-386，位于 `markPushed`(:387) 之前）；`TestPhase58_Drain_NoOUMappingGoesDead` 断言 `status == dead` 且 `PushedAt == nil` |
| T-58-04-09 Tampering（dead 原因混淆） | `last_error` 前缀分类：本 plan 新增 `unrecoverable:no-ou-mapping`；既有 `unrecoverable:{no-ad-config,no-ad-dn,user-not-found,target-not-found,unknown-action}` / `dead:max-attempts` / `retry:*` / `pool:*` 类各有独立测试断言 |

## 状态更新

- **STATE.md**：Current Position 更新为 Phase 58 / Plan 04 完成（Wave 3/6）；Decisions 追加 58-04 的 6 条执行期决策；progress `completed_plans 6→7`、`percent 60→70`；last_activity 改写。
- **ROADMAP.md**：Phase 58 Plans 列表 58-04 标记 complete；Progress Table 行更新为 `4/7 | In progress (wave 3 done 2026-10-19)`。
- **VALIDATION.md**：58-04/T1、T2、T3 三行 `⬜ pending` → `✅ green <commit>`。
- **REQUIREMENTS.md**：QUEUE-04 / QUEUE-05 均为跨 plan 需求，**不在本 plan 标记 complete**（沿用 58-01/58-03 分档口径；最终回填归 58-07）。

## Next Phase Readiness

- **Wave 4（58-05）可立即开** —— cron 接线的唯一构造形态仍是 `NewADPushDrainer(db, s.getPool())`（**禁** `addomain.NewAccountPool`，58-03 记录）；排水器行为面已完整，`DrainOnce` 的语义（含 OU 移动）不再变更。
- **58-06 必须遵守**：① 删两条直推路径时**一并清理**上文「Plan 06 调用点清单」的 5 项；② `moveUserToNewOU` / `moveSingleUserToOU` 在 06 之前必须保留（本 plan 已修其 F-A，删除属 06 范围）；③ 排水器**不依赖**这两个函数。
- **58-07 必须遵守**：① 端到端用例应覆盖"入队 → 排水 → compare-and-move → 终态"全链；② `unrecoverable:no-ou-mapping` 的**批量 dead 可见后果**须进人工 UAT 清单（部门同步未跑过的环境）；③ 五条 SC 的 grep 断言沿用本 SUMMARY 的表格口径。
- **无阻塞**：全仓 `go test ./...` exit 0；Phase 57 sentinel（`TestPhase57`）绿；`go vet ./internal/services/addomain/` 无输出。

---

*Phase: 58-compare-and-move*
*Completed: 2026-10-19*

## Self-Check: PASSED
