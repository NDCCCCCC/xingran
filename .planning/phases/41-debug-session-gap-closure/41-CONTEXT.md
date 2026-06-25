# Phase 41: Debug Session Gap Closure - Context

**Gathered:** 2026-06-25
**Status:** Ready for planning

<domain>
## Phase Boundary

闭环 Phase 40 遗留的 `audit-open` debug_sessions 缺口。Phase 40 完成了 v1.16 的 22 个 deferred debug session(16 root_cause_found + 5 awaiting_human_verify + 1 apikey)+ 6 个 audit 数据质量文件规范化,并把 validator pass rate 从 54.5% 拉到 100%(Standard 1 PASS)。但 `audit-open` 仍有 **29 个 scope 外历史 open session**,使 Standard 2(debug_sessions < 5)未达标(=29)。

本 phase 把这 29 个降到 < 5,达成 TECH-05 验收,让 v1.16 真正闭环。

**硬约束(沿用 Phase 40 / v1.16):**
- 不新增功能(纯清理 milestone 延续)
- 不修改 `gsd-sdk` 工具
- 不破坏 Phase 40 已落地的工具链(validate_debug_frontmatter.sh / verify_phase40.sh / fix_debug_frontmatter.py)
- 沿用 Phase 40 commit 契约:`fix(41): <slug> — <主题>` 一话一 commit / 每 commit 后 `go build ./...` 或 `npm run build` 退出 0
- 不 push(本地完成,结束统一 push)

</domain>

<decisions>
## Implementation Decisions

### 翻转验证深度 (Area 1)

- **D-01:** **12 个"实质已完成态"session 翻 resolved 前必须复测确认**(用户选"复测确认",Phase 40 风格)
  - 对每个 debug_complete / fixed / fix_applied / applied / fixed_pending_restart session:读其 `.md` 的 Resolution/Fix 章节 → grep/读代码确认所述修复确实在当前代码树 → 在 .md 追加 Phase 41 Closure 复测证据 → 翻 `status: resolved`
  - **不信任状态直接翻**:Phase 40 已证明历史 session 状态与代码可能不一致(12/17 复测发现已落地,但也有需实修的)
  - 若复测发现修复未真落地 → 转入"在途调查"流程(D-02)实修,不谎报 resolved

### 在途 session 处理策略 (Area 2)

- **D-02:** **17 个真正在途 session 用混合策略**(用户选"混合:修+wontfix")
  - **能修则修**:读 session 根因 → 实修代码 → `go build`/`npm run build` 通过 → 翻 resolved + Phase 41 Closure 证据
  - **陈年/重复/不适用者标 won't-fix 或 skip_audit**:在 .md Resolution 写明 won't-fix 理由(如"已被 Phase X 覆盖"/"问题无法复现"/"外部依赖问题非本项目范围")→ 翻 `status: resolved` + 加 `skip_audit: true`(若应完全移出 audit)或保留 resolved 但 .md 注明 won't-fix
  - won't-fix 判定必须有书面理由(防 T-41-02 repudiation),不能无理由批量关

### 目标硬度 (Area 3)

- **D-03:** **目标严格 < 5,不放宽 verify_phase40.sh 阈值**(用户隐含:混合策略就是为了达 <5)
  - verify_phase40.sh Standard 2 保持 `< 5`,本 phase 结束必须 exit 0
  - 不走"放宽到 <20"的退路(D-02 混合策略 + won't-fix/skip_audit 足以达 <5)

### 证据与工具链 (Area 4)

- **D-04:** **每个被处理的 session 在其 `.md` Resolution 章节追加 Phase 41 Closure 证据**(沿用 Phase 40 D-07)
  - 复测型:`verification: 2026-06-25 复测 <file:line> 确认修复落地` + `files_changed`
  - 实修型:`fix: ...` + `verification: go build/npm run build 退出 0` + `files_changed`
  - won't-fix 型:`won't_fix_reason: ...` + `status: resolved`(+ `skip_audit: true` 若移出 audit)

- **D-05:** **沿用 Phase 40 工具链,不新建脚本**
  - `scripts/validate_debug_frontmatter.sh` — 翻转/wontfix 后跑,确认仍 100% pass
  - `scripts/verify_phase40.sh` — phase 验收(Standard 1 + Standard 2 全 PASS)
  - `scripts/fix_debug_frontmatter.py` — 若 won't-fix 加 skip_audit 后 frontmatter 需校验,复用此脚本思路(但 won't-fix 是逐个判断,不批量)

### 提交粒度 (Area 5)

- **D-06:** **沿用一话一 commit**(`fix(41): <slug> — <主题>` / won't-fix 用 `docs(41): <slug> — won't-fix <理由>`)
  - 复测型 + 实修型:1 commit 改 .md(+ 代码若实修)
  - won't-fix 型:1 commit 改 .md(加 won't_fix_reason + status/skip_audit)
  - 不批量多 session 一个 commit(强可追溯,沿用 Phase 40 D-01)

### Claude's Discretion

下列由 Claude 在 discussion 中推荐,用户接受:
- D-01: 复测确认(非信任状态)
- D-02: 混合 修+wontfix
- D-03: 严格 <5 不放宽
- D-04: Phase 41 Closure 证据
- D-05: 沿用 Phase 40 工具链
- D-06: 一话一 commit

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents (researcher / planner) MUST read these before planning or implementing.**

