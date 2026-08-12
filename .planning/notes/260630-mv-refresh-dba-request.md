# DBA 请求：reconciliation_normalized MV 刷新性能诊断

**交接对象**: DBA / 运维
**发起**: 2026-06-30
**议题**: `REFRESH MATERIALIZED VIEW CONCURRENTLY reconciliation_normalized` 在应用侧撞穿 30s 超时（`core.go:448`），导致 `@every 5m` cron 自动刷新可能持续失败、MV 数据陈旧
**关联**: `.planning/notes/260630-mv-refresh-30s-timeout.md`（议题背景）
**预期产出**: 本文档 §5 结果模板填好后回传给开发

---

## 0. 背景（30 秒版）

我们刚修复了一个长期根因（migration_179：`normalize_iface()` 函数让 `GE2/25` / `GigabitEthernet2/25` / `Gi2/25` 跨厂商命名在 JOIN 时等值）。修复前 MV 查询产出 ~0 行（瞬间完成），修复后真正命中行数，CONCURRENTLY 刷新变重，撞穿应用硬编码的 30s。

**MV 从空→有数据本身证明修复生效了。** 现在要诊断"刷新到底慢在哪、慢多少"，以决定根治方案（函数索引 / timeout 配置化 / 查询重写）。

> ⚠ **重要**: 应用侧的 30s 是 Go context 超时（`core.go:448`），**不是 PostgreSQL 语句超时**。所以在 psql 里直接跑下面的 REFRESH 不会被 30s 杀掉 —— 这正好让我们测出**真实耗时**。

---

## 1. 执行环境

- 连生产 PostgreSQL（与 xingran 后端同一库）
- 用 psql / pgAdmin / Navicat 均可，需能看到执行计划（EXPLAIN）
- 建议在**业务低峰**执行（REFRESH 会加锁）

---

## 2. Section A —— 确认修复生效（快，~10s）

### A1. normalize_iface 函数等值性

```sql
SELECT
    normalize_iface('GE2/25')               AS v1,
    normalize_iface('GigabitEthernet2/25')  AS v2,
    normalize_iface('Gi2/25')               AS v3,
    normalize_iface('ge2/25')               AS v4,
    normalize_iface('GigabitEthernet 2/25') AS v5,
    normalize_iface('gigabitether2/25')     AS v6;
```

**判定**:
- ✅ 6 个值全部 = `ge2/25` → 函数正确
- ❌ 任一不等 → 函数有 bug，立即反馈（不要继续后面）

### A2. physical_chain 视图命中率

```sql
SELECT COUNT(*) AS physical_chain_total,
       COUNT(*) FILTER (WHERE physical_user_id IS NOT NULL) AS hit
FROM reconciliation_physical_chain;
```

**判定**:
- ✅ `hit > 0`（修复前 = 0）→ 物理链路 JOIN 已恢复
- ❌ `hit = 0` → 修复未生效或数据源问题

### A3. MV 命中率

```sql
SELECT COUNT(*) AS mv_total,
       COUNT(*) FILTER (WHERE physical_user_id IS NOT NULL) AS hit,
       ROUND(100.0 * COUNT(*) FILTER (WHERE physical_user_id IS NOT NULL)
             / NULLIF(COUNT(*), 0), 1) AS hit_pct
FROM reconciliation_normalized;
```

**判定**:
- ✅ `hit > 0`，`hit_pct` 与已接线工位比例大致相符
- 记录 `hit_pct` 数值回传

---

## 3. Section B —— 诊断刷新耗时（核心，~2-5min）

### B1. 实测 CONCURRENTLY 刷新真实耗时

```sql
\timing on    -- psql 开启计时；pgAdmin/Navicat 用各自的时间显示

-- 记录开始时间
SELECT clock_timestamp() AS start_ts;

REFRESH MATERIALIZED VIEW CONCURRENTLY reconciliation_normalized;

-- 记录结束时间
SELECT clock_timestamp() AS end_ts;
```

**判定**:
- 记录**实际耗时**（这是最关键的一个数）
- < 30s → 应用 context 设置过紧，方案 C（调大 timeout）即可
- 30-60s → 边界，建议 B+C 组合
- > 60s → 查询本身慢，必须方案 B（函数索引）+ 可能查询重写

### B2. 取 MV 底层查询定义

```sql
SELECT pg_get_viewdef('reconciliation_normalized'::regclass, true);
```

