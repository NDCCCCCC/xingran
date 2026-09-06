# Phase 94: 前端 API 工厂化 (🟡 中优 P2) - Context

**Gathered:** 2026-09-05
**Status:** Ready for planning

<domain>
## Phase Boundary

把 `xingran-react-frontend/src/lib/` 下 **13 个** `*Api.ts`（实测；ROADMAP 说 ~15）收敛到单一工厂 `createResourceApi<T>()`（落点 `src/lib/apiFactory.ts` + `src/types/apiFactory.ts`）：把 opsApi 内已在生产使用的私有 `createCrudApi<T>`（11 个资源实例）**提升**为共享工厂，其余文件按「能对上才委托」原则迁移；blob 下载基建归一到独立 `download.ts`；向后兼容承诺 = **全部导出签名零变化**（100+ 消费文件零改动）；验收 = 模板清零定性 + npm 三件套 0 错误 + 覆盖率 ≥45.13% 不降。

**关键现实校准（2026-09-05 侦察确认，与 ROADMAP 2026-09-03 审计基线有偏差，以本 CONTEXT 为准）：**

1. **工厂雏形已存在** — `opsApi.ts:49` 私有 `createCrudApi<T>()` 8 方法（list/get/create/update/delete/batch/statistics/searchOptions），11 个资源在用（building/floor/workstation/serverRoom/roomDevice/dedicatedLine/infoPoint/wall/door/floorPlanText/asset）。本期动作是**提升共享**而非从零设计
2. **ROADMAP 7 方法签名与行业标准冲突** — 行业对照（Context7 查证 react-admin dataProvider 9 方法 / refine 6必需+5可选）：核心五方法 list/getOne/create/update/delete(One)；**两大框架均不把 import/export 放 data provider**；批量是可选扩展。ROADMAP 的 `getByID`/`import`/`export` 经检验无行业依据且不同构（export 返回 void 触发下载 ≠ `Promise<BaseResponse<T>>`），措辞按 D-01 校准
3. **两种文件形态并存** — 对象 API 3 个（opsApi / vdiApi / rpaApi）+ 扁平函数导出 9 个（noticeApi/knowledgeApi/notificationConfigApi/dutyApi/menuApi/profileApi/columnConfigApi/adDomainApi/workorderApi）。adDomainApi(731行)/workorderApi(622行) 实为扁平风格，异构方法占比极高
4. **消费面（迁移风险度量）** — dutyApi 51 个消费文件、opsApi 37、adDomainApi 23、workorderApi 20、assetApi 19、noticeApi 16、knowledgeApi 11；每个 Api 文件都有配套 `.test.ts`（13 个测试文件 ~4600 行）
5. **blob 下载基建重复两份** — opsApi（blobAxios 5min 超时 + triggerBrowserDownload）与 rpaApi.ts:358-378（裸 fetch 手写完整下载链，无超时防护）同功能不同实现
6. **queryKeys.ts 已集中化**（React Query key factory）— 不在本期触碰范围

**不在本 phase：**
- React Query / queryKeys 与工厂的集成联动（保持现状）
- excelApi 的 entityType 二级工厂化重构（entityType 参数化形态保持，仅内部委托 download.ts）
- wall/door/floorPlanText 的 `batch(action, ids)` 签名与工厂核心 `batch(action, data)` 统一（D-07 保持现状）
- 异构函数（publishNotice/generateSchedule/assignWorkOrder 等）的对象化改造
- 路由/页面/组件等消费方改造（签名不变承诺下零改动）
- menuApi/profileApi/columnConfigApi 工厂化（D-06 非 CRUD 小文件不套）

</domain>

<decisions>
## Implementation Decisions

### 工厂形状 (Area 1)

- **D-01: 提升现有工厂，行业对齐** — 把 opsApi 私有 `createCrudApi<T>` 提升到 `src/lib/apiFactory.ts`，8 方法为核（list/get/create/update/delete/batch/statistics/searchOptions），与 react-admin/refine 核心五方法 1:1 对齐。ROADMAP 7 方法（`list/getByID/create/update/delete/import/export`）措辞按现实校准（REQUIREMENTS + ROADMAP 同 commit 修订，Phase 91/92 先例）：import/export 不进工厂核心、`getByID` 改名不采用（避免 37 个消费 opsApi 文件波及）。Context7 行业对照为论据（见 specifics）
- **D-02: 派生类型强化** — `src/types/apiFactory.ts` 定义派生类型助手（`CreatePayload<T> = Omit<T, 'id' | 'createdAt' | 'updatedAt' | 'deletedAt'>` 形态，具体排除集 researcher 定），create/update 用它替换 `Partial<T>`，对齐 refine 的 TVariables 显式派生模式；误传 id/时间戳编译期报错
- **D-03: 单一权威** — opsApi 私有 `createCrudApi` 定义删除，opsApi 统一 import apiFactory.ts（Phase 92 base 包单一权威哲学直接应用）
- **D-04: 独立 download.ts** — blobAxios / extractFilenameFromBlobResponse / triggerBrowserDownload / downloadFile 迁入新共享文件（最终命名 planner 定）；opsApi 与 rpaApi 都改用；excelApi/deptApi/assetApi.excel 导出位置不变、内部委托；rpaApi.downloadReport 裸 fetch 重复顺手消灭（白得 5min 超时防护）

