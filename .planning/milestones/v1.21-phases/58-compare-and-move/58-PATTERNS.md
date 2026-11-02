# Phase 58: 排水器与 compare-and-move 推送 — Pattern Map

**Mapped:** 2026-10-17
**Files analyzed:** 19（新建 6 / 修改 13）
**Analogs found:** 19 / 19（全部有 analog，全部 analog 逐行读过）

---

## 0. 必读前置（研究中已确认的三条"别写错"）

| # | 事实 | 出处 |
|---|------|------|
| F-C | `NormalizeDN` **全仓不存在**，本 phase 是新建单一真实源，不是"复用既有实现" | `grep -rn "NormalizeDN" --include=*.go` 0 命中 |
| P11 | 排水器放 `internal/services/addomain`（不是 scheduler），才能复用未导出的 `syncUserAttributes` / `moveUserToNewOU` | `user_ad_sync_service.go:142` / `:107` |
| P7 | `models.ADPushQueueActivePredicate` 是红线常量，Phase 58 **一个字都不改** | `models/ad_push_queue.go:53` |

---

## 1. File Classification

| 文件 | 角色 | 数据流 | 最近 analog | 匹配度 |
|------|------|--------|-----------|--------|
| `internal/services/addomain/ad_push_queue_drain_service.go`（新建） | service / cron worker | 批量 DB 读 → LDAP 写 → 状态回写 | `user_ad_sync_service.go:44-139`（FailoverClient 闭包 + 每行处理） | 精确 |
| `internal/services/addomain/ad_config_resolver.go`（新建） | service 辅助（纯函数为主） | 纯函数 | `ad_push_queue_service.go:64-120`（`HasADFieldChanged` / `DedupeEntityIDs` 纯函数先例） | 精确 |
| `internal/services/addomain/dn_util.go`（新建） | utility（纯函数） | 纯函数 | `ldap_client.go:430` `extractRDNFromDN` + go-ldap `EscapeDN` | 精确 |
| `internal/services/addomain/ad_push_queue_drain_test.go`（新建） | test | sqlite + mock LDAP | `internal/services/addomain/user_ou_service_test.go:14-70` 建表 fixture | 精确 |
| `internal/core/db/migrations/migration_216_*.go`（新建） | migration | 双方言 seed | `migration_203_connection_pool_sysconfig.go` 全文件 | 精确 |
| `internal/scheduler/ad_sync_tasks.go`（改） | cron 接线 | 定时触发 | 同文件 `Start():143-176` | 精确（自 analog） |
| `internal/scheduler/dept_sync_tasks.go`（改） | cron 接线 | — | 同文件 `:47-72`（被删的副本本身即 analog） | 精确 |
| `internal/api/v1/system/ou_mapping_handler.go`（改） | handler | request-response | 同文件 `:117` `.First()` | 精确 |
| `internal/api/v1/system/user_handler.go`（改） | handler（删路径） | — | 同文件 `:194-260`（被删代码本身） | 精确 |
| `internal/api/v1/system/user_import_handler.go`（改） | handler（换路径） | 事务外批量入队 | 同文件 `:114-148` `triggerADSyncAfterImport` | 精确 |
| `internal/services/system/user_service.go`（改） | service（事务接线） | CRUD | 同文件 `BatchDelete:625-660` | 精确 |
| `internal/services/system/ad_push_queue_test.go`（改） | test | sqlite | 同文件 `:717-735` 回滚用例 | 精确 |
| `internal/services/system/user_service_test.go`（改） | test fixture | sqlite | 同文件 `:48-128` `setupUserServiceTestDB` | 精确 |
| `internal/services/addomain/ad_push_queue_service.go`（改） | service | — | `:182` / `:273` 两处 `time.Now()` | 精确 |
| `internal/services/addomain/user_ad_sync_service.go`（改） | service | LDAP 写 | 同文件 `:126-135` `moveUserToNewOU` 回写段 | 精确 |
| `internal/services/addomain/dept_ou_mapper.go`（改） | service | CRUD 读 | 同文件 `:50-62` `FindOUDNByDeptID` | 精确 |
| `internal/services/addomain/dept_sync_service.go`（改） | service | — | 同文件 `:115` 拼接点 | 精确 |
| `internal/models/ad_push_queue.go`（改） | model | — | 同文件 `:92` / `:104` | 精确 |
| `internal/core/db/database.go`（改） | config（迁移注册） | — | 同文件 `:1050-1093` 双分支 | 精确 |

---

## 2. Pattern Assignments

### 2.1 `internal/services/addomain/ad_push_queue_drain_service.go`（新建）

**Analog:** `internal/services/addomain/user_ad_sync_service.go`

#### Imports pattern（照抄 `ad_push_queue_service.go:19-28` + `user_ad_sync_service.go:3-17` 合并）

```go
import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/go-ldap/ldap/v3"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
	applogger "github.com/ruoyi-next/ruoyi-go-backend/pkg/logger"
	"gorm.io/gorm"
)
```

#### 结构体 + 构造（DI 惯例照抄 `user_ad_sync_service.go:21-40`）

