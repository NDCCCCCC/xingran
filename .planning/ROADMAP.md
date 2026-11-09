# Roadmap: RuoYi-Next 运维管理系统

## Current Milestone: v1.55 LOCKED 残余根治 (Locked Residual Root-Cause Fix)

**Status:** ✅ SHIPPED 2026-11-09 · INITIATED/ROADMAP CREATED 2026-11-09 | Phases **145-151**（编号延续 v1.54 结束于 144，**不重置**）| **23 REQ / 7 Phase / 100% 映射 / 100% 交付**（CRYPTO-04 按硬 gate 合规跳过并登记；收档计数修正：立项期「24 REQ」把 optional 标注误计入，逐组实证 = 23） | **W1 `145 ∥ 146 ∥ 147` → W2 `148 ∥ 149 ∥ 150`（函数级归属表凭证）→ W3 `151` 最后执行，全部完成** | **① 的 A(150)+B(151) 均完成 → P1-5 侧信道根治达成；6 项 LOCKED / 7 子项全部从根源清偿** | 全量回归闸门全绿（go build / go test ./... / tsc / lint:ci ALL PASS / 全量 vitest / check-route-registry 713 零漂移）

**Goal:** 把 v1.52 收档时以「已接受 / 仅登记」名义挂账的 6 项 LOCKED 残余（7 个子项，③ 拆 A/B）从根源上修掉（而非打补丁 / 加告警 / 加方言分支），使约束从「靠约定」升级为「靠机制」；按用户 2026-11-09 裁定允许推翻 ① 的「范围快照回显」产品语义，① 纳入 A+B 组合根治。

**事实基线:** `.planning/REQUIREMENTS.md`（**权威**，24 REQ / 5 项裁定 / Constraints / 建议 Phase 表 / Traceability）｜`.planning/notes/261109-v154-locked-root-cause-analysis.md`（3 个并行只读 subagent + Lead 交叉核对的根因判定、方案对比表、「不算根治」边界）｜`.planning/notes/261109-v155-requirements-draft.md`（5 项决策已拍板的草案原文与产品语义说明）｜`.planning/PROJECT.md ## Current Milestone: v1.55`｜`.planning/notes/261108-v153-locked-residuals.md`（6 项 LOCKED 裁定原文，本期只读不动）。

**范围裁定（用户 2026-11-09 最终版）:** 纳入 **6 项 LOCKED / 7 子项全覆盖**（① ② ③-A ③-B ④ ⑤ ⑥）；DECISION-① 允许重定语义，① 按 A（150 响应体去规模化）+ B（151 统一 ScopedDB 防回归）组合根治；FUTURE-01 随 ① 关闭；CI-PG 本期不建 `services: postgres`、`TEST_PG_DSN` 双跑登记为前置依赖；存量密码免盘点直接上守卫；例外缓存要求立即一致。跳过领域研究、零新依赖。79 条人工验证跟踪表保持 0 勾选、FIX-01/04/09/13 验证类残余与 v1.54 新增 7 项 GAP 不纳入、不伪造闭环。

### Phases

- [x] Phase 145: AUDIT 静态门禁 Go 化 (AUDIT-01/02/03) — operlog e2e 静态段 Go 单测化 + cwd 锚定 + CI 硬门禁 — **completed 2026-11-09**（`145-SUMMARY.md`）
- [x] Phase 146: CRYPTO KDF 契约 (CRYPTO-01/02/03/04) — pbkdf2SM3 守卫 + `len==32` 校验 + 黄金向量（CRYPTO-04 optional 按硬 gate 合规跳过） — **completed 2026-11-09**（`146-SUMMARY.md`）
- [x] Phase 147: CACHE 失效契约 (CACHE-01/02/03) — 删死键/空函数 + `GetByWorkstation` 例外变更立即一致联动失效 — **completed 2026-11-09**（`147-SUMMARY.md`）
- [x] Phase 148: DIALECT 双方言 (DIALECT-01/02/03/04) — 删 `RightCast` + PG CTE 改 Go 层合并 + forbidigo 防复发 + CI PG 双跑前置登记 — **completed 2026-11-09**（`148-SUMMARY.md`）
- [x] Phase 149: SCOPE 豁免面收口 (SCOPE-01/02/03/04) — 5 项豁免零迁移收口 + 补挂 rooms/photos 中间件 + 保存侧裁剪 + UAT 重跑登记 — **completed 2026-11-09**（`149-SUMMARY.md`）
- [x] Phase 150: ① 侧信道根治 A — 响应体去规模化 (SIDECHANNEL-01/02/03) — 信息论根治：推翻「范围快照回显」产品语义 — **completed 2026-11-09**（`150-SUMMARY.md`）
- [x] Phase 151: 统一 ScopedDB / B — ① 防回归 + ② 机制化 (SIDECHANNEL-04 + ARCH-01) — **completed 2026-11-09**（`151-SUMMARY.md`；A+B 均完成，① 根治达成）

**依赖拓扑:** `W1: 145 ∥ 146 ∥ 147`（文件面互斥、零依赖并行）→ `W2: 148 ∥ 149 ∥ 150`（均动 `internal/services/operations/**`，以本文件「函数级归属表」为并行许可凭证；`internal/api/router.go` 的 149→151 顺序硬登记）→ `W3: 151`（触及 `base.GORMRepository` 全仓基座、依赖 148/149/150 经验，**必须最后执行**）。任一 W2 phase 实测出现跨 phase 写交集，则整组**显式降级为串行**，不得带冲突并行。

**REQ → Phase 映射（100% 覆盖、无孤儿无重复）：** 145=AUDIT-01/02/03 ｜ 146=CRYPTO-01/02/03/04(optional) ｜ 147=CACHE-01/02/03 ｜ 148=DIALECT-01/02/03/04 ｜ 149=SCOPE-01/02/03/04 ｜ 150=SIDECHANNEL-01/02/03 ｜ 151=SIDECHANNEL-04+ARCH-01。

### 各 Phase 详情

#### Phase 145: AUDIT 静态门禁 Go 化 (AUDIT-01/02/03)

- **需求映射:** AUDIT-01, AUDIT-02, AUDIT-03（唯一；对应 LOCKED ④）
- **文件面:** `internal/utils/operlog/*_test.go`（复用 `regression_test.go` 的 `go/ast` 25 常量锁 + `coverage_test.go` 脱敏，新增约 80-120 行 `go/parser` 扫描）、`scripts/operlog_e2e_verify.sh`、`tests/e2e/operlog_e2e_verify.sh`、两个 Go driver、`scripts/README.md`、`.github/workflows/ci.yml`。
- **实现要点:** 把 `scripts/operlog_e2e_verify.sh:68-151` 的**静态段从 shell 拆出改写为 Go 单测**并纳入 `go test ./internal/...`（`ci.yml:62-63` 天然硬门禁）；**必须同时修 cwd 锚定**（Go driver 设 `cmd.Dir`、静态扫描用仓库根绝对路径）；LIVE 段保留 shell、缺凭据仍 `exit 1`。
- **成功标准（可观察）:**
  1. 零 `ADMIN_USER`/`ADMIN_PASSWORD`/`DEV_MODE`/`SKIP_E2E`/`SKIP_LIVE`、零 running backend、零 DB、**任意 cwd** 下 `go test ./internal/...` 的静态 3 项断言真实执行且失败会红：① `operlog.Record|RecordWithBody` 全 `internal/` 调用数 ≥290 ② `sensitiveKeys` 条目 ≥17 ③ `internal/api/v1/**/*_handler.go` 的 `*Handler` 方法逐文件覆盖扫描（0 命中且不在 `READONLY_ALLOWLIST` → FAIL）。
  2. 在仓库根与任意包子目录分别跑同一 driver，`CALLS` 计数均 ≥290、目标 handler 文件均存在，不再出现包子目录 `CALLS=0` / 目标文件不存在（cwd 锚定修复的红绿变异验证）。
  3. 故意移除一处 `operlog.Record` 或删一个 `sensitiveKeys` 条目，CI 的 `go test ./internal/...` 立即红灯；`ci.yml` 真实覆盖新静态单测（不再「从未真正当过 CI 门禁」）。
  4. 两份 `operlog_e2e_verify.sh` 重复副本（仅 1 行注释不同）归并为单一权威并在 `scripts/README.md` 登记；LIVE 段仍在 shell、缺凭据 `exit 1` 行为不变（恢复脚本注释声明的语义）。
  5. **不做** shell 整体替换为 Go e2e（静态断言确认 LIVE 段保留在 shell，避免把 CI 硬门禁与需凭据 live 重新耦合）。

