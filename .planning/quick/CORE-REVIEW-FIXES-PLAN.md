# internal/core 代码审查修复计划（12 项）

> **来源：** `/golang-pro` 审查 `internal/core`（2026-08-13）
> **搁置：** S1（LDAP `InsecureSkipVerify`）—— 用户确认无证书，暂不处理
> **状态：** 计划已落盘，**未执行任何代码改动**，待新会话 fresh context 执行

---

## 执行前置（新会话第一步）

```bash
# 1. 基于 main 建独立分支（不要在 phase 57 的 refactor/config-ctx-and-viper-cleanup 上做）
git switch main && git pull --ff-only
git switch -c refactor/core-review-fixes

# 2. 全程约束
#    - 每个批次改完立即 `go build ./...` 验证编译
#    - 全部完成后 `go test ./internal/core/...` 跑回归
#    - 不动 internal/core/db/migrations/archive/（历史归档迁移）
#    - P3 大重构每项单独提交粒度
```

---

## 批次 1 — `db/database.go` 同文件小修复（合并改，1 次 go build）

### B1 [P1 正确性] 删除重复的 `auditConstraintNaming()` 调用
- **位置：** `db/database.go:399-403`
- **问题：** `auditConstraintNaming()` 连同其上方的注释被**复制粘贴了两次**，每次启动跑两遍 PG 审计查询。
- **修复：** 删除第二组（第 402-403 行的注释 + 调用），保留第一组。

### Q5 [P2 质量] 删除 `configureGORM` 死代码
- **位置：** `db/database.go:131-134`（空函数）+ `db/database.go:48`（调用处）
- **问题：** `func configureGORM(_ *gorm.DB) error { return nil }` 是空操作却被 `NewDatabase` 调用。
- **修复：** 删除函数定义，并删除 `NewDatabase` 里 `if err := configureGORM(db); err != nil { ... }` 这段调用（第 48-50 行）。

### S2 [P0 安全] `CREATE DATABASE` 标识符校验
- **位置：** `db/database.go:797`（`createDatabaseIfNotExists`）
- **问题：** `fmt.Sprintf("CREATE DATABASE %s", dbName)` 直接拼接，`dbName` 未校验。PG 的 `CREATE DATABASE` 不支持 `$1` 占位符所以必须拼接，但需防御非法标识符。
- **修复：** 在函数开头校验 `dbName`：
  ```go
  import "regexp"
  var dbIdentRe = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)
  // ...
  if !dbIdentRe.MatchString(dbName) {
      return fmt.Errorf("非法数据库名 %q（仅允许 [a-zA-Z_][a-zA-Z0-9_]*）", dbName)
  }
  ```
  拼接处可再用 `pq.QuoteIdentifier(dbName)` 双保险（需 import `github.com/lib/pq`，已在文件内）。

**批次 1 验证：** `go build ./...`

---

## 批次 2 — `security/password.go` 注释（无逻辑改动）

### Q7 [P2 质量] `pbkdf2SM3` 多块未实现的 latent bug 注释
- **位置：** `security/password.go:66-94`
- **问题：** 计数器硬编码 `0x00 0x00 0x00 0x01`（第 73 行），只算 PBKDF2 的 block 1。当 `keyLen == 32 == SM3 摘要长度`（密码哈希场景）正确；`keyLen > 32` 会出错。
- **修复：** 在函数 doc 注释里明确限定（不改逻辑）：
  > 注意：本实现仅支持 `keyLen ≤ 32`（= SM3 摘要长度）。计数器固定为 block 1。
  > 密码哈希场景 `keyLen=32` 安全。若未来以 `keyLen > 32` 调用，需补全多块循环（每个 block i 用 `INT32_BE(i)` 作计数器）。

**批次 2 验证：** `go build ./...`（注释改动，主要确认无语法误伤）

---

## 批次 3 — 子进程 reaper 日志库统一（2 文件）

### Q2 [P2 质量] reaper 从 `logrus` 改 `applogger`
- **位置：** `subprocess_reaper_linux.go`（import 第 10 行 + 3 处 `logrus.Info/Debug`）、`subprocess_reaper_windows.go`（import 第 8 行 + 1 处 `logrus.Debug`）
- **问题：** 整个 core 包用项目封装 `applogger`（`github.com/xingran-next/xingran-go-backend/pkg/logger`），唯独 reaper 两文件裸用 `logrus`，绕过统一日志格式/级别。
- **修复：**
  - 替换 import：`"github.com/sirupsen/logrus"` → `applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"`
  - `logrus.Info(...)` → `applogger.Infof(...)`，`logrus.Debug(...)` → `applogger.Debugf(...)`（参数格式从 logrus 的 `msg string, fields` 适配 applogger 的 printf 风格，注意 reaper 里调用都是纯字符串无 fields，直接 `Infof("%s", ...)` 或保持原串）
  - linux 文件：第 22、26、36 行；windows 文件：第 15 行

