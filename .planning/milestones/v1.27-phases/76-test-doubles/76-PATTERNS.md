# Phase 76: 测试基建落地 (test doubles + 注入缝) - Pattern Map

**Mapped:** 2026-08-23
**Files analyzed:** 14 项新增/修改交付物(6 新测试文件 + 1 testdata 目录 + go.mod + 6 生产/接口/mock 修改点 + 8 文件 20 处闭包机械替换)
**Analogs found:** 12 / 14(2 项无仓内类比,用 RESEARCH.md Code Examples + GOROOT 先例)

> 本 phase 无 CONTEXT.md;文件清单唯一来源 = 76-RESEARCH.md §Recommended Project Structure + §Phase Requirements(INFRA-01..05)。
> RESEARCH 的行号引用本次映射全部本地复核通过;唯一偏差:`ldap_iface.go` 实际 16 个方法(RESEARCH 记 18),+3 后为 19——planner 验收时按 16→19 口径,不要按 18→21。

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|-------------------|------|-----------|----------------|---------------|
| `go.mod` | config | — | 无(仓内无 test-only 依赖先例) | none |
| `pkg/cache/redis_miniredis_76_01_test.go` | test | CRUD(Redis 命令面) | `pkg/cache/cache_74_08_test.go` | role-match(同包测试风格,数据流 memory→redis) |
| `internal/services/operations/geocoding_httpmock_76_01_test.go` | test | request-response(出站 HTTP mock) | `internal/services/operations/geocoding_photo_floor_test.go` | exact(同一被测服务,仅 transport 机制不同) |
| `internal/device/scrapli_wrapper.go` | utility(驱动封装) | request-response(SSH 命令) | 自身:149-162 与 :212-226 重复段 | exact(工厂抽取自有重复代码) |
| `internal/device/driver_factory_76_02_test.go` | test | file-I/O(fixture 回放) | `internal/services/portwrite/port_write_e2e_test.go` | exact(FileTransport 选项组合 + fixture 定位先例) |
| `internal/device/testdata/`(fixture) | test fixture | file-I/O | `internal/services/portwrite/testdata/` | role-match(prompt 形态不同,须新建) |
| `internal/services/addomain/ldap_iface.go` | interface | — | 自身(:18-42 既有接口) | exact |
| `internal/services/addomain/failover_client.go` | service | event-driven(账号池故障切换遍历) | 自身(:20-28 struct) | exact |
| `internal/services/addomain/ldap_client_mock_test.go` | test(mock) | — | 自身(:12-43 preset 字段风格) | exact |
| `internal/services/addomain/failover_client_76_03_test.go` | test | CRUD(sqlite pool) | `internal/services/addomain/account_pool_test.go` | role-match(setupTestPool/insertAccount 直接复用) |
| 8 个闭包文件 × 20 处(group.go×3, group_management_service.go×4, group_sync_service.go×2, user.go×4, user_ad_sync_service.go×4, dept_sync_service.go×1, sync.go×1, `internal/scheduler/dept_sync_tasks.go`×1) | service | event-driven | `internal/services/addomain/group.go:129` | exact(同构机械替换) |
| `internal/agent/server/subprocess_stub_test.go` | test | streaming(子进程 stdin/stdout/signal) | 无仓内先例;GOROOT `src/os/os_test.go:2475-2516`(经 RESEARCH Pattern 4) | none |
| `internal/agent/server/subprocess_pgroup_test.go` | test | streaming | 自身(5 处 echo 即替换点) | exact |
| `internal/device/for_testing_guard_test.go` | test(meta AST 守护) | batch(全仓扫描) | `internal/models/status_constants_test.go:289-363` | role-match(parser 范式相同,遍历须 Glob→WalkDir) |

## Pattern Assignments

### 76-01a `pkg/cache/redis_miniredis_76_01_test.go` (test, Redis 命令面冒烟)

**Analog:** `pkg/cache/cache_74_08_test.go`(同包,同 testify 风格)

