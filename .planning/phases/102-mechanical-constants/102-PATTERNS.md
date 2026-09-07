# Phase 102: 机械常量化（缓存键 / 状态 / 分页） - Pattern Map

**Mapped:** 2026-09-07
**Files analyzed:** 26（修改 21 + 新建 3 + 删除 2）
**Analogs found:** 26 / 26（全部位点有仓内一等先例；其中「status 使用点 AST 扫描」为能力升级型先例——骨架可复制但 6 种 AST 形态需新写匹配逻辑）

**规约:** 本文件行号为 2026-09-07 实读快照（已逐处核实）；执行时按 RESEARCH.md Pitfall 2 先 `grep -n` 复核。所有代码摘录均为可直接复制的仓内真实代码。

---

## File Classification

| 文件 | 操作 | Role | Data Flow | 模拟文件 | Match Quality |
|------|------|------|-----------|----------|---------------|
| `pkg/constants/cache.go` | 修改 | config（常量注册表） | — | 自身既有 3 常量块（:9-18） | exact（就地扩展） |
| `internal/services/system/cache_keys.go` | 修改 | config（常量注册表） | — | 自身既有形态（:79-129 / :296-368） | exact（就地扩展） |
| `internal/core/captcha.go` | 修改 | service | request-response + cache-I/O | 自身 :385/:450/:491 先例 | exact |
| `internal/core/captcha_background.go` | 修改 | service | cache-I/O（池） | captcha.go 先例 + 自身池键构造 | exact |
| `internal/services/system/notice_cache_impl.go` | 修改 | service decorator | cache read-through | duty/network cache_impl 同型 | exact |
| `internal/services/system/settings_cache_impl.go` | 修改 | service decorator | cache read-through | 同上 | exact |
| `internal/services/system/widget_data_fetcher.go` | 修改 | service/utility | cache-I/O | 同上（注意 %x 动词） | exact |
| `internal/services/duty/duty_cache_impl.go` | 修改 | service decorator | cache read-through | 同上 | exact |
| `internal/services/workorder/workorder_cache_impl.go` | 修改 | service decorator | cache read-through | 同上 | exact |
| `internal/services/knowledge/knowledge_cache_impl.go` | 修改 | service decorator | cache read-through | 同上 | exact |
| `internal/services/network/cache_impl.go` | 修改 | service decorator | cache read-through | 同上 | exact |
| `internal/services/api_endpoint_service.go` | 修改 | service（根包） | cache read-through | mac_history_query_service.go（已 import constants） | role-match |
| `internal/services/mac_history_query_service.go` | 修改 | service（根包） | cache-I/O | 同上 | exact（只注册） |
| `internal/services/rpa/selector_learner.go` | 修改 | service | cache-I/O | 同上（只注册 :361-363 getCacheKey） | exact |
| `internal/scheduler/cron.go` | 修改 | service（调度引擎） | event-driven/batch | models 常量族 + workorder_tasks 同构位 | exact |
| `internal/scheduler/vdi_sync_tasks.go` | 修改 | scheduler task | batch | workorder_tasks.go 复合字面量先例 | exact |
| `internal/scheduler/workorder_tasks.go` | 修改 | scheduler task | batch | 同上（自身 :327/:450 已用常量） | exact |
| `internal/scheduler/reconciliation_tasks.go` | 修改 | scheduler task | batch | 同上 | exact |
| `internal/scheduler/mac_history_tasks.go` | 修改 | scheduler task | batch | 同上 | exact |
| `internal/scheduler/mac_history_matview_tasks.go` | 修改 | scheduler task | batch | 同上 | exact |
| `internal/services/scheduler/job_service.go` | 修改 | service | CRUD | 自身 :324 `models.JobStatus(status)` 转换先例 | exact |
| `internal/services/workorder/base.go` | 修改 | service | query（GORM IN） | workorder.go 常量族引用 | exact |
| `internal/api/v1/system/file_handler.go` | 修改 | controller | request-response | pkg/query/pagination.go API + Phase 99 先例 | role-match |
| `pkg/constants/cache_102_test.go` | **新建** | test | — | `pkg/constants/pagination_test.go`（Phase 89 AST 锁先例，见 cache_invariants_92_test.go:6 引用） | role-match |
| `internal/services/system/cache_keys_102_test.go` | **新建** | test | — | `cache_invariants_92_test.go`（同包同目录） | exact |
| `internal/models/status_constants_test.go` | 修改（扩展） | test | — | 自身（D-102-6 明文同文件扩展） | exact |
| `internal/utils/pagination.go` | **删除** | utility | — | —（Phase 99 D-03-3 零调用方即删除先例） | — |
| `internal/utils/utils_74_12_test.go` | 修改（删测试） | test | — | —（随文件删除项） | — |