```go
// ADPushDrainer 排水器（Phase 58）。零状态注入 db + 账号池，绝不自己 NewAccountPool（Pitfall 4）。
type ADPushDrainer struct {
	db   *gorm.DB
	pool AccountPool
	now  func() time.Time // 测试注入点，生产 time.Now
	mu   sync.Mutex       // cron 与手动触发共用的轮次互斥事实源
}

func NewADPushDrainer(db *gorm.DB, pool AccountPool) *ADPushDrainer {
	return &ADPushDrainer{db: db, pool: pool, now: func() time.Time { return time.Now().UTC() }}
}
```

#### 互斥（P-8：每轮 `now` 只取一次）

```go
func (d *ADPushDrainer) DrainOnce(ctx context.Context) error {
	if !d.mu.TryLock() { // Go ≥1.18 可用
		applogger.Debugf("[ad_push_queue] 上一轮排水未结束，本轮跳过")
		return nil
	}
	defer d.mu.Unlock()
	now := d.now() // 一轮一次，全轮透传（P8）
	...
}
```

#### FailoverClient 分组闭包（P1 + P12，形状照抄 `user_ad_sync_service.go:67-95` 与 `:264-289`）

```go
fc := NewFailoverClient(d.pool, cfg)
if err := fc.ExecuteWithFailover(ctx, func(c LDAPClientIface) error {
	for _, row := range group {
		d.drainRow(ctx, c, row, cfgID, now) // 收集行级错误，绝不上抛
	}
	return nil // ← 关键：返回 nil 避免换账号重跑整组 + MarkFailure 累加熔断
}); err != nil {
	// ErrAllAccountsUnavailable 之前就返回（failover_client.go:52-54），可重试不直接 dead
	if errors.Is(err, ErrAllAccountsUnavailable) { ... }
}
```

#### 行处理里复用既有方法（不要复制粘贴 `syncUserAttributes`）

```go
// 属性推送：复用未导出方法（user_ad_sync_service.go:142），DNExists 预检 + code 32 清 ad_dn 已封装
if err := svc.syncUserAttributes(ctx, c, &user, map[string]interface{}{
	"nickname": *user.Nickname, "email": *user.Email, "phone": *user.Phone, "deptId": *user.DeptID,
}); err != nil {
	if errors.Is(err, ErrADTargetNotFound) { /* → dead，不烧 attempts */ }
}
```

#### code 32 判定（P3/P6，语义与 `user_ad_sync_service.go:213` 完全一致）

```go
if ldap.IsErrorWithCode(err, ldap.LDAPResultNoSuchObject) { // errors.As 穿透 %w
	// 清 ad_dn / ad_ou_dn / ad_synced_at（同 user_ad_sync_service.go:196-203）
	return terminalDead(row, "unrecoverable:target-not-found")
}
```

#### 状态写回（照抄 `EnqueueDisable` 的 `Updates(map[string]interface{}{...})` 形状 `:297-303`）

```go
d.db.WithContext(ctx).Model(&models.ADPushQueue{}).Where("id = ?", row.ID).
	Updates(map[string]interface{}{
		"status": string(models.ADPushStatusFailed),
		"attempts": row.Attempts + 1,
		"next_retry_at": now.Add(backoffDuration(row.Attempts + 1)),
		"failed_at": now,               // F-12：统一 .UTC()
		"last_error": "retry:" + err.Error(),
		"updated_at": now,
	})
```

#### 拾取谓词（D-05 锁定，与 `idx_ad_push_queue_status_retry` 对齐）

```go
d.db.WithContext(ctx).
	Where(models.ADPushQueueActivePredicate). // 只读常量，不改写（models:53）
	Where("next_retry_at IS NULL OR next_retry_at <= ?", now).
	Order("created_at ASC").
	Limit(drainBatchLimit /* =100 */).
	Find(&rows)
```

#### 退避纯函数（纯函数先例照抄 `DedupeEntityIDs` `:106-120` 的注释+签名风格）

```go
// backoffDuration 退避曲线纯函数（60s→2m→4m→8m→16m，封顶 30m）。
// attempts 达 maxDrainAttempts(5) 转 dead —— 总重试窗 ~61min 桥接账号池 30min 熔断周期。
func backoffDuration(attempts int) time.Duration { ... }
```

---

### 2.2 `internal/services/addomain/ad_config_resolver.go`（新建，D-02 + D-03）

**Analog:** `internal/services/addomain/ad_push_queue_service.go:64-120`（纯函数注释风格 + `IsADSyncEnabled` DB 查询形状）

#### 最长后缀唯一匹配（P6：必须带逗号边界）

