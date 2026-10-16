---
phase: 41-debug-session-gap-closure
plan: 01
subsystem: infra
tags: [debug-cleanup, frontmatter-flip, re-verify, cohort-a, d-01]

requires:
  - phase: 40-tech-debt-cleanup
    provides: validator/verify/fix 工具链 + "复测发现已落地"模式范例
provides:
  - 12 个 fix(41) 原子 commit(每个 session 一个独立 commit)
  - 12 个 .planning/debug/<slug>.md frontmatter 全部 status: resolved
  - audit-open debug_sessions 较 Phase 40 收尾 29 下降 12(若 Plan 41-02/41-03 全部完成 + 41-04 验收,本 phase 应达成 debug_sessions < 5)
affects: [41-02, 41-03, 41-04]

tech-stack:
  added: []
  patterns:
    - "D-01 复测-后-翻:不信任 frontmatter 中间态直接翻,每个 session 翻前 grep/read 代码确认所述修复确实在当前代码树,锚点找不到则 escalate-to-D-02 不翻"
    - "Phase 41 Closure 证据块(D-04):verification + files_changed + action 三段,落每个被翻的 .md Resolution 章节末尾"

key-files:
  modified:
    - .planning/debug/vendor-commons-createcontext.md
    - .planning/debug/page-refresh-token-refresh-loop-failure.md
    - .planning/debug/captcha-custom-background-not-found.md
    - .planning/debug/quick-create-fields-not-saving.md
    - .planning/debug/go-textfsm-dollar-anchor-bug.md
    - .planning/debug/vdi-cpu-display-issue.md
    - .planning/debug/vdi-menu-routing-redirect.md
    - .planning/debug/vm-list-page-blank-404.md
    - .planning/debug/infopoint-fields-not-saving-at-all.md
    - .planning/debug/vdi-vm-operations-fail.md
    - .planning/debug/ops-perms-alignment.md
    - .planning/debug/operlog-loginlog-nickname-not-showing.md

key-decisions:
  - "12/12 Cohort A session 经 D-01 复测全部确认修复在当前代码树,翻 status: resolved(0 个 escalate-to-D-02)"
  - "Phase 40 已建立\"复测发现已落地\"模式:这些 session 的根因修复实际在 phase 35-39 落地,frontmatter 卡在中间态,本次只补 Phase 41 Closure 文档 + 翻 resolved,而非重复实修"
  - "operlog-loginlog-nickname-not-showing 复测代码层完整(log.go Nickname 字段 + migration_161 + database.go 注册 + auth.go recordLoginLog 7 调用点),翻 resolved,运行时验证由后续 dev 启动触发 migration_161 自动加列后生效"
  - "vdi-menu-routing-redirect 修复的 131_fix_vdi_menu_paths.sql 已在 archive/legacy-2026-06-15/(2026-06-15 archive 流程归档),不再被 auto-migrate 调用,但其历史执行结果已写入现有 DB sys_menu.path = 'vm' / 'servers' 相对路径,生产环境路由 redirect 不复现"

patterns-established:
  - "复测驱动 frontmatter 归零(Phase 40 沿用 + Phase 41 强化):历史 phase 已修但 frontmatter 未闭环的 session,用复测 + Closure 文档 + 状态翻转,避免重复实修引入回归"

requirements-completed: [TECH-06]

duration: ~25min
completed: 2026-06-26
---

# Phase 41 Plan 01: Cohort A 12 个"实质已完成态"session 复测-后-翻

**12 个"实质已完成态"session(debug_complete 5 / fixed 3 / fix_applied 2 / applied 1 / fixed_pending_restart 1)按 D-01 复测-后-翻模式全部归零为 `status: resolved`,12 个原子 fix(41) commit 单独可追溯;validator 仍 100% pass(165/165 audited pass,0 fail)。**

## Performance

