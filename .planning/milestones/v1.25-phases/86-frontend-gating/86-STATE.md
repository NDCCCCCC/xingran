# 86-STATE.md — Phase 86 前端门控铺开（Plan 00 / W0 口径定稿，2026-10-29）

> 本文件是 **Phase 86 的口径唯一落盘处**。W1~W5 派发时以本文件的数字与清单为准；
> 任何与本文件冲突的数字（含 ROADMAP「14/76」「78 页」）一律以本文件为准，需改动先改本文件。

---

## 一、口径定稿（四组数字 + 依据）

| # | 口径项 | **裁定值** | 依据（可复跑） |
|---|---|---|---|
| 1 | 业务页面总数 | **76** | `src/pages` 下任意层级 `index.tsx` 递归计数（`check-gating-coverage.mjs` 实测 = 76）。86-RESEARCH 报的 **78** 系把重复目录计入，**作废** |
| 2 | 已接入三态门控的页面 | **2** | `src/pages/system/user/index.tsx` + `src/pages/network/devices/index.tsx`（脚本实测「已门控页面：2」）。86-RESEARCH 的「3 个文件」含 `components/shared/ActionButtons.tsx`（组件）与 `pages/network/devices/hooks/useDeviceRowActions.tsx`（hook）—— **这两个是基建，不是页面** |
| 3 | ROADMAP 基线「14/76 已接入」 | **作废**，实际 **2/76** | 14 系按接线点/文件计（含非 `index.tsx` 文件），非页面口径 |
| 4 | 有码待铺开页面 | **25**（system 7 / ops 8 / network 7 / ad-domain 3） | `src/constants/permissionCodes.ts:21-233` 的 28 个模块键逐模块映射到页面；`ops:walls` / `ops:doors` / `ops:floor-plan` 三个码在页面侧无独立写入口（仅 floors 页的平面图保存消费 `ops:floor-plan:edit`） |
| 5 | N/A 豁免页面 | **51** | 26 `no-permission-code` + 12 `read-only` + 8 `non-route-page` + 5 `legacy-two-source`（见第四节的分类微调说明） |
| 6 | 窗口期行为 | **不特殊处理** | ready 无码 → `decide` 返回 `visible:false` → 不渲染，与 87 播种前后端 fail-closed 同向（`useActionPermission.ts:100-109`） |
| 7 | 与 Phase 90 交叠面 | **空** | 86 只动 `src/pages/**` + 1 个 scripts 脚本；90 只动 `src/index.css` + `src/design-system/**` |
| 8 | `PENDING_SEED_WHITELIST` | **79 条 `code:` 条目** | `scripts/check-permission-codes.mjs:57` 起（85-STATE 记 78，以文件为准）；**86 不改该文件** |

**恒等式**：`25 应门控 + 51 豁免 = 76 页面`；`已门控 2 + 未覆盖 23 = 应门控 25`。

---

## 二、76 页页面清单（分域）

| 域 | 页数 | 页面 |
|---|--:|---|
| system | 14 | apikeys / captcha-background / config / dept / dict / manager-orgs / manager-rules / menu / notice / post / role / settings / settings-page / user |
| operations | 14 | assets / building-spaces / building-spaces-3d / buildings / dedicated-lines / floors / info-points / room-devices / rpa / rpa-executions / rpa-tasks / rpa-workers / server-rooms / workstations |
| network | 10 | backups / command / credentials / devices / discoveries / executions / mac / mac-history / ports / templates |
| duty | 6 | config / holidays / management / my-duty / pools / schedules |
| ad-domain | 6 | computers / configs / groups / logs / ous / users |
| monitor | 5 | cache / dashboard / job / logs / server |
| asset | 5 | reconciliation / reconciliation-dashboard / reconciliation-exception-rules / reconciliation-exceptions / reconciliation-fix-suggestion |
| 根级（个人/非路由） | 5 | dashboard-system / login / my-notices / profile / settings |
| workorder | 4 | categories / orders / periodic-templates / statistics |
| knowledge | 3 | articles / articles-modals / view |
| vdi | 3 | VDIServerConfig / VirtualMachineDetail / VirtualMachineList |
| ad（旧目录） | 1 | SyncMonitor |
| **合计** | **76** | 与脚本 `EXPECTED_PAGE_COUNT` 断言一致 |

> `src/pages/ad/` 与 `src/pages/ad-domain/` 是**两套目录**，都按 `index.tsx` 计为页面（PLAN 判据表明确要求）。

---

## 三、25 页有码待铺开清单（分波次归属）

| 波次 | PLAN | 域 | 页 | 权限码组 |
|---|---|---|--:|---|
| **W1** | 86-01 | system | 7 | `systemUser`（user 复核）/ `systemRole` / `systemMenu` / `systemDept` / `systemPost` / `opsManagerOrg` / `opsManagerRule` |
| **W2** | 86-02 | operations | 8 | `opsBuilding` / `opsFloor` / `opsWorkstation` / `opsInfoPoint` / `opsServerRoom` / `opsDedicatedLine` / `opsRoomDevice` / `opsAsset`（+ `opsLocationAlias` 抽屉迁移） |
| **W3** | 86-03 | network | 7 | `networkDevice`（devices 复核扩面）/ `networkCredential` / `networkTemplate` / `networkBackup` / `networkDiscovery` / `networkCommand` |
| **W4** | 86-04 | ad-domain | 3 | `opsADConfig`（configs，含 AccountPoolTab / ReconciliationTab）/ `opsADGroup` / `opsADUser` |
| **W5** | 86-05 | —— | 0 | 收口：豁免清单定稿 + 覆盖率 100% 断言 + 挂 `lint:ci` + 8 页抽样组件测试 |

具体页面（25）：

- **system 7**：`system/user`（已门控，需复核）、`system/role`、`system/menu`、`system/dept`、`system/post`、`system/manager-orgs`、`system/manager-rules`
- **operations 8**：`operations/buildings`、`operations/floors`、`operations/workstations`、`operations/info-points`、`operations/server-rooms`、`operations/room-devices`、`operations/dedicated-lines`、`operations/assets`
- **network 7**：`network/devices`（已门控，需扩面）、`network/credentials`、`network/templates`、`network/backups`、`network/discoveries`、`network/command`、`network/executions`
- **ad-domain 3**：`ad-domain/configs`、`ad-domain/groups`、`ad-domain/users`

---

## 四、51 页 N/A 豁免清单（分类）

落盘位置：`xingran-react-frontend/scripts/gating-exemptions.json`（结构 `{ "exemptions": [ { page, category, reason, evidence } ] }`，每条 **reason + evidence 必填非空**）。

| category | 页数 | 含义 | 页面 |
|---|--:|---|---|
| `no-permission-code` | **26** | 有写入口，但前端单一真相源（`permissionCodes.ts`）无对应码区块 → **不自造码**，等后端拆分 phase 回补 | system/{dict, notice, config, apikeys, captcha-background}、duty/{config, holidays, pools, schedules, **management**}、workorder/{categories, orders, periodic/templates}、knowledge/articles、operations/rpa/{tasks, executions}、monitor/{cache, job, logs, server}、vdi/VDIServerConfig、network/mac、ad-domain/{ous, computers}、ad/SyncMonitor、asset/reconciliation/exception-rules |
| `legacy-two-source` | **5** | 页面内存在**旧式权限码直判**（`useMenuStore().permissions`），构成第二真相源，违反 PERM-06 → 86 **不改其判定逻辑**，待后端拆分 phase 统一收口 | network/ports（`network:port:write`）、network/mac/history（`network:mac:export`）、vdi/VirtualMachineList（`vdi:vm:*` × 7）、asset/reconciliation/exceptions（`asset:reconciliation:resolve`）、asset/reconciliation/fix-suggestion（`asset:reconciliation:fix:{accept,reject,rollback}`） |
| `read-only` | **12** | 无写入口（只读列表 / 统计 / 只读视图）；写入口特征集零命中 + 人工复核无写向按钮 | workorder/statistics、knowledge/view、duty/my-duty、operations/rpa、operations/rpa/workers、vdi/VirtualMachineDetail、ad-domain/logs、asset/reconciliation、asset/reconciliation/dashboard、operations/building-spaces、operations/building-spaces-3d、monitor/dashboard |
| `non-route-page` | **8** | 非路由页 / 个人页 / Modal 聚合 | login、profile、settings、system/settings、system/settings-page、my-notices、dashboard-system、knowledge/articles/modals |

### 与 86-00-PLAN 豁免分类表的差异（逐条）

**页数不变（51），仅分类按实测写入口/旧式码事实微调**（PLAN 允许「W5 按脚本实跑结果定稿微调」，此处提前按 W0 实跑事实落定）：

| 页面 | PLAN 分类 | 实测分类 | 差异原因（实测证据） |
|---|---|---|---|
| `duty/management` | read-only | **no-permission-code** | 有写入口：命中 `handleXxx` / `confirmDelete` / `onBatchDelete` / `batchDelete=`（`duty/management/components/DutyConfig.tsx` 值班配置 CRUD + `GenerateScheduleModal.tsx` 生成排班） |
| `operations/rpa/workers` | no-permission-code | **read-only** | 无写入口：写入口特征零命中，唯一操作按钮是「重置」（`index.tsx:282` `handleReset` 重置搜索条件，非业务写） |
| `monitor/dashboard` | no-permission-code | **read-only** | 无写入口：写入口特征零命中、人工复核无写向按钮（纯图表总览） |
| `network/ports` | no-permission-code | **legacy-two-source** | 页面内含旧式码字面量 `network:port:write`（`index.tsx:130-132`）+ `useMenuStore().permissions` 直判 |
| `network/mac/history` | read-only | **legacy-two-source** | 页面内含旧式码字面量 `network:mac:export`（`MACHistoryPage.tsx:117-119`） |
| `vdi/VirtualMachineList` | no-permission-code | **legacy-two-source** | 页面内含旧式码字面量 `vdi:vm:*` × 7（`index.tsx:46,52,109,929-935`） |
| `asset/reconciliation/exceptions` | no-permission-code | **legacy-two-source** | 页面内含旧式码字面量 `asset:reconciliation:resolve`（`index.tsx:85-86`） |
| `asset/reconciliation/fix-suggestion` | no-permission-code | **legacy-two-source** | 页面内含旧式码字面量 `asset:reconciliation:fix:{accept,reject,rollback}`（`index.tsx:64-67`） |

**应门控集合未受任何影响（仍为 25 页）** —— 分类变化只影响统计口径与豁免理由表述，不改变「哪 25 页要铺开门控」。

### 旧式直判第 6 处（`ad-domain/configs/ReconciliationTab.tsx`）的处置

PLAN 的「旧式判定页处置口径」列了 6 处，其中 `ad-domain/configs/ReconciliationTab.tsx`（旧式码 `ops:ad:config:reconciliation:list`）**落在应门控页 `ad-domain/configs` 内部**，因此**不能**进豁免清单（进了就会把该页从 25 页分母里挖掉）。处置：

- 该页**留在 W4 的应门控清单**内，由 86-04 在铺开 `opsADConfig` 门控时一并收口；
- 其旧式码同样不在 `permissionCodes.ts` → **不自造码**，以「读到 `ops:ad:config:reconciliation:list` 不在清单 → 按无权限处理」的方式与 87 fail-closed 同向；
- 该事实已登记在第六节「缺码待回补」。

---

## 五、扫描脚本先红证据（判定器有效性）

产物：`xingran-react-frontend/scripts/check-gating-coverage.mjs`（**零依赖**，只用 `node:fs` / `node:path` / `node:url`）。

### 首跑输出（W0 当前树，未铺开 → **预期红**）