**测试文件骨架模式**(cache_74_08_test.go:1-23):
```go
package cache

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// helper 构造器:t.Helper() + t.Cleanup 生命周期绑定
func newMem(t *testing.T, maxSize int) *MemoryCache {
	t.Helper()
	m := NewMemoryCache(maxSize, 0)
	t.Cleanup(func() { _ = m.Close() })
	return m
}
```
新文件照此写 `newMiniredisCache(t)`(RESEARCH Pattern 1 已给全:`miniredis.RunT(t)` → `net.SplitHostPort(mr.Addr())` → `NewRedisCache(&CacheConfig{Host, Port}, "xingran")`)。

**被测构造器**(pkg/cache/redis.go:30-66,注意 PING 在构造器内 = R-3 冒烟点):
```go
func NewRedisCache(config *CacheConfig, keyPrefix string) (*RedisCache, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr: fmt.Sprintf("%s:%d", config.Host, config.Port),
		...
	})
	ctx, cancel := context.WithTimeout(context.Background(), dialTimeout)
	defer cancel()
	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis连接失败: %w", err)  // ← HELLO+CLIENT SETINFO 握手在此走完
	}
	return &RedisCache{client: rdb, prefix: keyPrefix}, nil
}
```
`CacheConfig` 字段见 `pkg/cache/cache.go:81-88`(Type/Host/Port/Password/DB/PoolSize/TLS);`buildKey`(redis.go:68-73)自动加 `xingran:` 前缀——测试断言键名时记得前缀存在。

**错误语义:** `redis.Nil` → `ErrNotFound`(redis.go:78-80);过期删除断言用 `ErrNotFound`。

**IMPORTANT — 过期注释联动:** `cache_74_08_test.go:15` 现有注释"Redis 实现依赖真实 Redis,不在单测范围(D-12 禁加 miniredis 依赖)"在 76-01 后失实(v1.27 D-02 已解禁)。76-01 plan 应顺带把该注释更新为指向 redis_miniredis_76_01_test.go(一行 doc-only 改动,避免评审困惑)。

---

### 76-01b `internal/services/operations/geocoding_httpmock_76_01_test.go` (test, httpmock PoC)

**Analog:** `internal/services/operations/geocoding_photo_floor_test.go`(Phase 74-07 同服务测试)

**既有 fakeGeocodeTransport 模式**(geocoding_photo_floor_test.go:28-63):
```go
type fakeGeocodeTransport struct {
	responses map[string]string // address -> body
	fallback  string
	err       error
	calls     int
	seenAddrs []string
}

func (f *fakeGeocodeTransport) RoundTrip(req *http.Request) (*http.Response, error) { ... }

const geocodeOKBody = `{"status":0,"result":{"location":{"lng":116.404,"lat":39.915},"formatted_address":"北京市东城区","level":"城市"}}`

func newGeocodeSvc(rt *fakeGeocodeTransport, redis pkgcache.Cache) *GeocodingService {
	svc := NewGeocodingServiceWithCache("ak-test", redis)
	svc.httpClient = &http.Client{Transport: rt, Timeout: 2 * time.Second}  // ← 白盒替换 Transport
	svc.rateLimiter = NewRateLimiter(100, time.Hour)
	return svc
}
```

**新测试的关键差异(planner 必写进 action):** httpmock PoC 的价值在 const-URL 零注入点场景——构造改用 `NewGeocodingService("ak-test")`(geocoding_service.go:73-82,`&http.Client{Timeout: ...}` Transport==nil → DefaultTransport),**不**替换 `svc.httpClient`。`httpmock.Activate(t)` 拦 DefaultTransport + `RegisterResponder` 命中 `baiduGeocodingAPIURL`(geocoding_service.go:24,const)。断言风格沿用 analog:`require.NoError` + `assert.InDelta(lng/lat)`;`geocodeOKBody` 可复制。rateLimiter 是共享全局 `BaiduAPIRateLimiter`,单次 Geocode 调用无需动;若用例多次调用则照 analog 白盒替换(同包可及)。

