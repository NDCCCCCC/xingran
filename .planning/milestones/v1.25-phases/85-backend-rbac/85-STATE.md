# 85-STATE.md — Phase 85 收口状态（Plan 04，2026-10-29）

前置提交：85-01 `4adff1a9` → 85-02 `27604d75` → 85-03 `b5d40bc3`+`c7fbeae2`。
Plan 04 产物：`internal/api/strict_mount_count_test.go`（防回退守卫，红绿验证过）+ 本文件。

## 一、窗口期登记（85 合入 → 87 播种之间）

**现象**：85 全站拆分合入后、87 F 型菜单播种前，凡无对应 F 型动作码的现有角色，其相关域写操作 **fail-closed（HTTP 403）**：

- **system 4 组**（roles/menus/departments/posts）：`xtgl`/`kjgl`/`adyk`/`vm_user` 等现役角色此前靠组级 ANY-of（含 list）放行写操作；拆分后仅持写码者放行，而 `system:{role,menu,dept,post}` F 型码全为 0（tmp_ftype.json 实证，85-RESEARCH §B4）→ 新动作码无授予渠道 → Strict 精确匹配必拒。
- **ops import/export 域**（building/floor/workstation/asset/infoPoint/serverRoom/dedicatedLine/roomDevice 的 import/export 新码）与 **network 新码域**（device/credential/template 的逐动作拆分码）同理，凡 85-02/85-03 产生的新码在播种前均无 F 型菜单来源。

**定性 = 预期行为，非缺陷**：无授予渠道的权限本就不该放行；只读角色从"ANY-of 误放行"回到 403 正是 RBAC-01 的需求本体。

**缓解（87 执行）**：播种 migration 用 `GrantNewMenuToRolesHavingParent`（`internal/core/db/migrations/menu_grant_helpers.go:26`，已带幂等/参数化/PG-sqlite 双路径测试）给持父页面 C 型菜单的角色补授全部新动作码 → 现有管理类角色（xtgl/kjgl 等）零感知。85 内不做任何数据补授，**生产库零写入**。

## 二、归 87 移交清单（主责 phase：87）

| # | 移交项 | 内容 | 既有资产（可复跑） |
|---|--------|------|--------------------|
| 1 | **决定性对照实验** | ≥8 模块抽样：system 4 组（roles/menus/departments/posts）+ ops building + network device + network command + ad account，覆盖 Require→Strict / WithQuery 读组纯化 / 单码升 Strict 三种拆法；验证"只读角色调写接口 403 + 临时补授写码后 200" | #79 范式模板 `.planning/notes/2026-10-29-v124-perm-fixture.md`（单事务夹具 + 清理自检，只删夹具自身行）；超管 bypass 键 `role_key='admin'`（permission.go:281-302），夹具角色不得撞名 |
| 2 | **F 型菜单播种** | 85 产出的全部新码权限清单（各 plan Artifacts 节；system 4 组 ~14 码 + ops 域新码 + TS `PENDING_SEED_WHITELIST` 78 项全量播种后清空） | 播种 helper `GrantNewMenuToRolesHavingParent`（menu_grant_helpers.go:26）；白名单位置 `xingran-react-frontend/scripts/check-permission-codes.mjs` 的 `PENDING_SEED_WHITELIST`（87 播种 migration 落地后必须清空，脚本带非空警告防永久化） |
| 3 | **生产 6 角色回归面 SQL 实测** | 对每个拆分域跑 85-RESEARCH §B4 判定 SQL（列出"仅持组级读码、不持写码"的角色清单 = 拆分后 403 面），在生产 6 角色基线（admin/kjgl/xtgl/adyk/vm_user/user，admin bypass、user 写权限=0）上实测 | 判定 SQL 原文见 85-RESEARCH §B4 第 4 点；与 84 的 PG 遗留一起归 87 PG 实机验收（Lead 裁决 4），不在 85 阻塞 |

## 三、口径备忘

- **规模口径**：以 85-03 完成态源码重扫为准——**33 组写组 / 306 条写路由**（route-registry.json totals：mounts 207 / routes 697 / writeRoutes 306 / deferred 138）；Strict 挂载 188 条（其中写路由 164），`RequireStrictPermissions(` 源码引用 16 处（含 2 处 helper 复用点）。PLAN 里 "~110 条" 为早期估算，已被重扫口径取代。
- **tmp 缺口**：`tmp_routes.json` 有 system 4 组 0 条缺口，**不作事实源**；事实源 = `node scripts/scan-routes.cjs` 重扫产物 `scripts/route-registry.json`（入库 + TestRouteRegistryFreshness 同步守卫）。
- **SC1 校验脚本名偏差（checker ISSUE-3）**：ROADMAP SC1 字面为「未登记写路由 → `check-permission-codes.mjs` exit ≠ 0」，实际拦截职责 = **`check-route-registry.mjs`**（+ Go 侧 `TestRouteRegistryCoverage`/`TestRouteRegistryFreshness`）；`check-permission-codes.mjs` 只管 TS↔Go↔菜单码三方漂移（b/c 类），不拦路由登记。**验收时以 `node scripts/check-route-registry.mjs` 为 SC1 判据**。
- 工作区残留：`tmp_*` 系列为研究资产（勿删）；`asset_columns_schema.json` codegen 时间戳按惯例用 `git show HEAD:<file>` 复原，不入本 phase 提交。
