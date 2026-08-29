---
phase: 76-test-doubles
verified: 2026-08-23T09:25:00Z
status: passed
score: 6/6 must-haves verified
overrides_applied: 0
human_verification:
  - test: "Push 到 origin 并确认 ubuntu CI 绿(SC1 后半:Windows 本地 + ubuntu CI 双绿)"
    expected: "ci.yml 在 ubuntu runner 上全绿(无 Docker);87 个本地 commit(含 Phase 76 全部 13 个提交)推送后 `gh run watch` 确认"
    why_human: "main 领先 origin/main 87 个 commit,Phase 76 代码尚未到达 CI;最近一次 CI run(2026-08-23T01:30Z)早于 Phase 76 全部提交。推送是需要人/orchestrator 触发的动作,CI 结果是外部服务观测,本地无法程序化验证"
    resolved: "2026-08-23T09:25:00Z — push d156174..bddb2fc 完成;ci.yml run 32629496736 (headSha bddb2fc) conclusion=success,backend(frontend)双双 success、全程无 Docker。Windows 本地 + ubuntu CI 双绿达成"
---

# Phase 76: 测试基建落地 (test doubles + 注入缝) Verification Report

**Phase Goal:** 引入 miniredis/httpmock 两个 test-only 依赖并落地全部注入缝(Driver 工厂 / LDAPClientIface / re-exec stub / AST 守护),零 Docker、Windows 本地与 ubuntu CI 同构,使 5 个结构阻塞包的覆盖工作不再被"没有真实依赖基建"卡死。
**Verified:** 2026-08-23T08:37:27Z
**Status:** human_needed(本地可验证项全部实证通过;唯一悬置项为 ubuntu CI 双绿,代码未推送,无法本地观测)
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths

ROADMAP 5 条 Success Criteria 逐项核验;SC1 拆为本地半(CI1a)与 CI 半(SC1b)以便精确计分:

