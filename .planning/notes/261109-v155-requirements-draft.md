# Requirements (DRAFT): v1.55 LOCKED 残余根治

> ⚠️ **本文件是候选草案，不是当前权威 REQUIREMENTS.md**。当前 `.planning/REQUIREMENTS.md` 仍为 v1.54 SHIPPED 版本（Awaiting next milestone）。
> 用户 2026-11-09 已就 5 项决策**全部拍板**（见「决策裁定记录」段），本草案已按裁定更新；正式立项时由 `/gsd-new-milestone` 转为权威文件并清空/接管。
> **取数权威**：`.planning/notes/261109-v154-locked-root-cause-analysis.md`（3 个并行只读 subagent + Lead 交叉核对的逐项根因判定）。

**Defined:** 2026-11-09（草案）｜**Updated:** 2026-11-09（5 项决策裁定后）
**Milestone:** v1.55 LOCKED 残余根治（建议名，待确认）
**Core Value:** 把 v1.52 收档时以「已接受 / 仅登记」名义挂账的 LOCKED 残余，**从根源上**（而非打补丁/加告警/加方言分支）全部修掉，使约束从「靠约定」升级为「靠机制」；用户 2026-11-09 裁定允许推翻 ① 的「范围快照回显」产品语义，故 **6 项 LOCKED / 7 个子项全部纳入根治**。

## 决策裁定记录（用户 2026-11-09，全部按推荐项）

| 决策 | 裁定 | 在草案中的落点 | 实证前提 |
|---|---|---|---|
| **DECISION-①**（决定性） | **允许重定语义，① 纳入根治（A+B 组合）** —— 允许取消 `ScopeTotal/ScopeOutOfScope` 等回显、去掉全库 `countQ`、去导入 `Errors[].Row` 行号、作废 132-SUMMARY.md:74 前端四类「本次范围：共 N 条」文案 | 新增 `SIDECHANNEL-01~04`；`ARCH-01` 从「可选」升级为**必需**（B = 统一 ScopedDB 防回归）；132/133 backfill UAT 作废重跑 | 8 个 `/statistics` 已同源；残余是 `workstation_service.go:662-677` 刻意双口径差分，属信息论必然非代码缺陷 |
| **FUTURE-01** | **随 ① 根治一并关闭**，不再单独挂账 | ① 落地后由 Lead 在 v1.55 收档同步清理 `.planning/REQUIREMENTS.md`(v1.54) / `ROADMAP.md:91` / `STATE.md` / `261108-v153-locked-residuals.md:63` 的引用 | 触发条件「FIX-11 告警通道落地」在 v1.54 已成立，但未覆盖聚合计数面 |
| **DECISION-CI-PG** | **本期不建 PG service，登记前置依赖** | `DIALECT-04③` 改为「登记前置依赖、不伪造双跑」；v1.55 只做 `forbidigo` 门禁（XS）+ 视预算抽 dialect 语义原语 | `ci.yml` 无 `services: postgres`；`TEST_PG_DSN` 明确 unset，PG contract 只做「skip 可观测」，零真实 PG 验证 |
| **DECISION-存量密码** | **免盘点，直接上守卫** | `CRYPTO-02` 直接加 `len(hashBytes)==32` 校验，无需迁移 | 全库 `sys_user.password` 写入均经 `HashPassword`（`user_service.go:841`、`user_sync_service.go:511/646/1143`、`excel_service.go:1172`），`$sm3$` 恒 32 字节；无绕过写入路径。仅对「人工直接改库的历史脏数据」在上线说明里提示（代码层不产生） |
| **DECISION-例外陈旧窗口** | **要求立即一致，做联动失效** | `CACHE-02` 必做（非可选项）：规则写成功后 `GetByWorkstation` 的 exception 字段立即一致 | `reconciliation_service.go:821-834` 5min TTL 缓存含 `exceptionHit/AppliedActions`；同文件 `InvalidateWorkstationHealth` 已是正确范式但无联动 |

> 5 项裁定已全部落地到下方 REQ；原「Future Requirements / 待产品决策」段整体退役，其中未纳入本期的仅剩「CI PG 双跑」一条前置依赖（见 Out of Scope）。

## 范围裁定（用户 2026-11-09 最终版）

