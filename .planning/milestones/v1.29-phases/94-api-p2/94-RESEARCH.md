# Phase 94: 前端 API 工厂化 - Research

**Researched:** 2026-09-05
**Domain:** TypeScript 泛型工厂重构（`src/lib/` 13 个 `*Api.ts` 收敛到 `createResourceApi<T>()`）+ blob 下载基建归一
**Confidence:** HIGH（全部结论基于对本仓库源码的逐文件通读 + 项目自带 tsc 5.9.3 实测探针验证，无外部依赖引入）

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions（D-01..D-14，全部锁定，研究不探索替代方案）

- **D-01: 提升现有工厂，行业对齐** — 把 opsApi 私有 `createCrudApi<T>` 提升到 `src/lib/apiFactory.ts`，8 方法为核（list/get/create/update/delete/batch/statistics/searchOptions），与 react-admin/refine 核心五方法 1:1 对齐。ROADMAP 7 方法（`getByID`/`import`/`export`）措辞按现实校准：import/export 不进工厂核心、`getByID` 改名不采用（避免 37 个消费 opsApi 文件波及）
- **D-02: 派生类型强化** — `src/types/apiFactory.ts` 定义派生类型助手（`CreatePayload<T> = Omit<T, 'id' | 'createdAt' | 'updatedAt' | 'deletedAt'>` 形态，具体排除集 researcher 定），create/update 用它替换 `Partial<T>`，误传 id/时间戳编译期报错
- **D-03: 单一权威** — opsApi 私有 `createCrudApi` 定义删除，opsApi 统一 import apiFactory.ts
- **D-04: 独立 download.ts** — blobAxios / extractFilenameFromBlobResponse / triggerBrowserDownload / downloadFile 迁入新共享文件（最终命名 planner 定）；opsApi 与 rpaApi 都改用；excelApi/deptApi/assetApi.excel 导出位置不变、内部委托；rpaApi.downloadReport 裸 fetch 重复顺手消灭（白得 5min 超时防护）
- **D-05: 内部委托签名不变** — 9 个扁平文件中能对上工厂方法的函数内部改一行委托工厂实例；异构函数（publishNotice/generateSchedule/swapDuty 等）原样保持。导出签名零变化 → 100+ 消费文件零改动
- **D-06: 非 CRUD 小文件不套工厂** — menuApi / profileApi / columnConfigApi 保持现状
- **D-07: batch 签名保持现状** — wall/door/floorPlanText 三处 `batch(action, ids: string[])` 不与工厂核心 `batch(action, data)` 统一
- **D-08: 大文件「能对上才委托」统一适用** — adDomainApi/workorderApi 只把标准 CRUD 形状函数接上工厂；rpaApi 的 scriptApi 接入，taskApi/workerApi/executionApi（纯异构）保持；vdiApi 走 floorApi spread 模式
- **D-09: statistics/searchOptions 保持工厂内联**
- **D-10: 验收锚点 = 模板清零定性** — 所有 `*Api.ts` 不再有与工厂方法同构的手写 CRUD 函数体；不设 LOC 硬数字
- **D-11: 专门契约测试** — 新建 apiFactory.test.ts（锁路径拼接、泛型类型推导、CreatePayload 排除生效、statistics/searchOptions 解包）+ download.test.ts（锁 blob 下载链）
- **D-12: 扫描测试防线** — 匹配 `*Api.ts` 中手写 CRUD 五件套模板，发现即 fail（或 warning，planner 定）
- **D-13: CLAUDE.md Convention 段** — 收口时新增「前端 API 工厂 Convention」段 + 修订 § Frontend API Calling
- **D-14: 13 个 `.test.ts` 基本不动** — 仅 opsApi.test.ts 等涉及内部结构调整的小幅适配

### Claude's Discretion（planner/researcher 定，无需再问用户）
- `CrudApiConfig` 配置面扩展形态；apiFactory.ts / download.ts 最终命名与内部布局
- `CreatePayload<T>` 排除字段集精确清单
- 各扁平文件工厂实例命名与 basePath 常量提取方式
- 扫描测试（D-12）检测模式（正则 vs AST）与 fail/warning 级别
- 契约测试用例分组与覆盖清单；3 个 plan 间分组微调（保持 94-01→94-02→94-03 pilot→批量节奏）
- opsApi.test.ts 适配幅度与 rpaApi scriptApi 迁移的测试影响评估
- GET vs POST 端点细节保持各文件现状（不改后端契约）

### Deferred Ideas (OUT OF SCOPE)
- React Query / queryKeys 与工厂集成联动（queryKeys.ts 不动）
- excelApi entityType 二级工厂化重构（entityType 参数化形态保持）
- `batch` 签名统一（wall/door/floorPlanText 的 `(action, ids)`）
- ESLint no-restricted-syntax 机械防线（被 D-12 扫描测试方案替代）
- `getByID` 命名（已拒绝，D-01）
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description (REQUIREMENTS.md 原文) | Research Support |
|----|-----------------------------------|------------------|
| API-FACTORY-01 | 设计 `createResourceApi<T>(basePath, resourceName)` 工厂函数（7 个 CRUD 方法） | D-01 校准为提升 opsApi.ts:49 现有 8 方法工厂；完整方法体见本文 § 工厂最终形态；「7 方法」按 CONTEXT 不采纳 import/export/getByID |
| API-FACTORY-02 | 新建 `src/lib/apiFactory.ts` + 类型定义 `src/types/apiFactory.ts` | 落点确认两文件均不存在（全新创建）；`CreatePayload<T>` 排除集经 tsc 实测验证（§ D-02 设计）；必须处理 snake_case 时间戳（vdi 域） |
| API-FACTORY-03 | opsApi.ts (buildingApi/floorApi/workstationApi/assetApi 等) 迁移到工厂模式（保留同名导出） | 11 资源已在用私有工厂，动作 = 删私有定义 + import（D-03）；blob 基建迁 download.ts（D-04）；27 个生产消费文件零改动可达成 |
| API-FACTORY-04 | 其他 ~10 个 `*Api.ts` 逐个迁移（低风险优先） | § 逐文件迁移矩阵给出 13 文件每个函数的 DELEGATE/SPREAD+OVERRIDE/KEEP 判定与理由；低风险排序建议在 § Plan 间分组 |
| API-FACTORY-05 | 三件套全过 + 前端覆盖率不降（基线 45.13%） | § Validation Architecture 给出完整 gate 链（type-check/lint/vitest/coverage floors 脚本）；覆盖率门机制已核实（.coverage-fe-floors + check-frontend-coverage.sh） |
</phase_requirements>

## Summary

本 phase 的本质是**提升而非设计**：`opsApi.ts:49-97` 的私有 `createCrudApi<T>`（8 方法，11 资源生产验证）+ `rpaApi.ts:51-79` 的第二份私有 `createCrudApi<T>`（5 方法，7 资源在用）——**仓库里实际存在两份平行工厂定义**（CONTEXT 只点名了 opsApi 一份，这是本研究的重要新发现）。94-01 的动作是把两份合并为一个共享权威 `src/lib/apiFactory.ts`，对齐 Phase 92 base 包「单一权威」哲学。全部 13 个文件已逐文件通读，产出逐函数迁移矩阵（§ 迁移矩阵）：能对上的委托/接入，对不上的（HTTP 动词不同、请求类型解耦、返回类型带自定义 payload）按 D-05/D-08「能对上才委托」保持。

