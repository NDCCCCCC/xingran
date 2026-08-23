# Phase 58: 前后端路由契约对齐 - Context

**Gathered:** 2026-08-13
**Status:** Ready for planning

<domain>
## Phase Boundary

让前端 API Key 管理页对单条记录的**查询 / 更新 / 删除**三个操作不再 404，前后端路由方法与路径完全对齐（修复 P1-1 / CONTRACT-01）；并修复本 session 审计发现的更深的**字段命名契约断裂**（新增 CONTRACT-02），使 Create/Update 不再静默丢弃复合字段、List/详情/编辑表单的复合字段（IP 白名单、继承权限、启用状态、过期时间）真正回填与展示。操作真实生效，SC#1–SC#3 完全成立。

**本 phase 不在生产路由挂载 MultiAuth**（挂载启用是 Phase 60 AUTH-03 决策点）。契约对齐在现有 JWT 认证下即可验证——3 个 404 与字段命名断裂均与 MultiAuth 挂载无关。

**Requirements:** CONTRACT-01（P1-1，原 ROADMAP）, CONTRACT-02（字段命名断裂，本 session 新增并吸收）

</domain>

<decisions>
## Implementation Decisions

### 对齐方向（CONTRACT-01 / P1-1）
- **D-01:** **Option A — 前端 `apikey.ts` 三个操作改 POST 对齐后端既有路由**。getAPIKey → `POST /:id`、updateAPIKey → `POST /:id/update`、deleteAPIKey → `POST /:id/delete`。后端零改动，契约定型为 `apikey_router.go` 当前注册路径（满足 SC#4「与当前注册路径一致」）。改动局限在 3 个函数体（方法 + 路径）。
  - **理由:** 后端 `apikey_router.go` 已全 POST 并遵循 CLAUDE.md 标准 Router Pattern（User/Role/Dept/Post 一致），前端 `apikey.ts` 是全项目唯一用 GET/PUT/DELETE 的偏离模块。`/system/apikeys/*` 是 JWT 保护内部管理端点（管理密钥的 admin CRUD），非 API-Key 认证的外部 API 面 → RESTful 不构成选 B 理由。影响面极小：`getAPIKey` 零调用方（死代码），`updateAPIKey`/`deleteAPIKey` 各 1 个调用方（`index.tsx`）。

### 字段命名契约断裂（CONTRACT-02，本 phase 新增发现并吸收）
- **D-02:** Phase 58 **端到端吸收**字段命名断裂，记为新缺陷 **CONTRACT-02**（原 ROADMAP 缺陷表未列，本 session 审计发现；需同步 REQUIREMENTS.md / ROADMAP.md——见下方 docs-sync）。比 P1-1 更深、更严重：
  - 后端 camelCase（`ipWhitelist` / `inheritPerms` / `isActive` / `expiresAt` / `lastUsedAt`）vs 前端 snake_case（`ip_whitelist` / `inherit_perms` / `is_active` / `expires_at`）。
  - `api.ts` 请求/响应拦截器**均无 snake↔camel 转换**（请求拦截器仅做 SM2+SM4 加密 + token；响应拦截器仅做解密）。
  - **后果 ①（Create/Update 绑定静默失败）:** 前端发 `inherit_perms` / `ip_whitelist` / `expires_at`，后端 `CreateAPIKeyRequest` / `UpdateAPIKeyRequest` 绑 `inheritPerms` / `ipWhitelist` / `expiresAt` → 不匹配（Go json 大小写不敏感也救不了——**下划线位置不同**）→ 取零值，IP 白名单 / 继承权限 / 过期时间被**静默丢弃**。这是比 404 更严重的静默数据损坏。
  - **后果 ②（List/详情显示）:** 后端回 `ipWhitelist`（camel），前端读 `record.ip_whitelist`（snake）→ `undefined`。SC#1「表单字段完整回填」、SC#2「列表展示更新值」对复合字段无法成立。
- **D-03:** 字段命名对齐方向 = **前端 → camelCase**（审计修正了 discuss 初判「后端 → snake」）。审计确认**后端 camelCase 是全项目约定**（非 apikey 孤例）：`User`（`employeeNo` / `deptId` / `loginIp`）、`Dept`（`deptName` / `parentId` / `orderNum`）、`APIKeyUsageLog`（`apiKeyId` / `statusCode` / `clientIp` / `createdAt`）均 camelCase。apikey FE 的 snake 类型才是孤例。改 `types/apikey.ts`（`APIKey` / `CreateAPIKeyRequest` / `UpdateAPIKeyRequest` 接口字段名）+ `index.tsx` 页面字段访问 → camelCase。**后端零改动**（`maskAPIKeys` List map 与 GetByID 原始 struct 均已 camelCase，本就正确）。
- **D-04:** 字段命名修复**范围限定** API Key 管理 CRUD 类型（`APIKey` / `CreateAPIKeyRequest` / `UpdateAPIKeyRequest` + `index.tsx` 相关字段访问）。`APIKeyUsageLog` / `UsageSummary` 类型的 snake→camel 留 **Phase 59**（OBSERV，本就是修使用日志的 phase）。

