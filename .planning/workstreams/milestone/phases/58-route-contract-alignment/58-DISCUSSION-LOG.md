# Phase 58: 前后端路由契约对齐 - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-08-13
**Phase:** 58-前后端路由契约对齐
**Areas discussed:** 对齐方向（CONTRACT-01）, 编辑详情数据来源（含字段命名 CONTRACT-02）, 删除交互与错误语义（SC#3）

---

## 对齐方向（CONTRACT-01 / P1-1）

| Option | Description | Selected |
|--------|-------------|----------|
| A: 前端改 POST | apikey.ts 的 getAPIKey/updateAPIKey/deleteAPIKey 改 POST 对应后端既有路径（POST /:id、/:id/update、/:id/delete）。后端零改动，与 User/Role/Dept 全项目惯例一致。 | ✓ |
| B: 后端补 RESTful | apikey_router.go 新增 GET/PUT/DELETE /:id，让 apikey 成为唯一 RESTful 模块，偏离全项目 POST 惯例。 | |

**User's choice:** A: 前端改 POST
**Notes:** 后端 apikey_router.go 已全 POST 并遵循 CLAUDE.md 标准 Router Pattern；前端 apikey.ts 是唯一偏离模块。`/system/apikeys/*` 是 JWT 保护内部管理端点非外部 API 面，RESTful 不构成选 B 理由。改动局限在 3 个函数体，影响面极小（getAPIKey 零调用方，update/delete 各 1 调用方）。

---

## 编辑详情数据来源（含字段命名 CONTRACT-02）

### Q1 — 字段命名断裂范围

| Option | Description | Selected |
|--------|-------------|----------|
| 收紧:仅 HTTP 方法 | Phase 58 只修 P1-1 三个 404；字段命名断裂记为新缺陷 CONTRACT-02 独立处理。SC#1/SC#2 复合字段暂不成立。 | |
| 端到端:含字段命名 | Phase 58 同时修 HTTP 方法 + 对齐字段命名（记 CONTRACT-02），SC#1/SC#2 完全成立。消掉一个比 404 更严重的静默数据损坏 bug。 | ✓ |

**User's choice:** 端到端:含字段命名（CONTRACT-02）
**Notes:** 审计发现比 P1-1 更深的字段命名契约断裂——后端 camelCase vs 前端 snake_case，且无转换层。Create/Update 绑定静默丢弃 inherit_perms/ip_whitelist/expires_at（取零值）；List/详情复合字段 undefined。SC#1/SC#2 依赖字段对齐才能完全 PASS。

### Q2 — 字段命名对齐方向

| Option | Description | Selected |
|--------|-------------|----------|
| 前端→camelCase | 改 types/apikey.ts + index.tsx 字段名→camel。后端零改动。与全项目 camelCase 约定 + Option A 一致。（审计修正了初判） | ✓ |
| 后端→snake_case | 改 api_key.go 模型 + 请求结构体 json tag→snake。会让 apikey 后端偏离全项目 camelCase 约定。 | |

**User's choice:** 前端→camelCase
**Notes:** 审计确认后端 camelCase 是全项目约定（User employeeNo/deptId、Dept deptName/parentId、APIKeyUsageLog apiKeyId/statusCode/createdAt 均 camelCase）；apikey FE 的 snake 类型是孤例。discuss 初判曾倾向「后端→snake」（误以为贴合 DB snake 约定），审计后修正。→ Phase 58 几乎纯前端。

### Q3 — 编辑表单数据来源

| Option | Description | Selected |
|--------|-------------|----------|
| 复用列表行(最小) | 编辑表单继续用 editingRecord 回填（字段名修对 camel 后完整）。getAPIKey 契约修对但不接入编辑流。零新增请求。 | ✓ |
| 拉取单条详情 | 编辑时先 getAPIKey(id) 拉最新详情再回填，消除死代码、保证字段最新。多一次请求。 | |

**User's choice:** 复用列表行(最小)
**Notes:** 字段名修对 camel 后 editingRecord 完整；getAPIKey 本就是死代码（无调用方），保持契约正确即可，回归修复最小改动。

---

## 删除交互与错误语义（SC#3）

| Option | Description | Selected |
|--------|-------------|----------|
| (无决策——纯验证) | SC#3 由对齐 + 既有后端逻辑覆盖，无 AskUserQuestion | ✓（核实结论） |

**核实结论:** 后端 GetAPIKey（apikey_service.go:296）与 DeleteAPIKey（:363）对 not-found/软删记录已返回明确业务错误 `apperrors.CodeParamError "密钥不存在"`；前端 index.tsx:551 已有 `<Popconfirm>` 删除确认。HTTP 方法一对齐（Area 1），「404 路由缺失」消除、「明确错误」已具备。Area 3 无新代码决策，planner 仅需验证重复删除/再访问返回 `code != 0` + 「密钥不存在」而非 404。

---

## Claude's Discretion

- 未用 import 清理（Option A 后 `put`/`del` 从 apikey.ts 移除；`get` 保留供 getUsageSummary）。
- types/apikey.ts 字段名→camel 的逐字段映射（planner 对照后端 json tag）。
- 是否补前端单测（vitest）由 planner 按 TESTING.md 决定。

## Deferred Ideas

- `APIKeyUsageLog`/`UsageSummary` 字段命名 snake→camel → Phase 59（OBSERV）。
- `getAPIKey` 接入编辑流（拉取最新详情）→ 本 phase 不做，可选未来增强。
- 密钥轮换/吊销、配额告警 → FUTURE-APIKEY-03/04（v2 Future）。
- CONTRACT-02 docs-sync（注册进 REQUIREMENTS.md + ROADMAP.md）→ 本 session 已执行（见 git commit）。
