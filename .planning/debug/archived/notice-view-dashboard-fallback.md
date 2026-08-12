---
slug: notice-view-dashboard-fallback
trigger: "调查并修复 XingRan-Next 项目中'通知公告'和'我的通知'两个页面的'查看'按钮跳转到仪表盘的 bug"
status: awaiting_human_verify
created: 2026-07-04T00:00:00Z
updated: 2026-07-04T00:00:00Z
---

## Current Focus

hypothesis: 已修复
test: ✅ tsc --noEmit exit 0
expecting: ✅ 通过
next_action: 等待用户在前端手动验证两个"查看"按钮

## Symptoms

expected: 点击"通知公告"或"我的通知"列表中某条通知的"查看"按钮 → 进入通知详情页
actual: 跳转到仪表盘 /dashboard
errors: 无 JS 报错(被 catch-all Navigate 接住)
reproduction:
  - 进入 通知公告 管理端或 我的通知 列表
  - 点击任意条目的"查看"按钮
  - URL 变为 /system/notice/{id} 或 /my-notices/{id}
  - 立刻被路由表 catch-all 重定向到 /dashboard
started: 用户报告最近修改后出现(原菜单 path 从 my-notices → user/my-notices,migration 056)

## Eliminated

- hypothesis: 后端 sys_menu 没 seed / 系统未授权
  evidence: 后端只负责菜单数据,不存路由;前端 DynamicRoutes 完全不读 menu.path 拼接 /:id 子路由
  timestamp: 2026-07-04

- hypothesis: routeGenerator 没渲染过子菜单
  evidence: routeGenerator.ts:39-71 只对 menuType==='C' 生成路由,且 path 是 menu.path 原始值,绝不会有 :id 段
  timestamp: 2026-07-04

- hypothesis: detail.tsx 文件不存在
  evidence: 两个 detail.tsx 都存在(migration 042/056 时期就在);pageTitles.ts:34 已经有 /my-notices/[uuid] 正则匹配说明设计意图就是 /my-notices/:id
  timestamp: 2026-07-04

## Evidence

- timestamp: 2026-07-04
  checked: xingran-react-frontend/src/router/componentLoader.tsx:36
  found: `import.meta.glob(['/src/pages/**/index.tsx', '!**/login/**'])` 只匹配 index.tsx,排除 detail.tsx 等子页
  implication: 即使显式注册 /system/notice/:id 路由,createLazyComponent('pages/system/notice/detail') 也会失败 (moduleLoader undefined),要扩展 glob 同时支持 detail.tsx

- timestamp: 2026-07-04
  checked: xingran-react-frontend/src/router/DynamicRoutes.tsx:213-216
  found: 已认证路由只有 menu 动态生成的 + catch-all `path="*" element={<Navigate to="/dashboard" replace />}`
  implication: /my-notices/:id 和 /system/notice/:id 都不在此集合中 → 命中 catch-all

- timestamp: 2026-07-04
  checked: xingran-react-frontend/src/router/routeGenerator.ts:46-55
  found: 路由 path 是菜单 path 原值,没有能力塞 :id 动态段
  implication: 即便 sys_menu 补 seed 子菜单(系统通知管理),仍无 :id 路由;必须走静态 Route

- timestamp: 2026-07-04
  checked: internal/core/db/migrations/archive/056_fix_user_center_paths.sql:53-57
  found: UPDATE sys_menu SET path='user/my-notices' WHERE menu_name='我的通知' AND path='my-notices'
  implication: 菜单 path 改后,RouteGenerator.resolvePath 出来的实际路由是 user-center/user/my-notices(prefix parent);但代码 navigate 用 `/my-notices/${id}`(无双段)→ 完全不匹配
  这条说明 migrate 之后 唯一能访问列表的 URL 是 user-center/user/my-notices,而 navigate 的 /my-notices/${id} 永远是 broken 的

- timestamp: 2026-07-04
  checked: xingran-react-frontend/src/constants/pageTitles.ts:34
  found: DYNAMIC_ROUTE_PATTERNS 包含 `/^\/my-notices\/[a-f0-9-]+$/i`(用户中心路径是 /my-notices/:id 样式)
  implication: 即使现列表登记是 user-center/user/my-notices,设计意图/品牌路径就是 /my-notices/:id;同理 /system/notice/:id 是通知公告详情路径
  静态注册这俩路由 + 不动 navigate 路径是符合"原设计意图"的最小修复

- timestamp: 2026-07-04
  checked: xingran-react-frontend/src/pages/my-notices/detail.tsx + system/notice/detail.tsx + InnovativeLayout.tsx:96
  found:
    - detail 文件存在
    - InnovativeLayout 的快捷入口 path = '/user/my-notices'(列表);详情是 /my-notices/:id
  implication: 用 /my-notices/:id 作为详情路由 + 列表 user-center/user/my-notices 是当前的设计

