---
status: issues_found
files_reviewed: 19
critical: 0
warning: 4
info: 7
total: 11
---

# Code Review: config 包重构 — ctx 参数 + viper 全局隔离 + API 元数据规范化

**Reviewed:** 2026-08-12T17:30:00Z
**Depth:** standard
**Commit:** `e309829` (refactor/config-ctx-and-viper-cleanup)
**Files Reviewed:** 19
**Status:** issues_found

## 范围核实

| 项目 | 结论 |
|------|------|
| `go build ./...` 通过 | 已二次执行,无输出(成功)。 |
| `go test -race -count=1 ./internal/config/...` 通过 | 已二次执行,`ok ... 2.293s`,无 race 告警。但包内测试不触发跨包 lazy-load 路径(见 WR-2)。 |
| `rpa-worker/cmd/main.go:30` 调用 `config.Load(*configFile)` | **不属于本仓库构建图**。`rpa-worker/` 是独立 Go module(`github.com/xingran-next/rpa-worker`, Go 1.21),有自己的 `config` 包,不受本次签名变更影响。已确认。 |
| `scripts/.archive-*/` 内 4 处 `cfg := config.Load()` 旧签名调用 | **被 Go 工具忽略**(目录名以 `.` 开头,`go build ./...` 不扫描),不破坏构建。已确认。 |
| 15 个调用方 ctx 传递 | 启动期/main 入口均用 `context.Background()`(cmd/main.go、core.go、vdi/config.go、ad_ldap_client.go、11 个 scripts),语义合理。无把 request ctx 传给可能跨请求的操作的情况——所有 Load 调用都在启动路径或 sync.Once 懒加载里。 |
| 改动范围 | 4 个 config 包文件 + cmd/main.go + internal/core/core.go + ad_ldap_client.go + vdi/config.go + 11 个 scripts,共 19 文件 / 1048 插入。与 task description 一致。 |

## Summary

重构的"机械动作"基本正确:`Load(ctx)` 签名扩展 + 15 个调用方错误处理补全是完整的(逐一核对 git diff,无遗漏的 error 丢弃);`viper.Reset()` 隔离全局状态的意图正确,`TestLoad_ResetState` 也验证了 reload 场景下不污染;`LoadAPIMetadata` 的 goroutine + 缓冲 channel + select 取消实现是**正确的**——缓冲 channel 保证 goroutine 不泄漏、无 `default` 分支不忙等(已读实际代码逐行验证)。

但重构在**两个语义层面引入了行为变更**,且**索引化的 GetEndpointByRoute 有真实的查询-存储规范化不对称**:

1. **`GetEndpointByRoute` 只在查询侧规范化 `method`(ToUpper+TrimSpace),不规范化 `route`** —— 与 `normalize()` 在存储侧对 route 做的规范化不对称。HTTP handler(`dashboard_handler.go:694`)直接把用户 query string 传入,带尾斜杠/缺前导斜杠的合法路由会得到假阴性"端点不存在"。
2. **`GetEndpointByRoute` 改为返回指向 `c.Metadata[i].Endpoints[j]` 的实时指针**(通过 index map),而旧实现返回的是 range loop 变量的地址(Go 1.22+ 是每次迭代的副本拷贝)。这把"返回快照指针、调用方改不影响内部"的隐含契约改成了"返回内部状态可变引用",违背了 struct doc 声明的"从磁盘加载后只读"不变量。
3. **`viper.Reset()` 是全局 destructive 操作**。VDI(`tlsSkipVerifyOnce`)和 AD(`adTLSSkipVerifyOnce`)的懒加载是两个独立 `sync.Once`,首次 VDI 请求 + 首次 AD 请求并发时会触发两次 `config.Load()` → 在全局 viper 上 race(既有问题,被 Reset 加重)。
4. **`GetAllEndpoints` 浅拷贝只复制顶层 `ModuleMetadata` 切片**,嵌套的 `Endpoints []EndpointMeta` 切片共享底层数组——doc 声称"调用方可以安全地修改元素"过度,测试也没覆盖嵌套修改。

未发现 critical 级(security/data-loss/crash)。4 个 warning 集中在规范化不对称、可变引用契约变更、全局状态并发、浅拷贝语义;7 个 info 是测试弱点与既有行为变更记录。

---

## Critical

无。

---

## Warning

