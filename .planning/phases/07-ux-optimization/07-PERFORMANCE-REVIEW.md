---
checked_at: 2026-04-21
total_files: 516
status: issues_found
critical: 2
high: 4
medium: 6
low: 3
---

# 前端性能检查报告

基于 Vercel React 最佳实践对前端代码进行全面检查。

## 统计概览

| 指标 | 数量 | 说明 |
|------|------|------|
| 总文件数 | 516 | TypeScript/TSX 文件 |
| useEffect 使用 | 269 | 需检查依赖数组 |
| useMemo/useCallback | 406 | 优化使用良好 |
| 内联样式 | 1127 | ⚠️ 过多，影响性能 |
| Promise.all 使用 | 31 | 并行化良好 |
| 懒加载使用 | 10+ | Widget 组件已优化 |

## 关键问题 (CRITICAL)

### 1. Barrel Imports 导致 Bundle 膨胀

**规则:** `bundle-barrel-imports`

**问题文件:**
- `src/utils/iconUtils.tsx:6` - `import * as Icons from '@ant-design/icons'`
- `src/pages/operations/building-spaces-3d/components/BuildingModel3D.tsx:9` - `import * as THREE from 'three'`

**影响:** 导入整个图标库和 Three.js，导致 bundle 大幅膨胀。

**修复建议:**
```typescript
// 错误
import * as Icons from '@ant-design/icons';

// 正确 - 按需导入
import { UserOutlined, SettingOutlined, LogoutOutlined } from '@ant-design/icons';
```

### 2. 串行 await 导致瀑布流

**规则:** `async-parallel`

**问题文件:**
- `src/pages/knowledge/articles/index.tsx:123`

**影响:** 独立的异步操作串行执行，增加总等待时间。

**修复建议:**
```typescript
// 错误
await operation1();
await operation2();

// 正确 - 并行执行
await Promise.all([operation1(), operation2()]);
```

## 高优先级问题 (HIGH)

### 3. 过多内联样式 (1127处)

**规则:** `rendering-hoist-jsx`

**影响:** 
- 每次渲染创建新对象，触发子组件重渲染
- 无法利用 CSS 缓存
- 增加内存压力

**修复建议:**
```typescript
// 错误
<div style={{ display: 'flex', justifyContent: 'center', padding: '16px' }}>

// 正确 - 提取到 CSS
<div className="flex-center padding-4">
```

### 4. 链式 filter/map 操作

**规则:** `js-combine-iterations`

**问题文件:**
- `src/pages/system/notice/index.tsx:149-151` - 三次 filter + map
- `src/design-system/components/ColorSwitcher.tsx:320,347`

**影响:** 多次遍历数组，O(n*m) 复杂度。

**修复建议:**
```typescript
// 错误
const depts = items.filter(t => t.type === 'dept').map(t => t.id);
const roles = items.filter(t => t.type === 'role').map(t => t.id);

// 正确 - 单次遍历
const { depts, roles } = items.reduce((acc, t) => {
  if (t.type === 'dept') acc.depts.push(t.id);
  if (t.type === 'role') acc.roles.push(t.id);
  return acc;
}, { depts: [], roles: [] });
```

### 5. useEffect 缺少依赖数组检查

**规则:** `rerender-dependencies`

**统计:** 269 个 useEffect 使用

**建议:** 检查每个 useEffect 的依赖数组是否完整，避免闭包陷阱。

### 6. 大型第三方库未按需加载

**问题文件:**
- `src/pages/duty/holidays/utils.tsx:8` - `import * as XLSX from 'xlsx'`
- `src/pages/duty/management/utils/excelUtils.ts:6` - 同上

**修复建议:** 使用动态导入
```typescript
const exportToExcel = async (data) => {
  const XLSX = await import('xlsx');
  // 使用 XLSX
};
```

## 中等优先级问题 (MEDIUM)

### 7. 缺少 React.memo 优化

**建议:** 对以下组件添加 memo：
- 列表项组件（频繁重渲染）
- 接收复杂 props 的组件

### 8. 事件处理函数未使用 useCallback

**影响:** 每次渲染创建新函数，可能导致子组件不必要的重渲染。

### 9. 缺少 Suspense 边界

**规则:** `async-suspense-boundaries`

**建议:** 为懒加载组件添加 Suspense 边界和 fallback。

### 10. localStorage 未版本化

**规则:** `client-localstorage-schema`

**建议:** 为 localStorage 数据添加版本号，便于迁移和清理。

### 11. 缺少 passive 事件监听器

**规则:** `client-passive-event-listeners`

**建议:** 对 scroll/touch 事件使用 `{ passive: true }`。

### 12. SVG 动画性能

**规则:** `rendering-animate-svg-wrapper`

**建议:** 动画 SVG 外层的 div wrapper，而非 SVG 元素本身。

## 低优先级问题 (LOW)

### 13. 可使用 Set/Map 优化查找

**规则:** `js-set-map-lookups`

**场景:** 频繁的 `array.includes()` 或 `array.find()` 可改用 Set。

### 14. 可使用 LRU 缓存

**规则:** `server-cache-lru`

**建议:** 对频繁计算的派生数据使用 LRU 缓存。

### 15. 可使用 startTransition

**规则:** `rerender-transitions`

**建议:** 对非紧急状态更新使用 `startTransition`。

## 良好实践

✅ **懒加载:** Widget 组件使用 `lazy()` 动态导入
✅ **并行化:** 31 处使用 `Promise.all`
✅ **记忆化:** 406 处使用 `useMemo`/`useCallback`
✅ **类型安全:** TypeScript 全面应用

## 改进优先级

1. **立即修复:** Barrel imports (bundle 大小影响最大)
2. **短期优化:** 内联样式提取 (性能影响显著)
3. **中期优化:** 链式迭代合并、动态导入大型库
4. **长期优化:** 添加 memo、useCallback、Suspense 边界

---

*检查时间: 2026-04-21*
*检查工具: Vercel React Best Practices*