**批次 3 验证：** `go build ./...`

---

## 批次 4 — `core.go` 缩进 + `captcha.go` 限流

### Q1 [P2 质量] `Init()` gofmt 缩进 + 步骤编号
- **位置：** `core.go:346-433`（`{ ... }` 调度器注册代码块）
- **问题：** 内部缩进层次错乱（尤其 407-422 行多一层缩进）；步骤注释出现两个 "9."。
- **修复：** `gofmt -w internal/core/core.go` 自动修正缩进；手动把重复的 "9." 步骤注释改为连续编号（9 → 9.1 → 10 → 11...，或重排为 9/10/11）。gofmt 后务必 `git diff` 人工确认只动了空白与注释，没动逻辑。

### S3 [P1 安全] 验证码 IP 限流 fail-open
- **位置：** `captcha.go:255-272`（`GenerateCaptcha` 里 `IncrementWithExpire` 失败分支）
- **问题：** Redis 限流检查失败时 `count = 0` 放行（注释误称"为安全起见"），实际是 fail-open，限流失效时可无限刷验证码生成端点。
- **修复：** 改为 fail-close——限流基础设施不可用时拒绝请求（偏安全）：
  ```go
  count, err = redisCache.IncrementWithExpire(ctx, rateLimitKey, 1*time.Minute)
  if err != nil {
      applogger.Errorf("[Captcha] Rate limit check failed for IP %s: %v", clientIP, err)
      // fail-close：限流基础设施不可用时拒绝，避免被绕过
      return nil, fmt.Errorf("服务繁忙，请稍后再试")
  }
  ```
  修正原"为安全起见允许请求"的误导性注释。降级 `else` 分支（普通 Increment）同样处理。

**批次 4 验证：** `go build ./...`

---

## 批次 5 — 中等项（C2 时序、Q6 接口）

### C2 [P2 并发] `performCacheWarmUp` 启动延迟改就绪探测
- **位置：** `core.go:794-796`（`performCacheWarmUp` 开头 `time.Sleep(2 * time.Second)`）
- **问题：** 硬编码 2s sleep 等 DB 就绪，脆弱时序假设（慢机不够、快机白等）。
- **修复：** 改为 DB Ping 短轮询就绪探测，带总超时（如 10s）：
  ```go
  // 等待 DB 就绪（替代硬编码 sleep）
  readyCtx, readyCancel := context.WithTimeout(ctx, 10*time.Second)
  defer readyCancel()
  for {
      if c.GetDB() != nil {
          if sqlDB, err := c.GetDB().DB(); err == nil {
              if err := sqlDB.PingContext(readyCtx); err == nil { break }
          }
      }
      select {
      case <-readyCtx.Done():
          applogger.Warnf("缓存预热：等待 DB 就绪超时，放弃本轮预热")
          return
      case <-time.After(200 * time.Millisecond):
      }
  }
  ```

### Q6 [P2 质量] captcha 隐式接口断言改具名接口
- **位置：** `captcha.go:255`（`interface{ IncrementWithExpire(context.Context, string, time.Duration) (int64, error) }`）、`captcha.go:451`（`interface{ GetL2Cache() cache.Cache }`）
- **问题：** 鸭子类型断言，cache 实现改名即静默走 fallback，难察觉。
- **修复：** 在 `pkg/cache` 定义具名接口（如 `type RateLimitCache interface { IncrementWithExpire(ctx context.Context, key string, ttl time.Duration) (int64, error) }` 和 `type L2ExposingCache interface { GetL2Cache() Cache }`），让 `MultiLevelCache` 显式实现（编译期保证），captcha 处类型断言到具名接口。**注意：** 需先读 `pkg/cache` 确认 `IncrementWithExpire`/`GetL2Cache` 的确切签名与接收者，避免断言失败。

**批次 5 验证：** `go build ./...` + `go test ./internal/core/...`

---

## 批次 6 — P3 大重构（每项单独提交，逐项 go build + 人工 diff 审查）

> 这三项无单元测试覆盖，改完务必人工 `git diff` 逐行确认逻辑等价，并 `go test ./internal/core/...` 跑 `core_split_compat_test`。

