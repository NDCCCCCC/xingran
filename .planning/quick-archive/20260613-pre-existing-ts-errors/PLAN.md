# Quick Task: 清理 21 个预先存在的 TypeScript 错误

**Slug:** `20260613-pre-existing-ts-errors`
**Date:** 2026-06-13
**Status:** 📋 Planned
**Author:** Claude (gated by user)

---

## 🎯 目标

清理 `npx tsc -b --noEmit` 暴露的 21 个预先存在 TypeScript 错误（13 文件），使构建在 CI/CD 中能成功通过类型检查。

## 📊 现状

| 项 | 数值 |
|----|------|
| 错误总数 | 21 |
| 涉及文件数 | 13 |
| 验证命令 | `npx tsc -b --noEmit` |
| 当前结果 | 21 errors, EXIT=0 (build 报错但脚本返回 0 是 vite 缓存问题) |
| 触发来源 | 调试 columnWidth 时发现（与 columnWidth 修改无关，stash 前后错误一致） |

## 🔍 根因分类（来自调研）

### Category A — Stale imports（2 错 / 2 文件）🟢 LOW

| 文件 | 行 | 错误 | 修复 |
|------|----|----|------|
| `src/lib/api/networkApi.ts` | 1 | `Cannot find module './api'` | 改为 `'../api'` |
| `src/pages/network/mac/trajectory/index.tsx` | 2 | `Cannot find module './trajectory'` | 改为 `'../trajectory'` |

### Category B — Missing exports / type drift（5 错 / 4 文件）🟡 MEDIUM

| 文件 | 行 | 错误 | 修复 |
|------|----|----|------|
| `src/types/operations.ts` | 283 | `Cannot find name 'PageParams'` | 补 import |
| `src/lib/vdiApi.ts` | 21 | `No exported member 'VMIPConfigRequest'` | 移除未用 import + 改返回类型 `BaseResponse<void>` |
| `src/types/index.ts` | 20 | `Module './operations' has already exported a member named 'DeviceStatus'` | 移除 `operations.ts` 的 `DeviceStatus` export + 重命名为 `OpsDeviceStatus` |
| `src/pages/vdi/VirtualMachineList/index.tsx` | 711 | `unknown property 'resource_group_id'` | 在 `CreateVMRequest` 补 `resource_group_id?: string` |
| `src/pages/vdi/VirtualMachineList/index.tsx` | 70 | `useRef` 缺初始值 | 加 `undefined` |

### Category C — API/field signature drift（4 错 / 2 文件）🟡 MEDIUM

| 文件 | 行 | 错误 | 修复 |
|------|----|----|------|
| `src/pages/vdi/VirtualMachineDetail/index.tsx` | 251 | `Property 'cpu' does not exist` | `vm.cpu` → `vm.cpu_number` |
| `src/components/operations/WorkstationDeviceTable/index.tsx` | 51,52,53 | `getManual/getAD/getAsset` 方法不存在 | 改用现有 `getByWorkstation` + 本地过滤 |
| `src/components/operations/WorkstationDeviceTable/index.tsx` | 99 | `deviceName` 字段嵌套错误 | `result.data.deviceName` → `result.data.deviceName`（待核实） |
| `src/components/operations/WorkstationDeviceTable/index.tsx` | 181 | `setPrimaryAndSave` 方法不存在 | 拆分为 `setPrimary` + `update` |

### Category D — Type-cast / refactor pattern issues（4 错 / 4 文件）🟢 LOW

| 文件 | 行 | 错误 | 修复 |
|------|----|----|------|
| `src/components/charts/EChartsWrapper.tsx` | 43 | `Ref<unknown> not assignable` | `as Ref<unknown>` → `as Ref<EChartsReact>` |
| `src/components/network/MACTrajectoryChart.tsx` | 172 | `SyntheticEvent not assignable to Error` | `onError` 类型改为 `(error: unknown) => void` |
| `src/components/three/BuildingScene.tsx` | 46,52 | `Spread types may only be created from object types` | `as never` → `as Record<string, unknown>` |
| `src/components/table/VDIRow.tsx` | 90,91 | `Unintentional comparison` | 重构为显式 boolean |

### Category E — Trivial spread（1 错 / 1 文件）🟢 LOW

| 文件 | 行 | 错误 | 修复 |
|------|----|----|------|
| `src/pages/operations/assets/index.tsx` | 299 | `Spread types` | `(searchValues as object)` 或 `as Record<string, unknown>` |

---

## 🛠️ 执行计划（建议分 3 批）

### Batch 1 — Atomic One-liners（7 错 / 7 文件）🟢 预计 15-20 min

只做机械性最小修改，无业务逻辑变更。每个修复独立提交以便回滚：

