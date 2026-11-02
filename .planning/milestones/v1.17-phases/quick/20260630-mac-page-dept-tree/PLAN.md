---
status: in-progress
created: 2026-06-30
updated: 2026-06-30
goal: mac/index.tsx 加 DeptSidebar + 联动 deviceId 筛选 + URL 同步 + 移动端折叠
---

# Mac 地址页面 — 左侧部门树组件集成

## Context

用户报告:
1. mac 地址页面无法按"接口"列排序 — 已修复(在 `sorterMetas` + `interfaceName` 列加 sorter)
2. 页面左侧缺少部门树组件 — 本次 PLAN 范围

参考模板: `src/pages/operations/workstations/index.tsx`(已用 DeptSidebar)

约束(用户明确):
- 不改后端
- 复用现有 useTableManager + createSorterMeta 模式
- TypeScript strict 通过
- 接口排序 + 部门树 一起 commit

## 范围

| 改动 | 状态 |
|---|---|
| 接口排序(sorterMetas + interfaceName 列) | ✅ 已完成(未 commit) |
| Layout Row+Col + DeptSidebar | ⏳ 本 PLAN |
| DeptSidebar → deviceIds → 联动筛选 | ⏳ |
| URL 参数 `?deptId=xxx` 同步 | ⏳ |
| 移动端折叠 dept tree 进 Drawer | ⏳ |

## 方案

### 1. Layout 重构(Row + Col)
- 当前: 单 column 布局(`统计卡片 → 搜索表单 → 表格`)
- 改后: `Row { Col span={6,18} }` — 左侧 DeptSidebar(span=6),右侧原内容(span=18)
- 移动端 breakpoint 触发 Drawer 模式

### 2. 联动筛选(方案 C — 前端间接过滤)
- 选中部门 → 查 `/network/devices/list?deptId=xxx` 拿 deviceId 列表
- 把 deviceId 列表写进 search form `deviceIds`(后端支持多 deviceId 过滤)
- 如果后端 `/network/mac/list` 不支持 deviceIds 数组 → 退化:多次串行请求 / 提示后端不支持
- 如方案 C 不可行,降级到方案 A(只展示部门树,不联动)— 用户后续手动选 deviceId 筛选

### 3. URL 同步
- 选中部门 → `useSearchParams.set("deptId", id)`
- 加载时读 `deptId` 参数,自动选中并触发联动
- 用于刷新/分享 URL 保留状态

### 4. 移动端折叠
- `Grid.useBreakpoint()` 检测 xs
- 移动端: 部门树塞进 Drawer,通过"选择部门"按钮触发
- 桌面端: DeptSidebar 常驻左侧

## 验证

| 检查 | 命令 |
|---|---|
| TypeScript | `cd xingran-react-frontend && npx tsc --noEmit -p tsconfig.json` |
| ESLint | `cd xingran-react-frontend && npx eslint src/pages/network/mac/index.tsx` |
| 手动 | dev 启动 + 浏览器验证侧栏渲染 + 点击部门触发筛选 + URL `?deptId=` 同步 |

## 不在本范围

- 后端 `/network/mac/list` 新增 deptId 字段
- 后端新增 `GET /system/dept/devices` 专用端点
- Workstation 部门树外的其他模块集成