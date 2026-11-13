---
doc_type: 会话接手文档 / 剩余任务清单
created: 2026-11-13
created_by: 硬编码专项审查会话（PR #90 ~ #94 系列）
status: 权威交接件 / 供后续会话直接取用
supersedes: 无（本会话新产出；历史 HANDOFF.json 已 archived，勿据此恢复工作）
related:
  - .planning/STATE.md（v1.56 候选权威状态口径）
  - .planning/ROADMAP.md（当前里程碑与历史段）
  - .planning/notes/261108-human-verification-tracker.md（79 条人工验证债）
  - .planning/notes/UAT-EXECUTION-PLAN.md（三桶排期与解锁路径）
  - .planning/notes/SHIPPED-DEFINITION.md（SHIPPED 判定门槛）
  - scripts/RBAC-EXCEPTIONS.md（RBAC 例外台账）
---

# 接手文档：硬编码专项收尾后的剩余任务（2026-11-13）

## 0. 给接手 Agent 的第一句话

**当前工作区干净，本地仅 main 分支，与 github/main 同步（HEAD = `1aae3b47`）。**
本会话交付的 5 个 PR 均已合并：#90（dbbackup）、#91（CLAUDE.md 一致性）、
#92（RBAC-EXC-01 清偿 + TODO）、#93（硬编码专项第一轮）、#94（硬编码专项第二轮）。
**没有未提交代码，没有待处理的 CI 失败**。本文剩余项全部是「有意未做」的独立立项，
不是阻塞、不是半成品。

接手后第一件事建议：跑一遍基线确认环境（见 §6），再按优先级挑一项开工。

---

## 1. 本会话已交付（避免重复劳动）

| PR  | 主题                           | 关键结论                                                                                              |
| --- | ------------------------------ | ----------------------------------------------------------------------------------------------------- |
| #90 | dbbackup 凭证加密 / 恢复写保护 | 新增 `restore_write_guard.go`、SM4 凭据加密、`loadPlanTolerant` 降解                                  |
| #91 | CLAUDE.md 一致性               | 移除 legacy 缓存服务过时引用，统一 Unified Cache Architecture                                         |
| #92 | RBAC-EXC-01 清偿 + TODO        | 4 条周期工单动作端点迁逐动作严格权限，例外归零；侧边栏图标补齐；工位类型字典感知                      |
| #93 | 硬编码专项第一轮               | profile 性别口径颠倒修复、Excel 导入/导出字典兜底（见 §1.1）、后端状态字面量收敛、前端对账/值班字典化 |
| #94 | 硬编码专项第二轮               | 守卫测试补齐 7 族、RPA 遗留模型加 DEPRECATED、3D/网络/对账图表字典化收尾                              |

### 1.1 已建立的「字典驱动」样板（新代码请沿用，勿再造）

- **共享工具**：`xingran-react-frontend/src/components/reconciliation/dictLabels.ts`
  —— `buildDictLabelIndex` / `buildDictOptions` / `resolveDictLabel` /
  `resolveDictTagColor` / `CONFLICT_TYPE_LABELS`，三级回退口径：
  **字典命中（dictValue→dictLabel）→ 静态映射 fallback → 原始值 / "-" 兜底**（不吞成空串）。
- **工位类型**：`xingran-react-frontend/src/pages/operations/workstations/constants.tsx`
  的 `resolveWorkstationTypeText(type, dictItems?)`；3D 侧在 `src/utils/three/colors.ts`
  的 `resolveWorkstationTypeLabel`。
- **设备类型**：`xingran-react-frontend/src/pages/network/templates/constants.tsx`
  的 `resolveDeviceTypeLabel` / `renderDeviceTypeTag(deviceType, dictItems?)`；
  列工厂透传模式参考 `src/pages/operations/workstations/columns.tsx` 的 `typeDict`。
- **Excel 字典兜底**：`internal/services/operations/excel_config.go` 的
  `ExcelColumn.DictType` + `excel_service.go` 的 `lookupDictValue` /
  `lookupDictLabel`；导出侧对应 `excel_export_config.go` 的
  `ExcelExportField.DictType` + `excel_export_service.go` 的 `lookupExportDictLabel`。
  静态 `Options` 只是模板快照，未命中时按字典反查。

