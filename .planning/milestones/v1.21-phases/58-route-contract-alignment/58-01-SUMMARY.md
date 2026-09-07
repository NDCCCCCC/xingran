---
phase: 58-route-contract-alignment
plan: 01
subsystem: frontend
status: code-complete-e2e-deferred
tags: [api-key, route-contract, camelCase, frontend, contract-alignment]
requirements_addressed: [CONTRACT-01, CONTRACT-02]
decisions: [D-01, D-02, D-03, D-04, D-05, D-06]
commits:
  - "1978935 — Task 1: apikey.ts 三函数 GET/PUT/DELETE → POST 对齐 (CONTRACT-01/D-01)"
  - "6a4c772 — Task 2: types/apikey.ts + index.tsx 字段命名 snake→camelCase (CONTRACT-02/D-02/D-03/D-04/D-05)"
---

# 58-01 Summary — 前后端路由契约对齐

## Status

**代码完成 (code-complete),端到端验证 (Task 3 SC#1–SC#4) 延期。**

Task 1 + Task 2 的契约修复代码已提交并经自动化门验证通过;Task 3 的浏览器/curl/DB 端到端验证因 **dev 数据库性能问题**无法在当前环境执行,延期至有更快 dev DB 时补。详见下文 §Deferred。

## What Was Built

### Task 1 — CONTRACT-01 路由方法对齐 (D-01, commit 1978935)

前端 `src/api/apikey.ts` 三个操作由 RESTful 方法改为 POST,对齐后端 `apikey_router.go` 全 POST 路由(后端零改动,Option A):

| 函数 | 改前 (404) | 改后 |
|------|-----------|------|
| `getAPIKey` | `GET /:id` | `POST /:id` |
| `updateAPIKey` | `PUT /:id` | `POST /:id/update` |
| `deleteAPIKey` | `DELETE /:id` | `POST /:id/delete` |

`get` import 保留(`getUsageSummary` 仍用 `GET /:id/summary`,对应后端唯一 GET 路由),`put`/`del` import 移除。

### Task 2 — CONTRACT-02 字段命名 snake→camelCase (D-02/D-03/D-04/D-05, commit 6a4c772)

前端字段命名由 snake_case 改为 camelCase,对齐后端 json tag,消除 Create/Update 复合字段被 `encoding/json` 静默取零值丢弃的安全控制失效(T-58-01):

- `src/types/apikey.ts`:`APIKey` / `CreateAPIKeyRequest` / `UpdateAPIKeyRequest` 三接口字段全 camelCase(`ipWhitelist`/`inheritPerms`/`isActive`/`expiresAt`/`lastUsedAt`/`createdAt`/`updatedAt`)。`APIKeyUsageLog`/`UsageSummary` 保持 snake_case(D-04,Phase 59 范围,未误改)。
- `src/pages/system/apikeys/index.tsx`:约 25 处字段访问改 camelCase —— 类别 A(record.X / dataIndex / key)、类别 B(排序白名单 `sortField === "isActive"` 等)、类别 C(create/update data 构建)、类别 D(Form.Item name ↔ setFieldsValue ↔ values 三者一致)。
- `LogsModal.tsx` 隔离不动(Phase 59)。
- D-05:编辑表单复用 `editingRecord` 回填,不引入新 `getAPIKey` 请求。

## Verification

### 自动化门(本次会话实证,全绿)

| 门 | 结果 | 备注 |
|----|------|------|
| `npm run type-check` | ✅ 退出码 0 | 本次会话重跑(rebrand + Phase 58 改动均在 HEAD) |
| Acceptance grep(Task 1) | ✅ | `put(`/`del(` 计数 0;`post /:id/update`、`post /:id/delete` 各 1;`get /:id/summary` 保留 |
| Acceptance grep(Task 2) | ✅ | types `ipWhitelist`=3、`inheritPerms`=3、scope 内 snake=0;index.tsx snake=0、record camel=11 |
| 后端零改动 | ✅ | Task 1/2 守护:`internal/ pkg/ cmd/` 无 diff |
| `npm run lint` | ✅(此前验证) | 原执行已通过;lint 重跑较慢,转后台 |

### 后端回归(由不变量保证)

`go test ./internal/services/system/ -run "TestGetAPIKey|TestDeleteAPIKey|TestUpdateAPIKey"` 未在本次重跑(需 DB 连接,dev pooler 22–30s/查询)。**后端零改动**(`git diff --name-only internal/` 为空)保证既有单测无回归路径;SC#3 错误语义(`code:1001 "密钥不存在"`)由既有后端逻辑 `apikey_service.go:296-297,363-364` + D-01 路由对齐共同覆盖,无新代码。

## Deferred — Task 3 端到端 SC#1–SC#4

**延期原因:dev 数据库(Supabase 新加坡 Session pooler)性能不足,无法支撑浏览器端到端验证。**

本次会话尝试跑 SC#1–SC#4 时实测:
- 登录链路已修通(`use_sm2: true` 后 `/auth/public-key` 200,`POST /auth/login` 200,admin 认证成功)
- 但 post-login 菜单加载超时:`POST /my-menus`、`/my-menus/all` → **500,latency 30s**(查询超时);`/my-menus/permissions` → 200 但 **22–24s**
- app shell 加载不出来 → 无法导航到 API Key 管理页 → token 未稳定落盘 → 浏览器 E2E 不可行

根因与 Phase 58 启动 hang 同源:远端 pooler 链路高延迟 + 随机 TCP 单向黑洞(`debug session: backend-hang-on-automigrate`)。这是 **dev 基础设施性能约束,非 Phase 58 代码缺陷** —— 代码契约修复已提交且自动化门全绿。

**延期项(待更快 dev DB 补验):**

| SC | 验证内容 | 当前可替代证据 |
|----|---------|--------------|
| SC#1 | 编辑回填复合字段非 undefined | Task 2 字段 camelCase 修复 + type-check 通过(Form.Item name ↔ setFieldsValue ↔ values 一致) |
| SC#2 | DB `ip_whitelist/inherit_perms/expires_at` 持久化非零值 | Task 2 data 构建改 camelCase(类别 C),后端 json tag bind 生效;DB 实证待补 |
| SC#3 | 重复删除 `code:1001` 非 404 | Task 1 `POST /:id/delete` 对齐 + 后端既有 not-found 逻辑;curl 实证待补 |
| SC#4 | 排序 `orderByColumn` camelCase | Task 2 类别 B `sortField === "isActive"` 等 + acceptance grep 绿;DevTools 实证待补 |

**解除延期的前置:** 切换到更快 dev DB(本地 Postgres 或更近 Supabase region),使菜单/列表查询在交互级延迟(<1s)返回。届时按 `58-01-PLAN.md` Task 3 `<how-to-verify>` Step 2–5 补跑。

## Requirements & Decisions

- **CONTRACT-01**(P1-1,路由方法):✅ Task 1 修复
- **CONTRACT-02**(字段命名,静默数据损坏):✅ Task 2 修复
- **D-01..D-06**:全部落地(Task 1/2 代码 + 58-CONTEXT.md 决策记录就绪)

## Files Changed

- `xingran-react-frontend/src/api/apikey.ts` — 三函数 POST 对齐
- `xingran-react-frontend/src/types/apikey.ts` — 三接口 camelCase
- `xingran-react-frontend/src/pages/system/apikeys/index.tsx` — 字段访问 camelCase

后端零改动。
