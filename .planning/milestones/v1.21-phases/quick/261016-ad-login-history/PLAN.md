---
task: 261016-ad-login-history
status: pending
date: 2026-10-16
mode: quick
type: execute
depends_on: []
requirements: [AD-LOGIN-HISTORY]
autonomous: true
files_modified:
  - internal/models/ad_login_history.go
  - internal/core/db/migrations/migration_214_create_ad_login_history.go
  - internal/core/db/database.go
  - internal/services/addomain/login_history.go
  - internal/services/addomain/login_history_query.go
  - internal/services/addomain/computer.go
  - internal/api/v1/system/ad_login_history_handler.go
  - internal/api/v1/system/ad_domain_router.go
  - internal/services/addomain/login_history_partition.go
  - internal/scheduler/ad_login_history_tasks.go
  - internal/core/core.go
  - xingran-react-frontend/src/lib/adDomainApi.ts
  - xingran-react-frontend/src/pages/ad-domain/computers/index.tsx
user_setup: []
---

# AD 设备登录历史(模仿 mac_history 模式)

## 目标

AD 域控设备 `sys_ad_computer.last_logon` 目前在每次同步(5 分钟 cron)时被 upsert 覆盖,登录历史永久丢失。本任务新建"AD 设备登录历史":每次同步检测 lastLogon 变化并记录登录事件到新分区表 `sys_ad_login_history`,提供查询 API 与前端 Drawer 展示。

## 锁定决策(不可更改)

| # | 决策 |
|---|------|
| D1 | 记录触发: 同步时对比新旧 lastLogon,值变化才插一条;设备首次发现且 lastLogon 非 nil 记一条 `initial` |
| D2 | 前端展示: 设备列表行操作"登录历史"按钮 → Drawer 展示该设备登录历史表格(时间倒序) |
| D3 | 记录字段: 设备ID快照、计算机名快照、ad_config_id、登录时间(AD lastLogon 值)、检测时间、登录用户(description parts[1])、IP 地址 |
| D4 | 保留策略: 按月分区 + 定期清理(模仿 mac_history 的 PartitionService + cron 模式) |

## 关键代码位置(已复核)

- 同步写入点: `internal/services/addomain/computer.go`
  - `syncComputers()` L372-474 主流程;`processComputerEntry()` L548-578 分流 toCreate/toUpdate(**变化检测挂点,必须在调用 `updateComputerFields` 覆盖指针前快照旧值**)
  - `updateComputerFields()` L333 覆盖 `computer.LastLogon`;`batchUpdate()` L581 ON CONFLICT (ad_config_id, computer_name) upsert 覆盖 last_logon
  - `buildComputerFromEntry()` L263 新设备构建;`parseComputerDescriptionForUser()` L168 (parts[1]);`parseComputerDescription` 返回 map 中 `lastLogonUser`(index 1)/`ipAddress`(index 2) 可直接复用
  - `ComputerService` 在 `internal/services/addomain/sync.go:154` 每次 sync 现场构造 `NewComputerService(s.db)`
- toCreate 路径: `batchCreate` 后 GORM BeforeCreate hook 在切片元素上回填 ID(computer.go L22-27 同款模式)→ initial 事件的 computer_id 必须在 batchCreate 成功后解析
- toCreate 去重 L436-453: 转为 map 后回 slice,**保留第一条**;initial 事件需按 (configID, computerName) 去重避免同名双记
- mac_history 参照: 模型 `internal/models/device_mac_history.go`;cron `internal/scheduler/mac_history_tasks.go`(SetXXX 全局注入 + upsertMACHistoryJob);core.go L515-526 注册块;`migration_mac_history.go` 的 `CreateMACHistoryTable` 是**未被调用的遗留代码,不要模仿它的接线方式**,接线模仿 Migrate211/213
- 路由: `internal/api/v1/system/ad_domain_router.go` L128-136 `computers` group,权限 `ops:ad:computer:view`/`ops:ad:computer:detail`(**新端点加入此 group,复用现有权限,无新菜单/migration 授权**)
- Handler 风格: `ad_domain_handler.go` L690-703 `ListComputers`(ShouldBindJSON → service → `response.Page(c, list, total, current, pageSize)`)
- migration 接线: `internal/core/db/database.go` PG 分支 advisory-lock 块内顺序调用(L1034-1044 为 211-213 先例,非阻断 Errorf);SQLite 分支(else,L1045-1080)不注册 PG-only 迁移
- AutoMigrate 模型注册表: database.go L718-724(`&models.ADComputer{}` 等)——**新模型禁止加入此列表**
- 前端: `xingran-react-frontend/src/lib/adDomainApi.ts` L433-446 computers API 区块(`post` + `withDefaultPagination`);页面 `src/pages/ad-domain/computers/index.tsx`(columns L189 起,操作列 L280,详情 Descriptions L390-442,时间显示用 `new Date(...).toLocaleString("zh-CN")`)

