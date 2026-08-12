---
phase: 16-api-key-mgt
plan: 05a
type: execute
wave: 5
depends_on: [16-03]
files_modified:
  - xingran-react-frontend/src/types/apikey.ts
  - xingran-react-frontend/src/api/apikey.ts
autonomous: true
requirements: ["INDEPENDENT"]
must_haves:
  truths:
    - TypeScript 类型定义完整
    - API 客户端函数封装正确
    - 使用项目统一的 API 调用方式（@/lib/api.ts）
    - 所有 API 函数类型安全
    - 错误处理符合项目规范
  artifacts:
    - path: xingran-react-frontend/src/types/apikey.ts
      provides: TypeScript 类型定义
      min_lines: 80
      contains:
        - interface APIKey
        - interface CreateAPIKeyRequest
        - interface UpdateAPIKeyRequest
        - interface APIKeyListParams
        - interface APIKeyUsageLog
        - interface UsageSummary
    - path: xingran-react-frontend/src/api/apikey.ts
      provides: API 客户端
      min_lines: 80
      contains:
        - export function listAPIKeys
        - export function createAPIKey
        - export function updateAPIKey
        - export function deleteAPIKey
        - export function toggleAPIKeyStatus
        - export function listUsageLogs
        - export function getUsageSummary
  key_links:
    - from: xingran-react-frontend/src/api/apikey.ts
      to: xingran-react-frontend/src/lib/api.ts
      via: 依赖
      pattern: import.*post.*from.*@/lib/api
    - from: xingran-react-frontend/src/api/apikey.ts
      to: internal/api/v1/system/apikey_router.go
      via: HTTP REST API calls
      pattern: post.*system/apikeys
---

<objective>
创建 API 密钥管理的前端类型定义和 API 客户端

目的：实现 API 密钥管理的前端类型定义和 API 调用封装
输出：TypeScript 类型定义，API 客户端函数

**说明：** 这是独立功能模块，不依赖 REQUIREMENTS.md 中的具体需求 ID。本计划专注于类型定义和 API 客户端，页面组件在 16-05b 中实现。
</objective>

<execution_context>
@$HOME/.claude/get-shit-done/workflows/execute-plan.md
@$HOME/.claude/get-shit-done/templates/summary.md
</execution_context>

<context>
@.planning/phases/16-api-key-mgt/16-CONTEXT.md
@.planning/phases/16-api-key-mgt/16-PATTERNS.md
@xingran-react-frontend/src/lib/api.ts
@xingran-react-frontend/src/types/system.ts
</context>

<tasks>

<task type="auto" tdd="false">
  <name>Task 1: 定义 TypeScript 类型</name>
  <files>xingran-react-frontend/src/types/apikey.ts</files>
  <read_first>
    - xingran-react-frontend/src/types/system.ts
  </read_first>
  <action>
创建 xingran-react-frontend/src/types/apikey.ts 文件，定义类型：

1. 定义 APIKey 接口：
   - id: string
   - name: string
   - key: string（脱敏显示）
   - scopes: string[]（read, write, admin）
   - ip_whitelist: string[]
   - inherit_perms: boolean
   - expires_at?: string
   - last_used_at?: string
   - is_active: boolean
   - description?: string
   - created_at: string
   - updated_at: string

2. 定义 CreateAPIKeyRequest 接口：
   - name: string
   - description?: string
   - scopes: string[]
   - inherit_perms: boolean
   - ip_whitelist?: string[]
   - expires_at?: string

3. 定义 UpdateAPIKeyRequest 接口：
   - name?: string
   - description?: string
   - scopes?: string[]
   - inherit_perms?: boolean
   - ip_whitelist?: string[]
   - is_active?: boolean

4. 定义 APIKeyListParams 接口：
   - 继承 PageParams（current, pageSize）
   - keyword?: string
   - status?: boolean
   - scope?: string

5. 定义 APIKeyUsageLog 接口：
   - id: string
   - api_key_id: string
   - user_id: string
   - method: string
   - path: string
   - status_code: number
   - client_ip: string
   - user_agent?: string
   - duration: number
   - success: boolean
   - created_at: string

6. 定义 UsageSummary 接口：
   - total_requests: number
   - success_rate: number
   - avg_duration: number
   - requests_by_method: Record<string, number>
   - requests_by_path: Record<string, number>
   - errors_by_status: Record<number, number>

7. 导出所有接口（export）

使用 type 或 interface（根据项目约定）
所有时间字段使用 ISO 8601 字符串格式
  </action>
  <verify>
    <automated>grep -E "interface APIKey|interface CreateAPIKeyRequest|interface UpdateAPIKeyRequest|interface APIKeyListParams|interface APIKeyUsageLog|interface UsageSummary" xingran-react-frontend/src/types/apikey.ts</automated>
  </verify>
  <done>
    - 所有类型定义完整
    - 字段类型正确
    - 可选字段标记正确（?）
    - 导出语句正确
  </done>
</task>

<task type="auto" tdd="false">
  <name>Task 2: 实现 API 客户端函数</name>
  <files>xingran-react-frontend/src/api/apikey.ts</files>
  <read_first>
    - xingran-react-frontend/src/lib/api.ts
    - xingran-react-frontend/src/types/apikey.ts
  </read_first>
  <action>
