# v1.53 已裁定接受 / 仅登记残余结论（ACK-01 / ACK-02）

**日期**：2026-11-08
**性质**：LOCKED / 仅登记，**不驱动代码修复**。本文件是 Phase 141 的正式交付物，结论落盘后由 GSD 自动生成的 `v1.53-ROADMAP.md`（issues / tech-debt 段）与 `MILESTONES.md` v1.53 条目承载，不再维护手工 A/B/C 节清单。
**权威来源**：`.planning/MILESTONE-v1.52-SUMMARY.md:78-85`（v1.52 收档时点快照的「已知残余」权威登记）；2026-11-08 开放项盘点 A8 / B13①②（本次里程碑的临时输入清单，取数用）。

---

## ACK-01 — v1.52 收档已裁定接受 / 独立登记的 4 项残余

### ① P1-5 聚合计数侧信道（接受，不改代码）
- **裁定**：越界行数 / N / 行号仍为枚举侧信道；v1.52 收档明确裁定「已知残余风险、**接受**；未来应加**告警通道而非改回整批 403**」。
- **残余风险**：聚合接口仍可经响应体数值枚举推断越界规模；不泄露业务字段本身。
- **不改代码理由**：改回整批 403 会破坏 v1.52「按可见集裁剪 + 显式 403」的既定语义与已交付 UAT；裁定方向是补可观测告警通道（与 FIX-11 同步漂移告警通道同族，合并评估为 FUTURE-01）。
- **触发复核条件**：若未来出现「侧信道被实际利用」的证据，或 FIX-11 告警通道落地后需一并覆盖聚合计数面 → 另立项。
- **证据**：`MILESTONE-v1.52-SUMMARY.md:80`、`phases/132-write-scope/132-SUMMARY.md §6`（归档路径见 `ARCHIVE-INDEX.md`）。

### ② 5 项显式豁免面（ds05 裁定接受风险）
- **清单**：walls / doors / floor-plan-texts / rooms-photos / location-alias。
- **裁定**：ds05 裁定「已知残余面、**接受风险，不以『间接可见』结案**」；这些独立可直接调用的读端点返回的模型不含 `org_id` / `dept_id` 业务归属字段，不承载机构敏感信息；Three.js 编辑器联动回归面大，收口成本与收益不匹配。
- **残余风险**：几何坐标 / 展示文字 / 照片附件在已知楼层范围内可被越权读取（不含归属信息）。
- **不改代码理由**：v1.52 终态已逐项登记理由与残余风险，属收档时明确接受的终态，非遗漏。
- **证据**：`MILESTONE-v1.52-SUMMARY.md:81`、`phases/133-scope-final/133-UAT.md`（归档路径见 `ARCHIVE-INDEX.md`）、`REQUIREMENTS.md`（v1.52 归档副本）端点→DS 映射总表的「显式豁免」两行。

### ③ PG 专属 SQL 债（独立登记，scope constrainment）
- **清单**：
  - `internal/services/operations/excel_export_service.go:148` 的 `RightCast:"text"` 动态拼接（infopoint 导出）；
  - `internal/services/operations/workstation_device_service.go` 的 PG 专属 CTE（`DISTINCT ON` / `REGEXP_REPLACE` / `NULLS LAST`）。
- **裁定**：与 P1-8（已在 v1.52 二次修复中改为双侧 `CAST AS TEXT`）**相互独立**的技术债，其 SQLite 报错已被既有测试文档化为预期；v1.52 明确「本轮不动」。
- **残余风险**：这两条路径仅在 PG 可用，SQLite 方言下不可用；生产为 PG，不影响现行部署。
- **不改代码理由**：属独立技术债，改动面涉及导出查询与工位设备链路，超出「开放项清偿」的 LOCKED 口径；按 scope constrainment 保留登记。
- **证据**：`MILESTONE-v1.52-SUMMARY.md:84`。

