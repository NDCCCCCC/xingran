# 原型 Shell 复刻规范（Prototype Shell Reproduction Spec）

> **生成时间**：2026-08-18
> **来源**：`655aa291-9bfe-4e94-ad5d-b3c8b2d24984/` 53 张 HTML 原型 + `assets/theme.css` 主题 + v1.22 实测现状对比
> **目的**：把 prototype 的 sidebar / header / 每页标题 / 样式结构抽取成可执行规范，给 v1.22 后 shell 复刻提供基准
> **保留**：v1.22 已落地的品牌化色彩（深绿/铜金/奶油 + `#14532D` 侧栏实色、`#156031` 主按钮绿等）
> **修正方向**：把原型 CSS 中更精细的渐变/字体/分隔符结构真实落到 `xingran-react-frontend` 上

---

## 1. 核心尺寸对比

| 元素 | 原型规范 | v1.22 实测 | gap |
|---|---|---|---|
| **侧栏宽度** `--sidebar-w` | **240px** | 200px（layoutStore 默认）/ 280px（旧 layout）/ 折叠 64-80px | 中 |
| **顶栏高度** `--header-h` | **56px** | **64px**（v1.22 已统一改）| 需保留 v1.22 64px |
| **多页签栏高** `--tabs-h` | **40px** | 22px（实测 `.ant-tabs-nav` 高度）| 真实缺独立 tabs 栏，Antd 内嵌 |
| **正文 padding** `0 20px`（header） | 标准 | 待对齐 | 低 |
| **正文 main `padding: 24px`** | 标准 | 待对齐 | 低 |
| **卡片圆角** `border-radius: 10px` | 标准 | 待对齐 | 低 |

> **决策**：保留 v1.22 已改的 `--header-h: 64px`（CLAUDE.md 锚点「顶栏 64px」）；侧栏 240px vs 200/280px 折中 → 后续阶段讨论（layoutStore 已记录 64/56/80 折叠宽度）

## 2. 侧栏（`.sidebar`）复刻规范

### 2.1 原型 CSS（`assets/theme.css`）

```css
.sidebar {
  width: var(--sidebar-w);          /* 240px */
  flex-shrink: 0;
  display: flex;
  flex-direction: column;
  background: linear-gradient(168deg, #14532D 0%, #156031 58%, #1A6839 100%);
  color: var(--on-green);            /* #E0E0B0 浅黄 */
  position: sticky;
  top: 0;
  height: 100vh;
  transition: width 0.25s var(--ease);
}
```

### 2.2 v1.22 实测

```css
--sidebar-bg: #14532D (实色，无渐变)
```

### 2.3 差距与修复建议

| 项 | 原型 | 真实 | 修复 |
|---|---|---|---|
| 背景 | **渐变**（168deg 三阶品牌绿） | 实色 `#14532D` | 把 `--sidebar-bg` 改为 `linear-gradient(168deg, #14532D 0%, #156031 58%, #1A6839 100%)`（保留 v1.22 品牌色但补渐变质感） |
| 侧栏宽度 | 240px | 200/280px | 引入 `--sidebar-w: 240px` 作为 layoutStore 默认值；保留折叠 64/56/80 宽度切换 |
| 折叠态宽度 | 56px（推断）| 64/80px | layoutStore 已有 56px 默认折叠；原型折叠时仅显品牌 mark 与折叠按钮 |
| `.sidebar-brand` | `height: var(--header-h)`；品牌 mark + wordmark（中文/英文） | 已有 | ✓ 保留 |
| `.sidebar-nav` | flex 1列 + 滚动 | 已有 | ✓ 保留 |
| `.sidebar-foot` | 版本号 / 系统名 | 已有 | ✓ 保留 |

## 3. 顶栏（`.header`）复刻规范

### 3.1 原型 CSS

```css
.header {
  height: var(--header-h);          /* 56px 原型 / 64px v1.22 保留 */
  background: var(--surface);        /* #FFFFFF 白底 */
  border-bottom: 1px solid var(--border);  /* #DBD7CE */
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 0 20px;
  position: sticky;
  top: 0;
}
```

### 3.2 v1.22 实测

- 高度 64px ✓
- 白底 ✓
- border-bottom #DBD7CE ✓
- padding `0 24px`（实测 header 类为 `px-6` = 24px，原型 20px） — 微差 4px
- 面包屑「系统管理 / 用户管理」✓
- 右侧 actions（铃铛/搜索 ⌘K/用户菜单）✓

