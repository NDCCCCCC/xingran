# Phase 5: 前端交互完善 - Research

**Researched:** 2026-04-20
**Domain:** React 前端交互组件、Ant Design 6、Zustand 状态管理
**Confidence:** HIGH

## Summary

本阶段需要实现仪表盘编辑所需的四个核心交互组件：Widget 选择器、仪表盘设置面板、模板预览功能、Widget 编辑/删除交互。经过代码审计，发现项目已有部分组件实现（WidgetSelector、EndpointSelector、DashboardScopeSelector、TemplateSelector），需要增强和补充完整功能。

**Primary recommendation:** 复用现有组件架构，补充缺失的交互功能（DashboardSettings 抽屉、TemplatePreview 全屏预览、WidgetEditor 抽屉），增强 WidgetSelector 支持完整配置流程。

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

#### Widget 选择器设计
- **D-01:** 选择器位置 — 模态框
- **D-02:** 配置流程 — 单页表单
- **D-03:** API 端点选择 — 分类树 + 自动过滤

#### 仪表盘设置面板
- **D-04:** 面板位置 — 侧边抽屉
- **D-05:** 权限范围选择 — 下拉 + 条件字段
- **D-06:** 刷新间隔配置 — 预设 + 自定义

#### 模板预览功能
- **D-07:** 预览方式 — 全屏覆盖
- **D-08:** 预览数据 — 真实 API 数据
- **D-09:** 应用流程 — 直接创建并编辑

#### Widget 编辑/删除交互
- **D-10:** 编辑方式 — 侧边抽屉
- **D-11:** 数据源配置 — 动态表单
- **D-12:** 删除确认 — 确认对话框

### Claude's Discretion
- 表单验证规则：必填项、格式校验
- 错误提示位置：字段下方
- 加载状态：按钮 loading + 骨架屏
- 组件库：Ant Design 6（与项目一致）

### Deferred Ideas (OUT OF SCOPE)
None — discussion stayed within phase scope
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| FE-02a | 实现 Widget 选择器组件 | WidgetSelector 已存在，需增强数据源配置和显示配置 |
| FE-02b | 实现仪表盘设置面板 | 需新建 DashboardSettings 抽屉组件 |
| FE-02c | 实现模板预览功能 | 需新建 TemplatePreview 全屏覆盖组件 |
| FE-02d | 完善 Widget 编辑/删除交互 | 需新建 WidgetEditor 抽屉，增强删除确认 |
</phase_requirements>

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Widget 选择交互 | Browser / Client | — | 纯前端 UI 交互，不涉及服务端 |
| 仪表盘设置编辑 | Browser / Client | API / Backend | 前端表单 + 后端保存 API |
| 模板预览渲染 | Browser / Client | API / Backend | 前端渲染 + 后端数据获取 |
| Widget 配置编辑 | Browser / Client | API / Backend | 前端表单 + 后端更新 API |
| 删除确认交互 | Browser / Client | — | 纯前端确认对话框 |

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| React | 19.2 | UI 框架 | 项目标准，已验证 |
| TypeScript | 5.9 | 类型系统 | 项目标准，已验证 |
| Ant Design | 6.1.1 | UI 组件库 | 项目标准，Modal/Drawer/Form/Select 组件完备 |
| Zustand | 5.0.9 | 状态管理 | 项目标准，dashboardStore 已实现 |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| react-router-dom | 7.10.1 | 路由导航 | 模板预览后跳转编辑页 |
| dayjs | 1.11.19 | 日期处理 | 刷新间隔显示 |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Ant Design Modal | 自定义弹窗 | Ant Design 提供完整的可访问性和动画支持 |
| Ant Design Drawer | 自定义抽屉 | Ant Design 提供响应式宽度和手势关闭 |

**Installation:** 无需安装新依赖，所有依赖已存在于项目中。

## Architecture Patterns

### System Architecture Diagram

```
用户操作
    │
    ├─► 添加 Widget ──► WidgetSelector (Modal)
    │                       │
    │                       ├─► 选择 Widget 类型
    │                       ├─► EndpointSelector (分类树)
    │                       ├─► DataSourceForm (动态表单)
    │                       └─► addWidget() → dashboardStore
    │
    ├─► 编辑仪表盘设置 ──► DashboardSettings (Drawer)
    │                           │
    │                           ├─► 名称/描述编辑
    │                           ├─► DashboardScopeSelector
    │                           ├─► 刷新间隔配置
    │                           └─► updateDashboard() → API
    │
    ├─► 预览模板 ──► TemplatePreview (全屏覆盖)
    │                     │
    │                     ├─► getTemplates() → API
    │                     ├─► 渲染 Widget 布局
    │                     ├─► getWidgetData() → API (真实数据)
    │                     └─► createFromTemplate() → API
    │
    └─► 编辑/删除 Widget ──► WidgetEditor (Drawer)
                                │
                                ├─► 数据源配置 (动态表单)
                                ├─► 显示配置
                                ├─► updateWidget() → dashboardStore
                                └─► Modal.confirm() (删除确认)
```

