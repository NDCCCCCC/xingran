# `sys_rpa_schedules` 与 `sys_job` 的关系裁定：单表替代 / 双写 / 派生镜像判据

- **文档编号**：261127-rpa-schedule-vs-sysjob
- **波次**：v1.56 W7.2（任务 C —— 文档子任务）
- **日期**：2026-11-27
- **触发问题**：
  > `sys_rpa_schedules` 调用 `sys_job` 写，能解决双写问题吗？`sys_rpa_schedules` 能否直接使用 `sys_job` 代替？
- **交付性质**：**纯文档裁定**。本次不改任何 Go / TS 生产代码，不新增任何代码文件。
- **口径声明**：以下三条结论由用户/负责人给定，本文档**不改动结论**，只补充字段级证据、代码位置与推理链条。

---

## 0. 结论速览

| # | 结论 | 一句话 |
|---|------|--------|
| 1 | **不能单表替代** | `sys_job` 与 `sys_rpa_schedules` 承载的是**两类互不包含的语义**（业务语义 vs 执行语义），任一方向的单表合并都会丢字段或污染表。 |
| 2 | **「让 `sys_rpa_schedules` 调 `sys_job` 写」不解决双写** | 那只把双写**换了个方向**（业务表推执行表 → 执行表推业务表），双写本身还在，一致性问题原样保留。真正的解是**单写 + 对账**。 |
| 3 | **判据** | 一张表是「派生镜像」的**充要条件**是：**清空它 + 跑一次同步，能完全恢复**。`sys_job` 的 RPA 组满足 → 它**可以是**派生表；`sys_rpa_schedules` 不满足 → 它**必须是**真相源。 |

---

## 1. 背景：双写是怎么形成的

### 1.1 两张表各自是什么

| 表 | GORM 模型 | 定位 | 语义域 |
|----|-----------|------|--------|
| `sys_rpa_schedules` | `internal/models/rpa/schedule.go` → `rpamodels.Schedule` | **RPA 定时调度的业务真相源** | 业务语义：这个调度关联哪个 RPA 任务、什么时区、什么起止日期、跑过几次、上次什么时候跑的、备注是什么 |
| `sys_job` | `internal/models/log.go` → `models.Job` | **通用调度执行表** | 执行语义：怎么挂进 robfig/cron、错过窗口怎么办、允不允许并发、上次/下次执行时间 |

`sys_job` 是**全系统共用**的通用调度表，被 8 个非 RPA 分组复用（DEFAULT / NETWORK / notice / PERIODIC_WORKORDER / reconciliation / RECONCILIATION / SYSTEM 等；`job_group` 在 API 侧是自由文本字段，见 `internal/api/v1/scheduler/job_handler.go:321`、`internal/services/scheduler/job_service.go:141`，运维可自行填写）。RPA 是第 9 个分组。

### 1.2 镜像行是谁写的、怎么关联的

RPA 调度在 `sys_job` 里的镜像行由 `internal/scheduler/rpa_schedule_tasks.go` 的 `RPAScheduleJobSyncer` 维护，约定固定：

```
invoke_target = "rpa_schedule:<scheduleID>"      // 常量 rpaScheduleInvokeTarget = "rpa_schedule"
job_group     = "RPA"                            // 常量 rpaScheduleJobGroup
remark        = "rpa_schedule_id=<scheduleID>（…）" // 常量 rpaScheduleIDMarker —— 幂等查找键
```

三处关键实现：

- **幂等查找键**（`findJob`，`internal/scheduler/rpa_schedule_tasks.go:162-175`）：
  按 `remark LIKE '%rpa_schedule_id=<id>%'` 反查，而不是按 `invoke_target` 查 —— 因为 `invoke_target` 还带着 `rpa_schedule:` 前缀，而 `remark` 里的裸 id 才是稳定标记。
- **cron 位数归一**（`NormalizeRPACronExpr`，`internal/scheduler/rpa_schedule_tasks.go:62-74`）：
  调度引擎是 `cron.WithSeconds()`（6 位：秒 分 时 日 月 周），而用户习惯写 5 位（分 时 日 月 周）。本层做归一：5 位 → 前置 `"0 "`（即每分钟第 0 秒触发），6 位原样保留。**归一后的表达式写进 `sys_job`**。
  → 因此 `sys_rpa_schedules.cron_expr`（5/6 位原文）与 `sys_job.cron_expression`（归一后）**是同一语义的两个副本**。
- **镜像行的创建**（`createJob`，`internal/scheduler/rpa_schedule_tasks.go:319-342`）：
  固定写入 `MisfirePolicy: models.MisfirePolicyExecuteOnce`、`Concurrent: false`、`Status: models.JobStatusNormal` —— 即**执行策略是镜像方的默认值，不是业务表的字段**。

于是形成**双写**：同一个「调度」既是 `sys_rpa_schedules` 的一行，也是 `sys_job` 的一行。

### 1.3 调度器只认 `sys_job`

`internal/scheduler/cron.go` 的 `Scheduler.Start()`（`:225-251`）加载任务的口径是：

