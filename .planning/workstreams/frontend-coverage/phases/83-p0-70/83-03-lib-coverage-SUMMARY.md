---
phase: "83"
plan: "83-03"
subsystem: frontend-lib-coverage
tags: [testing, coverage, lib, api, dual-track-mock, contract-tests, ratchet]
requires:
  - "83-02 utils 全清 (utils floor 89.7 + gate 真相源决策)"
provides:
  - "lib 目录 86.56% 语句覆盖率 (gate 口径, >= P0 目标 70%)"
  - "D-07 双轨 mock 可复用范式 (真模块链 + vi.mock 加密依赖)"
  - "axios 拦截器级测试基建 (实例捕获 + 拦截器 handler 提取)"
  - "lib floor 86.0 (ratchet 10.4 -> 86.0)"
affects:
  - ".coverage-fe-floors (lib 行)"
  - ".planning/frontend-coverage-baseline.md (追加 83-03 ratchet 行)"
tech-stack:
  added: []
  patterns:
    - "vi.hoisted + vi.mock(axios) 工厂捕获多实例 (rawAxios=created[0], api=created[1])"
    - "拦截器 handler 从 interceptors.request.use.mock.calls[0][0] 提取后直接驱动"
    - "setEncryption() 经真实 initEncryptionConfig + mock rawAxios.get 控制模块私有加密开关"
    - "可控 Promise (resolveRefresh/rejectRefresh) 驱动 401 并发刷新队列"
key-files:
  created:
    - "xingran-react-frontend/src/lib/api.test.ts"
    - "xingran-react-frontend/src/lib/security.test.ts"
    - "xingran-react-frontend/src/lib/echarts.test.ts"
    - "xingran-react-frontend/src/lib/queryKeys.test.ts"
    - "xingran-react-frontend/src/lib/opsApi.test.ts"
    - "xingran-react-frontend/src/lib/menuApi.test.ts"
    - "xingran-react-frontend/src/lib/profileApi.test.ts"
    - "xingran-react-frontend/src/lib/columnConfigApi.test.ts"
    - "xingran-react-frontend/src/lib/adDomainApi.test.ts"
    - "xingran-react-frontend/src/lib/assetApi.test.ts"
    - "xingran-react-frontend/src/lib/dutyApi.test.ts"
    - "xingran-react-frontend/src/lib/knowledgeApi.test.ts"
    - "xingran-react-frontend/src/lib/noticeApi.test.ts"
    - "xingran-react-frontend/src/lib/notificationConfigApi.test.ts"
    - "xingran-react-frontend/src/lib/rpaApi.test.ts"
    - "xingran-react-frontend/src/lib/vdiApi.test.ts"
    - "xingran-react-frontend/src/lib/workorderApi.test.ts"
    - "xingran-react-frontend/src/lib/api/__tests__/macHeatmapApi.test.ts"
  modified:
    - ".coverage-fe-floors"
    - ".planning/frontend-coverage-baseline.md"
decisions:
  - "gate json 复算 86.56% 为真相源 (v8 text 93.15% 不采信,重申 83-02 决策)"
  - "floor 86.56-0.5=86.06 按一位小数向下截断为 86.0 (对齐 82-02 GLOBAL 截断纪律,余量 0.56pp)"
  - "IN-06 未触发: 无 vitest 版本漂移导致的失败,package.json 不做版本统一变更"
metrics:
  duration: 26m
  completed: 2026-08-24
---

# Phase 83 Plan 03: lib 全清 (P0 基建层 ≥70%) Summary

**一句话:** lib 目录语句覆盖率 10.94% → **86.56%** (gate 口径 902/1042),以 D-07 双轨直测 (api.ts 真模块链 + mock 加密依赖) 加 14 个 API wrapper 端点契约测试达成,全部 631 测试通过零回归。

## 任务执行结果

| Task | 内容 | Commit | 验证 |
|------|------|--------|------|
| 1 | api.ts 双轨直测 (36 tests,≥12 要求) + security/echarts/queryKeys 套件 | `4df1019` | `npm run test` 全绿 |
| 2 | 14 个 API wrapper 端点契约测试 (vi.mock @/lib/api) | `c12e0a3` | `npm run test` 全绿 |
| 3 | 覆盖率验证 + floor ratchet 10.4 → 86.0 + 基线行追加 (D-11 同 commit) | `3d57a8e` | gate: lib PASS + GLOBAL PASS + 28/28 目录 PASS |

