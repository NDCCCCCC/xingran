# Phase 27: 全局列自定义显示功能 - Context

**Gathered:** 2026-06-09
**Status:** Ready for planning

<domain>
## Phase Boundary

为系统中所有列表页面添加可自定义列显示的功能，解决资产列表页面字段过多（43列）的问题。用户可以选择显示哪些列、调整列顺序、设置列宽度，配置保存到数据库后永久生效。

**核心目标：**
1. **列显示/隐藏控制** — 用户可以选择显示哪些列
2. **列拖拽排序** — 用户可以调整列的显示顺序
3. **列宽度自定义** — 用户可以设置列宽度
4. **持久化存储** — 配置保存到数据库，永久生效
5. **重置功能** — 支持重置为默认配置
6. **性能优化** — localStorage 缓存 + 数据库同步，切换列响应 < 200ms

**不包含：**
- 不修改现有列表页面的核心功能
- 不影响现有的权限和过滤机制
- 不支持多配置预设（仅单一个人配置）

</domain>

<decisions>
## Implementation Decisions

### UI 交互与界面设计

#### D-01: 配置触发方式
**决策：** 使用设置按钮

在表格右上角添加列设置按钮，点击打开配置弹窗。这种方式符合 Ant Design 规范，用户易于发现和理解。

**实施方式：**
- 在 Table 组件的工具栏区域添加"列设置"按钮
- 按钮使用 SettingOutlined 或 ColumnHeightOutlined 图标
- 按钮位置：在表格右上角，与刷新、导出按钮并列

#### D-02: 配置界面呈现
**决策：** 使用模态弹窗

点击按钮弹出模态对话框，包含所有列的复选框和拖拽排序。弹窗方式清晰直观，易于实现。

**实施方式：**
- 使用 Ant Design Modal 组件
- 弹窗宽度：600-800px（根据列数调整）
- 弹窗内容：
  - 搜索框（顶部）
  - 列配置列表（中部，可滚动）
  - 操作按钮（底部：确定、取消、重置）

#### D-03: 列排序交互
**决策：** 支持拖拽排序

用户可以拖拽列调整顺序，提供完整的自定义能力。

**实施方式：**
- 使用 react-dnd 或 Ant Design 内置的拖拽支持
- 拖拽手柄：每行左侧的拖拽图标
- 拖拽反馈：拖拽时高亮目标位置，释放后自动重新排序

#### D-04: 列宽度自定义
**决策：** 支持自定义宽度

用户可以手动设置每列的宽度并保存，提供最完整的自定义能力。

**实施方式：**
- 在列配置列表中为每列提供宽度输入框
- 宽度单位：像素（px）
- 宽度范围：50px - 500px
- 提供重置宽度按钮（恢复默认宽度）

#### D-05: 列显示/隐藏选择
**决策：** 复选框列表

使用复选框控制每列的显示/隐藏状态，清晰直观。

**实施方式：**
- 每列一行，包含：
  - 复选框（控制显示/隐藏）
  - 拖拽手柄（控制排序）
  - 列名称（中文）
  - 宽度输入框（可选）
- 顶部提供"全选/取消全选"按钮

#### D-06: 列搜索功能
**决策：** 提供搜索框

对于 43 列的资产列表，搜索功能帮助用户快速定位列。

**实施方式：**
- 在弹窗顶部添加搜索输入框
- 实时过滤：输入时立即显示匹配的列
- 搜索范围：列名称（中英文）
- 搜索反馈：显示匹配列数（如"找到 5 列"）

#### D-07: 配置预设支持
**决策：** 单一配置

仅保存一个个人配置，不支持多预设，简化实现。

### 默认配置管理

#### D-08: 默认配置定义方式
**决策：** 代码定义

在代码中定义常量配置，版本可控，易于代码审查。

**实施方式：**
- 在每个列表页面组件中定义 `defaultColumnConfig` 常量
- 使用 TypeScript 类型确保配置结构正确
- 配置格式：
  ```typescript
  interface ColumnConfig {
    key: string;        // 列标识
    visible: boolean;   // 是否可见
    order: number;      // 显示顺序
    width?: number;     // 列宽度（可选）
  }

  const defaultColumnConfig: ColumnConfig[] = [
    { key: 'devicesn', visible: true, order: 1, width: 150 },
    { key: 'deviceModelName', visible: true, order: 2, width: 120 },
    // ...
  ];
  ```

#### D-09: 默认配置组织方式
**决策：** 组件内定义

每个列表页面组件内部定义默认列配置，配置与使用位置接近，易于维护。