> 落点二分（D-102-1 唯一字面修订，RESEARCH §关键约束）：8 模块键 → cache_keys.go；captcha 2 文件 + 根包 2 文件 → pkg/constants/cache.go（import 环实证）。

---

## Pattern Assignments

### A. 缓存键注册形态（写侧）

#### A1. 前缀常量 + GetXxxKey helper —— cache_keys.go 8 模块键（D-102-1/D-102-3）

**模拟文件（注册宿主）:** `internal/services/system/cache_keys.go`

**既有常量块形态**（:79-129，新模块常量按同风格追加独立 `const (...)` 块）:
```go
// 系统模块缓存键
const (
	// 用户相关
	CacheKeyUserByID       = "user:id"   // 用户详情: user:id:{uuid}
	CacheKeyUserByUserName = "user:name" // 用户名查询: user:name:{username}
	...
	// 配置相关
	CacheKeyConfigAll   = "config:all"  // 配置全量列表: config:all
)
```

**参数化键 helper 形态**（:296-301，D-102-3 点名的 `GetDictDataByTypeKey` 模式）:
```go
// GetDictDataByTypeKey 根据字典类型构建字典数据缓存键
// 参数：dictType 字典类型
// 返回：dict:data:{dictType}
func GetDictDataByTypeKey(dictType string) string {
	return CacheKeyDictData + ":" + dictType
}
```
> 新键含多参数时（如 `notice:my_notices:%s:page:%d:size:%d`），helper 体内可用 `fmt.Sprintf`（文件 :4 已 import fmt）；返回注释必须写明完整键样例（既有 helper 均带「返回：xxx:{param}」注释行）。

**失效 pattern 前缀派生**（D-102-4）。既有 `BuildPattern` **不可直接用**——`:34-41` 的 `*` 直接追加无 `:`：
```go
// BuildPattern 构建缓存键模式（用于模糊匹配）
// 例如：BuildPattern("user", "*") -> "xingran:user:*"
func (m *CacheKeyManager) BuildPattern(parts ...string) string {
	result := m.Build(parts...)
	// 确保模式以 * 结尾
	if !strings.HasSuffix(result, "*") {
		result += "*"
	}
	return result
}
```
`BuildPattern("my_notices", uid)` 产出 `notice:my_notices:{uid}*` ≠ 现值 `notice:my_notices:{uid}:*`。**安全做法 = 前缀常量 + Sprintf `":*"`**，先例 :282-292:
```go
func BuildModulePattern(module string) string {
	return fmt.Sprintf("%s:%s:*", CachePrefixPrefix, module)
}
func BuildInvalidatePattern(module string, keyType string) string {
	if keyType == "" {
		return BuildModulePattern(module)
	}
	return fmt.Sprintf("%s:%s:%s:*", CachePrefixPrefix, module, keyType)
}
```
全模块 pattern（`notice:*` / `duty:*` / `workorder:*` / `kb:category:*`）同理：`fmt.Sprintf("%s:*", CacheKeyNotice)` 形态的前缀派生 helper，**不新增独立 pattern 常量**。

**既有防冲突工具**（:373-375，键值不变则无需触碰）:
```go
func EscapeCacheKeyValue(value string) string {
	return strings.ReplaceAll(value, ":", "%3A")
}
```

#### A2. Sprintf 格式常量 —— pkg/constants 侧 captcha 6 格式 + 根包 2 格式（D-102-2/D-102-3）

**模拟文件（注册宿主）:** `pkg/constants/cache.go`（全文 19 行，形态即全部）:
```go
// Redis 键格式(仅保留生产代码实际使用的格式)。
//
// 调用方应使用这些常量而非内联字面量,避免 key 拼写分叉。
const (
	// TokenBlacklistKeyFormat Token 黑名单键格式
	TokenBlacklistKeyFormat = "token:blacklist:%s"

	// LoginLockKeyFormat 登录锁定键格式
	LoginLockKeyFormat = "login:lock:%s"

	// CaptchaVerifiedKeyFormat 验证码验证键格式
	CaptchaVerifiedKeyFormat = "captcha:verified:%s"
)
```
新增 8 个常量（6 captcha + `user_endpoints:%s` + `mac:vendor:%s`）直接续写该 const 块，每常量保留单行注释。captcha_background 池键族切分（discretion 推荐形态，与现构造一一对应）:
```go
CaptchaCachePoolPrefixFormat = "captcha:cache:pool:%s:%d"
// 调用点派生（captcha_background.go:243-245/:310-311 现状原样保留拼接式）:
// poolPrefix := fmt.Sprintf(constants.CaptchaCachePoolPrefixFormat, shape, difficulty)
// counterKey := poolPrefix + ":counter"
// itemKey    := fmt.Sprintf("%s:%d", poolPrefix, ...)
```

