---
phase: 83-p0-70
plan: 83-02
subsystem: frontend/utils
tags: [coverage, utils, sm-crypto, token, cache, ratchet]
requires:
  - 83-01 (CR-01 gate 修复——测试 PR 不再踩 diff gate 红)
provides:
  - utils 目录 21 个新测试文件（sm4/encoding/token×2 + 17 通用工具）
  - utils per-dir floor 89.7（ratchet 基线，供后续 plan 只升不降）
affects:
  - .coverage-fe-floors（utils 行）
  - .planning/frontend-coverage-baseline.md（ratchet 表）
tech-stack:
  added: [] # 零新依赖（sm-crypto/@testing-library 均已有）
  patterns:
    - 真实 sm-crypto 算法 + 确定性向量直测（D-08，篡改密文实测抛 padding is invalid）
    - vi.useFakeTimers 接管 Date.now 驱动 TokenManager 刷新循环（D-09）
    - vi.hoisted + vi.mock("@/lib/api") 标准解耦（PATTERNS Shared Pattern 1）
    - fake timers 下 doRefresh 首个 await 需 advanceTimersByTimeAsync(0) 冲刷微任务后再断言
key-files:
  created:
    - xingran-react-frontend/src/utils/sm4.test.ts
    - xingran-react-frontend/src/utils/encoding.test.ts
    - xingran-react-frontend/src/utils/token/TokenManager.test.ts
    - xingran-react-frontend/src/utils/token/SecureTokenStorageImpl.test.ts
    - xingran-react-frontend/src/utils/{dualLevelCache,geocodingCache,errorHandler,authHelpers,deptUtils,datetime,duration,lruCache,buildSearchParams,typeGuards,baidu-map,debounce,antdMessage}.test.ts
    - xingran-react-frontend/src/utils/cad/geometry.test.ts
    - xingran-react-frontend/src/utils/three/colors.test.ts
    - xingran-react-frontend/src/utils/iconUtils.test.tsx
    - xingran-react-frontend/src/utils/tableHelpers.test.tsx
  modified:
    - .coverage-fe-floors
    - .planning/frontend-coverage-baseline.md
decisions:
  - utils floor = 89.7（gate json 复算实测 90.21 − 0.5 一位小数；v8 text reporter 88.28% 不作数，D-12 以 gate 输出为真相源）
  - QUAL-01 不在本 plan 标记 complete（REQUIREMENTS 映射 Phase 88 里程碑收口；本 plan 仅贡献 399 pass / 0 fail 证据）
  - filterExternalOrgDepts 按 actual 行为测试（externalOrg 节点的非 externalOrg 子节点被丢弃），源码 docstring 示例与实现不一致仅记录不修改（范围边界：不修业务代码）
metrics:
  duration: 22min
  completed: 2026-08-24
  tests: 399 passed / 0 failed（存量 159 无回归，新增 240）
  coverage: utils 8.21% → 90.21%（857/950 stmts）；全局加权 3.85% → 7.46%
---

# Phase 83 Plan 02: utils 目录全清 ≥70% Summary

**One-liner:** utils 层 23 文件 950 stmts 语句覆盖率 8.21% → **90.21%**（floor ratchet 7.7 → 89.7），21 个新测试文件以真实 sm-crypto 算法向量 + fake timers 覆盖国密、token 生命周期、双级缓存、错误处理与全部通用工具，全量 399 测试 0 失败。

## What Was Built

### Task 1 — 国密与 token 工具真实向量直测（commit 1639090）

- **sm4.test.ts**（18 it）：SM4-CBC 中文/JSON 往返、空输入短路、密文篡改抛错（真实 sm-crypto `padding is invalid`）、错误密钥拒绝、确定性密文回归样本、ECB 密码加密往返（raw sm-crypto 解密互证）、encryptRequestBody/decryptRequestBody JSON 往返与篡改、isSM4Available、三类密钥生成器长度/随机性、fetchSM4KeyForPassword 四分支（mock `@/lib/api` 的 get）。
- **encoding.test.ts**（14 it）：hex↔base64 已知向量（00ff↔AP8=、奇数位补 0）、bytes↔hex 全字节域（0x00-0xff）往返、ArrayBuffer 往返、非法 Base64 atob 抛错、generateRandomHex/Bytes。
- **token/SecureTokenStorageImpl.test.ts**（12 it）：AccessToken 纯内存、RefreshToken 真实 SM4 加密落 sessionStorage（密文不含明文片段）、损坏数据回退 null 并清除、TokenMeta 持久化跨实例恢复、损坏 JSON 静默兜底、clear、isAccessTokenExpiringWithin 四象限。
- **token/TokenManager.test.ts**（12 it）：fake timers 驱动 initializeTokens/getAccessToken 自动刷新（refreshBeforeSeconds 命中）、定时器 31s 精确触发、并发刷新 single-flight（3 调用 1 请求）、陈旧锁超时放行、401/NETWORK/SERVER 错误映射、失败后锁释放可重试、clearTokens 停表、TokenRefreshError 形状（覆盖 SecureTokenStorage.ts 语句）。

