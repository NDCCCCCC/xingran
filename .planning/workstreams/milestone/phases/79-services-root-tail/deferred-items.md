# Phase 79 Deferred Items (out-of-scope discoveries)

## 2026-08-28 · 79-01 executor

| # | Item | Discovered during | Why not fixed here |
|---|------|-------------------|--------------------|
| 1 | `internal/services/usage_logger_test.go:516 TestLogUsagePerformance` 含 wall-clock 计时断言 `elapsed.Milliseconds() < 100`(:546),在全包满载跑(390s,叠加并行 Phase 88 前端会话)下可超时 flake:79-01 收口首跑 FAIL,复跑(failfast 全包)EXIT=0 通过。属预存 flaky,非 79-01 引入。 | 79-01 T5 全包覆盖率首跑 | Scope Constrainment:非本 plan 改动引入,零生产/既有测试改动纪律。建议 79-06 收口时统一处置(放宽阈值 / 改为异步完成计数断言)。 |
| 2 | QUIRK-79-01-A:`data_cache_service.go:68-70` 的 `data == "" → apperrors.CacheKeyNotFound()` 分支对既有生产装配不可达 —— `MemoryCache.Get(miss)` 返回 `("", ErrNotFound)`,`RedisCache.Get` 把 `redis.Nil` 翻译为 `ErrNotFound`(pkg/cache/redis.go:78-80),没有任何实现会返回 `("", nil)`。 | 79-01 T1 编写 Get miss 用例 | plan interfaces 段与实装不符,按 quirk 纪律就地记录 + 测试锁定现行为,并以接口合规 test double(`dcs7901EmptyGetCache`)保持该分支覆盖。删除该分支属生产改动,需单独裁决(建议保留作为 cache.Cache 接口契约防御 + 注释)。 |
| 3 | `data_cache_service.go` 剩余 3 个不可达错误分支:GetOrSet 写缓存失败 Warnf(:105-107)、DeleteByPattern 的 Keys 报错(:120-122)、MGet 报错(:132-134)——均需底层 cache 返回 error,MemoryCache/RedisCache 正常路径不可达;`template_cache.go` 写锁内二次检查(:37-39)仅并发双 Get 竞态可达(且禁 t.Parallel)。 | 79-01 T5 per-file 复盘 | 需新造故障注入装配才有确定性路径,投入产出比低;文件已达 96.1% / 94.4%,远超 ≥70% 目标。 |

## 2026-08-28 · 79-02 executor

| # | Item | Discovered during | Why not fixed here |
|---|------|-------------------|--------------------|
| 4 | QUIRK-79-02-J(⚠️ 现网可见):`GetHolidayList` 年过滤下界绑定**本地零点**(`2027-01-01 00:00:00+08:00`),而 `time.Parse("2006-01-02")` 产出的行存为 **UTC 零点**;glebarez sqlite 按 TEXT 逐字比较时 `+00` < `+08`,恰落在 1 月 1 日的节假日被排除出当年列表(+08 时区「元旦节假日不显示」)。测试已以 2027 元旦行证据化(`TestDhd7902_Holiday_DateShape`)。 | 79-02 T4 Holiday 年过滤用例 | 修复需动绑定形态(改传 UTC 端界)或存储形态(日期列归一化),属生产改动 + 回归面(导入/缓存种子同源),走 escape hatch 立项;本 phase 零生产改动纪律。 |
| 5 | QUIRK-79-02-D:`ManualDuty` 不校验值班池存在性(`duty_schedule_service.go:388-418` 无池查询),幽灵 poolID 照常落排班行;与 `GenerateSchedule`(:29 池校验)不对称。计划曾预期该错误分支,实装无,已按现行为锁定(`TestDsc7902_ManualDuty`)。 | 79-02 T2 编写 ManualDuty 用例 | 加校验属生产行为变更(影响 handler 语义与既有调用方),非测试补齐范畴;建议随 duty 模块行为加固单独立项。 |
| 6 | QUIRK-79-02-K:Holiday 软删不释放 `holiday_date` 硬唯一索引,同日期节假日「删后再建」撞 `UNIQUE constraint failed (2067)`(`TestDsv7902_FacadeDelegation` 证据化)。 | 79-02 T5 门面批量创建用例 | 唯一索引带 `where deleted_at is null` 的 partial index 在 sqlite/PG 双方言下的迁移形态需单独设计,属生产 schema 改动。 |
| 7 | duty 家族剩余 44 unc 全部为 DB 层报错包装 / Save 失败 / `Tx.Save` 失败分支(sqlite 单机健康库不可达),明细见 79-02-SUMMARY「Known gaps」。duty_schedule 91.4% / pool 83.3% / stats 90.6% / holiday 82.8% / config 83.3% / service 100%,均超 ≥70% 目标。 | 79-02 T6 per-file 复盘 | 需表损坏/连接故障注入装配,投入产出比低;SC-2「无单文件留在 <50%」已达成。 |
| 8 | `duty_stats_service.go:24` 的 `time.Now().Truncate(24*time.Hour)`「今天」是 UTC 日零点(+08 时区 08:00 前为本地昨日),与 `GetTodayDuty` 的 `time.Now().Local()` 本地日在凌晨时段不一致。测试侧已按各自构造同值种子锁定。 | 79-02 T4 GetMyDutyStats 种子设计 | 统一两处「今天」语义属生产行为变更(影响统计口径),建议随 79-02-J 一并裁决。 |
