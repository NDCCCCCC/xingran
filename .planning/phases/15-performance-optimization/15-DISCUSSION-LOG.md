# Phase 15: 性能优化 - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-06-15
**Phase:** 15-performance-optimization
**Areas discussed:** PERF-01 索引策略, PERF-02 物化视图范围, PERF-03 缓存粒度, PERF-04 热力图形态

---

## PERF-01 复合索引列顺序

| Option | Description | Selected |
|--------|-------------|----------|
| (device_id, mac_address, first_seen) | 适配 Phase 13 QueryPortHistory / QueryDeviceHistory 主查询路径, 高基数列在前 | ✓ |
| (device_id, first_seen DESC) | 适配按设备 + 时间范围倒序扫描, 性能在 1 年查询 (<5s 目标) 更佳 | |
| (mac_address, first_seen) | 适配 QueryMACTrajectory MAC 轨迹, 但 MAC 基数高, 索引体积大 | |

**User's choice:** (device_id, mac_address, first_seen)
**Notes:** 用户优先匹配 Phase 13 现有主查询路径, 避免重写查询逻辑。高基数列在前符合 B-tree 设计原则。

## PERF-01 BRIN 调优策略

| Option | Description | Selected |
|--------|-------------|----------|
| 补 B-tree + 保留 BRIN | 保留 Phase 12 D-05 BRIN 索引 + 另为 device_id + first_seen 加 B-tree | ✓ |
| 压缩 BRIN + 新增 B-tree | 保留 BRIN + 调优 pages_per_range=128, 适合 1 年以上时间范围查询 | |
| 只动 B-tree | 仅靠 BRIN + 分区裁剪, 不动 B-tree | |

**User's choice:** 补 B-tree + 保留 BRIN
**Notes:** 用户接受保留 Phase 12 BRIN + 新增 B-tree 复合索引, 索引膨胀风险可控。pages_per_range 沿用默认 32, 不调优。

## PERF-02 物化视图覆盖范围 (多选)

| Option | Description | Selected |
|--------|-------------|----------|
| 端口最新 MAC 状态 | MV-01: 端口当前最新 MAC, 用于端口详情 | ✓ |
| 设备 MAC 汇总统计 | MV-02: 设备下 MAC 总数/活跃数 | ✓ |
| 长期占用 Top-N | MV-03: 按 MAC + 设备维度预计算累计停留时长 | ✓ |
| 每日端口使用计次 | MV-04: 端口 × 日维度预聚合, PERF-04 数据源 | ✓ |

**User's choice:** 全部 4 个物化视图
**Notes:** 用户希望全量覆盖, 每个视图针对一个具体查询模式优化。

## PERF-02 物化视图刷新策略

| Option | Description | Selected |
|--------|-------------|----------|
| REFRESH MATERIALIZED VIEW CONCURRENTLY | 每个视图都建 UNIQUE 索引, 后台不锁表刷新 | ✓ |
| REFRESH MATERIALIZED VIEW (锁表) | 老式全量刷新, 锁表, 采集高峰可能锁等待 | |
| 混用: Top-N 全量, 其他 CONCURRENTLY | Top-N + 设备汇总全量刷新, 端口最新 + 每日计次 CONCURRENTLY | |

**User's choice:** REFRESH MATERIALIZED VIEW CONCURRENTLY (推荐)
**Notes:** 4 个视图统一使用 CONCURRENTLY, 全部建 UNIQUE 索引。失败不自动重试, 记录日志。

## PERF-02 调度粒度

| Option | Description | Selected |
|--------|-------------|----------|
| 统一 5 分钟调度 | 1 个 Cron 作业依次刷新 4 个视图, 实现简单 | ✓ |
| 4 个独立调度 (错峰) | 为每个视图独立 Cron, 错峰刷新, 均匀 DB 负载 | |
| PostgreSQL 触发 + 定时调度 | 走 pg_cron 扩展 / LISTEN/NOTIFY, 需安装扩展 | |

**User's choice:** 统一 5 分钟调度 (推荐)
**Notes:** 实现简单, 顺 D-09 风格, 任务内依次刷新。采集高峰期间 4 个叠加压力可接受。

## PERF-03 缓存范围

| Option | Description | Selected |
|--------|-------------|----------|
| port + device + trajectory + stats | 全量缓存, 4 个查询都加缓存 | |
| 仅 port + device | 只缓存点查型, trajectory 和 stats 走物化视图 | |
| port + device + stats (不含 trajectory) | 点查 + 统计走缓存, trajectory 走物化视图 | ✓ |

**User's choice:** port + device + stats (不含 trajectory)
**Notes:** trajectory 参数组合爆炸, 命中率低, 物化视图加速更合适。

## PERF-03 缓存键设计

| Option | Description | Selected |
|--------|-------------|----------|
| 键名空间 + SHA-256 | mac:query:<query-name>:<sha256(params)>, 哈希压缩参数 | ✓ |
| 全参数拼接 | mac:query:<query-name>:<param1>:<param2>:..., 可读性高, 需转义 | |
| 参数 JSON 序列化 | mac:query:<query-name>:<json>, 简单但 key 较长 | |