```go
// internal/scheduler/cron.go:233-237
var jobs []models.Job
if err := s.db.Where("status = ?", models.JobStatusNormal).Find(&jobs).Error; err != nil {
    return fmt.Errorf("加载任务失败: %w", err)
}
```

**只按 `status` 过滤，不区分 `job_group`** —— 全部启用 job 一律挂进 robfig/cron。这正是「派生表里的脏数据会直接变成生产故障」的放大器：一条 `job_group='RPA'` 的孤儿 job 照样被加载，到点调用 `rpa_schedule:<已删 id>` → handler 里 `TriggerNow` 查不到 schedule → 报错刷日志。

RPA 调度到点后的执行链路：`RegisterRPAScheduleTasks` 注册的 handler（`internal/scheduler/rpa_schedule_tasks.go:369-396`）→ `svc.TriggerNow(ctx, scheduleID)` → `TaskService.Execute`（`trigger_type=schedule`）。

### 1.4 W7.2 任务 A 已落地的「对账（prune）」机制

这是本文档结论 2 的事实基础，**已落地在当前工作树**：

| 机制 | 位置 | 作用 |
|------|------|------|
| `PruneOrphanJobs` | `internal/scheduler/rpa_schedule_tasks.go:204-238` | 从派生表反向回看业务表，摘除孤儿 job |
| `FindOrphanJobs` | `internal/scheduler/rpa_schedule_tasks.go:244-250` | 只读扫描（dry-run 报告），与真实清理**共用同一份判定实现** |
| `findOrphanJobs` | `internal/scheduler/rpa_schedule_tasks.go:255-294` | 扫 `job_group='RPA'` 且未软删的 job，从 `invoke_target` 解析 `scheduleID` |
| `classifySchedule` | `internal/scheduler/rpa_schedule_tasks.go:301-316` | 用 `Unscoped()` 查 schedule（含软删行），判定三种孤儿成因 |
| `findJob` | `internal/scheduler/rpa_schedule_tasks.go:162-175` | 幂等查找键（`remark` 标记） |
| `RemoveScheduleJob` | `internal/scheduler/rpa_schedule_tasks.go:144-160` | 摘除指定 schedule 对应的 job（幂等） |
| `SyncEnabledJobs` | `internal/services/rpa/schedule_service.go:530-564` | **两步**：正向 ensure + 反向对账 |
| `Delete` | `internal/services/rpa/schedule_service.go:353-368` | **先摘 job，成功后才软删 schedule** |

`classifySchedule` 的三种孤儿判定（这是「业务语义已不可逆地消失」的完整定义）：

```
不存在（物理删除）    → 孤儿
已软删除            → 孤儿
status != 0（停用） → 孤儿
存在且启用          → 保留
```

> 关键点：查询用 `Unscoped()`（**含软删行**）。软删正是孤儿的主要成因之一；用默认查询虽然结果同样是「不存在」，但拿不到 `deleted_at`，无法区分「物理删」与「软删」两种原因（运维审计需要这个区分，见 `internal/scheduler/rpa_schedule_tasks.go:296-300` 的注释）。

`SyncEnabledJobs` 的两步结构（`internal/services/rpa/schedule_service.go:539-561`）：

```go
// 第一步：正向同步（业务表 → 派生表）
for i := range all {
    sched := all[i]
    if sched.Status != ScheduleStatusEnabled {
        s.jobSyncer.RemoveScheduleJob(ctx, sched.ID)   // 停用 → 摘除
        continue
    }
    s.jobSyncer.EnsureScheduleJob(ctx, sched.ID, sched.Name, sched.CronExpr)
}

// 第二步：反向对账（派生表 → 业务表），摘除孤儿 job
pruneResult, pruneErr := s.jobSyncer.PruneOrphanJobs(ctx)
```

容错口径（第 1 步错误照旧返回；第 2 步对账失败**不阻断启动**，只记 ERROR 日志；第 1 步即使遇错也**跑完再返回**，保证第 2 步一定有机会执行 —— 否则一条脏 schedule 会让本次启动完全跳过对账）。

---

## 2. 结论 1：不能单表替代

### 2.1 字段证据（逐字段核对模型定义）

`internal/models/rpa/schedule.go` 的 `rpamodels.Schedule`（`:9-22`）：

| 字段 | GORM 列 | 说明 |
|------|---------|------|
| `ID` / `CreatedAt` / `UpdatedAt` / `DeletedAt` / … | — | 继承 `models.BaseModel`（`internal/models/base.go:11-19`） |
| `Name` | `name` | 调度名称 |
| **`TaskID`** | `task_id` | **关联哪个 RPA 任务 —— 业务必需** |
| `CronExpr` | `cron_expr` | cron 表达式（5/6 位原文） |
| **`Timezone`** | `timezone` | 时区（默认 `Asia/Shanghai`） |
| **`StartDate`** | `start_date` | 起始日期（可空） |
| **`EndDate`** | `end_date` | 截止日期（可空） |
| `Status` | `status` | 0=启用 1=停用 |
| `NextRunTime` | `next_run_time` | 下次执行时间 |
| **`LastRunTime`** | `last_run_time` | 上次执行时间 |
| **`RunCount`** | `run_count` | 已运行次数 |
| **`Description`** | `description` | 描述（业务说明） |