### 3.3 差距与修复

| 项 | 状态 |
|---|---|
| 顶栏结构 | ✓ 基本一致（除 4px padding 差异） |
| 面包屑分隔符 | 原型 `<span class="sep">`（颜色 `--border-strong`） — 真实用 Antd `Separator`，可接受 |
| ⌘K 全局搜索 | ✓ v1.22 已落地（brand ⌘K Tag） |
| 用户菜单 | ✓ 含个人中心 / 系统设置 / 退出 |

> **无需变更** —— 顶栏已与原型 95% 一致。

## 4. 多页签栏（`.tabs`）复刻规范

### 4.1 原型 CSS

```css
.tabs {
  height: var(--tabs-h);            /* 40px */
  background: var(--surface);        /* 白 */
  border-bottom: 1px solid var(--border);
  display: flex;
  align-items: stretch;
  padding: 0 12px;
  gap: 2px;
  overflow-x: auto;
  scrollbar-width: none;
}
```

### 4.2 v1.22 实测

- **缺失独立 tabs 栏** —— Antd `Tabs` 组件内嵌（不同语义），不是顶部多页签栏
- 真实 `TabBar` 组件（v1.22 之前存在，Phase 65 删除多主题时可能一并清理）需复核

### 4.3 差距与修复

| 项 | 修复 |
|---|---|
| 缺失顶部多页签栏 | **v1.23+ 决定**：要么恢复（经典后台模式），要么确认是「当前设计意图是无 tabs 栏」（与若依新版趋势一致） |

> **关键决策点**：用户需要明确「是否需要顶部多页签栏」。若需要，Phase 65 删除的 TabBar.tsx 应恢复（仅复刻结构，不引入多主题能力）。

## 5. 每页标题（`.page-title`）复刻规范

### 5.1 原型格式（57 个标题样本）

```html
<h1 class="page-title">系统<span class="dot">·</span>用户</h1>
<h1 class="page-title">工位<span class="dot">·</span>管理</h1>
<h1 class="page-title">运维<span class="dot">·</span>总览</h1>
```

**核心特征**：
- 标题分两段，中文短语 + `<span class="dot">·</span>`（金色分隔符）+ 中文短语
- 共 45 个唯一前缀：系统/工位/运维/工单/楼宇/资产/网络/设备/知识库/RPA/VDI/...
- 完整列表：见下方「标题映射表」

### 5.2 原型 CSS

```css
.page-title {
  font-family: var(--font-display);     /* 宋体/Serif 字体栈 */
  font-size: 24px;
  font-weight: 700;
  letter-spacing: -0.01em;
  line-height: 1.25;
}
.page-title .dot {
  color: var(--gold);                   /* #C09058 铜金 */
  font-weight: 300;                     /* 分隔符更轻盈 */
  margin: 0 0.15em;                     /* 前后间距 */
}
```

### 5.3 v1.22 实测

- `font-size: 20px`（实测）vs 原型 `24px`（差 4px）
- `font-weight: 700` ✓
- **缺 `.page-title` 类名** —— 真实系统用 `text-xl font-bold`（Tailwind）；Antd 框架 + 自定义样式
- **缺 `.dot` 金色分隔符** —— 真实标题是单段（如 "工位管理"），无双段结构

### 5.4 差距与修复

| 项 | 修复 |
|---|---|
| `.page-title` 类与样式 | 引入 `.page-title` 类：`font-family: var(--font-display); font-size: 24px; font-weight: 700; letter-spacing: -0.01em; line-height: 1.25;` |
| `.page-title .dot` 金色分隔符 | 引入 `.page-title .dot` 类：`color: var(--gold, #C09058); font-weight: 300; margin: 0 0.15em;` |
| 标题分段 | **53 个页面标题按「前缀·后缀」格式重写** —— 这是修复的核心工作量 |

### 5.5 53 张原型标题映射表（完整）

