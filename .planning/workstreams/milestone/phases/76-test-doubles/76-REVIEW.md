---
phase: 76-test-doubles
reviewed: 2026-08-23T08:28:54Z
depth: standard
files_reviewed: 22
files_reviewed_list:
  - internal/agent/server/subprocess_pgroup_test.go
  - internal/agent/server/subprocess_stub_test.go
  - internal/device/driver_factory_76_02_test.go
  - internal/device/for_testing_guard_test.go
  - internal/device/scrapli_wrapper.go
  - internal/device/testdata/huawei_vrp_open.fixture
  - internal/scheduler/dept_sync_tasks.go
  - internal/services/addomain/dept_sync_service.go
  - internal/services/addomain/failover_client.go
  - internal/services/addomain/failover_client_76_03_test.go
  - internal/services/addomain/group.go
  - internal/services/addomain/group_management_service.go
  - internal/services/addomain/group_sync_service.go
  - internal/services/addomain/ldap_client.go
  - internal/services/addomain/ldap_client_mock_test.go
  - internal/services/addomain/ldap_iface.go
  - internal/services/addomain/sync.go
  - internal/services/addomain/user.go
  - internal/services/addomain/user_ad_sync_service.go
  - internal/services/operations/geocoding_httpmock_76_01_test.go
  - pkg/cache/cache_74_08_test.go
  - pkg/cache/redis_miniredis_76_01_test.go
findings:
  critical: 0
  warning: 3
  info: 7
  total: 10
status: fixed
---

# Phase 76: Code Review Report

**Reviewed:** 2026-08-23T08:28:54Z
**Depth:** standard
**Files Reviewed:** 22
**Status:** issues_found

## Summary

本阶段为测试基建阶段（test doubles + 注入缝），审查包含三类工作：(1) `internal/device/scrapli_wrapper.go` 工厂 var 抽取；(2) `internal/services/addomain/*` 接口扩容（16→20 方法）+ FailoverClient `clientFactory` 注入字段 + 24 处机械类型替换；(3) `internal/scheduler/dept_sync_tasks.go` 闭包签名接口化。其余 17 个文件为测试/fixture。

**生产代码漂移审计（本阶段硬性要求，逐行 diff 比对 `6faaf3c..HEAD`）：通过，零行为漂移。**

- `scrapli_wrapper.go`：`newNetworkDriver` 工厂将原先两处内联的 `platform.NewPlatform` + `GetNetworkDriver` 原样收敛，错误包装文案（"创建平台实例失败"/"获取网络驱动失败"）与包装顺序 byte 不变，两个构造器的调用点改为 `return nil, err` 直传，语义等价。
- addomain 全部替换均为 `*LDAPClient` → `LDAPClientIface` 的纯类型行替换（group.go ×3、group_management_service.go ×4、group_sync_service.go ×2、sync.go ×1、user.go ×4、user_ad_sync_service.go ×7、dept_sync_service.go ×2、scheduler ×1，共 24 处）；新增 `SearchWithRequest` 为 `return c.conn.Search(searchRequest)` 纯转发，与替换前 `client.conn.Search` 直调等价。
- `go build ./...` 通过；`internal/device`、`internal/services/addomain`、`pkg/cache`、`internal/agent/server`、`internal/services/operations`（定向）测试全部实测通过；`go vet` 干净。

未发现 BLOCKER 级问题。3 个 WARNING 均为测试可靠性/覆盖归因问题，其中 WR-01 揭示了一个被测试刻意绕开的真实存量生产缺陷（`RedisCache.HKeys` 字段名被前缀长度截断），建议单独立项修复。

范围观察（非缺陷）：diff 区间内还有 `go.mod`/`go.sum`（新增 `miniredis`、`httpmock` test-only 依赖及 tidy 副作用，与本阶段一致）与 `xingran-react-frontend/package.json`、`vitest.config.ts`（属并行 frontend-coverage workstream）变更，均不在本次审查文件清单内。

## Structural Findings (fallow)

本审查未提供 structural_findings 预置载荷，无此项。

## Narrative Findings (AI reviewer)

### Warnings

#### WR-01: 测试刻意绕开 `RedisCache.HKeys` 的真实生产缺陷（字段名被前缀长度截断）

**File:** `pkg/cache/redis_miniredis_76_01_test.go:96-107`
**Issue:** 测试注释自述"字段名保持短于前缀长度，规避 HKeys 对字段名的历史前缀裁剪行为"。生产代码 `pkg/cache/redis.go:405-417` 的 `HKeys` 对返回的**字段名**（hash field，本身从不携带 key 前缀）执行 `key[len(prefix)+1:]` 截断：前缀 `"xingran"` 时 `prefixLen=8`，任何长度 >8 的字段名都会被裁掉前 8 个字符（如 `"longfieldname"` → `"ame"`）。本测试用 `"f1"`/`"f2"`（长度 2 < 8，`len(key) > prefixLen` 为假）恰好躲过裁剪分支，使 HKeys 在覆盖率报告中显示为"已验证通过"，而真实世界字段名几乎必然超过 8 字符，该路径一旦被业务调用即产生数据损坏。测试将一个 live bug 洗白成 green。
**Fix:** 修复生产代码（推荐，另开 fix 项——本阶段零漂移约束下不动 redis.go）：删除 HKeys 中对字段名的前缀裁剪逻辑（prefix 只属于 hash key，由 `buildKey` 处理，字段名无需也不应裁剪）：

