# Phase 14-04 Summary: Excel 导出功能

**执行时间:** 2026-06-14
**状态:** ✅ 完成
**计划文件:** `.planning/phases/14-frontend-ux/14-04-PLAN.md`

---

## 执行概述

成功实现 MAC 历史查询页面的 Excel 导出功能，包括"导出当前查询"和"导出全量"两个互斥按钮，符合 UI-02 需求。

## 核心交付

### 1. API 扩展 (`networkApi.ts`)
**文件:** `xingran-react-frontend/src/lib/api/networkApi.ts`

- ✅ 已实现 `exportMACHistory` 函数（lines 130-141）
- ✅ 使用 `api.get` + `responseType: 'blob'` 模式
- ✅ 通过 `format=xlsx` query 参数触发后端 Excel 导出
- ✅ 支持 `exportScope: 'current' | 'all'` 两种导出范围
- ✅ 超时设置 120 秒（适合大数据量导出）

```typescript
export const exportMACHistory = async (
  params: MACHistoryQueryParams,
  exportScope: 'current' | 'all' = 'current'
): Promise<Blob> => {
  const api = await import('../api');
  const response = await api.default.get('/network/history/list', {
    params: { ...params, format: 'xlsx', exportScope },
    responseType: 'blob',
    timeout: 120000,
  });
  return response as Blob;
};
```

### 2. 导出按钮与权限控制 (`history.tsx`)
**文件:** `xingran-react-frontend/src/pages/network/mac/history.tsx`

#### 2.1 权限检查
- ✅ 从 `useAuthStore` 获取用户权限
- ✅ 实现 `hasPermission` 工具函数（line 323-326）
- ✅ 严格检查 `network:mac:export` 权限点

#### 2.2 导出处理函数
- ✅ 实现 `handleExport` 函数（lines 329-407）
- ✅ 支持 `exportScope: 'current' | 'all'` 两种模式
- ✅ 导出当前查询时，传入当前表单过滤条件
- ✅ 导出全量时，清空所有过滤条件
- ✅ 异常处理：当 blob.size < 1024 时，解析 JSON 错误信息并展示
- ✅ 生成带时间戳的文件名（`mac_history_current_YYYYMMDD_HHmmss.xlsx` / `mac_history_all_YYYYMMDD_HHmmss.xlsx`）
- ✅ 使用 `URL.createObjectURL` + `a.download` 模式触发浏览器下载
- ✅ 下载后清理 URL 对象（`revokeObjectURL`）

#### 2.3 工具栏按钮组
- ✅ 在"查询"和"重置"按钮后新增两个导出按钮（lines 695-718）
- ✅ "导出当前查询"：蓝色 primary 按钮
- ✅ "导出全量"：灰色 default 按钮
- ✅ 两个按钮均使用 `<DownloadOutlined />` 图标
- ✅ 严格权限控制：仅当 `hasPermission('network:mac:export')` 为 true 时渲染

```typescript
{hasPermission('network:mac:export') && (
  <>
    <Button
      type="primary"
      icon={<DownloadOutlined />}
      onClick={() => handleExport('current')}
    >
      导出当前查询
    </Button>
    <Button
      type="default"
      icon={<DownloadOutlined />}
      onClick={() => handleExport('all')}
    >
      导出全量
    </Button>
  </>
)}
```

## 验收标准完成情况

| 验收标准 | 状态 | 说明 |
|---------|------|------|
| networkApi.ts 含 exportMACHistory 函数 export | ✅ | line 130 |
| exportMACHistory 内部使用 api.get + responseType: 'blob' | ✅ | line 137 |
| exportMACHistory 含 format: 'xlsx' 参数 | ✅ | line 136 |
| history.tsx 含 '导出当前查询' 与 '导出全量' 两个按钮文字 | ✅ | lines 702, 709 |
| history.tsx 含 createObjectURL + a.click() | ✅ | lines 399-404 |
| history.tsx 含异常处理分支 blob.size < 1024 | ✅ | line 377 |
| history.tsx 含 network:mac:export 权限可见性判断 | ✅ | line 695 |
| npm run type-check 0 错误 | ✅ | TypeScript 编译通过 |

## 依赖关系验证

