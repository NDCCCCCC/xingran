---
status: resolved
phase: 62-ai-internal-core-db
source: [62-VERIFICATION.md]
started: 2026-08-15T00:30:00+08:00
updated: 2026-09-06T12:00:00+08:00
resolution_note: 2026-09-06 v1.29 归档预检裁决——本文件属 v1.27 workstream 跨里程碑遗留，与 v1.29 无关；3 个 pending 场景 acknowledge 转入 v1.29 STATE.md Deferred Items（原里程碑语境已关闭，不再构成阻塞）
---

## Current Test

[awaiting human testing]

## Tests

### 1. Migrate176 R1/R2→R5 就地升级(schema 校验回退)
expected: 在带旧结构 MV(pre-R5,缺 asset_username/physical_user_id/ad_*/last_resolved_* 列)的真实 PG 上启动后端,快路径 information_schema 校验发现列缺失 → 回退 DROP+CREATE 慢路径,新列全部就位,启动日志含回退原因;已有新结构 MV 时走 REFRESH CONCURRENTLY 快路径无回退
result: [pending]

### 2. Advisory lock 双实例并发迁移保护
expected: 同一 PG 上启动第二个后端实例,第二实例日志出现 `[advisory-lock] 另一实例正在执行启动迁移,本实例跳过 175/176/202-205 迁移块` WARN 且正常完成启动;第一实例锁在迁移结束后正确释放(pgL advisory_locks 无残留)
result: [pending]

### 3. 空库首启 admin 种子凭据告警
expected: 空库 + 不设 SYS_ADMIN_BOOTSTRAP_PASSWORD 启动 → 日志出现含 `admin123` 的 WARN(默认凭据告警);设 SYS_ADMIN_BOOTSTRAP_PASSWORD=xxx 启动 → 种子用户密码为 env 值,无默认凭据告警;数据库中 admin 用户 salt 不为 "default"(空串或随机)
result: [pending]

## Summary

total: 3
passed: 0
issues: 0
pending: 3
skipped: 0
blocked: 0

## Gaps