### Q3 [P3] 拆分 `db/database.go`（1459 行 → 抽出 seed）
- **目标：** 把基础数据 seed 函数移到新文件 `db/init_data.go`，database.go 只留连接/AutoMigrate/约束管理。
- **要移动的函数：** `initData`、`createDefaultDept`、`createDefaultUser`、`createDefaultRole`、`createUserRoleRelations`、`createNetworkDeviceSystemParams`、`createNetworkDeviceScheduledJobs`、`createCaptchaBackgroundSystemParams`、`createOperationsManagementMenus`、`createRequestEncryptionToggleConfig`、`createADAuthConfig`、`NULL_STRING_PTR`（若仅 seed 用）。
- **注意：** 这些都是包内私有函数、共享 `import`，移动后两文件 import 各自补齐；`Database.InitData()` 仍调 `initData(d.DB)`，调用方不变。注释掉的 `createCaptchaBackgroundMenus` 大块可一并移走或删除。`go build` 验证无 `undefined`。

### Q4 [P3] 拆分 `core.go` `Init()`（~320 行 god function）
- **目标：** 把 Init 的 20 个步骤提取为私有方法，Init 只做编排：
  - `initDBAndData() error`（步骤 1-4：DB/迁移/基础数据/权限）
  - `initCacheAndWarmUp()`（步骤 5-6）
  - `initMetrics()`（步骤 7）
  - `initDeviceServices() error`（步骤 8-9.5：连接池/执行器/发现/采集/分区）
  - `initSchedulerAndTasks() error`（步骤 10-11：调度器 + 所有 RegisterXxxTasks）
  - `initCaptchaServices()`（步骤 13-14.1）
  - `initLogsAndAuth()`（步骤 15-16.5：OperLog/TokenBlacklist/AuthFactory）
  - `initRPAAndAPIAndReaper() error`（步骤 17-19）
- **约束：** 纯提取，**不改执行顺序、不改 fail-fast vs warn-continue 策略**（每处注释都解释了为何终止/为何只告警，保留）。`Init()` 变为顺序调用这些方法 + 保留步骤 20 的 RefreshView goroutine。`go build` + `core_split_compat_test` 必过。

### C1 [P3] `Close()` 真正可取消的优雅关闭
- **位置：** `core.go:542-608`
- **问题：** 30s timer goroutine 到点只让 `Close()` 返回，卡死的子调用并未被取消；`time.Sleep(100ms)` 赌异步写完成。
- **目标：** 给各子服务 `Stop()`/`Close()` 传带 deadline 的 `context.Context`；`time.Sleep(100ms)` 改 `sync.WaitGroup` 确定性等待。
- **风险：** 需改多个子服务方法签名（`DeviceInfoCollectionService.Stop()`、`Scheduler.Stop()`、`DeviceMonitorService.Close()`、`MetricsCacheService.Stop()` 等），牵连面大。
- **建议：** 若签名改动牵连过广，可分两步——(1) 先把 `Close()` 顶层建一个 `shutdownCtx, cancel := context.WithTimeout(context.Background(), coreShutdownTimeout)`，能传 ctx 的子服务先传（如已有 ctx 参数的）；(2) `time.Sleep` 改 WaitGroup。无法传 ctx 的子服务保留现状并在注释标明。逐子服务评估，不要强行统一签名引入回归。

---

## 全部完成后的收尾

```bash
go build ./...              # 全量编译
go test ./internal/core/... # 回归（含 core_split_compat_test、security/*_test）
go vet ./internal/core/...
```

## 回归测试参考

- `internal/core/core_split_compat_test.go` — Core 字段提升（Q4 重构必过）
- `internal/core/security/*_test.go` — 密码/JWT/认证（Q7 注释改动不应影响）
- `internal/core/db/migrations/*_test.go` — 迁移（B1/Q5/S2 不应影响）

## 提交粒度建议

- 批次 1-5（小修复）：可合并为 1-2 个 commit（如 `fix(core): security & quality fixes from review`）
- 批次 6 三项：各自独立 commit（`refactor(core): extract init_data from database.go` / `refactor(core): split Init() god function` / `refactor(core): cancellable graceful shutdown`）

---

## 执行进度（2026-08-13）

### 已完成：批次 1-5（9 项小修复）→ commit `1451fc3`

在分支 `refactor/core-review-fixes` 上，`go build ./...` + `go vet` 通过。改动文件：
`captcha.go / core.go / db/database.go / security/password.go / subprocess_reaper_linux.go / subprocess_reaper_windows.go / pkg/cache/cache.go`。

