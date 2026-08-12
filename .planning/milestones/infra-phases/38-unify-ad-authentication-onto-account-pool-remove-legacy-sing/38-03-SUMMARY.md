---
phase: 38-unify-ad-authentication-onto-account-pool-remove-legacy-sing
plan: 03
subsystem: infra
tags: [addomain, account-pool, single-admin-removal, frontend, binding, refactor, wave-2]

# Dependency graph
requires:
  - phase: 38-02 (Wave 1 FailoverClient 主体改造)
    provides: 所有 AD 连接经账号池 FailoverClient；decryptPassword(config.AdminPassword) 已是死调用
provides:
  - "单管理员密码 Go 使用逻辑全部删除 (decryptPassword(config.AdminPassword) grep=0)"
  - "ad_authenticator bindAdmin @Deprecated 死代码 + bindAdminWithFailover 双读 fallback 分支删除 (D-03 不保留双轨)"
  - "前后端 admin 字段配套删除 (SHA-4 同 commit)：service.go binding:required + adDomainApi.ts 三接口 + configs/index.tsx 表单 (D-04)"
affects:
  - 38-04-model-migration-cleanup (Wave 3：model struct tag 放宽 + migration_164 补迁 + 启动空池校验)

# Tech tracking
tech-stack:
  added: []
  removed:
    - "ADAuthenticator.decryptPassword 方法 (单管理员密码解密，Task 1 后无调用方)"
    - "ADAuthenticator.bindAdmin @Deprecated 方法 + bindAdminWithFailover accountPool==nil fallback 分支"
patterns:
  - "单管理员使用点删除模式：sed 批量删 *.AdminPassword = ...decryptPassword( 行 + Edit 结构性收尾"
  - "SHA-4 前后端配套删除：request struct 字段 + binding:required + 前端 TS interface + 表单逻辑同 commit"

key-files:
  created: []
  modified:
    - internal/core/security/ad_authenticator.go (删 decryptPassword 方法 + bindAdmin 死代码 + fallback 分支)
    - internal/services/addomain/sync.go (删 decryptPassword 调用)
    - internal/services/addomain/user.go (删 4 处)
    - internal/services/addomain/group.go (删 3 处)
    - internal/services/addomain/group_sync_service.go (删 2 处)
    - internal/services/addomain/group_management_service.go (删 4 处)
    - internal/services/addomain/dept_sync_service.go (删 1 处)
    - internal/services/addomain/user_ad_sync_service.go (删 3 处)
    - internal/services/addomain/config.go (删 TestConnection decryptPassword + CreateRequest/UpdateRequest admin 字段 + Create/Update 赋值/加密)
    - internal/scheduler/dept_sync_tasks.go (删 2 处 addomain.DecryptPassword)
    - internal/services/addomain/service.go (ADConfigCreateRequest/UpdateRequest 删 admin 字段 + 透传删除)
    - internal/api/v1/system/ad_domain_handler.go (CreateConfig/GetConfig 响应清空 AdminUsername)
    - xingran-react-frontend/src/lib/adDomainApi.ts (三接口删 admin 字段)
    - xingran-react-frontend/src/pages/ad-domain/configs/index.tsx (handleEdit/handleModalOpenChange/handleSubmit 简化)

key-decisions:
  - "bindAdminWithFailover 删除 nil fallback 后改防御性错误返回 (accountPool 未注入时返回明确错误，不 nil-deref panic 也不回退双轨)"
  - "保留 addomain decryptPassword/encryptPassword/encryptPasswordLegacyAES 函数定义 (plan 验收 ≥3；decryptPassword 仍被 tryBindAttempts 用)"
  - "encryptPassword 现已无生产调用方 (账号池经 core.SM4Cipher.Encrypt 加密) —— 按 plan 保留定义，标记 reserve，不级联清理 crypto helper (避免 scope 蔓延到 auth_strategy_factory 注入链)"

requirements-completed: [D-01, D-04]

# Metrics
duration: 25min
completed: 2026-06-23
---

# Phase 38 Plan 03: Wave 2（删单管理员使用逻辑 + 前端 admin 字段）Summary

