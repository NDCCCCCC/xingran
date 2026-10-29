---
phase: 86
title: 前端操作入口三态门控铺开 技术方案研究
date: 2026-10-29
researcher: gsd-phase-researcher（只读研究，未改业务代码、未 commit）
requirements: RBAC-02
confidence_marks: HIGH=读代码/脚本实证(附 文件:行号) / MEDIUM=推导 / LOW=需实机验证
---

# Phase 86 研究报告：前端操作入口三态门控铺开

## 现状复核

### 0. 页面口径先行声明（HIGH，执行前必须重对齐）

- **实测 `src/pages/**/index.tsx` = 78 个**（find 全量清单，含 `login/index.tsx` 与 `knowledge/articles/modals/index.tsx`）。ROADMAP 写 76（`ROADMAP.md:84/199`）。
- 差异来源：`login` 非业务页、`knowledge/articles/modals/index.tsx` 是 Modal 聚合而非路由页；扣除后 ≈76。**W0 必须用静态脚本定稿"业务页面清单"**（login / my-notices / profile / settings / dashboard 类是否计入，需要规划者裁决），否则 SC1 的 76/76 无法判真。
- ROADMAP 基线「14/76 已接入」（`ROADMAP.md:85`）与本扫**口径不一致（MEDIUM）**：
  - 引用 `useActionPermission` 或 `PERMISSION_CODES` 的页面文件仅 **3 个**（`pages/system/user/index.tsx`、`pages/network/devices/index.tsx`、`pages/network/devices/hooks/useDeviceRowActions.tsx` + `devices/constants.ts`）；
  - 另有 5 个 index.tsx 用**旧式** `useMenuStore` permissions / `hasPermission`（`asset/reconciliation/exceptions`、`asset/reconciliation/fix-suggestion`、`network/ports`、`operations/workstations`、`vdi/VirtualMachineList`）——已判权限但**未走三态 helper**，属 86 改造对象而非"已接入"。
  - "14" 可能按接线点/文件（含非 index 文件）计。执行时以重扫为准并在 STATE 里钉死口径。

### A. 页面盘点（按域分组，写入口密度 = index.tsx 内 handle*/Modal/confirmDelete 计数，MEDIUM 粗粒度）

**范式页（HIGH）**：`pages/system/user/index.tsx:44,50,349-355`——`import { confirmDelete, useActionPermission } from "@/components/shared"` + `const permGate = useActionPermission()` + `permGate.decide(PERMISSION_CODES.systemUser.edit)` 逐入口取决策。`network/devices` 走了更进一步 hooks 化（`hooks/useDeviceRowActions.tsx:33,105-106`）。

| 域 | 页面 | 典型写入口 | TS 权限码状态 | 复杂度 |
|---|---|---|---|---|
| system | user(✅范式) role menu dept post | 行操作/工具栏/Switch/批量/Excel | **有码**：systemUser/Role/Menu/Dept/Post（permissionCodes.ts:22-66） | 中 |
| system | dict notice config apikeys captcha-background settings×2 | 全 CRUD（dict 47 入口、apikeys 37） | **无码**（TS 清单缺失；85 也未拆） | 高 |
| system | manager-orgs manager-rules | CRUD | **有码**：opsManagerOrg/Rule（:208-217，F 型非零） | 低 |
| ops | buildings floors workstations info-points server-rooms room-devices dedicated-lines assets | 行操作+Excel import/export（11 页用 Excel 组件）+Switch+批量 | **有码**：ops* 8 模块（:73-138） | 高（入口最多） |
| ops | building-spaces(±3d) location-alias(路由挂载) | 图形编辑/floor-plan 保存 | 有码：opsWall/Door/FloorPlan（:145-157） | 中 |
| ops | rpa rpa/tasks rpa/executions rpa/workers | CRUD+执行控制 | **无码**（85 范围外域） | 中 |
| network | devices credentials templates backups discoveries command executions | 行操作+批量+restore/execute/diff | 大部分**有码**（networkDevice/Credential/Template/Command/Backup/Discovery :162-204）；executions 复用 command 码 | 高 |
| network | ports mac mac-history | ports 写操作（canWrite 局部 const，v1.24 REQUIREMENTS:46 实证） | ports **无 TS 块**；mac **无码** | 中 |
| ad-domain | configs groups users ous computers logs | 同步/启停/解锁/CRUD | configs/groups/users **有码**（opsADConfig/Group/User :218-232）；ous/computers/logs **无码** | 中 |
| asset | reconciliation +4 子页 | 异常处置/修复建议执行 | exceptions/fix-suggestion 有旧式判定；**TS 无独立码块**（可能复用 ops:asset） | 中 |
| duty | 5 页 | 排班/换班/节假日 CRUD+Excel | **无码**（85 明确范围外，85-RESEARCH:101） | 中 |
| workorder | 4 页 | 工单流转/审批 | **无码** | 中 |
| knowledge | 2 页 + modal | 文章 CRUD | **无码** | 低 |
| monitor | cache job logs server dashboard | 任务 CRUD/清缓存 | **无码** | 中 |
| vdi | VirtualMachineList(旧式判定) VDIServerConfig VirtualMachineDetail | VM 生命周期操作 | **无码** | 中 |
| 其他 | dashboard-system monitor/dashboard duty/my-duty my-notices profile settings login ad/SyncMonitor | 只读/个人页，0~2 写入口 | — | 极低（部分可标"无写入口"豁免） |

