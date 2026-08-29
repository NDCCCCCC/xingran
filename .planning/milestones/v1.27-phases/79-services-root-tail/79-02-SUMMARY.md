---
plan: 79-02
phase: 79-services-root-tail
executed: 2026-08-28
commits:
  - ad58efb (test: task 1 GenerateSchedule 分支表 + getDutyType/isWeekend 纯函数)
  - 0b5562b (test: task 2 duty_schedule CRUD 族)
  - b1202b9 (test: task 3 duty_pool 全 CRUD + round-trip + statistics)
  - a3f9417 (test: task 4 duty_stats/holiday/config 三小文件)
  - 23e095c (test: task 5 duty_service 门面 23 委托方法连通性)
  - cdce989 (test: task 2 追补 GetList 过期态过滤分支,收口任务 6 前落)
---

# 79-02 Summary — duty 家族 6 文件 0%/<5% → 全部 ≥70%(实际 82.8%–100%)

## 交付

3 个测试文件(1835 行,30 个新测试函数,零生产 .go 改动),与源码同包共置(D-79-06 命名法):

- `internal/services/duty_schedule_service_79_02_test.go`(917 行,**17 个 TestDsc7902_**)。
  helper `newDsc7902`(sqlite t.TempDir 文件库 + AutoMigrate duty 全链 model)/
  `seedPool7902`(池 + 成员行,MemberOrder=下标)/`seedSchedule7902`/`dsc7902LocalToday`
  (本地日期的 UTC 零点,与 `time.Parse("2006-01-02")` 落库形态同源)。
  GenerateSchedule 8 分支独立用例(池缺失/空成员/两个日期解析/类型过滤双向/轮询 12 槽
  × 3 成员恰 4 次/节假日双向/ClearExists 清除 vs 累计翻倍/空结果不插入)+
  getDutyType 三态表驱动(7 例,含「节假日但非休 × 周末」落到周末判断的 fall-through)
  + isWeekend 7 天表;CRUD 族:List 分页默认值/六类过滤/白名单升倒序/非法列回退/过期态
  三态、GetTodayDuty 今天命中+昵称展示格式+空库错误、GetMonthly 按日分组/跨月排除/
  月份归一化、SwapDuty 双方字段对调+历史记录+双侧不存在+自换、ManualDuty 替换语义、
  Delete/BatchDelete 软删计数。
- `internal/services/duty_pool_service_79_02_test.go`(373 行,**7 个 TestDpl7902_**)。
  helper `newDpl7902`/`dpl7902Create`(成员先落 sys_user 过存在性校验)。
  Create 成功(Members round-trip:写→GetByID 独立读回,成员集 + MemberOrder 0..n-1
  + CreatedBy 透传)/重名/成员不存在(事务回滚无半写)/空成员;Update 全字段读回 +
  整段换成员 + Status 指针语义 + 幽灵成员;List 分页/关键字/部门/状态过滤 +
  Preload("Members.User") 证据;GetByID 双分支;Delete 拒删(有排班)/连带清成员/
  空集幂等;Statistics 走 service 全链驱动(启停聚合 + 软删池成员排除),
  口径对照既有 `TestDutyPoolStatistics_NotDerivedFromCurrentPage`(未动,不回归)。
- `internal/services/duty_family_tail_79_02_test.go`(545 行,**6 个测试**:
  1 TestDst7902_ + 2 TestDhd7902_ + 2 TestDcf7902_ + 1 TestDsv7902_)。
  helper `newDtx7902`(一个库装配 stats/holiday/config 三 service)。
  GetMyDutyStats:今日在岗 + 本月/总计计数(fixture oracle 推导,非镜像 SQL)+
  NextDutyDate/PoolName 取最近未来行 + 零记录用户零值结构;Holiday 六方法 CRUD +
  跨两年年份列表 + 空批量 + 日期形态锁定;GetDutyConfig 空表回默认四字段 +
  Update 创建/更新双分支往返(ID/CreatedBy 保留);门面 TestDsv7902_FacadeDelegation
  按 D-79-07 轻量口径:池域 4 方法 round-trip + 排班域生成/列表/删除主干链,
  其余委托以调用形态 + 错误透传覆盖 —— **23 个委托方法全部被触达**
  (`grep -o "facade\.[A-Za-z]*" | sort -u` = 23,验收线 ≥10)。

## Coverage checkpoint(per-file 实测,`go test -count=1 -coverprofile` 全包一次)