### Recommended Project Structure
```
src/components/dashboard/
├── settings/
│   ├── WidgetSelector.tsx        # ✅ 已存在，需增强
│   ├── EndpointSelector.tsx      # ✅ 已存在
│   ├── DashboardScopeSelector.tsx # ✅ 已存在
│   ├── ParamsEditor.tsx          # ✅ 已存在
│   ├── DashboardSettings.tsx     # ❌ 待实现 (FE-02b)
│   └── RefreshIntervalSelector.tsx # ❌ 待实现 (FE-02b)
├── layout/
│   ├── TemplateSelector.tsx      # ✅ 已存在
│   └── TemplatePreview.tsx       # ❌ 待实现 (FE-02c)
├── widgets/
│   ├── WidgetEditor.tsx          # ❌ 待实现 (FE-02d)
│   └── configs/
│       └── widgetRegistry.ts     # ✅ 已存在
```

### Pattern 1: Modal 表单交互
**What:** 使用 Ant Design Modal + Form 实现配置弹窗
**When to use:** Widget 选择器（D-01 决策）
**Example:**
```typescript
// Source: 现有 WidgetSelector.tsx
<Modal
  title="添加 Widget"
  open={visible}
  onOk={handleConfirm}
  onCancel={handleCancel}
  width={800}
  destroyOnHidden={true}
>
  <Form form={form} layout="vertical">
    <Form.Item name="title" rules={[{ required: true }]}>
      <Input placeholder="请输入Widget标题" />
    </Form.Item>
  </Form>
</Modal>
```

### Pattern 2: Drawer 侧边抽屉
**What:** 使用 Ant Design Drawer 实现侧边配置面板
**When to use:** 仪表盘设置（D-04）、Widget 编辑（D-10）
**Example:**
```typescript
// Source: Ant Design 6 官方文档
<Drawer
  title="仪表盘设置"
  placement="right"
  open={visible}
  onClose={onClose}
  width={480}
>
  <Form layout="vertical">
    {/* 配置表单 */}
  </Form>
</Drawer>
```

### Pattern 3: 动态表单渲染
**What:** 根据数据源类型条件渲染不同的配置表单
**When to use:** Widget 数据源配置（D-11）
**Example:**
```typescript
// Source: 现有模式推断
const renderDataSourceForm = (type: 'api' | 'websocket' | 'static') => {
  switch (type) {
    case 'api':
      return <ApiDataSourceForm />;
    case 'websocket':
      return <WebSocketDataSourceForm />;
    case 'static':
      return <StaticDataSourceForm />;
  }
};
```

### Anti-Patterns to Avoid
- **直接使用 axios 调用 API:** 必须使用 `@/lib/api` 封装的 post/get 方法 [VERIFIED: AGENTS.md]
- **手动管理 loading 状态:** 使用 Ant Design Form 的内置验证和 loading 状态
- **内联样式定义:** 使用 Tailwind CSS 或 Ant Design 的 style 属性

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| 表单验证 | 自定义验证逻辑 | Ant Design Form rules | 内置验证、错误提示、异步验证 |
| 模态框动画 | 自定义 CSS 动画 | Ant Design Modal/Drawer | 可访问性、键盘导航、响应式 |
| 分类树选择 | 自定义树组件 | Ant Design Select + OptGroup | 搜索、虚拟滚动、异步加载 |
| 删除确认 | 自定义确认弹窗 | Modal.confirm() | 一致的用户体验 |

**Key insight:** Ant Design 6 提供了完整的交互组件，无需自定义实现。

## Common Pitfalls

### Pitfall 1: Modal/Drawer 状态未重置
**What goes wrong:** 关闭后再次打开，表单保留上次的数据
**Why it happens:** 未在关闭时调用 form.resetFields()
**How to avoid:** 使用 destroyOnHidden (Modal) 或在 onClose 中重置状态
**Warning signs:** 编辑不同 Widget 时显示上一个 Widget 的数据