#### Phase 146: CRYPTO KDF 契约 (CRYPTO-01/02/03/04)

- **需求映射:** CRYPTO-01, CRYPTO-02, CRYPTO-03, **CRYPTO-04（optional）**——可选能力扩展，非消除当前隐患的必要条件；对应 LOCKED ⑥
- **文件面:** `internal/core/security/password.go` + 就近 `*_test.go`（`password_owasp_test.go` 等）。
- **实现要点:** `pbkdf2SM3` 把未定义行为变定义行为（`keyLen<=0` / `>32` / `iterations<=0` 返回 `error`，项目禁 panic）；`HashPassword`/`VerifyPassword` 同步接错误；`VerifyPassword` 显式校验 `len(hashBytes)==32`；黄金向量锁 `keyLen=32` 逐字节不变；CRYPTO-04 仅在产品确认未来需要 >32 派生时补完整多块循环 + 独立 oracle 比对，**守卫仍为主防线**。
- **成功标准（可观察）:**
  1. `pbkdf2SM3` 对 `keyLen<=0` / `keyLen>32` / `iterations<=0` 显式返回 `error`（不 panic）；变异测试断言 `keyLen=33/64` 不再静默返回 `result[32:]` 全零弱密钥。
  2. `VerifyPassword` 对非 32 字节 `hashBytes` 显式报错 / 返回 false，不再把 `len(hashBytes)` 作 keyLen 喂给 KDF；人工构造 64 字节哈希的登录用例被拒绝。
  3. `keyLen=32` 黄金向量（固定 password/salt/iterations 固化 hex）重构前后**逐字节一致**；存量 `$sm3$<iter>$<salt>$<32字节hash>`（600k 默认 + 100k 旧哈希）Hash↔Verify 全绿、无需迁移、**无需生产盘点**（全库写入均经 `HashPassword`）。
  4. **CRYPTO-04（optional）**：若启用，完整多块循环（`INT32_BE(i)` / 每块独立 `New/Reset` / `dk[:keyLen]` 截断 / `make+append` 避免 `:75` slice 别名）与独立 oracle `golang.org/x/crypto/pbkdf2.Key(sm3.New,...)` 比对 `keyLen=33/64/65` 一致，并断言 `DK(64)[0:32]==DK(32)`；不启用时该能力缺席**不算 ⑥ 未根治**（守卫已消除隐患）。
  5. `$sm3$` 格式 / `DefaultPasswordConfig` / 25 个 operlog 常量值不重排；`go test ./internal/core/security/` 全绿；仅在上线说明提示「人工直接改库的历史脏数据」风险（代码层不产生）。

#### Phase 147: CACHE 失效契约 (CACHE-01/02/03)

- **需求映射:** CACHE-01, CACHE-02, CACHE-03（唯一；对应 LOCKED ⑤）
- **文件面:** `internal/services/asset/reconciliation_exception*.go`、`reconciliation_service.go`、`cache_keys.go` 及就近 `*_test.go`。
- **实现要点:** 删死键 `CacheKeyReconciliationExceptionRuleList` / `GetReconciliationExceptionRuleListKey()` 与空函数体 `invalidateCache()`（`reconciliation_exception.go:578-580`），4 个写路径调用点同步清理；按 `InvalidateWorkstationHealth`（`cache_keys.go:129-134`，接 cache 参数、nil-safe、真调 `Delete`）范式给 `GetByWorkstation` 5min TTL 缓存补「例外规则变更 → 广播失效」联动（或把 exception 字段从缓存结果剥离）；**保留 5min 缓存性能价值，但要求写后立即一致**（用户裁定必做、不允许 5min 陈旧）。
- **成功标准（可观察）:**
  1. 静态断言全仓零引用死键 / helper / `invalidateCache()`；4 个写路径（Create/Update/Delete/ImportFromExcel）不再调用空函数，悬空契约消除。
  2. 规则写成功后 `GetByWorkstation` 的 `exceptionHit/AppliedActions` 立即反映新规则（写后读必一致，5min TTL 缓存保留但 exception 不再陈旧）；List/GetByID/MatchTest/MatchException/DetectLayer3 同进程立即一致。
  3. `Cache.Type=memory`（`*MemoryCache`）与 `redis`（`*MultiLevelCache`）两种配置行为一致；失效动作真调 `Delete`、nil-safe、失败可传播（返回 error 或日志/metric），**禁止空函数体**。
  4. 明确作废触发条件「项目接入 Redis 后补 `core.Cache.Delete(ctx, 死键)`」（注释/文档不留该处方）；**不改**全局 `CacheProvider` 接口（20+ 消费文件 / 大量 mock 不连锁改，仅为缓解）。

#### Phase 148: DIALECT 双方言 (DIALECT-01/02/03/04)

- **需求映射:** DIALECT-01, DIALECT-02, DIALECT-03, DIALECT-04（唯一；对应 LOCKED ③-A + ③-B）
- **文件面:** `internal/services/operations/excel_export_config.go`、`excel_export_service.go`、`workstation_device_service.go` + 4+ 个既有测试（`workstation_device_77_01_test.go` / `excel_export_chain_77_02_test.go` / `workstation_physical_diagnosis_test.go`）、`.golangci.yml`。
- **实现要点:** 删 `RightCast` 承载 `"::"` 方言片段的机制（`excel_export_config.go:18,351/355/359`），改语义枚举 + service 统一 `CAST(id AS TEXT)`（**单侧**即可，`IN ?` 参数是 Go 字符串）；`GetPhysicalDevices`（`:426-535`）与 `GetPhysicalDevicesByWorkstations`（`:2151-2234`）从根源消除 PG 专属 raw CTE，拆多次 GORM 链式查询 + Go 层归一化/去重/排序（复用 `diagnosePhysicalChain:701-799` 与 `physicalDevicesFetcher` 注入，保证「不存在第二套 recall 逻辑」）；改写 4+ 个「报错即预期」测试；`.golangci.yml` 启用 `forbidigo`；`TEST_PG_DSN` 双跑登记为前置依赖。
- **成功标准（可观察）:**
  1. `RightCast:"text"` 机制删除，`resolveAssociations` 的 3 个关联列在 SQLite 与 PG 均产出相同非空结果，不再 `logger.Warnf + continue` 静默降级为整列空；P1-8 式 DryRun 静态断言「导出路径 SQL 不含 `::`」+ SQLite 实跑断言均通过。
  2. `DISTINCT ON` / `REGEXP_REPLACE` / `NULLS LAST` / `id::text` 在两处 recall 路径全部归零（静态断言）；SQLite 与 PG 对同一 fixture 返回逐字段一致结果（含 MAC 归一化 / 置信度 / Top-1 口径）。
  3. 4+ 个「把 PG CTE 报错当预期」的既有测试改写为真实结果断言、`SkipSoftDelete:true` 两处语义保留；`go test ./internal/services/operations/` 全绿。
  4. `.golangci.yml` 的 `forbidigo` 禁 8 类方言 token（`::text|::uuid|DISTINCT ON|REGEXP_REPLACE|NULLS LAST|ILIKE|ON CONFLICT`），存量先白名单逐步收敛（**不得一刀切禁全部 raw SQL**——406 文件含 `.Raw(`）；`lint:ci` 接入后新增方言片段会红；视预算抽 `dialect` 语义原语（`CastToText`/`Top1PerGroup`/`NormalizeMac`），配置里禁止出现 SQL 片段。
  5. `TEST_PG_DSN` 双跑在 REQ/SUMMARY 显式登记为**前置依赖**（需新建 PG service + 迁移引导后方可升级），本期不建 `services: postgres`、**不伪造双跑**；**不加** `isPostgres` 方言分支、**不把** CTE 下沉双方言视图（带 `?` 参数，视图无法参数化）。

#### Phase 149: SCOPE 豁免面收口 (SCOPE-01/02/03/04)