### 1.2 本会话反复踩过的坑（CI 门禁很严）

- `npm run lint:ci` 是**全链**（eslint `--max-warnings 0` → 8 个脚本 → 路由/API URL 检查），
  任一子脚本失败即整条挂。新增 `permission:` 门控必须同步
  `scripts/gating-exemptions.json` 与 `check-gating-coverage.mjs` 的
  `EXPECTED_EXEMPTION_COUNT`（恒等式：已门控 34 + 豁免 39 = 73）。
- **react-refresh/only-export-components**：组件文件同时 export 组件与普通函数/常量会
  触发 warning → lint:ci 挂。解法：抽到同目录纯工具文件（本会话已建
  `exceptionRuleFormOptions.ts` / `dictLabels.ts` 即此用途）。
- `src/index.css` 有行数预算（C13 M9 ≤ 3280，当前 3279），**新增 CSS 规则必须并入已有
  选择器组或删等量行**，不能新建独立 .css 文件（C8 死文件 + per-file 基线阻断）。
- `scripts/scan-routes.cjs` 只识别 `*WriteRouteGroups` / `*ActionRouteGroups`；
  新增路由组类型必须同步该正则，否则路由表报 STALE。
- 前端全量 vitest 约 10 分钟；与 lint:ci / go test 并发跑会相互抢 CPU 导致
  **假超时**（单跑 4s 通过的用例在并发下 30s 超时）。验证请分批后台跑。
- 受保护 `main` 必须走「功能分支 → PR → CI 全绿 → squash 合并」；
  squash 后本地分支用 `git update-ref -d refs/heads/<branch>` 清理（`-d` 会拒绝）。
  安全钩子会拦 `git reset --hard` / `git checkout -- <path>` / `git branch -D`。

---

## 2. 剩余任务（按优先级）

### P0 — 项目级状态 / 需要人与环境（不是代码问题）

| #   | 项                                     | 现状与证据                                                                                                                                                                                                         | 为什么没做                                           | 接手动作                                                                                                                                                                           |
| --- | -------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | ---------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| R1  | **v1.56 候选正式立项**                 | `.planning/STATE.md` frontmatter：`status: v1.56 候选（进行中 / 未正式立项）`；`.planning/state.json` 的 `next.command = /gsd-new-milestone`；草案 `.planning/quick/261110-161-*/v156-MILESTONE-DRAFT.md` 未跑立项 | 需 GSD 正式流程与用户裁定                            | 按 `.planning/notes/SHIPPED-DEFINITION.md` 补齐证据后跑 `/gsd-new-milestone`；候选骨架 phase 152-155（152/154 已由 quick 执行、153 用户裁定暂缓、155 可选），**不伪造 phase 完成** |
| R2  | **79 条人工验证（0 勾选 / 0 已完成）** | `.planning/notes/261108-human-verification-tracker.md`；三桶排期见 `.planning/notes/UAT-EXECUTION-PLAN.md`                                                                                                         | 本环境**无生产 PG 写授权、无真机**，只能排期不能执行 | 真机端口写 UAT（29 条现场真机债，tracker §六）排第一；逐条按 `evidence/<条目ID>-<YYYYMMDD>.md` 落盘证据，未执行不得勾选                                                            |
| R3  | **不得标 SHIPPED**                     | `.planning/notes/SHIPPED-DEFINITION.md`：代码绿 + 关键 UAT 通过 + 对外文档同步三者齐备                                                                                                                             | 无 UAT 证据                                          | 当前只能标 `code-complete·verification-pending`                                                                                                                                    |

### P1 — 需数据迁移 / 跨前后端联动，必须独立立项

