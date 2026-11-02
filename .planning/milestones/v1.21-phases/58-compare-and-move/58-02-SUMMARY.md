---
phase: 58-compare-and-move
plan: 02
subsystem: addomain / scheduler / system-api
tags: [go, ldap, dn-normalization, ad-config-routing, hygiene, compare-and-move, phase-58]

# Dependency graph
requires:
  - phase: 58-compare-and-move
    plan: 01
    provides: "排水测试基座（setupDrainTestDB / seedDrainADConfig 含 base_dn 列 / createDrainADConfigTable）"
  - phase: 57-ad-outbox-lite
    provides: "ad_push_queue 模型 + ADPushQueueActivePredicate + IsADSyncEnabled 双条件先例"
provides:
  - "addomain.NormalizeDN —— DN 归一化单一真实源（attr 名与值双侧 ToLower，仅用于比较）"
  - "addomain.BuildOUDN —— 部门名 → OU DN 拼接（ldap.EscapeDN + 前导 = 兜底），全仓唯一拼接点的唯一出口"
  - "addomain.MatchADConfigByDN —— D-02 最长后缀唯一匹配纯函数（带逗号边界；无匹配/歧义 → error）"
  - "addomain.FirstEnabledADConfigID（只判 status，F-D）/ FirstSyncEnabledADConfigID（status + sync_enabled）"
  - "scheduler.resolveADConfigIDFromParams —— 参数提取层薄 wrapper（原两份重复实现的唯一存续形态）"
affects: [58-03, 58-04, 58-05, 58-06, 58-07]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "纯函数优先：DN 语义（NormalizeDN / BuildOUDN）与路由判定（MatchADConfigByDN）全部无 DB、无副作用、可单测"
    - "逗号边界后缀判定 strings.HasSuffix(norm, \",\"+base) —— 禁裸 HasSuffix(dn, baseDN)（P6）"
    - "转义一律走 ldap.EscapeDN（RFC4514），禁手写转义表（P4）"
    - "同名语义分叉场景拆显式入口而非加开关参数（F-D：FirstEnabledADConfigID vs FirstSyncEnabledADConfigID）"

key-files:
  created:
    - internal/services/addomain/dn_util.go
    - internal/services/addomain/dn_util_test.go
    - internal/services/addomain/ad_config_resolver.go
    - internal/services/addomain/ad_config_resolver_test.go
  modified:
    - internal/services/addomain/dept_sync_service.go
    - internal/scheduler/ad_sync_tasks.go
    - internal/scheduler/dept_sync_tasks.go
    - internal/scheduler/ad_sync_tasks_80_02_test.go
    - internal/scheduler/dept_sync_tasks_80_02_test.go
    - internal/api/v1/system/ou_mapping_handler.go

key-decisions:
  - "BuildOUDN 用字符串拼接而非 fmt.Sprintf(\"OU=%s,%s\") —— plan 的静态证伪断言要求全仓 grep 'OU=%s,%s' 零命中，拼接与该格式化逐字等价"
  - "歧义（等长并列）的真实触发形态是两条启用配置归一后 base_dn 相同；数学上两个等长但不同的字符串不可能同时是同一 DN 的逗号边界后缀，测试据此构造"
  - "注释内不出现被删函数名（沿用 58-01 F-G tombstone 口径），满足 grep -rn getDefaultADConfigID… internal/ 零命中的验收断言"

requirements-completed: [HYGIENE-01, HYGIENE-02]

# Metrics
duration: 25min
completed: 2026-10-19
---

# Phase 58 Plan 02: DN 单一真实源 + D-02 后缀匹配 + D-03 三处收敛 Summary

**NormalizeDN/BuildOUDN 纯函数落地并接线唯一 OU DN 拼接点，MatchADConfigByDN 以逗号边界最长后缀唯一匹配锁定 D-02 契约（8 个用例含父子域反例），三处 First() 歧义收敛为两个语义分明的 addomain 入口，双份 getDefaultADConfigID 归零**

## Performance

- **Duration:** ~25 min
- **Tasks:** 3/3
- **Files modified:** 10（4 新建 + 6 修改）

## 导出签名（全文，下游 58-03/58-04/58-05 直接消费）