✅ **14-03** (菜单/权限/路由注册)：已完成，提供 `network:mac:export` 权限点
✅ **14-01** (列表页)：已完成，提供工具栏注入位置
✅ **14-05a** (移动端响应式)：已完成，避免 shared barrel 冲突

## 关键决策与实现细节

### D-01 锁定：复用现有端点
- 不新增 `/network/history/export` API
- 通过 `format=xlsx` query 参数分支现有 `/network/history/list` 端点
- 后端需要在现有 handler 增加 format 分支（属于"现有 endpoint 增强"）

### D-13 锁定：导出范围
- **导出当前查询**：传入当前表单过滤条件（时间/MAC/设备/端口/VLAN/事件类型/状态）
- **导出全量**：清空所有过滤条件，仅传 `format=xlsx&exportScope=all`

### D-15 锁定：blob 异常检测
- 当 blob.size < 1024 时，尝试解析为 JSON 错误体
- 使用 `FileReader.readAsText(blob, 'utf-8')` 读取
- 解析失败时显示通用错误提示

### 权限控制严格性
- 两个按钮必须以条件渲染方式呈现（`{hasPermission('network:mac:export') && (...)}`）
- 权限缺失时按钮完全不渲染（不使用 `disabled` 属性）
- 避免无条件 `return null`，符合 14-04 计划验收标准

## 文件修改清单

| 文件 | 修改类型 | 修改内容 |
|------|---------|---------|
| `xingran-react-frontend/src/lib/api/networkApi.ts` | 新增函数 | exportMACHistory (lines 130-141) |
| `xingran-react-frontend/src/pages/network/mac/history.tsx` | 新增导入 | useAuthStore, message (line 22) |
| `xingran-react-frontend/src/pages/network/mac/history.tsx` | 新增变量 | user (line 105) |
| `xingran-react-frontend/src/pages/network/mac/history.tsx` | 新增函数 | hasPermission (lines 323-326) |
| `xingran-react-frontend/src/pages/network/mac/history.tsx` | 新增函数 | handleExport (lines 329-407) |
| `xingran-react-frontend/src/pages/network/mac/history.tsx` | 修改 JSX | 工具栏新增导出按钮组 (lines 695-718) |

## 后续计划依赖

### 14-05b (Wave 3) 依赖本计划
- 14-05b 改造 history.tsx 时**必须保留**本计划注入的：
  1. "导出当前查询" / "导出全量" 两个按钮的 React 节点
  2. prop 名称 `exportScope='current'|'all'`
  3. 工具栏位置（查询/重置按钮之后）
  4. `handleExport` 函数与权限 `network:mac:export` 判断

- 14-05b 验收标准明确要求："导出按钮 prop 名称与位置在 14-05b 改造后完整保留"

## 成功标准验证

- [x] UI-02 完整满足（Excel 格式 + 当前查询范围 + 全量范围）
- [x] 不新增后端 API 端点（D-01 锁定）
- [x] 走 api.get(url, { params, responseType: 'blob' }) 拦截器（Phase 33 M2 锁定）
- [x] blob 异常检测 < 1KB（D-15 锁定）
- [x] 按钮权限 network:mac:export 严格隔离
- [x] 文件名含时间戳
- [x] depends_on 包含 14-05a，避免与 14-05a 改造冲突

## 风险与注意事项

1. **后端依赖**：本计划假设后端已实现 `/network/history/list?format=xlsx&exportScope=all` 分支，如后端未实现，需要配合后端开发
2. **大数据量超时**：超时设置为 120 秒，如实际导出时间更长，可能需要调整
3. **权限同步**：权限检查依赖前端 authStore 中的 permissions 数组，需确保后端登录接口返回最新的权限列表

## 总结

Phase 14-04 圆满完成，为 MAC 历史查询页面提供了完整的 Excel 导出能力。前端实现符合所有锁定决策与验收标准，为 14-05b 的三态打磨奠定了坚实基础。导出按钮的 prop 名称与位置已明确标记，确保后续改造时不会被意外覆盖。

---

**执行者:** Phase 14-04 executor (autonomous plan)
**审查者:** 14-04 plan-checker (iteration 1 fix validated)
**下一步:** Wave 3 → 14-05b (三态打磨 + 移动端 + 网络设备联动入口)
