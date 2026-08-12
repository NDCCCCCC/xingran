# Phase 15: 性能优化 - Context

**Gathered:** 2026-06-15
**Status:** Ready for planning
**Source:** `gsd:discuss-phase 15` (用户主动澄清 4 个 gray area, 性能验证归 deferred)

<domain>
## Phase Boundary

在 Phase 12 (分区表 + BRIN 索引) + Phase 13 (4 个查询 API + ECharts 轨迹图) + Phase 14 (前端 UX 集成) 的稳定能力上, 进一步压榨查询性能、引入物化视图与 Redis 缓存层, 并新增端口使用热力图可视化。**本阶段不动采集层**(已稳定), **不动 Phase 13 既有查询 API 的响应结构** (向后兼容), **新增**: 复合索引 + 物化视图 + Redis 缓存 + ECharts 热力图前端页。

**Requirements (锁定, 来自 `.planning/REQUIREMENTS.md` v1.5)**:
- **PERF-01**: 实现查询性能优化 (复合索引 + BRIN 强化 + 分区裁剪)
- **PERF-02**: 实现物化视图 (4 类预聚合 + CONCURRENTLY 刷新)
- **PERF-03**: 实现查询结果缓存 (Redis 5 分钟 TTL)
- **PERF-04**: 实现端口使用热力图可视化 (ECharts 设备 × 端口二维)

**5 个预期子计划 (ROADMAP.md 锁定, 本 phase 不再切分)**:
- 15-01: 数据库索引与物化视图迁移 (复合索引 + 4 个物化视图)
- 15-02: 物化视图定时刷新任务 (5 分钟 CONCURRENTLY)
- 15-03: Redis 缓存中间层集成 (port/device/stats 三个查询)
- 15-04: 热力图后端 API + 前端独立页 (设备 × 端口二维)
- 15-05: 端到端联调与 EXPLAIN ANALYZE 抽样验证

**Out of scope**:
- 实时 MAC 流处理 (超出运维系统职责)
- MAC 异常告警 (后续里程碑)
- 轨迹回放 / MAC 随机化识别 (后续里程碑)
- 独立性能基准测试脚本 (本阶段仅在 IMPLEMENT 阶段抽样 EXPLAIN ANALYZE, 不含正式 benchmark)

</domain>

<decisions>
## Implementation Decisions

### 来自前序 Phase 继承的锁定决策 (D-01..D-06)

#### D-01: Phase 15 = 数据库 + 后端增强为主, 前端新增 1 个独立页
- 后端: 数据库迁移 (复合索引 + 物化视图) + 缓存层 (pkg/cache) + 新增 1 个 API (heatmap)
- 前端: 1 个新独立页 `/network/mac/heatmap`, 复用 Phase 14 的 `useTableManager` / `useColumnConfig` 资产
- 不动 Phase 13 既有 4 个查询 API 的响应结构 (向后兼容)
- 不重写 Phase 13 ECharts 轨迹图组件

#### D-02: 复用 Phase 12 已交付资产
- 月度分区策略 (D-12-D-05) — 保留, 复合索引与物化视图都建在分区表上
- BRIN 索引 — 保留 (Phase 12 D-05 已建), 不重写 pages_per_range
- 采集流程变更检测 (D-12-D-01..03) — 不动, 本阶段不触发新数据写入路径
- 配置存储 `sys_config` 表 — 沿用 D-13-2.3 模式, 物化视图相关配置键走同一张表

#### D-03: 复用 Phase 13 已交付资产
- `DeviceMACHistory` 模型 (`internal/models/device_mac_history.go`) — 不动
- `macHistoryQueryService` (4 个查询方法) — 扩展缓存装饰, 不改签名
- `macHistoryServiceImpl.MergeFlappingRecords` — 不动
- ECharts Gantt 模式 (`MACTrajectoryChart.tsx`) — 不动, 热力图另起 `MACHeatmapChart.tsx`

#### D-04: 复用 Phase 14 已交付资产
- 前端基础设施: React Query 5 (placeholderData: keepPreviousData) + AntD 6 + ECharts for react 3.0.5
- 列表/筛选 hooks: `useTableManager`, `useColumnConfig`, `useRealtimeUpdates`
- 路由模式: 独立路由 `React.lazy` + Suspense
- 菜单注册风格: `network:mac:*` 权限点命名延续 D-14-D-04

