# Phase 58: 前后端路由契约对齐 - Research

**Researched:** 2026-08-13
**Domain:** 前后端 HTTP 契约对齐（路由方法/路径 + JSON 字段命名）
**Confidence:** HIGH

## Summary

Phase 58 是一个**纯前端改动**的契约对齐 phase，后端零改动。研究逐行核验了 CONTEXT.md 引用的全部源码行号——**零漂移**，所有引用仍准确。两个契约断裂（CONTRACT-01 路由方法、CONTRACT-02 字段命名）均已用当前源码与一个实证 Go 程序验证为真，决策 D-01..D-06 的技术前提全部成立。

最具价值的发现：**D-02 的核心技术论断已通过实证 Go 程序确证**——`encoding/json` 的大小写不敏感匹配只折叠字母（`INHERITPERMS` 能绑到 `inheritPerms`），但**不折叠下划线**（`inherit_perms` 绑不到 `inheritPerms`，取零值）。这把"前端 snake_case 静默丢字段"从假设变成了可复现的事实。此外研究发现 CONTRACT-02 还有 CONTEXT.md 未列的**第三个后果**：列表排序白名单键是 camelCase（`isActive`/`expiresAt`/`lastUsedAt`），前端发 snake_case 的 `orderByColumn` 匹配不上 → 这些列的排序当前也静默失效，D-03 字段重命名会顺带修复。

改动面已完全测绘：`apikey.ts` 3 个函数体 + 2 个 import 移除；`types/apikey.ts` 3 个接口共 13 个字段名；`index.tsx` 约 25 处字段访问（record.X / dataIndex / Form.Item name / sortField 比较）。`getAPIKey` 确认零调用方（死代码），`updateAPIKey`/`deleteAPIKey` 各 1 个调用方。LogsModal.tsx 使用 Phase 59 范围的 `APIKeyUsageLog`/`UsageSummary` 类型，与本 phase 隔离。

**Primary recommendation:** 按 D-01（前端三函数改 POST + `/:id/update`、`/:id/delete` 路径）与 D-03（前端三接口 + 页面字段访问全量改 camelCase）执行；推荐 Form.Item name 也一并改 camelCase 以保持页面内部类型一致性。验证用 curl 跑 SC#1-SC#3 的 HTTP 断言 + 复用既有 `apikey_service_test.go` 作回归守护。

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions
- **D-01 (CONTRACT-01 对齐方向):** Option A — 前端 `apikey.ts` 三个操作改 POST 对齐后端既有路由。getAPIKey → `POST /:id`、updateAPIKey → `POST /:id/update`、deleteAPIKey → `POST /:id/delete`。后端零改动。改动局限在 3 个函数体（方法 + 路径）。
- **D-02 (CONTRACT-02 吸收):** Phase 58 端到端吸收字段命名断裂（新缺陷 CONTRACT-02）。后端 camelCase vs 前端 snake_case；`api.ts` 拦截器无 snake↔camel 转换。后果：① Create/Update 绑定静默取零值丢字段；② List/详情复合字段显示 undefined。
- **D-03 (字段命名方向):** 前端 → camelCase（审计确认后端 camelCase 是全项目约定，FE snake 是孤例）。改 `types/apikey.ts` + `index.tsx`。后端零改动（`maskAPIKeys` List map 与 GetByID 原始 struct 均已 camelCase）。
- **D-04 (范围限定):** 字段命名修复范围限定 `APIKey` / `CreateAPIKeyRequest` / `UpdateAPIKeyRequest` + `index.tsx` 相关字段访问。`APIKeyUsageLog` / `UsageSummary` 留 Phase 59。
- **D-05 (编辑详情来源):** 编辑表单继续复用列表行 `editingRecord` 回填。`getAPIKey` 契约修对（POST /:id）保持可用但不接入编辑流。
- **D-06 (删除错误语义):** 无新代码决策——SC#3 由 D-01 对齐 + 既有后端逻辑覆盖（GetAPIKey/DeleteAPIKey 已返回 `CodeParamError "密钥不存在"`）。planner 仅需验证重复删除/再访问返回 `code != 0` + "密钥不存在" 而非 404。

### Claude's Discretion
- 未用 import 清理：Option A 后 `put` / `del` 从 `apikey.ts` 移除；`get` 保留（仍被 `getUsageSummary` 使用）。
- `types/apikey.ts` 字段名 → camel 的逐字段映射（planner 对照 `models/api_key.go` 与 `apikey_request.go` 的 json tag 逐一改）。
- 是否补充契约对齐的前端单测（vitest）由 planner 按 `.planning/codebase/TESTING.md` 决定。

### Deferred Ideas (OUT OF SCOPE)
- `APIKeyUsageLog` / `UsageSummary` 类型字段命名 snake→camel → Phase 59（OBSERV）。
- `getAPIKey` 接入编辑流（拉取最新详情） → 本 phase 不做（D-05）。
- 密钥轮换 / 吊销、配额告警 → FUTURE-APIKEY-03/04。
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| CONTRACT-01 | 前端 `getAPIKey` / `updateAPIKey` / `deleteAPIKey` 三个操作不再 404 — 与后端 `apikey_router.go` 注册的路由方法/路径对齐 | `apikey_router.go:18-25` 全 POST 已核验；`apikey.ts:87/108/124` 当前 GET/PUT/DELETE 已核验；D-01 改 POST 方向已验证后端零改动可行（见「路由契约对齐核验」） |
| CONTRACT-02 | 前后端字段命名契约对齐 — 后端 camelCase vs 前端 snake_case，`api.ts` 无转换层；修复方向=前端→camelCase，后端零改动 | Go json 行为已实证（见「Go json 大小写行为实证」）；字段映射表已逐字段对照 `models/api_key.go` + `apikey_request.go` json tag 建立（见「前端字段映射」）；`api.ts:207-274/277+` 拦截器确认无转换层 |
</phase_requirements>

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| API Key CRUD 路由定义（方法/路径） | API / Backend (`apikey_router.go`) | Frontend Client (`apikey.ts`) | 路由契约的 source of truth 是后端注册路径；前端必须对齐（D-01） |
| JSON 字段命名契约 | API / Backend (`models/*.go` json tag) | Frontend Client (`types/apikey.ts`) | 后端 json tag 是 wire-format 契约基准；前端类型必须镜像（D-03） |
| 请求/响应字段转换 | Frontend Client (`lib/api.ts` 拦截器) | — | 当前只做加密/解密，无命名转换；本 phase 不加转换层而是改前端字段名 |
| 编辑表单数据回填 | Browser / Client (`index.tsx`) | — | D-05 复用列表行 `editingRecord`，不发额外请求 |
| not-found 错误语义 | API / Backend (`apikey_service.go`) | — | GetAPIKey/DeleteAPIKey 已返回 `CodeParamError`，前端只消费（D-06） |