- **需求映射:** SCOPE-01, SCOPE-02, SCOPE-03, SCOPE-04（唯一；对应 LOCKED ②）
- **文件面:** `internal/services/operations/` 下 `wall_service.go` / `door_service.go` / `floor_plan_text_service.go` / `room_photo_service.go` / `location_alias_service.go`、`scope_entities.go`、`internal/api/router.go`（**149 先改，151 后改或拆 commit**）、`xingran-react-frontend/.../useFloorPlanEditor.ts`、`133-UAT.md`。
- **实现要点:** 方案 D **零迁移**——5 类读端点在统一父资源可见性校验下按可见集裁剪（walls/doors/floor-plan-texts 经 `floor_id → ops_floors/ops_buildings` 归属链、rooms-photos 经 `room_id → serverRoom → building` 两级链，复用 `scope_entities.go` 的 `entityExistsSub` + `wrapScopeExists`；location-alias 按已有 `dept_id ∈ 可见集` 直挂；断链 fail-closed 空集）；`router.go:1174-1177` 的 rooms/photos 组补挂 `middleware.DataScopePermission(core)`（当前**连中间件都没挂**）；写端点越界 403 / 批量按 `scope_write.go:282-320` 的 `visibleIDsForEntityWrite` 裁剪；**必须修保存侧** `useFloorPlanEditor.ts:156-186` 的「整层拉取 → 整体覆写」不得让非超管误删自己看不到的几何元素。
- **成功标准（可观察）:**
  1. Dept 档用户请求 walls/doors/floor-plan-texts/rooms-photos/location-alias 列表只得到可见集内行；断链（floor/room 软删或不存在）fail-closed 返回空集；`GetByID` / count / primary / batch-delete 前置全走可见集校验。
  2. `internal/api/router.go:1174-1177` 的 rooms/photos 组补挂 `DataScopePermission` 后实测生效（从「连挂都没挂」到注入 `ScopeResult`）；**Dept 档用户越界 `floorId`/`roomId` 返回空/403**、批量按 `visibleIDsForEntityWrite` 裁剪。
  3. 非超管在 Three.js 编辑器保存整层几何后，自己看不到的几何元素不因「整体覆写」被误删（前后几何行数断言）；响应形状不变（只裁剪行，不改 `list`/`total`/字段）。
  4. PG + SQLite 双方言 403 矩阵与断链脏数据抽查断言一致；`133-UAT.md:79-91` 的 D1/D2「确认未收口」结论作废并登记重跑窗口；若这些端点将来加缓存，key 必含 `scopeCacheKey(ctx)`。
  5. **零迁移**（方案 D）：不新增 `org_id` 冗余列（方案 E）、不把几何合并进楼层 JSONB（方案 F）；`scope_write.go` 的 `visibleIDsForEntityWrite` **仅复用不改**（该文件归 150）；`scope_filter.go` 归 151、本 phase 不得触碰。

#### Phase 150: ① 侧信道根治 A — 响应体去规模化 (SIDECHANNEL-01/02/03)

- **需求映射:** SIDECHANNEL-01, SIDECHANNEL-02, SIDECHANNEL-03（唯一；对应 LOCKED ① 的 A）
- **根因定性（写进说明）:** 根因报告判定 ① **不是 SQL 漏 WHERE，而是产品有意「一条带 scope 的 `candidateQ` + 一条不带 scope 的全库 `countQ` + 相减」回显越界规模**（`workstation_service.go:662-677`），8 个 `/statistics` 已同源；这是**信息论冲突，不是代码缺陷**。**根治 = 推翻「范围快照回显」产品语义**（用户 2026-11-09 DECISION-① 已裁定允许）。A = 响应体去规模化（信息论唯一彻底解），B = 统一 ScopedDB 防回归（Phase 151）。只做 A 不做 B，未来新增端点仍可能漏挂；只做 B 不做 A，响应体仍可显式算「全库−可见」差值。**A(150)+B(151) 均需完成，不伪造「统一架构已落地」。**
- **文件面:** `internal/services/operations/workstation_service.go`、`excel_service.go`、`scope_write.go`、`useBackfillDept.tsx`、132/133 backfill UAT。
- **实现要点:** 删 `BackfillDeptResult` 的 `ScopeTotal/ScopeVisible/ScopeOutOfScope/ScopeDeptTotal`（`:110-134`）及全库派生 `SkippedNoUser/SkippedBadUserDept`（`:681-682`），去掉 `workstation_service.go:666-669` 的全库 `countQ`（或挂 `visibleCond`），消除双口径；`ToFill`/`ToCorrect`（可见集内）保留；`ImportResult.Errors[].Row` 行号与 `ScopeOutOfScope` 不回显越界行位置与数量（越界行静默跳过或仅入服务端 operlog）；`ScopeWriteOperParam`（`scope_write.go:730-738`）的 `requested/visible/outOfScope` 保留服务端审计链但不进 HTTP 响应；作废前端四类文案与确认框；**All 档响应形状/数值零回归**；新增差分攻击测试 + 重跑 132/133 UAT。
- **成功标准（可观察）:**
  1. `BackfillDeptResult` 不再含 `ScopeTotal`/`ScopeVisible`/`ScopeOutOfScope`/`ScopeDeptTotal`/`SkippedNoUser`/`SkippedBadUserDept`；`workstation_service.go:666-669` 的全库 `countQ` 删除或改为带 `visibleCond`（静态 + 响应体断言「无全库规模字段」）；`ToFill`/`ToCorrect` 保留。
  2. `ImportResult.Errors[].Row` 实际行号与 `ScopeOutOfScope` 不再回显越界行位置/数量，越界行静默跳过或仅入服务端 operlog；`ScopeWriteOperParam` 聚合留在审计链、HTTP 响应零回显（operlog 审计链不降级）。
  3. 132-SUMMARY.md:74 的四类「本次范围：…（共 N 条）」文案与 `useBackfillDept.tsx` 确认框（`scopeWriteBackfillSnapshot`）作废，改为不含全库规模的可见集内表述；**All 档响应形状/数值零回归**。
  4. **差分攻击测试**：构造「全库 N 行、可见 M 行」，用 Dept 与 DeptChild 两个相邻 scope 各请求一次，断言响应体所有数值字段**仅随 M 变化、不随 N−M 变化**；重复请求不得通过 `Row` / 数组长度 / `total` / 状态码枚举出 N；重跑 132/133 backfill 相关 UAT（作废重跑窗口见 Known Gaps）。

#### Phase 151: 统一 ScopedDB / B — ① 防回归 + ② 机制化 (SIDECHANNEL-04 + ARCH-01) — 必须最后执行

- **需求映射:** SIDECHANNEL-04, ARCH-01（唯一；ARCH-01 用户裁定由「可选」升级为**必需**）
- **根因定性（写进说明）:** ① 是**信息论冲突**：A(150) 只把当前响应体去规模化，**不能防回归**；B 用统一 ScopedDB 把「同源」从靠约定升级为靠机制（GORM `Scopes` 强制唯一查询起链点 + 资源归属注册表 + 响应契约层）。**① 的根治门槛不在代码，在产品是否放弃范围快照回显**——用户已裁定放弃，故 A+B 都做；**B 未落地前不得宣称「统一架构已落地」**（不伪造）。同时解 ②：5 类端点在注册表登记后自动获得可见集谓词（修「中间件挂了没接线」）。
- **文件面:** `internal/models/base/GORMRepository`（全仓基座，485 Go 文件受影响）、`pkg/query`（新，资源归属注册表）、8 个 service 起链处、`internal/services/operations/scope_filter.go`（**归 151**）、`internal/api/router.go`（**149 先改、151 后改或拆 commit**）。
- **实现要点:** `DataScopePermission` 注入 `ScopeResult`（已有 `pkg/datascope` 单一真实源）→ GORM `Scopes` 强制唯一查询起链点（**禁止裸 `s.db.WithContext(ctx)` 起链**）+ 资源归属注册表（引用 `effectiveOrgExpr` / 归属链 EXISTS 骨架 / `dept_id` 直挂，禁止内联复制，符合「单一真实源」约定）+ 内建 `scopeCacheKey(ctx)` → **响应契约层**禁止任何「全库规模 − 可见规模」回显；保留「未注入 ctx → All 兜底」短路。
- **成功标准（可观察）:**
  1. 所有 `COUNT(*)` / `SUM` 与明细查询同源：8 个 `/statistics`、分页 `total`、`counts-by-floor`（`router.go:914`）、`dept-options`、`workstations-all`；`pg_contract_test.go` + SQLite 双跑对同一 `ScopeResult` 断言 COUNT 结果与 `List` 行数一致（SIDECHANNEL-04）。
  2. service 起链处静态断言「零裸 `s.db.WithContext(ctx)` 起链」，全部经 ScopedDB；② 的 5 类端点在注册表登记后自动获得可见集谓词，rooms/photos 的「中间件挂了没接线」由机制兜住。
  3. 保留「未注入 ctx → All 兜底」短路（`scope_filter.go:37-39`、`datascope.go:6`）；system/vdi 零回归，`base.GORMRepository` 改造后**全量** `go build ./...` + `go test ./...` + `tsc` + `lint:ci`（含 forbidigo）+ `vitest` 全绿。
  4. 响应契约层断言任何端点响应体不含「全库规模 − 可见规模」派生数值；operlog 审计链不降级。
  5. **不可行方向勿立项**：DB 视图（可见集每用户动态、无法参数化）、PG RLS（SQLite 无 RLS、Go 共享连接池串味需每请求 `SET LOCAL` 独占事务、双方言两套安全模型）、按用户分表（`org_id` 可变、与 UUID 单表模型冲突）；`EXISTS (?) + CAST AS TEXT` 已验证双方言。

