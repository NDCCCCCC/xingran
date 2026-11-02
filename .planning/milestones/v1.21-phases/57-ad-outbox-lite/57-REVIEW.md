# Phase 57 代码审查报告（AD Push Queue / outbox-lite）

**审查范围**：`migration_215_create_ad_push_queue.go`、`models/ad_push_queue.go`、`core/db/database.go`、`addomain/ad_push_queue_service.go`、`system/user_service.go` + 4 个测试文件
**深度**：standard（重点按 review_priorities 的 10 项加权）
**验证**：`go build ./...` 通过；`go test ./internal/services/system/ ./internal/api/v1/system/ -count=1` 全绿（16.3s / 9.5s）

---

## 一、总体评价

核心机制实现得**相当扎实**，review_priorities 里最容易翻车的几点全部正确规避，且都能在代码中验证：

| 优先项 | 结论 | 证据 |
|---|---|---|
| ① ON CONFLICT 对 PARTIAL 索引 | ✅ 正确 | 谓词放在 `clause.OnConflict.TargetWhere`（`ad_push_queue_service.go:125`），非 `Where`；文本与 `migration_215:79` 的索引谓词**逐字节一致**且**两侧都无表名限定**；已核对 gorm v1.30.5 `clause/on_conflict.go:35` 的 `Build` 顺序确实把 `TargetWhere` 输出在 `(cols)` 之后、`DO UPDATE` 之前 → `ON CONFLICT ("entity_type","entity_id","action_type") WHERE status IN ('pending','failed') DO UPDATE SET ...` |
| ② 模型/DDL 漂移 | ✅ 无漂移 | `id uuid` / `varchar(32|64|20)` / `integer NOT NULL DEFAULT 0` / `text` 逐列对齐；PG 侧 `gorm.io/driver/postgres` 的 `schema.Time → "timestamptz"`（已核对驱动源码 235-239 行）与 DDL 一致，**不会每次启动发 ALTER**，0A000 风险不成立 |
| ③ uniqueIndex tag | ✅ 模型无任何 uniqueIndex tag | `ad_push_queue.go` 只有 `primary_key` |
| ④ 软删 | ✅ 无 `DeletedAt`、未内嵌 `BaseModel` | 且 `ad_push_queue_test.go:135-139` 有 `deleted_at` 列不存在的断言守护 |
| ⑤ 事务边界 | ✅ 三处都在闭包内用 `tx`，未逃逸到外层 `db` | `user_service.go:226 / 270 / 610` |
| ⑥ 回滚正确性 | ✅ 无吞错、无缺失 `return tx.Error`、无闭包返回非 `*gorm.DB` | 失败均 `return apperrors.DatabaseError(err)` → GORM 跳过 Commit |
| ⑦ id 去重 | ✅ `EnqueueDisable` 去重（`ad_push_queue_service.go:204-218`） | 但**对应的测试在 sqlite 上是假红/假绿**，见 F-05 |
| ⑧ 迁移幂等 | ✅ 全 `IF NOT EXISTS`，重复执行 0 报错 | `ad_push_queue_test.go:57` 有幂等断言 |
| ⑨ 错误传播 | ✅ 不吞错，原样上抛 | `IsADSyncEnabled` 的查询错误也上抛（没有"兜底 false"） |
| ⑩ SQL 注入 | ✅ 无 | `clause.Expr` 是常量字面量，无字符串拼接 |

真正的问题集中在**跨模块语义缺口**和**部署/运维失效模式**上，共 4 条 WARNING（高）、4 条 WARNING（中）、6 条 INFO。没有发现会导致当前实现直接错误的 BLOCKER——所有高危项都指向 Phase 58 排水器或部署路径，属于"上线前必须收敛"而非"当前已坏"。

---

## 二、Findings

### F-01 `EnqueueDisable` 不取消同实体待推送的 `update_user` 行 → 删除用户后仍可能把属性推回 AD（高）

**位置**：`addomain/ad_push_queue_service.go:199-245`（`EnqueueDisable`）+ `models/ad_push_queue.go:25-29`（action_type 维度）

**问题**：部分唯一索引是 `(entity_type, entity_id, action_type)`，`action_type` 在键里，所以同一用户可以**同时**存在一行 `pending update_user` 和一行 `pending disable`。典型时序：

1. 管理员编辑用户 nickname → `pending update_user`（A 行）
2. 管理员删除该用户 → `pending disable`（B 行）
3. Phase 58 排水器按 `created_at` 升序处理：先 disable 再 update_user（A 行在 B 之前创建时就是这个顺序）