`internal/models/log.go` 的 `models.Job`（`:76-88`）：

| 字段 | GORM 列 | 说明 |
|------|---------|------|
| `JobName` | `job_name` | 任务名 |
| `JobGroup` | `job_group` | 分组（RPA 是 9 个分组之一） |
| `InvokeTarget` | `invoke_target` | 调用目标（`rpa_schedule:<id>`） |
| `CronExpression` | `cron_expression` | cron 表达式（**归一后 6 位**） |
| **`MisfirePolicy`** | `misfire_policy` | 错过窗口策略：默认/立即/执行一次/放弃 |
| **`Concurrent`** | `concurrent` | 是否允许并发 |
| `Status` | `status` | 0=正常 1=暂停 |
| `NextRunTime` | `next_run_time` | 下次执行时间 |
| **`PrevRunTime`** | `prev_run_time` | 上次执行时间 |
| `Remark` | `remark` | 备注（RPA 用它存 `rpa_schedule_id=` 标记） |

**差集汇总**：

| `sys_rpa_schedules` 有，`sys_job` 无 | `sys_job` 有，`sys_rpa_schedules` 无 |
|---|---|
| `TaskID`（**RPA 业务必需关联**） | `MisfirePolicy` |
| `Timezone` | `Concurrent` |
| `StartDate` | `PrevRunTime` |
| `EndDate` | |
| `RunCount` | |
| `LastRunTime` | |
| `Description` | |

两者共有的：`ID`/`CreatedAt`/`UpdatedAt`/`DeletedAt`、`Status`（取值域相同，0=启用/正常，1=停用/暂停）、`NextRunTime`。
语义重复但实现不同：`sys_rpa_schedules.cron_expr`（5/6 位原文）vs `sys_job.cron_expression`（归一后 6 位）。

**这张表说明了一件事**：`sys_job` 里没有任何一个字段能承载「这个调度要跑哪个 RPA 任务」。`TaskID` 只存在于 `sys_rpa_schedules`。

### 2.2 为什么「给 `sys_job` 加 RPA 专属业务列」不可取

1. **污染通用表**：`sys_job` 是全系统共用的通用调度表，被 8 个非 RPA 分组复用（DEFAULT / NETWORK / notice / PERIODIC_WORKORDER / reconciliation / RECONCILIATION / SYSTEM 等）。塞进 `task_id` / `timezone` / `start_date` / `end_date` / `run_count` / `description` 这些**只有 RPA 用得上**的列，另外 8 个分组全部要面对一堆恒为 NULL 的哑列 —— 表的语义从「通用调度执行表」退化成「调度表 + RPA 业务列的混合体」。
2. **破坏分组隔离**：现有代码已经吃过一次分组隔离的亏 —— `reconciliation_crons.go:21-26` 记录了 `job_group` 大小写分裂（`"RECONCILIATION"` 大写 vs `"reconciliation"` 小写）导致自愈逻辑必须专门做 `UPDATE` 统一。再往通用表加 RPA 专属列，会把「哪些 job 是 RPA 派生镜像」这一判断从 `job_group='RPA'` **退化为「哪些 job 的 `task_id` 非空」** —— 分组语义一旦被列语义污染，派生镜像的清理边界就再也切不干净。
3. **列语义会随 RPA 需求增长**：今天 RPA 只要 7 个字段，明天要加「重试次数」「超时」「失败告警」「优先级」——通用表的列会无限膨胀，且每一列对另外 8 个分组都是死重量。
4. **迁移与回滚成本**：通用表的 schema 变更要走全系统迁移，影响面是全部 8 个分组的作业，而收益只落在 RPA 一个分组。

### 2.3 为什么「把 RPA 业务塞进 `sys_job.remark` 的 JSON」不可取

