---
phase: 94
plan: 02
subsystem: frontend-api-layer
tags: [api-factory, crud, blob-download, migration, typescript-generics]
requires:
  - src/lib/apiFactory.ts createResourceApi<T> 8 方法共享工厂（94-01 产物）
  - src/lib/download.ts blobAxios/downloadFile/downloadFilePost（94-01 产物）
  - opsApi.ts:49-97 私有 createCrudApi（删除对象，D-03）
  - rpaApi.ts:49-79 第二份私有 createCrudApi + 裸 fetch downloadReport（删除对象，D-04/D-08）
provides:
  - opsApi.ts 纯消费方化（11 资源走 apiFactory + blob 域走 download.ts，D-03/D-04 达成）
  - rpaApi.ts 双工厂合并 + scriptApi spread 接入 + downloadReport 归一（D-08 达成）
  - vdiApi.ts vmApi SPREAD+OVERRIDE + vdiServerApi 纯 SPREAD（对象形态三文件全接入）
  - download.test.ts downloadReport 链路用例（D-11 组 6 落地）
affects:
  - 94-03（扁平文件委托 + D-12 扫描测试：对象形态硬档文件已全部工厂化）
  - 27 个 opsApi 生产消费文件 + rpaApi/vdiApi 消费文件（零改动，签名不变承诺兑现）
tech-stack:
  added: []  # 零新依赖
  patterns:
    - 删 + 换 import + spread 等价替换（零签名变化迁移）
    - SPREAD+OVERRIDE（vmApi list/create 原样覆盖防 interface 直传 TS2345）
    - POST-blob 内联三处收敛 downloadFilePost 单链
key-files:
  created: []
  modified:
    - xingran-react-frontend/src/lib/opsApi.ts
    - xingran-react-frontend/src/lib/opsApi.test.ts
    - xingran-react-frontend/src/lib/rpaApi.ts
    - xingran-react-frontend/src/lib/vdiApi.ts
    - xingran-react-frontend/src/lib/download.test.ts
decisions:
  - assetApi.excel.export 委托 downloadFilePost 后额外获得 content-disposition 文件名提取（plan 预告的 D-04 白得改善，原固定文件名降级为默认回退）
  - excelApi.export 非 2xx 错误文案由「导出失败」归一为 downloadFilePost 语义「下载失败: <默认文件名>」（无既有断言依赖旧文案，测试零适配）
  - vdiApi update/get/delete 走工厂 spread（生产零消费者，返回类型 BaseResponse<void>→BaseResponse<unknown> 无编译影响，type-check 仲裁通过）
  - STATE.md/ROADMAP.md 不由 executor 触碰，沿用 94-01 惯例由 orchestrator 集中写 tracking
metrics:
  duration: 49m
  completed: 2026-09-05
  tasks: 3
  files_created: 0
  files_modified: 5
---

# Phase 94 Plan 02: 对象形态 API 迁移 Summary

**opsApi/rpaApi/vdiApi 三文件以「删 + 换 import + spread」等价替换接入 94-01 共享工厂——双份私有工厂清零、blob 下载三处重复收敛、裸 fetch 消灭（白得 5min 超时），全部导出签名零变化**

## What Was Built

- **opsApi.ts**（792→652 行，净 −140）：私有 `createCrudApi<T>`/`CrudApiConfig` 整段删除，11 资源实例（building/floor/workstation/serverRoom/roomDevice/dedicatedLine/infoPoint/wall/door/floorPlanText/asset）改 `import { createResourceApi } from "./apiFactory"`，实参零改动（D-03）；`DropdownOption` 定义删除改 `export type { DropdownOption } from "./apiFactory"` 兜底 re-export（外部零消费方核实）；blob 四件套（blobAxios + 拦截器 + extractFilenameFromBlobResponse + triggerBrowserDownload + downloadFile）删除改消费 download.ts（D-04）；`excelApi.export` 与 `assetApi.excel.export` 内联 POST-blob 链各收敛为一行 `downloadFilePost` 委托；KEEP 区（locationAliasApi/roomPhotoApi/workstationDeviceApi/geocode 函数族/componentApi/wall-door-floorPlanText batch 覆盖层/assetApi.statistics 覆盖）经 `git diff -U0` 逐行核验零改动。
- **opsApi.test.ts**（612→613 行，适配 12 行）：`h.created[0]` 位置假设与 `blobRequestInterceptor` 位置猜测删除，改 `import { blobAxios } from "./download"` 直取实例 + `mockedBlobAxios` 测试视型（Pitfall 6）；excel/blob 断言链 URL/参数断言原样保留；其余 ~600 行零改动。
- **rpaApi.ts**（753→696 行，净 −57）：第二份私有工厂（5 方法版 + `CrudApiConfig<_T>` 死泛型）删除，7 个 spread 实例（task/worker/execution/schedule/variable/template/notification）换 `createResourceApi`（D-08 以 opsApi 8 方法版为权威，additive 新增 batch/statistics/searchOptions 行为中性）；`scriptApi` 手写五方法改 `{ ...scriptCrud, testAction, format }` spread 接入；`executionApi.downloadReport` 裸 fetch 全文删除改一行 `downloadFilePost` 委托（URL 语义保持 format 进 query、默认文件名 `execution_report_<id>.<format>`），`getAuthHeaders` import 随之删除（唯一使用点）；aiApi/statisticsApi/各主体异构方法零改动。
- **vdiApi.ts**（190→172 行，净 −18）：`vmApi` SPREAD+OVERRIDE——`get/update/delete` 来自工厂 spread，`list`（VMListParams interface 变量直传会被工厂 `PageParams & Record<string,unknown>` 签名拒绝，VirtualMachineList/index.tsx:186-191 实锤）与 `create`（CreateVMRequest 双向不可赋值）原样覆盖（Pitfall 1）；`vdiServerApi` 纯 SPREAD（CreatePayload snake_case 排除集使 VDIServerConfig 可赋值，RESEARCH tsc 实测兑现）+ testConnection 保留；类型 re-export 块零改动。
- **download.test.ts**（+40 行，D-11 组 6）：新增 `executionApi.downloadReport 经 downloadFilePost 链路` 2 用例——显式 format 进 URL query + 文件名断言、缺省 format=pdf + 非 2xx 抛「下载失败」（T-94-05 超时归一路径）。

