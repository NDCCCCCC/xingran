# Requirements (DRAFT): v1.55 LOCKED 残余根治

> ⚠️ **本文件是候选草案，不是当前权威 REQUIREMENTS.md**。当前 `.planning/REQUIREMENTS.md` 仍为 v1.54 SHIPPED 版本（Awaiting next milestone）。
> 用户 2026-11-09 裁定「要」后产出，供拍板；正式立项时由 `/gsd-new-milestone` 转为权威文件并清空/接管。
> **取数权威**：`.planning/notes/261109-v154-locked-root-cause-analysis.md`（3 个并行只读 subagent + Lead 交叉核对的逐项根因判定）。

**Defined:** 2026-11-09（草案）
**Milestone:** v1.55 LOCKED 残余根治（建议名，待确认）
**Core Value:** 把 v1.52 收档时以「已接受 / 仅登记」名义挂账的 LOCKED 残余，**从根源上**（而非打补丁/加告警/加方言分支）修掉可代码化的部分，使约束从「靠约定」升级为「靠机制」；对唯一需要产品让步的一项单列决策，不擅自推翻已交付语义。

## 范围裁定（用户 2026-11-09）

- **纳入（5 项，无需产品让步）**：② 5 项显式豁免面、③-A `RightCast:"text"`、③-B PG 专属 CTE、④ operlog e2e 静态门禁、⑤ `invalidateCache` no-op、⑥ `pbkdf2SM3` 契约 —— 合计 **6 个 LOCKED 子项**（③ 拆 A/B 两 REQ）。
- **① P1-5 聚合计数侧信道单列为「待产品决策」**，**不纳入本里程碑 REQ、不建 phase、不伪造闭环**。技术可根治但必须取消 `ScopeTotal/ScopeOutOfScope` 等回显、作废 132-SUMMARY.md:74 前端四类「本次范围：共 N 条」文案与 `useBackfillDept.tsx` 确认框 —— 属产品语义变更。
- **不纳入**：79 条人工验证跟踪表（保持 0 勾选）、FIX-01/04/09/13 验证类残余、v1.54 新增 7 项 Known Gaps（GAP-01~07）、FUTURE-01（随 ① 决策一并处置）。
- **Phase 编号**：延续 v1.54（结束于 144），从 **145** 续编。
- **跳过领域研究**：6 项均带本仓 `文件:行号` 实证与现成改写范本，无新领域能力需调研（沿用 v1.53/v1.54 先例）；根因报告即研究输入。
- **零新依赖**；遵循 Handler-Service / `response.Success()/Error()` / operlog 写操作约定 / SQLite+PG 双方言 / 前端 wrapped API / 构建产物只落 `out/`。

## v1.55 Requirements（草案）

### SCOPE — data_scope 豁免面根治（②）

- [ ] **SCOPE-01**: 5 类读端点（walls / doors / floor-plan-texts / rooms-photos / location-alias）在**统一父资源可见性校验**下按可见集裁剪——walls/doors/floor-plan-texts 经 `floor_id → ops_floors/ops_buildings` 归属链、rooms-photos 经 `room_id → serverRoom → building` 两级链（复用 `scope_entities.go` 的 `entityExistsSub` + `wrapScopeExists`），location-alias 按已有 `dept_id ∈ 可见集` 直挂；断链（floor/room 软删或不存在）fail-closed 空集。
- [ ] **SCOPE-02**: `internal/api/router.go:1174-1177` 的 rooms/photos 组补挂 `middleware.DataScopePermission(core)`（当前**连中间件都没挂**，比「豁免」更基础的接线遗漏）；`GetByID` / count / primary / batch-delete 前置全部走可见集校验。
- [ ] **SCOPE-03**: 写端点（Create/Update/Delete/Batch）对越界 `floor_id`/`room_id` 返回显式 403（单条）或按可见集裁剪（批量，复用 `scope_write.go:282-320` 的 `visibleIDsForEntityWrite`）；**必须修保存侧**：`useFloorPlanEditor.ts:156-186` 的「整层拉取 → 整体覆写」不得让非超管误删自己看不到的几何元素。响应形状不变（只裁剪行，不改 `list`/`total`/字段）。
- [ ] **SCOPE-04**: 双方言（PG + SQLite）403 矩阵 + 断链脏数据抽查断言一致；`133-UAT.md:79-91` 的 D1/D2「确认未收口」结论作废并登记重跑窗口；若这些端点将来加缓存，key 必须含 `scopeCacheKey(ctx)`。
  - **零迁移**（方案 D）：不新增 `org_id` 冗余列（方案 E）、不把几何合并进楼层 JSONB（方案 F）。

