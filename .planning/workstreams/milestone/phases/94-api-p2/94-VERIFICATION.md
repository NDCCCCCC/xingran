---
phase: 94-api-p2
verified: 2026-09-06T04:26:00+08:00
status: passed
score: 4/4 must-haves verified
overrides_applied: 0
re_verification:
  previous_status: none
  previous_score: n/a
  gaps_closed: []
  gaps_remaining: []
  regressions: []
human_verification:
  - test: "裁决前端 type-check gate 空转问题：npm run type-check（裸 tsc --noEmit）因 solution-style tsconfig.json（files: [] + references）实际未检查任何文件，exit 0 恒成立（CI 同款命令同受影响）。建议在 Phase 95 closeout 范围将脚本改为 tsc --noEmit -p tsconfig.app.json（或 tsc --build），并处置由此暴露的唯一存量错误 src/pages/network/backups/hooks/useRestoreTask.ts:47（TS2345，Phase 93 commit a4bfc71 引入，Phase 94 未触碰该文件及其依赖）。"
    expected: "type-check 脚本真实遍历 src；存量错误修复或显式豁免登记后 gate 保持 exit 0"
    why_human: "gate 语义变更影响全部后续 phase 与 CI，涉及存量错误修复决策（范围超出 Phase 94 契约），需人裁决排期与处置方式"
  - test: "产品确认两处已登记的外观级行为变化可接受：(1) assetApi.excel.export 保存文件名由恒定『资产列表_<ts>.xlsx』变为优先取 content-disposition 后端英文名（asset_* 等，excel_handler.go:75 确实发送该头）；(2) excel/下载报告错误文案统一为 downloadFilePost 语义『下载失败: <默认文件名>』（原『导出失败』/『下载报告失败』）。新行为已被 opsApi.test.ts:548-559 与 download.test.ts 锁定。若需保留中文文件名，按 94-03 SUMMARY deferred 方案为 downloadFilePost 增加 ignoreContentDisposition 选项。"
    expected: "产品/用户接受新文件名与新文案，或决定追加 ignoreContentDisposition 选项"
    why_human: "文件名与报错文案是最终用户可见行为，自动化测试无法判断产品可接受性"
---

## Human Verification Resolution（2026-09-06，v1.29 归档时落档）

1. **type-check gate 空转** → resolved：95-02 D-03 修复线落地——`package.json` scripts.type-check 改 `tsc --noEmit -p tsconfig.app.json`（真实检查面 3701 文件）；存量错误 `useRestoreTask.ts:47` TS2345 一并修复（`result.data ?? null`）；CI type-check 步骤同被救活（run 34007103013 绿）。
2. **两处外观级行为变化** → resolved（D-04 裁决接受现状）：asset 导出文件名英文化 + 下载错误文案归一，零代码回退；opsApi.test.ts / download.test.ts 已锁新行为；`ignoreContentDisposition` 选项留 deferred 候选（v1.29 audit known gap 3）。

# Phase 94: 前端 API 工厂化 Verification Report

**Phase Goal:** 设计 `createResourceApi<T>()` 工厂函数，迁移 `src/lib/` 下 13 个 `*Api.ts` 到工厂模式；保持向后兼容。（SC 措辞已按 D-01 校准：提升现有 8 方法工厂）
**Verified:** 2026-09-06T04:26:00+08:00
**Status:** human_needed
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths

