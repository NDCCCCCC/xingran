---
phase: 94-api-p2
reviewed: 2026-09-05T19:37:29Z
depth: standard
files_reviewed: 16
files_reviewed_list:
  - xingran-react-frontend/src/lib/adDomainApi.test.ts
  - xingran-react-frontend/src/lib/adDomainApi.ts
  - xingran-react-frontend/src/lib/apiFactory.invariants.test.ts
  - xingran-react-frontend/src/lib/apiFactory.test.ts
  - xingran-react-frontend/src/lib/apiFactory.ts
  - xingran-react-frontend/src/lib/download.test.ts
  - xingran-react-frontend/src/lib/download.ts
  - xingran-react-frontend/src/lib/dutyApi.ts
  - xingran-react-frontend/src/lib/knowledgeApi.ts
  - xingran-react-frontend/src/lib/noticeApi.ts
  - xingran-react-frontend/src/lib/opsApi.test.ts
  - xingran-react-frontend/src/lib/opsApi.ts
  - xingran-react-frontend/src/lib/rpaApi.ts
  - xingran-react-frontend/src/lib/vdiApi.ts
  - xingran-react-frontend/src/lib/workorderApi.ts
  - xingran-react-frontend/src/types/apiFactory.ts
findings:
  critical: 0
  warning: 2
  info: 3
  total: 5
status: issues_found
---

# Phase 94: Code Review Report

**Reviewed:** 2026-09-05T19:37:29Z
**Depth:** standard
**Files Reviewed:** 16
**Status:** issues_found

## Summary

对本 phase 全部 16 个文件做了 standard 深度审查，并以可执行验证对「等价替换重构」硬承诺做了实测：

**等价性验证（全部通过）：**
- `npm run type-check`（tsc --noEmit）**零错误** —— spread 替换虽改变了部分方法的 TS 泛型（见 IN-01），但全部 100+ 消费文件在零改动前提下编译通过，「导出签名零变化」承诺在编译层面成立。
- vitest 实测 **11 个测试文件 147 个用例全绿**：本 phase 新增 5 个（apiFactory / apiFactory.invariants / download / adDomainApi / opsApi，共 75 用例）+ 委托文件既有契约 6 个（duty/knowledge/notice/workorder/rpa/vdi，共 72 用例，含锁定单参 delete 契约的测试）。
- 逐函数 diff 审计（cc77e81..HEAD）：所有委托点 list/get/create/update 的 URL + wire body 与旧实现逐一相同；D-05 KEEP 的单参 `post(url)` delete 全部未被触碰；adDomain 的 GET 动词端点、`withDefaultPagination` 前置注入、`getADUserIds` 不注入默认分页等行为均精确保留。
- D-12 基线人工复核与测试实跑双确认：opsApi 4 / rpaApi 0 / vdiApi 1（硬档）；adDomain 3 / knowledge 5 / duty 5 / workorder 6 / notice 0（warning 档），与 `allowedResidues` 等值锁完全一致。
- :501 URL bug 修复确认：旧实现 `post(\`/ad-domain/mappings/${id}/delete}\`, {})` 必然 404，现修复为 `${id}/delete`，测试同步更新锁定，登记完备。
- downloadReport 迁移审计：旧 `getAuthHeaders()` 仅返回 `Authorization` 头（authHelpers.ts:14-18），blobAxios 拦截器无头丢失；新增 `{}` body 与 5min 超时为净改善。
- ESLint 对 16 个文件 0 error（仅 opsApi.ts:415 `_maxAge` 1 个 warning，为 diff 之外的前置存量）。

未发现 Critical 级问题。2 个 Warning（1 个未登记的行为偏差、1 个防线扫描盲区）与 3 个 Info 建议后续跟进，均不阻断本 phase 收敛。

## Warnings

### WR-01: assetApi.excel.export 文件名来源行为变化未在锁定决策清单登记

**File:** `xingran-react-frontend/src/lib/opsApi.ts:557-561`（链路 `xingran-react-frontend/src/lib/download.ts:79-92`）
**Issue:** 旧实现对 `/ops/asset/export` 忽略响应头、**始终**用硬编码 `资产列表_${Date.now()}.xlsx` 触发下载；改为 `downloadFilePost` 后会优先提取 `content-disposition` 文件名。后端该端点经 `SetupExcelRouter` 注册且**确实发送**该头（`internal/api/v1/operations/excel_handler.go:75`，`generateExcelFilename` 生成 `<entityType>_<suffix>_<ts>.xlsx`），因此用户实际保存的文件名由「资产列表_*」变为后端英文名（asset_* 等）。同时该处错误文案也由「导出失败」变为「下载失败: …」。锁定决策仅登记了「:501 URL 修复」与「excelApi.export 错误文案归一」两处，`asset excel export` 的文件名来源偏差不在登记清单内——「唯一登记的行为变更」的表述与实际 diff 不完全相符。影响为外观级（保存文件名/报错文案），且 `opsApi.test.ts:548-559` 与 `download.test.ts` 已锁定新行为，无断言依赖旧文案。
**Fix:** 将该偏差显式补登到 phase/deferred 文档，使「等价替换」承诺的边界与 diff 一致；若「资产列表」中文命名是产品要求，可为 `downloadFilePost` 增加忽略 content-disposition 的选项（或后端将 asset 导出文件名改为中文），二选一后更新对应测试注释。