**关键边界：** 本 phase 改动 100% 在 Frontend Client 与 Browser 两层（`apikey.ts` + `types/apikey.ts` + `index.tsx`）。API/Backend 与 Database 两层零改动——既不在本 phase 加 RESTful 方法（D-01 选 A），也不改后端 json tag（D-03 选前端对齐）。

## Standard Stack

本 phase 不引入新依赖，复用现有前端栈。版本来自 `xingran-react-frontend/package.json`。

### Core（现有，复用）
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| axios | (现有) | `@/lib/api` 的 `get`/`post` 封装底层 | 项目统一 API 客户端，CLAUDE.md 强制「不用裸 axios，用 @/lib/api 封装」 `[CITED: CLAUDE.md]` |
| Ant Design | 6.1 | `Form` / `Table` / `Popconfirm` 组件 | 项目 UI 标准组件库 `[CITED: CLAUDE.md]` |
| React + TypeScript | 19.2 / 5.9 | 类型系统 + 页面组件 | 项目前端框架 `[CITED: CLAUDE.md]` |

### Supporting（验证用，现有）
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| Vitest | (现有) | 前端单测（可选契约对齐断言） | planner 按 TESTING.md 决定是否补单测（见「Validation Architecture」） |
| testify | v1.11.1 | 后端回归测试 | 复用既有 `apikey_service_test.go`，本 phase 不改后端 |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| 前端改 camelCase 字段（D-03） | 在 `api.ts` 加全局 snake↔camel 转换层 | 全局转换层会影响所有模块（User/Dept 已 camel，会被二次转换破坏）；CONTEXT.md 已否决，改字段名是孤例修复 `[CITED: 58-CONTEXT.md D-03]` |
| 前端改 POST（D-01） | 后端补 RESTful GET/PUT/DELETE | 后端违背全项目 Router Pattern（CLAUDE.md/CONVENTIONS.md 全 POST + /:id/update）；影响面更大 `[CITED: 58-CONTEXT.md D-01]` |

**Installation:**
```bash
# 无新依赖安装 —— 纯编辑现有 .ts/.tsx 文件
```

**Version verification:** 本 phase 不安装包，跳过 registry 校验。所有依赖为现有锁定版本。

## Package Legitimacy Audit

本 phase **不安装任何外部包**（纯编辑现有 `apikey.ts` / `types/apikey.ts` / `index.tsx`）。Package Legitimacy Gate 不适用——没有新增 npm 依赖需要 slopcheck / registry 校验 / postinstall 审计。

## Architecture Patterns

### System Architecture Diagram

```
 当前（断裂）数据流：
 ┌─────────────────┐     GET /:id          ┌──────────────────┐
 │ index.tsx       │──── updateAPIKey ────►│ Gin Router       │
 │ (编辑/删除按钮)  │     PUT /:id          │ apikey_router.go │── 方法不匹配 ──► 404
 │                 │     DELETE /:id        │ (全 POST 注册)   │
 └─────────────────┘                       └──────────────────┘
        │
        │  Create/Update body: {inherit_perms, ip_whitelist, expires_at}  (snake)
        ▼
 ┌─────────────────┐   shouldBindJSON      ┌──────────────────┐
 │ api.ts 拦截器   │ ──────────────────►   │ Go encoding/json │── 下划线不折叠 ──► 取零值（静默丢字段）
 │ (仅加密,无转换)  │                       │ → CreateAPIKey   │
 └─────────────────┘                       │   Request{...}   │
                                           └──────────────────┘

 对齐后（D-01 + D-03）数据流：
 ┌─────────────────┐     POST /:id           ┌──────────────────┐
 │ index.tsx       │──── POST /:id/update ─►│ Gin Router       │── 方法匹配 ──► 200 code:0
 │ (camelCase)     │     POST /:id/delete    │ apikey_router.go │
 └─────────────────┘                        └──────────────────┘
        │
        │  Create/Update body: {inheritPerms, ipWhitelist, expiresAt}  (camel)
        ▼
 ┌─────────────────┐                        ┌──────────────────┐
 │ api.ts 拦截器   │ ──────────────────►    │ Go encoding/json │── 精确匹配 ──► 字段正确绑定
 │ (SM2/SM4 加密)  │                        │ → CreateAPIKey   │
 └─────────────────┘                        │   Request{...}   │── 不存在 ──► code:1001 "密钥不存在"
                                            └──────────────────┘   (非 404)
```

### Recommended Project Structure
```
xingran-react-frontend/src/
├── api/apikey.ts          # [CONTRACT-01] 3 函数改 POST + 移除 put/del import
├── types/apikey.ts        # [CONTRACT-02] APIKey/Create/Update 接口 → camelCase
└── pages/system/apikeys/
    └── index.tsx          # [CONTRACT-02] record.X / dataIndex / Form.Item / sortField → camelCase
```
（LogsModal.tsx 不改 —— 使用 Phase 59 范围的 APIKeyUsageLog/UsageSummary）

### Pattern 1: 标准 Router Pattern（后端基准，不改）
**What:** 全 POST + `/:id/update` `/:id/delete` 后缀
**When to use:** 所有 system 模块路由
**Example:**
```go
// Source: internal/api/v1/system/apikey_router.go:18-25 [VERIFIED: codebase]
r.POST("", apikeyHandler.Create)
r.POST("/list", apikeyHandler.List)
r.POST("/:id", apikeyHandler.GetByID)
r.POST("/:id/update", apikeyHandler.Update)
r.POST("/:id/delete", apikeyHandler.Delete)
r.POST("/:id/toggle", apikeyHandler.ToggleStatus)
r.POST("/:id/logs", apikeyHandler.ListUsageLogs)
r.GET("/:id/summary", apikeyHandler.GetUsageSummary)  // 唯一 GET
```
`[CITED: .planning/codebase/CONVENTIONS.md §Route Pattern]` 与 `apikey_router.go` 完全一致。

### Pattern 2: 前端 API 封装（对齐后）
**What:** 用 `@/lib/api` 的 `post`，路径与方法对齐后端
**Example:**
```typescript
// Source: 对齐后的 apikey.ts（D-01 目标） [VERIFIED: codebase 当前状态 + 路由基准]
import { get, post } from "@/lib/api";  // put/del 移除

export function getAPIKey(id: string): Promise<BaseResponse<APIKey>> {
  return post(`/system/apikeys/${id}`);                    // GET → POST
}
export function updateAPIKey(id: string, data: UpdateAPIKeyRequest): Promise<BaseResponse<void>> {
  return post(`/system/apikeys/${id}/update`, data);       // PUT /:id → POST /:id/update
}
export function deleteAPIKey(id: string): Promise<BaseResponse<void>> {
  return post(`/system/apikeys/${id}/delete`);             // DELETE /:id → POST /:id/delete（无 body）
}
export function getUsageSummary(keyID: string): Promise<BaseResponse<UsageSummary>> {
  return get(`/system/apikeys/${keyID}/summary`);          // 不改（后端 r.GET /:id/summary）
}
```