## 已确认的类型决策

- `sys_ad_computer.id` 是 **uuid**(BaseModel.ID = string + `gorm:"type:uuid"`,base.go L12)→ 新表 `computer_id`/`ad_config_id` 用 **uuid 列**(Go 侧 string + `gorm:"type:uuid"`),与 `DeviceMACHistory.DeviceID` 先例一致,JOIN 无类型冲突。不走 varchar(64) 方案。
- 时间列全部 **TIMESTAMPTZ**(GORM time.Time 默认),**不加 AfterFind 时区修补**——mac_history 的 AfterFind 是 TIMESTAMP 裸时间戳的历史包袱(分区键禁 ALTER 类型只能读取层修),新表从 schema 层根治。

## 风险与陷阱(实现时必读)

1. **分区表必须手写 SQL migration**(GORM AutoMigrate 不支持 PARTITION BY);新模型 `ADLoginHistory` **绝不加入** database.go AutoMigrate 注册列表,否则 GORM 会对分区父表做 ALTER 触发 42P16 类问题(memory: GORM AutoMigrate 被 MV 阻塞)。
2. **.sql 文件不会被自动加载**(memory: ruoyi-migrations-no-sql-autoloader)→ 必须写成 `migration_214_create_ad_login_history.go` 函数,并在 database.go PG 分支 Migrate213 之后显式调用。
3. **变化检测三分支**: 旧 nil→新非 nil 记 `initial`;旧非 nil→新值更大记 `login`;相等(无变化)与变小(AD 时钟回拨/多 DC lastLogon 回退)**都不记**。
4. **批量收集 + 一次性批量 INSERT**,禁止在 `processComputerEntry` 里逐条 INSERT;收集在同步写库(batchCreate/batchUpdate)**成功之后**统一 flush,避免同步失败后下次重复记录。
5. **历史插入失败不阻断主同步**: flush 出错仅 `applogger.Errorf` + continue,不 return err。
6. **initial 事件的 ID 时序**: toCreate 记录的 uuid 在 `batchCreate` 执行后才回填,且 toCreate 有同名去重(保留第一条)→ 事件收集用 `configID+"/"+computerName` 作 key,batchCreate 后从最终切片建 name→ID 映射解析;解析不到则跳过并 Warn。同名冲突时 initial 事件按 key 去重只留一条。
7. **timestamptz 列不加 AfterFind**(memory: pgx 读 timestamp without time zone 强制 UTC 的坑;timestamptz 无此问题,画蛇添足会二次偏移)。
8. **SQLite 全链路跳过**: writer/query/partition 三处都要 `db.Dialector.Name() != "postgres"` 守卫直接返回空/跳过,保证 sqlite 内存单测(`go test ./...`)与 dev 环境不因缺表报错(mac_history 先例: 迁移与分区在非 PG 直接跳过)。
9. **列宽对齐源表**(memory: migration-sql-name-must-match-model): `computer_name VARCHAR(255)`、`ip_address VARCHAR(50)`、`login_user VARCHAR(255)`(与 buildComputerFromEntry 的 safeAttr 尺寸一致)。
10. **cron 时段避开 02:00**(mac_history_cleanup)→ AD 分区维护用 `0 30 2 * * ?`(每天 02:30,7 段含秒)。
11. **前端 Drawer 打开才拉数据**,列表页不预取历史(避免 N+1);Table 分页服务端({current, pageSize}),Drawer 内翻页触发重新请求;不引入新的 useEffect 对象依赖(memory: useEffect unstable deps 死循环)。
12. **查询端点只读,不需要 operlog.Record**(operlog 仅写操作)。
13. 排序由后端固定 `login_time DESC`,前端列不加 server sorter,避免丢参类坑(memory: server-sort-loadfunc-param-drop)。

## 范围边界(不做)