### WR-02: D-12 AST 扫描盲区——变量声明器形态的箭头函数不进扫描口径

**File:** `xingran-react-frontend/src/lib/apiFactory.invariants.test.ts:189-215`（`scanFile` 的 `visit`）
**Issue:** 访问器仅匹配 `FunctionDeclaration` / `PropertyAssignment` / `MethodDeclaration` 三种节点。`export const x = (id: string) => post(\`/x/${id}/list\`)` 这类挂在 `VariableDeclaration` 上的箭头函数会同时逃过硬档与 warning 档两道计数——而这恰是最常见的现代写法。已核实当前 13 个 *Api.ts 无此形态（现计数准确），防线暂无实际漏报，但 D-12「防白名单腐烂」tripwire 在该形态上存在结构性漏洞，新增此形态的模板残留不会转红。
**Fix:**
```typescript
} else if (ts.isVariableDeclaration(node)) {
  const name = node.name;
  const init = node.initializer;
  if (ts.isIdentifier(name) && init && (ts.isArrowFunction(init) || ts.isFunctionExpression(init))) {
    owner = name.text;
    body = init.body;
  }
}
```
加入 `visit` 分支后重跑红绿演练确认计数不漂移。

## Info

### IN-01: spread 替换静默放宽了若干导出方法的泛型（返回 BaseResponse<unknown>）

**File:** `xingran-react-frontend/src/lib/rpaApi.ts:137-159`、`xingran-react-frontend/src/lib/vdiApi.ts:33-150`
**Issue:** 旧手写/私有工厂方法带显式泛型，spread 接入 `createResourceApi` 后变为工厂的无泛型默认：
- rpa `scriptApi.create/update`：`BaseResponse<Script>` → `BaseResponse<unknown>`；`scriptApi.list` 参数由 `PageParams` 收窄为 `PageParams & Record<string, unknown>`；
- vdi `vdiServerApi.create/update`：`BaseResponse<VDIServer|void>` → `BaseResponse<unknown>`；`vmApi.delete`：`BaseResponse<void>` → `BaseResponse<unknown>`；`vmApi.update` 参数由 `UpdateVMRequest`（`{name?, ip_address?, mac_address?}`）放宽为其超集 `Partial<CreatePayload<VirtualMachine>>`。

type-check 实测零错误 → 消费方零改动成立、wire 不变，属可接受代价；但新消费方在 `res.data` 上丢失类型推断，且参数收窄（scriptApi.list）对「interface 变量直传」是潜在 TS2345 陷坑（vmApi.list 正因此保留 OVERRIDE）。代码注释已按 D-08 登记，不阻断。
**Fix:** 后续 phase 可为工厂增加请求/响应泛型位（如 `createResourceApi<T, CReq, UReq>`）或对受影响实例做类型化 OVERRIDE，恢复推断而不动 wire。

### IN-02: 委托边界存在 9 处 `as unknown as` 双重断言

**File:** `xingran-react-frontend/src/lib/adDomainApi.ts:241,499,608`、`dutyApi.ts:163`、`knowledgeApi.ts:141,200-201,212-213,222-223`、`noticeApi.ts:32`、`workorderApi.ts:392,505,550`
**Issue:** 根因是工厂 `list` 参数类型 `PageParams & Record<string, unknown>` 拒收 interface 类型变量（TS 隐式索引签名规则），各包装函数被迫双断言透传。外层导出签名仍对调用方做强类型校验，风险被限制在委托边界内，但断言抹掉了该边界的编译期检查（如字段名拼错在此不报错）。
**Fix:** 后续可将工厂 `list` 参数放宽为 `PageParams & object`（等价去除索引签名要求），批量消除断言；属工厂契约演进，不建议在本等价替换 phase 内顺手做。

### IN-03: rpaApi.downloadReport 错误文案变更未列入登记清单

**File:** `xingran-react-frontend/src/lib/rpaApi.ts:300-306`
**Issue:** 旧裸 fetch 链抛「下载报告失败」，迁移后归一为「下载失败: execution_report_<id>.<format>」。与已登记的 excelApi.export 文案归一同类，但锁定决策清单未单独点名（`download.test.ts:241` 已锁定新文案，旧文案零断言依赖，已核实）。纯记录性问题。
**Fix:** 与 WR-01 一并补登即可，无需代码改动。

## 验证记录（判定依据）

| 项 | 命令 | 结果 |
|---|---|---|
| 类型门 | `npm run type-check` | 0 error |
| 单测（本 phase 5 文件） | `npx vitest run src/lib/{apiFactory,apiFactory.invariants,download,adDomainApi,opsApi}.test.ts` | 39+36 用例全绿 |
| 单测（委托文件契约 6 文件） | `npx vitest run src/lib/{duty,knowledge,notice,workorder,rpa,vdi}Api.test.ts` | 72 用例全绿 |
| Lint | `npx eslint`（16 文件） | 0 error / 1 前置存量 warning |
| D-12 基线 | invariants 测试实跑 + 人工逐函数复核 | 双向等值，无漂移 |
| wire 等价 | `git diff cc77e81..HEAD` 逐委托点审计 | list/get/create/update 均同 URL 同 body；单参 delete 全部未动 |

---

_Reviewed: 2026-09-05T19:37:29Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