**关键缺口结论（HIGH）**：TS 清单（23+21 区块，`permissionCodes.ts`）覆盖 system 5 组 + ops 13 + network 6 + manager 2 + AD 3 ≈ **约 45 页有码可接**；**duty / workorder / knowledge / rpa / monitor / vdi / mac / ports / ad-domain(ous/computers/logs) / dict / notice / config / apikeys / captcha-background ≈ 20~25 页的写操作码不在 TS 清单**——与 85-RESEARCH §B' 的"无权限中间件模块不在 85 范围"一致（`85-RESEARCH.md:101,166`）。这些页面**86 范围外或降级处理**（无码可判 → 门控无从谈起），需规划者裁决：留 TODO 注释还是标 N/A。

### B. 共用组件门控现状与推荐（HIGH）

- **ActionButtons 已内建三态**：`components/shared/ActionButtons.tsx:13,44-49`——组件内部自调 `useActionPermission()`，逐按钮 `gate.decide(action.permission)`；不传 `permission` 恒可见（45 个存量调用点零破坏）。**行操作门控不需要每页再包组件，页面只需给每个 action 加 `permission: PERMISSION_CODES.xxx.yyy` 一个字段**。
- 使用面：`ActionButtons` 18 页、`confirmDelete` 28 页、`PageTable` 9 页、`FormModal` 3 页。
- **推荐：契约组件层已就位，86 只做"页面传参铺开"，不再改组件**（置信 HIGH）：
  1. PageTable 的操作列 columns 是**每页自定义**的（PageTable 只是表格外壳），无法在 PageTable 层统一门控；
  2. FormModal 的提交按钮**无业务权限码语义**（能否打开弹窗已由入口按钮门控）；
  3. ConfirmDelete 是调用方触发的确认器，同理。
  - 唯一组件层小改动：`ExcelImport` / `ExcelExport` / `BatchDeleteButton` 若要支持传 permission 内部门控，可加可选字段（照 PERM-01 先例，加法扩字段）；但**更省事的是页面层直接 `permGate.decide(...).visible && <ExcelImport/>` 条件渲染**——推荐后者，零契约变更。

## 实现方案建议

### B1. 三态基建当前实现（HIGH，勿重做）

- `components/shared/useActionPermission.ts:24-53`：`state = error|loading|ready`（由 menuStore 显式位 `permissionsLoaded`/`permissionsError` 派生，:82-86）；`allowed(perm)`：未传码→true，未就绪→false，ready→`permissions.includes`；`decide(perm)`：未就绪→`{visible:true, disabled:true, hint}`，ready 无码→`{visible:false}`，ready 有码→可用（:100-109）。
- 禁止 `permissions.length===0` 判未就绪（:12-15 明文）。
- 就绪位维护点：`store/menuStore`（fetchAll/fetchPermissions/setPermissions/clearMenus 四处，v1.24-REQUIREMENTS PERM-02）+ `router/DynamicRoutes.tsx:154-158` 同步置位。
- ActionButtons disabled 态渲染：`ActionButtons.tsx:66-91,99-120`（span pointerEvents:none + title/aria，或菜单项 title）。

### B2. 76 页铺开的最小改动模式（HIGH，推荐"每页 3 行"而非 PageTable 级 hook）

模板（照 `system/user/index.tsx:349-355` 抄）：

```tsx
import { PERMISSION_CODES } from "@/constants/permissionCodes";
import { useActionPermission } from "@/components/shared";
// ① 页面顶部（一行）：
const permGate = useActionPermission();
// ② 行操作/工具栏每个写入口（一行）：传 permission 字段 或 条件渲染
{ key: "edit", label: "编辑", onClick: ..., permission: PERMISSION_CODES.opsBuilding.edit }
{ permGate.decide(PERMISSION_CODES.opsBuilding.export).visible && <ExcelExport ... /> }
// ③ 独立 Switch/按钮（两行）：
const st = permGate.decide(PERMISSION_CODES.systemRole.edit);
<Switch disabled={st.disabled} ... /> // st.visible===false 时整个不渲染
```

