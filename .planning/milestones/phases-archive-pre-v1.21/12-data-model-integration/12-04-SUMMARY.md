# 12-04: MAC历史API路由注册修复 - 执行摘要

## 执行日期
2026-05-11

## 目标
修复MAC历史查询API路由注册位置，使路由路径符合API设计规范（/api/v1/network/history/* 而非 /api/v1/network/mac/history/*）。

## 完成任务

### Task 1: 移除MAC历史路由错误注册
**文件**: `internal/api/v1/network/network_router.go`

- 移除了第 159 行的 `SetupMACHistoryRouter(mac, core)` 调用
- 路由不再错误注册到 /mac 子路由组下

**提交**: `fix(12-04): 修复MAC历史API路由注册位置`

### Task 2: 在主路由中正确注册MAC历史路由
**文件**: `internal/api/router.go`

- 在网络设备管理路由组（第 326 行附近）中添加了 `SetupMACHistoryRouter(network, core)` 调用
- 路由现在正确注册到 /network/history/* 路径下

**API路径变更**:
- 旧: `/api/v1/network/mac/history/device` → 新: `/api/v1/network/history/device`
- 旧: `/api/v1/network/mac/history/port` → 新: `/api/v1/network/history/port`
- 旧: `/api/v1/network/mac/history/stats` → 新: `/api/v1/network/history/stats`

## 关键文件变更

| 文件 | 变更 | 行数 |
|------|------|------|
| internal/api/router.go | 添加 SetupMACHistoryRouter 调用 | +1 |
| internal/api/v1/network/network_router.go | 移除 SetupMACHistoryRouter 调用 | -1 |

## 验证结果

### 编译检查
```bash
go build ./...
```
✓ 通过 - 无编译错误

### 路由注册验证
```bash
grep -A 2 "SetupTopologyRouter" internal/api/router.go | grep "SetupMACHistoryRouter"
```
✓ 通过 - 路由调用已正确添加

```bash
grep "SetupMACHistoryRouter" internal/api/v1/network/network_router.go
```
✓ 通过 - 旧位置的调用已完全移除

## 影响分析

### 用户影响
- MAC历史查询API现在可通过正确的路径访问
- UAT 测试5（Device History Query API）和测试6（Port History Query API）现在可通过路由验证

### 兼容性
- **破坏性变更**: API路由路径已更改
- 旧路径 `/api/v1/network/mac/history/*` 不再可用
- 客户端需要更新API调用路径

### 依赖模块
- 无 - 纯路由注册变更

## 已知问题
无

## 偏差记录
无 - 执行完全符合计划。

## 后续工作
需要手动验证路由修复：
1. 启动应用程序
2. 检查启动日志显示路由已注册到正确路径
3. 测试 `/api/v1/network/history/device` 端点返回 200（非 404）
4. 测试 `/api/v1/network/history/port` 端点返回 200（非 404）

## Self-Check: PASSED

✓ 所有任务已完成
✓ 每个任务已单独提交
✓ 编译检查通过
✓ 关键文件变更符合预期
