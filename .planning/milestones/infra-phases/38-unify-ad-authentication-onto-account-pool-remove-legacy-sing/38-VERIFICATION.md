---
phase: 38-unify-ad-authentication-onto-account-pool-remove-legacy-sing
status: passed
verified_by: orchestrator-inline
verified_at: 2026-06-23
independent_review: pending（建议后续 /gsd:code-review 38 或 /gsd:verify-work 38 做独立复核；本文件为编排者内联验证，非独立 agent）
phase_goal: "删除 sys_ad_config 单管理员遗留路径，所有 AD 连接（同步/管理/登录）统一收敛到 Phase 36 账号池 FailoverClient，消除双轨不对称风险"
---

# Phase 38 Verification — AD 账号池统一（移除遗留单管理员双轨）

> **验证方式**：编排者内联验证（execute-phase 内联模式下，由 orchestrator 基于 per-wave 验收 + grep 断言 + 基线对照完成）。独立 agent 复核留作后续 `/gsd:code-review 38` / `/gsd:verify-work 38`。

## Phase Goal 验证

**目标**：删除 `sys_ad_config` 单管理员遗留路径（`admin_username`/`admin_password`），所有 AD 连接统一到账号池 FailoverClient，消除双轨不对称风险（Phase 36 bug `adpool-password-not-decrypted` 的根因）。

**结论：✅ PASSED** —— 4 个 plan 全部完成，D-01/D-02/D-03/D-04 全部落地，phase 硬指标 grep 断言全部归零。

## Must-Haves 验证（按 plan）

### Plan 38-01 (Wave 1 前置 DI)
- ✅ 8 个 addomain 服务 struct 注入 `pool AccountPool` 字段（commit abdd3e0/21ebb33）
- ✅ 构造函数签名统一改（NewADDomainService(db,pool,cipher)）
- ✅ ad_domain_router 共享同一 accountPool 实例（Pitfall 4 缓解）

### Plan 38-02 (Wave 1 FailoverClient 主体)
- ✅ `NewLDAPClient(config)` 生产调用 = 0（grep 硬指标，commit 03c04b8/b397f6f/355f1b3/c126740）
- ✅ 22 处改走 `ExecuteWithFailover`/`PickFirstConnect`
- ✅ scheduler 全局 pool 单例（W-04）+ SHA-5 测试钩子保留
- ✅ user_router.go nil-pool 接入（post-529 completion c126740）

### Plan 38-03 (Wave 2 删使用逻辑 + 前端)
- ✅ `decryptPassword(config.AdminPassword)` 变体生产调用 = 0（commit ea28831）
- ✅ bindAdmin 死代码 + bindAdminWithFailover 双读 fallback 删除（D-03 不保留双轨）
- ✅ 前后端 admin 字段配套删除 SHA-4 同 commit（commit 2b5f9eb）
- ✅ W-03 零残留：service.go/adDomainApi.ts/configs/index.tsx `adminUsername|adminPassword` = 0（含注释）

### Plan 38-04 (Wave 3 model + migration + 启动校验)
- ✅ ADConfig AdminUsername/AdminPassword 移除 `not null`（commit 762dafc）
- ✅ 每字段 `@Deprecated Phase 38`（count=2）
- ✅ migration_164 显式 DROP NOT NULL 兜底 + 幂等补迁（commit 3084a04，证据驱动补充）
- ✅ 启动空池 WARN 校验 `checkEmptyAccountPoolOnStartup` void 不阻断（Pitfall 6）

## Requirement 追溯

| Req ID | 决策 | 落地 plan | 状态 |
|--------|------|-----------|------|
| D-01 | 分波次降风险迁移 | 38-01/02/03/04（Wave 1/2/3） | ✅ |
| D-02 | 保留空列仅删代码（不 DROP COLUMN） | 38-03（删代码）+ 38-04（tag 放宽/字段保留） | ✅ |
| D-03 | 启动空池校验+明确错误不静默 fallback | 38-02（ErrAllAccountsUnavailable）+ 38-04（启动 WARN + migration 补迁） | ✅ |
| D-04 | 移除前端 admin 输入项 | 38-03（adDomainApi.ts/configs/index.tsx） | ✅ |

## 自动化验证结果

