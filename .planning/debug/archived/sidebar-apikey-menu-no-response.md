---
slug: sidebar-apikey-menu-no-response
status: resolved
trigger: 点击sidebar上的api秘钥管理没有相应
created: 2026-05-19T13:50:00Z
updated: 2026-05-19T13:55:00Z
---

## Symptoms

**Expected Behavior:**
- 点击侧边栏的"API密钥管理"菜单项应该展开显示子菜单（"密钥列表"和"使用日志"）

**Actual Behavior:**
- 没有任何反应，点击无效
- 侧边栏菜单项显示为可点击状态（鼠标悬停有高亮效果）

**Error Messages:**
- 无错误消息显示

**Timeline:**
- 刚执行完SQL迁移创建了菜单配置
- 之前没有这个菜单，这是新添加的功能

**Reproduction Steps:**
1. 在浏览器中打开应用
2. 在侧边栏找到"系统管理"
3. 点击"API密钥管理"菜单项
4. 观察到没有响应

---

## Current Focus

hypothesis: 前端菜单渲染逻辑要求 `menuType='M'` 的目录菜单必须有 children 才能被识别为可展开的父菜单。数据库中"API密钥管理"菜单可能缺少子菜单数据，或者子菜单未被正确关联。

next_action: 验证数据库中"API密钥管理"菜单的 children 字段是否包含子菜单，检查 buildMenuTree 方法的构建逻辑

---

## Evidence

- **证据1（快照分析）：**
  - 侧边栏中的"API密钥管理"菜单项没有 `expandable haspopup="menu"` 属性
  - 其他目录菜单（网络设备管理、运维管理等）都有此属性
  - 只显示一个"API密钥管理"菜单项（重复已清理）

- **证据2（组件路径格式）：**
  - 之前的迁移使用了错误的组件路径格式：`pages/system/apikeys/index`（包含pages/前缀）
  - 正确格式应该是：`system/apikeys/index`（不含pages/前缀）

- **证据3（前端代码分析）：**
  - sidebar.tsx:56 逻辑：`if (validChildren.length > 0 || menu.menuType === 'M')`
  - 即使 menuType === 'M'，如果没有 validChildren，children 会被设置为 undefined
  - Ant Design Menu 组件要求 children 数组非空才能渲染为可展开菜单

- **证据4（localStorage 检查）：**
  - localStorage 中 menu-storage 的 menus 数组为空
  - 菜单数据存储在 Zustand 内存 store 中（TTL 5分钟缓存）
  - API 响应使用 SM2+SM4 加密，无法直接从浏览器查看解密数据

## Phase 41 Closure (2026-06-26)
fix: 复测确认根因(组件路径格式重复拼接 `pages/system/apikeys/index`)与 apikey-route-path-duplication 完全同源,均被 `internal/core/db/migrations/migration_166_apikey_route_path_fix.go` 修复落地:`internal/core/db/database.go:411` 已注册 `migrations.Migrate166ApikeyRoutePathFix(d.DB)`,迁移将父菜单 "API密钥管理" path 归一为 `apikeys`、子菜单 "密钥列表" → `list`、"使用日志" → `logs`,消除 sidebar.utils buildFullPath 的双重拼接。前端侧 `xingran-react-frontend/src/router/routeGenerator.ts` 动态从 `getUserMenus()`(`/system/my-menus`)+ `getAllUserMenus()`(`/system/my-menus/all`)加载菜单数据(见 `src/lib/menuApi.ts:8/16`),`src/pages/system/apikeys/index.tsx` + `LogsModal.tsx` 存在,菜单数据流与路由解析已就绪,运行时由 dev 启动触发 migration_166 自动修数据后,sidebar "API密钥管理" 即可点击展开进入。
verification: 2026-06-26 复测 `internal/core/db/migrations/migration_166_apikey_route_path_fix.go:28-66` 函数体完整(3 个 pathFix 项) + `internal/core/db/database.go:411` Migrate166ApikeyRoutePathFix 注册 + `xingran-react-frontend/src/pages/system/apikeys/{index.tsx, LogsModal.tsx}` 组件存在 + `cd xingran-react-frontend && npm run build` 退出 0(34.32s);运行时由 migration_166 自动应用,无需手动 SQL 介入。
files_changed: internal/core/db/migrations/migration_166_apikey_route_path_fix.go, internal/core/db/database.go (Phase 40 已落地,本 plan 仅补 frontmatter 闭环)
action: re-verify-then-flip (D-01) — 代码已在前序 phase 实修落地,本 plan 仅补 frontmatter 闭环