```go
// internal/services/addomain/dn_util.go
func NormalizeDN(dn string) string
func BuildOUDN(deptName, parentOUDN string) string

// internal/services/addomain/ad_config_resolver.go
func MatchADConfigByDN(dn string, configs []models.ADConfig) (*models.ADConfig, error)
func FirstEnabledADConfigID(ctx context.Context, db *gorm.DB) (string, error)
func FirstSyncEnabledADConfigID(ctx context.Context, db *gorm.DB) (string, error)

// internal/scheduler/ad_sync_tasks.go（本包私有）
func resolveADConfigIDFromParams(ctx context.Context, db *gorm.DB, params map[string]interface{}) (string, error)
```

### 语义要点（写死，勿回退）

| 符号 | 关键语义 |
|------|----------|
| `NormalizeDN` | 整体 TrimSpace → 按 `,` split 逐段 TrimSpace → 每段按首个 `=` 切分，**attr 名与值都 ToLower** → 按 `,` 无空格重拼；空串→空串；无 `=` 的段整段小写；**仅用于比较，绝不回写 AD / 生成 DN** |
| `BuildOUDN` | `ldap.EscapeDN` 转义后对**前导 `=`** 补 `\`（EscapeDN 不覆盖 `=`）；`parentOUDN == ""` 时返回 `"OU=" + escaped`，不产生悬空逗号 |
| `MatchADConfigByDN` | 只选 `status == ADConfigStatusEnabled`；跳过空 `base_dn`；命中判定 `strings.HasSuffix(norm, ","+base)`；最长者胜；等长并列 → `"AD 配置歧义"` error；无命中 → `"无匹配 AD 配置"` error；返回 `&configs[i]` |
| `FirstEnabledADConfigID` | `status = 0` ORDER BY `created_at ASC`；NotFound 文案 `"未找到启用的AD配置，请先在AD域配置中启用至少一个配置"`；成功打 `Infof("自动使用AD配置: %s (ID: %s)")` |
| `FirstSyncEnabledADConfigID` | `sync_enabled = true AND status = 0`，**不带 ORDER BY**（保持原 handler 语义）；NotFound 文案 `"查询 AD 配置失败: 未找到 sync_enabled=true 且 status=enabled 的配置"` |
| `resolveADConfigIDFromParams` | 只保留 `params["configId"]` / `params["adConfigId"]` 提取；自动挑选委派 `addomain.FirstEnabledADConfigID` |

## Task Commits

Each task was committed atomically:

1. **Task 1: NormalizeDN + BuildOUDN 单一真实源 + HYGIENE-02 接线** — `b2909660` (feat)
2. **Task 2: D-02 最长后缀唯一匹配 + D-03 双入口 helper** — `e1fca53a` (feat)
3. **Task 3: 三处调用点收敛（HYGIENE-01）** — `fec65e7d` (refactor)

## 收敛清单（Task 3）

| 位置 | 收敛前 | 收敛后 |
|------|--------|--------|
| `internal/scheduler/ad_sync_tasks.go:306` | `getDefaultADConfigID`（含参数提取 + 自动挑选 + 日志，全部内联） | `resolveADConfigIDFromParams`（只留参数提取）+ `addomain.FirstEnabledADConfigID` |
| `internal/scheduler/dept_sync_tasks.go:47` | `getDefaultADConfigIDForDept`（逐字拷贝的第二份） | **函数整体删除**；两处调用点（原 `:84` / `:138`）改用 `resolveADConfigIDFromParams` |
| `internal/api/v1/system/ou_mapping_handler.go:117` | `Where("sync_enabled = ? AND status = ?", true, ...).First(&adConfig)` | `addomain.FirstSyncEnabledADConfigID(ctx, h.db)`；`mapping.ADConfigID = adConfig.ID` → `= adConfigID` |

**测试同步点**（断言语义零改动，仅标识符改名）：
- `internal/scheduler/ad_sync_tasks_80_02_test.go`：`:308/:315/:510/:515/:520/:529` 调用点改名；用例名 `TestAds8002_GetDefaultADConfigID{,_ParamKeys,_QueryError}` → `TestAds8002_ResolveADConfigIDFromParams{,_ParamKeys,_QueryError}`
- `internal/scheduler/dept_sync_tasks_80_02_test.go`：`:114/:121/:302/:307/:316` 调用点改名；用例名同款 `TestDst8002_...`

## HYGIENE-02 存量口径决策（Q4）落地位置

`internal/services/addomain/dept_sync_service.go` `syncDeptTree` 函数上方注释 + `ouDN := BuildOUDN(dept.DeptName, parentOUDN)`：

> Phase 58 HYGIENE-02: OU DN 拼接走 BuildOUDN（全仓唯一拼接点），部门名特殊字符按 RFC4514 转义。存量 `sys_dept_ou_mapping.ou_dn` 为未转义值，含特殊字符的部门需重跑部门同步后才会写入转义 DN（Q4 决策：不做存量数据迁移）。

`NormalizeDN` 之所以走轻量字符串路径而非 `ldap.ParseDN`，正是因为存量 `ou_dn` 是**未转义**值，`ParseDN` 遇未转义 `,` 会解析失败（T-58-02-04 的误用防线）。

## Files Created/Modified

- `internal/services/addomain/dn_util.go`（**新建**）— `NormalizeDN` + `BuildOUDN`，文件头注释声明"DN 语义的单一真实源"
- `internal/services/addomain/dn_util_test.go`（**新建**）— 5 个 `TestPhase58_*` 用例
- `internal/services/addomain/ad_config_resolver.go`（**新建**）— `MatchADConfigByDN` + 两个 `First*` 入口
- `internal/services/addomain/ad_config_resolver_test.go`（**新建**，带 `!skip_db_tests` tag）— 8 个 `TestPhase58_*` 用例
- `internal/services/addomain/dept_sync_service.go`（L112–118）— 唯一 OU DN 拼接点改走 `BuildOUDN` + HYGIENE-02 注释
- `internal/scheduler/ad_sync_tasks.go`（L304–321）— wrapper 改名 + 委派
- `internal/scheduler/dept_sync_tasks.go`（L45–47 删除；`gorm` import 随之移除）— 重复实现删除 + 两处调用点
- `internal/scheduler/*_80_02_test.go`（2 个文件）— 标识符与用例名同步改名
- `internal/api/v1/system/ou_mapping_handler.go`（L115–124）— 第三处 First() 收敛

## 验收断言实测（全部通过）

| 断言 | 结果 |
|------|------|
| `grep -c "func NormalizeDN(dn string) string" dn_util.go` | 1 |
| `grep -c "func BuildOUDN(deptName, parentOUDN string) string" dn_util.go` | 1 |
| `grep -c "ldap.EscapeDN" dn_util.go` | 4（≥1） |
| `grep -v '^\s*//' dn_util.go \| grep -c escapeDN`（P4 防线） | 0 |
| `grep -v '^\s*//' dn_util.go \| grep -c ToLower`（值侧归一 ≥2） | 3 |
| `grep -c "BuildOUDN(dept.DeptName, parentOUDN)" dept_sync_service.go` | 1 |
| `grep -c 'fmt.Sprintf("OU=%s,%s"' dept_sync_service.go` | 0 |
| `grep -rn 'OU=%s,%s' --include=*.go internal/` | 0 |
| `grep -c 'func MatchADConfigByDN(dn string, configs \[\]models.ADConfig)' ad_config_resolver.go` | 1 |
| `grep -c "func FirstEnabledADConfigID" / "func FirstSyncEnabledADConfigID" ad_config_resolver.go` | 1 / 1（禁合并） |
| `grep -c 'HasSuffix(norm, ","+base)' ad_config_resolver.go`（P6 防线） | 1 |
| `grep -c "strings.HasSuffix" ad_config_resolver_test.go`（反例证据） | 1 |
| `grep -c "func getDefaultADConfigID" internal/scheduler/*.go` 每行 | 0 |
| `grep -rn "getDefaultADConfigIDForDept" internal/` | 0 |
| `grep -c "func resolveADConfigIDFromParams" ad_sync_tasks.go` | 1 |
| `grep -c "resolveADConfigIDFromParams(ctx, db, params)" dept_sync_tasks.go` | 2 |
| `grep -c "addomain.FirstEnabledADConfigID(ctx, db)" ad_sync_tasks.go`（F-D） | 1 |
| `grep -c "addomain.FirstSyncEnabledADConfigID(c.Request.Context(), h.db)" ou_mapping_handler.go` | 1 |
| `grep -c 'Where("sync_enabled = ? AND status = ?", true, models.ADConfigStatusEnabled).First' ou_mapping_handler.go` | 0 |