---

### B. 调用点替换形态（读侧）

#### B1. captcha.go —— 先例与位点同文件共存（CACHE-01）

**模拟 = 自身既有先例（目标 idiom，勿动）。** :385/:450 已用格式常量:
```go
verifiedKey := fmt.Sprintf(constants.CaptchaVerifiedKeyFormat, captchaID)
```
:491/:512:
```go
lockKey := fmt.Sprintf(constants.LoginLockKeyFormat, username)
```
import :13 已就位: `"github.com/xingran-next/xingran-go-backend/pkg/constants"`。

**待替换位点（唯一 mechanical diff = 格式串字面量 → 常量）:**
```go
// :249（TTL 1min 内联 IncrementWithExpire/Expire —— TTL 不动，Deferred）
rateLimitKey := fmt.Sprintf("captcha:rate:%s", clientIP)
// :297/:326/:361/:367/:419/:426（6 处同型）
storageKey := fmt.Sprintf("captcha:data:%s", captchaID)
// :304/:333/:356/:415（4 处同型）
attemptsKey := fmt.Sprintf("captcha:attempts:%s", captchaID)
// :503/:529（2 处同型；:507 的 1*time.Hour Expire 不动）
failKey := fmt.Sprintf("login:fail:%s", username)
```
替换后: `fmt.Sprintf(constants.CaptchaDataKeyFormat, captchaID)` 等，与 :385 先例逐字同构。

#### B2. captcha_background.go —— 池键族（CACHE-01，3 直接位点 + 4 派生点）

```go
// :146（TTL 5min 在 :187，不动）
cacheKey := fmt.Sprintf("captcha:bg:list:%s:%d", shape, difficulty)
// :243-245（preGenerateForConfig）
poolPrefix := fmt.Sprintf("captcha:cache:pool:%s:%d", shape, difficulty)
poolSize := s.config.CachePoolSize
counterKey := poolPrefix + ":counter"
// :295-296（itemKey 派生 + TTL 24h :296，不动）
itemKey := fmt.Sprintf("%s:%d", poolPrefix, (counter%int64(poolSize))+1)
	_ = s.cache.Set(ctx, itemKey, string(data), 24*time.Hour)
// :310-311（GetFromCachePool 同构）
poolPrefix := fmt.Sprintf("captcha:cache:pool:%s:%d", shape, difficulty)
counterKey := poolPrefix + ":counter"
// :326（同构派生）
itemKey := fmt.Sprintf("%s:%d", poolPrefix, (counter-1)%int64(poolSize)+1)
```
替换仅触 :146/:243/:310 三处格式串 → `constants.CaptchaBgListKeyFormat` / `constants.CaptchaCachePoolPrefixFormat`；`+ ":counter"` 与 `fmt.Sprintf("%s:%d", poolPrefix, ...)` 派生式**原样保留**。

#### B3. cache_impl 调用点三形态 —— CACHE-02 主战场（8 模块）

**模拟文件:** `internal/services/duty/duty_cache_impl.go`（import 块 = 4 个子包模块的引用 idiom，:3-14）:
```go
import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/services"
	"github.com/xingran-next/xingran-go-backend/internal/services/base"
	systemServices "github.com/xingran-next/xingran-go-backend/internal/services/system"
	"gorm.io/gorm"
)
```
> 别名固定 `systemServices`。duty/workorder/knowledge/network 四包已 import（零成本）；notice/settings/widget 与 cache_keys.go 同包（`system`，直接裸引用常量名）。

**形态 1 — plain-literal 键 → 裸常量**（duty :135-140）:
```go
// BEFORE:
return base.GetOrSetJSON(ctx, s.cache, "duty:today",
	s.getExpiration("cache.duty.today", 5*time.Minute),
	func() ([]services.TodayDutyMember, error) { return s.base.GetTodayDuty(ctx) })
// AFTER:
return base.GetOrSetJSON(ctx, s.cache, systemServices.CacheKeyDutyToday,
	s.getExpiration("cache.duty.today", 5*time.Minute), ...)   // TTL 行零改动
```

