# Phase 94: 前端 API 工厂化 - Pattern Map

**Mapped:** 2026-09-05
**Files analyzed:** 20（新建 6 + 修改 9 + KEEP 5，另含 CLAUDE.md 收口段）
**Analogs found:** 20 / 20（全部文件在库内有直接或角色匹配模拟；本项目无「从零发明」项——本 phase 本质是提升既有代码）

> 前置说明：本 phase 与常规「新功能找模拟」不同——**多数待改文件本身就是模拟**（工厂提升源、spread 先例、扁平委托样本均在库内生产运行）。PATTERNS.md 的价值在于：把「抄哪一行、抄成什么形态」逐文件钉死，planner 直接引用。

---

## File Classification

| 新建/修改文件 | Role | Data Flow | Closest Analog | Match Quality |
|---------------|------|-----------|----------------|---------------|
| `src/lib/apiFactory.ts`（新建 94-01） | utility（API 工厂） | request-response | `src/lib/opsApi.ts:43-97`（私有 `createCrudApi<T>`，提升源本体） | **exact**（同代码提升） |
| `src/types/apiFactory.ts`（新建 94-01） | types（type-only 派生类型） | transform | `src/types/base.ts`（文件形态）+ `src/lib/dutyApi.ts:259-263`（`Omit<T,...>` 派生在库 idiom） | role-match |
| `src/lib/download.ts`（新建 94-01） | utility（blob 下载链） | file-I/O（streaming） | `src/lib/opsApi.ts:310-369`（blobAxios 四件套，原样迁入源） | **exact** |
| `src/lib/apiFactory.test.ts`（新建 94-01） | test（契约） | request-response | `src/lib/opsApi.test.ts:26-180`（vi.mock + 路径断言模式） | **exact** |
| `src/lib/download.test.ts`（新建 94-01） | test（契约） | file-I/O | `src/lib/opsApi.test.ts:41-115`（axios create 工厂 mock）+ `:382-405`（content-disposition 断言） | **exact** |
| `src/lib/apiFactory.invariants.test.ts`（新建 94-03） | test（invariants 扫描） | batch（静态扫描） | 后端 `internal/services/system/cache_invariants_92_test.go`（双档分级 AST 扫描）+ 前端 `src/design-system/tokens/colors.test.ts:233-252`（node:fs 读源码先例） | role-match |
| `src/lib/opsApi.ts`（修改 94-02） | lib/api-client（对象形态） | request-response + file-I/O | 自身（:43-97 工厂 → 改 import；:310-369 → 迁 download.ts） | **exact** |
| `src/lib/rpaApi.ts`（修改 94-02） | lib/api-client（对象形态） | request-response + file-I/O | `opsApi.ts:120-128`（spread 先例）+ 新 download.ts（downloadReport 迁入） | role-match |
| `src/lib/vdiApi.ts`（修改 94-03） | lib/api-client（对象形态） | request-response | `opsApi.ts:636-700`（assetApi spread+override 精确同构：list/create 签名不同构即 override） | role-match |
| `src/lib/workorderApi.ts`（修改 94-03） | lib/api-client（扁平形态） | request-response | 扁平 wrapper 委托模式；categories cluster（:494-521）为最干净委托对象 | role-match |
| `src/lib/knowledgeApi.ts`（修改 94-03） | lib/api-client（扁平形态） | request-response | `workorderApi.ts` categories cluster 同构（:186-213） | role-match |
| `src/lib/dutyApi.ts`（修改 94-03） | lib/api-client（扁平形态） | request-response | duty pools cluster（:155-193）；51 个消费文件 = D-05 签名不变主验证对象 | role-match |
| `src/lib/noticeApi.ts`（修改 94-03） | lib/api-client（扁平形态） | request-response | admin notices cluster（:23-76） | role-match |
| `src/lib/adDomainApi.ts`（修改 94-03） | lib/api-client（扁平形态） | request-response | mappings/ou-group-mappings cluster（:482-502 / :580-605）；含 ：501 潜伏 bug | role-match |
| `src/lib/notificationConfigApi.ts`（KEEP） | lib/api-client | request-response | — 动词全不同构（GET list/PUT `/{id}`/DELETE），整文件不动 | KEEP |
| `src/lib/assetApi.ts`（KEEP） | lib/api-client | request-response（解包 res.data 契约） | — 与工厂「返回 BaseResponse」契约不同，整文件不动 | KEEP |
| `src/lib/menuApi.ts` / `profileApi.ts` / `columnConfigApi.ts`（KEEP） | lib/api-client（非 CRUD） | request-response | — D-06 点名不套（已核实确无 CRUD 五件套语义） | KEEP |
| `src/lib/opsApi.test.ts`（小幅适配 94-02） | test | — | 自身（:99 `h.created[0]` 位置假设 → 改直取 download 导出） | **exact** |
| `CLAUDE.md`（收口 D-13） | docs/config | — | CLAUDE.md 既有「Cache Service Convention」段（Phase 92 收口先例） | role-match |

**范围口径核对（与 RESEARCH 一致）：** `src/lib/*Api.ts` 实测 13 个（`src/lib/api/` 子目录的 networkApi/macHeatmapApi 不在 scope）；`src/lib/queryKeys.ts` 本期不动。