### Pattern 3: camelCase JSON 字段契约（对齐后）
**What:** 前端类型字段名镜像后端 Go struct json tag
**Example:**
```typescript
// Source: 对齐后的 types/apikey.ts（D-03 目标），对照 models/api_key.go:8-23 [VERIFIED: codebase]
export interface APIKey {
  id: string;
  name: string;
  key: string;
  scopes: string[];
  ipWhitelist: string[];      // ← ip_whitelist
  inheritPerms: boolean;      // ← inherit_perms
  expiresAt?: string;         // ← expires_at
  lastUsedAt?: string;        // ← last_used_at
  isActive: boolean;          // ← is_active
  description?: string;
  createdAt: string;          // ← created_at (BaseModel.CreatedAt json:"createdAt")
  updatedAt: string;          // ← updated_at (BaseModel.UpdatedAt json:"updatedAt")
}
```

### Anti-Patterns to Avoid
- **在 `api.ts` 加全局 snake↔camel 转换层：** User/Dept 等模块已 camelCase，全局转换会破坏它们。CONTRACT-02 是 apikey 模块孤例，改字段名即可。`[CITED: 58-CONTEXT.md D-03]`
- **改后端 json tag 接受 snake_case：** 违背后端 camelCase 全项目约定；且 `apikey_service_test.go` 直接用 Go struct field，改 tag 不影响测试但不一致。`[CITED: 58-CONTEXT.md D-03]`
- **改 Form.Item name 但不改 dataIndex（或反之）：** antd Table 排序 `sorter.field` 取自 `dataIndex`，若 dataIndex 与发往后端的 `orderByColumn`（camelCase 白名单）不一致，排序静默失效（见 Pitfall 3）。
- **删 `get` import：** `getUsageSummary`（行 195）与后端 `r.GET /:id/summary`（router 行 25）仍是 GET，`get` 必须保留。

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| snake↔camel 字段转换 | 自写递归 key 转换函数 | 改前端字段名为 camelCase（D-03） | 转换层全局生效会破坏已 camel 的模块；apikey 是孤例，直接改字段名最简 `[CITED: 58-CONTEXT.md]` |
| 路由方法对齐 | 后端补 RESTful GET/PUT/DELETE handler | 前端改 POST（D-01） | 后端已遵循全项目 Router Pattern；前端是唯一偏离，改前端影响面最小（3 函数体） |
| not-found 业务错误 | 在前端加 404→业务错误翻译 | 后端既有 `apperrors.CodeParamError "密钥不存在"` | D-06 确认后端已就绪，前端只需方法对齐让请求到达 service 层 `[VERIFIED: apikey_service.go:296-297, 363-364]` |

**Key insight:** 这个 phase 的全部价值在于**删除**手写的偏离（前端 GET/PUT/DELETE + snake_case 类型），让前端回归项目既有约定。没有任何新逻辑要造。

## Go json 大小写行为实证（D-02 验证）

**研究方法：** 用项目已安装的 Go 工具链跑一个最小复现程序（`/tmp/gojson_test/main.go`），实证 `encoding/json.Unmarshal` 对不同命名风格 JSON key 的绑定行为。`[VERIFIED: empirical Go test, Go 1.24]`

**结果：**
```
[exact camelCase]           InheritPerms=true  IPWhitelist=[1.2.3.4]  IsActive=true  ExpiresAt="2026-12-31"
[snake_case (current FE)]   InheritPerms=false IPWhitelist=[]          IsActive=false ExpiresAt=""        ← 全取零值
[case-folded letters]       InheritPerms=true  IPWhitelist=[]          IsActive=false ExpiresAt=""
[PascalCase]                InheritPerms=true  IPWhitelist=[]          IsActive=false ExpiresAt=""
```

**结论（确证 D-02 论断）：**
1. `inherit_perms`（snake）→ 绑不到 `InheritPerms json:"inheritPerms"`（camel），**取零值 false**。下划线不是字母，不参与大小写折叠。
2. `INHERITPERMS`（全大写，无下划线）→ **能绑定**（大小写不敏感匹配生效）。
3. 所以 Go json 的 case-insensitive 匹配**只折叠字母大小写，不忽略下划线**。`inherit_perms` vs `inheritPerms` 在下划线位置字符不同（`_` 0x5F vs `P` 0x50），永不匹配。

**对 Phase 58 的含义：** 这证明 CONTRACT-02 后果①（Create/Update 静默丢复合字段）是**真实的、可复现的数据损坏**，不是假设。修复必须改字段名（D-03 前端→camel），无法靠"Go 大小写不敏感"侥幸绕过。后端 json tag 不改（D-03）。

## 路由契约对齐核验（CONTRACT-01 / D-01）

### 后端契约基准（不改）`[VERIFIED: codebase]`
`internal/api/v1/system/apikey_router.go:18-25`：
- `POST ""` → Create
- `POST "/list"` → List
- `POST "/:id"` → GetByID
- `POST "/:id/update"` → Update
- `POST "/:id/delete"` → Delete
- `POST "/:id/toggle"` → ToggleStatus
- `POST "/:id/logs"` → ListUsageLogs
- `GET "/:id/summary"` → GetUsageSummary（唯一 GET，前端 `getUsageSummary` 已对齐，不改）

### 前端当前状态（待改）`[VERIFIED: codebase]`
`xingran-react-frontend/src/api/apikey.ts`：
| 函数 | 行 | 当前 | 目标（D-01） |
|------|----|------|--------------|
| `getAPIKey` | 86-88 | `get(`/system/apikeys/${id}`)` | `post(`/system/apikeys/${id}`)` |
| `updateAPIKey` | 107-109 | `put(`/system/apikeys/${id}`, data)` | `post(`/system/apikeys/${id}/update`, data)` |
| `deleteAPIKey` | 123-125 | `del(`/system/apikeys/${id}`)` | `post(`/system/apikeys/${id}/delete`)`（无 body） |

### Import 变更 `[VERIFIED: codebase]`
`apikey.ts:15` 当前 `import { get, post, put, del } from "@/lib/api";`
- `put` → 移除（仅 updateAPIKey 用过，已改 post）
- `del` → 移除（仅 deleteAPIKey 用过，已改 post）
- `get` → **保留**（`getUsageSummary` 行 195 仍用 `get`，对应后端唯一 GET 路由）
- `post` → 保留（改后被更多函数复用）

