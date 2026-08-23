---
phase: 83-p0-70
plan: 02
type: execute
wave: 2
depends_on:
  - 83-01
files_modified:
  - xingran-react-frontend/src/utils/sm4.test.ts
  - xingran-react-frontend/src/utils/encoding.test.ts
  - xingran-react-frontend/src/utils/token/TokenManager.test.ts
  - xingran-react-frontend/src/utils/token/SecureTokenStorageImpl.test.ts
  - xingran-react-frontend/src/utils/dualLevelCache.test.ts
  - xingran-react-frontend/src/utils/geocodingCache.test.ts
  - xingran-react-frontend/src/utils/errorHandler.test.ts
  - xingran-react-frontend/src/utils/authHelpers.test.ts
  - xingran-react-frontend/src/utils/deptUtils.test.ts
  - xingran-react-frontend/src/utils/datetime.test.ts
  - xingran-react-frontend/src/utils/duration.test.ts
  - xingran-react-frontend/src/utils/lruCache.test.ts
  - xingran-react-frontend/src/utils/buildSearchParams.test.ts
  - xingran-react-frontend/src/utils/cad/geometry.test.ts
  - xingran-react-frontend/src/utils/three/colors.test.ts
  - xingran-react-frontend/src/utils/typeGuards.test.ts
  - xingran-react-frontend/src/utils/baidu-map.test.ts
  - xingran-react-frontend/src/utils/iconUtils.test.tsx
  - xingran-react-frontend/src/utils/tableHelpers.test.tsx
  - xingran-react-frontend/src/utils/debounce.test.ts
  - xingran-react-frontend/src/utils/antdMessage.test.ts
  - .coverage-fe-floors
  - .planning/frontend-coverage-baseline.md
autonomous: true
requirements:
  - INFRA-02
  - QUAL-01
