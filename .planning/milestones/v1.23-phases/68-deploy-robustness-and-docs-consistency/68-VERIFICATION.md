---
phase: 68-deploy-robustness-and-docs-consistency
verified: 2026-08-20T00:12:00Z (closeout verification, generated as part of v1.22-v1.25 milestone audit follow-up)
status: passed
score: 5/5 success criteria verified
overrides_applied: 0
overrides: []
re_verification:
  previous_status: null
  previous_score: null
  gaps_closed: []
  gaps_remaining: []
  regressions: []
gaps: []
deferred:
  - "configs/config.yaml (live, user-managed) 与 .env 仍含 XINGRAN_JWT_SM2_* 字面量 — DEPLOY-01 闭环 6 个 *.example.yaml, 此 2 处由运维手工迁移到 secrets.env 注入路径, 不入 phase 范围 (ROADMAP Phase 68 备注)"
human_verification: []
---

# Phase 68: 部署稳健性 & 文档一致性（SM2 密钥配置闭环）Verification Report

**Phase Goal:** 闭环 5 项与 SM2 密钥配置相关的部署稳健性与文档一致性缺陷 —— 让 `public-key-500-after-subpath-fix.md` Specialist Review 列出的 4 项 related bugs + 1 项 sqlite 模板漂移一次性消除,避免再次发生"SM2 配置链路 → 500 + ssh grep"生产事件;纯部署/文档稳健性修复,非新功能。

**Verified:** 2026-08-20T00:12:00Z (closeout verification — generated as part of v1.22-v1.25 milestone audit follow-up; original execution 2026-08-19 5 commits a21dcec..25ded8f tracked in 68-01-SUMMARY.md)

**Status:** passed

## Goal Achievement

### Observable Truths (from ROADMAP SCs)

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | **DEPLOY-01 PASS**: `docs/deployment/{secret-management,single-machine-deployment,docker-compose}.md` + `configs/{config,config.prod,config.sqlite}.example.yaml` 共 17 处 `XINGRAN_JWT_SM2_*` 全部改为 `JWT_SM2_*`,与 `internal/config/config.go:359-360` BindEnv 一致;语义/表头/必填列保留;不动 secrets 注入值;`grep -rn "XINGRAN_JWT_SM2" docs/ configs/` 零命中 | ✓ VERIFIED | commit `a21dcec` (6 files: 3 docs + 3 example yaml); 17 occurrences normalized to JWT_SM2_* |
| 2 | **DEPLOY-02 PASS**: `scripts/deploy/setup-server.sh` 的 `/etc/xingran/secrets.env` heredoc 在 `SM4_KEY=` 后追加 `# === SM2 密钥对（非对称,不能动态生成,否则每次重启踢全部用户） ===` 注释 + `go run scripts/crypto/gen_sm2_keys/main.go` 一次性生成命令说明 + `JWT_SM2_PRIVATE_KEY=` / `JWT_SM2_PUBLIC_KEY=` 两行占位;脚本整体可幂等重跑;**不做** `use_sm2=true` 强制校验(user-managed config 保守处理) | ✓ VERIFIED | commit `a764825`; secrets.env heredoc grows 6 lines (4 comments + 2 placeholders); idempotent re-run preserved |
| 3 | **DEPLOY-03 PASS**: `scripts/crypto/gen_sm2_keys/main.go` line 2 注释改为 `go run scripts/crypto/gen_sm2_keys/main.go`,与文件实际路径一致;`//go:build ignore` 标签保留;docs 中 `scripts/gen-sm2-keys*` 历史路径同步修正 | ✓ VERIFIED | commit `65093b9`; T1 also fixed 2 docs-side references |
| 4 | **DEPLOY-04 PASS**: `internal/api/v1/auth.go` `getPublicKey` handler 在 `response.Error` 返回 500 之前打印 `applogger.Warnf("SM2 公钥不可用: useSM2=%v, sm2PublicKeyLoaded=%v, requestPath=%s, clientIP=%s", ...)`;`internal/core/security/jwt.go` 新增 `IsSM2Enabled()` 与 `HasSM2PublicKey()` 两个最小 getter;`go build ./...` + `go vet` 退出码 0;handler HTTP status / 响应文案零变化 | ✓ VERIFIED | commit `52685fd`; minimal getter pair added; existing GetPublicKey() body unchanged |
| 5 | **DEPLOY-05 PASS**: `configs/config.sqlite.example.yaml` line 58 `use_sm2: false` → `true`;line 60 env 名同步修正;文件顶部注释段新增【启用 SM2 必读】6 行迁移指引指向 `docs/deployment/secret-management.md §2.2` | ✓ VERIFIED | commit `25ded8f`; sqlite template aligned to prod (use_sm2: true); 6-line migration note pointing to secret-management §2.2 |