三个被 tsc 5.9.3 实测验证（非推断）的关键类型学事实决定了实现细节：(1) **interface 变量不可赋给 `Record<string, unknown>` 参数**——工厂 list 的现状参数签名会拒绝 `VMListParams` 等接口类型变量（`VirtualMachineList/index.tsx:191` 实锤会炸）；(2) **Omit 保留必选性**——`CreatePayload<T>` 会把 `Building.orgId` 等业务必选字段变成必传，严格版会影响 ~13 处 create + ~12 处 update 调用点（全部已定位）；(3) **camelCase-only 排除集对 vdi 域 snake_case 实体（`created_at`/`updated_at`）无效**。三者均有明确解法（§ Common Pitfalls + § D-02 设计）。

**Primary recommendation:** 94-01 先落 `apiFactory.ts`（含 D-02 强化类型）+ `download.ts` + 两个契约测试；94-02 迁 opsApi（删私有工厂 + blob 基建迁出，纯内部等价替换，风险最低）+ rpaApi（删第二份私有工厂 + scriptApi 接入 + downloadReport 迁 download.ts）；94-03 按矩阵低风险顺序迁 vdiApi → 扁平文件 clusters（workorder categories/periodic → knowledge → duty pools → notice → adDomain ou-group-mappings/mappings），D-12 扫描测试最后落（硬档 = 已迁移文件，warning 档 = 文档化 KEEP 名单），收口 D-13。

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| CRUD 端点封装（list/get/create/update/delete/batch） | 前端 lib 层（apiFactory.ts） | — | 纯前端请求拼装，后端契约零改动 |
| 请求加密/token 刷新/错误拦截 | lib 层（api.ts，**零改动**） | — | SM2+SM4 与 401 重放已在 axios 拦截器，工厂是其上层 |
| 派生类型（CreatePayload/响应类型） | types 层（types/apiFactory.ts） | — | 类型-only 文件，0 statements，不影响覆盖率 |
| blob 下载链（axios 实例+文件名提取+触发下载） | lib 层（download.ts 新建） | — | 三处重复实现归一（opsApi + rpaApi 裸 fetch） |
| Excel 导入导出 | opsApi（excelApi entityType 形态保持） | download.ts | D-04：导出位置不变、内部委托 |
| React Query key 管理 | lib 层（queryKeys.ts，**不动**） | — | CONTEXT 明确不在本期 |

## Standard Stack

### Core（全部已存在于项目，**零新依赖** — .npmrc legacy-peer-deps 陷阱不触发）
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| typescript | ~5.9.3（package.json 实测） | 泛型工厂 + Omit 派生类型 | 项目编译器，行为已实测探针验证 |
| vitest | ^4.0.18 | 契约测试 + 扫描测试 | 现有 13 个 .test.ts 同栈；jsdom + vi.mock 模式成熟 |
| axios | ^1.19.0 | download.ts 的 blobAxios | opsApi 现状即独立 axios 实例（不带响应拦截器） |
| @types/node | ^24.10.9 | 扫描测试 readFileSync/readdirSync | 已装；colors.test.ts 有 in-repo 先例 |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| vitest 扫描测试（D-12） | ESLint no-restricted-syntax | CONTEXT 已裁决 ESLint 路线 deferred；扫描测试可进 `npm run test` gate |
| 正则匹配（D-12） | TypeScript compiler API AST | AST 更稳（能区分「函数体 = 单条 return post(CRUD形 URL)」），`typescript` 已在 devDependencies，零成本 |
| tsc 探针（CreatePayload 负向断言） | vitest `--typecheck` + `*.test-d.ts` | typecheck 配置键为 [ASSUMED]；主推「运行时契约测试 + type-check gate」组合，typecheck 为可选增强 |

**Installation:** 无。本项目 phase 零安装动作。

## Package Legitimacy Audit

本 phase **不安装任何外部包**（零新依赖，全部使用 package.json 既有依赖），Package Legitimacy Gate 不触发。清单如下以防 planner 误加依赖：

| Package | 状态 | 说明 |
|---------|------|------|
| typescript / vitest / axios / @types/node | 既有 | package.json 已锁，无需审计 |

**Packages removed due to slopcheck [SLOP] verdict:** none（无新包）
**Packages flagged as suspicious [SUS]:** none
**红线提醒：** 若 planner 在执行中被诱惑引入 `ts-morph` / `ast-grep` 等扫描辅助包——不需要，`typescript` 编译器 API（`ts.createSourceFile`）已足够且已在依赖树内；任何新包都会触发 legacy-peer-deps 陷阱检查。

## Architecture Patterns

### System Architecture Diagram

```
                       消费方（100+ 页面/组件/store，零改动）
                              │ import { buildingApi, getNoticeList, ... }
                              ▼
   ┌──────────────────────────────────────────────────────────────┐
   │ src/lib/*Api.ts（13 个文件，导出签名零变化）                      │
   │                                                              │
   │  对象形态(3): opsApi / rpaApi / vdiApi                         │
   │    export const buildingApi = createResourceApi<Building>(…)  │
   │    export const floorApi  = { ...crud, tree() }   ← spread    │
   │    export const vmApi     = { ...crud, list(原样), create(原样)│
   │    export const assetApi  = { ...crud, statistics(原样覆盖) }  │
   │                                                              │
   │  扁平形态(9): noticeApi / dutyApi / workorderApi / ...         │
   │    const noticeCrud = createResourceApi<Notice>({basePath})   │
   │    export function getNoticeList(p) { return noticeCrud.list(p) }│
   │    export function publishNotice(id) { …原样保持… }            │
   └──────────────┬───────────────────────────┬───────────────────┘
                  │                           │
                  ▼                           ▼
   ┌────────────────────────────┐   ┌──────────────────────────┐
   │ src/lib/apiFactory.ts (新)  │   │ src/lib/download.ts (新)  │
   │ createResourceApi<T>(cfg)   │   │ blobAxios(5min 超时)      │
   │  8 方法 + CrudApiConfig     │   │ downloadFile(GET)         │
   │  唯一权威（D-01/D-03）       │   │ downloadFilePost(POST)    │
   └──────────────┬─────────────┘   │ extractFilename… / trigger│
                  │                 └────────────┬──────────────┘
                  ▼                              │
   ┌─────────────────────────────────────────────┴───┐
   │ src/lib/api.ts（零改动）                              │
   │ post/get/put/del/postFormData → axios 实例            │
   │ → SM2+SM4 加密拦截器 → 401 刷新/重放 → BaseResponse<T>│
   └───────────────────────────┬──────────────────────────┘
                               ▼
                     后端（契约零改动）
```

### Recommended Project Structure
```
xingran-react-frontend/src/
├── lib/
│   ├── apiFactory.ts          # 新建 94-01：createResourceApi<T> + CrudApiConfig + DropdownOption
│   ├── apiFactory.test.ts     # 新建 94-01：契约测试（D-11）
│   ├── download.ts            # 新建 94-01：blob 下载链归一（D-04）
│   ├── download.test.ts       # 新建 94-01：下载链契约测试（D-11）
│   ├── apiFactory.invariants.test.ts  # 新建 94-03 收口：扫描测试（D-12）
│   ├── opsApi.ts              # 94-02：删私有工厂 + blob 基建迁出
│   ├── rpaApi.ts              # 94-02：删第二份私有工厂 + scriptApi 接入 + downloadReport 迁出
│   ├── vdiApi.ts              # 94-03：vdiServerApi spread；vmApi spread+override
│   └── (扁平文件 9 个)         # 94-03：cluster 级委托
└── types/
    └── apiFactory.ts          # 新建 94-01：CreatePayload<T> 等派生类型（type-only，0 statements）
```

