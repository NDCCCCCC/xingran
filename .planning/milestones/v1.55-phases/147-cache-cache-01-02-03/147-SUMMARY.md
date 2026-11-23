# Phase 147 · CACHE 失效契约 — 执行 SUMMARY

> W1 并行 executor（145 ∥ 146 ∥ 147）产出。`commit:false`：**零 git 写操作**，源码停在磁盘，**由 Lead 统一集成提交**。
> `router_go_touched: false` —— **未改动 `internal/api/router.go`**（§4.2 未触发，无需移交 149/151 顺序登记）。

---

## ① 改动清单（逐文件）

| 文件 | 改动 |
|------|------|
| `internal/services/asset/cache_keys.go` | 删 2 个死键常量 `CacheKeyReconciliationExceptionRuleList` / `...ByID` + 2 个 helper `GetReconciliationExceptionRuleListKey()` / `...ByIDKey()`；`:23` R3 演进注释改为「例外规则缓存键已下线（Phase 147 CACHE-01，读路径直查 DB，无缓存契约）」，不留「待启用」暗示。其余常量/helper（`InvalidateWorkstationHealth` / `GetReconciliationHealthByWorkstationKey` / `CacheKeyReconciliationHealthByWorkstation` / `StripCachePrefix` 等）原样保留。 |
| `internal/services/asset/reconciliation_exception.go` | ① 结构体加 `cache base.CacheProvider`；② 构造函数改**向后兼容可变参数** `NewReconciliationExceptionService(db, cacheProviders ...base.CacheProvider)`（零参调用方零改动）；③ 删空函数 `invalidateCache()`（含 `:574-580` 定义段注释与「接入 Redis 再补 Delete」裁定注释）；④ 新增 `invalidateExceptionRuleCache(ctx) error`：nil-safe、真调 `s.cache.DeleteByPattern(ctx, fmt.Sprintf(CacheKeyReconciliationHealthByWorkstation, "*"))`、失败 `fmt.Errorf(...: %w)` 包装；⑤ 4 个写路径（Create/Update/Delete/ImportFromExcel）DB 写成功后统一 `if err := s.invalidateExceptionRuleCache(ctx); err != nil { applogger.Warnf("[reconciliation:exceptionRule] <Action> 后缓存失效失败（不阻断主流程）: %v", err) }`（失败不阻断主写但可观测）；⑥ 清 `:337` / `:391` 悬空/过时注释；⑦ import 追加 `applogger`。 |
| `internal/services/asset/reconciliation_service.go` | **仅注释级改动**：`getByWorkstationWithCache` 注释追加一句「Phase 147 CACHE-02：例外规则写路径经 invalidateExceptionRuleCache 广播失效本键前缀，保证规则变更后本缓存内 exceptionHit/AppliedActions 立即一致；本函数读路径不变」。`GetByWorkstation` / `getByWorkstationWithCache` / `computeByWorkstation` / `reconciliationHealthCacheTTL = 5min` **零改动**。 |
| `internal/services/asset/asset_gapfill_test.go` | 删 `:117,118` 两行死键字符串断言；`:768-771` 按 PLAN T1-4 **定案②** 改为 `require.NoError(t, impl.invalidateExceptionRuleCache(ctx))`（零参构造 cache=nil，顺带覆盖 nil-safe 语义），`:768` 含旧标识符的注释删除。 |
| `internal/services/asset/reconciliation_cache_contract_test.go` | **新增**：`contractCacheSpy`（实现 `base.CacheProvider` 全 9 方法，JSON 往返读穿透 + `DeleteByPattern` 真删）+ `countingExceptionSvc` + 11 个契约测试，详见 ③。复用 `newGapTestDB` / `seedWorkstationHealth` / `buildGapXLSX` / `gapXlsxFileHeader` 同包基建，未新造 SQL、未落盘 `.xlsx`。 |
| `internal/api/v1/asset/reconciliation_exception_router.go` | **唯一生产装配改动**：4 个写路径的 service 装配点 `asset.NewReconciliationExceptionService(core.DB.GetDB())` → `asset.NewReconciliationExceptionService(core.DB.GetDB(), system.NewCacheProvider(core.DataCacheService))` + import 追加 `internal/services/system`。该文件不在 ROADMAP.md:145 的 `router.go` 硬登记表内。 |
| `internal/api/router.go` | **未改动**（`:894 exceptionSvcForWs` 仍为零参构造；`reconciliation_router.go:35` 亦未动）。通过「可变参数 + 只在 `reconciliation_exception_router.go` 装配」完全规避 §4.2。 |
| `.planning/phases/147-cache-cache-01-02-03/147-SUMMARY.md` | 本文件（新增）。 |

**未触碰**：`internal/utils/operlog/**`（145）、`internal/core/security/**`（146）、`internal/services/operations/**`、`xingran-react-frontend/**`、`.golangci.yml`、`.github/workflows/ci.yml`、其它 phase 的 PLAN/SUMMARY、`base.CacheProvider` 接口 / `NoOpCacheProvider` / 20+ 消费文件、migration、79 条人工验证跟踪表。