### 编辑详情数据来源（SC#1）
- **D-05:** 编辑表单继续**复用列表行 `editingRecord`** 回填（字段名修对 camel 后完整回填）。`getAPIKey` 契约修对（POST /:id，D-01）保持可用但**不接入编辑流**（消除「编辑必须拉取单条」的假设）。零新增请求，回归修复最小改动。`getAPIKey` 当前本就是死代码（无调用方），保持契约正确即可，是否将来接入由后续需求决定。

### 删除交互与错误语义（SC#3）
- **D-06:** **无新代码决策**——SC#3 由 D-01 对齐 + 既有后端逻辑覆盖。后端 `GetAPIKey`（`apikey_service.go:296`）与 `DeleteAPIKey`（`:363`）对 not-found / 软删记录已返回明确业务错误 `apperrors.CodeParamError "密钥不存在"`；前端 `index.tsx:551` 已有 `<Popconfirm>` 删除确认。HTTP 方法一对齐，「404 路由缺失」消除、「明确错误」已具备。**planner 仅需验证**（测试 / curl）重复删除与再访问返回 `code != 0` + 「密钥不存在」消息，而非 404。

### Claude's Discretion
- 未用 import 清理：Option A 后 `put` / `del` 从 `apikey.ts` 移除（update/delete 改用 `post`）；`get` 保留（仍被 `getUsageSummary` 使用）。
- `types/apikey.ts` 字段名 → camel 的逐字段映射（planner 对照 `models/api_key.go` 与 `apikey_request.go` 的 json tag 逐一改）。
- 是否补充契约对齐的前端单测（vitest）由 planner 按 `.planning/codebase/TESTING.md` 决定。

### Folded Todos
（无——`cross_reference_todos` 仅命中 `operlog-exclude-paths.md`，score 0.2 关键词误匹配，与 Phase 58 无真实关联，未 fold。见 `<deferred>` Reviewed Todos。）

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### 需求与规划
- `.planning/ROADMAP.md` §Phase 58 — Goal / Depends on / Requirements（CONTRACT-01 + 本 session 新增 CONTRACT-02）/ Success Criteria（SC#1–SC#4）
- `.planning/REQUIREMENTS.md` — CONTRACT-01 定义 + CONTRACT-02（待 docs-sync 注册）
- `.planning/STATE.md` §根因调查结论 — P1-1 ground-truth（`apikey_router.go` vs `src/api/apikey.ts` 方法不匹配 → 404）

### 待对齐源码 —— 前端（本 phase 主战场，纯前端改动）
- `xingran-react-frontend/src/api/apikey.ts` — D-01 三个函数改 POST（getAPIKey/updateAPIKey/deleteAPIKey）；`get`/`put`/`del` import 调整
- `xingran-react-frontend/src/types/apikey.ts` — D-03 字段名 → camelCase（`APIKey` / `CreateAPIKeyRequest` / `UpdateAPIKeyRequest`）
- `xingran-react-frontend/src/pages/system/apikeys/index.tsx` — D-03 页面字段访问 → camel（`record.ip_whitelist` 等，行 244/246/277；create/update data 构建行 307-323）；D-05 编辑复用 `editingRecord`（行 325）；D-06 删除 Popconfirm（行 551-566）
- `xingran-react-frontend/src/lib/api.ts` — 请求拦截器（行 207-274，仅加密无命名转换）/ 响应拦截器（行 277+，仅解密无命名转换）——「无 snake↔camel 转换」的 D-02 证据

### 后端契约基准（不改，前端对齐目标）
- `internal/api/v1/system/apikey_router.go` — 全 POST 路由注册（D-01 对齐目标；SC#4 基准）
- `internal/api/v1/system/apikey_handler.go` — `GetByID`（行 127，返回完整 struct 仅 mask Key）/ `Update`（行 159）/ `Delete`（行 195）/ `maskAPIKeys` List map（行 323-343，camelCase 基准）
- `internal/models/api_key.go` — `APIKey` struct json tag（camelCase：`userId`/`isActive`/`ipWhitelist`/`inheritPerms`/`expiresAt`/`lastUsedAt`）—— D-03 对齐目标
- `internal/models/system/requests/apikey_request.go` — `CreateAPIKeyRequest`/`UpdateAPIKeyRequest` json tag（camelCase：`inheritPerms`/`ipWhitelist`/`expiresAt`/`isActive`）—— D-02 后果①证据
- `internal/services/system/apikey_service.go` — `GetAPIKey`（行 296）/ `DeleteAPIKey`（行 363）not-found 返回 `apperrors.CodeParamError "密钥不存在"`—— D-06 / SC#3 证据

