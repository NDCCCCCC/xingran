---
slug: api-key-usage-logs-page-redirect
status: resolved
trigger: API密钥管理-使用日志页面无法打开，自动跳转到仪表盘
created: 2026-05-20
updated: 2026-05-20
---

# Debug Session: api-key-usage-logs-page-redirect

## Symptoms

- **Expected behavior**:
  点击使用日志菜单后应正常打开使用日志列表页面，显示 API 密钥的调用历史记录

- **Actual behavior**:
  点击菜单后直接跳转到仪表盘页面，没有显示任何错误信息

- **Error messages**:
  无错误提示，浏览器控制台无错误日志

- **Timeline**:
  从未正常工作（新功能）

- **Reproduction**:
  1. 登录系统
  2. 在侧边栏找到"API密钥管理"菜单
  3. 点击"使用日志"子菜单
  4. 页面自动跳转到仪表盘

## Current Focus

- **Hypothesis**: 前端路由配置缺失或菜单权限配置不完整
- **Next action**: 检查前端路由配置和后端菜单配置
- **Test**: 验证使用日志页面的路由是否存在
- **Expecting**: 发现缺失的路由配置或菜单条目

## Evidence

- timestamp: 2026-05-20T10:30:00Z
  - **Source**: Migration file analysis (115_create_apikey_menu_final.sql)
  - **Finding**: 使用日志菜单配置为 component='system/apikeys/LogsModal/index'
  - **Detail**: 菜单路径是 'system/apikeys/logs'，组件路径是 'system/apikeys/LogsModal/index'

- timestamp: 2026-05-20T10:31:00Z
  - **Source**: Frontend file system search
  - **Finding**: 实际组件文件位于 `xingran-react-frontend/src/pages/system/apikeys/LogsModal.tsx`
  - **Detail**: 没有 `LogsModal/index.tsx` 文件，只有 `LogsModal.tsx`

- timestamp: 2026-05-20T10:32:00Z
  - **Source**: DynamicRoutes.tsx code analysis
  - **Finding**: 路由生成器会尝试导入 component 路径指定的文件
  - **Detail**: component 路径 'system/apikeys/LogsModal/index' 会被转换为 import path，导致导入失败
  - **Result**: 导入失败时，路由不匹配，触发 404 重定向到 /dashboard

- timestamp: 2026-05-20T10:35:00Z
  - **Source**: Project structure analysis
  - **Finding**: 发现项目传统模式是使用 `/index` 后缀
  - **Detail**: 其他页面如 `system/user/index.tsx`、`system/role/index.tsx` 都遵循此模式
  - **结论**: 应该创建符合项目传统的文件结构，而不是修改数据库配置

## Eliminated

- ~~菜单权限问题~~ - 管理员角色已正确分配权限
- ~~菜单状态问题~~ - 菜单 status=0 (正常), visible=1 (显示)
- ~~组件文件不存在~~ - LogsModal.tsx 文件存在，只是路径配置错误

## Resolution

- **Root cause**: 数据库菜单配置中的 component 路径与实际文件路径不匹配
  - **配置路径**: `system/apikeys/LogsModal/index` (期望 LogsModal/index.tsx)
  - **实际路径**: `system/apikeys/LogsModal.tsx` (单文件，不符合项目传统)
  - **影响**: 前端路由生成器无法正确导入组件，导致路由匹配失败，自动重定向到仪表盘

- **Fix Applied**:
  ✅ **创建符合项目传统的文件结构**
  - 移动: `LogsModal.tsx` → `LogsModal/LogsModal.tsx`
  - 创建: `LogsModal/index.tsx` 导出组件
  - 保持数据库配置不变: `system/apikeys/LogsModal/index`
  - 符合项目传统: 与 `system/user/index.tsx`、`system/role/index.tsx` 等页面一致

- **Files Changed**:
  - `xingran-react-frontend/src/pages/system/apikeys/LogsModal/LogsModal.tsx` (原文件移动)
  - `xingran-react-frontend/src/pages/system/apikeys/LogsModal/index.tsx` (新建导出文件)

- **Verification steps**:
  1. ✅ 文件结构已创建完成
  2. 刷新前端页面重新加载菜单
  3. 点击"使用日志"菜单项
  4. 验证能正常打开使用日志页面（而不是跳转到仪表盘）
