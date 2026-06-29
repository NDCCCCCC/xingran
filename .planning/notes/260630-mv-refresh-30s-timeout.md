# 议题：reconciliation_normalized MV 刷新 30s 超时

**来源**: debug 会话 `normalize-interface-ge-missing`(2026-06-29)的副作用
**状态**: 待 plan-phase(独立议题,非 normalize fix 本身)
**优先级**: 中(D-02 非阻断,但 cron 自动刷新可能持续失败 → MV 数据陈旧)
**关联**: migration_179 / `core.go:448` / `reconciliation_snapshot.go:51`

---

## 1. 症状

生产日志(2026-06-29 23:44):
```
ERRO REFRESH MATERIALIZED VIEW CONCURRENTLY reconciliation_normalized | 耗时: 30.0000439s | 错误: timeout: context deadline exceeded
WARN snapshot refresh failed, retry next cycle
ERRO Phase 42 R1 startup RefreshView failed (D-02 仅 log): timeout: context deadline exceeded
```

## 2. 根因(已诊断)

**这是 normalize fix 生效的副作用,不是 fix 的 bug:**

| 阶段 | reconciliation_physical_chain JOIN 产出 | MV 刷新成本 |
|------|----------------------------------------|------------|
| 修复前 | ~0 行(physical_user_id 恒 NULL,GE≠GigabitEthernet) | 极低,CONCURRENTLY 瞬间完成 |
| 修复后 | 真实命中(GE/GigabitEthernet/Gi 现在等值) | 变重,撞穿 30s |

修复后 `normalize_iface()`(10 个顺序 REGEXP_REPLACE)被 view JOIN 条件**逐行**调用
(port × mac 两侧),加上 CONCURRENTLY 要重建一份副本(2x 工作量)→ 超过 30s。

**30s 来源**:`internal/core/core.go:448` 硬编码
```go
refreshCtx, refreshCancel := context.WithTimeout(context.Background(), 30*time.Second)
```
`internal/services/asset/reconciliation_snapshot.go:51` 的 `RefreshView` 透传该 ctx。

**D-02 设计**:Phase 42 R1 有意"失败仅 log,retry next cycle",启动不阻断。
但隐患:`@every 5m` cron 走同一 RefreshView,若 cron context 也紧,**MV 可能持续刷新失败 → Layer3 检测读到陈旧数据**。

## 3. 关键事实

- migration_179 生产**已生效**(SQLSTATE 42601 已修复,app 跨过迁移进入 startup)
- MV 在 migration_179 step 4 的 `DROP + CREATE MATERIALIZED VIEW`(非 CONCURRENTLY,单遍)**已填充**
- 所以 dashboard 当前能读到正确数据;只是自动刷新路径超时
- **MV 从空→有数据本身就是 normalize fix 生效的证据**

## 4. 候选方案(plan-phase 时评估)

| 方案 | 改动 | 风险 | 预期 |
|------|------|------|------|
| A. normalize_iface 单遍正则 | 把 10 个顺序 REGEXP_REPLACE 重写为单次捕获+CASE | 低(纯函数内) | 可能压到 30s 内,但 JOIN cardinality 可能仍是主因 |
| B. 函数索引 | `CREATE INDEX ON sys_device_port_status (normalize_iface(interface_name))` + 同 mac 表 | 中(port_status 写密集,索引维护成本;需 EXPLAIN 确认被采用) | JOIN 走索引,大幅加速 |
| C. bump/配置化 timeout | core.go:448 30s→可配置(默认 120s) | 低 | 治标,不解决查询本身慢 |
| D. 物化物理链路中间表 | 把 reconciliation_physical_chain 从视图改物化视图,cron 只刷新它 | 高(schema 重构) | 分摊成本 |

**推荐组合**:B(函数索引)+ C(timeout 配置化)。A 作为附带优化。

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