### 调用方影响 `[VERIFIED: codebase grep]`
| 函数 | 调用方数 | 调用位置 |
|------|---------|---------|
| `getAPIKey` | **0**（死代码，仅定义 + JSDoc 示例） | — |
| `updateAPIKey` | 1 | `index.tsx:325`（handleSubmit 编辑分支） |
| `deleteAPIKey` | 1 | `index.tsx:262`（handleDelete） |
| `getUsageSummary` | 1 | `LogsModal.tsx:133`（不改，Phase 59 范围） |

`getAPIKey` 零调用方确证 D-01「死代码」判断；其函数体仍按 D-01 改 POST 保持契约正确（D-05：不接入编辑流但契约要对）。

## 前端字段映射（CONTRACT-02 / D-03 / D-04）

以下三表逐字段对照后端 json tag，planner 直接据此改 `types/apikey.ts`。`[VERIFIED: codebase — models/api_key.go:8-23, models/base.go:11-19, models/system/requests/apikey_request.go:8-27]`

### 表 1: `APIKey` 接口（types/apikey.ts:12-25）
| 当前 (snake) | 目标 (camel) | 后端来源（json tag） |
|--------------|--------------|---------------------|
| `id` | `id` | `BaseModel.ID` json:"id" |
| `name` | `name` | `APIKey.Name` json:"name" |
| `key` | `key` | `APIKey.Key` json:"key" |
| `scopes` | `scopes` | `APIKey.Scopes` json:"scopes" |
| `ip_whitelist` | **`ipWhitelist`** | `APIKey.IPWhitelist` json:"ipWhitelist" |
| `inherit_perms` | **`inheritPerms`** | `APIKey.InheritPerms` json:"inheritPerms" |
| `expires_at?` | **`expiresAt?`** | `APIKey.ExpiresAt` json:"expiresAt,omitempty" |
| `last_used_at?` | **`lastUsedAt?`** | `APIKey.LastUsedAt` json:"lastUsedAt,omitempty" |
| `is_active` | **`isActive`** | `APIKey.IsActive` json:"isActive" |
| `description?` | `description?` | `APIKey.Description` json:"description,omitempty" |
| `created_at` | **`createdAt`** | `BaseModel.CreatedAt` json:"createdAt" |
| `updated_at` | **`updatedAt`** | `BaseModel.UpdatedAt` json:"updatedAt" |

### 表 2: `CreateAPIKeyRequest` 接口（types/apikey.ts:30-37）
| 当前 (snake) | 目标 (camel) | 后端来源（json tag） |
|--------------|--------------|---------------------|
| `name` | `name` | `CreateAPIKeyRequest.Name` json:"name" |
| `description?` | `description?` | `.Description` json:"description" |
| `scopes` | `scopes` | `.Scopes` json:"scopes" |
| `inherit_perms` | **`inheritPerms`** | `.InheritPerms` json:"inheritPerms" |
| `ip_whitelist?` | **`ipWhitelist?`** | `.IPWhitelist` json:"ipWhitelist" |
| `expires_at?` | **`expiresAt?`** | `.ExpiresAt` json:"expiresAt" |

### 表 3: `UpdateAPIKeyRequest` 接口（types/apikey.ts:42-49）
| 当前 (snake) | 目标 (camel) | 后端来源（json tag） |
|--------------|--------------|---------------------|
| `name?` | `name?` | `UpdateAPIKeyRequest.Name` json:"name" |
| `description?` | `description?` | `.Description` json:"description" |
| `scopes?` | `scopes?` | `.Scopes` json:"scopes" |
| `inherit_perms?` | **`inheritPerms?`** | `.InheritPerms` json:"inheritPerms" |
| `ip_whitelist?` | **`ipWhitelist?`** | `.IPWhitelist` json:"ipWhitelist" |
| `is_active?` | **`isActive?`** | `.IsActive` json:"isActive" |

**注意：**
- 后端 `UpdateAPIKeyRequest` 还有 `ExpiresAt *string json:"expiresAt"`（apikey_request.go:26），但前端 `UpdateAPIKeyRequest` 当前无此字段（编辑表单未暴露过期时间修改，仅 create 表单有 `expires_at`）。**这不是契约断裂**，是前端功能缺口，**不在本 phase 范围**（不新增字段）。
- `APIKeyListParams`（types/apikey.ts:54-62）已是 camelCase（`orderByColumn`/`isAsc`/`current`/`pageSize`），**无需改动**。
- `APIKeyUsageLog` / `UsageSummary`（types/apikey.ts:69-94）是 snake_case 但属 **Phase 59 范围**（D-04），**本 phase 不改**。

## index.tsx 字段访问改动清单（CONTRACT-02）

`index.tsx` 中所有 snake_case 字段访问，按类别分组。planner 据此逐处改 camelCase。`[VERIFIED: codebase — index.tsx]`

### 类别 A: 数据契约字段（必须改，影响 wire format 与列表渲染）
| 行 | 当前 | 目标 |
|----|------|------|
| 244 | `record.inherit_perms` | `record.inheritPerms` |
| 245-246 | `record.ip_whitelist` | `record.ipWhitelist` |
| 249 | `record.expires_at` | `record.expiresAt` |
| 250 | `record.last_used_at` | `record.lastUsedAt` |
| 251 | `record.created_at` | `record.createdAt` |
| 277 | `record.is_active` | `record.isActive` |
| 535, 539 | `record.is_active` | `record.isActive` |
| 436-437 | `dataIndex/key: "inherit_perms"` | `"inheritPerms"` |
| 448-449 | `dataIndex/key: "ip_whitelist"` | `"ipWhitelist"` |
| 465-466 | `dataIndex/key: "is_active"` | `"isActive"` |
| 479-480 | `dataIndex/key: "expires_at"` | `"expiresAt"` |
| 500-501 | `dataIndex/key: "last_used_at"` | `"lastUsedAt"` |

### 类别 B: 排序白名单匹配（必须改，否则排序静默失效 —— 见 Pitfall 3）
| 行 | 当前 | 目标 |
|----|------|------|
| 470 | `sortField === "is_active"` | `sortField === "isActive"` |
| 483 | `sortField === "expires_at"` | `sortField === "expiresAt"` |
| 504 | `sortField === "last_used_at"` | `sortField === "lastUsedAt"` |