### WR-1: `GetEndpointByRoute` 查询侧不规范化 `route`,与存储侧 `normalize()` 不对称

**File:** `internal/config/api_metadata_loader.go:138-141`

**当前实现:**
```go
func (c *APIMetadataConfig) GetEndpointByRoute(route, method string) *EndpointMeta {
	c.indexOnce.Do(c.buildIndex)
	return c.index[c.endpointIndexKey(strings.ToUpper(strings.TrimSpace(method)), route)]
}
```

`method` 被 `strings.ToUpper(strings.TrimSpace(...))` 实时规范化,`route` **原样拼接**。但索引键是在 `normalize()`(行 80-91)把存储端的 `ep.Route` 规范化(TrimSpace + 剥首尾 `/` + 加单个前导 `/`)之后由 `buildIndex()` 构建的。

**结果:** 以下合法查询全部 miss(返回 nil):
- 查询 `"/system/users/list/"`(尾斜杠)→ 索引键 `"POST /system/users/list"`,查询键 `"POST /system/users/list/"` → miss
- 查询 `"system/users/list"`(无前导 `/`)→ miss
- 查询 `"  /system/users/list  "`(前后空格)→ miss

`endpointIndexKey`(行 49-53)的注释只描述了**存储侧**:
```go
// HTTP method 统一大写(GET/POST);Route 由 normalize() 规范化后已带前导 /、
// 无尾部 /。 直接拼接即可。
```
"直接拼接即可"对存储侧成立,对查询侧不成立——查询侧的 route 未走 normalize。

**实际触发路径(非纯理论):**
- `internal/api/v1/system/dashboard_handler.go:694-702`:`route := c.Query("route")` 直接取用户 query string,`h.service.ValidateEndpoint(route, method)` → `GetEndpointByRoute`。用户在前端输入带尾斜杠的路由即得到 `404 端点不存在`。
- `internal/services/system/widget_data_fetcher.go:153`:`apiConfig.Endpoint` 来自 DB 存储的 widget 配置。若 DB 中存的值带尾斜杠(数据录入错误),同样 miss。

**测试没覆盖这点:** `TestLoadAPIMetadata_Normalize`(`api_metadata_loader_test.go:119-169`)的用例名误导——"带尾部斜杠 → 命中"指的是 **YAML 源数据**带尾斜杠,而**查询**用的 route 是已规范化的 `/system/users/list`(无尾斜杠):
```go
{"带尾部斜杠 → 命中", "/system/users/list", "POST", true},  // 查询 route 无尾斜杠
```
没有任何用例验证"查询带尾斜杠 route 是否命中"。

**Fix 选项(任选):**
- **A(推荐,对称规范化):** 在 `GetEndpointByRoute` 内对 route 做与 `normalize()` 相同的规范化,提取成 helper:
  ```go
  func normalizeRoute(r string) string {
      r = strings.TrimSpace(r)
      r = strings.TrimLeft(r, "/")
      r = strings.TrimRight(r, "/")
      return "/" + r
  }
  // normalize() 和 GetEndpointByRoute 都调这个 helper
  ```
- **B:** 在 `endpointIndexKey` 内对入参统一规范化(method + route),让键构造对称。
- 至少补一个测试:查询 `"/system/users/list/"`、`"system/users/list"`、`"  /system/users/list  "` 应全部命中。

### WR-2: `viper.Reset()` 全局 destructive + VDI/AD 双懒加载 → 并发 race(既有问题被加重)

**File:** `internal/config/config.go:266`(新增 `viper.Reset()`)
**关联:** `internal/services/vdi/config.go:16,25-32`、`internal/services/ad_ldap_client.go:21,30-36`

`viper` 是进程级全局单例(`viper.SetDefault` / `viper.BindEnv` / `viper.Reset` 都操作 `globalViper`,内部 map 无 mutex)。

**race 路径(已逐行核对):**
- VDI 懒加载由 `tlsSkipVerifyOnce`(`vdi/config.go:16`)保护,AD 懒加载由 `adTLSSkipVerifyOnce`(`ad_ldap_client.go:21`)保护——**两个独立的 `sync.Once`**。
- 首次 VDI 请求 + 首次 AD 请求并发到达时:
  - goroutine A:`tlsSkipVerifyOnce.Do(...)` → `config.Load(ctx)` → `viper.Reset()` → 清空全局状态 → `setDefaults()` / `BindEnv` / `ReadInConfig` 一系列写
  - goroutine B:`adTLSSkipVerifyOnce.Do(...)` → `config.Load(ctx)` → `viper.Reset()` → 再次清空(可能清掉 A 刚 set 的状态)→ 再次写
  - 两者的全局 viper map 写入交错 → data race(`go test -race` 会告警,结果是 undefined behavior)