- **纳入：6 项 LOCKED / 7 个子项全覆盖**（③ 拆 A/B）：① P1-5 侧信道（A+B）、② 5 项豁免面、③-A `RightCast:"text"`、③-B PG 专属 CTE、④ operlog e2e 静态门禁、⑤ `invalidateCache` no-op、⑥ `pbkdf2SM3` 契约。
- **不纳入**：79 条人工验证跟踪表（保持 0 勾选）、FIX-01/04/09/13 验证类残余、v1.54 新增 7 项 Known Gaps（GAP-01~07）、CI 新建 PG service（登记为前置依赖）。
- **FUTURE-01 随 ① 根治一并关闭**，不再单列挂账。
- **Phase 编号**：延续 v1.54（结束于 144），从 **145** 续编。
- **跳过领域研究**：7 个子项均带本仓 `文件:行号` 实证与现成改写范本，无新领域能力需调研（沿用 v1.53/v1.54 先例）；根因报告即研究输入。
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
- [ ] **DIALECT-04**（防复发机制）: ① `.golangci.yml` 启用 `forbidigo` 禁方言 token（`::text|::uuid|DISTINCT ON|REGEXP_REPLACE|NULLS LAST|ILIKE|ON CONFLICT`，存量先白名单逐步收敛；不得一刀切禁全部 raw SQL——406 文件含 `.Raw(`）② 视预算抽 `dialect` 语义原语（`CastToText` / `Top1PerGroup` / `NormalizeMac`），配置里禁止出现 SQL 片段 ③ **CI 注入 `TEST_PG_DSN` 双跑登记为前置依赖**（用户裁定本期不建 `services: postgres`），在 REQ/SUMMARY 显式写「需新建 PG service + 迁移引导后方可升级」，**不伪造双跑**。

### AUDIT — operlog 静态门禁根治（④）

- [ ] **AUDIT-01**: 把 `scripts/operlog_e2e_verify.sh:68-151` 的**静态段从 shell 拆出改写成 Go 单测**并纳入 `go test ./internal/...`（`ci.yml:62-63` 天然硬门禁）：① `operlog.Record|RecordWithBody` 全 `internal/` 调用数 ≥290 ② `sensitiveKeys` 条目 ≥17 ③ `internal/api/v1/**/*_handler.go` 的 `*Handler` 方法文件逐文件覆盖扫描（0 命中且不在 `READONLY_ALLOWLIST` → FAIL）。可复用 `internal/utils/operlog/regression_test.go`（`go/ast` 锁 25 常量）与 `coverage_test.go`（脱敏）现成基建，补约 80-120 行 `go/parser` 扫描。
- [ ] **AUDIT-02**: 验收「零 `ADMIN_USER`/`ADMIN_PASSWORD`/`DEV_MODE`/`SKIP_E2E`/`SKIP_LIVE`、零 running backend、零 DB、**任意 cwd** 下静态 3 项断言真实执行且失败会红」；LIVE 段保留 shell、缺凭据仍 `exit 1`（恢复脚本注释声明的语义，不改 LIVE 行为）。
- [ ] **AUDIT-03**: 顺带处置两份 `operlog_e2e_verify.sh` 重复副本（仅 1 行注释不同）与 `scripts/README.md` 登记（AGENTS.md 目录治理）；**不做** shell 整体替换为 Go e2e（会把 CI 硬门禁与需凭据 live 重新耦合，复现同类缺陷）。
  - **必须同时修 cwd 锚定**：当前 Go driver 未设 `cmd.Dir`、脚本用相对仓库根路径，包子目录下 `CALLS=0` / 目标文件不存在 → 仅挪门禁顺序连缓解都不算（假绿）。

### CACHE — 缓存失效契约根治（⑤）

- [ ] **CACHE-01**: 删除死键 `CacheKeyReconciliationExceptionRuleList` / `GetReconciliationExceptionRuleListKey()` 与空函数体 `invalidateCache()`（`reconciliation_exception.go:578-580`），消除「定义缓存键却从不读写 + 失效函数为空」的悬空契约；4 个写路径（Create/Update/Delete/ImportFromExcel）的调用点同步清理。
- [ ] **CACHE-02**（**用户裁定：必做，不允许 5min 陈旧**）: 补**真实陈旧点**的联动失效：`reconciliation_service.go:821-834` 的 `GetByWorkstation` 5min TTL 缓存中 `exceptionHit/AppliedActions` 随例外规则变更陈旧 —— 规则写成功后按同文件 `InvalidateWorkstationHealth`（`cache_keys.go:129-134`，接 cache 参数、nil-safe、真调 Delete）范式广播失效，或把 exception 字段从缓存结果中剥离；**保留 5min 缓存不动其性能价值，但要求写后立即一致**。
- [ ] **CACHE-03**: 验收「写后读必一致」（List/GetByID/MatchTest/MatchException/DetectLayer3/`GetByWorkstation` 同进程立即反映新规则）、`Cache.Type=memory`（`*MemoryCache`）与 `redis`（`*MultiLevelCache`）两种配置行为一致、失效动作可观测且失败可传播（返回 error 或日志/metric，禁止空函数）。
  - **明确作废**触发条件「项目接入 Redis 后补 `core.Cache.Delete(ctx, 死键)`」——L1/L2 都支持 Delete，该处方删的是从没人写过的键，且无 Redis 时同样不生效。
  - **不推荐**改全局 `CacheProvider` 接口禁 no-op（20+ 消费文件/大量 mock 连锁改，未命中真实陈旧点，仅为缓解）。

