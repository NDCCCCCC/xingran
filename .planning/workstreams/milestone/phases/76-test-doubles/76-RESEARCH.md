# Phase 76: 测试基建落地 (test doubles + 注入缝) - Research

**Researched:** 2026-08-23
**Domain:** Go 测试替身(miniredis/httpmock)+ 依赖注入缝(Driver 工厂 / LDAPClientIface / re-exec stub)+ AST 守护测试
**Confidence:** HIGH(两个外部库经 registry + 官方 GitHub + 本地 module 源码三重实证;全部 seam 经本地代码实证;基线测试当天全绿)

## Summary

本 phase 的全部 5 项需求(INFRA-01..05)在代码层面的落点均已实证存在且可达。两个 test-only 依赖(miniredis/v2 v2.38.0、httpmock v1.4.2)经 Go module proxy 版本核实 + 官方 GitHub README 核实 + **本地下载 pinned 版本源码逐文件核实**,可直接落地。三项关键纠偏:(1) 里程碑 pitfalls 研究的 R-3("最新 miniredis 已把 CLIENT SETINFO 实现为 no-op")**与 v2.38.0 源码不符**——SETINFO 实际返回 `ERR unknown subcommand`;但 go-redis v9.7.0 的 `initConn` 对 SETINFO 错误**显式丢弃**(`_, _ = p.Exec(ctx)`),且 miniredis 的 `setDirty` 在非事务连接上是 no-op,故 **v2.38.0 + v9.7.0 组合开箱即用,无需 DisableIdentity 特殊处理**——兼容性实证本身就用 `NewRedisCache`(构造器内 PING 走完整 HELLO+SETINFO 握手)作为冒烟用例。(2) httpmock v1.4.x 的 `Activate(t)` 接受 `testing.TB` 自动清理,使用纪律比里程碑研究记载的更简。(3) `ExecuteWithFailover` 的 operation 闭包签名是 `func(*LDAPClient) error` 具体类型,共 **28 处生产调用点**分布在 ~9 个文件——接口化是本 phase 最大的机械改动面,闭包体内仅用 3 个非接口方法(UpdateGroupAttribute/CreateOU/DNExists),接口扩展只需加这 3 个。

注入缝设计方面:`NewScrapliWrapper`/`NewScrapliWrapperWithPort`(scrapli_wrapper.go:111/:174)尾部完全相同的 `platform.NewPlatform + GetNetworkDriver` 段是天然工厂抽取点,包级工厂 var 方案对 connection_pool.createConnection(:434-436)自动生效且零调用方改动;AST 守护可完全仿照 status_constants_test.go 的 go/parser 模式,但**必须跳过点前缀目录**——`.claude/worktrees/` 下有 6 份完整仓库拷贝,不跳过会误报。当天基线验证:`go test ./pkg/cache/... ./internal/agent/server/... ./internal/device/... ./internal/services/addomain/... ./internal/models/...` 全绿。

**Primary recommendation:** 按 76-01(miniredis+httpmock 落地 + pkg/cache 冒烟)→ 76-02(Driver 工厂 var)→ 76-03(LDAPClientIface 扩展 3 方法 + FailoverClient 工厂字段 + 28 处闭包签名机械替换)→ 76-04(TestHelperProcess helper + 替换 5 处 echo)→ 76-05(AST 守护)顺序落地;每步以"现有测试全绿 + go build ./..."为硬门。

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| INFRA-01 | 引入 miniredis/v2 (v2.38+) 与 httpmock (v1.4.x) test-only 依赖(MIT);redismock/testcontainers/gock 明令禁止;miniredis 三坑防护(TTL 用 FastForward / INFO 断言降级 / CLIENT SETINFO 兼容) | §Standard Stack + §Code Examples:两库 pinned 版本源码已本地核实(API 面/许可证/传递依赖);三坑的精确语义已从源码定级(R-3 实为"无需防护,冒烟即证");pkg/cache 命令面(INCR/EXPIRE/SCAN/HSET/INFO/EVAL/DBSIZE/PING)与冒烟写法已给出 |
| INFRA-02 | ScrapliWrapper 新增可注入 Driver 工厂入口(小重构,生产路径不变) | §Pattern 2:工厂 var 抽取点(scrapli_wrapper.go:149-162 与 :212-226 重复段)、签名(经 scrapligo v1.4.0 源码核实 `platform.NewPlatform(f interface{}, host string, opts ...util.Option)` definition.go:190 + `GetNetworkDriver()` :296)、对 pool 路径自动生效、覆盖恢复纪律 |
| INFRA-03 | addomain 走 LDAPClientIface 扩展 stub 主推线(零新依赖) | §Pattern 3:现接口 18 方法(ldap_iface.go);闭包体内非接口方法仅 3 个(UpdateGroupAttribute group.go:201 / CreateOU dept_sync_service.go:118 / DNExists user_ad_sync_service.go:187);FailoverClient 需 clientFactory 字段 + operation 签名改 LDAPClientIface;28 调用点/~9 文件清单;PickFirstConnect 保持具体类型不动(ad_authenticator.go:215 需 .Conn()) |
| INFRA-04 | agent 子进程 stub 统一 os/exec TestHelperProcess re-exec 模式(替换 exec.Command("echo")) | §Pattern 4:Go stdlib 自身用法已在 GOROOT src/os/os_test.go:2475-2516 核实(GO_WANT_HELPER_PROCESS + `-test.run` 过滤 + exe 重执行);echo 5 处调用点定位(subprocess_pgroup_test.go:13/:29/:40/:58/:69);Windows 分歧根源(echo 是 cmd.exe 内建,Git Bash 的 echo.exe 掩盖问题) |
| INFRA-05 | 测试隔离治理:ForTesting 后缀 + AST 守护测试(仿 status_constants_test.go) | §Pattern 5:全仓生产文件中 ForTesting 符号仅 internal/device/e2e_helpers.go 1 个文件;守护规则(禁 CallExpr/SelectorExpr 引用 ForTesting 后缀符号 + 定义文件白名单);**必须跳过点前缀目录**(.claude/worktrees 有 6 份仓库拷贝,status_constants 模式的 filepath.Glob 不能直接套用,须 WalkDir + 目录过滤) |
</phase_requirements>

## Project Constraints (from CLAUDE.md)

- **编译验证强制:** 任何 Go 改动后立即 `go build ./...`,逐文件修复,禁止批量修复
- **测试运行方式:** `go test ./...` 或按包 `go test ./internal/services/operations/`;单测先 cd 到包目录
- **临时文件纪律:** 根目录 `temp_*.go`/`test_*.go` 会造成 main 重声明——本 phase 新增文件全部落在正规包目录
- **状态常量真相源:** `internal/models/` 命名常量 + `status_constants_test.go` AST 锁值——AST 守护测试(76-05)与该文件同范式,新增守护不改既有锁值
- **模块路径:** import 用完整 `github.com/xingran-next/xingran-go-backend/...`
- **Handler-Service 模式:** 76-03 的 FailoverClient 工厂字段延续 struct 注入风格,不引入新全局单例模式
- **Git 提交前确认:** commit 前跑 build + test 并请求用户确认(GSD execute-phase 流程已覆盖)
- **scope 约束:** 只做 INFRA-01..05,不顺手修未报告问题(CLAUDE.md Debugging & Scope Constrainment)

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Redis 替身(miniredis) | 测试进程内(pkg/cache _test.go) | — | 纯 Go TCP server,RunT(t) 生命周期绑定单测;生产 RedisCache 零改动 |
| HTTP 出站 mock(httpmock) | 测试进程内(替换 DefaultTransport) | — | geocoding 的 `&http.Client{}` Transport==nil → 走 DefaultTransport → Activate 即拦截,零生产改动 |
| SSH Driver 构造 | internal/device 生产代码(工厂 var) | 测试(覆盖 var) | 工厂默认实现即生产路径;var 覆盖仅存在于 _test.go |
| LDAP 客户端构造 | internal/services/addomain(FailoverClient.clientFactory 字段) | mock(_test.go) | 生产默认 NewLDAPClient;测试注入 LDAPClientIface mock |
| 子进程 stub | 测试二进制自重执行(os.Args[0]) | — | stdlib 官方模式,跨平台(Windows 上 os.Args[0] 即 test .exe) |
| ForTesting 隔离契约 | 编译器(_test.go 物理隔离)+ 命名后缀 + AST 守护 | — | 三层:前两层已有,AST 层本 phase 补齐 |