### 扁平函数文件处置 (Area 2)

- **D-05: 内部委托签名不变** — 9 个扁平文件中能对上工厂方法的函数（getList/create/update/delete 形状）内部改一行委托工厂实例；异构函数（publishNotice/generateSchedule/swapDuty 等）原样保持。导出签名零变化 → 100+ 消费文件零改动，向后兼容承诺最硬
- **D-06: 非 CRUD 小文件不套工厂** — menuApi（3 个只读函数）/ profileApi（单例资源+密码+头像）/ columnConfigApi（按 pageKey 存取）无标准 CRUD 语义，保持现状（硬套只造无意义包装）

### 异构方法与大文件边界 (Area 3)

- **D-07: batch 签名保持现状** — wall/door/floorPlanText 三处 spread 重定义的 `batch(action, ids: string[])` 不与工厂核心 `batch(action, data)` 统一（向后兼容优先；签名统一属 planner 权衡，本期不做）
- **D-08: 大文件「能对上才委托」统一适用** — adDomainApi/workorderApi（扁平）只把标准 CRUD 形状函数接上工厂；rpaApi 的 scriptApi（标准五方法对象）接入，taskApi/workerApi/executionApi（纯异构）保持；vdiApi（对象 API）走 floorApi spread 模式（工厂 + 自定义方法展开）
- **D-09: statistics/searchOptions 保持内联** — 11 个资源过半真实在用，ops 域高频需求；挪出会给每资源加 spread 样板违背减重复初衰。工厂定位 = ops 域资源工厂，非通用框架
- **D-10: 验收锚点 = 模板清零定性** — 所有 `*Api.ts` 不再有与工厂方法同构的手写 CRUD 函数体（直调 post 的五件套模板清零）；不设 LOC 硬数字（前端模板每处 3-8 行，净减估 100-200 行量级，设数字无意义）。Phase 91 定性+量化混合标准的前端适配版

### 测试与收口防线 (Area 4)

- **D-11: 专门契约测试** — 新建 apiFactory.test.ts（锁路径拼接七端点形态、泛型类型推导、CreatePayload 排除生效、statistics/searchOptions 解包）+ download.test.ts（锁 blob 下载链）。Phase 91 base/service_test.go 泛型契约测试的前端对等物
- **D-12: 扫描测试防线** — 收口时新增扫描测试：匹配 `*Api.ts` 中手写 CRUD 五件套模板（直调 post 的同构函数体），发现即 fail（或 warning，planner 定）。Phase 92 `cache_invariants_92_test.go` 前端版，与 D-10 模板清零锚点配套、可回归
- **D-13: CLAUDE.md Convention 段** — 收口时新增「前端 API 工厂 Convention」段：锁 `createResourceApi` 单一权威路径 + 新资源必须走工厂 + 模板清零约定；同步修订 § Frontend API Calling 段（Phase 90/91/92 收口惯例）
- **D-14: 13 个 `.test.ts` 基本不动** — 签名不变承诺（D-05）的直接推论；仅 opsApi.test.ts 等涉及内部结构调整的小幅适配，幅度 researcher 评估

### Claude's Discretion

以下细节由 planner/researcher 决定，无需再问用户：
- `CrudApiConfig` 配置面扩展形态（dropdownPath 之外是否需要新配置项）
- apiFactory.ts / download.ts 的最终文件命名与内部布局
- `CreatePayload<T>` 排除字段集精确清单（'id'/'createdAt'/'updatedAt'/'deletedAt' 之外是否扩展）
- 各扁平文件工厂实例的命名与 basePath 常量提取方式
- 扫描测试（D-12）的检测模式（正则 vs AST）与 fail/warning 级别
- 契约测试用例分组与覆盖清单
- 迁移在 3 个 plan 间的分组微调（保持 94-01 工厂 → 94-02 opsApi → 94-03 其余的 pilot→批量节奏）
- opsApi.test.ts 适配幅度与 rpaApi scriptApi 迁移的测试影响评估
- GET vs POST 端点细节保持各文件现状（工厂方法内部用 post/get 与现状一致，不改后端契约）

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### v1.29 milestone 全局
- `.planning/REQUIREMENTS.md` § API-FACTORY（:88-96）— API-FACTORY-01..05 需求原文（**01 的 7 方法签名按 D-01 校准为「提升现有 8 方法工厂」；02 的 types 落点新增派生类型助手（D-02）**）
- `.planning/workstreams/milestone/ROADMAP.md` § Phase 94 — Goal / 3 plans 拆分 / 4 Success Criteria（**SC-1 措辞按 D-01 修订；SC-2 「~15 个」实测 13 个**）
- `.planning/PROJECT.md` — v1.29 Current Milestone 段 D-01..D-06 locked decisions（D-03 零回归 / D-04 atomic commit / D-05 范围边界）
- `.planning/workstreams/milestone/phases/92-p2/92-CONTEXT.md` — Phase 92 决策先例：单一权威 + type alias 渐进 + invariants 扫描（D-03/D-12 的模式来源）
- `.planning/workstreams/milestone/phases/91-crud-base-repository-t-p1/91-CONTEXT.md` — Phase 91 决策先例：pilot→批量节奏 + 定性+量化混合验收（D-10 模式来源）