```
=== Phase 86 W0 门控覆盖扫描 ===

页面清单：76 页（断言 EXPECTED_PAGE_COUNT = 76）
豁免清单：51 条（no-permission-code 26 / read-only 12 / non-route-page 8 / legacy-two-source 5）
应门控页面：25（= 76 页面 − 51 豁免）
已门控页面：2（src/pages/network/devices/index.tsx, src/pages/system/user/index.tsx）
已门控 ∩ 豁免：0
未覆盖页面：23

=== 未覆盖页面清单（23 页，需铺开三态门控）===
  ✗ src/pages/ad-domain/configs/index.tsx
      归属文件：…
      写入口特征：ActionButtons, confirmDelete, handleXxx
      Switch 命中：src/pages/ad-domain/configs/index.tsx:437 onChange=(无 —— 表单字段) [form-field]
  …（其余 22 页）
  ✗ src/pages/system/role/index.tsx
      写入口特征：ActionButtons, confirmDelete, handleXxx

⚠ 豁免清单非空（51 条）—— 仅限 Phase 86 铺开期间的窗口期。…
✗ 门控覆盖未达成：未覆盖 23 页 / 脚本错误 0 项 —— 退出码 1
```

**红得对的三条证据**：

1. `未覆盖页面：23`（= 25 应门控 − 2 已门控），**清单非空（23 ≥ 20）**；
2. `脚本错误 0 项` —— 红的原因是「**未铺开的页**」，**不是脚本崩溃**；
3. 输出用 `=== 未覆盖页面清单 ===` 与 `=== 脚本自身错误 ===` **两个分区**，两类问题在文本上可区分（闸门要求）。

### 幂等性

连跑两次输出**逐字节一致**（均 exit 1）：

```
run1_exit=1
run2_exit=1
IDEMPOTENT=YES
```

### 自检（红绿自检，均已实跑）

| 自检项 | 做法 | 结果 |
|---|---|---|
| 悬空豁免 | 把某条 `page` 改成不存在的路径 | ✓ 报 `[悬空豁免] src/pages/__not_exist__/index.tsx 不在页面清单中` + `[豁免条数漂移] 实测 50，断言 51` → exit 1 |
| 门控/豁免冲突 | 把已门控页 `network/devices` 塞进豁免清单 | ✓ 报 `[门控/豁免冲突] src/pages/network/devices/index.tsx 同时命中门控特征与豁免清单 —— 二者必须互斥` → exit 1 |
| 幂等 | 连跑两次 | ✓ 输出逐字节一致 |

> 自检后 `gating-exemptions.json` 已按 md5 校验还原（`59febbda0e6659b871c1c634b869cf34`），无残留改动。

### 踩坑登记（写脚本时踩到，已修）

**JSDoc 块注释里贴目录通配 glob 会提前闭合注释**：`src/pages` + 双星号 + `/index.tsx` 中的「星号紧邻斜杠」是合法的块注释闭合符，会让脚本在**解析期**崩掉（首跑曾因此 `SyntaxError` 而非业务性红）。已改为文字描述并在注释里写明该坑。**后续 phase 若在 `.mjs` 注释里写 glob，务必避开星号紧邻斜杠。**

---

## 六、`<Switch>` 人工判定台账（裁定 7 落盘）

PLAN 裁定 7：**「绑定写 handler 的状态切换 Switch」**才算写入口；纯视图切换不计入，允许保留 `read-only` 豁免（reason 须注明）。

**脚本不做判定，只打印证据**（`=== Switch 命中台账 ===`，21 页 / 31 处命中），人工判定的结论：

| 命中形态 | 判定 | 实例 |
|---|---|---|
| `<Form.Item valuePropName="checked">` 内、**无 onChange** | **表单字段**，随表单提交生效 → **非独立写入口** | 26 处（如 `system/menu:469,474`、`system/dict:938`、`monitor/job:338`、`duty/*`、`vdi/VDIServerConfig:326` 等） |
| `onChange={setXxx}` | **视图切换 Switch，非业务写** | `operations/building-spaces-3d/index.tsx:116` `onChange={setUseWebGL}`（只切 WebGL/普通渲染，不落库）—— 该页保留 `read-only` 豁免，reason 已注明 |
| `onChange` 调 `updateLayout` / `handleUpdate` | **本地偏好切换**，不触发后端写 | `settings/SettingCards.tsx:173`（侧边栏折叠态 → layoutStore）；该页 `non-route-page` |
| `onChange` 调 `onUpdateRow` | **表单行数据更新**，最终由批量保存提交 | `duty/holidays/modals/BatchModal.tsx:108` |
| `onChange` 调写 handler | **业务写候选，必须门控** | `system/user/index.tsx:270` `handleUpdateStatus(record.id, checked ? 0 : 1)`（该页已门控） |

**结论：51 页豁免中，无一个 `<Switch>` 属于「绑定写 handler 的状态切换 Switch」** —— 豁免的 Switch 全部是表单字段 / 视图切换 / 本地偏好。故**全部 51 条豁免成立**，不存在「read-only 却漏了业务写 Switch」的情形，86-05 的 R3 人工改判口子不会因本清单触发。

---

## 七、`lint:ci` 接线延后决策（⚠ 防后续 phase 误判）

**决策：本 wave 只挂 `npm run check:gating`，**不**挂 `lint:ci`；`lint:ci` 接线推迟到 86-05 全绿后。**

- `xingran-react-frontend/package.json` 新增一行：`"check:gating": "node scripts/check-gating-coverage.mjs"`；
- `lint:ci` 聚合链 **本 phase 不改**（`git diff package.json` 只出现 `check:gating` 一行；`dependencies` / `devDependencies` 段零 diff）。

**理由**：脚本在 W1~W4 期间**按设计必然是红的**（未覆盖页数 23 → 0 逐波下降）。挂进 `lint:ci` 会让整条链在 phase 中途持续失败，阻塞并行工作，也会让「红了」与「真回归」无法区分。Lead 要求的「先红后绿」正是这个语义。

**给后续 phase 的明确指令**：

> W0~W4 期间，`npm run check:gating` **退出码 1 是预期状态，不是回归**。
> 此时 `npm run lint:ci` **仍然应当 exit 0**（因为它尚未挂 `check:gating`）。
> 若某波发现 `npm run lint:ci` 挂了 → **那是真回归**，按常规处理；
> 若某波发现 `npm run check:gating` 仍红 → 看「未覆盖页面数」是否按预期下降（23 → … → 0），下降即正常。

**86-05 的接线动作**：`check:gating` 挂进 `lint:ci` 聚合链后，`lint:ci` 从此包含门控门禁；届时 `lint:ci` exit 0 = 门控覆盖 100%。

---

## 八、结构性局限登记（页级判定器 ≠ 入口级判定）

**`check-gating-coverage.mjs` 是页级判定器**：它判定「**这一页是否接入了门控机制**」（归属文件合并文本命中 `useActionPermission` **或** `permission:` + `PERMISSION_CODES`），**不判定「该页的每一个写入口是否都已门控」**。

具体地说，脚本**无法**发现下列情况：

1. 一页里有 5 个写入口，**只门控了 3 个**（其余 2 个裸奔）→ 页级判定器判「已门控」，实际仍有缺口；
2. 门控码**选错**（接了 `systemUser.list` 之类读码，而不是对应的 `add` / `edit` / `remove`）；
3. 三态语义写错（该用 `visible` 的地方用了 `disabled`，或 `notReadyHint` 未透传）；
4. 豁免页实际藏有未登记的写入口（特征集未覆盖的 handler 命名形态）。

**兜底手段（三层，缺一不可）**：

| 层 | 手段 | 负责兜住 |
|---|---|---|
| 1 | **施工图**（各 wave 的 PLAN 施工图逐入口列码） | 入口级覆盖 —— 逐入口登记该用哪个权限码，不依赖脚本 |
| 2 | **人工 review**（86-05 R3 抽样逐入口核对 + `reviewMode`） | 码选错 / 三态写错 / 豁免页藏写入口 |
| 3 | **抽样组件测试**（86-05 收口，8 页 × 三路径：无权限不在 DOM / 未就绪 disabled / 有权限可点） | 行为级回归 —— 三态语义真的生效 |

**结论**：脚本绿灯 **不等于** 门控覆盖 100%。绿灯只意味着「25 页全部接入了门控机制」。**86-05 的 100% 断言必须同时满足：脚本绿灯 + 8 页抽样组件测试全绿 + R3 人工核对通过**三者才算收口。

---

## 九、窗口期口径（ready 无码 → 不渲染）

- `useActionPermission.ts:100-109` 的 `decide` 语义：**未传码 → 恒可见可用**；**未就绪（loading/error）→ 可见但 disabled + 提示**；**ready 且无该码 → 不渲染**。
- 窗口期内（87 播种前）凡 `permissionCodes.ts` 里没有的码，`decide` 返回 `visible:false` → **按钮不渲染**，与 87 播种前后端 fail-closed（HTTP 403）**同向**，无需特殊处理。
- 推论：**86 不为缺码页面自造任何权限码字符串**。铺开过程中若发现某写入口在 `permissionCodes.ts` 无对应码 → 登记进第十节「缺码待回补」，**不**在页面里写死码字面量（PERM-06 单一真相源）。

---

## 十、缺码待回补登记（等后端拆分 phase，本 phase 不处理）

86 全程**不改** `src/constants/permissionCodes.ts`（Lead 裁定 4）。下列写域在 TS 清单无对应区块，已按 `no-permission-code` / `legacy-two-source` 登记豁免，**等后端拆分 phase 回补 TS 声明后逐条移除豁免**：

| 域 | TS 清单现状 | 后端 `pkg/permission/config.go` 现状 | 待回补内容 |
|---|---|---|---|
| system 的 dict / notice / config / apikey / captchaBackground | 无对应区块 | **有** `system:dict:*` / `system:notice:*` / `system:config:*` / `system:apikey:*` / `system:captchaBackground:*` | 补 TS 区块（后端已有常量，只缺前端声明） |
| monitor 全域 | 无 monitor 区块 | 有 `monitor:cache:*` / `monitor:job:*` / `monitor:log*` / `monitor:server:*` 等 28 条 | 补 TS 区块 |
| network 的 mac / port | 无 networkMac / networkPort 区块 | 有 `network:mac:*` / `network:port:*` | 补 TS 区块 |
| ad-domain 的 ou / computer | AD 域仅 `opsADConfig` / `opsADGroup` / `opsADUser` | `ops:ad` 下无 ou / computer 操作码 | 需后端先拆出操作码 |
| `ad-domain/configs/ReconciliationTab` 的 `ops:ad:config:reconciliation:list` | 不在 TS 清单 | 后端无该码 | **旧式直判**，随 W4 收口（不豁免） |
| duty / workorder / knowledge / vdi / rpa / asset（旧域） | 无对应区块 | **后端也没有**对应前缀（85 已明确为范围外域） | 需后端拆分 phase 整体拆分 |

---

## 十一、与 Phase 90 线零交叠声明

- 86-00 改动面**仅限**：`src/pages/**`（W1~W4，本 wave 为 0）、`xingran-react-frontend/scripts/check-gating-coverage.mjs`、`xingran-react-frontend/scripts/gating-exemptions.json`、`xingran-react-frontend/package.json`（仅 `scripts` 段一行）、本文件。
- **未触碰** `src/index.css`、**未触碰** `src/design-system/**`、**未触碰** 任何 `.css` 文件 → 与 90 线（CSS/设计令牌）**零交叠**。
- **未触碰**后端（85 已完成，路由登记 / Strict 挂载不动）；**未做** F 型播种与授予（归 87）。
- **未新增**任何 npm 依赖；**未做** git 操作（本 wave 不 commit / 不 add）。

---

## 十二、W5 收口提示（给 86-05 executor）