| # | Truth（ROADMAP SC 校准版） | Status | Evidence |
| --- | --- | --- | --- |
| 1 | SC-1: apiFactory.ts 工厂完整（8 方法 + 类型推导）+ types/apiFactory.ts 派生类型 | ✓ VERIFIED | `src/lib/apiFactory.ts` 实读：`createResourceApi<T>` 8 方法（list/get/create/update/delete/batch/statistics/searchOptions）齐全，create/update 参数均为 `Partial<CreatePayload<T>>`（:49/:53），statistics/searchOptions 保留 `res.data ?? {}` / `res.data ?? []`（D-09），值导入 `post` 自 `./api`、`import type CreatePayload`；`src/types/apiFactory.ts` 实读：`ServerGeneratedKeys` 恰 9 键（camelCase 4 + snake_case 3 + createdBy/updatedBy）+ `CreatePayload<T> = Omit<T, ServerGeneratedKeys>` 纯 type-only。D-02 编译期行为探针（本验证独立执行）：`api.create({ id: "x", name: "n" })` 在 `tsc --noEmit -p tsconfig.app.json` 下报 `error TS2353: ... 'id' does not exist in type 'Partial<CreatePayload<ProbeEntity>>'`（探针文件已删除） |
| 2 | SC-2: 13 个 *Api.ts 全部按迁移矩阵处置（3 对象迁移 + 5 扁平委托 + 5 KEEP），向后兼容（导出签名零变化、消费文件零改动） | ✓ VERIFIED | grep 实测：opsApi/rpaApi 中 `createCrudApi`/`CrudApiConfig` 定义 0（rpaApi:52 仅为删除说明注释）；`createResourceApi` 命中 opsApi 12 / rpaApi 10 / vdiApi 3 / workorder 4 / knowledge 3 / duty 2 / notice 2 / adDomain 4；D-06 五个 KEEP 文件（menuApi/profileApi/columnConfigApi/notificationConfigApi/assetApi）`createResourceApi` 均 0；D-07 wall(:315)/door(:349)/floorPlanText(:382) 三处 `batch(action, ids: string[])` 覆盖层原样；vdiApi `vmCrud` spread + `list: VMListParams`(:41)/`create: CreateVMRequest`(:45) OVERRIDE + `vdiServerCrud` spread；opsApi `export type { DropdownOption } from "./apiFactory"`(:32)；`git diff --name-only cc77e81..HEAD -- src` 共 16 文件全部位于 src/lib + src/types/apiFactory.ts，pages/components/store/hooks 改动数 = 0；invariants 测试 `EXPECTED_FILES` 写死 13 文件且实测通过 |
| 3 | SC-3: npm run type-check + lint + test 0 错误 | ✓ VERIFIED | 本验证进程独立实跑：`npm run type-check` exit 0；`npm run lint` 0 errors / 1389 warnings（存量基线）exit 0；`npx vitest run` 全量 **554 文件 / 3800 测试全部通过**。⚠ 附带发现（WARNING，非本 phase 缺陷）：裸 `tsc --noEmit` 受 solution-style tsconfig 影响**未检查任何文件**（见 human_verification #1）；用真实配置 `-p tsconfig.app.json` 复核：16 个 phase 文件 **0 错误**，唯一存量错误在 `pages/network/backups/hooks/useRestoreTask.ts:47`（Phase 93 a4bfc71 引入，涉事文件均不在 94 改动清单，可证前置） |
| 4 | SC-4: 前端覆盖率 ≥45.13%（不下降） | ✓ VERIFIED | 本验证进程独立实跑 `npm run test:coverage`（约 17 分钟）+ `bash .github/scripts/check-frontend-coverage.sh xingran-react-frontend/coverage/coverage-final.json .coverage-fe-floors`（仓库根基准）：**gate exit 0，45/45 目录达标**，`lib 90.48% >= 87.2% (922/1019 stmts)`（与 94-03 SUMMARY 报告值完全一致；模板删除使 lib 分母变小、占比上升，SC-4「不降」锚点由 floors gate 成立） |

**Score:** 4/4 truths verified

### D-01..D-14 决策 Honoring 复核