### 类别 C: create/update data 构建（必须改，否则绑定失败）
| 行 | 当前 | 目标 |
|----|------|------|
| 307 | `inherit_perms: values.inherit_perms` | `inheritPerms: values.inheritPerms` |
| 308 | `ip_whitelist: ipWhitelist` | `ipWhitelist: ipWhitelist` |
| 309 | `expires_at: values.expires_at...` | `expiresAt: values.expiresAt...` |
| 321 | `inherit_perms: values.inherit_perms` | `inheritPerms: values.inheritPerms` |
| 322 | `ip_whitelist: ipWhitelist` | `ipWhitelist: ipWhitelist` |

### 类别 D: Form.Item name + setFieldsValue（planner 决策点 —— 见 Open Questions Q1）
| 行 | 当前 | 目标（推荐） |
|----|------|--------------|
| 209 | `inherit_perms: true`（setFieldsValue） | `inheritPerms: true` |
| 226 | `inherit_perms: record.inherit_perms` | `inheritPerms: record.inheritPerms` |
| 227 | `ip_whitelist: record.ip_whitelist?.join` | `ipWhitelist: record.ipWhitelist?.join` |
| 294-298 | `values.ip_whitelist`（handleSubmit 解析） | `values.ipWhitelist` |
| 731 | `Form.Item name="inherit_perms"` | `name="inheritPerms"` |
| 739 | `Form.Item name="ip_whitelist"` | `name="ipWhitelist"` |
| 749 | `Form.Item name="expires_at"` | `name="expiresAt"` |

## 后端契约基准确认（D-03 / D-06，零改动）

### List 响应已是 camelCase `[VERIFIED: apikey_handler.go:323-343]`
`maskAPIKeys` 显式构造 map，键全 camelCase：`userId` / `expiresAt` / `lastUsedAt` / `isActive` / `scopes` / `ipWhitelist` / `inheritPerms` / `createdAt` / `updatedAt`。前端类型改 camel 后两端一致。

### GetByID 响应已是 camelCase `[VERIFIED: apikey_handler.go:127-145 + models/api_key.go]`
返回原始 `*models.APIKey` struct（仅 mask Key），其 json tag 全 camelCase（含 BaseModel 的 `createdAt`/`updatedAt`）。前端类型改 camel 后 `getAPIKey` 响应可正确反序列化。

### not-found 错误语义已就绪 `[VERIFIED: apikey_service.go:296-297, 363-364 + pkg/errors/codes.go]`
- `GetAPIKey` not-found → `apperrors.CodeParamError "密钥不存在"`（service 行 296-297）
- `DeleteAPIKey` not-found → `apperrors.CodeParamError "密钥不存在"`（service 行 363-364）
- `CodeParamError = 1001`（codes.go:13）→ HTTP 400（`c >= 1000 && c < 1020 → StatusBadRequest`，codes.go:203-204）
- **即：** 对齐后 not-found/软删记录返回 HTTP 400 + body `{code:1001, message:"密钥不存在"}`，而非 Gin 路由缺失的 HTTP 404。这正是 SC#3 要验证的语义。

## 源码行号核验（CONTEXT.md 引用 vs 当前）

**结论：零漂移。** CONTEXT.md `<canonical_refs>` 与 `<specifics>` 引用的所有行号经逐行核对仍准确。`[VERIFIED: codebase]`

| 引用 | CONTEXT.md 称 | 实际 | 状态 |
|------|--------------|------|------|
| apikey.ts 三函数 | — | getAPIKey:86-88, updateAPIKey:107-109, deleteAPIKey:123-125 | ✓ |
| index.tsx 244/246/277 | record.inherit_perms/ip_whitelist/is_active | 244, 245-246, 277 | ✓ |
| index.tsx 307-323/325 | create/update data 构建 + updateAPIKey 调用 | 307-310, 317-323, 325 | ✓ |
| index.tsx 551-566 | Popconfirm 删除确认 | 551-566 | ✓ |
| api.ts 207-274 | 请求拦截器（仅加密无命名转换） | 207-274 | ✓ |
| api.ts 277+ | 响应拦截器（仅解密无命名转换） | 277+ | ✓ |
| apikey_handler.go 127/159/195/323-343 | GetByID/Update/Delete/maskAPIKeys | 127, 159, 195, 323-343 | ✓ |
| apikey_service.go 296/363 | GetAPIKey/DeleteAPIKey not-found 分支 | 296-297, 363-364 | ✓ |

## SM2/SM4 加密交互确认

D-01 把 getAPIKey/updateAPIKey/deleteAPIKey 从 GET/PUT/DELETE 改 POST 后，加密行为变化：`[VERIFIED: api.ts:186-204 shouldEncryptRequest]`

| 函数 | 当前方法 | 当前加密 | 改后方法 | 改后加密 | 影响 |
|------|---------|---------|---------|---------|------|
| getAPIKey | GET | 否（GET 不加密） | POST | 有 body 才加密；此函数无 body → 不加密 | 无 |
| updateAPIKey | PUT | 是（PUT 在加密集） | POST | 是（POST 在加密集，有 body） | 无（加密路径不变） |
| deleteAPIKey | DELETE | 否（DELETE 不加密） | POST | 有 body 才加密；此函数无 body → 不加密 | 无 |

**结论：** 改 POST 后无加密副作用。`shouldEncryptRequest`（api.ts:191）只对 `["POST","PUT","PATCH"]` 且 `config.data` 非空的请求加密。getAPIKey/deleteAPIKey 改 POST 后因无 body 不触发加密；updateAPIKey 改 POST 后仍走原 PUT 的加密路径（后端请求加密中间件解密后 `ShouldBindJSON`，行为不变）。

## Common Pitfalls

### Pitfall 1: 改类型字段名但漏改 dataIndex（排序静默失效）
**What goes wrong:** antd Table 列排序时 `sorter.field` 取自列的 `dataIndex`。若只改 `record.ip_whitelist`→`record.ipWhitelist` 但 `dataIndex` 仍为 `"ip_whitelist"`，则：(a) 列渲染读不到值（dataIndex 指向不存在的字段）；(b) `sorter.field="ip_whitelist"` 发给后端作 `orderByColumn`，匹配不上 camelCase 白名单 → 排序失效。
**Why it happens:** `dataIndex` 是字符串字面量，TypeScript 不报错（`ColumnsType<APIKey>` 的 dataIndex 是 ` keyof APIKey \| ...`，但实际编译常宽松）。
**How to avoid:** 改 `record.X` 的同时**必须**改对应列的 `dataIndex` 与 `key`（见类别 A 表）。planner 应把「列 dataIndex 改名」作为单条 task 的 done-criteria。
**Warning signs:** 列表能显示但点列头排序无反应 / 列空白。