---

## Pattern Assignments

### `src/lib/apiFactory.ts`（utility, request-response）— 94-01 新建

**Analog:** `xingran-react-frontend/src/lib/opsApi.ts`（提升源本体，D-01）

**底层依赖 import 模式**（opsApi.ts:5-8 + api.ts:526-540）：

```typescript
// opsApi.ts 现状 import —— apiFactory.ts 照抄此风格
import { post, get, postFormData } from "./api";

// api.ts:526-540 —— 工厂全部方法的传输层（零改动，只消费不修改）
export function get<T = unknown>(url: string, params?: unknown): Promise<BaseResponse<T>> {
  return api.get(url, { params });
}
export function post<T = unknown>(url: string, data?: unknown): Promise<BaseResponse<T>> {
  return api.post(url, data);
}
```

**核心工厂模式（提升对象，opsApi.ts:43-97 原文）**：

```typescript
interface CrudApiConfig {
  basePath: string;
  /** 自定义 dropdown-options 端点路径,默认 "/dropdown-options" */
  dropdownPath?: string;
}

function createCrudApi<T>(config: CrudApiConfig) {
  const { basePath, dropdownPath = "/dropdown-options" } = config;

  return {
    list: async (params: PageParams & Record<string, unknown>) => {
      return await post<PageResponse<T>>(`${basePath}/list`, params);
    },
    get: async (id: string) => {
      return await post<T>(`${basePath}/${id}`, {});
    },
    create: async (data: Partial<T>) => {          // ← D-02: 此参数类型按 CreatePayload<T> 强化
      return await post(basePath, data);
    },
    update: async (id: string, data: Partial<T>) => {  // ← D-02: Partial<CreatePayload<T>>
      return await post(`${basePath}/${id}/update`, data);
    },
    delete: async (id: string) => {
      return await post(`${basePath}/${id}/delete`, {});
    },
    batch: async (action: string, data: Record<string, unknown>) => {
      return await post(`${basePath}/batch`, { action, ...data });
    },
    statistics: async (params: Record<string, unknown> = {}) => {
      const res = await post<Record<string, number>>(`${basePath}/statistics`, params);
      return res.data ?? {};                        // ← 解包语义（工厂与裸 post 唯一行为差异，D-11 必锁）
    },
    searchOptions: async (params: Record<string, unknown> = {}) => {
      const res = await post<DropdownOption[]>(`${basePath}${dropdownPath}`, params);
      return res.data ?? [];
    },
  };
}
```

**提升时的三处已知差异（rpaApi 版对照）**——rpaApi.ts:51-79 存在第二份平行工厂，合并时以 opsApi 版为准：

| 维度 | opsApi 版（权威） | rpaApi 版（废弃） |
|------|------------------|------------------|
| 方法数 | 8（含 batch/statistics/searchOptions） | 5 |
| CrudApiConfig | `interface CrudApiConfig`（无泛型参数） | `interface CrudApiConfig<_T>`（无用泛型参数，:51 死代码） |
| dropdownPath | 支持 | 无 |

**DropdownOption 迁移**（opsApi.ts:36-39，D-12 矩阵登记「移至 apiFactory.ts，opsApi 保留 re-export 兜底」）：

```typescript
export interface DropdownOption {
  value: string;
  label: string;
}
```

**注意（Pitfall 7）**：`verbatimModuleSyntax` 已开启（tsconfig.app.json 实测），新文件 type-only import 必须写 `import type`：

```typescript
import type { PageParams, PageResponse } from "@/types";       // type-only ✓
import { post } from "./api";                                   // 值导入 ✓
```

**Anti-pattern（RESEARCH 已锁，勿违反）**：不要把工厂 list 参数放宽成 `object` 或改泛型 `<P extends PageParams>`——会破坏现有 27 个 ops 消费文件的 `Record<string, unknown>` 入参兼容性。**保持现状签名**。

---

### `src/types/apiFactory.ts`（types, transform）— 94-01 新建

**Analog 文件形态:** `xingran-react-frontend/src/types/base.ts`（纯 interface/type 导出、零运行时代码的 types 文件范式）；**派生类型在库 idiom:** `dutyApi.ts:259-263`。

**在库 `Omit` 派生先例（dutyApi.ts:259-263）——D-02 的 CreatePayload 不是新发明，仓库已有同 idiom：**

```typescript
// dutyApi.ts:259-263 —— 手写「排除服务端字段」的既有惯例（create/update 各一处）
export function createHoliday(
  data: Omit<Holiday, "id" | "createdAt" | "createdBy">
): Promise<BaseResponse<Holiday>> {
  return post("/duty/holidays", data);
}
```

**D-02 目标形态（RESEARCH § D-02 设计已定稿，排除集取双命名并集）**：