1. **豁免清单定稿**：按脚本实跑核对 **51 条**（`EXPECTED_EXEMPTION_COUNT`）；若铺开过程中出现豁免增减，必须**同批**改 `check-gating-coverage.mjs` 的 `EXPECTED_EXEMPTION_COUNT` **并在本文件说明理由**（页数增减会直接打破 `25 + 51 = 76` 恒等式）。
2. **覆盖率 100% 断言**：脚本「未覆盖页面：0」+ 已门控 25 + 豁免 51 = 76。
3. **挂 `lint:ci`**：把 `check:gating` 加进 `lint:ci` 聚合链（放在最后），并把第七节的「预期红」说明改成「预期绿」。
4. **8 页抽样组件测试全绿**（三路径 × 8 页，见 `86-05-PLAN.md`）。
5. **人工 review（R3）**：逐入口核对 25 页的权限码选型与三态语义（脚本管不到，见第八节）。
6. **零改动核验**：`git diff --stat -- src/constants/permissionCodes.ts` 必须为空；`git diff --stat -- src/index.css src/design-system/**` 必须为空。
7. **ROADMAP / REQUIREMENTS 勾选** + `86-VERIFICATION.md` 落盘。

---

## 十三、W1 铺开进度（Plan 01 / 2026-10-29）

### 覆盖率推进（脚本实跑）

| 项 | W0 基线 | W1 完成后 | 变化 |
|---|---:|---:|---|
| 已门控页面 | 2 | **8** | +6 |
| 未覆盖页面 | 23 | **17** | −6 |
| 豁免条数 | 51 | **51** | 0（未变动） |
| 应门控页面 | 25 | 25 | 0 |

**恒等式复核**：`8 已门控 + 17 未覆盖 = 25 应门控`；`25 + 51 = 76 页面` —— 两条恒等式均保持。
`EXPECTED_EXEMPTION_COUNT`（= 51）**未改动**（豁免清单零增减，故无须登记理由）。

### ⚠️ 与 Lead 裁定「23 → 16」的偏差登记（口径澄清）

Lead（父 agent）给 W1 的预期是「未覆盖 23 → **16**（system 域 7 页铺开）」；**脚本实跑为 17**，差 1 页。原因是**计数口径**而非实现缺陷：

- `system/user` 页在 W0 就**已门控**（它是「已门控 2」基线之一：`system/user` + `network/devices`），
  因此**不在 23 页未覆盖清单内**；W1 对它做的是**复核**（补齐遗漏入口），复核不改变覆盖计数。
- W1 真正**新增**门控的只有 6 页：role / menu / dept / post / manager-orgs / manager-rules。
- 故：`2 基线 + 6 新增 = 8 已门控`，`25 − 8 = 17 未覆盖`（= ops 8 + network 6 + ad-domain 3）。

**裁定**：以脚本实跑的 **8 / 17** 为准（与 86-01-PLAN Task 1.6 的 acceptance_criteria「已门控计数 = 8」完全一致）；
Lead 的「16」若按「7 页里 user 复核也算一页覆盖」的算法得出，则与「已门控 2」基线重复计数，**不采纳**。
后续 W2~W4 请沿用同一口径：**新增门控页数 = 未覆盖页数下降数**。

### W1 逐页门控入口与权限码（施工图落地结果）

| 页 | 门控入口 | 权限码 |
|---|---|---|
| **user**（复核） | 行编辑 / 行重置密码 / 行删除 / 批量删除 / 新增 / 导入 / 状态 Switch（7） | `systemUser.edit / resetPwd / remove / add / import`（**零改动**，既有 7 用例全绿） |
| **role** | 工具栏新增 / 批量删除 / 行编辑 / 行停用·启用 / 行删除（5） | `systemRole.add / remove / edit / edit / remove` |
| **menu** | 工具栏新增 / 批量删除 / 行编辑 / 行删除 / 弹窗显示状态 Switch / 弹窗菜单状态 Switch（6） | `systemMenu.add / remove / edit / remove / edit·add / edit·add` |
| **dept** | 工具栏新增 / **导入触发按钮** / **导出触发按钮** / 行编辑 / 行删除（5） | `systemDept.add / import / export / edit / remove` |
| **post** | 工具栏新增 / 批量删除 / 行编辑 / 行删除（4） | `systemPost.add / remove / edit / remove` |
| **manager-orgs** | 工具栏「从部门架构生成」 / 工具栏「新增机构」 / 行编辑 / 行删除 / 弹窗启停 Switch（5） | `opsManagerOrg.edit / edit / edit / delete / edit` |
| **manager-rules** | 工具栏「新增规则」 / 行编辑 / 行删除 / 弹窗启停 Switch（4） | `opsManagerRule.edit / edit / delete / edit` |

**合计门控入口 38 个**（user 7 + role 5 + menu 6 + dept 5 + post 4 + manager-orgs 5 + manager-rules 4）。

### 非 CRUD 写入口的归码理由（逐处）

| 入口 | 归码 | 理由 |
|---|---|---|
| role 行「停用/启用」 | `systemRole.edit` | `systemRole` 码组只有 list/view/add/edit/remove/export，**无独立 status 键**；启停属状态写动作，就近归 `edit`（不自造新码） |
| manager-orgs「新增机构」 | `opsManagerOrg.edit` | `opsManagerOrg` 码组只有 list/edit/delete，**无 add 键**；按「归最接近既有码」裁定归 `edit` |
| manager-orgs「从部门架构生成机构」 | `opsManagerOrg.edit` | 同上（无 generate 键）；⚠️ **不可归 delete** —— 否则持 delete 无 edit 的用户可批量生成机构（越权写） |
| manager-orgs 弹窗「是否启用」Switch | `opsManagerOrg.edit` | 表单字段，随新增/编辑机构提交生效 → 就近归所属写动作 |
| manager-rules「新增规则」 | `opsManagerRule.edit` | `opsManagerRule` 码组只有 list/edit/delete，**无 add 键** |
| manager-rules 弹窗「是否启用」Switch | `opsManagerRule.edit` | 表单字段，随新增/编辑规则提交生效 → 就近归所属写动作 |
| menu 弹窗「显示状态 / 菜单状态」Switch | `systemMenu.edit`（编辑弹窗）/ `systemMenu.add`（新增弹窗） | 表单字段，随新增/编辑菜单提交生效 → **随弹窗上下文归所属写动作**（避免「有 add 无 edit 的用户新增菜单时看不到状态字段」的功能退化） |

### 实现模式核对（对照 PLAN「实现模式」段）

- 每页 **1 行** `useActionPermission()` hook（`permGate`）+ 每入口 1 个 `permission:` 传参 / 条件渲染；
- **无** PageTable 级 hook、**无** 路由→码映射表、**无** 自造权限码字符串；
- `src/components/shared/**` 与 `src/constants/permissionCodes.ts` **零 diff**；
- `ActionButtons` 行操作走其内建 `permission:` 字段（组件内已有三态门控）；**裸 `<Button>` 行操作**（menu / manager-orgs / manager-rules）走页面层条件渲染 + `disabled` + `title`；
- Excel 类入口（dept 导入/导出）门控点落在**触发按钮**，modal 容器保留无条件渲染（触发按钮不渲染 ⇒ 弹窗永远打不开）；
- 批量删除全部走**页面层条件传参**（role/menu/dept 独立 Button 条件渲染；post 走 `PageTable.batchDelete={visible ? {...} : undefined}`）。

### W1 组件测试

- `src/pages/system/role/__tests__/permission.gate.test.tsx`：7 例（全绿）；
- `src/pages/system/menu/__tests__/permission.gate.test.tsx`：8 例（全绿，含 `<Switch>` 形态三态）；
- `src/pages/system/user/__tests__/permission.gate.test.tsx`：既有 7 例**零回归**（未改动该文件）。

---

## 十四、W2 铺开进度（Plan 02 / 2026-10-29）

### 覆盖率推进（脚本实跑）

| 项 | W1 完成后 | W2 完成后 | 变化 |
|---|---:|---:|---|
| 已门控页面 | 8 | **16** | +8 |
| 未覆盖页面 | 17 | **9** | −8 |
| 豁免条数 | 51 | **51** | 0（未变动） |
| 应门控页面 | 25 | 25 | 0 |

**恒等式复核**：`16 已门控 + 9 未覆盖 = 25 应门控`；`25 + 51 = 76 页面` —— 两条恒等式均保持。
未覆盖 9 页 = network 6（backups / command / credentials / discoveries / executions / mac-history... 见脚本清单）+ ad-domain 3（configs / groups / users）。
`EXPECTED_EXEMPTION_COUNT`（= 51）**未改动**（豁免清单零增减）。

### W2 逐页门控入口与权限码（施工图落地结果）

| 页 | 门控入口（含 Excel 触发按钮 + 导出类入口） | 权限码 |
|---|---|---|
| **buildings** | 行编辑 / 行删除 / 批量删除 / 导入触发 / 导出触发 / 新增 / 卡片视图编辑·删除（7） | `opsBuilding.edit / .delete / .delete / .import / .export / .add` |
| **floors** | 行编辑 / 行「查看平面图」/ 行删除 / 批量删除 / 导入触发（`FloorSearchForm`）/ 导出触发（`FloorSearchForm`）/ 新增 / 平面图编辑模式（`FloorPlanEditorView`）/ 卡片视图编辑·平面图·删除（9） | `opsFloor.edit / .delete / .import / .export / .add`；平面图 → **四码合并**（见下） |
| **workstations** | 行编辑 / 行删除 / 批量删除 / 导入触发 / 「下载部门映射表」/ 导出触发 / alias 抽屉入口 / 新增 / **平面图视图入口** / 卡片视图编辑·删除（10） | `opsWorkstation.edit / .delete / .import / .export / .export / .add`；alias → `opsLocationAlias.list/add/delete` |
| **info-points** | 行编辑 / 行删除 / 批量删除 / 导入触发 / 导出触发 / 新增 / 卡片视图编辑·删除（7） | `opsInfoPoint.edit / .delete / .import / .export / .add` |
| **server-rooms** | 行编辑 / 行删除 / 批量删除 / 导入触发 / 导出触发 / 新增 / 卡片视图编辑·删除（7） | `opsServerRoom.edit / .delete / .import / .export / .add` |
| **room-devices** | 行编辑 / 行删除 / 批量删除 / 导入触发 / 导出触发 / 新增 / 卡片视图编辑·删除（7） | `opsRoomDevice.edit / .delete / .import / .export / .add` |
| **dedicated-lines** | 行编辑 / 行删除 / 批量删除 / 导入触发 / 导出触发 / 新增 / 卡片视图编辑·删除（7） | `opsDedicatedLine.edit / .delete / .import / .export / .add` |
| **assets** | 行删除 / 批量删除 / 「下载模板」/ 导入触发 / 导出触发 / 新增（6） | `opsAsset.delete / .import / .export / .export / .add`（本页无行编辑入口） |

**合计门控入口 60 个**（buildings 7 + floors 9 + workstations 10 + info-points 7 + server-rooms 7 + room-devices 7 + dedicated-lines 7 + assets 6）。

### 导出类写入口逐页 grep 复核（NEW-02 两处）

| 页 | 导出类入口 | 门控码 | 复核方式 |
|---|---|---|---|
| **assets** | 「下载模板」`index.tsx:639-654` → `excelApi.downloadTemplate("asset")` | `opsAsset.export` | `grep -rn "downloadTemplate\|excelApi" src/pages/operations/assets/` → 仅此一处，已条件渲染 + disabled + title |
| **workstations** | 「下载部门映射表」`index.tsx:834-836` → `deptApi.exportMapping()` | `opsWorkstation.export` | `grep -rn "exportMapping" src/pages/operations/workstations/` → 仅此一处，已条件渲染 + disabled + title |
| 其余 6 页 | 均只有 `ExportOutlined` 触发按钮（无额外下载/模板入口） | 各页 `.export` | 逐页 grep `ExportOutlined / DownloadOutlined / downloadTemplate / exportMapping`，无漏网 |

