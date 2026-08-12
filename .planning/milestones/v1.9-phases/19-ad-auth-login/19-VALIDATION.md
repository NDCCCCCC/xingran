---
phase: 19
slug: ad-auth-login
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-05-21
---

# Phase 19 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go testing + Vitest |
| **Config file** | `vitest.config.ts` (前端) |
| **Quick run command** | `go test ./internal/core/security/... -v` |
| **Full suite command** | `go test ./... -v` |
| **Estimated runtime** | ~30 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test ./internal/core/security -v`
- **After every plan wave:** Run `go test ./... -v`
- **Before `/gsd-verify-work`:** Full suite must be green
- **Max feedback latency:** 30 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 19-01-01 | 01 | 1 | AUTH-01 | T-19-01 | 策略模式认证系统正确选择认证器 | unit | `go test ./internal/core/security -run TestAuthStrategy` | ❌ W0 | ⬜ pending |
| 19-01-02 | 01 | 1 | AUTH-02 | T-19-02 | AD认证器LDAP连接安全 | integration | `go test ./internal/core/security -run TestADAuthenticator` | ❌ W0 | ⬜ pending |
| 19-02-01 | 02 | 2 | AUTH-03 | T-19-03 | 初次登录用户同步到sys_user | integration | `go test ./internal/services/system -run TestUserSync` | ❌ W0 | ⬜ pending |
| 19-03-01 | 03 | 3 | AUTH-04 | T-19-04 | 前端SM4密码加密正确性 | unit | `npm test src/utils/sm4.test.ts` | ❌ W0 | ⬜ pending |
| 19-04-01 | 04 | 4 | AUTH-05 | T-19-05 | 参数管理AD登录开关生效 | integration | `go test ./internal/api/v1/system -run TestADAuthConfig` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `internal/core/security/authenticator_test.go` — 认证器接口测试
- [ ] `internal/core/security/local_authenticator_test.go` — 本地认证器测试
- [ ] `internal/core/security/ad_authenticator_test.go` — AD认证器测试
- [ ] `internal/core/security/hybrid_authenticator_test.go` — 混合认证器测试
- [ ] `internal/services/system/user_sync_service_test.go` — 用户同步服务测试
- [ ] `xingran-react-frontend/src/utils/sm4.test.ts` — SM4加密工具测试
- [ ] `xingran-react-frontend/src/pages/login/index.test.tsx` — 登录页面测试
- [ ] Framework install: 无需安装（依赖已存在）

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| AD域控实际连接测试 | AUTH-02 | 需要真实AD域控环境 | 1. 配置测试AD服务器 2. 执行登录测试 3. 验证LDAP连接成功 |
| 前端登录界面交互 | AUTH-04 | 需要浏览器环境验证UI交互 | 1. 启动前端开发服务器 2. 选择AD认证模式 3. 验证SM4密码加密流程 |
| 参数管理页面配置 | AUTH-05 | 需要管理员权限访问参数管理 | 1. 登录管理员账号 2. 修改ad_auth_enabled参数 3. 验证登录行为变化 |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 30s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