### Pitfall 2: 表单验证时机错误
**What goes wrong:** 用户未填写完就触发验证，或提交时才显示错误
**Why it happens:** validateTrigger 配置不当
**How to avoid:** 使用 `validateTrigger: 'onBlur'` 或默认的 `onChange`
**Warning signs:** 用户抱怨"为什么还没填完就报错"

### Pitfall 3: 异步操作未处理 loading 状态
**What goes wrong:** 点击提交后无反馈，用户重复点击
**Why it happens:** 未设置按钮 loading 状态
**How to avoid:** 使用 useState 管理 loading，禁用提交按钮
**Warning signs:** 重复提交、用户困惑

### Pitfall 4: Zustand store 状态未同步
**What goes wrong:** 修改后仪表盘未更新，或切换仪表盘后显示旧数据
**Why it happens:** 未调用 store 的 update 方法，或未清除缓存
**How to avoid:** 始终通过 store actions 更新状态，使用 hasUnsavedChanges 标记
**Warning signs:** 刷新页面后数据丢失

## Code Examples

### Widget 选择器增强 (FE-02a)
```typescript
// Source: 现有 WidgetSelector.tsx + 需求增强
import { Modal, Form, Input, Select, Divider } from 'antd';
import { widgetRegistry, getWidgetTypes } from '../widgets/configs/widgetRegistry';
import EndpointSelector from './EndpointSelector';

interface WidgetSelectorProps {
  visible: boolean;
  onClose: () => void;
  onSelect: (widget: WidgetConfig) => void;
  editingWidgetId?: string | null;
  editingWidget?: WidgetConfig | null;
}

export const WidgetSelector: React.FC<WidgetSelectorProps> = ({
  visible,
  onClose,
  onSelect,
  editingWidgetId,
  editingWidget,
}) => {
  const [selectedType, setSelectedType] = useState<WidgetType | null>(null);
  const [form] = Form.useForm();

  const handleConfirm = async () => {
    const values = await form.validateFields();
    const widget: WidgetConfig = {
      id: editingWidgetId || `widget-${Date.now()}`,
      type: values.type,
      title: values.title,
      position: editingWidget?.position || { x: 0, y: 0, ...widgetDefaultSizes[values.type] },
      dataSource: { api: { type: 'api', endpoint: values.endpoint, method: values.method } },
      display: createDisplayConfig(values.type),
      enabled: true,
    };
    onSelect(widget);
    form.resetFields();
    onClose();
  };

  return (
    <Modal title="添加 Widget" open={visible} onOk={handleConfirm} onCancel={onClose} width={800}>
      {/* Widget 类型选择 + 配置表单 */}
    </Modal>
  );
};
```

### 仪表盘设置抽屉 (FE-02b)
```typescript
// Source: 需新建 DashboardSettings.tsx
import { Drawer, Form, Input, InputNumber, Select } from 'antd';
import { useDashboardStore } from '@/store/dashboardStore';
import DashboardScopeSelector from './DashboardScopeSelector';

interface DashboardSettingsProps {
  visible: boolean;
  onClose: () => void;
}

export const DashboardSettings: React.FC<DashboardSettingsProps> = ({ visible, onClose }) => {
  const { currentDashboard, updateDashboard } = useDashboardStore();
  const [form] = Form.useForm();

  const handleSave = async () => {
    const values = await form.validateFields();
    await updateDashboard(currentDashboard!.id, values);
    onClose();
  };

  return (
    <Drawer title="仪表盘设置" placement="right" open={visible} onClose={onClose} width={480}>
      <Form form={form} layout="vertical" initialValues={currentDashboard}>
        <Form.Item name="name" label="名称" rules={[{ required: true }]}>
          <Input />
        </Form.Item>
        <Form.Item name="description" label="描述">
          <Input.TextArea />
        </Form.Item>
        <Form.Item name="scope" label="可见范围">
          <DashboardScopeSelector />
        </Form.Item>
        <Form.Item name="refreshInterval" label="刷新间隔">
          <Select options={[
            { label: '30秒', value: 30 },
            { label: '1分钟', value: 60 },
            { label: '5分钟', value: 300 },
          ]} />
        </Form.Item>
      </Form>
    </Drawer>
  );
};
```

