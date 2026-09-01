---
status: awaiting_human_verify
trigger: "github上的ci工作流一直失败，请排查原因并修复"
created: 2026-08-30
updated: 2026-08-30
---

# Debug Session: ci-lint-hardcoded-ip

## Current Focus

**Verification passed.** Fix applied and confirmed locally. Awaiting user confirmation of CI result.

## Symptoms

**Expected behavior:**
push 到 main 后 `ci` workflow 全绿，`frontend` job 的 Lint → Type check → Test → Coverage gate → Build 全部通过，下游 `deploy` workflow 得以触发。

**Actual behavior:**
`frontend` job 在 **Lint** 步骤 exit code 1，后续 Type check / Test (coverage) / Coverage gate / Build 全部被跳过。

6 个 error 全部为同一规则 `no-restricted-syntax`，message: `禁止硬编码内网 IP,应通过 VITE_API_BASE_URL / VITE_WS_BASE_URL 等环境变量配置`

命中位置（全部是测试文件）：

| 文件 | 位置 |
|------|------|
| `src/pages/vdi/VirtualMachineList/__tests__/index.render.test.tsx` | 55: `ipAddress: "10.0.0.5"` |
| `src/pages/vdi/VirtualMachineDetail/__tests__/index.render.test.tsx` | 48: `ipAddress: "10.0.0.5"` |
| `src/pages/operations/rpa/workers/__tests__/columns.test.tsx` | 21: `ipAddress: "10.0.0.1"` |
| `src/pages/network/discoveries/hooks/useDiscoveryData.test.tsx` | 69: `{ ip: "10.0.0.1" }` |
| `src/pages/network/discoveries/__tests__/index.render.test.tsx` | 34: `ipRange: "10.0.0.0/24"` |
| `src/pages/network/devices/__tests__/index.render.test.tsx` | 47: `ipAddress: "10.0.0.1"` |

## Evidence

- timestamp: 2026-08-30
  observation: CI run 33309704912 frontend job Lint 步骤 exit 1，6 errors 全部 no-restricted-syntax 内网 IP 规则
  source: `gh run view 33309704912 --log-failed`

- timestamp: 2026-08-30
  observation: backend job 同 run 内通过，说明失败面仅限前端 lint
  source: `gh run view 33309704912`

- timestamp: 2026-08-30
  observation: CI lint 步骤在 ci.yml line 161 定义为 `npm run lint`（无 `--max-warnings`）
  source: `ci.yml:161`

- timestamp: 2026-08-30
  observation: 6 个命中文件的问题行均为 mock fixture 数据，非真实源码硬编码
  source: Grep across 6 flagged files

- timestamp: 2026-08-30
  observation: eslint.config.js 的 `no-restricted-syntax` 规则无测试文件 override，全局应用
  source: `eslint.config.js`

- timestamp: 2026-08-30
  observation: `npm run lint` 本地执行后：0 errors, 984 warnings（原来 6 errors, 958 warnings）
  source: local lint run

- timestamp: 2026-08-30
  observation: 单独 lint 6 个问题文件：0 errors，30 warnings（均为 @typescript-eslint/no-explicit-any）
  source: local lint run on 6 files

## Resolution

root_cause: eslint.config.js 的 `no-restricted-syntax` 内网 IP 规则（error 级）未对测试文件做 override；Phase 88 新增的渲染测试 mock fixture 使用 `10.0.0.x` 等 RFC1918 私网段 IP 触发 lint error；CI lint 步骤无 `--max-warnings` 标志，error 直接 exit 1

fix: 在 eslint.config.js 中插入 override config，对 `**/__tests__/**`、`**/*.test.*`、`**/*.spec.*` 文件将 `no-restricted-syntax` 规则关闭

verification: 本地 `npm run lint` exit 0，0 errors；CI run 33313281998 确认 Format check ✓ / Lint ✓ / Type check ✓
files_changed: ["xingran-react-frontend/eslint.config.js", "xingran-react-frontend/src/pages/monitor/cache/__tests__/index.render.test.tsx", "xingran-react-frontend/src/pages/monitor/server/__tests__/index.render.test.tsx"]
commit: dcf8374

## 后继问题（原始 bug 已解，此为被遮蔽的下一层）

**状态：已修复，随 `159d35d` 落地**

Lint 转绿后，CI 前进到 **Test (coverage)** 步骤并稳定失败。该步骤此前被 lint 遮蔽 15+ 次，从未在 CI 上执行过。