### Pattern 1: spread + override（对不上时的既定惯例，本轮的核心迁移手法）
**What:** 工厂实例 spread 进导出对象，对不上的方法用原始实现覆盖（签名保持原样）。
**When to use:** 个别方法的请求/返回类型与工厂契约不同构（如 vmApi.list/create、assetApi.statistics）。
**Example（既有先例 opsApi:638-668 assetApi.statistics 覆盖）:**
```typescript
// Source: xingran-react-frontend/src/lib/opsApi.ts:636-700（assetApi 现状）
const vmCrud = createResourceApi<VirtualMachine>({ basePath: "/vdi/vms" });
export const vmApi = {
  ...vmCrud,
  // override：list/create 签名与工厂不同构（VMListParams 接口 / CreateVMRequest），原样保留
  list: async (params: VMListParams) => post<VMPageResponse>("/vdi/vms/list", params),
  create: async (data: CreateVMRequest) => post<VirtualMachine>("/vdi/vms", data),
  // get/update/delete 直接来自 spread（类型兼容，已验证）
};
```

### Pattern 2: 扁平文件 wrapper 委托（D-05）
**What:** 文件内建私有工厂实例，同构函数一行委托；导出函数保留显式返回类型注解。
**Example:**
```typescript
// workorderApi.ts 迁移后形态
const orderCrud = createResourceApi<WorkOrder>({ basePath: "/workorder/orders" });

export function getWorkOrderList(
  params: WorkOrderListRequest
): Promise<BaseResponse<PageResponse<WorkOrder>>> {
  return orderCrud.list(params as unknown as PageParams & Record<string, unknown>);
}
export function getWorkOrder(id: string): Promise<BaseResponse<WorkOrder>> {
  return orderCrud.get(id);  // T=WorkOrder，返回类型天然一致，零 cast
}
export function deleteWorkOrder(id: string): Promise<BaseResponse<{ message: string }>> {
  return orderCrud.delete(id) as Promise<BaseResponse<{ message: string }>>;  // 单 as 下转合法
}
// createWorkOrder/updateWorkOrder：请求类型 WorkOrderCreateRequest 与 CreatePayload 不同构 → 保持原样
```

### Pattern 3: downloadFilePost（D-04 的补全设计）
**What:** download.ts 除 GET 版 `downloadFile` 外，提供 POST 版本，服务 excelApi.export、assetApi.excel.export、rpaApi.downloadReport 三个 POST-blob 场景。
**Why:** CONTEXT 的 D-04 四件套只覆盖 GET 链；但 excelApi.export（opsApi:385-396）与 asset excel export（:685-698）是 `blobAxios.post` + 文件名提取的内联重复，rpaApi.downloadReport（:355-379）是裸 fetch。三者同构 = POST-blob-提取文件名-触发下载。**不补 POST 版本则三处重复继续存在，D-10 模板清零在下载域不成立。**

### Anti-Patterns to Avoid
- **硬套工厂到异构函数**（D-06/D-08 已锁）：给 publishNotice/testADConnection 造包装 = 无意义样板
- **改 HTTP 动词/路径来迁就工厂**：CONTEXT 明确「GET vs POST 端点细节保持各文件现状，不改后端契约」——notificationConfigApi 的 PUT/DELETE 是整文件 KEEP 的根因
- **在扫描测试里 grep 全仓 src**：D-12 范围 = `src/lib/*Api.ts`（api.ts/apiFactory.ts/download.ts 自身豁免），扩大范围会误伤页面内联 post
- **给工厂方法加副作用**（默认分页填充等）：adDomainApi 的 `withDefaultPagination` 必须留在 wrapper 内前置调用，工厂保持纯透传

## 逐文件迁移矩阵（本研究核心交付物）

> 判定图例：**DELEGATE** = 一行委托（类型兼容）｜**DELEGATE+CAST** = 委托 + wrapper 内 cast/注解保持签名｜**SPREAD+OVERRIDE** = 工厂 spread + 个别方法原样覆盖｜**KEEP** = 保持现状（异构/动词不同/类型契约不同构/默认值注入）

### 1. opsApi.ts（792 行，对象形态，27 个生产消费文件）— 94-02
| 成员 | 判定 | 说明 |
|------|------|------|
| 私有 `createCrudApi<T>`（:43-97） | **删除** | D-03：改 `import { createResourceApi } from "./apiFactory"` |
| 11 资源实例（building/floor/workstation/serverRoom/roomDevice/dedicatedLine/infoPoint/wall/door/floorPlanText/asset） | **DELEGATE** | 已是工厂调用，仅换来源；floor/workstation/wall/door/floorPlanText/asset 的 spread 自定义方法零改动 |
| wall/door/floorPlanText `.batch(action, ids)` | **KEEP**（覆盖层） | D-07：spread 重定义保持 |
| assetApi.statistics（:662-668，返回 `{total,normal,stopped,nbf}`） | **KEEP**（覆盖层） | 返回类型与工厂 `Record<string,number>` 不同构，现状即 override |
| `blobAxios`/`extractFilenameFromBlobResponse`/`triggerBrowserDownload`/`downloadFile`（:317-369） | **迁出** | → download.ts（D-04）；excelApi/deptApi/assetApi.excel 导出不变、内部 import |
| locationAliasApi（:183-205） | **KEEP** | 请求体用 `pageNum/pageSize`（非 current/pageSize）+ create 注入默认 `scope:"workstation"`——两处行为差异，硬套即行为变更 |
| roomPhotoApi（:220-272） | **KEEP** | GET query 参数 URL + upload/setPrimary 等异构为主 |
| workstationDeviceApi（:718-792） | **KEEP** | 全异构（getManual/syncAD/setPrimaryAndSave…） |
| componentApi / geocode 函数族 | **KEEP** | 非 CRUD |
| `DropdownOption`（:36-39） | **移至 apiFactory.ts** | 实测当前零外部消费方；opsApi 保留 re-export 兜底 |
| `CrudApiConfig` | 上移 apiFactory.ts | 两处均为私有定义，无导出兼容问题 |

### 2. rpaApi.ts（753 行，对象形态）— 94-02
| 成员 | 判定 | 说明 |
|------|------|------|
| 私有 `createCrudApi<T>`（:51-79，**第二份平行工厂**） | **删除** | 同 D-03 逻辑（CONTEXT 未点名，属 D-10 模板清零应有之义）；7 个 spread 实例（task/worker/execution/schedule/variable/template/notification）换 import 后**新增** batch/statistics/searchOptions 三方法（additive，行为中性） |
| scriptApi（:160-215，手写五方法 + testAction/format） | **SPREAD+OVERRIDE** | D-08 点名接入对象：`{ ...scriptCrud, testAction, format }`；五方法签名与工厂完全同构（list 参数 PageParams 字面兼容） |
| taskApi/workerApi/executionApi/scheduleApi/variableApi/templateApi/notificationApi | **KEEP**（spread 层零改动） | D-08：纯异构主体保持；仅工厂来源切换 |
| executionApi.downloadReport（:355-379 裸 fetch） | **迁出** | → download.ts 的 POST 链；白得 5min 超时防护；`getAuthHeaders` import 随之可删（该文件唯一使用点） |
| aiApi / statisticsApi | **KEEP** | 非 CRUD |