| # | Truth | Status | Evidence |
| --- | --- | --- | --- |
| 1a | go.mod 仅新增 2 个 test-only 依赖(v2.38.0 / v1.4.2,行尾带 `// test-only (v1.27 D-02)`),生产依赖零版本变更,`go test ./...` Windows 本地全绿零 Docker | ✓ VERIFIED | go.mod:8 `miniredis/v2 v2.38.0 // test-only (v1.27 D-02)`、go.mod:17 `httpmock v1.4.2 // test-only (v1.27 D-02)`;diff(6faaf3c..HEAD)= direct +2 + indirect +1(gopher-lua)+ 2 处分类修正(go-sqlite indirect→direct、x/net direct→indirect,版本零变更,与 SUMMARY 预写口径一致);orchestrator 实测 `go build ./...` EXIT=0 + `go test ./...` EXIT=0(72 包);本 verifier 独立复跑 build EXIT=0 + 8 个目标包全绿;全部替身进程内(miniredis/httpmock/re-exec),本机无 Docker 参与即绿 |
| 1b | ubuntu CI 同构双绿 | ? UNCERTAIN | **非代码缺陷,系未推送**:main 领先 origin/main 87 commit,Phase 76 全部提交未达远端;`gh run list` 最近 run(2026-08-23T01:30Z)早于 Phase 76 提交窗口(05:59Z–08:28Z)。平台分歧根源已结构性消除(echo→re-exec `os.Args[0]`、miniredis 进程内),但"CI 绿"这一事实需推送后观测 → 转人工验证 |
| 2 | miniredis 三坑防护(R-1 FastForward / R-2 INFO 降级 err==nil+key 存在性 / R-3 SETINFO 兼容)+ pkg/cache 命令面(INCR/EXPIRE/SCAN/HSET/INFO/EVAL)冒烟实证 | ✓ VERIFIED | `pkg/cache/redis_miniredis_76_01_test.go`(218 行):TestRedisTTLFastForward(:137,`mr.FastForward(11*time.Second)` 后断言 ErrNotFound)、TestRedisGetStatsDegraded(:158,断言 err==nil + key_count 真实 + keyspace_info 键存在、降级字段缺席——与 ROADMAP"降级为 err==nil + key 存在性"措辞吻合)、newMiniredisCache(:34,RunT→SplitHostPort→NewRedisCache 构造器 PING 即 R-3 哨兵);命令面:Increment/IncrementBy/IncrementWithExpire(INCR+EVAL :82-87,:197)、Expire/TTL(:142,:144)、Keys(SCAN :123)、HSet 家族(:97-107)、GetStats(INFO :165)。本 verifier 复跑 `go test -run TestRedis ./pkg/cache/` PASS |
| 3 | ScrapliWrapper 可注入 Driver 工厂入口(生产路径行为不变)+ FileTransport 注入 Open/SendCommand 链 | ✓ VERIFIED | scrapli_wrapper.go:115 `var newNetworkDriver = func(platformName interface{}, host string, opts ...util.Option)`,:172/:226 两构造器均经工厂(pattern `newNetworkDriver(platformName` 命中);错误文案"创建平台实例失败"(:122)/"获取网络驱动失败"(:127)在工厂内部、byte 不变,`platform.NewPlatform(` 代码直调仅 1 处(:116,:92 为既有注释);driver_factory_76_02_test.go(116 行)含 orig→t.Cleanup 恢复→覆盖纪律、WithTransportType(transport.FileTransport)(:71)、经公开构造器 NewScrapliWrapper→Open→SendCommand 全链(:88-:99)、断言 Contains "Huawei"(:111);fixture 首行 `<dummy-host>`;全文件无 t.Parallel。本 verifier 复跑 `go test ./internal/device/` 全绿(2.0s) |
| 4 | LDAPClientIface mock 补全(walk/分页)+ FailoverClient 顺序遍历/maxHops 接口驱动;agent 子进程 stub 统一 TestHelperProcess re-exec,echo 分歧根源清除 | ✓ VERIFIED | **前半**:ldap_iface.go 20 方法(16+3 计划内+SearchWithRequest 编译门强制,后者为纯委托 ldap_client.go:221-223 `return c.conn.Search(searchRequest)`);双侧编译期断言(ldap_iface.go:47 与 ldap_client_mock_test.go:150);闭包零残留门 `grep -rnE "func\((client|ldapClient) \*(addomain\.)?LDAPClient\)"` EXIT=1(零命中)、辅助签名零残留门 EXIT=1;failover_client.go:25 clientFactory 字段、:34 newClient、:46 `operation func(client LDAPClientIface) error`、:99 PickFirstConnect 仍返回 *LDAPClient;failover_client_76_03_test.go(146 行)三用例:顺序遍历(:27,factoryCalls==2 + failure_count 落库断言)、maxHops(:80,DefaultMaxHops 常量断言 + NotErrorIs 空池哨兵)、多批次 walk(:114,searchUsersFn 优先于 searchUsersRes)。**后半**:subprocess_stub_test.go(174 行)TestHelperProcess 守卫第一行(:26)、四形态分支各达 os.Exit(:31/:34/:43/:51)、sigterm-armed 就绪标记(:40);subprocess_pgroup_test.go `grep '"echo"'` EXIT=1(零命中),5 处替换 = 组 A newCommand 直连(:28/:90)+ 组 B t.Setenv+runCommand/runCommandOutput(:45/:57/:73,生产函数覆盖保留);生产三文件(subprocess.go/sysproc_*.go)自初始 commit 零改动。本 verifier 复跑三包(addomain/scheduler/core-security 26.4s)+ agent/server + operations 全绿 |
| 5 | AST 守护上线:生产 .go 禁引用 *ForTesting,三层隔离契约(编译器+命名+AST) | ✓ VERIFIED | for_testing_guard_test.go(127 行):WalkDir("../..")(:58)+ 点前缀 SkipDir 与根豁免(:65-:76)+ 白名单 filepath.Rel 归一化匹配 e2e_helpers.go(:85-:87)+ CallExpr.Fun/SelectorExpr.Sel 双检测分支(:98-:109,fset.Position 报 file:line:col)+ scannedFiles==0 防呆(:118);三层契约头注释(:14-:33)。本 verifier 复跑:`712 production .go files scanned, 0 violations` PASS(0.28s,与 SUMMARY 记载逐字一致);注毒双毒株自证记录在 76-05-SUMMARY(输出含 file:line+符号名,毒株已还原——`git diff` 干净),检测分支逻辑经代码级复核属实 |