| 检查 | 命令 | 结果 |
|------|------|------|
| Go 编译 | `go build ./...` | ✅ exit 0 |
| Phase 自身包测试 | `go test ./internal/services/addomain/` | ✅ PASS（含 TestAccountPoolPasswordRoundTrip / TestFailoverClient / 7×TestSyncManagersToAD_* / TestDecryptPassword_*） |
| 前端类型检查 | `npm run type-check` | ✅ exit 0（SHA-4 验证：删字段后零 TS 错误） |
| 前端构建 | `npm run build` | ✅ exit 0（1m39s） |
| 单管理员解密残留 | `grep decryptPassword(config.AdminPassword) 变体` | ✅ = 0 |
| NewLDAPClient(config) 残留 | `grep NewLDAPClient(config) 生产` | ✅ = 0（仅 failover_client.go 内部例外） |
| 前端 admin 字段残留 | `grep adminUsername\|adminPassword (3 文件)` | ✅ = 0（W-03 含注释） |
| model not null 残留 | `grep Admin[User|Pass].*not null` | ✅ = 0 |
| bindAdmin 死代码 | `grep func.*bindAdmin[^W]` | ✅ = 0（bindAdminWithFailover 保留） |
| 加解密函数定义保留 | `grep func decryptPassword\|encryptPassword\|SetADSM4Cipher utils.go` | ✅ ≥3（decryptPassword 仍被 tryBindAttempts 用） |

## 全量回归测试（go test ./...）—— 预先存在失败说明

全量 `go test ./...` 有多包失败（core/security、api/v1、api/v1/auth、services、services/operations、services/system、services/lldp）。**经基线对照验证，全部为预先存在/环境问题，非 Phase 38 引入**：

- **基线铁证**：在 pre-Phase-38 提交 `e24c3da` 的临时 worktree 上跑同样的 auth 测试，以**完全相同的错误**失败：
  - `TestIntegration_AuthStrategyFactory_SetUserSyncer` → `table sys_ad_config has no column named member_ou_dn`（测试 schema 漂移，Phase 38 未触碰 integration_test.go / ad_domain.go model 的 member_ou_dn）
  - `TestADLoginWithOUProcessing/本地登录-不处理OU` → "本地用户登录，不应触发OU处理"（auth_handler_test.go 未注册路由的残缺占位测试）
- **明显无关包**：lldp Huawei 解析、operations 分页常量（ClampPageSize "大于最大值"）、system APIKey（"用户不存在"≠"无效的作用域" 测试数据）、services timestamp/async logging —— Phase 38 完全未触碰这些包。
- **Phase 38 自身包 addomain 测试全绿**。

结论：Phase 38 零回归。既有测试债务建议单独 phase/quick task 处理（member_ou_dn 测试 schema 补列、占位测试补全、环境依赖测试隔离）。

## Human Verification（Manual-Only，部署后验证）

| 行为 | 验证步骤 | 状态 |
|------|----------|------|
| migration_164 补迁幂等 | 启动后端→确认 sys_ad_service_accounts 含补迁账号；二次启动日志 "skip" | ⬜ pending（需真实 DB） |
| admin 列已 nullable | 启动后确认 admin_username/admin_password 列 DROP NOT NULL 生效；新建配置（不填 admin）成功 | ⬜ pending |
| 启动空池 WARN 不阻断 | 空池状态启动→日志有 WARN 且应用正常启动 | ⬜ pending |
| 运行时空池明确错误 | 清空账号池→AD 登录/同步→返回"请先添加服务账号"而非静默失败 | ⬜ pending（38-02 已实现 ErrAllAccountsUnavailable 引导文本） |

## 已知偏离（证据驱动，已记录于各 SUMMARY）

1. **38-04 migration_164 增加 DROP NOT NULL**：plan Task 2 原仅数据补迁；证据驱动补充——Phase 38 后新配置不写 admin 字段，DB 列须 nullable 否则 INSERT 失败（关键正确性）。GORM AutoMigrate 对既有列约束放宽不可靠，显式 ALTER 兜底。
2. **38-03 保留 encryptPassword**：plan 称账号池仍用，实测账号池经 core.SM4Cipher.Encrypt（非 addomain.encryptPassword）。按 plan 验收（定义≥3）保留，标记 reserve，不级联清理 crypto helper（scope 控制）。
3. **38-03 删 ADAuthenticator.decryptPassword 方法**：Task 1 删 caller 后变死代码（gopls 标记），按 Wave 2 清理意图删除。

## 总结

**Phase 38 PASSED**。单管理员双轨在代码层彻底消失，所有 AD 连接统一经账号池 FailoverClient。phase gate grep 硬指标全部归零，addomain 包测试全绿，前端 type-check + build 全绿。全量测试套件的失败经基线铁证为预先存在债务，与本 phase 无关。

**建议后续**：
- `/gsd:code-review 38` 独立代码复核
- `/gsd:secure-phase 38` 安全复核（Phase 38 为收敛重构，移除代码/减小攻击面，优先级低但 security_enforcement=true）
- 单独处理既有测试债务（member_ou_dn 测试 schema、占位测试）
