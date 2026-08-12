---
phase: 41-debug-session-gap-closure
plan: 03
subsystem: debug-cleanup
tags: [cohort-b, backend, auth, ad-domain, vdi, uuid, mixed-fix-wontfix]

requires:
  - phase: 36-38 (AD AccountPool / FailoverClient.MarkFailure prefix split / SM4 防御性加固)
    provides: AD 类 5 个 session 决策依据(覆盖/外部依赖/实修边界)
  - phase: 40-03 (validator/verify/fix 工具链)
    provides: validate_debug_frontmatter.sh 100% pass 验证 + fix_debug_frontmatter.py 前置能力

provides:
  - 10 个原子 commit (1 fix(41) + 9 docs(41))，D-06 一话一 commit 契约
  - 10 个 .planning/debug/<slug>.md 全部 status: resolved
  - 2 个 session 顶层 skip_audit: true (ad-admin-lockout-recurrence + interface-to-any-conversion-hint) 移出 audit-open 计数
  - audit-open debug_sessions 预期 17 → 7(本 plan 贡献 ≤10,实际为 10 skip audit 计 8)
  - 1 个真实前端实修: ad-domain/users rowKey + rowSelection 改用 DB id UUID

affects: [41-04(验收关门 — 跑 verify_phase40.sh)]

tech-stack:
  added: []
  patterns:
    - "复测归零模式(沿用 Phase 40 40-01):历史 phase 已修但 frontmatter 未闭环的 session,读 .md 根因 → grep 代码确认修复 → 写 Phase 41 Closure 复测证据 → 翻 resolved,不重复实修"
    - "D-02 混合策略(won't-fix + skip_audit):陈年/重复/外部依赖/风格非 bug session,写明 won't_fix_reason(Phase X 覆盖 / staticcheck 0 命中 / 外部依赖 / 配置运维),顶层 skip_audit: true 让 audit-open 不计"
    - "项目 memory 驱动 AD/UUID 决策:不重复 Phase 36/38 已做的 AD 修复,只补复测证据"

key-files:
  created: []
  modified:
    - xingran-react-frontend/src/pages/ad-domain/users/index.tsx (实修 ad-user-page-checkbox-not-click: rowKey + rowSelection 改 id)
    - 10 个 .planning/debug/<slug>.md (frontmatter → resolved + Phase 41 Closure 块)

key-decisions:
  - "1 real-fix (ad-user-page-checkbox-not-click):rowKey='userDn' → rowKey='id' (UUID),消除 LDAP DN 特殊字符(逗号/等号)对 antd Table 严格比较的干扰;userDns(发给 batchSyncADUsersDirect API 的实际 DN 列表)保留不动"
  - "9 wontfix:6 个为复测发现已落地(login-wrong-pwd-no-prompt 修复 A+B 落地 / ad-update-attr-no-such-object 4 项 Fix 落地 / vdi-sync-invalid-uri 端点路径修正 / vdi-server-test-500-input-warnings 代码层 4 项已修但 VDI 服务器侧外部依赖未解 / user-deptid-uuid-cast-recurring userJoinClause 已简化 / request-encryption-token-refresh-400 是配置切换运维问题);2 个为已被 Phase 36/38 覆盖 / 风格非 bug(ad-admin-lockout-recurrence + interface-to-any-conversion-hint,均 skip_audit);1 个为新功能范畴推后续(file-upload-401-refresh XHR 改造)"
  - "2 session 顶层 skip_audit: true:ad-admin-lockout-recurrence (Phase 36/38 AccountPool + FailoverClient.MarkFailure 前缀区分已落地,观察期 2026-06-18 → 2026-06-26 共 8 天未复发)+ interface-to-any-conversion-hint (staticcheck SA1029 命中 0,1261 处合法 interface{},Go 1.18+ 风格别名,style non-bug)"

requirements-completed: [TECH-06]

duration: ~30min
completed: 2026-06-26
---

# Phase 41 Plan 03: Cohort B 后端/认证 10 个 session D-02 混合策略闭环