### Pitfall 2: Form.Item name 与 values key 耦合（构造数据时 values.X 取不到）
**What goes wrong:** `handleSubmit` 的 `values = await form.validateFields()` 返回的 key 等于 `Form.Item name`。若 Form.Item name 改 camel 但 handleSubmit 读 `values.inherit_perms`（旧 snake），得 undefined。
**Why it happens:** Form 字段名是隐式契约，跨三个函数（handleAdd setFieldsValue / handleEdit setFieldsValue / handleSubmit validateFields）。
**How to avoid:** Form.Item name、setFieldsValue key、validateFields 读取的 key **三者必须一致**。推荐全改 camelCase（类别 D），保持 `values.inheritPerms` ↔ `Form.Item name="inheritPerms"` ↔ `{inheritPerms: values.inheritPerms}` 一致。
**Warning signs:** 编辑表单字段空白 / 提交后字段变默认值。

### Pitfall 3: 排序白名单键 camelCase（CONTRACT-02 第三后果）
**What goes wrong:** 后端 `apiKeyAllowedSortFields`（apikey_service.go:19-26）的 key 是 camelCase：`isActive`/`expiresAt`/`lastUsedAt`/`createdAt`/`updatedAt`/`name`。前端当前发 `orderByColumn: "is_active"`（snake，来自 dataIndex），匹配不上 → `ApplySort` 跳过 → 回退默认 `created_at DESC`。即：状态/过期/最后使用 三列的排序**当前静默失效**。
**Why it happens:** 前端 dataIndex snake_case 与后端白名单 camelCase 不一致。
**How to avoid:** 类别 A + B 改完后，dataIndex 变 camelCase → sorter.field 变 camelCase → orderByColumn 变 camelCase → 匹配白名单。**这是 D-03 字段重命名的顺带修复**，无需额外代码。
**Warning signs:** 改字段名后重新验证：点列头排序，检查网络请求 `orderByColumn` 是否为 camelCase 且结果顺序变化。

### Pitfall 4: 误删 `get` import
**What goes wrong:** 把 `get` 连同 `put`/`del` 一起删，导致 `getUsageSummary`（行 195）编译失败。
**Why it happens:** D-01 说"三个函数改 POST"，容易过度推断所有 HTTP 动词 import 都删。
**How to avoid:** 只删 `put`/`del`。`getUsageSummary` 对应后端 `r.GET "/:id/summary"`（router 行 25），保持 GET。`get` 必须保留。

## Code Examples

### 例 1: deleteAPIKey 改 POST（无 body 不触发加密）
```typescript
// Source: D-01 目标 [VERIFIED: apikey.ts:123-125 当前状态 + apikey_router.go:22]
// 改前:  return del(`/system/apikeys/${id}`);
// 改后:
export function deleteAPIKey(id: string): Promise<BaseResponse<void>> {
  return post(`/system/apikeys/${id}/delete`);  // 无第二参 → config.data 为 undefined → 不加密
}
```

### 例 2: not-found 错误响应（SC#3 断言基准）
```bash
# Source: apikey_service.go:296-297,363-364 + pkg/errors/codes.go:13,203-204 [VERIFIED]
# 删除后再访问（软删记录）的预期响应：
HTTP/1.1 400 Bad Request
{ "code": 1001, "message": "密钥不存在", "data": null, "timestamp": ..., "request_id": "..." }
# 注意：code=1001（CodeParamError），HTTP=400 —— 这是「明确业务错误」
# 对比当前（契约未对齐）：HTTP 404 + Gin 路由缺失（方法不匹配）—— 这才是要消除的
```

## Validation Architecture

> `workflow.nyquist_validation: true`（config.json）。本节描述 SC#1-SC#4 的可测断言，供后续生成 VALIDATION.md。

### Test Framework
| Property | Value |
|----------|-------|
| Framework (前端) | Vitest（vitest.config.ts 已配置，jsdom + setupFiles `./src/test/setup.ts` + `@` alias） |
| Framework (后端) | Go testing + testify v1.11.1（既有，本 phase 不改后端） |
| Config file | `xingran-react-frontend/vitest.config.ts` / Go 标准 `_test.go` 同包 |
| Quick run command (前端) | `cd xingran-react-frontend && npm run lint && npm run type-check` |
| Full suite command (前端) | `cd xingran-react-frontend && npm run test` |
| Backend regression | `go test ./internal/services/system/ -run "TestGetAPIKey|TestDeleteAPIKey|TestUpdateAPIKey" -v` |

### Phase Requirements → Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| CONTRACT-01 SC#1 | getAPIKey 用 POST /:id 不再 404，返回 code:0 | 集成 (curl) | `curl -X POST -H "Authorization: Bearer <jwt>" http://localhost:9000/system/apikeys/<id>` → 断言 code:0 | ❌ Wave 0（手动 curl 脚本） |
| CONTRACT-01 SC#2 | updateAPIKey 用 POST /:id/update 返回 code:0，DB updated_at 刷新 | 集成 (curl + DB 查询) | curl 更新 + `SELECT updated_at FROM sys_api_keys WHERE id=...` | ❌ Wave 0 |
| CONTRACT-01 SC#3 | deleteAPIKey 用 POST /:id/delete，软删后重复删除返回 code:1001 "密钥不存在" | 集成 (curl) | 两次 curl delete，第二次断言 code:1001 且 message 含"密钥不存在" | ❌ Wave 0 |
| CONTRACT-02 SC#1 | 编辑表单 camelCase 复合字段（ipWhitelist/inheritPerms/expiresAt）完整回填 | 手动 (UI) / 前端单测 | 打开编辑 Modal 检查字段值 / vitest 断言 record→form 映射 | ❌ Wave 0（可选单测） |
| CONTRACT-02 SC#2 | Create/Update 发 camelCase body，后端字段非零值持久化 | 集成 (curl + DB 查询) | curl create 带 ipWhitelist + `SELECT ip_whitelist FROM sys_api_keys` 非空 | ❌ Wave 0 |
| CONTRACT-02 排序 | 列排序 orderByColumn 为 camelCase，匹配后端白名单生效 | 手动 (UI) | 点列头排序，查网络请求 orderByColumn=isActive | ❌ Wave 0 |
| 回归 | 后端 not-found / 软删 / 更新 / 删除 service 行为不变 | 单元 (Go) | `go test ./internal/services/system/ -run "TestGetAPIKey|TestDeleteAPIKey|TestUpdateAPIKey"` | ✅ 现有 `apikey_service_test.go`（不改） |

### Sampling Rate
- **Per task commit:** `cd xingran-react-frontend && npm run lint && npm run type-check`（类型检查是契约对齐最关键的回归门——类型错误会立刻暴露漏改的 record.X）
- **Per wave merge:** `npm run test`（前端）+ `go test ./internal/services/system/`（后端回归）+ 手动 curl SC#1-SC#3
- **Phase gate:** 启动后端 + 前端，手动跑 SC#1-SC#4 全部成立（编辑回填 / 更新持久化 / 软删错误语义 / 决策记录）