## Resolution

root_cause:
  路由表层级缺失 —— 列表页有 dynamic route(sys_menu seed 决定),但 detail 子页无 Route。
  - DynamicRoutes.tsx 没有静态 `/system/notice/:id` / `/my-notices/:id` 注册
  - componentLoader.tsx glob 只匹配 `**/index.tsx`,不匹配 `detail.tsx`(即使静态 import 也行不通——createLazyComponent 走 glob 查表失败 → fallback 错误页)
  - 当前命中 catch-all `<Route path="*" element={<Navigate to="/dashboard" replace />} />`
  - pages/system/notice/index.tsx:451 + pages/my-notices/index.tsx:105 + components/NotificationBell.tsx:111 三个跳点全部落入 catch-all

fix:
  A. DynamicRoutes.tsx 顶部静态 import AdminNoticeDetailPage + MyNoticeDetailPage
  B. DynamicRoutes.tsx 在 Layout Outlet 下加 2 个静态 Route(/system/notice/:id, /my-notices/:id)
  C. componentLoader.tsx glob 扩展为 '/src/pages/**/{index,detail}.tsx'
  D. routeGenerator.ts getComponentPattern 同步扩展(只动 1 行,保持一致性)

  原则:
  - 列表/详情路径对齐 pageTitles.ts 设计意图与现有 navigate() 调用
  - 不动 sys_menu seed(题目禁止)
  - 不动 navigate() 路径(扩散到 3 个文件 + InnovativeLayout 是反方向错配)
  - 静态 import detail(不走 glob)——避免 glob 模式复杂化

verification:
  - cd xingran-react-frontend && npx tsc --noEmit -p tsconfig.json 退出码 0 ✅
  - 用户手动验证:
    1. 通知公告列表 → 任意一条 → 查看 → 进入 /system/notice/{id} 显示详情(不再 redirect dashboard)
    2. 我的通知列表 → 任意一条 → 查看 → 进入 /my-notices/{id} 显示详情
    3. 顶部铃铛 NotificationBell → 任意一条 → 同样进入我的通知详情
    4. 浏览器返回按钮回到列表

files_changed:
  - xingran-react-frontend/src/router/DynamicRoutes.tsx (+6 lines: 2 imports + 2 Routes + 2 注释)
  - xingran-react-frontend/src/router/componentLoader.tsx (+3 -1 lines: 注释 + glob)
  - xingran-react-frontend/src/router/routeGenerator.ts (+1 -1 lines: getComponentPattern)
  - 净增 9 行,变动 3 文件,全部在 xingran-react-frontend/src/router/ 下


## Current Focus

hypothesis: DynamicRoutes 路由表不包含 `/my-notices/:id` 和 `/system/notice/:id` 这两个子路由,所以 catch-all redirect to /dashboard
test: 写最小修复:在 DynamicRoutes.tsx 注册两个静态 Route,匹配 detail.tsx
expecting: navigate 跳转不再落 catch-all,详情页正常渲染
next_action: 应用修复 + 跑 tsc --noEmit

## Symptoms

expected: 点击"通知公告"或"我的通知"列表中某条通知的"查看"按钮 → 进入通知详情页
actual: 跳转到仪表盘 /dashboard
errors: 无 JS 报错(被 catch-all Navigate 接住)
reproduction:
  - 进入 通知公告 管理端或 我的通知 列表
  - 点击任意条目的"查看"按钮
  - URL 变为 /system/notice/{id} 或 /my-notices/{id}
  - 立刻被路由表 catch-all 重定向到 /dashboard
started: 用户报告最近修改后出现(原菜单 path 从 my-notices → user/my-notices,migration 056)

## Eliminated

- hypothesis: 后端 sys_menu 没 seed / 系统未授权
  evidence: 后端只负责菜单数据,不存路由;前端 DynamicRoutes 完全不读 menu.path 拼接 /:id 子路由
  timestamp: 2026-07-04

- hypothesis: routeGenerator 没渲染过子菜单
  evidence: routeGenerator.ts:39-71 只对 menuType==='C' 生成路由,且 path 是 menu.path 原始值,绝不会有 :id 段
  timestamp: 2026-07-04

- hypothesis: detail.tsx 文件不存在
  evidence: 两个 detail.tsx 都存在(migration 042/056 时期就在);pageTitles.ts:34 已经有 /my-notices/[uuid] 正则匹配说明设计意图就是 /my-notices/:id
  timestamp: 2026-07-04

## Evidence