- 不做独立登录历史页面(仅设备列表行 Drawer)
- 不做导出(Excel/CSV)
- 不改动 mac_history 相关任何文件
- 不做登录历史统计/图表/TOP-N
- 不新增菜单与权限(复用 `ops:ad:computer:view`)
- 不记录 lastLogon 之外的事件(pwdLastSet/logonCount 变化不记)
- 不做存量数据回填(仅上线后增量记录)

---

## 任务分解(每任务一个 commit,严格顺序执行)

### Task 1: 模型 + migration_214 分区表

**文件**:
- 新建 `internal/models/ad_login_history.go`
- 新建 `internal/core/db/migrations/migration_214_create_ad_login_history.go`
- 修改 `internal/core/db/database.go`(仅 PG 分支加一处调用)

**关键改动**:
1. `models/ad_login_history.go`: 定义 `ADLoginEventType` string 常量 `ADLoginEventInitial = "initial"`、`ADLoginEventLogin = "login"`;`ADLoginHistory` struct 字段: `ID`(type:uuid primary_key)、`ComputerID`(type:uuid not null)、`ComputerNameSnapshot`(size:255 not null)、`ADConfigID`(type:uuid not null)、`LoginTime`(time.Time not null)、`DetectedAt`(time.Time not null)、`LoginUser`(size:255)、`IPAddress`(size:50)、`EventType`(size:20 not null)、`CreatedAt`;`TableName()` 返回 `sys_ad_login_history`;`BeforeCreate` hook 空 ID 时 `uuid.New().String()`(仿 DeviceMACHistory)。**不写 AfterFind**。
2. `migration_214_create_ad_login_history.go`: `func Migrate214CreateADLoginHistory(db *gorm.DB) error`——仿 `migration_211_create_config_restore_task.go` 风格: `isPostgreSQL(db)`(migration_helpers.go 已有)为 false 直接 return nil;SQL 依次执行(全部幂等 `IF NOT EXISTS`):
   - `CREATE TABLE IF NOT EXISTS sys_ad_login_history (...列同模型, 时间列 TIMESTAMPTZ, id UUID NOT NULL DEFAULT gen_random_uuid()) PARTITION BY RANGE (login_time)`(分区表不建 PRIMARY KEY,同 mac_history 注释理由)
   - 当前月 + 未来 2 个月的分区 `sys_ad_login_history_YYYY_MM`(计算边界逻辑仿 migration_mac_history.go L52-84)
   - BRIN 索引 `idx_ad_login_history_login_time_brin ON sys_ad_login_history USING BRIN (login_time) WITH (pages_per_range = 128)`
   - 复合索引 `idx_ad_login_history_computer_time ON sys_ad_login_history (computer_id, login_time)`
3. `database.go`: PG 分支(advisory-lock 块内)`migrations.Migrate213RecomputeBuildingTotalFloors` 调用(L1042-1044)之后追加 Migrate214 调用,错误处理同款非阻断 `applogger.Errorf(...非阻断,留待下次启动...)`。**SQLite 分支不注册**。

**验收**:
- `go build ./...` 通过
- `grep -n "models.ADLoginHistory" internal/core/db/database.go` 零匹配(确认未进 AutoMigrate 列表)
- `go test ./internal/core/db/...` 存量测试全绿

**Commit**: `feat(261016-ad-login-history): AD登录历史模型 + migration_214 月分区表`

---

### Task 2: 同步集成——lastLogon 变化检测 + 批量写入

**文件**:
- 新建 `internal/services/addomain/login_history.go`
- 修改 `internal/services/addomain/computer.go`
- 新建 `internal/services/addomain/login_history_test.go`

**关键改动**:
1. `login_history.go`(package addomain):
   - `type ADLoginHistoryWriter struct{ db *gorm.DB }` + `NewADLoginHistoryWriter(db)`;`isPostgres()` 守卫(非 PG 一切跳过)
   - 纯函数 `detectLoginEvent(oldLL, newLL *time.Time) (models.ADLoginEventType, bool)`: 双 nil→false;旧 nil 新非 nil→(`initial`, true);旧非 nil 且新>旧→(`login`, true);相等或新<旧→false
   - 内部收集结构 `pendingLoginEvent{ computerKey string; computerID string; eventType; loginTime; loginUser; ipAddress; detectedAt time.Time }`(toCreate 用 computerKey, toUpdate 直接带 computerID)
   - `flushLoginEvents(ctx, db, events)`: 先按需解析 toCreate 的 computerKey→ID(由调用方传入映射),构建 `[]models.ADLoginHistory` 后 `CreateInBatches(ctx, rows, 500)`;**任何错误仅 log,不向上返回**