## Test Results

```
go build ./...                                                              → exit 0
go test ./internal/services/addomain/ -count=1                              → ok
go test ./internal/scheduler/ -count=1                                      → ok
go test ./internal/api/v1/system/ -count=1                                  → ok
go test ./internal/services/system/ -run TestPhase57 -count=1（sentinel）    → ok
go test ./...                                                               → exit 0（全仓）
go vet ./internal/services/addomain/ ./internal/scheduler/                  → 无输出
```

D-02 契约 8 个用例（含 CONTEXT 点名的父子域共享后缀反例与 P6 逗号边界 `DC=xcorp` 反例）全 PASS；HYGIENE-02 的 5 个用例全 PASS。

## Decisions Made

1. **`BuildOUDN` 用字符串拼接而非 `fmt.Sprintf("OU=%s,%s", ...)`** —— plan 的 Task 1 静态证伪断言要求 `grep -rn 'OU=%s,%s' --include=*.go internal/` 零命中，而 plan 的 `<action>` 又建议用该格式化串。两者冲突时以**可证伪的验收断言**为准：拼接与格式化逐字等价，且满足断言。（见 Deviations #1）
2. **歧义用例的真实构造** —— plan 建议的 `{DC=x,DC=corp,DC=com}` / `{DC=y,DC=corp,DC=com}` + `CN=u,DC=x,DC=y,DC=corp,DC=com` 在逗号边界算法下**只有后者命中**（前者的 `,dc=x,dc=corp,dc=com` 不是该 DN 的后缀），不构成歧义。数学上两个等长但不同的字符串不可能同时是同一字符串的逗号边界后缀，故等长并列的唯一真实形态是"两条启用配置归一后 base_dn 相同"（同一域被重复配置）。测试据此构造 `{DC=corp,DC=com}` / `{dc=CORP,dc=COM}`，仍真实锁定"等长并列 → 报错不取首条"。（见 Deviations #2）
3. **注释内抹掉被删函数名** —— 沿用 58-01 F-G 的 tombstone 口径，`dept_sync_tasks.go` / `ad_sync_tasks.go` / `ad_config_resolver.go` 的说明注释一律不出现旧标识符字面量，以满足 `grep -rn "getDefaultADConfigIDForDept" internal/` 零命中的验收断言。

