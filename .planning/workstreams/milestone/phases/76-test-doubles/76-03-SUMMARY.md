---
phase: 76-test-doubles
plan: "03"
subsystem: testing
tags: [addomain, ldap, interface-seam, failover, test-doubles, mock]

# Dependency graph
requires: [76-01]
provides:
  - LDAPClientIface 接口 seam（16→20 方法，20 处 failover 闭包全接口化）
  - FailoverClient.clientFactory 注入字段 + newClient 辅助方法（生产默认 NewLDAPClient 行为不变）
  - ExecuteWithFailover operation 签名 func(client LDAPClientIface) error
  - mockLDAPClient 多批次驱动能力（searchUsersFn 函数字段优先于 searchUsersRes）
  - FailoverClient 顺序遍历 / maxHops 封顶接口驱动测试（零真实网络）
affects: [phase-78-block05-addomain-coverage]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "clientFactory struct 字段注入（per-instance FailoverClient 构造，符合仓库 DI 风格，非包级 var）"
    - "编译门强制接口补入：闭包内裸字段访问 client.conn.Search 经纯委托方法 SearchWithRequest 入接口"
    - "接口扩展三件套：iface 追加（调用方注释）→ 具体类型（已存在/纯委托）→ mock（preset + 计数）双侧编译期断言"
    - "机械替换纪律：diff 审查 24 处类型行逐一核对，闭包体/函数体零改动"

key-files:
  created:
    - internal/services/addomain/failover_client_76_03_test.go
  modified:
    - internal/services/addomain/ldap_iface.go
    - internal/services/addomain/ldap_client.go
    - internal/services/addomain/ldap_client_mock_test.go
    - internal/services/addomain/failover_client.go
    - internal/services/addomain/group.go
    - internal/services/addomain/group_management_service.go
    - internal/services/addomain/group_sync_service.go
    - internal/services/addomain/user.go
    - internal/services/addomain/user_ad_sync_service.go
    - internal/services/addomain/dept_sync_service.go
    - internal/services/addomain/sync.go
    - internal/scheduler/dept_sync_tasks.go

key-decisions:
  - "编译门强制第 20 个接口方法 SearchWithRequest：group_sync_service.go:115 闭包原直接访问 client.conn.Search（具体类型私有字段，计划/pattern-mapper 双双遗漏），接口化后不可达。经纯委托方法（return c.conn.Search(req)）入接口，闭包内单行替换，行为与错误语义零漂移"
  - "maxHops 用例断言按真实语义修正：封顶耗尽返回「账号池 N 个账号均失败」聚合错误，ErrAllAccountsUnavailable 仅空池返回——计划 behavior 块笔误按 Rule 1 修正，NotErrorIs 显式锁定该语义"
  - "顺序遍历测试依赖 sqlite rowid 插入序（ListAvailable 为无 ORDER BY 的 GORM Find，rowid 序即插入序，与本包既有测试同一依赖面）"

patterns-established:
  - "Pattern: FailoverClient mock 工厂注入 = fc.clientFactory 按账号 Username 分流 mock，落库断言照 account_pool_test.go 的 db.Raw 查询式"
  - "Pattern: 接口扩展必须双侧编译期断言（*LDAPClient 与 *mockLDAPClient），mock 侧即零遗漏守护点"

requirements-completed: [INFRA-03]

# Metrics
duration: 18min
completed: 2026-08-23
---

# Phase 76 Plan 03: LDAPClientIface 扩展 + FailoverClient 注入缝 Summary

**LDAPClientIface 16→20 方法（3 计划内 + 1 编译门强制）、FailoverClient 增 clientFactory 注入字段、ExecuteWithFailover 闭包签名接口化并完成 24 处类型行（20 闭包 + 4 辅助签名）机械替换零行为漂移，顺序遍历/maxHops/多批次三组接口驱动测试零真实网络全绿**

## 接口方法数偏差说明（verifier 验收口径）

计划 must_haves 记「19 方法（16+3）」。实际交付 **20 方法**：计划的 Task 2 action 自带编译门逃生条款（「若调用了 19 方法接口之外的方法（编译门会拦），按同一「调用方」注释惯例把该方法补入接口」），本 plan 执行中该条款被触发一次——group_sync_service.go:115 闭包原直接访问具体类型私有字段 `client.conn.Search(searchRequest)`（PATTERNS §76-03e 记「闭包体内非接口方法仅 3 处」时漏点），接口化后该行不可达。详见 Deviations 1。

## Performance

- **Duration:** 18 min（07:08:53Z → 07:27:06Z）
- **Tasks:** 3/3
- **Files modified:** 13（1 新测试文件 + 12 修改）
- **Commits:** aa6e31c / dba8065 / e0fbcc7