```typescript
// src/types/apiFactory.ts
/** 服务端审计/主键字段——客户端 create/update 误传即编译期报错（D-02） */
export type ServerGeneratedKeys =
  | "id" | "createdAt" | "updatedAt" | "deletedAt"       // camelCase（ops/knowledge/duty/rpa 域）
  | "created_at" | "updated_at" | "deleted_at"           // snake_case（vdi 域，Pitfall 3 实测必需）
  | "createdBy" | "updatedBy";                            // 审计人（后端从 JWT 落库）

export type CreatePayload<T> = Omit<T, ServerGeneratedKeys>;
```

**排除集依据（全部有库内证据）**：
- snake_case 键：`src/types/vdi.ts` 的 `VirtualMachine`/`VDIServer`/`VMAccount` 用 `created_at`/`updated_at`（tsc 实测 camelCase-only 会让 `VDIServerConfig` 不可赋值）
- `createdBy`/`updatedBy`：dutyApi.ts:16-17（`DutyPool`）、knowledgeApi.ts:33-35（`KnowledgeArticle`）、workorderApi.ts:93-96（`WorkOrder`）均带此二审计字段
- `deletedAt`：仅 Asset 有（`deletedAt?: string`）
- **不排除**任何业务字段（orgId/buildingId/status/memberIds 全部保留原必选性）

**strict vs 折中（Pitfall 2，planner 决策点）**：`Omit` 保留必选性 → `CreatePayload<Building>` 要求 `orgId/name/code/status` 必传，影响 create 13 处 + update 12 处调用点（RESEARCH 已全部定位）。两档方案：严格版（D-02 字面）或 `Partial<CreatePayload<T>>` 折中（excess property check 仍锁误传 id）。plan 内写明两档、以 type-check 反馈为准。

---

### `src/lib/download.ts`（utility, file-I/O）— 94-01 新建

**Analog:** `xingran-react-frontend/src/lib/opsApi.ts:310-369`（四件套原样迁入，含全部注释）

**blobAxios 实例（opsApi.ts:310-328 原文，5min 超时注释一并迁移）**：

```typescript
import axios from "axios";
import type { AxiosInstance, AxiosResponse, InternalAxiosRequestConfig } from "axios";
import { getAccessToken } from "@/utils/authHelpers";

// 专用于文件下载的 axios 实例：
// - 复用与主 API 客户端相同的 baseURL（去掉硬编码 /api/v1/ 前缀）
// - 不挂载响应拦截器，因为响应是 Blob 二进制流，无法 JSON 解析
// - 请求拦截器只做 Token 注入，行为与其他 CRUD 保持一致
// - 工位导出 1643 工位 + ~6000 行设备数据 → xlsx ~5-10 MB → 默认 30s timeout 易中招
//   改 5min 给足缓冲;普通 CRUD 不走 blobAxios 不会受影响
const blobAxios: AxiosInstance = axios.create({
  baseURL: import.meta.env.VITE_API_BASE_URL || "/api/v1",
  timeout: 300000,
});

blobAxios.interceptors.request.use(async (config: InternalAxiosRequestConfig) => {
  const token = await getAccessToken();
  if (token && config.headers) {
    config.headers.set("Authorization", `Bearer ${token}`);
  }
  return config;
});
```

**关键约束（RESEARCH Don't-Hand-Roll）**：token 注入必须走异步 `getAccessToken()` 请求拦截器——它是异步 SecureTokenStorage，同步拼头会拿到空值。download.ts 必须导出 `blobAxios` 实例本身（opsApi.test.ts 适配依赖此导出，见 Pitfall 6）。

**文件名提取 + 触发下载（opsApi.ts:331-358 原文）**：

```typescript
function extractFilenameFromBlobResponse(
  response: AxiosResponse<Blob>,
  defaultFilename: string
): string {
  const contentDisposition: string | undefined = response.headers["content-disposition"];
  if (!contentDisposition) {
    return defaultFilename;
  }
  const match = contentDisposition.match(/filename[^;=\n]*=((['"]).*?\2|[^;\n]*)/);
  if (match && match[1]) {
    return decodeURIComponent(match[1].replace(/['"]/g, ""));
  }
  return defaultFilename;
}

function triggerBrowserDownload(blob: Blob, filename: string): void {
  const blobUrl = window.URL.createObjectURL(blob);
  const a = document.createElement("a");
  a.href = blobUrl;
  a.download = filename;
  document.body.appendChild(a);
  a.click();
  window.URL.revokeObjectURL(blobUrl);
  document.body.removeChild(a);
}

async function downloadFile(url: string, filename: string): Promise<void> {
  const response = await blobAxios.get<Blob>(url, { responseType: "blob" });
  if (response.status < 200 || response.status >= 300) {
    throw new Error(`下载失败: ${filename}`);
  }
  triggerBrowserDownload(response.data, filename);
}
```

**新增 downloadFilePost（RESEARCH Pattern 3 补全设计）**——三个 POST-blob 场景归一的依据：

| 现有重复实现 | 位置 | 缺角 |
|-------------|------|------|
| `excelApi.export` 内联 | opsApi.ts:385-396（blobAxios.post + 状态检查 + 提取文件名 + 触发） | 无提取 helper 复用 |
| `assetApi.excel.export` 内联 | opsApi.ts:685-698（同构复制，但文件名不提取、固定默认名） | 丢 content-disposition |
| `executionApi.downloadReport` 裸 fetch | rpaApi.ts:355-379 | **无超时防护**（fetch 无 timeout，白得 5min 防护即此项） |