| # | 真实路由 | 原型文件 | 标题（page-title） |
|---|---|---|---|
| 1 | `/dashboard` | dashboard.html | 运维·总览 |
| 2 | `/system/user` | system-users.html | 系统·用户 |
| 3 | `/system/role` | system-roles.html | 角色·管理 |
| 4 | `/system/menu` | system-menus.html | 菜单·管理 |
| 5 | `/system/dept` | system-dept.html | 部门·管理 |
| 6 | `/system/post` | system-post.html | 岗位·管理 |
| 7 | `/system/dict` | system-dict.html | 字典·管理 |
| 8 | `/system/config` | system-config.html | 参数·配置 |
| 9 | `/system/notice` | system-notice.html | 通知·公告 |
| 10 | `/system/settings` | system-config.html | （settings 内嵌） |
| 11 | `/system/list` | system-keys.html | 密钥·列表 |
| 12 | `/operations/workstations` | workstations.html | 工位·管理 |
| 13 | `/operations/building` | buildings.html | 楼宇·管理 |
| 14 | `/operations/building-spaces` | building-spaces.html | 楼宇·空间 |
| 15 | `/operations/building-spaces-3d` | building-spaces-3d.html | 楼宇空间·3D |
| 16 | `/operations/floor` | floors.html | 楼层·管理 |
| 17 | `/operations/server-room` | server-rooms.html | 机房·管理 |
| 18 | `/operations/room-device` | room-devices.html | 机房·设备管理 |
| 19 | `/operations/info-point` | info-points.html | 信息点·管理 |
| 20 | `/operations/dedicated-line` | dedicated-lines.html | 专线·管理 |
| 21 | `/ops/workorder/orders` | workorders.html | 工单·管理 |
| 22 | `/ops/workorder/categories` | workorder-categories.html | 工单·分类 |
| 23 | `/ops/workorder/statistics` | workorder-statistics.html | 工单·统计 |
| 24 | `/ops/workorder/workorder/periodic/templates` | periodic-workorders.html | 周期性·工单 |
| 25 | `/ops/duty/schedules` | duty-schedules.html | 排班·管理 |
| 26 | `/ops/duty/management` | duty-config.html | 值班·配置 |
| 27 | `/ops/duty/my-duty` | my-duty.html | 我的·值班 |
| 28 | `/ops/knowledge/articles` | knowledge-articles.html | 知识库·文章 |
| 29 | `/ops/knowledge/view` | knowledge-view.html | 知识库·查看 |
| 30 | `/ad-domain/users` | ad-users.html | 域控·用户 |
| 31 | `/ad-domain/groups` | ad-groups.html | 域·用户组 |
| 32 | `/ad-domain/computers` | ad-computers.html | 电脑·设备 |
| 33 | `/ad-domain/ous` | ad-ous.html | 组织单元·OU |
| 34 | `/ad-domain/logs` | ad-sync-logs.html | 同步·日志 |
| 35 | `/ad-domain/configs` | ad-config.html | 域控·配置 |
| 36 | `/assets/list` | assets.html | 资产·管理 |
| 37 | `/assets/dashboard` | reconcile-board.html | 资产·对账看板 |
| 38 | `/assets/exceptions` | asset-anomalies.html | 异常·列表 |
| 39 | `/assets/exception-rules` | exception-rules.html | 例外规则·管理 |
| 40 | `/assets/fix-suggestion` | repair-suggestions.html | 修复·建议 |
| 41 | `/vdi/vm` | vm-list.html | 虚拟机·列表 |
| 42 | `/vdi/servers` | vdi-config.html | VDI·服务器配置 |
| 43 | `/vdi/rpa` | rpa-management.html | RPA·管理 |
| 44 | `/monitor/dashboard` | monitor-dashboard.html | 监控·告警中心 |
| 45 | `/monitor/server` | server-monitor.html | 服务·监控 |
| 46 | `/monitor/logs` | system-logs.html | 日志·管理 |
| 47 | `/monitor/scheduled-jobs` | scheduled-jobs.html | 定时·任务 |
| 48 | `/monitor/cache-management` | cache-management.html | 缓存·管理 |
| 49 | `/alerts` | alerts.html | 监控·告警中心 |
| 50 | `/audit` | audit.html | 国密·审计日志 |
| 51 | `/index` | index.html | 方案总览（root） |
| 52 | `/network/device-discovery` | device-discovery.html | 设备·发现 |
| 53 | `/network/device` | device-management.html | 设备·管理 |
| 54 | `/network/device-credentials` | device-credentials.html | 授权·凭证 |
| 55 | `/network/command-distribution` | command-distribution.html | 命令·分发 |
| 56 | `/network/config-execution` | config-execution.html | 配置·执行 |
| 57 | `/network/config-backup` | config-backup.html | 配置·备份 |
| 58 | `/network/config-templates` | config-templates.html | 配置·模板 |
| 59 | `/network/mac-address` | mac-address.html | MAC·地址 |

