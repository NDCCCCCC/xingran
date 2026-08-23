---
phase: 83-p0-70
plan: 03
type: execute
wave: 2
depends_on:
  - 83-01
files_modified:
  - xingran-react-frontend/src/lib/api.test.ts
  - xingran-react-frontend/src/lib/opsApi.test.ts
  - xingran-react-frontend/src/lib/menuApi.test.ts
  - xingran-react-frontend/src/lib/profileApi.test.ts
  - xingran-react-frontend/src/lib/columnConfigApi.test.ts
  - xingran-react-frontend/src/lib/adDomainApi.test.ts
  - xingran-react-frontend/src/lib/assetApi.test.ts
  - xingran-react-frontend/src/lib/dutyApi.test.ts
  - xingran-react-frontend/src/lib/knowledgeApi.test.ts
  - xingran-react-frontend/src/lib/noticeApi.test.ts
  - xingran-react-frontend/src/lib/notificationConfigApi.test.ts
  - xingran-react-frontend/src/lib/rpaApi.test.ts
  - xingran-react-frontend/src/lib/vdiApi.test.ts
  - xingran-react-frontend/src/lib/workorderApi.test.ts
  - xingran-react-frontend/src/lib/api/__tests__/macHeatmapApi.test.ts
  - xingran-react-frontend/src/lib/security.test.ts
  - xingran-react-frontend/src/lib/echarts.test.ts
  - xingran-react-frontend/src/lib/queryKeys.test.ts
  - xingran-react-frontend/package.json
  - .coverage-fe-floors
  - .planning/frontend-coverage-baseline.md
autonomous: true
requirements:
  - INFRA-01
  - QUAL-01