三处同构 = POST → blob → 提取文件名 → 触发下载，`downloadFilePost(url, body, defaultFilename)` 一个签名全覆盖。**不补 POST 版则 D-10 模板清零在下载域不成立。**

---

### `src/lib/apiFactory.test.ts`（test, request-response）— 94-01 新建

**Analog:** `xingran-react-frontend/src/lib/opsApi.test.ts:11-55`（mock 模式照抄）

**Mock 骨架（opsApi.test.ts:11-38 原文——vi.hoisted + 双路径 vi.mock 注册）**：

```typescript
import { beforeAll, beforeEach, describe, expect, it, vi } from "vitest";

const h = vi.hoisted(() => {
  const created: any[] = [];
  return { created, mockGetAccessToken: vi.fn<() => Promise<string>>() };
});

const mockPost = vi.fn();
const mockGet = vi.fn();
vi.mock("@/lib/api", () => ({
  post: (...args: unknown[]) => mockPost(...args),
  get: (...args: unknown[]) => mockGet(...args),
}));
vi.mock("./api", () => ({   // 双注册：@/lib/api 与 ./api 都要 mock（相对/别名导入都会命中）
  post: (...args: unknown[]) => mockPost(...args),
  get: (...args: unknown[]) => mockGet(...args),
}));
```

**契约断言风格（opsApi.test.ts:134-180 原文风格——D-11 的 5 组用例直接照此展开）**：

```typescript
describe("通用 CRUD 工厂 — buildingApi(/ops/building)", () => {
  it("list POST /ops/building/list 透传分页筛选", async () => {
    mockPost.mockResolvedValueOnce({ code: 0, data: { list: [], total: 0 } });
    const params = { current: 1, pageSize: 10, name: "研发楼" };
    await buildingApi.list(params);
    expect(mockPost).toHaveBeenCalledWith("/ops/building/list", params);
  });

  it("statistics 解包 data,空 data 回退 {}", async () => {
    mockPost.mockResolvedValueOnce({ code: 0, data: { total: 5, enabled: 3 } });
    expect(await buildingApi.statistics()).toEqual({ total: 5, enabled: 3 });
    mockPost.mockResolvedValueOnce({ code: 0, data: null });
    expect(await buildingApi.statistics({ status: 0 })).toEqual({});   // 空回退语义必须锁
  });
});
```

D-11 用例清单（RESEARCH § D-11 设计定稿）：① 路径拼接八连（list/get/create/update/delete/batch/statistics/dropdown-options）② `CrudApiConfig.dropdownPath` 自定义覆盖 ③ 泛型透传不解包 ④ statistics/searchOptions 解包与空回退 ⑤ batch 参数合并（`{action, ...data}`）。**注意（Pitfall 8）**：CreatePayload 排除生效的「编译期断言」在现有 gate 下不可自动验证（tsconfig exclude 了 `.test.ts`、vitest 不做类型检查）——不要交付永远不会运行的 `@ts-expect-error` 测试；编译期行为靠 `npm run type-check` gate。

---

### `src/lib/download.test.ts`（test, file-I/O）— 94-01 新建

**Analog:** `xingran-react-frontend/src/lib/opsApi.test.ts:41-115`（axios 工厂 mock + URL 打桩）+ `:335-422`（blob 断言链）

**axios create 工厂 mock（opsApi.test.ts:41-55 原文）**：

```typescript
vi.mock("axios", () => {
  const createInstance = () => {
    const instance = Object.assign(vi.fn(), {
      get: vi.fn(),
      post: vi.fn(),
      interceptors: {
        request: { use: vi.fn() },
        response: { use: vi.fn() },
      },
    });
    h.created.push(instance);
    return instance;
  };
  return { default: { create: () => createInstance() } };
});
```

**jsdom URL 打桩（opsApi.test.ts:103-115 原文——jsdom 不实现 createObjectURL）**：

```typescript
beforeAll(() => {
  Object.defineProperty(URL, "createObjectURL", {
    configurable: true, writable: true,
    value: vi.fn(() => "blob:fake-url"),
  });
  Object.defineProperty(URL, "revokeObjectURL", {
    configurable: true, writable: true, value: vi.fn(),
  });
});
```

**content-disposition 提取断言（opsApi.test.ts:382-399 原文，含 URL 编码文件名用例）**：

```typescript
it("export 从 content-disposition 提取文件名", async () => {
  blobAxios().post.mockResolvedValueOnce({
    status: 200,
    data: new Blob(["xlsx"]),
    headers: { "content-disposition": 'attachment; filename="%E6%A5%BC%E5%AE%87.xlsx"' },
  });
  await excelApi.export("building", { status: 0 });
  expect(blobAxios().post).toHaveBeenCalledWith("/ops/building/export", { status: 0 }, {
    responseType: "blob",
  });
  expect(URL.createObjectURL).toHaveBeenCalled();
});
```