### 并行冲突面 / 函数级归属表（沿用 v1.53/v1.54 惯例）

**W1 可并行 `145 ∥ 146 ∥ 147`**（文件面互斥）：145 = `internal/utils/operlog/*_test.go` + `scripts/e2e` + `tests/e2e` + `scripts/README.md` + `.github/workflows/ci.yml`；146 = `internal/core/security/password.go` + `*_test.go`；147 = `internal/services/asset/reconciliation_exception*.go` / `reconciliation_service.go` / `cache_keys.go`。三者写集合两两交集 = ∅，按模块隔离并行。

**W2 `148 / 149 / 150` 都动 `internal/services/operations/**`**：按下列**函数级归属表**逐文件/逐函数登记独占写集合，确认两两交集 = ∅ 方可并行；若 `plan-phase` 实测出现跨 phase 写交集（例如 `excel_service.go` 与 `excel_export_service.go` 被误并、或 `scope_write.go` 被 149 反向修改），**整组显式降级为串行**，不得带冲突并行。

| 文件（写集合） | 函数 / 顶层符号（独占归属） | 归属 phase | 与同波其它 phase 交集 |
|---|---|---|---|
| `operations/excel_export_config.go` | `RightCast` 字段删除、语义枚举、`resolveAssociations` 列配置 | **148** | ∅ |
| `operations/excel_export_service.go` | `resolveAssociations`（`CAST(id AS TEXT)` 单侧、去 `Warnf+continue`） | **148** | ∅ |
| `operations/workstation_device_service.go` | `GetPhysicalDevices` / `GetPhysicalDevicesByWorkstations`（改 GORM 链 + Go 层合并）、复用 `diagnosePhysicalChain` / `physicalDevicesFetcher` | **148** | ∅ |
| `workstation_device_77_01_test.go` / `excel_export_chain_77_02_test.go` / `workstation_physical_diagnosis_test.go` 等 4+ 测试 | 「PG CTE 报错即预期」断言 → 真实结果断言 | **148** | ∅ |
| `.golangci.yml` | `forbidigo` 方言 token 规则（存量白名单） | **148** | ∅ |
| `operations/wall_service.go` / `door_service.go` / `floor_plan_text_service.go` / `room_photo_service.go` / `location_alias_service.go` | `List` / `GetByID` / count / primary / `BatchOperation`（挂 `entityExistsSub` + `wrapScopeExists`、按 `visibleIDsForEntityWrite` 裁剪） | **149** | ∅ |
| `operations/scope_entities.go` | `entityExistsSub` / `wrapScopeExists`（149 复用并按需扩展，**不把唯一收口入口改成 ScopedDB**） | **149** | ∅（与 `scope_filter.go` / 151 不同文件） |
| `internal/api/router.go`（rooms/photos 组 `:1174-1177`） | 补挂 `middleware.DataScopePermission(core)` | **149 先改** | 与 151 同文件 → 硬登记先后 |
| `useFloorPlanEditor.ts`（`:156-186`） | 保存侧 `BatchOperation` 可见集裁剪（不误删看不到几何） | **149** | ∅ |
| `133-UAT.md` | D1/D2 旧结论作废 + 重跑窗口登记 | **149** | ∅（150 改 132/133 backfill UAT 另一段，逐文件核对） |
| `operations/workstation_service.go` | `BackfillDeptFromUser` / `BackfillDeptResult`（删 4+2 字段）/ 全库 `countQ` | **150** | ∅ |
| `operations/excel_service.go` | `ImportResult`（去 `Errors[].Row` / `ScopeOutOfScope`） | **150** | ∅（**≠** 148 的 `excel_export_service.go`） |
| `operations/scope_write.go` | `ScopeWriteOperParam`（聚合留审计链不进 HTTP）；`visibleIDsForEntityWrite`（`:282-320`）**仅被 149 复用、不在 149 写集合** | **150** | ∅ |
| `useBackfillDept.tsx` | `scopeWriteBackfillSnapshot` 确认框、四类「本次范围：共 N 条」文案 | **150** | ∅ |
| `132/133 backfill UAT` | 作废重跑 + 差分攻击测试 | **150** | ∅ |

**W2 并行许可凭证：** 上表 148/149/150 写集合两两交集 = ∅（148 的 `excel_export_service.go` ≠ 150 的 `excel_service.go`；149 仅复用不写 `scope_write.go`；`scope_entities.go` 归 149、`scope_filter.go` 归 151），故 `148 ∥ 149 ∥ 150` 成立；任一格实测交集非空 → 整组降级串行。

**`151` 必须最后执行**：触及 `base.GORMRepository` 全仓基座（485 Go 文件）、依赖 148/149/150 的经验（双方言改写 / 豁免面接线 / 响应体去规模化），且 ① 的 A(150) + B(151) 按用户裁定**都需完成**，B 未落地前不得宣称「统一架构已落地」。

**全局唯一归属硬登记：**

| 文件 | 冲突来源 | 处置 |
|---|---|---|
| `internal/api/router.go` | 149 补挂 rooms/photos 中间件；151 ScopedDB 挂载 | **149 先改、151 后改或拆 commit**（同一文件不并发；151 在 149 commit 之后基于最新文件改） |
| `internal/services/operations/scope_filter.go` | 唯一收口入口，属 ① 的 B 基座 | **归 151**；149 只动 `scope_entities.go`，不得改 `scope_filter.go` |
| `.golangci.yml` | forbidigo 防复发 | **归 148** |
| `.github/workflows/ci.yml` | 145 的 AUDIT/④ CI 硬门禁（`go test ./internal/...`）；148 的 forbidigo 也动 lint 配置（经 `lint:ci` 接入 CI） | **归 145**：145 先落 `ci.yml` 的静态单测硬门禁与 `lint:ci` 调用；148 **只改 `.golangci.yml`、不动 `ci.yml`**。若 148 实测必须改 `ci.yml`（如新增 lint step），**在 145 落定后由 148 单独追加一个 commit**，不得与 145 并发改同一文件；两处改动的合并方式 = 145 预留步骤名、148 只提供 `.golangci.yml` 规则 |

### 关键约束（全局硬约束，沿用 REQUIREMENTS.md Constraints + 5 项裁定）

**Constraints（inherited + v1.55 新增）：**
1. **Handler-Service**；service 返回 `error`；`response.Success()` / `response.Error()`（禁止裸 `c.JSON`）；写操作 success path 在 `response.Success` **之前** `operlog.Record(...)`；敏感端点 `RecordWithBody`。
2. `0=正常 1=停用`；UUID 主键；**SQLite+PG 双方言**（纯 GORM / `CAST AS TEXT` / 禁 `::uuid`/`ILIKE`/裸 `ON CONFLICT`）；PG contract 门控变量名 `TEST_PG_DSN`（本期仍为前置依赖）。
3. 前端 wrapped API（`@/lib/api.ts`）、Zustand、禁止裸 axios、`useEffect` 依赖稳定。
4. Go 单测就近 `*_test.go`；集成 `tests/integration/`；e2e `tests/e2e/`；前端 `*.test.ts(x)`。
5. 构建产物只落 `out/`；Excel 模板动态生成、禁止落盘 `.xlsx`；`data/` 不入库。
6. **向后兼容**：`keyLen=32` KDF 输出逐字节不变、25 个 operlog 常量值不重排、② 响应形状不变、**① All 档响应形状/数值零回归**、`datascope` 未注入 ctx → All 兜底短路不变。
7. **① 专项**：132-SUMMARY.md:74 四类前端文案与 `useBackfillDept.tsx` 确认框作废需产品/QA 确认；132/133 backfill 相关 UAT 作废重跑窗口由 QA 裁定；~~FUTURE-01 在 ① 落地后由 Lead 同步清理 4 处引用~~ → **✅ v1.55 收档已清理（`REQUIREMENTS.md` 决策表 / 本表 Known Gaps / `STATE.md` Deferred Items / `261108-v153-locked-residuals.md:63`）**。
8. **零新依赖**；跳过领域研究（7 子项均带本仓 `文件:行号` 实证与现成改写范本，根因报告即研究输入）。
9. **全量回归闸门**：`go build ./...` + `go test ./...` + `tsc` + `lint:ci`（含新增 forbidigo）+ `vitest` 全绿；ARCH-01（151）改动后必须跑全量矩阵；**提交由 Lead 统一执行**（roadmapper 不 git commit）。