- **不推荐**再抽 PageTable 级 hook 或配置式 per-route 码表（路由→码映射表会成为第二真相源，违背 PERM-06 单一真相源原则）；页面级显式传参与 v1.24 范式一致、review 面最小。
- 预估改动量：~45 页 × 平均 4-6 个入口 ≈ 200-270 处传参，全部是加法（MEDIUM）。

### B3. 窗口期交互推断——**确认成立**（HIGH）

- 85 窗口期（85-STATE §1）：system 4 组写码 + ops import/export 码 + network 新码在 87 播种前无 F 型授权来源 → 无码角色后端 403。
- 前端照码门控后：ready 且无码 → `decide` 返回 `visible:false`（useActionPermission.ts:106）→ 按钮不渲染。**这与窗口期后端 fail-closed 行为完全同向**（前端藏入口 + 后端 403 双保险），无需任何特殊处理或 feature flag。
- 注意点（LOW）：前端权限集来自 `POST /system/my-menus/permissions`（menuApi.ts:23-26，v1.24 REQUIREMENTS:47 实证），87 播种后用户需刷新页面/重新拉权限才能看到新入口——这属于 87 的验收口径，不是 86 的问题。

### C. 与 Phase 90 的文件边界（HIGH）

- Phase 90（Input 域交还 / STYLE-03）事实源：`index.css:877/912/5563`（圆角）+ `design-system/components/AntdThemeBridge.tsx` + `design-system/tokens/colors.ts`（grep AntdThemeBridge 命中清单）——**全部是 CSS/token 层文件**；T1-T3 圆角目标也是 `.css` 规则（ROADMAP:129），不碰页面 JSX/内联样式。
- 86 改动面：`src/pages/**/index.tsx`（+个别 hooks 文件）；**不改任何 `.css`、不改 `design-system/**`**。
- **交叠面 = 空**。唯一相邻面：`components/shared/`（86 若改 ExcelImport/BatchDeleteButton 加可选字段）与 90 无涉；86 在 ActionButtons disabled 态用的 `style={{pointerEvents,opacity}}` 已存在（ActionButtons.tsx:72），是 JS 内联样式不是 CSS 文件。
- 并行安全边界（双方自律即可，无冲突文件）：86 不碰 `index.css` / `src/**/*.css` / `design-system/**`；90 不碰 `src/pages/**`。**结论：两线可完全并行，无需文件锁**（置信 HIGH）。

## Validation Architecture（必需）

### V1. 组件测试（三路径，PERM-03 范式复用，HIGH）

- 范式：`pages/system/user/__tests__/permission.gate.test.tsx`（mock menuStore 三态：`permissionsLoaded=false` → disabled+在 DOM；`loaded+无码` → 不在 DOM；`loaded+有码` → 可点，7 用例）。`ActionButtons` 自身已有 `components/shared/__tests__` 契约测试。
- 76 页不可能全测，**抽样策略**：
  1. **每域 1 个代表页**写三路径测试（system/role、ops/buildings、network/devices、ad-domain/configs、manager-rules、vdi 列表 ≈ 6-8 页）——覆盖 decide 直调、ActionButtons 传参、条件渲染三种接法各至少 1 页；
  2. **入口形态矩阵**覆盖：行操作（ActionButtons）、工具栏按钮、Switch 状态切换、批量操作、Excel 导入导出——5 种形态 × 三路径；
  3. 其余页面靠 V2 静态扫描保证"接了"，不逐页写测试。

### V2. 静态覆盖率脚本（SC1/SC2 的"76/76 可判真"判定法，HIGH，推荐新增 node 脚本）

- 新增 `scripts/check-gating-coverage.mjs`（零新依赖，仿 check-permission-codes.mjs 风格）：
  - 扫 `src/pages/**/index.tsx`（+同目录 hooks），对**含写入口特征**（`ActionButtons`/`confirmDelete`/`handleDelete`/`handleCreate`/`<Switch`/`ExcelImport|Export`/`batchDelete`）的页面断言：**至少出现一处 `useActionPermission` 或 `permission:` 传参**，否则 exit 1 并列出页面清单；
  - 页面清单内置 **N/A 豁免数组**（只读页 login/dashboard/my-notices… + 无码域 duty/workorder/… 规划者裁决后的范围外页），豁免需带理由（仿 exemption-reasons 先例）；
  - 产物即覆盖率数字（如 47/47 有码页面 100%），挂进 `lint:ci` 链（`package.json:14`）。
- 先红后绿：W0 落脚本（当前必红）→ 每波结束跑绿。

### V3. 回归守卫

| 守卫 | 挂载点 |
|---|---|
| `check-permission-codes.mjs` exit 0（TS↔Go↔菜单三方） | `npm run lint` / lint:ci（ROADMAP:88） |
| v1.22 冻结契约测试（AppInput/AppSelect/ConfirmDelete/PageTable/FormModal） | `npm run test`（ROADMAP:87） |
| gating-coverage 脚本 | lint:ci（新增） |
| 既有 ActionButtons / useActionPermission 单测不回退 | `npm run test` |