**额外发现并补齐 1 处施工图未列的写入口**：workstations「平面图」视图（`Radio.Button value="floorplan"`）—— 该视图内拖动工位会调 `workstationApi.updatePositions`（`hooks/useWorkstationView.ts:60`）**写工位位置**，与 floors 平面图同一性质（可保存编辑器）→ 归 `opsWorkstation.edit`，已条件渲染 + disabled + hint，并在抽样测试中覆盖。

### `FloorSearchForm` 三通道契约（本波唯一组件契约加法）

最终 TS 签名：

```ts
export interface FloorSearchFormProps {
  // …既有字段零变更…
  onImport?: () => void;          // 通道①：必填 → 可选（不传 ⇒ 按钮不渲染）
  onExport?: () => void;          // 通道①：必填 → 可选
  importDisabled?: boolean;       // 通道①：新增可选位（默认 false）
  exportDisabled?: boolean;       // 通道①：新增可选位（默认 false）
  notReadyHint?: string;          // 通道①：未就绪提示（透传到触发按钮 title）
  onBatchDelete: () => void;      // 通道②：保持必填（页面层 noop 包装兜底）
  batchDeleteDisabled?: boolean;  // 通道②：新增可选位（默认 false）
  showBatchDelete?: boolean;      // 通道②：新增可选位（默认 true —— 决定渲不渲染）
}
```

三态语义核对（批删按钮「批量删除 (N)」）：

| 权限态 | `decide()` 返回 | `showBatchDelete` | `batchDeleteDisabled` | 实测按钮表现 |
|---|---|---|---|---|
| ready + 有 `opsFloor.delete` | `{visible:true, disabled:false}` | `true` | `false` | 渲染且可用 |
| ready + **无**权限 | `{visible:false, disabled:false}` | **`false`** | `false` | **不渲染（不在 DOM）** |
| **未就绪**（loading / error） | `{visible:true, disabled:true}` | `true` | **`true`** | **渲染但 disabled + hint** |

通道③（shared 零 diff 边界）：上述加法**全部落在** `src/pages/operations/floors/components/FloorSearchForm.tsx`；`src/components/shared/**` 实跑 `git diff --stat` = **空**。

### 平面图四码合并口径的落地

- 决策对象在 `floors/index.tsx` 内构造（页面层唯一判定点）：
  `permFloorPlanEdit ∧ permWall(opsWall.edit) ∧ permDoor(opsDoor.edit) ∧ permWorkstationEdit`；
- 合并语义：任一码**未就绪** → `{visible:true, disabled:true, hint}`（渲染但禁用）；
  四码全就绪且**全持** → 可用；全就绪但**缺任一码** → `{visible:false}`（不渲染）；
- 同一决策对象同时喂给**三处入口**：① 行「查看平面图」（经 `FloorTableColumns` 的 `floorPlanDecision` prop）；
  ② 平面图工具栏「编辑模式」按钮（经 `FloorPlanEditorView` 的 `editDecision` prop）；③ 卡片视图平面图图标；
- 源码注释写明合并口径与「漏 `opsWorkstation.edit` 即越权」的理由（`useFloorPlanEditor.ts:249-255`
  的 `saveWorkstations` 在同一次保存里改写工位位置）。

### alias 旧式直判迁移（本波唯一迁移项）

- `workstations/index.tsx:122` 的 `menuPermissions.includes("ops:location:alias:list")` → `permAliasList = permGate.decide(PERMISSION_CODES.opsLocationAlias.list)`；
- `LocationAliasDrawer.tsx:62-65` 的 `useMenuStore` + 自造 `hasPermission` + 硬编码字面量 → `permGate.decide(opsLocationAlias.add / .delete)`；
- `grep -rn "useMenuStore" src/pages/operations/workstations/` 命中数 = **0**；
- **行为变更登记（本波唯一）**：旧写法「权限集不含该码 → 入口不渲染」→ 新写法「ready 无码 → 不渲染；
  **未就绪 → 渲染但 disabled + hint**」。属收紧到更安全一侧（不再让用户「看不到入口也无从得知原因」），与 v1.24 PERM-02 设计意图一致。

### W2 组件测试

- `src/pages/operations/buildings/__tests__/permission.gate.test.tsx`：**5 例**（全绿）—— ① ready 只持读码全部入口不在 DOM；② 未就绪 loading → 在 DOM 且 disabled + hint；②-2 未就绪 error → 同上；③ 全写码 → 可用；④ ready 空权限集 → 不渲染（与未就绪可区分）；
- `src/pages/operations/workstations/__tests__/permission.gate.test.tsx`：**5 例**（全绿）—— 同上四路径 + 覆盖「列在 `columns.tsx` 的接法」与「alias 抽屉入口条件渲染的接法」+ 「下载部门映射表」+ 平面图视图入口；
- **两个新测试文件专项类型检查错误数 = 0**（临时 tsconfig：`extends ./tsconfig.app.json` + `types` 加 `vitest/globals` + 指定 include + `exclude: []`）；
- **floors 测试目录类型债基线 36 未新增**（复跑仍 = 36）；
- floors 既有测试 **10 files / 71 tests 零回归**；workstations 既有 14 files / 86 tests 零回归。

### 既有测试的语义适配（未删任何断言）

| 测试文件 | 适配内容 | 理由 |
|---|---|---|
| `operations/dedicated-lines/__tests__/index.pageTable.test.tsx:84` | 源码字面量断言 `batchDelete={{ onDelete: handleBatchDelete, target: "专线" }}` → 放宽为「`batchDelete={` + 载荷片段 + `permDelete.visible` 条件传参在场」 | `batchDelete` 已改为**条件传参**（PageTable 契约无 permission 位）；契约意图（仍走 PageTable.batchDelete / 仍以 handleBatchDelete+target 为载荷）不变 |
| `pages/__tests__/batch53.modalOpen.test.tsx` | `openModalAndAssert` 前置显式构造 `permissionsLoaded:true` + 授予 ops 8 页写码 | 三态语义下「未就绪」按设计 disabled，而该 helper 的判定对象是「点新增 → Modal 打开」交互分支 |

### 结构性局限（页级判定器 ≠ 入口级判定）—— W2 实证

`check-gating-coverage.mjs` 是**页级**判定器（详见第八节）：它只判「该页是否命中写入口特征 且 是否出现门控特征」，
**不判单个入口是否已门控**。W2 实证：施工图曾遗漏 **2 处导出类写入口**（assets「下载模板」`index.tsx:639-654`、
workstations「下载部门映射表」`index.tsx:834-836`，均为**数据外带面**）与 **1 处视图内写入口**
（workstations 平面图视图 `updatePositions`）—— 脚本对这三处**全部无感**，是**逐页 grep 写 API 调用点**才发现的。
→ 归口 86-05 收口时做一次**全 ops 域导出类入口 + 视图内写入口**抽样复核；入口级覆盖只能靠
**施工图 + 人工 review + 抽样测试**三层兜底（脚本绿灯 ≠ 门控覆盖 100%）。

---

## 十五、W3 铺开进度（Plan 03 / 2026-10-29）

### 覆盖率推进（脚本实跑）

| 项 | W2 完成后 | W3 完成后 | 变化 |
|---|---:|---:|---|
| 已门控页面 | 16 | **22** | +6（devices 属基线 2 页，不重复计数） |
| 未覆盖页面 | 9 | **3** | −6 |
| 豁免条数 | 51 | **51** | 0（未变动） |
| 应门控页面 | 25 | 25 | 0 |

**恒等式复核**：`22 已门控 + 3 未覆盖 = 25 应门控`；`25 + 51 = 76 页面` —— 两条恒等式均保持。
未覆盖 3 页 = ad-domain 3（`ad-domain/configs` / `ad-domain/groups` / `ad-domain/users`）。
`EXPECTED_EXEMPTION_COUNT`（= 51）与 `EXPECTED_PAGE_COUNT`（= 76）**均未改动**。

### W3 逐页门控入口与权限码（施工图落地结果）

| 页 | 门控入口 | 权限码 |
|---|---|---|
| **devices**（复核扩面） | 行「采集端口」（`useDeviceRowActions.tsx:131-140`，走降级解析 `collectPerm`）/ 批量删除（`DeviceTablePanel` 的 `batchDelete` 条件传参）/ **批量导出**（`DeviceTablePanel.tsx:157` 的 `<NetworkExport>` 条件渲染）/ 「快速创建」+「手动新增」（`permission.add` 条件渲染） | `networkDevice.edit`（采集端口）/ `.delete` / `.export` / `.add`；编辑·删除沿用既有 `editPerm`/`removePerm` |
| **credentials** | 行「设为默认」/ 行编辑 / 行删除 / 批量删除 / **批量导出**（`:477` 的 `<NetworkExport>`）/ 新增 | `networkCredential.edit`（设为默认就近归码）/ `.edit` / `.delete` / `.delete` / `.export` / `.add` |
| **templates** | 行「克隆」/ 行编辑 / 行删除 / 批量删除 / **批量导出**（`index.tsx:352` 的 `<NetworkExport>`）/ 新增；「预览」**不门控**（读） | `networkTemplate.edit`（克隆就近归码）/ `.edit` / `.delete` / `.delete` / `.export` / `.add` |
| **backups** | 行「还原」/ 行「版本对比」/ 还原确认（`RestoreModals.tsx` 的「确认还原」页脚）/ 版本抽屉「恢复到指定版本」/ 新增备份 / **批量导出**（`:523` 的 `<NetworkExport>`）；「版本列表」**不门控**（读） | `networkBackup.restore`（**不是 delete**）/ `.diff` / `.restore` / `.restore` / `.add` / `.export` |
| **discoveries** | 行删除（`createDeleteAction` 的 `permission` 位）/ 新增任务 / **批量导出**（`:253` 的 `<NetworkExport>`）；「查看结果」**不门控**（读） | `networkDiscovery.delete` / `.add` / `.export` |
| **command** | 「快速命令」/ 行「取消」/ **批量导出**（`:275` 的 `<NetworkExport>`）；行「查看」**不门控**（读） | `networkCommand.execute` / `.execute` / `.export` |
| **executions** | 「执行配置模板」/ 行「取消执行」/ **批量导出**（`:291` 的 `<NetworkExport>`）；行「查看」**不门控**（读） | `networkCommand.execute`（执行记录取消**复用**命令执行码）/ `.execute` / `.export` |

**合计门控入口 31 个**（devices 5 + credentials 6 + templates 6 + backups 6 + discoveries 3 + command 3 + executions 3）。

### backups「还原 ≠ 删除」语义核对证据（MEDIUM-4）

| 位置 | 实际语义证据 | 正确归码 |
|---|---|---|
| `backups/index.tsx:197` `handleConfirmRestore` | `title:"确认恢复配置"` / `actionLabel:"恢复"` / `actionContext:"恢复配置"` / `successText:"已恢复配置"` / `onDelete: async () => { await confirmRestore(); }` | `networkBackup.restore` |
| `backups/index.tsx:240` `handleRestore` | 同上文案 + `openRestoreModal(backup)` + `confirmRestore(backup)` | `networkBackup.restore` |
| `backups/modals/RestoreModals.tsx` 版本抽屉「恢复」行操作 | `title:"确认恢复到此版本"` / `actionLabel:"恢复"` / `successText:"已恢复配置"` | `networkBackup.restore` |

**证据结论**：`confirmDelete` 只是二次确认 helper（形参名 `onDelete` 与业务语义无关），三处的业务动作都是**还原（下发配置）**。
本页**无任何删除入口**（无 `handleDelete` / 无 `createDeleteAction` / 无 `batch-delete` 端点调用）。

- `grep -rn "networkBackup\.delete" src/pages/network/backups/ | wc -l` → **0**（源码中不出现该键，注释也改写为「删除类码」避免字面量命中）；
- 决策变量名：`permRestore`（**不叫** `permDelete`），源码注释写明「本页无删除入口，这两处 confirmDelete 承载的是还原动作」；
- 回归防线：`backups/__tests__/permission.gate.test.tsx` 用例 ③-3（持 restore **无** delete → 还原仍可用）与 ③-4（持 delete **无** restore → 还原不在 DOM）双向锁定。