**形态 2 — 参数化 Sprintf → GetXxxKey helper**（duty :142-148）:
```go
cacheKey := fmt.Sprintf("duty:monthly:%d:%d", year, month)
// AFTER:
cacheKey := systemServices.GetDutyMonthlyKey(year, month)
```

**形态 3 — Invalidate/InvalidatePattern 实参**（network :287-321，全部 8 个失效方法的通用形）:
```go
keys := []string{fmt.Sprintf("network_device:detail:%s", deviceID)}
base.Invalidate(ctx, s.cache, keys, "NETWORK_DEVICE")
...
base.InvalidatePattern(ctx, s.cache, []string{"network_device:*"}, "NETWORK_DEVICE")
// AFTER:
keys := []string{systemServices.GetNetworkDeviceDetailKey(deviceID)}
base.InvalidatePattern(ctx, s.cache, []string{systemServices.GetNetworkDevicePattern()}, "NETWORK_DEVICE")
```
notice 的多键失效（:294-298）与全部清空（:300-304）同法:
```go
base.InvalidatePattern(ctx, s.cache, []string{fmt.Sprintf("notice:my_notices:%s:*", userID), fmt.Sprintf("notice:unread_count:%s", userID)}, "NOTICE")
base.InvalidatePattern(ctx, s.cache, []string{"notice:*"}, "NOTICE")
```

**特殊形态 — helper 内 Sprintf**（notice :204-213，注释说明两格式串为逐字节等价搬迁，**注册后 helper 体改引常量、注释保留**）:
```go
func buildMyNoticesKey(userID string, page, pageSize int, status *string) string {
	cacheKey := fmt.Sprintf("notice:my_notices:%s:page:%d:size:%d", userID, page, pageSize)
	if status != nil {
		cacheKey = fmt.Sprintf("notice:my_notices:%s:page:%d:size:%d:status:%s", userID, page, pageSize, *status)
	}
	return cacheKey
}
```

**特殊形态 — %x 动词**（widget :58-66，**`%x` 原样保留进格式常量**）:
```go
func buildWidgetCacheKey(widgetID string, params map[string]interface{}) string {
	if len(params) == 0 {
		return fmt.Sprintf("widget:data:%s", widgetID)
	}
	paramsJSON, _ := json.Marshal(params)
	paramsHash := sha256.Sum256(paramsJSON)
	return fmt.Sprintf("widget:data:%s:%x", widgetID, paramsHash[:8])
}
```

**根包 2 文件（→ pkg/constants，D-102-2 同机理豁免）:**
```go
// api_endpoint_service.go:63 与 :194（同格式两处；该文件现状不 import constants，需新增 import）
cacheKey := fmt.Sprintf("user_endpoints:%s", userID)
// mac_history_query_service.go:256（import 块 :17 已有 pkg/constants —— 零 import 成本）
cacheKey := fmt.Sprintf("mac:vendor:%s", oui)
// selector_learner.go:360-363（键构造函数集中单点；:169-174/:226 闭包位点 Phase 103 才迁，本相不动）
func (l *selectorLearnerImpl) getCacheKey(pageURL, elementID string) string {
	return fmt.Sprintf("rpa:selector:best:%s:%s", pageURL, elementID)
}
```

---

### C. STATUS 替换形态（STATUS-01，21 位点全映射既有常量、零新增）

#### C1. 常量真相源（internal/models，禁改值）

`internal/models/log.go` :67-73 与 :90-96:
```go
// JobStatus 任务状态枚举（遵循状态值规范：0=正常,1=暂停）
type JobStatus int
const (
	JobStatusNormal JobStatus = 0 // 0: 正常
	JobStatusPause  JobStatus = 1 // 1: 暂停     ← 注意命名是 Pause，不是 Stopped/Disabled
)
// JobLogStatus 任务执行日志状态枚举（0=成功, 1=失败）
type JobLogStatus int
const (
	JobLogStatusSuccess JobLogStatus = 0 // 成功
	JobLogStatusFailure JobLogStatus = 1 // 失败
)
```
`internal/models/workorder.go` :13-22（**typed 常量，`[]int` 场景需 `int()` 转换**）:
```go
type WorkOrderStatus int
const (
	WorkOrderStatusPending    WorkOrderStatus = 0 // 待处理
	WorkOrderStatusProcessing WorkOrderStatus = 1 // 处理中
	WorkOrderStatusCompleted  WorkOrderStatus = 2 // 已完成
	WorkOrderStatusClosed     WorkOrderStatus = 3 // 已关闭
	WorkOrderStatusRejected   WorkOrderStatus = 4 // 已拒绝
)
```
另有 `duty.go:25` `DutyStatusNormal = 0`、`vdi.go:51-52` `VDIServerStatusNormal/Stopped`（均已核实，登记于 status_constants_test.go 值锁表）。