1. **无法索引**：`remark` 是 `size:500` 的普通文本列（`internal/models/log.go:87`），没有 JSON 类型、没有 GIN 索引。RPA 镜像行的 JSON 里若含 `task_id`，那么「查某 RPA 任务的所有调度」只能写成 `remark LIKE '%"task_id":"<uuid>"%'` —— 全表扫描 + 靠字符串格式匹配，**格式一改（比如加空格）查询就失效**。
2. **无法查询**：列表接口要拿 `task_id` 做筛选 / 排序 / 关联，只能在应用层把 `remark` 取出来 `json.Unmarshal` 再判。RPA 调度列表接口（`internal/api/v1/rpa/schedule_handler.go:45-56`，`POST /rpa/schedules/list`）已经在按 `task_id` 做 `Where` 筛选（`internal/services/rpa/schedule_service.go:401-403`）—— 换成 JSON 方案后，**筛选条件必须从 SQL 下推到应用层**，分页 `total` 与 `limit` 也会跟着失真。
3. **前端必须解析 JSON**：`rpamodels.Schedule` 的 `TaskID` 带 `json:"taskId"` 标签，前端列表直接取 `taskId` 即可；若数据搬到 `remark` JSON，前端列表必须先解析 remark JSON 才能拿到 `taskId`，多一层脆弱的字符串协议。
4. **已有先例证明这条路走不通**：RPA 镜像行的幂等查找键**已经是**塞在 `remark` 里的（`rpa_schedule_id=<id>`）。而这个设计立刻暴露了两个成本：
   - 查找用的是 `remark LIKE '%...%'`（`findJob`，`internal/scheduler/rpa_schedule_tasks.go:165-167`），而不是等值索引；
   - `remark` 被人工改过会导致 `findJob` 匹配不到 —— `PruneOrphanJobs` 因此必须额外加一段「按 `job.ID` 兜底摘除」（`internal/scheduler/rpa_schedule_tasks.go:221-229`），并且要专门打一条 WARN 日志。
   **RPA 业务塞 `remark` JSON，是把同样的问题放大到所有业务字段上。**

### 2.4 反向也不行：不能把执行侧字段塞进 `sys_rpa_schedules`

上面三条否掉了「`sys_job` → 业务字段」的合并。反方向同样不成立：

- `MisfirePolicy` / `Concurrent` 属于**调度引擎语义**（错过窗口补不补、允不允许并发），不是业务语义。RPA 业务并不关心「这条调度错过窗口后要不要补跑」—— 这是引擎该管的事。
- 把它们塞进 `sys_rpa_schedules`，会让**业务表被执行策略污染**：业务表凭空多出两个只有「镜像方」才读的列，且这两个列的值没有独立来源（`createJob` 里固定写 `MisfirePolicyExecuteOnce` / `Concurrent: false`，见 `internal/scheduler/rpa_schedule_tasks.go:332-333`）—— 业务表里的这两列会与镜像行脱节，变成**第二处双写**，问题从 1 处变成 2 处。
- 同理 `PrevRunTime` 也不该搬进业务表：`sys_rpa_schedules` 已经有 `LastRunTime`（由 `markRun` 回写，`internal/services/rpa/schedule_service.go:466-480`），两者语义重叠但**回写者不同**（引擎写 `prev_run_time`，服务写 `last_run_time`），搬进业务表就是第三处双写。

**结论 1 的一句话**：`sys_job` 与 `sys_rpa_schedules` 承载的是**两类互不包含的语义**。任一方向的单表合并，要么丢字段（要么 `TaskID`、要么 `MisfirePolicy`），要么污染表（通用表塞业务列 / 业务表塞执行策略）。**双表不是冗余，是必要的语义分离。**

---

## 3. 结论 2：「让 `sys_rpa_schedules` 调 `sys_job` 写」不解决双写

### 3.1 核心论点：那只换了方向，双写还在

「`sys_rpa_schedules` 调 `sys_job` 写」= 让**业务表去推执行表**（业务表 → 执行表）。
而现状已经是**执行表侧在维护镜像**（`RPAScheduleJobSyncer` 的 ensure/remove 也在写 `sys_job`）。

两种方向下的数据流完全对称：

```
现状：   业务表 ──(Ensure/Remove, 正向)──▶ 派生表(sys_job RPA 组)
                        │
                        └──(Prune, 反向对账)──◀ ┘   ← 只摘孤儿，不补业务字段

提议：   业务表 ──(直接写 sys_job, 正向)──▶ 派生表(sys_job RPA 组)
                        │
                        └──(反向对账)──◀ ┘
```

**两个方向写的是同一张派生表、同一份字段**，一致性问题的**形态**一模一样：

| 一致性问题 | 「业务表推执行表」能否解决 |
|---|---|
| 派生表被绕过 API 直接改坏 | ❌ 不能。业务表推得再勤，运维直接 UPDATE `sys_job` 一样能改坏派生表 |
| 派生表多出业务表没有的行 | ❌ 不能。反向对账**照旧是唯一的发现手段** —— 与方向无关 |
| 业务表删了、派生表没删 | ❌ 不能。正向同步看不见「已被绕过 API 删掉的行」（GORM 软删自动排除），这正是生产库 10 条孤儿 job 的根因 |
| 两边 cron 表达式口径漂移 | ❌ 不能。归一（`NormalizeRPACronExpr`）是**派生方**的职责，改方向不改变「一份原文、一份归一后」的事实 |
| 一边写成功、一边写失败 | ❌ 不能。除非引入分布式事务/补偿队列（且派生表本来就不需要这么强的保证） |

**结论 2 的第一句话**：双写的本质不是「谁写」，而是「**同一份语义被写了两遍**」。改成「业务表调 `sys_job` 写」只是把双写从「业务表推执行表」换成「业务表还是推执行表」—— 因为**写的方向本来就一直是业务表 → 执行表**（`Create` / `Update` / `Delete` / `SetStatus` 都是先写 `sys_rpa_schedules` 再同步 job，见 `internal/services/rpa/schedule_service.go:180-230` / `:233-336` / `:353-368` / `:430-435`）。换句话说：**现状就是「`sys_rpa_schedules` 调 `sys_job` 写」，它已经在这么做了，而且它没有解决双写。**