### DIALECT — 双方言 SQL 根治（③）

- [ ] **DIALECT-01**（③-A）: 删除 `RightCast` 承载 `"::"` 方言片段的机制（`excel_export_config.go:18,351/355/359`），改为语义枚举 + service 统一 `CAST(id AS TEXT)`（**单侧**即可，`IN ?` 参数是 Go 字符串）；`resolveAssociations` 的 3 个关联列在 SQLite 与 PG 均产出相同非空结果，不再 `logger.Warnf + continue` 静默降级为整列空；补 P1-8 式 DryRun 静态断言（导出路径 SQL 不含 `::`）+ SQLite 实跑断言。
- [ ] **DIALECT-02**（③-B）: 单工位 `GetPhysicalDevices`（`workstation_device_service.go:426-535`）与批量 `GetPhysicalDevicesByWorkstations`（`:2151-2234`）**从根源消除 PG 专属 raw CTE**——推荐拆多次 GORM 链式查询 + Go 层归一化/去重/排序（复用 `diagnosePhysicalChain:701-799` 与 `physicalDevicesFetcher` 注入，保证「不存在第二套 recall 逻辑」）；`DISTINCT ON` / `REGEXP_REPLACE` / `NULLS LAST` / `id::text` 全部归零，SQLite 与 PG 返回逐字段一致结果（含归一化/置信度/Top-1 口径）。
- [ ] **DIALECT-03**（③-B 配套）: 改写 4+ 个「把 PG CTE 报错当预期」的既有测试（`workstation_device_77_01_test.go:9-11,568-620`、`excel_export_chain_77_02_test.go:458-480`、`workstation_physical_diagnosis_test.go:401-403`）为真实结果断言；`SkipSoftDelete:true` 两处语义保留。
  - **明确禁止**：加 `isPostgres` 方言分支（两套 SQL 口径必分叉，是补丁不是根治）；CTE 下沉双方言视图（带 `?` 参数，视图无法参数化）。
- [ ] **DIALECT-04**（防复发机制，三合一才算根治）: ① `.golangci.yml` 启用 `forbidigo` 禁方言 token（`::text|::uuid|DISTINCT ON|REGEXP_REPLACE|NULLS LAST|ILIKE|ON CONFLICT`，存量先白名单逐步收敛；不得一刀切禁全部 raw SQL——406 文件含 `.Raw(`）② 视预算抽 `dialect` 语义原语（`CastToText` / `Top1PerGroup` / `NormalizeMac`），配置里禁止出现 SQL 片段 ③ 若 CI 已有 `services: postgres` 则注入 `TEST_PG_DSN` 把 3 个 contract 测试从 `t.Skip` 升级为双跑硬门禁（无 PG service 时登记为前置依赖，不伪造双跑）。

### AUDIT — operlog 静态门禁根治（④）

- [ ] **AUDIT-01**: 把 `scripts/operlog_e2e_verify.sh:68-151` 的**静态段从 shell 拆出改写成 Go 单测**并纳入 `go test ./internal/...`（`ci.yml:62-63` 天然硬门禁）：① `operlog.Record|RecordWithBody` 全 `internal/` 调用数 ≥290 ② `sensitiveKeys` 条目 ≥17 ③ `internal/api/v1/**/*_handler.go` 的 `*Handler` 方法文件逐文件覆盖扫描（0 命中且不在 `READONLY_ALLOWLIST` → FAIL）。可复用 `internal/utils/operlog/regression_test.go`（`go/ast` 锁 25 常量）与 `coverage_test.go`（脱敏）现成基建，补约 80-120 行 `go/parser` 扫描。
- [ ] **AUDIT-02**: 验收「零 `ADMIN_USER`/`ADMIN_PASSWORD`/`DEV_MODE`/`SKIP_E2E`/`SKIP_LIVE`、零 running backend、零 DB、**任意 cwd** 下静态 3 项断言真实执行且失败会红」；LIVE 段保留 shell、缺凭据仍 `exit 1`（恢复脚本注释声明的语义，不改 LIVE 行为）。
- [ ] **AUDIT-03**: 顺带处置两份 `operlog_e2e_verify.sh` 重复副本（仅 1 行注释不同）与 `scripts/README.md` 登记（AGENTS.md 目录治理）；**不做** shell 整体替换为 Go e2e（会把 CI 硬门禁与需凭据 live 重新耦合，复现同类缺陷）。
  - **必须同时修 cwd 锚定**：当前 Go driver 未设 `cmd.Dir`、脚本用相对仓库根路径，包子目录下 `CALLS=0` / 目标文件不存在 → 仅挪门禁顺序连缓解都不算（假绿）。