### CRYPTO — KDF 契约根治（⑥）

- [ ] **CRYPTO-01**: `pbkdf2SM3`（`password.go:69-97`）把未定义行为变定义行为——`keyLen<=0` / `keyLen>32` / `iterations<=0` 显式返回 `error`（项目禁 panic），杜绝 `result[32:]` 静默全零弱密钥；`HashPassword`/`VerifyPassword` 同步接错误。
- [ ] **CRYPTO-02**（**用户裁定：免盘点，直接上守卫**）: `VerifyPassword:149` 当前用 `len(hashBytes)` 作 keyLen（裁定「全部调用点 32」不成立）——显式校验 `len(hashBytes)==32`，非 32 字节哈希属损坏/伪造数据，按产品决策报错或返回 false，不再喂给 KDF。代码实证所有写入路径均经 `HashPassword` 产出恒 32 字节 `$sm3$`，**无需生产盘点、无需迁移**；仅在上线说明提示「人工直接改库的历史脏数据」风险（代码层不产生）。
- [ ] **CRYPTO-03**: **向后兼容硬约束**——`keyLen=32` 时输出逐字节不变；新增黄金向量测试（固定 password/salt/iterations 固化 hex，锁重构前后一致，不能只靠 Hash↔Verify 自洽），存量 `$sm3$<iter>$<salt>$<32字节hash>` 与 600k 默认 / 100k 旧哈希无需迁移；`$sm3$` 格式、`DefaultPasswordConfig`、25 个 operlog 常量均不变。
- [ ] **CRYPTO-04**（可选能力扩展，非消除当前隐患的必要条件）: 若产品确认未来需要 >32 派生（当前**无** SM4/会话/token 派生用途），再补完整 PBKDF2-SM3 多块循环（INT32_BE(i)、每块独立 `New/Reset`、`dk[:keyLen]` 截断、`make+append` 避免 `:75` slice 别名），用独立 oracle `golang.org/x/crypto/pbkdf2.Key(sm3.New,...)` 比对 `keyLen=33/64/65`、断言 `DK(64)[0:32]==DK(32)`。**守卫仍为主防线**，补循环不叠加守卫只算缓解。

### SIDECHANNEL — P1-5 聚合计数侧信道根治（①，用户 2026-11-09 裁定纳入）

> **根治定义**：任何非 All 档用户，通过单请求或多次请求组合，都无法从响应体（含状态码、任何数值/序号/数组长度）推导出「可见集之外的资源规模」或「越界资源是否存在/有多少」。
> **方案 A+B 组合**（用户裁定）：A = 响应体去规模化（信息论根治），B = 统一 ScopedDB 防回归（见 `ARCH-01` / `SIDECHANNEL-04`）。只做 A 不做 B，未来新增端点仍可能漏挂；只做 B 不做 A，响应体仍可显式算「全库−可见」差值。

- [ ] **SIDECHANNEL-01**（A-响应体）: 删除 `BackfillDeptResult` 的 `ScopeTotal` / `ScopeVisible` / `ScopeOutOfScope` / `ScopeDeptTotal`（`workstation_service.go:110-134`）及全库派生的 `SkippedNoUser` / `SkippedBadUserDept`（`:681-682`）；**去掉 `workstation_service.go:666-669` 的全库 `countQ`**（或改为带 `visibleCond`），消除「一条带 scope 的 candidateQ + 一条不带 scope 的 COUNT + 相减」双口径；`ToFill`/`ToCorrect`（可见集内）可保留。
- [ ] **SIDECHANNEL-02**（A-导入/审计边界）: `ImportResult.Errors[].Row` 实际行号（`excel_service.go:62-67`）与 `ScopeOutOfScope`（`:83`）不回显越界行位置与数量，越界行改为静默跳过或仅记录到服务端 operlog；`ScopeWriteOperParam`（`scope_write.go:730-738`）的 `requested/visible/outOfScope` **保留在服务端审计链但不进入 HTTP 响应**。
- [ ] **SIDECHANNEL-03**（A-前端/验收/All 档）: 作废 132-SUMMARY.md:74 的四类「本次范围：…（共 N 条）」文案与 `useBackfillDept.tsx` 确认框（`scopeWriteBackfillSnapshot`），改为不含全库规模的可见集内表述；**All 档响应形状/数值零回归**；新增「差分攻击测试」：构造「全库 N 行、可见 M 行」，用两个相邻 scope（Dept / DeptChild）各请求一次，断言响应体所有数值字段仅随 M 变化、不随 N−M 变化，重复请求不得通过 `Row`/数组长度/`total` 枚举出 N；重跑 132/133 backfill 相关 UAT。
- [ ] **SIDECHANNEL-04**（B-防回归，与 `ARCH-01` 同 phase）: 所有 `COUNT(*)` / `SUM` 与明细查询同源——8 个 `/statistics`、分页 `total`、`counts-by-floor`（`router.go:914`）、`dept-options`、`workstations-all`；用 `pg_contract_test.go` + SQLite 双跑，对同一 `ScopeResult` 断言 COUNT 结果与 `List` 行数一致；operlog 审计链不降级。