按 D-05 的设计（"排水闭包重读 DB 后全量推送"），A 行推送时会**重读 `sys_user`**——而该行已被软删，GORM 默认软删作用域会把它过滤掉。排水器要么：

- 拿不到用户 → 判失败重试 → 永远推不出去（僵尸行），或
- 拿到零值 → **把 nickname/email/phone/dept 清空写进 AD 账号**（破坏性）。

**建议**：`EnqueueDisable` 在同一事务内把该实体所有 `status IN ('pending','failed') AND action_type='update_user'` 的行转终态（`dead` 或新增 `superseded` 状态并写 `last_error='superseded by disable'`），让"删除"在队列层面对"编辑"形成显式压制。这是队列语义的一部分，不应推给 Phase 58 兜底。

```go
// EnqueueDisable 内，批量 upsert 之前
if err := tx.Model(&models.ADPushQueue{}).
    Where("entity_type = ? AND entity_id IN ? AND action_type = ? AND status IN ?",
        models.ADPushEntityUser, deduped, models.ADPushActionUpdateUser,
        []models.ADPushStatus{models.ADPushStatusPending, models.ADPushStatusFailed}).
    Updates(map[string]interface{}{
        "status":     models.ADPushStatusDead,
        "last_error": "superseded by disable",
        "updated_at": now,
    }).Error; err != nil {
    return err
}
```

---

### F-02 `Update` 不入队 disable / enable：系统停用用户后 AD 账号仍然可登录（高，安全相关）

**位置**：`system/user_service.go:170`（`user.Status = req.Status`）与 `:465-467`（diff 明确排除 status）

**问题**：模型注释把双向推送排除在 v1.21 之外，这个决策本身合理。但要注意 **停用有两个入口**：

- `UpdateStatus()`（`:625`）——模型注释已覆盖；
- **`Update()`（编辑表单里带 status 字段）** —— 前端用户管理表单保存时会带上 status，此时 `0→1` 的停用**既不入队 disable，也没有任何 AD 侧动作**。结果是系统里用户已停用，AD 账号仍是启用态，域内仍可登录——这是**权限状态与域账号状态的分叉**，属于安全语义缺口，而非单纯的"功能没做"。

**建议**：至少在 `EnqueueUpdateIfChanged` 里显式处理 status 跃迁（`oldUser.Status != req.Status && req.Status == disabled` → 额外入队 `disable`），或在 Phase 57-03 的验收标准里**显式记录该缺口并挂到 Phase 59 登录守卫**，避免"以为已经处理"。当前代码对此毫无痕迹，未来读代码的人会误以为 status 已被考虑。

---

### F-03 `SKIP_AUTOMIGRATE=true` 旁路下队列表不存在 → 用户编辑/删除整体 500 且回滚（高，部署失效模式）

**位置**：`core/db/database.go:283-295`（PG 旁路 → 只跑 `BootstrapMissingTables`）+ `core/db/database.go:1055` / `1088`（`Migrate215` 只在 `AutoMigrate()` 里被调用）

**问题**：`core.go:286` 规定 `SKIP_AUTOMIGRATE=true && postgres` 时完全跳过 `AutoMigrate()`，改跑 `BootstrapMissingTables()`——而后者只补建 `sys_api_keys` / `sys_api_keys_usage_logs`，**不建 `ad_push_queue`**，`Migrate215` 在这条路径上一次都不会执行。

此时 `IsADSyncEnabled` 查到启用配置后，`INSERT INTO ad_push_queue` 报 `no such table` → error 上抛 → 用户事务回滚 → **用户管理的新增/编辑/删除全部失败**（D-02 第二层把队列写失败当致命是设计意图，但这里的失败原因是"表压根没建"这种部署态问题，不该由业务路径承担）。

**建议**：在 `BootstrapMissingTables()` 里补一段（model 派生 + 显式索引，与 `Migrate215` 同源）：

```go
if !d.DB.Migrator().HasTable(&models.ADPushQueue{}) {
    if err := d.DB.Migrator().CreateTable(&models.ADPushQueue{}); err != nil { ... }
}
// 部分唯一索引不能由 model 派生，必须显式 DDL（否则合并不生效 → 42P10）
```

---

### F-04 `Migrate215` 在 PG 上是"非阻断"执行，而它是 ON CONFLICT 的硬前提（高）

**位置**：`core/db/database.go:1055-1057`（`applogger.Errorf(...)` 不返回）+ `migration_215:77-82`