### 3. vdiApi.ts（190 行，对象形态，无任何工厂）— 94-03（低风险先行）
| 成员 | 判定 | 说明 |
|------|------|------|
| vdiServerApi（:144-168，五方法 + testConnection） | **SPREAD** | 唯一前提：`CreatePayload` 排除集必须含 snake_case（`VDIServer` 用 `created_at/updated_at`，tsc 实测 camelCase-only 会把必选时间戳残留在 CreatePayload 里导致 `VDIServerConfig` 不可赋值）；`VDIServerConfig` 含全部必选业务字段，可赋值 ✓ |
| vmApi（:31-140，五方法 + 14 异构） | **SPREAD+OVERRIDE** | list（VMListParams 接口变量直传 `VirtualMachineList/index.tsx:186-191`，tsc 实测会被 `Record<string,unknown>` 签名拒绝）与 create（`CreateVMRequest` 含 vtp_id/count 等实体上不存在的字段，且缺 `vm_id` 必选字段——两方向都不可赋值）必须 override 原样；update（UpdateVMRequest ⊆ Partial）可走 spread |
| 类型 re-export 块（:172-190） | 零改动 | — |

### 4-12. 扁平文件（9 个）— 94-03
| 文件（行数） | 可委托 cluster | 保持部分 | 整文件判定 |
|------|------|------|------|
| workorderApi.ts（622） | **categories**（list/get/create/update/delete 全 POST ✓，最干净）；**periodic templates**（同上 5 方法）；orders 的 list/get/delete | orders 的 create/update（WorkOrderCreateRequest 解耦类型）；batchDeleteWorkOrders（`/batch-delete` + `{ids}`，非工厂 batch 形状）；assign/comments/ratings/config/user/dept 等全部异构 | 三 cluster 委托 |
| knowledgeApi.ts（243） | **articles** list/get/delete；**categories** 五方法全 POST ✓；**tags** delete | create/update（KnowledgeArticleCreateRequest 含 tagIds 等实体外字段 + 缺 viewCount/likeCount 必选字段，双向不可赋值）；search/like/convert 异构；getAllKnowledgeTags | 部分委托；create/update 的保留是 D-12 warning 档 whitelist 项 |
| dutyApi.ts（362） | **duty pools** list/get/delete（getDutyPool 返回 `BaseResponse<DutyPool>` 与工厂 T 完全一致，零 cast） | create/update（DutyPoolCreateRequest 含 memberIds）；schedules 几乎全异构（generate/swap/manual/monthly）；holidays（list 参数是 `year: number` 非 PageParams；create 已是手写 `Omit<Holiday,…>` 本地派生——本项目已有该 idiom 的在库证据）；config 单例；getUserList/getDeptList/getDeptTree | 单 cluster 委托 |
| noticeApi.ts（188） | **admin notices** list/get/delete（+create 若 planner 接受 cast：CreateNoticeRequest 需核对 types/notice.ts:142 的形状后定） | batchDeleteNotices（`/batch-delete` 形状）；statistics（GET 动词）；publish/withdraw/用户端 my-notices 全族；buildWebSocketUrl | 主 cluster 委托 |
| adDomainApi.ts（731） | **ou-group-mappings**（五方法全 POST ✓ 全文件最干净 cluster）；**mappings** 的 list/create/update（**delegation 顺手修复 ：501 潜伏 bug**）；**configs** 的 list/create/update/delete | getADConfig/getMapping（GET 动词，工厂 get 是 POST，动词不可改）；groups/users/computers/logs/sync/test/enable/disable/unlock/accounts（id-in-body 非 REST 路径）等大量异构 | 部分委托；**⚠ :501 `/delete}` 多余右括号为潜伏 URL bug（deleteMapping 当前必然 404）——委托即修复 = 行为变更，plan 须显式登记（v1.29 D-05 例外条款同款纪律：bugfix 附回归说明）** |
| notificationConfigApi.ts（208） | 无 | **整文件 KEEP**：get 用 GET 动词、update 用 PUT `/{id}`（非 `/{id}/update`）、delete 用 DELETE 动词、list 参数用 `page`（非 current）——传输语义四处全部与工厂不同构，任何委托都是后端契约变更 | KEEP（D-12 warning 档 whitelist 项） |
| assetApi.ts（656，独立文件） | 无 | reconciliationApi（6 统计 + 2 exception，全部解包 res.data 返回裸数据——与工厂「返回 BaseResponse」契约不同）+ fixSuggestionApi（list/getById 返回解包后的 PageResult/Detail + accept/reject/apply/rollback/stats 动作族） | KEEP 整文件（D-12 warning 档 whitelist 项） |
| menuApi.ts（26）/ profileApi.ts（62）/ columnConfigApi.ts（35） | 无 | D-06 点名不套 | KEEP（已核实三者确无 CRUD 语义：只读菜单树/单例 profile+密码+头像/pageKey 存取） |

**量化口径修正：** CONTEXT「~15 个」实测 13 个 `src/lib/*Api.ts`（`src/lib/api/` 子目录的 networkApi.ts/macHeatmapApi.ts 不在 scope，与 CONTEXT 边界一致）；「opsApi 37 消费文件」本研究按 import 语句宽匹配实测 **27 个生产文件**（差异或含测试文件，不影响结论）；「13 文件 9823 行」生产文件实测 4868 行（9823 应含 ~4600 行配套测试）。

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| blob 下载（实例+超时+文件名+触发） | 第四份手写下载链 | download.ts（D-04） | 已有 3 份重复（opsApi blobAxios / rpaApi 裸 fetch / excelApi.export 内联），每份都各缺一角（fetch 无超时、内联无提取） |
| token 注入 | 手动取 token 拼头 | 复用 blobAxios 请求拦截器模式（opsApi:322-328 原样搬） | getAccessToken 是异步 SecureTokenStorage，同步拼头会拿到空值 |
| 请求加密/刷新/重放 | 任何绕过 api.ts 的直连 axios | 工厂方法一律走 `./api` 的 post/get | SM2 公钥轮换重放（api.ts:481-508）只在主实例拦截器里 |
| CRUD 拼装 | 新资源再手写五件套 | createResourceApi（D-13 收口后为 Convention） | 本 phase 的存在意义 |
| AST 解析（D-12） | 手写字符串切片 | `import ts from "typescript"` + `ts.createSourceFile` | 已在 devDependencies；正则对多行模板字符串 URL 极脆 |

**Key insight:** 本仓库的每一条「重复基建」都已经有至少一份生产验证的正确实现——本 phase 的全部工作是让它们只剩一份，而不是发明新的。

## Runtime State Inventory

> 本 phase 属前端纯代码重构（无 rename 字符串扩散、无数据迁移），按规程逐类显式回答：

| Category | Items Found | Action Required |
|----------|-------------|-----------------|
| Stored data | None — API 工厂是运行时对象；dualLevelCache 只缓存 geocode 结果（键 `geocode_<address>`，与 API 形状无关）；React Query queryKeys（queryKeys.ts）为独立 key factory，本期不动，键值零变化 | none |
| Live service config | None — 前端 SPA 无外部服务持有函数名/路径配置 | none |
| OS-registered state | None — 无 pm2/任务计划/plist 涉及前端 | none |
| Secrets/env vars | None new — download.ts 复用既有 `VITE_API_BASE_URL`/`VITE_WS_BASE_URL` 读取（opsApi:318 同款表达式原样搬） | none |
| Build artifacts | `xingran-react-frontend/dist/` 为陈旧产物（gitignore），随下次 build 重建 | none |

## Common Pitfalls