user_setup: []
must_haves:
  truths:
    - "[D-10][D-12] utils 目录 23 文件 950 stmts 的语句覆盖率从 8.21% 提升到 ≥70%"
    - "新增测试覆盖国密工具（D-08）、token 生命周期（D-09 fake timers）、缓存、错误处理、鉴权辅助、部门工具、日期时间、防抖、message 桥接等核心 utils"
    - "[D-11] utils 目录 floor 在 .coverage-fe-floors 中被 bump 到实测 −0.5pp 并追加基线文档"
  artifacts:
    - path: xingran-react-frontend/src/utils/*.test.ts
      provides: utils 层测试文件
    - path: .coverage-fe-floors
      provides: utils floor 更新
    - path: .planning/frontend-coverage-baseline.md
      provides: ratchet 历史追加行
  key_links:
    - from: utils tests
      to: src/utils/*.ts
      via: 同目录 *.test.ts 导入被测模块
      pattern: "import.*from \"\\./(sm4|encoding|dualLevelCache|errorHandler|authHelpers|deptUtils|datetime|debounce|antdMessage)\""
---

<objective>
将 utils 目录（23 文件 950 stmts）语句覆盖率从 8.21% 提升到 ≥70%，覆盖国密算法、token 生命周期、缓存、错误处理、鉴权辅助、部门/日期/几何/颜色/防抖/message 桥接等工具函数，并在同一 commit 中 bump utils floor 与基线文档。

Purpose: utils 是 frontend 最底层依赖，先清 utils 可为 lib/hooks/store 测试提供 mock 与确定性工具。
Output: 18+ 个测试文件、utils floor 提升、基线文档追加行。
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
@xingran-react-frontend/src/utils/sm2.test.ts
@xingran-react-frontend/src/utils/sm4.ts
@xingran-react-frontend/src/utils/encoding.ts
@xingran-react-frontend/src/utils/token/TokenManager.ts
@xingran-react-frontend/src/utils/token/SecureTokenStorageImpl.ts
@xingran-react-frontend/src/utils/dualLevelCache.ts
@xingran-react-frontend/src/utils/geocodingCache.ts
@xingran-react-frontend/src/utils/errorHandler.ts
@xingran-react-frontend/src/utils/authHelpers.ts
@xingran-react-frontend/src/utils/deptUtils.ts
@xingran-react-frontend/src/utils/datetime.ts
@xingran-react-frontend/src/utils/duration.ts
@xingran-react-frontend/src/utils/lruCache.ts
@xingran-react-frontend/src/utils/buildSearchParams.ts
@xingran-react-frontend/src/utils/cad/geometry.ts
@xingran-react-frontend/src/utils/three/colors.ts
@xingran-react-frontend/src/utils/typeGuards.ts
@xingran-react-frontend/src/utils/baidu-map.ts
@xingran-react-frontend/src/utils/iconUtils.tsx
@xingran-react-frontend/src/utils/tableHelpers.tsx
@xingran-react-frontend/src/utils/debounce.ts
@xingran-react-frontend/src/utils/antdMessage.ts
@xingran-react-frontend/src/test/setup.ts
@.coverage-fe-floors
@.planning/frontend-coverage-baseline.md
</context>

<tasks>

<task type="auto">
  <name>Task 1: 国密与 token 工具真实向量直测</name>
  <files>
    xingran-react-frontend/src/utils/sm4.test.ts
    xingran-react-frontend/src/utils/encoding.test.ts
    xingran-react-frontend/src/utils/token/TokenManager.test.ts
    xingran-react-frontend/src/utils/token/SecureTokenStorageImpl.test.ts
  </files>
  <read_first>
    - xingran-react-frontend/src/utils/sm4.ts（加密函数与导出）
    - xingran-react-frontend/src/utils/encoding.ts（hex/base64/bytes 转换函数）
    - xingran-react-frontend/src/utils/token/TokenManager.ts（TokenManager 类与配置接口）
    - xingran-react-frontend/src/utils/token/SecureTokenStorageImpl.ts（实现与接口）
    - xingran-react-frontend/src/utils/sm2.test.ts（既有国密测试模式参考）
    - xingran-react-frontend/src/lib/__tests__/loginPreflight.test.ts（fake timers 与 vi.mock 模式）
  </read_first>
  <action>
    1. 创建 src/utils/encoding.test.ts：覆盖 hexToBase64/base64ToHex/bytesToHex/hexToBytes/generateRandomHex 的往返、空输入、非法输入、边界长度。
    2. 创建 src/utils/sm4.test.ts：使用确定性 SM4 密钥与 IV 覆盖 encryptSM4CBC/decryptSM4CBC 往返、空明文、密文篡改抛错、encryptRequestBody/decryptRequestBody JSON 往返、encryptPasswordWithSM4 ECB 模式、generateSessionKey 长度、isSM4Available 返回 true（sm-crypto 已安装）。
    3. 创建 src/utils/token/SecureTokenStorageImpl.test.ts：覆盖 setAccessToken/getAccessToken/removeAccessToken、setRefreshToken 加密写入与解密读取、sessionStorage 损坏数据回退 null、SM4 密钥边界。
    4. 创建 src/utils/token/TokenManager.test.ts：使用 vi.useFakeTimers() 覆盖 initializeTokens、getAccessToken 在过期前 refreshBeforeSeconds 自动触发刷新、并发刷新队列只发一次请求、refreshToken 失败清理状态并停止后续刷新、clearTokens。
    5. 所有国密测试使用真实 sm-crypto（D-08），TokenManager 使用 fake timers（D-09），业务 API 调用使用 vi.mock('@/lib/api') 或 vi.mock('@/utils/sm4') 解耦。
  </action>
  <verify>
    <automated>
      cd xingran-react-frontend && npx vitest run src/utils/sm4.test.ts src/utils/encoding.test.ts src/utils/token/SecureTokenStorageImpl.test.ts src/utils/token/TokenManager.test.ts --reporter=verbose
    </automated>
  </verify>
  <done>
    - 国密与 token 测试文件创建并通过，覆盖 sm4/encoding/token 目录核心分支。
  </done>
  <acceptance_criteria>
    - src/utils/sm4.test.ts 包含至少 8 个 it 块，覆盖 CBC 往返、篡改报错、ECB、request body、isSM4Available。
    - src/utils/encoding.test.ts 包含至少 6 个 it 块，覆盖 hex/base64/bytes 往返与非法输入。
    - src/utils/token/TokenManager.test.ts 使用 vi.useFakeTimers，覆盖自动刷新与并发队列。
    - src/utils/token/SecureTokenStorageImpl.test.ts 覆盖 refresh token 加密写入与损坏回退。
    - 四个测试文件全部通过（exit 0）。
  </acceptance_criteria>
</task>

<task type="auto">
  <name>Task 2: 缓存、错误处理、鉴权辅助与通用工具测试</name>
  <files>
    xingran-react-frontend/src/utils/dualLevelCache.test.ts
    xingran-react-frontend/src/utils/geocodingCache.test.ts
    xingran-react-frontend/src/utils/errorHandler.test.ts
    xingran-react-frontend/src/utils/authHelpers.test.ts
    xingran-react-frontend/src/utils/deptUtils.test.ts
    xingran-react-frontend/src/utils/datetime.test.ts
    xingran-react-frontend/src/utils/duration.test.ts
    xingran-react-frontend/src/utils/lruCache.test.ts
    xingran-react-frontend/src/utils/buildSearchParams.test.ts
    xingran-react-frontend/src/utils/typeGuards.test.ts
    xingran-react-frontend/src/utils/baidu-map.test.ts
    xingran-react-frontend/src/utils/cad/geometry.test.ts
    xingran-react-frontend/src/utils/three/colors.test.ts
    xingran-react-frontend/src/utils/iconUtils.test.tsx
    xingran-react-frontend/src/utils/tableHelpers.test.tsx
    xingran-react-frontend/src/utils/debounce.test.ts
    xingran-react-frontend/src/utils/antdMessage.test.ts
  </files>
  <read_first>
    - xingran-react-frontend/src/utils/dualLevelCache.ts
    - xingran-react-frontend/src/utils/geocodingCache.ts
    - xingran-react-frontend/src/utils/errorHandler.ts
    - xingran-react-frontend/src/utils/authHelpers.ts
    - xingran-react-frontend/src/utils/deptUtils.ts
    - xingran-react-frontend/src/utils/datetime.ts
    - xingran-react-frontend/src/utils/duration.ts
    - xingran-react-frontend/src/utils/lruCache.ts
    - xingran-react-frontend/src/utils/buildSearchParams.ts
    - xingran-react-frontend/src/utils/typeGuards.ts
    - xingran-react-frontend/src/utils/baidu-map.ts
    - xingran-react-frontend/src/utils/cad/geometry.ts
    - xingran-react-frontend/src/utils/three/colors.ts
    - xingran-react-frontend/src/utils/iconUtils.tsx
    - xingran-react-frontend/src/utils/tableHelpers.tsx
    - xingran-react-frontend/src/utils/debounce.ts
    - xingran-react-frontend/src/utils/antdMessage.ts
    - xingran-react-frontend/src/lib/__tests__/loginPreflight.test.ts
  </read_first>
  <action>
    1. dualLevelCache.test.ts：覆盖 L1 命中、L2 localStorage 命中、L2 缺失回源、TTL、delete、clear、序列化异常回退。
    2. geocodingCache.test.ts：覆盖 set/get/has/clear、TTL 过期、key 生成规则。
    3. errorHandler.test.ts：覆盖 handleHttpResponseError 各状态码、handleNetworkError、containsXSS、escapeHtml、message 提取边界。
    4. authHelpers.test.ts：覆盖 getAccessToken/getRefreshToken 正常路径与缺失、getAuthHeaders 构造、vi.mock('@/lib/api') 或 vi.mock('@/utils/token/SecureTokenStorage') 解耦。
    5. deptUtils.test.ts：覆盖树形扁平化、ID→名称映射、递归查找父节点/子节点、空值边界。
    6. datetime.test.ts：覆盖格式化、解析、相对时间、时区无关比较、空输入。
    7. duration.test.ts：覆盖秒/分/小时/天转换、human readable、零值与负值。
    8. lruCache.test.ts：覆盖 get/set/delete/clear/size、容量淘汰、访问顺序更新。
    9. buildSearchParams.test.ts：覆盖对象转 URLSearchParams、数组/空值/日期处理。
    10. typeGuards.test.ts：覆盖 isString/isNumber/isObject/isArray/isFunction 等。
    11. baidu-map.test.ts：覆盖坐标转换、地址解析辅助函数（mock @/lib/api）。
    12. cad/geometry.test.ts 与 three/colors.test.ts：覆盖纯几何函数与颜色辅助函数（不依赖 canvas/three 渲染上下文）。
    13. iconUtils.test.tsx 与 tableHelpers.test.tsx：覆盖 JSX 辅助函数与表格工具，使用 @testing-library/react 渲染断言。
    14. debounce.test.ts：覆盖 leading/trailing 选项、wait 默认 300ms、取消后执行、边界调用次数（13 stmts，虽小但属于 utils 分母，补齐可避免拉低目录 pct）。
    15. antdMessage.test.ts：覆盖 setAppMessageInstance/getAppMessage/noop 短路、实例注入与重置（5 stmts，测试简单且为 message mock 上游契约）。
  </action>
  <verify>
    <automated>
      cd xingran-react-frontend && npx vitest run src/utils --reporter=verbose
    </automated>
  </verify>
  <done>
    - utils 目录新增测试全部通过，高扇出文件优先覆盖，长尾小文件（debounce/antdMessage）已补齐。
  </done>
  <acceptance_criteria>
    - src/utils 下新增 *.test.ts(x) 文件数量 ≥17。
    - npx vitest run src/utils 输出 Tests ... passed，exit 0。
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
    - .coverage-fe-floors（确认 utils 当前 floor 与 ratchet 规则）
    - .planning/frontend-coverage-baseline.md（确认追加行格式）
    - .github/scripts/check-frontend-coverage.sh（确认 --init 与 gate 输出格式）
  </read_first>
  <action>
    1. 运行 cd xingran-react-frontend && npm run test:coverage，等待生成 coverage/coverage-final.json。
    2. 运行 bash .github/scripts/check-frontend-coverage.sh xingran-react-frontend/coverage/coverage-final.json .coverage-fe-floors，确认 gate 通过。
    3. 从 gate 输出读取 utils 目录实测 statements 覆盖率 pct，将 .coverage-fe-floors 中 utils 行从 7.7 bump 至 max(70.0, pct − 0.5) 并保留一位小数。
    4. 在 .planning/frontend-coverage-baseline.md 追加一行 ratchet 记录：日期、phase 83-02、weighted_avg、total_stmts、total_covered、0pct_pkg_count、当前 commit 短 SHA、ratchet_from、ratchet_to。
    5. 重新运行 gate 脚本确认 utils 行 PASS。
  </action>
  <verify>
    <automated>
      cd xingran-react-frontend && npm run test:coverage 输出 Tests 159 passed（加上新增测试后总数上升，但 159 存量不回归）；
      bash .github/scripts/check-frontend-coverage.sh xingran-react-frontend/coverage/coverage-final.json .coverage-fe-floors 输出 PASS: utils ... >= 70.0%。
    </automated>
  </verify>
  <done>
    - utils 目录 floor 更新，基线文档追加，gate 通过。
  </done>
  <acceptance_criteria>
    - .coverage-fe-floors 中 utils 行 ≥70.0。
    - .planning/frontend-coverage-baseline.md 新增 ratchet 行包含 83-02、commit SHA、utils ratchet 值。
    - gate 脚本输出 utils PASS。
    - npm run test:coverage 全量通过，无失败测试。
  </acceptance_criteria>
</task>

</tasks>

<threat_model>
## Trust Boundaries

| Boundary | Description |
|----------|-------------|
| 测试 → sm-crypto | 国密测试调用真实算法库，输入为确定性测试向量 |
| 测试 → sessionStorage/localStorage | jsdom storage 在每个测试进程隔离；测试后由 vitest 清理 |
| TokenManager → @/lib/api | 测试通过 vi.mock 拦截 refresh 端点，避免真实网络请求 |

## STRIDE Threat Register

| Threat ID | Category | Component | Disposition | Mitigation Plan |
|-----------|----------|-----------|-------------|-----------------|
| T-83-02-01 | Information Disclosure | SecureTokenStorageImpl 测试 | mitigate | 仅使用假 refresh token 测试向量，不引入真实凭证 |
| T-83-02-02 | Tampering | SM4 篡改测试 | mitigate | 使用真实 sm-crypto 验证篡改密文解密抛错（ASVS V6） |
| T-83-02-03 | Denial of Service | TokenManager fake timers | mitigate | 统一使用 vi.useFakeTimers()，避免真实短 TTL 导致 flaky（ASVS V2） |
| T-83-02-04 | Elevation of Privilege | TokenManager 并发刷新 | mitigate | 测试覆盖刷新锁与队列，验证仅一次刷新请求（ASVS V2） |
| T-83-02-SC | Tampering | npm/pip/cargo installs | accept | 本 plan 不引入新包；无安装步骤 |
</threat_model>

<verification>
1. cd xingran-react-frontend && npm run test:coverage 全量通过，159 存量测试不回归。
2. bash .github/scripts/check-frontend-coverage.sh xingran-react-frontend/coverage/coverage-final.json .coverage-fe-floors 通过，utils 行显示 PASS。
3. git diff 检查：仅新增测试文件、.coverage-fe-floors utils 行、.planning/frontend-coverage-baseline.md 追加行。
4. 抽样检查 gate 输出中 utils pct ≥70.00%。
</verification>

<success_criteria>
- utils 目录 statements 覆盖率 ≥70%（gate 输出 PASS）。
- 新增测试文件覆盖 sm4/encoding/token/cache/error/auth/dept/datetime/debounce/antdMessage 等核心 utils。
- .coverage-fe-floors 中 utils floor bump 至 ≥70.0，基线文档追加同一 commit。
- 全量 vitest 0 失败，159 存量测试不回归。
</success_criteria>

<output>
Create `.planning/workstreams/frontend-coverage/phases/83-p0-70/83-02-utils-coverage-SUMMARY.md` when done
</output>