```go
// MatchADConfigByDN 按 ad_dn 后缀解析所属 AD 配置（D-02 契约）。
// 判定必须带逗号边界：strings.HasSuffix(normDN, ","+normBaseDN)
// —— 裸 HasSuffix 会让 "…,DC=xcorp,DC=com" 被 "DC=corp,DC=com" 误命中（P6）。
func MatchADConfigByDN(dn string, configs []models.ADConfig) (*models.ADConfig, error) {
	norm := NormalizeDN(dn)
	var best *models.ADConfig
	bestLen, ambiguous := -1, false
	for i := range configs {
		if configs[i].Status != models.ADConfigStatusEnabled { continue }
		base := NormalizeDN(configs[i].BaseDn)
		if !strings.HasSuffix(norm, ","+base) { continue }
		if len(base) > bestLen { best, bestLen = &configs[i], len(base); ambiguous = false } else if len(base) == bestLen {
			ambiguous = true
		}
	}
	switch {
	case ambiguous: return nil, fmt.Errorf("配置歧义: %s 匹配到多个等长 base_dn", dn)
	case best == nil:  return nil, fmt.Errorf("无匹配 AD 配置: %s", dn)
	}
	return best, nil
}
```

#### D-03 合并后的两个显式入口（F-D：语义不许混）

```go
// FirstEnabledADConfigID 只判 status=enabled，ORDER BY created_at ASC。
// 保留 ad_sync_tasks.go:306-330 / dept_sync_tasks.go:47-72 的现行行为（合并后若统一加上
// sync_enabled，"配置启用但同步关闭"时手动同步任务会从"能同步"变成"报未找到配置" —— 行为变更）。
func FirstEnabledADConfigID(ctx context.Context, db *gorm.DB) (string, error) { ... }

// FirstSyncEnabledADConfigID 判 sync_enabled=true AND status=enabled，供 ou_mapping_handler.go:117 与排水器。
func FirstSyncEnabledADConfigID(ctx context.Context, db *gorm.DB) (string, error) { ... }
```

**⚠ 迁移时的参数语义陷阱：** 两份本地函数先读 `params["configId"]/["adConfigId"]`，再回落到自动挑选。合并到 addomain 后这层参数提取保留在 scheduler 侧，只把"自动挑选"部分换成 helper（`ad_sync_tasks.go:308-313` 的 1./2. 两段照搬）。

---

### 2.3 `internal/services/addomain/dn_util.go`（新建，QUEUE-04 + HYGIENE-02）

**Analog:** `ldap_client.go:430-436` `extractRDNFromDN`（同包 DN 语义归属）+ go-ldap `EscapeDN`

```go
// NormalizeDN DN 归一化（仅用于比较，**不用于回写 AD**）。
// 纯字符串轻量归一，永不失败、永不 panic、可单测：
//   TrimSpace → 按 ',' split → 逐段 TrimSpace → 段内按首次 '=' 切分（attr 名 ToLower，值去首尾空格原样保留）
// 刻意不用 ldap.ParseDN：存量 sys_dept_ou_mapping.ou_dn 是 dept_sync_service.go:115 写的未转义值，
// ParseDN 遇未转义特殊字符会失败 → 只能退回字符串路径。注释必须写明"仅比较"。
func NormalizeDN(dn string) string { ... }

// BuildOUDN 部门名 → OU DN 拼接（HYGIENE-02，全仓唯一拼接点的唯一出口）。
// ⚠ ldap.EscapeDN **不转义 '='**：前导 '=' 的部门名是残留边界，必要时自行补 '\'（P4）。
func BuildOUDN(deptName, parentOUDN string) string {
	return fmt.Sprintf("OU=%s,%s", ldap.EscapeDN(deptName), parentOUDN)
}
```

**调用点改造：** `dept_sync_service.go:115`
```go
-	ouDN := fmt.Sprintf("OU=%s,%s", dept.DeptName, parentOUDN)
+	ouDN := BuildOUDN(dept.DeptName, parentOUDN)
```

---

### 2.4 `internal/services/addomain/ad_push_queue_drain_test.go`（新建，Wave 0）

**Analog A（建表 fixture）:** `internal/services/addomain/user_ou_service_test.go:14-70` — 包级 `const createXxxTable = \`CREATE TABLE ...\`` + `gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})`

```go
const (
	createDrainQueueTable = `CREATE TABLE ...` // 或直接调 migrations.Migrate215CreateADPushQueue
	createDrainUserTable  = `CREATE TABLE sys_user (
		id TEXT PRIMARY KEY, ..., ad_dn TEXT, ad_ou_dn TEXT, ad_synced_at DATETIME,
		deleted_at DATETIME)` // ⚠ 必须含 deleted_at（P2：排水器走 Unscoped 但 F-07 孤儿用例要构造软删行）
	createDrainADConfigTable = `CREATE TABLE sys_ad_config (
		id TEXT PRIMARY KEY, config_name TEXT,
		base_dn TEXT,                       // ⚠ F-F：必须含 base_dn + config_name，否则 D-02 测试只能绕开 GORM
		sync_enabled INTEGER NOT NULL DEFAULT 0, status INTEGER NOT NULL DEFAULT 0,
		created_at DATETIME, updated_at DATETIME, deleted_at DATETIME)`
	createDrainDeptOUMappingTable = `CREATE TABLE sys_dept_ou_mapping (...)` // 照抄 user_ou_service_test.go:14-27
	createDrainDeptTable = `CREATE TABLE sys_dept (...)`
)

func setupDrainTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, migrations.Migrate215CreateADPushQueue(db)) // 索引靠迁移，不手写
	...
}
```

