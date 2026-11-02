---
phase: "84"
slug: "pmodel-gaps"
status: human_needed
nyquist_compliant: true
wave_0_complete: true
verified: "2026-10-28"
requirements_satisfied: 4
requirements_total: 4
audit_acknowledged:
  milestone: v1.25
  at: 2026-10-30
  status: human_needed
---

# Phase 84 — VERIFICATION

> 本 phase 由 4 个 subagent executor 直接执行（沿用 v1.24「subagent 直执行 + 按刀原子提交」先例），故无 GSD 契约的 per-plan `SUMMARY.md`；本文件由 Lead 汇总**已提交的实测输出**生成，未重跑验证。
> 证据来源：各 executor 返回报告 + Lead 独立复核命令输出 + git commit 链。

---

## Requirements 验收对照（4/4 Satisfied）

| REQ | 需求 | Phase | Status | 关键证据（已提交） |
|-----|------|-------|--------|---------------------|
| PMODEL-01 | `data_scope` 被列表查询真实消费 | 84-04 | ✅ Satisfied | `2fc83665`：`applyUserDataScope` 五档过滤（All/0 不过滤 · Custom 子查询 · Dept `dept_id=(SELECT …)` · DeptChild ancestors 5 条件单查询 · **Self 按 `sys_user.id` 主键**（修正 `permission.go:575` 的 `dept_id=userID` 错位形态）· 未知枚举 `1=0` fail-closed）；过滤插在 **Count 之前**（total 同步）；`StatisticsWithDataScope` 同口径；`buildListCacheKey` 加 `:ds:<n>:uid:<x>` 段（`CacheKeyUserList + "*"` 失效兼容保留） |
| PMODEL-02 | `GetUserPermissions` 过滤 `menu.status` | 84-01 | ✅ Satisfied | `305ff44f`：`system/menu_service.go` 的 `GetUserPermissions` 补 `status = models.MenuStatusNormal`；**判定侧 `pkg/permission/service.go` 零 diff**（其 SQL 已有 `AND m.status = ?`）；未接 `InvalidateMiddlewarePermCache`（v1.25 Future） |
| PMODEL-03 | Step 2「子菜单继承」语义固化 | 84-02 | ✅ Satisfied | `bb92089c`：内联块抽为私有函数 `hasInheritedButtonPerm` + 权威四步链注释 + **方向声明「F→父成立 / 父→子不成立」**；`TestStep2_InheritDirection` 锁反向 |
| PMODEL-04 | 迁移记录表 + GORM schema 漂移登记 | 84-03 | ✅ Satisfied | `b3210144`：`migration_225` 建 `sys_schema_migrations(version,name,applied_at)`；`registry.go` 的 `ActiveMigrations`(26) / `MissingRecords` 对账；`database.go` **52 调用点**接 `runAndRecord`（记录失败只 WARN 不阻断；**零 skip 控制、幂等重跑语义零改变**）；漂移登记 `.planning/notes/260628-gorm-schema-drift-register.md`（2 条） |

---

## 验证证据

### 红绿验证链（每个改行为的 plan 均先红后绿）

| Plan | 红（回退实现即失败） | 绿 |
|------|---------------------|-----|
| 84-01 | `menu_service_test.go:537` — `Error: []string{"menu:stopped:perm", "menu:normal:perm"} should not contain "menu:stopped:perm"` / `--- FAIL` | `--- PASS: TestGetUserPermissions_StatusDisabledMenuExcluded` |
| 84-02 | 判定行为零变化：抽取前 `go test ./pkg/middleware/` → `ok 3.335s`；抽取后 → `ok 3.088s`（**既有测试零 diff**） | `--- PASS: TestStep2_InheritDirection (0.03s)`；`grep -c hasInheritedButtonPerm` = 4（≥3） |
| 84-03 | — | `TestMigrate225` / `TestRunAndRecord_Idempotent` / `TestRunAndRecord_NonBlocking` / `TestReconcile` 四用例全 PASS（NonBlocking 锁「记录失败不阻断启动」方向） |
| 84-04 | `user_data_scope_test.go:188` — `Should not be: (*string)(nil)`（第二次 scope=5 命中第一次 scope=3 的缓存 → **跨用户泄漏**）｜`:199` — `Not equal: expected: 1, actual: 5`（total 复用缓存）｜`:210` — `"user:list:orderBy::isAsc:default:page:1:size:100" does not contain ":ds:3:uid:uA1"`（key 无 scope 段） | `PASS: TestUserServiceList_DataScope`（七档逐档行集+total）｜`PASS: TestDataScope_NoChangeForAllAndAdmin`｜`PASS: TestUserListCache_ScopeIsolation`（双 scope 双用户互不串，key 含 `:ds:3:uid:uA1` / `:ds:5:uid:uA2`，前缀仍 `user:list`） |