| 决策 | Status | Evidence |
| --- | --- | --- |
| D-01 提升既有 8 方法工厂 | ✓ | apiFactory.ts 头注释「自 opsApi.ts 私有 createCrudApi<T> 逐行提升」+ 8 方法形状；ROADMAP § Phase 94 `getByID` 计数 0、`~15` 计数 0（sed 范围实测）；REQUIREMENTS API-FACTORY-01 行含「提升 opsApi 既有 8 方法工厂」、API-FACTORY-04 行含「12 个」 |
| D-02 CreatePayload 派生类型 | ✓ | 9 键排除集 + TS2353 探针实证（见 Truth 1） |
| D-03 opsApi 单一权威消费 | ✓ | opsApi `createCrudApi`/`CrudApiConfig` 定义 0、`axios.create` 0、import createResourceApi |
| D-04 download.ts 单一权威 | ✓ | download.ts 5 导出齐全（blobAxios/extractFilenameFromBlobResponse/triggerBrowserDownload/downloadFile/downloadFilePost）、`timeout: 300000`、异步拦截器 `Bearer` 注入；opsApi excelApi.export(:265)/assetApi.excel.export(:558) 委托 downloadFilePost；rpaApi downloadReport 委托（:301），`fetch(` 0 命中、`getAuthHeaders` 0 命中 |
| D-05 扁平委托签名不变 | ✓ | noticeApi/workorderApi 实读委托形态（私有实例 + 一行委托 + 显式返回类型注解保留）；`git diff` 消费文件 0 改动；3800 测试绿 |
| D-06 三小文件不套 | ✓ | menuApi/profileApi/columnConfigApi `createResourceApi` 0 命中 |
| D-07 batch 签名保持 | ✓ | opsApi:315/349/382 三处 `(action, ids: string[])` 原样 |
| D-08 大文件能对上才委托 | ✓ | rpaApi 第二工厂删除（含 `CrudApiConfig<_T>` 死泛型）、scriptApi `{ ...scriptCrud, ... }` spread（:132/:138）；vdiApi SPREAD+OVERRIDE 按矩阵 |
| D-09 statistics/searchOptions 内联 | ✓ | apiFactory.ts:66-83 内联 + `?? {}` / `?? []` 原样 |
| D-10 模板清零定性 | ✓ | D-12 扫描实测（硬档基线 + 白名单等值锁全绿）；基线非 0 项均为矩阵显式 KEEP（已在偏离 #2 登记为 allowedResidues 等值锁，强于单向 >0 fail） |
| D-11 契约测试 | ✓ | apiFactory.test.ts（20 expect：路径八连含 dropdown-options/statistics/batch 合并、解包空回退）+ download.test.ts（Bearer 注入、`%E6%A5%BC%E5%AE%87.xlsx` URL 编码、timeout 300000 锁值）实跑 20/20 绿 |
| D-12 扫描防线 | ✓ | apiFactory.invariants.test.ts 实读：ts.createSourceFile AST、范围仅 src/lib、EXEMPT 三文件、HARD_ALLOWED 等值锁（opsApi 4/rpaApi 0/vdiApi 1 各附理由）+ WARNING_WHITELIST 10 文件等值锁 + 13 文件清单写死断言；WR-02 VariableDeclaration 补强分支在位（:203-215，commit 3055ff5）。**独立红绿演练**：向 vdiApi.ts 注入 `__drillTempDelete` 模板 → 硬档测试转红（定位 `vdiApi.ts:175 __drillTempDelete()`）→ git checkout 还原 → 4/4 复绿 |
| D-13 CLAUDE.md Convention | ✓ | CLAUDE.md:432 `### Frontend API Factory Convention` 三段式（单一权威 8 方法 + Rules 含 D-12 守护/KEEP 例外/纯透传约束 + Migration status）；§ Frontend API Calling 含 createResourceApi 正例（:530-532） |
| D-14 测试基本不动 | ✓ | `git diff --name-only cc77e81..HEAD` 中 13 个既有 .test.ts 仅 2 个变更且均为登记项：opsApi.test.ts（+25/−21 小幅适配，D-14 预告范围）、adDomainApi.test.ts（:302 单条断言随 ：501 bugfix 修正，diff 实读确认） |

### Required Artifacts