#### C2. 形态 1 — 复合字面量字段（scheduler 6 处，同构先例 = 相邻 MisfirePolicy 行）

`internal/scheduler/workorder_tasks.go` :320-331（**:327 就是同结构体既有常量引用先例**，:328 是替换对象；:443-454 同构第二处）:
```go
newJob := &models.Job{
	...
	MisfirePolicy:  models.MisfirePolicyExecuteOnce,   // ← 既有先例行
	Status:         0,                                 // ← 替换为 models.JobStatusNormal
	...
}
```
同型位点: `mac_history_tasks.go:127`、`mac_history_matview_tasks.go:56`、`reconciliation_tasks.go:196`（其 :195 的 `MisfirePolicy: 1, // MisfirePolicyImmediately` 为 RESEARCH Open Question #1 顺带清候选——1 行，与同族一致性对齐）、`cron.go:43`（`Status: 0, // 成功` → `models.JobLogStatusSuccess`）。

#### C3. 形态 2 — 赋值语句（cron.go:62）

```go
jobLog.Status = 1 // 失败
// AFTER:
jobLog.Status = models.JobLogStatusFailure
```

#### C4. 形态 3 — Where/Update 占位符参数（cron.go 3 处 + vdi 1 处）

```go
// cron.go:235
if err := s.db.Where("status = ?", 0).Find(&jobs).Error; err != nil {
// AFTER: s.db.Where("status = ?", models.JobStatusNormal)
// cron.go:407 / :435（GORM Update 列值）
if err := s.db.Model(&models.Job{}).Where("id = ?", jobID).Update("status", 0).Error; err != nil {
// AFTER: Update("status", models.JobStatusNormal)   /  :435 → models.JobStatusPause（不是 Stopped！）
// vdi_sync_tasks.go:48
if err := db.Where("status = ?", 0).Find(&servers).Error; err != nil {
// AFTER: db.Where("status = ?", models.VDIServerStatusNormal)
// vdi_sync_tasks.go:85（BinaryExpr 比较）
if server.Status != 0 {
// AFTER: if server.Status != models.VDIServerStatusNormal {
```

#### C5. 形态 4 — raw SQL 内嵌字面量（cron.go:832 专属，禁字符串拼接）

```go
// BEFORE:
	Where("ds.schedule_date = ? AND ds.status = 0", date)
// AFTER（占位符 + 常量作参）:
	Where("ds.schedule_date = ? AND ds.status = ?", date, models.DutyStatusNormal)
// workorder_tasks.go:195 同法（已有注释自证 // DutyStatusNormal = 0，替换后注释可保留或随之精简）:
	Where("schedule_date = ? AND status = ?", today, 0).
```

#### C6. 形态 5/6 — int 入参比较 + IN 切片

`internal/services/scheduler/job_service.go` :318-344（:324 展示 status 的 int 本质，:332 为替换点）:
```go
	job.Status = models.JobStatus(status)   // :324 —— status 是 int 入参
	...
	if status == 0 { // 启用
// AFTER: if status == int(models.JobStatusNormal) {   （最小 diff：仅比较处加转换）
```
`internal/services/workorder/base.go` :180-197（**整切片表达式替换，WorkOrderStatus 为 typed 常量**）:
```go
	Where("status IN ?", []int{0, 1}). // 待处理或处理中     // :183
	Where("status IN ?", []int{0, 1}).                       // :191（无注释孪生）
// AFTER（两处同改）:
	Where("status IN ?", []int{int(models.WorkOrderStatusPending), int(models.WorkOrderStatusProcessing)}).
```

#### C7. 白名单位点（唯一豁免）

`internal/services/operations/geocoding_service.go` :332-335:
```go
	// F 簇：百度地图 API 返回码契约，不迁移到 models 常量（见 scripts/check-status-literals.sh 白名单）
	if baiduResp.Status != 0 {
```

---

### D. 分页迁移形态（PAGI-01，D-102-9 + 继承锁定删除）

**删除对象全貌:** `internal/utils/pagination.go`（49 行整文件——`ParsePagination` :15-29 / `PaginationParams` :6-9 / `Offset()` :32-34 / `Limit()` :37-39 / `BuildPaginationResponse` :42-49；后者零其他调用方）。

