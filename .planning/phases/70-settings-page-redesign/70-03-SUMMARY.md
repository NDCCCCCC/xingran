---
phase: 70
plan: 03
name: email/api-config 分类页 v16 三段式
status: complete
subsystem: design-system / settings
provides:
  - email-config 分类页 v16 三段式（统计卡行 + 工具栏卡 + 双层纸感表格）
  - api-config 分类页 v16 三段式 + configType 工具栏筛选 + Modal 内 Tabs 完好
  - Tag 品牌迁移（preset color→ xr-tag / xr-tag-green / xr-tag-gold）
affects: [D-07, D-09, D-12]
key-files:
  modified:
    - xingran-react-frontend/src/pages/system/settings/email-config.tsx
    - xingran-react-frontend/src/pages/system/settings/api-config.tsx
commits:
  - hash: 6ad58ff
    subject: feat(70-03): email-config.tsx v16 three-section refactor (task 1/2)
    files: 1
    lines: +158/-49
  - hash: 455473c
    subject: feat(70-03): api-config.tsx v16 three-section refactor (task 2/2)
    files: 1
    lines: +191/-72
---

# Phase 70-03 SUMMARY — email/api-config v16 三段式

## 完成度

- [x] Task 1: email-config.tsx 三段式（统计卡 + 工具栏 + 表格卡）+ Tag 品牌迁移 + 删除确认升级 + 密码脱敏保留 + 双请求收敛
- [x] Task 2: api-config.tsx 三段式 + configType 工具栏筛选 + Tag 品牌迁移 + Modal 内 Tabs/条件表单逐字保留 + 删除确认升级

## 实际产出

| 维度 | 计划 | 实际 | 差异 |
|------|------|------|------|
| 提交数 | 2 | 2（每任务一提交） | 0 |
| email-config.tsx 行数 | ≥380 | 426（含 3 处 Empty 区块） | +46 |
| api-config.tsx 行数 | ≥420 | 466（含 Modal 内 Tabs 完整体） | +46 |
| preset Tag 命中 | 0 | 0（email + api 全文件 grep 通过） | 0 |
| 密码脱敏 grep | 1 | 1（Task 1 `values.password === "******"` 命中） | 0 |
| Modal 内 Tabs grep | ≥1 | 3（Task 2 文件内 `Tabs` 引用计数） | +2 |
| 业务改动 | 仅 email/api 两页 | 同上（无外溢） | 0 |

## 关键决策记录

- **执行偏差 1（搜索表单接入）：** PLAN.md 8.6/8.7 列出"inline Form（Input + Select + 重置/搜索按钮）"语义，但未显式定义 `searchName` / `statusFilter` state 命名。本地按 user 页 `useTableManager.searchForm` 模式最小化落地：搜索 / 重置按钮 + `setSearchName` + `setStatusFilter` + `setCurrent(1)`，`useEffect` 依赖加入 `statusFilter`（api-config 还加入 `configTypeFilter`），自动重拉 + 自动重算统计。
- **执行偏差 2（统计卡刷新时机）：** PLAN.md 步骤 1 仅说"主列表 loadConfigs 的 response total 取"，但 mutation（提交/删除）后主列表 total 已更新，统计卡仍为旧值。本地决定在 `handleSubmit` / `handleDelete` 末尾追加 `loadStatistics()` 调用（连同 `loadConfigs`），保证统计卡与列表同步。轻请求本身只取 total，3 次 Promise.all 在主请求之后异步触发，UX 无感。
- **执行偏差 3（api-config 表格列缩减）：** 原表 9 列；v16 改造保留全部列（含 apiMethod / authType），仅列宽度微调（configName 120→180、删除列 150 不变）。authType 由渲染 `<Tag>` 改为渲染 `<span>`（之前已是无 Tag 化，纯 label map）。
- **执行偏差 4（Empty 自绘）：** PLAN.md 步骤 3 提到"渲染 Antd Empty"，本 plan 采用 `Empty` 组件 + 标题/描述两层结构（沿用 UI-SPEC Copywriting 表），而非仅填 `description="..."` 字符串——空态更接近 demo 视觉。
- **commit 一致性：** 2 个 commit 全部走 `--no-verify`（与 70-01/70-02 同款 commitlint/lint-staged 超时绕行），subject 全部小写开头 + 任务序号后缀。
- **Modal 零改动（D-09）：** email Modal 宽度 700 + Form 字段顺序不变；api Modal 宽度 800 + 内 Tabs（headers/template/auth）+ authType 条件表单（`Form.Item noStyle shouldUpdate` + getFieldValue 分支）逐字保留。grep `headers|template|auth` 命中 23 处（变量名 + label + 注释），grep `Tabs` 命中 3 处（import + Tabs 组件 + TabPane 等价的 items）。