## Deviations from Plan

**1. [Rule 1 - 计划内部矛盾] `BuildOUDN` 改用字符串拼接，规避 `OU=%s,%s` 静态证伪断言**

- **Found during:** Task 1
- **Issue:** plan 的 `<action>` 指定 `fmt.Sprintf("OU=%s,%s", escaped, parentOUDN)`，但同一 task 的 `<acceptance_criteria>` 要求 `grep -rn 'OU=%s,%s' --include=*.go internal/` 无命中 —— 两者不可同时满足。
- **Fix:** 改为 `"OU=" + escaped + "," + parentOUDN`（逐字等价），并移除 `dn_util.go` 不再使用的 `fmt` import。验收断言（可证伪）优先于 action 的示意代码。
- **Files modified:** `internal/services/addomain/dn_util.go`
- **Verification:** `grep -rn 'OU=%s,%s' --include=*.go internal/` → 0；`go test ./internal/services/addomain/ -run TestPhase58_BuildOUDN -count=1` → PASS。
- **Committed in:** `b2909660`

**2. [Rule 1 - 计划内部矛盾] 歧义用例改用"重复配置"构造**

- **Found during:** Task 2
- **Issue:** plan 给出的歧义测试数据在逗号边界算法下不构成歧义（见 Decisions #2 的推导），照抄会让 `TestPhase58_MatchADConfigByDN_AmbiguousEqualsLength` 红而无法"让测试变绿而不弱化断言"。
- **Fix:** 构造两条归一后 base_dn 相同的启用配置（`DC=corp,DC=com` / `dc=CORP,dc=COM`），仍断言 `err != nil && cfg == nil && 文案含 "AD 配置歧义"`；并在用例注释中写明为何等长并列只有这一种真实形态。断言强度未降低。
- **Files modified:** `internal/services/addomain/ad_config_resolver_test.go`
- **Verification:** `go test ./internal/services/addomain/ -run TestPhase58_MatchADConfigByDN_AmbiguousEqualsLength -v` → PASS。
- **Committed in:** `e1fca53a`

**3. [Rule 3 - Blocking] `dept_sync_tasks.go` 移除 `gorm` import**

- **Found during:** Task 3
- **Issue:** 删除 `getDefaultADConfigIDForDept` 后该文件不再引用 `gorm`，编译报 `"gorm.io/gorm" imported and not used`。
- **Fix:** 移除该 import。
- **Files modified:** `internal/scheduler/dept_sync_tasks.go`
- **Verification:** `go build ./...` → exit 0。
- **Committed in:** `fec65e7d`