**Score:** 5/6 truths verified(SC1b 转人工)

**执行偏差裁决**(orchestrator 列出的 4 类偏差,逐项核验后均判定为正确解决,不影响 must_haves):

- **76-03 接口 19→20**:SearchWithRequest 为 2 行纯委托(零逻辑),触发 PLAN Task 2 自带的编译门逃生条款(group_sync_service.go:115 裸字段访问 `client.conn.Search` 接口化后不可达);seam 严格宽于计划,意图(接口驱动 failover 闭包全量)完整保留。**接受**。
- **76-01 R-2 断言改写**:按 pinned v2.38.0 真实降级行为(INFO section 返回错误→GetStats 静默跳过)断言;与 ROADMAP SC2 的降级措辞(err==nil + key 存在性)精确吻合,且比计划原稿(断言非空,必红)更强。**接受**。PoC 私有令牌桶(SUMMARY Deviation 2)为计划明文许可手法,httpClient 未动,const-URL 拦截示范意义不变。
- **76-05 白名单 Rel 归一化 + 根豁免**:PLAN 字面 `ToSlash(path)` 直比在带 `../..` 前缀的 WalkDir 路径下永不命中(白名单会失效);`path == repoRoot` 豁免防 `".."` 点前缀跳过整仓。两处修正均为实现级必要,守护语义不变——712 文件扫描零违规即为白名单正确工作的实证(否则 e2e_helpers.go:34 必报)。**接受**。
- **Code review 3 WARNING**:WR-01(HKeys 短字段名绕开存量生产缺陷)/WR-02(mock 自测试命名)/WR-03(sqlite rowid 行序依赖)——全部为测试可靠性类,无一击穿 must_haves(SC 逐条对照见 Truths 表);生产漂移审计(零行为漂移)经本 verifier 独立复核确认:phase 区间生产 .go 改动恰为声明的 12 文件,类型行机械替换。WR-01 揭示的 HKeys 存量缺陷系 phase 之前已存在(redis.go 前缀裁剪逻辑),phase 的零漂移约束下不动 redis.go 是正确执行而非遗漏 → 建议单独立项修复(见 Anti-Patterns 表)。

### Required Artifacts

| Artifact | Expected | Status | Details |
| -------- | -------- | ------ | ------- |
| `pkg/cache/redis_miniredis_76_01_test.go` | miniredis 冒烟+三坑防护(≥80 行) | ✓ VERIFIED | 218 行;三坑具名用例 + 命令面全覆盖;无 time.Sleep/DisableIdent |
| `internal/services/operations/geocoding_httpmock_76_01_test.go` | httpmock PoC(≥40 行) | ✓ VERIFIED | 55 行;Activate(t)+RegisterResponder+生产构造器+InDelta;无 svc.httpClient 白盒 |
| `internal/device/scrapli_wrapper.go` | `var newNetworkDriver` 工厂 | ✓ VERIFIED | :115 定义;两构造器经工厂;错误文案 byte 不变 |
| `internal/device/driver_factory_76_02_test.go` | FileTransport 注入演示(≥50 行) | ✓ VERIFIED | 116 行;t.Cleanup 恢复 + 全链驱动 + Contains "Huawei" |
| `internal/device/testdata/huawei_vrp_open.fixture` | Open 场景 fixture | ✓ VERIFIED | 7 行,首行 `<dummy-host>` |
| `internal/services/addomain/ldap_iface.go` | 接口扩展(contains DNExists) | ✓ VERIFIED | 20 方法;DNExists:40;SearchWithRequest:44;双侧断言 |
| `internal/services/addomain/failover_client.go` | clientFactory 注入字段 | ✓ VERIFIED | :25 字段 + :34 newClient + :46 接口化签名 |
| `internal/services/addomain/failover_client_76_03_test.go` | 顺序遍历/maxHops 测试(≥60 行) | ✓ VERIFIED | 146 行;三具名用例零真实网络 |
| `internal/agent/server/subprocess_stub_test.go` | TestHelperProcess helper(≥80 行) | ✓ VERIFIED | 174 行;守卫第一行+四形态+os.Exit |
| `internal/agent/server/subprocess_pgroup_test.go` | 5 处 echo 全替换 | ✓ VERIFIED | `grep '"echo"'` 零命中;两组改法落地 |
| `internal/device/for_testing_guard_test.go` | AST 守护(≥60 行) | ✓ VERIFIED | 127 行;双检测分支+双防呆+三层契约 |