### 死 modal 债登记（`<BatchExportModal>` × 7，本波**不门控不改逻辑**）

grep 证据：`grep -rn "setBatchModalVisible(true)" src/pages/network/ | wc -l` → **0**；
`grep -rn "setBatchModalVisible(false)" src/pages/network/ | wc -l` → **18**（全部是关闭路径）。

| 页 | `<BatchExportModal>` 位置 | `setBatchModalVisible` 调用 | 判定 |
|---|---|---|---|
| devices | `components/DeviceTablePanel.tsx:226` | `useDeviceActions.tsx` + `index.tsx`（均 `false`） | **死 modal**（`visible` 恒 false） |
| credentials | `index.tsx:493` | `:197` `:460`（`false`） | **死 modal** |
| templates | `index.tsx:361` | `:190` `:347`（`false`） | **死 modal** |
| backups | `index.tsx:590` | `:262` `:556`（`false`） | **死 modal** |
| discoveries | `index.tsx:335` | `:196` `:308`（`false`） | **死 modal** |
| command | `index.tsx:288` | `:140` `:270`（`false`） | **死 modal** |
| executions | `index.tsx:307` | `:136` `:284`（`false`） | **死 modal** |

**结论**：7 处 `BatchExportModal` 的 `visible` 永远为 `false`（无任何 `(true)` 调用）→ **永远打不开**。
真正的导出入口是 `<NetworkExport>`（按钮在 `NetworkExport.tsx:120` 组件内部）。
本波对 7 处死 modal **零改动**（不改即无行为变化），登记为**死 modal 债**，交由后续 phase 决定删除或接线。

### 导出类 / 视图内写入口逐页 grep 复核

复核方式：对 7 页各目录 grep `NetworkExport` / `ExcelExport` / `ExcelImport` / `downloadTemplate` / `exportMapping` / `batchExport` / `await post(` / `onImport`。

| 页 | 导出类入口 | 门控 | 复核结果 |
|---|---|---|---|
| devices | `DeviceTablePanel.tsx:157` `<NetworkExport>` | ✓ `networkDevice.export` | 组件目录内 `NetworkExport` 仅 1 处；`batchExport` 仅 `useDeviceActions`（死 modal 的 `handleBatchExport`，已随 modal 一起不门控） |
| credentials | `index.tsx:477` `<NetworkExport>` | ✓ `.export` | `batchExport` 调用仅在 `handleBatchExport`（死 modal 通道） |
| templates | `index.tsx:352` `<NetworkExport>` | ✓ `.export` | 同上 |
| backups | `index.tsx:523` `<NetworkExport>` | ✓ `.export` | 同上 |
| discoveries | `index.tsx:253` `<NetworkExport>` | ✓ `.export` | **另发现 `ResultModal` 的「导入设备」按钮**（`onImport` → `POST /network/discoveries/{id}/import`，落库导入发现的设备）—— 见下方遗留缺口 |
| command | `index.tsx:275` `<NetworkExport>` | ✓ `.export` | 无额外导出/导入入口 |
| executions | `index.tsx:291` `<NetworkExport>` | ✓ `.export` | 无额外导出/导入入口 |

**结构性局限复核结论**：7 页均无 `ExcelExport` / `ExcelImport` / `downloadTemplate` / `exportMapping`（network 域导出统一走 `NetworkExport` + 后端 `/api/v1/network/{entity}/export`）。
**视图内写入口**：7 页均无平面图/画布类视图（network 域无 `saveFloorPlan` / `updatePositions` 类调用）→ 本波无「视图内写入口」缺口。

### W3 组件测试

| 文件 | 用例数 | 覆盖 |
|---|---:|---|
| `network/credentials/__tests__/permission.gate.test.tsx` | **5** | ① 只持 `.list` → 新增/行操作/批删/导出均不在 DOM；② loading → 新增 disabled + 导出被 `aria-disabled` 包裹 + 行操作禁用；②-2 error 同上；③ 全写码 → 新增/导出可用 + 下拉含「设为默认·编辑·删除」且未禁用；④ ready 空权限集 → 不渲染 |
| `network/backups/__tests__/permission.gate.test.tsx` | **9** | ① 只持 `.list` → 创建备份/导出不在 DOM、只读「版本列表」保留；② loading / ②-2 error → 入口 disabled；③ 持 restore 无 diff / ③-2 持 diff 无 restore（两码**不得互换**）；**③-3 越权回归（持 restore 无 delete → 还原仍可用）**；**③-4 反向越权（持 delete 无 restore → 还原不在 DOM）**；③-5 全码；④ ready 空权限集 |
| `network/devices/__tests__/permission.gate.test.tsx` | **6** | ① **降级解析路径**（只持页面级 `network:device` → 采集端口/编辑/删除/快速创建/手动新增全部仍可见）；② 动作码路径（持 `network:device:edit` → 编辑可见、删除不在）；②-2 只持 `network:device:view` → 编辑不在 DOM（证明已切动作码判定）；③ loading → 快速创建 disabled + 行操作禁用；③-2 error；④ 导出码门控（无 `.export` → 「导出」按钮不在 DOM；持该码 → 可用） |

- **三个新测试文件专项类型检查错误数 = 0**（临时 tsconfig `tsconfig.gate-test.check.json`：`extends ./tsconfig.app.json` + `types` 加 `vitest/globals` + 指定 include + `exclude: []`）；
- `npx vitest run src/pages/network` → **50 files / 274 tests 全绿**（devices 既有测试零回归）；
- 既有测试语义适配 **1 处**（`pages/__tests__/batch53.modalOpen.test.tsx`：`OPS_WRITE_PERMISSIONS` 前置集追加 `networkDevice.list/add/edit/delete/export` —— devices 的「手动新增」在 W3 接入 `networkDevice.add` 降级解析，未删任何断言）。

### W3 结构性局限复核（页级判定器 vs 入口级判定）

`check-gating-coverage.mjs` 对 network 7 页的判定：本波铺开后 7 页全部判「已门控」。
逐页 grep 复核**未发现**新的导出类/视图内写入口漏网（network 域导出统一走 `NetworkExport`，无 Excel 容器、无画布视图）。

**唯一登记项（未纳入本波门控，需后续裁定）**：`network/discoveries/modals/ResultModal.tsx:50-60` 的「导入设备」按钮
（`onImport` → `POST /network/discoveries/{id}/import`，把发现的设备**落库**为正式设备）—— 这是**写入口**（数据入库），
但 PLAN 3.1 的施工图未列该入口（PLAN 只列了「查看结果（读）/ 删除 / 新增任务 / 导出」四类）。
按纪律「不扩范围」，本波**未改动**该文件；登记为 **W3 遗留缺口**，建议 86-04 或 86-05 收口时按
`networkDiscovery.add`（发现任务创建/导入同属 discovery 写动作域）补门控。

> ✅ **该缺口已闭环**：commit `499b2991`（`fix(86-03)`）按 `networkDiscovery.add` 补齐了
> `ResultModal` 的「导入设备」入口门控（新增 `importVisible/importDisabled/importHint` 三个可选 prop，
> 默认值保持既有调用方零改动）。W4 复核：该文件已含 `useActionPermission` + `PERMISSION_CODES.networkDiscovery.add`，
> 不再是遗留项。

---

## 十六、W4 铺开进度（Plan 04 / 2026-10-29）

### 覆盖率推进（脚本实跑）

| 项 | W3 完成后 | W4 完成后 | 变化 |
|---|---:|---:|---|
| 已门控页面 | 22 | **25** | +3（ad-domain 3 页） |
| 未覆盖页面 | 3 | **0** | −3 |
| 豁免条数 | 51 | **51** | 0（未变动） |
| 应门控页面 | 25 | 25 | 0 |

**恒等式复核**：`25 已门控 + 0 未覆盖 = 25 应门控`；`25 + 51 = 76 页面` —— 两条恒等式均保持。
`EXPECTED_EXEMPTION_COUNT`（= 51）与 `EXPECTED_PAGE_COUNT`（= 76）**均未改动**。

**🔴 先红后绿链条在本波闭合**：`npm run check:gating` 由 W3 的 **exit 1（未覆盖 3 页）** → **exit 0（未覆盖 0 页）**。
红→绿证据链完整：W0（红：已门控 2 / 未覆盖 23）→ W1（8/17）→ W2（16/9）→ W3（22/3）→ **W4（25/0，转绿）** → W5（挂 `lint:ci`）。

### W4 施工图对照表 —— configs 页 + AccountPoolTab（PLAN 的 14 行逐条回填）

> PLAN「3 页 × 入口 × 权限码映射」中 configs 一节的 **14 行追溯清单**，逐条回填三态结论。
> **无空缺行**。行号列同时给出 PLAN 施工图行号与**落地后实际行号**（W4 改动使行号整体下移）。

| # | 入口 | PLAN 行号 → 落地行号 | 权限码 | 门控形态 | **结论** |
|---|---|---|---|---|---|
| 1 | 测试连接 | `configs/index.tsx:297-300` → `:325-332` | `opsADConfig.test` | ActionButtons 的 `permission:` | ✅ 已门控 |
| 2 | 同步数据 | `:304-307` → `:334-341` | `opsADConfig.sync` | ActionButtons 的 `permission:` | ✅ 已门控 |
| 3 | 详情 | `:311-314` → `:343-349` | —（读） | **读 → 不门控** | ✅ 已登记不门控 |
| 4 | 删除 | `:317-325` → `:350-365` | `opsADConfig.delete` | ActionButtons 的 `permission:`（**见 W4-偏离 ②**） | ✅ 已门控 |
| 5 | 新增AD配置 | `:348` → `:387-397` | `opsADConfig.add` | 条件渲染 + disabled + title | ✅ 已门控 |
| 6 | 刷新 | `:351` → `:399-401` | —（读） | **读 → 不门控** | ✅ 已登记不门控 |
| 7 | 使用SSL (LDAPS) Switch | `:437` → `:487-491` | **随弹窗上下文**：新增→`add` / 编辑→`edit`（**见 W4-偏离 ③**） | 条件渲染 + disabled + title | ✅ 已门控 |
| 8 | 使用TLS (StartTLS) Switch | `:441` → `:493-497` | 同上 | 条件渲染 + disabled + title | ✅ 已门控 |
| 9 | 启用自动同步 Switch | `:445` → `:499-503` | 同上 | 条件渲染 + disabled + title | ✅ 已门控 |
| 10 | **编辑此配置（详情 Drawer 内）** | `:505-515` → `:563-583` | `opsADConfig.edit` | 条件渲染 + disabled + title | ✅ 已门控（HIGH-1 闭环） |
| 11 | 账号池：编辑 / 立即解锁 / 启用·停用 | `AccountPoolTab.tsx:323-348` → `:359-393` | `opsADConfig.edit` | 条件渲染 + disabled + title（**见 W4-偏离 ①**） | ✅ 已门控 |
| 12 | 账号池：删除 | `:349-359` → `:395-414` | `opsADConfig.delete` | 条件渲染 + disabled + title（裸 Button + confirmDelete 触发器） | ✅ 已门控 |
| 13 | 账号池：新增账号 | `:406-408` → `:459-469` | `opsADConfig.add` | 条件渲染 + disabled + title | ✅ 已门控 |
| 14 | 账号池：刷新 | `:397-405` → `:449-457` | —（读） | **读 → 不门控** | ✅ 已登记不门控 |

**第 14 行再次确认**：账号池的「刷新」是**只读入口**（`onClick` → `loadList()` / `loadStats()`，不落库），
与第 13 行的「新增账号」（`onClick` → `handleCreate`，落库）**分列为两行**，未合并 —— 与 PLAN NEW-03 的拆分要求一致。

