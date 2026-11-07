# v1.52 / Phase 133 开工输入裁定：DS-05 三项归属 / 范围结论

- **裁定日期**：2026-11-07
- **裁定人**：Lead（用户 2026-11-07 授权自主裁定，按工程最优 + fail-closed 原则）
- **状态**：LOCKED —— Phase 133 **只消费结论 + 落地 + UAT，不承载裁定动作**
- **依据**：`.planning/REQUIREMENTS.md` DS-05 / DS-07 映射总表、`STATE.md` v1.52 范围裁定（P1-4）、`ROADMAP.md` Phase 133 开工输入

> 本文与 `261107-v152-self-scope-decision.md`（Self 档语义）、`261107-v152-batch-import-decision.md`（批量/导入豁免）共同构成 v1.52 三项开工输入裁定，全部在 phase 开工前落盘。

## 裁决总原则

ops 域「有效机构」一律走 `internal/services/operations/dept_filter.go:effectiveOrgExpr` 单一真实源；实体无直接机构列时，按**归属链**回溯到最近的带机构列实体，链断则 **fail-closed 空集**（不回退 All、不猜测）。

---

## 裁定 1：dedicatedLine / roomDevice 归属链 scope

| 实体 | 归属链 | 有效机构表达式（实现口径） |
|---|---|---|
| dedicatedLine（专线） | `ops_dedicated_line.building_id → ops_building.org_id`；无 building_id 时 `org_id` 直挂（若列存在） | 优先直挂列，缺失则 JOIN building 取 `b.org_id`，双侧 `CAST AS TEXT` |
| roomDevice（机房设备） | `ops_room_device.room_id → ops_server_room.building_id → ops_building.org_id` | 两级 JOIN 取 `b.org_id`；任一级断链 → fail-closed 空集 |

**结论**：二者**不定义独立机构语义**，完全继承楼宇机构；归属链断（脏数据 / 软删楼宇）时该行对所有非 All 档不可见、写端点 403。

- 与 `serverRoom`（楼宇链）保持同一族实现，避免第四套子树展开；
- 读端点（`/statistics`、`/dropdown-options`、`/:id`）按 `可见集` 过滤；
- 不新增迁移（不消耗 236 编号），纯查询侧收口。

## 裁定 2：workstation-device 归属

`workstation-device`（`:885-920`，工位↔设备关联）**不定义独立 scope**，跟随其 `workstation_id` 所指工位的可见集：

- 读：以工位可见集为前置条件，未通过工位可见性校验的关联一律不返回；
- 写：Create/Update/Delete 先校验目标工位 ∈ 可见集，越界 **403**（单条写端点，适用 `261107-v152-batch-import-decision.md` §2 的「单条不豁免」）；
- 批量：适用同文档 §2 的「按可见集裁剪」；
- Self 档：工位按 `user_id` 过滤（见 Self 裁定 §2），关联表随之收窄；
- **不存在「设备自带机构」语义**，禁止引入 `device.org_id` 之类的旁路判定。

## 裁定 3：location-alias 查询面

`location-alias`（`:861-874`，位置别名/展示名映射）是**展示型字典**，不含 `org_id/dept_id` 业务归属字段：

- **显式豁免（已知残余面，接受风险）**，与 `REQUIREMENTS.md` 映射总表中 `walls / doors / floor-plan-texts / rooms/photos` 同批处置；
- 理由：越权读取不泄露业务归属信息；Three.js 编辑器联动回归面大，收口收益低于风险；
- **但不以「间接可见」结案**：登记残余风险 = 别名文本可能包含机构名称等人工录入内容，若未来录入规范允许含敏感机构名，应升级为按关联楼宇过滤；
- 133 的 UAT 清单中登记一条「location-alias 豁免复核」人工项，不阻断自动验收。

---

## 附：v1.52 全量「显式豁免」清单（终态登记）

> **审查补正（2026-11-07，agent-8 独立审查 P1-1）**：`GET /dept-mapping-template`（`internal/api/router.go:818` → `internal/api/v1/operations/excel_handler.go:227-237 DownloadDeptMappingTemplate`）返回全库部门树（部门名 / 全链路 parent_path / 部门代码），零范围过滤，**原清单未点名，属于第 4 个未登记越权面**。按本文「其余 ops 端点一律收口」的总原则，裁定：**不豁免，由 Phase 131 注入可见集过滤**（`sys_dept.id ∈ 可见部门集`，All 档短路不变，不改响应形状）。已在下表中补登记为「已收口」而非豁免。

| 面 | 豁免理由 | 残余风险 |
|---|---|---|
| walls / doors / floor-plan-texts（独立 `/list` + `/:id`） | 返回楼层平面几何坐标/展示文字，无业务归属列 | 几何图可能间接暴露布局；Three.js 编辑器联动回归面大 |
| rooms/photos（独立 `GET /list/:roomId` + `GET /:id`） | 照片/房间附件，模型无机构列 | 附件内容人工录入可能含敏感信息 |
| location-alias（裁定 3） | 展示型字典，无归属列 | 别名文本可能含机构名 |
| dept-mapping-template（审查 P1-1 补收口） | **不豁免**：Phase 131 按可见集过滤部门树 | 全链路 parent_path 含父部门名，过滤后仍可能暴露上级名称；登记为已知残余 |

**以上三项均为「已知残余面、接受风险」，不是「间接可见所以安全」的结案**；其余 ops 端点一律收口（含 `workstations-all(:813)`、Excel 导入/导出）。

---

## Phase 133 消费清单（不重新裁定）

1. 四实体（asset / serverRoom / dedicatedLine / roomDevice）读写 403 矩阵 + 归属链过滤（裁定 1）；
2. workstation-device 按工位可见集收口（裁定 2）；
3. location-alias 登记豁免 + UAT 人工复核项（裁定 3）；
4. vdi 生产调用方迁移到 ScopeProvider + 删 `pkg/middleware` 死实现与 `permission_74_08_test.go` 死测试（Self 裁定 §4）；
5. Dept/Self 批量语义推广到四实体（批量裁定 §5）；
6. 前端 DeptSidebar 按可见集过滤 + UAT。

## 残余风险登记

- 归属链断链的处理为 fail-closed，若生产存在历史脏数据（building 软删但专线仍引用），非超管将看不到这些专线 → 建议 133 的 UAT 增加一条脏数据抽查；本裁定不为脏数据开旁路。
- 两级 JOIN 的性能需在 PG contract 中留一条 `EXPLAIN` 观察项（非阻断）。
