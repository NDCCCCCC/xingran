---
tracking_copy: true
copied_from: .planning/milestones/v1.52-phases/133-scope-final/133-UAT.md
copied_by: Phase 140 / VAL-05
copy_date: 2026-11-08
original_is_archive: true
original_must_not_be_modified: true
status: 跟踪副本 / 归档原件不修改
---

# v1.52 Phase 133 人工 UAT 清单 — 跟踪副本（VAL-05）

> **本文件是 .planning/milestones/v1.52-phases/133-scope-final/133-UAT.md 的跟踪副本**，
> 由 2026-11-08 Phase 140（VAL-05）按 `.planning/OPEN-ITEMS.md` #A8 附带说明
> （本次里程碑的临时输入清单，取数用）与 `.planning/MILESTONE-v1.52-SUMMARY.md:78-84`「已知残余权威登记」
> 建立。**归档原件不修改**；本副本用于统一跟踪表逐项登记与人工执行后的证据挂载。
>
> **口径**：全部条目保持 `- [ ]`（未勾选 / 待执行），**严禁伪造已完成 / 已勾选**。
> 依据 REQUIREMENTS.md VAL 共同约定与第 13 行权威回收裁定：实际执行需真实 PG + 跨角色浏览器；
> 执行完成后在 GSD 自动生成的产物或未来里程碑的 REQUIREMENTS / UAT 清单中登记证据并勾选，
> **未执行不得勾选**。回写与跟踪见 `.planning/notes/261108-human-verification-tracker.md` 与
> `.planning/notes/261108-uat-writeback-policy.md`。

## 分段与计数（跟踪副本口径）

| 段 | 条目 | 条数 | 状态 |
|---|---|---|---|
| 前置（开关工波次确认） | 133-P1~P3 | 3 | 全部 `- [ ]` 待执行 |
| A. 四实体读写 403 矩阵 | 133-A1~A11 | 11 | 全部 `- [ ]` 待执行 |
| B. workstation-device 按工位可见集 | 133-B1~B4 | 4 | 全部 `- [ ]` 待执行 |
| C. ds05 残余风险专项 | 133-C1~C2 | 2 | 全部 `- [ ]` 待执行 |
| D. 显式豁免面复核 | 133-D1~D3 | 3 | 全部 `- [ ]` 待执行 |
| E. vdi 迁移 + 死实现删除 | 133-E1~E4 | 4 | 全部 `- [ ]` 待执行 |
| F. 前端 DeptSidebar 可见集联动 | 133-F1~F4 | 4 | 全部 `- [ ]` 待执行 |
| G. 提交与闸门复核 | 133-G1~G6 | 6 | 全部 `- [ ]` 待执行 |
| **合计** | | **37** | **全部待执行（0 勾选）** |

> 说明：OPEN-ITEMS.md #A8 表述为「A1-A11 / B1-B4 / C / D / E / F / G，约 30 项」；
> 归档原件实际含 37 个 `- [ ]` 复选框（含前置 3 项与 G 段 6 项）。本跟踪副本逐项全录，不删不并。

## 前置：开关工波次确认（已由自动验收锁定 — 人工复核项）

- [ ] **133-P1**：`git log --oneline -12` 可见 130（`06e892ee` + `cb04d0f1`）、131、132 三波 commit 均落地
- [ ] **133-P2**：`git ls-files` 含 `internal/services/operations/scope_filter.go` 与 `scope_write.go`
- [ ] **133-P3**：130 已在 ops 各读组/写组挂 `DataScopePermission`（本 phase 未动 router.go）

## A. 四实体读写 403 矩阵（人工复核）

> 自动验收：service 层 ScopeResult 注入 + 403 矩阵单测（SQLite）。
> 人工项在真实 PG + 真实浏览器跨角色复核「越界不可见 / 越界写 403 / 超管零回归」。

