---
phase: 38
slug: unify-ad-authentication-onto-account-pool-remove-legacy-sing
status: draft
nyquist_compliant: true
wave_0_complete: false
created: 2026-06-23
---

# Phase 38 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.
>
> 本 phase 是 AD 认证统一重构（移除遗留单管理员双轨）。验证目标：**证明所有 AD 连接已走账号池 FailoverClient、无残留单管理员路径**。详见 `38-RESEARCH.md` `## Validation Architecture` 章节。

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test (Go 1.24) + 现有 addomain 包测试 |
| **Config file** | 无独立配置 — 测试随包内 `*_test.go` |
| **Quick run command** | `go build ./... && go test ./internal/services/addomain/ ./internal/core/security/` |
| **Full suite command** | `go test ./...` |
| **Estimated runtime** | ~60-120 秒（含 addomain 包回归） |

**已有回归守护（不可破坏）：**
- `TestAccountPoolPasswordRoundTrip`（账号池加解密闭环契约）
- `TestFailoverClient`（FailoverClient 多账号故障切换契约）
- `internal/services/addomain/manager_sync_test.go`（SyncManagersToAD 7 个回归测试依赖 `updateUserAttributeFn` 钩子，重构必须保留）
- `TestDecryptPassword_InvalidCiphertext`（解密函数防御性契约，函数本身保留）

---

## Sampling Rate

- **After every task commit:** Run `go build ./... && go test ./internal/services/addomain/ ./internal/core/security/`
- **After every plan wave:** Run `go test ./...`
- **Before `/gsd:verify-work`:** Full suite must be green + grep 断言全部通过
- **Max feedback latency:** 120 秒

---

## 残留路径断言（本 phase 特有，必须全绿）

重构完成的硬性证明 — 以下 grep 在 phase 收尾时必须返回 **0 匹配**（排除测试钩子与 @Deprecated 兼容读取）：

| 断言 | 命令 | 期望 |
|------|------|------|
| 无单管理员密码手动解密 | `grep -rn "decryptPassword(config.AdminPassword)" internal/` | 0 |
| 无 NewLDAPClient 单参数（config）生产调用 | `grep -rn "NewLDAPClient(config)" internal/` 或不带 account 的变体 | 0（仅 failover_client.go 内部例外） |
| 前端无 admin 输入项 | `grep -c "adminUsername\|adminPassword" xingran-react-frontend/src/pages/ad-domain/configs/index.tsx xingran-react-frontend/src/lib/adDomainApi.ts` | 0（W-03 精确零残留，含注释） |

---

## Per-Task Verification Map