## Commits

| Task | Commit | Content |
| ---- | ------ | ------- |
| 1 | b207d65 | refactor(94-02): migrate opsApi to shared factory and download module (D-03/D-04) |
| 2 | 9138687 | refactor(94-02): merge second rpa factory into shared factory, normalize downloadReport (D-04/D-08) |
| 3 | b02c1c2 | refactor(94-02): wire vdiApi vm/vdiServer resources into shared factory (D-08) |

## Verification Results

- Task 1：`npx vitest run src/lib/opsApi.test.ts` 36/36 绿；type-check exit 0；lint 0 errors（1390 warnings = 存量基线）；`npx vitest run` 全量 553 文件 / 3794 测试绿
- Task 2：rpaApi.test.ts + rpaApi.batch56.unit.test.ts 零改动绿 + download.test.ts 新用例绿（3 文件 33 测试）；type-check exit 0；lint 0 errors（1389 warnings，较基线 −1：删除的裸 fetch 代码带走一条存量 warning）；全量 553 / 3796 绿
- Task 3：vdiApi.test.ts 零改动 10/10 绿；type-check exit 0（VirtualMachineList 消费点不炸，Pitfall 1 红线守住）；lint 0 errors（1389 warnings）；全量 553 / 3796 绿
- grep 总验收：`createCrudApi`/`CrudApiConfig` 在 opsApi.ts/rpaApi.ts 均 0 命中；opsApi.ts `axios.create` 0 命中；rpaApi.ts `fetch(` 0 命中；vdiApi.ts spread 接入 2 处；`downloadFilePost` 调用点 opsApi 2 + rpaApi 1
- KEEP 红线：`git diff -U0` 对 locationAliasApi/roomPhotoApi/workstationDeviceApi/geocode/componentApi 区块 0 内容行改动
- 零文件删除、零新文件（5 个既有文件修改，净 −163 行生产代码）

## Deviations from Plan

**1. [Rule 1 - Bug] excelApi.export 非 2xx 错误文案随 downloadFilePost 归一**
- **Found during:** Task 1
- **Issue:** 原内联链抛 `Error("导出失败")`，downloadFilePost 语义为 `` `下载失败: ${defaultFilename}` ``——委托后文案变化属等价替换的必然结果
- **Fix:** 无需处理——全仓测试对旧文案零断言（已核实 opsApi.test.ts 及消费方），plan 已预告按 downloadFilePost 语义最小修正断言，实际无需修正
- **Files modified:** 无

**2. [Rule 2 - Critical] assetApi.excel.export 获得内容处置头文件名提取（plan 预告的白得改善落地）**
- **Found during:** Task 1
- **Issue:** 原实现忽略 content-disposition、恒用固定名 `资产列表_<ts>.xlsx`
- **Fix:** 委托 downloadFilePost 后 content-disposition 优先、固定名降级为默认回退——plan action 4 明文预期（「归一后额外获得 content-disposition 提取能力，属 D-04 白得改善」）
- **Files modified:** xingran-react-frontend/src/lib/opsApi.ts

无其他偏离——plan 按原文执行（D-07 batch 覆盖层保持、D-14 仅 opsApi.test.ts 适配 + download.test.ts 计划内新增、scriptApi/vdiServerApi spread 编译器仲裁一次通过无需 override 升级）。

## TDD Gate Compliance

本 plan 三个任务均为 `type="auto"`（无 tdd="true"），且本质是对既有行为的等价替换迁移——验证以「既有测试零回归 + type-check/lint gate」为主引擎，新增测试仅 D-11 组 6（downloadReport 新链路锁定，落地于 download.test.ts，随 Task 2 提交）。plan frontmatter 为 `type: execute`，plan 级 RED/GREEN 双 commit 门不适用。

## Security & Threat Notes

- T-94-04（token 注入）：downloadReport 由裸 fetch + getAuthHeaders 手写头换 blobAxios 异步拦截器注入——同为 Bearer token 语义等价，download.test.ts 新用例锁定链路
- T-94-05（DoS/无超时悬挂）：裸 fetch 无超时 → 归一后继承 blobAxios 300000ms 超时，本 phase 唯一安全正向改善点兑现
- T-94-06（Tampering/URL 拼接）：全部等价替换，路径模板字符串原样迁移，GET/POST 动词与后端契约零改动；type-check + 3796 测试锁定
- 无威胁面外溢：所有改动均在 plan threat_model 登记范围内，无新增端点/信任边界

## Known Stubs

无——全部交付物为删除 + 等价替换 + 计划内测试增补，零 placeholder/TODO/未接线数据。

## Self-Check: PASSED

- 5/5 修改文件存在且已入库（git log 各 commit 可见）
- 3/3 task commit 存在：b207d65 / 9138687 / b02c1c2
- 工作树干净（仅存量 untracked 文件，与本 plan 无关）；STATE.md/ROADMAP.md 未触碰（沿用 94-01 惯例，orchestrator 集中写）
