---
phase: "86"
slug: "frontend-gating"
status: passed
nyquist_compliant: true
wave_0_complete: true
verified: "2026-10-29"
requirements_satisfied: 1
requirements_total: 1
---

# Phase 86 — VERIFICATION

> 本 phase 由 6 个 plan 逐波执行（W0 底座 → W1 system → W2 ops → W3 network → W4 ad-domain → W5 收口），每个 plan 一个原子提交。
> 证据来源：脚本实跑输出 + 组件测试实跑 + git commit 链 + `86-STATE.md`（W0~W5 完整落盘）。

---

## 一、验收矩阵（判据 + 证据）

| # | 闸门 | 判据 | 实测输出 | 判定 |
|---|---|---|---|---|
| 1 | 覆盖扫描 | `npm run check:gating` exit 0 | 已门控 **25** / 豁免 **51** / 未覆盖 **0** / 恒等式 `25 + 51 = 76` ✓ | ✅ |
| 2 | 覆盖率口径 | 已门控 **25** + 豁免 **51** = **76**；豁免每条带 `reason` + `evidence` + `category`；豁免与已门控页**双向互斥** | `exemptions=51 unique=51`；`categories={no-permission-code:26, legacy-two-source:5, read-only:12, non-route-page:8}`；`missing fields=0`；门控 ∩ 豁免 = **0** | ✅ |
| 3 | 组件测试（抽样） | **8 页**（system 2 / ops 2 / network 3 / ad-domain 1）三路径全绿 | `npx vitest run`（8 页 + shared + user 既有）→ **100 files / 680 tests 全绿** | ✅ |
| 4 | lint | `npm run lint` **0 error 0 warning** | **0 problems**（`--max-warnings 0`） | ✅ |
| 5 | type-check | `tsc --noEmit -p tsconfig.app.json` 零错 | **0 错误**（exit 0） | ✅ |
| 6 | 全量测试 | `npx vitest run` 全绿 | **578 files / 4022 tests 全绿**（exit 0） | ✅ |
| 7 | build | `tsc -b && vite build` ✓ | **✓ built in 45.55s**；vendor-react gzip **650.74 kB** ≤ 776 kB 基线 | ✅ |
| 8 | lint:ci | 整链 exit 0（**含新增的 check:gating**） | **exit 0** —— eslint → assert-lint-budget → check-exemption-reasons → **check-gating-coverage** → check-hardcoded-colors → css-contract-check `ALL PASS` → check-route-registry `OK 697/306/138` | ✅ |
| 9 | 零改动 | `src/constants/permissionCodes.ts` 零 diff | `git diff --stat` **空** | ✅ |
| 10 | 零交叠 | 本 phase diff 不含 `*.css` / `src/design-system/**` | `git diff --name-only \| grep -E "\.css$\|src/design-system/"` **空** | ✅ |
| 11 | 零依赖 | `package.json` dependencies / devDependencies 零 diff | **零 diff**（只加 `scripts` 一段） | ✅ |

---

## 二、RBAC-02 验收对照（1/1 Satisfied）