失败特征（CI run 33313281998，完整跑完未被抢占）:
```
Test Files  315 passed (315)      ← 断言全部通过
Unhandled Errors: 18 × Unhandled Rejection   ← vitest 因此 exit 非零
```

即：没有任何测试断言失败，vitest 是被未捕获的 promise rejection 判定为失败的。

**为何 15+ 次 push 无人察觉**：`vitest run --coverage | tail -25` 的退出码来自 tail 而非 vitest（bash 管道默认取末位命令退出码），且 tail 会截掉 "Unhandled Errors" 段。本地看起来"绿"，实际 vitest 已 exit 1。

### 修复内容（4 处，2 生产 + 2 测试）

**生产代码 — 真实缺陷，非仅测试问题：**

1. `src/hooks/useTableManager.ts` — `loadData` 只有 `finally` 没有 `catch`。调用方多为
   fire-and-forget（`handleRefresh` / `handleReset` / 分页回调），请求失败时 rejection
   逃逸成 unhandled，且用户看不到任何错误提示。补 `catch`：`handleApiError` 提示 + 清空表格。
   **线上 API 故障时同样受益** —— 这不只是测试问题。

2. `src/components/layout/sidebar.tsx` — useEffect 里 `fetchMenus()` 是 fire-and-forget。
   `menuStore.fetchMenus` 末尾的 `throw error` **必须保留**（`pages/login/index.tsx:71`
   用 `await Promise.all([fetchMenus(), ...])` 依赖它显示登录失败），故修在调用点而非 store 层。

**测试 mock — 与接口契约不符：**

3. `src/pages/__tests__/batch58.moreOps.render.test.tsx`
   - WorkorderStatistics：URL 写成 `/workorder/statistics/list`，实际是 `/workorder/statistics`
     （见 `getWorkOrderStatistics`），**mock 从未命中**；且 data 需满足 `WorkOrderStatistics`
     契约 —— 空对象 `{}` 是真值会穿过组件的 `stats &&` 守卫，令 `Object.entries(stats.byPriority)`
     对 undefined 抛错。
   - WorkorderCategories：契约是 `BaseResponse<WorkOrderCategory[]>`，data 为数组；误用
     `{list,total}` 会让 categories 变成对象，渲染期 `flatCategories` 的 `list.forEach` 抛错。

4. `src/pages/__tests__/batch54.modalSweep.test.tsx` — 补 `/workorder/categories/enabled` mock，
   否则 `buildCategoryTree` 的 `categories.map` 抛错。

验证：Unhandled Rejection **18 → 0**，原有 Unhandled Error **4 → 0**；lint 0 errors /
prettier 全绿 / tsc 通过。

### 提交事故记录

这 4 处修复原本要以独立 fix commit 提交，但 commit 时并行会话推进了 HEAD，
git 报 `fatal: cannot lock ref 'HEAD'`。lint-staged 的 stash/restore 结束后，
改动被并行会话的 `159d35d test(88): batch206 captcha service` 一并 commit 并 push。
**修复内容已生效，但原 commit message 中的修复理由丢失** —— 故完整记录于本文件。

## 附带发现（非阻断，供后续参考）

1. **该 lint 规则的实际拦截面比宣称的窄。** `src/components/asset/reconciliation/ExceptionRuleForm.tsx` 有 `placeholder="192.168.0.0/16"`，不在 ignores 列表内，却不报错 — selector `Literal[value=/.../]` 命中对象/变量字面量，抓不到 JSX 属性值与模板字符串。"防止新增硬编码"的防护力有限。

2. **CI concurrency 抢占。** 并行会话高频 push 时，每个 run 都被后续 push 以 `Canceling since a higher priority waiting request exists` 取消，导致连续多个 run 显示 cancelled 而非真实结果。判定 CI 状态需找未被抢占、完整跑完的 run。

3. **管道退出码陷阱。** `vitest run --coverage | tail -25` 的 exit code 是 tail 的，不是 vitest 的；且 tail 会截掉 "Test Files" 与 "Unhandled Errors" 段。验证测试结果必须直接取 vitest 退出码。这是本次 15+ 次 CI 连红无人察觉的直接原因。

4. **coverage/.tmp 并发冲突。** 两个 vitest 实例共用同一 `coverage.reportsDirectory` 会互删临时文件，报 `Something removed the coverage directory`。多会话并行时验证测试应去掉 `--coverage`。

5. **同工作树多会话的提交竞争。** 并行会话高频 commit 时，本会话未提交的改动可能被对方 `git add -A` 卷入其 batch commit，或令本会话 commit 失败于 `cannot lock ref 'HEAD'`。改动完成后应尽快提交。