**为什么 `viper.Reset()` 使既有 race 变更糟:**
- 旧 `Load()` 没有 Reset,最坏情况是重复 `SetDefault`/`SetConfigName`(近似幂等)。
- 新 `Load()` 的 Reset 是 **destructive** —— B 的 Reset 可以在 A 的 `setDefaults()` 与 `viper.Unmarshal()` 之间清空 A 刚写入的全部默认值,导致 A 拿到一个被部分清空的 Config(零值)。
- 影响字段包括 `VDI.TLSSkipVerify` / `AD.TLSSkipVerify`(默认 true,零值 false)。若 race 让某一方读到零值 false,会在自签名证书环境触发连接失败;反向(零值 true 覆盖配置的 false)则是静默 TLS 校验关闭 —— 安全风险。

**既有 vs 新增:** race 本身在旧 `Load()` 就存在(懒加载模式未变),但本次重构新增的 `viper.Reset()` 扩大了破坏面。属于"重构加重的既有问题"。

**复现窗口:** 极窄(仅"服务启动后,第一次 VDI 请求"与"第一次 AD 请求"必须正好并发)。但 data race 一旦触发是 undefined behavior,不应依赖窗口窄来豁免。

**Fix 选项:**
- **A(根治,推荐):** 让 `Load` 不再依赖全局 viper。内部用 `viper.NewWithOptions(...)` 创建独立实例,所有 `SetDefault`/`BindEnv`/`ReadInConfig`/`Unmarshal` 都在私有实例上操作。这样彻底消除全局状态,Reset 也变得不必要。
- **B(缓解):** 给 `Load` 加一个 package 级 `sync.Mutex`,串行化所有 Load 调用。简单但 viper 全局仍被多份配置共用(若未来有 reload 场景会受限)。
- **C(最小改动):** VDI / AD 的懒加载不再各自调 `config.Load()`;改为在 `core.Core` 启动时一次性加载并把 `*Config` 注入这两个包(消除请求路径上的 Load 调用)。

`go test -race ./internal/config/...` 当前通过,因为包内测试不构造跨包并发 Load;需要跨包集成测试或 race scenarios 才能复现。

### WR-3: `GetEndpointByRoute` 返回指向内部状态的实时指针,违背"只读"契约(行为变更)

**File:** `internal/config/api_metadata_loader.go:61-69`(buildIndex),`:138-141`(GetEndpointByRoute)

**新实现(buildIndex 存的是原始指针):**
```go
func (c *APIMetadataConfig) buildIndex() {
	c.index = make(map[string]*EndpointMeta, len(c.Metadata)*8)
	for i := range c.Metadata {
		for j := range c.Metadata[i].Endpoints {
			ep := &c.Metadata[i].Endpoints[j]        // ← 指向 c.Metadata 内部
			c.index[c.endpointIndexKey(ep.Method, ep.Route)] = ep
		}
	}
}
```
`GetEndpointByRoute` 返回 `c.index[key]` —— 一个指向 `c.Metadata[i].Endpoints[j]` 的实时指针。

**旧实现(main 分支):**
```go
for _, endpoint := range module.Endpoints {
    if endpoint.Route == route && endpoint.Method == method {
        return &endpoint   // ← Go 1.22+ range 变量是每次迭代的副本
    }
}
```
Go 1.22+ 的 range 变量每次迭代是独立变量,`&endpoint` 指向该迭代的**栈/堆副本**,调用方修改不影响 `c.Metadata`。

**差异(行为变更):**
- 旧:返回快照指针,调用方改写 `ep.Route = "x"` 不污染配置。
- 新:返回内部状态指针,调用方改写 `ep.Route = "x"` **直接修改 `c.Metadata` 并使索引键失效**(索引里 `"POST /x"` 这个键不存在,原键 `"POST /original"` 仍指向被改的元素)。