**问题**：PG 分支下 AutoMigrate 先建表（`database.go:955`，模型无 uniqueIndex tag → 只建表**不建部分唯一索引**），随后 `Migrate215` 才建索引。若该语句失败（权限不足、并发 DDL、索引已存在但定义不同等），启动只打一行 ERROR 就继续。运行期第一次用户编辑就会命中：

```
ERROR: there is no unique or exclusion constraint matching the ON CONFLICT specification (42P10)
```

且因为入队失败会回滚业务事务，**结果是用户管理全线不可用**，而根因日志在几小时前的启动日志里。

**建议**：(a) `Migrate215` 的索引创建失败改为**阻断启动**（返回 error 或 panic），与"队列是耐久性保证"的红线一致；(b) 或在启动后加一次断言（`SELECT to_regclass('uq_ad_push_queue_active') IS NOT NULL`），缺失即 fail-fast。

---

### F-05 去重测试在 sqlite 上是假断言：删掉去重逻辑该用例依然通过（中，测试有效性）

**位置**：`ad_push_queue_test.go:363-373`（`TestPhase57_EnqueueDisable_DedupesDuplicateIDs`）

**问题**：我实测验证（临时探针，已删除）——**sqlite 对同一条 INSERT 语句内重复命中的 UPSERT 行不会报错**，会逐行处理并合并：

```
batch dup err=<nil> rows=1
```

也就是说，即使把 `EnqueueDisable` 的去重循环整个删掉，这条用例在 sqlite 上仍然得到 `countDisableRows == 2` 而变绿。它唯一能防住的 PG 专属故障（`ON CONFLICT DO UPDATE command cannot affect row a second time`）在 sqlite 上永远不会发生 → **该用例的 RED 状态不会被真正触发**，违反"每个测试都必须能红"的要求。

**建议**：
1. 保留该用例但改注释为"sqlite 行为记录"，并在 `EnqueueDisable` 抽出一个可单测的纯函数 `dedupeIDs([]string) []string`，对**去重逻辑本身**做断言（这才是能红的断言）；
2. 若要覆盖 PG 语义，需在 PG 测试环境跑（当前仓库无 PG 测试基座，属可接受缺口，但应写进 Phase 58 的验证清单）。

---

### F-06 diff 语义（nil = 无变更）与写语义（nil = 清空）不一致：省略字段会静默清库字段却不入队（中）

**位置**：`addomain/ad_push_queue_service.go:80-89`（`adFieldDiffers`）vs `system/user_service.go:165-182`（`tx.Save(&user)`）

**问题**：`Update` 用 `Save` 全字段覆盖，`req.Nickname == nil` 会把 `sys_user.nickname` **写成 NULL**（`Save` 写零值）；而 `adFieldDiffers` 把 nil 解释为"无变更 → 不入队"。handler 是 `ShouldBindJSON` 直绑（`user_handler.go:177`，无 `binding:"required"`），所以任何只带部分字段的请求（第三方脚本、curl、前端漏传）都会：

- 静默清空 DB 里的 nickname/email/phone/dept_id/dept_name；
- **不产生任何队列行** → AD 侧永远保留旧值，系统与域账号进一步分叉。

这是继承自旧代码的行为，但 Phase 57 把它**升级成了契约级不一致**（diff 的文档注释明确写"req 对应字段为 nil → 该字段视为无变更"）。

**建议**：二选一并写进代码注释 —— (a) 让 diff 与写路径一致：nil 视作 `""`（即"清空也是变更"），入队由排水器重读 DB 决定最终值；(b) 或在 handler 层对四字段加 `binding:"required"`。当前状态下这个契约是隐式的。

---

### F-07 `BatchDelete` 不校验 id 是否存在 / 是否已被软删 → 产生孤儿 disable 行（中）

**位置**：`system/user_service.go:587-622` + `ad_push_queue_service.go:220-243`

**问题**：`BatchDelete` 直接把请求里的 `ids` 原样交给 `EnqueueDisable`（内部只过滤空串和去重），不与 `tx.Where("id IN ?", ids).Delete()` 实际影响的行数对齐。因此：

- 传入不存在的 UUID → 软删 0 行，但仍入队一条 `disable`，排水器必然推空；
- 传入**已软删**的用户 → 重复 disable（幂等，尚可）；
- 任意调用方可无限刷入队行（无存在性校验、无速率限制）。

**建议**：在事务内先 `tx.Model(&models.User{}).Where("id IN ?", ids).Pluck("id", &existing)`（软删作用域天然排除已删行），只对 `existing` 入队；这同时把"孤儿行"从根上掐掉，也让 `Delete`/`BatchDelete` 的入队数量与软删数量一致（可作为断言口径）。