#### D-05: 复用项目既有缓存规范 (来自 CLAUDE.md + Phase 13 D-13-3.2)
- Redis 键前缀 `xingran:` (Core 初始化自动加, 调用时不要带)
- 缓存键命名空间风格: `mac:query:<query-name>:<hash>` (与 `mac:vendor:lookup` 保持一致)
- 缓存值序列化: JSON (复用 `DataCacheService` 已有能力)
- Redis 不可用降级: SQL 直查 (与 OUI L1 缓存降级策略保持一致)

#### D-06: 配置键命名约定
- `sys_config` 表新增键:
  - `network.mac.perf.mat_view_refresh_cron` (默认 `0 */5 * * * *`  — 每 5 分钟)
  - `network.mac.perf.cache_ttl_seconds` (默认 `300`)
  - `network.mac.perf.heatmap_top_n` (默认 `100`, 控制热力图最大端口数)
- 沿用 Phase 12 D-10 / Phase 13 D-13-2.3 风格: `network.mac.<area>.<key>`

### PERF-01 索引策略 (用户澄清)

#### D-07: 复合索引列顺序 = (device_id, mac_address, first_seen)
- 主查询路径 (Phase 13 `QueryPortHistory` / `QueryDeviceHistory`) 全部从 device_id 起步
- 高基数列在前 (device_id 基数 = 设备数, mac_address 基数 = MAC 数)
- first_seen 末位, 支持时间范围过滤 + ORDER BY first_seen
- 与已有 BRIN 索引互补: BRIN 走时间范围扫描, 复合索引走点查 + 排序

#### D-08: 索引方案 = 补 B-tree 复合索引 + 保留 BRIN
- 新增 B-tree 复合索引 `(device_id, mac_address, first_seen)` 1 个
- 保留 Phase 12 D-05 的 BRIN 索引 (`first_seen` 上)
- 不动 BRIN 的 `pages_per_range` (沿用 PostgreSQL 默认 32)
- 不加额外 B-tree 索引, 避免索引膨胀

### PERF-02 物化视图策略 (用户澄清)

#### D-09: 4 个物化视图全部覆盖
- **MV-01**: `mv_mac_port_latest` — 端口最新 MAC 状态 (device_id, interface_name, mac_address, last_seen, event_type)
- **MV-02**: `mv_mac_device_summary` — 设备 MAC 汇总 (device_id, mac_count, active_count, last_update)
- **MV-03**: `mv_mac_long_occupancy_top` — 长期占用 Top-N (mac_address, total_duration, last_port, snapshot_at), 限 Top 50
- **MV-04**: `mv_mac_port_daily_count` — 每日端口使用计次 (device_id, interface_name, date, change_count), PERF-04 数据源
- 全部建在分区表 `sys_device_mac_history` 上, 走分区裁剪

#### D-10: 刷新策略 = REFRESH MATERIALIZED VIEW CONCURRENTLY
- 所有 4 个物化视图都建 UNIQUE 索引 (CONCURRENTLY 强制要求)
- UNIQUE 索引设计:
  - MV-01: `(device_id, interface_name)`
  - MV-02: `(device_id)`
  - MV-03: `(mac_address, last_port)`
  - MV-04: `(device_id, interface_name, date)`
- 不用锁表刷新, 避免采集高峰期间锁等待
- 失败时不自动重试, 记录错误日志, 下个周期自然恢复

#### D-11: 调度 = 1 个统一 5 分钟 Cron 任务
- 注册任务 ID: `mac_history_matview_refresh`
- Cron 表达式: `0 */5 * * * *` (沿用项目 6 字段格式, 与 Phase 12 D-09 风格一致)
- 任务内依次刷新 4 个物化视图 (顺序: MV-01 → MV-02 → MV-03 → MV-04)
- 复用 `internal/scheduler` 已有的 `RegisterTask` 接口
- 单个物化视图刷新失败不影响后续, 但整个任务标记为部分失败

### PERF-03 缓存策略 (用户澄清)

#### D-12: 缓存范围 = port-history + device-history + stats, 不含 trajectory
- 缓存方法:
  - `QueryPortHistory` — 加缓存
  - `QueryDeviceHistory` — 加缓存
  - `QueryConnectionStats` — 加缓存
  - `QueryMACTrajectory` — **不加缓存** (参数组合爆炸, 命中率低, 由物化视图加速)
- 缓存值 = 整个 `*MACHistoryQueryResult` / `*ConnectionStatsResponse` JSON 序列化
- 缓存键 = `mac:query:<method>:<sha256(deviceId|interfaceName|startTime|endTime|current|pageSize)>`