**⚠ Pitfall 6（opsApi.test.ts 适配的唯一实质点）**：opsApi.test.ts:99 `h.created[0]` 假设 opsApi 模块加载时创建第一个 axios 实例。D-04 后该实例由 download.ts 创建。适配方向：download.test.ts / opsApi.test.ts 改为 `import { blobAxios } from "./download"` 直取实例（download.ts 必须导出实例本身），删除位置猜测。excel/blob 断言链（:335-423）逻辑不变。

---

### `src/lib/apiFactory.invariants.test.ts`（test, invariants 扫描）— 94-03 收口新建

**Analog（后端双档权威）:** `internal/services/system/cache_invariants_92_test.go:1-60`；**前端 fs 先例:** `src/design-system/tokens/colors.test.ts:233-252`

**双档分级模式（cache_invariants_92_test.go:11-16 注释原文——D-12 点名的对齐对象）**：

```go
// 断言双档（D-10② 与 Phase 89/90 硬锁惯例的调和）：
//   - 硬档：system + operations 残留必须 == allowedResidues（初始 0），超出即 fail
//   - warning 档：duty/knowledge/network/workorder 的同构残留仅计数日志不 fail
//     （A5 范围外站点，v1.30+ 迁移候选）
var allowedResidues = map[string]int{}  // 白名单：文件名 → 允许残留数，diff 可见
```

前端对等设计：**硬 fail 档** = opsApi/rpaApi/vdiApi（对象形态已接工厂，残留即回归）；**warning 档** = 迁移矩阵登记的 KEEP 模板 whitelist（knowledgeApi create/update、notificationConfigApi 整文件、assetApi.ts 整文件、duty/notice/adDomain/workorder 的异构 create/update），**收口时 warning 计数 == 白名单数**（防白名单腐烂）。

**node:fs 读源码模式（colors.test.ts:238-252 原文，含 ESLint typed-lint 豁免说明）**：

```typescript
import { readFileSync } from "node:fs";
import { dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";

// 注：ESLint typed-lint 程序未含 @types\node，node:fs/url 的调用被标记为
// unsafe —— 属工具链误报，测试运行时（vitest/node）类型完全正常，定向豁免。
function loadIndexCss(): string {
  const here = dirname(fileURLToPath(import.meta.url));
  return readFileSync(resolve(here, "../../index.css"), "utf-8");
}
```

**AST 选型（RESEARCH D-12 设计）**：用 `import ts from "typescript"` + `ts.createSourceFile`（typescript 已在 devDependencies，零新装），遍历 PropertyAssignment/FunctionDeclaration → body 单 ReturnStatement → CallExpression 标识符 ∈ {post,get,put,del} → URL 模板字符串后缀匹配 CRUD 五件套。**范围仅 `src/lib/*Api.ts`**（13 文件），豁免 `api.ts`/`apiFactory.ts`/`download.ts`；**禁止** grep 全仓 src（会误伤页面内联 post）。

---

### `src/lib/opsApi.ts`（lib/api-client, request-response + file-I/O）— 94-02 修改

**Analog:** 自身。动作全部是「删 + 换 import」，无新写逻辑。

1. **删私有工厂**（:41-97 整段）→ 顶部加 `import { createResourceApi } from "./apiFactory";`（D-03）
2. **11 资源实例零改动**——它们已是工厂调用形态，如 opsApi.ts:108 / :216 / :284 / :294 / :306：

```typescript
export const buildingApi = createCrudApi<Building>({ basePath: "/ops/building" });   // :108
export const serverRoomApi = createCrudApi<ServerRoom>({ basePath: "/ops/serverRoom" }); // :216
```

3. **spread 自定义方法零改动**（floorApi :120-128 是全仓库 spread 惯例的源头样本）：

```typescript
const floorCrudApi = createCrudApi<Floor>({ basePath: "/ops/floor" });   // :120

export const floorApi = {
  ...floorCrudApi,                                                        // :123
  tree: async () => {
    return await post<Floor[]>("/ops/floor/tree", {});                    // :125-127
  },
};
```

4. **blob 四件套迁出**（:310-369）→ `import { blobAxios, downloadFile, extractFilenameFromBlobResponse, triggerBrowserDownload } from "./download";`；excelApi（:371-404）/ deptApi（:414-418）/ assetApi.excel（:674-699）**导出位置不变、内部委托**（D-04）
5. **DropdownOption 迁 apiFactory.ts + 本文件 re-export 兜底**（:36-39）
6. **KEEP 不动区**：locationAliasApi（:183-205，pageNum/pageSize + scope 默认注入两处行为差异）、roomPhotoApi（:220-272）、wall/door/floorPlanText 的 `batch(action, ids)` 覆盖层（:442-448 等，D-07）、assetApi.statistics 覆盖（:662-668）、componentApi（:707-716）、workstationDeviceApi（:718-792）、geocode 函数族（:515-632）

---

### `src/lib/rpaApi.ts`（lib/api-client, request-response + file-I/O）— 94-02 修改

**Analog:** `opsApi.ts` 的 spread 实例模式 + 新 `download.ts`

