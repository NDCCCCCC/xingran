---
slug: column-config-reset-after-save
status: resolved
trigger: (legacy session, see body)
created: 2026-06-25
updated: 2026-06-25
session_type: bug
---
# Debug Session: Column Configuration Reset After Save

**Status:** resolved
**Trigger:** 列设置保存后刷新页面就还原了，怀疑修改没有持久化，数据库没有更新。需要系统性调试这个问题。
**Created:** 2026-06-09
**Updated:** 2026-06-09

---

## Symptoms

### Expected Behavior
- 列配置保存后应该持久化到数据库
- 刷新页面或重新打开时，应该加载用户保存的配置
- 每个用户的列配置应该独立保存

### Actual Behavior
- 保存后显示成功提示
- 刷新页面后，列配置恢复到默认状态
- 保存的配置丢失

### Error Messages
- **观察到：** 显示保存成功提示
- **实际：** 刷新后配置丢失

### Timeline
- **何时开始：** 刚实现的新功能，从未正常工作过

### Reproduction Steps
1. 打开资产列表页面
2. 修改列配置（显示/隐藏列，调整顺序等）
3. 点击保存
4. 看到保存成功提示
5. 刷新页面
6. 列配置恢复到默认状态

---

## Current Focus

**Hypothesis:** 前端loadConfig函数的defaultColumns依赖导致useCallback每次重新创建，触发useEffect重复执行
**Test:** 验证useCallback的依赖项是否稳定
**Expecting:** defaultColumns虽然是常量数组，但传递给hook时可能被视为新的引用
**Next Action:** 检查defaultColumns的传递方式，确认是否需要使用useMemo包装
**Reasoning Checkpoint:** 已确认根因
**TDD Checkpoint:** 未设置

---

## Evidence

- timestamp: 2026-06-09 - Initial symptom collection completed
- timestamp: 2026-06-09 - 分析前端代码发现useColumnConfig的useEffect依赖项包含loadConfig，这可能导致无限循环或配置重置
- timestamp: 2026-06-09 - 前端代码分析：
  - useColumnConfig.ts第219行：useEffect(() => { loadConfig(); }, [loadConfig]);
  - loadConfig依赖于pageKey、defaultColumns、enableCache
  - 这意味着每次这些值变化时都会重新加载配置
- timestamp: 2026-06-09 - 后端代码分析：
  - column_config_service.go第59-64行：Save函数使用Unscoped().Delete()永久删除旧配置
  - 第66-79行：然后批量插入新配置
  - 这是一个事务操作，应该保证原子性
- timestamp: 2026-06-09 - **ROOT CAUSE IDENTIFIED**：
  1. **useCallback依赖项问题（根因）**：
     - useColumnConfig.ts第94行：`const { pageKey, defaultColumns, enableCache = true } = options;`
     - 每次组件渲染时，从options对象解构会创建新的defaultColumns引用
     - 第136行：`loadConfig`的useCallback依赖于[pageKey, defaultColumns, enableCache]
     - 由于defaultColumns引用每次都不同，useCallback每次都重新创建loadConfig函数
     - 第219-221行：useEffect监听[loadConfig]，由于loadConfig每次都是新函数，useEffect每次都执行
     - **关键问题**：在saveConfig保存成功后，useEffect被触发，重新执行loadConfig，可能覆盖刚保存的配置
  2. **transformToColumnConfig函数逻辑问题**：
     - 第75-85行：遍历userConfigs时使用index作为order
     - 但第88-90行：又重新计算order并排序
     - 这导致order字段被重复计算，可能丢失用户自定义的顺序
  3. **loadConfig函数缓存逻辑问题**：
     - 第104-108行：从localStorage加载缓存后立即setConfig
     - 但第111-128行：继续执行API调用，覆盖缓存结果
     - 这导致缓存意义降低，且可能造成配置闪烁

---

## Eliminated

- 后端数据库持久化：后端Save函数使用事务正确保存数据，无问题
- API路由注册：路由正确注册在/internal/api/router.go第125行
- 数据库查询：GetByPageKey正确按display_order排序查询

---

## Resolution

**ROOT CAUSE (Final):** 前端 API 客户端 `columnConfigApi.ts` 调用了错误的 API 端点。

**Initial Hypothesis (Incorrect):** useColumnConfig hook的useCallback依赖数组包含defaultColumns，导致函数重新创建和 useEffect 重复执行。

**Actual Root Cause:** `columnConfigApi.ts` 中的所有 API 调用都使用了错误的端点：
- 错误端点：`/system/settings` （系统设置 API）
- 正确端点：`/system/column-config` （列配置 API）

这导致列配置保存请求被发送到系统设置 API，虽然返回"保存成功"，但实际上没有保存到列配置表中。

**Fix Applied:**

1. ✅ **API 端点修复**（`columnConfigApi.ts` 第26-38行）：
   ```typescript
   // 修复前
   get<UserColumnConfig[]>(`/system/settings/${pageKey}`)
   post('/system/settings', data)
   del(`/system/settings/${pageKey}`)

   // 修复后
   get<UserColumnConfig[]>(`/system/column-config/${pageKey}`)
   post('/system/column-config', data)
   del(`/system/column-config/${pageKey}`)
   ```

2. ✅ **useEffect依赖优化**（`useColumnConfig.ts` 第218-224行）：
   - 从 `[loadConfig]` 改为 `[pageKey]`
   - 这是一个防御性优化，避免潜在的问题

3. ✅ **useCallback依赖优化**（`useColumnConfig.ts`）：
   - loadConfig 和 resetConfig 移除 `defaultColumns` 依赖
   - 这是一个防御性优化

4. ✅ **transformToColumnConfig函数优化**：
   - 使用 `displayOrder` 字段排序
   - 这是一个代码质量优化

**Verification:**
- ✅ TypeScript 类型检查通过
- ✅ API 端点已修复为正确的 `/system/column-config`

**Files Changed:**
- `xingran-react-frontend/src/lib/columnConfigApi.ts`（关键修复）
- `xingran-react-frontend/src/hooks/useColumnConfig.ts`（防御性优化）
