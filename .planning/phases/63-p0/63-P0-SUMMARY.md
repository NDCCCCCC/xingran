# Phase 63「P0 止血 — 数据可达与安全」执行 SUMMARY

- 执行人：gsd-executor（glm-5.3-flash / DeepSeek Harness）
- 日期：2026-10-24
- PLAN：`63-P0-PLAN.md`（T0–T8 全部执行）；契约：`63-UI-SPEC.md` §1–§7
- 分支：main（branching_strategy: none），每任务独立 commit

## 逐任务完成状态

| 任务 | 状态 | Commit | 要点 |
|------|------|--------|------|
| T0 | ✅ | —（只读） | 静态复核通过：`internal/api/v1/system/ad_domain_handler.go:421,429` setDefaultPagination → `pkg/query/pagination.go:51` NormalizePagination → PageSize 钳制上限 100（`internal/constants/pagination.go:20` MaxPageSize=100）；`internal/services/addomain/user.go:45-79` GetList 无额外截断，Count+Offset/Limit 返回真实 total。**运行时冒烟未做**：dev 后端未运行（curl 9000 无响应），以静态复核代替（见偏离 D1）。T3 的 pageSize=50 在上限内安全。 |
| T1 | ✅ | 8b4daf31 | `src/hooks/usePagination.ts:122` showTotal 改 `(t) => (t === 0 ? "" : \`共 ${t} 条 / ${Math.ceil(t / pageSize)} 页\`)`，51 页消费点自动生效。grep Math.max 0 命中；hook 测试 6/6。 |
| T2 | ✅ | 689a18d3 | `backups/index.tsx` / `discoveries/index.tsx` 删手动 setCurrent/setPageSize 双写，保留 `searchParams` 的 `pagination.current ?? 1`；hook 内部 total 回填 `setTotal(hookTotal)`（useEffect）。grep `_setTotal` 0 命中；network 测试 248 通过。 |
| T3 | ✅ | 5e2d8233 | `ad-domain/ous/index.tsx` OU 用户表改服务端分页：`usePagination({ pageSize: 50 })`，fetchUsers 以 current/pageSize 拉取并回填 total；handleOUSelect 重置 `userSetCurrent(1)`。grep `pageSize: 100` 0 命中；ad-domain 测试 49 通过。**偏离 D2**：hook 的 options.pageSize 为硬约束（冻结 pageSize），故 `showSizeChanger: false`，用户不可切换每页条数（可切换归 hook 增强，见上报 R4）。 |
| T4 | ✅ | f11e1684 | 11 处批删统一在 handler 内包 `Modal.confirm`（title「确认批量删除」/ content「将删除选中的 {N} 项，删除后不可恢复，是否继续？」/ okText「删除」/ cancelText「取消」/ danger；成功文案统一「已删除」）。覆盖：system user/config/post/dict×2/role/notice + network devices/credentials/ports/mac/templates（模板页闸门下沉 hook，调用点无法绕过）。menu 为对照组（useMenuActions 已有确认，未动）。测试同步：role/dict 批删用例改 `vi.spyOn(Modal,"confirm")` 触发 onOk。 |
| T5 | ✅ | a9fdc12e | ① `assets/index.tsx` 删除 `handleEdit`（「编辑功能待实现」假入口），`AssetRow` onEdit 改条件渲染，留 TODO(P67)；② assets「下载模板」接真 `excelApi.downloadTemplate("asset")`；③ `ReconciliationDrawer.tsx` 只留「冲突摘要」Tab（timeline/exception 永久空态隐藏，activeKey 钳 summary，留 TODO(P67)）。 |
| T6 | ✅ | 0cc526bc | 16 处 `confirmLoading` 与 async onOk 双写全删（asset/reconciliation 3、network/devices 1、operations/buildings 1、system captcha UploadModal+settings 5、manager-orgs/rules 2、vdi 4），防重统一走 onOk Promise（antd 自动 loading）。grep confirmLoading 0 命中。 |
| T7 | ✅ | 7664ff1f | fixed 列与 scroll.x 成对，共补 8 处（PLAN 定位 5 处 + 排查出的同类 3 处）：user、config、dict×2、captcha-background、monitor/server、duty/HolidayManagement、ad-domain/configs。均 `scroll={{ x: "max-content" }}`，未删任何 fixed 声明。 |
| T8 | ✅ | df0bb094 | `63-UI-SPEC.md` Checker Sign-Off 按块替换为标准 7 维（Layout & Structure / Typography / Color / Spacing / Copywriting / Interaction States / Accessibility）+ `Approval: approved`。 |

## 偏离记录