| Artifact | Expected | Status | Details |
| -------- | -------- | ------ | ------- |
| `xingran-react-frontend/src/lib/apiFactory.ts` | createResourceApi 8 方法 + CrudApiConfig + DropdownOption | ✓ VERIFIED | 85 行实读核对；被 8 个文件 import 消费 |
| `xingran-react-frontend/src/types/apiFactory.ts` | ServerGeneratedKeys + CreatePayload（type-only） | ✓ VERIFIED | 35 行实读核对；9 键并集 |
| `xingran-react-frontend/src/lib/download.ts` | blob 五导出 + 300000 超时 + Bearer 拦截器 | ✓ VERIFIED | 92 行实读核对；被 opsApi/rpaApi 消费 |
| `xingran-react-frontend/src/lib/apiFactory.test.ts` | D-11 工厂契约测试 | ✓ VERIFIED | 实跑绿，断言非恒真（toHaveBeenNthCalledWith 逐方法） |
| `xingran-react-frontend/src/lib/download.test.ts` | D-11 下载链测试（含 D-11 组 6 report 链路） | ✓ VERIFIED | 实跑绿 |
| `xingran-react-frontend/src/lib/apiFactory.invariants.test.ts` | D-12 双档扫描 | ✓ VERIFIED | 266 行实读 + 实跑绿 + 独立红绿演练转红/复绿 |
| `xingran-react-frontend/src/lib/{opsApi,rpaApi,vdiApi}.ts` | 对象形态迁移（D-03/D-04/D-07/D-08） | ✓ VERIFIED | grep + 实读核对（无私有工厂/无 axios.create/无裸 fetch/DropdownOption re-export） |
| `xingran-react-frontend/src/lib/{workorder,knowledge,duty,notice,adDomain}Api.ts` | 扁平 cluster 委托（D-05/D-08） | ✓ VERIFIED | 实读委托形态；adDomain `withDefaultPagination` 10 处命中（3 处委托 + helper 留守）、`/delete}` 0 命中 |
| `CLAUDE.md` | D-13 Convention 段 | ✓ VERIFIED | :432 段落三段式齐备 |
| `.planning/workstreams/milestone/ROADMAP.md` + `.planning/REQUIREMENTS.md` | D-01 措辞校准 | ✓ VERIFIED | § Phase 94 `getByID`==0 且 `~15`==0；REQUIREMENTS 01 行「8 方法」、04 行「12 个」 |

### Key Link Verification

| From | To | Via | Status | Details |
| ---- | --- | --- | ------ | ------- |
| apiFactory.ts | api.ts | 值导入 post | ✓ WIRED | `import { post } from "./api"`（:15） |
| apiFactory.ts | types/apiFactory.ts | import type CreatePayload | ✓ WIRED | :17 |
| download.ts | utils/authHelpers.ts | 拦截器异步 getAccessToken | ✓ WIRED | :14/:29，download.test.ts 锁 Bearer 注入 |
| opsApi.ts | apiFactory.ts | 11 资源工厂调用 | ✓ WIRED | 12 处 createResourceApi 命中 |
| opsApi.ts | download.ts | blob 四件套消费 | ✓ WIRED | :7 import；downloadFile×2 + downloadFilePost×2 调用 |
| rpaApi.ts | apiFactory.ts / download.ts | spread 实例 + downloadReport 委托 | ✓ WIRED | :7/:132/:138/:301 |
| vdiApi.ts | apiFactory.ts | vmCrud/vdiServerCrud spread | ✓ WIRED | :31/:34/:142/:145，list/create OVERRIDE 保留 |
| 扁平 5 文件 | apiFactory.ts | 私有实例 + 一行委托 | ✓ WIRED | 每文件 import + 实例 + 委托调用（noticeApi/workorderApi 实读抽样） |
| invariants.test.ts | 13 个 *Api.ts | createSourceFile AST 扫描 | ✓ WIRED | 实跑绿 + 红绿演练转红实证 |
| CLAUDE.md | apiFactory.ts | Convention 段锁定权威路径 | ✓ WIRED | :432/:452 |

### Data-Flow Trace (Level 4)

| Artifact | Data Variable | Source | Produces Real Data | Status |
| -------- | ------------- | ------ | ------------------ | ------ |
| apiFactory.ts | post 请求链 | ./api 主实例（SM2+SM4/401 拦截器） | 是（既有传输层零改动） | ✓ FLOWING |
| download.ts | blobAxios | axios.create + VITE_API_BASE_URL + 异步 token | 是（契约测试锁定注入/超时/文件名链） | ✓ FLOWING |