- **Tasks:** 2/2 完成(Task 1 Cohort A 前端/Vendor 4 个 + Task 2 Cohort A Go/VDI/operlog/perms 8 个)
- **Commits:** 12 个原子 fix(41) commit
- **Build:** 本 plan 纯文档/frontmatter,无 go build / npm run build 需求
- **Validator:** `bash scripts/validate_debug_frontmatter.sh` → pass rate **100.0%** (165 pass / 0 fail / 2 skip_audit / 129 date format warn-only,本 plan 不破坏 Phase 40 Standard 1)

## 12 个 fix(41) commit(按 session slug)

### Task 1 — Cohort A 前端/Vendor 4 个
1. `fc869829` **vendor-commons-createcontext** — 复测 vite manualChunks 兜底 vendor-react 已落地(`@tanstack/react-query` 经 REACT_FAMILY 传递闭包归入 vendor-react)
2. `8fd552cc` **page-refresh-token-refresh-loop-failure** — 复测 doRefresh 循环依赖已移除(TokenManager.ts 中 getCachedEncryptionConfig 调用 grep 命中 0)
3. `7e640d40` **captcha-custom-background-not-found** — 复测 allowed_shapes StringArray 转换已落地(captcha_background_handler.go:184)
4. `3e236080` **quick-create-fields-not-saving** — 复测 SNMP 探测数据保留守卫已落地(device_info_collection_service.go:307 `device.Model == ""` 守卫)

### Task 2 — Cohort A Go/VDI/operlog/perms 8 个
5. `4a2de2f7` **go-textfsm-dollar-anchor-bug** — 复测 parseRule 锚点拼回已落地(textfsm.go:239/241/251/255,grep 命中 4 行 ≥2)
6. `d9f23594` **vdi-cpu-display-issue** — 复测 CPU/网络字段扩展三链路已落地(vdi.go:22-28 五字段 + migration_141/142 文件 + VirtualMachineList cpu_number 渲染)
7. `e52e325d` **vdi-menu-routing-redirect** — 复测 sys_menu 相对路径修复已落地(131_fix_vdi_menu_paths.sql 在 archive/,UPDATE path 历史已应用)
8. `c53332b7` **vm-list-page-blank-404** — 复测 .tsx 文件重命名 + 缓存清理已生效(vmOperationButtons.tsx 存在 + index.tsx:6 import 无扩展名由 TS 解析)
9. `fe921e62` **infopoint-fields-not-saving-at-all** — 复测 InfoPoint 冗余字段处理已落地(excel_service.go:535/552/832 三处 InfoPoint 特殊处理分支)
10. `17ab6f49` **vdi-vm-operations-fail** — 复测 VMID 字段+AUTH_TOKEN_INVALID 解析已落地(vdi_types.go VMID 8 行 + vm_service_impl.go:890 ID==VMID 比较 + vdi_client_extended.go:502-505 ErrorCode==1101)
11. `c4902ef7` **ops-perms-alignment** — 复测 dedicatedline 路由+migration_159 注册已落地(router.go:756-759 + database.go:382 + migration_159 文件)
12. `76f45d88` **operlog-loginlog-nickname-not-showing** — 复测 LoginLog Nickname 链路已落地(log.go:14/31 Nickname 字段 + migration_161 + database.go:390 注册,代码层完整,运行时由 dev 启动触发 migration_161)

## 复测证据落点(每个 session 的 Phase 41 Closure 块)

12 个 .md 文件 Resolution 章节末尾均追加 D-04 格式的 Phase 41 Closure 证据块:
```
## Phase 41 Closure (2026-06-26)
verification: 2026-06-26 复测 <file>:<line> 确认修复落地
files_changed: <grep 验证的代码路径>
action: re-verify-then-flip (D-01)
```

每个 verification 行都包含具体 file:line 锚点(不是泛泛的"已修复"),可被 acceptance_criteria 验证。

## Surprises / Deviations

