# 工位设备展开功能优化计划

## 任务描述

优化工位管理页面的设备展开功能，包括分页、行点击展开、展开行数限制、数据源逻辑确认和自动同步。

## 关键文件

### 前端文件
- `xingran-react-frontend/src/pages/operations/workstations/index.tsx` - 工位管理主页面
- `xingran-react-frontend/src/components/operations/WorkstationDeviceTable/index.tsx` - 工位设备子表格组件
- `xingran-react-frontend/src/lib/opsApi.ts` - API 方法
- `xingran-react-frontend/src/hooks/usePagination.ts` - 分页 Hook

## 实施步骤

### 1. 子表格分页功能
**目标**: 当工位下设备较多时，显示分页控件

**修改点**:
- `WorkstationDeviceTable/index.tsx`:
  - 添加 `usePagination` hook
  - 将 `pagination={false}` 改为 `pagination={paginationProps}`
  - 添加分页状态管理（current, pageSize, total）
  - 添加分页变化处理函数

### 2. 行点击展开机制
**目标**: 移除"查看设备/收起设备"按钮，改为点击行展开

**修改点**:
- `workstations/index.tsx`:
  - 移除 `expandIcon` 自定义配置
  - 使用默认的展开图标（或自定义更小的图标）
  - 添加 `onRow` 点击处理

### 3. 展开行数限制
**目标**: 限制同时展开的最大行数，节约资源

**修改点**:
- `workstations/index.tsx`:
  - 添加 `expandedRowKeys` 状态管理
  - 添加展开行数限制逻辑
  - 达到限制时显示提示
- 配置参数:
  - 建议参数键名: `sys.workstation.max.expanded.rows`
  - 建议默认值: 5
  - 可通过参数管理页面配置

### 4. 设备数据源逻辑确认
**目标**: 确认三种来源的数据处理方式

**当前实现**:
- **手动添加 (manual)**: 数据保存到数据库 `workstation_device` 表
- **资产同步 (asset)**: 实时查询资产表，不保存到工位设备表
- **AD同步 (ad)**: 实时查询AD域控，不保存到工位设备表

**验证点**:
- 查看 API 端点 `/ops/workstation-device/{workstationId}` 返回的数据源
- 确认同步接口只返回临时数据，不持久化

### 5. 展开时自动同步
**目标**: 展开行时自动触发设备信息同步

**修改点**:
- `WorkstationDeviceTable/index.tsx`:
  - 添加 `autoSync` 属性
  - 在组件首次加载时触发自动同步（AD + 资产）
  - 添加同步状态指示器

## 设计考虑

### 子表格样式
- 使用 `size="small"` 使子表格更紧凑
- 背景色使用浅灰色区分主表格
- 内边距适当缩小

### 分页配置
- 子表格分页大小独立于主表格
- 建议默认值为 10 条/页
- 复用 `usePagination` hook 的逻辑

### 用户体验
- 行点击展开时提供视觉反馈
- 达到展开限制时友好提示
- 自动同步时显示加载状态

## 执行顺序

1. ✅ 代码探索（已完成）
2. ⏳ 子表格分页功能
3. ⏳ 行点击展开机制
4. ⏳ 展开行数限制
5. ⏳ 设备数据源逻辑确认
6. ⏳ 展开时自动同步功能
7. ⏳ 测试验证