**违背的 doc 契约(`api_metadata_loader.go:14-17`):**
```go
// APIMetadataConfig API 元数据配置。
//
// 线程安全:本结构从磁盘加载后只读,所有方法可并发调用。
// Metadata 字段在初始化完成后不再修改;endpointIndex 由 sync.Once
// 保护懒加载,只构建一次。
```
"只读 / 初始化完成后不再修改"是 struct 级不变量,但 `GetEndpointByRoute` 返回的可变 `*EndpointMeta` 让调用方**有能力**违反它。若任一调用方写返回值,会与并发读 race。

**当前调用方安全吗?** `api_endpoint_service.go:155-178` 只读字段(`endpointMeta.Route` 等),未改写。所以**当前**无 bug。但 API 表面已经暴露了可变引用,未来调用方或测试一旦改写就会触发 race + 索引失效。

**Fix:**
- 返回值拷贝(改动小,API 兼容):
  ```go
  func (c *APIMetadataConfig) GetEndpointByRoute(route, method string) *EndpointMeta {
      c.indexOnce.Do(c.buildIndex)
      ep := c.index[c.endpointIndexKey(strings.ToUpper(strings.TrimSpace(method)), route)]
      if ep == nil {
          return nil
      }
      copy := *ep   // 值拷贝,返回独立副本的指针
      return &copy
  }
  ```
- 或:doc 显式声明"返回的指针指向内部状态、调用方必须只读",并把 struct 注释从"只读"改为"调用方不得修改返回的 EndpointMeta"。前者(返回拷贝)更安全,推荐。

### WR-4: `GetAllEndpoints` 浅拷贝不彻底,doc 与测试均高估安全性

**File:** `internal/config/api_metadata_loader.go:143-152`

**当前实现:**
```go
func (c *APIMetadataConfig) GetAllEndpoints() []ModuleMetadata {
	snapshot := make([]ModuleMetadata, len(c.Metadata))
	copy(snapshot, c.Metadata)
	return snapshot
}
```

`copy` 对 `[]ModuleMetadata` 做浅拷贝:顶层 `ModuleMetadata` struct 按值复制,但 `ModuleMetadata.Endpoints []EndpointMeta` 是 slice header——**底层数组与 `c.Metadata[i].Endpoints` 共享**。

**doc 声称(api_metadata_loader.go:144-147):**
```go
// 返回浅拷贝切片,调用方可以安全地 append 或修改元素,不会影响内部状态。
// 指针字段(*EndpointMeta)仍指向内部,不可修改其内容。
```
- "调用方可以安全地修改元素"对**顶层** `snapshot[i].Module = "x"` 成立(改的是副本 struct 的字段)。
- 对**嵌套** `snapshot[0].Endpoints[0].Route = "x"` **不成立**——这直接写共享底层数组,污染 `c.Metadata[0].Endpoints[0]`。
- "指针字段(*EndpointMeta)"措辞也有误:`Endpoints` 元素类型是值类型 `EndpointMeta`,不是 `*EndpointMeta`。doc 提到的指针并不存在于这个 slice 里。

**测试(`api_metadata_loader_test.go:264-268`)只验证顶层修改:**
```go
all[0].Module = "tampered"
if cfg.Metadata[0].Module == "tampered" {   // ← 只测顶层字段
    t.Error(...)
}
```
没有任何用例测 `all[0].Endpoints[0].Route = "x"` 是否污染 `cfg.Metadata`。

**Fix:**
- **A(根治):** 深拷贝,嵌套 slice 也复制:
  ```go
  snapshot := make([]ModuleMetadata, len(c.Metadata))
  for i := range c.Metadata {
      snapshot[i] = c.Metadata[i]
      eps := make([]EndpointMeta, len(c.Metadata[i].Endpoints))
      copy(eps, c.Metadata[i].Endpoints)
      snapshot[i].Endpoints = eps
  }
  ```
- **B(改 doc + 补测试):** 如果有意保持浅拷贝(性能),doc 改成"顶层元素可改、嵌套 Endpoints 仍是共享引用、调用方不得修改 Endpoints 元素",并补一个测试验证嵌套修改**会**污染(锁定契约)。

---

## Info

### IN-1: `Load(ctx)` 的 ctx 当前被显式丢弃(`_ = ctx`)

**File:** `internal/config/config.go:261-262`

```go
func Load(ctx context.Context) (*Config, error) {
	_ = ctx // 当前 viper 不支持 ctx 取消,保留参数为未来扩展。
```