1. **删第二份私有工厂**（:49-79 整段，5 方法版 + `CrudApiConfig<_T>` 无用泛型死代码一并消灭）→ 换 import apiFactory
2. **7 个 spread 实例换来源**（task :95 / worker :228 / execution :315 / schedule :400 / variable :466 / template :515 / notification :618），spread 层零改动。⚠ 已知副作用（Pitfall 4）：共享工厂 8 方法 → 各导出对象**新增** batch/statistics/searchOptions（additive，行为中性）；已核实 rpaApi.test.ts:192-208 只断言聚合对象 10 个子 API 名，无子 API 形状断言 ✓

```typescript
const taskCrudApi = createCrudApi<Task>({ basePath: "/rpa/tasks" });   // :95 现状
export const taskApi = { ...taskCrudApi, execute(...){...}, ... };     // spread 层零改动
```

3. **scriptApi 接入工厂（D-08 点名，手写五方法 :160-215 → spread+override）**：

```typescript
// 现状（rpaApi.ts:160-215）：手写 list/get/create/update/delete + testAction/format
// 迁移后形态：
const scriptCrud = createResourceApi<Script>({ basePath: "/rpa/scripts" });
export const scriptApi = {
  ...scriptCrud,             // 五方法签名与工厂完全同构（list 参数 PageParams 字面兼容）
  testAction: async (action: Action, url?: string) => { ... },   // :199-207 原样保留
  format: async (script: Script) => { ... },                      // :212-214 原样保留
};
```

4. **downloadReport 迁 download.ts（:355-379 裸 fetch 消灭）**——现状全文是要消灭的反面教材：`fetch(...)` 无超时 + 手写 createObjectURL 链。迁移后 `downloadFilePost` 一行；`getAuthHeaders` import（:6）是该文件唯一使用点，随之可删。taskApi/workerApi/executionApi 主体、aiApi（:564-606）、statisticsApi（:660-735）全部 KEEP。

---

### `src/lib/vdiApi.ts`（lib/api-client, request-response）— 94-03 修改

**Analog:** `opsApi.ts:636-700`（assetApi = spread + 个别方法 override 的精确同构先例）

**vmApi 迁移形态（spread+override，list/create 必须覆盖——tsc 实测依据）**：

```typescript
// 现状 vdiApi.ts:33-43 —— list/create 与工厂不同构：
//   list: (params: VMListParams) —— interface 类型变量直传消费点 VirtualMachineList/index.tsx:186-191，
//         工厂签名 PageParams & Record<string,unknown> 会拒绝 interface（TS2345，Pitfall 1 实测）
//   create: (data: CreateVMRequest) —— 含 vtp_id/count 等实体外字段，且缺 vm_id 必选字段（双向不可赋值）
const vmCrud = createResourceApi<VirtualMachine>({ basePath: "/vdi/vms" });
export const vmApi = {
  ...vmCrud,
  // override：原样保留现状签名（vdiApi.ts:33-35 / :41-43 一字不改）
  list: async (params: VMListParams) => post<VMPageResponse>("/vdi/vms/list", params),
  create: async (data: CreateVMRequest) => post<VirtualMachine>("/vdi/vms", data),
  // get/update/delete 来自 spread（类型兼容已验证；update 的 UpdateVMRequest ⊆ Partial）
};
```

**vdiServerApi 迁移形态（纯 spread，:144-168）**：五方法 + testConnection，其中五方法走 spread、`testConnection`（:165-167）作为自定义方法保留。**前提**：`CreatePayload` 排除集必须含 snake_case（`VDIServer` 用 `created_at/updated_at`，Pitfall 3）——94-01 的 types/apiFactory.ts 先行即是此依赖。类型 re-export 块（:172-190）零改动。

---

### 扁平文件 wrapper 委托（D-05 统一模式）— 94-03 修改

**适用文件：** workorderApi / knowledgeApi / dutyApi / noticeApi / adDomainApi

**委托模式（扁平文件唯一手法：文件内私有工厂实例 + 同构函数一行委托 + 导出签名零变化）**：

```typescript
// workorderApi.ts categories cluster 迁移形态（现状 :494-521）
const categoryCrud = createResourceApi<WorkOrderCategory>({ basePath: "/workorder/categories" });

export function getWorkOrderCategory(
  id: string
): Promise<BaseResponse<WorkOrderCategory>> {
  return categoryCrud.get(id);   // T=WorkOrderCategory，返回类型天然一致，零 cast
}
export function deleteWorkOrderCategory(id: string): Promise<BaseResponse<{ message: string }>> {
  return categoryCrud.delete(id) as Promise<BaseResponse<{ message: string }>>;  // 单 as 下转合法
}
export function createWorkOrderCategory(data: WorkOrderCategoryCreateRequest): ... {
  // KEEP 原样：请求类型解耦（WorkOrderCategoryCreateRequest 与 CreatePayload 不同构）
}
```

**逐文件委托清单（判定的库内证据）**：

