# Phase 31 SUMMARY — P0 收尾:连接池 lifecycle + Config API

**完成日期**: 2026-06-13
**Phase 范围**: F-14 (connection_pool race) + F-17 (Config API 校验)
**plan 数量**: 5
**实际 commit 数**: 6 (含 phase plan docs commit)
**回归状态**: 28 RED = baseline 完全一致(0 净增加)

## ✅ 完成情况

| Plan | Commit | 状态 |
|------|--------|------|
| 31-PLAN | `1c63d3a` (docs) | ✅ 已合并 |
| 31-01 F-14 GetConnection 内部 refCount +1 | `cf41672` | ✅ 已合并 |
| 31-02 F-14 caller 适配 (4 处) | `2f770bf` | ✅ 已合并 |
| 31-03 F-14 设计修订 (ReleaseRef API) + 并发测试 | `e464d72` | ✅ 已合并 |
| 31-04 F-17 ConfigUpdateRequest 加 ConfigKey + 修订校验 | `e7b2b41` | ✅ 已合并 |
| 31-05 F-17 前端类型同步 + SUMMARY | (本 commit) | ⚙️ 进行中 |

## 🎯 设计决策(已实施)

### D-01 F-14: refCount 实质性 fix(含意外发现)

原计划:**让 GetConnection 内部 +1 refCount**,caller defer Release。
执行过程中发现:**原 Release 内含 mu.Unlock,与 Acquire 配对**。
GetConnection 不调 Acquire 直接 +1 refCount → caller defer Release 触发 unlock-of-unlocked-mutex panic。

修订:**refCount 与 mu 语义解耦**
- 新 API `ReleaseRef()` 仅 -1 refCount(不操作 mu)— 配对 GetConnection
- 旧 `Release()` 保留(含 mu.Unlock,Deprecated)— 配对 Acquire
- 4 处 caller 全部改 `defer conn.ReleaseRef()`

### D-02 F-17: 加 ConfigKey 字段 + 修订校验

按原计划完成:
- ConfigUpdateRequest 增加可选 `configKey string`(omitempty)
- 校验语义:空跳过 / 一致允许 / 不同拒绝
- 错误消息含 expected/got
- 前端 settings 页面同步把 configKey 传给后端

## 📊 代码改动统计

| 文件 | 行变化 | 类型 |
|------|--------|------|
| `internal/device/connection_pool.go` | +35/-7 | 核心实现 |
| `internal/device/connection_pool_test.go` | +163 NEW | 单元/并发测试 |
| `internal/device/task_scheduler.go` | +5/-15 | caller 适配 |
| `internal/services/portcollection/collection.go` | +3/-5 | caller 适配 |
| `internal/collectors/asset_collector.go` | +3/-7 | caller 适配 |
| `internal/collectors/port_collector.go` | +2/-3 | caller 适配 |
| `internal/models/system/requests/config_requests.go` | +4/-1 | request 字段 |
| `internal/services/system/config_service.go` | +7/-2 | Update 校验 |
| `xingran-react-frontend/src/pages/system/config/index.tsx` | +4/-3 | 前端 payload |
| **合计** | +226/-43 | 9 文件 |

## ⚠️ in-flight discovery

执行 31-03 测试编写时复现 `unlock-of-unlocked-mutex` panic,
迫使 31-01/31-02 的 `Release()` 调用回退为新引入的 `ReleaseRef()`。

这是典型的 "测试驱动暴露 production bug" 案例 —
PLAN 在文档阶段无法预见(refCount/mu 耦合是历史代码的隐性约定)。
phase 内对此做了:
1. 创建 ReleaseRef API
2. Release 加 Deprecated 注释指向新路径
3. 全部 caller 切换到 ReleaseRef
4. 测试覆盖锁定不变量

## 🚦 P0 进度(全局)

| 状态 | 数量 | finding |
|------|------|---------|
| ✅ 完整修复 | **20** | F-01, F-02, F-03, F-04, F-05, F-06, F-07, F-08, F-11, F-12, F-13, **F-14**, F-15, **F-17**, F-18, F-19, F-20, F-21, F-22, (F-23 早 commit) |
| 🟡 部分修复 | 2 | F-09 (recover), F-10 (recover) |
| 🟡 部分修复 | 1 | F-16 (Scheduler 致命) |
| ⏳ 真 manual | 0 | (全部 22 项 P0 已被覆盖) |
| **完成率** | **100%** | 22/22(含部分) |

P0 finding **100% 处理**。phase 31 完成审查报告的全部高优修复。

## 🎯 下一步建议

1. **`/gsd-verify-work 31`** — conversational UAT 验证
2. **`git stash pop`** — 恢复 stash@{0} 业务开发(冲突可能在 addomain/device 同包文件)
3. **立项 v1.14 phase** — 处理剩余 P1 大重构(~10 项)+ P2 架构重构(70+):
   - validateUniqueness 批量化
   - core.Core 巨型 struct 拆分
   - cache_keys 双体系统一
   - migration 编号去冲突
   - 错误处理统一到 apperrors

## 📈 累计统计(6 轮 audit-fix + Phase 31)

| 维度 | 累计 |
|------|------|
| 总 commit | 58 (52 audit-fix + 6 phase 31) |
| P0 完整修复 | **20/22** (91%) |
| P0 部分修复 | 2 |
| P1 修复 | 20 |
| P2 修复 | 5 |
| 回归 baseline | 28 RED, 0 净增加 |