### Task 2 — 缓存/错误/鉴权/通用工具测试（commit 3e055af，17 文件）

- **dualLevelCache / geocodingCache**：L1/L2 命中与回填、TTL 过期回落（fake timers）、persistToStorage=false、损坏 JSON 回退、cleanup、统计 hitRate、单例。
- **errorHandler**：HTTP 状态码→文案映射（400/401/404/500/未知回落）、SM2 解密失败清公钥缓存（mock `@/utils/sm2`）、网络/解析错误、消息提取六级优先级、withErrorHandling、safeAsync/safeSync。
- **authHelpers**：getAuthHeaders Bearer 构造与空 token 短路、withAuth 头合并、refreshEncryptionConfig 清缓存重拉、getEncryptionConfigStatus fail-safe true（mock authStore/encryptionConfig）。
- **deptUtils**：多形状 nodeId、externalOrg 过滤、深度查找、子树收集、title 收窄、全路径/短名转换（startFromLevel=2 语义）、同 key 去重。
- **长尾补齐**：datetime（Z 后缀剥离时区无关）、duration（秒→人类可读全档位）、lruCache（容量淘汰/访问刷新）、buildSearchParams（空值过滤+orgId+分页）、typeGuards、baidu-map、cad/geometry（18 个纯几何函数含射线法多边形）、three/colors（状态/类型/标记色+回落）、iconUtils（图标渲染+搜索+去重）、tableHelpers（五类列工厂+排序器五类型+formatFileSize）、debounce（leading/trailing/默认 300ms）、antdMessage（noop 桥接）。

### Task 3 — 覆盖率验证与 ratchet bump（commit d9e52e6）

- `npm run test:coverage`：**399 passed / 0 failed**（存量 159 无回归）。
- gate 脚本：`PASS: utils 90.21% >= 89.7% (857/950 stmts)`，28/28 目录 PASS，GLOBAL 7.46% ≥ 3.8%。
- `.coverage-fe-floors` utils 行 7.7 → **89.7**；`.planning/frontend-coverage-baseline.md` 追加 ratchet 行（同 commit，D-11）。

## Verification

| 验收项 | 结果 |
|--------|------|
| `npx vitest run src/utils` exit 0 | ✅ 22 文件 241 测试全过 |
| `npm run test:coverage` 全量通过 | ✅ 399/399（159 存量不回归） |
| gate 输出 utils PASS ≥70 | ✅ 90.21%（floor 89.7） |
| floors utils 行 ≥70.0 | ✅ 89.7 |
| baseline 追加行含 83-02/SHA/ratchet 值 | ✅ d9e52e6, from 7.7 to 89.7 |
| tsc --noEmit / eslint | ✅ 0 error（pre-commit hook 双重把关） |

## Deviations from Plan

**1. [scope note] debounce 无 cancel API，"取消后执行"用例未写**
- debounce.ts 返回纯函数，无 .cancel 方法——计划中"取消后执行"测试点无对应实现。已覆盖 leading/trailing/默认 300ms/新周期等其余全部测试点。属计划对源码 API 的假设偏差，非缺陷。

**2. [scope note] filterExternalOrgDepts 源码 docstring 示例与实现不一致**
- docstring 声称 externalOrg 节点"连同其全部后代"保留，实现会丢弃其后代中无 externalOrg 的纯内部子树。按范围边界（不修业务代码）测试锁定 actual 行为，SUMMARY 记录供后续 plan 参考。

**3. [process] baseline 行 commit SHA 回填**
- ratchet 行 SHA（d9e52e6）无法自引用（fixed-point 不可能），沿用 82-05 回填先例：ratchet commit 落盘时置 pending，紧随其后的 docs commit 回填。

**4. [requirement scope] QUAL-01 未标记 complete**
- plan frontmatter 列了 QUAL-01，但 REQUIREMENTS.md 将其映射到 Phase 88（milestone 收口 gate"全量 0 失败 0 flaky"）。本 plan 仅贡献证据（399 pass/0 fail），在 Phase 88 收口时才标记。

## Known Stubs

None — 全部测试为真实实现直测，无占位/mock 桩数据流。

## Threat Flags

None — 零生产代码变更；无新增网络面/存储面。威胁登记表 T-83-02-01~04 全部以测试形式落地（假 token 向量 / 真实篡改抛错 / fake timers 防 flaky / single-flight 验证）。

## Self-Check: PASSED

- 13/13 关键文件存在（4 token/crypto + 9 通用工具代表 + floors + baseline）
- 3/3 任务 commit 存在：1639090 / 3e055af / d9e52e6
- utils 测试文件 22 个（utils 根 18 + token/ 2 + cad/ 1 + three/ 1，含既有 sm2.test.ts），`vitest run src/utils` 241 全过
