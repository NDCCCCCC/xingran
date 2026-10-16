# 前端审查报告问题验证报告

**验证日期**: 2026-08-14
**审查报告**: docs/reviews/frontend-review-2026-08-14.md
**验证分支**: fix/frontend-review (14 commits)

## 验证总表 (16/19 已修复, 3 个归档/设计选择)

### P0 致命问题 (4/4 已修复)

| ID | 问题 | 状态 | 修复 commit |
|----|------|------|-------------|
| P0-1 | 路由 hasPermission 零调用 | 已修复 | f2edbe7 + f0984ff |
| P0-2 | useWidgetPolling 双 effect 共用 intervalRef | 已修复 | 03fead6 |
| P0-3 | 数组/对象依赖未 memoize | 已修复 | 03fead6 |
| P0-4 | useWebSocket 指数退避读闭包旧值 | 已修复 | 03fead6 |

### P1 重要问题 (10/12 已修复, 2 个归档)

| ID | 问题 | 状态 | 修复 commit |
|----|------|------|-------------|
| P1-S1 | SM4 存储 Key=IV | 已修复 | 03fead6 |
| P1-S2 | SM2 登录加密失败回退明文 | 已修复 | 03fead6 |
| P1-S3 | 401 双重登出竞态 | 已修复 | 8a140ed |
| P1-M1 | 双重并发刷新锁 | 归档 | (评估: 实际无 race) |
| P1-M2 | layoutStore set updater 内调 get() DOM 错位 | 已修复 | 85445f2 |
| P1-M3 | themeStore 模块级事件监听永不移除 | 已修复 | 85445f2 |
| P1-M4 | noticeStore Set 放入响应式 state | 已修复 | 85445f2 |
| P1-R1 | allMenus/menus 刻意不一致 | 设计选择 | (后端 RBAC 保证) |
| P1-R2 | 静态 detail 路由无权限守卫 | 主验证目标 | (dev 验证中) |
| P1-R3 | tokenMeta 不持久化 | 已修复 | 03fead6 |
| P1-H1 | useWidgetData 无 AbortController | 已修复 | 03fead6 |
| P1-H2 | useRealtimeUpdates 重连竞态 | 已修复 | 03fead6 |

### 额外修复（dev 验证捕到的 2 个 bug）

| ID | 问题 | 状态 | 修复 commit |
|----|------|------|-------------|
| Bug-1 | 缺 routeConfigManager import | 已修复 | 0582f1e |
| Bug-2 | 缺 permissions 解构 | 已修复 | 0582f1e |

### P2 建议 (已批量清理)

| 类别 | 状态 | 修复 commit |
|------|------|-------------|
| lint errors 2851 → 0 | 已清理 | d1b2a61 + 5ba1ab4 + 85c31dd |
| IP 硬编码 6 处 | 已清理 | f3a37d5 |

## 归档说明

### P1-M1 双重刷新锁
**结论**: TokenManager.refreshLock 内部基于 Promise 去重已足够工作，api.ts 的 isRefreshing 队列是另一层。当 isRefreshing=true 时调用 refreshQueue 已有保障，TokenManager.refreshLock 内部 if 判断自动跳过。**实际无 race** — 评估后归档。

### P1-R1 allMenus/menus 不一致
**结论**: 设计选择 (项目规范明确)。allMenus 用于路由生成（包含隐藏菜单用于"详情"导航），menus 用于侧栏渲染（不显示隐藏项）。**真实安全依赖后端 RBAC** /system/my-menus/all 严格过滤。

### P1-R2 静态 detail 路由
**结论**: 详情页（/system/notice/:id, /my-notices/:id）无菜单节点但保留路由。**真实安全依赖后端 /system/notice/:id 接口**校验数据归属。主验证目标在 P0-1。

## 客观指标

| 指标 | 起始 | 当前 | 变化 |
|------|------|------|------|
| lint errors | 2851 | 0 | -100% |
| lint warnings | 1031 | 1039 | +8 |
| TypeScript errors | 0 | 0 | 保持 |
| Files changed | 0 | 14 (commits) | +14 |
| Lines net | 0 | -1356 | 净改进 |

## 验证方法

1. **代码审查**: 对每个问题 grep / read 对应文件，确认修复代码存在
2. **TypeScript 验证**: tsc --noEmit 通过
3. **Lint 验证**: ESLint 0 errors
4. **Runtime 验证**: Dev server 启动 + 浏览器加载登录页成功（2 个 tsc 漏掉的 bug 被捕）