### CACHE — 缓存失效契约根治（⑤）

- [ ] **CACHE-01**: 删除死键 `CacheKeyReconciliationExceptionRuleList` / `GetReconciliationExceptionRuleListKey()` 与空函数体 `invalidateCache()`（`reconciliation_exception.go:578-580`），消除「定义缓存键却从不读写 + 失效函数为空」的悬空契约；4 个写路径（Create/Update/Delete/ImportFromExcel）的调用点同步清理。
- [ ] **CACHE-02**: 补**真实陈旧点**的联动失效：`reconciliation_service.go:821-834` 的 `GetByWorkstation` 5min TTL 缓存中 `exceptionHit/AppliedActions` 随例外规则变更陈旧 —— 规则写成功后按同文件 `InvalidateWorkstationHealth`（`cache_keys.go:129-134`，接 cache 参数、nil-safe、真调 Delete）范式广播失效，或把 exception 字段从缓存结果中剥离；保留 5min 缓存不动其性能价值。
- [ ] **CACHE-03**: 验收「写后读必一致」（List/GetByID/MatchTest/MatchException/DetectLayer3/`GetByWorkstation` 同进程立即反映新规则）、`Cache.Type=memory`（`*MemoryCache`）与 `redis`（`*MultiLevelCache`）两种配置行为一致、失效动作可观测且失败可传播（返回 error 或日志/metric，禁止空函数）。
  - **明确作废**触发条件「项目接入 Redis 后补 `core.Cache.Delete(ctx, 死键)`」——L1/L2 都支持 Delete，该处方删的是从没人写过的键，且无 Redis 时同样不生效。
  - **不推荐**改全局 `CacheProvider` 接口禁 no-op（20+ 消费文件/大量 mock 连锁改，未命中真实陈旧点，仅为缓解）。

### CRYPTO — KDF 契约根治（⑥）

- [ ] **CRYPTO-01**: `pbkdf2SM3`（`password.go:69-97`）把未定义行为变定义行为——`keyLen<=0` / `keyLen>32` / `iterations<=0` 显式返回 `error`（项目禁 panic），杜绝 `result[32:]` 静默全零弱密钥；`HashPassword`/`VerifyPassword` 同步接错误。
- [ ] **CRYPTO-02**: `VerifyPassword:149` 当前用 `len(hashBytes)` 作 keyLen（裁定「全部调用点 32」不成立）——显式校验 `len(hashBytes)==32`，非 32 字节哈希属损坏/伪造数据，按产品决策报错或返回 false，不再喂给 KDF。
- [ ] **CRYPTO-03**: **向后兼容硬约束**——`keyLen=32` 时输出逐字节不变；新增黄金向量测试（固定 password/salt/iterations 固化 hex，锁重构前后一致，不能只靠 Hash↔Verify 自洽），存量 `$sm3$<iter>$<salt>$<32字节hash>` 与 600k 默认 / 100k 旧哈希无需迁移；`$sm3$` 格式、`DefaultPasswordConfig`、25 个 operlog 常量均不变。
- [ ] **CRYPTO-04**（可选能力扩展，非消除当前隐患的必要条件）: 若产品确认未来需要 >32 派生（当前**无** SM4/会话/token 派生用途），再补完整 PBKDF2-SM3 多块循环（INT32_BE(i)、每块独立 `New/Reset`、`dk[:keyLen]` 截断、`make+append` 避免 `:75` slice 别名），用独立 oracle `golang.org/x/crypto/pbkdf2.Key(sm3.New,...)` 比对 `keyLen=33/64/65`、断言 `DK(64)[0:32]==DK(32)`。**守卫仍为主防线**，补循环不叠加守卫只算缓解。
  - **前置盘点**：存量 `sys_user.password` 是否存在非 32 字节数据（若有，CRYPTO-02 会让这些账号无法登录，需单独迁移决策；区分「输出不变」与「输入校验收紧」）。

