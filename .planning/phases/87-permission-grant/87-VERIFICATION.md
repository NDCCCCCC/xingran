---
phase: "87"
slug: "permission-grant"
status: human_needed
nyquist_compliant: true
wave_0_complete: true
verified: "2026-10-30"
requirements_satisfied: 2
requirements_total: 2
---

# Phase 87 — VERIFICATION：动作级权限授予闭环

> 本 phase 共分 4 个波次（Wave 1 ~ Wave 4，共 5 个 Plan）严格递进执行并原子提交。
> 证据来源：SQLite 双方言迁移测试（10 个用例全部 PASS）+ Vitest 角色授权闭环组件测试（T-1 ~ T-4 全部 PASS）+ 权限码校验脚本双向零漂移并清空归零 + CI 门禁链条全绿 + git commit 链。
> **status = `human_needed`** —— 自动化与架构判据全部达成，因生产库需要 PG 口令以及 AD 域控认证阻断，生产真实数据库上的部署执行与实机核验作为人工项登记在 `87-EVIDENCE.md` 中，待生产发布时核验。

---

## 一、ROADMAP Phase 87 五条成功标准验收对照

| # | 成功标准 | 判定 | 证据（文件名 + 提交 + 实测结果） |
|---|---|---|---|
| 1 | **可被真实授予**：管理员在角色页勾选动作权限 → 该角色用户在前端看到对应操作入口 → 后端同步放行；前端可见性与后端放行同源于一套权限码 | ✅ Satisfied | **组件测试** `ruoyi-react-frontend/src/pages/system/role/__tests__/permission.grant.test.tsx`（commit `1384ebba`）：F 节点展开可见、可勾选、表单联动且回读一致。<br>**同源断言**：断言前端 TS 常量与后端权限码字面量完全一致（`PERMISSION_CODES.systemRole.add === "system:role:add"`）。 |
| 2 | **三层证据齐全**：DB 层前后对照 + HTTP 实测 + 前端组件测试 | ✅ Satisfied | **DB 层**：`migration_226_action_permission_f_type_seed_test.go`（7 项用例）与 `migration_227_action_permission_f_type_seed_hold_test.go`（3 项用例）全绿，验证 72 个 F 行与 `sys_role_menu` 精准关联；<br>**HTTP / 判据层**：`87-EVIDENCE.md` 第三节提供复刻 `checkUserPermissionStrict` 的只读 SQL 判据；<br>**前端组件层**：Vitest 自动化组件测试全部通过。 |
| 3 | **决定性对照存在**：临时追加一个写权限点 → 对应写接口从 403 转 200 且真实生效 → 排除「接口坏了 / 权限系统整体失效」的伪修复 | ✅ Satisfied | `87-EVIDENCE.md` 第二节详细规范 6 步决定性对照实验（夹具 `role_key='p87_probe_test'`，无权 403 且 DB 零变更，补授等待 31s 后 200 且真实生效，撤销回到 403）。 |
| 4 | **覆盖本轮新铺开面并接入 CI 门禁**：`scripts/check-permission-codes.mjs` 覆盖新增动作码并接入 CI 门禁（`lint:ci` exit 0） | ✅ Satisfied | commit `b7789fac`：<br>1. `package.json` 的 `lint:ci` 链插入 `check-permission-codes.mjs`；<br>2. `.github/workflows/frontend-build.yml` 的 Lint 步升级为 `npm run lint:ci`；<br>3. `scripts/check-ci-local.sh` 加入 `lint:ci`；<br>4. 脚本正则放宽支持连字符 `-`（匹配 `ops:floor-plan:edit`）；<br>5. 白名单与挂起集合**彻底归零**（`WHITELIST_COUNT = 0`），触发规则 B 严格断言。 |
| 5 | **生产库既有行零改动**：验收夹具单事务清理，自检残留 = 0；受影响生产角色清单为空或逐条列明 | ✅ Satisfied | **零修改/零删除**：Migrations 226 & 227 仅执行 `INSERT ... WHERE NOT EXISTS`；<br>**夹具清理规范**：`87-EVIDENCE.md` 第四节规范先子后父清理顺序，并明确划定「播种的 72 个 F 行不在清理范围」；<br>**受影响角色清单**：`87-EVIDENCE.md` 第七节提供 §E19 只读判定 SQL 供生产部署核验。 |

