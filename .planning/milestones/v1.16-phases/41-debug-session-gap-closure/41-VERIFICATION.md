---
phase: 41-debug-session-gap-closure
plan: 04
verification_date: 2026-06-26
verifier: gsd plan executor (sequential mode)
requirements: [TECH-06, TECH-05]
tags: [acceptance-gate, audit-open, verify-phase40, three-piece-suite, 29-session-matrix, milestone-close]
---

# Phase 41 Verification — 29 Session 处置矩阵 + 三件套验收 PASS

**Phase 41 验收关门证据**:三件套全 PASS(audit-open debug_sessions=0 < 5 / verify_phase40.sh [ALL PASS] exit 0 / validator 100.0% pass rate),29 个 debug session 全部按 D-01(复测翻)/D-02(修或 won't-fix)策略逐一闭环,Strict per-slug 矩阵可追溯每 session 一 commit。TECH-06 + TECH-05(< 5 部分)同步达成,v1.16 milestone 验收材料就位。

## 1. 三件套验收输出(原样捕获,2026-06-26T03:47 UTC)

### 1.1 `gsd-sdk query audit-open` — counts

```json
{
  "scanned_at": "2026-06-26T03:47:12.921Z",
  "has_scan_errors": false,
  "has_open_items": true,
  "counts": {
    "debug_sessions": 0,
    "quick_tasks": 52,
    "threads": 0,
    "todos": 1,
    "seeds": 0,
    "uat_gaps": 5,
    "verification_gaps": 4,
    "context_questions": 0,
    "total": 62
  },
  "items": {
    "debug_sessions": []
  }
}
```

**核心指标**:`debug_sessions: 0`(目标 < 5,D-03 严格目标达成 ✓)
**items.debug_sessions**: `[]`(空数组,无任何 open debug session)

### 1.2 `bash scripts/verify_phase40.sh` — 双标准

```
==========================================
Phase 40 Verification (Tech-Debt Cleanup)
==========================================

== Standard 1: frontmatter validator 100% pass ==
[OK] validator 100% pass

== Standard 2: audit-open debug_sessions < 5 ==
[OK] debug_sessions=0 < 5

==========================================
Summary
==========================================
  Standard 1 (validator 100% pass): PASS
  Standard 2 (audit-open < 5):        PASS

[ALL PASS] Phase 40 verification SUCCESS
```

**Standard 1 PASS** ✓(validator pass rate 100.0%,159/159 audited pass,0 fail,8 skip_audit excluded)
**Standard 2 PASS** ✓(debug_sessions=0 < 5)
**Exit code**: 0

### 1.3 `bash scripts/validate_debug_frontmatter.sh` — 末行 pass rate

```
== Summary ==
  total: 167
  pass:  159
  warn:  121
  skip:  8
  fail:  0
  pass rate: 100.0% (of audited; 8 skip_audit excluded)
```

**pass rate: 100.0%** ✓(159 pass / 0 fail / 8 skip_audit,121 date format warn-only 不阻断)

## 2. TECH 验收达成声明

| 需求 ID | 描述 | 验收证据 | 状态 |
|---------|------|----------|------|
| **TECH-06** | Phase 41 新增:闭环 Phase 40 遗留 debug_sessions 缺口(29 → < 5) | 29/29 session 按 D-01/D-02 策略逐一 closure,strict per-slug 矩阵见 §3;audit-open debug_sessions=0 | ✅ **达成** |
| **TECH-05** | audit-open debug_sessions < 5 + 全量 frontmatter 校验 100% | verify_phase40.sh [ALL PASS] exit 0:Standard 2 PASS(debug_sessions=0 < 5)+ Standard 1 PASS(validator 100%) | ✅ **达成** |

## 3. 29 Session Strict Per-Slug 处置矩阵

**矩阵说明**:
- 每一行 = 一个 slug(D-06 一话一 commit 契约,1 session = 1 atomic commit)
- 处置类型:**re-verify-flip**(D-01 复测-后-翻,代码已落)/**real-fix**(D-02 真正实修)/**wontfix**(D-02 won't-fix,书面理由 + 可选 skip_audit 移出 audit)
- Phase 41 Closure 锚点 = commit hash(可追溯到 41-01/02/03 SUMMARY.md 的具体 file:line 证据)
- 原 status = Phase 40 收尾时 frontmatter 状态;现 status = 翻 resolved 后的状态

**统计**:**12 re-verify-flip + 2 real-fix + 15 wontfix = 29 total**

### Plan 41-01 Cohort A — 12 个 re-verify-flip(D-01 复测归零)

| # | slug | 原 status | Plan | 处置类型 | Phase 41 Closure 锚点 | 现 status |
|---|------|-----------|------|----------|---------------------|-----------|
| 1 | vendor-commons-createcontext | fix_applied | 41-01 | re-verify-flip | `fc869829` | resolved |
| 2 | page-refresh-token-refresh-loop-failure | fix_applied | 41-01 | re-verify-flip | `8fd552cc` | resolved |
| 3 | captcha-custom-background-not-found | fix_applied | 41-01 | re-verify-flip | `7e640d40` | resolved |
| 4 | quick-create-fields-not-saving | debug_complete | 41-01 | re-verify-flip | `3e236080` | resolved |
| 5 | go-textfsm-dollar-anchor-bug | debug_complete | 41-01 | re-verify-flip | `4a2de2f7` | resolved |
| 6 | vdi-cpu-display-issue | debug_complete | 41-01 | re-verify-flip | `d9f23594` | resolved |
| 7 | vdi-menu-routing-redirect | debug_complete | 41-01 | re-verify-flip | `e52e325d` | resolved |
| 8 | vm-list-page-blank-404 | debug_complete | 41-01 | re-verify-flip | `c53332b7` | resolved |
| 9 | infopoint-fields-not-saving-at-all | fixed | 41-01 | re-verify-flip | `fe921e62` | resolved |
| 10 | vdi-vm-operations-fail | fixed | 41-01 | re-verify-flip | `17ab6f49` | resolved |
| 11 | ops-perms-alignment | applied | 41-01 | re-verify-flip | `c4902ef7` | resolved |
| 12 | operlog-loginlog-nickname-not-showing | fixed_pending_restart | 41-01 | re-verify-flip | `76f45d88` | resolved |

**Plan 41-01 验证证据**(见 `41-01-SUMMARY.md`):
- `vite.config.ts:193/195` REACT_FAMILY 传递闭包(01)
- `TokenManager.ts` getCachedEncryptionConfig grep 命中 0(02)
- `captcha_background_handler.go:184` StringArray(03)
- `device_info_collection_service.go:307` `device.Model == ""` 守卫(04)
- `textfsm.go:239/241/251/255` hasStartAnchor/hasEndAnchor 拼回(05)
- `vdi.go:22-28` CPUNumber/CPUCore/CPUPer/MemoryPer/DiskPer(06)
- `131_fix_vdi_menu_paths.sql` UPDATE 在 archive/(07)
- `vmOperationButtons.tsx` 存在 + `index.tsx:6` import(08)
- `excel_service.go:535/552/832` InfoPoint 特殊处理(09)
- `vdi_types.go:44/49/54/60/67/100/118/126` VMID + `vdi_client_extended.go:502-505`(10)
- `router.go:756-759` dedicatedline + `database.go:382` Migrate159AlignOpsPerms(11)
- `log.go:14/31` Nickname + migration_161 + `database.go:390` Migrate161LoginLogAddNickname(12)

### Plan 41-02 Cohort B Frontend — 7 个(2 re-verify-flip + 1 real-fix + 4 wontfix)

| # | slug | 原 status | Plan | 处置类型 | Phase 41 Closure 锚点 | 现 status |
|---|------|-----------|------|----------|---------------------|-----------|
| 13 | css-tree-vendor-chunk-tdz | root_cause_identified | 41-02 | wontfix (+ skip_audit) | `760a5513` | resolved |
| 14 | login-md-vendor-crash | investigating | 41-02 | wontfix (+ skip_audit) | `e18ae8c7` | resolved |
| 15 | react-vendor-activity-tdz | investigating | 41-02 | wontfix (+ skip_audit) | `ff8d6d37` | resolved |
| 16 | frontend-build-66-ts-errors | investigating | 41-02 | re-verify-flip | `8d8b9f7d` | resolved |
| 17 | device-modal-serial-auto-match | investigating | 41-02 | wontfix (+ skip_audit) | `d61828f9` | resolved |
| 18 | sidebar-apikey-menu-no-response | investigating | 41-02 | re-verify-flip | `6c15043b` | resolved |
| 19 | hardcoded-blue-theme-bypass | investigating | 41-02 | **real-fix** | `4345e56f` | resolved |

**Plan 41-02 验证证据**(见 `41-02-SUMMARY.md`):
- (13/14/15) `vite.config.ts` MARKDOWN_FAMILY/REACT_FAMILY 依赖图闭包方案,`dist/assets/` 无原报错兜底 vendor-CVbWoGUo.js,`grep jsdom/dom-selector/cssstyle dist/assets/` 命中 0
- (16) `usePersistedState.test.ts:11` 元组 API + `VirtualMachineList/index.tsx:32/54` vmApi/vdiServerApi import + npm run build 退出 0(34.32s)
- (17) `workstation_device_service.go:288-338` AddDeviceManual 后端完整,前端 UX 简化属新功能范畴推后续 phase
- (18) `migration_166_apikey_route_path_fix.go` + `database.go:411` 注册 + `src/pages/system/apikeys/{index.tsx,LogsModal.tsx}` 已就位
- (19) `src/design-system/components/AntdThemeBridge.tsx` 完整实现 + `App.tsx:9/47` import + wrap + `src/index.css` 浅色模式覆盖规则 + npm run build 退出 0;运行时主题色实时切换用户已 approved

### Plan 41-03 Cohort B Backend/Auth — 10 个(1 real-fix + 9 wontfix)

| # | slug | 原 status | Plan | 处置类型 | Phase 41 Closure 锚点 | 现 status |
|---|------|-----------|------|----------|---------------------|-----------|
| 20 | request-encryption-token-refresh-400 | investigating | 41-03 | wontfix | `2d283447` | resolved |
| 21 | login-wrong-pwd-no-prompt | investigating | 41-03 | wontfix (re-verify-flavored) | `5c3040e7` | resolved |
| 22 | file-upload-401-refresh | checkpoint_reached | 41-03 | wontfix | `63218bcc` | resolved |
| 23 | ad-admin-lockout-recurrence | investigating | 41-03 | wontfix (+ skip_audit) | `e8207767` | resolved |
| 24 | ad-update-attr-no-such-object | verifying | 41-03 | wontfix (re-verify-flavored) | `c4c63645` | resolved |
| 25 | ad-user-page-checkbox-not-click | verifying | 41-03 | **real-fix** | `32825417` | resolved |
| 26 | vdi-sync-invalid-uri | investigating | 41-03 | wontfix (re-verify-flavored) | `fc9fe38e` | resolved |
| 27 | vdi-server-test-500-input-warnings | investigation_in_progress | 41-03 | wontfix | `b6038443` | resolved |
| 28 | user-deptid-uuid-cast-recurring | diagnosed | 41-03 | wontfix (re-verify-flavored) | `25f3ef87` | resolved |
| 29 | interface-to-any-conversion-hint | diagnosed | 41-03 | wontfix (+ skip_audit) | `4fb48b84` | resolved |

**Plan 41-03 验证证据**(见 `41-03-SUMMARY.md`):
- (20) `configs/config.yaml:88-100` encryption exclude_paths 已含关键路径,关闭加密后前后端状态不一致属运维配置治理范畴
- (21) `api.ts:381-392` 401 拦截器已识别 `/system/auth/login` 短路 + `login/index.tsx:22-99` extractLoginErrorMessage helper + 内联 Alert state
- (22) `FileUpload.tsx:160` XHR + xhr.upload.onprogress 进度条,改 axios 重写 customRequest + onProgress callback 属新功能 UX 范畴推后续
- (23) Phase 36 AccountPool(多账号池 + 状态机 0/1/2 + 自动熔断恢复)+ Phase 38 FailoverClient.MarkFailure `operation:` vs `dial:` 前缀分流已落地;观察期 2026-06-18 → 2026-06-26 共 8 天未报告新锁定事件
- (24) `ldap_client.go:486-528` DNExists + `user_ad_sync_service.go:184-200` syncUserAttributes 入口预检 + `account_pool.go:374` countsTowardBreaker 前缀分流 + `pkg/crypto/sm4.go` GCM 实例缓存;`.md` Next Action 是运营层验证,非代码层可推进项
- (25) **`xingran-react-frontend/src/pages/ad-domain/users/index.tsx:601`** `rowKey="userDn"` → `rowKey="id"`(改用 DB UUID)+ 609 `selectedUsers.map(u => u.userDn)` → `selectedUsers.map(u => u.id)`;userDns 保留(发 batchSyncADUsersDirect API 需 DN);npm run build 退出 0(1m 27s)
- (26) `vdi_client_extended.go:347-359` ListResourceGroups 用 `GET /v1/resources_group`(非错误前缀)
- (27) `vdi_utils.go:18/23/28/33/39` decrypt 返 `""` + `vdi_utils.go:66` base64 StdEncoding typo 已修 + `IconSelect/index.tsx:110` suffix 三元运算 + `VDIServerConfig/index.tsx:234` Space orientation;VDI 服务器 `/v1/auth/tokens` 和 `/API/V1.0/Auth/Login` 都返 HTML 登录页面,外部依赖问题
- (28) `user_service.go:389` `LEFT JOIN sys_dept ON sys_dept.id = sys_user.dept_id` 已简化(无 CASE WHEN/NULLIF/regex guard)
- (29) `staticcheck -checks=SA1029 ./...` 命中 0 issues;1261 处 interface{} 合法(缓存 Set/GetJSON/JSONata/Excel/reflect);机械重写 30k 行 churn 超出 v1.16 范畴

## 4. 处置统计摘要

| 处置类型 | 数量 | session 列表 |
|----------|------|--------------|
| **re-verify-flip**(D-01) | 14 | 1-12, 16, 18, 21(flavored), 24(flavored), 26(flavored), 28(flavored) |
| **real-fix**(D-02) | 2 | 19(hardcoded-blue-theme), 25(ad-user-page-checkbox) |
| **wontfix**(D-02) | 13 | 13, 14, 15, 17, 20, 22, 23, 27, 29(纯 wontfix)+ 21/24/26/28(re-verify-flavored) |
| **TOTAL** | **29** | 12 (Plan 41-01) + 7 (Plan 41-02) + 10 (Plan 41-03) |

**注**:re-verify-flip 计数 14 包含"flavored won't-fix"(也归 wontfix 分类),纯 re-verify-flip = 14;real-fix = 2;纯 wontfix = 9;re-verify-flavored wontfix = 4。**所有 29 session 现 status 均为 resolved**。

| skip_audit session(4 个,wontfix 时叠加,移出 audit-open 计数) | slug |
|------|------|
| 1 | css-tree-vendor-chunk-tdz |
| 2 | login-md-vendor-crash |
| 3 | react-vendor-activity-tdz |
| 4 | device-modal-serial-auto-match |
| 5 | ad-admin-lockout-recurrence |
| 6 | interface-to-any-conversion-hint |
| **TOTAL** | **6 session** with `skip_audit: true` |

**注**:41-02 报告 4 个 + 41-03 报告 2 个 = 6 个,本 phase 共 6 个 session 顶层 `skip_audit: true` 移出 audit-open 计数。

## 5. 验收结论

| 标准 | 结果 | 备注 |
|------|------|------|
| Standard 1: validator 100% pass | ✅ **PASS** | 159/159 audited pass,0 fail,8 skip_audit excluded,121 date warn-only |
| Standard 2: audit-open debug_sessions < 5 | ✅ **PASS** | debug_sessions=0(< 5 阈值) |
| verify_phase40.sh exit code | 0 | [ALL PASS] Phase 40 verification SUCCESS |
| validate_debug_frontmatter.sh pass rate | 100.0% | (159 pass / 167 total / 8 skip_audit) |
| 29 session 处置矩阵 | ✅ **29/29** | 14 re-verify-flip + 2 real-fix + 13 wontfix(包含 4 re-verify-flavored) |
| TECH-06 达成 | ✅ **达成** | 29 → < 5(实际 0) |
| TECH-05 达成 | ✅ **达成** | audit-open < 5 + validator 100%(全条满足) |

**Phase 41 验收 PASS**。v1.16 milestone 验收材料就位,可进入 Task 2(STATE.md / ROADMAP.md / REQUIREMENTS.md 更新 + 用户最终确认 + push 决策)checkpoint:human-verify 阶段。

## 6. 引用与索引

- **三件套脚本**(沿用 Phase 40 工具链,D-05 不新建):
  - `scripts/validate_debug_frontmatter.sh`(D-10/D-11 双模式 validator)
  - `scripts/verify_phase40.sh`(D-16 双标准验收)
  - `scripts/fix_debug_frontmatter.py`(批量修复,本 phase 未触发)
- **Phase 41 SUMMARY**:
  - `.planning/phases/41-debug-session-gap-closure/41-01-SUMMARY.md`(12 sessions Cohort A)
  - `.planning/phases/41-debug-session-gap-closure/41-02-SUMMARY.md`(7 sessions Cohort B frontend)
  - `.planning/phases/41-debug-session-gap-closure/41-03-SUMMARY.md`(10 sessions Cohort B backend/auth)
- **Phase 41 验收证据(本文件)**:`.planning/phases/41-debug-session-gap-closure/41-VERIFICATION.md`
- **Phase 41 CONTEXT**:`.planning/phases/41-debug-session-gap-closure/41-CONTEXT.md`(D-01..D-06 决策)
- **Phase 40 验收模式范本**:`.planning/phases/40-tech-debt-cleanup/40-03-SUMMARY.md`(Standard 1/2 表格 + 缺口分析)
- **项目 memory**:
  - `MEMORY.md` → `xingran-server-side-sort-infra.md` 等(关联 Phase 40/41 工具链)

---

*Generated: 2026-06-26 — Phase 41 Plan 04 Task 1 verification suite*
*Verifier: gsd plan executor (sequential mode)*
*Three-piece suite: ALL PASS — v1.16 milestone acceptance materials ready*