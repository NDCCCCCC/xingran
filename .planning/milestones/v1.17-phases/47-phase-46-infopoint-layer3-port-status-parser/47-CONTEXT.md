# Phase 47: 修复資產對賬系統根因（infoPoint 配置、Layer3 引擎、port_status 漂移、parser 校驗） - Context

**Gathered:** 2026-07-03
**Status:** Ready for planning

<domain>
## Phase Boundary

**根因修复** phase — Phase 45 R5 reconciliation 系统识别出 4 个数据完整性问题：

| R | 修复目标 | 状态 |
|---|---------|------|
| R1 | 4F001 工位 infoPoint 4D217 指向错设备的 port_status（drift 引发） | ✅ **已闭合**（`migration_188_fix_info_points_port_id_drift.go`） |
| R2/R3 | sys_data_reconciliation Layer 3 引擎只 INSERT 不 UPDATE 既有 conflict；124 个 D 异常卡住无法重新分类；不再吃 SQLSTATE 23505 | ❌ **本 phase 处理** |
| R4 | sys_device_port_status 行 id 与 device_id 一一对应，永不漂移 | ✅ **已闭合**（`migration_183_add_port_status_device_fk.go`，FK `fk_port_status_device`） |
| R5 | parseRuijiePortSecurityLine 加 MAC 格式校验（12位 hex），过滤 `#`/`flags`/`total` 等噪声 | ❌ **本 phase 处理** |

**Phase 47 实施范围 = R2/R3（UPSERT）+ R5（parser 校验 + 数据清理脚本）**，R1/R4 锁为"已闭合"，不重复实施。

**已锁定的高层决策**（沿用 v1.17 R1-R4 / v0.3-v0.5 / ROADMAP / 项目记忆 `user-prefers-code-fixes-no-db-triggers`，不重复展开）：
- 策略：Observe-only + 不写 DB TRIGGER（代码层修复优先）
- Layer 3.5 例外匹配已在循环内一次写入（`exception_rule_id` + `applied_actions`，R3 锁定）
- 7d 静默期 + 24h 节流 + auto-resolve-on-healthy（Type A 路径）保留
- operlog 强制约定 + 25 OperType 常量
- `auto-resolve-on-healthy` 已新增（D-09-fix-01）— Phase 47 不改动此路径

**与 R5 的关系边界**：
- R5 改写 `mac_collection_service.go:parseRuijiePortSecurityLine` 内部校验 — 不改 `parseMACLine`（其已经校验）
- 一次性数据清理 migration 与 UPSERT migration **同 batch** 提交
- 仅软删除（`deleted_at = NOW()`） + 保留 `mac_history` 历史链不断（AUDIT-02）

</domain>

<decisions>
## Implementation Decisions

### R2/R3 — Layer 3 UPSERT 语义（Area 1）

- **D-01：复用 partial uniqueIndex `uniq_recon_asset_type_open`**：
  - 该索引约束 `(asset_id, conflict_type) WHERE resolved_at IS NULL AND deleted_at IS NULL` 已在 `migration_168_reconciliation_tables.go` 落地
  - UPSERT 用 GORM `.Clauses(clause.OnConflict{ Columns: []clause.Column{{Name:"asset_id"},{Name:"conflict_type"}}, Target: "uniq_recon_asset_type_open", DoUpdates: clause.Assignments(...) })`
  - **不**新建 partial uniqueIndex；**不**改用普通 `(asset_id, conflict_type)` 全 unique
  - 已 resolved 行不参与 conflict index → 不会再被 UPSERT 重写（审计链不断 + R2 7d 静默期语义保留）
  - 同时移除 `isReconciliationDuplicate(err)` 的 23505 catch 静默 skip（不再有死信，"inserted" 计数现在包含 UPSERT 命中）

- **D-02：UPSERT 升级集（DoUpdates SET 字段）**：
  ```
  severity           = EXCLUDED.severity,
  raw_snapshot       = EXCLUDED.raw_snapshot,
  physical_value     = EXCLUDED.physical_value,
  declared_value     = EXCLUDED.declared_value,
  ad_value           = EXCLUDED.ad_value,
  asset_ip           = EXCLUDED.asset_ip,
  exception_rule_id  = EXCLUDED.exception_rule_id,
  applied_actions    = EXCLUDED.applied_actions,
  confidence_score   = EXCLUDED.confidence_score,
  detected_at        = NOW()
  ```
  - `resolved_at` / `resolved_by` / `resolution_note` **不更新**（已 resolved 的不参与 conflict index 路径，未 resolved 的保持 NULL）
  - `id` / `created_at` / `deleted_at` 由 PG default + ON CONFLICT 默认不重写
  - `detected_at = NOW()` 而非 EXCLUDED.detected_at：原检出时间更稳定，Phase 45 R4 7d 健康窗口按此计算