### ARCH — 统一 ScopedDB（① 的 B 组成 + ② 机制化，**用户裁定升级为必需**）

- [ ] **ARCH-01**: `DataScopePermission` 注入 `ScopeResult`（已有 `pkg/datascope` 单一真实源）→ **统一 ScopedDB**：GORM `Scopes` 强制唯一查询起链点（禁止裸 `s.db.WithContext(ctx)` 起链）+「资源归属注册表」（引用 `effectiveOrgExpr` / 归属链 EXISTS 骨架 / `dept_id` 直挂，禁止内联复制，符合项目「单一真实源」约定）+ 内建 `scopeCacheKey(ctx)` → **响应契约层**禁止任何「全库规模 − 可见规模」回显。
  - **同时解 ①②**：② 的 5 类端点在注册表登记后自动获得可见集谓词（修「中间件挂了没接线」）；① 的任何 `COUNT(*)` 从 ScopedDB 起链后天然同源。
  - **硬约束**：触及 `base.GORMRepository` 全仓基座（485 Go 文件）与 system/vdi 零回归保证，需全量 `go build ./...` + `go test ./...` 守护；**必须保留「未注入 ctx → All 兜底」短路**（`scope_filter.go:37-39`、`datascope.go:6`）；`EXISTS (?) + CAST AS TEXT` 已验证双方言。
  - **不可行方向（勿立项）**：DB 视图（可见集每用户动态、无法参数化）、PG RLS（SQLite 无 RLS、Go 共享连接池串味需每请求 `SET LOCAL` 独占事务、双方言两套安全模型）、按用户分表（`org_id` 可变、与 UUID 单表模型冲突）。

## Out of Scope

| Feature | Reason |
|---|---|
| 79 条人工验证跟踪项的实际执行与勾选 | 需生产 PG 只读 / 写授权 / 现场真机 / DBA 回传；保持 0 勾选，AI 不擅自写生产数据 |
| FIX-01 实机 AD UAT / FIX-04 真机逐型号取证 / FIX-09 HTTP P99 压测 / FIX-13 生产库 migration 236/237 | 验证类残余，不伪造闭环 |
| v1.54 新增 7 项 Known Gaps（GAP-01~07） | 已在 `STATE.md ## Deferred Items` v1.54 行登记，属后续候选；本里程碑专注 LOCKED 根治 |
| **CI 新建 `services: postgres` + 注入 `TEST_PG_DSN` 双跑** | 用户 2026-11-09 裁定本期不建，登记为 **DIALECT-04③ 前置依赖**；不伪造双跑 |
| PG RLS / DB 视图 / 按用户分表 | Go+GORM+SQLite/PG 双方言下不可行（见 ARCH-01） |
| shell 整体替换为 Go e2e / 加 `isPostgres` 方言分支 / 加告警通道代替 ① 根治 / 「接 Redis 后补 Delete」/ 仅调门禁顺序 | 均为补丁或无效处方，按用户「不打补丁」标准排除 |
| 新领域能力、新外部依赖 | 7 个子项均有本仓现成范本与实证，零新依赖 |
| FUTURE-01 单独挂账 | 用户裁定随 ① 根治一并关闭，不再单列 |