user_setup: []
must_haves:
  truths:
    - "[D-12] lib 目录 20 文件 1042 stmts 的语句覆盖率从 10.94% 提升到 ≥70%"
    - "[D-07][D-09] api.ts 加密客户端完成双轨直测（拦截器、加密编排、401 刷新队列、400 解密失败重放，TokenManager 自动刷新使用 fake timers）"
    - 各 API wrapper 模块通过 vi.mock('@/lib/api') 完成端点契约测试
    - "[D-11] lib 目录 floor 被 bump 到实测 −0.5pp 并追加基线文档"
  artifacts:
    - path: xingran-react-frontend/src/lib/api.test.ts
      provides: api.ts 双轨直测
    - path: xingran-react-frontend/src/lib/*Api.test.ts
      provides: API wrapper 契约测试
    - path: .coverage-fe-floors
      provides: lib floor 更新
  key_links:
    - from: src/lib/api.test.ts
      to: src/lib/api.ts
      via: 真实 axios 实例 + vi.mock 底层依赖（sm2/sm4/TokenManager/errorHandler）
    - from: src/lib/*Api.test.ts
      to: src/lib/api.ts
      via: vi.mock('@/lib/api') 拦截 post/get/put/del
---

<objective>
将 lib 目录（20 文件 1042 stmts）语句覆盖率从 10.94% 提升到 ≥70%，其中 api.ts 采用真实链路 + mock 加密层的双轨直测覆盖拦截器、SM2/SM4 加密编排、401 Token 刷新队列、400 解密失败重放；其余 API wrapper 模块通过 vi.mock('@/lib/api') 做端点契约测试；并在同一 commit 中 bump lib floor 与基线文档。

Purpose: lib 是 hooks/store/组件的 HTTP 通信依赖，补齐后上层测试可安全 mock '@/lib/api'。
Output: 17+ 个测试文件、lib floor 提升、基线文档追加行。
</objective>

<execution_context>
@$HOME/.claude/get-shit-done/workflows/execute-plan.md
@$HOME/.claude/get-shit-done/templates/summary.md
</execution_context>

<context>
@.planning/workstreams/frontend-coverage/ROADMAP.md
@.planning/workstreams/frontend-coverage/REQUIREMENTS.md
@.planning/workstreams/frontend-coverage/phases/83-p0-70/83-CONTEXT.md
@.planning/workstreams/frontend-coverage/phases/83-p0-70/83-RESEARCH.md
@.planning/workstreams/frontend-coverage/phases/83-p0-70/83-PATTERNS.md
@xingran-react-frontend/src/lib/api.ts
@xingran-react-frontend/src/lib/opsApi.ts
@xingran-react-frontend/src/lib/menuApi.ts
@xingran-react-frontend/src/lib/profileApi.ts
@xingran-react-frontend/src/lib/columnConfigApi.ts
@xingran-react-frontend/src/lib/adDomainApi.ts
@xingran-react-frontend/src/lib/assetApi.ts
@xingran-react-frontend/src/lib/dutyApi.ts
@xingran-react-frontend/src/lib/knowledgeApi.ts
@xingran-react-frontend/src/lib/noticeApi.ts
@xingran-react-frontend/src/lib/notificationConfigApi.ts
@xingran-react-frontend/src/lib/rpaApi.ts
@xingran-react-frontend/src/lib/vdiApi.ts
@xingran-react-frontend/src/lib/workorderApi.ts
@xingran-react-frontend/src/lib/api/macHeatmapApi.ts
@xingran-react-frontend/src/lib/security.ts
@xingran-react-frontend/src/lib/echarts.ts
@xingran-react-frontend/src/lib/queryKeys.ts
@xingran-react-frontend/src/lib/__tests__/loginPreflight.test.ts
@xingran-react-frontend/src/lib/api/__tests__/networkApi.test.ts
@xingran-react-frontend/package.json
@.coverage-fe-floors
@.planning/frontend-coverage-baseline.md
</context>

<tasks>

<task type="auto">
  <name>Task 1: api.ts 双轨直测与基础 lib 工具测试</name>
  <files>
    xingran-react-frontend/src/lib/api.test.ts
    xingran-react-frontend/src/lib/security.test.ts
    xingran-react-frontend/src/lib/echarts.test.ts
    xingran-react-frontend/src/lib/queryKeys.test.ts
    xingran-react-frontend/package.json
  </files>
  <read_first>
    - xingran-react-frontend/src/lib/api.ts（拦截器、加密编排、401/400 处理）
    - xingran-react-frontend/src/lib/security.ts
    - xingran-react-frontend/src/lib/echarts.ts
    - xingran-react-frontend/src/lib/queryKeys.ts
    - xingran-react-frontend/src/utils/sm2.ts
    - xingran-react-frontend/src/utils/sm4.ts
    - xingran-react-frontend/src/utils/token/TokenManager.ts
    - xingran-react-frontend/src/utils/errorHandler.ts
    - xingran-react-frontend/src/lib/__tests__/loginPreflight.test.ts（vi.mock 与 fake timers 模式）
    - xingran-react-frontend/package.json（确认 axios 版本，判断是否需要 axios-mock-adapter）
  </read_first>
  <action>
    1. 创建 src/lib/api.test.ts：
       - 使用 vi.spyOn(axios, 'create') 或 vi.mock('axios') 控制 axios 实例，推荐 vi.mock('axios') 返回可配置的 mock adapter，避免引入 axios-mock-adapter 新依赖。
       - mock 底层依赖：@/utils/sm2（fetchPublicKey 返回固定公钥、clearPublicKeyCache）、@/utils/sm4（generateSM4Key/generateIV/encryptRequestBody/decryptSM4CBC/hexToBase64/base64ToHex 返回确定性值）、@/utils/token/TokenManager（getAccessToken/refreshToken/clearTokens）、@/store/menuStore（clearMenus）、@/utils/errorHandler（handleHttpResponseError/handleNetworkError）、@/utils/antdMessage（getAppMessage）。
       - 测试分支：initEncryptionConfig 成功/失败重试、refreshEncryptionConfig 成功/失败、get/post/put/del/upload/postFormData/postLongRequest 调用 axios 方法、请求拦截器附加 Authorization 与 X-Request-ID、请求加密 POST 体（启用加密时）、响应解密（encrypted/data/iv 与 x-response-encrypted 头）、401 非登录接口触发刷新队列（一个请求 401，第二个请求入队，刷新成功后两个请求都重试）、401 登录接口短路返回后端 message、401 refresh 失败跳转登录、400 "解密失败" 清缓存重放一次、无响应网络错误走 handleNetworkError。
    2. 创建 src/lib/security.test.ts：覆盖 XSS 检测/转义、URL/对象/数组场景。
    3. 创建 src/lib/echarts.test.ts：覆盖图表配置辅助函数与 dispose 包装。
    4. 创建 src/lib/queryKeys.test.ts：覆盖各 query key 工厂函数输出结构。
    5. 可选：若 package.json 中 axios 版本与 mock 方案不兼容，统一 vitest 生态包声明为 ^4.1.10（IN-06）。仅当发现版本声明失同步导致测试异常时才执行；否则不修改 package.json。
  </action>
  <verify>
    <automated>
      cd xingran-react-frontend && npx vitest run src/lib/api.test.ts src/lib/security.test.ts src/lib/echarts.test.ts src/lib/queryKeys.test.ts --reporter=verbose
    </automated>
  </verify>
  <done>
    - api.ts 双轨直测与基础 lib 工具测试创建并通过，覆盖 api.ts 主要分支。
  </done>
  <acceptance_criteria>
    - src/lib/api.test.ts 包含至少 12 个 it 块，覆盖加密请求、响应解密、401 刷新队列、400 重放、登录 401 短路。
    - src/lib/security.test.ts 覆盖 containsXSS 与 escapeHtml。
    - src/lib/echarts.test.ts 与 src/lib/queryKeys.test.ts 通过。
    - 四个测试文件全部通过（exit 0）。
  </acceptance_criteria>
</task>

<task type="auto">
  <name>Task 2: API wrapper 模块端点契约测试</name>
  <files>
    xingran-react-frontend/src/lib/opsApi.test.ts
    xingran-react-frontend/src/lib/menuApi.test.ts
    xingran-react-frontend/src/lib/profileApi.test.ts
    xingran-react-frontend/src/lib/columnConfigApi.test.ts
    xingran-react-frontend/src/lib/adDomainApi.test.ts
    xingran-react-frontend/src/lib/assetApi.test.ts
    xingran-react-frontend/src/lib/dutyApi.test.ts
    xingran-react-frontend/src/lib/knowledgeApi.test.ts
    xingran-react-frontend/src/lib/noticeApi.test.ts
    xingran-react-frontend/src/lib/notificationConfigApi.test.ts
    xingran-react-frontend/src/lib/rpaApi.test.ts
    xingran-react-frontend/src/lib/vdiApi.test.ts
    xingran-react-frontend/src/lib/workorderApi.test.ts
    xingran-react-frontend/src/lib/api/__tests__/macHeatmapApi.test.ts
  </files>
  <read_first>
    - xingran-react-frontend/src/lib/opsApi.ts
    - xingran-react-frontend/src/lib/menuApi.ts
    - xingran-react-frontend/src/lib/profileApi.ts
    - xingran-react-frontend/src/lib/columnConfigApi.ts
    - xingran-react-frontend/src/lib/adDomainApi.ts
    - xingran-react-frontend/src/lib/assetApi.ts
    - xingran-react-frontend/src/lib/dutyApi.ts
    - xingran-react-frontend/src/lib/knowledgeApi.ts
    - xingran-react-frontend/src/lib/noticeApi.ts
    - xingran-react-frontend/src/lib/notificationConfigApi.ts
    - xingran-react-frontend/src/lib/rpaApi.ts
    - xingran-react-frontend/src/lib/vdiApi.ts
    - xingran-react-frontend/src/lib/workorderApi.ts
    - xingran-react-frontend/src/lib/api/macHeatmapApi.ts
    - xingran-react-frontend/src/lib/api/__tests__/networkApi.test.ts（端点契约测试模板）
  </read_first>
  <action>
    1. 为每个 API wrapper 模块创建同目录 *.test.ts，使用 vi.mock('@/lib/api', () => ({ post: mockPost, get: mockGet, put: mockPut, del: mockDel, upload: mockPost, postFormData: mockPost, postLongRequest: mockPost }))。
    2. 每个测试文件覆盖：list/create/update/delete/export/import/downloadTemplate/geocode 等公开方法调用正确端点、传递正确参数、返回 response.data；对 CRUD 工厂（opsApi）验证工厂生成的对象结构一致。
    3. 优先覆盖导出函数数量多、语句数高的模块（opsApi、menuApi、profileApi、columnConfigApi），对简单 queryKeys 模块已在 Task 1 覆盖。
    4. 不测试真实网络；所有 API 调用通过 mock 断言。
  </action>
  <verify>
    <automated>
      cd xingran-react-frontend && npx vitest run src/lib --reporter=verbose
    </automated>
  </verify>
  <done>
    - lib 目录新增 API wrapper 测试全部通过。
  </done>
  <acceptance_criteria>
    - src/lib 下新增 *.test.ts 文件数量 ≥14。
    - npx vitest run src/lib 输出 Tests ... passed，exit 0。
    - 159 存量测试不回归（后续 Task 3 通过 test:coverage 复核）。
  </acceptance_criteria>
</task>

<task type="auto">
  <name>Task 3: 覆盖率验证与 ratchet bump</name>
  <files>
    .coverage-fe-floors
    .planning/frontend-coverage-baseline.md
  </files>
  <read_first>
    - .coverage-fe-floors（确认 lib 当前 floor）
    - .planning/frontend-coverage-baseline.md（确认追加行格式）
    - .github/scripts/check-frontend-coverage.sh（确认 gate 输出格式）
  </read_first>
  <action>
    1. 运行 cd xingran-react-frontend && npm run test:coverage。
    2. 运行 bash .github/scripts/check-frontend-coverage.sh xingran-react-frontend/coverage/coverage-final.json .coverage-fe-floors。
    3. 读取 gate 输出 lib 目录实测 pct，将 .coverage-fe-floors 中 lib 行从 10.4 bump 至 max(70.0, pct − 0.5) 并保留一位小数。
    4. 在 .planning/frontend-coverage-baseline.md 追加 ratchet 行：日期、phase 83-03、weighted_avg、total_stmts、total_covered、0pct_pkg_count、当前 commit 短 SHA、ratchet_from、ratchet_to。
    5. 重新运行 gate 脚本确认 lib 行 PASS 且 global PASS。
  </action>
  <verify>
    <automated>
      cd xingran-react-frontend && npm run test:coverage 全量通过；
      bash .github/scripts/check-frontend-coverage.sh xingran-react-frontend/coverage/coverage-final.json .coverage-fe-floors 输出 PASS: lib ... >= 70.0%。
    </automated>
  </verify>
  <done>
    - lib 目录 floor 更新，基线文档追加，gate 通过。
  </done>
  <acceptance_criteria>
    - .coverage-fe-floors 中 lib 行 ≥70.0。
    - .planning/frontend-coverage-baseline.md 新增 ratchet 行包含 83-03、commit SHA、lib ratchet 值。
    - gate 脚本输出 lib PASS 与 global PASS。
    - npm run test:coverage 全量通过。
  </acceptance_criteria>
</task>

</tasks>

<threat_model>
## Trust Boundaries

| Boundary | Description |
|----------|-------------|
| api.test.ts → axios | 通过 vi.mock('axios') 拦截，不发出真实 HTTP 请求 |
| api.test.ts → sm2/sm4/TokenManager | 通过 vi.mock 拦截，使用固定密钥向量 |
| API wrapper 测试 → @/lib/api | 通过 vi.mock('@/lib/api') 拦截，验证端点契约 |

## STRIDE Threat Register

| Threat ID | Category | Component | Disposition | Mitigation Plan |
|-----------|----------|-----------|-------------|-----------------|
| T-83-03-01 | Information Disclosure | api.ts 测试 | mitigate | 使用假 token、假 SM2 公钥、假 SM4 密钥，不引入真实凭证 |
| T-83-03-02 | Tampering | SM2 公钥缓存 400 重放 | mitigate | 测试覆盖 400 "解密失败" 分支：清缓存、恢复明文、重新加密重放 |
| T-83-03-03 | Denial of Service | 401 刷新队列 | mitigate | 测试覆盖并发 401 只触发一次刷新，队列中请求复用结果 |
| T-83-03-04 | Elevation of Privilege | 登录 401 短路 | mitigate | 测试验证登录 401 不走刷新队列，直接 reject 后端 message |
| T-83-03-SC | Tampering | npm/pip/cargo installs | accept | 不引入 axios-mock-adapter 等新包；必要时仅统一 vitest 生态声明 |
</threat_model>

<verification>
1. cd xingran-react-frontend && npm run test:coverage 全量通过，159 存量测试不回归。
2. bash .github/scripts/check-frontend-coverage.sh xingran-react-frontend/coverage/coverage-final.json .coverage-fe-floors 通过，lib 行 PASS。
3. git diff 检查：仅新增测试文件、.coverage-fe-floors lib 行、.planning/frontend-coverage-baseline.md 追加行（可能含 package.json 版本统一）。
4. 抽样检查 gate 输出中 lib pct ≥70.00%。
</verification>

<success_criteria>
- lib 目录 statements 覆盖率 ≥70%（gate 输出 PASS）。
- api.ts 双轨直测覆盖加密编排、401 队列、400 重放、响应解密。
- 各 API wrapper 模块完成端点契约测试。
- .coverage-fe-floors 中 lib floor bump 至 ≥70.0，基线文档追加同一 commit。
- 全量 vitest 0 失败，159 存量测试不回归。
</success_criteria>

<output>
Create `.planning/workstreams/frontend-coverage/phases/83-p0-70/83-03-lib-coverage-SUMMARY.md` when done
</output>