**目标 API:** `pkg/query/pagination.go` :47-62（唯一实现核心，直接可用）:
```go
// NormalizePaginationWithMax 归一化分页参数的唯一实现核心（V130R-09 D-03-2）。
// current <= 0 → DefaultCurrent；pageSize <= 0 → DefaultPageSize；
// pageSize > maxPageSize → maxPageSize（钳制，不静默重置默认）。
// 不做 MinPageSize 下限放大：pageSize 1-9 原样透传。
func NormalizePaginationWithMax(current, pageSize, maxPageSize int) (int, int) {
	if current <= 0 {
		current = constants.DefaultCurrent
	}
	if pageSize <= 0 {
		pageSize = constants.DefaultPageSize
	}
	if pageSize > maxPageSize {
		pageSize = maxPageSize
	}
	return current, pageSize
}
```

**唯一生产调用方迁移:** `internal/api/v1/system/file_handler.go` :153-174:
```go
// BEFORE:
	pagination := utils.ParsePagination(req.Page, req.PageSize)                            // :160
	files, total, err := h.service.ListFiles(c.Request.Context(), req.BusinessType, req.UserID, pagination.Offset(), pagination.Limit())  // :162
	...
	response.Success(c, utils.BuildListResponse(result, total, pagination.Page, pagination.PageSize))  // :173
// AFTER（D-102-9：cap 严格保持 100 = constants.MaxListPageSize，非 pkg 默认 200）:
	// 历史分叉自证：本端点 cap=100（constants.MaxListPageSize），刻意不用 NormalizePagination 默认 cap=200
	current, pageSize := query.NormalizePaginationWithMax(req.Page, req.PageSize, constants.MaxListPageSize)
	offset := (current - 1) * pageSize
	files, total, err := h.service.ListFiles(c.Request.Context(), req.BusinessType, req.UserID, offset, pageSize)
	...
	response.Success(c, utils.BuildListResponse(result, total, current, pageSize))   // BuildListResponse 保留（CONTEXT 明确不动）
```
**import 处置注意:** file_handler.go 的 `internal/utils` import **不能删**——:48/:73-74 仍用 `utils.GetUserID/GetUsername/GetClientIP`，:173 用 `utils.BuildListResponse`，:200 用 `utils.BuildCountResponse`。新增 `"github.com/xingran-next/xingran-go-backend/pkg/query"` 与 `"github.com/xingran-next/xingran-go-backend/pkg/constants"`。

**配套删除:** `internal/utils/utils_74_12_test.go` 的 `TestParsePaginationAndOffset`（:203-226，随 D-102-9 继承锁定一并删除）。

---

### E. 守护测试形态（D-102-5② / D-102-6）

#### E1. cache-key 内联扫描 + 等价快照 —— 模拟文件 `internal/services/system/cache_invariants_92_test.go`（全文 203 行）

**骨架五件套**（新测试文件建议同目录同包，复用全部手法）:

1. 白名单 map（:31-33）:
```go
// allowedResidues 白名单：文件名 → 允许的 interface{} 闭包式 GetOrSet 残留数。
// 初始全空 = 期望 0；未来确需豁免必须在此显式登记并附说明（diff 可见）。
var allowedResidues = map[string]int{}
```
2. AST 扫描函数（:43-81，`t.Helper()` + `parser.ParseFile` + `ast.Inspect` + 精确节点匹配 + `fmt.Sprintf("%s:%d", filepath.Base(...), pos.Line)` 命中记录）:
```go
func cacheImplResidue92(t *testing.T, path string) []string {
	t.Helper()
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, nil, 0)
	if err != nil {
		t.Fatalf("解析 %s 失败: %v", path, err)
	}
	var hits []string
	ast.Inspect(f, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		...
	})
	return hits
}
```
3. runtime.Caller 相对定位（:121-130，**v1.27 测试纪律：禁本地绝对路径断言**）:
```go
func servicesRoot92(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller 失败——无法定位测试文件位置")
	}
	// thisFile = .../internal/services/system/cache_invariants_92_test.go
	return filepath.Dir(filepath.Dir(thisFile))
}
```
4. 硬档/白名单调和（:151-191，`len(hits) > allowed` 逐条 Errorf + 白名单卫生 Logf）:
```go
		allowed := allowedResidues[name]
		if len(hits) > allowed {
			for _, h := range hits[allowed:] {
				t.Errorf("... 残留：%s（... 硬档——已清零面不可倒退；确需豁免请显式登记 ...）", h)
			}
		}
```
5. 扫描器空转防护（:155-157）:
```go
		if len(files) == 0 {
			t.Errorf("目录 %s 未匹配到任何 cache_impl 文件——扫描器空转，检查路径口径", dir)
			continue
		}
```
> cache-key 扫描的匹配对象（新写）：**`fmt.Sprintf` 首参为含 `:` 的 string 字面量**（键格式形态），窄扫 D-102-7 的 12 文件清单。Phase 103 待迁闭包位点（selector_learner:169-174/:226 的 Get/Set 调用）天然不命中——扫描按键格式字面量匹配，不含调用形态。

