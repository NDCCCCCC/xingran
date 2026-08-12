# 议题：reconciliation_normalized MV 刷新 30s 超时

**来源**: debug 会话 `normalize-interface-ge-missing`(2026-06-29)的副作用
**状态**: ✅ 已修复(migration_180,2026-06-30);待 DBA 生产验证耗时
**优先级**: 中→已降级(原 D-02 非阻断,现已根治 O(N²))
**关联**: migration_179 / migration_180 / `core.go:448` / `reconciliation_snapshot.go:51`

---

## 1. 症状

生产日志(2026-06-29 23:44):
```
ERRO REFRESH MATERIALIZED VIEW CONCURRENTLY reconciliation_normalized | 耗时: 30.0000439s | 错误: timeout: context deadline exceeded
WARN snapshot refresh failed, retry next cycle
ERRO Phase 42 R1 startup RefreshView failed (D-02 仅 log): timeout: context deadline exceeded
```

DBA 进一步诊断(2026-06-30):连最简单的 `SELECT COUNT(*) FROM reconciliation_physical_chain`
都触发 PostgreSQL **statement_timeout** → 视图查询本身灾难性慢,不只是 MV 刷新机制问题。

## 2. 根因(已诊断 — 修正版)

**不是"更多数据流过"那么简单,是视图结构 O(N×M) + migration_179 的 PL/pgSQL 函数放大:**

migration_179 把 interface_name 归一化从内联 `LOWER(REGEXP_REPLACE)` 换成 PL/pgSQL 函数
`normalize_iface()`(10 个顺序 REGEXP_REPLACE,单次约慢 10x)。逻辑正确,但视图的
`ops_info_points` JOIN 是**相关子查询**:

```sql
LEFT JOIN ops_info_points ip
       ON ip.port_id::text IN (
              SELECT id::text FROM sys_device_port_status
               WHERE device_id::text = n.device_id::text          -- 引用外层 norm.n
                 AND normalize_iface(interface_name) = n.mac_norm_iface  -- 逐行调函数
                 AND normalize_iface(interface_name) = n.port_norm_iface)
```

子查询引用外层 `n` → 每个 norm 行重跑,全表扫 port_status 且逐行调 normalize_iface 2 次 →
**O(norm_rows × port_status_rows × 2)** 函数调用。量级估算 N=M=5000 → 5000万次 × 10 正则 →
数十分钟级。叠加 `device_id::text` 强转 UUID → 索引失效,顺序扫描。

**30s 来源**:`internal/core/core.go:448` 硬编码
```go
refreshCtx, refreshCancel := context.WithTimeout(context.Background(), 30*time.Second)
```
`internal/services/asset/reconciliation_snapshot.go:51` 的 `RefreshView` 透传该 ctx。

**D-02 设计**:Phase 42 R1 有意"失败仅 log,retry next cycle",启动不阻断。
但隐患:`@every 5m` cron 走同一 RefreshView,若 cron context 也紧,**MV 可能持续刷新失败 → Layer3 检测读到陈旧数据**。

## 2.5 修复(migration_180,2026-06-30)✅

经 DBA `SELECT COUNT(*) FROM reconciliation_physical_chain` 触发 statement_timeout 确诊 O(N²) 后,
不再走 plan-phase,直接用 migration_180 重构视图(commit `4caa48b9`):

- `port_norm` / `latest_mac` CTE 预算 `normalize_iface`(O(N+M) 调用,非 O(N×M))
- 消除 `ops_info_points` 相关子查询:`norm` CTE 已 JOIN port 但未透传 `port.id`,
  重写后透传 `port.id`,外层改 `ip.port_id = port.id::text`(普通 JOIN)
- 去掉 `device_id::text` 强转(uuid=uuid,索引可用);`ip.port_id`/`workstation_id`/`ws.user_id`
  的 `::text` 保留(真 uuid↔text 类型不匹配,见 model 审计)
- `mac_join` 改用已 JOIN 的 `a.mac1/a.mac2`(消除 2 个相关子查询)

**normalize_iface 函数不动**:结构修复后调用从 ~5000万→1万次,已非主因;改函数有再次引入
42601 类 bug 风险(见 migration_179 v1 教训)。

**语义等价**:对 MV 消费者(`DISTINCT ON asset_id`)结果一致;直查视图去除"同设备多端口归一化
等值"时的交叉重复行(更干净)。消费者审计:仅 MV + migration_176 `ops_asset_physical`
(`ON CONFLICT asset_id`)— 均按 asset 去重,安全。

## 3. 关键事实

- migration_179 生产**已生效**(SQLSTATE 42601 已修复,app 跨过迁移进入 startup)
- MV 在 migration_179 step 4 的 `DROP + CREATE MATERIALIZED VIEW`(非 CONCURRENTLY,单遍)**已填充**
- migration_180 已重构视图(O(N+M)),待 DBA 验证 COUNT/REFRESH 回到秒级
- **MV 从空→有数据本身就是 normalize fix 生效的证据**

## 4. 候选方案(migration_180 已采用结构重构,余下为可选后续)

| 方案 | 状态 | 说明 |
|------|------|------|
| ✅ 视图结构重构(消除相关子查询) | **已做(migration_180)** | 主治,把 O(N×M)→O(N+M) |
| A. normalize_iface 单遍正则 | 不必要 | 结构修复后调用仅 ~1万次,已非主因;改函数风险高 |
| B. 函数索引 | 可选后续 | 若 180 后仍慢再加 `normalize_iface(interface_name)` 索引 |
| C. core.go:448 timeout 配置化 | 可选后续 | 30s→可配置;180 后应已足够,留作防御 |
| D. 物化物理链路中间表 | 不需要 | 180 已解决 |

**当前判断**:180 结构重构应已根治;B/C 作为 DBA 验证后的防御性后续。

## 5. plan-phase 前置数据(待 DBA 采集)

```sql
-- 1. 看实际刷新耗时分布(EXPLAIN ANALYZE,非 CONCURRENTLY 先测)
EXPLAIN (ANALYZE, BUFFERS) REFRESH MATERIALIZED VIEW reconciliation_normalized;

-- 2. 当前 MV 行数 + physical_user_id 命中率
SELECT COUNT(*),
       COUNT(*) FILTER (WHERE physical_user_id IS NOT NULL) AS hit,
       ROUND(100.0 * COUNT(*) FILTER (WHERE physical_user_id IS NOT NULL) / NULLIF(COUNT(*),0), 1) AS hit_pct
FROM reconciliation_normalized;

-- 3. 确认 cron context timeout 配置
-- 查 internal/scheduler 的 JobExecutor 默认 timeout(本议题需读代码确认)

-- 4. 表行数(评估 normalize_iface 调用量级)
SELECT COUNT(*) FROM sys_device_port_status WHERE deleted_at IS NULL;
SELECT COUNT(*) FROM sys_device_mac_address WHERE deleted_at IS NULL;
SELECT COUNT(*) FROM ops_info_points WHERE deleted_at IS NULL;
```

## 6. 临时缓解(若 cron 持续失败影响 R5 上线)

- 短期:core.go:448 临时改 120s(单点改动,可快速验证)
- 中期:走方案 B 加函数索引

## 7. 关联记忆

- [[gorm-automigrate-blocked-by-matview]] — MV 重建顺序
- migration_179 的 normalize_iface() 是本次引入的权威归一化函数,函数索引依赖其 IMMUTABLE
