# Phase 13: 查询层与轨迹 - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-06-13
**Phase:** 13-query-layer-trajectory
**Areas discussed:** MAC 轨迹查询策略, 停留时长计算口径, OUI 厂商库来源与更新, ECharts 轨迹图形态

---

## Area 1: MAC 轨迹查询策略

| Option | Description | Selected |
|--------|-------------|----------|
| PostgreSQL 窗口函数 | LAG() OVER (PARTITION BY mac_address ORDER BY first_seen), 跨分区透明 | ✓ |
| 应用层多查询 + Go 聚合 | 多次查询后内存排序, N+1 风险 | |
| PostgreSQL 递归 CTE | WITH RECURSIVE 走状态转换图, 杀鸡用牛刀 | |
| 轨迹节点粒度: 历史记录粒度 | 每个事件一条节点 | |
| 轨迹节点粒度: 聚合后区间 | 合并连续状态, 保留 event_type 元数据 | ✓ (Claude 建议) |
| 轨迹节点粒度: 双粒度可切换 | Query 参数 ?aggregation=true/false | |
| 当前状态混合: 仅 history 表 | 简单一致 | ✓ |
| 当前状态混合: 混合 history + current | LEFT JOIN sys_device_mac_address | |
| 跨分区: LAG() 跨分区透明 | 原生分区裁剪自动处理 | ✓ |
| 跨分区: 应用层补偿 | Go 手动 UNION ALL | |

**User's choice:** PostgreSQL 窗口函数（推荐）+ 使用最佳实践（Claude 建议 = 聚合后区间粒度）
**Notes:** 4 个子问题一次性回答，前 2 选推荐项，后 2 由 Claude 决策。

---

## Area 2: 停留时长计算口径

| Option | Description | Selected |
|--------|-------------|----------|
| 口径: 仅固化区间 | duration = last_seen - first_seen | ✓ |
| 口径: 固化 + 未结束状态估算 | last_seen 距 NOW() < 采集周期则 NOW() | |
| 口径: 混合方案 (状态字段区分) | completed_duration + is_active + current_duration | |
| 单位: UTC 秒数 | int64 存, UI 格式化 | ✓ |
| 单位: PostgreSQL interval | 原生 INTERVAL 类型, 跨夏令时 | |
| 单位: 浮点小时数 | 73.25 小时, 精度风险 | |
| 阈值: 可配置 | sys_config 配置, 默认 30 天 | ✓ |
| 阈值: 硬编码 30 天 | 代码硬编码 | |
| 阈值: 多级别 (7/30/90) | 长期 / 超长期 / 异常占用 | |
| 统计输出: 明细 + Top-N | 明细 + 按 MAC 长期 Top + 按端口热门 | ✓ |
| 统计输出: 仅聚合 | 总时长 / 平均 / 最大 | |
| 统计输出: 完整分析 | 加变动率 / OUI 分布 | |

**User's choice:** 全部选推荐项
**Notes:** 决策路径明确，所有选项都倾向"标准做法"。

---

## Area 3: OUI 厂商库来源与更新

| Option | Description | Selected |
|--------|-------------|----------|
| 存储: 独立表 | sys_mac_oui_vendor 表 + 启动导入 | ✓ |
| 存储: Go 内嵌静态 map | 编译期嵌入, 不可热更新 | |
| 存储: 远程拉取 | IEEE 官方 CSV, 外部依赖 | |
| 性能: DB LEFT JOIN + L1 缓存 | 启动加载到 Redis, 降级 SQL | ✓ |
| 性能: 每次 SQL LEFT JOIN | N+1 风险 | |
| 性能: 完全 Redis 缓存 | 零 DB 负载, 故障=不可用 | |
| 更新: JSON 快照随仓库 | configs/oui-vendors.json git 版本化 | ✓ |
| 更新: 后台 API 手动 | POST /network/oui/import | |
| 更新: 定时任务远程拉取 | 运行时拉 IEEE | |
| 未知 OUI: Unknown Vendor | 简单, 不影响主流程 | ✓ |
| 未知 OUI: 识别随机化 MAC | Apple/Android 随机化前缀 | |
| 未知 OUI: 双标签输出 | known / unknown / randomized | |

**User's choice:** 全部选推荐项
**Notes:** 用户对"识别随机化 MAC"明确说不（避免 Phase 13 复杂度），契合 D-13-3.4。

---

## Area 4: ECharts 轨迹图形态

| Option | Description | Selected |
|--------|-------------|----------|
| 范围: 单 MAC 焦点视图 | 专注单一设备轨迹 | ✓ |
| 范围: 多 MAC 对比视图 | Gantt 变体, 安全审计 | |
| 范围: 单 MAC + 顶邻联动 | 折中方案 | |
| 高亮: 颜色 + tooltip | 颜色编码 + 详细信息 | ✓ |
| 高亮: 仅颜色 + 点击跳转 | 跳转详情页 | |
| 高亮: 简化 Gantt 样式 | 不标记事件 | |
| 跨设备: 按设备分组纵轴 | 设备 A 端口组 / 设备 B 端口组 | ✓ |
| 跨设备: 扁平端口列表 | 简单但无归属 | |
| 跨设备: 设备泳道 + 连接线 | 更直观但复杂 | |
| 空状态: Ant Design Empty | 标准组件 + 骨架屏 + Alert | ✓ |
| 空状态: 智能引导推荐原因 | 首次出现 / 不在范围 / 时间无事件 | |
| 空状态: 仅表格降级 | 不展示图表 | |

**User's choice:** 全部选推荐项
**Notes:** 用户对"前端"决策都倾向"标准/简单"路径，未要求高级交互。

---

## Claude's Discretion

- D-13-1.2 节点粒度（聚合 vs 历史）— 由 Claude 推荐"聚合后区间"
- D-13-1.3 是否混合 current 表（仅 history）— Claude 推荐"仅 history"
- D-13-1.4 跨分区处理（LAG 透明）— Claude 推荐
- D-13-3.4 未知 OUI 处理（Unknown Vendor）— Claude 推荐
- D-13-4.4 空状态（Ant Design 默认）— Claude 推荐

总计 5 项由 Claude 决策, 11 项用户直接选择。

## Deferred Ideas

- **MAC 随机化识别** (REQUIREMENTS v1.7+ ADV-01) — D-13-3.4 不实现
- **多 MAC 对比视图** — Area 4 选项 B
- **轨迹回放** (REQUIREMENTS v1.7+ ADV-02) — 新能力
- **异常告警通知** (REQUIREMENTS v1.6+ ALERT-01/02) — 新能力
- **MAC flapping 告警** — 统计输出, 不发告警
- **Phase 12 清理任务注册** — 不属 Phase 13 scope, 但验证时需检查