### ④ `operlog_e2e_verify.sh` 既有语义缺陷（独立登记）
- **现状**：`scripts/operlog_e2e_verify.sh:57-66` 凭据门禁先于 `SKIP_LIVE:162`，导致静态检查实际跑不到；修通 Windows bash 路径拼接 127 后，两 e2e 包走「凭据缺失 → 可观测 skip → PASS」，属既有行为。
- **裁定**：v1.52 收档「单独登记」，非本轮引入、非阻断。
- **残余风险**：静态检查在缺凭据环境下被门禁跳过，覆盖率口径需人工留意。
- **不改代码理由**：既有语义缺陷 + 单独登记结论；调整门禁顺序会改变 e2e 脚本语义，超出本里程碑 LOCKED 范围。
- **证据**：`MILESTONE-v1.52-SUMMARY.md:85`。

---

## ACK-02 — B13 仅登记 / 已知约束 2 项

### ① `invalidateCache` 无 Redis 场景 no-op（R3+ TODO）
- **现状**：`internal/services/asset/reconciliation_exception.go:578-580` 的 `invalidateCache()` 在无 Redis 场景为 no-op；例外规则写后缓存未失效。
- **裁定**：低优先级 / 已知约束，**仅登记、不补实现**。告警通路由 `DetectLayer3` 内存匹配驱动、不依赖该缓存，缓存陈旧窗口在 cron 周期内可接受。
- **触发复核条件**：项目正式接入 Redis 后，按注释补 `core.Cache.Delete(ctx, CacheKeyReconciliationExceptionRuleList)`。
- **证据**：`reconciliation_exception.go:579`、`internal/services/asset/reconciliation_exception.go` 内注释。

### ② `pbkdf2SM3` 未实现 `keyLen>32` 多块循环
- **现状**：`internal/core/security/password.go:67` 未实现 `keyLen>32` 的多块循环；**全部调用点 `keyLen=32`**，安全不受影响。
- **裁定**：已知约束，**仅登记、不补实现**（无实际调用面）。
- **触发复核条件**：未来出现 `keyLen>32` 的调用点时，须补全多块循环并配对应用例。
- **证据**：`password.go:67`。

---

## 与「权威交回 GSD」的关系

1. 本文件的结论由 **`.planning/milestones/v1.53-REQUIREMENTS.md` 的 traceability / Future Requirements / Out of Scope 段**（ACK-01/02 两项 REQ + 6 项列名 + FUTURE-01）与 `.planning/MILESTONES.md` v1.53 条目（Known Gaps「LOCKED 残余」行）承载；`.planning/milestones/v1.53-ROADMAP.md` 仅在 Phase 141 验收段与「关键约束」段写入「LOCKED 项不写修复代码」（**该文件并无 issues / technical-debt 段**，2026-11-09 Lead 流向复核更正——原表述指向不存在的段，勿再据此引用）；本文件作为 `.planning/notes/` 下的裁定原文保留。
2. **不再**在临时清单里维护「移入 C 节」；临时清单整体归档后，其 A/B/C 节不再是任何当前状态的判据。
3. ~~FUTURE-01（P1-5 告警通道产品化）与 FIX-11 同步漂移告警通道合并评估，属未来里程碑候选~~ → **✅ v1.55 收档已关闭（2026-11-09）**：用户裁定允许推翻 ① 的「范围快照回显」产品语义，① 改由 A（Phase 150 响应体去规模化）+ B（Phase 151 统一 ScopedDB 防回归）从根源根治，P1-5 侧信道已消除，告警通道合并评估不再需要；本裁定原文 ① 的「接受/不改代码」结论按清偿结果由 `.planning/STATE.md ## Deferred Items` 与 `.planning/MILESTONES.md` v1.55 条目承载（本文件 6 项裁定原文保持只读）。
4. 待人工验证类残余（VAL-01~06）的承载物是 `STATE.md` 的 `## Deferred Items` 表 + `MILESTONES.md` + `.planning/notes/261108-human-verification-tracker.md`，不属本文件范围。