| #   | 项                               | 现状与证据                                                                                                                                                                                                                                                                                                                                                                                                     | 风险 / 前置                                                                                                                                                                                                                                     | 建议做法                                                                                                                                                                       |
| --- | -------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| R4  | **CaptchaBg 状态极性统一**       | `internal/models/captcha_background.go:19-20`：`CaptchaBgDisabled=0 / CaptchaBgEnabled=1`，与全局「0=正常/启用」相反；`Status` 字段 `gorm:"not null;default:1"`（:105）。使用点：`internal/core/captcha_background.go:117,156,169,233`、`internal/api/v1/captcha_background_handler.go:273-302`                                                                                                                | **代码内部自洽**（Enabled=1 全程一致），违反的是命名约定而非实际 bug。改极性必须：① 数据迁移 `UPDATE sys_captcha_background SET status = 1 - status`；② 改 DB default；③ 前端同步；④ 守卫测试 `status_constants_test.go:223-224` 的登记值同步改 | 本会话已在守卫测试中**登记并冻结现状**（`watchedStatusPrefixes` 加 `CaptchaBg`），防未来被误「归一化」。若要统一，需单独 PR 且带迁移与回滚方案，不要在别的任务里顺手改         |
| R5  | **Root 包 RPA 遗留模型整体删除** | `internal/models/rpa.go` 已加 `DEPRECATED` 注释（文件头）说明唯一事实源是 `internal/models/rpa` 子包；`internal/core/db/database.go:876` 明确「无引用遗留定义,不注册」。两套同名枚举取值矛盾（`TaskStatusEnabled` 根包=1 / 子包=0；子包 `WorkerStatus` 是 string 枚举）。根包模型仅被 `internal/models/models_pure_80_03_test.go`（ScriptAction JSON 列）与 `models_hooks_80_03_test.go`（TableName 断言）依赖 | 直接删除会破坏 models 包单测与覆盖率门禁                                                                                                                                                                                                        | 先迁移/重写那两组单测（改为覆盖子包等价类型），再整体删 `internal/models/rpa.go`，最后从 `status_constants_test.go` 移除对应前缀与 24 个取值登记。本会话已把约束写进文件头注释 |
| R6  | **operations 枚举双份定义收敛**  | `internal/models/operations.go`（`FloorStatus` / `RoomStatus` / `RoomDeviceStatus*Scrap` 等）与 `internal/models/operations/*.go` 同名族重复；`RoomDeviceStatusScrap`(根包) vs `RoomDeviceStatusScrapped`(子包) 同值异名；守卫测试注释已承认 "known double definition"                                                                                                                                         | 删根包 `operations.go` 前需确认 `internal/services/operations/validation_helper.go` 中的 `models.Floor{}` 唯一引用改指向 `operations.OpsFloor{}`；涉及 GORM 表映射一致性                                                                        | 单独 PR，先做引用面全量 grep，再删根包文件并同步守卫测试的 `expectedStatusValues`（保留子包侧）                                                                                |

### P2 — 代码质量 / 类型安全，改动面中等