### Pitfall 1: interface 变量不可赋给 `Record<string, unknown>` 参数（tsc 5.9.3 实测 VERIFIED）
**What goes wrong:** 工厂 list 现状签名 `(params: PageParams & Record<string, unknown>)` 拒绝一切 interface 类型的参数变量（TS 无隐式索引签名规则）。`VirtualMachineList/index.tsx:186-191` 把 `const params: VMListParams` 变量传给 `vmApi.list` —— 若 vmApi.list 直接 spread 工厂，此处 type-check 必炸；扁平 wrapper 函数体内同样会炸。
**Why it happens:** interface 不获得隐式 index signature（type alias 会）——已用项目自带 tsc 实测：alias 变量通过、interface 变量报 TS2345。
**How to avoid:** 三选一（推荐 a）：(a) vmApi.list 走 SPREAD+OVERRIDE 保留原签名；(b) 把个别 ListParams interface 改 type alias（1 行/个，types/* 面扩散）；(c) wrapper 体内 `params as unknown as PageParams & Record<string, unknown>`（直接 `as` 不合法，两型互不可赋）。**不要**为此把工厂 list 参数放宽成 `object`——会丢失现有 ops 消费方 `Record<string, unknown>` 入参的兼容性（`Record<string,unknown>` 的 current 是 unknown，不满足 `current?: number`），泛型 `<P extends PageParams>` 同理排除 Record 入参。**保持现状签名 = 对 27 个 ops 消费文件零风险**（今天 type-check 全绿即证明）。
**Warning signs:** 迁移某文件后 `npm run type-check` 报 `Index signature for type 'string' is missing`。

### Pitfall 2: CreatePayload 严格版改变 create/update 调用点的编译契约（VERIFIED）
**What goes wrong:** `Omit<T, …>` 保留字段的必选性：`CreatePayload<Building>` 要求 `orgId/name/code/status` 必传（现状 `Partial<Building>` 全可选）。严格替换后，传部分字面量的调用点全部编译报错。生产调用点已普查：**create 13 处 + update 12 处**（集中在 pages/operations/* 与 pages/vdi/*，多为变量传递——变量是否含全部必选字段要逐个看）。opsApi.test.ts:146 `buildingApi.create({ name: "新楼" })` 这类字面量也受影响，但 **tsconfig.app.json exclude 了 `src/**/*.test.ts`，测试文件从不进 `tsc --noEmit`**，故测试不受影响（vitest/esbuild 不做类型检查）。
**Why it happens:** D-02 的本意就是让「漏传关键字段」在编译期报错——报错是特性不是缺陷。
**How to avoid:** planner 二选一并写进 plan：(a) **严格版**（D-02 字面）：接受 type-check 驱动的 ~25 处调用点修补（compiler 即 checklist，一次 task 可清）；(b) **`Partial<CreatePayload<T>>` 折中**：对象字面量里误写 `id`/`createdAt` 仍报错（excess property check）达成 D-02 目标，且全量调用点零修补。create 用严格版、update 用折中版（PATCH 语义本就全可选）是常见组合。研究倾向 (b) 起步、有余力再收紧——但两版都满足 D-02 的「误传 id 编译期报错」，最终由 planner 定。
**Warning signs:** 迁移后 type-check 错误数 >20 且集中在 pages/。

### Pitfall 3: camelCase-only 排除集对 vdi/rpa 域 snake_case 实体失效（VERIFIED）
**What goes wrong:** `VirtualMachine`/`VDIServer`/`VMAccount` 用 `created_at`/`updated_at`（snake_case）。`Omit<T,'id'|'createdAt'|'updatedAt'|'deletedAt'>` 不排除它们 → 必选时间戳残留在 CreatePayload → `vdiServerApi.create` 的 `VDIServerConfig`（无时间戳字段）不可赋值，tsc 实测报 `missing: vm_id, created_at`。
**How to avoid:** 排除集取**双命名并集**（详见 § D-02 设计）。

### Pitfall 4: rpaApi 的 7 个 spread 实例换工厂后方法集「变大」
**What goes wrong:** 现本地工厂 5 方法，共享工厂 8 方法 → taskApi 等导出对象新增 batch/statistics/searchOptions。运行时行为中性，但 (1) 任何对导出对象做 `Object.keys` 全等断言的测试会炸——已核实 rpaApi.test.ts:194 只断言 `rpaApi` 聚合对象（10 子 API 名），子 API 无形状断言 ✓；(2) knip（deadcode）若对这些「新增未调用方法」告警属预期噪音。
**How to avoid:** 无需处理；若 lint/knip 报警在 plan 里注明即可。

### Pitfall 5: wrapper 前置处理丢失
**What goes wrong:** adDomainApi 的 list wrapper 都先过 `withDefaultPagination(params)`（:226 本地泛型 helper）再 post——直接 `crud.list(params)` 会丢默认分页填充，属行为变更。
**How to avoid:** 委托写成 `crud.list(withDefaultPagination(params) as …)`；helper 留在原文件。

### Pitfall 6: blobAxios「第一个 axios.create 实例」的测试假设
**What goes wrong:** opsApi.test.ts:99 `h.created[0]` 假设 opsApi 模块加载时创建第一个 axios 实例。D-04 后该实例改由 download.ts 在模块加载时创建——opsApi import download.ts，模块图中仍是第一个 create ✓，但若测试只 import opsApi 而 download.ts 因摇树/顺序问题后加载，断言错位。
**How to avoid:** opsApi.test.ts 适配时改为 `import { blobAxios } from "./download"` 直取（download.ts 应导出实例本身），删除位置猜测；excel/blob 断言链（:335-423）逻辑不变。这正是 D-14 预告的「小幅适配」全部范围。

### Pitfall 7: `verbatimModuleSyntax` 已开启（tsconfig.app.json 实测）
**What goes wrong:** 新文件里 type-only import 不写 `import type` 会编译/报错。
**How to avoid:** apiFactory.ts/download.ts/新测试一律 `import type { BaseResponse, PageResponse, PageParams } from …`（仓库既有文件均如此，照抄即可）。

### Pitfall 8: 类型层负向断言（CreatePayload 排除生效）在现有 gate 下不可自动验证
**What goes wrong:** D-11 要求锁「CreatePayload 排除生效」，但 (1) tsconfig.app.json exclude 所有 `.test.ts` → `npm run type-check` 不检查测试文件；(2) vitest 默认不做类型检查（esbuild 转译）→ `@ts-expect-error` 写进测试永远不会被验证（写错了也不报）。
**How to avoid:** 主线 = 运行时契约测试锁 URL 拼接/解包（这些是真实可断言的）+ `npm run type-check` 作为生产代码类型 gate；可选增强 = vitest `--typecheck` + `*.test-d.ts`（配置键 [ASSUMED]，启用前须实测）。不要交付一个永远不会运行的 @ts-expect-error 测试。

## D-02 设计：`CreatePayload<T>` 排除字段集精确清单（research 开放点 #1 的答案）

```typescript
// src/types/apiFactory.ts
/** 服务端审计/主键字段——客户端 create/update 误传即编译期报错（D-02） */
export type ServerGeneratedKeys =
  | "id" | "createdAt" | "updatedAt" | "deletedAt"       // camelCase（ops/knowledge/duty/rpa 域）
  | "created_at" | "updated_at" | "deleted_at"           // snake_case（vdi 域，Pitfall 3）
  | "createdBy" | "updatedBy";                            // 审计人（服务端从 JWT 落库，前端传值无效）