2. `computer.go`:
   - `syncComputers` 内声明 `var pending []pendingLoginEvent`(或 map 兼 initial 去重),作为新参数传入 `processComputerEntry`
   - `processComputerEntry` 两个 existing 分支: 调 `updateComputerFields` **之前** `oldLL := existing.LastLogon` 快照,调用之后取 `newLL := existing.LastLogon`,跑 `detectLoginEvent`,命中则追加事件;`loginUser`/`ipAddress` 取 `parsedDesc["lastLogonUser"]`/`parsedDesc["ipAddress"]`(parseComputerDescription 已解析,不重复切 description)
   - toCreate 分支: `newComputer.LastLogon != nil` 时追加 `initial` 事件(带 computerKey);**initial 事件按 computerKey 去重**(配合 L436 的同名保留第一条逻辑)
   - `syncComputers` 步骤 7 `batchUpdate` 成功之后: 若 `batchCreate` 走过且 pending 有 toCreate 事件,从最终 `computersToCreate` 切片(此时 ID 已被 BeforeCreate 回填)建 `computerName→ID` 映射解析;解析不到的跳过 + Warn;最后调 `flushLoginEvents`,失败仅 Errorf
3. `login_history_test.go`: `detectLoginEvent` 表驱动单测——双 nil、nil→值(initial)、值→更大(login)、相等(无)、回拨变小(无)至少 5 例;**不依赖 DB**

**验收**:
- `go build ./...` 通过
- `go test ./internal/services/addomain/` 全绿(含新单测;sqlite 内存库下 writer 被 isPostgres 守卫跳过,存量 computer_78_06_test 不受影响)

**Commit**: `feat(261016-ad-login-history): AD同步 lastLogon 变化检测 + 批量写入登录历史`

---

### Task 3: 查询 API(handler + router + service)

**文件**:
- 新建 `internal/services/addomain/login_history_query.go`
- 新建 `internal/api/v1/system/ad_login_history_handler.go`
- 修改 `internal/api/v1/system/ad_domain_router.go`

**关键改动**:
1. `login_history_query.go`(package addomain):
   - `ADLoginHistoryQuery{ ComputerID string \`json:"computerId" binding:"required"\`; Current, PageSize int; StartTime, EndTime string(可选,RFC3339) }`
   - `ADLoginHistoryService` 接口 + 私有 impl + `NewADLoginHistoryService(db)` 构造(Handler-Service 标准模式)
   - `ListComputerLoginHistory`: 非 PG 直接返回空列表+0;分页钳制(`Current<=0→1`,`PageSize<=0→默认`,`>100→constants.MaxListPageSize`,仿 computer.go L79-89);条件 `computer_id = ?`(+可选时间范围),`ORDER BY login_time DESC`,先 Count 后 Offset/Limit
2. `ad_login_history_handler.go`: `ADLoginHistoryHandler{ svc addomain.ADLoginHistoryService }` + `NewADLoginHistoryHandler(db *gorm.DB)`(内部构造 service);方法 `ListComputerLoginHistory(c *gin.Context)`: ShouldBindJSON(错误→`response.Error` 400)→ service → `response.Page(c, list, total, req.Current, req.PageSize)`(仿 ADDomainHandler.ListComputers L690-703);带 Swagger 注解 @Tags AD域管理。**不加 operlog**(只读)。
3. `ad_domain_router.go`: 在 `SetupADDomainRouter` 内构造 `historyHandler := NewADLoginHistoryHandler(core.GetDB())`,并在 L128-136 `computers` group 花括号内追加 `computers.POST("/login-history", historyHandler.ListComputerLoginHistory)`(自动继承 group 上 `ops:ad:computer:view`/`ops:ad:computer:detail` 权限,零新权限)。

**验收**:
- `go build ./...` 通过
- `go test ./internal/api/v1/system/` 存量测试全绿
- `grep -n "login-history" internal/api/v1/system/ad_domain_router.go` 命中 1 行

**Commit**: `feat(261016-ad-login-history): AD设备登录历史查询 API(/computers/login-history)`

---

### Task 4: 分区服务 + cron 定时清理