**实施方式：**
- 在页面组件文件中定义（如 `AssetList.tsx`）
- 与组件代码放在一起，便于同步更新
- 导出配置供测试使用

#### D-10: 默认配置应用时机
**决策：** 首次自动应用

首次访问页面时自动应用默认配置，用户友好的默认行为。

**实施方式：**
- 检查用户是否有自定义配置
- 如果没有配置，自动应用默认配置
- 如果有配置，使用用户配置

#### D-11: 默认列宽指定
**决策：** 指定默认宽度

在默认配置中为每列指定推荐宽度，提供更精确的初始显示。

**实施方式：**
- 在 `defaultColumnConfig` 中为每列指定 `width` 字段
- 宽度基于列内容类型：
  - 短文本（如状态）：80-100px
  - 中等文本（如名称）：120-150px
  - 长文本（如描述）：200-300px
  - 日期时间：120-150px

#### D-12: 角色配置区分
**决策：** 全局统一

所有用户使用相同的默认配置，不根据角色区分，简化实现。

### 性能优化与缓存

#### D-13: 缓存机制
**决策：** 双层缓存

使用 localStorage（前端）+ Redis（后端）双层缓存，性能最佳。

**实施方式：**
- **前端缓存（localStorage）**：
  - 缓存键：`column_config:{page_key}:{user_id}`
  - 缓存值：JSON 字符串（列配置数组）
  - 过期时间：5 分钟
  - 用途：页面加载时优先使用，减少服务器请求

- **后端缓存（Redis）**：
  - 缓存键：`column_config:{user_id}:{page_key}`
  - 缓存值：JSON 字符串（列配置数组）
  - 过期时间：30 分钟
  - 用途：减少数据库查询

#### D-14: 缓存过期时间
**决策：** 标准时长

- localStorage：5 分钟
- Redis：30 分钟

这个设置平衡了性能和数据新鲜度。

#### D-15: 缓存失效策略
**决策：** 立即清除

用户修改配置后立即清除缓存，确保数据一致性。

**实施方式：**
- 前端：保存配置后删除 localStorage 缓存
- 后端：保存配置后删除 Redis 缓存
- 下次加载时重新从数据库获取

#### D-16: 配置同步策略
**决策：** 立即同步

用户修改配置后立即同步到服务器并清除本地缓存，确保多端一致性。

**实施方式：**
- 用户点击"确定"按钮后立即调用 API 保存
- 保存成功后清除前端缓存
- 保存失败时显示错误提示，保持本地缓存

#### D-17: 页面加载策略
**决策：** 渐进加载

页面加载时先显示默认列，然后异步加载用户配置，快速首屏。

**实施方式：**
1. 首次渲染：使用默认列配置渲染表格
2. 并行请求：异步获取用户列配置
3. 配置加载后：对比默认配置，如有变化则更新表格
4. 性能目标：首屏渲染 < 100ms，配置加载 < 200ms

### 重置功能与持久化

#### D-18: 重置功能范围
**决策：** 单页重置

仅重置当前页面的列配置，影响范围明确。

**实施方式：**
- 在配置弹窗中提供"重置"按钮
- 重置行为：删除当前页面的用户配置
- 重置后：恢复为默认配置
- 确认机制：重置前弹出确认对话框

#### D-19: 重置确认机制
**决策：** 需要确认

需要用户确认后才执行重置，防止误操作。

**实施方式：**
- 使用 Ant Design Modal.confirm
- 确认文案："确定要重置列配置吗？此操作将清除您的个人设置。"
- 确认后：调用重置 API，清除本地缓存

#### D-20: 配置持久化时机
**决策：** 立即保存

每次列配置更改立即保存到数据库，确保不丢失。

**实施方式：**
- 用户点击"确定"按钮后立即调用 API
- API 请求：`POST /api/v1/system/column-config`
- 请求体：包含 `page_key` 和 `column_config` 数组
- 保存成功后：显示成功提示，清除缓存

#### D-21: 保存错误处理
**决策：** 提示并保持本地

保存失败时显示错误提示，保持本地缓存，用户友好的错误处理。

**实施方式：**
- API 调用失败时：使用 Ant Design message.error 显示错误
- 本地缓存：保持不变，用户可以继续使用
- 错误提示文案："保存列配置失败，请稍后重试"
- 错误日志：记录到前端日志系统

### Claude's Discretion

以下方面可以由实现者决定：