| File | 基线(79-RESEARCH §2) | 实测 | 目标 | 结果 |
|---|---|---|---|---|
| duty_schedule_service.go | 0%(174 unc) | **91.4%**(159/174) | ≥70% | ✅ |
| duty_pool_service.go | 4.9%(97 unc) | **83.3%**(85/102) | ≥70% | ✅ |
| duty_stats_service.go | 0%(32 unc) | **90.6%**(29/32) | ≥70% | ✅ |
| duty_holiday_service.go | 0%(29 unc) | **82.8%**(24/29) | ≥70% | ✅ |
| duty_config_service.go | 0%(24 unc) | **83.3%**(20/24) | ≥70% | ✅ |
| duty_service.go | 0%(24 unc) | **100.0%**(24/24) | ≥70% | ✅ |

- 合计清欠:**~380 unc → 44 unc**(家族 385 stmts,covered 5 → 341,+336,家族聚合 88.6%)。
- **滚动累计**:root 包总口径 11.3%(589/5202,79-RESEARCH §2)→ **20.4%**
  (最终树全包 profile 实测);本 plan 段贡献 +336 covered,79-01 段 +155,
  Phase 79 累计约 +491。
- SC-2 discharge:**duty 家族 6 文件全部脱离 <50% 区**(最低 82.8%),本 plan 前全部为
  0% 或 4.9%。
- Must-have truths 全兑现:GenerateSchedule 8 分支 + getDutyType 三态 + isWeekend 双分支
  + 6 CRUD 方法;CreateDutyPool 校验/序列化(关联形态)分支;四小文件全方法;
  duty 家族脱离 <50% 区;`<source>_79_02_test.go` 命名 + 7902 后缀 helper +
  sqlite t.TempDir 文件库 + 禁 t.Parallel。

## Quirks 处置(全部「只锁不修」,零生产改动;R7 / Phase 73-04 Q5 同款)

- **QUIRK-79-02-A** `duty_schedule_service.go:210-214`:非法 `orderByColumn` 走
  ApplySort 白名单回退(仅 warn 日志、不追加 Order),又因 `OrderByColumn != ""`
  不再补默认 `schedule_date ASC` → 顺序退化为 sqlite 自然序(插入序)。断言无错误 +
  总数正确,不锁具体顺序(实现细节)。
- **QUIRK-79-02-B** `GetMonthlyDutySchedule`:month=0/13 不报错,`time.Date` 归一化到
  2025-12 / 2027-01,返回空 map(非 nil)。
- **QUIRK-79-02-C** `SwapDuty` From==To 自换:实装不禁止 —— 同一行查两次对调同值,
  人员不变、Status 置 Exchanged、再记一条调班历史,均不报错。
- **QUIRK-79-02-D** `ManualDuty` 不校验值班池存在性(:388-418 无池查询;计划曾预期
  「池不存在 → 错误」分支,实装无)。幽灵 poolID 照常落排班行,已按现行为锁定。
- **QUIRK-79-02-E** `BatchDeleteDutySchedules([])`:GORM 无 WHERE 条件报
  `ErrMissingWhereClause`,被包装为「批量删除排班记录失败」(测试容错两种 GORM 形态)。
- **QUIRK-79-02-F** `CreateDutyPool` MemberIDs 为空:实装跳过成员段(:87 `len>0`),
  池照常创建且无成员;`required,min=1` 校验在 handler binding 层,service 层不拦。
- **QUIRK-79-02-G** `UpdateDutyPool` 不校验成员存在性(:246-258 无存在性查询),
  与创建段(:87-105 逐个校验)不对称 —— 幽灵成员照常落库。
- **QUIRK-79-02-H** `GetHolidayYears` 实装 `sort.Reverse` → **降序**(最近年份在前);
  计划文案写「去重升序」,以实装为准锁定 `[]int{2027, 2026, 2025}`。
- **QUIRK-79-02-I** `BatchCreateHolidays([])`:GORM `ErrEmptySlice`,包装为
  「批量创建节假日失败」。
- **QUIRK-79-02-J**(⚠️ 现网可见)`GetHolidayList` 年过滤下界是**本地零点**
  (`2027-01-01 00:00:00+08:00`),而 `time.Parse("2006-01-02")` 产出的行存为
  **UTC 零点**(`2027-01-01 00:00:00+00:00`);sqlite 对 TEXT 逐字比较时
  `+00` < `+08` → 恰落在 1 月 1 日的节假日被排除出当年列表(测试以 2027 元旦行
  证据化)。+08 时区下「元旦节假日不显示」属现网行为;修复需动绑定/存储形态,
  属生产改动走 escape hatch,**建议 Phase 79-06 收口或独立 quick 立项裁决**。