### 3.2 真正的解：把双写降级为「单写 + 对账」

既然消灭不了双写，就把它**降级**成一种可管理、可验证的状态：

| 角色 | 表 | 性质 | 规则 |
|------|----|------|------|
| **唯一真相源** | `sys_rpa_schedules` | 业务语义的**唯一权威** | 业务语义的**所有**修改**只能**经由这里发生（API 或受控脚本） |
| **派生镜像** | `sys_job`（`job_group='RPA'` 且 `invoke_target` 前缀 `rpa_schedule:`） | **可随时重建**的投影 | 允许被重建、被丢弃、被重算；**不允许被外部直接写** |

具体四条规则：

1. **唯一真相源 = `sys_rpa_schedules`**。业务语义（`TaskID` / `Timezone` / `StartDate` / `EndDate` / `RunCount` / `LastRunTime` / `Description` / `CronExpr`）**只在这张表**。派生表里出现的任何业务字段都是**投影**，不是权威。

2. **`sys_job` 的 RPA 组 = 可随时重建的派生镜像**。
   - 它**丢了也能从业务表全量重建**（见 §3.3 的可验证性论证）。
   - 它**不需要独立的一致性保证**，只需要「能重建」这一条性质。
   - 这样一来，「双写」从「两个都要对的权威源」降级为「一个权威源 + 一个随时可扔的投影」—— **一致性问题从「不能出错」变成「出错可自动修复」**。

3. **对账（prune）保证派生表不会长出孤儿**。
   这是 W7.2 任务 A 已落地的机制（§1.4）：`PruneOrphanJobs` / `FindOrphanJobs` / `classifySchedule` 从派生表反向回看业务表（`Unscoped()`，含软删行），把「schedule 已物理删 / 已软删 / 已停用」的孤儿 job 摘掉。
   - 没有 prune → 派生表会**单调地**积累垃圾，且永不自愈（生产库实证：10 条孤儿 job 长期存活，每次重启都被挂进 cron）。
   - 有 prune → 派生表的「脏」是**有界且自动收敛**的。

4. **派生表不允许被外部直接写**。
   - 运维改数据**必须走 API**（`internal/api/v1/rpa/schedule_handler.go:240-258` 的 CRUD / 触发 / 启停端点），因为 API 路径内建了「先摘 job 再软删 schedule」的顺序保证（`Delete`，`internal/services/rpa/schedule_service.go:353-368`）。
   - 万一绕过 API 直接改了 `sys_job`（例如改坏 `remark` 标记、改错 cron、删了行），**改完后必须触发一次对账**：
     ```bash
     # 只读报告（dry-run，不改数据）
     go run ./scripts/ops/rpa_orphan_job_prune -dry-run
     # 真清理（走与后端启动时完全相同的代码路径）
     go run ./scripts/ops/rpa_orphan_job_prune
     ```
     该工具**不自带判定逻辑** —— 清理走与后端启动时完全相同的代码路径（`rpav1.ScheduleService.SyncEnabledJobs` → `scheduler.RPAScheduleJobSyncer` 的同一个 prune 实现，见 `scripts/ops/rpa_orphan_job_prune/main.go:13-16`），保证「工具清掉的」与「后端启动会清的」是同一批。

### 3.3 可验证性：派生镜像可以被重建

「派生镜像可以被重建」不是口头承诺，而是**可执行的验证**：

**验证步骤**：

```sql
-- 步骤 1：清空派生表（只删 RPA 派生镜像，不动其它 8 个分组）
DELETE FROM sys_job
 WHERE job_group = 'RPA'
   AND invoke_target LIKE 'rpa_schedule:%';
-- （若字段可空且兼容软删口径，用 UPDATE ... SET deleted_at = now() 亦可）

-- 步骤 2：跑一次同步（等价于后端重启时 RegisterRPAScheduleTasks 的启动流程）
--   go run ./scripts/ops/rpa_orphan_job_prune
--   或直接重启后端（RegisterRPAScheduleTasks → svc.SyncEnabledJobs，internal/scheduler/rpa_schedule_tasks.go:392-395）

-- 步骤 3：验证等价
SELECT
  (SELECT count(*) FROM sys_job
    WHERE job_group = 'RPA' AND deleted_at IS NULL)              AS mirror_rows,
  (SELECT count(*) FROM sys_rpa_schedules
    WHERE status = 0 AND deleted_at IS NULL)                      AS truth_rows;
```

**为什么这一定能恢复**：