- **D1（T0）**：运行时冒烟未执行（dev 后端未起）。以静态代码复核代替：分页链路（handler→NormalizePagination→service→response.Page）证据完整，PageSize 上限 100。风险低（T3 pageSize=50）。
- **D2（T3）**：`usePagination({ pageSize })` 是硬约束，运行时不可变，故 OU 用户表禁用 size changer 而非「默认 50 可切换」。
- **D3（T4）**：PLAN 验证 grep `onClick={handleBatchDelete}` 期望 0 命中不成立——采用「在既有 handler 内包 confirm」实现而非换 `BatchDeleteButton` 组件，8 处 onClick 保留，二次确认闸门在 handler 内部（UI-SPEC §5 覆盖范围与文案均满足）。grep 期望的口径差记上报 R5。
- **D4（T7）**：PLAN 定位 5 处 + 「第 6 处 network/monitor 侧同类命中」，实际排查发现同类命中 3 处（monitor/server、duty/HolidayManagement、ad-domain/configs），按 PLAN「多于 1 处一并修」全部修复，共 8 处。
- **D5（全局收口）**：`npm run lint` 与 `npm run test` 无法全绿——存量问题与本次改动无关（见下）。

## Verification 结果（收口总闸）

| 项 | 结果 |
|----|------|
| `npm run type-check` | ✅ 0 错误 |
| `npm run lint` | ⚠️ 125 errors / 1529 problems——errors 数与改动前基线持平（125），全部为存量（no-explicit-any 等）；本次触碰文件/行 0 新增 error |
| `npm run test` | ⚠️ 3835 passed / 8 failed：7 个存量失败（`src/lib/apiFactory.invariants.test.ts` 3 + `src/pages/system/user/__tests__` 4，stash 验证与本次改动无关）；1 个 flake（`asset/reconciliation/dashboard/index.test.tsx`「导出为函数组件」全量跑偶发超时，单跑通过，已复跑验证） |
| `npm run build` | ✅ built in 1m20s |
| grep `Math.max` usePagination.ts | ✅ 0 命中 |
| grep `_setTotal` src/pages/network | ✅ 0 命中 |
| grep `pageSize: 100` ad-domain/ous | ✅ 0 命中 |
| grep `待实现` src/pages/operations | ✅ 0 命中 |
| grep confirmLoading（T6 加验） | ✅ 0 命中 |

## 上报裁决项（范围外发现，未擅改）

1. **operations 之外残留「待实现」假入口**：`src/components/dashboard/layout/LayoutToolbar.tsx:130,136`（Widget 选择器 / 仪表盘设置）、`src/components/shared/FloorPlanEditor.tsx:352,355`（删除 / 复制）——同样违反 STOP-06 精神，但在 PLAN/UI-SPEC 处置清单（仅 3 处）之外，建议归 Phase 64/67 处置。
2. **T4 验证口径**：`onClick={handleBatchDelete}` 模式仍存在 8 处（确认闸门已入 handler）。若 planner 期望的验收是「统一换 BatchDeleteButton 组件」，需 Phase 65/66 契约组件落地时重审。
3. **`src/pages/duty/holidays/columns.tsx:89`** 存在 `fixed: "right"` 但该 columns 导出（getHolidayColumns）无页面消费（仅测试引用）——死代码，未补 scroll；建议清理归后续 phase。
4. **usePagination pageSize 硬约束**：如需「默认 50 可切换」语义，需 hook 增强（options.pageSize 改初始值），超出本 phase 零新封装约束。
5. **存量质量债（阻断「全绿」闸）**：lint 125 errors（no-explicit-any 等）+ 测试 7 个存量失败（apiFactory invariants 3、user render 4）——建议单独立任务清理，否则后续 phase 收口闸持续无法达成。
6. **工作区脏文件（非本次产物）**：执行期间工作区存在未提交改动 `.planning/state.json`、`internal/services/system/asset_columns_schema.json`、未跟踪 `.planning/notes/frontend-audit/`——非 executor 产生，未纳入任何 commit，请 lead 处置。
7. **UI-SPEC §5 覆盖口径**：契约文字为「system 6 处 + network 5 处」，T4 实际触碰 11 个文件实现点（与之一致）；menu/useMenuActions 已有确认，作为对照组未改，与契约「重叠页去重」语义一致，提请确认。

## 手工冒烟建议（留给 lead / QA）

按 PLAN「手工冒烟总清单」逐 STOP：任一列表页看「共 N 条 / M 页」、backups/discoveries 空态分页、OU 用户表翻到第 3 页、任一批删双击只发一次请求、批删弹确认框、assets 行内无编辑按钮、「下载模板」真实下载、对账抽屉仅摘要 Tab、宽表横向滚动操作列贴右。

## 运行时冒烟验收（2026-10-24，Lead agent-browser 只读 + 用户人工复核）

环境：本地前端 4000 / 后端 9000，账号 chenchao-076（AD 域账号，system 域菜单不可见）。冒烟严格只读（批量删除确认框弹出后立即取消，零写入）。