**Analog B（mock LDAP）:** `ldap_client_mock_test.go:113-142` — `mockLDAPClient` 已实现全部所需方法且**错误可注入**（`moveUserErr` / `disableUserErr` / `dnExistsRes` / `dnExistsErr` / `updateUserAttrErr`），code 32 用例直接造 `&ldap.Error{ResultCode: ldap.LDAPResultNoSuchObject}`。

⚠ `mockLDAPClient` 无状态字段可"按用例分次返回不同结果"，多轮场景需扩字段或新写可配置 fake。

**Analog C（mock AccountPool）:** `account_pool_test.go` / `account_pool_78_06_test.go` — `ListAvailable` 返回空切片即驱动 `ErrAllAccountsUnavailable`（`failover_client.go:52-54`）。

---

### 2.5 `internal/core/db/migrations/migration_216_adpush_drain_sysconfig.go`（新建）

**Analog（整文件照抄）:** `internal/core/db/migrations/migration_203_connection_pool_sysconfig.go`

```go
package migrations

// Migrate216ADPushDrainSysConfig 排水间隔移入 sys_config（Phase 58 D-05 单旋钮）。
// 启动时读取、修改后需重启后端生效（同 network.connection_pool.* 先例）。
// 非阻断：缺旋钮只回落默认 30s，不影响业务（对照 migration 215 建表的"失败阻断启动"档，seed 不该阻断）。
func Migrate216ADPushDrainSysConfig(db *gorm.DB) error {
	log.Println("Running migration 216: ad_push_queue drain interval sys_config seed")

	const (
		seedKey   = "ad_push_queue.drain_interval_seconds"
		seedValue = "30"
	)
	var existingCount int64
	if err := db.Model(&models.Config{}).Where("config_key = ?", seedKey).Count(&existingCount).Error; err != nil {
		return err
	}
	if existingCount > 0 { // 幂等
		return nil
	}
	cfg := &models.Config{
		ConfigName:  "AD推送队列-排水间隔(秒)",
		ConfigKey:   seedKey,
		ConfigValue: seedValue,
		ConfigType:  models.ConfigTypeNo,     // N = 非系统参数，前端"参数设置"页可编辑
		IsSystem:    models.ConfigIsSystemNo,
		Remark:      "ad_push_queue 排水器轮询间隔(秒);建议 30,放宽上限 300;修改后需重启后端生效",
	}
	if err := db.Create(cfg).Error; err != nil { /* log + continue，不阻断 */ }
	applogger.Infof("[迁移] sys_config seed: %s = %s", seedKey, seedValue)
	return nil
}
```

**双分支注册（P13）:** `internal/core/db/database.go`
- PG 分支追加在 `:1058` `ensureADPushQueue()` 之后（非阻断风格，与 `:1050` Migrate214 同款）：
  ```go
  if err := migrations.Migrate216ADPushDrainSysConfig(d.DB); err != nil {
      applogger.Errorf("ad_push_queue 排水间隔 sys_config seed 失败 (非阻断,留待下次启动): %v", err)
  }
  ```
- sqlite 分支追加在 `:1091-1093` 之后（同款非阻断）。

---

### 2.6 `internal/scheduler/ad_sync_tasks.go`（改）

#### (a) `Start()` 内追加第三个 entry（P9：走 `AddJob`，`AddFunc` 不暴露 chain wrapper）

Analog 位置：同文件 `Start():143-176`

```go
// Phase 58 D-05: 排水器（禁注册 sys_job —— 里程碑锁定，不写 sys_job seed / 不加 RegisterTask）
interval := addomain.LoadDrainIntervalSeconds(s.db) // 启动读一次
spec, ok := addomain.SecondsToCronSpec(interval)
if !ok {
	applogger.Warnf("[ad_push_queue] drain_interval_seconds=%d 无法映射为 cron spec，回落 30s", interval)
}
_, err = s.cron.AddJob(spec,
	cron.NewChain(cron.Recover(lg), cron.SkipIfStillRunning(lg)).Then(cron.FuncJob(func() {
		addomain.NewADPushDrainer(s.db, s.getPool()).DrainOnce(context.Background())
	})))
```

⚠ `s.getPool()` 必须复用（`ad_sync_tasks.go:407-415`），**禁止** `addomain.NewAccountPool`（Pitfall 4）。
⚠ `SkipIfStillRunning` 的 channel 是闭包私有的，**不能**当互斥事实源；事实源是 Drainer 内的 `sync.Mutex`。

#### (b) 删除 `getDefaultADConfigID`（`:304-331`）

调用点（`:84` 附近 `executeADDataSyncTask` 等）改为：