## 覆盖率结果 (gate 真相源)

| 指标 | 前 (83-02 后) | 后 (83-03) |
|------|---------------|------------|
| lib statements | 10.94% (114/1042) | **86.56% (902/1042)** |
| GLOBAL weighted avg | 7.46% (1609/21574) | **11.11% (2397/21574)** |
| 测试总数 | 399 (40 files) | **631 (58 files,零回归)** |
| lib floor | 10.4 | **86.0** |

Gate 终态输出: `PASS: lib 86.56% >= 86.0%` / `PASS: weighted avg 11.11% >= threshold 3.80%` / `PASS: per-dir floor gate — 28/28 directories >= floor`。

## D-07 双轨直测落地 (Task 1 核心)

- **真模块链:** api.test.ts 导入真实 `@/lib/api` 模块,initEncryptionConfig/加密开关/拦截器/包装函数全部走真实代码。
- **mock 边界 (vi.mock):** axios (工厂捕获双实例: rawAxios=created[0] 供 rawAxios.get 配置拉取,api=created[1] 为被测实例)、sm2/sm4、TokenManager、errorHandler、menuStore、antdMessage、encryptionConfig。
- **拦截器直驱:** 从 `interceptors.request.use.mock.calls[0][0]` 提取 request/response handler 直接调用,配合 AxiosHeaders/InternalAxiosRequestConfig stub,覆盖加密备份 (`__originalPlainData`)、密钥消费链、code 分支、错误映射。
- **关键链路 (威胁模型 T-83-03-01..04 全覆盖):** 加密请求信封 (仅 fake key,无真实密钥)、x-response-encrypted 响应解密、401 并发刷新队列 (isRefreshing + refreshQueue,可控 Promise 断言单次 refresh + 双请求重放)、400 解密失败单次重放 (`__sm2DecryptRetried` 防死循环)、login 401 短路 (不进刷新队列直接登出)。

## 14 个 wrapper 契约测试 (Task 2)

统一模式 `vi.mock("@/lib/api")` 锁定各模块端点 URL/请求体形状/返回透传: opsApi (blob 下载 + URL.createObjectURL stub)、menuApi、profileApi、columnConfigApi、adDomainApi (账号池 8 端点 + 分页归一)、assetApi (对账/修复建议 data-null 回退)、dutyApi、knowledgeApi、noticeApi (SecureTokenStorageImpl class mock 驱动 buildWebSocketUrl)、notificationConfigApi、rpaApi (10 子 API 聚合结构)、vdiApi (snake_case 请求体)、workorderApi (buildCategoryTree 纯函数)、macHeatmapApi。

## Deviations from Plan

### 计划内偏差 (条件分支未触发)

**IN-06 (package.json vitest 版本统一 ^4.1.10) 未触发** — 计划要求仅在版本漂移导致测试失败时执行;实际 631 测试零失败,依据计划条件分支不做变更。

### 范围外发现 (Scope Boundary — 已登记未修)

**1. adDomainApi.deleteMapping URL 疑似 bug**
- **Found during:** Task 2 (adDomainApi 契约测试)
- **Issue:** 源码 URL 模板 `/ad-domain/mappings/${id}/delete}` 末尾有多余 `}`;且后端 router 无 `/ad-domain/mappings` 路由注册,该 API 疑似整体不可用。
- **处置:** 契约测试按现状锁定实际 URL 并注释标记;不修业务代码 (超出本 plan 范围),登记 `deferred-items.md` 供后续 plan 处理。

### 测试迭代修正 (Task 1 编写期,非源码 bug)

- escapeHtml 期望值修正为真实双重转义行为;sanitizeObject 载荷改用含 HTML 特殊字符的 `<script>` 串;getCSPConfig 默认分支断言反转 (vitest DEV 默认 true);echarts/core 无 getVersion 导出改为断言实际导出面;`as const` 数组无运行时冻结改断言 Array.isArray + length。均为测试期望与真实行为对齐,源码零改动。

## Known Stubs

无。本 plan 全部产出为测试文件与 gate 配置,未触碰业务源码,无占位实现。

## Threat Flags

无新增攻击面。T-83-03-01 纪律已在测试中执行: 全部加解密用例仅使用 fake key/假密文,无真实密钥材料入测试。

## Self-Check: PASSED

18/18 created test files exist on disk; commits 4df1019 / c12e0a3 / 3d57a8e all present in git log.