**5 项裁定（用户 2026-11-09，全部按推荐项）：**

| 决策 | 裁定 | ROADMAP 落点 |
|---|---|---|
| **DECISION-①**（决定性） | 允许重定语义，① 纳入根治（A+B 组合）：取消 `ScopeTotal/ScopeOutOfScope` 等回显、去掉全库 `countQ`、去导入 `Errors[].Row` 行号、作废 132-SUMMARY.md:74 四类「本次范围：共 N 条」文案 | Phase 150（A）+ Phase 151（B，`ARCH-01` 升级必需）；132/133 UAT 作废重跑 |
| **FUTURE-01** | ✅ v1.55 收档已关闭（随 ① 的 A+B 根治一并关闭，不再单独挂账） | 4 处引用已由 Lead 于 2026-11-09 收档清理完毕（见 Known Gaps 段） |
| **DECISION-CI-PG** | 本期不建 PG service，登记前置依赖，不伪造双跑 | Phase 148 的 DIALECT-04③：只做 `forbidigo` 门禁（XS）+ 视预算抽 dialect 语义原语；`TEST_PG_DSN` 双跑为前置依赖 |
| **DECISION-存量密码** | 免盘点，直接上守卫 | Phase 146 的 CRYPTO-02 直接加 `len(hashBytes)==32` 校验，无需迁移 |
| **DECISION-例外陈旧窗口** | 要求立即一致，做联动失效 | Phase 147 的 CACHE-02 必做（非可选）：规则写成功后 `GetByWorkstation` exception 字段立即一致 |

**「明确不算根治」边界（禁区，按用户「不打补丁」标准排除）：**
- 禁**告警通道代替 ①**（FUTURE-01 告警 / 粗粒度桶 `0/<10/10-100/100+` / 频率限制）——告警只检测利用行为，不关闭差分通道；
- 禁**方言分支**（`isPostgres` 两套 SQL 口径必分叉，是补丁不是根治）；
- 禁**「接 Redis 补 Delete」**——删的是从没人写过的死键，且无 Redis 时同样不生效，是无效处方；
- 禁**仅调门禁顺序**——cwd 锚定缺陷未解时连缓解都不算（假绿）；
- 禁 **shell 整体替换为 Go e2e**——把 CI 硬门禁与需凭据 live 重新耦合，复现同类缺陷；
- 禁 **RLS 与视图**（DB 视图无法参数化每用户可见集 / PG RLS 在 SQLite 无对应且共享连接池串味 / 按用户分表与 UUID 单表模型冲突）；
- 禁**伪造 PG 双跑**（本期 `TEST_PG_DSN` unset，contract 只做「skip 可观测」，零真实 PG 验证）；
- ⑥ 只补多块循环不加守卫 = 能力扩展（CRYPTO-04 optional），不算隐患根治。

**账务与人工验证（不伪造闭环）：**
- **79 条人工验证跟踪表保持 0 勾选**（`.planning/notes/261108-human-verification-tracker.md`），按 `notes/261108-uat-writeback-policy.md` 由人工执行后回写，AI 不擅自写生产数据；
- **LOCKED 裁定原文不动**：`.planning/notes/261108-v153-locked-residuals.md`（6 项裁定原文）与 `.planning/notes/261108-*` 归档件本期只读；**收档时按清偿结果更新**该裁定状态，不在立项期改写；
- 不要动 `.planning/milestones/` 归档件；不要创建 `.planning/phases/` 目录。

### Known Gaps（登记，非阻断）

| 项 | 边界 | 处置 |
|---|---|---|
| FIX-01 实机 AD/LDAP UAT | 需生产 AD 实机 | 验证类残余，不建 phase、不伪造闭环；79 条跟踪表承载 |
| FIX-04 三厂商物理参数真机逐型号取证 | 需现场真机 | 同上 |
| FIX-09 端到端 HTTP P99 压测 | 需真实压测环境 | 同上 |
| FIX-13 生产库 migration 236/237 | 需生产库写授权 | 待 DBA/人工执行，AI 不擅自写生产数据 |
| 79 条人工验证跟踪表 | 全部待执行、**0 勾选** | 按 `261108-uat-writeback-policy.md` 人工回写 |
| v1.54 新增 7 项 GAP（GAP-01~07） | 告警「标记已读」未实现 / 企微钉钉路线 B / `sys_object_id` 未落库 / 批量无自动回滚 / 发现导入顺带隐患 / e2e 归位 / `RequiresInterfaceView` 联动点 | 已在 `STATE.md ## Deferred Items` v1.54 行登记，属后续候选；本里程碑专注 LOCKED 根治 |
| **CI PG 双跑前置依赖** | `ci.yml` 无 `services: postgres`、`TEST_PG_DSN` unset，PG contract 只做「skip 可观测」 | 用户裁定本期不建；登记为 `DIALECT-04③` 前置依赖（需新建 PG service + 迁移引导后方可升级），**不伪造双跑** |
| ~~**FUTURE-01（P1-5 侧信道告警通道合并评估）**~~ | ~~触发条件「FIX-11 告警通道落地」在 v1.54 已成立，但未覆盖聚合计数面~~ | **✅ v1.55 收档已关闭**：① 改由 A(150 响应体去规模化)+B(151 统一 ScopedDB) 根治，告警通道合并评估不再需要；4 处引用已由 Lead 清理（`REQUIREMENTS.md` 决策表 / 本表 / `STATE.md` Deferred Items / `261108-v153-locked-residuals.md:63`），不再挂账 |
| **132/133 UAT 作废重跑窗口** | ① 推翻产品语义后，132-SUMMARY.md:74 四类文案与 `133-UAT.md:79-91` D1/D2「确认未收口」结论作废 | 由 QA 裁定重跑窗口与验收人；Phase 149/150 只登记、不擅自勾选完成 |

**v1.55 交付证据（7 份 SUMMARY + 2 份 UAT 载体）**：`.planning/phases/145-audit-go-audit-01-02-03/145-SUMMARY.md`、`146-crypto-kdf-crypto-01-02-03-04/146-SUMMARY.md`、`147-cache-cache-01-02-03/147-SUMMARY.md`、`148-dialect-dialect-01-02-03-04/148-SUMMARY.md`、`149-scope-scope-01-02-03-04/149-SUMMARY.md` + `149-UAT-RECHECK.md`、`150-a-sidechannel-01-02-03/150-SUMMARY.md` + `150-UAT-BACKFILL.md`、`151-scopeddb-b-sidechannel-04-arch-01/151-SUMMARY.md`。根因判定 `.planning/notes/261109-v154-locked-root-cause-analysis.md`；草案 `.planning/notes/261109-v155-requirements-draft.md`。

**v1.55 收档新增登记（非阻断，收档时点）**：trunk operlog 真实缺口（`port_write_trunk_handler.go:43-80` 成功路径无 `operlog.Record`，W2 未覆盖，待后续 quick）｜`computeByWorkstation:1074` append 缺陷｜两份 `operlog_e2e_verify.sh` 重复副本未物理合并｜forbidigo 能力边界（只匹配 AST 标识符，反向变异实际由 Go 源码静态断言承担）｜`room_photo`/`location_alias` 两 handler HTTP 透传口径｜`TEST_PG_DSN` 双跑前置依赖｜149/150 UAT 重跑窗口待 QA。逐项见 `STATE.md ## Deferred Items` 的 v1.55 行。