---

## ② CACHE-01 / 02 / 03 满足情况（对应 ROADMAP Phase 147 四条成功标准）

| 成功标准 | 满足情况 |
|----------|----------|
| **#1 死键 / helper / 旧标识符清零，悬空契约消除** | ✅ 4 个死符号全仓零引用；`invalidateCache` 旧标识符全库 **6 处命中**全部处置：4 个生产写路径（`:381/:449/:474/:622`）+ 函数定义 `:578` 及 `:574-580` 定义段注释 + `asset_gapfill_test.go:768` 注释与 `:771` 测试调用（定案②改调新失效函数并 `require.NoError`）+ `:337` 悬空注释；`asset_gapfill_test.go:117,118` 两行删除。删旧函数后 `go test -count=1 ./internal/services/asset/` 编译且通过，包不再 undefined。 |
| **#2 规则写成功后 `GetByWorkstation` exception 字段立即一致；List/GetByID/MatchTest/MatchException/DetectLayer3 同进程立即一致** | ✅ 方案 (a) 广播失效落地：4 个写路径写成功后失效 `reconciliation:health:workstation:*`，写后读必然回源重算（不等 5min / cron `@every 6m`）；`TestExceptionRuleWrite_ReadYourWrites` 4 子用例（Create/Update/Delete/ImportFromExcel）全绿；`reconciliationHealthCacheTTL=5min` 与 `getByWorkstationWithCache` / `computeByWorkstation` 未改（静态守护 `TestReadPath_TTL_Unchanged`）；反向验证注释失效调用后必红。 |
| **#3 memory/redis 两配置行为一致；失效真删、nil-safe、失败可传播、禁空函数体** | ✅ `invalidateExceptionRuleCache(ctx) error` 有 ctx / 有 error / 真调 `DeleteByPattern` / nil-safe；memory 真删由 `TestInvalidate_MemoryCache_ReallyDeletes`（`pkgcache.NewMemoryCache` + `services.NewDataCacheService` + `system.NewCacheProvider`）实跑断言；两种生产 `CacheProvider`（`cacheProviderAdapter` / `NoOpCacheProvider`）共用 `DataCacheService.DeleteByPattern = Keys→MDelete`，静态断言 `TestCacheProviderImplementations_ShareDeleteByPattern` 锁定；`base.CacheProvider` 接口与两实现零改动；mock 断言 `DeleteByPattern ≥1`。 |
| **#4 明确作废「接入 Redis 后补 `core.Cache.Delete(ctx, 死键)`」；不改全局接口** | ✅ 死键与裁定注释已删，代码/注释/SUMMARY 均不留该处方作为待办或触发条件（本段仅作「已作废」历史说明）。3 条 grep CLEAN；`base/cache_provider.go` / `NoOpCacheProvider` / 20+ 消费文件零改动。 |

---

## ③ 验证结果

| 命令 | 结果 |
|------|------|
| `go test -count=1 ./internal/services/asset/` | **全绿**（`ok ... 0.593s`；含 11 个新增契约测试与全部既有测试） |
| `go build ./...` | **通过**（`BUILD_EXIT=0`，W1 并行期未受邻 phase 影响） |
| `go test -count=1 -run TestExceptionRuleWrite_ReadYourWrites ./internal/services/asset/`（临时注释 4 处失效调用） | **红**：4 个子用例全部报「写路径必须触发 DeleteByPattern」（`DeleteByPattern` 调用数 0）→ 证明联动失效是「立即一致」的真因 |
| 还原后 `go test -count=1 ./internal/services/asset/` | **绿**（`ok`），临时改动已 100% 还原（4 处调用恢复、`error(nil)` 残留 0） |
| 反向验证手段 | `sed -i 's/if err := s.invalidateExceptionRuleCache(ctx); err != nil {/if err := error(nil); err != nil {/g'` → 测试 → `cp` 备份还原；未留在最终文件、未提交 |

**3 条 grep CLEAN**（判据 = `internal/services/asset/` + `internal/api/v1/asset/` 全 `.go`，覆盖 `*_test.go` 与注释）：
1. `CacheKeyReconciliationExceptionRuleList|GetReconciliationExceptionRuleListKey|CacheKeyReconciliationExceptionRuleByID|GetReconciliationExceptionRuleByIDKey` → **全 `internal/` 0 命中 / CLEAN**
2. `invalidateCache` → **Phase 147 文件面 0 命中 / CLEAN**
3. `接入 Redis.*补.*Delete|未来接入 Redis 再补` → **全 `internal/` 0 命中 / CLEAN**

> grep 范围说明：`internal/` 下另有 **10 处** `invalidateCache` 命中，全部来自 `internal/services/system/dict_cache_impl.go` 等**其它模块功能正常的同名私有失效方法**（与 Phase 147 asset 包死函数无关，非本 phase 文件面，按 PLAN 禁区不触碰）；另有 `internal/server/xingran-react-frontend/dist/assets/index-C2ZQmXw2.js` 前端构建产物含 menuStore 的 `invalidateCache`（JS、非 Go、非本 phase 面）。二者均**不计入** Phase 147 判据，此处如实登记以免 Lead 集成时误判。