**文件头注释风格:** 照 analog :22-26 的 `// ======== Phase 76-01: ... ========` 分节注释 + 纪律说明(Activate(t) 自动清理优先)。

---

### 76-02a `internal/device/scrapli_wrapper.go` (MODIFY: 工厂 var 抽取)

**Analog:** 自身重复段——:148-162 与 :212-226 完全相同的六行:

```go
	// 创建平台实例
	p, err := platform.NewPlatform(
		platformName,
		device.IPAddress,
		opts...,
	)
	if err != nil {
		return nil, fmt.Errorf("创建平台实例失败: %w", err)
	}

	// 获取网络驱动
	d, err := p.GetNetworkDriver()
	if err != nil {
		return nil, fmt.Errorf("获取网络驱动失败: %w", err)
	}
```
(首段位于 `NewScrapliWrapper` :111-171;次段位于 `NewScrapliWrapperWithPort` :174-235。)

**Imports 已就绪**(scrapli_wrapper.go:12-17)——`platform`/`options`/`transport`/`util`/`network` 全部已 import,工厂 var 零新 import:
```go
import (
	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/scrapli/scrapligo/driver/network"
	"github.com/scrapli/scrapligo/driver/options"
	"github.com/scrapli/scrapligo/platform"
	"github.com/scrapli/scrapligo/transport"
	"github.com/scrapli/scrapligo/util"
)
```

**替换目标形态(照 RESEARCH Pattern 2):** 包级 `var newNetworkDriver = func(platformName, host string, opts ...util.Option) (*network.Driver, error)`;两处调用点改为 `d, err := newNetworkDriver(platformName, device.IPAddress, opts...)`,错误包装留在调用点保持文案不变。`connection_pool.go` 的 `createConnection`(:434-436 一带)经构造器零改动自动生效。**不要**顺手改 `platformIdentifier`(:93-98,锐捷 patched YAML 逻辑)或 Telnet/SSH 分支。

---

### 76-02b `internal/device/driver_factory_76_02_test.go` + `testdata/` (test, FileTransport 注入演示)

**Analog:** `internal/services/portwrite/port_write_e2e_test.go`(仓库唯一已验证的 FileTransport 先例)

**FileTransport 选项组合(照搬,勿自创)**(port_write_e2e_test.go:71-90):
```go
	p, err := platform.NewPlatform(
		"huawei_vrp",
		"dummy-host",
		options.WithTransportType(transport.FileTransport),
		options.WithFileTransportFile(fixturePath),
		options.WithTransportReadSize(1),
		options.WithReadDelay(0),
	)
	if err != nil {
		return fmt.Errorf("e2e: create platform: %w", err)
	}
	d, err := p.GetNetworkDriver()
	...
	if err := d.Open(); err != nil { ... }
	// Intentionally no defer Close — see type comment.
```

**禁 Close 纪律(注释原文,port_write_e2e_test.go:42-48):** "We deliberately do NOT call d.Close() ... the platform's `network-on-close` operations ... need to read more bytes from the FileTransport after the fixture has been fully consumed, which would block on FileTransport.Read `select{}`"。新测试必须同注释。

**fixture 路径定位(防 cwd 漂移)**(port_write_e2e_test.go:106-113):
```go
func e2eFixturePath(t *testing.T, name string) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Join(filepath.Dir(thisFile), "testdata", name)
}
```
fixture 放 `internal/device/testdata/` 新建(RESEARCH Open Question 3 已裁决:不复用 portwrite 的,Open 场景 prompt 形态不同);fixture 保持 LF(.gitattributes 已强制),断言用 TrimSpace 比对。

**var 覆盖纪律:** `orig := newNetworkDriver; t.Cleanup(func(){ newNetworkDriver = orig })`(先注册 Cleanup 再覆盖的写法);测试禁 `t.Parallel()`。

---