> 真实路由与原型 slug 在 PROTOTYPE-VS-ACTUAL.md §0.1 已记录差异；本规范以「路由 → 标题」映射对齐为目标。

## 6. 各屏标题实现方式建议

### 6.1 数据驱动方案（推荐）

```tsx
// 设计：维护 titleMap + 路由 meta，自动渲染 .page-title
// src/design-system/components/PageTitle.tsx (新建)
type PageTitleProps = { routeKey: string };
const titleMap: Record<string, [string, string]> = {
  '/system/user': ['系统', '用户'],
  '/operations/workstations': ['工位', '管理'],
  '/dashboard': ['运维', '总览'],
  // ... 53 条
};

export const PageTitle = ({ routeKey }: PageTitleProps) => {
  const [pre, post] = titleMap[routeKey] || ['未命名', ''];
  return (
    <h1 className="page-title">
      {pre}{post && <span className="dot">·</span>}{post}
    </h1>
  );
};
```

### 6.2 各页面集成

```tsx
// /src/pages/system/user/index.tsx
import { PageTitle } from '@/design-system/components/PageTitle';

const UserPage = () => (
  <>
    <PageTitle routeKey="/system/user" />
    <div className="page-content">{/* filters + table */}</div>
  </>
);
```

### 6.3 全局样式补齐

```css
/* src/index.css 末尾补 */
.page-title {
  font-family: 'Songti SC', 'Source Han Serif SC', ui-serif, Cambria, Georgia, serif;
  font-size: 24px;
  font-weight: 700;
  letter-spacing: -0.01em;
  line-height: 1.25;
  color: var(--theme-text-primary);
  margin: 24px 0 16px;
}
.page-title .dot {
  color: var(--theme-brand-accent, #C09058);  /* 铜金 */
  font-weight: 300;
  margin: 0 0.15em;
}
```

## 7. 修复工作量评估

| 任务 | 工作量 | 风险 |
|---|---|---|
| 侧栏渐变背景补齐 | 0.5h | 极低（CSS 单变量替换）|
| `.page-title` 类与 `.dot` 样式 | 1h | 极低（CSS 新增）|
| `PageTitle` 组件 + 53 条 titleMap | 2h | 低（纯数据）|
| 53 个页面替换 `<h1>` | 4-6h | 中（涉及 grep/replace 53 文件；需逐页确认路由映射）|
| 顶部多页签栏（若需要）| 3-5h | 中（Phase 65 删除的 TabBar 恢复） |
| 顶栏 padding 微调（24→20px） | 0.5h | 极低 |
| **合计** | **11-15h ≈ 1-2 天** | |

## 8. 范围与启动建议

- **v1.23 候选 milestone 名称**：`Frontend Shell Reproduction` 或 `Prototype Shell Fidelity`
- **ROADMAP 分 phase 建议**：
  - Phase 68: 侧栏渐变 + `.page-title` 基础 + PageTitle 组件
  - Phase 69: 53 屏标题替换（4-6h 批量）+ QA 复刻
  - Phase 70:（可选）顶部多页签栏恢复 + 顶栏 padding 调整

## 9. 与 v1.22 design-system 层兼容

| 已兼容项 | 备注 |
|---|---|
| 深绿 / 铜金 / 奶油色 | ✓ 保留 `#14532D` / `#156031` / `#1A6839` 渐变三阶 + `#C09058` 金色 dot |
| 字体栈 | 沿用 v1.22 引入的 Songti SC / Source Han Serif SC / PingFang SC |
| 按钮 / 表头 / Tag | 不动（v1.22 已落地）） |
| `--sidebar-bg` / `--header-bg` 变量 | 保留 —— 渐变改为 `.sidebar` 类内部 `linear-gradient()` 即可，与变量系统无冲突 |

---

**建议下一步**：将本规范作为 v1.22+ shell 复刻 phase 的 UI-SPEC 输入；启动 `/gsd:new-milestone` 创建新 milestone + ROADMAP（Phase 68-70 三 phase）；或先用 `/gsd:sketch` 把 prototype 的某 1 屏 1:1 还原成 `<iframe>` 对照 demo，再决定是否进入正式施工。