export type CreatePayload<T> = Omit<T, ServerGeneratedKeys>;
```

依据：
- **双命名并集**为实测必需（Pitfall 3；Omit 对不存在的键无害，camelCase 实体不受 snake_case 键影响，反之亦然）
- **createdBy/updatedBy**：rpa/knowledge/duty 实体带 `createdBy?` 字段；该字段是审计字段（后端从登录态落库），前端回传无意义；且对无此字段的 ops 实体零影响
- **deletedAt**：仅 Asset 有（`deletedAt?: string`），保留在排除集防误传
- **不排除**任何业务字段（orgId/buildingId/status/memberIds 等全部保留原必选性/可选性）
- 若某资源将来确需客户端设置同名字段，在该资源用 SPREAD+OVERRIDE 局部放宽（模式已存在：assetApi.statistics）

## D-11 设计：契约测试用例清单（research 开放点 #4 的答案）

**apiFactory.test.ts**（mock 模式照抄 opsApi.test.ts:26-38 的 `vi.mock("./api")` + hoisted fn）：
1. **路径拼接八连**：`createResourceApi<Building>({ basePath: "/ops/building" })` 逐方法断言 `mockPost` 收到 `/ops/building/list`、`/ops/building/:id`、`/ops/building`、`/ops/building/:id/update`、`/ops/building/:id/delete`、`/ops/building/batch`(含 `{action,...data}` 合并)、`/ops/building/statistics`、`/ops/building/dropdown-options`
2. **CrudApiConfig.dropdownPath**：自定义值覆盖默认（serverRoomApi 曾用形态）
3. **泛型透传**：`list` 返回值即 mock 的 `BaseResponse<PageResponse<T>>` 原对象（透传不解包）
4. **statistics/searchOptions 解包语义**：`res.data ?? {}` / `res.data ?? []`（data 为 undefined 时返回空对象/空数组——这是工厂与裸 post 的唯一行为差异，必须锁）
5. **batch 参数合并**：`batch("enable", { ids })` → body = `{ action: "enable", ids }`

**download.test.ts**（mock 模式照抄 opsApi.test.ts:41-71 的 axios create 工厂 + URL.createObjectURL 打桩 beforeAll）：
1. 请求拦截器注入 `Bearer <token>`（异步 getAccessToken）
2. `downloadFile` GET + `responseType: "blob"` + 非 2xx 抛「下载失败」
3. `downloadFilePost` POST + content-disposition 文件名提取（含 `%E6%A5%BC%E5%AE%87.xlsx` URL 编码用例，照抄 opsApi.test.ts:384）+ 默认文件名回退
4. `triggerBrowserDownload`：createObjectURL → a.click → revokeObjectURL 顺序
5. blobAxios timeout 配置 = 300000（锁 5min 超时语义不回退）
6. rpaApi.downloadReport 经新链路（format 参数进 URL、文件名 `execution_report_<id>.<format>`）

## D-12 设计：扫描测试检测模式选型（research 开放点 #3 的答案）

**推荐：TypeScript compiler API AST（硬档 + warning 档双级），放 `src/lib/apiFactory.invariants.test.ts`**

| 维度 | 正则 | TS AST（推荐） |
|------|------|----------------|
| 依赖 | 无 | `typescript` 已在 devDependencies（零安装） |
| 识别「函数体 = 单条 `return post<…>(CRUD形URL)`」 | 多行模板字符串 URL 极脆 | 遍历 PropertyAssignment/FunctionDeclaration → body 单 ReturnStatement → CallExpression 标识符 ∈ {post,get,put,del} → URL 参数模板字符串后缀匹配 `(/list \| /update \| /delete \| /batch-delete) \}` 等 |
| 先例 | colors.test.ts（node:fs 读源码先例 ✓，但用正则） | 后端 Phase 92 `cache_invariants_92_test.go`（go/parser + 硬档/外围 warning 档）——D-12 点名的对齐对象 |
| 误报控制 | 低 | 可精确豁免 download.ts/apiFactory.ts/api.ts |

**分级建议**（镜像 Phase 92 D-10①②）：**硬 fail 档** = opsApi/rpaApi/vdiApi（对象形态，工厂已接入，残留 = 回归）；**warning 档** = § 迁移矩阵中登记的 KEEP 模板（knowledgeApi create/update、notificationConfigApi 整文件、assetApi.ts 整文件、duty/notice/adDomain/workorder 的异构 create/update）——whitelist 数组写进扫描测试并附理由注释，收口时 warning 计数应 == 白名单数（防白名单腐烂）。
**范围**：仅 `src/lib/*Api.ts`（13 文件），glob 排除 `api.ts/apiFactory.ts/download.ts`。

## Plan 间分组与顺序（research 开放点 #5 的答案）

| Plan | 内容 | 风险 | 依据 |
|------|------|------|------|
| 94-01 | apiFactory.ts + types/apiFactory.ts + download.ts + apiFactory.test.ts + download.test.ts（全部新建文件，不碰既有文件） | 最低（纯增量） | pilot 先行（Phase 91 节奏）；download.ts 先落，94-02 的两处迁出才有落点 |
| 94-02 | opsApi（删私有工厂 + blob 迁出 + DropdownOption 迁移）→ rpaApi（删第二份工厂 + scriptApi 接入 + downloadReport 迁出）；opsApi.test.ts 适配 | 低（对象形态，工厂已在用，等价替换） | opsApi 37(实测27) 消费文件但零签名变化；rpaApi.test.ts 两个文件已核实无形状断言 |
| 94-03 | vdiApi → 扁平文件 clusters（按矩阵顺序：workorder categories/periodic → knowledge → duty pools → notice → adDomain）→ 扫描测试（D-12）→ CLAUDE.md Convention（D-13） | 中（逐 cluster 判定） | 低风险优先（API-FACTORY-04 原文）；adDomain 含 ：501 潜伏 bug 修复登记，放最后并单独 commit 注明 |

每个 plan 收口 gate：`npm run type-check` + `npm run lint` + `npx vitest run`（+ 94-03 追加 `npm run test:coverage` + 覆盖率 gate 脚本）。

## Code Examples

### 工厂最终形态（94-01 交付物骨架，提升自 opsApi.ts:43-97 + D-02 强化）
```typescript
// src/lib/apiFactory.ts
import { post } from "./api";
import type { PageParams, PageResponse } from "@/types";
import type { CreatePayload } from "@/types/apiFactory";

export interface DropdownOption { value: string; label: string; }  // 自 opsApi 迁入

export interface CrudApiConfig {
  basePath: string;
  /** 自定义 dropdown-options 端点路径,默认 "/dropdown-options" */
  dropdownPath?: string;
}