1. `SyncEnabledJobs` 的第一步遍历 `sys_rpa_schedules` 的**全部**行（`internal/services/rpa/schedule_service.go:531-534`），对每条启用态 schedule 调 `EnsureScheduleJob`（`:549`）。
2. `EnsureScheduleJob` 是**逐条 ensure** 的幂等实现（`internal/scheduler/rpa_schedule_tasks.go:92-119`）：
   - `findJob` 找不到 → `createJob` 建一条新行（含 `job_group='RPA'`、`invoke_target=rpa_schedule:<id>`、归一后 cron、`MisfirePolicyExecuteOnce`、`Concurrent=false`、`Status=JobStatusNormal`、`remark` 带 `rpa_schedule_id=` 标记）；
   - `findJob` 找到且 cron 一致 → 直接返回，**不重复 AddJob**（否则每次重启都会多挂一条 cron）；
   - `findJob` 找到但 cron 变了 → `rebuildJob`（摘旧 + 建新）。
   → **清空后重建，与「从未建过」走的是同一条 `createJob` 路径**，产出的镜像行与原镜像行在所有字段上等价。
3. 第二步 prune 在清空场景下是**空操作**（没有行可摘），不会误伤。

**结论 2 的第二句话**：真正的解不是「换一个方向写」，而是「**把双写降级为单写 + 对账**」—— 唯一真相源是 `sys_rpa_schedules`，`sys_job` 的 RPA 组退化为**可随时重建的派生镜像**（丢了也能全量重建），对账机制保证派生表不长孤儿，派生表不允许被外部直接写。

---

## 4. 结论 3：判据（供未来评审复用）

### 4.1 判据本身

> **一张表是「派生镜像」的充要条件是：清空它 + 跑一次同步，能完全恢复。**

展开成可操作的两问：

1. **可重建性**：清空这张表后，跑一次同步，**是否能把全部语义恢复回来**？
2. **完备性**：同步逻辑**是否覆盖了业务表的全部语义**（没有字段只存在于派生表）？

两个都成立 → 派生表可以随便扔。
任一不成立 → 这张表**必须**是真相源（或必须补齐缺失的同步逻辑）。

### 4.2 套用到两张表

| | `sys_job` 的 RPA 组 | `sys_rpa_schedules` |
|---|---|---|
| 判据第 1 问：清空 + 跑一次同步能完全恢复？ | ✅ **能**。`EnsureScheduleJob` 逐条 ensure（幂等），`createJob` 重建全部镜像字段 | ❌ **不能**。`TaskID` **只在这里**；`sys_job` 里没有任何字段承载它。`Timezone`/`StartDate`/`EndDate`/`RunCount`/`LastRunTime`/`Description` 同理 |
| 判据第 2 问：同步逻辑覆盖业务表全部语义？ | ✅ 覆盖（cron 归一后写入；其余业务字段不在镜像里，符合「镜像只承载执行语义」的分工） | ✅ 全部业务语义的所在地 |
| **判定** | **可以是**派生表 | **必须是**真相源 |
| **前提条件** | 必须有 W7.2 任务 A 的 prune 机制到位 —— 否则：① 清空后重建会**漏**（正向同步看不见软删/物理删的 schedule，重建结果与原始不一致）；② 旧的孤儿**永远回不来**（没有反向对账，脏数据只增不减） | 无 |

> **prune 是判据成立的前提，不是附属功能**：没有反向对账，派生表就不是「派生镜像」，只是「另一个没人负责的副本」。

### 4.3 未来评审怎么用这条判据

新增任何「派生镜像表」（或者有人提议「这两张表能不能合成一张」）时，**先问这一个问题**：

> **「清空这张表 + 跑一次同步，能完全恢复吗？」**

- **能** → 派生表。可以放心让它是派生的：允许被删、被重建、被清空；一致性交给同步 + 对账；**不允许外部直接写**。
- **不能** → 真相源。这张表里一定存在**别处没有的语义**（本例中是 `TaskID`）；任何把它「并进另一张表」的提议都会导致这部分语义丢失或退化成不可索引的字符串。

配套的两条工程纪律：

1. **派生表的写入路径必须收敛到一条**。本例中是 `SyncEnabledJobs`（正向 ensure + 反向 prune）这一条路径，运维工具 `scripts/ops/rpa_orphan_job_prune` 也**复用同一实现**，不另写一套判定逻辑 —— 保证「工具清掉的」与「后端启动会清的」是同一批。
2. **反向对账（prune）必须与正向同步同时存在**。只有正向同步的派生表 = 单调积累垃圾的表，不满足判据。

---

## 5. 关键代码位置索引