> Q6 纠正：计划原文说"让 MultiLevelCache 显式实现 RateLimitCache"不精确。`IncrementWithExpire` 实际只在 `*RedisCache`（redis.go:177），`GetL2Cache` 在 `*MultiLevelCache`（redis.go:733），二者不同类型。实际做法：在 `pkg/cache/cache.go` 定义具名接口 + 对**真正实现的类型**加 `var _ RateLimitCache = (*RedisCache)(nil)` / `var _ L2ExposingCache = (*MultiLevelCache)(nil)` 编译期断言（已落地，build 通过即证明签名匹配）。

### ⚠️ 预先存在的测试失败（与本次改动无关，已 git stash 在 clean main 复现确认）

1. **`TestCoreSplit_NewConstructorPopulatesInfraAndServices`**（internal/core）——`minimalTestConfig()`（core_split_compat_test.go:88）未设 `Security.SM4Key`，而 `New()`→`initSM4Cipher()` 现强制要求 SM4_KEY 非空。**修法（一行，但属 scope 外）**：给 minimalTestConfig 加 `Security: config.SecurityConfig{SM4Key: "dGVzdC1zZWNyZXQxNiEhIQ=="}`。
   - 注：字段提升相关的另两个测试 `TestCoreSplit_BackwardCompat` / `TestCoreSplit_FieldPromotionMatchesCoreInfra` **通过**——这才是 Q4 真正关心的回归。
2. **`internal/core/security` 4 个 Integration 测试**——`table sys_user has no column named ad_dn`。集成测试自建 SQLite schema 缺 `ad_dn`/`ad_ou_dn`/`ad_synced_at` 列（User 模型有这些字段）。属测试 schema 漂移，scope 外。

### ✅ 已完成：批次 6（Q3 / Q4 / C1）— 2026-08-13，3 commits on `refactor/core-review-fixes`

| 项 | Commit | 说明 |
|----|--------|------|
| Q3 | `7a113cf` | `database.go`（1456→521 行）拆出 `init_data.go`（+943），纯移动、逐块字节级一致 |
| Q4 | `56ab9ca` | `Init()` 拆为 8 个私有方法（+116/-15），纯提取、保持执行顺序/策略/注释 |
| C1 | `f2364a0` | `Close()` 加 `shutdownCtx` deadline（+46/-16），子服务签名未改、`Sleep` 保留并注释原因 |

**gate：** `go build ./...` + `go vet ./internal/core/...` 通过；`TestCoreSplit_BackwardCompat` / `TestCoreSplit_FieldPromotionMatchesCoreInfra` 必过；预先存在的 5 个失败（SM4_KEY / `ad_dn`）与基线一致、非本次引入。详见 `CORE-REVIEW-FIXES-SUMMARY.md`。

> 以下为执行前的就绪分析（保留作历史参考）：

### 待做：批次 6（Q3 / Q4 / C1）——移交新会话（已完成，见上）

新会话执行前的**已就绪分析**：

**Q3（拆 database.go，1455 行）——移动边界已摸清：**
- 移动到新文件 `db/init_data.go`：
  - **Block A = 486-768**：`initData` / `createDefaultDept` / `createDefaultUser` / `createDefaultRole` / `createUserRoleRelations`
  - **Block B = 808-1455(EOF)**：`createNetworkDeviceSystemParams` / `createNetworkDeviceScheduledJobs` / `createCaptchaBackgroundSystemParams` / `createCaptchaBackgroundMenus`(注释死代码，1059-1163 `/* */`) / `NULL_STRING_PTR`(1166) / `createOperationsManagementMenus` / `createRequestEncryptionToggleConfig` / `createADAuthConfig`
- 保留在 `database.go`：**1-485**（imports/struct/连接/约束/AutoMigrate/`auditConstraintNaming`/`InitData`）+ **769-807**（`dbIdentRe` var / `createDatabaseIfNotExists`）
- `NULL_STRING_PTR` 仅被移动块使用（createOperationsManagementMenus + 注释函数）→ **随移动** ✓
- `createCaptchaBackgroundMenus` 是 `/* */` 注释死代码 → 随移动（保持纯 move，不做逻辑删除）✓
- **import 拆分关键**：移动后 database.go 仍需几乎所有现有 import（AutoMigrate 用 models/operations/migrations，连接用 postgres/sqlite/sql/pq/gorm，校验用 regexp）；init_data.go 只需子集（fmt / models / operations / security / gorm / applogger，可能 time）。**建议**：先纯机械移动 → `go build` 看 "imported and not used" / "undefined" 报错精确修 import。若环境有 `goimports` 可直接 `-w` 自动修。
- `Database.InitData()`（480，保留）仍调 `initData(d.DB)`，调用方不变 ✓

**Q4 / C1**：按计划批次 6 原文执行，逐项 `go build` + 人工 `git diff` + `core_split_compat_test` 字段提升测试必过。