ctx 参数纯文档化("配置加载可被取消"的意图),当前无任何取消语义。`LoadAPIMetadata` 的 ctx 确实生效(goroutine + select),但 `Load` 的不会。注释已坦白说明,不算 bug,但调用方可能误以为传 `context.WithTimeout` 能限制 Load 的阻塞时间——实际不能(`viper.ReadInConfig` 阻塞读、不可中断)。

**Fix:** 可选——在 doc 里更明确写"ctx 当前为 placeholder,不影响 Load 的阻塞行为",或直接在 viper 替换为可中断 reader 后再补 ctx 参数(避免 API 提前泛化)。

### IN-2: `TestLoad_ResetState` 断言过弱,不验证默认值回归

**File:** `internal/config/config_test.go:246-254`

```go
// 第二次:不再设 SERVER_PORT,期望 Reset() 已清空,走默认 8080。
t.Setenv("SERVER_PORT", "")
cfg2, err := Load(context.Background())
...
if cfg2.Server.Port == 12345 {
    t.Errorf("第二次 Load Port 仍为 12345,viper.Reset() 未生效或 SERVER_PORT 空串被当作 12345")
}
```

只断言 `!= 12345`,不断言 `== 8080`。若 viper 在空 env 下返回 0(int 零值),测试仍通过,但 Port=0 是无效的服务端口,实际行为已经 broken。

**Fix:**
```go
if cfg2.Server.Port != 8080 {
    t.Errorf("第二次 Load Port 应回归默认 8080,实际 %d", cfg2.Server.Port)
}
```

### IN-3: `normalize()` 仅在 `LoadAPIMetadata` 调用,未作为构造不变量强制

**File:** `internal/config/api_metadata_loader.go:128`(调用点),`:80-91`(实现)

`APIMetadataConfig` 是导出类型,调用方可以直接字面量构造:
```go
cfg := &config.APIMetadataConfig{Metadata: []ModuleMetadata{{Endpoints: []EndpointMeta{{Route: "/foo", Method: "get"}}}}}
cfg.GetEndpointByRoute("/foo", "get")  // 查询侧 method 规范化为 GET
```
此时 `buildIndex` 用未规范化的 `"get"` 存键,查询用 `"GET"` 查键 → miss。`normalize()` 没被强制执行。

当前仓库内唯一构造路径是 `LoadAPIMetadata`(安全),所以**当前**无 bug。但导出类型 + 导出方法意味着外部 importer 可能踩坑。

**Fix(可选):** 在 `buildIndex` 开头调一次 `c.normalize()`(幂等,多次调用无害),保证无论构造路径如何索引都基于规范化字段。或把 `APIMetadataConfig` 改为非导出 + 只暴露构造函数。

### IN-4: `buildIndex` 静默覆盖重复键,无重复检测

**File:** `internal/config/api_metadata_loader.go:61-69`

```go
c.index[c.endpointIndexKey(ep.Method, ep.Route)] = ep
```

map 赋值对重复键静默覆盖(后写赢)。若 YAML 中两条端点经 `normalize()` 后产生相同键(例如同一 route 同时有 `method: "get"` 和 `method: "GET"`,normalize 后都是 `GET`),第二条覆盖第一条,**无任何告警或日志**。

**行为变更:** 旧线性扫描返回**第一个**匹配;新 map 返回**最后一个**(后写赢)。调用方无法察觉数据丢失。

**Fix(可选):** buildIndex 时检测重复键并 `applogger.Warnf` 记录;或返回 error 让 LoadAPIMetadata 拒绝加载有歧义的元数据。

### IN-5: 行为变更 —— 默认值移除 + `Validate()` 强制非空,既有部署升级会启动失败(重构引入,有意识的安全加固)

**Files:** `internal/config/config.go:497`(sm4_key 默认 "" ),`:454-457`(password 默认注释),`:388-390`(Validate 拒绝空 SM4Key)

变更(对比 main 分支):
- `security.sm4_key` 默认值 `"dGVzdC1zZWNyZXQxNiEhIQ=="`(公开已知测试串)→ `""`,且 `Validate()` 现在拒绝空值。
- `database.password` 默认值 `"postgres"` → `""`。
- `jwt.secret_key` 默认值在更早已是 `""`,本次新增 `Validate()` 不强制非空(仅 `warnSecurityRisks` 告警)。