## Accomplishments

- **Task 1（aa6e31c）**：ldap_iface.go 追加 UpdateGroupAttribute/CreateOU/DNExists（全部在 *LDAPClient 上已存在，零新实现，OUExists 外部零使用不加）；mockLDAPClient 补 3 方法 + preset + 计数；双侧 `var _ LDAPClientIface = ...` 编译期断言通过
- **Task 2（dba8065）**：failover_client.go 增 clientFactory 字段 + newClient 辅助（nil 走 NewLDAPClient）；operation 签名接口化；ExecuteWithFailover :54 调用改走 f.newClient，PickFirstConnect :106 保持原样（ad_authenticator.go:216 需 .Conn()，实测不受波及）；20 处闭包（client 14 处 + ldapClient 6 处）+ 4 处辅助签名（syncDeptTree/moveUserToNewOU/syncUserAttributes/moveSingleUserToOU）机械替换，diff 逐行核对闭包体/函数体零改动
- **Task 3（e0fbcc7，tdd）**：mock 增 searchUsersFn 函数字段（非 nil 优先于 searchUsersRes）+ searchUsersCalls 计数；failover_client_76_03_test.go 三个具名用例（TestFailover 前缀），复用 setupTestPool/insertAccount，全部经 clientFactory 注入 mock 零真实网络

## Task Commits

1. **Task 1: 接口 +3 方法与 mock 补齐（编译闭环）** - `aa6e31c` (feat)
2. **Task 2: clientFactory 字段 + 签名接口化 + 24 处类型行替换** - `dba8065` (refactor)
3. **Task 3: mock walk/分页能力 + FailoverClient 接口驱动测试** - `e0fbcc7` (test)

## Decisions Made

- **SearchWithRequest 命名与形态**：选择「接受 *ldap.SearchRequest 的纯委托」而非「语义化 SearchGroupByDN(groupDN)」——前者闭包体仅 1 行改动（client.conn.Search → client.SearchWithRequest），后者要移动 ~10 行请求构造逻辑；零行为漂移优先。刻意不把既有 `Conn()` 方法入接口（会把 wire 级 *ldap.Conn 泄漏给全部 mock 消费方，破坏 seam，且与 PickFirstConnect 保持具体类型的红线设计矛盾）
- **顺序遍历断言组合**：operation 恰 1 次 + 工厂恰 2 次 + username-0 failure_count=1 落库 + username-1 (failure_count=0 AND last_success_at NOT NULL) —— 同时锁定 T-76-03-03（failover 簿记不被注入缝绕过）
- **maxHops 断言用常量名** DefaultMaxHops（非魔法数 10），与既有 TestFailoverClient_MaxHopsConstant 呼应

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] 编译门强制：接口第 20 个方法 SearchWithRequest**
- **Found during:** Task 2（类型替换后 go build）
- **Issue:** group_sync_service.go:115（SyncSingleGroup 闭包体）直接访问 `client.conn.Search(searchRequest)` —— conn 是 *LDAPClient 私有字段，闭包参数接口化后不可达。PATTERNS §76-03e「闭包体内非接口方法仅 3 处」的清点漏了这处裸字段访问（它不是方法调用，grep 方法名清点法天然漏检）
- **Fix:** 按计划 Task 2 自带的编译门逃生条款处理：ldap_client.go 新增纯委托方法 `SearchWithRequest(searchRequest *ldap.SearchRequest) (*ldap.SearchResult, error)`（return c.conn.Search(searchRequest)，零逻辑），入接口（带「调用方」注释），mock 补 preset + 计数；闭包内单行替换。接口 16→20（3 计划内 + 1 编译门强制），must_haves 的「19 方法」口径由此偏差
- **Files modified:** ldap_client.go / ldap_iface.go / ldap_client_mock_test.go / group_sync_service.go
- **Verification:** go build exit 0 + 三包回归全绿 + diff 中该文件仅 2 行变更（类型行 + 委托调用行）
- **Committed in:** dba8065

**2. [Rule 1 - Bug] maxHops 用例的 ErrAllAccountsUnavailable 断言与真实语义不符**
- **Found during:** Task 3（写用例时核对 failover_client.go 返回路径）
- **Issue:** 计划 behavior 块称「12 账号全 Connect 失败 → ExecuteWithFailover 返回 ErrAllAccountsUnavailable（errors.Is）」；实际代码该场景返回 `fmt.Errorf("账号池 %d 个账号均失败: %w", maxAttempts, lastErr)`（lastErr=连接错误），ErrAllAccountsUnavailable 仅 len(available)==0 空池路径返回
- **Fix:** 断言改为 Contains "均失败" + `assert.NotErrorIs(err, ErrAllAccountsUnavailable)`（显式锁定「池非空不返回空池哨兵」语义，比原计划断言更强）；工厂调用次数 == DefaultMaxHops 断言不变
- **Files modified:** failover_client_76_03_test.go
- **Verification:** TestFailover_MaxHops_CapsAtDefaultMaxHops PASS（工厂恰 10 次）
- **Committed in:** e0fbcc7