#### E2. 等价快照测试（D-102-5①）—— 快照表驱动，期望列 = 原字面量

```go
cases := []struct{ name, want, got string }{
	{"CaptchaDataKeyFormat", "captcha:data:%s", constants.CaptchaDataKeyFormat},
	{"CacheKeyNoticeMyNotices", "notice:my_notices", systemServices.CacheKeyNoticeMyNotices},
	// ... 39 格式全量（期望值来源 = RESEARCH §位点全量清单的字面量列）
}
```
**防同义反复（RESEARCH Pitfall 5）:** 期望列必须是字符串字面量；测试文件内出现 `constants.Xxx` 作期望值 = 守护失效。放置包拆两处（RESEARCH Open Question 3 建议）: `pkg/constants/cache_102_test.go`（captcha 6 + 根包 2）+ `internal/services/system/cache_keys_102_test.go`（31 格式 + 内联扫描）。

#### E3. status 使用点 AST 扫描（D-102-6）—— 宿主 `internal/models/status_constants_test.go`

**同文件扩展（明文）。** 可复用的既有件:
- 值锁表 :86-200 `expectedStatusValues`（**不动**——本相零新增常量，表全程绿的前提）
- AST const 读取器 :289-348 `readStatusConsts()`（parser + Glob + `token.INT` 字面量解析，手法可复用）
- 双向断言纪律 :214-230（expected→actual 缺失/漂移 + actual→expected 未登记）
- `_test.go` 排除 :301-303、跨包同名同值容忍 :334-341

**需新写的匹配逻辑**（6 种 AST 形态，RESEARCH §D-102-6 清单）: ①复合字面量键 `Status: 0`（KeyValueExpr, Key==Ident("Status")）②赋值 `x.Status = 1`（AssignStmt LHS SelectorExpr.Sel=="Status"）③二元比较（.Status 选择器或 status 标识符 vs int 字面量）④⑤GORM `Where/Update/Raw` 调用中含 "status" SQL 串的 int 字面量实参 ⑥`[]int{...}` 切片实参（status IN ?）。

**白名单表形态**（文件 → 豁免原因，invariants_92 卫生纪律）:
```go
var statusLiteralWhitelist = map[string]string{
	"geocoding_service.go": "百度地图 API 返回码（非 DB status，F 簇契约，audit 定性）",
}
```

**必须排除的面**（广谱扫描实测）: `internal/core/db/migrations/archive/applied/*.go`（迁移历史，禁改禁扫）；字符串型 status（AST 限定数字字面量自然排除）；非 status 数值字面量（`captcha_background.go:213` difficulties、SNMP 类别码——按标识符名匹配自然排除）；`_test.go` 全部。

---

## Shared Patterns

### SP-1: 「格式串字面量 → 具名常量」最小 diff 纪律
**来源:** captcha.go :385 先例 + CLAUDE.md §Cache Service Convention
**Apply to:** 全部 39 种键格式的调用点（B1-B3）
键值必须逐字符等于原字面量（Redis 运行时契约）；**每处替换只改键实参一个 token**，TTL 表达式 / `getExpiration(configKey, ...)` / 缓存调用形态（base.GetOrSetJSON/Invalidate/InvalidatePattern）零改动（diff 审查标准：本相只允许键实参行变化）。

### SP-2: 占位符参数优先
**来源:** cron.go:235/:407 既有占位符形态
**Apply to:** STATUS 全部 SQL 位点（C4-C6）
`Where("status = ?", models.XxxStatus)`；禁 `fmt.Sprintf` 拼 SQL（引入注入面模式先例）。`status IN ?` 走整切片表达式替换 + typed 常量 `int()` 转换。

### SP-3: typed 常量转换口径
**来源:** models/log.go:68 `type JobStatus int` + workorder.go:14 `type WorkOrderStatus int`
**Apply to:** C2-C6
- 赋给 typed 结构体字段（`Status JobStatus`）→ 直接引用，无转换
- 与 int 变量比较（job_service.go:332）→ `status == int(models.JobStatusNormal)`
- `[]int` 切片实参（base.go:183/191）→ `[]int{int(models.WorkOrderStatusPending), ...}`（`[]models.WorkOrderStatus{...}` 亦可，GORM 均可序列化，**择一统一**，A2 建议 `[]int{int(...)}` 零风险形态）