### 本期改造核心代码
- `xingran-react-frontend/src/lib/opsApi.ts` — **主迁移对象**（792 行）：私有 `createCrudApi<T>`（:43-97 提升对象）、11 个资源实例、blobAxios/downloadFile 基建（:317-369 → D-04 迁 download.ts）、excelApi/deptApi（entityType 二级工厂保持）、assetApi（:638-700 spread 模式样本）
- `xingran-react-frontend/src/lib/api.ts` — get/post/put/del/upload/postFormData/postLongRequest 基础封装（:526-585），工厂方法的底层依赖，零改动
- `xingran-react-frontend/src/types/base.ts` — BaseResponse<T>/PageResponse<T>/PageParams（:8/:29/:39），types/apiFactory.ts 的类型基础
- `xingran-react-frontend/src/lib/rpaApi.ts` — scriptApi 标准五方法（:160）接入样本 + downloadReport 裸 fetch 重复（:358-378 → D-04 消灭对象）
- `xingran-react-frontend/src/lib/vdiApi.ts` — vmApi/vdiServerApi（:31/:144）五方法核心 + 14 异构方法的 spread 模式迁移样本
- `xingran-react-frontend/src/lib/dutyApi.ts` — 51 个消费文件的最大消费面扁平文件（D-05 签名不变的主验证对象）
- `xingran-react-frontend/src/lib/adDomainApi.ts` / `workorderApi.ts` — 731/622 行扁平大文件（D-08 迁移边界样本）
- `xingran-react-frontend/src/lib/queryKeys.ts` — React Query key factory（已集中化，本期不动，防混淆）

### 测试基建
- `xingran-react-frontend/src/lib/opsApi.test.ts` — 612 行配套测试（D-14 适配评估对象；vitest mock 模式参照）
- 13 个 `*.test.ts` 与源文件一比一配套 — 签名不变承诺下基本不动（D-14）
- 前端覆盖率基线 45.13%（API-FACTORY-05 硬约束；`npm run test:coverage` 验证）

### 项目级约束
- `CLAUDE.md` § Frontend API Calling — 「use wrapped API functions, NOT raw axios」+ opsApi 用法段（D-13 收口修订点）
- `CLAUDE.md` § Frontend/React Best Practices — useEffect 依赖纪律（迁移不涉及但须遵守）
- `npm run type-check` + `npm run lint` + `npm run test` — API-FACTORY-05 三件套 gate
- `.npmrc` legacy-peer-deps 陷阱 — 直接 import 的包必须写进 package.json（本期零新依赖则不触发）

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `opsApi.ts:49` `createCrudApi<T>` — 工厂雏形已存在且经 11 资源生产验证，提升而非重写（D-01）
- `api.ts` 封装族（get/post/put/del/upload/postFormData/postLongRequest）— 工厂方法底层，含 SM2+SM4 加密与 token 刷新拦截器，零改动
- `types/base.ts` 三件套（BaseResponse/PageResponse/PageParams）— 类型地基现成
- `blobAxios` + `downloadFile` + `triggerBrowserDownload`（opsApi）— 下载基建成熟（5min 超时设计有注释说明），D-04 归一对象
- `floorApi` spread 模式（opsApi:120-128）— 对象 API 扩展的既有惯例，D-08 vdiApi 迁移直接复用

### Established Patterns
- **spread 扩展模式**：`const xxxApi = { ...crudApi, customMethod }` — 自定义方法接入的唯一既定惯例（floor/workstation/wall/door/floorPlanText/asset 六处在用）
- **两种文件形态**：对象 API（3 个）vs 扁平函数导出（9 个）— 迁移语义按形态分派（D-05/D-08）
- **一比一配套测试**：每个 Api 文件一个同名 .test.ts — D-14 基本不动策略的依据
- **React Query key factory**（queryKeys.ts as const 元组）— 已集中化，与工厂无耦合
- **atomic commit + pilot→批量**（v1.29 D-04 + Phase 91 节奏）— 94-01 工厂 → 94-02 opsApi → 94-03 其余