**3. [Rule 1 - Bug] 测试字面量 models.ADConfig{ID: ...} 编译失败（提升字段不可字面量初始化）**
- **Found during:** Task 3 RED 运行
- **Issue:** ADConfig.ID 是 BaseModel 嵌入提升字段，Go 复合字面量不能直接按名初始化
- **Fix:** `&models.ADConfig{BaseModel: models.BaseModel{ID: configID}}`
- **Files modified:** failover_client_76_03_test.go
- **Verification:** 编译通过，用例绿
- **Committed in:** e0fbcc7

---

**Total deviations:** 3 auto-fixed（1 × Rule 3 blocking，2 × Rule 1 bug）
**Impact on plan:** 全部为编译/正确性必需；生产改动严格限于 failover seam + 24 类型行 + 8 行纯委托方法；无 scope 膨胀。

## TDD 说明

Task 3 标记 `tdd="true"`：RED 先行实证——先写全部用例（含 searchUsersFn 引用）跑 `-run TestFailover` 得到编译失败（`unknown field searchUsersFn` / `mock.searchUsersCalls undefined`，即未实现前的失败信号）；GREEN 步为 mock 增字段/计数/优先分支后全绿。因全部改动均在 _test.go 文件内（mock 能力本身是测试基建，无生产 GREEN 步可拆），按 76-01 Task 2 同例单 commit 落地，未拆 RED/GREEN 两 commit（避免提交不可编译的中间态）。plan 非 `type: tdd`，无 plan 级三段门。

## Verification Results（plan 收尾门全项）

- `go build ./...` → **PASS**（exit 0）
- 闭包零残留门 `grep -rnE "func\((client|ldapClient) \*(addomain\.)?LDAPClient\)" internal/services/addomain/ internal/scheduler/` → **PASS**（零命中，覆盖 client/ldapClient 两参数名）
- 辅助签名零残留门 `grep -rnE "\) (syncDeptTree|moveUserToNewOU|syncUserAttributes|moveSingleUserToOU)\(ctx context\.Context, ldapClient \*LDAPClient"` → **PASS**（零命中）
- `go test -count=1 ./internal/services/addomain/... ./internal/scheduler/... ./internal/core/security/...` → **PASS**（三包全绿；core/security 26.5s 全量含 ad_authenticator PickFirstConnect 消费方）
- `go vet ./internal/services/addomain/`（+ scheduler）→ **PASS** 干净
- `bash scripts/check-ci-local.sh backend` → **PASS EXIT=0**（lint 0 issues + 全量测试 + coverage gate 全过，P1/P2 floor 无倒退）
- `git diff` 纪律审查 → **PASS**：9 个机械文件合计 25 处改动 = 24 类型行 + 1 委托调用行（group_sync_service.go:115，Deviation 1）；NewDeptToADSyncService/NewUserADSyncService 构造器与 struct 字段 ldap 保持 *LDAPClient 未动；PickFirstConnect 签名仍返回 *LDAPClient

## User Setup Required

None - 全部进程内（sqlite :memory: + mock 工厂）。

## Next Phase Readiness

- Phase 78 BLOCK-05（addomain ≥70%）的接口 seam 全部就绪：mock 形态（preset + fn + 计数）、clientFactory 注入、setupTestPool/insertAccount helper 均可直接复用
- 窄边界成文：ldap_client.go:243-295 的 searchWithPaging 分页降级与 wire 级 LDAP server（vjeantet/ldapserver）不在本 phase，Phase 78 按缺口决定（objective 边界声明兑现）
- PickFirstConnect（ad 认证链）仍是具体类型——Phase 78 若需 mock 该路径需单独决策

## Self-Check: PASSED

- 文件存在：failover_client_76_03_test.go（146 行 ≥ 60 门槛）/ ldap_iface.go（contains DNExists + SearchWithRequest）/ failover_client.go（contains clientFactory）全部 FOUND
- 提交存在：aa6e31c / dba8065 / e0fbcc7 全部在 git log 中 FOUND
- 工作树干净（coverage.out 生成物已清理）

---

*Phase: 76-test-doubles*
*Completed: 2026-08-23*