export function createResourceApi<T>(config: CrudApiConfig) {
  const { basePath, dropdownPath = "/dropdown-options" } = config;
  return {
    list: async (params: PageParams & Record<string, unknown>) =>
      post<PageResponse<T>>(`${basePath}/list`, params),
    get: async (id: string) => post<T>(`${basePath}/${id}`, {}),
    create: async (data: CreatePayload<T>) => post(basePath, data),          // D-02（或 Partial<CreatePayload<T>>，见 Pitfall 2）
    update: async (id: string, data: Partial<CreatePayload<T>>) =>
      post(`${basePath}/${id}/update`, data),
    delete: async (id: string) => post(`${basePath}/${id}/delete`, {}),
    batch: async (action: string, data: Record<string, unknown>) =>
      post(`${basePath}/batch`, { action, ...data }),
    statistics: async (params: Record<string, unknown> = {}) => {
      const res = await post<Record<string, number>>(`${basePath}/statistics`, params);
      return res.data ?? {};
    },
    searchOptions: async (params: Record<string, unknown> = {}) => {
      const res = await post<DropdownOption[]>(`${basePath}${dropdownPath}`, params);
      return res.data ?? [];
    },
  };
}
```
> 签名与 opsApi 现状逐一对照过：除 create/update 参数类型按 D-02 强化外全部 1:1；`post<T>` 默认 `T=unknown`，create/update 返回 `Promise<BaseResponse<unknown>>` 与现状一致。

### download.ts 表面（D-04 + Pattern 3 补全）
```typescript
// src/lib/download.ts —— blobAxios / extractFilenameFromBlobResponse /
// triggerBrowserDownload 自 opsApi.ts:317-368 原样迁入（含 5min 超时注释），
// 新增 POST 变体消除 excelApi.export / asset excel / rpa downloadReport 三处内联：
export { blobAxios };
export function extractFilenameFromBlobResponse(response: AxiosResponse<Blob>, fallback: string): string;
export function triggerBrowserDownload(blob: Blob, filename: string): void;
export function downloadFile(url: string, filename: string): Promise<void>;                       // GET
export function downloadFilePost(url: string, body: unknown, defaultFilename: string): Promise<void>; // POST+文件名提取
```

### 扫描测试骨架（D-12）
```typescript
// src/lib/apiFactory.invariants.test.ts —— 先例：src/design-system/tokens/colors.test.ts (node:fs) 
// + 后端 cache_invariants_92_test.go (双档分级)
import { readFileSync, readdirSync } from "node:fs";
import ts from "typescript";
// 遍历 src/lib/*Api.ts（排除 api/apiFactory/download）→ ts.createSourceFile →
// 访问 ObjectLiteralProperty / FunctionDeclaration，body 为单条 return post|get|put|del(...)
// 且 URL 模板匹配 CRUD 五件套后缀 → 命中：硬档文件 fail / warning 档文件须在 whitelist
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| `create: (data: Partial<T>)` | `CreatePayload<T> = Omit<T, ServerGeneratedKeys>`（D-02） | 本 phase | 误传 id/时间戳编译期报错；与 refine TVariables 显式派生对齐 [CITED: 94-CONTEXT.md specifics——react-admin 9 方法 / refine 6 必需+5 可选，Context7 查证 2026-09-05] |
| 两份私有 createCrudApi（opsApi 8 法 + rpaApi 5 法） | 单一 apiFactory.ts 权威（D-01/D-03） | 本 phase | 与 Phase 92 `internal/services/base` 单一权威哲学同构 |
| 3 份 blob 下载实现（含 1 份无超时裸 fetch） | download.ts 单链 | 本 phase | rpa downloadReport 白得 5min 超时防护 |
| 手写 CRUD 五件套直调 post | 工厂/wrapper 委托 | 本 phase | D-10 定性清零 + D-12 可回归 |

**Deprecated/outdated:** ROADMAP 的 `getByID`/import/export 进工厂核心措辞（D-01 校准，行业无依据 + 职责混合）；`Partial<T>` 工厂签名。

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | vitest `--typecheck` + `*.test-d.ts` 是 vitest 4 的类型测试机制（配置键名未实测） | Pitfall 8 / Alternatives | 低——仅为可选增强，主线（运行时契约测试 + type-check gate）不依赖它 |
| A2 | CONTEXT「opsApi 37 消费文件」口径含测试文件或更宽匹配；本研究宽匹配实测 27 个生产文件 | 迁移矩阵 | 低——结论（签名不变 → 零改动）不受计数影响 |
| A3 | CreateNoticeRequest/WorkOrderCreateRequest 等「解耦请求类型」与各自实体 T 的 Omit 双向不可赋值——基于已读类型定义的结构推断（抽查的 Building/vm/VDIServer 四例均经 tsc 证实同规律），未对全部 ~15 个请求类型逐个跑探针 | 迁移矩阵 | 中——若个别类型意外可赋值，该函数从 KEEP 升级为 DELEGATE（方向安全：type-check 会即时暴露，plan 执行时以编译器为准微调矩阵） |
| A4 | adDomainApi:501 `deleteMapping` 的 `/delete}` URL 在生产必然 404（gin 路由不会匹配带右括号的路径）——未起后端实测 | 迁移矩阵 | 低——即使后端有诡异的容错匹配，委托后改为正确 URL 也是修 bug 语义，plan 登记流程不变 |

## Open Questions (RESOLVED)

> **决策落点（2026-09-05 修订登记）：** 两条开放问题均已裁决并落 plan——Q1（CreatePayload 严格版 vs 折中版）→ 94-01 Task 1 planner 定档：create/update 均采用 `Partial<CreatePayload<T>>` 折中版，本 phase 不收紧严格版；Q2（mappings 委托是否顺带修 ：501 URL bug）→ 94-03 Task 2 采纳推荐案：修复 + 独立 atomic commit 登记（v1.29 D-05 例外条款纪律）。以下保留研究期原文，仅作决策过程备查。

1. **CreatePayload 严格版 vs 折中版（Pitfall 2 的 (a)/(b)）**
   - What we know: 两版都满足 D-02 字面目标；严格版需修补 ~25 处调用点（已全部定位），折中版零修补
   - What's unclear: 用户对「调用点补全必选字段」是否视为价值（更完整的 payload）还是噪音
   - Recommendation: plan 94-01 按折中版落地（`create: CreatePayload<T>` 用于 ops 域调用面最小、`update: Partial<CreatePayload<T>>`），若 type-check 报错 <10 处顺手收紧为全严格——以编译器反馈为准，plan 内写明两档
2. **adDomainApi mappings 委托是否顺带修 ：501 URL bug**
   - What we know: 当前 deleteMapping URL 带多余右括号，必然失配
   - What's unclear: 该功能是否有真实用户流量（修复即行为变更）
   - Recommendation: 修复 + 在 plan/commit 显式登记（v1.29 D-05 例外条款纪律）；不修复则该函数 KEEP 并在扫描测试 whitelist 注明原因——**不建议**带着已知 bug 委托（委托后 URL 自然变对，等于隐性修复，更须登记）

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Node.js | npm scripts 全部 | ✓ | v24.19.0 | — |
| npm | 安装/脚本 | ✓ | 11.17.0 | — |
| typescript | 工厂类型 + 扫描测试 AST | ✓（本地 devDep） | ~5.9.3 | — |
| vitest | 契约测试/扫描测试 | ✓（本地 devDep） | ^4.0.18 | — |
| @vitest/coverage-v8 | 覆盖率 gate | ✓（本地 devDep） | ^4.1.10 | — |
| check-frontend-coverage.sh + .coverage-fe-floors | API-FACTORY-05 覆盖率 gate | ✓（.github/scripts/ + 仓库根） | — | — |

**Missing dependencies with no fallback:** none
**Missing dependencies with fallback:** none（零新依赖，legacy-peer-deps 陷阱不触发）

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Vitest ^4.0.18 + jsdom + @vitest/coverage-v8 ^4.1.10 |
| Config file | `xingran-react-frontend/vitest.config.ts`（globals/jsdom/testTimeout 15s/maxWorkers 4/coverage include 全 src 口径） |
| Quick run command | `cd xingran-react-frontend && npx vitest run src/lib/apiFactory.test.ts src/lib/download.test.ts` |
| Full suite command | `cd xingran-react-frontend && npx vitest run`（CI 等价 `npm run test`） |