（本 phase 为纯数据层等价替换，无 UI 渲染型 artifact；wire 等价性由 94-REVIEW 逐委托点 diff 审计 + 3800 测试双重锁定。）

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
| -------- | ------- | ------ | ------ |
| 契约+扫描测试 | `npx vitest run src/lib/{apiFactory,download,apiFactory.invariants}.test.ts` | 3 文件 20/20 绿 | ✓ PASS |
| 全量回归 | `npx vitest run` | 554 文件 / 3800 测试全绿 | ✓ PASS |
| type-check（项目 gate） | `npm run type-check` | exit 0（gate 空转，见 WARNING） | ✓ PASS |
| type-check（真实配置复核） | `npx tsc --noEmit -p tsconfig.app.json` | phase 16 文件 0 错误；1 处 Phase 93 存量错误 | ✓ PASS |
| lint | `npm run lint` | 0 errors / 1389 存量 warnings，exit 0 | ✓ PASS |
| 覆盖率 gate | `npm run test:coverage` + `bash .github/scripts/check-frontend-coverage.sh …` | exit 0；45/45 dirs；lib 90.48% ≥ 87.2 | ✓ PASS |
| D-02 编译期约束 | 临时探针 `api.create({ id: … })` + `tsc -p tsconfig.app.json` | TS2353 报错指向 'id'（探针已删） | ✓ PASS |
| ROADMAP 措辞 | sed § Phase 94 + grep `getByID` / `~15` | 计数均 0 | ✓ PASS |

### Probe Execution

| Probe | Command | Result | Status |
| ----- | ------- | ------ | ------ |
| D-12 红绿演练（独立复跑，不采信 SUMMARY 叙述） | vdiApi.ts 追加 `__drillTempDelete` → `npx vitest run src/lib/apiFactory.invariants.test.ts` → `git checkout` 还原 | 注入即红（`vdiApi.ts:175 __drillTempDelete()` 定位）；还原后 4/4 绿 | PASS / PASS |

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
| ----------- | ---------- | ----------- | ------ | -------- |
| API-FACTORY-01 | 94-01 | 提升 opsApi 既有 8 方法工厂为共享 createResourceApi<T>(config) | ✓ SATISFIED | apiFactory.ts 8 方法实读 + 契约测试 + D-02 探针 |
| API-FACTORY-02 | 94-01 | 新建 src/lib/apiFactory.ts + types/apiFactory.ts | ✓ SATISFIED | 两文件存在且实质（85 行/35 行），CreatePayload 编译期生效 |
| API-FACTORY-03 | 94-02 | opsApi 迁移工厂模式（保留同名导出、向后兼容） | ✓ SATISFIED | 私有工厂定义 0；同名导出保留；27 消费文件 0 改动（git diff 实证） |
| API-FACTORY-04 | 94-02/94-03 | 其余 12 个文件按迁移矩阵三态处置 | ✓ SATISFIED | 2 对象 + 5 委托 + 5 KEEP 逐一 grep/实读核对；invariants 13 文件清单锁 |
| API-FACTORY-05 | 94-03 | 三件套全过 + 覆盖率不降 | ✓ SATISFIED | 三件套实跑 exit 0；coverage gate exit 0（45/45 dirs） |

**Orphaned requirements:** 0（REQUIREMENTS.md Traceability 表 Phase 94 ↔ API-FACTORY-01..05 双向对齐，无未认领项）

**记账备注（Info，非代码缺口）:** `.planning/REQUIREMENTS.md` 中 API-FACTORY-01/02/03 复选框仍为 `- [ ]`（04/05 已勾选），措辞校准已由 commit 1771e1d 落地但三个已完成项的勾选状态漏更；workstream ROADMAP Progress 表 Phase 94 行亦为「Pending 0/3」存量状态。两者均属 orchestrator 集中记账范围，建议随本报告一并翻正。

### 已登记偏离的合理性复核（goal-backward 追问项）

