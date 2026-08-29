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

## 2026-08-28 · 79-05 executor

| # | Item | Discovered during | Why not fixed here |
|---|------|-------------------|--------------------|
| 9 | QUIRK-79-05-H(⚠️ 现网可见):`MergeFlappingRecords` 分组键 `fmt.Sprintf("%v", hist.VLANID)` 打印 **\*int 指针地址**(mac_history_service.go:332),每行读回指针不同 → 非 nil VLAN 的记录永不分组、flapping 合并对带 VLAN 数据永不触发。测试以 nil-VLAN(可合并)/ 非 nil(不合并)双向证据化(`TestMhs7905_MergeFlapping`)。 | 79-05 T5 编写 flapping 合并用例 | 修复需解引用改组键(生产行为变更,影响现网合并结果与历史数据),按 R7/Phase 73-04 quirk 纪律锁定不修;走 escape hatch 立项。 |
| 10 | QUIRK-79-05-K(⚠️ 现网可见):`LDAPClient.Connect` Bind 失败路径只调 `c.conn.Close()` 不把 `c.conn` 置 nil(ad_ldap_client.go:91-93)→ `IsConnected()` 在连接失败后仍返回 true,上层以它判可用性会误判。测试证据化(`TestAdl7905_Connect_BindFails_RealTCP`)。 | 79-05 T7 wire 守卫表前置用例 | 置 nil 属生产行为变更(影响 addomain 调用方对失败后状态的语义预期),走 escape hatch 立项。 |
| 11 | QUIRK-79-05-F(⚠️ 现网可见):`collectDeviceMAC` 顶层 panic-recovery(mac_collection_service.go:118-122)吞掉 executor 异常后函数返回 nil → `CollectAllDevices` 返回 `[]*MACCollectionResult{nil}`、`CollectDevice` 返回 (nil,nil),调用方无法区分失败与空结果。executor 真路径 79-06 接入后应复核该断言。 | 79-05 T4 nil-executor 边界用例 | recover 后返回 nil 是控制流改动(recover 需显式落 result 并上抛),属生产变更;先随 79-06 executor 接入复核再裁决。 |
| 12 | QUIRK-79-05-A:`MACPerfConfigCacheTTLSeconds` 不命中 `CacheConfigService.LoadConfigs` 的 LIKE 通道(cache.% / rate_limit.%)→ perf 配置键实际永远读不到,`perfCacheTTL` 恒回兜底;键名 seconds 与 GetDuration 分钟语义互相矛盾但不可达。 | 79-05 T1 perfCacheTTL 用例 | 让 perf 键进入配置体系需扩 LoadConfigs 通道或改键前缀,属生产配置语义变更;建议随 MAC 性能调优单独立项。 |
| 13 | QUIRK-79-05-D:`queryConnectionStatsFromDB` 统计 SQL 为 PG 专属(`EXTRACT(EPOCH FROM ...)`/`::bigint`/`COUNT(*) FILTER`),sqlite 不可达 ~35 stmts;`database.type=sqlite` 部署下连接时长统计接口必报错。 | 79-05 T2 QueryConnectionStats 用例 | 兼容 sqlite 需改写统计 SQL(方言分支),属生产改动;R6 纪律禁为覆盖率改写,建议随 sqlite 部署支持单独立项。 |
| 14 | mac_history_partition.go 剩余 35 unc 全部需真实 PostgreSQL 分区表(pg_inherits 遍历 / DROP 分支 / DDL 真执行);单文件 63.2% 未到 70%,但五文件组 80.9% 已达标。 | 79-05 T8 per-file 复盘 | R6 明令禁真建分区;PG 集成环境属 infra 范畴,不在测试补齐范畴。 |
| 15 | ad_ldap_client.go 剩余 55 unc 全部为 wire 真路径(Search/Modify 的 entries 映射与成功段),D-79-04/78-07 Conclusion B 明确不再尝试 BER fake。 | 79-05 T8 per-file 复盘 | 需真实域控或兼容的 LDAP wire fake;69.3% 已 ≥55% 目标并脱离 SC-2 <50% 区。 |
| 16 | QUIRK-79-05-E/G/M/I/J/L:`GetMACAddressList` 非法排序列退化自然序(与 79-02-A/79-03-B 同族)、部分 MAC 输入退化全表匹配、`APISenderService.Send` 聚合结果丢弃末次 HTTPCode/ResponseBody、`parseMACLine` 兜底接口名不走 Normalize、`buildFromTemplate` 空 recipients 保留占位符、`formatUsername` 空 DomainName 产出前导反斜杠形态。均为低危行为不一致,已在各自测试注释锁定。 | 79-05 T4/T7 各用例 | 单项修复均为生产行为变更,收益低;建议归入通知/采集模块行为加固批量裁决。 |