创建 xingran-react-frontend/src/api/apikey.ts 文件，实现 API 客户端：

1. 导入必要的模块：
   - import { post, get, put, del } from '@/lib/api'
   - import type { ... } from '@/types/apikey'

2. 实现 listAPIKeys 函数：
   - 参数：params?: APIKeyListParams
   - 返回：Promise<BaseResponse<PageData<APIKey>>>
   - 调用：post('/system/apikeys/list', params)

3. 实现 createAPIKey 函数：
   - 参数：data: CreateAPIKeyRequest
   - 返回：Promise<BaseResponse<{ key: string }>>
   - 调用：post('/system/apikeys', data)

4. 实现 getAPIKey 函数：
   - 参数：id: string
   - 返回：Promise<BaseResponse<APIKey>>
   - 调用：get(`/system/apikeys/${id}`)

5. 实现 updateAPIKey 函数：
   - 参数：id: string, data: UpdateAPIKeyRequest
   - 返回：Promise<BaseResponse<void>>
   - 调用：put(`/system/apikeys/${id}`, data)

6. 实现 deleteAPIKey 函数：
   - 参数：id: string
   - 返回：Promise<BaseResponse<void>>
   - 调用：del(`/system/apikeys/${id}`)

7. 实现 toggleAPIKeyStatus 函数：
   - 参数：id: string
   - 返回：Promise<BaseResponse<void>>
   - 调用：post(`/system/apikeys/${id}/toggle`)

8. 实现 listUsageLogs 函数：
   - 参数：keyID: string, params?: { current: number; pageSize: number }
   - 返回：Promise<BaseResponse<PageData<APIKeyUsageLog>>>
   - 调用：post(`/system/apikeys/${keyID}/logs`, params)

9. 实现 getUsageSummary 函数：
   - 参数：keyID: string
   - 返回：Promise<BaseResponse<UsageSummary>>
   - 调用：get(`/system/apikeys/${keyID}/summary`)

10. 导出所有函数（export）

使用项目统一的 API 调用方式（@/lib/api.ts）
错误处理由调用方处理
  </action>
  <verify>
    <automated>grep -E "export function listAPIKeys|export function createAPIKey|export function updateAPIKey|export function deleteAPIKey|export function toggleAPIKeyStatus|export function listUsageLogs|export function getUsageSummary" xingran-react-frontend/src/api/apikey.ts</automated>
  </verify>
  <done>
    - 所有 API 函数实现正确
    - HTTP 方法正确（get/post/put/del）
    - 路径正确
    - 类型定义正确
    - 导出语句正确
  </done>
</task>

<task type="auto" tdd="false">
  <name>Task 3: 验证类型检查和编译</name>
  <files>-</files>
  <read_first>
    - xingran-react-frontend/src/types/apikey.ts
    - xingran-react-frontend/src/api/apikey.ts
  </read_first>
  <action>
验证 TypeScript 类型检查和编译：

1. 运行类型检查：
   - cd xingran-react-frontend
   - npm run type-check
   - 确保无类型错误

2. 验证类型导出：
   - 所有接口正确导出
   - API 函数类型签名正确

3. 检查导入路径：
   - @/lib/api 导入正确
   - @/types/apikey 导入正确

4. 验证与后端 API 一致性：
   - 路径与后端路由匹配
   - 请求/响应类型匹配
   - 分页参数格式一致

5. 准备前端组件集成：
   - 确认所有类型可被组件导入
   - 确认 API 函数可被组件调用
  </action>
  <verify>
    <automated>cd xingran-react-frontend && npm run type-check</automated>
  </verify>
  <done>
    - TypeScript 类型检查通过
    - 所有类型正确导出
    - 导入路径正确
    - 与后端 API 一致
  </done>
</task>

</tasks>

<threat_model>
## Trust Boundaries

| Boundary | Description |
|----------|-------------|
| 类型定义 | TypeScript 类型需要与后端 API 保持一致 |
| API 调用 | API 请求需要正确的错误处理和超时控制 |

## STRIDE Threat Register

| Threat ID | Category | Component | Disposition | Mitigation Plan |
|-----------|----------|-----------|-------------|-----------------|
| T-16-21 | Information Disclosure | 类型定义 | mitigate | 类型定义不包含敏感信息，仅用于前端显示和验证 |
| T-16-22 | Tampering | 参数验证 | mitigate | TypeScript 类型检查提供编译时验证，运行时由后端验证 |
| T-16-23 | Elevation of Privilege | 权限检查 | mitigate | 后端验证权限，前端仅做 UI 控制 |
</threat_model>

<verification>
1. 检查所有 TypeScript 文件是否存在且语法正确
2. 运行 npm run type-check 验证类型
3. 验证 API 客户端函数正确
4. 验证类型定义完整
5. 测试导入导出正确
</verification>

<success_criteria>
1. TypeScript 类型定义完整且正确
2. API 客户端函数封装正确
3. 类型检查通过
4. 与后端 API 一致
5. 准备好与页面组件集成
</success_criteria>

<output>
执行完成后，创建 .planning/phases/16-api-key-mgt/16-05a-SUMMARY.md
</output>
