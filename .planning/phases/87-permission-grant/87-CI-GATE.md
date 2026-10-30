# Phase 87 门禁机制与规范：权限码三方校验与 CI 接入

> 记录 Phase 87（RBAC-06）权限码校验脚本的硬断言设计、挂起集合机制以及全流程 CI 门禁接入拓扑。
> 更新日期：2026-10-30

---

## 一、挂起集合机制（Phase 87 · OQ-6）

在 Phase 87 中，第一批 65 条动作权限码已通过 Migration 226 成功播种，但以下 7 条动作权限码的父菜单（CAD 平面图编辑器、墙体和门的实际业务管理入口页）尚未人工确认（推断为 `ops:floor:list` 楼层管理，但可能属于「楼宇空间」等页面）：

- **挂起码清单（恰好 7 条）**：
  1. `ops:walls:add`
  2. `ops:walls:edit`
  3. `ops:walls:delete`
  4. `ops:doors:add`
  5. `ops:doors:edit`
  6. `ops:doors:delete`
  7. `ops:floor-plan:edit`
- **裁定依据**：用户已于 2026-10-30 明确裁定（OQ-6）——「先播 65 条，7 条挂起待确认」。猜错会导致菜单树展示错位以及 Step 2 子菜单继承方向错位，因此挂起待确认优于盲目猜测。
- **实现载体**：在 `ruoyi-react-frontend/scripts/check-permission-codes.mjs` 中显式声明为常量：
  ```javascript
  const PENDING_SEED_HOLD_CODES = [
    "ops:walls:add",
    "ops:walls:edit",
    "ops:walls:delete",
    "ops:doors:add",
    "ops:doors:edit",
    "ops:doors:delete",
    "ops:floor-plan:edit",
  ];
  const PENDING_SEED_HOLD_SET = new Set(PENDING_SEED_HOLD_CODES);
  ```

---

## 二、硬断言三条规则（RBAC-06 防永久化）

为防止白名单成为永久豁免的逃生通道，校验脚本 `check-permission-codes.mjs` 引入了不可规避的硬断言规则，位于 `errors.length > 0` 判定之前，触犯即 `exit 1`：

1. **规则 A（严格限定）**：
   - 白名单每一条都必须在 `PENDING_SEED_HOLD_SET` 挂起集合内。
   - 挂起集合之外的任何一条残留白名单项均会触发 `errors.push` 并导致进程退出码为 1。第一批 65 条既已播种，绝不许在白名单中继续豁免。
2. **规则 B（收口自动升级）**：
   - 当 `PENDING_SEED_HOLD_CODES.length === 0`（即第二批补播完成且挂起集合清空）时，若 `PENDING_SEED_WHITELIST.length > 0`，立即报错退出。
   - 确保第二批补播完成后，脚本自动升级为「白名单必须归零」的最终严格态。
3. **规则 C（已播种自检警告）**：
   - 若白名单里的码已经出现在 `F_TYPE_SEED` 块中（说明已实际播种），脚本控制台输出警告提示开发者及时从白名单中移除，防止陈旧豁免残留。

---

## 三、白名单归零的收口条件（遗留项登记）

- **归零收口条件**：在 **Plan 87-05** 执行第二批补播（通过 `migration_227` 播种 7 条动作码）完成后，开发者必须同步将 `scripts/check-permission-codes.mjs` 中的 `PENDING_SEED_HOLD_CODES` 和 `PENDING_SEED_WHITELIST` **同时完全清空（归零）**。
- **状态追踪**：在白名单归零之前，Phase 87 的 RBAC-06 门禁处于「允许 OQ-6 挂起」的过渡态。白名单彻底归零是 Phase 87 完全验收闭环的最终充要条件。

---

## 四、门禁链路拓扑与接线

RBAC-06 要求「校验脚本覆盖本轮新铺开的面并接入 CI 门禁」。接线覆盖了本地开发、CI 运行与本地对齐脚本三处：

### 1. `lint:ci` 完整执行链路（`ruoyi-react-frontend/package.json`）
```bash
eslint . --max-warnings 0 && \
node scripts/assert-lint-budget.mjs && \
node scripts/check-exemption-reasons.mjs && \
node scripts/check-gating-coverage.mjs && \
node scripts/check-permission-codes.mjs && \  # <--- 本 phase 新增接线点
node scripts/check-hardcoded-colors.mjs && \
node scripts/css-contract-check.js && \
node ../scripts/check-route-registry.mjs
```

### 2. GitHub Actions 生产门禁（`.github/workflows/frontend-build.yml`）
将 Lint 步骤由松散的 `npm run lint` 升级为严谨的 `npm run lint:ci`：
```yaml
- name: Lint (lint:ci)
  run: npm run lint:ci
```

### 3. 本地 CI 对齐脚本（`scripts/check-ci-local.sh`）
在前端构建检查循环中加入 `lint:ci`：
```bash
for s in format:check lint lint:ci type-check test build; do
```
保障开发者在本地运行 `check-ci-local.sh` 通过时，提交至 GitHub 亦必全绿（「本地绿 = CI 绿」）。

---

## 五、红线声明与不变量保持

本 phase 的门禁接入完全遵循仓库既有架构红线：
1. **不改动权限码真相源**：`ruoyi-react-frontend/src/constants/permissionCodes.ts` 零 diff。
2. **不改动 86 线门控恒等式**：`ruoyi-react-frontend/scripts/gating-exemptions.json` 与 `ruoyi-react-frontend/scripts/check-gating-coverage.mjs` 零 diff，`EXPECTED_PAGE_COUNT=76` / `EXPECTED_EXEMPTION_COUNT=51` 恒等式 `25 + 51 = 76` 持续有效。
3. **零新增依赖**：不添加任何新的 npm / Go 包依赖，不自创独立 script 命令。
