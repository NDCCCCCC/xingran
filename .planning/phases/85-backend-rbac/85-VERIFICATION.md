---
phase: "85"
slug: "backend-rbac"
status: validated
nyquist_compliant: true
wave_0_complete: true
verified: "2026-10-29"
requirements_satisfied: 3
requirements_total: 3
---

# Phase 85 — VERIFICATION

> 4 个 subagent executor 直接执行（v1.24 先例），Lead 汇总**已提交实测输出**生成本文件，未重跑验证。
> 证据来源：executor 报告 + Lead 独立复核 + `85-STATE.md`（窗口期/移交清单/口径备忘）+ git commit 链。

---

## Requirements 验收对照（3/3 Satisfied）

| REQ | 需求 | Plan | Status | 关键证据（已提交） |
|-----|------|------|--------|---------------------|
| RBAC-01 | 只读角色任意模块调写接口一律 403（写路由逐动作单一权限点，去 ANY-of） | 85-03 | ✅ Satisfied | `b5d40bc3`+`c7fbeae2`：三批串行（ops 55 + network 39 + system 其余 39）= **133 条写路由全部 RequireStrictPermissions 单一权限点**（authorized 兄弟组，读组 ANY-of 保持）；`TestStrictWriteGroupsSingleCode` 守护 166 条三对齐；deferred 269→**138**（余为范围外冻结项：duty/knowledge/rpa 无挂载模块 + topology/mac 读组等，清单留档）；restores 组**结构缺陷修复**（v129 遗留 ANY-of 读组子组 → 兄弟组） |
| RBAC-03 | 动作映射死表真实接线并可校验 | 85-01 | ✅ Satisfied | `4adff1a9`：`scan-routes.cjs` 全站源码重扫（**697 路由 / 306 写路由 / 108 挂载点**，正规化四类盲区）+ 死表漂移 3 域重建（roles `/batch`、departments 前缀、posts/batch-delete + 2 条漂移登记删除）+ **双层守卫**（`TestRouteRegistryCoverage`/`Freshness` + `check-route-registry.mjs`）接 `lint:ci`；红绿：注入假路由 → 双拦 FAIL → 复原 exit 0 |
| RBAC-04 | system 域其余 4 组逐动作拆分 | 85-02 | ✅ Satisfied | `27604d75`：roles/menus/departments/posts **21 条写路由**全部 strict 兄弟组（读组 ANY-of 保持；DataScopePermission 对齐 usersWrite 先例）；死表遮蔽顺序修正（`:id` 取首命中 → 2 段具体路径前移，补登 5 条） |

---

## 验证证据

### 85-02 修正后口径（checker ISSUE-1/2）

- **Go +2 常量**：`DeptImport` / `DeptExport`（role/menu/dept/post 的 add/edit/remove 全部已存在）
- **TS 4 区块 23 码全量声明**（14 新动作码 + 9 既有 Go 码补声明）——b 类零豁免
- **c 类 `PENDING_SEED_WHITELIST`**：23→**78 项**（85-03 扩），每项带 reason「system 4 组 F 型菜单 0 声明，87 播种后移除」+ 证据；**白名单非空自检**（输出移除时限警告防永久化）；87 播种 migration 落地后必须清空

### 防回退守卫（85-04，`bfe54533`）

`internal/api/strict_mount_count_test.go` 三道断言（红绿：注入回退 16→14 → FAIL → 复原绿）：
1. `TestStrictMountSourceCount`：internal/api 源码 `RequireStrictPermissions(` ≥ **16** 处
2. `TestStrictMountRegistryCount`：registry Strict 挂载 ≥ **188** 条（写路由 ≥ **164**）
3. `TestSystemFourGroupsWritePurity`：system 4 组 21 条已知写路由全部保持 strict

### 窗口期（显式声明，非缺陷）

85 合入 → 87 播种之间，无 F 型码域的写操作 fail-closed（403）= **预期行为**（无授予渠道的权限本不该放行）。影响域：system 4 组（xtgl/kjgl/adyk/vm_user）+ ops import/export 新码域 + network 新码域。缓解 = 87 播种时 `GrantNewMenuToRolesHavingParent`（幂等/双路径测试已备）给持父页面 C 型菜单的角色补授，现役管理角色零感知。**已登记 `85-STATE.md`**。

### 归 87 移交清单（85-STATE.md §2）

① 决定性对照实验（≥8 模块抽样，覆盖三种拆法；#79 范式夹具）② F 型菜单播种（PENDING_SEED_WHITELIST **78 项全量**）③ 生产 6 角色回归面 SQL 实测（85-RESEARCH §B4；与 84 PG 遗留一起）

### 闸门（Lead 独立复核抽查，2026-10-29）

| 命令 | 结果 |
|------|------|
| `go build ./...` | ✅ |
| `go test ./...` 全量 | ✅ **exit 0 全绿**（Lead 复跑；85-04 报告的 2 个 flaky/环境失败本次未复现） |
| `go test ./internal/api/ -run 'TestStrictMount\|TestSystemFourGroups' -count=1` | ✅ ok |
| `node scripts/check-permission-codes.mjs` | ✅ exit 0（双向零漂移；白名单 78 项警告正常输出） |
| `node scripts/check-route-registry.mjs` | ✅ exit 0（697 / 306 / deferred 138） |
| `npm run lint:ci` | ✅ exit 0（含 C14 族 + check-route-registry 串联） |

### 边界核对

- ✅ duty/knowledge/rpa 等无挂载模块不纳入（功能新增 = Out of Scope）
- ✅ `RequireStrictPermissions` 语义零改动（只增使用点；middleware/permission.go 85 线零 diff）
- ✅ admin 零变化；读组 ANY-of 语义零变化
- ✅ 生产库零写入（对照实验归 87）
- ✅ 范围外冻结项清单留档（topology 3 条无挂载 / mac·ports 读组 6 条单码 ANY-of / system 12 条无挂载 / notification-configs·userUnlock 非混合组 / knowledge 3 组不在 PLAN 清单——后续候选）

---

## Git 范围

| Plan | Commit |
|------|--------|
| 85-01 | `4adff1a9` |
| 85-02 | `27604d75` |
| 85-03 | `b5d40bc3` + `c7fbeae2`（补守护扩展） |
| 85-04 | `bfe54533` |

**Phase 完成判定**：3/3 Satisfied + 166 条写路由 strict 三对齐守护 + 双层 registry 守卫 + 三道防回退断言 + 窗口期登记 + 全量测试绿 → **phase_complete: true**

**解锁**：Phase 86（前端三态门控铺开，76 页面）——后端逐动作权限点已就绪；Phase 87（授予闭环，含 F 型播种清空白名单）依赖 85+86。