### W4 施工图对照表 —— groups 页

| # | 入口 | PLAN 行号 → 落地行号 | 权限码 | 门控形态 | **结论** |
|---|---|---|---|---|---|
| 1 | 行「查看成员」 | `groups/index.tsx:378` → `:391-394` | —（读） | **读 → 不门控** | ✅ 已登记不门控 |
| 2 | 行「编辑」 | `:381` → `:396-405` | `opsADGroup.edit` | 条件渲染 + disabled + title | ✅ 已门控 |
| 3 | 成员行「移除」（`confirmDelete` 触发器） | `:414-427` → `:437-456` | `opsADGroup.edit` | 条件渲染 + disabled + title（**触发按钮前置条件渲染**，非 handler 内 return） | ✅ 已门控 |
| 4 | 工具栏「同步组」 | `:478-486` → `:508-520` | `opsADGroup.mappingSync` | 条件渲染 + disabled + title | ✅ 已门控 |
| 5 | 抽屉「添加成员」 | `:544` → `:578-589` | `opsADGroup.edit` | 条件渲染 + disabled + title | ✅ 已门控 |
| 6 | 弹窗内「添加」 | `:607-613` → `:648-666` | `opsADGroup.edit` | 条件渲染 + disabled + title | ✅ 已门控 |

### W4 施工图对照表 —— users 页

| # | 入口 | PLAN 行号 → 落地行号 | 权限码 | 门控形态 | **结论** |
|---|---|---|---|---|---|
| 1 | 行「禁用/启用」 | `users/index.tsx:521-526` → `:545-552` | `opsADUser.edit` | ActionButtons 的 `permission:` | ✅ 已门控 |
| 2 | 行「移动」 | `:527-532` → `:555-561` | `opsADUser.edit` | ActionButtons 的 `permission:` | ✅ 已门控 |
| 3 | 行「编辑」 | `:534-541` → `:564-569` | `opsADUser.edit` | ActionButtons 的 `permission:` | ✅ 已门控 |
| 4 | **工具栏「批量同步」** | **施工图未列** → `:659-679` | `opsADUser.edit` | 条件渲染 + disabled + title | ✅ 已门控（**施工图外补齐**，见下节 W4-外-①） |
| 5 | 行状态 Select（筛选条件） | `:595-615` | —（筛选，不是写入口） | **不门控** | ✅ 已登记不门控 |
| 6 | 移动弹窗的 TreeSelect 确认 | `:708-732` | —（随触发按钮不可达） | **不单独门控** | ✅ 已登记（见 W4-登记 ③） |

### W4 施工图外发现（1 处，已补齐 + 登记）

**① `ad-domain/users` 工具栏「批量同步」按钮**（落地行号 `users/index.tsx:659-679`）

- **实跑证据**：该按钮 `onClick` → `handleBatchSync` → `batchSyncADUsersDirect` → `POST /ad-domain/users/batch-sync`，
  把 AD 用户**写进系统用户表**（落库）→ 是**业务写入口**，不是只读。
- **后端硬证据**（`internal/api/v1/system/ad_domain_router.go:125-146`，原文注释）：

  ```go
  // 85-03 RBAC-01：读组保留 ANY-of，用户写操作走 strict 兄弟组（move/enable/
  // disable/batch-sync 均为编辑语义，归 ops:ad:user:edit）。
  userEdit.Use(middleware.RequireStrictPermissions([]string{string(permission.OpsADUserEdit)}, core))
  userEdit.POST("/:id/move", handler.MoveUser)
  userEdit.POST("/:id/enable", handler.EnableUser)
  userEdit.POST("/:id/disable", handler.DisableUser)
  userEdit.POST("/batch-sync", syncHandler.BatchSyncADUsers)
  ```

  即后端把 **move / enable / disable / batch-sync 四条写路由挂在同一个 strict 单码** `ops:ad:user:edit` 下
  → 前端四条写入口必须同码，否则会出现「前端放行 / 后端 403」的错配。
- **处置裁定**：**补门控**（归 `opsADUser.edit`），而非仅登记。依据：
  1. PLAN 自身的「结构性局限」节已写明「施工图遗漏的入口不会被页级扫描器捕获，入口级覆盖靠施工图 + 人工 review + 抽样测试兜底」
     —— 本波正是「人工 review」环节，且已拿到后端硬证据；
  2. W2 已有同型先例（assets「下载模板」、workstations「下载部门映射表」两处施工图外写入口**直接补齐**并写进 86-STATE）；
  3. W4 是本 phase 的最后一个铺开波次，**若只登记则该写入口将永久裸露**（RBAC-02 破口）；
  4. 改动面极小（1 个按钮），同页同码组，无跨域影响。
- **本条已在本节与 `86-VERIFICATION.md` 双登记**，请 Lead 复核该裁定（若 Lead 裁定「只登记不改」，回退成本 = 删 `users/index.tsx:659-679` 一段）。

### W4 偏离登记（PLAN 施工图与实跑的三处冲突，均已自行裁定并写明依据）

| 编号 | PLAN 原文 | 实跑事实 | 裁定 | 依据 |
|---|---|---|---|---|
| **W4-偏离 ①** | 第 11 行「账号池：编辑 / 解锁 / 启用停用 … 门控形态 = `permission:`」 | `AccountPoolTab.tsx` 的行操作是**裸 `<Button>`**（不是 `ActionButtons`），`ActionButton.permission` 位只存在于 `ActionButtons` 组件的 action 形参上 | 改为**页面层条件渲染 + disabled + title**（与该页其余裸 Button 一致） | 裸 Button 无处传 `permission:`；条件渲染的三态效果与 `permission:` 完全等价（ready 无权限→不渲染 / 未就绪→渲染但 disabled+提示） |
| **W4-偏离 ②** | 第 4 行「删除 … 门控形态 = 触发按钮前置条件渲染（`confirmDelete` 无 permission 位）」 | configs 页的删除是 **`ActionButtons` 的 action**（`onClick` → `confirmDelete`），**不是裸 Button** | 用 ActionButtons 内建的 `permission:` 位 | `ActionButtons` 的 `permission` 位本身即「逐按钮三态门控」（`ActionButtons.tsx:44-49`），ready 无权限→该 action 被 `visible` 过滤掉（不渲染）、未就绪→渲染但 disabled + 提示。**`confirmDelete.ts` 零改动**（PLAN 的硬约束「不改 confirmDelete.ts」成立） |
| **W4-偏离 ③** | 第 7~9 行「三个 Switch → `opsADConfig.edit` 决策（`visible===false` 时整个 Switch 不渲染）」 | 新增弹窗与编辑弹窗**共用同一个 Form**（`editingConfig ? "编辑AD配置" : "新增AD配置"`），三个 Switch 是**表单字段**，随弹窗提交生效 | 改为**随弹窗上下文归所属写动作**：新增弹窗 → `opsADConfig.add`，编辑弹窗 → `opsADConfig.edit` | 若一律归 `edit`：「有 `add` 无 `edit`」的用户点「新增AD配置」后**看不到 SSL / TLS / 自动同步三个字段** → 提交时三字段缺失 → **新建的配置被迫走明文 + 不自动同步 = 功能退化**。该口径与 86-01 `system/menu` 页的 `permFormStatus`（`editingMenu ? edit : add`）**逐字一致**，是同一 phase 内的既有先例；抽样测试用例 ⑥ 锁定该行为 |

### W4 旧式判定登记表（第二真相源）—— 6 处逐处定论 + 1 处已迁对照

> **登记 ≠ 改造**：本波 6 处**一律不改判定逻辑**（改了要么自造码、要么改变现有可见性行为）。
> 本表是「第二真相源债」的正式登记处，回补归口统一为**后端动作级拆分 phase**。

| # | 位置 | 判定用的码 | TS 清单有？ | 后端 `pkg/permission/config.go` 有？ | **定论** | 回补归口 |
|---|---|---|---|---|---|---|
| 1 | `src/pages/network/ports/index.tsx:130-132` | `network:port:write` | ❌ 无 `networkPort` 区块 | ✅ 有 `network:port:` 前缀 | **登记（不迁）**：迁它需新增 TS 码 → 违反「零新增码」；且后端 `network:port:write` 是组级单码，与逐动作拆分方向冲突 | 后端动作级拆分 phase（补 TS `networkPort` 区块 + 后端按动作拆 `network:port:{write→list/edit/...}`） |
| 2 | `src/pages/vdi/VirtualMachineList/index.tsx:46,52,109,929-935` | `vdi:vm:add/bind/remove/restart/start/stop/sync`（7 个） | ❌ 无 `vdi` 区块 | ❌ 无 `vdi:` 前缀 | **登记（不迁）**：码在后端**根本不存在** → 前端判的是一个**永远拿不到的码**（入口对所有人隐藏）→ 登记为「**疑似死判定**」显式债 | 后端先拆出 `vdi:vm:*` 动作码（后端也没有 → 需先有码） |
| 3 | `src/pages/asset/reconciliation/exceptions/index.tsx:85-86` | `asset:reconciliation:resolve` | ❌ | ❌ | **登记（不迁）**：同上（死判定） | 后端拆出 `asset:reconciliation:*` 动作码 |
| 4 | `src/pages/asset/reconciliation/fix-suggestion/index.tsx:64-67` | `asset:reconciliation:fix:{accept,reject,rollback}` | ❌ | ❌ | **登记（不迁）**：同上（死判定） | 后端拆出 `asset:reconciliation:fix:*` 动作码 |
| 5 | `src/pages/network/mac/history/MACHistoryPage.tsx:117-119` | `network:mac:export` | ❌ 无 `networkMac` 区块 | ✅ 有 `network:mac:` 前缀 | **登记（不迁）**：同 #1 | 后端动作级拆分 phase（补 TS `networkMac` 区块 + 后端按动作拆） |
| 6 | `src/pages/ad-domain/configs/ReconciliationTab.tsx:42,63-64` | `ops:ad:config:reconciliation:list` | ❌（`opsADConfig` 无该子码） | ❌ | **登记（不迁）**：该子组件是**读**门控（非写入口），保留判定；本波只把**同页的写入口**门控化 | 后端拆出 `ops:ad:config:reconciliation:*`（后端 `ad_domain_router.go:177` 现挂 `ops:ad:config:reconciliation:list` 只读码） |
| — | ✅ **已迁 1 处**：`src/pages/operations/workstations/LocationAliasDrawer.tsx` + `src/pages/operations/workstations/index.tsx`（`ops:location:alias:*` 在 TS 清单内） | — | ✅ `opsLocationAlias` 区块存在 | ✅ | **86-02 已迁为三态门控**，本波复核未被回退 | 已闭环 |

**为什么豁免清单里带「旧式直判」字样的恰为 **5** 条**：全站旧式直判共 6 处，但第 6 处
`ad-domain/configs/ReconciliationTab.tsx` 所在页 `src/pages/ad-domain/configs/index.tsx` 在本波**已门控**（有码页）
→ 按 86-00 的 R2「同一页面不得同时出现在已门控与豁免两侧」→ **不进豁免清单**（只在表里登记子组件）。
故豁免清单中带「旧式直判（第二真相源）」字样的条目恰为 **5** 条
（`network/ports`、`network/mac/history`、`vdi/VirtualMachineList`、`asset/reconciliation/exceptions`、`asset/reconciliation/fix-suggestion`）。

**实跑校验**：`grep -c "旧式直判" scripts/gating-exemptions.json` → **5**；
`node -e "…filter(e=>(e.reason||'').includes('旧式直判')).length"` → **5**；
`git diff --stat` 对 6 个旧式直判文件 → **空（零 diff）**。

### W4 其他登记项

**① `ReconciliationTab.tsx` 零 diff**：`ReconciliationTab.tsx` 与其既有测试 `ReconciliationTab.test.tsx` **零改动、全绿**
（本波只登记，不改判定逻辑）。该子组件的读门控仍按 `RECONCILIATION_PERM`（`ops:ad:config:reconciliation:list`）字面量直判。

