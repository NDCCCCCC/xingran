# Phase 79 Deferred Items (out-of-scope discoveries)

## 2026-08-28 · 79-01 executor

| # | Item | Discovered during | Why not fixed here |
|---|------|-------------------|--------------------|
| 1 | `internal/services/usage_logger_test.go:516 TestLogUsagePerformance` 含 wall-clock 计时断言 `elapsed.Milliseconds() < 100`(:546),在全包满载跑(390s,叠加并行 Phase 88 前端会话)下可超时 flake:79-01 收口首跑 FAIL,复跑(failfast 全包)EXIT=0 通过。属预存 flaky,非 79-01 引入。 | 79-01 T5 全包覆盖率首跑 | Scope Constrainment:非本 plan 改动引入,零生产/既有测试改动纪律。建议 79-06 收口时统一处置(放宽阈值 / 改为异步完成计数断言)。 |
| 2 | QUIRK-79-01-A:`data_cache_service.go:68-70` 的 `data == "" → apperrors.CacheKeyNotFound()` 分支对既有生产装配不可达 —— `MemoryCache.Get(miss)` 返回 `("", ErrNotFound)`,`RedisCache.Get` 把 `redis.Nil` 翻译为 `ErrNotFound`(pkg/cache/redis.go:78-80),没有任何实现会返回 `("", nil)`。 | 79-01 T1 编写 Get miss 用例 | plan interfaces 段与实装不符,按 quirk 纪律就地记录 + 测试锁定现行为,并以接口合规 test double(`dcs7901EmptyGetCache`)保持该分支覆盖。删除该分支属生产改动,需单独裁决(建议保留作为 cache.Cache 接口契约防御 + 注释)。 |
| 3 | `data_cache_service.go` 剩余 3 个不可达错误分支:GetOrSet 写缓存失败 Warnf(:105-107)、DeleteByPattern 的 Keys 报错(:120-122)、MGet 报错(:132-134)——均需底层 cache 返回 error,MemoryCache/RedisCache 正常路径不可达;`template_cache.go` 写锁内二次检查(:37-39)仅并发双 Get 竞态可达(且禁 t.Parallel)。 | 79-01 T5 per-file 复盘 | 需新造故障注入装配才有确定性路径,投入产出比低;文件已达 96.1% / 94.4%,远超 ≥70% 目标。 |