### 模板预览全屏覆盖 (FE-02c)
```typescript
// Source: 需新建 TemplatePreview.tsx
import { Modal, Button, Spin } from 'antd';
import { DashboardGrid } from '../layout/DashboardGrid';
import { dashboardService } from '@/services/dashboardService';

interface TemplatePreviewProps {
  visible: boolean;
  template: Dashboard | null;
  onClose: () => void;
  onApply: (templateId: string, name: string) => void;
}

export const TemplatePreview: React.FC<TemplatePreviewProps> = ({
  visible,
  template,
  onClose,
  onApply,
}) => {
  const [loading, setLoading] = useState(false);
  const [previewData, setPreviewData] = useState<Map<string, unknown>>(new Map());

  useEffect(() => {
    if (visible && template) {
      loadPreviewData();
    }
  }, [visible, template]);

  const loadPreviewData = async () => {
    setLoading(true);
    try {
      const widgetIds = template!.layout.widgets.map(w => w.id);
      const data = await dashboardService.getBatchWidgetData(widgetIds);
      setPreviewData(data);
    } finally {
      setLoading(false);
    }
  };

  return (
    <Modal
      open={visible}
      onCancel={onClose}
      footer={null}
      width="100vw"
      style={{ top: 0, paddingBottom: 0 }}
      bodyStyle={{ height: 'calc(100vh - 110px)', overflow: 'auto' }}
    >
      <Spin spinning={loading}>
        <DashboardGrid dashboard={template} widgetData={previewData} />
        <div style={{ position: 'fixed', bottom: 24, right: 24 }}>
          <Button onClick={onClose}>取消</Button>
          <Button type="primary" onClick={() => onApply(template!.id, `${template!.name} - 副本`)}>
            使用此模板
          </Button>
        </div>
      </Spin>
    </Modal>
  );
};
```