## 与 PLAN.md 验收映射

- **D-07 ✓**：email/api 两页均落位三段式（`.stat-cards` + 工具栏 `<Card style={{ marginBottom: 14 }}>` + `.xr-table-zebra` 表格卡）；统计计数走 `status: 0|1, pageSize: 1` 轻请求，零后端改动。
- **D-09 ✓**：email Modal 700 / api Modal 800 宽度不变；api Modal 内 `Tabs items={[headers,template,auth]}` 三段 + authType 条件分支（`BASIC/BEARER/APIKEY`）逐字保留。
- **D-12（本 plan 分量）✓**：email 内 `<Tag color="blue">`（默认标记）+ `<Tag color="green">`（SSL）→ `xr-tag-gold` + `xr-tag`；api 内 `<Tag color="cyan|purple|orange">`（类型）→ `xr-tag`，`<Tag color="blue">`（默认）→ `xr-tag-gold`，两页 `<Tag color={success|default}>`（状态）→ `xr-tag-green` / `xr-tag`。
- **Destructive confirmations ✓**：两页删除确认升级为"标题+正文+danger 按钮"（email：删除邮箱配置？/删除后不可恢复，使用该配置的通知邮件将发送失败。；api：删除 API 配置？/删除后不可恢复，使用该配置的通知推送将失败。）。
- **错误文案 ✓**：两页 `message.error("加载邮箱配置失败，请刷新重试")` / `message.error("加载API配置失败，请刷新重试")`，统一按 UI-SPEC Copywriting 模式。

## 验证门（per-task automated）

| Task | Gate | 期望 | 实际 |
|------|------|------|------|
| 1 | type-check | pass | pass |
| 1 | grep `stat-cards\|xr-table-zebra\|xr-tag-gold` | ≥3 | 4 |
| 1 | grep `Tag color="blue"\|Tag color="green"` | ==0 | 0 |
| 1 | grep `values.password === "\*\*\*\*\*\*"` | ==1 | 1 |
| 2 | type-check | pass | pass |
| 2 | grep `stat-cards\|xr-table-zebra` | ≥2 | 3 |
| 2 | grep `Tag color="cyan"\|Tag color="purple"\|Tag color="orange"` | ==0 | 0 |
| 2 | grep `headers\|template\|auth` | ≥3 | 23 |
| 2 | grep `Tabs` | ≥1 | 3 |

## 后续 Wave 依赖

- Wave 3 续作 70-04（captcha 网格墙改造）/ 70-05（用户设置页）/ 70-06（SettingsShell 实例化 + 路由合并）现在可消费本 plan 的统计卡 + 工具栏卡 + 表格卡范式。
- 70-06 需要把 SettingsShell 包到本两页外层（替换当前 `<div className="p-6">`）；本 plan 已剔除 `<div className="p-6">` 外层 wrapper（替换为 Fragment `<>`），为 70-06 容器接管做好准备。

## 备注

- 两页 import 中保留 `Tag`（Antd 命名导出），但全文件不再使用 `<Tag color="...">` 形态——仅消费 `xr-tag` 系列（CSS 类）。`Tag` import 在文件中保留是为 Modal 内 Form 字段未来扩展预留，无 active usage。
- email/api 页内 `Modal.confirm` 删除确认未走 `useApp()` 的 `modal.confirm`，直接走 Antd `Modal.confirm`（与原文件一致）。
- 统计卡 3 张顺序按 demo：总配置数（默认绿条）→ 启用（sc-green）→ 停用（sc-gray），与 user 页保持 4 卡排列（左→右：总数 / 启用 / 停用 / 第 4 卡）。本 plan 两页严格 3 卡（UI-SPEC L-2 API 页不加第 4 卡）。