#### D-13: 缓存键哈希 = 键名空间 + SHA-256
- 命名空间前缀: `mac:query:port-history` / `mac:query:device-history` / `mac:query:stats`
- 参数规范化后再哈希 (时间格式统一 RFC3339, MAC 地址转大写无分隔符)
- 哈希算法: SHA-256 (项目内已有 crypto 库, Go 标准库 `crypto/sha256`)
- 完整键示例: `mac:query:port-history:a1b2c3d4e5f6...` (64 字符哈希)

#### D-14: 失效策略 = 依赖 TTL
- 统一 TTL = 5 分钟 (300 秒), 与物化视图刷新周期对齐
- 不主动失效, 不监听采集完成事件
- 5 分钟内采集新数据可能查不到 (可接受, 采集本身是 5-15 分钟周期)
- 不做穿透/雪崩特殊处理 (走标准 cache-aside, 未命中时回源 DB)

#### D-15: 降级行为 = Redis 不可用时跳过缓存
- `GetOrSet` 调用返回错误时, 不阻断主查询, 走 DB 直查
- 沿用 Phase 13 D-13-3.2 OUI 缓存的降级模式
- 错误日志记录 `perf_cache_unavailable`, 不影响响应

### PERF-04 热力图策略 (用户澄清)

#### D-16: 热力图形态 = 设备 × 端口 二维
- X 轴: 端口 (interface_name, 按设备分组展示)
- Y 轴: 设备 (device_name_snapshot)
- 颜色深度: MAC 变化计数 (来自 MV-04)
- 时间范围: 默认近 7 天 (与 Phase 14 列表页 D-07 一致)
- ECharts series: `heatmap`, 复用 `echarts-for-react` (Phase 13 已引入)

#### D-17: 数据源 = 走物化视图 MV-04
- 后端 API: `POST /network/history/heatmap` (与 Phase 13 `POST /network/history/trajectory` 风格一致)
- 接收参数: `{ startTime, endTime, topN? }` (topN 默认 100, 走 D-06 `sys_config`)
- 直接查 `mv_mac_port_daily_count`, 不走 `sys_device_mac_history` 原表
- API 走 Redis 缓存 (5 分钟 TTL), 缓存键 = `mac:query:heatmap:<sha256(params)>`

#### D-18: 前端集成入口 = 独立路由 `/network/mac/heatmap`
- 路由: `xingran-react-frontend/src/pages/network/mac/heatmap.tsx` (与 `history.tsx`, `trajectory.tsx` 同级)
- 菜单: 父菜单 `network` 下与"历史查询" / "轨迹可视化"并列
- 权限点: `network:mac:heatmap` (沿用 D-14-D-04 命名风格)
- 复用 Phase 14 时间预设 + RangePicker (D-14-D-07)
- 移动端: 沿用 D-14-D-05 卡片视图策略

### Claude's Discretion

- 物化视图 UNIQUE 索引的 `INCLUDE` 列设计 (是否加 last_seen 等覆盖列)
- 物化视图刷新任务的错误重试策略 (建议: 失败后下次自然重试, 不加内存级重试)
- SHA-256 哈希前参数序列化的具体格式 (建议: JSON 序列化, 按 key 字母序排序确保稳定)
- 热力图颜色梯度 (建议: ECharts 默认蓝-绿-黄-红, 与 Phase 13 颜色编码风格协调)
- 缓存预热策略 (建议: 不预热, 启动后由用户查询自然填充)

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### 需求与里程碑
- `.planning/REQUIREMENTS.md` § v1.5 — PERF-01..04 需求定义与性能目标
- `.planning/ROADMAP.md` — Phase 15 边界 (Phases 12-15 v1.5 MAC地址历史数据管理)

### Phase 12-14 已锁决策 (继承)
- `.planning/phases/12-data-model-integration/12-CONTEXT.md` — D-01..D-10 全部继承
- `.planning/phases/13-query-layer-trajectory/13-CONTEXT.md` — D-13-1.1..4.4 全部继承
- `.planning/phases/14-frontend-ux/14-CONTEXT.md` — D-01..D-08 全部继承 (前端架构规范)

### 现有代码 (实现基础)
- `internal/models/device_mac_history.go` — 历史表模型 (继承使用)
- `internal/models/device_mac_address.go` — 当前 MAC 模型 (不动)
- `internal/services/mac_history_partition.go` — 分区管理服务 (沿用)
- `internal/services/mac_history_service.go` — 变更检测 + flapping 合并 (不动)
- `internal/services/mac_history_query_service.go` — 4 个查询方法 (扩展缓存装饰)
- `internal/services/data_cache_service.go` — JSON 序列化缓存 (复用)
- `internal/services/cache_config_service.go` — TTL 配置管理 (复用)
- `internal/api/v1/network/mac_history_handler.go` — 端点处理 (扩展, 不改签名)
- `internal/api/v1/network/mac_history_router.go` — 路由注册 (新增 heatmap 端点)
- `internal/scheduler/mac_history_tasks.go` — 定时任务模板 (新增 matview_refresh 任务)
- `internal/core/db/migrations/migration_mac_history.go` — 历史表迁移 (参考模式)

