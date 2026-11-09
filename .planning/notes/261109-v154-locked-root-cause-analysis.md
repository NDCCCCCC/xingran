# v1.52 LOCKED 6 项「能否从根源修复」深度判定报告

**日期**：2026-11-09
**性质**：只读分析交付物，**不改任何代码、不改 LOCKED 裁定结论、不动 79 条人工验证跟踪表**。供用户重新判定 6 项 HOW 处置。
**方法**：3 个并行只读 subagent（数据权限侧 / SQL+e2e / 缓存+密码学）+ Lead 交叉核对（data_scope 单一真实源、聚合计数、router 挂载、脚本执行顺序、password 原文）。

> **总判定（与 v1.52 原裁定的关系）**：6 项中 **5 项在代码事实层面裁定不成立或理由被高估，可以或应该根治**；**仅 ① P1-5 属产品语义取舍**，技术上能根治但必须推翻已交付的「范围快照回显」语义，否则只能缓解。

---

## 0. 结论速览

| # | 项 | 原 LOCKED 理由 | 深挖后的根因定性 | 能否根治 | 判定 |
|---|-----|----------------|------------------|----------|------|
| ① | P1-5 聚合计数侧信道 | 接受；改回 403 破坏 v1.52 语义 | **不是 SQL 漏 WHERE**，是产品有意「带 scope 查询 + 不带 scope 全库 COUNT + 相减」回显越界规模 | 能，但须推翻产品语义 | **不应在现行语义下修**；产品重定语义则可根治 |
| ② | 5 项显式豁免面 | Three.js 联动回归面大、成本收益不匹配 | 归属链 + `wrapScopeExists` 基建现成；4 个组挂了中间件但 service 不消费，**rooms/photos 连中间件都没挂** | **能，零迁移** | **应该重开评估、可根治**（成本被高估） |
| ③-A | `RightCast:"text"` | 与 P1-8 独立、SQLite 报错已文档化 | 同源 `::` 方言片段，只是走了另一条查询路径；报错被 `logger.Warnf` 静默吞掉 | **能（S）** | **应根治** |
| ③-B | PG 专属 CTE | 同上 | 项目内**已有两套双方言改写范本**（窗口函数 / Go 层合并）；加方言分支才是补丁 | **能（M）** | **应根治**（推荐 Go 层合并，从根源消除 raw SQL） |
| ④ | `operlog_e2e_verify.sh` | 调门禁顺序会改 e2e 语义 | 凭据门禁职责错位 + **新发现 cwd 锚定缺陷**（静态检查在包子目录下读到空数据）；且静态段从未进 CI | **能（S-M），最便宜** | **应根治**；裁定理由不成立 |
| ⑤ | `invalidateCache` no-op | 无 Redis、待接入后补 Delete | **缓存键是死键**（全库无 Set/Get）+ 空函数体 + 服务未注入 cache；L1/L2 都支持 Delete，「无 Redis」是伪命题 | **能** | **应根治**；「接入 Redis 后补 Delete」是错误处方 |
| ⑥ | `pbkdf2SM3` 多块循环 | 全部调用点 32、无实际调用面 | `VerifyPassword:149` 用 `len(hashBytes)`（**裁定事实有误**）；>32 时静默返回后半段全零弱密钥 | **能（修契约即可）** | **应根治**；隐患本质是非法输入静默弱结果 |

**意外收获（比 6 项本身更值得关注）**：
- ② 里 `router.go:1174-1177` 的 rooms/photos 读组**漏挂 `DataScopePermission`**，是比「豁免」更基础的接线遗漏。
- ④ 里静态检查用相对仓库根路径、Go driver 未设 `cmd.Dir`，**即使注入凭据/挪门禁，静态检查仍读到空数据**（仓库根 `CALLS=373` vs 包子目录 `CALLS=0`）；且 `ci.yml` 的 `go test ./internal/...` 根本不覆盖 `scripts/e2e/` 与 `tests/e2e/` → 静态检查**从未真正当过 CI 门禁**。
- ⑤ 里真正的陈旧点在 `reconciliation_service.go:821-834` 的 `GetByWorkstation` 5min TTL 缓存（exception 字段），同文件 `InvalidateWorkstationHealth` 已是正确范式但无「规则变更 → 广播失效」联动。

