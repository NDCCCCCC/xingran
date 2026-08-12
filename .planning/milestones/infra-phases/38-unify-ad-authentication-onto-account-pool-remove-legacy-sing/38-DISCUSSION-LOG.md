# Phase 38: AD 账号池统一（移除遗留单管理员双轨）- Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-06-23
**Phase:** 38-unify-ad-authentication-onto-account-pool-remove-legacy-sing
**Areas discussed:** 执行策略, 字段去留, 数据迁移校验, 前端配置页

---

## 执行策略

| Option | Description | Selected |
|--------|-------------|----------|
| 分波次降风险 | Wave1 连接层统一（NewLDAPClient→FailoverClient，保留单管理员作回退）→ Wave2 移除使用代码+前端 → Wave3 字段清理+migration。每波可独立 build/test/回滚 | ✓ |
| 一次性全量 | 单次提交完成所有改造。最快但回滚粒度粗，某处遗漏全面受影响 | |
| 你决定 | 由 planner 根据代码依赖图决定最优波次划分 | |

**User's choice:** 分波次降风险
**Notes:** ~15 处调用点 + 字段 + 前端 + migration，分波次可独立验证与回滚，风险最低。

---

## 字段去留

| Option | Description | Selected |
|--------|-------------|----------|
| 保留空列仅删代码 | 删 Go 使用逻辑（model struct tag 可移除），DB 列保留为空。避免 PostgreSQL DROP COLUMN 破坏性 DDL，回滚仅需恢复代码 | ✓ |
| DROP COLUMN 物理删 | migration DROP COLUMN 彻底移除。DB 最干净但破坏性 DDL 不可逆，升级失败难回滚 | |
| 你决定 | 由 planner 权衡（生产 DB 通常倾向保留空列） | |

**User's choice:** 保留空列仅删代码
**Notes:** Phase 36 原话"1 版本后清理"未明确物理删 vs 逻辑删；选保留空列符合 @Deprecated 渐进语义，且规避 PG 锁表/重写风险。

---

## 数据迁移校验

| Option | Description | Selected |
|--------|-------------|----------|
| 校验+告警+明确错误 | 启动空池记 WARN 不阻断；登录/同步遇空池返回明确错误引导配置；migration_162 补迁保证现有账号已入池；不静默 fallback | ✓ |
| 硬失败阻断启动 | 启用的 AD config 账号池为空时启动即失败。最严格但阻塞新环境首次部署 | |
| 保留单管理员 fallback | 账号池为空时回退读单管理员。与"删代码"冲突，重新引入双轨复杂度 | |

**User's choice:** 校验+告警+明确错误
**Notes:** 删单管理员代码后无 fallback，不能让登录静默失败；明确错误引导配置优于硬阻断（新环境首次部署未配 AD 不应被阻塞）。

---

## 前端配置页

| Option | Description | Selected |
|--------|-------------|----------|
| 移除输入项 | AD 配置表单移除 admin_username/admin_password 输入框，账号管理收敛到 AccountPoolTab。后端删代码与前端移除最一致 | ✓ |
| 保留只读显示 | 输入框改只读灰显展示历史配置。过渡期友好但留 UI 冗余 | |
| 你决定 | 由 planner/UI 决定 | |

**User's choice:** 移除输入项
**Notes:** 账号池 tab 已独立于 AccountPoolTab.tsx；移除输入项避免用户误填已废弃字段，与后端删代码一致。

---

## Claude's Discretion

- Wave 内部具体波次划分顺序、`operation` 闭包封装风格、model struct tag 清理粒度——planner 决定。
- 测试策略：复用 `TestAccountPoolPasswordRoundTrip`，新增同步改 FailoverClient 后的回归测试由 planner + executor 设计。
- operlog 记录保持不变（账号池 CRUD 已覆盖）。
- 配置测试连接（`config.go:208`）改用账号池首个可用账号，具体由 planner 决定。

## Deferred Ideas

- DROP COLUMN 彻底清理（确认无回滚需求后的新 phase）。
- 同步任务 failover 上报粒度精细化（账号失败 vs 单用户操作失败）。
- 账号池为空的前端引导式 UX 增强。