## Standard Stack

### Core(本 phase 新增,test-only)

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| github.com/alicebob/miniredis/v2 | **v2.38.0**(proxy 实测最新) | 进程内 Redis server,真 TCP 接口 | go-redis v9 官方兼容组合;纯 Go 零 Docker;README 命令清单覆盖本仓全部命令面;MIT [VERIFIED: Go module proxy + 官方 README + 本地源码] |
| github.com/jarcoal/httpmock | **v1.4.2**(v1.4.x 最新) | 出站 HTTP mock(拦截 DefaultTransport) | 官方声明支持 Go 1.16–1.26(1.24 在列);`Activate(t)` 自动清理;MIT [VERIFIED: Go module proxy + 官方 README + 本地源码] |

### 既有依赖(本 phase 复用,零变更)

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| github.com/redis/go-redis/v9 | v9.7.0(生产既有) | miniredis 的客户端侧 | pkg/cache 冒烟经 NewRedisCache 真链路 |
| github.com/scrapli/scrapligo | v1.4.0(生产既有) | FileTransport fixture 回放 | 76-02 工厂注入的演示测试 |
| golang.org/x/crypto | v0.46.0(indirect 既有) | fake SSH server(Phase 78 用) | **本 phase 不 import**,go.mod 零变动 |
| glebarez/sqlite | v1.11.0(生产既有) | AccountPool DB 测试 | 76-03 FailoverClient 测试的 pool 侧 |
| (stdlib) go/parser + go/ast | Go 1.24.5 | AST 守护测试 | 76-05,零新依赖 |

### Alternatives Considered(明令禁止项,来自 REQUIREMENTS INFRA-01)

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| miniredis/v2 | go-redis/redismock | 硬淘汰:锁死 go-redis v8,本库 v9.7.0 不可用 [CITED: v1.27-stack.md web 验证] |
| miniredis/v2 | testcontainers-redis | 硬淘汰:强制 Docker,Windows dev 断裂(本机实测 Docker absent),双环境不同构 |
| httpmock | h2non/gock | 禁止:社区休眠,多年无 release [CITED: v1.27-stack.md] |
| httpmock | httptest.Server | 并用而非替代:凡 httpClient/baseURL 可注入场景一律优先 httptest(零依赖);httpmock 专治 geocoding 类 const-URL 无注入点场景 |

**Installation(注意注释与 tidy 语义):**
```bash
go get github.com/alicebob/miniredis/v2@v2.38.0
go get github.com/jarcoal/httpmock@v1.4.2
# 然后手工在 go.mod 两个 require 行尾追加注释:
#   github.com/alicebob/miniredis/v2 v2.38.0 // test-only (v1.27 D-02)
#   github.com/jarcoal/httpmock v1.4.2 // test-only (v1.27 D-02)
```