### Wave 0 Gaps
- [ ] 手动 curl 验证脚本（SC#1-SC#3 的 HTTP 断言）—— 需要可运行的 backend + 一个已知 API Key ID + JWT token
- [ ] 可选：`xingran-react-frontend/src/api/apikey.test.ts` —— 断言三函数调用 POST 且路径含 /update、/delete（vitest mock `post`）；planner 按 TESTING.md 决定是否补
- [ ] 可选：`types/apikey.ts` 编译期断言（改后 `index.tsx` 的 record.X 访问若漏改，`npm run type-check` 会报错 —— 这是天然的契约门）

**验证策略要点：** `npm run type-check` 是本 phase 最高性价比的回归门——`APIKey` 接口字段改 camelCase 后，`index.tsx` 中任何漏改的 `record.ip_whitelist` 会立刻触发 TS 编译错误（`Property 'ip_whitelist' does not exist on type 'APIKey'`），强制全量改完才能通过。planner 应在每个改字段名的 task 后跑 type-check。

## Security Domain

> `security_enforcement` 未在 config.json 显式设为 false（缺席 = 启用）。本 phase 安全面极小。

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | no | 不改认证（沿用 Phase 57 就绪的 JWT） |
| V3 Session Management | no | 不改 token 机制 |
| V4 Access Control | no | 不改权限（路由仍 JWT 保护，handler 不变） |
| V5 Input Validation | 间接 | 字段命名对齐让后端 `ShouldBindJSON` + binding tag（`required` 等）真正生效——之前 snake 字段被丢意味着 required 校验也绕过 |
| V6 Cryptography | no | SM2/SM4 加密路径不变（见「SM2/SM4 加密交互确认」） |

### Known Threat Patterns
| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| 字段绑定失败导致权限字段（inheritPerms/IPWhitelist）取默认值 | Tampering | D-03 对齐字段名，让 inheritPerms/ipWhitelist 真正写入 DB（当前静默丢字段 = 安全控制被绕过） |

**安全收益备注：** CONTRACT-02 不只是功能 bug——`inheritPerms`（继承用户权限）与 `ipWhitelist`（IP 白名单）是 API Key 的安全控制字段。当前前端发了 snake_case 版本被后端丢弃 → 这两个字段在 DB 里保持默认值（inheritPerms=false, ipWhitelist=[]）。若用户以为设置了 IP 白名单限制但实际没存，是**安全控制的静默失效**。D-03 修复后这些安全字段才真正生效。

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Node.js 24+ | 前端 lint/type-check/build | ✓（CLAUDE.md 标注要求） | — | — |
| Go 1.24 | 后端 build 回归（`go build ./...`） | ✓（项目工具链） | — | — |
| PostgreSQL 18 | curl 集成验证 SC#1-SC#3 | 需启动 | — | 用既有 Go 单测（SQLite in-memory）作 fallback 回归 |
| Redis 7.4 | 后端启动（cache） | 需启动 | — | 跳过 curl 集成，只跑类型检查 + 单测 |

**Missing dependencies with no fallback:**
- 无。本 phase 是纯前端编辑，`npm run type-check` + `npm run lint` 不依赖任何外部服务即可跑。

**Missing dependencies with fallback:**
- PostgreSQL/Redis 若未启动 → 无法跑 curl 集成验证 SC#1-SC#3，但可用 `go test ./internal/services/system/`（SQLite in-memory）验证后端 not-found/软删逻辑回归（后端本 phase 零改动，回归风险极低）。

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| 前端 apikey.ts 用 GET/PUT/DELETE | 改 POST 对齐后端全 POST Router Pattern | Phase 58（本 phase） | 三个操作不再 404 |
| 前端 types/apikey.ts 用 snake_case | 改 camelCase 对齐后端 json tag | Phase 58（本 phase） | 复合字段不再静默丢弃 |

**本 phase 不引入任何新技术**——是把偏离的前端代码回归项目既有约定（CLAUDE.md + CONVENTIONS.md）。无 deprecated API、无版本迁移。

## Assumptions Log

> 本 phase 改动范围与决策均已由 discuss 锁定并由本次研究逐行核验。剩余假设极少。

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | `npm run type-check` 会在 `APIKey` 接口字段改名后对 `record.ip_whitelist` 等漏改报错 | Validation Architecture | 若 TS 配置过松（如 `noImplicitAny: false` + dataIndex 类型宽松），编译期不报错——但 `npm run type-check:strict`（CLAUDE.md 有此命令）会兜底。风险低 |
| A2 | 手动 curl 验证可启动后端 + 拿到 JWT + 已知 API Key ID | Validation Architecture | 若环境不具备，降级为 Go 单测 + 类型检查。不阻塞 phase 完成（后端零改动） |

**注：** Go json 行为、路由注册、字段 json tag、行号、调用方数、not-found 错误码、加密逻辑——这些核心技术事实均已 `[VERIFIED]`（codebase grep + 实证 Go 程序），不在假设表内。

## Open Questions (RESOLVED — 58-01-PLAN.md 已采纳推荐)

1. **Form.Item name 是否一并改 camelCase（类别 D）？** — RESOLVED: 采纳推荐（是），plan Task 2 类别 D 全改 camelCase。
   - What we know: Form.Item name 是前端内部约定，不直接接触 wire format（wire 字段在类别 C 的 data 构建处决定）。理论上可只改类别 A/B/C + 类别 D 的 record.X 读取，保留 Form.Item name 为 snake 并在构造时映射。
   - What's unclear: 哪种策略更不易出错。
   - Recommendation: **全改 camelCase（含 Form.Item name）**。理由：(a) 页面已在多处编辑，一次性统一最清晰；(b) 避免 `values.inherit_perms`（snake）↔ `inheritPerms: values.inherit_perms`（camel 构造）的认知负担；(c) Form.Item name 改 camel 后 `values.inheritPerms` 直接赋给 `data.inheritPerms`，零映射。属 Claude's Discretion，planner 可定。

2. **是否补前端 vitest 单测（api/apikey.test.ts）？**
   - What we know: TESTING.md 明示「无组件测试」「前端覆盖极有限」。vitest.config.ts 已配置可用。
   - What's unclear: 契约对齐是否值得破例补一个单测。
   - Recommendation: **可选，优先级低**。最高性价比的验证是 `npm run type-check`（天然契约门）+ curl SC#1-SC#3。若 planner 想加，最小单测应 mock `post` 断言三函数路径含 `/update`、`/delete` 且用 POST 方法。属 Claude's Discretion。