### Phase 40 产出(本 phase 直接依赖)
- `.planning/phases/40-tech-debt-cleanup/40-CONTEXT.md` — Phase 40 锁定决策 D-01..D-16(commit 契约 / 验证深度 / 工具链设计)
- `.planning/phases/40-tech-debt-cleanup/40-01-SUMMARY.md` — "复测发现已落地"模式范例(12/17 session 复测型)
- `.planning/phases/40-tech-debt-cleanup/40-03-SUMMARY.md` — validator/verify/fix 工具链 + 批量修复
- `scripts/validate_debug_frontmatter.sh` / `scripts/verify_phase40.sh` / `scripts/fix_debug_frontmatter.py`

### 29 个 open session 源文件(在 `.planning/debug/`,全部 frontmatter 已合规但 status 非 resolved)

**12 个"实质已完成态"(D-01 复测后翻 resolved):**

| 状态 | slug |
|------|------|
| debug_complete (5) | go-textfsm-dollar-anchor-bug / quick-create-fields-not-saving / vdi-cpu-display-issue / vdi-menu-routing-redirect / vm-list-page-blank-404 |
| fixed (3) | infopoint-fields-not-saving-at-all / vdi-vm-operations-fail / vendor-commons-createcontext |
| fix_applied (2) | captcha-custom-background-not-found / page-refresh-token-refresh-loop-failure |
| applied (1) | ops-perms-alignment |
| fixed_pending_restart (1) | operlog-loginlog-nickname-not-showing |

**17 个真正在途(D-02 混合 修+wontfix):**

| 状态 | slug |
|------|------|
| investigating (10) | ad-admin-lockout-recurrence / device-modal-serial-auto-match / frontend-build-66-ts-errors / hardcoded-blue-theme-bypass / login-md-vendor-crash / login-wrong-pwd-no-prompt / react-vendor-activity-tdz / request-encryption-token-refresh-400 / sidebar-apikey-menu-no-response / vdi-sync-invalid-uri |
| verifying (2) | ad-update-attr-no-such-object / ad-user-page-checkbox-not-click |
| diagnosed (2) | interface-to-any-conversion-hint / user-deptid-uuid-cast-recurring |
| root_cause_identified (1) | css-tree-vendor-chunk-tdz |
| checkpoint_reached (1) | file-upload-401-refresh |
| investigation_in_progress (1) | vdi-server-test-500-input-warnings |

### 项目规范
- `CLAUDE.md` — "Build check before commit" + "Always ask user before committing" + 操作日志记录约定
- `.planning/debug/resolved/asset-export-404-error.md` — resolved 风格 frontmatter 范本

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- **Phase 40 三脚本工具链** — validator / verify / fix 已就位,本 phase 直接复用验收
- **"复测发现已落地"模式**(Phase 40 40-01) — 读 session Resolution → grep 代码确认修复在 → .md 写 Closure → 翻 resolved。本 phase 12 个"已完成态"走同款
- **skip_audit: true 顶层字段**(Phase 40 D-13) — knowledge-base 已用;本 phase won't-fix 且需移出 audit 的 session 沿用

### Established Patterns
- commit 格式:`fix(<phase>): <slug> — <主题>`(Phase 40 风格,本 phase 用 41)
- frontmatter 范式:`slug/status/trigger/created/updated/session_type`(Phase 40 已全量合规)
- 验证证据落 .md Resolution 章节(Phase 40 D-07)

### Integration Points
- `gsd-sdk query audit-open` — 本 phase 验收时输出 `debug_sessions < 5`
- `bash scripts/verify_phase40.sh` — phase 结束跑,两条标准全 PASS(exit 0)

</code_context>

<threat_model>
## Trust Boundaries

| Boundary | Description |
|----------|-------------|
| dev 终端 → git 仓库 | ~29 个 fix(41)/docs(41) commit,本地可回滚 |
| .md 状态 → audit 计数 | won't-fix/skip_audit 决定 audit-open 是否计某 session,误标会掩盖真问题 |

## STRIDE 威胁(摘要)

| Threat ID | Category | Component | Disposition | Mitigation |
|-----------|----------|-----------|-------------|------------|
| T-41-01 | Tampering | 批量翻 resolved 掩盖未真修 | mitigate | D-01 复测确认;不复测不翻 |
| T-41-02 | Repudiation | won't-fix 无理由关单 | mitigate | D-02 won't-fix 必须书面理由(被 Phase X 覆盖/无法复现/外部依赖等) |
| T-41-03 | Tampering | 实修引入回归 | mitigate | 每 commit 后 go build/npm run build;沿用 CLAUDE.md operlog 约定 |

</threat_model>

<deferred>
## Deferred Ideas

| 想法 | 原因 |
|------|------|
| 给所有 debug session 加 created/updated 日期格式化(消除 129 个 date WARN) | date 格式 warn-only 不阻断 Standard 1,非本 phase 目标 |
| 把 .planning/debug/resolved/ 与 .planning/debug/ 目录结构统一 | 目录重组属新范畴,可能影响 gsd-debugger 检索路径 |
| 写 E2E 覆盖 17 个在途 session 的修复 | 沿用 Phase 40 决策,新框架属后续 milestone |

</deferred>

---

*Phase: 41-Debug Session Gap Closure*
*Context gathered: 2026-06-25*
*Depends on: Phase 40 (shipped 3/3, debug_sessions=29 缺口待本 phase 闭环)*
