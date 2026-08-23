---
phase: 61
status: clean
depth: standard
files_reviewed: 16
files_reviewed_list:
  - pkg/permission/resource_action_map.go
  - pkg/permission/resource_action_map_test.go
  - pkg/permission/config.go
  - internal/middleware/apikey.go
  - internal/middleware/apikey_test.go
  - internal/middleware/apikey_integration_test.go
  - internal/middleware/apikey_resource_permission_test.go
  - internal/middleware/apikey_inherit_integration_test.go
  - internal/middleware/apikey_rate_limit_test.go
  - internal/middleware/select_scope.go
  - internal/middleware/select_scope_test.go
  - internal/services/rate_limiter.go
  - internal/services/rate_limiter_test.go
  - internal/services/cache_config_service.go
  - internal/services/cache_config_service_test.go
  - internal/api/router.go
critical: 0
warning: 0
info: 3
total: 0
reviewed_at: 2026-08-13T12:00:00Z
fixed_at: 2026-08-13T18:30:00Z
---

## Fixes Applied

All blocker (1) + warning (4) findings resolved. Info findings (IN-01~IN-03) deferred per scope decision (documentation/cleanup, not functional).

| Finding | Severity | Fix | Commit | Verification |
|---------|----------|-----|--------|--------------|
| BL-01 | blocker | getRequiredScope 覆盖 map 全部写 action (list/view→read; add/edit/remove/create/delete/import/resetPwd→write; export→read);未知 action 在 RequireAPIKeyResourcePermission 路径 fail-closed | a137441 | TestRequireAPIKeyResourcePermission_WriteActionReadScopeDenied + TestSelectScope_WriteActionReadScopeDenied 红→绿 |
| WR-01 | warning | router.go apikey 组检查 "system:apikey:delete" → "system:apikey:remove",与 D-04 system:* remove 约定一致 | 6d9e311 | grep router.go apikey group check = remove |
| WR-02 | warning | calculateReset 增加 window duration 参数(hour/day 窗口不再误报 1 分钟重置);Retry-After 改为 time.Until(ResetAt) 动态值 | a673b07 | TestCalculateResetWindowAware |
| WR-03 | warning | InheritPerms 分支 GetUserPermissions 前加 permSvc==nil||db==nil 防御 → 401 fail-closed(非 panic) | dc5ab29 | TestMultiAuth_InheritPermsNilService |
| WR-04 | warning | 新增写 action(remove)+ 只读 scope=[read] 拒绝用例(middleware + SelectScope 双路径) | (含 BL-01 commit a137441) | 红绿验证 + 回归锚 |

## Deferred Info Findings (out of fix scope)

| Finding | Severity | Reason |
|---------|----------|--------|
| IN-01 | info | D-17 默认值四处真相源统一,留 FUTURE-APIKEY-04 per-key 限流 override 时一并处理 |
| IN-02 | info | getLimit 三次独立 RLock 跨字段一致性,D-19 已声明 reload 仅影响新请求,语义无安全影响 |
| IN-03 | info | apikey.go:205-208 注释顺序自述混乱,纯文档,不影响逻辑 |



## Findings

### BL-01 (blocker) — getRequiredScope 词汇表与 resource_action_map 不匹配：只读 scope 可通过写操作的资源权限检查

- File: internal/middleware/apikey.go
- Line: 348-363 (getRequiredScope) + 328-335 (RequireAPIKeyResourcePermission union 检查)
- Severity: blocker
- Issue: `resource_action_map.go` 的 action 词汇表为 `list/view/add/edit/remove/export/import/resetPwd`（D-04），但 `getRequiredScope` 只识别 `list/view/create/edit/delete`，其余一律兜底返回 `"read"`。由此 `RequireAPIKeyResourcePermission("system:user", "remove")` 的 `requiredScope = getRequiredScope("remove") = "read"`，一个**仅持有 read scope 的只读 API Key** 在 union 检查第③支路（粗粒度 scope）即命中 `scope == "read"`，直接通过删除/新增/导出/导入/重置密码全部写操作的资源权限校验。这构成权限提升（privilege escalation）路径，直接违反 CONTEXT.md 审查焦点「D-12 strict matching — no fallback」的意图（此处 fallback 到 read 恰恰发生在写操作上）。当前生产风险被 D-05（中间件未挂载到 apikey_router.go）暂时掩盖，但该 helper 是 AUTH-04 的核心交付物，一旦后续 phase 挂载即引入真实漏洞。`RateLimitByScope` 同样受影响：`action="add"` 的写请求会被错误地按 read 档限流而非 write 档。
- Fix: 让两个词汇表对齐。`getRequiredScope` 应覆盖 map 实际使用的全部写 action：
  ```go
  scopeMap := map[string]string{
      "list": "read", "view": "read",
      "add": "write", "edit": "write", "remove": "write",
      "create": "write", "delete": "write",
      "export": "read", "import": "write", "resetPwd": "write",
  }
  ```
  更稳妥的做法是对未知 action 在 `RequireAPIKeyResourcePermission` 路径上 fail-closed（粗粒度支路不匹配而非兜底 read），与 D-03/D-12 的 fail-closed 取向一致。同时补测试：action="add"/"remove" + scopes=["read"] 必须 403（见 WR-04）。

