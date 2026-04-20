# Phase 5: 前端交互完善 - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-04-20
**Phase:** 05-frontend-interactions
**Areas discussed:** Widget 选择器设计, 仪表盘设置面板, 模板预览功能, Widget 编辑/删除交互

---

## Widget 选择器设计

| Option | Description | Selected |
|--------|-------------|----------|
| 模态框 | 点击添加按钮后弹出模态框，包含完整的 Widget 配置流程 | ✓ |
| 侧边抽屉 | 从右侧滑出的抽屉，可同时看到仪表盘内容 | |
| 下拉面板 | 点击按钮后直接展开下拉面板，适合快速添加 | |

**User's choice:** 模态框（推荐）

| Option | Description | Selected |
|--------|-------------|----------|
| 分步向导 | 先选 Widget 类型 → 再配置数据源 → 最后配置显示样式 | |
| 单页表单 | 所有配置项在一个表单中，适合高级用户 | ✓ |
| 模板优先 | 先选择预设模板，然后微调配置 | |

**User's choice:** 单页表单

| Option | Description | Selected |
|--------|-------------|----------|
| 分类树 + 自动过滤 | 按模块/分类分组显示端点，用户选择后自动填充支持的 Widget 类型 | ✓ |
| 平铺列表 | 平铺所有端点，用户自行判断 | |
| 搜索输入 | 用户输入端点路径，系统验证 | |

**User's choice:** 分类树 + 自动过滤（推荐）

---

## 仪表盘设置面板

| Option | Description | Selected |
|--------|-------------|----------|
| 侧边抽屉 | 从右侧滑出，可同时看到仪表盘内容变化 | ✓ |
| 模态框 | 弹出模态框，需要确认后才能看到变化 | |
| 顶部展开面板 | 在仪表盘顶部展开，适合快速修改 | |

**User's choice:** 侧边抽屉（推荐）

| Option | Description | Selected |
|--------|-------------|----------|
| 下拉 + 条件字段 | 下拉选择 private/dept/global，部门选择器在 scope=dept 时显示 | ✓ |
| 单选按钮组 | 三个单选按钮，每个带说明文字 | |
| 单选 + 帮助提示 | 使用 Radio.Group + Tooltip 说明 | |

**User's choice:** 下拉 + 条件字段（推荐）

| Option | Description | Selected |
|--------|-------------|----------|
| 预设 + 自定义 | 预设选项（30s/1m/5m/15m/30m/1h）+ 自定义输入 | ✓ |
| 仅预设选项 | 只有预设选项，不允许自定义 | |
| 数字输入 + 单位 | 数字输入框 + 单位选择（秒/分/时） | |

**User's choice:** 预设 + 自定义（推荐）

---

## 模板预览功能

| Option | Description | Selected |
|--------|-------------|----------|
| 全屏覆盖 | 在当前页面覆盖显示预览，点击外部或按钮关闭 | ✓ |
| 悬浮卡片 | 在模板列表项右侧显示小预览窗口 | |
| 侧边抽屉 | 在右侧抽屉中显示预览 | |

**User's choice:** 全屏覆盖（推荐）

| Option | Description | Selected |
|--------|-------------|----------|
| 静态示例数据 | 使用模板配置的默认数据或静态数据，不调用真实 API | |
| 真实 API 数据 | 实际调用后端 API 获取真实数据 | ✓ |
| 仅布局预览 | 显示 Widget 布局但不显示数据 | |

**User's choice:** 真实 API 数据

| Option | Description | Selected |
|--------|-------------|----------|
| 直接创建并编辑 | 预览中点击"使用此模板"后直接创建仪表盘并跳转编辑 | ✓ |
| 回到表单填写 | 关闭预览后回到创建表单，用户填写名称后再创建 | |
| 预览中填写名称 | 在预览中直接填写名称并创建 | |

**User's choice:** 直接创建并编辑（推荐）

---

## Widget 编辑/删除交互

| Option | Description | Selected |
|--------|-------------|----------|
| 侧边抽屉 | 点击 Widget 后从右侧滑出配置面板，可同时看到 Widget 变化 | ✓ |
| 模态框 | 弹出模态框，需要确认后才能看到变化 | |
| 内联编辑 | 在 Widget 内部直接显示配置表单 | |

**User's choice:** 侧边抽屉（推荐）

| Option | Description | Selected |
|--------|-------------|----------|
| 动态表单 | 根据数据源类型（API/WebSocket/Static）显示不同的配置表单 | ✓ |
| 平铺表单 | 所有配置项平铺，用户根据类型填写 | |
| Tab 切换 | 使用 Tabs 切换不同数据源类型的配置 | |

**User's choice:** 动态表单（推荐）

| Option | Description | Selected |
|--------|-------------|----------|
| 确认对话框 | 弹出确认对话框，需要用户确认 | ✓ |
| 撤销提示 | 删除后显示撤销提示，几秒后自动消失 | |
| 直接删除 | 直接删除，无确认 | |

**User's choice:** 确认对话框（推荐）

---

## Claude's Discretion

- 表单验证规则：必填项、格式校验
- 错误提示位置：字段下方
- 加载状态：按钮 loading + 骨架屏
- 组件库：Ant Design 6（与项目一致）

## Deferred Ideas

None — discussion stayed within phase scope