**② 账号池码归属裁决（已落源码注释）**：`AccountPoolTab.tsx` 文件头注释写明「账号池**无独立 TS 码区块**（后端有
`ops:ad:config:account:list/edit/delete`，TS 未声明）→ 按 AD 配置域操作码 add/edit/delete 归属；后端 account 子码回补归后端拆分 phase」。
后端证据：`pkg/permission/config.go:531-538` + `ad_domain_router.go:66-93`（三组已 `RequireStrictPermissions` 逐动作单码）。
⚠️ 该归属是**前端侧就近归码**，与后端 account 子码**不等价**（后端 `account:edit` 与 `config:edit` 是两个不同码），
回补 TS 声明后应改回 `ops:ad:config:account:*` 精确码。

**③ 弹窗内确认点「随触发按钮不可达」的登记**：
`ad-domain/users` 的移动弹窗内确认（`TreeSelect onChange` → `handleMoveConfirm`，落地 `:708-732`）**未单独门控** —— 理由：
移动弹窗只能由行操作「移动」打开，而「移动」已挂 `opsADUser.edit` 的 `permission:`（ready 无权限 → 该 action 不渲染
⇒ 弹窗永不打开）。同理，configs / AccountPoolTab / groups 的弹窗 `onOk` 提交按钮（`handleSubmit` / `handleSubmit` 账号池 /
`handleUpdateGroup`）**未单独门控**，理由相同：**触发按钮已门控 ⇒ 弹窗不可达**。
（与 W1~W3 的既有口径一致：Excel 类入口门控点落在**触发按钮**，modal 容器保留无条件渲染。）

**④ 组写路由的后端缺口（备查，本波不改）**：实跑发现 `ad_domain_router.go` **未注册**以下前端在调用的写路由：

| 前端调用 | 后端路由 | 状态 |
|---|---|---|
| `addADGroupMember` → `POST /ad-domain/groups/:id/members/add` | 未注册 | ⚠️ 404 |
| `removeADGroupMember` → `POST /ad-domain/groups/:id/members/remove` | 未注册 | ⚠️ 404 |
| `syncADGroups` → `POST /ad-domain/groups/sync-by-config/:configId` | 未注册 | ⚠️ 404 |
| `updateADGroup` → `POST /ad-domain/groups/:id/update` | 已注册（`ops:ad:group:edit` strict） | ✅ |

即「组成员增删」与「按配置同步组」三个入口**当前在后端无路由**（前端调用恒 404）。
**与本 phase 门控无关**（门控只管前端入口可见性），本波**不改任何后端代码**，仅登记备查 → 归 85/87 线的后端拆分 phase。

**⑤ `useActionPermission` 的导入路径偏离（桶文件 → 直接模块路径）**：
本波 4 个 ad-domain 文件统一改为 `import { useActionPermission } from "@/components/shared/useActionPermission";`，
**不走** `@/components/shared` 桶文件。实跑依据：桶文件 re-export 了 `FloorPlanEditor` → `@react-three/fiber` → **three.js**，
引入它会让 `AccountPoolTab` 的动态 import 从 **~6.3s** 涨到 **>15s 直接超时**（`AccountPoolTab.render.test.tsx` /
`ad-domain/__tests__/modules.test.tsx` 两个既有测试由绿转红，实测已复现并修复）。
`useActionPermission` 本身零依赖（只依赖 `react` + `menuStore`），直接引模块路径更轻。
（注：其余页面仍走桶文件，属既有风格；本波只对本 phase 新增的 4 个文件采用直接路径。）

**⑥ 并行线噪音登记（Phase 90 线 B，非本波改动）**：W4 执行期间工作区存在 **Phase 90 线 B 的并行改动**：
`xingran-react-frontend/src/index.css`（100 行变更）与 `xingran-react-frontend/scripts/css-contract-check.js`（13 行变更）。
**这两个文件不属于本波改动面**，本波**未触碰**（红线 1：不得修改 `src/index.css` 与任何 `.module.css`），
且**不纳入本波 commit**。W4 的 `git diff --name-only -- src/ scripts/` 中这两项为并行线产物，已在本节登记。

### W4 测试用例表

**新增文件**：`src/pages/ad-domain/configs/__tests__/permission.gate.test.tsx`（**8 例**，全绿）

| # | 用例 | 覆盖路径 | 关键断言 |
|---|---|---|---|
| ① | 只持 `opsADConfig.list` | ready + 只有读码 | 「新增AD配置」「测试连接」「同步数据」「删除」**均不在 DOM**；「详情」（读）保留 |
| ④ | ready + 空权限集 | ready + 空 | 四个写入口**均不渲染**（与「未就绪」在数据上可区分） |
| ② | 未就绪（`permissionsLoaded=false`） | loading | 「新增AD配置」在 DOM 且 `disabled` + `title` 含「权限」；行操作 4 项全可见 → 收进「操作」下拉，菜单项 `.ant-dropdown-menu-item-disabled` > 0 且带 `title` 含「权限」 |
| ②-2 | 未就绪（`permissionsError` 非空） | error | 同上 |
| ③ | 持 `opsADConfig.test` | ready + 单写码 | 「测试连接」存在且 `disabled === false`；**「同步数据」「删除」「新增AD配置」仍不在 DOM**（逐码断言，不是一揽子） |
| ⑤ | 只持 `list`（无 `edit`） | HIGH-1 | 点「详情」打开抽屉（以抽屉标题 `AD 配置详情: 公司主AD域` 为锚点，避免「抽屉没打开」被误判成「按钮被门控掉」）→「编辑此配置」**不在 DOM** |
| ⑤-2 | 持 `opsADConfig.edit` | HIGH-1 正向 | 「编辑此配置」存在且 `disabled === false` |
| ⑥ | 只持 `opsADConfig.add`（无 `edit`） | 偏离 ③ | 点「新增AD配置」→ 新增弹窗内三个 Switch（使用SSL / 使用TLS / 启用自动同步）**均可见且未禁用** |

**专项类型检查**：临时 tsconfig `tsconfig.gate-test.check.json`（`extends ./tsconfig.app.json` + `types` 加 `vitest/globals`
+ 指定 include + `exclude: []`）跑 `npx tsc --noEmit` → **错误数 0**；该临时文件跑完即删（不入 commit）。

**既有测试零回归**：`npx vitest run src/pages/ad-domain` → **15 files / 49 tests 全绿**（含 `ReconciliationTab.test.tsx`、
`AccountPoolTab.render.test.tsx`、`modules.test.tsx` 等）。

**既有测试语义适配**：本波 **1 处**（未删任何断言、未改门控语义，仅补齐测试隔离）：

| 测试文件 | 适配内容 | 理由（根因，非「为了让测试变绿」） |
|---|---|---|
| `src/pages/operations/buildings/__tests__/permission.gate.test.tsx`（**W2 产出，非本波产出**） | 新增 `vi.mock("@/hooks/useDeptTree", …)` 一段（与同目录既有 `hooks.test.tsx:17` / `useDepartmentData.test.tsx:17` **完全同款**） | **该用例是环境依赖型 flaky，在 HEAD（499b2991）上即已假红**（本波用 `git stash` 把**全部**工作区改动（含 90 线并行改动）暂存后复跑，失败完全一致 → 证明与本波无关）。根因链：`buildings/index.tsx:107` 消费 `useDeptTree()` → `getDeptTree()`（`@/lib/dutyApi`）→ `@/lib/api` 的**真实** post → 开发机**有后端在跑**（实测 `:8080` / `:9000` LISTENING）时返回 `INVALID_TOKEN` → `@/lib/api` 失效处理链调用 `menuStore.clearMenus()` → `permissionsLoaded` 被重置为 `false` → 门控翻成「未就绪」→ 用例 ③ 的 `addBtn.disabled` 变 `true` → **假红**。即该用例的成败取决于「本机有没有后端在跑」。修法是切断真实网络请求（mock `useDeptTree`），**不删任何断言、不降任何断言** |

> 该修复的**归属说明**：`buildings` 页属 86-05 Task 5.3 的 8 页抽样清单之一，原计划在 W5 收口；本波提前修复的原因是
> 86-04 Task 4.5 的验收判据要求 `npx vitest run` **全绿**，且该红是 W4 开工前既已存在的红（不得留一个「红的 HEAD」给 W5）。
> 修复内容是**测试隔离**（与产品代码零关系），已逐条登记。

**其他既有测试语义适配**：**0 处**（ad-domain 3 页的既有测试全部零改动通过，未删任何断言）。

### W4 结构性局限复核（页级判定器 vs 入口级判定）

`check-gating-coverage.mjs` 对 ad-domain 3 页的判定：本波铺开后 3 页全部判「已门控」，**未覆盖清单为空 → 脚本 exit 0（转绿）**。

⚠️ **本节的页级绿灯 ≠ 入口级覆盖 100%**（第八节口径的第三次复现）：
本波**人工逐入口复核**（页 × 入口 × 权限码）时，发现了 **1 处施工图未列的写入口**（users「批量同步」，已补齐）。
这再次证明 `check-gating-coverage.mjs` **只判页级**（该页是否出现 `useActionPermission` / `permission:` + `PERMISSION_CODES`），
**不判单个入口是否已门控** → 施工图遗漏的入口不会被脚本捕获。
入口级覆盖仍只能靠 **施工图 + 人工 review + 抽样测试** 三层兜底。

### W4 闸门结果

| 闸门 | 结果 |
|---|---|
| `npm run check:gating` | **exit 0**（已门控 25 / 豁免 51 / 未覆盖 0） |
| `npx tsc --noEmit -p tsconfig.app.json` | **0 错误** |
| `npm run lint` | **0 problems**（0 error / 0 warning） |
| `npx vitest run src/pages/ad-domain` | **15 files / 49 tests 全绿**（含新增 8 例） |
| `npx vitest run`（全量） | **578 files / 4022 tests，0 failed**（首跑 2 failed → 均已定位并修复：① buildings 门控测试的**既有**环境依赖假红，已修；② `operations/rpa/tasks/__tests__/columns.test.tsx` 的 `await import("../columns")` **15s 超时**——单独复跑 **6.4s 通过**，属全量并行下的**机器负载产物**（`import` 总耗时 2651s），非真实回归，登记为噪音） |
| `git diff --stat -- src/constants/permissionCodes.ts` | **空（零改动）** |
| `git diff --stat -- src/components/shared/**` | **空（零改动）** |
| 6 处旧式直判文件 diff | **空（零改动）** |
| `grep -rn "useMenuStore" src/pages/operations/workstations/ --include=*.ts --include=*.tsx \| grep -v __tests__ \| wc -l` | **0**（86-02 的迁移未被回退） |

⚠️ **PLAN 的一处期望值与实跑不符（登记）**：86-04 Task 4.4 的 verify 命令写作
`grep -rn "useMenuStore" src/pages/operations/workstations/ | wc -l` 期望 **0**，实跑为 **2**。
实跑证据：2 处命中**全部**在 `src/pages/operations/workstations/__tests__/permission.gate.test.tsx`
（`import { useMenuStore } from "@/store/menuStore"` + `useMenuStore.setState({...})`）——
即 **86-02 自己新建的抽样测试**用于构造三态权限集的 harness，**不是旧式直判**。
排除 `__tests__` 后的业务源码命中数 = **0**（PLAN 的实质判据成立）。
原因：86-STATE 记录的「命中数 = 0」是在 86-02 **创建该测试文件之前**测得的，测试文件落地后计数自然变为 2。



---

## 十七、W5 收口（Plan 05 / 2026-10-29）

### 17.1 豁免清单定稿（R1~R6 逐条核对）

**结构校验（脚本实跑）**：

```
exemptions= 51  unique= 51
categories= {"no-permission-code":26, "legacy-two-source":5, "read-only":12, "non-route-page":8}
missing fields= 0
```