```go
-	adConfigID, err := getDefaultADConfigID(ctx, db, params)
+	adConfigID, err := getDefaultADConfigIDFromParamsOrFirst(ctx, db, params) // 参数提取保留在本文件
```
其中 `FromParamsOrFirst` 保留 `:308-313` 的 `configId` / `adConfigId` 两段，尾部换成：
```go
	// 自动挑选：语义并入 addomain（只判 status，保留现行行为 —— F-D）
	return addomain.FirstEnabledADConfigID(ctx, db)
```

---

### 2.7 `internal/scheduler/dept_sync_tasks.go`（改）

删除 `getDefaultADConfigIDForDept`（`:45-72`），两个调用点（`:84`、`:138`）按 §2.6(b) 同款改写。

**保留** `:22-28` `getGlobalADAccountPool()` 与其"**禁止**在本文件内临时创建账号池"注释 —— 排水器注释里要照抄同款约束（`dept_sync_tasks.go:16-21`）。

---

### 2.8 `internal/api/v1/system/ou_mapping_handler.go`（改）

Analog：同文件 `:115-121`

```go
-	var adConfig models.ADConfig
-	if err := h.db.WithContext(c.Request.Context()).Where("sync_enabled = ? AND status = ?", true, models.ADConfigStatusEnabled).First(&adConfig).Error; err != nil {
-		applogger.Errorf("查询 AD 配置失败: %v", err)
-		response.Error(c, http.StatusInternalServerError, "查询 AD 配置失败")
-		return
-	}
+	// Phase 58 D-03（HYGIENE-01）：收敛到 addomain helper。
+	// 这里**必须**用 FirstSyncEnabledADConfigID（原语义判 sync_enabled + status），
+	// 不是 FirstEnabledADConfigID —— 它是"当前操作的 AD 配置"，无 dn 可依据，
+	// 收敛只消除重复实现，不改语义。
+	adConfigID, err := addomain.FirstSyncEnabledADConfigID(c.Request.Context(), h.db)
+	if err != nil { applogger.Errorf(...); response.Error(c, http.StatusInternalServerError, "查询 AD 配置失败"); return }
```
下方 `mapping.ADConfigID = adConfig.ID` 改为 `= adConfigID`。

---

### 2.9 `internal/api/v1/system/user_handler.go`（改，删直推）

删除四块（源码即 analog）：
1. `:28-32` `adSyncInFlight sync.Map` 字段 + 注释；
2. `:189-238` Update 内 `if h.userADSyncService != nil { ... go func(){...} }` 整块 goroutine 直推；
3. `:244-260` `buildADSyncMap`；
4. 随之为 nil 的 import：`errors` / `time` / `addomain`（编译后 `go build ./...` 会指出，逐个删）。

**保留：** `:28` `userADSyncService *addomain.UserADSyncService` 字段 —— `SyncManagersToAD`（`user_import_handler.go:230`）与 `user_router.go:35-38` 注入仍需。

**保留：** `user_handler_test.go:29-46` `TestUpdateUserWithADSync`（空壳 `assert.True(true)`，无编译依赖）。

**F-G 必须同 wave 处置：** `internal/api/v1/system/user_handler_test.go:568-582` `TestUserHandler_BuildADSyncMap` 直接调 `h.buildADSyncMap(req)` → 删该用例或改写为"函数已下线"记录性用例，否则 `go test ./...` 编译红（P10）。

---

### 2.10 `internal/api/v1/system/user_import_handler.go`（改，D-01）

删除 `:104-148` `triggerADSyncAfterImport`（含 goroutine + recover + `Pluck("id")`）。
把 `:77-81` 换成**同步**批量入队（`Pluck` 逻辑平移）：

```go
	// Phase 58 D-01: 导入后改批量入队 ad_push_queue（删直推，SC6）。
	// ⚠ 事务边界：ExcelService 的 Transaction 已提交（excel_service.go:427），此处入队是
	// outbox-lite "同生共死"不变量的**一处已接受例外**（crash 窗口丢推送）—— 注释必须写明。
	if len(result.AffectedKeys) > 0 {
		if err := h.enqueueImportUsersToAD(result.AffectedKeys); err != nil {
			applogger.Errorf("[AD-SYNC] 导入后批量入队失败（不影响导入结果）: %v", err)
		}
	}
```

```go
func (h *UserHandler) enqueueImportUsersToAD(usernames []string) error {
	return h.core.GetDB().Transaction(func(tx *gorm.DB) error {
		var userIDs []string
		if err := tx.Model(&models.User{}).
			Where("username IN ?", usernames).
			Where("ad_dn IS NOT NULL AND ad_dn <> ''"). // 原 Pluck 预过滤平移（:129）
			Pluck("id", &userIDs).Error; err != nil {
			return err
		}
		return addomain.NewADPushEnqueuer().EnqueueUpdateBatch(tx, userIDs)
	})
}
```
**operlog 不动**（`:84-86`），入队是后台动作，operlog 仍记 `OperTypeImport`。

---

### 2.11 `internal/services/addomain/ad_push_queue_service.go`（改，新增批量变体 + F-12）

#### 新增 `EnqueueUpdateBatch`（模板照抄 `EnqueueDisable:263-332`）