1. `networkApi.ts(1)`: `'./api'` → `'../api'`
2. `trajectory/index.tsx(2)`: `'./trajectory'` → `'../trajectory'`
3. `types/operations.ts(283)`: 补 `PageParams` 到 import
4. `VirtualMachineList/index.tsx(70)`: `useRef(undefined)` 初始值
5. `EChartsWrapper.tsx(43)`: `as Ref<EChartsReact>`
6. `BuildingScene.tsx(46,52)`: `as Record<string, unknown>`
7. `assets/index.tsx(299)`: `(searchValues as object)` cast

**验证**: `npx tsc -b --noEmit` → 14 errors（21 - 7 = 14）

### Batch 2 — Type Definition Updates（3 错 / 3 文件）🟡 预计 25-35 min

需要先看代码上下文再补类型：

1. `lib/vdiApi.ts(21)`: 删除 `VMIPConfigRequest` import + 改 `configIP` 函数返回 `BaseResponse<void>`
2. `types/vdi.ts` `CreateVMRequest` 接口: 补 `resource_group_id?: string`
3. `types/index.ts(20)` & `types/operations.ts`: 移除 `DeviceStatus` 重导出冲突，operations 中重命名为 `OpsDeviceStatus`（如果 grep 找到内部使用则同步改名）

**验证**: `npx tsc -b --noEmit` → 11 errors

### Batch 3 — Field Renames & API Reconciliation（11 错 / 3 文件）🟠 预计 35-50 min

需要看调用方上下文：

1. `VirtualMachineDetail/index.tsx(251)`: `vm.cpu` → `vm.cpu_number`
2. `VDIRow.tsx(90,91)`: 重构 disabled 计算逻辑为显式 boolean
3. `WorkstationDeviceTable/index.tsx` (5 错):
   - 51-53: 调查实际 API 形态，可能需要用 `getByWorkstation` + 本地过滤
   - 99: 核实 `deviceName` 嵌套层级
   - 181: `setPrimaryAndSave` 拆分为 `setPrimary` + `update`
4. `MACTrajectoryChart.tsx(172)`: `onError` 类型 `(error: unknown) => void`

**验证**: `npx tsc -b --noEmit` → **0 errors** 🎉

---

## ✅ 最终验证

```bash
# 全量检查
npx tsc -b --noEmit
# 预期：EXIT=0, 0 errors

# Build 完整流程
npm run build
# 预期：tsc -b 阶段无错，vite build 成功
```

## 🚫 范围约束

按 CLAUDE.md Scope Constrainment 原则：

- ✅ **仅修复类型错误**，不改业务逻辑
- ✅ **不改后端 API**
- ✅ **最小代码 diff**（优先一行/几行修改）
- ✅ **不改文件结构**（不删除文件，不重构布局）
- ❌ **不**做顺手清理（无关注释、风格、命名）
- ❌ **不**改测试通过的功能代码
- ❌ **不**"fix"未在调研中报告的错误

## 📁 文件清单（13 个）

```
src/lib/api/networkApi.ts                                          # Cat A
src/pages/network/mac/trajectory/index.tsx                        # Cat A
src/types/operations.ts                                            # Cat B
src/lib/vdiApi.ts                                                  # Cat B
src/types/index.ts                                                 # Cat B
src/types/vdi.ts                                                   # Cat B (新增字段)
src/pages/vdi/VirtualMachineList/index.tsx                        # Cat B + E
src/pages/vdi/VirtualMachineDetail/index.tsx                      # Cat C
src/components/operations/WorkstationDeviceTable/index.tsx        # Cat C
src/components/charts/EChartsWrapper.tsx                          # Cat D
src/components/network/MACTrajectoryChart.tsx                     # Cat D
src/components/three/BuildingScene.tsx                            # Cat D
src/components/table/VDIRow.tsx                                   # Cat D
src/pages/operations/assets/index.tsx                             # Cat E
```

## 📝 提交策略

每 Batch 一个 commit，遵循 conventional commits：

```
fix(types): [Batch 1] clean up 7 stale imports and one-liner TS errors
fix(types): [Batch 2] resolve type drift in vdi/types modules (3 errors)
fix(types): [Batch 3] fix field renames and WorkstationDeviceTable API drift (11 errors)
```

## 📚 参考文档

- `D:\CODE\ClaudeCode\xingran-go-backend\.planning\STATE.md` — 当前项目状态
- `D:\CODE\ClaudeCode\xingran-go-backend\CLAUDE.md` — Scope Constrainment 原则
- `D:\CODE\ClaudeCode\xingran-go-backend\xingran-react-frontend\tsconfig.json` — Project References 根配置
- `D:\CODE\ClaudeCode\xingran-go-backend\xingran-react-frontend\package.json` — `type-check` 脚本（已识别为假绿灯，**待用户决定是否修复**）