- **QUIRK-79-02-K** Holiday 软删不释放 `holiday_date` 硬唯一索引:同日期节假日
  「删后再建」撞 `UNIQUE constraint failed (2067)`。
- **(形态注意)** `duty_stats_service.go:24` 的 `time.Now().Truncate(24*time.Hour)`
  是 **UTC 日**零点(在 +08 时区 08:00 前其本地日期是「昨天」),而
  `GetTodayDuty` 用 `time.Now().Local()` 的本地日 —— 两处「今天」在凌晨时段不一致。
  测试侧:stats 用例以同一构造种子(绑定参数与存储文本逐字一致,`=` 才可命中),
  GetTodayDuty 用例以本地日期的 UTC 零点种子(`DATE()` 命中本地今天);
  `dsc7902LocalToday` 注释已记录该陷阱。生产语义是否要统一,超本 plan 范围。

## Deviations from Plan

1. **[Rule 3 - 流程] Task 4 与 Task 5 同文件,两次原子 commit 以「暂时摘除/还原」方式拆分**:
   tail 文件初稿即含两任务内容;为兑现计划各自的 `<commit>` 指令,先摘除 Task 5 段
   提交 Task 4(`a3f9417`,构建+三前缀用例全绿),再还原 Task 5 段提交(`23e095c`,
   构建+门面用例全绿)。无 stash/无破坏性 git 操作,两 commit 各自可独立构建。
2. **[Rule 3 - 补齐] Task 6 收口时追补 GetList Expired 过滤分支**(`cdce989`):
   全包 profile 显示 `:185-193` 过期态分支(约 5 stmts)是 duty_schedule 唯一
   sqlite 可达且未覆盖的行为分支,属 Task 2「过滤分支」既定范围 → 只增不改补三态
   用例,duty_schedule 88.5% → 91.4%。
3. **[环境] `go test -race` 本地不可执行**:Windows 本机 cgo 工具链故障
   (`cgo.exe exit status 2`),与 79-01 SUMMARY Deviation #2 / 78-01 Deviation #5
   同源(改动前既有测试同样构建失败)。race 纪律由 t.Cleanup 单次 Close +
   禁 t.Parallel + 被测路径零 goroutine 防护,ci.yml Linux race job 兜底。
4. **[计划-实装口径] `models.DutyPool.Members` 非序列化字段**:plan interfaces 段写
   「Members(序列化字段,以 model 实际类型为准)」,实装是 `[]DutyPoolMember` 关联
   (foreignKey PoolID,落 `sys_duty_pool_member`,Preload 装载)。按 plan 自带的
   「以 model 实际类型为准」条款采用关联形态:种子经成员表落库,
   round-trip 断言改为「请求成员集 → 成员表 → Preload 读回」三方可比。
5. **[计划-实装口径] ManualDuty「池不存在 → 错误」分支不存在**:见 QUIRK-79-02-D,
   按现行为锁定(quirk 纪律),不视为未完成项。

## Known gaps(剩余 44 unc,全部为 DB 层报错 / Save 失败 / 并发不可达分支,不追 100%)

- `duty_schedule_service.go`(15 unc):节假日查询失败(:52)、清除/保存排班失败
  (:70/:110,需表损坏注入)、GetList Count/Find 失败(:197/:217)、SwapDuty 两次
  Save 失败(:363/:366/:379)、ManualDuty 创建失败(:412)、DeleteDutySchedule 失败
  (:423)、GetTodayDuty 查询失败(:234)、GetMonthlyDutySchedule 查询失败(:289)、
  GetList Preload User 无昵称分支的 Phone nil 子分支 —— sqlite 单机健康库不可达。
- `duty_pool_service.go`(17 unc):名称检查/创建/成员创建/统计/列表 Count 与 Find
  的 DB 层失败包装(:65/:82/:117/:131/:165/:185/:43/:51)、GetByID 非 NotFound 错误
  (:200)、Update 名称检查/Updates/删旧成员/加成员失败(:216/:237/:242/:255)、
  Delete 排班检查/删成员/删池失败(:269/:277/:282)。
- `duty_stats_service.go`(3 unc):今日/本月/总计三处查询失败包装(:31/:62/:70)。
- `duty_holiday_service.go`(5 unc):Create/List/Update/Delete/Years 五处 DB 失败包装。
- `duty_config_service.go`(4 unc):查询失败(:35)、创建失败(:50)、查询失败(:56)、
  Save 失败(:64)。
- `duty_service.go`(0 unc):门面 100%,无缺口。

## Pre-existing flakes / 环境备注