### ARCH — 共性防回归（可选，视预算）

- [ ] **ARCH-01**: 视 ② ③ ⑤ 实施结果评估是否立项**统一 ScopedDB**：`DataScopePermission` 注入 `ScopeResult`（已有 `pkg/datascope` 单一真实源）→ GORM `Scopes` 强制唯一查询起链点 + 「资源归属注册表」（引用 `effectiveOrgExpr` / 归属链 EXISTS / `dept_id` 直挂，禁止内联复制）+ 内建 `scopeCacheKey` → 响应契约层禁止「全库规模 − 可见规模」回显。
  - **约束**：触及 `base.GORMRepository` 全仓基座（485 Go 文件）与 system/vdi 零回归保证，回归面大；**必须保留「未注入 ctx → All 兜底」短路**（`scope_filter.go:37-39`、`datascope.go:6`）。
  - **不可行方向（勿立项）**：DB 视图（可见集每用户动态、无法参数化）、PG RLS（SQLite 无 RLS、Go 共享连接池串味需每请求 `SET LOCAL` 独占事务、双方言两套安全模型）。
  - 若本里程碑预算不足，登记为 v1.56 候选，**不在本里程碑伪造「统一架构已落地」**。

## Future Requirements / 待产品决策（不纳入 v1.55 REQ、不建 phase）

| 项 | 决策内容 | 影响 |
|---|---|---|
| **DECISION-① P1-5 根治** | 是否允许取消 `BackfillDeptResult` 的 `ScopeTotal/ScopeVisible/ScopeOutOfScope/ScopeDeptTotal` 与 `SkippedNoUser/SkippedBadUserDept` 回显、去掉 `workstation_service.go:666-669` 全库 `countQ`、去 `ImportResult.Errors[].Row` 行号与 `ScopeOutOfScope` | **决定性**：不允许则①只能缓解（粗粒度桶 / 频率限制 / FUTURE-01 告警通道），按「不打补丁」标准不根治；允许则 A+B 组合（响应体去规模化 + 统一 ScopedDB 防回归）可根治，需作废 132-SUMMARY.md:74 四类前端文案并重跑 132/133 backfill UAT |
| **DECISION-FUTURE-01** | FUTURE-01（P1-5 告警通道与 FIX-11 合并评估）触发条件「FIX-11 告警通道落地」在 v1.54 已成立但未覆盖聚合计数面 | 随 ① 重定语义一并关闭，或维持「仅登记」 |
| **DECISION-CI-PG** | CI 是否已有 `services: postgres` 可复用 | 决定 DIALECT-04③ 的工作量；无则登记前置依赖 |
| **DECISION-存量密码** | `sys_user.password` 非 32 字节数据盘点结果 | 决定 CRYPTO-02 是否需配套迁移 |
| **DECISION-例外陈旧窗口** | `GetByWorkstation` 5min TTL 是否要求「规则变更立即一致」 | 允许 5min → CACHE-02 可只删死代码；要求立即 → 必须联动失效 |

> 上述 5 项**不勾选、不建 phase、不伪造闭环**；用户/产品拍板后另立里程碑或 quick 任务。

## Out of Scope

| Feature | Reason |
|---|---|
| 79 条人工验证跟踪项的实际执行与勾选 | 需生产 PG 只读 / 写授权 / 现场真机 / DBA 回传；保持 0 勾选，AI 不擅自写生产数据 |
| FIX-01 实机 AD UAT / FIX-04 真机逐型号取证 / FIX-09 HTTP P99 压测 / FIX-13 生产库 migration 236/237 | 验证类残余，不伪造闭环 |
| v1.54 新增 7 项 Known Gaps（GAP-01~07） | 已在 `STATE.md ## Deferred Items` v1.54 行登记，属后续候选；本里程碑专注 LOCKED 根治 |
| 6 项 LOCKED 残余中 ① 的代码修改 | 待产品决策 DECISION-①；未拍板前维持 LOCKED 原状 |
| PG RLS / DB 视图 / 按用户分表 | Go+GORM+SQLite/PG 双方言下不可行（见 ARCH-01） |
| shell 整体替换为 Go e2e / 加 `isPostgres` 方言分支 / 加告警通道代替 ① 根治 / 「接 Redis 后补 Delete」 | 均为补丁或无效处方，按用户「不打补丁」标准排除 |
| 新领域能力、新外部依赖 | 6 项均有本仓现成范本与实证，零新依赖 |

