---
phase: 38
slug: unify-ad-authentication-onto-account-pool-remove-legacy-sing
status: draft
nyquist_compliant: false
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
- `internal/services/addomain/manager_sync_test.go`（SyncManagersToAD 7 个回归测试依赖 `updateUserAttributeFn` 钩子，重构必须保留）

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
| 无 NewLDAPClient 单参数（config）生产调用 | `grep -rn "NewLDAPClient(config)" internal/` 或不带 account 的变体 | 0（仅测试 helper 例外） |
| 前端无 admin 输入项 | `grep -rn "adminUsername\|adminPassword" ruoyi-react-frontend/src/pages/ad-domain/configs/ ruoyi-react-frontend/src/lib/` | 0（API 类型定义同步删除） |

---

## Per-Task Verification Map

> Planner 在 PLAN.md 创建后回填本表（Task ID / Plan / Wave / 对应断言）。

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 38-01-xx | 01 | 1 | D-01 Wave1 连接层统一 | — | FailoverClient.ExecuteWithFailover 接管所有 AD 连接 | unit + build | `go build ./... && go test ./internal/services/addomain/` | ✅ | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] 现有 `TestAccountPoolPasswordRoundTrip` 必须保持绿色（不可破坏）
- [ ] `manager_sync_test.go` 7 个回归测试的 `updateUserAttributeFn` 钩子在 SyncManagersToAD 重构后必须保留

*现有基础设施已覆盖本 phase 全部回归需求，无需新增测试桩。新增"同步改 FailoverClient 后的回归测试"由 planner + executor 设计。*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| 空账号池时登录返回明确错误 | D-03 | 需真实 AD config + 空池运行时态 | 配置启用的 AD config → 清空 sys_ad_service_accounts → 尝试登录 → 断言返回"请先在 AD 配置页添加服务账号"而非静默失败 |
| 启动空池记 WARN 不阻断 | D-03 | 需应用启动观测 | 空池状态启动后端 → 检查日志有 WARN 且应用正常启动 |
| migration_162 补迁幂等 | D-03 | 需 DB 有遗留单管理员数据 | 单管理员账号未入池时跑迁移 → 确认入池；再次跑 → 跳过（幂等） |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 120s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