- `-race` 本地不可执行(见 Deviation #3),非本 plan 引入。
- 全包 profile 首跑 383.6s(79-01 记录 378.9s 同量级),无新增 flake;
  79-01 已记录的 `TestLogUsagePerformance` 计时断言 flake 未在本 plan 触碰范围内复现
  (处置建议不变:Phase 79-06 统一裁决)。

## Acceptance criteria 对照

| 标准 | 结果 |
|---|---|
| TestDsc7902_ ≥16 / TestDpl7902_ ≥7 / tail 前缀合计 ≥7 | ✅ 17 / 7 / 6(Dst 1 + Dhd 2 + Dcf 2 + Dsv 1) |
| duty_schedule ≥70%(基线 0%)/ duty_pool ≥70%(基线 4.9%) | ✅ 91.4% / 83.3% |
| duty_stats/holiday/config/duty_service 四文件 ≥70%(基线 0%) | ✅ 90.6% / 82.8% / 83.3% / 100% |
| GenerateSchedule 8 分支 + 轮询 + 节假日双向断言齐备 | ✅(8 独立用例 + 12 槽轮询 + holiday 两向) |
| Members round-trip 断言存在 / GetHolidayYears 去重断言 / GetDutyConfig 空表默认断言 | ✅(三处均在) |
| 门面委托方法 ≥10 被调用 | ✅ 23/23 |
| 既有 duty 统计测试不回归 | ✅ `TestDutyPoolStatistics_NotDerivedFromCurrentPage` 原样通过 |
| `go build ./...` == 0 | ✅ |
| `go test ./internal/services/` == 0 | ✅ 最终树复跑 383.6s ok;最终树全包 profile 383.7s ok(coverage 20.4%) |
| `go test ./...` == 0 | ✅ repo_full_test_7902.log EXIT=0(该跑起于 cdce989 入库前;cdce989 触及的 services 包以最终树复跑兜底,见上行) |
| 生产 .go 改动 = 0 | ✅(6 个 commit 全部 *_test.go) |
| 日期断言无 time.Now 参与期望值(schedule 文件) | ✅(grep time.Now 仅 dsc7902LocalToday/today 语义方法,且不断言日期值) |

## 手注(给 79-03..79-06)

- 可复用同包 helper:`newDsc7902` / `seedPool7902` / `seedSchedule7902` /
  `dsc7902User`(sys_user 种子,Password/Salt not null 已处理)/ `dsc7902LocalToday`
  / `dsc7902Count` / `dsc7902Rows` / `newDpl7902` / `dpl7902Create` / `newDtx7902` /
  `dtx7902ShiftOffMonthStart`(月初边界自适应)。
- **duty 日期纪律(重要,后续 notice/template 等含日期字段的面直接沿用)**:
  ① 与 `time.Parse("2006-01-02")` 比较的种子一律用 UTC 零点;② 与
  `time.Now().Truncate(24h)` 做 `=` 的种子必须用同一构造;③ fixture 行避开
  本地月初 00:00(driver 存 `+00:00`、绑定参数 `+08:00`,TEXT 比较在边界行翻转,
  QUIRK-79-02-J 同根);④ `DATE()` 函数在 glebarez sqlite 下可用,前提是存储文本
  以 `YYYY-MM-DD` 开头。
- 集成测试写法先例:`TestDsv7902_FacadeDelegation` 的「主干 round-trip + 其余
  错误透传」口径可复制到其他门面型文件(D-79-07 泛化)。
- 若要修 QUIRK-79-02-J(元旦节假日年过滤丢失)或 QUIRK-79-02-D(ManualDuty 无池校验),
  均属生产改动,先立项再动手;测试已把现行为证据化,修复时改断言即可。

## Self-Check: PASSED

- 文件存在:duty_schedule_service_79_02_test.go / duty_pool_service_79_02_test.go /
  duty_family_tail_79_02_test.go / 79-02-SUMMARY.md — 全 FOUND。
- 提交存在:ad58efb / 0b5562b / b1202b9 / a3f9417 / 23e095c / cdce989 —
  全 FOUND(git log)。
- `go build ./...` exit 0。
- `go test ./internal/services/` exit 0 × 2:①最终树全量套件复跑 383.6s(含 cdce989);
  ②最终树全包 coverage profile 383.7s,coverage 20.4%(上表 per-file 数字即出自该 profile)。
- `go test ./...` exit 0(repo_full_test_7902.log EXIT=0;该跑起于 cdce989 入库前,
  cdce989 所在包以上一行最终树复跑兜底)。