### 前端代码 (集成基础)
- `xingran-react-frontend/src/pages/network/mac/history.tsx` — 历史查询页 (参考 UI 模式)
- `xingran-react-frontend/src/pages/network/mac/trajectory.tsx` — 轨迹页 (参考 UI 模式)
- `xingran-react-frontend/src/components/network/MACTrajectoryChart.tsx` — ECharts Gantt (复用样式, 不重写)
- `xingran-react-frontend/src/hooks/useTableManager.ts` — 列表管理 hook (复用)
- `xingran-react-frontend/src/hooks/useColumnConfig.ts` — 列配置 hook (复用)
- `xingran-react-frontend/src/lib/api.ts` — API 包装 (走 post/get 函数)

### 项目规范
- `CLAUDE.md` § Cache System — Redis 键前缀 `xingran:` + CacheProvider 接口
- `CLAUDE.md` § Handler-Service Pattern — 标准服务接口模式
- `CLAUDE.md` § 数据库命名 — 复合索引/物化视图命名规范
- `.planning/codebase/ARCHITECTURE.md` — 分层架构、Core DI
- `.planning/codebase/STACK.md` — ECharts 6.0 / echarts-for-react 3.0.5 / React Query v5.90.12

### 外部参考
- PostgreSQL Materialized Views — https://www.postgresql.org/docs/current/rules-materializedviews.html
- PostgreSQL REFRESH MATERIALIZED VIEW CONCURRENTLY — https://www.postgresql.org/docs/current/sql-refreshmaterializedview.html
- PostgreSQL Partition Pruning — https://www.postgresql.org/docs/current/ddl-partitioning.html (MV 跨分区透明)
- PostgreSQL EXPLAIN ANALYZE — https://www.postgresql.org/docs/current/using-explain.html (Phase 15 抽样验证)
- ECharts Heatmap Series — https://echarts.apache.org/en/option.html#series-heatmap (PERF-04 实现)

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets (from Phase 12-14)
- **`macHistoryQueryService` 接口** (`internal/services/mac_history_query_service.go`) —
  4 个方法签名已稳定, Phase 15 在 service 实现内部加缓存装饰 (装饰器模式), 不改接口。
- **`PartitionService`** (`internal/services/mac_history_partition.go`) —
  Phase 12 已建分区+定时清理, 物化视图可建在分区表上享受分区裁剪。
- **`Scheduler` 框架** (`internal/scheduler/`) —
  `RegisterTask` 接口已就位, 直接注册 `mac_history_matview_refresh` 任务。
- **`DataCacheService`** (`internal/services/data_cache_service.go`) —
  通用 JSON 缓存能力, 提供 `Get` / `Set` / `Delete` / `DeleteByPattern` 方法。
- **`CacheConfigService`** (`internal/services/cache_config_service.go`) —
  动态 TTL 配置, 读取 `sys_config` 表键值, 支持运行时修改不重启。
- **Phase 14 ECharts 组件** (`MACTrajectoryChart.tsx`) —
  按需加载策略、tooltip 样式、错误兜底组件, Phase 15 复用样式不重写。
- **Phase 14 时间预设 + RangePicker** (D-14-D-07) —
  5 个预设按钮 (近 1h/24h/7d/30d/90d) + 自定义 RangePicker, Phase 15 热力图直接复用。

### Established Patterns
- **缓存键命名空间 + 哈希** — Phase 13 D-13-3.2 `mac:vendor:lookup` 已用, Phase 15 沿用风格。
- **cache-aside 降级** — Redis 不可用时 SQL 直查, 不阻断主流程 (Phase 13 OUI 缓存已验证)。
- **配置存储 `sys_config` 表** — `network.mac.<area>.<key>` 命名风格 (Phase 12 D-10, Phase 13 D-13-2.3)。
- **CONCURRENTLY 物化视图刷新** — 不锁表, 需 UNIQUE 索引, 失败可恢复 (本阶段 D-10 锁定)。
- **前端 ECharts 按需加载** — `echarts/charts` 引入模式 (Phase 30 Plan 02, Phase 13 D-13 已用)。
- **菜单注册风格** — `network:mac:<action>` 权限点 (Phase 14 D-14-D-04 已建 `list`/`query`/`export`)。