### V4. E2E 抽查（LOW，可选）

选 3 页人工/E2E 抽查真实角色：admin（全可见）、只读角色（入口全不渲染）、权限接口失败 mock（disabled+title）——system/role + ops/workstations + network/backups（覆盖窗口期 fail-closed 三类码）。

## 波次建议

```
W0  基建核对 + 口径定稿：业务页面清单脚本化（76/77/78 对齐）；范围外（无码）页面豁免清单由规划者裁决；
    gating-coverage 脚本落地（先红）；不写页面。依赖：无（基建 100% 现成）。
W1  system 域有码页：role menu dept post manager-orgs manager-rules + user 复核（~7 页）
    —— 码全在 TS（含 85-02 新码，窗口期 fail-closed 无碍，见 B3），先行验证铺开范式。
W2  ops 域：buildings/floors/workstations/info-points/server-rooms/room-devices/dedicated-lines/assets
    + Excel 入口（~8-10 页）——入口密度最高，单独一波。
W3  network 域：devices/credentials/templates/backups/discoveries/command/executions（~8 页）。
W4  ad-domain configs/groups/users + vdi/asset（有旧式判定改三态）（~6-8 页）。
W5  收口：覆盖率脚本转绿、抽样组件测试补齐、lint/test 全绿、SC 口径落盘。
```

- **依赖**：基建（useActionPermission/ActionButtons/permissionCodes）全部 v1.24+85 已交付，**无前置开发依赖**，W0 后即可铺开。
- **串行约束**：W1-W4 各页互不相干可并行；但 `permissionCodes.ts` 本波**零改动**（码已全量声明），消除了多波并发改常量的冲突面。
- **范围外裁决项**（W0 必须定）：无码域（duty/workorder/knowledge/rpa/monitor/vdi 写操作/ports/mac/dict/notice/config/apikeys/captcha-background/ous/computers/logs）≈ 20-25 页——建议标 N/A 豁免进覆盖率脚本，等这些域后端拆分 phase 再回补（避免 86 范围爆炸，与 85-RESEARCH:166 同构决策）。

## 边界与不做什么

- 不做后端（85 已完成，RequireStrictPermissions/路由登记不碰）；不做 F 型播种与授予（87）；不碰 CSS/token/design-system（90）；不改 `useActionPermission`/`ActionButtons`/`PageTable`/`FormModal`/`ConfirmDelete` 契约语义（只消费）。
- 不新增任何依赖；不新增权限码（若铺开中发现缺码 → 记录并上报，回 85/87 口径处理，86 不自造字符串码）。
- 窗口期 fail-closed 页面**不做降级特判**（B3 已证行为同向）。
- `.planning` 外只写 `src/pages/**` + 新增 1 个 scripts 脚本 + 对应测试文件。

## Sources

- `ruoyi-react-frontend/src/components/shared/useActionPermission.ts:24-115`（三态判定）、`ActionButtons.tsx:13,24-32,44-120`（permission 字段 + 逐按钮门控）
- `ruoyi-react-frontend/src/pages/system/user/index.tsx:44,50,349-355`（范式页）、`pages/system/user/__tests__/permission.gate.test.tsx`（三路径 7 用例）、`pages/network/devices/hooks/useDeviceRowActions.tsx:33,105-106`（hooks 化范式）
- `ruoyi-react-frontend/src/constants/permissionCodes.ts:21-233`（23+21 区块全量码；缺码域判定依据）
- `ruoyi-react-frontend/scripts/check-permission-codes.mjs:57,145,426-451`（PENDING_SEED_WHITELIST 79 项 `code:` 条目，85-STATE 记 78——执行时以文件为准复核一次）
- `ruoyi-react-frontend/src/router/DynamicRoutes.tsx:154-158`（permissionsLoaded 同步置位）
- `.planning/ROADMAP.md:79-89,194-203`（Phase 86/87 SC）、`.planning/REQUIREMENTS.md:29-34,124-134`（RBAC-02 + 映射）
- `.planning/phases/85-backend-rbac/85-STATE.md:5-15`（窗口期 fail-closed 登记）、`85-RESEARCH.md:98-102,166`（范围外域先例）
- `.planning/milestones/v1.24-REQUIREMENTS.md:43-54`（PERM-01~06 设计依据与完成记录）
- Phase 90 边界依据：`.planning/ROADMAP.md:56-57,127-131`（M 指标全在 index.css/CSS 层）+ grep AntdThemeBridge 命中文件清单（design-system/**）
- 页面盘点：find `src/pages/**/index.tsx`（78 个）+ grep 写入口/组件使用计数（本会话 2026-10-29 快照）