**操作**: 把输出**整段 SELECT** 复制下来，用于 B3。

### B3. EXPLAIN ANALYZE 底层查询（关键）

> ⚠ `EXPLAIN ANALYZE REFRESH MATERIALIZED VIEW` **不是合法语法**（REFRESH 是 utility 命令，EXPLAIN 只会显示 "Utility Statement" 看不到计划）。正确做法是对 MV 的底层 SELECT 跑 EXPLAIN ANALYZE。

把 B2 的输出粘到下面 `<B2 的 SELECT>` 位置：

```sql
EXPLAIN (ANALYZE, BUFFERS, VERBOSE)
<B2 输出的整段 SELECT>;
```

**重点关注（把这几项回传）**:
1. `Execution Time:` 总耗时（毫秒）
2. 哪个节点耗时最多（找 `actual time` 最大的几行，通常是某个 Seq Scan 或 Hash Join）
3. 是否有 `normalize_iface` 相关的 Function Scan / 大量函数调用
4. 是否有 Hash Join 的 `Buckets` / `Memory Usage` 异常 / `Rows Removed`
5. Buffer 命中情况（`hit` vs `read`，read 多说明磁盘 IO）

### B4. physical_chain 视图本身的计划（normalize_iface 在这里被逐行调用）

```sql
EXPLAIN (ANALYZE, BUFFERS)
SELECT * FROM reconciliation_physical_chain WHERE physical_user_id IS NOT NULL;
```

**重点**: 这个视图的 JOIN 条件里 `normalize_iface(port.interface_name) = normalize_iface(mac.interface_name)` 会被逐行调用。看这里的耗时分布能直接定位是不是 normalize_iface 成本主因。

---

## 4. Section C —— 规模评估（快，~5s）

这些表的行数决定 normalize_iface 的调用次数量级：

```sql
SELECT 'sys_device_port_status' AS tbl, COUNT(*) AS n
FROM sys_device_port_status WHERE deleted_at IS NULL
UNION ALL
SELECT 'sys_device_mac_address', COUNT(*)
FROM sys_device_mac_address WHERE deleted_at IS NULL
UNION ALL
SELECT 'ops_info_points', COUNT(*)
FROM ops_info_points WHERE deleted_at IS NULL
UNION ALL
SELECT 'ops_asset', COUNT(*)
FROM ops_asset WHERE deleted_at IS NULL
UNION ALL
SELECT 'sys_workstation', COUNT(*)
FROM sys_workstation WHERE deleted_at IS NULL;
```

**判定**: 行数 × 2（port + mac 两侧）≈ normalize_iface 单次刷新调用次数。万级以上时函数索引收益显著。

---

## 5. 结果模板（回传给开发）

请把以下填好后回传（截图或文字均可）：

```
### Section A
A1 normalize_iface 6 值: ____ (期望全 ge2/25)
A2 physical_chain hit: ____ / ____ (期望 hit>0)
A3 MV hit: ____ / ____ (hit_pct = ____%)

### Section B
B1 CONCURRENTLY 刷新实测耗时: ____ 秒
B2 MV 定义: (粘贴 pg_get_viewdef 输出)
B3 EXPLAIN ANALYZE:
    - Execution Time: ____ ms
    - 最慢节点: ____
    - normalize_iface 是否成本主因: 是 / 否
    - Buffer hit/read: ____ / ____
B4 physical_chain 计划:
    - Execution Time: ____ ms
    - normalize_iface 调用成本占比: ____%

### Section C
sys_device_port_status: ____ 行
sys_device_mac_address: ____ 行
ops_info_points: ____ 行
ops_asset: ____ 行
sys_workstation: ____ 行

### 异常/观察
（任何报错、锁等待、意外现象）
```

---

## 6. 开发拿到数据后的决策树

| B1 实测耗时 | 主因（B3/B4） | 决策 |
|------------|--------------|------|
| < 30s | — | 方案 C：core.go timeout 配置化（应用侧过紧） |
| 30-60s | normalize_iface 成本高 + Section C 行数大 | 方案 B：函数索引 + 方案 C |
| > 60s | JOIN cardinality / Seq Scan | 方案 B + 查询重写 + 方案 C |
| 任意 | 其它（锁、IO） | 独立诊断 |

详见 `.planning/notes/260630-mv-refresh-30s-timeout.md` §4 候选方案。