### SP-4: AST 守护测试纪律
**来源:** cache_invariants_92_test.go:16 + :121-130
**Apply to:** 两个新/扩展守护（E1/E3）
文件定位一律 runtime.Caller 相对推导（禁本地绝对路径）；白名单显式登记 + 附原因 + 卫生提示；扫描器空转自检；纯 AST 精确匹配（D-102-6 明文否决字符串 grep）。

### SP-5: 既有 grep 棘轮脚本的共存关系（**planner 需决策**）
**来源:** `scripts/check-status-literals.sh`（RESEARCH.md 未收录的发现）
既有 Phase 69 grep 棘轮守护，ALLOWED 表仅一条 `["internal/services/operations/geocoding_service.go"]=1`（:46-48，豁免原因与 D-102-6 白名单同源）。要点：
- 其扫描范围仅 `internal/api/v1 + internal/services`，**不含 internal/scheduler**（本相 15/21 位点在其盲区，替换与否脚本都绿）
- workorder/base.go 的 `[]int{0, 1}` 不命中其 4 种模式（无 `status = N` / `Status: N` / 大写 S 比较 / `"status": N` 形态）
- D-102-6 AST 扫描上线后两者并存无冲突；是否顺势退役该脚本（其 header :28-29 自述「CI candidate, NOT wired」）属 planner discretion——建议本相不动（最小 diff），白名单原因文案保持同源即可

### SP-6: 值锁表注册缺口知情项（**planner 知情即可**）
**来源:** status_constants_test.go :63-80/:86-200 实读核对
RESEARCH.md 声称「全部已登记于 expectedStatusValues」对 **WorkOrderStatus 族不成立**——watchedStatusPrefixes 无 `"WorkOrderStatus"` 前缀、值锁表无该族条目（WorkstationStatus 有、WorkOrderStatus 无）。影响：本相零新增常量、不改 models，值锁不涉及该族，**无阻塞**；但 planner 勿在 plan 中声称「base.go 替换受既有值锁保护」——该族当前不在锁内。若 planner 判断需补登记（+1 前缀 +5 行），须走 CLAUDE.md 既有流程并随本相顺带交付，属可选扩展。

---

## No Analog Found

无完全无模拟的文件。两处「部分模拟」如实标注：

| 文件 | Role | Data Flow | 缺口与替代 |
|------|------|-----------|-----------|
| `internal/models/status_constants_test.go` 扩展（使用点扫描） | test | — | 值锁（常量定义侧）先例在位，但**使用点扫描**（业务代码侧）是本相新增能力——6 种 AST 形态的匹配逻辑需新写，骨架（parser/Inspect/白名单/runtime.Caller）复制 invariants_92 |
| `pkg/constants/cache_102_test.go` | test | — | 快照表驱动等价测试无逐字同构先例（`pkg/constants/pagination_test.go` Phase 89 为 AST 锁先例，形态相近）；表驱动结构参照 RESEARCH Code Examples 与 E2 |

---

## Metadata

**Analog search scope:** internal/core, internal/services/{system,duty,workorder,knowledge,network,rpa}, internal/services 根包, internal/scheduler, internal/models, internal/api/v1/system, internal/utils, pkg/constants, pkg/query, scripts/

**Files scanned:** 28 个文件实读（全量或定点窗口），含 1 个 RESEARCH 未收录发现（check-status-literals.sh）

**关键核实结论（对 RESEARCH 的增量）:**
1. `scripts/check-status-literals.sh` 既有 grep 棘轮存在，geocoding 白名单先例已双登记（脚本 ALLOWED + :332 注释）→ SP-5
2. `WorkOrderStatus` 族未登记进 status_constants_test.go 值锁（RESEARCH 表述偏差）→ SP-6
3. duty/workorder/knowledge/network cache_impl 的 system 包 import 别名实为 `systemServices`（:12）
4. `api_endpoint_service.go` 现状不 import pkg/constants（需新增）；`mac_history_query_service.go` 已 import（:17，零成本）
5. file_handler.go 的 `internal/utils` import 不可随迁移删除（:48/:73/:173/:200 仍用其他 helper）
6. reconciliation_tasks.go:195 `MisfirePolicy: 1` 位点实读确认，相邻 :196 即替换对象
7. 21 处 STATUS 位点行号全部与 RESEARCH 清单吻合（零漂移）；CACHE 位点抽查吻合

**Pattern extraction date:** 2026-09-07