> **后续处置（2026-11-09 quick 261109-wdu，非本快照内容）**：A 类代码缺口全部收口 —— ① trunk operlog ② append 顺序 ③ 两份脚本物理合并到 `tests/e2e/` 唯一权威 ④ `physical_e2e_test.go` 归位按 AGENTS.md 登记「就近单测合规豁免、不迁移」 ⑤ 发现导入事务与 Save Error 均清偿（含红绿回归断言）；`user_id.(string)` 硬断言实证已不存在、属过时登记。**仍挂账**：forbidigo 能力边界、两 handler HTTP 透传口径、B 类需产品决策项、C 类验证/人工项（79 条 0 勾选）。终态以 `STATE.md ## Deferred Items` 最新行为准。

---

## Milestones（历史 SHIPPED 列表）

<details>
<summary>历史里程碑 SHIPPED（v1.20 ~ v1.54，折叠；非当前状态）</summary>

- ✅ **v1.54 真实代码缺口清偿（Code Gap Clearance）** — Phases 142-144 (shipped 2026-11-09, 7 REQ / 3 Phase / 100% 交付，W1 四 executor 并行 + Lead 集成) — 3 个 phase 详情见下方「✅ v1.54 ...」折叠段
- ✅ **v1.53 开放项全量清偿（Open Items Clearance）** — Phases 134-141 (shipped 2026-11-08, 24 REQ / 21 条开放项全覆盖) — [v1.53-ROADMAP.md](milestones/v1.53-ROADMAP.md) · [v1.53-REQUIREMENTS.md](milestones/v1.53-REQUIREMENTS.md)
- ✅ **v1.52 运维域数据权限统一收口（终态）** — Phases 130-133 (shipped 2026-11-07, 7 REQ；archived 2026-11-08) — [v1.52-ROADMAP.md](milestones/v1.52-ROADMAP.md) · [v1.52-REQUIREMENTS.md](milestones/v1.52-REQUIREMENTS.md)
- ✅ **v1.51 端口写生产级操作体验与平台质量收口** — Phases 121-129 (shipped 2026-11-02, 21 REQ) — [v1.51-REQUIREMENTS.md](milestones/v1.51-REQUIREMENTS.md)
- ✅ **v1.25 全站权限治理与样式收敛** — Phases 84-93 (shipped 2026-10-30)
- ✅ **v1.24 视觉回归修复与遗留债务清偿** — Phases 76-83 (shipped 2026-10-28)
- ✅ **v1.23 前端契约全量收尾与遗留清偿** — Phases 69-75 (shipped 2026-10-28)
- ✅ **v1.22 前端组件契约与设计系统对齐** — Phases 63-68.1 (shipped 2026-10-27)
- ✅ **v1.21 后端架构统一与接口解耦** — Phases 57-62 (shipped 2026-10-22)
- ✅ **v1.20 工位管理与空间可视化演进** — Phases 48-56 (shipped 2026-07-10)

</details>

---

## Phases（历史归档，SHIPPED）

<details>
<summary>✅ v1.54 真实代码缺口清偿 (Phases 142-144) — SHIPPED 2026-11-09</summary>

**Status:** ✅ SHIPPED 2026-11-09 · INITIATED/ROADMAP CREATED 2026-11-08 | Phases **142-144**（编号延续 v1.53 结束于 141，不重置）| **7 REQ / 3 Phase / 100% 映射 / 100% 交付** | **W1 四 executor 零依赖并行 + Lead 集成**（按模块隔离）| 全量回归闸门全绿（go build / go test ./... / tsc / lint:ci ALL PASS / 全量 vitest / check-permission-codes 双向零漂移 / check-route-registry）

**Goal:** 清偿 `.planning/STATE.md ## Deferred Items` 中 v1.53 收档时显式披露的 4 项真实代码缺口（发现设备真实导入 / 同步漂移告警通道 / 端口写批量物理参数 / 型号级能力矩阵），使「代码可修」残余真正落地为可运行、可测试的产品能力；不触碰 LOCKED 残余、不伪造人工验证闭环。

**事实基线:** `.planning/STATE.md ## Deferred Items`（v1.53 `code_gaps` / `verification_gaps` 两行）｜`.planning/MILESTONES.md:19-24`（v1.53 Known Gaps）｜`.planning/milestones/v1.53-phases/*-SUMMARY.md`（Phase 自报边界）｜3 个并行只读探索 subagent 的代码面事实 + 本仓实证｜`.planning/PROJECT.md ## Current Milestone: v1.54`。

**范围裁定（用户 2026-11-08）:** 仅 4 项纯代码缺口纳入（拆为 7 REQ）；验证类 3 项 + FIX-04 真机逐型号取证 + 79 条人工验证跟踪 + 6 项 LOCKED 残余不纳入（见下方 Known Gaps / `.planning/REQUIREMENTS.md` Future Requirements & Out of Scope）。跳过领域研究，零新依赖。

### Phases

- [x] Phase 142: 发现设备真实导入 (DISC-01/02) — stub → 真实写 `sys_network_device` — **completed 2026-11-09**（`142-SUMMARY.md`）
- [x] Phase 143: 端口写物理参数收口 (PORT-01/02) — 批量物理参数路径 + 型号级 `vendorCapabilities` 规则 — **completed 2026-11-09**（`143-SUMMARY.md`）
- [x] Phase 144: 同步漂移告警通道收口 (ALRT-01/02/03) — 前端告警页 / WS 广播 / 企微钉钉 webhook — **completed 2026-11-09**（`144-01-backend-SUMMARY.md` + `144-02-frontend-SUMMARY.md`；本 phase 拆为「后端」与「前端」两个并行 executor，文件面互斥）

**依赖拓扑:** `142 ∥ 143 ∥ 144`（W1 单波、三者零依赖并行，按模块隔离）；无 W2、无收口 phase。

#### Phase 142: 发现设备真实导入 (DISC-01/02)

- **需求映射:** DISC-01, DISC-02（唯一 `code_gaps` 权威登记项）
- **文件面:** `internal/services/device_discovery_service*.go`（`device_discovery_service.go:509-523` stub → 真实实现 + 新增 service 单测）、`internal/api/v1/network/discovery_handler.go`（缓存失效注入）、`internal/api/v1/network/discovery_*_test.go`（2 个既有测试文件删 stub 断言）、网络模块测试；**无需新 migration**。
- **实现要点:** 读 `sys_device_discovery_result` 按 `ip_address` 去重 → 软删恢复（`Unscoped` 命中同 ip 已软删行则 `deleted_at` 置空）→ 单事务批量写 `sys_network_device`；字段映射、部分成功/失败语义、错误逐层返回。
- **成功标准（可观察）:**
  1. 用户从一次发现任务执行「导入」后，`sys_device_discovery_result` 中按 `ip_address` 去重的设备真实出现在 `sys_network_device`，字段映射与发现结果一致；重复 ip 只产生 1 条网络设备记录（去重可用 SQL `COUNT` 断言）。
  2. 同 ip 已软删的 `sys_network_device` 行被软删恢复（`deleted_at IS NULL`）而非新建重复行；单事务批量写在「中途失败」用例下整批回滚，`sys_network_device` 零残留（SQLite 用例可证伪）。
  3. `ImportDiscoveredDevices` 不再 `return len(deviceIDs), nil`；新增 service 单测覆盖「空结果 / 全重复 / 软删恢复 / 事务回滚 / 错误逐层返回」5 类分支，既有 2 个测试文件的「简化实现」断言已删除并改为真实写入断言。
  4. success path 在 `response.Success(...)` 之前调用 `operlog.Record(...)`（`OperType=Import`），`discovery_handler.go` 注入缓存失效使网络设备列表即时可见；错误路径由 handler 用 `response.Error(...)` 包装；SQLite + PG 双方言 `go test ./internal/services/ ./internal/api/v1/network/` 均通过（uuid 双侧 CAST）。

#### Phase 143: 端口写物理参数收口 (PORT-01/02)