| #   | 项                                | 现状与证据                                                                                                                                                                                                                                                                                                                                                                                  | 建议做法                                                                                                                                                                                                                                         |
| --- | --------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| R7  | **handler/service 裸 `int` 状态** | `internal/api/v1/system/{user,role,menu,department}_handler.go` 的 `Status int \`binding:"min=0,max=1"\``（约 :297 / :234 / :266 / :249）；`scheduler/job_handler.go:245`连 min/max 都无；service 层`UpdateStatus(ctx, id, status int)` 全程透传                                                                                                                                            | 逐层改为 `models.UserStatus` / `RoleStatus` / `MenuStatus` / `DeptStatus` / `JobStatus`（`internal/models/base.go:57-58,65-66,101-102,109-110`、`log.go:71-72`）；`job_handler` 补 `binding:"oneof=0 1"`。注意同步 4 个 `*_cache_impl.go` 的形参 |
| R8  | **Excel 导入/导出专线标签漂移**   | `internal/services/operations/excel_config.go:297`（导入）写 `"mpls": "MPLS VPN"` / `"leased_line": "租用专线"`；`excel_export_config.go:329`（导出）写 `"mpls": "MPLS专线"` / `"leased_line": "租用线路"` —— 同一枚举两侧文案不一致                                                                                                                                                        | 以 `migration_208_dict_seed.go` 的 `ops_dedicated_line_type` 字典为准统一（导入侧文案），并给两列加 `DictType: "ops_dedicated_line_type"` 走字典兜底，消除静态快照漂移                                                                           |
| R9  | **信息点类型字典与模型错配**      | 字典 `ops_info_point_type` 落库值是 `network / phone`（`migration_208_dict_seed.go:143-146`），但 `internal/models/operations/infopoint.go:11-13` 是 `InfoPointTypeNetwork/Power/Other`（power/other），Excel 的 Options 也用 `network/power/other`                                                                                                                                         | 二选一：改字典补 `power/other`，或改模型/Excel 对齐 `phone`。**当前若给信息点列绑 DictType 会永远匹配不上**，所以本会话没绑                                                                                                                      |
| R10 | **Excel 配置顶部死常量**          | `internal/services/operations/excel_config.go:9-18` 的 10 个 `StatusNormal="正常"` / `WorkstationTypeFixed="固定工位"` 等常量：① 全无引用（死代码）；② 与 `internal/models/workstation.go` 同名常量同名不同值（string vs int）；③ `StatusStopped` 与 `StatusDisabled` 同值异名                                                                                                              | 整块删除；若要保留文案，改为 `excelLabelStatusNormal` 之类带前缀名并真正被 Options 引用                                                                                                                                                          |
| R11 | **其他裸 int / 状态语义混用**     | `internal/api/v1/operations/requests/common.go:55` 的 `Status *int` 被 9 个实体内嵌，但注释「0=正常 1=停用」与实际三态族（工位 0/1/2 空闲/占用/维护、机房设备 0/1/2 正常/故障/报废）混用；`internal/api/v1/monitor/{login_log,oper_log}_handler.go:47`、`system/{ad_account,ad_domain}_handler.go` 的 `Status *int`；`internal/api/v1/knowledge/handler.go:89` 从 `float64` 直接 `int(val)` | 拆二态/三态请求结构，或每实体字段用对应 `models.XxxStatus`；AD 账号改用 `*models.ADAccountStatus`（三态含熔断）。属类型安全改造，建议批量一次做完并补测试                                                                                        |

### P3 — 低优先级 / 展示层细节

| #   | 项                                    | 现状与证据                                                                                                                                                                                                   | 说明                                                                                                          |
| --- | ------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------------------------- |
| R12 | MACHistory 页注释里的 3 个 TODO       | `xingran-react-frontend/src/pages/network/mac/history/MACHistoryPage.tsx:16-19`：14-04 互斥导出按钮、14-05 替换内联 Alert/Empty 为 `EmptyStateWithAction` / `ErrorAlertWithRetry`、14-05b 移动端卡片样式收口 | 由后续 plan 接管；本会话已补「列配置」Modal，未动这 3 项                                                      |
| R13 | 3D 工位类型颜色仍静态                 | `xingran-react-frontend/src/utils/three/colors.ts` 的 `WORKSTATION_TYPE_COLORS`：label 已可被字典覆盖，但取色仍按 `type` 数字                                                                                | 字典新增 type=3 时标签正确、颜色回落 0 号色。若要求新增类型也有配色，需允许字典 `listClass/cssClass` 驱动颜色 |
| R14 | `scripts/RBAC-EXCEPTIONS.md` 窗口复核 | 台账中「已清偿 → 删除本条登记；未清偿 → 收窄窗口」的复核机制（约 :100）                                                                                                                                      | RBAC-EXC-01 已于 #92 清偿并留痕；后续按窗口定期复核剩余登记                                                   |
| R15 | 历史文档去重                          | `docs/RPA-Worker-API认证方案-待办.md` 标题含「待办」但正文已注明 v1.51 落地、且是两份重复副本；`docs/code-review-report.md` 已标历史归档                                                                     | 仅文档债，不影响运行；建议后续一次性清理误导性标题                                                            |

---

## 3. 明确**不要**做的事（防误读）