### WR-01 (warning) — system:apikey 删除权限三处词汇不一致：remove vs delete vs 未播种

- File: pkg/permission/config.go:94; internal/api/router.go:247
- Severity: warning
- Issue: Phase 61 新增常量 `APIKeyRemove = "system:apikey:remove"` 并在 `resource_action_map` 中以 `remove` 注册；但 `router.go:247` 路由组级 `RequirePermissions` 检查的是字面量 `"system:apikey:delete"`；而菜单种子（migrations/archive 全部 apikey menu SQL）只播种 `system:apikey:list` / `system:apikey:logs`，`add/edit/delete/remove` 均无任何菜单授权来源。结果：① 路由组的 delete 权限实际无人能被授予（只能靠 list 的 any-of 通过）；② 未来挂载 `RequireAPIKeyResourcePermission("system:apikey", "remove")` 时，`permCode = "system:apikey:remove"` 与既有授权体系（delete）对不上，即使 admin 通配之外的细粒度授权也必然 403。该不一致由本 phase 新增常量正式固化，应在引入时对齐。
- Fix: 统一为一个词汇（建议沿用 system:* 模块的 `remove` 约定，与 D-04 一致）：router.go 组检查改 `"system:apikey:remove"`，并补菜单种子/迁移授权 `system:apikey:add/edit/remove`；或在 config.go 常量处显式注释说明与 router 字面量的差异是历史遗留并给出对齐 TODO。

### WR-02 (warning) — calculateReset 对 hour/day 窗口违规返回错误的 ResetAt；Retry-After 恒为 60

- File: internal/services/rate_limiter.go:246-253（calculateReset），151-170（调用点）; internal/middleware/apikey.go:414
- Severity: warning
- Issue: `calculateReset` 无条件返回 `times[0].Add(time.Minute)`，但 `Check` 在小时级（rate_limiter.go:156）和天级（:167）超限路径也调用它。当触发的是每小时/每天限额时，`X-RateLimit-Reset` 声称约 1 分钟后重置，实际窗口要 1 小时/24 小时才滑动——响应头对客户端产生严重误导（客户端会在 1 分钟后密集重试，加剧超限）。`apikey.go:414` 的 `Retry-After: 60` 硬编码同样与真实 ResetAt 脱钩。属于正确性问题（输出错误的协议元数据），非性能问题。
- Fix: `calculateReset` 增加窗口时长参数：`calculateReset(times, time.Hour)` / `(times, 24*time.Hour)`，分钟路径保持 `time.Minute`；`Retry-After` 改为 `strconv.Itoa(int(time.Until(result.ResetAt).Seconds()))`（下限 1）。

### WR-03 (warning) — MultiAuth 对 permSvc==nil 且 InheritPerms=true 的组合无防御，直接 nil 解引用 panic

- File: internal/middleware/apikey.go
- Line: 218（`permSvc.GetUserPermissions(db, *apiKey.UserID)`）
- Severity: warning
- Issue: 新签名 `MultiAuth(apiKeyService, usageLogger, permSvc, db)` 的 docstring 明确说「InheritPerms=false 时传 nil 也无副作用」，但反向组合（permSvc=nil + 某 key 开了 InheritPerms=true）会在第 218 行 nil 接口调用 panic，导致整个请求 500 且 gin recovery 之前的中间件状态不一致。生产 router.go:254-259 传了真实 service 故当前安全，但测试基建（apikey_integration_test.go 多处 `MultiAuth(fakeSvc, logger, nil, nil)`）已经把 nil 变成合法调用形态，任何后续维护者新开一条 InheritPerms=true 的用例踩坑即 panic。D-09 的 fail-closed 语义也要求「加载失败 → 401」而非 panic。
- Fix: 在 InheritPerms 分支入口加防御：
  ```go
  if permSvc == nil || db == nil {
      applogger.Errorf("[API_KEY] InheritPerms=true 但 permSvc/db 未注入: apiKeyID=%s", apiKey.ID)
      response.Error(c, response.ErrUnauthorized, "用户权限加载失败")
      c.Abort()
      return
  }
  ```

### WR-04 (warning) — 测试盲区：无写 action + 只读 scope 的拒绝用例，BL-01 因此漏网

- File: internal/middleware/apikey_resource_permission_test.go
- Line: 55-126（全部 8 个用例）
- Severity: warning
- Issue: 现有 8 个 `RequireAPIKeyResourcePermission` 用例覆盖了 admin 通配、read 命中、细粒度 PermissionCode 命中、edit 缺 write 拒绝、未映射 resource/action 拒绝、scopes 缺失/类型错——但**没有任何用例用 map 真实词汇表中的写 action（add/remove/export/import/resetPwd）+ scopes=["read"] 断言 403**。正是这个盲区让 BL-01 的词汇表错配通过全部测试。同理 `apikey_rate_limit_test.go` 7 个用例也只用 list/edit，未覆盖 add/remove 经 `getRequiredScope` 兜底 read 的路径。
- Fix: 补两类用例：① `runProbeWithScopes(t, "system:user", "remove", []string{"read"})` 断言 403；② `SelectScope([]string{"read"}, false, "remove")` 断言 ("", false)（修复 BL-01 后）。这两个用例在修复前应失败、修复后转绿，兼作回归锚。