- **D-03：DetectLayer3 返回签名保留**（不破坏现有调用方契约）：
  - `func DetectLayer3(ctx) (inserted, skipped, skippedSilence, skippedThrottle int, err error)` 不变
  - UPSERT 命中计入 `inserted`（不再是 skipped）— 让 dashboard `exception_by_type` 与 `inserted_rate` 真实反映工作
  - 仅 7d 静默期命中计入 `skippedSilence`，仅 24h 节流命中计入 `skippedThrottle`，其他仍 `skipped`（Type A + auto-resolve 等）

### R5 — MAC parser 校验（Area 2）

- **D-04：parseRuijiePortSecurityLine 末加 `isCanonicalMAC` 校验**：
  ```go
  entry.MACAddress = NormalizeMACAddress(fields[2])  // 既有
  // —— 新增 ——
  if entry.MACAddress == "" || !isCanonicalMAC(entry.MACAddress) {
      return MACAddressEntry{}, false
  }
  // —— END ——
  ```
  - 复用 `internal/services/mac_normalize.go:32` 的 `isCanonicalMAC`（已存在，标准大写冒号正则 `^[0-9A-F]{2}(:[0-9A-F]{2}){5}$`）
  - 与 `parseMACLine:502` 同样的丢弃语义，不入库不入 mac_history
  - **不**改 parseRuijiePortSecurityLine 其余逻辑（VLAN 接口名解析保留）

- **D-05：解析层零副作用 = 软删除清理 migration**：
  - 新建 `migration_NNN_cleanup_dirty_mac_rows.go` 与 UPSERT migration 同 batch
  - 先备份脏行（保留审计链）：
    ```sql
    CREATE TABLE sys_dirty_mac_rows_backup AS
      SELECT id, device_id, mac_address, interface_name, collected_at, created_at
      FROM sys_mac_address
      WHERE mac_address IS NOT NULL
        AND mac_address !~ '^[0-9A-F]{2}(:[0-9A-F]{2}){5}$';
    ```
  - 软删除脏行：
    ```sql
    UPDATE sys_mac_address
      SET deleted_at = NOW()
      WHERE mac_address IS NOT NULL
        AND mac_address !~ '^[0-9A-F]{2}(:[0-9A-F]{2}){5}$'
        AND deleted_at IS NULL;
    ```
  - **不**动 `mac_history`（脏数据的轨迹本身是审计价值，保留）
  - 幂等保护：再次执行时 SELECT 过滤已 `deleted_at IS NOT NULL`

### Claude's Discretion

下列实现细节由 planner/researcher 在 plan-phase 自决：

- UPSERT migration 文件名编号（接在 `migration_193` 之后最近可用号，建议 `migration_194`）
- 数据清理 migration 与 UPSERT 是否合并为 1 个 migration 文件、还是拆 2 个独立 `migration_194` + `migration_195`（推荐拆 2：职责单一、回滚友好）
- DetectLayer3 内部变量命名（`rec.ID = ""` 显式置空确保 GORM 不带 PK UPDATE）
- 数据清理 migration 的备份表是否建索引（按 `device_id, mac_address`）— 不建即可，仅 7d 内查
- 是否在数据清理 migration 输出一条 `logrus.Infof("cleaned %d dirty MAC rows", affected)` 统计
- 是否升级 `parseRuijiePortSecurityLine` 的单元测试 `TestParseRuijiePortSecurityLine` 增 `isCanonicalMAC` 负向用例（推荐：增加表头行/汇总行/#注释行/空字段 4 个 sub-test）

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents (researcher / planner) MUST read these before planning or implementing.**

### 前序上下文（必读，R2/R3 直接构建于其上）
- `.planning/phases/42-r1/42-CONTEXT.md` — R1 全部 18 个决策（D-01~D-18），含 sys_data_reconciliation schema、partial uniqueIndex `uniq_recon_asset_type_open`、cron sys_job 模式、operlog 边界
- `.planning/phases/43-r2/43-CONTEXT.md` — R2 转单 cron + WS/SysNotice + resolve API + 7d 静默期/24h 节流 guard + auto-resolve-on-healthy（D-09-fix-01）
- `.planning/phases/44-ip-r3/44-CONTEXT.md` — R3 Layer 3.5 例外匹配（applied_actions + exception_rule_id 写入点的 D-R3-A1-02）
- `.planning/phases/45-r4/45-CONTEXT.md` — R4 整合 + D-A2-02 "本周"窗口语义 + D-A4-04 缓存失效触发