```go
func (r *RedisCache) HKeys(ctx context.Context, key string) ([]string, error) {
	// 字段名不携带 key 前缀，直接返回；原先按 prefixLen 裁剪字段名是错误逻辑
	return r.client.HKeys(ctx, r.buildKey(key)).Result()
}
```

本阶段内的最低限度补救：在测试中显式断言该缺陷行为并标注跟踪项（长字段名被截断），而非静默选短字段名；同时确认 `redis.go:980` 的两层缓存透传无生产调用方（当前仓库内无业务调用，风险暂为潜伏态）。

#### WR-02: `TestLDAPClient_*` 系列测试的是 mock 自身而非生产 `LDAPClient`，造成覆盖归因误导

**File:** `internal/services/addomain/ldap_client_mock_test.go:154-292`
**Issue:** 9 个测试（`TestLDAPClient_Connect_Success`、`TestLDAPClient_SearchGroups_ReturnsEntries`、`TestLDAPClient_CreateGroup_Error` 等）全部只构造 `mockLDAPClient` 并断言它返回预设值——被测对象是 `_test.go` 里的 stub，生产 `LDAPClient` 的 `Connect`/`SearchGroups`/`CreateGroup` 一行都未执行。测试名以 `TestLDAPClient_` 开头，读测试报告的人会误以为真实客户端已被验证；这属于"用 mock 测 mock"的循环验证，对生产行为零覆盖但抬高了表观测试数。
**Fix:** 重命名为 `TestMockLDAPClient_*` 以如实归因；或删除这批自证测试，将断言合并进真正驱动生产代码的用例（如经 `FailoverClient`/service 层 mock 边界的测试）。生产 `LDAPClient` 的可测路径（如 `extractRDNFromDN`、`parseGroupTypeFromLDAP`、`DNExists` 的错误分支）应通过纯函数/接口缝直接覆盖。

#### WR-03: `TestFailover_SequentialTraversal_StopsOnFirstSuccess` 依赖未定义的 sqlite 返回行序

**File:** `internal/services/addomain/failover_client_76_03_test.go:31-58`
**Issue:** 测试断言"工厂恰好构造 2 个客户端（username-0 失败 → username-1 成功）"，其前提是 `ListAvailable`（`account_pool.go`，GORM `Find` 无 `ORDER BY`）按插入顺序（sqlite rowid 扫描序）返回账号。这是引擎实现细节而非 SQL 语义保证：一旦 `ListAvailable` 将来加排序、驱动/GORM 版本变化、或测试改跑 Postgres，username-1 可能排首——operation 首次即成功，`factoryCalls==1`，测试假红。测试注释已自认此前提，但没有消除它。
**Fix:** 将断言改为次序无关（推荐，零生产改动）：工厂按 username 记录每个账号是否被尝试，断言"被尝试的账号集合 = {username-0, username-1} 且恰一个成功"；或断言 `factoryCalls <= 2` + `operationCalls == 1` + DB 中恰一行 `failure_count=1`。长期方案是给 `ListAvailable` 加确定排序（属生产变更，须另走漂移评审）。

### Info

#### IN-01: `mockLDAPClient.closeErr` 为死字段

**File:** `internal/services/addomain/ldap_client_mock_test.go:15`
**Issue:** `Close()` 无返回值，`closeErr` 声明后从未被任何方法读取，纯粹误导后续维护者以为 Close 错误可注入。
**Fix:** 删除该字段。

#### IN-02: `PickFirstConnect` 绕过 `clientFactory` 注入缝，seam 不对称

**File:** `internal/services/addomain/failover_client.go:117`
**Issue:** `ExecuteWithFailover` 走 `f.newClient(acct)`（支持测试注入），而 `PickFirstConnect` 仍硬编码 `NewLDAPClient(f.config, acct)`，且返回具体类型 `*LDAPClient`。ad_authenticator 的"绑管理员"路径因此无法用 mock 驱动，76-03 的注入缝只覆盖了一半调用面。
**Fix:** 后续阶段将 `PickFirstConnect` 改走 `f.newClient` 并返回 `LDAPClientIface` + 账号（调用方仅需 `Conn()` 时可加接口方法）。本阶段零漂移约束下可接受现状，建议在 76-PATTERNS.md 记录该已知缺口。

#### IN-03: `TestRunCommand_ContextCancellation` 无法因行为回归而失败