```go
// EnqueueUpdateBatch 批量入队 update_user（D-01 导入路径专用）。
//
// 与 EnqueueUpdateIfChanged 的**语义差异**（必须在注释写明）：本变体**不做 diff**。
// 导入场景无导入前旧值快照（BatchSyncUsersToAD:679-683 同样承认），
// "无 diff 就全量入队"是正确语义，排水器闭包重读 DB 后推最新值。
//
// 行为：空切片 no-op → DedupeEntityIDs → IsADSyncEnabled 前置门控（WARN 放行）→
//       CreateInBatches(rows, pushQueueBatchSize=500) + adPushOnConflictClause(now)。
func (e *ADPushEnqueuer) EnqueueUpdateBatch(tx *gorm.DB, userIDs []string) error {
	if len(userIDs) == 0 { return nil }
	deduped := DedupeEntityIDs(userIDs)
	if len(deduped) == 0 { return nil }
	enabled, err := IsADSyncEnabled(tx)
	if err != nil { return err }
	if !enabled {
		applogger.Warnf("[ad_push_queue] 无启用的 AD 配置，跳过批量 update 入队: %d 个用户", len(deduped))
		return nil
	}
	now := time.Now().UTC() // F-12
	rows := make([]models.ADPushQueue, 0, len(deduped))
	for _, id := range deduped {
		rows = append(rows, models.ADPushQueue{
			EntityType: models.ADPushEntityUser, EntityID: id,
			ActionType: models.ADPushActionUpdateUser, Status: models.ADPushStatusPending,
		})
	}
	return tx.Clauses(adPushOnConflictClause(now)).CreateInBatches(&rows, pushQueueBatchSize).Error
}
```
> ⚠ **不做 F-01 压制**：update_user 之间不存在压制关系（压制只发生在 disable 打 update_user 上，见 `:275-306`）。

#### F-12 UTC 两行改
`:182` `tx.Clauses(adPushOnConflictClause(time.Now()))` → `time.Now().UTC()`
`:273` `now := time.Now()` → `now := time.Now().UTC()`

---

### 2.12 `internal/services/addomain/user_ad_sync_service.go`（改，F-A 阻断级）

`moveUserToNewOU:124-135` 与 `moveSingleUserToOU:315-325` 两处同款缺陷。**排水器自身的 compare-and-move 必须回写 `ad_dn`**，同时把这两处一并修：

```go
 	// 3. 移动用户到新OU
 	if err := ldapClient.MoveUser(*user.AdDn, ouDN); err != nil {
 		return fmt.Errorf("移动用户到新OU失败: %w", err)
 	}
-	// 4. 更新用户的ad_ou_dn
-	if err := s.db.WithContext(ctx).Model(&models.User{}).Where("id = ?", userID).Update("ad_ou_dn", ouDN).Error; err != nil {
-		applogger.Warnf("更新用户ad_ou_dn失败: %v", err)
-	}
+	// 4. F-A：ModifyDN 后 AD 侧 DN 已变，必须同时回写 ad_dn（= RDN + "," + ouDN）与 ad_ou_dn。
+	//    否则 D-02 下一轮按陈旧 ad_dn 解析配置 → 误判"配置无匹配" → 误转 dead；
+	//    属性 Modify 也会打到旧 DN → code 32 → 又一次误判 dead。
+	newAdDn := fmt.Sprintf("%s,%s", extractRDNFromDN(*user.AdDn), ouDN)
+	if err := s.db.WithContext(ctx).Model(&models.User{}).Where("id = ?", userID).
+		Updates(map[string]interface{}{"ad_dn": newAdDn, "ad_ou_dn": ouDN}).Error; err != nil {
+		applogger.Warnf("更新用户 ad_dn/ad_ou_dn 失败: %v", err)
+	}
```
`extractRDNFromDN` 是同包未导出函数（`ldap_client.go:430`），同包直接调。

**比较幂等点（P3）：** 排水器移动前比较 `NormalizeDN(*user.AdOuDn) == NormalizeDN(targetOUDN)` → 相同则跳过 `MoveUser`（重复调用 `MoveUser` 同一 DN 第二次会 code 32，`ldap_client.go:350-361`）。

**Q1 宽容语义照抄现成先例：** `SyncUserUpdateToAD:77-81`（OU 移动失败仅 ERROR 日志、继续推属性）—— 排水器保持同款，但把 OU 失败写进 `last_error` 并按可重试处理。

---

### 2.13 `internal/services/addomain/dept_ou_mapper.go`（改，F-B）

新增（**保留旧方法不动**，其它调用方 `:105` / `:244` 及 `user_ou_service` 系列不受影响）：