---

## 1. 逐项根因与根治方案

### ① P1-5 聚合计数侧信道

**根因（纠正认知）**：8 个 `/statistics` 端点**已全部**在 SQL 层用 `wrapScopeExists` / `apply*Scope` 注入可见集（如 `workstation_service.go:159/169`、`asset_service.go:69/74`），COUNT 与明细同源。残余不在这些端点，而在：
- `BackfillDeptFromUser`（`workstation_service.go:606-682`）：`candidateQ` 带 `visibleCond` 裁剪（:645-647），但**另外跑一条刻意不带 scope 的全库 COUNT**（:666-669，注释明写「COUNT 保持全库口径 = 请求行数…若给 COUNT 也挂可见集条件会让越界恒 0，丢失审计语义」），再 `ScopeOutOfScope = ScopeTotal - ScopeVisible`（:677）→ 响应体 `scopeTotal/scopeVisible/scopeOutOfScope/skippedNoUser/skippedBadUserDept`（:130-132,681-682）可被差分出越界规模。
- Excel 导入：`ImportResult.Errors[].Row` 实际行号 + `ScopeOutOfScope`（`excel_service.go:62-84`、`scope_write.go:687-722`）。
- 批量写：operlog `requested/visible/outOfScope`（`scope_write.go:730-738`），不直接回前端但入审计链。

→ **这是产品要求「同一响应里同时给出可见集规模与全库规模」导致的信息论必然，不是实现缺陷**。132-SUMMARY.md:65/:103 三处裁定一致接受，`Samples` 已收敛（非 All 档恒空）但聚合数值仍在。

**根治定义**：任何非 All 档用户，通过单请求或多次请求组合，都无法从响应体（含任何数值/序号/数组长度）推导出可见集外的资源规模或越界资源是否存在。

**方案对比**：

| 方案 | 改动 | 对既有语义 | 工作量 | 能否根治 |
|---|---|---|---|---|
| **A 响应体去规模化**：删 `ScopeTotal/ScopeVisible/ScopeOutOfScope/ScopeDeptTotal` 与全库派生字段、删全库 `countQ`（或挂 `visibleCond`）、去 `Errors[].Row` 与 `ScopeOutOfScope`、operlog 保留聚合但不回 HTTP | 8-10 文件（含前端 `useBackfillDept.tsx` 四类「本次范围：共 N 条」文案） | **推翻 132 产品语义**（范围快照可回显/可审计） | 中 | ✅ 信息论唯一彻底解 |
| **B 统一 ScopedDB**：scope 谓词下沉到 GORM `Scopes`/`Session`，所有 COUNT/SUM/Find 从带 scope 的 DB 起链，禁止裸 `s.db.WithContext(ctx)`；新增「资源归属注册表」 | 基座 `base.GORMRepository`（485 Go 文件）+ 8 个 service 起链处 | 完全兼容（不改 403、不改裁剪） | 高、回归面大 | ✅ 防回归，但**单独用不够**（响应仍可显式算差值） |
| C DB 视图 / PG RLS | 连接池 + 双方言安全模型 | 兼容 | 极高 | ❌ 双方言下不可行（SQLite 无 RLS、共享连接池串味、视图无法参数化每用户可见集） |

**推荐 A+B 组合**：B 保证「同源不靠自觉」，A 保证「响应体不可差分」。根治的**门槛不在代码，在产品是否愿意放弃范围快照回显**。

**判定**：**在现行 LOCKED 语义下不应修**（只能缓解：粗粒度桶 `0/<10/10-100/100+`、频率限制、FUTURE-01 告警通道——告警是检测利用行为，不关闭差分通道，按「不打补丁」标准不算根治）。若产品正式重定语义 → 技术上无阻塞，可根治，需重跑 132/133 backfill 相关 UAT。
> FUTURE-01 触发条件「FIX-11 告警通道落地」在 v1.54 已成立，但告警实现未覆盖聚合计数面；若①重定语义，应一并关闭 FUTURE-01。

---

### ② 5 项显式豁免面