| 文件 | DELEGATE cluster | KEEP 部分（证据行号） |
|------|-----------------|---------------------|
| `workorderApi.ts` | categories（:494-521，五方法全 POST 最干净）；periodic templates 的 list/get/delete（:531-571）；orders 的 list/get/delete（:385-429） | createWorkOrder/updateWorkOrder（:417-426，解耦请求类型）；batchDeleteWorkOrders（:432-436 `/batch-delete` 非 batch 形状）；assign/comments/ratings/config（:438-598）全异构 |
| `knowledgeApi.ts` | articles list/get/delete（:133-174）；categories 五方法（:186-213）；tags delete（:232-234） | create/update（:159-170，`KnowledgeArticleCreateRequest` 含 tagIds 实体外字段）→ D-12 warning 档 whitelist 项；search/like/convert（:176-184, :238-243） |
| `dutyApi.ts` | duty pools 的 list/get/delete（:155-193；getDutyPool 返回 `BaseResponse<DutyPool>` 与工厂 T 完全一致零 cast） | create/update（:176-189，`memberIds` 请求类型）；schedules 全异构（:197-251）；holidays（:255-284，list 参数 `year: number` 非 PageParams）；config 单例（:288-296）；getUserList 默认值注入（:319-331，`current:1,pageSize:1000` 前置——同 withDefaultPagination 性质，**不得委托**） |
| `noticeApi.ts` | admin notices list/get/delete（:23-76；statistics :41-43 亦可评估） | batchDeleteNotices（:81-83 `/batch-delete`）；getNoticeStatistics（:88-90 **GET 动词**，工厂 get 是 POST，动词不可改）；publish/withdraw（:95-104）；用户端 my-notices 全族（:111-169，GET 动词为主）；buildWebSocketUrl（:176-188） |
| `adDomainApi.ts` | ou-group-mappings（:580-605，五方法全 POST 全文件最干净）；mappings list/create/update（:482-498）；configs list/create/update/delete（:232-255） | getADConfig/getMapping（:242-244/:492-494 **GET 动词**）；groups/users/computers/logs 全族（id-in-body 非 REST 路径）；accounts 池（:674-731）；**⚠ 见下方 bug 登记项** |

**两个必须写进 plan 的特殊项：**

1. **withDefaultPagination 前置处理（Pitfall 5）**——adDomainApi.ts:224-228 本地泛型 helper，全部 list wrapper 先过它再 post。委托必须保留前置调用，工厂保持纯透传：

```typescript
// adDomainApi.ts:224-228 现状（helper 留在原文件）
const DEFAULT_PAGINATION = { current: 1, pageSize: 10 };
function withDefaultPagination<T extends { current?: number; pageSize?: number }>(params: T): T {
  return { ...DEFAULT_PAGINATION, ...params };
}
// 委托写法：crud.list(withDefaultPagination(params) as ...) —— 直接 crud.list(params) = 丢默认分页 = 行为变更
```

2. **adDomainApi:501 潜伏 bug（RESEARCH Open Question 2）**——`deleteMapping` 的 URL 带多余右括号：

```typescript
// adDomainApi.ts:500-502 —— `${id}/delete}` 末尾多一个 `}`，当前必然 404（潜伏 bug 实锤）
export function deleteMapping(id: string): Promise<BaseResponse<null>> {
  return post(`/ad-domain/mappings/${id}/delete}`, {});
}
```

委托后 URL 自然变对 = **隐性修复 = 行为变更**。plan 必须显式登记（v1.29 D-05 例外条款纪律：bugfix 附回归说明）+ 单独 commit 注明；或不修复而 KEEP 该函数 + whitelist 注明。**不建议**带着已知 bug 委托不登记。

**签名保持纪律（D-05 最硬承诺）**：所有导出函数保留显式返回类型注解；wrapper 体内若类型不兼容用 `as unknown as` 双跳（直接 `as` 在互不可赋时非法，Pitfall 1）；GET vs POST 端点细节保持各文件现状，不改后端契约。

---

### KEEP 文件（不动，whitelist 依据存档）

| 文件 | KEEP 依据（已核实原文） |
|------|----------------------|
| `notificationConfigApi.ts` | :58-60 list 参数用 `page`（非 current）；:63-65 get 用 **GET 动词**；:73-75 update 用 **PUT `/{id}`**（非 `/{id}/update`）；:78-80 delete 用 **DELETE 动词**——传输语义四处全部与工厂不同构，任何委托都是后端契约变更 |
| `assetApi.ts` | reconciliationApi/fixSuggestionApi 全部解包 `res.data` 返回裸数据，与工厂「返回 BaseResponse」契约不同 |
| `menuApi.ts` | :7-26 仅 3 个只读函数（my-menus 树/全量/权限），无 CRUD 语义 |
| `profileApi.ts` | :10-62 单例 profile + 密码 + 头像 + 偏好设置，无资源集合语义 |
| `columnConfigApi.ts` | :26-35 按 pageKey 存取（getByPageKey/save/reset），无标准 CRUD 路径形状 |

### `src/lib/opsApi.test.ts`（test）— 94-02 小幅适配

**Analog:** 自身。适配范围（D-14 预告的全部）：
- :99-101 `h.created[0]` / `blobRequestInterceptor()` 位置假设 → 改 `import { blobAxios } from "./download"` 直取
- :335-422 excel/blob 断言链逻辑不变（URL/参数断言原样）
- 其余 ~550 行零改动（工厂行为经 apiFactory.test.ts 另锁）