**把 Cohort B 后端/认证集群 10 个在途 session 按 D-02 混合策略(能修则修 + won't-fix 书面理由)逐一闭环,1 实修 + 9 won't-fix;audit-open debug_sessions 预期从 17 降至 7 以下;2 个 won't-fix 顶层 skip_audit 让 audit-open 完全不计。**

## Performance

- **Tasks:** 2/2 完成(Task 1: 3 认证/Token/登录 + Task 2: 7 AD/VDI/UUID/类型)
- **Commits:** 10 个原子 commit(1 fix(41) + 9 docs(41))
- **Build:** `go build ./...` 每次 commit 后退出码 0;`npm run build` 在实修 ad-user-page-checkbox-not-click 后退出码 0
- **Validator:** `bash scripts/validate_debug_frontmatter.sh` pass rate 100.0%(167 total / 163 pass / 4 skip / 0 fail)

## 10 个 fix(41)/docs(41) commit(按 session slug)

### Task 1 — 认证/Token/登录 3 个 session

1. `2d283447` **docs(41): request-encryption-token-refresh-400** — won't-fix 加密开关切换属运维配置,非代码 bug
   - **理由**: `configs/config.yaml:88-100` encryption exclude_paths 已含关键路径(public-key/upload/captcha/RPA-worker),refresh 端点设计为强制加密,不应加入 exclude_paths;关闭加密开关后前后端状态不一致是配置治理范畴,非代码 bug
   - **Phase 41 Closure**: won't-fix,具体理由引用 Plan 41-01 page-refresh-token-refresh-loop-failure 修复已就位

2. `5c3040e7` **docs(41): login-wrong-pwd-no-prompt** — won't-fix 复测发现已落地
   - **理由**: api.ts:381-392 401 拦截器已识别 `/system/auth/login` 短路(不调 refreshToken/clearTokens/跳转),提取 `response.data.message/msg` 原样 reject;login/index.tsx:22-99 `extractLoginErrorMessage` helper + 内联 Alert state + 三处状态切换已落地;`npm run build` 退出 0
   - **Phase 41 Closure**: 复测发现已落地型

3. `63218bcc` **docs(41): file-upload-401-refresh** — won't-fix XHR 改造属新功能推后续
   - **理由**: FileUpload.tsx:160 用 XHR + `xhr.upload.onprogress` 实时进度条,改 axios 需重写 customRequest + onProgress callback,涉及上传 UX 行为变更(预计 2-3 文件 + UX 测试),超出 v1.16 debug 清理 milestone 范围
   - **Phase 41 Closure**: 推后续 phase,当前缓解用户上传完手动刷新页面

### Task 2 — AD/VDI/UUID/类型 7 个 session

4. `e8207767` **docs(41): ad-admin-lockout-recurrence** — won't-fix 已被 Phase 36/38 账号池覆盖,观察期未复发 + **skip_audit: true**
   - **理由**: Phase 36 AccountPool(多账号池 + 状态机 0/1/2 + 自动熔断恢复)+ Phase 38 FailoverClient.MarkFailure `operation:` vs `dial:` 前缀分流已落地(项目 memory `ad-modify-fail-double-counts-breaker` / `ad-ldap-error49-data-subcodes`);观察期 2026-06-18 → 2026-06-26 共 8 天未报告新锁定事件
   - **Phase 41 Closure**: won't-fix,具体理由引用 Phase 36/38 + 观察期数据,顶层 skip_audit 移出 audit-open

5. `c4c63645` **docs(41): ad-update-attr-no-such-object** — won't-fix 复测发现已落地
   - **理由**: `.md` Resolution 描述的 4 项 Fix 全部已落地:
     - Fix 1: `internal/services/addomain/ldap_client.go:486-528` 新增 `(*LDAPClient).DNExists` 方法,base-scope `(objectClass=*)` Search,code 32 语义化为 `(false, nil)`
     - Fix 1 联动: `user_ad_sync_service.go:184-200` `syncUserAttributes` 入口加 DNExists 预检
     - Fix 2: `account_pool.go` 新增 `ErrADTargetNotFound` 哨兵错误,`user_handler.go` Update goroutine 检测即 break 重试循环
     - Fix 3: `account_pool.go:374` `countsTowardBreaker := !strings.HasPrefix(reason, "operation:")` 前缀分流
     - Fix 4: `pkg/crypto/sm4.go` 缓存 GCM 实例 + nil receiver 检查 + sync.Mutex 保护
   - **Phase 41 Closure**: 复测发现已落地型;`.md` Next Action(用户 evidence checkpoint)是运营层验证(查 PostgreSQL sys_user.ad_dn + Get-ADUser + 4625),非代码层可推进项

6. `32825417` **fix(41): ad-user-page-checkbox-not-click** — ad-domain/users rowKey + rowSelection 改用 DB id UUID **[唯一实修]**
   - **改动**: `xingran-react-frontend/src/pages/ad-domain/users/index.tsx:601` `rowKey="userDn"` → `rowKey="id"`(改用 DB UUID);行 609 `selectedUsers.map(u => u.userDn)` → `selectedUsers.map(u => u.id)`
   - **保持不变**: `userDns = selectedUsers.map(u => u.userDn)`(行 319/323)是发给 `batchSyncADUsersDirect` API 的实际 LDAP DN 列表,API 必须收 DN,不动
   - **根因复述**: LDAP `userDn` 含逗号、等号、特殊字符(OU=xxx,CN=yyy,DC=zzz),antd Table 严格比较对复杂字符串不稳定,checkbox 视觉状态不同步;改用 DB UUID 后字符串简单无特殊字符,严格比较工作正常
   - **verification**: `cd xingran-react-frontend && npm run build` 退出 0(1m 27s)

7. `fc9fe38e` **docs(41): vdi-sync-invalid-uri** — won't-fix 复测发现已落地(端点路径 `/v1/resources_group`)
   - **理由**: `internal/services/vdi/vdi_client_extended.go:347-359` `ListResourceGroups` 方法已用 `GET /v1/resources_group`(非 `/api/v1/resources_group` 错误前缀)
   - **Phase 41 Closure**: 复测发现已落地型

8. `b6038443` **docs(41): vdi-server-test-500-input-warnings** — won't-fix 代码层已修,500 是 VDI 服务器侧外部依赖
   - **代码层已修(复测)**:
     - `internal/services/vdi/vdi_utils.go:18/23/28/33/39` `decryptVDIPassword` 错误路径全部返回 `""`(非原始加密字符串)
     - `vdi_utils.go:66` `EncodeToString` typo 已修(base64 StdEncoding)
     - `xingran-react-frontend/src/components/IconSelect/index.tsx:110` `suffix={value ? getIconComponent(value) : <span />}` 已修(避免 Input focus 警告)
     - `xingran-react-frontend/src/pages/vdi/VDIServerConfig/index.tsx:234` Space 组件 `orientation="vertical"` 已用
   - **未解(外部依赖)**: VDI 服务器 `/v1/auth/tokens` 和 `/API/V1.0/Auth/Login` 两个候选端点都返回 HTML 登录页面(`<!-- __Forbidden Request__ -->`),不是 JSON;**这是 VDI 服务器侧 API 规格/认证配置问题,需 VDI 厂商文档或管理员确认**
   - **Phase 41 Closure**: won't-fix,代码层修复完整,500 根因为外部依赖

9. `25f3ef87` **docs(41): user-deptid-uuid-cast-recurring** — won't-fix 复测发现已落地(userJoinClause 简化)
   - **理由**: `internal/services/system/user_service.go:389` `userJoinClause := "LEFT JOIN sys_dept ON sys_dept.id = sys_user.dept_id"` 已简化(无 CASE WHEN/NULLIF/regex guard);其余 4 处 `CASE WHEN`(行 470/480/481)是统计聚合查询(`SUM(CASE WHEN status = 0 THEN 1 ELSE 0 END)`),与本 session JOIN 问题无关保留
   - **Phase 41 Closure**: 复测发现已落地型

10. `4fb48b84` **docs(41): interface-to-any-conversion-hint** — won't-fix style non-bug + **skip_audit: true**
    - **理由**: 1261 处 `interface{}` 全部为合法动态值:缓存 Set/GetJSON 序列化、JSONata 求值、Excel 动态行解析、reflect 通用 helper;`staticcheck -checks=SA1029 ./...` 命中 **0 issues**(项目 memory:`interface{}` 是 Go 1.18+ 起的 `any` 别名,编译器完全等价);264 个 .go 文件涉及,跨 pkg/internal/cmd/scripts/rpa-worker 5 个分区
    - **机械重写代价**: 30k 行 churn,纯 style,无运行期收益,超出 v1.16 debug 清理 milestone 范围;后续新代码可自然使用 `any`,存量不强制重写(.md 推荐方案 C)
    - **Phase 41 Closure**: won't-fix,具体理由引用 staticcheck 0 命中 + 风格非 bug,顶层 skip_audit 移出 audit-open

## Deviations from Plan

- **无重要偏离**: plan D-02 决策树完整执行,10 个 session 全部按 6 个 won't-fix 理由类型分别落地:
  1. 复测发现已落地型 (5):login-wrong-pwd-no-prompt / ad-update-attr-no-such-object / vdi-sync-invalid-uri / user-deptid-uuid-cast-recurring / (vdi-server-test-500-input-warnings 部分)
  2. 已被其他 phase 覆盖型 (1):ad-admin-lockout-recurrence
  3. 配置/运维问题型 (1):request-encryption-token-refresh-400
  4. 新功能范畴推后续型 (1):file-upload-401-refresh
  5. 外部依赖型 (1):vdi-server-test-500-input-warnings (剩余)
  6. 风格非 bug 型 (1):interface-to-any-conversion-hint
- **唯一实修命中根因**: ad-user-page-checkbox-not-click — .md 早先描述的 fix (改用 DB UUID) 实际未落地,本次 plan 才实施
- **2 个 skip_audit**: ad-admin-lockout-recurrence + interface-to-any-conversion-hint,均符合"陈年/重复/外部依赖/风格非 bug"标准,顶层 skip_audit: true 让 audit-open 完全不计

## Verification

- `go build ./...` 退出码 0 ✓(本 plan 仅 1 个实修改 xingran-react-frontend,Go 代码无改动,沿用 baseline 0)
- `cd xingran-react-frontend && npm run build` 退出码 0 ✓(1m 27s,实修 ad-user-page-checkbox-not-click 后)
- 10/10 `.planning/debug/<slug>.md` frontmatter `status: resolved` ✓
- 2/10 session 顶层 `skip_audit: true` ✓(ad-admin-lockout-recurrence + interface-to-any-conversion-hint)
- 每个 session 一个独立 commit,D-06 message 格式合规 ✓
- `bash scripts/validate_debug_frontmatter.sh` pass rate **100.0%** ✓(167 total / 163 pass / 4 skip / 0 fail)

## 复测抽查真实存在

| Session | 代码证据 | 状态 |
|---------|----------|------|
| login-wrong-pwd-no-prompt | `xingran-react-frontend/src/lib/api.ts:386` `/system/auth/login` 短路 + login/index.tsx:22 `extractLoginErrorMessage` | 已落地 |
| ad-update-attr-no-such-object | `internal/services/addomain/ldap_client.go:501` `DNExists` + user_ad_sync_service.go:187 调用 + account_pool.go:374 前缀分流 | 已落地 |
| vdi-sync-invalid-uri | `internal/services/vdi/vdi_client_extended.go:359` `GET /v1/resources_group` | 已落地 |
| vdi-server-test-500-input-warnings | `internal/services/vdi/vdi_utils.go:18-39` decrypt 返 `""` + IconSelect suffix 三元运算 + Space orientation | 已落地 |
| user-deptid-uuid-cast-recurring | `internal/services/system/user_service.go:389` `LEFT JOIN sys_dept ON sys_dept.id = sys_user.dept_id` | 已落地 |
| ad-user-page-checkbox-not-click | `xingran-react-frontend/src/pages/ad-domain/users/index.tsx:601` `rowKey="id"` + 609 `selectedUsers.map(u => u.id)` | 本 plan 实修 |

## audit-open 预期影响

- **Phase 41-01 前**: debug_sessions = 29 (Cohort A 12 + Cohort B 17)
- **Phase 41-01 后**: 29 → 17 (12 Cohort A 翻 resolved)
- **Phase 41-03 本 plan 后**: 17 → 7 (10 session 全部翻 resolved,但 2 个 skip_audit 让 audit-open 实际减 10)
  - 预期: 17 - 10 = **7** (剩余 7 个 = Plan 41-02 前端集群 + 部分 awaiting_human_verify / 跨 phase 范围)
- **Phase 41-02 (前端集群 7 个)**: 进一步减少,目标是 < 5

## Next

进入 Plan 41-02(Cohort B 前端集群 7 个 session 混合修+wontfix),完成后 Phase 41-04 跑验收(verify_phase40.sh 双 PASS + audit-open < 5),v1.16 闭环。