### 约定参考
- `.planning/codebase/CONVENTIONS.md` — 后端 camelCase json 全项目约定（D-03 审计依据）
- `CLAUDE.md` 标准 Router Pattern — 全 POST + `/:id/update` `/:id/delete`（D-01 依据）

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `apikey.ts` 的 `post` / `get` 封装（`@/lib/api`）：Option A 后 update/delete 复用 `post`，getAPIKey 改 `post`，`getUsageSummary` 仍用 `get`。
- Ant Design `Popconfirm`（`index.tsx:551`）：删除确认已存在，无需新增。
- `apperrors`（`pkg/errors`）：后端 not-found 业务错误已就绪（D-06）。

### Established Patterns
- **后端 camelCase json 是全项目约定**（User/Dept/APIKeyUsageLog/APIKey 一致）；DB 列名为 snake_case（gorm tag），与 json tag 独立——勿混淆。
- **CLAUDE.md 标准 Router Pattern**：全 POST + `/:id/update` `/:id/delete` 后缀（User/Role/Dept/Post 均如此）；apikey_router.go 已遵循，前端 apikey.ts 是唯一偏离。
- 前端不用裸 axios，统一用 `@/lib/api` 的 `post`/`get`；POST/PUT/PATCH 走 SM2+SM4 请求加密（GET 不加密）——getAPIKey 改 POST 后会走加密（无影响，因其未接入编辑流）。

### Integration Points
- 前端 `apikey.ts` ↔ `apikey_router.go`（POST 路由契约）。
- `types/apikey.ts` ↔ `models/api_key.go` + `apikey_request.go` json tag（camelCase 字段契约）。
- 后端 List（`maskAPIKeys` camelCase map）+ GetByID（原始 struct camelCase）→ 前端类型 camelCase 对齐后两端一致。

</code_context>

<specifics>
## Specific Ideas

- 用户选「端到端吸收字段命名（CONTRACT-02）」+「前端→camelCase」，反映其一贯取向：**API Key 系统要「真正可用」而非仅消除报错**（承自 Phase 57 specifics——用户希望系统「真正可用」）。planner 应理解：Phase 58 不只是让 3 个操作不 404，还要让 Create/Update/List 的复合字段（IP 白名单、继承权限、过期时间）**真正生效**——这些字段此前被静默丢弃/显示为 undefined，是比 404 更隐蔽的功能缺失。
- discuss 中审计**修正了初判**：最初倾向「后端 camel→snake」（误以为贴合 DB snake 约定），经全项目模型 json tag 审计后确认后端 camelCase 才是 API 约定、FE snake 是孤例，方向改为「前端→camel」。这一修正在 CONTEXT 留痕，避免 planner 重蹈初判。

</specifics>

<deferred>
## Deferred Ideas

- **`APIKeyUsageLog` / `UsageSummary` 类型字段命名 snake→camel** → **Phase 59**（OBSERV，本就是修使用日志的 phase；与 Phase 58 的 API Key 管理 CRUD 解耦）。
- **`getAPIKey` 接入编辑流（拉取最新详情回填）** → 本 phase 不做（D-05 复用列表行）；可选未来增强，需评估是否值得多一次请求换取字段新鲜度。
- **密钥轮换 / 吊销、配额告警** → FUTURE-APIKEY-03/04（仍 v2 Future）。

### Reviewed Todos (not folded)
- `operlog-exclude-paths.md`（`todo.match-phase` 得分 0.2，关键词「phase」误匹配，area=general）——与 Phase 58（前后端路由契约对齐）无真实关联，不 fold。

### Docs-sync（本 session 决策产生的待办）
- **CONTRACT-02 需注册进 REQUIREMENTS.md + ROADMAP.md**：CONTRACT 区新增 CONTRACT-02；Phase 58 Requirements 行加 CONTRACT-02；Coverage Map / 计数（13→14）同步；ROADMAP Phase 58 详情 Requirements 改为「CONTRACT-01, CONTRACT-02」。本 session discuss 已据此执行该同步（见 git commit）。

</deferred>

---

*Phase: 58-前后端路由契约对齐*
*Context gathered: 2026-08-13*