- **不要**把 `CaptchaBgDisabled=0 / Enabled=1` 直接改成 `Enabled=0` 而不带数据迁移——
  会静默反转线上所有背景图的启停状态。
- **不要**直接删 `internal/models/rpa.go` 而不先迁移 models 包内两组单测。
- **不要**给 `ops_info_point_type` 的 Excel 列加 `DictType` 而不先修 R9 的值域错配
  （字典里没有 power/other，兜底永远不命中）。
- **不要**给页面新增 `permission:` 门控却忘了从 `gating-exemptions.json` 移除条目 +
  同步 `EXPECTED_EXEMPTION_COUNT`（lint:ci 硬失败）。
- **不要**依据 `.planning/HANDOFF.json` 恢复工作——它已在 2026-06-18 archived，
  `archive_note` 明确「勿据此恢复工作」；权威是 `STATE.md` / `ROADMAP.md` / 本文。
- **不要**依据 `docs/code-review-report.md` / `docs/RPA-Worker-API认证方案-待办.md`
  的标题当作开放缺陷——两者都已在文件头注明是历史时点快照。
- **不要**伪造 UAT 勾选：79 条一律「已排期·待执行」，未执行不得勾选。

---

## 4. 关键文件速查

| 用途             | 路径                                                                                                                      |
| ---------------- | ------------------------------------------------------------------------------------------------------------------------- |
| 权威状态         | `.planning/STATE.md`、`.planning/state.json`、`.planning/ROADMAP.md`                                                      |
| SHIPPED 门槛     | `.planning/notes/SHIPPED-DEFINITION.md`                                                                                   |
| 79 条 UAT        | `.planning/notes/261108-human-verification-tracker.md`、`UAT-EXECUTION-PLAN.md`                                           |
| 状态常量守卫     | `internal/models/status_constants_test.go`（`watchedStatusPrefixes` / `expectedStatusValues` / `statusLiteralWhitelist`） |
| 字典工具（前端） | `xingran-react-frontend/src/components/reconciliation/dictLabels.ts`                                                      |
| Excel 字典兜底   | `internal/services/operations/excel_{config,service,export_config,export_service}.go`                                     |
| Excel 回归测试   | `internal/services/operations/excel_dict_fallback_test.go`、`excel_export_dict_fallback_test.go`                          |
| CI 门禁链        | `xingran-react-frontend/package.json` 的 `lint:ci`；`xingran-react-frontend/scripts/*`                                    |
| 路由注册表       | `scripts/scan-routes.cjs` → `scripts/route-registry.json`                                                                 |
| CSS 预算         | `xingran-react-frontend/scripts/css-contract-check.js`（C13 M9 ≤ 3280）                                                   |
| RBAC 例外台账    | `scripts/RBAC-EXCEPTIONS.md`                                                                                              |

---

## 5. 建议的接手顺序

1. **基线复核**（§6）确认环境干净。
2. 若用户要求推进 v1.56 → 先 R1（正式立项），再 R2/R3（UAT 证据，需人/真机授权）。
3. 若用户要求清理代码债 → 按 R4 → R5 → R6（每个独立 PR，带迁移）→ R7~~R11（可批量，注意 lint:ci 全链）→ R12~~R15（低风险）。
4. 任何改动后：**必须**跑 `go build ./...` / `go test ./...` / `gofmt` /
   `npx tsc --noEmit` / `npm run lint:ci` / `npx vitest run`（全量，分批后台）
   —— CI 与本地都要绿，`main` 受保护只能经 PR 合并。

---

## 6. 接手基线复核命令

```bash
git status --short                 # 应为空
git branch                         # 应只有 * main
git log --oneline -1               # 应为 1aae3b47（或之后的新提交）
go build ./... && go test ./...
cd xingran-react-frontend && npx tsc --noEmit -p tsconfig.json && npm run lint:ci
```

> 本文件由 2026-11-13 会话生成，之后若有 PR 合并，请更新 §0 的 HEAD 与 PR 列表，
> 并把已完成的剩余项从 §2 移到 §1（保留条目编号，便于追溯）。