- **需求映射:** PORT-01, PORT-02
- **文件面:** `internal/services/portwrite/`（批量物理参数路径 + 型号级 `vendorCapabilities` 规则）及其 `*_test.go`、可能动前端端口写 Modal（`xingran-react-frontend/src/pages/network/ports/` 端口写组件）。与 142/144 不重叠。
- **实现要点:** 沿用 v1.51 Phase 122 批量写实时进度事件，补齐 speed/duplex/mtu 动作的批量分发、逐项结果与错误聚合；`vendorCapabilities` 规则以设备已有型号字段为来源、厂商级默认值回退、未知型号 fail-safe 结论显式登记；全部用非真机 fixture 单测覆盖。
- **成功标准（可观察）:**
  1. 用户在批量端口写弹窗勾选多端口并选择 speed/duplex/mtu 提交后，逐端口下发并复用 v1.51 Phase 122 实时进度事件；`BatchResult` 的 succeeded/failed/skipped 三类逐项聚合，错误可定位到具体端口。
  2. 单条物理参数写（3 厂商 × 3 参数）零回归：`internal/services/portwrite/` 既有单条测试全绿，批量与单条共享同一「校验 / 审计 / 命令分发」函数（无第二套实现，可静态断言）。
  3. 型号级 `vendorCapabilities`：命中型号→按能力矩阵放行/拒绝，未命中→厂商级默认值回退，完全未知→fail-safe（拒绝或降级到安全默认值，结论写在代码注释与测试名）；fixture 单测覆盖「命中 / 厂商回退 / 未知 fail-safe / 型号字段缺失」全部分支，不依赖真机。
  4. 前端端口写 Modal 按型号能力约束禁用不支持的动作或给出提示，API 走 `@/lib/api.ts` wrapped functions；`tsc` / `lint:ci` / port-write 相关 `vitest` 全绿、vendor-react gzip 零回归。
  5. `go build ./...` + `go test ./internal/services/portwrite/` 全绿。

#### Phase 144: 同步漂移告警通道收口 (ALRT-01/02/03)

- **需求映射:** ALRT-01, ALRT-02, ALRT-03
- **文件面:** `internal/services/addomain/ad_sync_drift_alert_*.go`（FIX-11 后端服务已存在）、`internal/scheduler/ad_sync_tasks.go`、`internal/websocket/notice_hub.go`、`internal/api/v1/system/ad_drift_alert_*.go`（新）、`internal/api/router.go`（**只此 phase 动**）、`internal/core/db/migrations/migration_238_*.go`（新，菜单/权限/seed，**只此 phase 动**）、`xingran-react-frontend/src/{lib,store,types,pages/system/ad-drift-alert,constants}`。与 142/143 不重叠。
- **实现要点:** 补齐 FIX-11 自报边界——后端查询/配置端点 + WS 广播 + 前端告警页 + 企微/钉钉 webhook（凭证脱敏、失败可观测、不阻塞主流程）。
- **成功标准（可观察）:**
  1. 用户在「同步漂移告警」页面看到告警列表（分页/筛选/详情/已读），数据来自新端点（如 `POST /system/ad-drift-alert/list`）；已读状态切换持久化、刷新后保留。
  2. 漂移告警产生时 `internal/websocket/notice_hub.go` 复用现有广播 API 向在线客户端推送，前端 store 订阅后无需刷新即实时插入列表（WS 消息体可断言）；WS 广播失败不阻塞主同步流程。
  3. 企微/钉钉 webhook 通道：URL + 启用开关在后端可查询/配置，推送在非阻塞 goroutine 执行、失败可观测（日志/计数）；`operlog.RecordWithBody` 把 webhook URL/secret 脱敏为 `******`，单测断言脱敏且明文 URL 不落日志。
  4. `migration_238_*` 播种菜单/权限码/seed 且 SQLite+PG 双方言幂等，`internal/api/router.go` 注册新路由；受限角色无权限码→403、授权后→200（handler/fixture 断言，不依赖生产库）。
  5. 前端页面用 `@/lib/api.ts` wrapped functions（禁止裸 axios）+ Zustand store + `src/types` 类型；`go build ./...`、`go test ./internal/services/addomain/ ./internal/api/v1/system/ ./internal/websocket/`、`tsc` / `lint:ci` / `vitest` 全绿。

### 并行冲突面（W1 并行许可凭证 · 函数级归属表）

沿用 v1.53 惯例：W1 并行前以**函数级归属表**登记每个 phase 独占的写集合与顶层符号，确认两两交集为空；共享只读基座（`pkg/response`、`internal/utils/operlog`、`internal/models`、`internal/core`）只 import 不修改。

| Phase | 独占文件面（写集合） | 函数级归属（独占符号） | 与其它 phase 交集 |
|-------|----------------------|------------------------|-------------------|
| 142 | `internal/services/device_discovery_service*.go`、`internal/api/v1/network/discovery_*`、网络模块测试 | `ImportDiscoveredDevices`、`DeviceDiscoveryService` 导入方法、`DiscoveryHandler` 缓存失效注入 | ∅ |
| 143 | `internal/services/portwrite/*` + 其 `*_test.go`、前端端口写 Modal | `BatchWritePorts` 物理参数分发、`vendorCapabilities` 规则表、型号能力判定函数 | ∅ |
| 144 | `internal/services/addomain/ad_sync_drift_alert_*`、`internal/scheduler/ad_sync_tasks.go`、`internal/websocket/notice_hub.go`、`internal/api/v1/system/ad_drift_alert_*`(新)、`internal/api/router.go`、`internal/core/db/migrations/migration_238_*`(新)、`xingran-react-frontend/src/{lib,store,types,pages/system/ad-drift-alert,constants}` | `AdDriftAlertHandler` / `SetupAdDriftAlertRouter`、`notice_hub` 漂移广播函数、webhook 推送函数、migration 238 `Up()` | ∅ |

**W1 并行许可凭证:** 三者写集合两两交集 = ∅，按模块（发现导入 / 端口写 / 告警通道）隔离，无共享可变文件，故 `142 ∥ 143 ∥ 144` 同波并行成立。两处全局唯一归属硬登记：**`internal/api/router.go` 与 `internal/core/db/migrations/migration_238_*` 仅归 Phase 144**，142/143 不得触碰；若 `plan-phase` 实测出现跨 phase 交集（如前端 `src/lib/api.ts` barrel 同时被 143/144 改导出），须补函数级坐标并把该文件降级串行或拆 commit，否则维持并行判定。

### 关键约束（全局硬约束，沿用 v1.53 惯例）

1. **Handler-Service 分层**：service 返回 `error`；handler 用 `response.Success()` / `response.Error()` 包装，禁止裸 `c.JSON()`；`c.Request.Context()` 逐层透传。
2. **operlog 写操作约定**：所有业务写操作 success path 在 `response.Success(...)` **之前**调用 `operlog.Record(...)`（OperType 按语义，25 常量不可重排）；含 token/secret/webhook 的端点用 `operlog.RecordWithBody` 脱敏（`******`）。
3. **Status `0=正常 1=停用`**；UUID 主键；**SQLite + PG 双方言**（避免 PG 专属 SQL、uuid 双侧 CAST、migration 幂等）。
4. **前端 wrapped API**：`@/lib/api.ts` 的 `post` 等 + Zustand store，禁止裸 axios；`useEffect` 依赖用 primitive/useMemo 稳定。
5. **构建产物只落 `out/`**；Excel 模板动态生成、禁止落盘 `.xlsx`；`data/` 不入库；Go 单测就近 `*_test.go`、前端 `*.test.ts(x)`；零新依赖。
6. **不伪造人工验证闭环**：验证类残余（见 Known Gaps）不建 phase、不勾选、不回写为已完成；本里程碑只交付可代码化能力。
7. **LOCKED 残余不写修复代码**：`.planning/notes/261108-v153-locked-residuals.md` 6 项只维持「接受/仅登记」结论。
8. **79 条人工验证跟踪表保持 0 勾选**：`.planning/notes/261108-human-verification-tracker.md` 按 `notes/261108-uat-writeback-policy.md` 由人工执行后回写。
9. **回归基线**：`go build ./...` + `go test ./...` + `tsc` + `lint:ci` + `vitest` 全绿；提交由 Lead 统一执行（roadmapper 不 git commit）。

### Known Gaps（登记，非阻断）

以下 4 项为**验证类 / 真机类残余**，在 v1.54 显式登记，**不建 phase、不纳入 7 REQ、不伪造闭环**；保持 `.planning/REQUIREMENTS.md` Future Requirements / Out of Scope 与 STATE.md Deferred Items 的原登记状态：

| 项 | 边界 | 处置 |
|----|------|------|
| FIX-01 实机 AD/LDAP UAT（UAC 544 + 登录恢复） | 需生产 AD 实机 | 待人工执行，79 条跟踪表承载 |
| FIX-09 端到端 HTTP P99 压测 | v1.53 仅计算路径近似基线，需真实压测环境 | 待人工执行 |
| FIX-13 生产库 migration 236/237 | 需生产库写授权后受限角色才真正 403→200 | 待 DBA/人工执行，AI 不擅自写生产数据 |
| FIX-04 真机逐型号取证 | 三厂商物理参数逐型号真机结论 | 待现场取证；PORT-02 只做可代码化的规则框架，不替代真机结论 |