```go
// FindOUDNByDeptIDForConfig 配置作用域的 OU 查询（F-B）。
// 旧方法 FindOUDNByDeptID 只按 dept_id + sync_enabled 过滤、无 ad_config_id、无 ORDER BY，
// 多配置下 First() 顺序不确定 → 排水器会把 A 域用户移进 B 域的 OU。
// 根因：sys_dept_ou_mapping 唯一键是 (dept_id, ad_config_id)，一个 dept 在 N 个配置下有 N 行合法记录。
func (m *DeptOUmapper) FindOUDNByDeptIDForConfig(ctx context.Context, deptID, adConfigID string) (string, error) {
	var mapping models.DeptOUMapping
	err := m.db.WithContext(ctx).
		Where("dept_id = ? AND ad_config_id = ? AND sync_enabled = ?", deptID, adConfigID, true).
		First(&mapping).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return "", fmt.Errorf("未找到部门 %s 在 AD 配置 %s 下的 OU DN", deptID, adConfigID)
		}
		return "", fmt.Errorf("查询部门映射失败: %w", err)
	}
	return mapping.OUDN, nil
}
```
> 错误文案形状照抄原 `:56-59`（区分 NotFound 与真错误两段）。

---

### 2.14 `internal/services/system/user_service.go`（改，F-07 入队侧 Pluck）

`BatchDelete:631-653` 事务内，**软删之后、入队之前**插一步存在性过滤：

```go
		// Phase 58 F-07: 入队前 Pluck 存在性 —— 只对真实存在的行入队，
		// 堵住"重复/已删 id 无限刷孤儿 disable 行"（排水器侧另有 not-found → dead 兜底）。
		var existing []string
		if err := tx.Model(&models.User{}).Where("id IN ?", ids).Pluck("id", &existing).Error; err != nil {
			return apperrors.DatabaseError(err)
		}
		if err := addomain.NewADPushEnqueuer().EnqueueDisable(tx, existing); err != nil {
			return apperrors.DatabaseError(err)
		}
```
`Delete`（单条）是否对称由 planner 定（CONTEXT 只锁 BatchDelete）。

**回归面（必跑）：** `go test ./internal/services/system/ -run TestPhase57 -count=1` —— 42 个既有用例含 `TestPhase57_UserService_BatchDelete_EnqueuesDisable`（`ad_push_queue_test.go:918-936`）。

---

### 2.15 `internal/models/ad_push_queue.go`（改，F-12 + F-14）

```go
-	Status     ADPushStatus     `gorm:"size:20;not null" json:"status"`
+	Status     ADPushStatus     `gorm:"size:20;not null;default:'pending'" json:"status"` // F-14：与 migration_215 DDL 的 DEFAULT 'pending' 对齐
```
`:92` `now := time.Now()` → `time.Now().UTC()`；`:104` `q.UpdatedAt = time.Now()` → `time.Now().UTC()`。

**回归：** `TestPhase57_ADPushQueue_ModelRoundTrip`（`ad_push_queue_test.go:114-140`）—— 注意该用例不依赖 GORM 写 status 默认值，BeforeCreate 兜底仍有效。

---

### 2.16 `internal/services/system/ad_push_queue_test.go` + `user_service_test.go`（改，Wave 0）

#### F-F fixture 补列（`user_service_test.go:117-126`）
```sql
CREATE TABLE IF NOT EXISTS sys_ad_config (
    id TEXT PRIMARY KEY,
    config_name TEXT,   -- 新增
    base_dn TEXT,       -- 新增（D-02 后缀匹配测试必需）
    sync_enabled INTEGER NOT NULL DEFAULT 0,
    status INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME, updated_at DATETIME, deleted_at DATETIME)
```

#### F-10 回滚用例（复制 `TestPhase57_UserService_Update_Disable_RollsBackWhenQueueInsertFails:717-735` 的 DROP 表手法）
```go
require.NoError(t, db.Exec("DROP TABLE ad_push_queue").Error)
err := svc.Delete(ctx, userID)
require.Error(t, err)
var deletedAt *string
require.NoError(t, db.Raw("SELECT deleted_at FROM sys_user WHERE id = ?", userID).Scan(&deletedAt).Error)
assert.Nil(t, deletedAt, "事务必须回滚：deleted_at 不得被写入")
var roleRows int64
require.NoError(t, db.Table("sys_user_role").Where("user_id = ?", userID).Count(&roleRows).Error)
assert.Equal(t, int64(1), roleRows, "sys_user_role 行数不得变化")
require.NoError(t, migrations.Migrate215CreateADPushQueue(db)) // 重建保不影响同包其它用例
```

#### F-16（`TestPhase57_Migration215_BareInsertWithoutID:97-111`）加 `t.Skip` + 保留 PG 契约说明注释。

#### F-15（`user_cache_impl_test.go:154-165`、`user_handler_test.go:157-172`）补"此处故意不 seed 配置行"注释。

---

## 3. Shared Patterns（跨文件统一，planner 直接引用）

### S1. FailoverClient 闭包不变量（P1，最易踩）
- **来源:** `failover_client.go:75-91`
- **适用:** 排水器分组闭包
- 闭包内收集行级错误、**恒返回 nil**；否则一次 Modify 失败 → `MarkFailure("operation:...")` + 换账号重跑整组 → 3 次即熔断 30min（记忆 `ad-modify-fail-double-counts-breaker`）。