**User's choice:** 键名空间 + SHA-256 (推荐)
**Notes:** 与 Phase 13 D-13-3.2 `mac:vendor:lookup` 命名风格保持一致。

## PERF-03 缓存失效策略

| Option | Description | Selected |
|--------|-------------|----------|
| 依赖 TTL (推荐) | 不主动失效, 5 分钟 TTL 自然过期, 与物化视图刷新周期一致 | ✓ |
| 主动失效 + 降级 TTL | 采集完成 pub/sub DEL 缓存, 更准确但增加耦合 | |
| 多层 (5min TTL + 写后失效) | cache-aside + 主动 DEL + 30s TTL 安全网 | |

**User's choice:** 依赖 TTL (推荐)
**Notes:** 实现简单, 5 分钟内数据延迟可接受 (采集本身 5-15 分钟周期)。

## PERF-04 热力图形态

| Option | Description | Selected |
|--------|-------------|----------|
| 设备 × 端口 二维 | X 轴端口, Y 轴设备, 颜色 MAC 计数, 透视设备与端口关系 | ✓ |
| 端口 × 时间网格 (日历图) | X 轴日期, Y 轴端口, 颜色计数, 传统热力图 | |
| 时间 × 设备 趋势图 | X 轴时间, Y 轴设备, 颜色频次, 偏趋势 | |

**User's choice:** 设备 × 端口 二维 (推荐)
**Notes:** 与 PERF-02 MV-04 (设备+端口+日期) 数据源高度匹配, 透视"哪些设备下哪些端口最活跃"。

## PERF-04 数据源

| Option | Description | Selected |
|--------|-------------|----------|
| 走物化视图 MV-04 | 后端新增 API, 直接查 MV-04, 走物化视图 + 缓存 | ✓ |
| 现算后缓存 | 不走物化视图, 现算后缓存, 实时同步但压力大 | |
| 物化视图 + 现算双路径 | 默认物化视图, realtime=true 跳过, 灵活但维护成本高 | |

**User's choice:** 走物化视图 (推荐)
**Notes:** 与 PERF-02 MV-04 强关联, 单一数据源, 简单一致。

## PERF-04 前端集成入口

| Option | Description | Selected |
|--------|-------------|----------|
| 独立页 /network/mac/heatmap | 独立路由, 与 history/trajectory 并列 | ✓ |
| 历史查询页新增 Tab | 作为 /network/mac/history 页面的 viewMode 切换 | |
| Dashboard Widget | 作为可配置仪表盘 Widget, 复用 useWidgetPolling | |

**User's choice:** 独立页 /network/mac/heatmap (推荐)
**Notes:** 复用 Phase 14 菜单注册风格, 权限点 network:mac:heatmap, 与 history/trajectory 三件套并列。

## 性能验证范围

| Option | Description | Selected |
|--------|-------------|----------|
| 不包含, 仅实施 | IMPLEMENT 阶段抽样 EXPLAIN ANALYZE, 不含正式 benchmark 脚本 | ✓ |
| 包含基准验证脚本 | 新增 15-XX-perf-benchmark.go, 预热数据 + EXPLAIN ANALYZE 计时断言 | |

**User's choice:** 不包含, 仅实施 (推荐)
**Notes:** 用户主动 deferred, 减轻 phase 负担。后续如有需要可单开 phase。

---

## Claude's Discretion

- 物化视图 UNIQUE 索引的 `INCLUDE` 列设计 (是否加 last_seen 等覆盖列)
- 物化视图刷新任务的错误重试策略 (建议: 失败后下次自然重试, 不加内存级重试)
- SHA-256 哈希前参数序列化的具体格式 (建议: JSON 序列化, 按 key 字母序排序确保稳定)
- 热力图颜色梯度 (建议: ECharts 默认蓝-绿-黄-红, 与 Phase 13 颜色编码风格协调)
- 缓存预热策略 (建议: 不预热, 启动后由用户查询自然填充)

## Deferred Ideas

- **独立性能基准测试脚本** (用户主动 deferred) — Phase 15 仅在 IMPLEMENT 阶段抽样 EXPLAIN ANALYZE, 不含 `15-XX-perf-benchmark.go`
- **缓存预热机制** — Phase 15 不做, 启动后由用户查询自然填充
- **实时 MAC 变化推送** (WebSocket / SSE) — 超出 Phase 15 范围, 属后续里程碑
- **物化视图增量刷新** (基于 NOTIFY/LISTEN) — 超出 Phase 15 范围
- **缓存主动失效 (pub/sub)** — 沿用 TTL 自然过期
- **MAC 厂商识别强化** (Phase 13 D-13-3 已交付) — 不在 Phase 15 范围
- **异常告警 (ALERT-01/02)** — REQUIREMENTS v1.6+ 范围
- **轨迹回放 (ADV-02)** — REQUIREMENTS v1.7+ 范围