### Key Link Verification

| From | To | Via | Status | Details |
| ---- | -- | --- | ------ | ------- |
| redis_miniredis_76_01_test.go | pkg/cache NewRedisCache | miniredis.RunT(t).Addr() 拆 Host/Port → 构造器 PING | ✓ WIRED | :36-:48 同 helper 内 RunT→SplitHostPort→NewRedisCache 链路完整(R-3 哨兵) |
| geocoding_httpmock_76_01_test.go | baiduGeocodingAPIURL | httpmock.Activate(t) + RegisterResponder | ✓ WIRED | :34-:38,URL 引用常量,拦截命中计数==1 断言(:54) |
| NewScrapliWrapper / NewScrapliWrapperWithPort | var newNetworkDriver | `d, err := newNetworkDriver(platformName, device.IPAddress, opts...)` | ✓ WIRED | scrapli_wrapper.go:172/:226 两调用点,pattern 命中 |
| driver_factory_76_02_test.go | FileTransport 选项 | WithTransportType(transport.FileTransport) 覆盖工厂 | ✓ WIRED | 测试 :67-:75,回放链 Open→SendCommand 实跑通过 |
| FailoverClient.ExecuteWithFailover | LDAPClientIface | `operation func(client LDAPClientIface) error` | ✓ WIRED | failover_client.go:46;全仓闭包零残留(grep 门 EXIT=1) |
| failover_client_76_03_test.go | mockLDAPClient | `fc.clientFactory = ` 同包白盒赋值 | ✓ WIRED | 测试 :42/:91,按 Username 分流 mock |
| subprocess_pgroup_test.go | TestHelperProcess | os.Args[0] + `-test.run=^TestHelperProcess$` + GO_WANT_HELPER_PROCESS=1 | ✓ WIRED | 组 A :28/:90 + 组 B t.Setenv :45/:57/:73 |
| for_testing_guard_test.go | 全仓生产 .go | WalkDir("../..") + parser.ParseFile + ast.Inspect | ✓ WIRED | :58-:112,712 文件实测扫描 |
| 白名单 | e2e_helpers.go | repo 相对路径精确匹配 | ✓ WIRED | :85-:87 filepath.Rel 归一化;零违规即生效实证 |

### Data-Flow Trace (Level 4)

本 phase 全部交付物为测试基建(非渲染组件),静态 data-flow 追踪不适用;以**行为级实证替代且更强**——所有"数据流"即替身驱动真实生产路径,由本 verifier 在本进程实际执行(见下表),每一链路均为真实运行而非静态推断:

| Artifact | 数据变量/链路 | 实跑结果 | Status |
| -------- | -------------- | -------- | ------ |
| redis_miniredis_76_01_test.go | NewRedisCache→miniredis TCP 真链路 | `ok pkg/cache 0.724s` | ✓ FLOWING |
| driver_factory_76_02_test.go | FileTransport fixture 回放→Open/SendCommand | `ok internal/device 2.003s`(整包) | ✓ FLOWING |
| failover_client_76_03_test.go | clientFactory mock→顺序遍历/maxHops→sqlite 落库断言 | 3 用例全 PASS(0.01s 级) | ✓ FLOWING |
| subprocess_stub_test.go | 测试二进制 re-exec 子进程输出捕获 | Default/StdoutFlood/StdinClose PASS;IgnoreSigterm SKIP(Windows 预期) | ✓ FLOWING |
| for_testing_guard_test.go | 全仓 712 生产文件 AST 流 | PASS 0 violations | ✓ FLOWING |
| geocoding_httpmock_76_01_test.go | DefaultTransport 拦截→坐标解析 | `ok internal/services/operations 0.184s` | ✓ FLOWING |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
| -------- | ------- | ------ | ------ |
| miniredis 冒烟含 R-1/R-2/R-3 | `go test -count=1 -run 'TestRedis' ./pkg/cache/` | ok 0.724s | ✓ PASS |
| AST 守护全仓扫描 | `go test -count=1 -run 'TestNoProductionForTestingReferences' -v ./internal/device/` | 712 files, 0 violations, PASS 0.28s | ✓ PASS |
| FailoverClient 接口驱动三语义 | `go test -count=1 -run 'TestFailover' -v ./internal/services/addomain/` | 5 用例(3 新+2 既有)全 PASS | ✓ PASS |
| re-exec 守卫+形态自验 | `go test -count=1 -run 'TestSubprocessStub\|TestHelperProcess' -v ./internal/agent/server/` | 4 PASS + 1 SKIP(Windows 预期) | ✓ PASS |
| geocoding httpmock PoC | `go test -count=1 -run 'TestGeocoding' ./internal/services/operations/` | ok 0.184s | ✓ PASS |
| device 包全量(生产路径不变) | `go test -count=1 ./internal/device/` | ok 2.003s | ✓ PASS |
| 76-03 三包回归 | `go test -count=1 ./internal/scheduler/... ./internal/core/security/...` | ok 0.269s / ok 26.423s | ✓ PASS |
| 全仓编译 | `go build ./...` | EXIT=0 | ✓ PASS |
| 闭包/辅助签名零残留双 grep 门 | 见 Truth #4 两条 grep | 双 EXIT=1(零命中) | ✓ PASS |
| echo 零残留 | `grep -n '"echo"' subprocess_pgroup_test.go` | EXIT=1 | ✓ PASS |

### Probe Execution

| Probe | Command | Result | Status |
| ----- | ------- | ------ | ------ |
| (无 probe-*.sh) | `find scripts -path '*/tests/probe-*.sh'` | 零命中 | SKIPPED(仓库无 probe 约定;phase 门为 `scripts/check-ci-local.sh backend`,orchestrator 已实测 EXIT=0 且各 SUMMARY 记录在案,本 verifier 以 8 项定向复跑三角验证) |

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
| ----------- | ---------- | ----------- | ------ | -------- |
| INFRA-01 | 76-01 | miniredis/httpmock test-only 依赖 + 三坑防护 | ✓ SATISFIED | go.mod:8/:17 带注释;三坑具名用例;tidy 保活(import 锚点 geocoding 测试实际存在) |
| INFRA-02 | 76-02 | ScrapliWrapper 可注入 Driver 工厂 | ✓ SATISFIED | newNetworkDriver 工厂 + FileTransport 注入实证 + 生产路径零变化 |
| INFRA-03 | 76-03 | LDAPClientIface 扩展 stub 主推线(零新依赖) | ✓ SATISFIED | 接口 20 方法 + clientFactory + 三组接口驱动测试;go.mod 零新生产依赖 |
| INFRA-04 | 76-04 | TestHelperProcess re-exec 统一 | ✓ SATISFIED | 守卫+四形态落地;5 处 echo 清零;生产覆盖保留 |
| INFRA-05 | 76-05 | ForTesting 治理 + AST 守护 | ✓ SATISFIED | 守护测试 712 文件零违规 + 注毒双毒株自证记录 |

**孤儿需求检查:** REQUIREMENTS.md 中映射到 Phase 76 的需求恰为 INFRA-01..05 五项,与 5 个 plan 的 requirements 字段一一对应,无孤儿。