| 文件 | 关键符号 | 行号 | 用途 |
|------|---------|------|------|
| `internal/models/rpa/schedule.go` | `rpamodels.Schedule` / `TableName` | `:9-22` / `:25-27` | 业务真相源的字段定义（表名 `sys_rpa_schedules`） |
| `internal/models/log.go` | `models.Job` | `:76-88` | 通用调度执行表的字段定义（表名 `sys_job`） |
| `internal/models/log.go` | `MisfirePolicy` / `JobStatus` | `:57-73` | 执行策略与状态枚举 |
| `internal/models/base.go` | `models.BaseModel` | `:11-19` | 两张表共有的 id / 时间戳 / `deleted_at` |
| `internal/scheduler/cron.go` | `Scheduler.Start` | `:225-251` | 加载口径：`Where("status = ?", JobStatusNormal).Find(&jobs)`（`:235`）—— **不区分 job_group** |
| `internal/scheduler/rpa_schedule_tasks.go` | 常量 `rpaScheduleInvokeTarget` / `rpaScheduleJobGroup` / `rpaScheduleIDMarker` | `:41-50` | 镜像行的三个约定 |
| `internal/scheduler/rpa_schedule_tasks.go` | `NormalizeRPACronExpr` | `:62-74` | cron 位数归一（5 位 → 前置 `"0 "`） |
| `internal/scheduler/rpa_schedule_tasks.go` | `RPAScheduleJobSyncer` | `:81-90` | 镜像维护器本体 |
| `internal/scheduler/rpa_schedule_tasks.go` | `EnsureScheduleJob` | `:92-119` | **幂等 ensure —— 「清空可重建」的根据** |
| `internal/scheduler/rpa_schedule_tasks.go` | `UpdateScheduleJobCron` | `:121-142` | cron 变更重挂 |
| `internal/scheduler/rpa_schedule_tasks.go` | `RemoveScheduleJob` | `:144-160` | 摘除（幂等） |
| `internal/scheduler/rpa_schedule_tasks.go` | `findJob` | `:162-175` | 幂等查找键（`remark LIKE '%rpa_schedule_id=…%'`） |
| `internal/scheduler/rpa_schedule_tasks.go` | `PruneOrphanJobs` | `:204-238` | 反向对账：摘除孤儿 |
| `internal/scheduler/rpa_schedule_tasks.go` | `FindOrphanJobs` | `:244-250` | 只读扫描（与真实清理共用判定） |
| `internal/scheduler/rpa_schedule_tasks.go` | `findOrphanJobs` | `:255-294` | 扫 `job_group='RPA'` 未软删行 |
| `internal/scheduler/rpa_schedule_tasks.go` | `classifySchedule` | `:301-316` | 三种孤儿成因判定（用 `Unscoped()`） |
| `internal/scheduler/rpa_schedule_tasks.go` | `createJob` / `rebuildJob` | `:319-342` / `:350-359` | 镜像行字段的**唯一**写入点 |
| `internal/scheduler/rpa_schedule_tasks.go` | `RegisterRPAScheduleTasks` | `:369-396` | handler 注册 + 启动时 `SyncEnabledJobs` |
| `internal/services/rpa/schedule_service.go` | `ScheduleJobSyncer` 接口 | `:60-73` | `Ensure`/`Update`/`Remove`/`Prune` 四方法契约 |
| `internal/services/rpa/schedule_service.go` | `ScheduleJobPruneItem` / `ScheduleJobPruneResult` | `:76-95` | 对账结果结构（含 `Reason` 审计字段） |
| `internal/services/rpa/schedule_service.go` | `Create` / `Update` | `:180-230` / `:233-336` | 正向同步的触发点（先写业务表，再同步 job） |
| `internal/services/rpa/schedule_service.go` | `Delete` | `:353-368` | **先摘 job，成功后才软删 schedule**（顺序加固） |
| `internal/services/rpa/schedule_service.go` | `SetStatus` / `TriggerNow` / `markRun` | `:430-435` / `:441-463` / `:466-480` | 停用即摘除；触发；回写 `last_run_time`/`run_count` |
| `internal/services/rpa/schedule_service.go` | `ComputeNextRunTime` | `:485-506` | 时区 + cron → 下次执行时间 |
| `internal/services/rpa/schedule_service.go` | `SyncEnabledJobs` | `:530-564` | **正向 ensure + 反向 prune 两步** |
| `internal/api/v1/rpa/schedule_handler.go` | `SetupScheduleRouter` | `:240-258` | CRUD / 触发 / 启停 API（运维改数据的**唯一**正规入口） |
| `internal/api/v1/rpa/schedule_handler.go` | 列表端点 `POST /rpa/schedules/list` | `:46-62` | 前端列表直接取 `taskId`（`json:"taskId"` 标签） |
| `scripts/ops/rpa_orphan_job_prune/main.go` | `snapshot` / `main` | `:105-126` / `:128-225` | 运维工具：连接 → 报告 → 清理 → 验证；**复用后端同一实现** |
| `internal/scheduler/reconciliation_crons.go` | `reconciliationJobGroup` | `:21-26` | 「`job_group` 大小写分裂」的历史教训（分组隔离不是想当然） |
| `internal/services/scheduler/job_service.go` / `internal/api/v1/scheduler/job_handler.go` | `job_group` 的写入点 | `:141` / `:321` | `job_group` 是**自由文本**字段 —— 分组数会随运维自由增长 |

**测试与运维资产**（W7.2 任务 A 随附，均在当前工作树）：

- `internal/scheduler/rpa_schedule_prune_test.go` —— prune 判定与摘除的用例
- `internal/services/rpa/schedule_delete_order_test.go` —— 「先摘 job 再软删」顺序的用例
- `internal/services/rpa/cleanup158_test.go` —— 调度同步相关回归
- `scripts/ops/rpa_orphan_job_prune/` —— 生产清理工具（`-dry-run` 只读报告 / 默认真清理）