### Phase Requirements → Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| API-FACTORY-01 | 工厂 8 方法路径拼接/参数合并/解包语义 | unit（契约） | `npx vitest run src/lib/apiFactory.test.ts` | ❌ Wave 0（94-01 建） |
| API-FACTORY-02 | apiFactory.ts + types/apiFactory.ts 落地 + CreatePayload 编译期行为 | unit + type-check | `npx vitest run src/lib/apiFactory.test.ts && npm run type-check` | ❌ Wave 0 |
| API-FACTORY-03 | opsApi/rpaApi 迁移后行为不变（全部既有断言绿） | unit（既有回归） | `npx vitest run src/lib/opsApi.test.ts src/lib/rpaApi.test.ts src/lib/__tests__/rpaApi.batch56.unit.test.ts` | ✅（opsApi.test.ts 小幅适配） |
| API-FACTORY-04 | 其余文件迁移 + 模板清零可回归 | unit（既有回归）+ 扫描 | `npx vitest run src/lib/ && npm run type-check` | 扫描 ❌ Wave 0（94-03 建）；其余 ✅ |
| API-FACTORY-05 | 三件套 + 覆盖率不降 | gate | `npm run type-check && npm run lint && npx vitest run && npm run test:coverage && bash .github/scripts/check-frontend-coverage.sh xingran-react-frontend/coverage/coverage-final.json .coverage-fe-floors` | ✅ gate 脚本既有 |
| D-04 | blob 下载链（GET/POST/文件名/超时） | unit（契约） | `npx vitest run src/lib/download.test.ts` | ❌ Wave 0（94-01 建） |
| D-12 | *Api.ts 模板残留扫描 | unit（invariants 扫描） | `npx vitest run src/lib/apiFactory.invariants.test.ts` | ❌ Wave 0（94-03 建） |

### Sampling Rate
- **Per task commit:** 迁移涉及文件的 vitest run（单文件级）+ `npm run type-check`
- **Per wave merge:** `npx vitest run` 全量（`npm run type-check` + `npm run lint` 并行 gate）
- **Phase gate:** 三件套全绿 + `npm run test:coverage` + check-frontend-coverage.sh exit 0 后才可 `/gsd:verify-work`

### Wave 0 Gaps
- [ ] `src/lib/apiFactory.test.ts` — 覆盖 API-FACTORY-01/02（§ D-11 设计的 5 组用例）
- [ ] `src/lib/download.test.ts` — 覆盖 D-04（§ D-11 设计的 6 组用例）
- [ ] `src/lib/apiFactory.invariants.test.ts` — 覆盖 D-12（94-03 收口时建，双档分级）
- [ ] Framework install: 无需（vitest 既有）

**覆盖率注意事项（API-FACTORY-05）：** 覆盖率口径为全 src（vitest.config.ts include，GLOBAL floor 3.8 + `lib` 目录 floor 87.2，见 `.coverage-fe-floors`）。新增 apiFactory.ts/download.ts 自带契约测试 → 接近全覆盖；模板删除使 lib 分母变小、被删行本就低覆盖 → `lib` 目录占比只升不降；types/apiFactory.ts 为 type-only 0 statements 无影响。45.13%（v1.28 口径）不下降的约束在此 gate 结构下自动满足，无需新 floor bump。

## Security Domain

### Applicable ASVS Categories
| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | no | 不涉及（token 注入沿用既有拦截器，零改动） |
| V3 Session Management | no | 不涉及（401 刷新/重放在 api.ts，零改动） |
| V4 Access Control | no | 不涉及（后端 RBAC；前端无权限逻辑变化） |
| V5 Input Validation | partial | 工厂 `CreatePayload<T>` 编译期约束是本 phase 的类型安全增强；运行时校验仍归后端，前端不加 |
| V6 Cryptography | no | 不涉及（SM2/SM4 全链在 api.ts/拦截器，零触碰） |

### Known Threat Patterns for 本 stack
| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| blob 下载链绕过 token 注入 | Info Disclosure | download.ts 统一走异步 `getAccessToken` 请求拦截器（rpaApi 裸 fetch 现状已带 header，迁移后仍带 + 获得超时） |
| content-disposition 文件名注入/路径逃逸 | Tampering | 沿用既有 `extractFilenameFromBlobResponse`（decodeURIComponent + 引号剥离）+ `a.download` 仅取文件名不落盘路径；不新造解析 |
| 内网 IP 硬编码 | Info Disclosure | eslint no-restricted-syntax 既有 error 档守护；download.ts 的 baseURL 沿用 `VITE_API_BASE_URL` env 模式 |

> 说明：本 phase 为纯前端请求封装重构，不新增认证面/加密面/权限面；security_enforcement 配置缺失按启用处理，故保留本节如上（均无新增防线需求）。

## Project Constraints (from CLAUDE.md)

- **§ Frontend API Calling**：「Use wrapped API functions, NOT raw axios」——工厂方法底层必须走 `./api` 的 post/get/postFormData；operations 模块 CRUD 用 opsApi 导出（D-13 收口时同步修订本段）
- **§ Common Gotchas / 临时文件**：根目录 `temp_*.go`/`test_*.go` 禁忌（后端）；前端等价物 = 不要把探针脚本留在 `xingran-react-frontend/` 根
- **§ Frontend/React Best Practices**：useEffect 依赖 memoize 纪律——本 phase 不触碰组件层，但 D-13 文档修订时保留该段
- **Git Workflow**：commit 前须 build+test+lint 全绿并征得用户确认；GSD 流程内 commit 按 `commit_docs: true` 走 gsd-sdk
- **数据库/状态约定**（Status 0/1、缓存 base 权威等）与本 phase 无交集，迁移中不得顺手触碰
- **`.npmrc` legacy-peer-deps 陷阱**（MEMORY + CONTEXT）：直接 import 的包必须写进 package.json——本期零新依赖即天然合规

## Sources

### Primary (HIGH confidence)
- 本仓库源码逐文件通读：`xingran-react-frontend/src/lib/{opsApi,rpaApi,vdiApi,noticeApi,knowledgeApi,dutyApi,notificationConfigApi,adDomainApi,workorderApi,assetApi,menuApi,profileApi,columnConfigApi,api,queryKeys}.ts` + `src/types/{base,operations,vdi,notice}.ts`
- tsc 5.9.3 实测探针（项目自带编译器，`node_modules/.bin/tsc --noEmit --strict`）：interface/Record 不可赋、alias/Record 可赋、interface/object 交集可赋、Omit 必选性保留、snake_case 排除缺失复现——5 项结论全部实证
- `xingran-react-frontend/vitest.config.ts` + `tsconfig.app.json` + `package.json` + `.coverage-fe-floors` + `.github/scripts/check-frontend-coverage.sh`（gate 机制原文）
- 测试基建原文：`src/lib/opsApi.test.ts`（mock 模式 + blob 断言链）、`src/lib/rpaApi.test.ts`（形状断言核实）、`src/lib/__tests__/rpaApi.batch56.unit.test.ts`、`src/design-system/tokens/colors.test.ts`（node:fs 扫描先例）
- 调用点普查：`grep` 全 src（create 13 处 / update 12 处 / opsApi import 27 文件 / `VirtualMachineList/index.tsx:186-191` 实锤）

### Secondary (MEDIUM confidence)
- [CITED: 94-CONTEXT.md specifics] react-admin dataProvider 9 方法 / refine 6 必需+5 可选（讨论阶段 Context7 查证 2026-09-05，D-01 论据，本研究不重复验证亦不推翻）

### Tertiary (LOW confidence)
- [ASSUMED: A1] vitest 4 `--typecheck`/`*.test-d.ts` 配置机制（仅可选增强项）

## Metadata

**Confidence breakdown:**
- 迁移矩阵（Standard Stack 等价物）：HIGH — 每个判定都有一手源码证据 + 类型学实测
- 工厂形态/类型设计：HIGH — 基于现有生产代码提升 + tsc 探针验证，无凭记忆断言
- Pitfalls：HIGH — 8 项中 3 项经编译器实证，其余为源码直读事实
- Plan 分组建议：MEDIUM — 依赖 planner 对严格版/折中版 CreatePayload 的取舍

**Research date:** 2026-09-05
**Valid until:** 2026-10-05（前端依赖树稳定，零新依赖引入；30 天内有效）