**影响:** 既有部署若依赖默认 SM4_KEY 而未设环境变量,升级到本次重构后 `Load()` 直接返回 error → 进程启动失败(cmd/main.go `os.Exit(1)`)。这是**有意的安全加固**(移除公开已知弱默认),但在一个标记为 "refactor" 的 commit 里夹带 startup-fail 行为变更,容易让运维意外。

**Fix(流程性,非代码):** 在 commit message / CHANGELOG / 升级文档中显式标注"破坏性变更:必须显式提供 SM4_KEY",不要把它埋在 refactor commit 里。代码层无需改动(加固方向正确)。

### IN-6: `bindEnvVars` 改用 viper 的 `cast.ToInt` / `cast.ToBool`,与旧的 `strconv.Atoi` / 自定义 bool 解析存在细微差异

**File:** `internal/config/config.go:322-370`(新 bindEnvVars),对照 main 分支旧 `overrideFromEnvInt` / `overrideFromEnvBool`

旧:
```go
// overrideFromEnvInt
if p, err := strconv.Atoi(value); err == nil { *target = p }
// overrideFromEnvBool
switch strings.ToLower(value) { case "true","1","yes": ...; case "false","0","no": ... }
```
新(委托给 viper Unmarshal,viper 内部用 `cast`):
- `cast.ToInt("0x10")` → 16(支持十六进制);`strconv.Atoi("0x10")` → error,保持原值。
- `cast.ToBool` 接受 `1/t/T/true/TRUE/True`(更宽松);旧实现只接受 `true/1/yes/false/0/no`。

**实际影响:** 极低(谁会用 `SERVER_PORT=0x1F90`),但属于"重构里的隐含语义变更"。

**Fix(可选):** 如果要严格保留旧行为,继续在 `Unmarshal` 后做一次显式 `os.Getenv` + `strconv.Atoi` 覆盖(但失去 bindEnvVars 的集中声明好处)。或在 doc 注释里记录这个差异。

### IN-7: `TestLoadAPIMetadata_Normalize` 用例名与实际查询 route 不符,造成"已覆盖尾斜杠查询"的错觉

**File:** `internal/config/api_metadata_loader_test.go:138-149`

```go
{"带尾部斜杠 → 命中", "/system/users/list", "POST", true},
```
用例名暗示"查询带尾斜杠",但查询 route 是 `/system/users/list`(无尾斜杠)。三个用例的查询 route 全是规范化形态,YAML 源数据才有尾斜杠/小写/空格差异。

这个测试**没验证 WR-1**(查询侧 route 规范化),反而给人"WR-1 已覆盖"的错觉。

**Fix:** 补一组用例,查询用未规范化 route(`"/system/users/list/"`、`"system/users/list"`、`"  /system/users/list  "`)验证命中;或修用例名让"带尾斜杠"明确指 YAML 源。

---

## 已验证为正确的实现点(免复查)

- **`LoadAPIMetadata` goroutine + select 无泄漏、无忙等。** `readCh` 是 buffered chan(size 1),goroutine 永远能投递结果后退出,无泄漏;`select` 无 `default` 分支,不会忙等;ctx 取消时主流程立即返回,goroutine 后台读完文件后向 buffered chan 投递并退出。实现正确。
- **15 个调用方的 error 检查完整。** 逐一核对 diff:cmd/main.go(exit 1)、core.go(LoadAPIMetadata 已检查 err)、ad_ldap_client.go(panic)、vdi/config.go(panic)、11 个 scripts(exit 1 / log.Fatal)。无遗漏的 `_ =` 丢弃。
- **`bindEnvVars` 的 error 返回路径被 `Load` 正确检查。** `config.go:293-295`:`if err := bindEnvVars(); err != nil { return nil, fmt.Errorf(...) }`。无丢弃。
- **`TestLoad_*` 三例均 `t.Setenv("SM4_KEY", ...)`,避免被新 `Validate()` 拦截。** 测试与 Validate 契约一致。
- **`viper.Reset()` 的单线程 reload 场景(测试场景)生效。** `TestLoad_ResetState` 验证了"第二次 Load 不受第一次 BindEnv 污染"(虽然断言弱,见 IN-2)。

---

_Reviewed: 2026-08-12T17:30:00Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