### S2. 账号池单例（Pitfall 4）
- **来源:** `ad_sync_tasks.go:404-415` + `dept_sync_tasks.go:16-28`
- **适用:** 排水器构造函数
- 一律 `s.getPool()` / `getGlobalADAccountPool()`；新代码注释必须写"禁止 NewAccountPool"。

### S3. 队列状态写回统一形状
- **来源:** `ad_push_queue_service.go:297-303`（`Updates(map[string]interface{}{...})`）
- **适用:** drainer 全部状态跃迁
- 一律 `Model(&models.ADPushQueue{}).Where("id = ?", id).Updates(map[...])`；时间戳统一 `.UTC()`（F-12）；`last_error` 用可 grep 前缀（`unrecoverable:` / `retry:` / `dead:` / `superseded by disable`）。

### S4. 终态退出活跃谓词（P5）
- `ADPushStatusPushed` / `ADPushStatusDead` 都**不在** `models.ADPushQueueActivePredicate` 内 → 天然退出唯一索引。排水器新增任何中间态（如 `processing`）时**绝不能**加进该常量。

### S5. 纯函数先例
- **来源:** `HasADFieldChanged:64` / `DedupeEntityIDs:106` / `backoffDuration`
- 纯函数放同文件或独立 `dn_util.go`，注释写清"无 DB、无副作用、可单测"；单测直接调函数（这才是**能红**的断言；sqlite 行为断言常为假绿，见 `ad_push_queue_test.go:358-361` 注释）。

### S6. sqlite fixture 约定
- **来源:** `user_service_test.go:48-128`、`ad_push_queue_test.go:28-33`
- `gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})`；建表用裸 `db.Exec(CREATE TABLE ...)`；索引靠直调 `migrations.Migrate215CreateADPushQueue(db)`；时间列写 `DATETIME`（**不写 timestamptz**，否则扫不回 `time.Time`，见 `migration_215:23-25`）。

### S7. 数据库注册点
- **来源:** `database.go:1050-1093`
- PG 分支在 advisory-lock 块内；sqlite 分支在 `else` 内；`sys_config` seed 类迁移 **非阻断**（只打日志）。

### S8. operlog 约定
- 排水器是后台任务 → **不适用** `operlog.Record`。
- 若新增手动触发端点：按 CLAUDE.md，`response.Success` 前 `operlog.Record(..., operlog.OperTypeOther)`。
- 本 wave 唯一被改的 handler 端点（ImportUser）operlog 保持不变。

---

## 4. No Analog Found

| 项 | 原因 | 替代 |
|----|------|------|
| 手动触发排水 HTTP 端点 | 仓库无同类"排障型手动触发"端点 | Q7 倾向不加；若加，宿主先例 `ad_domain_router.go:117` `users/batch-sync`，并按 S8 记 operlog |
| `SecondsToCronSpec` 放置位置 | 无先例 | 建议放 `addomain`（与 `LoadDrainIntervalSeconds` 同文件），测试随之放 `internal/services/addomain/`；RESEARCH §8.3 写的是 `./internal/scheduler/`，planner 二选一并保持函数与测试同包 |

---

## 5. Planner 需在 plan 中显式锁定的禁令（抄自 RESEARCH §6/§7，防回归）

| # | 禁令 |
|---|------|
| P1 | 闭包恒返回 nil |
| P2 | 读 sys_user 一律 `Unscoped()` |
| P3 | MoveUser 后必须回写 `ad_dn` + `ad_ou_dn` |
| P4 | 禁手写 DN 转义表，用 `ldap.EscapeDN` |
| P5 | 新状态绝不入 `ADPushQueueActivePredicate` |
| P6 | 禁裸 `HasSuffix`，必须带逗号边界 |
| P7 | 禁改 `ADPushQueueActivePredicate` 文本 |
| P8 | 每轮 `now := time.Now().UTC()` 一次 |
| P9 | `drain_interval_seconds` 必须经 `SecondsToCronSpec` 校验，非法值回落 + WARN |
| P10 | 删 `buildADSyncMap` 必须同 wave 处置 `user_handler_test.go:568-582` |
| P11 | 排水器放 addomain 包 |
| P12 | 按配置分组，每组一次闭包（禁每行一次 `ExecuteWithFailover`） |
| P13 | migration_216 PG + sqlite 双分支各注册一次 |
| P14 | 不注册 sys_job（可用 `grep -rn "ad_push_queue" internal/core/db/migrations/*.go \| grep -i sys_job` 证伪） |
| F-11 | **禁裸 `attempts` 判 dead**（合并子句每次编辑清零 attempts，`:161-169`） |

---

## 6. Metadata

**Analog search scope:** `internal/services/addomain/`、`internal/scheduler/`、`internal/models/`、`internal/core/db/`、`internal/core/db/migrations/`、`internal/api/v1/system/`、`internal/services/system/`
**Files scanned:** 21（逐一 Read，含 grep 定位）
**Extraction date:** 2026-10-17
**未读但已定位行号的引用:** `dept_sync_service.go:115`、`account_pool.go:29-45`、`ldap_client.go:430-436` — 摘要已从 RESEARCH 摘录，planner 实施前应实读确认。