| REQ | 需求 | Plan | Status | 关键证据（已提交） |
|---|---|---|---|---|
| RBAC-02 | 前端 76 个业务页面的行操作 / 批量操作 / 工具栏写入口按**动作级权限码**三态渲染 | 86-00 ~ 86-05 | ✅ Satisfied | **25/25 有码页全部接入 `useActionPermission` 三态门控**（无权限不渲染 / 未就绪 disabled+提示 / 有权限可用），**51 页带理由 N/A 豁免** → 恒等式 `25 + 51 = 76`（脚本硬断言守护）。红→绿链完整：W0（红：已门控 2 / 未覆盖 23）→ W1（8/17）→ W2（16/9）→ W3（22/3）→ W4（25/**0**，转绿）→ W5（挂 `lint:ci` 变不可回退）。`check-gating-coverage.mjs` 三条新硬断言 + 红绿自检（人为删 1 条豁免 → 4 项错误 / exit 1 → 还原 → exit 0） |

---

## 三、四项口径更正登记（Lead 裁定）

| # | 原口径 | 实测更正 | 依据 |
|---|---|---|---|
| 1 | 业务页面 **76** 个 | ✅ **76** 确认（`EXPECTED_PAGE_COUNT` 实跑一致） | `check-gating-coverage.mjs` 页面清单实跑 |
| 2 | 起点 **14/76** 页面已接入 | ⚠️ **实为 2/76** —— 原「14」系把 `useMenuStore.permissions` 的**引用数**误计为 `useActionPermission` 的**接入数**；真实已接入仅 `system/user` + `network/devices` 两页 | 86-00 口径裁定（脚本实跑：`useActionPermission` 命中 2 页） |
| 3 | 「76/76 页面接入」 | ⚠️ **更正为「25/25 有码页接入 + 51 页带理由 N/A 豁免 = 76/76」** —— 语义从「页面数达标」改为「**每个需要门控的页都门控了，每个不需要的页都有可核实的豁免理由**」 | 86-00 口径裁定 + R1~R6 豁免定稿规则 |
| 4 | `PENDING_SEED_WHITELIST` **79** 条 | ⚠️ **以文件为准** —— 该清单在 Phase 87 的 `pkg/permission` 侧，实际条数以文件实跑为准，不在本 phase 断言 | 85-04 移交清单登记 |

**ROADMAP 更新**：Phase 86 的 `Plans: TBD` → `Plans: 6 plans（00, 01, 02, 03, 04, 05 —— 全部执行完成）`；SC1 的「76/76 页面接入（起点 14/76）」**保留原始措辞并追加更正说明**（不抹掉历史口径）。

**REQUIREMENTS 更新**：`:30` 的 `- [ ] **RBAC-02**` → `- [x]`；`:126` 映射表 `RBAC-02 | 86 | Pending` → `Complete`。

---

## 四、豁免清单定稿（R1~R6 逐条核对）

| 规则 | 判据 | 实测 | 判定 |
|---|---|---|---|
| R1 悬空 | 豁免页面必须存在于 76 页清单 | 脚本「悬空豁免」检查 **0 命中** | ✅ |
| R2 重叠 | 不得同时在「已门控」与「豁免」两侧 | 门控 ∩ 豁免 = **0** | ✅ |
| R3 无写入口 | `read-only` 类须确实无业务写入口 | 12 条逐条复核；**`<Switch` 人工判定口子已执行** —— `operations/building-spaces-3d` 的 Switch 为**视图切换 Switch，非业务写**（`onChange={setUseWebGL}`，不落库不触发后端写），`reason` 已注明该措辞 → **保留 `read-only`，未被 R3 强制改判** | ✅ |
| R4 无码可证 | `no-permission-code` 类的码确实不在 `permissionCodes.ts` | 26 条逐条 grep 核，**0 条命中** | ✅ |
| **R4′ 码键存在性（TS2339 防线）** | 所有 `PERMISSION_CODES.<组>.<键>` 引用的键必须真实存在 | **全仓实跑 0 非法引用**（解析 `permissionCodes.ts` 得 **28 个组**，逐键比对全部引用）→ 尤其 `ops*.remove`：八组 ops 码只有 `delete`，**实测零处 `.remove` 误用** | ✅ |
| R5 理由非空 | 每条 `reason` + `evidence` 非空 | **missing fields = 0** | ✅ |
| R6 总量 | 豁免数 + 已门控数 = 76 | **25 + 51 = 76**（脚本硬断言） | ✅ |

**豁免总数未变动**（51 → 51）→ `EXPECTED_EXEMPTION_COUNT` 无需改动。

---

## 五、红绿自检（硬断言有效性证明）

**操作**：人为从 `scripts/gating-exemptions.json` 删除 `src/pages/knowledge/view/index.tsx` 一条 → 实跑 `npm run check:gating`：

```
未覆盖页面：1
覆盖率恒等式：已门控 25 + 豁免 50 = 75（断言 76） ✗
=== 未覆盖页面清单（1 页，需铺开三态门控）===
  ✗ src/pages/knowledge/view/index.tsx
  ✗ 豁免条数漂移：实测 50，断言 51
  ✗ 覆盖率恒等式不成立：已门控 25 + 豁免 50 = 75 ≠ 76
  ✗ 已门控覆盖率不足：应门控 26 页中仅 25 页命中门控特征
  ✗ 未覆盖页面 1 页 —— 应门控页面必须全部接入三态门控
✗ 门控覆盖未达成：未覆盖 1 页 / 脚本错误 4 项 —— 退出码 1
```

→ **四条硬断言全部命中，exit 1** ✅ → 还原豁免清单后复跑 → **exit 0** ✅。**「绿」已升级为「不可回退」**。

---

## 六、⚠️ 结构性局限登记（能力边界，不是待办）

> **`check-gating-coverage.mjs` 是「页级」门控判定器 ⇒ 「入口级」覆盖靠施工图 + 人工 review + 抽样测试兜底，施工图遗漏的入口不会被脚本捕获。**

**本 phase 的四次实证**（同一局限在四个波次各暴露一次）：

| 波次 | 脚本判定 | 人工 review 发现的漏网入口 |
|---|---|---|
| 86-02 | 该域 8 页判「已门控」 | assets「下载模板」`:639-654`、workstations「下载部门映射表」`:834-836` 两处**导出类写入口**（数据外带面）→ 已补齐 |
| 86-03 | 该域 7 页判「已门控」 | network/discoveries `ResultModal.tsx:50-60`「导入设备」（`POST /network/discoveries/{id}/import`，**落库**写入口）→ 已由 `499b2991` 补齐 |
| 86-04 | ad-domain 3 页判「已门控」 | users「批量同步」写入口（施工图未列）→ 已补齐 |
| **86-05（本波）** | **25 页全判「已门控」，exit 0** | 人工逐入口复核**未发现新的漏网入口**（导出类抽样复核见第七节，ops 域 9 页零漏网） |

⇒ 本 phase **不新增入口级扫描器**（超出范围，且禁止新增 npm 依赖）。本登记已按 Task 5.2 第 5 条要求，在 `86-STATE.md`（§17.3）与本文件**各登记一次**。

---

## 七、导出类入口抽样复核（ops 域，人工兜底）

| 页 | 导出类入口 | 权限码 | 判定 |
|---|---|---|---|
| `operations/assets/index.tsx` | 下载模板 `:639-654` | `opsAsset.export` | ✅ |
| `operations/workstations/index.tsx` | 下载部门映射表 `:834-836` | `opsWorkstation.export` | ✅ |
| `operations/buildings/index.tsx` | Excel 导入/导出 | `opsBuilding.import` / `.export` | ✅ |
| `operations/floors/index.tsx` | Excel 导入/导出 | `opsFloor.import` / `.export` | ✅ |
| `operations/workstations/index.tsx` | Excel 导入/导出 | `opsWorkstation.import` / `.export` | ✅ |
| `operations/info-points/index.tsx` | Excel 导入/导出 | `opsInfoPoint.import` / `.export` | ✅ |
| `operations/server-rooms/index.tsx` | Excel 导入/导出 | `opsServerRoom.import` / `.export` | ✅ |
| `operations/room-devices/index.tsx` | Excel 导入/导出 | `opsRoomDevice.import` / `.export` | ✅ |
| `operations/dedicated-lines/index.tsx` | Excel 导入/导出 | `opsDedicatedLine.import` / `.export` | ✅ |

**结论**：ops 域 **9 页**导出/下载类入口全部已挂权限码并接入三态决策，**零漏网**；导出类「数据外带面」完整覆盖。门控点统一落**触发按钮**（非 modal 容器）。

---

## 八、既有测试语义适配登记（1 处）

| 测试文件 | 适配内容 | 根因（非「为了让测试变绿」） |
|---|---|---|
| `src/pages/operations/buildings/__tests__/permission.gate.test.tsx` | 新增 `vi.mock("@/hooks/useDeptTree", …)` 一段（与同目录既有 `hooks.test.tsx:17` / `useDepartmentData.test.tsx:17` **完全同款**） | **环境依赖型 flaky，在 HEAD 即已假红**（把全部工作区改动 stash 后复跑，失败完全一致 → 与本波无关）。根因链：`buildings/index.tsx:107` 消费 `useDeptTree()` → `getDeptTree()`（`@/lib/dutyApi`）→ `@/lib/api` **真实** post → 开发机有后端在跑（实测 `:8080`/`:9000` LISTENING）时返回 `INVALID_TOKEN` → `@/lib/api` 失效链调用 `menuStore.clearMenus()` → `permissionsLoaded` 重置为 `false` → 门控翻「未就绪」→ 用例 ③ 的 `addBtn.disabled` 变 `true` → **假红**。修法是切断真实网络请求，**未删任何断言、未降任何断言、未改门控语义** |

**其他既有测试语义适配**：**0 处**（ad-domain 3 页既有测试零改动通过）。

---

## 九、零改动 / 零交叠核验

| 项 | 核验命令 | 结果 |
|---|---|---|
| 权限码单一真相源 | `git diff --stat -- src/constants/permissionCodes.ts` | **空**（全程零新增权限码） |
| 与 90 线零交叠 | `git diff --name-only \| grep -E "\.css$\|src/design-system/"` | **空**（`src/index.css` 属 90 线 commit `89c2f62d`，不在 86 改动面） |
| 共享组件零改动 | `git diff --stat -- src/components/shared/**` | **空** |
| 6 处旧式直判文件 | `git diff`（对应文件） | **空**（零改动，移交 Phase 87） |
| 零新依赖 | `package.json` 的 `dependencies` / `devDependencies` | **零 diff**（只加 `scripts` 一段） |
| workstation 旧式直判未回退 | `grep -rn "useMenuStore" src/pages/operations/workstations/ --include=*.ts --include=*.tsx \| grep -v __tests__ \| wc -l` | **0** |

---

## 十、遗留（移交 Phase 87，不阻塞本 phase 完成）

| 项 | 为何人工/为何移交 | 归属 |
|---|---|---|
| 51 条豁免的逐条移除 | 豁免的「回补时限」依赖后端动作级拆分 / 权限码播种落地（跨 phase 依赖） | Phase 87（清单见 `86-STATE.md` 第十节） |
| 6 处旧式直判（第二真相源） | 直判码不在前端 `PERMISSION_CODES` → 迁移等于自造权限码，须后端先出码 | Phase 87（`legacy-two-source` 5 条 + `ad-domain/configs/ReconciliationTab` 1 条） |

---

## 十一、Git 范围

| Plan | Commit |
|---|---|
| 86-00 | `ff12f3da` |
| 86-01 | `b4ffbfa5` |
| 86-02 | `e98f469f` |
| 86-03 | `2d3ce848` |
| 86-03 补漏 | `499b2991` |
| 86-04 | `4a437667` |
| 86-05 | `397da415` |

**Phase 完成判定**：1/1 需求 Satisfied + 恒等式 `25 + 51 = 76` 被脚本硬断言守护 + 红绿自检证明「绿」不可回退 + 11 项闸门全绿 + 零改动/零交叠/零依赖三项核验通过 → **phase_complete: true**

**解锁**：Phase 87（动作级权限授予闭环）—— 后端逐动作权限点（Phase 85）已就绪、前端入口按动作级码渲染（Phase 86）已完成，F 型菜单播种与三层证据验收可立即开工。
