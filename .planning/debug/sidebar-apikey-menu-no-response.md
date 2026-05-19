---
slug: sidebar-apikey-menu-no-response
status: investigating
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