- [ ] **133-A1 asset（dept_id 直挂）**：Dept 档用户分别访问 `/ops/asset/list`、`/statistics`、`/dropdown-options`、`/:id`，断言只返回 `dept_id ∈ 可见集` 的资产；越界 ID 的 GetByID 返回空/404
- [ ] **133-A2 serverRoom（楼宇链）**：Dept 档用户 List / Statistics / GetByID / dropdown-options 只返回 `building.org_id ∈ 可见集` 的机房；跨机构楼宇下的机房不可见
- [ ] **133-A3 dedicatedLine（source/dest 双链 OR 命中）**：构造「源机房在本机构、目的机房在跨机构」→ 本机构可见；构造「两端都在其它机构」→ fail-closed 空集；Create/Update 提交两端都越界的 `source_room_id`/`dest_room_id` → 403
- [ ] **133-A4 roomDevice（两级 JOIN）**：Dept 档用户只返回 `room → building.org_id ∈ 可见集` 的机房设备；任一级断链的设备不可见
- [ ] **133-A5 单条写越界 403**：四实体 Create / Update / Delete 各挑一条越界目标，浏览器 Network 断言 HTTP 403（统一错误文案，DS-04 / 132）
- [ ] **133-A6 批量按可见集裁剪**：Batch 请求混入越界 ID，断言只处理可见集内行、越界计数入 operlog、不整批 403
- [ ] **133-A7 超管零回归**：admin（All 档）对四实体读全量 / 写任意目标放行，行为与 130 前一致
- [ ] **133-A8 Self 档空集**：Self 档用户对四实体（无用户归属列）读返回空集、写 403
- [ ] **133-A9 异常枚举 fail-closed**：构造非 0 且 ∉ 1..5 的 data_scope 角色，读返回空 / 写 403
- [ ] **133-A10 归属链断链脏数据抽查**（ds05 残余风险）：测试库造「building 已软删但 dedicatedLine / roomDevice 仍引用」的脏数据，断言非超管看不到、系统不开旁路；记录抽查行数与结论
- [ ] **133-A11 两级 JOIN `EXPLAIN` 观察项**（非阻断）：真实 PG 跑一次 `EXPLAIN`，记录是否出现全表扫描 / 计划退化

## B. workstation-device 按工位可见集（人工复核）

- [ ] **133-B1 读**：Dept 档用户访问 `/ops/workstation-device/:id`（及 /ad /asset /physical /physical-diagnosis），断言目标工位 ∈ 可见集才返回；越界工位不返回设备
- [ ] **133-B2 写**：Create / Update / Delete 目标工位越界 → 403（单条写端点，批量裁定 §2「单条不豁免」）
- [ ] **133-B3 Self 档**：随工位 `user_id` 收窄，只返回本人绑定工位的设备关联
- [ ] **133-B4 无独立机构语义**：确认不存在 `device.org_id` 之类旁路判定（裁定 2）

## C. ds05 残余风险专项

- [ ] **133-C1 脏数据抽查**（对应 A10）：生产/预发库抽查 `ops_dedicated_lines` / `ops_room_devices` 中归属链指向软删楼宇或不存在机房的行数，登记数量与处置建议；**不改 fail-closed 语义**
- [ ] **133-C2 location-alias 别名文本复核**（对应 D2）：见 D 段

## D. 显式豁免面复核（只登记不改代码）

> 豁免面共 5 项：walls / doors / floor-plan-texts / rooms-photos / location-alias。
> 均为「已知残余面、接受风险」，**不以「间接可见所以安全」结案**。

- [ ] **133-D1 几何/附件面抽查**：以 Dept 档用户直接调用 walls / doors / floor-plan-texts 独立 `/list` + `/:id`、rooms/photos 独立 `GET /list/:roomId` + `GET /:id`，确认返回不含业务归属列的几何/展示/附件数据，记录「是否已出现含机构名称的展示文字/附件」抽查结论
- [ ] **133-D2 location-alias 豁免复核**（裁定 3）：以 Dept 档用户直接调用 `/ops/location-alias` 读端点确认该面未收口；抽查 `sys_dept_location_alias` 别名文本确认不含机构名称等敏感人工录入内容；若发现含敏感机构名 → 登记为「应升级为按关联楼宇过滤」的后续需求（本 phase 不改代码）；结论：通过 / 发现敏感项（附行号与内容摘要，不含完整敏感值）
- [ ] **133-D3 dept-mapping-template 已收口复核**（审查 P1-1）：以 Dept 档用户访问 `GET /dept-mapping-template`，确认部门树已按可见集过滤（Phase 131 交付），且全链路 `parent_path` 过滤后是否仍暴露上级部门名（登记为已知残余，不阻断）

