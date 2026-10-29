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

