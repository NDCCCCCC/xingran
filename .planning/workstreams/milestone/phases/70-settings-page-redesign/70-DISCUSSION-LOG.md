# Phase 70: settings-page-redesign - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-08-19
**Phase:** 70-settings-page-redesign
**Areas discussed:** 系统设置布局形态, 两页视觉关系, Tab 内部 CRUD 布局, 目录整理与残留清理

---

## 系统设置布局形态

### Q1 布局形态

| Option | Description | Selected |
|--------|-------------|----------|
| 左侧分类导航 | GitHub/Vercel/GitLab 式：左侧竖排分类导航，右侧渲染配置页；双层纸感契合；未来新增分类零成本 | ✓ |
| 保留 Tabs 品牌化 | 改动最小，Antd Tabs 已接令牌；但与「重新设计布局」目标张力大 | |
| 单页分区锚点 | 一页滚动+锚点吸顶；重 CRUD 页堆叠会很长 | |

### Q2 内容宽度策略

| Option | Description | Selected |
|--------|-------------|----------|
| 混合策略 | 表格类撑满（对齐 v16），纯表单类限宽 720-880px | ✓ |
| 全部撑满 | 与 v16 完全一致；表单行较宽 | |
| 全部限宽 | 经典设置页范式；表格列多时拥挤 | |

### Q3 激活分类持久化

| Option | Description | Selected |
|--------|-------------|----------|
| URL 参数化 | ?cat=email：可分享/收藏/刷新还原，active 与 URL 同步 | ✓ |
| 保持 localStorage | 沿用 usePersistedStateController；改动最小但不可分享 | |

### Q4 窄屏降级

| Option | Description | Selected |
|--------|-------------|----------|
| 顶部 segmented | <lg 时降级为顶部横向 segmented，内容全宽 | ✓ |
| 收窄为图标列 | 保持左右结构，图标语义弱 | |
| Claude 决定 | 按现有布局断点惯例定 | |

---

## 两页视觉关系

### Q1 骨架同构

| Option | Description | Selected |
|--------|-------------|----------|
| 完全同构 | 共用 SettingsShell；用户设置左导航=界面/布局/数据，系统设置=邮箱/API/验证码；合并菜单入口需动 sys_menu，不做 | ✓ |
| 各自最优 | 用户设置保持轻量单卡片表单仅品牌化；两页形态不一致 | |

### Q2 表单形态

| Option | Description | Selected |
|--------|-------------|----------|
| 行式设置项 | label+描述+右侧控件，分组卡片包裹；明暗模式分段卡片选择器 | ✓ |
| 垂直表单品牌化 | 保持 Antd 垂直 Form 仅接令牌；较传统 | |

---

## Tab 内部 CRUD 布局

### Q1 对齐程度

| Option | Description | Selected |
|--------|-------------|----------|
| 完整对齐 v16 | 统计卡行（总配置数/启用/停用）+品牌工具栏+双层纸感表格 | ✓ |
| 工具栏+表格 | 不加统计卡；工作量减半但与 v16 头部不一致 | |
| 仅令牌继承 | 不动结构；与 phase 目标不符 | |

### Q2 验证码页形态

| Option | Description | Selected |
|--------|-------------|----------|
| 图片网格墙 | 缩略图卡片网格+上传/启用/删除；贴合图片管理语义 | ✓ |
| 同范式表格 | 统计卡+表格列表；一致性强但图片浏览体验弱 | |

### Q3 表单容器

| Option | Description | Selected |
|--------|-------------|----------|
| 保持 Modal | 仅接品牌令牌；符合纯布局重构边界 | ✓ |
| 改 Drawer | 长表单体验更好；交互模式升级，重构面变大 | |

---

## 目录整理与残留清理

### Q1 未提交改动处置

| Option | Description | Selected |
|--------|-------------|----------|
| phase 吸收提交 | 首任务原子提交 default-theme 清理（-716 行），正是 phase 目标一部分 | ✓ |
| 用户先自行提交 | phase 从干净树开始；忘记提交会与重构混合 | |

### Q2 目录整理

| Option | Description | Selected |
|--------|-------------|----------|
| 合并+菜单迁移 | 合并为单一 system/settings/ + sys_menu 组件路径数据迁移 SQL；pages/settings/ 不变 | ✓ |
| 目录不动 | 仅内部重构；零菜单风险但目录混乱残留 | |
| 仅合并两个 system 子目录 | 路径保持可寻址；需验证 glob 兼容 | |

### Q3 清理边界

| Option | Description | Selected |
|--------|-------------|----------|
| settings 范围 | 持久化键/旧 className/页内硬编码色/settingsStore 死字段 | ✓ |
| 全仓扫描 | 越出本 phase 范围，工作量不可控 | |

---

## Todos 交叉审查

| Todo | 匹配分 | 处置 |
|------|--------|------|
| operlog.exclude_paths 配置驱动白名单 | 0.6（关键词浅匹配） | 审阅未合并（后端运维项，无关） |
| v1.17-reconciliation-decisions | 0.6（关键词浅匹配） | 审阅未合并（资产模块待办，无关） |

## Claude's Discretion

- SettingsShell 组件归属目录与命名
- API 配置页统计卡指标口径
- URL 参数名（示例 ?cat=）
- 明暗模式分段卡片选择器具体视觉（UI-SPEC 细化）
- 图片网格墙列数/卡片比例/空状态（UI-SPEC 细化）
- QA 验证方式（建议 QA-04 风格截图对比 + build/lint/test 回归门）
- system/captcha-background/ 与 settings/captcha-background.tsx 关系盘点

## Deferred Ideas

- 合并用户设置与系统设置为单一菜单入口（需动 sys_menu 菜单结构，越纯前端边界）
- 配置表单容器升级 Drawer（本 phase 保持 Modal）
- PROTO 系列逐屏对齐（PROTO-01~04，Future Requirements）