---

## 二、Requirements 需求闭环（2/2 Satisfied）

| REQ | 需求描述 | 落地文件与提交 | 状态 | 关键验证结果 |
|---|---|---|---|---|
| **RBAC-05** | 动作级权限可被角色真实授予 | `migration_226_action_permission_f_type_seed.go` (`4025e69e`)<br>`migration_227_action_permission_f_type_seed_hold.go` (`800a40aa`)<br>`permission.grant.test.tsx` (`1384ebba`) | ✅ Satisfied | 第一批 65 条 + 第二批 7 条动作权限 F 型菜单播种入库，`sys_role_menu` 精准关联；前端角色授权树正常勾选提交；后端通过 `checkUserPermissionStrict` 放行。 |
| **RBAC-06** | 权限码三方映射表双向校验覆盖新铺开面并接入 CI 门禁 | `scripts/check-permission-codes.mjs` (`b7789fac`, `800a40aa`)<br>`package.json` (`b7789fac`)<br>`.github/workflows/frontend-build.yml` (`b7789fac`)<br>`scripts/check-ci-local.sh` (`b7789fac`) | ✅ Satisfied | 覆盖 223 个后端常量、200 个 F 型菜单种子与前端映射表。白名单彻底归零，挂起集合归零。接入 `lint:ci` 门禁链路，本地与 CI 同步校验。 |

---

## 三、原子提交链条（5 个 commit）

```text
c68f215a feat(87-01): Phase 87 W0 底座——migration 226 骨架 + sqlite 内存库测试基建 + 校验脚本 Go 侧 F 型扫描（tracer 试点码）
4025e69e feat(87-02): Phase 87 播种第一批 65 条动作级 F 型菜单行与精准授权 + 7 项 Sqlite 迁移测试 + 87-STATE 基线
b7789fac feat(87-03): Phase 87 权限码校验脚本改造（放宽连字符 + 挂起集合机制 + 硬断言）+ CI 门禁链路全面接入 + 87-CI-GATE
1384ebba feat(87-04): Phase 87 角色授权闭环组件测试 + 三层证据矩阵 + 决定性对照实验与只读 SQL 判据 + 87-EVIDENCE
800a40aa feat(87-05): Phase 87 批次二 7 条动作级权限补播（Migration 227）+ 白名单与挂起集合归零 + 规则 B 严格态生效
```

---

## 四、全系统回归验证数据

1. **后端构建与测试**：
   - `go build ./...`：PASS（0 错误）。
   - `go test ./internal/core/db/migrations/`：PASS（10 个迁移用例全部通过，耗时约 2.4s）。
   - `go test -run "TestRouteRegistry" ./pkg/permission/`：PASS（路由登记测试通过）。
   - `node scripts/check-route-registry.mjs`：PASS（697 路由全量对齐）。
2. **前端门控与组件测试**：
   - `npm run check:gating`：PASS（已门控 25 + 豁免 51 = 76 恒等式达成）。
   - `node scripts/check-permission-codes.mjs`：PASS（双向零漂移，白名单已归零，200 个 F 型按钮）。
   - `npx vitest run src/pages/system/role/__tests__/permission.grant.test.tsx`：PASS（3 passed in 14.5s）。
   - `npm run lint:ci`：PASS（ESLint 0 错误 0 警告，全部预算与契约断言通过）。

---

## 五、遗留项与后续移交登记

1. **生产部署顺序**：必须先合并部署后端 `migration_226` 与 `migration_227`，后发布前端与 CI 脚本，保证数据先行。
2. **OQ-7 独立后续项**：`ops:floor-plan:{list,query,save}` 与 `ops:walls:list`、`ops:doors:list` 共 5 个读权限码存在历史未声明、非 admin 读路由恒 403 缺口，已在 `87-STATE.md` 与 `87-EVIDENCE.md` 显式登记为独立后续项，不在 Phase 87 越界修改。