### 已闭合项验证（不重实施）
- `internal/core/db/migrations/migration_183_add_port_status_device_fk.go` — R4 FK `fk_port_status_device` (device_id → sys_network_device.id)
- `internal/core/db/migrations/migration_188_fix_info_points_port_id_drift.go` — R1 修复脚本

### 关键实施定位（必读）
- `internal/services/asset/reconciliation_detection.go:209-404` — DetectLayer3 循环；**R2/R3 INSERT 段在 :385-394**，需替换为 UPSERT
- `internal/services/asset/reconciliation_detection.go:407-420` — `isReconciliationDuplicate` helper（D-01 UPSERT 引入后失去作用，但保留不删 — 防回归 / 单元测试可复用）
- `internal/services/mac_collection_service.go:515-540` — `parseRuijiePortSecurityLine` 函数体，**R5 校验插入点 :540 前**（return 前）
- `internal/services/mac_collection_service_test.go:196` — `TestParseRuijiePortSecurityLine` 既有测试（D-05 推荐补 4 个负向 sub-test）
- `internal/services/mac_normalize.go:32-34` — `isCanonicalMAC` 函数（已存在，R5 复用）
- `internal/models/reconciliation.go` — `SysDataReconciliation` 模型字段定义（D-02 UPSERT SET 列名以实际 DB schema 为准，非 GORM 推导）