---

### F-08 遗留即时推送路径仍在，用户编辑会"双推"且存在乱序风险（中，跨模块）

**位置**：`api/v1/system/user_handler.go:194-243`（`buildADSyncMap` → goroutine `SyncUserUpdateToAD`）

**问题**：handler 在 `service.Update` 成功后**立即**起 goroutine 直连 AD 推属性（带重试与 in-flight 去重）。Phase 57 落地后，同一次编辑会同时产生：① 遗留 goroutine 直推；② 队列行由 Phase 58 排水器再推一次。两条路径无序、无去重，可能出现"遗留路径用旧值后写、队列路径用新值先写"的乱序，且两者都在 AD 侧做 Modify。

本 phase 有意不改 handler（范围说明已确认），但**这个重叠必须写进 Phase 58 的验收清单**，否则排水器上线当天会出现难以复现的"AD 值偶尔回退"。

**建议**：在 `ad_push_queue_service.go` 顶部或 57-03 的 SUMMARY 里加一条显式 TODO：`Phase 58 上线时必须同步下线 user_handler.go:194 的直推路径（或改为仅在队列不可用时降级直推）`。

---

### F-09 队列表不携带目标域标识，多 AD 配置下排水器无解（中，Phase 58 输入不足）

**位置**：`models/ad_push_queue.go:53-68`（无 `ad_config_id`）+ `ad_push_queue_service.go:39-47`（`IsADSyncEnabled` 只判存在性）

**问题**：`sys_ad_config` 允许存在多行（`ADConfig` 是带 `config_name` 的列表模型，无单行约束），`IsADSyncEnabled` 用的是 `Count(&cnt) > 0`。队列行只有 `entity_id`，排水器必须自己反推目标域：先读 `sys_user.ad_dn`，再按 DN 后缀匹配某个 config。多域场景下这套反推一旦错配，就是**把 A 域用户的属性写进 B 域同名账号**。

**建议**：要么在队列表加 `ad_config_id` 列（Phase 58 前置迁移，越晚加越贵），要么在 Phase 58 的设计文档里显式写明"按 ad_dn 后缀匹配 config，匹配不到即置 dead 不推"。当前 Phase 57 的注释只提到"按 entity_id → sys_user.ad_dn 路由"，没有覆盖"ad_dn 匹配不上 config"这一分支。

---

### F-10 `BatchDelete` / `Delete` 的回滚路径缺测试（中，测试覆盖缺口）

**位置**：`ad_push_queue_test.go:477-497` 只覆盖了 `Update` 的回滚

`TestPhase57_UserService_Update_RollsBackWhenQueueInsertFails` 是好用例（DROP 表 → 断言 nickname 未写入 → 能红）。但 `Delete` / `BatchDelete` 的"队列表不可写 → 软删必须回滚"没有对应用例。这两处的失败模式与 Update 不同：`Delete` 先删 `sys_user_role`/`sys_user_post` 再删用户再入队，回滚要验证的是**关联表与用户行的 deleted_at 都没留下痕迹**。

**建议**：补两个用例，DROP 表后调用 `svc.Delete` / `svc.BatchDelete`，断言 `deleted_at IS NULL` 且 `sys_user_role` 行数不变。

---

### F-11 多次编辑会把 `attempts` 归零，毒丸行永远进不了 dead（中，Phase 58 契约缺口）

**位置**：`ad_push_queue_service.go:129-137`（`"attempts": 0` + 清空退避字段）

**问题**：合并语义是"以最新编辑为准"，但**对同一个持续失败的行，每次新编辑都会把 attempts 清零**，Phase 58 若按 attempts 判定 `dead`，一个"每次编辑后都失败一次"的属性（如 AD 侧 nickname 策略拒绝）会永远停在 pending/failed 循环里。

**建议**：要么在 Phase 58 的 dead 判定里引入独立的 `dead_after_first_failure_for_action` 标记（不依赖 attempts），要么明确记录"编辑即续命"是有意为之并在注释里写清后果。当前注释只说了"让 Phase 58 排水器重新拾取"，没提这个副作用。

---

### F-12 UTC 约定不一致（低）

`database.go` 两个连接的 `NowFunc` 都是 `time.Now().UTC()`，而 `models/ad_push_queue.go:91` 的 `BeforeUpdate` 与 `ad_push_queue_service.go:150/230` 的 `now := time.Now()` 都是**本地时区**。PG 侧列是 `timestamptz`（按会话时区归一化存储，读出来是正确的 instant），功能上无碍；但与项目"全项目 UTC 一致性"的既定约定不一致，且排障时看 Go 侧日志与 DB 侧时间戳会有偏移错觉。建议统一 `.UTC()`。

