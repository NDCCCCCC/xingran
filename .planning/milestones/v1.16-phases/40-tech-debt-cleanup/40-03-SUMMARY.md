---
phase: 40-tech-debt-cleanup
plan: 03
subsystem: infra
tags: [frontmatter, validator, audit-tooling, tech-debt]

requires:
  - phase: 40-01 + 40-02(已完成)
    provides: 22 个 deferred session 已 resolved,为 audit 计数下降奠基
provides:
  - 6 个 audit 数据质量问题 frontmatter 规范化(D-09/D-13/D-14/D-15)
  - scripts/validate_debug_frontmatter.sh 双模式 validator(D-10/D-11)
  - scripts/verify_phase40.sh 验收脚本(D-16)
  - scripts/fix_debug_frontmatter.py 批量修复工具(Standard 1 扩展)
  - 72 个历史 debug 文件 frontmatter 批量规范化(validator 54.5%→100%)
affects: [audit-open 计数可持续校验, 后续 frontmatter 合规 follow-up]

tech-stack:
  added: []
  patterns: ["audit 工具链分离:validator(规则)+ verify(验收)+ fix(修复)三脚本", "validator 容忍 CRLF + YAML quoted scalar + 全状态枚举"]

key-files:
  created:
    - scripts/validate_debug_frontmatter.sh
    - scripts/verify_phase40.sh
    - scripts/fix_debug_frontmatter.py
  modified:
    - .planning/debug/login-400-bad-request.md(补 frontmatter)
    - .planning/debug/ops-asset-constraint-uni-ops-asset-devicesn-not-exist.md
    - .planning/debug/sys-mac-filter-rules-relation-does-not-exist.md
    - .planning/debug/knowledge-base.md(skip_audit: true)
    - .planning/debug/info-point-type-null-import.md(metadata.* → resolved/ 风格)
    - .planning/debug/apikey-route-path-duplication.md(补 session_type)
    - 72 个历史 debug 文件(批量 slug/字段规范化,见 0b1e5f53)

key-decisions:
  - "Standard 1 批量修复(用户批准):validator 从 54.5% 提到 100%,超出原 6 文件 scope"
  - "validator 枚举扩充:checkpoint_reached/fixed_pending_restart/fixing/investigation_in_progress/complete/applied 是 gsd-debugger 真实生命周期状态,D-11 原枚举不全,扩充接受(非破坏)"
  - "validator 增强:CRLF 容忍(剥 CR)+ YAML quoted scalar 剥引号 + pass rate 分母排除 skip_audit"
  - "Standard 2(debug_sessions<5)保留严格 + 记缺口:29 个 scope 外历史 open session,建议 follow-up phase(其中 ~12 个是 debug_complete/fixed/fix_applied 等实质已完成态,可批量翻 resolved 快速下降)"

patterns-established:
  - "audit 三脚本分离:validate(校验规则,可独立跑)+ verify(Phase 验收,组合标准)+ fix(批量修复,幂等)"
  - "validator 设计:容错(CRLF/quoted/全枚举)优先于严格,严格模式留给 --strict/pre-commit"

requirements-completed: [TECH-04, TECH-05]

duration: ~40min
completed: 2026-06-25
---

# Phase 40 Plan 03: frontmatter 规范化 + 验证脚本 + 批量修复

**6 个 audit 数据质量问题规范化 + 双脚本工具链就位 + 用户批准的批量修复让 validator 从 54.5% 升到 100% pass;Standard 2(debug_sessions<5)因 29 个 scope 外历史 session 未达标,记为已知缺口待 follow-up phase。**

## Performance

- **Tasks:** 3/3(Task 1 batch frontmatter + Task 2 validator + Task 3 verify 脚本)+ Standard 1 批量修复扩展
- **Commits:** 4 个(docs standardize + validator + verify + bulk-standardize)
- **验收:** `bash scripts/verify_phase40.sh` → Standard 1 PASS / Standard 2 FAIL(exit 1)