**根因（成本被高估）**：
- 归属链**现成**：walls/doors/floor-plan-texts `floor_id → ops_floors.org_id/building_id → ops_buildings.org_id`，正是 `effectiveOrgExpr("f.org_id","b.org_id")`；photos `room_id → serverRoom → building`；**`location-alias` 甚至已有 `DeptID`**（`sys_dept_location_alias.go:22`，裁定「5 类模型都不含归属字段」对前 4 类成立、对它不成立）。
- 收口基建**现成**：`scope_entities.go:81-183` 的 `entityExistsSub` + `wrapScopeExists` + `visibleIDsForEntityWrite`；`serverRoom/dedicatedLine/roomDevice` 已按楼宇链收口，`workstation-device` 已按工位可见集收口。
- 4 个读组挂了 `DataScopePermission` 但 5 个 service 的 `List` **完全不消费 ctx 里的 ScopeResult**（`wall_service.go:84-90` 等直接 `repo.List`）；**`router.go:1174-1177` rooms/photos 连中间件都没挂**。
- Three.js 编辑器是「按 `floorId` 一次拉整层（`pageSize:1000`）→ 内存渲染 → 保存时整体覆写」（`useFloorPlanEditor.ts:52-73,156-186`）；同页 `getFloorWorkstationsAll` 已收口 → **同一页面工位受 scope、几何不受，收口后语义反而一致**。

**根治方案 D（推荐，零迁移）**：5 个 service 起链处统一挂 `entityExistsSub` + `wrapScopeExists`；`GetByID`/count/primary/batch-delete 全走 `visibleIDsForEntityWrite`；location-alias 按 `dept_id ∈ 可见集` 直挂；rooms/photos 组补挂 `DataScopePermission`；**保存侧 `BatchOperation` 走可见集裁剪**（否则非超管保存会误删自己看不到的几何元素——这是「联动回归面大」的实质，也是唯一必须改的前端/后端边界）。响应形状不变、不引入整批 403、与 v1.52 语义完全一致。

| 方案 | 数据迁移 | 前端面 | 判定 |
|---|---|---|---|
| **D 父资源可见性校验（推荐）** | 零 | 低-中（仅保存侧裁剪） | ✅ 根治，成本远低于裁定假设 |
| E 补 `org_id` + migration + 回填 | 中-大（4 表 + 双写一致性） | 同 D | ✅ 但引入新一致性面，性价比低 |
| F 几何合并进楼层 JSONB | 大 | 极大（重写编辑器） | ✅ 但回归面与裁定担忧吻合，最不划算 |

**判定**：**可以根治、建议重开评估**。LOCKED 理由是成本判断而非技术/语义不可行；需作废 `133-UAT.md:82-91` D1/D2「确认未收口」结论并重跑 UAT。

---

### ③-A `RightCast:"text"`

**根因**：`excel_export_config.go:18,351/355/359` 让配置承载 `"text"` 方言片段，`excel_export_service.go:148` 拼成 `id::text IN (?)`（PG 下 uuid 列与 varchar 左侧参数类型对齐所需），`::` 在 SQLite 报 `unrecognized token ":"`，错误被 `:160-165` 的 `logger.Warnf + continue` **静默吞掉 → 关联列整列空但导出仍成功**。P1-8 已把**另一条** `excel_query_builder.go` 路径的 7 处 `::` 改成双侧 `CAST AS TEXT`（`:253-256`），`resolveAssociations` 是独立路径所以没被覆盖——「独立」指路径独立，根因同是 `::`。

**根治（S）**：删掉 `RightCast` 承载方言片段的机制，改语义枚举 + service 统一 `CAST(id AS TEXT)`（**只需单侧**，P1-8 双侧是因 JOIN 两边都是列）+ P1-8 式 DryRun 静态断言 + SQLite 实跑断言「关联列非空、无静默降级」。
**不应修**：导出改内存计算（全表拉取、新一致性风险）。
**判定**：**应根治**（小工作量，且能消除静默降级）。

---

### ③-B PG 专属 CTE

**根因**：`workstation_device_service.go:426-535`（单工位）、`:2151-2234`（批量）用 `db.Raw()` 写 PG 专属 CTE 做 MAC→port→infoPoint→workstation 反推链，含 4 处 `DISTINCT ON`、`REGEXP_REPLACE`（接口名/MAC 归一化）、`NULLS LAST`、6+ 处 `id::text`。现有 SQLite 测试**故意把报错当预期**（`workstation_device_77_01_test.go:9-11`「勿为覆盖率改 SQL」、4+ 个 `require.Error`），PG 侧真结果只有 `TEST_PG_DSN` 门控且 CI 不注入 DSN → 零真实验证。