3. **`UpdateAPIKeyRequest` 前端是否补 `expiresAt` 字段对齐后端？**
   - What we know: 后端支持更新过期时间（apikey_request.go:26 + service.go:330-339），但前端 `UpdateAPIKeyRequest`（types/apikey.ts:42-49）无此字段，编辑表单也不暴露。
   - Recommendation: **本 phase 不补**。这是功能缺口而非契约断裂（前端不发该字段，后端不会出错）。补它会扩大 scope 到 UI（编辑表单加日期选择器），超出契约对齐范畴。留未来增强。

## Sources

### Primary (HIGH confidence)
- **Codebase 逐行核验**（`[VERIFIED: codebase]`）：
  - `internal/api/v1/system/apikey_router.go:18-25`（全 POST 路由注册）
  - `internal/api/v1/system/apikey_handler.go:127,159,195,323-343`（GetByID/Update/Delete/maskAPIKeys）
  - `internal/models/api_key.go:8-23`（APIKey struct json tag，camelCase）
  - `internal/models/base.go:11-19`（BaseModel json tag：createdAt/updatedAt）
  - `internal/models/system/requests/apikey_request.go:8-27`（Create/Update Request json tag，camelCase）
  - `internal/services/system/apikey_service.go:19-26,292-302,359-375`（排序白名单 + GetAPIKey/DeleteAPIKey not-found）
  - `xingran-react-frontend/src/api/apikey.ts:15,86-88,107-109,123-125,194-196`（三函数 + getUsageSummary + import）
  - `xingran-react-frontend/src/types/apikey.ts:12-62`（三接口字段名）
  - `xingran-react-frontend/src/pages/system/apikeys/index.tsx`（全文字段访问测绘）
  - `xingran-react-frontend/src/lib/api.ts:186-274`（shouldEncryptRequest + 请求拦截器）
  - `pkg/errors/codes.go:13,203-204`（CodeParamError=1001 → HTTP 400）
- **实证 Go 程序**（`[VERIFIED: empirical Go test]`）：encoding/json 大小写折叠行为，Go 1.24 工具链实跑
- **既有后端测试**（`[VERIFIED]`）：`internal/services/system/apikey_service_test.go:644-649,757-770`（not-found + 软删后不可查询，覆盖 SC#3 错误语义）

### Secondary (MEDIUM confidence)
- `.planning/codebase/CONVENTIONS.md` §Route Pattern（全 POST + /:id/update、/:id/delete）`[CITED]`
- `.planning/codebase/TESTING.md`（前端测试现状：Vitest 配置但覆盖极有限）
- `CLAUDE.md`（Router Pattern、API Response Format、Status Convention、前端 API calling 约定）
- `58-CONTEXT.md` `<decisions>` D-01..D-06（locked，本研究验证其技术前提）

### Tertiary (LOW confidence)
- 无。所有核心事实均经 codebase 或实证程序验证，无 WebSearch-only 假设。

## Metadata

**Confidence breakdown:**
- Standard stack（现有栈复用）: HIGH — 不引入新依赖，现有版本来自 package.json
- 路由契约对齐（CONTRACT-01）: HIGH — 后端 router 行号 + 前端函数体均已逐行核验，零漂移
- 字段命名对齐（CONTRACT-02）: HIGH — Go json 行为实证确证 + 三表逐字段对照 json tag
- 改动面测绘（index.tsx）: HIGH — 全文读取，按类别列出约 25 处改动点
- 验证策略: HIGH — type-check 作天然契约门 + curl SC#1-SC#3 + 既有 Go 单测回归

**Research date:** 2026-08-13
**Valid until:** 2026-09-12（30 天；本 phase 是对当前源码的契约对齐，若 Phase 58 执行前 apikey 相关源码被其他 phase 改动需重新核验行号）

## RESEARCH COMPLETE

**Phase:** 58 - 前后端路由契约对齐
**Confidence:** HIGH

### Key Findings
- **CONTEXT.md 行号零漂移**：所有引用（apikey.ts 三函数、index.tsx 244/246/277/307-323/325/551-566、api.ts 拦截器、handler/service 行号）经逐行核对仍准确。
- **D-02 核心论断实证成立**：Go `encoding/json` 大小写不敏感匹配只折叠字母不折叠下划线 —— `inherit_perms` 确实绑不到 `inheritPerms` 取零值（实证复现），证明 CONTRACT-02 是真实的静默数据损坏，修复必须改字段名。
- **发现 CONTRACT-02 第三后果（排序失效）**：后端排序白名单键是 camelCase（`isActive`/`expiresAt`/`lastUsedAt`），前端发 snake_case 的 `orderByColumn` 匹配不上 → 这三列排序当前静默失效。D-03 字段重命名顺带修复，无需额外代码。
- **改动面完全测绘**：`apikey.ts` 3 函数 + 移除 put/del import（get 保留）；`types/apikey.ts` 3 接口 13 字段；`index.tsx` 约 25 处字段访问分 4 类（数据契约/排序白名单/data 构建/Form.Item）。LogsModal 隔离不改（Phase 59 范围）。
- **`npm run type-check` 是天然契约门**：APIKey 接口改 camel 后，index.tsx 漏改的 `record.ip_whitelist` 会触发 TS 编译错误，强制全量改完才通过 —— 最高性价比回归门。

### File Created
`D:\code\ClaudeCode\guoguo\.planning\phases\58-route-contract-alignment\58-RESEARCH.md`

### Confidence Assessment
| Area | Level | Reason |
|------|-------|--------|
| Standard Stack | HIGH | 不引入新依赖，纯编辑现有文件 |
| 路由契约对齐 (CONTRACT-01) | HIGH | router/handler 前端函数体逐行核验，零漂移 |
| 字段命名对齐 (CONTRACT-02) | HIGH | Go json 实证 + 逐字段对照 json tag 三表 |
| 改动面测绘 | HIGH | index.tsx 全文读取，25 处分类列出 |
| 验证策略 | HIGH | type-check 契约门 + curl SC 断言 + 既有 Go 单测 |

### Open Questions (RESOLVED — 见 58-01-PLAN.md)
- Q1: Form.Item name 是否一并改 camelCase — RESOLVED: 采纳推荐（是），plan 类别 D 全改 camel
- Q2: 是否补 vitest 单测 — RESOLVED: 采纳推荐（否），type-check 为天然契约门
- Q3: UpdateAPIKeyRequest 是否补 expiresAt — RESOLVED: 采纳推荐（否），超 scope

### Ready for Planning
研究完成，决策 D-01..D-06 技术前提全部验证成立，改动面逐字段/逐行测绘完毕。Planner 可直接据「前端字段映射」三表 + 「index.tsx 字段访问改动清单」四类别创建 PLAN.md，无需再做源码探索。