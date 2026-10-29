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

落盘位置：`ruoyi-react-frontend/scripts/gating-exemptions.json`（结构 `{ "exemptions": [ { page, category, reason, evidence } ] }`，每条 **reason + evidence 必填非空**）。

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

产物：`ruoyi-react-frontend/scripts/check-gating-coverage.mjs`（**零依赖**，只用 `node:fs` / `node:path` / `node:url`）。

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

- `ruoyi-react-frontend/package.json` 新增一行：`"check:gating": "node scripts/check-gating-coverage.mjs"`；
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

- 86-00 改动面**仅限**：`src/pages/**`（W1~W4，本 wave 为 0）、`ruoyi-react-frontend/scripts/check-gating-coverage.mjs`、`ruoyi-react-frontend/scripts/gating-exemptions.json`、`ruoyi-react-frontend/package.json`（仅 `scripts` 段一行）、本文件。
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