**删除单管理员密码的全部 Go 使用逻辑（22 处 decryptPassword 调用）+ ad_authenticator bindAdmin 死代码 + 前后端配套删除 admin 字段（SHA-4 同 commit）。完成 D-01 Wave 2 + D-04，单管理员双轨在代码层彻底消失。**

## Performance

- **Duration:** ~25 min（内联执行，绕开模型网关 529 限流）
- **Tasks:** 2（Task 1 Go 清理 + Task 2 前后端 SHA-4 配套，各独立 commit）
- **Files modified:** 14（10 Go + 2 前端 + 2 跨层）

## Accomplishments

- **22 处 `decryptPassword(config/adConfig.AdminPassword)` Go 调用全删**（grep 硬指标 = 0，含 adConfig / a. / addomain.DecryptPassword 变体）
- **ad_authenticator.go 清理**：删 `@Deprecated bindAdmin` 死代码方法（Wave 1 后无 caller）+ `bindAdminWithFailover` 的 `accountPool==nil` Phase 36 双读 fallback 分支（D-03 不保留双轨，改防御性错误返回避免 nil panic）+ 现已死亡的 `ADAuthenticator.decryptPassword` 方法 + 陈旧注释
- **config.go TestConnection** 残留 decryptPassword 删除 + 解密相关陈旧注释清理
- **前后端 admin 字段配套删除（SHA-4 同 commit `2b5f9eb`）**：service.go `ADConfigCreateRequest`/`ADConfigUpdateRequest` 删 `AdminUsername`/`AdminPassword`（含 `binding:"required"`）+ config.go CreateRequest/UpdateRequest + Create/Update 赋值/加密 + handler 响应清空 AdminUsername + adDomainApi.ts 三接口 + configs/index.tsx 表单
- **W-03 零残留**：service.go / adDomainApi.ts / configs/index.tsx 的 `adminUsername|adminPassword` grep = 0（含注释）
- **保留** addomain `decryptPassword`/`encryptPassword`/`SetADSM4Cipher` 函数定义（plan 验收 ≥3；`decryptPassword` 仍被账号池 `tryBindAttempts` 使用）

## Task Commits

1. **Task 1: Go 删除 22 处 decryptPassword + bindAdmin 死代码** — `ea28831` (refactor)
2. **Task 2: 前后端配套删除 admin 字段（SHA-4 同 commit）** — `2b5f9eb` (refactor)

## Decisions Made

- **bindAdminWithFailover 防御性 nil 检查**：删除 fallback 后，若 `a.accountPool == nil` 返回明确错误"AD 账号池未初始化"而非 nil-deref panic。不回退旧 bindAdmin 路径（D-03 不保留双轨）。生产中 core.go initAuthFactory 保证 pool 非空。
- **保留 encryptPassword（plan 偏差，证据驱动）**：plan 称"账号池路径仍用 encryptPassword"，但实测账号池经 `h.core.SM4Cipher.Encrypt`（ad_account_handler:104/148）加密，不经过 `addomain.encryptPassword`。该函数现已 0 调用方。**按 plan 验收（定义保留 ≥3）保留定义**，不级联清理 `encryptPasswordLegacyAES` → `getLegacyAESEncryptKey`（避免 scope 蔓延到 crypto helper / auth_strategy_factory sm4Cipher 注入链）。decrypt 路径完整保留（decryptPassword + decryptPasswordLegacyAES，tryBindAttempts + 历史 AES 密文解密仍用）。

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] ADAuthenticator.decryptPassword 方法变死代码**
- **Found during:** Task 1（删 line 99 + bindAdmin 后，a.decryptPassword 无调用方，gopls unusedfunc 标记）
- **Fix:** 删除 `ADAuthenticator.decryptPassword` 方法（53-70 行，单管理员密码解密逻辑，正是 Wave 2 清理目标）
- **Files modified:** internal/core/security/ad_authenticator.go
- **Committed in:** ea28831（Task 1）

**2. [Rule 3 - Documentation] config.go TestConnection 陈旧注释**
- **Found during:** Task 1（sed 删 decryptPassword 行后，"decryptPassword 行暂保留"注释孤立）
- **Fix:** 清理 config.go TestConnection 解密相关陈旧注释块
- **Committed in:** ea28831（Task 1）