**新增测试清单（均绿）**：`TestExceptionRuleWrite_ReadYourWrites`（4 子用例）、`TestInvalidate_ReturnsError_OnDeleteByPatternFailure`、`TestWritePath_InvalidationFailure_DoesNotBlockButObservable`、`TestInvalidate_NilCache_IsNoOp`、`TestInvalidate_DeleteByPattern_CalledAtLeastOnce`（4 子用例）、`TestInvalidate_MemoryCache_ReallyDeletes`、`TestInvalidate_MemoryAndRedis_BehaveIdentically`（`t.Skip`，见 Known Gaps）、`TestCacheProviderImplementations_ShareDeleteByPattern`、`TestDeadSymbols_AreGone`、`TestBroadcastPattern_SharesPrefixWithReadKey`、`TestReadPath_TTL_Unchanged`。

---

## ④ 选定方案与性能评估

- **方案 (a) 广播失效（Lead 定案，未改选 (b)）**：例外规则服务注入 `base.CacheProvider`，写成功后 `DeleteByPattern("reconciliation:health:workstation:*")`；与 `core.DataCacheService` 同一后端（`DataCacheService.cache` 即 `c.Cache`），memory/redis 两实现都走 `Keys → MDelete` 真删，`NoOpCacheProvider` 因 `GetOrSet` 从不写缓存而自洽。**不新增常量**：pattern 由 `fmt.Sprintf(CacheKeyReconciliationHealthByWorkstation, "*")` 得出，与读缓存键单一真实源，`TestBroadcastPattern_SharesPrefixWithReadKey` 防「删平行宇宙的键」。
- **不采用 (b) 剥离 exception 字段**：会改缓存值形状、每次读多跑 `preloadActiveRules + MatchException`，削掉 6 步聚合 5min 缓存性能价值。
- **性能**：成本仅落在低频写路径 = 1 次 `Keys(pattern)`（memory 遍历 map / redis 合并 L1+L2）+ 1 次 `MDelete`；工位健康度缓存条目数 = 工位数，量级小；失效后首个读回源一次 6 步聚合，之后继续 5min 命中。**读路径零额外开销、TTL 不变、`computeByWorkstation` 6 步查询口径不变**。
- **未选方案**：① 改全局 `CacheProvider` 接口（20+ 文件连锁改、未命中真实陈旧点）；② 规则版本号 + 读时校验（需 migration）——均按 PLAN 禁区排除。

---

## ⑤ Known Gaps（如实登记，交 Lead 集成复核）

1. **Redis 真机双跑未实跑**：`TestInvalidate_MemoryAndRedis_BehaveIdentically` 显式 `t.Skip`（原因：需真 `*MultiLevelCache` + 真 Redis，本 phase 不启动长驻进程、不连生产）。已用「memory 真删实跑 + 两种生产实现共用 `DeleteByPattern` 静态断言 + mock 调用次数」覆盖一致性，**未伪造双跑**。集成环境若具备 Redis 可补跑。
2. **契约测试「写后立即一致」判据口径**：排查发现 `computeByWorkstation`（`reconciliation_service.go:1074`）先 `append(assetItems, item)`、再在 `:1085-1094` 修改 `item` 的 `ExceptionRuleID` / `AppliedActions` —— 切片里存的是副本，per-asset matcher 结果未进入响应。这是**生产既有缺陷**（Phase 45 R4 遗留），且 PLAN 禁区 8 禁止改 6 步聚合，故本 phase 未修。契约测试因此采用更严格且不受该缺陷影响的等价判据：① 写路径真删缓存键；② 写后读确实回源重算（`missCalls` / `matchCalls` 增加）；③ 直查 DB 读路径（`MatchException` / `List` / `MatchTest`）同进程立即反映新规则。**建议 Lead 后续单独立项修 `append` 顺序缺陷**（本 phase 不越界）。
3. **ImportFromExcel 子用例**：使用真实 `svc.ImportFromExcel`（复用同包内存 xlsx 基建，`result.Inserted==1`），未用最小替身；`postProcessImportedRules` 既有 QUIRK（中文表头/英文键名错配致 `exception_actions` 覆写为 NULL）保持原样、本 phase 未修。
4. **其它模块同名 `invalidateCache`（10 处）与前端 dist JS 命中**：见 ③ 说明，非本 phase 文件面，未触碰。
5. W1 并行期 `go build ./...` 本次实测通过、未受 145/146 邻 phase 影响；若 Lead 集成后全量矩阵有变化，以集成重跑为准。

---

## ⑥ 提交状态

**未执行任何 git 写操作**（`git add` / `commit` / `stash` / `checkout` / `reset` / `merge` / `rebase` / `tag` 全未运行）。本 phase 全部改动仅停在工作区磁盘，**由 Lead 统一集成提交**。未启动长驻进程、未写生产数据、未改其它 phase 文件。

**`internal/api/router.go`：未改动**（`router_go_touched: false`，§4.2 未触发），无需 149/151 顺序移交。
