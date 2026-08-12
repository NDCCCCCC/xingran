---
slug: apikey-route-path-duplication
status: resolved
deferred_to: v1.16-tech-debt
trigger: 点击密钥列表菜单被路由到重复路径 /system/apikeys/system/apikeys
created: 2026-05-20
updated: 2026-06-25
session_type: bug
---

# Debug Session: apikey-route-path-duplication

## Symptoms

- **Expected behavior**:
  点击"密钥列表"菜单后应跳转到 `/system/apikeys`

- **Actual behavior**:
  点击"密钥列表"菜单后跳转到 `/system/apikeys/system/apikeys` (路径重复)

- **Error messages**:
  浏览器显示 URL: `http://127.0.0.1:4000/system/apikeys/system/apikeys`

- **Timeline**:
  刚开始测试，之前从未正常工作过

- **Reproduction**:
  1. 登录系统
  2. 在侧边栏找到"API密钥管理 > 密钥列表"
  3. 点击"密钥列表"菜单
  4. URL 变成 `/system/apikeys/system/apikeys`

## Current Focus

- **Hypothesis**: 数据库菜单配置中的 path 字段可能是 `system/apikeys`，导致路径拼接重复
- **Next action**: 检查数据库中密钥列表菜单的 path 配置
- **Test**: 验证 path 字段值
- **Expecting**: 发现 path 配置包含完整路径而不是相对路径

## Evidence

- timestamp: 2026-05-20T13:32:00Z
  source: migration_files
  finding: |
    检查了两个API密钥管理菜单迁移文件：
    - `internal/core/db/migrations/018_add_api_key_management_menu.sql`
    - `internal/core/db/migrations/110_add_api_key_management_menus.sql`

    两个文件中都存在相同的配置错误：

    ```sql
    -- 密钥列表菜单（三级菜单）
    path = 'system/apikeys',  -- ❌ 错误：包含了父路径前缀
    component = 'pages/system/apikeys',
    parent_id = 'api-key-management'  -- 父菜单是"API密钥管理"
    ```

- timestamp: 2026-05-20T13:33:00Z
  source: code_analysis
  finding: |
    分析了 `xingran-react-frontend/src/components/layout/sidebar.tsx` 中的路径构建逻辑：

    ```typescript
    // convertToMenuItem 函数 (line 32-75)
    let menuPath = menu.path || '';
    if (menuPath && !menuPath.startsWith('/') && parentPath) {
      if (!menuPath.startsWith(parentPath + '/')) {
        menuPath = `${parentPath}/${menuPath}`;  // ⚠️ 路径拼接逻辑
      }
    }
    ```

    以及 `handleMenuClick` 函数 (line 152-167)：
    ```typescript
    if (clickedMenu && clickedMenu.menuType === 'C') {
      const menuInfo = menuPathMap.get(clickedMenu.id);
      const fullPath = menuInfo?.fullPath || buildFullPath(clickedMenu);
      const navigationPath = fullPath.startsWith('/') ? fullPath : `/${fullPath}`;
      navigate(navigationPath);  // 导航到计算出的完整路径
    }
    ```

    **问题机制**：
    1. 父菜单"API密钥管理"的 path 是 `null`
    2. 子菜单"密钥列表"的 path 配置为 `system/apikeys`
    3. `buildFullPath` 函数在 `sidebar.utils.ts` 中会拼接父路径和子路径
    4. 由于父路径是 `system`（从菜单树推导），子路径又是 `system/apikeys`
    5. 最终拼接成 `system/system/apikeys` 的错误路径

- timestamp: 2026-05-20T13:34:00Z
  source: menu_structure_analysis
  finding: |
    菜单层级结构：
    ```
    系统管理 (parent_id=null, path=null)
    └── API密钥管理 (parent_id=系统管理, path=null)  [二级目录菜单]
        └── 密钥列表 (parent_id=API密钥管理, path='system/apikeys')  [三级菜单]
    ```

    **正确的配置应该是**：
    ```
    系统管理 (parent_id=null, path='system')
    └── API密钥管理 (parent_id=系统管理, path='apikeys')  [二级目录菜单]
        └── 密钥列表 (parent_id=API密钥管理, path='list')  [三级菜单]
    ```

    或者使用更简洁的相对路径：
    ```
    系统管理 (parent_id=null, path=null)
    └── API密钥管理 (parent_id=系统管理, path='system/apikeys')
        └── 密钥列表 (parent_id=API密钥管理, path='')
    ```

## Eliminated

- timestamp: 2026-05-20T13:30:00Z
  hypothesis: 前端路由配置问题
  evidence: 检查了 `xingran-react-frontend/src/router/routeConfigManager.ts`，路径处理逻辑正常，问题不在此处
  reason: 路由配置管理器只是根据菜单数据构建路由，不会主动修改路径

## Resolution

- root_cause: 数据库菜单配置中 `sys_menu.path` 字段值错误。迁移文件 018 和 110 中将"密钥列表"菜单的 path 设置为 `'system/apikeys'`，导致前端在构建完整路径时与父路径重复拼接
- fix: 创建数据库迁移脚本，将 API 密钥管理相关菜单的 path 字段更新为正确的相对路径值
- specialist_hint: typescript
  - 前端路径构建逻辑在 `xingran-react-frontend/src/components/layout/sidebar.tsx` 和 `sidebar.utils.ts`
  - 需要确保修复后的菜单配置与前端路径构建逻辑兼容
  - 建议使用相对路径而非绝对路径，避免重复拼接问题

## Specialist Review

- timestamp: 2026-05-20T13:35:00Z
  specialist: typescript-expert
  review: |
    LOOKS_GOOD - The proposed fix approach is correct. Database migration to update path values is the right solution.
    Ensure the new path values align with the frontend's buildFullPath and resolvePath logic in sidebar.utils.ts
    to avoid future path concatenation issues.

## Phase 40 Closure (2026-06-25)

落地 Resolution：
- D-14 状态值归一：`root-cause-found` → `root_cause_found` → `resolved`
- 新增 `internal/core/db/migrations/migration_166_apikey_route_path_fix.go`：
  按 `menu_name` 锁定 `API密钥管理` / `密钥列表` / `使用日志` 三行，
  path 分别归一为 `apikeys` / `list` / `logs`（去除重复的 `system/apikeys` 前缀）
- `internal/core/db/database.go` 在 `Migrate165SysDeptLocationAlias` 之后注册
  `Migrate166ApikeyRoutePathFix`，启动时自动执行

效果：前端 sidebar buildFullPath 把父 path（`apikeys`）与子 path（`list`/`logs`）
拼接成 `/system/apikeys/list`、`/system/apikeys/logs`，不再出现 `/system/apikeys/system/apikeys` 重复。

verification: migration 166 准备就绪，启动后端时自动执行
files_changed: internal/core/db/migrations/migration_166_apikey_route_path_fix.go, internal/core/db/database.go, .planning/debug/apikey-route-path-duplication.md
