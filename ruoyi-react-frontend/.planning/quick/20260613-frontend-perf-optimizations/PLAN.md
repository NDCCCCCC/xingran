# 前端性能优化

## 优化目标

优化三个前端性能问题，减少 bundle size 和提升请求性能：

### 1. Ant Design 图标按需导入 (MEDIUM)

**问题：** `src/utils/iconUtils.tsx:6` 导入整个图标库 (200-400KB)
```typescript
// ❌ 问题：导入整个图标库
import * as Icons from '@ant-design/icons';
```

**解决方案：** 改为按需导入，只导入 `iconCategories` 中实际使用的 99 个图标
```typescript
// ✅ 修复：按需导入图标
import { DashboardOutlined, HomeOutlined, ... } from '@ant-design/icons';

const iconMap: Record<string, ComponentType<any>> = {
  DashboardOutlined,
  HomeOutlined,
  // 只包含实际使用的图标
};
```

**预期收益：** bundle size 减少 15-25%

### 2. API 加密并行化 (LOW-MEDIUM)

**问题：** `src/lib/api.ts:230-237` 串行执行加密操作
```typescript
// ❌ 当前：串行等待
const publicKey = await fetchPublicKey();
const sm4KeyHex = generateSM4Key();
const ivHex = generateIV();
const encryptedDataHex = await encryptRequestBody(config.data, sm4KeyHex, ivHex);
const encryptedSM4Key = await encryptWithSM2(sm4KeyHex, publicKey);
```

**解决方案：** 使用 Promise.all 并行执行独立操作
```typescript
// ✅ 优化：并行执行
const [publicKey, sm4KeyHex, ivHex] = await Promise.all([
  fetchPublicKey(),
  Promise.resolve(generateSM4Key()),
  Promise.resolve(generateIV()),
]);

const [encryptedDataHex, encryptedSM4Key] = await Promise.all([
  encryptRequestBody(config.data, sm4KeyHex, ivHex),
  encryptWithSM2(sm4KeyHex, publicKey),
]);
```

**预期收益：** 加密请求延迟减少 30-40%

### 3. Excel 组件懒加载 (LOW)

**问题：** ExcelImport 组件在首屏加载，增加 50-100KB

**解决方案：** 使用 React.lazy() 懒加载
```typescript
// ✅ 创建懒加载包装器
const ExcelImport = lazy(() => import('./ExcelImport'));

export function ExcelImportLazy(props) {
  return (
    <Suspense fallback={<Spin />}>
      <ExcelImport {...props} />
    </Suspense>
  );
}
```

**预期收益：** 首屏减少 50-100KB

## 执行计划

1. **图标按需导入** - 修改 `iconUtils.tsx`，移除全量导入，只导入使用的图标
2. **API 加密并行化** - 修改 `api.ts`，使用 Promise.all 并行化独立操作
3. **Excel 懒加载** - 创建 `ExcelImportLazy` 包装器，更新高优先级页面

## 验证方法

每完成一个优化后，运行 `npm run build` 验证：
- 没有引入新的 TypeScript 错误
- 构建成功完成