**备注(信息级,orchestrator 收尾时处理):** REQUIREMENTS.md 追踪表 INFRA-01..05 仍标 `Pending`、checkbox 未勾(对照:Phase 75 已在验证后的 ship commit 中同步为 Complete)。属 phase 收尾文档同步步骤,不影响代码层达成——按 Phase 75 先例(b478aed)在 phase 76 ship commit 中一并同步即可。

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
| ---- | ---- | ------- | -------- | ------ |
| pkg/cache/redis_miniredis_76_01_test.go | 96 | HKeys 用短字段名(f1/f2)绕开存量生产缺陷(字段名被 prefixLen 裁剪)——WR-01 | ⚠️ Warning | 测试自述规避;HKeys 在覆盖率报告显示"已验证"而真实长字段名会被截断。系 phase 前已存在的生产缺陷(redis.go:405-:417),本 phase 零漂移约束下不动 redis.go 为正确执行。**建议单独立项修复**(review 已给修复方案);当前仓库内 HKeys 无生产调用方,风险潜伏态 |
| internal/services/addomain/failover_client_76_03_test.go | 31 | 顺序遍历依赖 sqlite rowid 插入序(ListAvailable 无 ORDER BY)——WR-03 | ⚠️ Warning | 引擎实现细节依赖;当前全绿(本 verifier 复跑通过),若 ListAvailable 未来加排序会假红。建议后续改次序无关断言 |
| internal/services/addomain/ldap_client_mock_test.go | 154-292 | TestLDAPClient_* 测 mock 自身,归因误导——WR-02 | ℹ️ Info | 表观测试数抬高;对生产行为零覆盖但无害;建议改名 TestMockLDAPClient_* |
| internal/agent/server/subprocess_stub_test.go | 61,63 | 注释含 "echo" 字面量 | ℹ️ Info | 纯文档说明(解释被替换物);验收 grep 门只针对 pgroup 文件,不构成违规 |
| internal/services/addomain/sync.go | 209 | `size:XXX`(gorm 标签字面量) | ℹ️ Info | phase 前已存在(6faaf3c 基线含此行),非本 phase 引入,非债务标记 |

**债务标记门:** phase 全部 22 个改动文件中零 `TBD`/`FIXME`/`XXX` 债务标记(sync.go:209 为预存 gorm 标签字面量,非标记)。

### Human Verification Required

### 1. ubuntu CI 同构双绿(SC1 后半)

**Test:** 推送本地 main(领先 origin 87 commit,含 Phase 76 全部 13 个提交:e3e9b04/eb08571/6903703/288403d/d9e9d84/af98afb/aa6e31c/dba8065/e0fbcc7/298ee21/a812b0c/e7f838c/0ea3ab7——本 verifier 已逐一确认存在于 git)到 origin/main,然后 `gh run watch` 盯 ci.yml。
**Expected:** ci.yml 在 ubuntu runner 上全绿(gate 55.5 floor 不倒退),全程无 Docker;重点观察 pkg/cache / internal/device / internal/services/addomain / internal/agent/server 四个新测试包的跨平台表现(re-exec 的 os.Args[0] 与 TestHelperProcess 在 linux 下应与 Windows 同构;TestSubprocessStub_IgnoreSigterm 在 linux 会实际执行而非 SKIP)。
**Why human:** 推送是动作决策(87 commit 批次归属 orchestrator/用户);CI 结果是外部服务观测,本地无法程序化验证。这是本 phase 唯一无法在本地闭合的 Success Criteria 分量。

### Gaps Summary

无代码层缺口。5 项 INFRA 需求对应的全部注入缝、替身、守护均已在代码库落地并经本 verifier 独立复跑实证(8 项定向测试 + 双 grep 门 + build,全部通过;与 orchestrator 预采集的全量 `go test ./...` EXIT=0/72 包、AST 守护 712/0、各 plan 收尾门 EXIT=0 三方吻合)。

执行偏差 4 类(接口 20 vs 19 / R-2 按实况改写 / 白名单 Rel 归一化+根豁免 / review 3 WARNING)逐项核验后均判定为正确解决,无一击穿 must_haves。

唯一悬置项:SC1 的"ubuntu CI 双绿"分量因代码未推送而无法观测(非缺陷),已列 Human Verification;Code review 的 WR-01(HKeys 存量生产缺陷)建议在后续独立 fix 项处理,不阻塞本 phase。

---

_Verified: 2026-08-23T08:37:27Z_
_Verifier: Claude (gsd-verifier)_