### Integration Points
- **后端** —
  - `internal/api/router.go:325` — MAC 历史路由注册位置, Phase 15 在 `mac_history_router.go` 内新增 `heatmap` 端点。
  - `internal/scheduler/` — 定时任务注册位置, 沿用 `RegisterMACHistoryTasks` 模式。
  - `internal/core/db/migrations/` — 新增 `migration_NNN_mac_perf_indexes.go` 和 `migration_NNN_mac_matview.go`。
- **前端** —
  - `xingran-react-frontend/src/router/` — 路由配置, 新增 `/network/mac/heatmap`。
  - `xingran-react-frontend/src/components/Layout/Menu/` — 菜单配置, 新增 "端口使用热力图" 菜单项。
  - `xingran-react-frontend/src/pages/network/mac/` — 新增 `heatmap.tsx` 与 `trajectory.tsx` / `history.tsx` 并列。

### Known Constraints
- **Phase 12 UAT 阻塞项** (清理任务注册) — 已在 Phase 12 D-09 标记, Phase 15 沿用 `mac_history_cleanup` 任务, 仿写 `mac_history_matview_refresh`。
- **Phase 13 端点签名锁定** — 4 个查询方法响应结构 Phase 15 不改, 缓存装饰必须在 service 层透明完成。
- **物化视图 CONCURRENTLY 失败处理** — 失败后下次自然重试, 不引入内存级重试 (D-10 已锁定)。
- **采集周期 (5-15 分钟)** — 缓存 5 分钟 TTL 与采集周期一致, 数据延迟可接受 (D-14 已确认)。

</code_context>

<specifics>
## Specific Ideas

### 来自 Phase 12-14 经验的可复用模式

- **物化视图命名规范**: `mv_mac_<area>_<granularity>` (如 `mv_mac_port_latest`, `mv_mac_port_daily_count`), 与项目命名风格一致。
- **UNIQUE 索引顺序与覆盖列**: 优先选常用于 JOIN / WHERE 的列在最前, `INCLUDE` 可选覆盖列加速 index-only scan。
- **缓存装饰器实现位置**: 在 `macHistoryQueryServiceImpl` 内部 (服务层), 不在 handler 层 (保持接口签名稳定)。
- **热力图 ECharts 配置**: 复用 Phase 13 颜色编码 (appeared=绿 / disappeared=红 / moved=黄 / vlan_changed=蓝), 颜色梯度范围 0 → max 计数。
- **EXPLAIN ANALYZE 抽样验证**: 在 `query_test.go` 中加 3-5 个典型 SQL 的 EXPLAIN ANALYZE 输出断言, 不写独立 benchmark 脚本 (D-15 已锁定)。
- **前端热力图移动端策略**: 桌面端 ECharts heatmap 完整展示, 移动端降级为 Top-20 端口列表 + 颜色卡片 (避免 ECharts 移动端渲染压力)。

### 性能目标对照 (来自 REQUIREMENTS.md v1.5 PERF-01)

- 90 天范围查询 < 2 秒
- 1 年范围查询 < 5 秒
- 这些目标在 Phase 15 IMPLEMENT 阶段抽样验证, 不写独立 benchmark 脚本 (用户已确认 deferred)

</specifics>

<deferred>
## Deferred Ideas

- **独立性能基准测试脚本** (用户主动 deferred) — Phase 15 仅在 IMPLEMENT 阶段抽样 EXPLAIN ANALYZE, 不含 `15-XX-perf-benchmark.go`。后续如有需要可单开 phase。
- **缓存预热机制** (Claude's discretion 已记) — Phase 15 不做, 启动后由用户查询自然填充。
- **实时 MAC 变化推送** (WebSocket / SSE) — 超出 Phase 15 范围, 属后续里程碑 (与 Phase 14 列表页 D-14 实时刷新区分)。
- **物化视图增量刷新** (基于 NOTIFY/LISTEN) — 超出 Phase 15 范围, 5 分钟全量刷新已能满足性能目标。
- **缓存主动失效 (pub/sub)** — 沿用 TTL 自然过期 (D-14 已锁定), 不监听采集完成事件。
- **MAC 厂商识别强化** (Phase 13 D-13-3 已交付) — 不在 Phase 15 范围。
- **异常告警 (ALERT-01/02)** — REQUIREMENTS v1.6+ 范围, 不在 v1.5。
- **轨迹回放 (ADV-02)** — REQUIREMENTS v1.7+ 范围, 不在 v1.5。

</deferred>

---

*Phase: 15-performance-optimization*
*Context gathered: 2026-06-15*