### Integration Points
- 100+ 消费文件（pages/components/store/hooks）— 签名不变承诺下零改动，是验收红线
- `CLAUDE.md` § Frontend API Calling — 新资源走工厂的文档权威（D-13 修订点）
- `npm run type-check` / `lint` / `test` — 三件套 gate（API-FACTORY-05）
- 覆盖率 45.13% 基线 — 契约测试（D-11）自然抬升，模板删除使分母变小

</code_context>

<specifics>
## Specific Ideas

### 行业对照结论（D-01 论据，Context7 查证 2026-09-05）

| 方法 | react-admin (9) | refine (6必需+5可选) | 本项目 createCrudApi | ROADMAP 7 方法 |
|------|----------------|---------------------|---------------------|---------------|
| 列表 | getList | getList（必需） | list ✓ | list |
| 按 ID 取单个 | getOne | getOne（必需） | get ≈ 简写 ✓ | getByID ✗ 无行业依据 |
| 创建/更新/删除 | create/update/delete | create/update/deleteOne | ✓✓✓ | ✓ |
| 批量 | *Many（可选） | *Many（可选） | batch（扩展）✓ | 无 |
| 非 CRUD | — | custom 逃生舱（可选） | statistics/searchOptions 内联 + spread | 无 |
| **Excel 导入导出** | **不含**（exporter 是 List 组件选项） | **不含** | 不含 ✓ | **import/export ✗ 职责混合** |

结论：现有 8 方法形状高度符合行业标准；ROADMAP 的 getByID/import/export 经检验不采納（D-01）。

### 迁移后样板形态预览（D-01/D-02）

```typescript
// src/lib/apiFactory.ts（提升自 opsApi.ts:49，含 D-02 类型强化）
export interface CrudApiConfig { basePath: string; dropdownPath?: string }
export function createResourceApi<T>(config: CrudApiConfig) {
  return {
    list: (params: PageParams & Record<string, unknown>) => post<PageResponse<T>>(`${basePath}/list`, params),
    get: (id: string) => post<T>(`${basePath}/${id}`, {}),
    create: (data: CreatePayload<T>) => post(basePath, data),
    update: (id: string, data: Partial<CreatePayload<T>>) => post(`${basePath}/${id}/update`, data),
    delete: (id: string) => post(`${basePath}/${id}/delete`, {}),
    batch: (action: string, data: Record<string, unknown>) => post(`${basePath}/batch`, { action, ...data }),
    statistics: ..., searchOptions: ...,
  };
}

// opsApi.ts 迁移后（D-03 单一权威）
export const buildingApi = createResourceApi<Building>({ basePath: "/ops/building" });
```

### 工厂最终命名说明
讨论中「createResourceApi」为 ROADMAP 命名、内部实现是现有 createCrudApi 形状；最终导出名 planner 可保持 `createResourceApi`（ROADMAP 措辞）或 `createCrudApi`（现状名），二者皆符合 D-01——决策锁定的是**形状**（8 方法）不是名字。

### 估算依据（供 planner 参考）
- 13 文件 9823 行；工厂提升 + download.ts 新建 ~150 行；模板清零净减估 100-200 行量级（D-10 不设硬数字的原因）
- 迁移热点：opsApi（37 消费文件但工厂已在用，动作最轻）> vdiApi/rpaApi scriptApi（对象/spread 模式）> 9 个扁平文件（逐函数判断「能对上才委托」）
- 测试：13 个 .test.ts 基本不动 + 2 个新契约测试文件

</specifics>

<deferred>
## Deferred Ideas

### 范围外但相关的后续候选
- **React Query / queryKeys 与工厂集成** — useQuery 联动工厂实例（queryKey 由工厂派生）可再收敛一层样板；本期 queryKeys.ts 不动，若未来 React Query 使用面扩大再评估
- **excelApi entityType 二级工厂化** — entityType 参数化形态已是「工厂的工厂」，结构合理；仅当新增 ops 实体类型变得频繁时再考虑类型安全化（entityType 联合类型收窄）
- **`batch` 签名统一** — wall/door/floorPlanText 的 `(action, ids)` vs 工厂核心 `(action, data)`（D-07 保持现状）；若未来 CAD 平面图实体增多可统一
- **ESLint no-restricted-syntax 机械防线** — 被 D-12 扫描测试方案替代；若扫描测试误报困扰可再评估 ESLint 路线
- **`getByID` 命名** — 已拒绝（D-01）；记录理由：行业用 getOne，改名波及 37 消费文件零收益

None — discussion stayed within phase scope（无 scope creep 提案）

</deferred>

---

*Phase: 94-前端 API 工厂化*
*Context gathered: 2026-09-05*