**Score:** 5/5 success criteria verified (DEPLOY-01..05 全部 PASS)

## Commit Hash Table

| Task | Name | Commit | Files |
| ---- | ---- | ------ | ----- |
| T1   | DEPLOY-01 — env var names alignment | `a21dcec` | 6 files (3 docs + 3 example yaml) |
| T2   | DEPLOY-02 — secrets.env SM2 segment | `a764825` | `scripts/deploy/setup-server.sh` |
| T3   | DEPLOY-03 — gen_sm2_keys header path | `65093b9` | `scripts/crypto/gen_sm2_keys/main.go` |
| T4   | DEPLOY-04 — getPublicKey observability | `52685fd` | `internal/api/v1/auth.go` + `internal/core/security/jwt.go` |
| T5   | DEPLOY-05 — sqlite use_sm2 default | `25ded8f` | `configs/config.sqlite.example.yaml` |

## Quality Gates

| Gate | Result | Notes |
|------|--------|-------|
| `go build ./...` | ✓ exit 0 | 后端无 schema 变化 |
| `go vet ./...` | ✓ exit 0 | |
| `grep -rn "XINGRAN_JWT_SM2" docs/ configs/` | ✓ 0 hits | DEPLOY-01 闭环确认（*.example.yaml + docs） |
| secrets.env heredoc 幂等重跑 | ✓ preserved | setup-server.sh 可多次运行不破坏 |

## Deferred / 范围外

**`configs/config.yaml`（live, user-managed）与 `.env` 仍含 `XINGRAN_JWT_SM2_*` 字面量**

- `grep XINGRAN_JWT configs/*.example.yaml docs/` → 0 hits（DEPLOY-01 闭环）
- `grep XINGRAN_JWT configs/config.yaml .env` → 2 hits（live user-managed，超出 Phase 68 范围）
- ROADMAP Phase 68 备注：DEPLOY-01 完成的 6 个示例 yaml 与 docs 全部一致；live 配置由运维手工迁移到 secrets.env 注入路径，**不入 phase 范围**
- 影响：zero — DEPLOY-01 完成的 6 个示例 yaml 与 BindEnv 一致；live 配置由运维职责

## Source Verification

| Source | Status | Reference |
|--------|--------|-----------|
| 68-01-SUMMARY.md | ✓ exists · 5/5 DEPLOY-XX PASS | `.planning/phases/68-deploy-robustness-and-docs-consistency/68-01-SUMMARY.md` |
| 68-01-PLAN.md | ✓ exists | `.planning/phases/68-deploy-robustness-and-docs-consistency/68-01-PLAN.md` |
| 5 git commits | ✓ verified | `a21dcec..25ded8f` (commits 2ca52f6 audit 之前 5 commit) |
| ROADMAP phase 68 段 | ✓ 标记 EXECUTED | 5 SC 全部已 PASS, completion date 2026-08-19 |

## Cross-Phase Integration Check

Phase 68 与 v1.22-25 启动块接线（per integration-checker report）：
- **vs Phase 69 (DICT)**：Phase 68 改 docs/configs/scripts（部署层）；Phase 69 改 sys_dict（业务层）；schema 无交集
- **vs Phase 70 (settings)**：Phase 68 SM2 公钥加载（auth.go）独立路由；Phase 70 settings 重构不触 auth；零 c.Cache 共享冲突
- **vs Phase 64-67 (v1.22 品牌化)**：Phase 68 6 个 yaml 改动 + Phase 64 令牌层 + Phase 70 设置 UI 三者无 token 引用关系；视觉与配置完全解耦

---

*Phase 68 closeout verification generated 2026-08-20 as part of v1.22-v1.25 milestone audit (audit commit 2ca52f6) follow-up.*
*Original execution 2026-08-19 tracked in 68-01-SUMMARY.md.*
*This VERIFICATION.md restores the missing formal verification document to bring Phase 68 into compliance with the closeout pattern established by Phases 64-67/69/70.*