### `CLAUDE.md`（docs）— 94-03 收口（D-13）

**Analog:** CLAUDE.md 既有「### Cache Service Convention」段（Phase 92 收口先例：单一权威路径 + 规则列表 + Migration status 三段式）。新增「前端 API 工厂 Convention」段照此结构：锁 `createResourceApi` 单一权威路径（`src/lib/apiFactory.ts`）+ 新资源必须走工厂 + 模板清零约定 + D-12 扫描测试回归守护说明；同步修订「### Frontend API Calling」段补工厂用法。

---

## Shared Patterns

### 1. 传输层单一入口（所有工厂方法/wrapper 的底层）
**Source:** `src/lib/api.ts:526-570`
**Apply to:** apiFactory.ts、download.ts 外的全部 `*Api.ts`
```typescript
export function post<T = unknown>(url: string, data?: unknown): Promise<BaseResponse<T>> {
  return api.post(url, data);   // SM2+SM4 加密 / 401 刷新重放全在主实例拦截器（api.ts:225-522）
}
```
**红线：** 禁止绕过 api.ts 直连 axios（SM2 公钥轮换重放 api.ts:486-508 只在主实例）；blob 下载例外走 download.ts 的 blobAxios（无响应拦截器是 Blob 语义的必然，token 注入拦截器保留）。

### 2. `import type` 纪律（verbatimModuleSyntax 已开启）
**Source:** `opsApi.ts:6` / `dutyApi.ts:2` 等全部在库文件
**Apply to:** 全部新文件
```typescript
import type { BaseResponse, PageResponse, PageParams } from "@/types";
import { post } from "./api";   // 值导入不加 type
```

### 3. spread + override 扩展（对象形态接入工厂的唯一既定惯例）
**Source:** `opsApi.ts:120-128`（floorApi）、`:442-448`（wallApi batch 覆盖）、`:636-700`（assetApi statistics 覆盖）
**Apply to:** rpaApi scriptApi、vdiApi vmApi/vdiServerApi
```typescript
export const xxxApi = { ...crud, customFn(...){...}, overriddenFn: 原样实现 };
```

### 4. 扁平 wrapper 委托（签名零变化）
**Source:** RESEARCH Pattern 2 + `dutyApi.ts:155-193`（显式返回类型注解现状）
**Apply to:** 5 个扁平文件的 DELEGATE 函数
**纪律：** 导出函数保留显式返回类型注解；cast 用 `as unknown as` 双跳；wrapper 前置处理（withDefaultPagination / 默认参数注入）必须保留在委托之前。

### 5. 前置参数注入不进工厂
**Source:** `adDomainApi.ts:224-228`（withDefaultPagination）、`dutyApi.ts:319-331`（getUserList 默认 current:1/pageSize:1000）、`opsApi.ts:184-189`（locationAliasApi pageNum 默认）
**Apply to:** 所有含默认值/预处理逻辑的函数——一律 KEEP 原样或 wrapper 内前置调用，工厂保持纯透传。

### 6. 测试 mock 三件套
**Source:** `opsApi.test.ts:11-38`（vi.hoisted + 双路径 vi.mock）、`:41-55`（axios create 工厂）、`:103-115`（URL.createObjectURL 打桩）
**Apply to:** apiFactory.test.ts、download.test.ts

### 7. 双档 invariants 扫描（硬 fail + warning whitelist）
**Source:** `internal/services/system/cache_invariants_92_test.go:11-16, 31-33`（后端权威先例）+ `colors.test.ts:238-252`（前端 node:fs 读源码先例）
**Apply to:** apiFactory.invariants.test.ts（范围仅 `src/lib/*Api.ts`，豁免 api/apiFactory/download；白名单显式登记 + 计数 == 白名单数）

---

## No Analog Found

无。所有新建文件均有 exact/role-match 模拟（本 phase 的「新文件」全部是既有生产代码的提升或归一）。

** 半模拟备注（planner 注意，非缺失）：**
| 项 | 说明 |
|----|------|
| `downloadFilePost` | 三处内联同构归一的新签名（excelApi.export :385-396 为最佳提取源），非全新逻辑 |
| `CreatePayload<T>` | 双命名并集排除集是新类型，但 `Omit<T,...>` idiom 在 dutyApi.ts:259-263 已有在库先例 |
| TS AST 扫描器 | 前端首个 TS compiler API 测试，但 go/ast 同构先例（cache_invariants_92_test.go）+ typescript 包已在 devDependencies |

---

## Metadata

**Analog search scope:** `xingran-react-frontend/src/lib/`（全部 25 个 .ts）、`xingran-react-frontend/src/types/`、`xingran-react-frontend/src/design-system/tokens/`、`internal/services/system/`（Phase 92 先例）
**Files fully/t partially read:** 16 个源文件 + 3 个测试文件 + 1 个后端先例
**Pattern extraction date:** 2026-09-05
**与 RESEARCH 的一致性:** 迁移矩阵判定全部沿用 94-RESEARCH.md § 逐文件迁移矩阵；本文件补充的是每个判定的**代码级证据行号与照抄原文**，供 plan action 直接引用。