---

**Total deviations:** 3 auto-fixed（2 处 Rule 1 计划内部矛盾 + 1 处 Rule 3 blocking）
**Impact on plan:** 均未降低任何验收断言的强度；两处矛盾均以"plan 自己写的可证伪断言"为准裁决。

## Issues Encountered

- **`gsd-sdk` CLI 在本执行环境不可用**（沿用 58-01 结论）。STATE.md / ROADMAP.md / REQUIREMENTS.md / VALIDATION.md 的进度回填按现有结构手工完成，未调用 `state.advance-plan` 等 SDK 动词。
- 工作区存在大量**与本 plan 无关**的未暂存 `.planning/phases/*` 删除（执行前既有状态）。全程只用 `git add <具体文件>` 逐文件暂存。

## Known Stubs

无。本 plan 交付的全部符号均已被生产代码或测试真实引用（`BuildOUDN` 被 `dept_sync_service.go` 调用；`FirstEnabledADConfigID` 被 scheduler wrapper 调用；`FirstSyncEnabledADConfigID` 被 `ou_mapping_handler.go` 调用；`MatchADConfigByDN` / `NormalizeDN` 供 58-04 排水器消费，本 wave 由契约测试锁定）。

## Threat Flags

无新增安全相关表面。本 plan 的威胁登记项（T-58-02-01 ~ T-58-02-07）全部以**代码 + 可证伪测试**落实：

| Threat | 落实位置 |
|--------|----------|
| T-58-02-01 Tampering（后缀误命中） | `MatchADConfigByDN` 的 `HasSuffix(norm, ","+base)` + `ParentChildSharedSuffix`（同时锁"陷阱存在"与"已规避"）+ `CommaBoundaryRejectsXcorp` |
| T-58-02-02 Injection（DN 注入） | `BuildOUDN` 走 `ldap.EscapeDN` + 前导 `=` 兜底；`TestPhase58_BuildOUDN_EscapesSpecialChars` |
| T-58-02-03 Tampering（合并加 sync_enabled 改变语义） | 拆两个入口 + `TestPhase58_FirstEnabledADConfigID_SkipsSyncDisabled` |
| T-58-02-04 Info Disclosure（NormalizeDN 误用于生成 DN） | 函数注释写死"仅用于比较"；不用 `ParseDN` |
| T-58-02-05 Tampering（歧义静默取首条） | 等长并列显式 error + `AmbiguousEqualsLength` |
| T-58-02-06 DoS（空 base_dn 恒命中） | 显式 `continue` + `NoMatchIsError` 第三子用例 |
| T-58-02-07 Repudiation（收敛后无痕迹） | `FirstEnabledADConfigID` 保留 `Infof`；`MatchADConfigByDN` 错误文案带完整 DN 与并列 base_dn |

## 状态更新

- **STATE.md**：Current Position 更新为 Phase 58 / Plan 02 完成；Decisions 追加本 plan 的 key-decisions；Performance Metrics 追加执行记录。
- **ROADMAP.md**：Phase 58 进度行 Plan 02 标记 complete（2/7 plans）。
- **VALIDATION.md**：58-02/T1、T2、T3 三行状态 `⬜ pending` → `✅ passed`（执行期回填）。
- **REQUIREMENTS.md**：HYGIENE-01 / HYGIENE-02 两项标记 complete（本 plan 即为二者的全部落地范围）。

## Next Phase Readiness

- **Wave 2（58-03）可立即开** —— 队列拾取谓词与退避曲线的实现不再依赖任何未落地符号。
- **Wave 3（58-04）排水器可直接消费**：`MatchADConfigByDN(dn, configs)`（无匹配/歧义 → 队列行 dead）、`NormalizeDN`（compare-and-move 幂等比较）、`BuildOUDN`。
- **58-04 必须遵守**：`TestPhase58_Drain_NormalizedComparisonSkipsMove` 的三处大小写语义必须与 `NormalizeDN` 的值侧 ToLower 契约一致（attr 名与值**双侧**）。
- **无阻塞**：全仓 `go test ./...` exit 0；Phase 57 sentinel（`TestPhase57`）绿。

---

*Phase: 58-compare-and-move*
*Completed: 2026-10-19*

## Self-Check: PASSED