**文件**:
- 新建 `internal/services/addomain/login_history_partition.go`
- 新建 `internal/scheduler/ad_login_history_tasks.go`
- 修改 `internal/core/core.go`

**关键改动**:
1. `login_history_partition.go`(package addomain,单模块内聚,不放 services 根):
   - `ADLoginHistoryPartitionService` 接口 4 方法(`CreateMonthlyPartition` / `EnsurePartitionsExist` / `DropExpiredPartitions` / `GetRetentionDays`)+ 私有 impl + `NewADLoginHistoryPartitionService(db)`
   - 整体仿 `internal/services/mac_history_partition.go`: 每方法 isPostgres 守卫;分区名 `sys_ad_login_history_YYYY_MM`;分区名正则 `^sys_ad_login_history_(\d{4})_(\d{2})$`;pg_inherits 查询父表换为 `sys_ad_login_history`;保留期读 sys_config key `ops.ad.login_history.retention_days`,默认 120 天、最小 30 天(低于最小值回退默认 + Warn)
2. `ad_login_history_tasks.go`(package scheduler,仿 mac_history_tasks.go):
   - 本包内声明最小 `ADPartitionService` 接口(避免依赖 addomain 包,mac 同款防环手法)+ `SetADLoginHistoryPartitionService`(sync.Once 全局注入)
   - `RegisterADLoginHistoryTasks(scheduler *Scheduler, db *gorm.DB)`: `scheduler.RegisterTask("ad_login_history_cleanup", handler)`;handler 内依次 `EnsurePartitionsExist(ctx, 2)` + `DropExpiredPartitions(ctx)`;db 非 nil 时 upsert sys_job 记录(自带小 helper,仿 upsertMACHistoryJob: job 名"AD登录历史分区维护",invokeTarget `ad_login_history_cleanup`,cron `0 30 2 * * ?`,MisfirePolicyExecuteOnce + JobStatusNormal)
3. `core.go`: 在 MAC 历史注册块(L515-526)之后仿 L521"单独构造,不挂 Core 字段"模式: `scheduler.SetADLoginHistoryPartitionService(addomainServices.NewADLoginHistoryPartitionService(c.GetDB()))` + `scheduler.RegisterADLoginHistoryTasks(c.Scheduler, c.GetDB())` + 一行 Infof(注意 core.go 已 import addomainServices 与 scheduler,无需新增 import;若包别名不同以现文件为准)

**验收**:
- `go build ./...` 通过
- `go test ./internal/scheduler/` 存量测试全绿
- 启动日志(人工/后续验收)出现"AD登录历史"注册 Infof;sys_job 出现新记录(PG 环境)

**Commit**: `feat(261016-ad-login-history): AD登录历史分区维护 cron(每日 02:30)`

---

### Task 5: 前端——设备行"登录历史" Drawer

**文件**:
- 修改 `xingran-react-frontend/src/lib/adDomainApi.ts`
- 修改 `xingran-react-frontend/src/pages/ad-domain/computers/index.tsx`

**关键改动**:
1. `adDomainApi.ts`(L433-446 computers 区块追加):
   - `export interface ADLoginHistoryRecord { id: string; computerId: string; computerNameSnapshot: string; adConfigId: string; loginTime: string; detectedAt: string; loginUser?: string; ipAddress?: string; eventType: "initial" | "login"; createdAt: string }`
   - `export function getADComputerLoginHistory(params: { computerId: string; current?: number; pageSize?: number }): Promise<BaseResponse<PageResponse<ADLoginHistoryRecord>>>` → `post("/ad-domain/computers/login-history", withDefaultPagination(params))`(与 L438 getADComputerList 同款封装,**禁止裸 axios**)
2. `computers/index.tsx`:
   - 操作列(L280 render)追加"登录历史"按钮(antd Button size small type link,onClick 打开 Drawer 并立即发起首次请求——**数据只在 Drawer 打开/翻页时拉取,列表页不预取**)
   - 新增 state: `historyOpen`、`historyComputer: ADComputerDetail | null`、`historyList: ADLoginHistoryRecord[]`、`historyLoading`、`historyPagination{current,pageSize,total}`
   - 新增 Drawer(antd 6,`width={720}`,title=`登录历史 - ${historyComputer?.computerName ?? ""}`,destroyOnClose): 内嵌 Table,rowKey="id",columns: 登录时间(loginTime,`new Date(...).toLocaleString("zh-CN")` 与 L432 现有惯例一致)、事件类型(initial→"首次发现"/login→"登录")、登录用户(loginUser 空显示 "-")、IP 地址、检测时间(detectedAt);`pagination` 服务端模式 onChange 触发重新请求;**不加 server sorter**(后端固定 login_time DESC)
   - 加载函数用普通 async 函数由 onClick/onChange 显式调用,**不要**用带对象依赖的 useEffect(防死循环,memory: useEffect unstable deps)