### Widget 编辑抽屉 (FE-02d)
```typescript
// Source: 需新建 WidgetEditor.tsx
import { Drawer, Form, Input, Select, Button, Modal } from 'antd';
import { useDashboardStore } from '@/store/dashboardStore';

interface WidgetEditorProps {
  visible: boolean;
  widget: WidgetConfig | null;
  onClose: () => void;
}

export const WidgetEditor: React.FC<WidgetEditorProps> = ({ visible, widget, onClose }) => {
  const { updateWidget, removeWidget } = useDashboardStore();
  const [form] = Form.useForm();

  const handleSave = async () => {
    const values = await form.validateFields();
    updateWidget(widget!.id, values);
    onClose();
  };

  const handleDelete = () => {
    Modal.confirm({
      title: '确认删除',
      content: `确定要删除 Widget "${widget!.title}" 吗？`,
      onOk: () => {
        removeWidget(widget!.id);
        onClose();
      },
    });
  };

  return (
    <Drawer
      title="编辑 Widget"
      placement="right"
      open={visible}
      onClose={onClose}
      width={480}
      footer={
        <div style={{ display: 'flex', justifyContent: 'space-between' }}>
          <Button danger onClick={handleDelete}>删除</Button>
          <Button type="primary" onClick={handleSave}>保存</Button>
        </div>
      }
    >
      <Form form={form} layout="vertical" initialValues={widget}>
        {/* 数据源配置 + 显示配置 */}
      </Form>
    </Drawer>
  );
};
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| 内联样式 | Tailwind CSS + Ant Design style | 项目初始 | 一致的设计系统 |
| Redux | Zustand | 项目初始 | 更简洁的状态管理 |
| 自定义表单 | Ant Design Form | 项目初始 | 内置验证和错误处理 |

**Deprecated/outdated:**
- 无已弃用模式，项目使用现代 React 19 + TypeScript 技术栈

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | Ant Design 6 Modal/Drawer API 与 v5 兼容 | Architecture Patterns | 需查阅迁移文档 |
| A2 | dashboardStore 的 updateWidget 支持部分更新 | Code Examples | 需验证 store 实现 |

**验证结果:**
- A1: Ant Design 6 Modal API 保持兼容，新增 `destroyOnHidden` 属性替代 `destroyOnClose` [VERIFIED: 现有代码已使用]
- A2: dashboardStore 的 updateWidget 使用 `{ ...w, ...updates }` 实现部分更新 [VERIFIED: dashboardStore.ts L295-307]

## Open Questions

1. **模板预览数据获取策略**
   - What we know: 后端有 getBatchWidgetData API
   - What's unclear: 是否需要权限验证？预览数据是否需要缓存？
   - Recommendation: 使用现有 getBatchWidgetData API，失败时显示错误提示，不缓存预览数据

2. **刷新间隔自定义输入实现**
   - What we know: D-06 决策要求预设 + 自定义
   - What's unclear: 自定义输入的 UI 形式（InputNumber + 单位选择？）
   - Recommendation: 使用 Select 预设选项 + 条件显示 InputNumber（选择"自定义"时显示）

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| React | 所有组件 | ✓ | 19.2 | — |
| Ant Design | Modal/Drawer/Form | ✓ | 6.1.1 | — |
| Zustand | 状态管理 | ✓ | 5.0.9 | — |
| TypeScript | 类型检查 | ✓ | 5.9 | — |
| react-router-dom | 路由跳转 | ✓ | 7.10.1 | — |

**Missing dependencies with no fallback:** 无

**Missing dependencies with fallback:** 无

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Vitest 4.0.18 |
| Config file | vitest.config.ts (项目根目录) |
| Quick run command | `npm run test` |
| Full suite command | `npm run test:coverage` |

### Phase Requirements → Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| FE-02a | Widget 选择器添加 Widget | unit | `vitest run WidgetSelector.test.tsx` | ❌ Wave 0 |
| FE-02a | Widget 选择器编辑 Widget | unit | `vitest run WidgetSelector.test.tsx` | ❌ Wave 0 |
| FE-02b | 仪表盘设置保存 | unit | `vitest run DashboardSettings.test.tsx` | ❌ Wave 0 |
| FE-02b | 权限范围选择 | unit | `vitest run DashboardScopeSelector.test.tsx` | ❌ Wave 0 |
| FE-02c | 模板预览渲染 | unit | `vitest run TemplatePreview.test.tsx` | ❌ Wave 0 |
| FE-02c | 模板应用创建仪表盘 | integration | `vitest run TemplatePreview.test.tsx` | ❌ Wave 0 |
| FE-02d | Widget 编辑保存 | unit | `vitest run WidgetEditor.test.tsx` | ❌ Wave 0 |
| FE-02d | Widget 删除确认 | unit | `vitest run WidgetEditor.test.tsx` | ❌ Wave 0 |

### Sampling Rate
- **Per task commit:** `npm run test` (快速运行)
- **Per wave merge:** `npm run test:coverage` (覆盖率检查)
- **Phase gate:** Full suite green before `/gsd-verify-work`

### Wave 0 Gaps
- [ ] `ruoyi-react-frontend/src/components/dashboard/settings/__tests__/WidgetSelector.test.tsx` — covers FE-02a
- [ ] `ruoyi-react-frontend/src/components/dashboard/settings/__tests__/DashboardSettings.test.tsx` — covers FE-02b
- [ ] `ruoyi-react-frontend/src/components/dashboard/layout/__tests__/TemplatePreview.test.tsx` — covers FE-02c
- [ ] `ruoyi-react-frontend/src/components/dashboard/widgets/__tests__/WidgetEditor.test.tsx` — covers FE-02d
- [ ] Framework install: 已安装 — 无需额外安装

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | no | — |
| V3 Session Management | no | — |
| V4 Access Control | yes | DashboardScopeSelector 基于用户权限限制可见范围选项 |
| V5 Input Validation | yes | Ant Design Form rules + TypeScript 类型检查 |
| V6 Cryptography | no | — |

### Known Threat Patterns for React + Ant Design

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| XSS (用户输入) | Tampering | React 自动转义 + Ant Design Input 组件 |
| CSRF | Tampering | API 层已实现 Token 认证 (api.ts) |
| 权限绕过 | Elevation | DashboardScopeSelector 检查用户 dataScope |

## Sources

### Primary (HIGH confidence)
- 现有代码文件: `dashboardStore.ts`, `dashboardService.ts`, `widgetRegistry.ts`, `WidgetSelector.tsx`, `EndpointSelector.tsx`, `DashboardScopeSelector.tsx`, `TemplateSelector.tsx` - [完整实现参考]
- 类型定义: `ruoyi-react-frontend/src/types/dashboard.ts` - [完整类型定义]
- 项目指令: `AGENTS.md` - [编码规范和 API 调用模式]

### Secondary (MEDIUM confidence)
- package.json - [依赖版本确认]

### Tertiary (LOW confidence)
- 无

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - 所有依赖已存在于项目中，版本已验证
- Architecture: HIGH - 现有组件提供完整参考，模式清晰
- Pitfalls: HIGH - 基于现有代码审计和 React 最佳实践

**Research date:** 2026-04-20
**Valid until:** 30 days (技术栈稳定)