### 76-03a `internal/services/addomain/ldap_iface.go` (MODIFY: 接口 +3 方法)

**Analog:** 自身既有接口(:18-42)。当前 16 方法(实证计数;RESEARCH 记 18 系笔误):

```go
type LDAPClientIface interface {
	// 连接与生命周期
	Connect() error
	Close()
	// 只读搜索（分页）
	SearchOUs(baseDN string) ([]*ldap.Entry, error)
	...
	// 用户属性管理（user_ou_service 使用）
	UpdateUserAttribute(userDN string, attrs map[string]string) error
	MoveUser(userDN, newOUDN string) error
	EnableUser(userDN string) error
	DisableUser(userDN string) error
}
```

**追加位置与方法分组注释风格**(照 :29/:37 的"调用方"注释惯例):
```go
	// 组属性管理 / OU 管理 / DN 存在性（failover 闭包使用）
	UpdateGroupAttribute(groupDN string, attrs map[string]string) error
	CreateOU(ouDN, ouName string) error
	DNExists(dn string) (bool, error)
```
**零新实现已实证:** 三方法在具体类型上存在——`ldap_client.go:311`(UpdateGroupAttribute)/`:440`(CreateOU)/`:501`(DNExists);`OUExists`(:465)外部零使用不加。文件尾部既有编译期断言 `var _ LDAPClientIface = (*LDAPClient)(nil)`(:46)自动验证零遗漏,mock 侧同理(见 76-03c)。

---

### 76-03b `internal/services/addomain/failover_client.go` (MODIFY: clientFactory 字段 + 签名接口化)

**Analog:** 自身 struct(failover_client.go:20-28):

```go
type FailoverClient struct {
	pool   AccountPool
	config *models.ADConfig
}
```
改为加第三字段 `clientFactory func(*models.ADConfig, *models.ADServiceAccount) LDAPClientIface`(nil → `NewLDAPClient`)。

**两处 `NewLDAPClient(f.config, acct)` 调用点:** `ExecuteWithFailover` :54 与 `PickFirstConnect` :106——**只有 :54 走 newClient() 辅助方法;:106 保持原样**(`PickFirstConnect` 返回 `*LDAPClient` 具体类型签名不动,ad_authenticator.go:215 需 `.Conn()`)。

**签名改动点**(:33-36):
```go
func (f *FailoverClient) ExecuteWithFailover(
	ctx context.Context,
	operation func(client *LDAPClient) error,   // → func(client LDAPClientIface) error
) error {
```
循环体内 `client.Connect()` / `operation(client)` / `client.Close()`(:55-65)全在接口方法集内,零其它改动。错误路径(`MarkFailure` "dial:"/"operation:" 前缀、`ErrAllAccountsUnavailable`、maxHops 封顶 :45-48)一律不动。

---

### 76-03c `internal/services/addomain/ldap_client_mock_test.go` (MODIFY: mock 补 3 方法 + 调用记录)

**Analog:** 自身 preset 字段风格(:12-43):
```go
type mockLDAPClient struct {
	// 连接
	connectErr error
	closeErr   error
	// 搜索（按方法分别返回）
	searchUsersErr    error
	searchUsersRes    []*ldap.Entry
	...
	// 调用计数
	connectCalls   int
	closeCalls     int
	searchGrpCalls int
}
```
方法体照 :45-48(计数 + 返回 preset)与 :95-97(直接返回 err)的最简风格:
```go
func (m *mockLDAPClient) Connect() error {
	m.connectCalls++
	return m.connectErr
}
```
新增:`updateGroupAttrErr`/`createOUErr`/`dnExistsRes`+`dnExistsErr` preset 字段 + 对应计数;"walk/分页语义"按 RESEARCH Pattern 3 改动 4——`searchUsersFn func() ([]*ldap.Entry, error)` 函数字段(非 nil 时优先于 `searchUsersRes`,支持多批次)。**尾部编译期断言必须同步**:`var _ LDAPClientIface = (*mockLDAPClient)(nil)`(:112)——接口 +3 后漏补方法此处编译失败,即守护点。