**根治（M，项目内已有两套可抄范本）**：
- **推荐 ② 拆多次 GORM 查询 + Go 层归一化/去重/排序**：`diagnosePhysicalChain`（`:701-799`）注释明写「刻意用 GORM chainable + Go 侧集合求交，避免 PG 专属 CTE」，且 `physicalDevicesFetcher` 注入保证「不存在第二套 recall 逻辑」→ **从根源消除 raw SQL 与方言分叉**。
- 备选 ① 双方言窗口函数重写：`sqlite_reconciliation_views.go:14-35` 给出 `DISTINCT ON → ROW_NUMBER() OVER(...) + WHERE rn=1`、`REGEXP_REPLACE → 嵌套 REPLACE`、`NULLS LAST → (x IS NULL), x` 的逐条翻译与等价性论证。
- **明确是补丁不是根治**：加 `isPostgres` 方言分支 → 两套 SQL 口径必然分叉（被 `:166-168` 注释反对）。
- 不适用：双方言视图（CTE 带 `?` 参数，视图无法参数化）。
- **必须同步改写 4+ 个「报错即预期」测试**为真实结果断言，否则根治后测试红。

**共性防复发（三者组合才叫根治，单一只是缓解）**：① `.golangci.yml` 启用 `forbidigo` 禁方言 token（XS，先白名单存量逐步收敛）② 抽 `dialect` helper 提供 `CastToText/Top1PerGroup/NormalizeMac` 语义原语，配置里禁止出现 SQL 片段（M）③ CI 起 PG service 注入 `TEST_PG_DSN`，把 3 个 contract 测试从「skip 可观测」升级为双跑硬门禁（S-M，取决于 CI 是否已有 `services: postgres`）。

**判定**：**应根治**（推荐 ② + 共性 ①）。

---

### ④ `operlog_e2e_verify.sh`

**根因（比裁定更严重）**：
- 凭据门禁 `:57-66` 的 `exit 1` 在静态检查 `:68-151` **之前**（`SKIP_LIVE` 在 `:162`，静态段本就声称「even SKIP_LIVE=1 也跑」）→ 缺凭据时静态检查永远到不了；两个 Go driver 把 `exit 1` 转成 `t.Skipf` → CI 两包「凭据缺失 → 可观测 skip → PASS」。
- **新发现 cwd 缺陷**：静态检查用相对仓库根路径（`grep -r ... internal/`、`:88` 读 `operlog.go`、`:115` 遍历 `internal/api/v1/`），而两个 driver 都没设 `cmd.Dir`，Go test cwd 是包子目录 → 实测仓库根 `CALLS=373 / HANDLER_FILES=84`、包子目录 `CALLS=0 / 目标文件不存在 / HANDLER_FILES=0`。**即使挪门禁或注入凭据，静态检查仍读空数据（假绿）**。
- **`scripts/e2e/` 与 `tests/e2e/` 根本不在 CI 测试范围**（`ci.yml` 只跑 `./internal/... ./pkg/... ./cmd/...`）→ 静态检查从未真正当过门禁。
- 两份 329 行脚本仅 1 行注释不同，属重复副本（AGENTS.md 目录治理问题）。

**根治（S-M，最便宜、收益最大）**：**静态段从 shell 拆出改写成 Go 单测 + CI 硬门禁**——`internal/utils/operlog/regression_test.go`（已用 `go/ast` 锁 25 常量/签名/敏感关键词）与 `coverage_test.go`（已测脱敏）现成，只差补一个 `go/parser` 遍历 `internal/api/v1/**/*_handler.go` 的 handler 覆盖扫描（约 80-120 行），进 `go test ./internal/...` 天然硬门禁，零凭据/零 DB/任意 cwd 真跑。LIVE 段保留 shell、缺凭据仍 `exit 1`。
**不应修**：shell 整体替换为 Go e2e（把 CI 硬门禁与需凭据 live 重新绑回同一二进制，复现同类缺陷）；仅调门禁顺序（cwd 问题未解，连缓解都不算）。
**裁定理由「调门禁顺序会改 e2e 脚本语义」不成立**——独立静态段是**恢复脚本注释声明的语义**。