**传递依赖预告(IMPORTANT,SC#1 措辞联动):** `go get` 后 go.mod 会额外增加 indirect 行——miniredis→`github.com/yuin/gopher-lua v1.1.1`(Lua/EVAL 支撑),httpmock→`github.com/maxatome/go-testdeep v1.14.0`(其测试框架依赖)。"go.mod 仅新增 2 个 test-only 依赖"指 **direct 依赖**;indirect 行属模块解析必然产物,SUMMARY 需按 phase Notes 预先说明,避免被误判违反 D-02。[VERIFIED: 两库 go.mod 本地核实]

**tidy 陷阱:** `go mod tidy` 会**丢弃没有任何 import 的依赖**。76-01 若只 `go get` httpmock 而不写任何使用它的测试,后续 tidy 即把它删掉——httpmock 必须在本 phase 就有一个真实 import 它的测试(建议:geocoding 冒烟 PoC,兼作使用纪律的活样板)。miniredis 由 pkg/cache 冒烟自然保住。

**Version verification(2026-08-23 实测):**
```
go list -m -versions github.com/alicebob/miniredis/v2  → 最新 v2.38.0(REQUIREMENTS "v2.38+" 满足)
go list -m -versions github.com/jarcoal/httpmock       → 最新 v1.4.2(v1.4.x 满足)
两模块已 go mod download 到本地缓存,源码级核实完成
```

## Package Legitimacy Audit

> slopcheck 安装失败(本机 pip 环境拒绝,protocol 要求的 best-effort 已尝试)。按协议降级处理。

| Package | Registry | Age | Downloads | Source Repo | slopcheck | Disposition |
|---------|----------|-----|-----------|-------------|-----------|-------------|
| github.com/alicebob/miniredis/v2 | Go module proxy | ~10 年(项目始于 2015) | Go 生态 Redis 替身事实标准 | github.com/alicebob/miniredis | 不可用 | **Approved**——但按降级规则标 `[ASSUMED-slopcheck]`;缓解:包名由 REQUIREMENTS INFRA-01 用户锁定 + registry + 官方源码三重核实 |
| github.com/jarcoal/httpmock | Go module proxy | ~11 年(2014 起,LICENSE 版权年份) | 同上 | github.com/jarcoal/httpmock | 不可用 | **Approved**——同上降级标注 |

**Packages removed due to slopcheck [SLOP] verdict:** none
**Packages flagged as suspicious [SUS]:** none

*slopcheck 不可用的降级说明:两个包并非 Claude 自由检索发现,而是 REQUIREMENTS.md INFRA-01 的用户锁定决策(等价 CONTEXT.md Decisions 效力),且已通过 (a) Go module proxy 版本存在性、(b) 官方 GitHub README 全文、(c) 本地下载 pinned 版本逐文件源码核实(LICENSE=MIT、API 面、无 postinstall 类风险——Go 无该机制)三重验证。幻觉风险已实质消除;planner 无需 checkpoint:human-verify,但安装命令必须精确使用上表版本号。*

## Architecture Patterns

### System Architecture Diagram

```
                    Phase 76 交付物全景(5 个注入缝,全部进程内)

┌─ 76-01 ──────────────────────────────────────────────────────────────────┐
│ go.mod +2 direct test-only deps                                          │
│   miniredis/v2 v2.38.0 ──→ gopher-lua(indirect)                          │
│   httpmock v1.4.2 ──────→ go-testdeep(indirect)                          │
│                                                                          │
│ pkg/cache 冒烟测试:                                                      │
│   mr := miniredis.RunT(t)                                                │
│     → NewRedisCache({Host,Port}=mr.Addr())   ← 构造器内 PING = R-3 实证   │
│     → INCR / EXPIRE+TTL+FastForward(R-1) / SCAN / HSET 家族              │
│     → GetStats(R-2:只断言 err==nil + key_count) / IncrementWithExpire(EVAL)│
│                                                                          │
│ geocoding httpmock PoC(兼 tidy 保活 + 纪律样板):                         │
│   httpmock.Activate(t) → RegisterResponder(const URL) → Geocode() 断言    │
└──────────────────────────────────────────────────────────────────────────┘
┌─ 76-02 ──────────────────────────────────────────────────────────────────┐
│ internal/device/scrapli_wrapper.go                                       │
│   NewScrapliWrapper ─┐                                                   │
│   NewScrapliWrapperWithPort ─┬→ var newNetworkDriver(工厂,默认=生产实现)  │
│                              │     ↕ 测试覆盖(t.Cleanup 恢复)            │
│   DeviceConnectionPool.createConnection(零改动,经构造器自动生效)          │
│                    → FileTransport/自定义 transport 进 Open/SendCommand 链│
└──────────────────────────────────────────────────────────────────────────┘
┌─ 76-03 ──────────────────────────────────────────────────────────────────┐
│ LDAPClientIface(18 方法)+3 = 21 方法(UpdateGroupAttribute/CreateOU/DNExists)│
│ FailoverClient: +clientFactory 字段(默认 NewLDAPClient)                 │
│   ExecuteWithFailover(ctx, func(client LDAPClientIface) error)  ← 签名接口化│
│   PickFirstConnect(签名不动,ad_authenticator 需具体 .Conn())             │
│ mockLDAPClient:补 3 方法 + walk/分页语义驱动 + FailoverClient 测试       │
└──────────────────────────────────────────────────────────────────────────┘
┌─ 76-04 ──────────────────────────────────────────────────────────────────┐
│ internal/agent/server/subprocess_stub_test.go                            │
│   TestHelperProcess(t) ← GO_WANT_HELPER_PROCESS 守卫 + os.Args 形态分支   │
│   exec.Command(os.Args[0], "-test.run=TestHelperProcess", "--", ...)     │
│   替换 subprocess_pgroup_test.go 5 处 exec.Command("echo")               │
└──────────────────────────────────────────────────────────────────────────┘
┌─ 76-05 ──────────────────────────────────────────────────────────────────┐
│ AST 守护测试(建议 internal/device/ 共置)                                │
│   WalkDir("../..") 跳过 .前缀/_test.go/testdata/tests/前端目录           │
│   生产 .go 文件中 CallExpr/SelectorExpr 引用 *ForTesting → FAIL          │
│   白名单:internal/device/e2e_helpers.go(定义文件自身)                   │
└──────────────────────────────────────────────────────────────────────────┘
```

### Recommended Project Structure(新增文件)

```
pkg/cache/
├── redis_miniredis_76_01_test.go        # miniredis 冒烟(三坑防护用例)
internal/services/operations/
├── geocoding_httpmock_76_01_test.go     # httpmock PoC + 纪律样板
internal/device/
├── scrapli_wrapper.go                   # MODIFY:抽取工厂 var(唯一生产改动)
├── driver_factory_76_02_test.go         # 工厂注入演示测试
├── for_testing_guard_test.go            # 76-05 AST 守护(全仓扫描)
internal/services/addomain/
├── ldap_iface.go                        # MODIFY:+3 方法
├── ldap_client.go                       # (无需改——3 方法已在具体类型上存在)
├── failover_client.go                   # MODIFY:clientFactory 字段 + operation 签名
├── ldap_client_mock_test.go             # MODIFY:mock 补 3 方法 + 调用记录
├── failover_client_76_03_test.go        # 顺序遍历/maxHops 接口驱动测试
├── {group,group_management_service,group_sync_service,user,user_ad_sync_service,
│    dept_sync_service,sync}.go          # MODIFY:闭包参数类型 *LDAPClient→LDAPClientIface
internal/scheduler/dept_sync_tasks.go    # MODIFY:同上(唯一包外调用点)
internal/agent/server/
├── subprocess_stub_test.go              # NEW:TestHelperProcess helper
├── subprocess_pgroup_test.go            # MODIFY:5 处 echo → re-exec
```

### Pattern 1: miniredis 冒烟(含三坑防护)——76-01

**What:** 经 `NewRedisCache` 真链路连 miniredis,实证 pkg/cache 命令面。
**When to use:** 76-01 的 pkg/cache 冒烟;Phase 78 BLOCK-03 的 core Init 链复用同模式。

```go
// pkg/cache/redis_miniredis_76_01_test.go
func newMiniredisCache(t *testing.T) *RedisCache {
    t.Helper()
    mr := miniredis.RunT(t) // 官方 API:失败即 t.Fatal,测试结束自动关闭
    host, port, _ := net.SplitHostPort(mr.Addr())
    p, _ := strconv.Atoi(port)
    cache, err := NewRedisCache(&CacheConfig{Host: host, Port: p}, "xingran")
    if err != nil { t.Fatalf("NewRedisCache: %v", err) } // 构造器内 PING 已走完
                                                            // HELLO + CLIENT SETINFO 握手 → R-3 就此实证
    return cache
}
```

三坑防护用例(每坑一个具名测试):
- **R-1(TTL):** `cache.Set` + `cache.Expire(10s)` → `TTL>0` 断言 → `mr.FastForward(11*time.Second)` → `cache.Get` 返回 `ErrNotFound`。README 原文:"TTLs don't decrease automatically... m.FastForward(d) can be used to decrement all TTLs. All TTLs which become <= 0 will be removed." [VERIFIED: 官方 README]
- **R-2(INFO 降级):** `GetStats(ctx)` 只断言 `err==nil` + `stats["key_count"]` 存在(DBSize 正常)+ `stats["keyspace_info"]` 非空。README 原文:"INFO -- partly, returns only \"clients\" section with one field \"connected_clients\""。GetStats 源码(:427-486)对所有 Info 段错误静默忽略(`err == nil` 守卫),miniredis 下 redis_version/used_memory/keyspace_hits 全部缺席 → hit_rate=0.0,**不断言具体数值**。[VERIFIED: 官方 README + 本地源码 redis.go:427-486]
- **R-3(SETINFO 兼容):** 不写专门用例——`newMiniredisCache` 里 `NewRedisCache` 的 PING 已经走完 go-redis v9.7.0 完整握手(HELLO→RESP3→CLIENT SETINFO×2)。源码级结论:go-redis initConn 对 SETINFO 用 `_, _ = p.Exec(ctx)` **显式丢弃错误**(redis.go:349-357);miniredis v2.38.0 cmd_client.go 对 SETINFO 返回 `ERR unknown subcommand` 但 `setDirty` 在非事务上是 no-op(miniredis.go:596 注释原文 "Is an no-op then")。组合**开箱即用**;若未来升级 go-redis,此冒烟测试是回归哨兵。**勿用 `DisableIdentity` 选项:v9.7.0 里字段名拼写是 `DisableIndentity`(上游 typo,后续版本才修正)。**[VERIFIED: 双方 pinned 源码]

命令面清单(全部经 RedisCache 方法可达,grep 实测):`PING(构造器)/SET/GET/DEL/EXISTS/MGET/MSET/INCR(:166)/INCRBY(:171)/EXPIRE(:208)/TTL(:213)/SCAN(scanKeys :219-223,COUNT=500)/FLUSHDB(:259)/HGET(:389)/HSET(:394)/HGETALL(:399)/HDEL(:404)/HKEYS(:409)/INFO×4段(GetStats)/DBSIZE(:455)/EVAL(IncrementWithExpire Lua :186-203)`。全部在 miniredis 实现清单内(SCAN 游标一次归零兼容 scanKeys 的 next==0 循环——不要写"分批大小"断言,R-4)。

### Pattern 2: ScrapliWrapper Driver 工厂注入——76-02

**What:** 把两个构造器尾部重复的 platform→driver 构造段提为包级工厂 var。
**When to use:** 76-02 落地;Phase 78 BLOCK-04 用它注入 FileTransport/fake SSH。

```go
// internal/device/scrapli_wrapper.go(MODIFY——唯一生产改动)
// newNetworkDriver 是 platform→network.Driver 的默认工厂。
// 生产路径行为与内联调用完全一致;测试通过临时替换该 var 注入
// FileTransport/自定义 transport(见 driver_factory_76_02_test.go),
// 替换必须 t.Cleanup 恢复且所在测试禁止 t.Parallel()。
var newNetworkDriver = func(platformName, host string, opts ...util.Option) (*network.Driver, error) {
    p, err := platform.NewPlatform(platformName, host, opts...)
    if err != nil {
        return nil, err
    }
    return p.GetNetworkDriver()
}
```

两个构造器(:111/:174)把 `platform.NewPlatform(...) + p.GetNetworkDriver()` 六行替换为 `d, err := newNetworkDriver(platformName, device.IPAddress, opts...)`,错误文案保持原样(`创建平台实例失败`/`获取网络驱动失败` 可合并为一条或保留二段——建议工厂内不包装、调用点保留原包装,行为与错误字符串完全不变)。

**测试注入示例(照搬 portwrite 已验证的 FileTransport 选项组合):**
```go
// driver_factory_76_02_test.go
orig := newNetworkDriver
t.Cleanup(func() { newNetworkDriver = orig })
newNetworkDriver = func(_, _ string, _ ...util.Option) (*network.Driver, error) {
    p, err := platform.NewPlatform("huawei_vrp", "dummy-host",
        options.WithTransportType(transport.FileTransport),
        options.WithFileTransportFile(fixturePath),
        options.WithTransportReadSize(1),
        options.WithReadDelay(0),
    )
    if err != nil { return nil, err }
    return p.GetNetworkDriver()
}
w, err := NewScrapliWrapper(&models.NetworkDevice{...}, "u", "p", models.ProtocolTypeSSH)
// w.Open() → driver.Open() 读 fixture 首个 prompt → StateReady → SendCommand 链可驱动
```
scrapligo v1.4.0 API 签名已核实:`options.WithTransportType(transportType string)`(driver/options/generic.go:14)、`options.WithFileTransportFile(s string)`(driver/options/transportfile.go:10)、`platform.NewPlatform(f interface{}, host string, opts ...util.Option) (*Platform, error)`(platform/definition.go:190)、`(p *Platform) GetNetworkDriver() (*network.Driver, error)`(:296)。[VERIFIED: 本地 module 源码]

**为什么 var 而非构造器参数:** connection_pool.createConnection(:434-436)与散在的构造器调用零改动即获得注入能力;scrapli_wrapper.go 已 import platform/util/network,无新 import。**并发纪律:** 覆盖 var 的测试禁 t.Parallel(当前 device 包 3 个测试文件均无 Parallel,风险为零),必须 t.Cleanup 恢复。

**已知边界(Phase 78 才处理,本 phase 只留入口):** FileTransport 下 `OpenContext` 的 GetPrompt 轮询(:382-392)要求 fixture 首行是确定性 prompt;driver 不 Close(portwrite 先例:close 时读字节会 hang)。fake SSH server 的 PasswordCallback 返回 `(nil,false)` 而非 error(S-3)。

### Pattern 3: LDAPClientIface 扩展 + FailoverClient 工厂 seam——76-03

**What:** 接口 +3 方法;FailoverClient 增 clientFactory 字段;operation 闭包签名接口化。
**When to use:** 76-03;Phase 78 BLOCK-05 的 addomain ≥70% 全部建立在此 seam 上。

**改动 1 — ldap_iface.go +3 方法**(全部已在 `*LDAPClient` 上存在,零新实现):
```go
    // 组属性管理(group.go:201 闭包使用)
    UpdateGroupAttribute(groupDN string, attrs map[string]string) error
    // OU 管理(dept_sync_service.go:118 闭包使用)
    CreateOU(ouDN, ouName string) error
    // DN 存在性(user_ad_sync_service.go:187 闭包使用)
    DNExists(dn string) (bool, error)
```
`var _ LDAPClientIface = (*LDAPClient)(nil)` 编译期断言自动验证零遗漏。(OUExists 外部零使用,不加——最小接口原则。)

**改动 2 — failover_client.go:**
```go
type FailoverClient struct {
    pool   AccountPool
    config *models.ADConfig
    // clientFactory 构造每账号客户端;nil 时用 NewLDAPClient。
    // 测试注入 mock 工厂以驱动顺序遍历/maxHops(零真实网络)。
    clientFactory func(*models.ADConfig, *models.ADServiceAccount) LDAPClientIface
}

func (f *FailoverClient) newClient(acct *models.ADServiceAccount) LDAPClientIface {
    if f.clientFactory != nil { return f.clientFactory(f.config, acct) }
    return NewLDAPClient(f.config, acct) // 生产路径行为不变
}

// ExecuteWithFailover 的 operation 签名:func(client *LDAPClient) error → func(client LDAPClientIface) error
// 循环体内 client.Connect() / operation(client) / client.Close() 全部在接口方法集内
```
`PickFirstConnect` **签名保持不变**(返回 `*LDAPClient`,ad_authenticator.go:215-238 需要具体 client 的 `.Conn()` 传给 ldap.Search;config.go TestConnection 只 Close)。

**改动 3 — 28 处闭包参数类型机械替换**(`*LDAPClient`/`*addomain.LDAPClient` → `LDAPClientIface`/`addomain.LDAPClientIface`),文件清单(grep 实测):group.go(3)、group_management_service.go(4)、group_sync_service.go(2)、user.go(4)、user_ad_sync_service.go(4)、dept_sync_service.go(1)、sync.go(1)、internal/scheduler/dept_sync_tasks.go(1)= 20 处闭包签名 + NewFailoverClient 调用点若干(不改)。闭包体内调用的方法经逐一核对全部落在扩展后的 21 方法接口内。

**改动 4 — mockLDAPClient 补全:**
- 3 个新方法的 preset 字段(与现有风格一致);
- "walk/分页语义"驱动能力:按序返回(`searchUsersResult` 支持多批次回调风格或 `searchUsersFn func() ([]*ldap.Entry, error)` 函数字段,供大结果集/多次调用场景);
- FailoverClient 测试形态:clientFactory 按账号序号返回不同 mock(账号 0 → Connect 失败 mock,账号 1 → 成功 mock)→ 断言 MarkFailure/MarkSuccess 次数与 operation 执行次数 = 顺序遍历语义;账号数 > DefaultMaxHops 的池 → 断言尝试次数封顶 = maxHops 语义。

### Pattern 4: TestHelperProcess re-exec——76-04

**What:** 以测试二进制自身重执行充当子进程 stub,替换平台相关外部命令。
**When to use:** subprocess_pgroup_test.go 重写;Phase 78 BLOCK-02(agent/server)与 BLOCK-03(Core reaper)沿用此 helper 模式。

Go stdlib 自身在 GOROOT src/os/os_test.go:2475-2516(TestStatStdin/TestGetppid)使用该模式:`if Getenv("GO_WANT_HELPER_PROCESS") == "1" { ...; Exit(0) }` 守卫 + `cmd := Command(exe, "-test.run=^TestXxx$")` + `cmd.Env = append(Environ(), "GO_WANT_HELPER_PROCESS=1")`。[VERIFIED: go1.24.5 GOROOT 源码]

推荐集中式 helper(多形态分支):
```go
// internal/agent/server/subprocess_stub_test.go
func TestHelperProcess(t *testing.T) {
    if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" { return } // 正常 go test 直接跑时静默通过
    args := os.Args[len(os.Args)-1:]                          // "--" 之后的形态参数
    switch {
    case len(args) > 0 && args[0] == "sleep-until-stdin-close": // 常驻形态
        io.Copy(io.Discard, os.Stdin)
    case len(args) > 0 && args[0] == "ignore-sigterm":          // 忽略 SIGTERM 形态
        sig := make(chan os.Signal, 1); signal.Notify(sig, syscall.SIGTERM)
        <-sig; fmt.Println("still-alive")
    case len(args) > 0 && args[0] == "stdout-flood":            // 大量 stdout 形态
        for i := 0; i < 1000; i++ { fmt.Println("line") }
    default:                                                     // 秒退/echo 形态
        fmt.Println("hello")
    }
    os.Exit(0)
}

// 测试侧构造(跨平台:Windows 上 os.Args[0] 即测试 .exe):
cmd := newCommand(ctx, os.Args[0], "-test.run=^TestHelperProcess$", "--", "sleep-until-stdin-close")
cmd.Env = append(os.Environ(), "GO_WANT_HELPER_PROCESS=1")
```

**Windows 细节:** `-test.run` 精确锚定(`^...$`)防前缀误匹配其它测试;`--` 后参数由 os.Args 直取(不经 flag 解析);`newCommand` 已挂 setProcessGroup(sysproc_linux.go Setpgid / sysproc_windows.go CREATE_NEW_PROCESS_GROUP 拆分保持不动);`syscall.SIGTERM` 在 Windows 上无实义——涉及信号的形态分支用 `runtime.GOOS` 守卫或选 Windows 等价行为,断言按平台分支(延续现有 sysproc_*.go 拆分哲学)。

**echo 分歧根源(实证):** subprocess_pgroup_test.go 5 处(:13/:29/:40/:58/:69)`exec.Command("echo", ...)`——`echo` 是 cmd.exe 内建(无独立 exe),Windows 上 `exec.LookPath("echo")` 依赖 PATH 里恰有 Git Bash 的 usr/bin/echo.exe 才碰巧能跑 → 本地/CI 行为分歧(74-08 记载的 agent-server "env-branch divergence" 来源之一)。re-exec 模式根除:stub 进程 = 测试二进制自身,任何环境语义一致。

### Pattern 5: AST 守护测试——76-05

**What:** 仿 status_constants_test.go 的 go/parser 范式,禁止生产 .go 文件引用 `*ForTesting` 符号。
**When to use:** 76-05 一次落地,此后 ForTesting 家族任何新增自动受保护。

```go
// internal/device/for_testing_guard_test.go
// 规则:非 _test.go 的 .go 文件中,任何以 ForTesting 结尾的标识符
// *调用/引用*(CallExpr.Fun 或 SelectorExpr.Sel)都使测试失败。
// 白名单:internal/device/e2e_helpers.go(定义文件自身内部引用)。
func TestNoProductionForTestingReferences(t *testing.T) {
    root := "../.." // go test 的 cwd = 包目录 → 仓库根
    var violations []string
    filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
        if err != nil { return err }
        name := d.Name()
        if d.IsDir() {
            if strings.HasPrefix(name, ".") || name == "vendor" ||
               name == "node_modules" || name == "xingran-react-frontend" ||
               name == "testdata" || name == "tests" {
                return fs.SkipDir   // 关键:.claude/worktrees/ 下有 6 份完整仓库拷贝!
            }
            return nil
        }
        if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") { return nil }
        if filepath.ToSlash(path) == "internal/device/e2e_helpers.go" { return nil } // 白名单
        fset := token.NewFileSet()
        f, perr := parser.ParseFile(fset, path, nil, 0)
        if perr != nil { return fmt.Errorf("parse %s: %w", path, perr) }
        ast.Inspect(f, func(n ast.Node) bool {
            switch x := n.(type) {
            case *ast.CallExpr:      // direct call: newScrapliWrapperForTesting(...)
                if id, ok := x.Fun.(*ast.Ident); ok && strings.HasSuffix(id.Name, "ForTesting") {
                    violations = append(violations, fmt.Sprintf("%s: call to %s", path, id.Name))
                }
            case *ast.SelectorExpr:  // qualified: device.NewPooledConnectionForTesting(...)
                if strings.HasSuffix(x.Sel.Name, "ForTesting") {
                    violations = append(violations, fmt.Sprintf("%s: reference to %s", path, x.Sel.Name))
                }
            }
            return true
        })
        return nil
    })
    // 断言 violations 为空,逐条打印 file:line
}
```

**与 status_constants_test.go 的差异(为什么不能照抄 Glob):** 该先例只扫自己包目录的 `*.go`;ForTesting 契约是全仓的,必须 WalkDir——由此引入两点新要求:(1) **跳过点前缀目录**(.claude/worktrees 实测 6 份仓库拷贝,不跳过会扫到陈旧副本甚至误报);(2) 白名单机制(e2e_helpers.go 是生产文件但合法定义+内部引用 ForTesting 符号)。FuncDecl 的函数名是声明不是引用,不进 CallExpr/SelectorExpr 分支,天然不误报。

**三层隔离契约成文(SC#5):** (1) 编译器——_test.go 内符号生产物理不可 import;(2) 命名——跨包需要触达私有字段时用 ForTesting 后缀生产符号(v1.27-architecture 约定 2);(3) AST——生产代码引用 ForTesting 即测试失败。三层互补:层 1 免费,层 2 管跨包,层 3 兜底防呆。

### Anti-Patterns to Avoid

- **在测试里用 `time.Sleep` 等 TTL 过期:** miniredis TTL 不自动流逝,必假绿/假红;一律 `mr.FastForward(d)`(R-1)
- **断言 GetStats 的具体字段值:** miniredis INFO 只有 connected_clients,redis_version/used_memory/keyspace_hits 全缺席(R-2)
- **给 go-redis v9.7.0 设 `DisableIdentity`:** 字段名实为 `DisableIndentity`(上游 typo),且根本不需要——SETINFO 错误已被客户端丢弃(R-3 纠偏)
- **覆盖工厂 var 的测试加 t.Parallel():** 包级 var 全局可变,并行互踩;覆盖必须 t.Cleanup 恢复
- **`go get` httpmock 后不写使用测试就 `go mod tidy`:** tidy 直接丢弃未 import 依赖,SC#1 假性满足后又被静默拆走
- **AST 守护用 filepath.Glob 扫描仓库:** Glob 不递归且无法跳 worktrees;必须 WalkDir + 目录过滤
- **FileTransport 测试调用 driver.Close():** fixture 耗尽后 close 路径读字节会 block(portwrite 先例注释 :42-48)
- **把 `PickFirstConnect` 也接口化:** ad_authenticator 需要 `client.Conn()` 具体方法;动了会扩大 blast radius 到 core/security

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Redis 测试替身 | 自写 RESP server / 内存 map 假客户端 | miniredis/v2 | 命令语义/TTL/事务/Lua 边角极多;miniredis 有 integration 目录对照真 Redis 8.4 |
| 出站 HTTP mock | 自写 RoundTripper 拦截器(每处手搓) | httpmock(Activate)或 httptest.Server | URL 匹配算法(sorted query/path 六形态)、responder 序列、调用计数都是现成的;geocoding 场景 const-URL 无注入点,httpmock 零改动命中 |
| 子进程 stub | exec.Command("echo"/"sleep"/平台命令) | TestHelperProcess re-exec | 平台二进制分歧正是要根除的问题;stdlib 官方模式跨平台同构 |
| ForTesting 误用检测 | bash grep 脚本 | go/ast 守护测试 | grep 分不清声明/引用/注释;AST 精确到 CallExpr/SelectorExpr 且 file:line 报告;仓库哲学(D-01 自实现)下 go/parser 就是零依赖自实现 |
| LDAP 服务端假件 | 自写 wire 级 LDAP server | LDAPClientIface mock 扩展 | vjeantet/ldapserver 停更风险 + 自写 ber 编解码成本;接口 stub 零新依赖(REQUIREMENTS INFRA-03 主推线) |

**Key insight:** 五个 seam 全部选择"最小注入点 + 生产默认行为"而不是"接口化一切"——本 phase 的价值在于给 Phase 77/78 解锁,不在于追求抽象完备。

## Runtime State Inventory

> INFRA-02/03 含小型重构,按协议逐类显式回答。

| Category | Items Found | Action Required |
|----------|-------------|------------------|
| Stored data | None — verified by:本 phase 不触任何表/迁移;改动是构造缝与测试文件,DB schema 零变更 | none |
| Live service config | None — verified by:不触 configs/*.yaml、Redis/PG/AD 运行时配置;miniredis 仅存在于测试进程内 | none |
| OS-registered state | None — verified by:无 Task Scheduler/pm2/systemd 注册;git worktrees(.claude/worktrees 6 份)是 agent 会话产物,本 phase 只需 AST 守护**跳过**它们,不清理不修改 | none |
| Secrets/env vars | None — verified by:不新增 env 依赖;GO_WANT_HELPER_PROCESS 仅测试进程内约定,不进 .env/CI secrets | none |
| Build artifacts | `go.sum` 将新增 miniredis/httpmock/gopher-lua/go-testdeep 的哈希行(预期内);`.claude/worktrees/*/` 内 stale 副本不受影响(git 忽略) | go.sum 变更随 go get 自然产生,SUMMARY 记录 |

## Common Pitfalls

### Pitfall 1: go.mod "2 个依赖" 措辞与 indirect 行
**What goes wrong:** `go get` 后 go.mod 出现 4+ 行变动(2 direct + 2~3 indirect),评审误判违反"生产依赖零变更"。
**Why it happens:** miniredis 依赖 gopher-lua(EVAL 支撑)、httpmock 依赖 go-testdeep;Go 模块解析必然把它们写为 indirect。
**How to avoid:** SUMMARY 预写"direct +2(带 // test-only 注释)/indirect +2(gopher-lua、go-testdeep),生产 require 块零变更";diff 审查对照该清单。
**Warning signs:** PR diff 中出现未预告的 require 行。

### Pitfall 2: tidy 丢弃 httpmock
**What goes wrong:** 76-01 只装不用,后续任何 `go mod tidy`(CI/go get 触发)静默删掉 httpmock。
**How to avoid:** 同 plan 内落地 geocoding httpmock PoC 测试(真实 import)。
**Warning signs:** go.mod 里 httpmock 行消失;CI 编译失败找不到包。

### Pitfall 3: R-3 的过时认知(与里程碑 pitfalls 研究冲突)
**What goes wrong:** 按里程碑 pitfalls 写 `DisableIdentity: true` 防护——v9.7.0 拼写是 `DisableIndentity`,编译失败;或误以为必须升级 miniredis 才兼容。
**Why it happens:** v1.27-pitfalls.md 记载"最新 miniredis 已将 CLIENT SETINFO 实现为 no-op"——v2.38.0 源码实证为误(仍返回 unknown subcommand);真正的兼容来自 go-redis 侧丢弃错误。
**How to avoid:** 不做任何 SETINFO 特殊处理;靠 NewRedisCache 冒烟(握手即覆盖)作回归哨兵。
**Warning signs:** 测试代码出现 DisableIndentity/DisableIdentity 字样。

### Pitfall 4: AST 守护扫到 .claude/worktrees
**What goes wrong:** WalkDir 全仓时扫进 6 份 worktree 拷贝——陈旧副本里可能有已删除的 ForTesting 引用或解析失败的中间态文件,守护测试随机红。
**How to avoid:** 点前缀目录一律 SkipDir(顺带跳过 .git);对解析失败 fail 而非 skip(status_constants 先例语义)。
**Warning signs:** 守护测试报出的路径含 `.claude/worktrees`。

### Pitfall 5: 76-03 的 28 处闭包漏改/错改
**What goes wrong:** 接口化后某闭包仍声明 `*LDAPClient` → 编译错误(好情况);或顺手"改进"闭包体内逻辑 → 生产行为漂移(坏情况)。
**How to avoid:** 纯机械替换(仅参数类型),逐文件 `go build ./...`;scheduler 是唯一包外调用点,勿漏;改完跑 addomain + scheduler + core/security 三包测试(ad_authenticator 是 PickFirstConnect 消费方)。
**Warning signs:** diff 中出现类型替换以外的闭包体改动。

### Pitfall 6: 工厂 var 覆盖后未恢复
**What goes wrong:** 测试失败路径( Fatal 在 Cleanup 前)或遗忘恢复 → 同包后续测试拿到 FileTransport 工厂,级联失败难定位。
**How to avoid:** 覆盖即 `t.Cleanup(func(){ newNetworkDriver = orig })`(先注册覆盖再改 var 的顺序写法);禁 t.Parallel。
**Warning signs:** device 包测试顺序相关的偶发失败。

### Pitfall 7: Windows 路径/换行污染 fixture
**What goes wrong:** 76-02 演示测试的 fixture 路径 Join 出反斜杠、或 fixture 被 git 转成 CRLF → FileTransport 读取行为漂移。
**How to avoid:** 沿用 portwrite 的 `runtime.Caller` 定位 + testdata 目录;fixture 保持 LF(仓库 .gitattributes 已强制);断言用 TrimSpace 比对(W-4)。

### Pitfall 8: TestHelperProcess 被正常 go test 执行时误动作
**What goes wrong:** re-exec 二进制与正常测试共用同一进程入口,若守卫缺失,普通 `go test` 跑到 helper 分支会挂起(等 stdin)或输出污染。
**How to avoid:** 函数第一行 `if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" { return }`;每个形态分支必达 `os.Exit`(防 testing 框架继续解析后续参数)。
**Warning signs:** `go test ./internal/agent/server/...` 挂起无输出。

## Code Examples

(已内嵌于各 Pattern,来源标注:)
- miniredis RunT/FastForward/INFO/命令清单:官方 README(github.com/alicebob/miniredis,2026-08-23 读取)+ 本地 v2.38.0 源码
- go-redis initConn SETINFO 丢弃:本地 GOMODCACHE go-redis v9.7.0 redis.go:280-362
- httpmock Activate(t)/RegisterResponder:官方 README + 本地 v1.4.2 源码 transport.go:1124
- 工厂 var + FileTransport 注入:port_write_e2e_test.go:71-95(仓库已验证先例)+ scrapligo v1.4.0 源码签名
- TestHelperProcess:GOROOT src/os/os_test.go:2475-2516
- AST 守护:internal/models/status_constants_test.go:289-348(仓库先例,适配 WalkDir)

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| `httpmock.Activate()` + `defer DeactivateAndRestore()` | `httpmock.Activate(t)`(testing.TB 自动清理) | v1.4.x(transport.go:1124 `func Activate(t ...testing.TB)`) | 纪律简化:优先 Activate(t);Ginkgo 等无 t 场景才用无参形态+手动 Deactivate |
| go-redis v8 + redismock | go-redis v9 + miniredis | go-redis v9.5 引入 CLIENT SETINFO | redismock 淘汰;miniredis+go-redis v9 组合经源码实证开箱即用 |
| `exec.Command("echo")` 平台命令 stub | os.Args[0] re-exec(TestHelperProcess) | Go stdlib 既有(os_test.go 多年使用) | Windows/CI 同构;agent-server 覆盖率环境分歧根源消除 |
| build tag 隔离测试 | _test.go 编译隔离 + ForTesting 命名 + AST 守护 | v1.27-architecture Q3(2026-08-23 锁定) | 覆盖率目标依赖每次 go test 全跑,tag 方案拆台 |

**Deprecated/outdated:**
- gock:社区休眠,禁止引入 [CITED: v1.27-stack.md §4]
- vjeantet/ldapserver:停更,仅作 Phase 78 wire 线备选(本 phase 不引入)
- testcontainers:Windows 无 Docker 断裂本地测试(本机实测 Docker absent),禁止进默认测试面

## Assumptions Log

> 本 phase 所有关键主张均经本地源码/registry/官方 README 实证;以下为残余假设。

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | "2 个 direct test-only 依赖"的 SC#1 措辞允许 indirect 行(gopher-lua/go-testdeep)自然出现 | Pitfall 1 / Summary | 若 SC#1 严格解读为 go.mod 零其他新增行,需在 SUMMARY 显式请示;实质不可能避免(Go 模块机制) |
| A2 | "walk/分页语义"解读为:mock 支持多批次/函数式返回驱动 service 层遍历逻辑 + 大结果集;具体 LDAPClient 内部 searchWithPaging 降级逻辑不在本 phase 范围 | Pattern 3 | 若 planner 理解为要测具体客户端的分页递归(ldap_client.go:243-295),那需要 wire 级 server(Phase 78 备选线),scope 会膨胀——建议 plan 前与用户确认 |
| A3 | AST 守护测试放 internal/device 包(与 ForTesting 符号共置),扫描根用 `../..` | Pattern 5 | 放独立新包(如 internal/guards)也可,但多一个无生产代码的包;不影响正确性 |
| A4 | FailoverClient 工厂用 struct 字段(clientFactory)而非包级 var,与 device 的 var 方案不对称 | Pattern 3 | 两种都正确;字段方案因 FailoverClient 本身就是 per-instance 构造(NewFailoverClient),字段更符合仓库 DI 风格;planner 可统一改 var,但 28 调用点不受影响 |
| A5 | httpmock PoC 选 geocoding(NewGeocodingService 无 Redis 依赖路径) | Pattern 1 | 若 PoC 改选别的调用点需确认该点同样走 DefaultTransport(凡 `&http.Client{}` Transport==nil 均满足) |

## Open Questions (RESOLVED)

1. **INFRA-01 的 SC#1 措辞精度(gomap diff 口径)**
   - What we know:direct +2、indirect +2、go.sum 若干行是模块机制必然。
   - What's unclear:SC#1 "生产依赖零变更"是否把 indirect 行视为"变更"。
   - Recommendation:76-01 SUMMARY 首段即写明 diff 形态预览(见 Pitfall 1),verifier 按此口径验收;无需阻塞 plan。
   - RESOLVED: 采纳 Recommendation——已嵌入 76-01 Task 1 action 与 success_criteria（SUMMARY 首段预写 go.mod diff 形态：direct +2 带 test-only 注释 / indirect +2 gopher-lua+go-testdeep / 生产 require 零变更），verifier 按此口径验收。
2. **A2 的"walk/分页语义"边界**
   - What we know:接口 stub 线能驱动 service 层;具体客户端分页降级测试需 wire server。
   - What's unclear:用户是否预期本 phase 覆盖后者。
   - Recommendation:按 REQUIREMENTS INFRA-03 原文("扩展 stub 主推线,零新依赖")取窄解读——本 phase 只做接口/工厂/mock;wire 线留给 Phase 78 按缺口决定。planner 在 76-03 plan 里写明该边界即可。
   - RESOLVED: 按窄解读裁决——76-03 objective 已写边界声明（本 phase 只做接口/工厂/mock；ldap_client.go:243-295 searchWithPaging 分页降级与 wire 级 server 留 Phase 78 按缺口决定）。
3. **76-02 演示测试的 fixture 放哪**
   - What we know:portwrite fixtures 在 internal/services/portwrite/testdata/;device 包自身无 testdata。
   - What's unclear:device 包新建 testdata/ 还是复用 portwrite 的。
   - Recommendation:internal/device/testdata/ 新建(自包含;portwrite 的 fixture 是命令回放序列,device 的 Open 场景 prompt 形态不同)。
   - RESOLVED: 新建 internal/device/testdata/ 自包含——76-02 Task 2 已按此规划（Open 场景 prompt 形态与 portwrite 命令回放序列不同）。

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Go toolchain | 全部 | ✓ | 1.24.5 windows/amd64(与 go.mod toolchain go1.24.5 一致) | — |
| Go module proxy 网络 | go get 新依赖 | ✓(本 session `go list -m -versions`/`go mod download` 实测成功) | — | vendor 不适用 |
| Docker | **不需要**(零 Docker 是硬要求) | ✗(本机 absent——反而实证了要求合理性) | — | 不引入任何容器依赖 |
| golangci-lint | 本地 lint 镜像 | ✓ | 2.12.2(与 CI action pin 完全一致) | CI 侧兜底 |
| WSL | -race 本地验证(可选,P75 惯例) | 未探测 | — | Go 的 race detector 在 windows/amd64 原生支持,可直接 `go test -race` 单包验证 |
| miniredis v2.38.0 | 76-01 | ✓ 已下载入 GOMODCACHE | v2.38.0 | — |
| httpmock v1.4.2 | 76-01 | ✓ 已下载入 GOMODCACHE | v1.4.2 | — |

**Missing dependencies with no fallback:** none
**Missing dependencies with fallback:** none(WSL 可选,见上)

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go 原生 testing + stretchr/testify v1.11.1(既有;mock 手写风格,不用 gomock——仓库锁定) |
| Config file | 无(go test 直跑);lint 配置 .golangci.yml v2 schema |
| Quick run command | `go test -count=1 ./pkg/cache/... ./internal/device/... ./internal/services/addomain/... ./internal/agent/server/...` |
| Full suite command | `bash scripts/check-ci-local.sh backend`(lint + test + coverage gate 三合一,CI 同构) |

### Phase Requirements → Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| INFRA-01 | go.mod +2 test-only 依赖 + 双环境绿 | 结构断言(manual diff 审查 + CI) | `git diff go.mod` + `go test ./...` | ❌ Wave 0(随 76-01 产生) |
| INFRA-01 | miniredis 冒烟:INCR/EXPIRE/TTL/SCAN/HSET/INFO/EVAL 经真 NewRedisCache 链 | integration(进程内替身) | `go test -count=1 -run TestRedis ./pkg/cache/` | ❌ Wave 0(redis_miniredis_76_01_test.go) |
| INFRA-01 | R-1 FastForward / R-2 INFO 降级断言 / R-3 握手兼容 | unit(三坑各一具名测试) | 同上(-run 三坑用例名) | ❌ Wave 0 |
| INFRA-01 | httpmock PoC + 纪律(Activate(t)) | unit | `go test -count=1 -run TestGeocoding ./internal/services/operations/` | ❌ Wave 0(geocoding_httpmock_76_01_test.go) |
| INFRA-02 | 工厂注入后生产路径不变 | regression(现有 3 个 device 测试文件全绿) | `go test -count=1 ./internal/device/` | ✅ 基线绿(今日实测) |
| INFRA-02 | 工厂可注入 FileTransport 进 Open/SendCommand 链 | integration(fixture 回放) | `go test -count=1 -run TestDriverFactory ./internal/device/` | ❌ Wave 0(driver_factory_76_02_test.go) |
| INFRA-03 | 接口 +3 后 mock/具体类型双满足(编译期断言) | compile-time | `go build ./...` + `go vet ./internal/services/addomain/` | ❌ Wave 0(随 76-03) |
| INFRA-03 | FailoverClient 顺序遍历/maxHops 接口驱动 | unit(sqlite pool + mock 工厂) | `go test -count=1 -run TestFailover ./internal/services/addomain/` | ❌ Wave 0(failover_client_76_03_test.go) |
| INFRA-03 | 28 处闭包机械替换零行为漂移 | regression | `go test -count=1 ./internal/services/addomain/... ./internal/scheduler/... ./internal/core/security/...` | ✅ 基线绿(今日实测) |
| INFRA-04 | re-exec helper 跨平台跑通 + echo 清零 | unit(子进程) | `go test -count=1 ./internal/agent/server/` | ❌ Wave 0(subprocess_stub_test.go;改写 subprocess_pgroup_test.go) |
| INFRA-05 | AST 守护:生产文件零 ForTesting 引用 + 守护自证(注毒用例) | meta-test | `go test -count=1 -run TestNoProductionForTesting ./internal/device/` | ❌ Wave 0(for_testing_guard_test.go) |

### Sampling Rate
- **Per task commit:** quick run command(目标包 + `go build ./...`)+ `golangci-lint run --timeout=5m ./internal/... ./pkg/...`(新增文件)
- **Per wave merge:** `bash scripts/check-ci-local.sh backend`(lint + 全量 test + coverage gate 55.5 floor)
- **Phase gate:** 全量绿 + push 后 `gh run watch` 盯 ci.yml backend Coverage gate(memory push-watch-ci 惯例);本地可选 `go test -race ./pkg/cache/... ./internal/device/... ./internal/agent/server/...`(工厂 var 与 helper 进程的并发安全)

### Wave 0 Gaps
- [ ] `pkg/cache/redis_miniredis_76_01_test.go` — INFRA-01 冒烟 + 三坑
- [ ] `internal/services/operations/geocoding_httpmock_76_01_test.go` — INFRA-01 httpmock(tidy 保活)
- [ ] `internal/device/driver_factory_76_02_test.go` — INFRA-02 注入演示
- [ ] `internal/services/addomain/failover_client_76_03_test.go` — INFRA-03 接口驱动
- [ ] `internal/agent/server/subprocess_stub_test.go` — INFRA-04 helper
- [ ] `internal/device/for_testing_guard_test.go` — INFRA-05 AST 守护
- [ ] `internal/device/testdata/` — Open 场景 fixture(prompt 形态)

## Security Domain

> security_enforcement 未显式关闭(配置缺省 = 启用)。本 phase 为测试基建,攻击面极小,逐类核对如下。

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | no | 本 phase 不触认证链(ad_authenticator 仅受 76-03 签名波及,行为不变) |
| V3 Session Management | no | 无会话逻辑改动 |
| V4 Access Control | no | 无路由/权限变更 |
| V5 Input Validation | 间接 | fixture 文件路径经 runtime.Caller 派生(仓库先例),不接收外部输入;TestHelperProcess 的 os.Args 分支仅在测试二进制内生效 |
| V6 Cryptography | no | 不触 SM2/SM3/SM4;密码解密链(connection_pool passwordCipher)不动 |
| V14 Config | yes(轻) | go.mod 依赖引入 = 供应链配置:两库 MIT + 源码级核实 + 版本 pin,禁止 `@latest` 浮动 |

### Known Threat Patterns for Go test doubles

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| 测试替身泄漏进生产二件(ForTesting 被业务调用) | Tampering/Elevation | 三层契约(编译器+命名+AST 守护)——本 phase 交付物即此防线 |
| 供应链投毒(伪依赖/typo squatting) | Tampering | REQUIREMENTS 锁定包名 + 精确版本 pin + slopcheck 三重降级核实(见 Package Legitimacy Audit) |
| fixture 中泄漏真实凭据/IP | Information Disclosure | fixture 全部 dummy 值(dummy-host 先例);不复制生产设备密码 |
| mock 绕过生产安全检查(连接池簿记/凭据解密) | Elevation | e2e_helpers 头注释已声明调用 ForTesting 会跳过的安全步骤;AST 守护防生产误用 |

## Sources

### Primary (HIGH confidence)
- 本地 module 源码(全部 2026-08-23 逐文件核实):miniredis/v2@v2.38.0(README/LICENSE/cmd_client.go/miniredis.go:596/go.mod)、httpmock@v1.4.2(transport.go:51/:677/:1124/:1159、response.go:437/LICENSE/go.mod)、go-redis/v9@v9.7.0(redis.go:280-362 initConn)、scrapli/scrapligo@v1.4.0(driver/options/generic.go:14、driver/options/transportfile.go:10、platform/definition.go:190/:296)、GOROOT go1.24.5(src/os/os_test.go:2475-2516)
- 仓库代码实证:scrapli_wrapper.go(:111-235/:301-423)、connection_pool.go(:393-482)、e2e_helpers.go(全文)、ldap_iface.go(全文)、ldap_client_mock_test.go(全文)、failover_client.go(全文)、subprocess.go + subprocess_pgroup_test.go(全文)、sysproc_*.go、status_constants_test.go(:289-363)、port_write_e2e_test.go(:40-113)、geocoding_service.go(:20-95)、pkg/cache/redis.go(命令面全量)、go.mod、.github/workflows/ci.yml、.golangci.yml、scripts/check-ci-local.sh、.github/scripts/check-coverage.sh
- 官方 GitHub README(github.com/alicebob/miniredis、github.com/jarcoal/httpmock,2026-08-23 webReader 全文读取)
- Go module proxy(`go list -m -versions` 版本核实)

### Secondary (MEDIUM confidence)
- .planning/research/v1.27-stack.md / v1.27-pitfalls.md / v1.27-architecture.md(2026-08-23 里程碑研究;其中 R-3 表述已被本 phase 源码级证据**纠偏**)

### Tertiary (LOW confidence)
- 无(本 phase 无仅凭单一 web 来源的关键主张)

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — 两库 pinned 源码 + registry + 官方 README 三重实证;唯一降级项是 slopcheck 不可用(已按协议标注,但包为用户锁定决策)
- Architecture: HIGH — 全部 seam 落点行号级实证;28 调用点 grep 全量清点;基线测试当日全绿
- Pitfalls: HIGH — R-1/R-2/R-3 直接读 pinned 源码定级;R-3 与里程碑研究的冲突以源码为准并留纠偏记录

**Research date:** 2026-08-23
**Valid until:** 2026-09-22(依赖版本 30 天窗口;miniredis/httpmock 均低频发版,风险低)