---

### 76-03d `internal/services/addomain/failover_client_76_03_test.go` (test, 接口驱动 failover)

**Analog:** `internal/services/addomain/account_pool_test.go`——**其 helper 同包直接复用,勿重写**:

```go
// account_pool_test.go:17-49
func setupTestPool(t *testing.T) (AccountPool, *gorm.DB, string) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Exec(`CREATE TABLE sys_ad_service_accounts (...)`).Error)
	configID := uuid.NewString()
	pool := NewAccountPool(db, nil)
	return pool, db, configID
}

// account_pool_test.go:52-62
func insertAccount(t *testing.T, db *gorm.DB, configID, username string, status int) string
```
测试形态(RESEARCH Pattern 3 改动 4):`fc := NewFailoverClient(pool, cfg); fc.clientFactory = func(...acct) LDAPClientIface {...}`(同包白盒赋值)——账号 0 → Connect 失败 mock、账号 1 → 成功 mock,断言 MarkFailure/MarkSuccess 落库(failure_count 查 sys_ad_service_accounts 表,照 analog TC 用例的 db.Exec 查询式断言)与 operation 执行次数;池账号数 > DefaultMaxHops 时断言尝试封顶。断言风格:`assert.ErrorIs` / `require.NoError`(analog :69-70)。

---

### 76-03e 8 文件 × 20 处闭包机械替换 (MODIFY)

**Analog(代表性形态):** `group.go:128-136`:
```go
	fc := NewFailoverClient(s.pool, config)
	if err := fc.ExecuteWithFailover(ctx, func(client *LDAPClient) error {
		return client.AddGroupMember(groupDN, userDN)
	}); err != nil {
		if errors.Is(err, ErrAllAccountsUnavailable) {
			return fmt.Errorf("AD 账号池无可用账号，请先在 AD 配置页（详情 → 服务账号池 Tab）添加服务账号: %w", err)
		}
		return err
	}
```
仅把闭包参数 `*LDAPClient` → `LDAPClientIface`(scheduler 包为 `*addomain.LDAPClient` → `addomain.LDAPClientIface`);`NewFailoverClient` 调用、错误处理块(errors.Is ErrAllAccountsUnavailable 分支)全部原样。

**grep 实证清单(2026-08-23 复核,与 RESEARCH 一致):**

| 文件 | 行号 |
|------|------|
| `internal/services/addomain/group.go` | 129, 159, 200 |
| `internal/services/addomain/group_management_service.go` | 89, 160, 219, 280 |
| `internal/services/addomain/group_sync_service.go` | 60, 104 |
| `internal/services/addomain/user.go` | 154, 199, 220, 241 |
| `internal/services/addomain/user_ad_sync_service.go` | 68, 264, 535, 719 |
| `internal/services/addomain/dept_sync_service.go` | 79 |
| `internal/services/addomain/sync.go` | 105 |
| `internal/scheduler/dept_sync_tasks.go` | 183(唯一包外点,import addomain 限定符) |

闭包体内非接口方法仅 3 处:`group.go:201`(UpdateGroupAttribute)、`dept_sync_service.go:118`(CreateOU)、`user_ad_sync_service.go:187`(DNExists)——即 76-03a 接口扩展的精确依据。

---

### 76-04a `internal/agent/server/subprocess_stub_test.go` (test, TestHelperProcess re-exec)

**仓内无先例**(现有 5 处 echo 正是要清除的反模式)。权威参照:GOROOT `src/os/os_test.go:2475-2516`(经 RESEARCH Pattern 4 核实)。实现照 RESEARCH Pattern 4 的集中式 helper 骨架:
```go
func TestHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" { return } // 守卫必在第一行
	args := os.Args[len(os.Args)-1:]                          // "--" 之后的形态参数
	switch {
	case args[0] == "sleep-until-stdin-close": io.Copy(io.Discard, os.Stdin)
	...
	}
	os.Exit(0) // 每分支必达
}
```
测试侧:`newCommand(ctx, os.Args[0], "-test.run=^TestHelperProcess$", "--", "形态")` + `cmd.Env = append(os.Environ(), "GO_WANT_HELPER_PROCESS=1")`。SIGTERM 形态分支在 Windows 无实义——按 `runtime.GOOS` 守卫或平台分支断言(延续 sysproc 拆分哲学,见下)。