1. **拖拽库选择** — react-dnd、@dnd-kit、或 Ant Design 内置拖拽
2. **搜索框防抖延迟** — 300ms、500ms 或其他值
3. **弹窗动画效果** — 淡入淡出、滑入滑出或其他
4. **错误提示样式** — 使用 message 或 notification
5. **重置按钮位置** — 底部工具栏、弹窗标题栏或其他
6. **缓存键命名规范** — 遵循现有项目规范即可

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### 项目架构文档
- `docs/项目概述和架构设计.md` — 整体架构
- `docs/开发规范.md` — 开发规范（前端组件、命名规范）
- `docs/API响应规范.md` — API 响应格式

### 数据库设计
- `.planning/phases/27-column-customization/PHASE.md` — 数据库表结构（sys_user_column_config）
- `internal/core/db/migrations/` — 迁移脚本目录

### 前端参考
- `xingran-react-frontend/src/components/` — 现有组件库
- `xingran-react-frontend/src/store/authStore.ts` — 权限存储模式参考
- `xingran-react-frontend/src/hooks/` — 自定义 Hooks 模式
- Ant Design Table 文档 — https://ant.design/components/table

### 后端参考
- `internal/api/router.go` — 路由配置参考
- `internal/models/base.go` — BaseModel 模式
- `pkg/middleware/permission.go` — 权限中间件参考

### Phase 25 上下文（UI 权限控制）
- `.planning/phases/25-vm-data-scope-permissions/25-CONTEXT.md` — 按钮级权限控制模式

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- **Ant Design Table** — 现有表格组件，支持列配置、自定义渲染
- **authStore** — 权限存储模式，可作为配置存储参考
- **自定义 Hooks 模式** — 项目已有的 Hook 组织方式
- **localStorage 缓存模式** — 项目已有的前端缓存实现
- **Redis 缓存服务** — `pkg/cache/redis.go`，已实现的缓存层

### Established Patterns
- **Handler-Service 模式** — 后端 API 层的标准模式
- **组件内配置** — 前端组件常量定义方式
- **响应式设计模式** — 前端响应式布局模式
- **API 响应格式** — 统一的响应包装格式

### Integration Points
- **前端路由** — 需要为列配置 API 添加路由
- **主路由** (`internal/api/router.go`) — 系统模块路由组
- **Table 组件** — 需要扩展支持动态列配置
- **localStorage** — 前端缓存层

### Known Constraints
- 资产列表有 43 列，需要特别优化性能
- 所有列表页面应保持一致的配置体验
- 配置必须基于 user_id 隔离
- 必须兼容现有的 Table 组件功能

</code_context>

<specifics>
## Specific Ideas

### 配置界面布局
```typescript
<Modal title="列配置" width={700} open={visible} onOk={handleSave} onCancel={onClose}>
  <Input.Search placeholder="搜索列..." onChange={handleSearch} />
  <div className="column-list">
    {columns.map(col => (
      <div key={col.key} className="column-item">
        <Checkbox checked={col.visible} onChange={(e) => toggleColumn(col.key, e.target.checked)}>
          {col.title}
        </Checkbox>
        <InputNumber value={col.width} onChange={(val) => setWidth(col.key, val)} />
      </div>
    ))}
  </div>
  <Button onClick={handleReset}>重置</Button>
</Modal>
```

### 缓存键规范
```typescript
// 前端 localStorage 缓存键
const CACHE_KEY = `column_config:${pageKey}:${userId}`;

// 后端 Redis 缓存键
const REDIS_KEY = `column_config:${userId}:${pageKey}`;
```

### API 接口设计
```typescript
// 获取用户列配置
GET /api/v1/system/column-config/:page_key
Response: {
  code: 0,
  data: {
    page_key: string,
    column_config: ColumnConfig[]
  }
}

// 保存列配置
POST /api/v1/system/column-config
Request: {
  page_key: string,
  column_config: ColumnConfig[]
}
Response: {
  code: 0,
  message: "success"
}

// 重置列配置
DELETE /api/v1/system/column-config/:page_key
Response: {
  code: 0,
  message: "success"
}
```

</specifics>

<deferred>
## Deferred Ideas

以下想法不在本期范围：

- **多配置预设** — 支持多个命名配置方案（如"详细视图"、"简化视图"）
- **跨页面同步** — 在多个页面间共享列配置
- **配置导入/导出** — 允许用户导入和导出列配置
- **列分组** — 支持将列分组管理（如"基本信息"、"技术参数"）
- **角色级默认配置** — 不同角色使用不同的默认列配置
- **配置历史版本** — 记录配置变更历史，支持回滚

</deferred>

---

*Phase: 27-column-customization*
*Context gathered: 2026-06-09*