**File:** `internal/agent/server/subprocess_pgroup_test.go:72-84`
**Issue:** 预取消 ctx 下 `err == nil` 与 `err != nil` 均被接受（注释自述"Tolerant by design"）。若 `runCommand` 的 ctx 取消/组击杀逻辑被整体破坏（如不再用 `exec.CommandContext`），该测试依然通过——它只能防 panic/hang，不能防取消失效。
**Fix:** 至少收紧为二选一语义断言：err 非 nil 时断言 `errors.Is(err, context.Canceled)`；或用一个"忽略取消信号仍存活会超时"的 stub shape 反向验证（linux 下可用 ignore-sigterm 变体 + 短超时）。

#### IN-04: 注入的工厂丢弃 `platformName`/`opts`，opts 构造回归不可见

**File:** `internal/device/driver_factory_76_02_test.go:67-80`
**Issue:** 替换工厂签名含 `_ interface{}, _ string, _ ...util.Option`，全部弃用。若有人删掉 `NewScrapliWrapper` 里的 cipher 列表、Telnet/SSH 分支或 `WithAuthPassword`，该测试照常通过——它只验证"工厂可被替换 + 全链可回放"，不验证构造器喂给工厂的实参。
**Fix:** 在工厂内补两条廉价断言：`platformName == "huawei_vrp"`（经 `platformIdentifier(VendorHuawei)` 真实映射），以及 `len(opts) >= 4`（username/password/NoStrictKey/transport 至少在列）。

#### IN-05: `mockLDAPClient` 调用计数无同步保护

**File:** `internal/services/addomain/ldap_client_mock_test.go:49-57`
**Issue:** 计数字段（`connectCalls`、`searchUsersCalls` 等）是裸 int。生产路径 `SyncManagersToAD`/`BatchSyncUsersToAD` 会在 errgroup（`MaxConcurrentADSync=3`）里并发调用 `UpdateUserAttribute` 等方法；未来任何用该 mock 驱动这两条路径的测试在 `-race` 下必炸。
**Fix:** 计数改 `atomic.Int64`，或给将来并发使用的方法加 `sync.Mutex`；至少在 mock 头部注释标注"非并发安全，勿用于 errgroup 路径"。

#### IN-06: miniredis 降级断言将依赖 v2.38.0 的怪癖固化为测试契约

**File:** `pkg/cache/redis_miniredis_76_01_test.go:173-189`
**Issue:** 断言 `redis_version`/`used_memory` **缺席**、`keyspace_info == ""`——这些是 miniredis v2.38.0 "section not supported" 的具体表现。miniredis 未来补齐 INFO 支持时，生产代码无恙而这些断言翻红（失败方向是误报而非漏报，尚可接受，但会被误读为生产回归）。
**Fix:** 在断言处注明"miniredis >= v2.39 若补齐 INFO server 支持需同步更新本用例（预期失败方向：假红）"，或改为宽松断言（存在即 int64、缺席即跳过）。

#### IN-07: `assertProcessGroupSet` 仅检查 `SysProcAttr != nil`，无法捕获平台标志丢失

**File:** `internal/agent/server/subprocess_pgroup_test.go:34-36`（helper 在 `subprocess.go:44-49`，存量代码）
**Issue:** `setProcessGroup` 赋的是共享包级变量 `&sysProcAttr` 的指针；断言只验证非 nil。若 `sysproc_linux.go` 丢失 `Setpgid: true` 或 `sysproc_windows.go` 丢失 `CREATE_NEW_PROCESS_GROUP`（例如重构为零值结构体），测试依旧通过——进程组隔离真正失效不会被检出。
**Fix:** 按平台断言具体字段（linux/darwin: `cmd.SysProcAttr.Setpgid == true`；windows: `CreationFlags & CREATE_NEW_PROCESS_GROUP != 0`），可用 `runtime.GOOS` 分支的期望文件（`//go:build`）实现。

## Fix Log

`/gsd:code-review --fix`（2026-08-23，范围：3 个 WARNING；Critical=0，Info 不在修复范围）：

- **WR-01** → `febceaa`（fix(cache)）：修复 `RedisCache.HKeys` 对字段名的错误前缀裁剪（`pkg/cache/redis.go`）；miniredis 测试改用长字段名（`department_name`/`display_order`，均长于 prefixLen=8）实证修复。`redis.go:980` 两层缓存透传确认为纯转发且全仓无 HKeys 生产调用方（仅接口定义/mock/测试），未改动。
- **WR-02** → `45ded45`（test(addomain)）：mock 自证测试 `TestLDAPClient_*` → `TestMockLDAPClient_*` 纯重命名（实际 10 个，非 9 个），断言零改动。
- **WR-03** → `504279d`（test(addomain)）：`TestFailover_SequentialTraversal_StopsOnFirstSuccess` 断言改为次序无关（按尝试序的 positional mock 工厂 + failure_count/last_success_at 计数断言，不再依赖 username 具体行序），零生产改动。

验证：`go build ./...`、`go vet ./pkg/cache/ ./internal/services/addomain/`、`go test ./pkg/cache/ ./internal/services/addomain/ ./internal/device/ -timeout 120s` 全部通过。

---

_Reviewed: 2026-08-23T08:28:54Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