> Planner 在 PLAN.md 创建后回填本表（Task ID / Plan / Wave / 对应断言）。

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 38-01-T1 | 01 | 1 | D-01 Wave1 前置 DI | T-38-01-DI | 8 服务 struct 注入 pool 字段，构造函数接收 pool；钩子字段保留 | unit + build | `go build ./internal/services/addomain/ && grep -c "pool AccountPool" {8 files}` | ✅ | ⬜ pending |
| 38-01-T2 | 01 | 1 | D-01 Wave1 前置 DI | T-38-01-DI | NewADDomainService 签名变更 (db,pool,cipher)，router/scheduler 复用同一实例（Pitfall 4） | unit + build + integration | `go build ./... && go test ./internal/services/addomain/ -run TestSyncManagersToAD -v` | ✅ | ⬜ pending |
| 38-02-T1 | 02 | 2 | D-01 Wave1 连接层 | T-38-03-Closure, T-38-05-EmptyPool | 11 处单次操作型 caller 改 ExecuteWithFailover；闭包内 client 使用；空池引导文本 | unit + build | `go build ./... && go test ./internal/services/addomain/ -count=1` | ✅ | ⬜ pending |
| 38-02-T2 | 02 | 2 | D-01 Wave1 + D-03 + Pitfall 4 | T-38-04-TestHook, T-38-09-PoolSingleton | ad_sync_tasks.go 暴露 pool 单例字段；dept_sync_tasks 复用全局 pool（NewAccountPool count=0）；测试钩子保留 | unit + build + integration | `go build ./... && go test ./internal/services/addomain/ ./internal/scheduler/ -count=1 && grep -c "NewAccountPool" internal/scheduler/dept_sync_tasks.go` | ✅ | ⬜ pending |
| 38-02-T3 | 02 | 2 | D-01 Wave1 完成硬指标 | T-38-08-Fallback | config.go TestConnection 改 PickFirstConnect；全量 NewLDAPClient(config)=0 | unit + build + grep gate | `go build ./... && grep -rn "NewLDAPClient(config)" internal/ \| grep -v failover_client.go \| wc -l` | ✅ | ⬜ pending |
| 38-03-T1 | 03 | 3 | D-01 Wave2 Go 清理 | T-38-10-ResidualFallback, T-38-11-CryptoFn | 22 处 decryptPassword(config.AdminPassword) 全删；bindAdmin 死代码删除；decryptPassword 函数定义保留 | unit + build + grep gate | `go build ./... && grep -rn "decryptPassword(config.AdminPassword)" internal/ \| wc -l` | ✅ | ⬜ pending |
| 38-03-T2 | 03 | 3 | D-04 + SHA-4 同 commit | T-38-09-FrontBack, T-38-12-BindRequired | 前后端 admin 字段配套删除（同 commit）；service.go/adDomainApi.ts/configs/index.tsx 零残留（W-03） | unit + build + frontend type-check + git log | `go build ./... && cd xingran-react-frontend && npm run type-check && git log --oneline -- {service.go,adDomainApi.ts} \| head -1` | ✅ | ⬜ pending |
| 38-04-T1 | 04 | 4 | D-02 + D-03 启动校验 | T-38-13-Startup, T-38-17-DropColumn | model struct tag 放宽（移除 not null）+ @Deprecated；checkEmptyAccountPoolOnStartup 返回 void（不阻断启动） | unit + build + grep gate | `go build ./... && grep -cE "AdminUsername\s+string\s+\x60[^\x60]*not null" internal/models/ad_domain.go` | ✅ | ⬜ pending |
| 38-04-T2 | 04 | 4 | D-03 migration 幂等 | T-38-14-Idempotent, T-38-16-Constraint | migration_164_phase38_verify_admin_migrated.go 幂等补迁 + 在 database.go 第 401 行后注册（不是 migrate.go） | unit + build + grep gate + manual | `go build ./... && grep -c "Migrate164Phase38VerifyAdminMigrated" internal/core/db/database.go` | ✅ | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] 现有 `TestAccountPoolPasswordRoundTrip` 必须保持绿色（不可破坏）
- [ ] `manager_sync_test.go` 7 个回归测试的 `updateUserAttributeFn` 钩子在 SyncManagersToAD 重构后必须保留
- [ ] `TestFailoverClient` 保持绿色（38-02 改造依赖此契约）
- [ ] `TestDecryptPassword_InvalidCiphertext` 保持绿色（38-03 删除 caller 但保留函数定义）

*现有基础设施已覆盖本 phase 全部回归需求，无需新增测试桩。新增"同步改 FailoverClient 后的回归测试"由 planner + executor 设计。*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| 空账号池时登录返回明确错误 | D-03 | 需真实 AD config + 空池运行时态 | 配置启用的 AD config → 清空 sys_ad_service_accounts → 尝试登录 → 断言返回"请先在 AD 配置页添加服务账号"而非静默失败 |
| 启动空池记 WARN 不阻断 | D-03 | 需应用启动观测 | 空池状态启动后端 → 检查日志有 WARN 且应用正常启动 |
| migration_164 补迁幂等 | D-03（I-02） | 需 DB 有遗留单管理员数据 + 实际启动后端 | 单管理员账号未入池时启动后端 → 确认 migration_164 自动执行入池；二次启动 → 日志出现 "skip" 确认幂等 |

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verify or Wave 0 dependencies
- [x] Sampling continuity: no 3 consecutive tasks without automated verify
- [x] Wave 0 covers all MISSING references（现有 Test* 全覆盖，无 MISSING）
- [x] No watch-mode flags
- [x] Feedback latency < 120s
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