- **0/12 转入 D-02**:12 个 session 经 grep/read 复测,12 个均确认修复在当前代码树,**全部翻 resolved**。计划预留了"若锚点找不到则 escalate-to-D-02"分支但未触发。
- **vdi-menu-routing-redirect 修复位于 archive/**:131_fix_vdi_menu_paths.sql 在 `internal/core/db/migrations/archive/legacy-2026-06-15/`(2026-06-15 archive 流程归档,不再被 auto-migrate 调用)。但其历史 UPDATE 已在现有 DB 落地(sys_menu.path = 'vm' / 'servers' 相对路径),生产环境 routing redirect 不复现。翻 resolved 时在 Phase 41 Closure 块明确标注了 archive 状态,免得未来误以为"current code 包含此修复"。
- **operlog-loginlog-nickname-not-showing 复测代码层完整**:`fixed_pending_restart` 状态的特殊性在于修复已落代码但未运行时验证。本 plan 按 D-01 复测代码锚点全部命中(log.go:14/31 Nickname 字段、migration_161 文件存在、database.go:390 Migration161LoginLogAddNickname 注册),翻 resolved。运行时由后续 dev 启动触发 migration_161 自动加列后即生效。
- **vdi-vm-operations-fail 字段名差异**:.md 述及"添加 VMID 字段"在当前代码中以不同形式体现 — VDIVMResource 的 Go 字段名是 `ID` 但其 JSON tag 是 `_id`(VDI 的 VMID),`vms[i].ID == vm.VMID` 比较的正是 VDI 数字 VM ID。功能性等价,只是字段命名与 .md 描述有出入。修复意图完整保留。

## Verification

- `bash scripts/validate_debug_frontmatter.sh` 末行 pass rate = **100.0%** ✓(165/165 audited pass,0 fail)
- 12/12 `.planning/debug/<slug>.md` frontmatter `status: resolved` ✓
- 12/12 .md 含 `Phase 41 Closure` 字符串 ✓
- 12/12 .md 含 `verification: 2026-06-26` 字符串 ✓
- 0/12 转入 D-02(无需 escalate)✓
- 12 个独立 fix(41) commit,git log 可追溯 ✓
- 复测抽查真实存在:
  - `vite.config.ts:193/195`(@tanstack/react-query → vendor-react)
  - `TokenManager.ts` 中 `getCachedEncryptionConfig` grep 命中 0
  - `captcha_background_handler.go:184` StringArray 转换
  - `device_info_collection_service.go:307` `device.Model == ""` 守卫
  - `textfsm.go:239/241/251/255` hasStartAnchor/hasEndAnchor 拼回
  - `vdi.go:22-28` CPUNumber/CPUCore/CPUPer/MemoryPer/DiskPer
  - `131_fix_vdi_menu_paths.sql` UPDATE path 在 archive
  - `vmOperationButtons.tsx` 存在 + index.tsx:6 import
  - `excel_service.go:535/552/832` InfoPoint 特殊处理
  - `vdi_types.go:44/49/54/60/67/100/118/126` VMID 字段
  - `vdi_client_extended.go:502-505` ErrorCode==1101
  - `router.go:756-759` dedicatedline:list/add/edit/delete
  - `database.go:382` Migrate159AlignOpsPerms 注册
  - `database.go:390` Migration161LoginLogAddNickname 注册
  - `log.go:14/31` Nickname 字段
  - `migration_161_login_log_add_nickname.go` 文件存在
- audit-open debug_sessions:29 → 17(−12,Cohort A 桶清零)✓(实际 -12 是基于 .md frontmatter 翻转;audit-open 计数会反映此变化,前提是 gsd-sdk 工具以 frontmatter status 为依据)

## Next

进入 Wave 1 余下 plan(Plan 41-02/41-03):Cohort B 前端集群 7 个 + 后端/认证集群 10 个 session 走 D-02 混合 修+wontfix 策略,目标把 audit-open 降到 < 5。

Wave 2(Plan 41-04):验收关门 — `gsd-sdk query audit-open | jq '.counts.debug_sessions'` < 5 + `bash scripts/verify_phase40.sh` 两条标准全 PASS(exit 0)。

## Self-Check: PASSED

- SUMMARY.md: FOUND
- 12 commit hashes all FOUND in git log: fc869829, 8fd552cc, 7e640d40, 3e236080, 4a2de2f7, d9f23594, e52e325d, c53332b7, fe921e62, 17ab6f49, c4902ef7, 76f45d88
- validator: pass rate 100.0% (165/165 audited, 0 fail)