---

### 76-04b `internal/agent/server/subprocess_pgroup_test.go` (MODIFY: 5 处 echo → re-exec)

**Analog:** 自身。5 处 `exec.Command`/`runCommand`/`runCommandOutput` 的 "echo" 实参(:13, :29, :40, :58, :69):
```go
	cmd := newCommand(ctx, "echo", "hello")        // :13
	err := runCommand(ctx, "echo", "test")          // :29
	output, err := runCommandOutput(ctx, "echo", "hello")  // :40
	err = runCommand(ctx, "echo", "should not run") // :58
	cmd := newCommand(ctx, "echo", "test")          // :69
```
注意 :29/:40/:58 走的是 `runCommand`/`runCommandOutput`(subprocess.go:13-24,内部 `exec.CommandContext` + `setProcessGroup`),非 `newCommand`——这三处无法直接传 `cmd.Env`,需改测 `newCommand` 全链(cmd := newCommand(...); cmd.Env = ...; cmd.Run()/Output())或为 re-exec 场景统一走 newCommand 形态;planner 在 plan 中明确每处的改法。

**平台拆分参照(sysproc_windows.go 全文 12 行):**
```go
//go:build windows

package server

import "syscall"

var sysProcAttr = syscall.SysProcAttr{
	CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP,
}
```
`sysproc_linux.go` 对应 `Setpgid: true`。两文件本 phase 不动,仅作为"平台行为分歧用构建标签隔离"的哲学参照。

---

### 76-05 `internal/device/for_testing_guard_test.go` (test, AST 守护)

**Analog:** `internal/models/status_constants_test.go:289-348`(`readStatusConsts`)——parser 范式与 parse 失败即报错的语义直接照搬:

```go
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, file, nil, 0)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", file, err)  // ← fail 而非 skip,守护测试沿用
	}
```
另参照 :297-299 的"零匹配即报错"防呆:`if len(files) == 0 { return nil, fmt.Errorf("no Go files matched %s — run tests from the ... package directory", ...) }`——WalkDir 版应同设"扫描到 0 个 .go 文件即 Fail"守卫(防 cwd 漂移假绿)。

**与 analog 的两处必须偏离(RESEARCH Pattern 5 已给完整骨架):**
1. 遍历:`filepath.Glob`(analog :291-293,仅本包目录)→ `filepath.WalkDir("../..")` + 目录过滤(点前缀/vendor/node_modules/xingran-react-frontend/testdata/tests 一律 `fs.SkipDir`——`.claude/worktrees/` 下有 6 份仓库拷贝);
2. 检测目标:const 值锁定 → `*ForTesting` 后缀的 CallExpr.Fun / SelectorExpr.Sel 引用。

**白名单文件(守护规则的目标与豁免):** `internal/device/e2e_helpers.go:10-32`——头注释即三层契约的成文范本("Production code MUST NOT call this function. The naming suffix `ForTesting` is the physical isolation contract... Calling this in production silently skips connection pool bookkeeping, device-level locking, reachability checks, and credential decryption")。白名单判定用 `filepath.ToSlash(path) == "internal/device/e2e_helpers.go"` 精确匹配。全仓 `ForTesting` 符号现仅此一文件(RESEARCH grep 实证)。

---

## Shared Patterns

