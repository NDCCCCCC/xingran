# v1.52 / Phase 132 开工输入裁定：Dept/Self 批量与 Excel 导入豁免

- **裁定日期**：2026-11-07
- **裁定人**：Lead（用户 2026-11-07 授权自主裁定，按工程最优 + fail-closed 原则）
- **状态**：LOCKED —— Phase 132 依此定案；Phase 133 只消费结论
- **依据**：`.planning/REQUIREMENTS.md` DS-04 / DS-07、`STATE.md` v1.52 硬约束、`ROADMAP.md` Phase 132 开工输入

## 1. 结论总表

| 场景 | 裁定 | 语义 |
|---|---|---|
| 单条 Create / Update / Delete 越界 | **不豁免** | **403 拒绝整条请求**（DS-04 明文） |
| 批量 BatchOperation（四域 Delete/Batch） | **不豁免**，但**按可见集裁剪目标集** | 只处理可见集内的行；越界行剔除 + 计数入 operlog；**不因含越界行而 403 整批** |
| backfill-dept 回填 | **不豁免**，按可见集裁剪 | 预览与执行两次都按同一可见集裁剪；前端确认框显示范围（见 §3） |
| Excel 导入 | **不豁免** | 逐行校验 org_id/dept_id ∈ 可见集；越界行计入 `ImportResult` 错误明细，**不阻断整批** |
| Excel 导出 | **不豁免**（DS-06 已定，Phase 131 落地） | 两条独立查询路径注入可见集过滤 |
| 跨机构批量导入的正当场景 | **不开豁免口子** | 由「超管（All 档）执行 + operlog 审计强化」满足 |
| 超管（admin / All） | **语义不变** | 全部放行，零回归 |

## 2. 为什么批量采用「裁剪」而非「整批 403」

安全目标是「**不写越界数据**」，裁剪已经完整达成：可见集外的目标行根本不进入 UPDATE/DELETE 集合，攻击者既不能改数据、也不能通过响应差异枚举越界 ID（越界行统一计入聚合计数，不逐条回显）。

- 整批 403 无额外安全收益，却会破坏 `quick-261106-wsb` 已交付的 backfill 可用性：backfill 本质是「全库兜底扫描」动作，若任一越界行即拒整批，则任何非超管都无法使用。
- 与项目既有 Excel 导入范式一致（`匹配失败留空不阻断导入`，PROJECT.md Validated 段）。
- 单条写端点仍严格 403，因为单条请求的「目标唯一」不存在可用性权衡，且 403 是 DS-04 的明文验收口径。

**裁剪的硬约束（防止裁剪变绕过）**：

1. 裁剪必须在 **service 层**、在**任何写 SQL 之前**完成，且以 `ScopeResult`（Phase 130）为唯一输入；禁止在 handler 层先查后传 ID 列表（TOCTOU）。
2. 目标集查询必须一次性带可见集条件（如 `WHERE id IN ? AND <有效机构 ∈ 可见集>`），**不得**「先取全量 ID 再在内存过滤后写」——后者仍会写出可见集外的行（若过滤实现有 bug）。
3. 越界计数必须写 operlog（module 用既有中文名，OperType 用 `OperTypeBatch`/`OperTypeUpdate`），含 `scope档位 / 请求行数 / 可见集内行数 / 越界行数`。
4. All 档短路时同样记录 `All` 档位，保证审计链完整。

## 3. backfill 前端确认框范围文案（Phase 132 验收项）

- 预览与确认两步使用**同一个** `ScopeResult` 快照，禁止确认阶段重新解析（防范围漂移）。
- 文案规范：
  - All 档：`本次范围：全库（共 N 条）`
  - Custom/Dept/DeptChild 档：`本次范围：{部门名}及子部门（共 N 条）`，多部门时列前 3 个 + 「等 M 个部门」
  - Self 档：`本次范围：仅本人关联工位（共 N 条）`
  - 异常枚举 fail-closed：`本次范围：无（数据权限异常，请联系管理员）` 且确认按钮 disabled
- 前端断言由 vitest 组件测试锁定文案与 disabled 状态。

## 4. Excel 导入落点（Phase 132 / PG contract 覆盖）

- 导入路径在 `excel_service.go` 的 `BatchUpsert` / 引用解析链；每行解析出的 `org_id`（楼层覆盖优先的「有效机构」）与 `dept_id` 都要过可见集。
- 越界行：**不写入、不阻断整批**，计入 `ImportResult` 的越界错误明细（含行号，不含敏感值）。
- 复用 `effectiveOrgExpr` 单一真实源（`dept_filter.go`），禁止内联复制。
- uuid/varchar 比较双侧 `CAST AS TEXT`（PG 42883 教训），新增导入路径查询纳入 `pg_contract_test.go` 的 `TEST_PG_DSN` 门控覆盖。
- 导入 operlog 强化：`OperTypeImport` + 范围档位 + 越界行数；导出同理（`OperTypeExport`）。

## 5. 与 Phase 130 / 131 / 133 的交接

- 130 交付 `ScopeResult` + `Resolve`（本裁定所有「可见集」均以它为准）；
- 131 已按 DS-06 落地导出不豁免，132 不重复动导出代码；
- 133 按本裁定把「Dept/Self 批量」语义推广到 asset/serverRoom/dedicatedLine/roomDevice，**不重新裁定**。

## 6. 残余风险登记

- 批量裁剪把「越界」降级为审计计数而非 403，若未来要求「越界即告警」，应在 operlog 之上加告警通道，而不是改回整批 403（会破坏 backfill）。
- 逐行导入校验会增加导入耗时（每行一次集合判断）：实现用集合 map 做 O(1) 判定，避免逐行查库。