---

### F-13 `oldUserSnapshot := user` 的注释与实际语义不符，且安全性依赖"没人原地改指针指向的值"（低）

`user_service.go:157-160` 注释写"user 是指针，闭包内 user.Nickname = req.Nickname 等是重新赋值指针"，但 `user` 是**值类型** `models.User`。当前之所以安全，是因为四字段都是 `*string` 且闭包内只做**指针重绑定**、从不 `*user.Nickname = ...` 原地写。一旦后续有人改成原地改（很常见的写法），`oldUserSnapshot` 与 `user` 会共享同一字符串 → diff 恒为 false → **入队静默失效，且没有任何测试会红**（现有测试都是先构造独立 `old` 值，不走 `Update` 的快照路径）。

**建议**：改注释为事实描述，并显式加一句不变量：`// 不变量:oldUserSnapshot 的四个 *string 指针不得被原地解引用改写,否则 diff 恒 false`；或改用 `user.Select("nickname","email","phone","dept_id")` 做一次只读查询生成快照（不受别名影响，但多一次 RTT，需权衡）。

---

### F-14 model 与迁移 DDL 的 `status` 默认值不一致（低）

`migration_215:38` 写 `status varchar(20) NOT NULL DEFAULT 'pending'`，而模型 `Status ADPushStatus gorm:"size:20;not null"` 无 `default` tag。由于 AutoMigrate 先跑、迁移后跑，**全新部署实际建出来的是模型版本（无 DEFAULT）**，迁移里的 DEFAULT 永远不生效。应用层靠 `BeforeCreate` 兜底，所以不致命——但两个事实源不一致，未来若有人写裸 SQL 排水器 UPDATE，在两种建表路径下的行为会不同。建议模型加 `default:'pending'` 与 DDL 对齐（或在迁移注释里写明"该 DEFAULT 仅对已存在表的手工补建有效"）。

---

### F-15 三处 sqlite fixture 靠"没有 AD 配置行"走降级分支（低，测试可读性）

`user_cache_impl_test.go:154-165` 与 `user_handler_test.go:157-172` 都新建了 `sys_ad_config` 但**不插数据**，因此这些套件里 disable 入队永远走 D-02 降级分支（`WARN + return nil`）。这让 fixture 改动看起来像"给删除用例打了补丁"，实际并没有验证入队。HYGIENE-06 的目的是"别让这些用例因报表不存在而转红"，这个目的达到了；但建议在 fixture 注释里补一句"此处故意不 seed 配置行，删除用例验证的是放行路径"，否则后来者会误以为这些套件也覆盖了入队。

---

### F-16 `TestPhase57_Migration215_BareInsertWithoutID` 不断言任何契约（低）

该用例只 `t.Logf`，无 assert。作为"行为记录"是诚实的（注释也写明了非契约断言），但它长期占位且每次跑都打印噪音。建议要么加 `t.Skip` + 移到文档，要么把它升级为契约断言（`require.Error` on PG / 明确 sqlite 容忍），否则它对回归没有保护力。

---

## 三、给 Phase 58 的输入清单（本次审查顺带产出）

1. 排水器读 `sys_user` 必须用 `Unscoped()`——软删用户的 `ad_dn` 仍在行内，默认作用域会读不到（F-01）。
2. `pushed_at` / `failed_at` 目前**没有任何写入方**，终态转换与这两列的一致性无人守护。
3. `next_retry_at` 为 NULL 的行是否参与拾取，需要明确定义（当前索引是 `(status, next_retry_at)`）。
4. 处理顺序必须考虑 `update_user` 与 `disable` 的相对次序，或按 F-01 的建议让 disable 在入队时就压制 update_user。
5. `user_handler.go:194` 的遗留直推路径需要同步下线或降级（F-08）。

---

## 四、结论

- **可合并**：F-01 ~ F-04 建议在本 phase 内修掉（都是小改动：压制 update_user / BootstrapMissingTables 补建 / 迁移失败 fail-fast / status 跃迁决策落文）。
- **可延后**：F-05 ~ F-11 属 Phase 58 的输入契约，不阻塞本 phase，但 F-09（ad_config_id）越早决策越便宜。
- **无 BLOCKER**：核心 SQL 构造、事务边界、模型/DDL 对齐、回滚传播四项高风险点均验证通过，全量测试绿。