- timestamp: 2026-07-04
  checked: xingran-react-frontend/src/router/componentLoader.tsx:36
  found: `import.meta.glob(['/src/pages/**/index.tsx', '!**/login/**'])` 只匹配 index.tsx,排除 detail.tsx 等子页
  implication: 即使显式注册 /system/notice/:id 路由,createLazyComponent('pages/system/notice/detail') 也会失败 (moduleLoader undefined),要扩展 glob 同时支持 detail.tsx

- timestamp: 2026-07-04
  checked: xingran-react-frontend/src/router/DynamicRoutes.tsx:213-216
  found: 已认证路由只有 menu 动态生成的 + catch-all `path="*" element={<Navigate to="/dashboard" replace />}`
  implication: /my-notices/:id 和 /system/notice/:id 都不在此集合中 → 命中 catch-all

- timestamp: 2026-07-04
  checked: xingran-react-frontend/src/router/routeGenerator.ts:46-55
  found: 路由 path 是菜单 path 原值,没有能力塞 :id 动态段
  implication: 即便 sys_menu 补 seed 子菜单(系统通知管理),仍无 :id 路由;必须走静态 Route

- timestamp: 2026-07-04
  checked: internal/core/db/migrations/archive/056_fix_user_center_paths.sql:53-57
  found: UPDATE sys_menu SET path='user/my-notices' WHERE menu_name='我的通知' AND path='my-notices'
  implication: 菜单 path 改后,RouteGenerator.resolvePath 出来的实际路由是 user-center/user/my-notices(prefix parent);但代码 navigate 用 `/my-notices/${id}`(无双段)→ 完全不匹配
  这条说明 migrate 之后 唯一能访问列表的 URL 是 user-center/user/my-notices,而 navigate 的 /my-notices/${id} 永远是 broken 的

- timestamp: 2026-07-04
  checked: xingran-react-frontend/src/constants/pageTitles.ts:34
  found: DYNAMIC_ROUTE_PATTERNS 包含 `/^\/my-notices\/[a-f0-9-]+$/i`(用户中心路径是 /my-notices/:id 样式)
  implication: 即使现列表登记是 user-center/user/my-notices,设计意图/品牌路径就是 /my-notices/:id;同理 /system/notice/:id 是通知公告详情路径
  静态注册这俩路由 + 不动 navigate 路径是符合"原设计意图"的最小修复

- timestamp: 2026-07-04
  checked: xingran-react-frontend/src/pages/my-notices/detail.tsx + system/notice/detail.tsx + InnovativeLayout.tsx:96
  found:
    - detail 文件存在
    - InnovativeLayout 的快捷入口 path = '/user/my-notices'(列表);详情是 /my-notices/:id
  implication: 用 /my-notices/:id 作为详情路由 + 列表 /user-center/user/my-notices(虽然丑,但与 sys_menu seed 一致) 是当前的设计

## Resolution

root_cause:
  路由表层级缺失 —— 列表页有 dynamic route(sys_menu seed 决定),但 detail 子页无 Route。
  - DynamicRoutes.tsx 没有静态 `/system/notice/:id` / `/my-notices/:id` 注册
  - componentLoader.tsx glob 只匹配 `**/index.tsx`,不匹配 `detail.tsx`(即使静态 import 也行不通——createLazyComponent 走 glob 查表失败 → fallback 错误页)
  - 当前命中 catch-all `<Route path="*" element={<Navigate to="/dashboard" replace />} />`
  - pages/system/notice/index.tsx:451 + pages/my-notices/index.tsx:105 + components/NotificationBell.tsx:111 三个跳点全部落入 catch-all

fix:
  A. 在 DynamicRoutes.tsx `<Layout><Outlet/></Layout>` 内追加 2 个静态 `<Route>`(在 menu 动态路由之后):
       <Route path="system/notice/:id" element={<NoticeDetail />} />
       <Route path="my-notices/:id" element={<MyNoticeDetail />} />
  B. 让 componentLoader 也能加载 detail.tsx:扩展 glob `'/src/pages/**/index.tsx'` → `'/src/pages/**/{index,detail}.tsx'` 之类
  C. 在 detail 组件中使用 useParams from react-router-dom 取 id

  题目要求"不要扩展到不相关模块",所以只动 DynamicRoutes + componentLoader 两个文件。
  原则:
  - 列表/详情路径对齐 pageTitles.ts 设计意图与现有 navigate() 调用
  - 不动 sys_menu seed(题目禁止)
  - 不动 navigate() 路径(代码侧统一传 /my-notices/:id 与 /system/notice/:id,改它会扩散到 3 个文件)

verification:
  - cd xingran-react-frontend && npx tsc --noEmit -p tsconfig.json 退出 0
  - 用户手动:点查看 → 进入详情页,不再 redirect

files_changed:
  - xingran-react-frontend/src/router/DynamicRoutes.tsx (新增 2 个静态 Route)
  - xingran-react-frontend/src/router/componentLoader.tsx (glob 扩展)