**验收**:
- `cd xingran-react-frontend && npm run type-check` 通过
- `npm run build` 通过(可选)

**Commit**: `feat(261016-ad-login-history): 前端设备列表登录历史 Drawer`

---

### Task 6: 全量测试 + 人工验收

**文件**:
- 新建 `internal/services/addomain/login_history_query_test.go`
- (如 Task 2 未覆盖)补 `detectLoginEvent` 边界用例

**关键改动**:
1. `login_history_query_test.go`: sqlite 内存库构造 service → `ListComputerLoginHistory` 返回空列表 + total 0 且不报错(验证 isPostgres 守卫;仿同包既有 sqlite 测试建库手法)
2. 回归: `go build ./...` + `go test ./...` 全绿;前端 `npm run type-check`
3. **人工验收清单**(PG 环境,`checkpoint:human-verify`,执行器完成后暂停交用户确认):
   - 启动后端,确认启动日志无 migration_214 报错,PG 中 `sys_ad_login_history` 表 + 3 个月分区 + 2 个索引存在(`\d+ sys_ad_login_history`)
   - 手动触发一次 AD 同步(`POST /system/ad-domain/configs/:id/sync`)→ 查 `sys_ad_login_history` 有 `initial` 记录(设备已存在且 last_logon 非 nil 的场景下首轮为 login/initial 混合,首轮全量都是"相对 DB 旧值的变化")
   - 等待/再触发一次同步(无新登录)→ 行数不增长(无变化不记录)
   - 前端 AD域控 → 电脑设备 → 任一行"登录历史" → Drawer 出数据,时间倒序,翻页正常
   - 5 分钟 cron 跑过若干轮后 `sys_ad_computer.last_logon` 更新时,历史表新增对应 `login` 行

**验收**:
- `go test ./...` 全绿;`npm run type-check` 通过
- 人工清单全部确认 → 用户回复"approved"

**Commit**: `test(261016-ad-login-history): 查询服务 sqlite 守卫单测 + 回归`

---

## must_haves(目标倒推)

- **truths**:
  - AD 同步检测到 lastLogon 变化时,`sys_ad_login_history` 新增一行,同步失败/历史写失败互不阻断
  - lastLogon 无变化或回拨时,行数不增长
  - 用户在设备列表点"登录历史"能看到该设备时间倒序的登录事件(登录时间/用户/IP/检测时间)
  - 过期分区被每日 cron 自动清理,保留期默认 120 天可经 sys_config 调整
- **artifacts**:
  - `internal/models/ad_login_history.go`(ADLoginHistory + 事件常量)
  - `internal/core/db/migrations/migration_214_create_ad_login_history.go`(手写分区 SQL,进 database.go PG 分支)
  - `internal/services/addomain/login_history.go`(writer + detectLoginEvent)
  - `internal/services/addomain/login_history_query.go` + `internal/api/v1/system/ad_login_history_handler.go` + 路由 `POST /system/ad-domain/computers/login-history`
  - `internal/services/addomain/login_history_partition.go` + `internal/scheduler/ad_login_history_tasks.go`(sys_job: ad_login_history_cleanup)
  - 前端 `getADComputerLoginHistory` + computers/index.tsx Drawer
- **key_links**:
  - `processComputerEntry`(旧值快照→updateComputerFields→detectLoginEvent)→ pending 收集 → batchCreate/batchUpdate 后 flush
  - `ad_domain_router.go` computers group → login-history 端点(继承 ops:ad:computer:view)
  - `adDomainApi.getADComputerLoginHistory` → Drawer Table(打开才拉取)

## 完成定义

- [ ] 6 个 commit 按序落库,`go build ./...` / `go test ./...` / `npm run type-check` 全绿
- [ ] 人工验收清单(Task 6)全部通过
- [ ] mac_history 相关文件零改动(`git diff --name-only` 确认)
