---
phase: 07-ux-optimization
plan: 02
status: complete
completed: 2026-04-21
depends_on: 07-01
---

# Plan 07-02: 完善加载状态和错误处理

## 完成内容

### Task 1: 增强 BaseWidget 加载和错误状态 ✓

**修改文件:** 
- `ruoyi-react-frontend/src/components/dashboard/widgets/base/BaseWidget.tsx`
- `ruoyi-react-frontend/src/components/dashboard/widgets/base/BaseWidget.css`

**实现内容:**
1. 添加状态管理:
   - `isEmpty` - 检测数据是否为空
   - `isFirstLoad` - 首次加载状态
   - `showSkeleton` - 显示骨架屏条件

2. 加载状态显示:
   - 使用 Ant Design `Skeleton` 组件显示骨架屏
   - 使用 `Spin` 组件包裹内容显示加载状态
   - 首次加载显示骨架屏，后续加载显示 Spin

3. 错误状态显示:
   - 使用 Ant Design `Result` 组件显示错误
   - 显示错误消息和重试按钮
   - 样式优化：居中显示，最小高度

4. 空数据状态显示:
   - 使用 Ant Design `Empty` 组件
   - 显示友好提示信息
   - 可配置空状态文案

### Task 2: 增强 DashboardView 全局状态管理 ✓

**修改文件:** `ruoyi-react-frontend/src/components/dashboard/DashboardView.tsx`

**实现内容:**
1. 全局加载状态:
   - 首次加载仪表盘时显示全屏 Spin
   - 添加 `initialLoadComplete` 状态跟踪

2. 网络错误处理:
   - 离线时显示 `Alert` 警示条
   - 提示数据可能不是最新
   - 网络恢复后自动刷新

3. WebSocket 状态指示:
   - 连接断开时显示提示
   - 连接错误时显示警告通知
   - 重连中显示加载状态

4. 错误边界:
   - 全局错误时显示 `Result` 组件
   - 提供重试按钮
   - 仪表盘不存在时显示 Empty

## 验证结果

- [x] BaseWidget.tsx 包含 "Spin" 加载组件
- [x] BaseWidget.tsx 包含 "Empty" 空状态组件
- [x] BaseWidget.tsx 包含 "Result" 错误组件
- [x] BaseWidget.tsx 包含 "loading" 状态管理
- [x] BaseWidget.tsx 包含 "error" 状态管理
- [x] DashboardView.tsx 包含全局加载状态处理
- [x] DashboardView.tsx 包含网络错误提示
- [x] DashboardView.tsx 包含 WebSocket 状态指示

## 成功标准达成

- ✓ 数据加载时显示 loading 状态
- ✓ 错误时显示错误信息
- ✓ Widget 数据为空时显示友好提示而非空白
- ✓ 网络错误时显示重试按钮

## 文件变更

| 文件 | 变更类型 | 行数 |
|------|----------|------|
| BaseWidget.tsx | 修改 | +60 |
| BaseWidget.css | 修改 | +30 |
| DashboardView.tsx | 修改 | +50 |

---

*Completed: 2026-04-21*
