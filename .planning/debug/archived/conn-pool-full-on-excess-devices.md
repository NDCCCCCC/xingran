---
slug: conn-pool-full-on-excess-devices
status: resolved
goal: find_and_fix
trigger: |
  WARN[2026-07-08 12:22:17] [端口采集] CX-WH-RUITONG-25F-SWL2-HW-S5735-1 (10.62.25.252): 获取连接失败: 连接池已满: 当前=20, 最大=20
  用户问: 确认设备 > 20 台、连接池满后的处理逻辑; 并质疑"应每命令执行完自动清理连接让位新设备"。
created: 2026-07-08
updated: 2026-07-08
---

# Debug: 连接池满 (设备数 > 池容量)

## Symptoms
- 端口采集 WARN: `获取连接失败: 连接池已满: 当前=20, 最大=20`, 该设备本轮采集跳过。
- 前端: 受影响设备 collected_at 停在几天前 (间歇性, 池满→失败→等 5min idle 清理→短暂恢复→再满)。

## 处理逻辑 (已确认, 代码确定)
`GetConnection` (`connection_pool.go:265-273`): 池满 (currentCount >= maxConnections) **立即快速失败**, 不阻塞/不等待/不重试。`collectDevicePort` 捕获后该设备本轮跳过 (`collection.go:113-117`)。

## 根因机制
- 池按 deviceID 缓存 (`connection_pool.go:238,311`), 连接采完 `ReleaseRef` 只 refCount--, **连接留池不关**, idle 满 5min 才被 cleanup (每 1min 扫) 清理。
- `MaxConnections=20` 硬编码 (`core.go:260`), `maxConcurrent=10` (`collection.go:32`)。
- 池槽被 idle 连接长期占用 → 设备数 > 20 时新设备 GetConnection 必然池满失败。

## 三档定性
1. **刻意过载保护 (非 bug)**: core.go:260 注释 "降低并发连接数, 减少 scrapligo panic"。
2. **容量配置缺陷 (已知 follow-up)**: 20 对 >20 台不够, research/PITFALLS Phase 51 规划移 sys_config 未实施。
3. **TOCTOU 竞态 (已知 deferred bug)**: `connection_pool.go:266-273` count check (RLock) 与 create/store 间释放锁, 并发下实际连接数可短暂超 max (曾观察 24/20)。PROJECT.md:197 / 49-02-SUMMARY:165 标记未修。

## SSH 连接硬约束 (澄清用户质疑)
SSH/Telnet 连接**绑定 deviceID, 不可跨设备共享** (IP+凭据不同)。"清理 A 连接让位 B" = 关 A 连接 + B 新建自己连接, 非复用。

## 选定方案 (用户确认)
TOCTOU 修复 + LRU 退让 + 配置化 (默认 50)。

### 1. TOCTOU 修复
`DeviceConnectionPool` 新增 `connecting map[string]struct{}` 占位字段。GetConnection 新建路径: 全程 `poolLock.Lock` 保护 `count = len(connections) + len(connecting)` check; 占位 `connecting[deviceID]` 后释放锁; **createConnection 锁外执行** (不持 poolLock/deviceLock, 不死锁); 成功后锁内写入 connections + 清占位。count 含 connecting → 原子, 无 TOCTOU。

### 2. LRU 退让
池满时 (count >= max) 锁内遍历 connections 找 `IsIdle()` 且 `lastUsed` 最老的连接, Close + delete 腾位, 继续建新。全部活跃 (无 idle) 才返回错误。因 maxConcurrent=10 保证同时活跃 ≤10 < max, 理论永不真正拒绝。兼顾复用 (idle 仍留池等用) + 不满 (池满自愈)。

### 3. 配置化
- migration seed: `sys.network.connection_pool.max_connections` (默认 50), `sys.network.connection_pool.max_idle_seconds` (默认 300)。
- core.go:258-261 去硬编码, 启动时用 c.GetDB() 查 sys_config (带默认兜底)。
- 前端: 复用现有"参数设置"页 (`pages/system/config/index.tsx`), 无需新页面。修改后需重启后端生效。

## Fix Plan
1. connection_pool.go: 加 connecting 字段 + 重构 GetConnection。
2. connection_pool_test.go: 加 LRU + 不超 max 回归测试。
3. migration: seed 2 个 sys_config key。
4. core.go: 配置从 sys_config 读。
5. go build + go test。

## Evidence
- 2026-07-08: GetConnection (220-318) + createConnection (335-423, 不持 poolLock) + removeConnection (427, 持 poolLock, 注释要求先释放 deviceLock) + cleanupIdleConnections (473-500)。
- 2026-07-08: core.go:258-261 硬编码 MaxConnections=20。
- 2026-07-08: 前端 pages/system/config/index.tsx 参数管理页已存在, sys_config 有 GetByKey API。
- 2026-07-08: connection_pool_test.go:171 已锁定 2026-07-06 同款 24/20 bug (refCount 泄漏, F-14 修复)。

## Resolution
- root_cause: 池按 deviceID 缓存 + 长 idle (5min) + 容量 20 硬编码 + 设备数>20 + TOCTOU (count check 与 create 间释放锁) 放大, 致池满时 GetConnection 快速失败、间歇性端口采集跳过。
- fix: (1) connection_pool.go 加 connecting 占位 map 防 TOCTOU + GetConnection 池满时 LRU 淘汰最老 idle 连接腾位 (oldestIdleConnectionLocked) + createConnection 锁外执行不死锁; (2) migration 203 seed sys_config (network.connection_pool.max_connections 默认50, max_idle_seconds 默认300, ConfigType=N web 可编辑); (3) core.go loadConnectionPoolConfig 启动时从 sys_config 读配置替代硬编码。
- files_changed: internal/device/connection_pool.go, internal/device/connection_pool_test.go, internal/core/db/migrations/migration_203_connection_pool_sysconfig.go, internal/core/db/database.go, internal/core/core.go
- verification: go build ./... EXIT 0; go test ./internal/device/ PASS — 3 个新 LRU 测试 (PicksOldestIdle_SkipsActive / NoIdleReturnsNil / EmptyPool) 全 PASS。
- deploy_note: 需重新编译部署后端; sys_config 改 max_connections/max_idle_seconds 后重启生效 (连接池启动时读)。默认 20→50 已大幅缓解设备>20 池满。
- design_note: SSH 连接绑定 deviceID 不可跨设备共享, 故"用完即关"会丧失跨任务复用; LRU 退让兼顾复用 (idle 留池) + 不满 (池满淘汰 idle 给新设备让位), 优于用完即关。