---

### ⑤ `invalidateCache` no-op

**根因（裁定在代码事实层面不成立）**：
- `CacheKeyReconciliationExceptionRuleList`（`cache_keys.go:37`）是**死键**：全库 6 处命中全是定义/注释/helper/测试断言，**没有任何 Set/Get/GetOrSet 使用它** → 「写后缓存未失效」这个现象在代码里不存在。
- `reconciliation_exception.go:578-580` 是**空函数体**（零行代码、无 ctx、无 error 返回），服务结构体只有 `db *gorm.DB`、3 处装配全只传 db；4 个写路径（Create/Update/Delete/Import）顺序正确地调用了它，但调的是空函数。
- **「无 Redis」是伪命题**：`core.go:883` 的 else 分支返回功能完整的 `*MemoryCache`（非 nil），`memory.go:103-113` 与 `redis.go:697-700` 的 Delete 都是真实实现；`NoOpCacheProvider.Delete` 返回 nil 是自洽的（其 `GetOrSet` 从不写缓存）。→ **「接入 Redis 后补 `core.Cache.Delete(ctx, 死键)`」是错误处方**（删一个从没人写过的键）。
- 读路径（List/GetByID/MatchTest/MatchException/DetectLayer3）**全部直查 DB**，本就不缓存例外规则。
- **裁定遗漏的真实陈旧点**：`reconciliation_service.go:821-834` 的 `GetByWorkstation` 5min TTL 缓存（`computeByWorkstation` 6 步聚合，有性能价值）里的 `exceptionHit/AppliedActions` 会随规则变更陈旧；同文件 `InvalidateWorkstationHealth`（`cache_keys.go:129-134`，接 cache 参数、nil-safe、真调 Delete）是正确范式，但按 wsID 反查、无「规则变更 → 广播失效」联动。

**根治（推荐方案③，最小）**：删死键 + 删空函数（消除悬空契约）+ 给 `GetByWorkstation` 补例外变更联动失效（或剥离缓存里的 exception 字段）；保留 5min 缓存不动其性能价值。验收：写后读必一致、`memory`/`redis` 两配置行为一致、失效动作可观测可失败传播。
**方案②（规则版本号 + 读时校验）**也是真根治且环境无关，但需一处 schema/migration。
**方案①（改全局 `CacheProvider` 接口禁 no-op）**仅缓解：20+ 消费文件/大量 mock 连锁改，且未命中真实陈旧点、为一个死键改全局接口，收益比最差。

**判定**：**应根治**；「触发条件 = 接入 Redis 后补 Delete」应作废。

---

### ⑥ `pbkdf2SM3` 多块循环

**根因（裁定事实有误）**：
- `password.go:75-94` 计数器硬编码 `INT32_BE(1)`（只有 T_1），`result := make([]byte, keyLen)`、`copy(result, block)` 只拷 32 字节、XOR 循环 `j < keyLen && j < len(block)` 恒 ≤32 → **keyLen>32 时不 panic、不报错、不截断，而是 `result[32:]` 保持全 0**，静默返回「看似合法但后半段全零」的低熵密钥（最危险失败模式）。附带 `:75` `append(password, salt...)` slice 别名隐患。
- **「全部调用点 keyLen=32」不成立**：`HashPassword:110` 写死 32，但 **`VerifyPassword:149` 用 `len(hashBytes)`**（base64 解码存储哈希）→ 库中若存在/被构造 64 字节哈希即触发；`$sm3$` 格式（`:118`）未编码 keyLen。
- 无 SM4/会话/token 派生用途（全库 `pbkdf2SM3` 仅 2 处调用 + 1 测试 + 1 注释），32 是 SM3 摘要长度这一实现便利，非协议约束。