### 闸门（Lead 独立复核，2026-10-28）

| 命令 | 结果 |
|------|------|
| `go build ./...` | ✅ 干净（`BUILD_EXIT=0`） |
| `go test ./internal/services/system/ ./pkg/middleware/ ./internal/core/db/... ./internal/api/v1/system/` | ✅ 全 ok（11.408s / 2.963s / 9.715s / 1.862s / 4.664s） |
| `go build ./... && go test ./...` | ✅ exit 0（全量；executor 另用 `grep -E "^(FAIL\|---)"` 复扫全量输出 → **0 命中**） |
| 零接触断言 `git diff --stat internal/api/v1/system/user_router.go pkg/permission/ internal/api/v1/vdi/` | ✅ **空输出**（写组 / 判定侧 / VDI 三处零越界） |
| `pkg/middleware/permission.go` 改动面 | ✅ **仅 +5 行注释**（Self 分支代码零改动） |
| `DataScopePermission` 挂载数 | ✅ 维持 **4 处**（router.go:187/212/272/988） |

### 边界核对（Lead 逐条验）

- ✅ 不接 `InvalidateMiddlewarePermCache`（30s 中间件缓存属 v1.25 Future）
- ✅ `depts` / `vms` 的 data_scope 零接触（depts 有「组织全树低敏感」决策注释 router.go:262；vms 已自带消费）
- ✅ `usersWrite` 写组 data_scope 零接触（写操作由 v1.24 #16 严格权限中间件兜底）
- ✅ 不修 sqlite `ON CONFLICT` 方言（Future）
- ✅ `system` 域 4 组 ANY-of 拆分 / 死表接线 / F 型播种**均未做**（Phase 85/87）
- ✅ 未知 scope 枚举 fail-closed（`1=0`）；admin 与 data_scope=1 **零变化**（`TestDataScope_NoChangeForAllAndAdmin`）

---

## 遗留（人工验证项，不阻塞 phase 完成）

| 项 | 要求 | 为何人工 | 状态 |
|----|------|----------|------|
| PG 实机 data_scope=3 复测 total | PMODEL-01 | 生产 PG 与 sqlite 行为差异（v1.24 #79 同款对照） | ⏳ 待用户执行（方法已登记于 `84-VALIDATION.md`） |
| `sys_role_menu`/`sys_user_role` 生产库实际列集 | PMODEL-04 | 库表侧需 `\d` 实查 | ⏳ 待用户执行；模型侧缺列已实证并登记为显式债务（note 标注 LOW→HIGH） |
| v1.24 #79 验收脚本是否断言过用户列表行数 | 回归 | 历史脚本行为需人工核对 | ✅ **已核对**：grep `users/list|4231|用户列表行` 无行数断言，无需登记更新 |

---

## Git 范围

| Plan | Commit |
|------|--------|
| 84-01 | `305ff44f` |
| 84-02 | `bb92089c` |
| 84-03 | `b3210144` |
| 84-04 | `2fc83665` |
| state 同步 | `5e91af6b` / `512fdef3` |

**Phase 完成判定**：4/4 需求 Satisfied + 全量测试绿 + 边界零越界 + 红绿链完整 → **phase_complete: true**