## Accomplishments

### Task 1: 6 文件 frontmatter 规范化(D-15 batch commit `8a041ab4`)
- login-400-bad-request / ops-asset-constraint / sys-mac-filter-rules:补标准 frontmatter
- knowledge-base:加 `skip_audit: true` 顶层(D-13)
- info-point-type-null:metadata.* 嵌套 → resolved/ 扁平(D-09)
- apikey-route-path-duplication:补 session_type(6 字段齐全)

### Task 2: validate_debug_frontmatter.sh(D-10/D-11,commit `b...`)
双模式:默认 warn-only / `--strict` exit 1。校验状态枚举 + 必填字段(slug/status/trigger/created/updated)+ slug 格式 + 日期格式 + skip_audit 识别 + frontmatter 解析。

### Task 3: verify_phase40.sh(D-16)
两条独立标准:validator 100% pass + audit-open debug_sessions < 5。

### Standard 1 批量修复(用户批准,commit `0b1e5f53`)
首次 validator 揭示真实情况:167 文件 pass 仅 54.5%(75 fail),CONTEXT "130/143 合规"假设失实。用户批准批量修复:
- `scripts/fix_debug_frontmatter.py` 修 72 文件(缺 slug 从文件名推导 / 字段补齐 / messy status 归一)
- validator 增强:CRLF 容忍 + quoted scalar 剥引号 + 枚举扩充 + pass rate 排除 skip_audit
- 结果:**167 文件 pass 165 / fail 0 / skip 2,pass rate 100.0%(audited)**

## Verification(verify_phase40.sh)

| 标准 | 结果 | 说明 |
|------|------|------|
| Standard 1: validator 100% pass | ✅ PASS | 165/165 audited pass,2 skip_audit 排除 |
| Standard 2: audit-open debug_sessions < 5 | ❌ FAIL | 29 个 scope 外历史 open session |

**Standard 2 缺口分析(29 个 open session)**:
- investigating: 10 / debug_complete: 5 / fixed: 3 / verifying: 2 / fix_applied: 2 / diagnosed: 2 / root_cause_identified: 1 / checkpoint_reached: 1 / fixed_pending_restart: 1 / applied: 1 / investigation_in_progress: 1
- 跨 AD/VDI/前端 vendor chunk/login/captcha 等模块,均非 v1.16 的 22 个 deferred session
- **~12 个是实质已完成态**(debug_complete/fixed/fix_applied/applied/fixed_pending_restart),follow-up phase 可批量翻 resolved 快速下降

## Surprises / Deviations

- **CONTEXT "130/143 合规"严重失实**:真实 pass rate 54.5%。validator 是真相之源,揭示 planning 假设错误。
- **批量修复超原 6 文件 scope**:用户批准后扩展到 72 文件,达成 Standard 1。
- **D-11 状态枚举不全**:checkpoint_reached/fixed_pending_restart/fixing/investigation_in_progress/complete/applied 是 gsd-debugger 真实状态,扩充接受(非破坏)而非强行归一文件。
- **CRLF 陷阱**:Python 脚本 Windows 文本模式写文件引入 CRLF,致 validator 误判;修复为 validator CRLF 容忍 + 脚本 `newline='\n'`。
- **Standard 2 不可达**:29 个 open session 中仅 ~12 个可快速翻 resolved,其余 ~17 个是真正在途调查,需独立 phase。

## Next

Phase 40 三 plan 全部完成。建议:
1. **follow-up phase(Phase 41 候选)**:批量翻 ~12 个"实质已完成态"session(debug_complete/fixed/fix_applied)→ resolved,debug_sessions 可从 29 降到 ~17。
2. 余下 ~17 个真正在途 session(investigating/verifying/diagnosed)按优先级逐个闭环。
3. `bash scripts/verify_phase40.sh` 可持续用于回归校验。