## Constraints (inherited)
- Handler-Service；service 返回 `error`；`response.Success()` / `response.Error()`（禁止裸 `c.JSON`）；写操作 success path 在 `response.Success` **之前** `operlog.Record(...)`；敏感端点 `RecordWithBody`。
- `0=正常 1=停用`；UUID 主键；**SQLite+PG 双方言**（纯 GORM / `CAST AS TEXT` / 禁 `::uuid`/`ILIKE`/裸 `ON CONFLICT`）；PG contract 门控变量名 `TEST_PG_DSN`。
- 前端 wrapped API（`@/lib/api.ts`）、Zustand、禁止裸 axios、`useEffect` 依赖稳定。
- Go 单测就近 `*_test.go`；集成 `tests/integration/`；e2e `tests/e2e/`；前端 `*.test.ts(x)`。
- 构建产物只落 `out/`；Excel 模板动态生成、禁止落盘 `.xlsx`；`data/` 不入库。
- **向后兼容**：`keyLen=32` KDF 输出不变、25 个 operlog 常量值不重排、`response.Success` 响应形状（②）不变、All 档 data_scope 零回归。
- 回归基线：`go build ./...` + `go test ./...` + `tsc` + `lint:ci`（含新增 forbidigo） + `vitest` 全绿。

## Traceability（草案，待 roadmapper 生成 ROADMAP 时填充）

| Requirement | Phase | Status |
|-------------|-------|--------|
| SCOPE-01 | — | Pending |
| SCOPE-02 | — | Pending |
| SCOPE-03 | — | Pending |
| SCOPE-04 | — | Pending |
| DIALECT-01 | — | Pending |
| DIALECT-02 | — | Pending |
| DIALECT-03 | — | Pending |
| DIALECT-04 | — | Pending |
| AUDIT-01 | — | Pending |
| AUDIT-02 | — | Pending |
| AUDIT-03 | — | Pending |
| CACHE-01 | — | Pending |
| CACHE-02 | — | Pending |
| CACHE-03 | — | Pending |
| CRYPTO-01 | — | Pending |
| CRYPTO-02 | — | Pending |
| CRYPTO-03 | — | Pending |
| CRYPTO-04（可选） | — | Pending |
| ARCH-01（可选） | — | Pending |

**Coverage:** 20 REQ 草案（含 2 项可选）/ 0 mapped（待 ROADMAP）/ 6 个 LOCKED 子项全覆盖（① 单列决策）

### 建议的 Phase 划分（供 gsd-roadmapper 输入，非定案）
| Phase | 主题 | REQ | 文件面（互斥检查用） |
|---|---|---|---|
| 145 | AUDIT 静态门禁 Go 化 | AUDIT-01/02/03 | `internal/utils/operlog/*_test.go`、`scripts/operlog_e2e_verify.sh`、`tests/e2e/operlog_e2e_verify.sh`、两 driver、`scripts/README.md`、`.github/workflows/ci.yml` |
| 146 | CRYPTO KDF 契约 | CRYPTO-01/02/03/04 | `internal/core/security/password.go` + `*_test.go` |
| 147 | CACHE 失效契约 | CACHE-01/02/03 | `internal/services/asset/reconciliation_exception*.go`、`reconciliation_service.go`、`cache_keys.go` |
| 148 | DIALECT 双方言 | DIALECT-01/02/03/04 | `excel_export_config/service*.go`、`workstation_device_service.go` + 4+ 测试、`.golangci.yml` |
| 149 | SCOPE 豁免面收口 | SCOPE-01/02/03/04 | 5 个 `*_service.go`、`scope_entities.go`、`internal/api/router.go`、`useFloorPlanEditor.ts`、`133-UAT.md` |
| 150（可选） | ARCH 统一 ScopedDB | ARCH-01 | `base/GORMRepository`、`pkg/query`（新）、8 个 service 起链处 |

> W1 可并行的候选：145 ∥ 146 ∥ 147（文件面互斥）；148 与 149 均可能动 `internal/services/operations/**`，需函数级归属表或降级串行；150 依赖 148/149 经验，应最后。

---
*Draft defined: 2026-11-09 after 用户裁定「要」整理 v1.55 REQUIREMENTS 草案*
*Last updated: 2026-11-09 — 待产品/用户拍板后由 `/gsd-new-milestone` 转为权威 REQUIREMENTS.md*