### IN-01 (info) — D-17 默认值存在两处真相源，有漂移风险

- File: internal/services/rate_limiter.go:85-94（staticRateLimitProvider）; internal/services/cache_config_service.go:340-353（rateLimitDefaults）+ 773-880（GetConfigInfo 的 Default 字段）
- Severity: info
- Issue: 同一组 read=30/500/5000、write=100/1500/15000、admin=200/5000/50000、default=120/2000/20000 默认值被硬编码三处（static provider、setDefaultsIfNeeded、ConfigInfo.Default）。将来调任何一处默认值而漏改其余两处，会产生「nil provider 兜底阈值 ≠ sys_config 播种阈值 ≠ 范围校验重置阈值」的静默不一致。`getLimit` 的兜底 defaultValue（120/2000/20000，rate_limiter.go:73-75）是第四处。
- Fix: 提取一个包级 `var defaultRateLimits = map[string]RateLimit{...}` 作为唯一真相源，static provider / setDefaultsIfNeeded / GetConfigInfo 均从中读取。

### IN-02 (info) — getLimit 三次独立 RLock 读取，reload 瞬间可能产生混合快照

- File: internal/services/rate_limiter.go:68-77
- Severity: info
- Issue: `getLimit` 对 per_minute/per_hour/per_day 各调一次 `GetRateLimit`，每次独立 `RLock/RUnlock`（cache_config_service.go:407-415）。`ReloadConfig` 持写锁整体替换 map，因此单次请求可能读到「新 minute + 旧 hour」的混合配置。限流语义上无安全影响（任一窗口都会独立生效），且 D-19 已声明 reload 仅影响新请求，但跨字段一致性未定义。
- Fix: 可接受现状；若要严格，加 `GetRateLimitTriple(scope)` 单次 RLock 返回 `RateLimit` 结构。

### IN-03 (info) — apikey.go 注释与逻辑顺序自述混乱

- File: internal/middleware/apikey.go
- Line: 205-208
- Severity: info
- Issue: 注释「注:顺序在 c.Set("auth_type", "api_key") 之后、c.Set("api_key_id") 之前不重要」描述的是一个并不存在的顺序约束（InheritPerms 分支实际位于两个 c.Set 之后），读起来像残留的开发注记，增加后续维护者的理解成本。
- Fix: 删除该注记或改写为「InheritPerms 分支在 abort 路径上不影响已设 context 键」一句即可。

## Summary

Phase 61 的两个交付面质量不均。**做得好的部分**：`resource_action_map.go` 的 59 项静态映射与 `LookupResourceAction` 的 fail-closed 查询语义（D-02/D-03）实现干净、测试断言完整（命中/未命中/MapKeys/范围防御）；InheritPerms 并集合并（D-06）、401 fail-closed（D-09）、User 关联读取（D-10）在 apikey.go 中均正确落地，sqlite 集成测试覆盖了合并/UserID nil/DB 错误/username 来源/InheritPerms=false 五条关键路径；`SelectScope` 纯函数抽取（D-12/D-13）语义正确且 9 个单测覆盖五路径；`CacheConfigService` 的 12 个 rate_limit.* 键（D-16）、默认值对齐（D-17）、`RateLimitProvider` 接口解耦（D-18）、reload 语义（D-19）实现与测试均到位；QUAL-01 回归锚（strconv.Itoa 响应头）保持绿。**核心缺陷**：`getRequiredScope` 的 action 词汇表（create/delete）与 `resource_action_map` 的词汇表（add/remove/export/import/resetPwd）错配，导致只读 API Key 可通过写操作的资源权限检查（BL-01），且现有测试恰好没有这个盲区组合（WR-04）。另有 apikey 删除权限 remove/delete 三处不一致（WR-01）、限流 ResetAt/Retry-After 元数据错误（WR-02）、permSvc nil 防御缺失（WR-03）。

## Recommendations

1. **先修 BL-01 再进入 phase verification**：对齐 `getRequiredScope` 与 map 词汇表，写 action 全部映射 write（或 fail-closed），并用 WR-04 的两个用例做红-绿验证。虽然 D-05 未挂载中间件使当前无生产暴露面，但这是 AUTH-04 交付物的核心正确性问题。
2. WR-01 的 remove/delete 对齐可与 BL-01 同批处理（同属权限词汇表一致性问题），避免后续 phase 挂载资源中间件时再踩一次。
3. WR-02 / WR-03 为独立小修，可合并为一个 quick 任务。
4. IN-01 的默认值真相源收敛建议在做 FUTURE-APIKEY-04（per-key 限流 override）时一并处理，避免第三次复制默认值表。