**根治 = 把未定义行为变定义行为（F+G），不是补多块循环**：
- F：底层 `pbkdf2SM3` 对 `keyLen<=0` / `>32` / `iterations<=0` 显式返回 error（项目禁 panic）；
- G：`VerifyPassword` 显式校验 `len(hashBytes)==32`，非 32 字节哈希属损坏/伪造数据，显式拒绝。
- 补完整多块循环是**能力扩展**（当前无需求），可做但非消除当前隐患的必要条件；若做，必须：黄金向量先行（固定输入固化 hex 锁 `keyLen=32` 输出逐字节不变，因 `l=1,i=1` 与当前实现一致）、独立 oracle `golang.org/x/crypto/pbkdf2.Key(sm3.New,...)` 比对 `keyLen=33/64/65`、断言 `DK(64)[0:32]==DK(32)`、改 `make+append` 避免别名。
- **向后兼容硬约束**：`keyLen=32` 输出逐字节不变 → 存量 `$sm3$<iter>$<salt>$<32字节hash>` 不变、600k 默认与 100k 旧哈希无需迁移。⚠️ 另需盘点**存量 `sys_user.password` 是否存在非 32 字节数据**（若有，加 G 守卫会让这些账号无法登录——这是「输入校验收紧」而非「输出漂移」，需单独决策）。

**判定**：**应根治**（先立契约，补算法可选）；两项都不该维持 LOCKED。

---

## 2. 共性架构结论

**6 项中有 4 项（①②③⑤）指向同一个架构层缺陷：约束靠「约定」而非「机制」强制。**
- `scope_filter.go:225-227` 把「统一收口入口」定义成**一个函数** `wrapScopeExists`，而不是**不可绕过的查询层** → ② 5 个 service 不自觉调用即绕过、① backfill 可另跑一条不带 scope 的 COUNT。
- raw SQL 方言片段无 CI 准入 → ③；缓存失效靠空函数「约定」→ ⑤。

**统一解（应用层，非 DB 层）**：`DataScopePermission` 注入 ScopeResult（已有单一真实源 `pkg/datascope`）→ **统一 ScopedDB / QueryBuilder 作用域**（GORM `Scopes`，强制唯一查询起链点，内置「资源归属注册表」引用 `effectiveOrgExpr` / 归属链 EXISTS / `dept_id` 直挂，内建 `scopeCacheKey`）→ **响应契约层**禁止任何「全库规模 − 可见规模」回显。零件全部现成（`wrapScopeExists`/`entityExistsSub`/`scopeCacheKey`/`base.GORMRepository`/`sqlite_reconciliation_views.go` 范本），`EXISTS (?) + CAST AS TEXT` 已验证双方言。
**DB 视图/RLS 不可行**：SQLite 无 RLS、Go 共享连接池串味需每请求 `SET LOCAL` 独占事务、可见集每用户动态导致视图无法参数化、双方言需两套安全模型。

**④⑥ 是独立的局部契约缺陷**，与上述架构无关，但同样可根治且成本低。

---

## 3. 建议的处置排序（供重新拍板）

| 优先级 | 项 | 动作 | 工作量 | 是否需产品拍板 |
|---|---|---|---|---|
| P0 | ④ | 静态 operlog 覆盖检查 Go 单测化 + CI 硬门禁（同时修 cwd） | S-M | 否 |
| P0 | ⑥ | keyLen/iterations 守卫 + `len(hashBytes)==32` 校验（+ 存量密码盘点） | S | 否（盘点结果可能需） |
| P1 | ⑤ | 删死键/空函数 + `GetByWorkstation` 例外变更联动失效 | S | 允许 5min 陈旧则更小 |
| P1 | ③-A | 删 `RightCast` 方言机制 + 统一 `CAST AS TEXT` + DryRun/SQLite 断言 | S | 否 |
| P1 | ② | 方案 D：5 端点挂可见集 + rooms/photos 补挂中间件 + 保存侧裁剪 + UAT 重跑 | M | 否（UAT 作废需 QA） |
| P2 | ③-B | Go 层合并重写 CTE + 改写 4+ 个「报错即预期」测试 + forbidigo 门禁 | M | 否（性能量级需确认） |
| P3 | ① | **仅在产品重定「范围快照回显」语义后**：A+B 组合根治 | M-H | **是（决定性）** |
| 贯穿 | 共性 | `forbidigo` 禁方言 token → dialect 语义原语 → CI 注入 `TEST_PG_DSN` 双跑 | XS→M | 否 |