> 另：79 条人工验证跟踪项（0 勾选）、6 项 LOCKED 残余、FUTURE-01（P1-5 侧信道告警通道合并评估）继续挂账，见 `.planning/REQUIREMENTS.md` 与 `.planning/notes/261108-*`。
>
> **v1.54 交付后新增登记（2026-11-09，非阻断）**：① 告警「标记已读」未实现（后端 5 端点无该契约、`logs` 来自 `sys_oper_log` 无已读字段）② 企微/钉钉专用 markdown + 加签（路线 B）未做，当前只走路线 A 通用 Webhook ③ `sys_object_id` 已采集未落库、型号 key 仅 `NetworkDevice.Model` ④ 批量物理参数无自动回滚（预览/审计已显式警示）⑤ 发现导入链路顺带隐患（`user_id.(string)` 硬断言 / `persistDiscoveryResults` 未开事务 / `ExecuteDiscovery` 未检查 Save Error）⑥ e2e 归位 ⑦ `RequiresInterfaceView` 与型号画像 `RequiresView` 联动点。逐项见各 phase `*-SUMMARY.md` 与 `STATE.md ## Deferred Items` 的 v1.54 行。
>
> **交付证据（4 份 SUMMARY）**：`.planning/phases/142-disc-01-02/142-SUMMARY.md`、`.planning/phases/143-port-01-02/143-SUMMARY.md`、`.planning/phases/144-alrt-01-02-03/144-01-backend-SUMMARY.md`、`.planning/phases/144-alrt-01-02-03/144-02-frontend-SUMMARY.md`。架构裁定 `.planning/notes/261108-v154-architecture-decisions.md`；集成清单 `.planning/notes/261108-v154-integration-checklist.md`。

</details>

<details>
<summary>✅ v1.53 开放项全量清偿 (Phases 134-141) — SHIPPED 2026-11-08</summary>

- [x] Phase 134: AD 同步域 —— enable 方向 + 全量同步节流 + 漂移告警产品化 (FIX-01/10/11) — completed 2026-11-08
- [x] Phase 135: Swagger / 文档产物链路修复与死文件清理 (FIX-02) — completed 2026-11-08
- [x] Phase 136: 接线型端点缺口 —— 用户导出 + 工单评价 + OQ-7 五个读权限码 (FIX-03/06/13) — completed 2026-11-08
- [x] Phase 137: Stub 与双副本漂移治理 —— GetDiscoveryResults + rpa-worker (FIX-07/08) — completed 2026-11-08
- [x] Phase 138: 功能分支补齐 —— scheduler Rotation + 端口写物理参数 + 3 处排序 (FIX-04/05/14) — completed 2026-11-08
- [x] Phase 139: 低优先级技术债 —— 对账 MV 超时配置化 + 登录加密 benchmark 实测 (FIX-09/12) — completed 2026-11-08
- [x] Phase 140: 人工验证债组织 —— 5 份缺失 UAT 清单重建 + 79 条跟踪表 + 回写口径 (VAL-01~07) — completed 2026-11-08
- [x] Phase 141: 登记接受残余 + 临时清单归档 + 权威交回 GSD (ACK-01/02 + GOV-01) — completed 2026-11-08

**交付口径**：18 REQ 交付 + 6 项人工验证跟踪（VAL-01~06 交付物已落盘、实际验证待执行，不伪造闭环）；21/21 条开放项全覆盖。全量回归闸门：go build / go test 全仓 / tsc / lint:ci / vitest 592 文件 4128 用例全绿。完整 phase 详情见 [v1.53-ROADMAP.md](milestones/v1.53-ROADMAP.md) 与 `.planning/milestones/v1.53-phases/`。

**Known Gaps（登记，非阻断）**：人工验证跟踪表 `.planning/notes/261108-human-verification-tracker.md`（79 条全部待执行）；LOCKED 残余结论 `.planning/notes/261108-v153-locked-residuals.md`。

</details>

<details>
<summary>✅ v1.52 运维域数据权限统一收口（终态）(Phases 130-133) — SHIPPED 2026-11-07</summary>

- [x] Phase 130: ScopeProvider 基建 (DS-01/02) — completed 2026-11-07
- [x] Phase 131: 四域读端点 + Excel 导出独立收口 (DS-03/06) — completed 2026-11-07
- [x] Phase 132: 四域写端点收口 (DS-04) — completed 2026-11-07
- [x] Phase 133: 扩展实体 + vdi 合并迁移 + UAT (DS-05/06/07) — completed 2026-11-07

</details>

<details>
<summary>✅ v1.51 端口写生产级操作体验与平台质量收口 (Phases 121-129) — SHIPPED 2026-11-02</summary>

- [x] Phase 121: 端口写 dry-run 预览 + 命令预览 UI — completed 2026-10-31
- [x] Phase 122: 批量写实时进度 SSE/WS — completed 2026-11-01
- [x] Phase 123: 安全三件套 — 快照回滚 + 并发写互斥 + 不可达预检 — completed 2026-11-01
- [x] Phase 124: VLAN 扩展 — trunk native/allowed + hybrid — completed 2026-11-01
- [x] Phase 125: 端口安全 — port-security/sticky/静态 MAC 绑定 — completed 2026-11-01
- [x] Phase 126: Cisco IOS-XE 第 4 厂商 + 跨固件差异矩阵 — completed 2026-11-01
- [x] Phase 127: 审计 UI + 审计治理 + 3 厂商 e2e — completed 2026-11-01
- [x] Phase 128: 平台质量收口 — completed 2026-11-01
- [x] Phase 129: 目录改名 + 双线 planning 合流 + UAT 回补组织 — completed 2026-11-01

</details>

<details>
<summary>✅ v1.25 全站权限治理与样式收敛 (Phases 84-93) — SHIPPED 2026-10-30</summary>

- [x] Phase 84: 权限模型三缺口 — completed 2026-10-28
- [x] Phase 85: 后端动作级严格化 — completed 2026-10-29
- [x] Phase 86: 前端操作入口三态门控铺开 — completed 2026-10-29
- [x] Phase 87: 动作级权限授予闭环 — completed 2026-10-30
- [x] Phase 88: 验证底座与零风险清理 — completed 2026-10-29
- [x] Phase 89: 变量层与明暗色板重构 — completed 2026-10-29
- [x] Phase 90: Input 域交还 variant/token + 圆角统一 — completed 2026-10-30
- [x] Phase 91: Select / InputNumber / Form / Table / Modal 域交还 — completed 2026-10-30
- [x] Phase 92: 暗色 mega-cluster 合并 — completed 2026-10-30
- [x] Phase 93: room-devices 拆分 + 视觉回归验收 — completed 2026-10-30

</details>

<details>
<summary>✅ v1.24 视觉回归修复与遗留债务清偿 (Phases 76-83) — SHIPPED 2026-10-28</summary>

- [x] Phase 76: 双框 P0 修复 + CSS 审计登记册 — completed 2026-10-28
- [x] Phase 77: 双框视觉回归验证 — completed 2026-10-28
- [x] Phase 78: #16 前后端双端闭环 — completed 2026-10-28
- [x] Phase 79: #16 PG 实机验收 — completed 2026-10-28
- [x] Phase 80: 3f130 空因诊断前端接入 — completed 2026-10-28
- [x] Phase 81: F5 刷新落地页修复 — completed 2026-10-28
- [x] Phase 82: lint 清零 + 豁免理由表修复 — completed 2026-10-28
- [x] Phase 83: devices 页拆分 — completed 2026-10-28

</details>

---
*Roadmap v1.55 created: 2026-11-09 by gsd-roadmapper（Phases 145-151，24 REQ / 100% mapped / W1 `145∥146∥147` → W2 `148∥149∥150`（函数级归属表凭证）→ W3 `151` 最后执行）；v1.54 当前段完整保留为折叠 `<details>` 历史段（含 3 个 phase 详情 / 并行冲突面 / 关键约束 / Known Gaps），历史里程碑 SHIPPED 列表（v1.54~v1.20）与 v1.53~v1.24 Phases 详情保留为折叠 `<details>`；当前段沿用本仓手工编辑惯例；提交由 Lead 统一执行*
