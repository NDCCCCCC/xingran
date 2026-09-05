---
phase: 94
plan: 01
subsystem: frontend-api-layer
tags: [api-factory, crud, blob-download, contract-tests, typescript-generics]
requires:
  - opsApi.ts:49-97 私有 createCrudApi<T>（提升源，本 plan 只读不改）
  - src/lib/api.ts post/get 传输层（零改动消费）
  - src/types/base.ts BaseResponse/PageResponse/PageParams 类型地基
  - src/utils/authHelpers.ts getAccessToken 异步签名
provides:
  - src/lib/apiFactory.ts createResourceApi<T> 8 方法共享工厂 + CrudApiConfig + DropdownOption（94-02/94-03 消费契约）
  - src/types/apiFactory.ts ServerGeneratedKeys + CreatePayload<T> 派生类型
  - src/lib/download.ts blobAxios/extractFilenameFromBlobResponse/triggerBrowserDownload/downloadFile/downloadFilePost（94-02 两处 blob 迁出的落点）
  - src/lib/apiFactory.test.ts + src/lib/download.test.ts 契约测试（D-11）
affects:
  - 94-02（opsApi/rpaApi 迁移：删私有工厂 + blob 迁出依赖本 plan 落点）
  - 94-03（扁平文件委托 + D-12 扫描测试依赖工厂权威路径）
tech-stack:
  added: []  # 零新依赖（typescript/vitest/axios 全部既有）
  patterns:
    - 共享泛型工厂提升（opsApi 私有工厂 → src/lib/apiFactory.ts 单一权威，D-01/D-03）
    - Omit 派生类型做编译期 payload 约束（CreatePayload<T>，D-02 折中版 Partial<CreatePayload<T>>）
    - axios 实例 + 异步 token 注入拦截器的独立下载链（download.ts，D-04）
key-files:
  created:
    - xingran-react-frontend/src/types/apiFactory.ts
    - xingran-react-frontend/src/lib/apiFactory.ts
    - xingran-react-frontend/src/lib/download.ts
    - xingran-react-frontend/src/lib/apiFactory.test.ts
    - xingran-react-frontend/src/lib/download.test.ts
  modified: []  # 纯增量 plan，零既有文件改动
decisions:
  - create/update 参数采用 Partial<CreatePayload<T>> 折中版（plan 锁定，不采用 Omit 严格版，避免 ~25 处调用点修补）
  - ServerGeneratedKeys 取双命名并集 9 键（camelCase + snake_case + createdBy/updatedBy）
  - 工厂/下载链均不 import opsApi（防 94-02 迁移成环）
  - Task 3 按计划顺序对已实现代码做契约锁定（非严格 TDD RED→GREEN，见 TDD Gate Compliance）
metrics:
  duration: 31m
  completed: 2026-09-05
  tasks: 3
  files_created: 5
---

# Phase 94 Plan 01: API 工厂基础设施 Summary

**共资源工厂 createResourceApi<T>(8 方法) + CreatePayload 派生类型 + blob 下载链 download.ts(GET/POST 双变体) + 双契约测试，全部新建文件零既有改动**

## What Was Built

- **src/types/apiFactory.ts**（35 行，纯 type-only 零运行时）：`ServerGeneratedKeys` 双命名并集联合类型（id/createdAt/updatedAt/deletedAt + created_at/updated_at/deleted_at + createdBy/updatedBy 共 9 键）+ `CreatePayload<T> = Omit<T, ServerGeneratedKeys>` 派生类型。
- **src/lib/apiFactory.ts**（85 行）：`createResourceApi<T>` 8 方法（list/get/create/update/delete/batch/statistics/searchOptions）自 opsApi.ts:49-97 逐行提升；唯一类型变更为 create/update 参数 `Partial<CreatePayload<T>>`（D-02）；statistics/searchOptions 保持内联与 `res.data ?? {}` / `res.data ?? []` 解包回退（D-09）；导出 DropdownOption（JSDoc 原样迁入）+ CrudApiConfig。
- **src/lib/download.ts**（92 行）：blobAxios 实例（300000ms 超时 + 异步 getAccessToken Bearer 注入拦截器，5min 超时注释全文迁移）+ extractFilenameFromBlobResponse / triggerBrowserDownload / downloadFile（GET）原样迁入并导出 + 新增 downloadFilePost（POST 变体，归一 excelApi.export / asset excel export / rpaApi.downloadReport 三处内联的目标签名）。
- **src/lib/apiFactory.test.ts**（145 行，D-11）：路径拼接八连 / dropdownPath 覆盖 / 泛型透传不解包 / 解包空回退 / batch 合并，共 6 用例。
- **src/lib/download.test.ts**（209 行，D-11）：拦截器 Bearer 注入 / GET 链含非 2xx 抛错 / POST 链含 %E6%A5%BC%E5%AE%87 URL 编码文件名提取与默认回退 / 触发顺序 createObjectURL→click→revokeObjectURL / timeout 300000 锁值，共 8 用例。