## Constraints (inherited + v1.55 新增)
- Handler-Service；service 返回 `error`；`response.Success()` / `response.Error()`（禁止裸 `c.JSON`）；写操作 success path 在 `response.Success` **之前** `operlog.Record(...)`；敏感端点 `RecordWithBody`。
- `0=正常 1=停用`；UUID 主键；**SQLite+PG 双方言**（纯 GORM / `CAST AS TEXT` / 禁 `::uuid`/`ILIKE`/裸 `ON CONFLICT`）；PG contract 门控变量名 `TEST_PG_DSN`（本期仍为前置依赖）。
- 前端 wrapped API（`@/lib/api.ts`）、Zustand、禁止裸 axios、`useEffect` 依赖稳定。
- Go 单测就近 `*_test.go`；集成 `tests/integration/`；e2e `tests/e2e/`；前端 `*.test.ts(x)`。
- 构建产物只落 `out/`；Excel 模板动态生成、禁止落盘 `.xlsx`；`data/` 不入库。
- **向后兼容**：`keyLen=32` KDF 输出逐字节不变、25 个 operlog 常量值不重排、② 响应形状不变、**① All 档响应形状/数值零回归**、`datascope` 未注入 ctx → All 兜底短路不变。
- **① 专项**：132-SUMMARY.md:74 四类前端文案与 `useBackfillDept.tsx` 确认框作废需产品/QA 确认；132/133 backfill 相关 UAT 作废重跑窗口由 QA 裁定；FUTURE-01 在 ① 落地后由 Lead 同步清理 4 处引用。
- 回归基线：`go build ./...` + `go test ./...` + `tsc` + `lint:ci`（含新增 forbidigo） + `vitest` 全绿；ARCH-01 改动后必须跑全量矩阵。

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
| SIDECHANNEL-01 | — | Pending |
| SIDECHANNEL-02 | — | Pending |
| SIDECHANNEL-03 | — | Pending |
| SIDECHANNEL-04 | — | Pending |
| ARCH-01（必需） | — | Pending |

**Coverage:** 24 REQ 草案（23 必需 + CRYPTO-04 可选）/ 0 mapped（待 ROADMAP）/ **6 项 LOCKED / 7 个子项 100% 覆盖**（① ② ③-A ③-B ④ ⑤ ⑥），① 的用户裁定已纳入。

### 建议的 Phase 划分（供 gsd-roadmapper 输入，非定案）
| Phase | 主题 | REQ | 文件面（互斥检查用） |
|---|---|---|---|
| 145 | AUDIT 静态门禁 Go 化 | AUDIT-01/02/03 | `internal/utils/operlog/*_test.go`、`scripts/operlog_e2e_verify.sh`、`tests/e2e/operlog_e2e_verify.sh`、两 driver、`scripts/README.md`、`.github/workflows/ci.yml` |
| 146 | CRYPTO KDF 契约 | CRYPTO-01/02/03/04 | `internal/core/security/password.go` + `*_test.go` |
| 147 | CACHE 失效契约 | CACHE-01/02/03 | `internal/services/asset/reconciliation_exception*.go`、`reconciliation_service.go`、`cache_keys.go` |
| 148 | DIALECT 双方言 | DIALECT-01/02/03/04 | `excel_export_config/service*.go`、`workstation_device_service.go` + 4+ 测试、`.golangci.yml` |
| 149 | SCOPE 豁免面收口 | SCOPE-01/02/03/04 | 5 个 `*_service.go`、`scope_entities.go`、`internal/api/router.go`、`useFloorPlanEditor.ts`、`133-UAT.md` |
| 150 | ① 侧信道根治 A（响应体去规模化） | SIDECHANNEL-01/02/03 | `workstation_service.go`、`excel_service.go`、`scope_write.go`、`useBackfillDept.tsx`、132/133 UAT |
| 151 | 统一 ScopedDB（B，① 防回归 + ② 机制化） | SIDECHANNEL-04 + ARCH-01 | `base/GORMRepository`、`pkg/query`（新）、8 个 service 起链处、`scope_filter.go` |

> **并行冲突面**：W1 候选 `145 ∥ 146 ∥ 147`（文件面互斥）。`148 / 149 / 150` 均动 `internal/services/operations/**`（148 动 excel_export/workstation_device，149 动 wall/door/text/photo/alias + router，150 动 workstation_service/excel_service + 前端），须产出**函数级归属表**否则降级串行。`151` 触及全仓基座且依赖 148/149/150 的经验，**必须最后执行**；① 的 A（150）与 B（151）按用户裁定均需完成，不伪造「统一架构已落地」。

---
*Draft defined: 2026-11-09 after 用户裁定「要」整理 v1.55 REQUIREMENTS 草案*
*Updated: 2026-11-09 — 用户拍板 5 项决策（① 纳入根治 / FUTURE-01 随 ① 关闭 / CI-PG 登记前置依赖 / 密码免盘点 / 例外缓存立即一致），草案已按裁定更新；待 `/gsd-new-milestone` 转为权威 REQUIREMENTS.md*