| 规则 | 判据 | 实跑结果 |
|---|---|---|
| **R1 悬空** | 豁免条目指向的页面必须存在于 76 页清单 | ✅ 脚本「悬空豁免」检查 0 命中 |
| **R2 重叠** | 同一页面不得同时在「已门控」与「豁免」两侧 | ✅ 脚本「门控/豁免冲突」检查 0 命中（已门控 ∩ 豁免 = 0） |
| **R3 无写入口** | `read-only` 类必须确实无业务写入口特征命中 | ✅ 12 条逐条复核；**`<Switch` 人工判定口子已执行** —— `operations/building-spaces-3d/index.tsx` 的 Switch 为**视图切换 Switch，非业务写**（`onChange={setUseWebGL}` 只切 WebGL/普通渲染，不落库、不触发后端写请求），其 `reason` 已注明该措辞 → **保留 `read-only` 豁免，未被 R3 强制改判**；其余 11 条均为「扫描器写入口特征零命中 + 人工复核无写向按钮」 |
| **R4 无码可证** | `no-permission-code` 类的码必须确实不在 `permissionCodes.ts` 任何区块中 | ✅ 26 条逐条 grep 核，0 条命中（后端有对应常量但前端无区块者归 `legacy-two-source`） |
| **R4′ 码键存在性** | 所有 `PERMISSION_CODES.<组>.<键>` 引用的键必须真实存在（TS2339 防线） | ✅ **全仓实跑 0 非法引用**：解析 `permissionCodes.ts` 得 28 个组、逐键比对全部 `PERMISSION_CODES.<组>.<键>` 引用 → **非法引用数 = 0**（尤其 `ops*.remove` —— 八组 ops 码只有 `delete`，实测零处 `.remove` 误用） |
| **R5 理由非空** | 每条 `reason` + `evidence` 非空 | ✅ missing fields = 0 |
| **R6 总量** | 豁免数 + 已门控数 = 76 | ✅ 25 + 51 = 76（脚本硬断言，恒等式成立） |

**豁免总数未变动**（51 → 51）→ `EXPECTED_EXEMPTION_COUNT` **无需改动**。

⚠️ **W4 的 5 条 `legacy-two-source` 理由做了措辞补强**（内容语义不变）：统一补上「含旧式直判（第二真相源），待后端动作级拆分 phase 统一收口」这半句，使 `scripts/check-exemption-reasons.mjs` 的理由格式校验一致（5 条 `legacy-two-source` 原本已含「等后端拆分 phase 统一收口到 PERMISSION_CODES 后再铺开三态门控」，但被该脚本判定为措辞不统一 → 补齐后仍走同一归口语义，**不新增豁免、不删豁免**）。

### 17.2 扫描脚本硬断言升级 + 挂 `lint:ci`

**`scripts/check-gating-coverage.mjs` 新增三条硬断言**（不改变判据语义，只把「软提示」升为「硬失败」）：

| # | 断言 | 失败条件 |
|---|---|---|
| ① | 覆盖率恒等式：已门控（不含豁免）+ 豁免 == `EXPECTED_PAGE_COUNT`（76） | `25 + 50 = 75 ≠ 76` → exit 1 |
| ② | 已门控覆盖完整：应门控页面数 == 已门控页面数 | 应门控 26 页中仅 25 页命中 → exit 1 |
| ③ | 未覆盖页面数 == 0 | 未覆盖 1 页 → exit 1 |

另有原有的**双向互斥**检查（豁免里挂着已门控页 → exit 1；未豁免且未门控 → exit 1）→ 白名单**不能靠无限扩张让脚本变绿**。

**红绿自检（Task 5.2 验收要求，必做）**：人为从豁免清单删除 `src/pages/knowledge/view/index.tsx` 一条 → 实跑输出：

```
未覆盖页面：1
覆盖率恒等式：已门控 25 + 豁免 50 = 75（断言 76） ✗
=== 未覆盖页面清单（1 页，需铺开三态门控）===
  ✗ src/pages/knowledge/view/index.tsx
  ✗ 豁免条数漂移：实测 50，断言 51 —— 页数增减须同批改 EXPECTED_EXEMPTION_COUNT 并在 86-STATE.md 说明理由
  ✗ 覆盖率恒等式不成立：已门控 25 + 豁免 50 = 75 ≠ 76（断言 EXPECTED_PAGE_COUNT）
  ✗ 已门控覆盖率不足：应门控 26 页中仅 25 页命中门控特征
  ✗ 未覆盖页面 1 页 —— 应门控页面必须全部接入三态门控
✗ 门控覆盖未达成：未覆盖 1 页 / 脚本错误 4 项 —— 退出码 1
```

→ **四条硬断言全部命中，exit 1** ✅；还原豁免清单后复跑 → **exit 0** ✅（红→绿链完整）。

**挂 `lint:ci`**：`package.json` 的 `lint:ci` 链追加 `&& node scripts/check-gating-coverage.mjs`（置于 `check-exemption-reasons` 之后、`check-hardcoded-colors` 之前）：

```
"lint:ci": "eslint . --max-warnings 0 && node scripts/assert-lint-budget.mjs && node scripts/check-exemption-reasons.mjs && node scripts/check-gating-coverage.mjs && node scripts/check-hardcoded-colors.mjs && node scripts/css-contract-check.js && node ../scripts/check-route-registry.mjs"
```

**`package.json` 的 `dependencies` / `devDependencies` 零 diff**（只加 scripts 一段）。

### 17.3 结构性局限登记（第八节口径的第四次复现 —— 这是登记，不是待办）

⚠️ **`check-gating-coverage.mjs` 是「页级」门控判定器 ⇒ 「入口级」覆盖靠施工图 + 人工 review + 抽样测试兜底，施工图遗漏的入口不会被脚本捕获**。

**本 phase 的四次实证**（同一局限在四个波次各暴露一次）：

| 波次 | 脚本绿灯 | 人工 review 发现的漏网入口 |
|---|---|---|
| 86-02 | 该域 8 页判「已门控」 | assets「下载模板」`index.tsx:639-654`、workstations「下载部门映射表」`index.tsx:834-836` 两处**导出类写入口**（数据外带面） |
| 86-03 | 该域 7 页判「已门控」 | network/discoveries 的 `ResultModal.tsx:50-60`「导入设备」（`POST /network/discoveries/{id}/import`，**落库**写入口） |
| 86-04 | ad-domain 3 页判「已门控」 | users「批量同步」写入口（施工图未列，已补齐） |
| **86-05（本波）** | **25 页全判「已门控」，脚本 exit 0** | 人工逐入口复核**未发现新的漏网入口**（导出类抽样复核见 17.4） |

⇒ **本 phase 不新增入口级扫描器**（超出范围，且 86-05 禁止新增 npm 依赖）。**该局限已在本文件与 `86-VERIFICATION.md` 各登记一次**（Task 5.2 第 5 条要求）。

### 17.4 导出类入口抽样复核（ops 域，收口动作 —— 入口级覆盖的人工兜底）

对 `src/pages/operations/**` 逐页 grep 导出/下载类触发点（`downloadTemplate` / `exportMapping` / `excelApi.export` / `BatchExportModal` / `handleExport` / `ExportOutlined` / `DownloadOutlined`），逐个确认其权限码已接入 `decide` 决策：

| 页 | 导出类入口 | 权限码 | 复核结果 |
|---|---|---|---|
| `operations/assets/index.tsx` | 「下载模板」`:639-654` | `opsAsset.export` | ✅ 已接入 |
| `operations/workstations/index.tsx` | 「下载部门映射表」`:834-836` | `opsWorkstation.export` | ✅ 已接入（86-02 补齐项） |
| `operations/buildings/index.tsx` | Excel 导入/导出 | `opsBuilding.import` / `opsBuilding.export` | ✅ 门控点落**触发按钮**（非 modal 容器） |
| `operations/floors/index.tsx` | Excel 导入/导出 | `opsFloor.import` / `opsFloor.export` | ✅ 同上 |
| `operations/workstations/index.tsx` | Excel 导入/导出 | `opsWorkstation.import` / `opsWorkstation.export` | ✅ 同上 |
| `operations/info-points/index.tsx` | Excel 导入/导出 | `opsInfoPoint.import` / `opsInfoPoint.export` | ✅ 同上 |
| `operations/server-rooms/index.tsx` | Excel 导入/导出 | `opsServerRoom.import` / `opsServerRoom.export` | ✅ 同上 |
| `operations/room-devices/index.tsx` | Excel 导入/导出 | `opsRoomDevice.import` / `opsRoomDevice.export` | ✅ 同上 |
| `operations/dedicated-lines/index.tsx` | Excel 导入/导出 | `opsDedicatedLine.import` / `opsDedicatedLine.export` | ✅ 同上 |

**结论**：ops 域 **9 页**的导出/下载类入口**全部**已挂权限码并接入三态决策，**零漏网**。导出类「数据外带面」已被完整覆盖。

### 17.5 W5 闸门结果

| 闸门 | 结果 |
|---|---|
| `npm run check:gating` | **exit 0**（已门控 25 / 豁免 51 / 未覆盖 0 / 恒等式 `25 + 51 = 76` ✓） |
| `npm run lint:ci`（含新增 check:gating） | **exit 0**（eslint 0 problems → assert-lint-budget → check-exemption-reasons → **check-gating-coverage** → check-hardcoded-colors → css-contract-check ALL PASS → check-route-registry OK） |
| `npm run lint` | **0 problems**（0 error / 0 warning） |
| `npm run type-check` | **0 错误**（`tsc --noEmit -p tsconfig.app.json` exit 0） |
| `npx vitest run`（8 页抽样 + shared + user 既有） | **100 files / 680 tests 全绿** |
| `npx vitest run`（全量） | **578 files / 4022 tests 全绿**（exit 0） |
| `npm run build` | **✓ built in 45.55s**；vendor-react gzip **650.74 kB** ≤ 776 kB 基线 ✅ |
| `git diff --stat -- src/constants/permissionCodes.ts` | **空（零改动）** |
| `git diff --name-only \| grep -E "\.css$\|src/design-system/"` | **空（零交叠）** —— `src/index.css` 的改动属 90 线 commit `89c2f62d`，**不在 86 的改动面内** |
| `package.json` dependencies / devDependencies | **零 diff**（只加 scripts 一段） |

### 17.6 零交叠声明（与 90 线）

- 86 的改动面**仅限**：`src/pages/ad-domain/**`、`src/pages/operations/buildings/__tests__/**`、`scripts/check-gating-coverage.mjs`、`scripts/gating-exemptions.json`、`package.json`（仅 `scripts` 段）、`.planning/phases/86-frontend-gating/**`、`.planning/{ROADMAP,REQUIREMENTS}.md`。
- **未触碰** `src/index.css`、**未触碰**任何 `.css` / `.module.css`、**未触碰** `src/design-system/**` → 与 90 线（CSS/设计令牌）**零交叠**。
- 90 线的 `src/index.css` 改动已在 commit `89c2f62d` 中独立成 commit，**86 的任何 commit 均不含 CSS 文件** ⇒ **86 与 90 可完全并行，无文件锁**。

### 17.7 遗留（不阻塞 phase 完成）

| 项 | 为何人工 | 状态 |
|---|---|---|
| 51 条豁免的逐条移除 | 豁免的「回补时限」依赖**后端动作级拆分 / 权限码播种**（Phase 87）落地 → 属跨 phase 依赖 | ⏳ 移交 Phase 87（`86-STATE.md` 第十节「缺码待回补登记」已列全量清单） |
| 6 处旧式直判（第二真相源） | 直判码不在前端 `PERMISSION_CODES` → 迁移等于自造权限码，须后端先出码 | ⏳ 移交 Phase 87（`legacy-two-source` 5 条 + `ad-domain/configs/ReconciliationTab` 1 条） |