**3. [Documentation] configs/index.tsx 陈旧 fallback 注释**
- **Found during:** Task 2（注释称"admin 字段作为账号池为空时的 fallback"，但 D-03 已删 fallback）
- **Fix:** 重写注释，移除 admin 字段提及 + 陈旧 fallback 描述
- **Committed in:** 2b5f9eb（Task 2）

**Total deviations:** 3（2 Rule 3 auto-fix + 1 文档）。无 scope creep。

## Issues Encountered

- **预先存在的测试失败（非本 plan 回归）**：`internal/core/security/integration_test.go::TestIntegration_AuthStrategyFactory_SetUserSyncer` 失败，错误 `table sys_ad_config has no column named member_ou_dn`。经 `git stash` 验证：在 Wave 2 HEAD（c126740，不含本 plan 改动）上同样失败；git forensics 确认 Phase 38 未触碰 `integration_test.go` 与 `ad_domain.go` model，`member_ou_dn` 字段在 Phase 38 之前（e24c3da）就存在于 model 而测试 schema 漏建该列。**纯粹的项目陈债**（某早期 phase 加字段未同步测试 schema），与本 plan 无关，留作既有技术债。
- **encryptPassword 现无调用方**：见 Decisions。按 plan 保留，标记 reserve。

## User Setup Required

None - 纯代码删除 + 前端字段清理，无外部配置。

## Next Phase Readiness

- ✅ D-01 Wave 2 完成：单管理员密码 Go 使用逻辑 grep = 0
- ✅ D-04 完成：前端 admin 输入项移除，账号管理收敛到 AccountPoolTab；前后端零残留
- ✅ 回归守护：addomain 包测试全绿（含 TestAccountPoolPasswordRoundTrip / TestDecryptPassword_InvalidCiphertext / 7 个 TestSyncManagersToAD_*）
- ✅ 前端 type-check + build 全绿（SHA-4 验证：删字段后零 TS 错误）
- ⚠️ 38-04 Wave 3 待办：model struct tag 放宽（AdminUsername/AdminPassword 移除 not null，@Deprecated）+ migration_164 补迁幂等 + 启动空池 WARN 校验（D-03）

## Known Stubs

None - 纯删除清理，无占位。

## Threat Flags

None - 本 plan 无新增端点/认证路径/信任边界。threat_model 全部缓解：
- T-38-09-FrontBack（前后端 400）：SHA-4 同 commit + acceptance grep + CI build 验证 ✓
- T-38-10-ResidualFallback（残留单管理员路径）：grep decryptPassword(config.AdminPassword)=0 + bindAdminWithFailover fallback 删除 ✓
- T-38-11-CryptoFn（误删加解密函数）：acceptance 锁 utils.go 定义 ≥3，decryptPassword 保留 ✓
- T-38-12-BindRequired（binding:required 残留致 400）：SHA-4 配套删除 + grep service.go=0 ✓

---

*Phase: 38-unify-ad-authentication-onto-account-pool-remove-legacy-sing*
*Plan: 03*
*Completed: 2026-06-23*

## Self-Check: PASSED

- ✅ Commit `ea28831` (Task 1) found in git log
- ✅ Commit `2b5f9eb` (Task 2, SHA-4 前后端同 commit) found in git log
- ✅ `git log --oneline -- internal/services/addomain/service.go xingran-react-frontend/src/lib/adDomainApi.ts | head -1` = `2b5f9eb`（前后端同 commit）
- ✅ `grep -rn "decryptPassword(config\.AdminPassword)|..." internal/` = 0
- ✅ `grep -c "func.*bindAdmin[^W]" ad_authenticator.go` = 0；bindAdminWithFailover 保留
- ✅ `grep -c "adminUsername|adminPassword" service.go / adDomainApi.ts / configs/index.tsx` = 0（W-03 含注释）
- ✅ `go build ./...` exits 0
- ✅ `go test ./internal/services/addomain/` PASS
- ✅ `cd xingran-react-frontend && npm run type-check && npm run build` PASS
- ✅ decryptPassword / encryptPassword / SetADSM4Cipher 定义保留（utils.go count ≥3）