**缓解 ≠ 根治的边界（按用户「不打补丁」标准）**：
- ① 加告警通道（FUTURE-01）、粗粒度桶、频率限制 = 缓解；
- ③-B 加 `isPostgres` 方言分支 = 补丁；
- ⑤「接入 Redis 后补一行 Delete」= 无效处方；
- ④ 仅调门禁顺序 = 连缓解都不算（cwd 假绿）；
- ⑥ 只补多块循环不加守卫 = 能力扩展，未消除非法输入静默弱结果。

---

## 4. 需要产品 / DBA / QA 拍板的未决问题

1. **①（决定性）**：是否允许取消 `BackfillDeptResult` 的 `ScopeTotal/ScopeVisible/ScopeOutOfScope/ScopeDeptTotal` 与 `Skipped*` 回显、作废 132-SUMMARY.md:74 前端四类「本次范围：共 N 条」文案？不允许则①只能缓解。
2. **①**：`ImportResult.Errors[].Row` 行号与 `ScopeOutOfScope` 是否可去掉/改粗粒度桶；「范围快照」是否降级为仅服务端 operlog。
3. **②**：4 类几何/附件按归属链 JOIN（零迁移，D）还是补 `org_id` 冗余列（E，需回填 + 双写一致性）；`location-alias.dept_id` 定性为「逻辑映射键」还是「资源归属」。
4. **②**：`133-UAT.md:79-91` D1/D2「确认未收口」作废重跑的窗口与验收人。
5. **③-B**：CI 是否已有 `services: postgres` 可复用（决定 `TEST_PG_DSN` 双跑工作量）；MAC/AD/资产表生产行数（决定 Go 层合并的拉取量）；`REGEXP_REPLACE('\s+')` 与 SQLite 嵌套 REPLACE 的已接受偏差是否适用于接口名。
6. **⑤**：`GetByWorkstation` 5min TTL 是否纳入根治范围（允许陈旧 → 只需删死代码）；死键按注释本应「R3: F4 启用」，是否彻底删除。
7. **⑥**：存量 `sys_user.password` 是否有非 32 字节数据；`VerifyPassword` 对非法长度报错还是返回 false；是否真需要 >32 派生能力（无则只做守卫）。
8. **架构**：是否立项把 `base.GORMRepository` 改造成统一 ScopedDB（回归面覆盖 485 Go 文件 + system/vdi 零回归保证）。
9. **目录治理**：两份 `operlog_e2e_verify.sh` 重复副本是否合并为单一权威（AGENTS.md `scripts/README.md` 登记）。
10. **账务**：FUTURE-01 在 FIX-11 落地后仍未覆盖聚合计数面，是否随①重定语义一并关闭。

---

## 5. 证据索引

| 主题 | 权威文件 |
|---|---|
| 6 项裁定原文 | `.planning/notes/261108-v153-locked-residuals.md` |
| 上游 v1.52 收档快照 | `.planning/MILESTONE-v1.52-SUMMARY.md:78-85` |
| ① 裁定与实现 | `.planning/milestones/v1.52-phases/132-write-scope/132-SUMMARY.md:65,103`；`workstation_service.go:606-682`；`scope_filter.go:225-227` |
| ② 裁定与 UAT | `.planning/notes/261107-v152-ds05-scope-decision.md:39-61`；`.planning/milestones/v1.52-phases/133-scope-final/133-UAT.md:77-94`；`router.go:1174-1177,1269-1355` |
| ③ 双方言范本 | `internal/core/db/sqlite_reconciliation_views.go:14-54`；`workstation_device_service.go:701-799`；`excel_query_builder.go:253-261` |
| ④ CI 范围 | `.github/workflows/ci.yml:62-76`；`scripts/operlog_e2e_verify.sh:52-165`；`internal/utils/operlog/regression_test.go` |
| ⑤ 正确范式 | `internal/services/asset/cache_keys.go:129-134`；`pkg/cache/memory.go:103-113`、`redis.go:697-700` |
| ⑥ 兼容基线 | `internal/core/security/password.go:69-152`；`password_owasp_test.go:61-97` |

> 本报告为**只读判定交付物**：未修改任何代码或裁定文档、未执行 git 写操作、未启动长驻进程。6 项 LOCKED 状态**保持原样**，等待用户重新判定后再立项。