| 偏离 | Status | 复核结论 |
| --- | --- | --- |
| delete 单参保留（94-03 偏离 #1） | ✓ 合理 | plan 的 DELEGATE 清单与其自身「.test.ts 零改动」红线互斥，executor 以「测试即规约」裁决；workorderApi:530/585 源码注释登记 + D-12 白名单带理由计数；仅 3 处既有测试本就断言 `{}` body 的 delete（deleteNotice/deleteOUGroupMapping/deleteMapping）委托，逻辑自洽 |
| D-12 硬档基线非 0（allowedResidues 等值锁，偏离 #2） | ✓ 合理 | plan「期望命中 0」被同 plan 引用的 KEEP 矩阵证伪（opsApi 4 + vdiApi 1 均为矩阵显式 KEEP）；等值锁（双向 fail）强于单向断言，镜像 cache_invariants_92_test.go，白名单逐项理由注释经实读确认 |
| WR-01/IN-03 外观级变化补登（commit 3055ff5） | ✓ 合理 | 94-03 SUMMARY 偏离 #6 补登 + 文件名变化已被测试锁定新行为；产品接受度转 human_verification #2 |
| WR-02 扫描盲区修复（commit 3055ff5） | ✓ 已修 | VariableDeclaration 分支在位（invariants :203-215），且本验证红绿演练证明检测力真实 |

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
| ---- | ---- | ------- | -------- | ------ |
| （16 个 phase 文件扫描 TBD/FIXME/XXX/TODO/HACK/PLACEHOLDER/空实现） | — | 0 命中 | — | 无 |

### Warnings（非 gap，不阻断）

1. **前端 type-check gate 空转（项目级前置缺陷，非 94 引入）**：`package.json` 的 `type-check` = 裸 `tsc --noEmit`，而根 `tsconfig.json` 为 solution-style（`files: []` + references）→ 实际检查 0 个文件、恒 exit 0（CI ci.yml:164 同受影响）。实证：探针文件在 src 下时裸 tsc 报 0 错误，`-p tsconfig.app.json` 报错。真实配置下全仓仅 1 处错误：`src/pages/network/backups/hooks/useRestoreTask.ts:47`（Phase 93 commit a4bfc71 引入；涉事文件 useRestoreTask.ts、../types、lib/api.ts 均不在 94 的 16 文件改动清单内，可证前置存量）。**Phase 94 的 16 个交付文件在真实配置下 0 类型错误**，且向后兼容证据不依赖该 gate（3800 测试 + 消费文件 0 diff + review 逐委托点 wire 审计三重独立支撑）。处置建议已列 human_verification #1。
2. **REQUIREMENTS 勾选状态漏更**（见 Requirements Coverage 记账备注）。

### Human Verification Required

### 1. 前端 type-check gate 空转的处置决策

**Test:** 将 `xingran-react-frontend/package.json` 的 `type-check` 改为 `tsc --noEmit -p tsconfig.app.json`（或 `tsc --build`），跑一次并处置 `src/pages/network/backups/hooks/useRestoreTask.ts:47` 的 TS2345 存量错误。
**Expected:** 真实遍历 src 后 gate 可红可绿；存量错误修复或显式豁免后保持 exit 0。
**Why human:** gate 语义变更波及全部后续 phase 与 CI 红绿，存量错误修复属 Phase 94 范围外的新决策（建议纳入 Phase 95 closeout「所有 gate 全绿」项下）。

### 2. 两处外观级行为变化的产品确认

**Test:** 实际触发 asset excel 导出与 RPA 报告下载，确认保存文件名（后端英文名优先）与失败文案（「下载失败: …」）可接受。
**Expected:** 产品接受；或决定为 downloadFilePost 增加 ignoreContentDisposition 选项恢复中文默认名。
**Why human:** 文件名/文案为最终用户可见行为，测试只能锁定现状、无法判断商业可接受性。

---

## Gaps Summary

无 gap。Phase goal 的四个成功标准全部在代码库实测成立：工厂单一权威落地且 8 方法形状与提升源 1:1（含 D-02 编译期约束实证）；13 个 *Api.ts 按 3+5+5 矩阵全部处置且消费文件零改动（git diff 实证）；三件套 gate 与覆盖率 gate 在本验证进程独立复跑全绿；D-01..D-14 全部被 honoring，4 项已登记偏离经复核均合理且可审计（独立 commit + 源码注释 + 白名单理由）。D-12 防线的检测力经独立红绿演练证实。状态定为 human_needed 仅因两项需要人裁决/确认的事项（gate 空转处置、外观级变化产品确认），二者均不否定 phase goal 达成。

---

_Verified: 2026-09-06T04:26:00+08:00_
_Verifier: Claude (gsd-verifier)_