### 1. testify 断言 + helper 构造器(t.Helper + t.Cleanup)
**Source:** `pkg/cache/cache_74_08_test.go:18-23`、`internal/services/addomain/account_pool_test.go:17-49`、`internal/services/portwrite/port_write_e2e_test.go:106-113`
**Apply to:** 本 phase 全部 6 个新测试文件。构造 helper 一律 `t.Helper()`;持有生命周期资源的(miniredis、sqlite pool、被覆盖的 var)一律 `t.Cleanup` 注册恢复/关闭,且 Cleanup 注册先于资源生效。

### 2. 编译期接口断言
**Source:** `internal/services/addomain/ldap_iface.go:46`、`internal/services/addomain/ldap_client_mock_test.go:112`
```go
var _ LDAPClientIface = (*LDAPClient)(nil)
var _ LDAPClientIface = (*mockLDAPClient)(nil)
```
**Apply to:** 76-03 接口扩展后的两侧断言(mock 侧即"补 3 方法零遗漏"的编译守护);76-02 若引入工厂类型别名同样适用。

### 3. 同包白盒测试(internal test package)
**Source:** `geocoding_photo_floor_test.go:60`(`svc.httpClient = ...` 直接改私有字段)、`account_pool_test.go`(调用未导出 NewAccountPool 路径)
**Apply to:** 本 phase 所有新测试文件一律 `package <生产包名>`(非 `_test` 后缀外部包)——geocoding PoC 不换 Transport 但需读私有字段场景、failover 测试赋值 `fc.clientFactory`、device 测试覆盖包级 var,全部依赖同包可见性。

### 4. 注入缝覆盖纪律
**Source:** RESEARCH Pattern 2(反例清单)+ portwrite 先例注释
**Apply to:** `driver_factory_76_02_test.go`(覆盖 `newNetworkDriver`)、`failover_client_76_03_test.go`(赋值 `clientFactory`):覆盖/注入必须 `t.Cleanup` 恢复、测试禁 `t.Parallel()`、覆盖动作与断言之间不得有 `t.Fatal`(用 `t.Errorf` 或先构造后覆盖,防 Cleanup 前退出)。

### 5. fixture 定位与卫生
**Source:** `port_write_e2e_test.go:106-113`(runtime.Caller + testdata)
**Apply to:** `internal/device/testdata/` 新 fixture:LF 行尾(.gitattributes 强制)、dummy 值(dummy-host 先例,禁真实凭据/IP)、路径经 runtime.Caller 派生。

### 6. ForTesting 三层隔离契约文案
**Source:** `internal/device/e2e_helpers.go:10-31`
**Apply to:** 76-05 守护测试的文件头注释按此契约成文(编译器层/命名层/AST 层),SC#5 验收即引用该文案。

## No Analog Found

| File | Role | Data Flow | Reason |
|------|------|-----------|--------|
| `go.mod` 修改 | config | — | 仓内无 test-only 依赖先例;require 注释格式按 RESEARCH SC#1 固定文案 `// test-only (v1.27 D-02)`,版本 pin v2.38.0 / v1.4.2 |
| `internal/agent/server/subprocess_stub_test.go` | test | streaming | 仓内首例 TestHelperProcess;实现照 RESEARCH Pattern 4(源头 GOROOT src/os/os_test.go:2475-2516) |

miniredis/httpmock 的 API 用法本身(RunT/FastForward/Activate(t)/RegisterResponder)同样无仓内先例——planner 直接引用 RESEARCH §Pattern 1 与 §Code Examples 的已核实片段,勿让执行者自由发挥。

## Metadata

**Analog search scope:** pkg/cache、internal/services/{operations,addomain,portwrite}、internal/device、internal/agent/server、internal/models、internal/scheduler
**Files scanned:** 约 30(Glob 定位 4 包全清单 + 精读 12 + grep 复核 ExecuteWithFailover 20 处 / LDAPClient 3 方法 / CacheConfig / fakeGeocodeTransport)
**行号复核声明:** RESEARCH 引用的全部关键行号本次逐一 Read/Grep 复核一致;唯一偏差 = ldap_iface.go 方法数 16(非 18)。
**Pattern extraction date:** 2026-08-23