## Commits

| Task | Commit | Content |
| ---- | ------ | ------- |
| 1 | b3bc745 | feat(94-01): create shared resource factory (D-01/D-02/D-09) |
| 2 | 5d35520 | feat(94-01): create blob download chain module (D-04) |
| 3 | f1c35be | test(94-01): add D-11 contract tests for factory and download |

## Verification Results

- `npm run type-check` exit 0（CreatePayload 编译期约束在生产代码侧生效）
- `npm run lint` exit 0（0 errors；1390 warnings 全部为存量 no-explicit-any，新文件零命中）
- `npx vitest run src/lib/apiFactory.test.ts src/lib/download.test.ts`：14/14 绿
- `npx vitest run` 全量：553 文件 / 3794 测试全绿，13 个既有测试零回归（本 plan 未修改任何既有文件）
- `git diff cc77e81..HEAD` 仅含 5 个新建文件（566 insertions），零既有文件改动
- `grep "from \"./opsApi\""` 在 apiFactory.ts / download.ts / 两个测试文件中均 0 命中（无反向依赖）

## Deviations from Plan

**1. [Rule 3 - Blocking] worktree 缺 node_modules，junction 复用主仓依赖**
- **Found during:** Task 1 首次运行 `npm run type-check`（tsc 不存在）
- **Issue:** 隔离 worktree 未安装依赖，三件套 gate 无法运行
- **Fix:** `mklink /J` 将 worktree 的 `xingran-react-frontend/node_modules` 链接到主仓同名目录（同一 package-lock 谱系；node_modules 已被 .gitignore 覆盖，零 git 足迹，worktree 销毁时随之消失）
- **Files modified:** 无（gitignored 环境产物）

无其他偏离——plan 按原文执行（含 D-02 折中版定档、D-09 解包回退保留、Pitfall 8 负向类型断言规避）。

## TDD Gate Compliance

Task 3 标记 `tdd="true"`，但 plan 自身任务序为「Task 1/2 先实现 → Task 3 契约测试」（测试对象是已交付实现），故 RED 门在结构上不适用：测试首跑即绿属 plan 预期结果而非跳过 RED。已核实：(1) 测试按 behavior 清单真实断言实现行为（mockPost 路径/参数/返回值逐项验证，非恒真用例）；(2) plan frontmatter 为 `type: execute`（非 `type: tdd`），plan 级 RED/GREEN 双 commit 门不适用；(3) Task 3 变更为纯测试文件，按语义提交单个 `test(94-01)` commit。

## Security & Threat Notes

- T-94-01（token 注入）：blobAxios 拦截器自 opsApi.ts:322-328 原样迁移（异步 getAccessToken，无同步拼头），download.test.ts Test 1 锁注入行为。
- T-94-02（文件名解析）：沿用既有 decodeURIComponent + 引号剥离正则，未新造解析。
- T-94-03（baseURL）：沿用 VITE_API_BASE_URL env 模式回退 "/api/v1"，无硬编码内网地址。
- 零新依赖，无供应链新面（T-94-SC）。
- 无威胁面外溢：所有新增网络触点均在 plan threat_model 登记范围内。

## Known Stubs

无——全部交付物为完整实现与真实断言的契约测试；downloadFilePost 的消费方接线（excelApi/rpaApi 迁出）按 plan 留待 94-02，属计划内排程而非 stub。

## Self-Check: PASSED

- 6/6 文件存在（5 交付物 + 本 SUMMARY）
- 4/4 commit 存在：b3bc745 / 5d35520 / f1c35be + 本 SUMMARY 的 docs 提交（自身）
- 工作树干净；未触碰 STATE.md / ROADMAP.md（orchestrator 集中写）