---

## 6. 生产实证数据（W7.2）

生产库（PostgreSQL）实测，工具：`scripts/ops/rpa_orphan_job_prune`。

### 6.1 清理前 → 清理后

| 指标 | 清理前 | 清理后 | 说明 |
|------|--------|--------|------|
| **RPA 分组未删 job 数** | **10** | **0** | `count(sys_job WHERE job_group='RPA' AND deleted_at IS NULL)` |
| **调度器启动将加载的任务数** | **34** | **24** | `count(sys_job WHERE status=0 AND deleted_at IS NULL)` —— 差值 10 正是被摘掉的孤儿 |
| **启用态 schedule 数** | 0 | 0 | `count(sys_rpa_schedules WHERE status=0 AND deleted_at IS NULL)` |
| **孤儿 job 数** | **10** | **0** | 判定口径与清理口径**同一份实现**（`findOrphanJobs`） |

**10 条孤儿的成因分布**（`classifySchedule` 的三种判定）：

| 成因 | 条数 | 判定文案 |
|------|------|---------|
| 调度记录已**物理删除**（不存在） | **8** | `调度记录已不存在（物理删除）` |
| 调度**已停用**（`status != 0`） | **2** | `调度已停用(status=1)` |

> 注：生产库当前**没有启用态的 RPA 调度**（`status=0` 的 schedule 为 0 条），所以那 10 条 RPA 组 job **全部**是孤儿 —— 它们的 schedule 行早已被删或被停用，job 却因为「正向同步看不见已软删/物理删的 schedule」而永久残留。

### 6.2 一致性验证

```
count(sys_job WHERE job_group='RPA' AND deleted_at IS NULL)
  == count(sys_rpa_schedules WHERE status=0 AND deleted_at IS NULL)
  → 0 == 0   ✅ 成立
```

这正是 §4.1 判据在生产上的落地形态：**派生镜像（`sys_job` 的 RPA 组）与真相源（`sys_rpa_schedules` 的启用态行）完全对齐**。清空派生表 + 跑一次 `SyncEnabledJobs`，可从 0 条启用 schedule 重建出 0 条镜像 —— **幂等、可重建、可验证**。

### 6.3 故障放大链（这才是问题的真正代价）

```
绕过 API 删掉 schedule（软删/物理删/停用）
      ↓
正向同步看不见它（GORM 软删自动排除）
      ↓
sys_job 的镜像行仍是 deleted_at IS NULL / status=0
      ↓
Scheduler.Start() 按 status 加载全部 job（不分 job_group）→ 挂进 robfig/cron
      ↓
到点调用 rpa_schedule:<已删 id>
      ↓
TriggerNow 查不到 schedule → 报错刷日志
      ↓
且：没有任何机制会回收它（这正是 10 条孤儿长期存活的原因）
```

W7.2 任务 A 补上的反向对账（prune）把这条链**从「单调恶化」改成「自动收敛」**。

---

## 7. 附：本裁定未覆盖的相邻问题

以下问题在本波次内**未处理**，列出以免被误读为已解决：

1. **RPA 镜像行的 `MisfirePolicy` / `Concurrent` 目前是硬编码默认值**（`MisfirePolicyExecuteOnce` / `false`，`internal/scheduler/rpa_schedule_tasks.go:332-333`），业务侧没有暴露配置入口。这不影响本文档三条结论（它们属于执行语义，本就不该进业务表），但若将来要开放配置，**配置项应落在派生表的镜像行上**（`sys_job` 的 `misfire_policy` / `concurrent`），而不是业务表。
2. **`remark` 被人工改写会导致幂等查找键失效** —— 目前靠 `PruneOrphanJobs` 的「按 `job.ID` 兜底摘除」（`internal/scheduler/rpa_schedule_tasks.go:221-229`）兜住，但 `EnsureScheduleJob` 侧没有对应的兜底（找不到就新建一条，可能产生重复镜像）。这是「派生表不允许被外部直接写」这条纪律存在的原因之一。
3. **`sys_job` 的 `job_group` 是自由文本字段**（`internal/services/scheduler/job_service.go:141`），运维随手填的名字会不断分裂分组。`reconciliation` 大小写分裂已是先例。若要给 RPA 派生镜像做更强的隔离，需要考虑把「是否派生镜像」从「`job_group='RPA'`」升级为更强的标记（例如 `invoke_target` 前缀 —— 但 `findOrphanJobs` 目前已经在解析这个前缀了，`:270`），或对 `job_group` 加枚举约束。
4. **调度器启动加载口径不分 `job_group`**（`internal/scheduler/cron.go:235`）—— 这是 §6.3 故障链的放大器。是否要改成「按分组白名单加载」属于**任务 B（Engine 竞态）**的范围，不在本文档内。

---

*本文档由 W7.2 波次任务 C 产出，仅新增本文件；未改动任何 Go / TS 生产代码，未执行 commit。*
