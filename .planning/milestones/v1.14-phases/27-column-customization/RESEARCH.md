# Phase 27: 全局列自定义显示功能 - 技术调研

## 研究目标

为全局列自定义显示功能选择最佳技术方案和实现模式。

## 技术选型研究

### 1. 前端拖拽库选择

**候选方案**:
1. **dnd-kit** (推荐)
   - 优点: 现代 API，性能优秀，TypeScript 支持好，无额外依赖
   - 缺点: 学习曲线稍陡
   - 示例: 使用 `SortableContext` + `useSortable`

2. **react-dnd**
   - 优点: 成熟稳定，社区活跃
   - 缺点: 依赖较多，API 较旧
   - 结论: 备选方案

**选择**: **dnd-kit**

### 2. localStorage 缓存策略

**缓存设计**:
```typescript
// 缓存结构
{
  "column_config:asset.list": {
    data: [...],
    timestamp: 1717880800000,
    expiresAt: 1717881100000  // 5分钟后
  }
}
```

**缓存策略**:
1. **读取流程**:
   - 优先读取 localStorage
   - 检查 expiresAt，未过期则使用
   - 过期或不存在则调用 API

2. **写入流程**:
   - 配置变更立即写入 localStorage
   - 同时防抖调用 API 保存（500ms）
   - API 成功后更新缓存时间戳

**选择**: **双重缓存** (localStorage 5分钟 + Redis 30分钟)

### 3. 默认配置结构

**资产列表 43 列分组**:

```typescript
const defaultAssetColumns = {
  // 核心标识 (3列)
  devicesn: { key: 'devicesn', label: '设备序列号', visible: true, order: 1, group: '核心标识' },
  sequenceNo: { key: 'sequenceNo', label: '序列号', visible: false, order: 2, group: '核心标识' },
  fixAssetNo: { key: 'fixAssetNo', label: '固定资产编号', visible: true, order: 3, group: '核心标识' },

  // 设备信息 (4列)
  deviceModelName: { key: 'deviceModelName', label: '设备型号', visible: true, order: 4, group: '设备信息' },
  deviceTypeName: { key: 'deviceTypeName', label: '设备类型', visible: true, order: 5, group: '设备信息' },
  deviceCategorySecondName: { key: 'deviceCategorySecondName', label: '设备中类', visible: false, order: 6, group: '设备信息' },
  deviceBasicTypeName: { key: 'deviceBasicTypeName', label: '是否固定资产', visible: true, order: 7, group: '设备信息' },

  // ... 其他分组
}
```

**配置加载**:
- 后端: 在 Service 中硬编码 `defaultConfigs` map
- 前端: 在 Hook 中定义相同结构，与服务端保持一致
- 首次加载: 无用户配置时返回默认配置

### 4. 性能优化方案

**目标**: 列切换响应 < 200ms

**优化措施**:
1. **前端优化**:
   - 使用 `useMemo` 缓存列分组结果
   - 使用 `useCallback` 包装事件处理
   - 防抖 API 保存（500ms）

2. **后端优化**:
   - Redis 缓存用户配置（30分钟 TTL）
   - 批量保存使用事务（先 DELETE 再 INSERT）
   - 索引优化: `(user_id, page_key)`

3. **网络优化**:
   - localStorage 优先读取
   - 配置变更批量提交
   - 失败重试机制

## 默认配置数据库

**各页面默认配置**:

| 页面 | Page Key | 默认列数量 | 核心列 |
|------|----------|-----------|--------|
| 资产管理 | asset.list | 43 | 设备序列号、型号、状态、领取人 |
| 用户管理 | user.list | 12 | 用户名、昵称、部门、状态 |
| 角色管理 | role.list | 8 | 角色名、权限、状态、排序 |
| 部门管理 | dept.list | 10 | 部门名、负责人、电话、状态 |

## 技术决策总结

| 决策点 | 选择方案 | 理由 |
|--------|---------|------|
| 拖拽库 | dnd-kit | 现代 API，性能优秀，TypeScript 支持 |
| 缓存策略 | localStorage + Redis 双层 | 前端优先 + 后端持久化 |
| 防抖时间 | 500ms | 平衡响应速度和请求频率 |
| 缓存有效期 | localStorage 5分钟，Redis 30分钟 | 前端短，后端长 |
| 配置隔离 | user_id + page_key | 用户级隔离，页面独立 |
| 默认配置 | 后端硬编码 | 便于版本管理和更新 |

## 参考实现

- **Ant Design Table.Column**: 列显示/隐藏控制
- **dnd-kit Sortable**: 列拖拽排序
- **Zustand**: 状态管理模式参考
- **localStorage**: 浏览器缓存 API