### 项目记忆（规划时必查，已在 MEMORY.md）
- `user-prefers-code-fixes-no-db-triggers` — **禁 DB TRIGGER 路线**（PG-level REGEXP_REPLACE BEFORE INSERT/UPDATE 被用户拒绝）；本 phase 走代码层校验
- `migration-sql-name-must-match-model` — UPSERT/SET 列名以实际 DB schema 为准（D-02 字段名核对）
- `xingran-migrations-no-sql-autoloader` — migrations/*.sql 不会被自动加载，必须用 `migration_NNN_*.go` 显式调用
- `xingran-gorm-sql-constraint-naming-conflict` — GORM `uniqueIndex` 命名 `uni_*_*`，本 phase 用既有 PG-level partial index 名称 `uniq_recon_asset_type_open`（D-01 已锁）
- `GORM AutoMigrate 被 PG 物化视图阻塞` — UPSERT migration 必须不在 reconciliation_normalized 重构前后空白窗口期内
- `mac-collection-fresh-binary-still-dirty` — R5 解析层修复需新二进制部署 + 重启 cron 守护（GOCACHE 风险已被反复验证）
- `mac_normalize.go` 已下沉 `pkg/normalize.MACAddress` — R5 复用既有函数，不二次封装

### 项目级 CLAUDE.md（强约束）
- `CLAUDE.md` "操作日志记录约定 (operlog convention) — 强制" — 11 关键词 + 25 OperType 常量
- `CLAUDE.md` "Status Value Convention" — 0=启用 1=停用
- `CLAUDE.md` "Migration 编写模板" — 编号递增 + AutoMigrate 注册

### 现有代码参考（实施时查）
- `internal/services/asset/reconciliation_detection_test.go` — DetectLayer3 既有测试，D-03 返回签名变更后需补 UPSERT 分支用例
- `internal/services/mac_collection_service_test.go:196` — R5 sub-test 锚点
- `internal/core/db/migrations/migration_168_reconciliation_tables.go` — partial uniqueIndex `uniq_recon_asset_type_open` DDL 原点
- `gorm.io/gorm/clause` — `clause.OnConflict{ Columns, Target, DoUpdates }` 接口（D-01 直接使用）

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `internal/services/asset/reconciliation_detection.go:DetectLayer3` — 循环结构 + Layer 3.5 例外匹配 + 24h 节流 + 7d 静默期已有，**R2/R3 只改 INSERT 段**（:385-394）即可
- `internal/services/mac_normalize.go:isCanonicalMAC` — 复用既有函数，零重复
- `internal/core/db/migrations/migration_168_reconciliation_tables.go:uniq_recon_asset_type_open` — partial uniqueIndex，D-01 直接复用作为 ON CONFLICT target
- `gorm.io/gorm/clause.OnConflict` — GORM 原生 UPSERT DSL，D-01 用 `Target` 字段引用 partial index name

### Established Patterns
- **GORM UPSERT 模式**：`db.Clauses(clause.OnConflict{...}).Create(rec)` — R2/R3 跟随项目既有用法
- **migration 编号递增**：已有 193，接 194/195（推荐拆 2）
- **operlog 强制约定**：DetectLayer3 是 cron 触发的后台任务，**不入 operlog**（既有注释 "R1 detection 是后台 cron，operlog 跳过以免心跳刷屏"，保留）
- **软删除（soft delete）模式**：`deleted_at = NOW()`，不物理删除（D-05 严格遵循）
- **备份表后清理模式**：参考 `migration_188_fix_info_points_port_id_drift.go` (line 99+ `CREATE TABLE sys_info_points_port_id_drift_backup`)，同样模式套用到 dirty MAC rows

### Integration Points
- `internal/services/asset/reconciliation_detection.go:385-394` — INSERT 段替换位置（D-02 DoUpdates 清单）
- `internal/services/mac_collection_service.go:540` — `parseRuijiePortSecurityLine` return MACAddressEntry 前插入 `isCanonicalMAC` 校验
- `internal/core/db/migrations/migration_NNN_cleanup_dirty_mac_rows.go` — 新建（推荐 194 或 195）
- `internal/services/mac_collection_service_test.go` — TestParseRuijiePortSecurityLine 新增 4 个 sub-test
- `internal/services/asset/reconciliation_detection_test.go` — 新增 UPSERT 冲突 + UPSERT 升级 sub-test（含 raw_snapshot 字段一致性）

</code_context>

<specifics>
## Specific Ideas

- **R2/R3 UPSERT 性能影响**：现有 open 表行数 << 10K，D-02 一次最多 1-2 个 conflict_type 升级，时延 < 50ms 几乎无感。不需 cron 周期调短。
- **R5 解析层校验 "2 道防线" 语义**：项目记忆 `mac-collection-fresh-binary-still-dirty` 已暗示解析层仅兜底其一；下个 phase（v1.18 RQ-002 路线）可加 transport 层（设备 raw_text → struct 的入口处）与 DB 层（非本 phase，禁 trigger）。本 phase 只动解析层。
- **脏行备份表保留期**：建议 30 天后续手动 DROP；不写 cron（DROP 本身是审计敏感动作）。
- **DetectLayer3 单元测试可观测性**：UPSERT 升级的 raw_snapshot 应当用同一 raw snapshot + 新 severity；测试用 sqlmock 验证 DoUpdates SET 字段完整。
- **变更日志（D-04 实施提示）**：
  ```
  // 2026-07-03 phase 47 R5: 写库前 canonical MAC 校验, 防止 'Flags:'/'total'/'#'/注释行入库污染
  if entry.MACAddress == "" || !isCanonicalMAC(entry.MACAddress) {
      return MACAddressEntry{}, false
  }
  ```

</specifics>

<deferred>
## Deferred Ideas

下列决策**显式推后**，本 phase 不实现：

### 已闭合项（不重实施，告知而非折叠）
- **R1 (infoPoint.port_id drift)** — `migration_188_fix_info_points_port_id_drift.go` 已完成
- **R4 (port_status id/device_id drift)** — `migration_183_add_port_status_device_fk.go` 已完成（含 NOT VALID + VALIDATE 异步验证）

### 显式不做（不允许 scope creep）
- Phase 47 不动 R5 之外的现有 4 处 MAC parser（`parseMACLine` 已校验；huawei/h3c/maipu 等 TextFSM 模板无 batch input 风险）
- 不引入 DB TRIGGER（项目记忆 `user-prefers-code-fixes-no-db-triggers` 锁定）
- 不改 `auto-resolve-on-healthy` (D-09-fix-01) 路径
- 不动 7d 静默期/24h 节流阈值（按 R3/R2 锁定）
- 不动 `isReconciliationDuplicate` 函数（D-01 仍保留以防回归测试）
- 不动 mac_history 清理（脏数据的历史链本身是审计价值，符合 AUDIT-02）

### 下个 phase 候选（如有需要）
- v1.18 RQ-002 路线：transport 层校验（设备 raw_text → struct 入口，加 `strings.ContainsAny(text, "Flags:|Total|#")` 过滤）
- v1.18 RQ-002 路线：MAC parser fuzz test（go test -fuzz）

### Reviewed Todos (not folded)
`cross_reference_todos` 命中 2 项（score=0.6，均低分），审阅后**不折叠**进 R2/R3 + R5 scope：
- `operlog-exclude-paths.md`（operlog.exclude_paths 配置驱动白名单）— Phase 35 范围，与 R2/R3、R5 无关
- `v1.17-reconciliation-decisions.md`（v1.17 决策点追踪，`resolves_phase: 42`）— R2/R3 + R5 相关项已被本 CONTEXT D-01~D-05 锁定；不重复跟踪

</deferred>

---

*Phase: 47-修复資產對賬系統根因（infoPoint 配置、Layer3 引擎、port_status 漂移、parser 校驗）*
*Context gathered: 2026-07-03*