| STOP | 页面 | 结果 | 证据 |
| --- | --- | --- | --- |
| 01 分页总条数 | backups / assets / users | ✅ | 「共 171 条 / 4 页」「共 6641 条 / 133 页」「共 4463 条 / 90 页」 |
| 02 数据可达 | network/backups | ✅ | 19 行真实数据，翻第 2 页正常（修复前 total 恒 0 只能看第 1 页） |
| 02 空态 | network/discoveries | ✅ | 真实无数据时「暂无数据」、不渲染分页器（total=0 契约） |
| 03 OU 服务端分页 | ad-domain/ous | ✅ | 左树 28 节点；选「武汉中心支公司」→「共 534 条 / 11 页」每页 50 条服务端分页，翻第 2 页生效（修复前固定拉 100 条、第 3 页起不可见） |
| 05 批量删除确认 | assets/list | ✅ | 勾选 1 条 →「确定要删除选中的 1 条资产吗？」确认框弹出（未执行） |
| 06 假入口 | assets/list | ✅ | 假「编辑」按钮已消失；「下载模板」真实请求 `/api/v1/ops/asset/template` |
| 04 弹窗防重 | 用户人工 | ✅ pass | 提交期间确定 loading、取消禁用，双击无重复写入 |
| 07 fixed 列 | 用户人工 | ✅ pass | 横向滚动时操作列贴右固定 |

**结论：7/7 项 STOP 全部运行时验收通过，Phase 63 完整收官。**

## Verifier B1 修复（Phase 63 追加执行，2026-10-24）

**缺口**：STOP-04 统一采用方案①（onOk 返回 Promise，antd 自动 loading ok 按钮）时，误认为已覆盖取消按钮；实际 UI-SPEC §4 契约要求「提交期间取消按钮同样禁用」（`cancelButtonProps={{ disabled: submitting }}`），方案①不覆盖，提交中点取消存在半写风险。修复前 `cancelButtonProps` 在 src/pages/ 0 命中（B1 blocking）。

**修复方式**（最小侵入，零新组件；复用已有 loading state 优先，无则就地加 state 并在 onOk 的 finally 复位）：

1. **业务 `<Modal>` / BaseEditModal 调用点**（100% 覆盖 T5 清单 + 0cc526bc 13 文件内全部异步提交弹窗，共 45+ 处）：
   - 已有 loading state 直接复用（仅加 cancelButtonProps）：manager-orgs、manager-rules（saving）、api-config/email-config（submitting/testing）、settings/captcha-background（uploading）、network/devices（creating 不覆盖手动新增，新增共享 submitting）、fix-suggestion rejectModal、reconciliation exceptions resolveModal、RollbackModal（submitting prop）等。
   - 新增 state + onOk wrapper（`setSubmitting(true); try { await handler(); } finally { setSubmitting(false); }`）：operations 五列表页（buildings/dedicated-lines/room-devices/server-rooms/info-points）、floors/FloorModal、workstations/EditModal、rpa EditModal、ad-domain 各弹窗、duty 各弹窗（useScheduleModals hook 内共享 submitting）、workorder 三页、system user/role/menu/config/dict/apikeys/post/dept、captcha-background（新增 updating）、network/devices 两弹窗、vdi 两页四弹窗、notice/NoticeForm（自定义 footer：取消 Button disabled + 提交 Button loading）。
   - 封装组件：`BaseEditModal`（src/components/modal/BaseEditModal.tsx）新增 `cancelButtonDisabled` prop 透传（post、workstations 调用点使用）；FloorModal/DeptEditModal/CaptchaEditModal 等内部 Modal 就地加。
2. **函数式 Modal.confirm**（hard rule 3）：0cc526bc 13 文件内全部 10 处改为 `const confirmInstance = Modal.confirm({ onOk: async () => { confirmInstance.update({ cancelButtonProps: { disabled: true } }); try { ... } finally { confirmInstance.update({ cancelButtonProps: { disabled: false } }); } } })`（manager-orgs×2、manager-rules、settings 三件、operations/buildings×2、network/devices×2）。

**豁免（只读/无默认取消按钮，无需禁用）**：ad-domain computers 两详情弹窗、users 移动/OU 弹窗、ad-domain ous/groups 用户组弹窗 footer=null、building-spaces/BuildingModal footer=null、rpa AIScriptEditor（自定义 footer，生成 loading 期间取消按钮 disabled={loading}）、network/devices 详情弹窗、rpa ExecutionDetailModal 只读详情。

**已知遗留（上报 planner 裁决）**：T5 清单文件内其余函数式 `Modal.confirm`（约 45 处，几乎全部为单条/批量删除确认，见 duty/schedules、duty/management、system/dict hooks、system/role hooks、system/menu hooks、system/notice、workorder、operations 各 columns 等）未做实例 update 禁用——删除类 confirm 同样存在提交中点取消的窗口，建议后续统一抽 helper（如 `withConfirmSubmitting`）一次性覆盖；本修复优先保证 B1 blocking 指向的业务表单弹窗 100% 达标。

**验证**：
| 检查 | 结果 |
| --- | --- |
| `npm run type-check` | ✅ 0 错误 |
| 抽查 buildings handleSave / captcha-background handleUpdate wrapper | ✅ finally 置回 false |
| 抽查 manager-orgs 两处 confirm | ✅ update 禁用 + finally 恢复 |
| cancelButtonProps 命中 | ✅ 业务异步提交弹窗 100% 覆盖 |