## E. vdi 迁移 + 死实现删除（人工复核）

- [ ] **133-E1 vdi 五档浏览器复核**：用 All / Custom / Dept / DeptChild / Self 五个角色分别访问 `/vdi/vm/list`，断言可见 VM 集合与迁移前一致（尤其 Self 只看 `bound_user_id = 登录人`、非 All 档隐藏未绑定 VM、Custom 走角色-部门映射）
- [ ] **133-E2 超管零回归**：admin（0/unset → All）vdi 列表不过滤、未绑定 VM 仍可见
- [ ] **133-E3 异常枚举 fail-closed**：非 0 且 ∉ 1..5 的 data_scope 角色 vdi 列表返回空，不出现「放行全库」
- [ ] **133-E4 旧实现删除确认**：代码库检索确认 `pkg/middleware/permission.go` 已无 `ApplyDataScope` / `ApplyDataScopeFromContext` / `getChildDepts` / `isValidDataScopeField`，且 `DataScopePermission` / `getUserMaxDataScope` / `RequirePermissionsWithQuery` / `RequireStrictPermissions` / `injectScopeRequestContext` 全部保留；`permission_74_08_test.go` 中 `TestDataScopePermission` / `TestGetUserMaxDataScope` 仍存在且通过

## F. 前端 DeptSidebar 可见集联动（人工复核）

- [ ] **133-F1 两不同 scope 用户浏览器对照**：用两个不同 data_scope 的账号登录，打开任一含 `DeptSidebar` 的运维页面，断言左侧部门树渲染不同的部门集
- [ ] **133-F2 无前端硬编码确认**：确认前端没有按机构层级 / 名称 / 角色自行推断可见性的过滤规则；切换用户后部门树随后端返回变化
- [ ] **133-F3 All 档零回归**：admin 账号部门树渲染全量，与 Phase 133 前一致
- [ ] **133-F4 缓存/刷新**：切换账号后刷新页面，部门树不残留上一个账号的越权部门

## G. 提交与闸门复核

- [ ] **133-G1**：`go build ./...` 全绿
- [ ] **133-G2**：`go test -count=1 ./internal/... ./pkg/... ./cmd/...` 全绿（不带 coverprofile）
- [ ] **133-G3**：前端 vitest / prettier / `npm run lint:ci` 全绿（含 antd 6 弃用 prop 门禁）
- [ ] **133-G4**：`internal/api/strict_mount_count_test.go` 下限 16/188/164 通过
- [ ] **133-G5**：原子提交 commit hash 回填至 `133-SUMMARY.md` §4
- [ ] **133-G6**：未提交 `coverage.out`、未碰 `xingran-react-frontend/.env.production`、未 `git add -A`

## 人工项汇总（便于复制进验收记录 / 回写口径引用）

1. 前置波次确认（133-P1~P3）
2. 四实体读写 403 矩阵（A1-A9）+ 归属链断链脏数据抽查（A10/C1）+ EXPLAIN 观察（A11）
3. workstation-device 可见集读/写/Self（B1-B4）
4. location-alias 豁免复核（D2）+ 几何/附件面抽查（D1）+ dept-mapping-template 已收口复核（D3）
5. vdi 五档/超管/异常枚举浏览器复核（E1-E3）+ 死实现删除确认（E4）
6. DeptSidebar 两 scope 对照 / 无硬编码 / All 零回归 / 缓存刷新（F1-F4）
7. 闸门与 commit hash 回填（G1-G6）

> **前置授权 / 执行人 / 证据位**：133 全段需真实 PG + 跨角色浏览器（ Dept / Self / All / Custom 账号），
> 由现场运维同事 / PG 授权方执行；证据按 `.planning/notes/261108-uat-writeback-policy.md`
> 落盘并登记到 GSD 自动生成的产物或未来里程碑的 REQUIREMENTS / UAT 清单，未执行不得勾选。
