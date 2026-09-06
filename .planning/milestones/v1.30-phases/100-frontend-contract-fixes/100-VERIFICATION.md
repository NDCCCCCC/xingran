---
phase: 100-frontend-contract-fixes
verified: 2026-09-07T02:21:18Z
status: human_needed
score: 6/6 must-haves verified
overrides_applied: 0
gaps: []
human_verification:
  - test: "VDI 批量操作端到端（真实环境）：在 VirtualMachineList 页面勾选多台虚拟机执行批量开机/关机/重启/快照操作"
    expected: "请求 POST /api/v1/vdi/vms/operate 不再 404；带 vdi:vm:edit 权限的账号操作成功，无权限账号被 403 拒绝"
    why_human: "需要运行中的后端 + 真实深信服 VDI 设备联动，无法在静态代码检查中验证外部服务集成"
  - test: "浏览器下载体验：在任一网络模块页面触发批量导出，分别验证正常导出与后端返回 200+JSON 错误体两种场景"
    expected: "正常导出保存的文件可打开；后端错误时不再落地伪 .xlsx 文件，页面提示错误体的 message 内容"
    why_human: "真实浏览器下载行为 + 真实后端错误注入，单测 mock 无法覆盖端到端文件落地体验"
  - test: "VM 详情页视觉走查：打开任一虚拟机详情页检查 Tab 结构"
    expected: "「账号管理」Tab 已消失，概览/操作记录/监控三个 Tab 渲染正常、无空白残留"
    why_human: "视觉外观与 Tab 布局需人工目检（render 测试仅覆盖挂载不报错）"
---

# Phase 100: 前端契约修复 — 验证报告

**Phase Goal:** 前端 API 契约与后端真实路由对齐——幽灵方法处置、rpaApi 全族契约对齐专项、networkApi 下载链收敛到权威 download.ts。
**Verified:** 2026-09-07T02:21:18Z
**Status:** human_needed（6/6 项 SC 机器验证全部通过；3 项真实环境/浏览器行为需人工确认）
**Re-verification:** No — initial verification

## Goal Achievement

### Success Criteria 逐项裁定

| # | Success Criteria | Status | 证据（file:line 实测） |
|---|------------------|--------|------------------------|
| 1 | V130R-10 幽灵方法处置：rpaApi 8 spread 站点 + vdiApi 2 站点清除（pick 化显式方法集）+ invariants 守卫 | ✓ VERIFIED | `rpaApi.ts`（683→158 行）3 个工厂站点全部显式 pick（:44-57 taskApi、:66-84 workerApi、:93-111 executionApi），零 spread、零 batch/statistics/searchOptions；`vdiApi.ts`（:43-45 vmApi、:134-138 vdiServerApi）同型 pick 化；`apiFactory.invariants.test.ts` HARD_ALLOWED rpaApi=0/vdiApi=0（:65-69）+ POST_CLEANUP_BASELINE 6 对象 keys 等值锁（:300-330） |
| 2 | V130R-11 rpaApi 对账：RECONCILIATION.md 全族 116 方法表；17 存活端与后端路由对照；死方法零残留；守卫防回增 | ✓ VERIFIED | `RECONCILIATION.md` 十族六列子表合计 116 方法（15+10+15+14+14+12+13+6+12+5）；17 个存活方法逐一对照 `rpa_router.go` 实际注册（task :48-53 / worker :64,:66 / execution :84-89 / ai :101-108，全部命中，超出抽查要求）；死方法 grep 零残留（仅守卫注释与 not.toContain 断言命中）；守卫 = invariants keys 基线 + `rpaApi.test.ts` 契约测试双向锁 |
| 3 | V130R-11 附加（D-100-10/11）：`POST /vdi/vms/operate` 注册 + 401 测试；accounts 4 方法 + Tab + 类型全链删除 | ✓ VERIFIED | `vm_router.go:32` `r.POST("/operate", middleware.RequirePermissions([]string{"vdi:vm:edit"}, core), vmHandler.Operate)`；链路 `vm_handler.go:204` → `vm_service_impl.go:764` OperateVM；swagger 笔误已修（:203 `@Router /vdi/vms/operate`）；`vm_router_test.go` 两用例实跑 PASS（路由注册断言 + 未认证 401 断言）；accounts 族 grep 全 src/ 仅命中守卫断言（`vdiApi.test.ts:126-129` not.toContain）与注释，`VirtualMachineDetail/index.tsx` 零 accounts 引用（Tab 现为概览/操作记录/监控） |
| 4 | V130R-12：networkApi 下载链收敛 + downloadFile/Post JSON 错误体检测 + invariants 递归化 15 文件 + CLAUDE.md 同步 | ✓ VERIFIED | `networkApi.ts` 零 axios import/私有 blobAxios/triggerBrowserDownload/createObjectURL/size<1024（唯一命中为 JSDoc 说明旧嗅探已被替换），改 import `downloadFile/downloadFilePost`（:2）；exportMACHistory 薄壳（:111-124）+ batchExport 单行薄壳（:137-143），9 个页面消费者零改动；`download.ts:74-93` `throwIfJsonErrorResponse`（content-type application/json → 解析 message → throw）挂载于 downloadFile :103 与 downloadFilePost :124，双函数返回 Promise<string>；invariants `readdirSync(..., { recursive: true })` + sep 归一（:129-134）、EXPECTED_FILES 15 含 `api/networkApi.ts`/`api/macHeatmapApi.ts`（:72-88）、whitelist 显式登记 2 键（:110-111）；`CLAUDE.md:456` 已同步「all 15 ... (recursive, includes src/lib/api/*)」+ 硬档基线 0 + POST_CLEANUP_BASELINE 说明 |
| 5 | 回归纪律：3 项修复各有回归测试 | ✓ VERIFIED | `rpaApi.test.ts` 17 方法 URL+动词断言 + 4 对象 keys 断言（:27-148）；`vm_router_test.go` 2 用例实跑 PASS；`download.test.ts`「200+JSON 错误体检测 (D-100-7)」4 用例（:218-268：POST throw / POST 非法 JSON 回退 / POST blob 防误伤 / GET 同构防护）；invariants keys 基线 6 对象——全部包含在实跑的 553 文件 / 3792 用例 0 失败中 |
| 6 | Gate（实跑）：lint 0 errors warnings≤1389 / type-check / vitest 全量 0 失败 / coverage 45/45 dirs | ✓ VERIFIED | lint：**0 errors / 1378 warnings**（≤1389）exit 0；type-check：tsc 零输出 exit 0；vitest：**553 files / 3792 tests 全部通过，0 失败**（与 100-01 预测的删 batch56 后 ~553 基线吻合）；coverage gate：`check-frontend-coverage.sh` **PASS: per-dir floor gate — 45/45 directories >= floor**（GLOBAL 加权 60.07%，lib 96.76% ≥ 87.2 下限——100-01 重点盯防项无回退） |

**Score:** 6/6 truths verified（机器验证口径）

### Required Artifacts

| Artifact | Expected | Status | Details |
| -------- | -------- | ------ | ------- |
| `xingran-react-frontend/src/lib/rpaApi.ts` | 17 存活方法 4 对象 pick 化 | ✓ VERIFIED | 158 行，4 导出对象恰 17 方法，聚合 rpaApi 4 键（:153-158） |
| `xingran-react-frontend/src/lib/vdiApi.ts` | vmApi 16 / vdiServerApi 6，accounts 全删 | ✓ VERIFIED | 162 行，vmApi 恰 16 方法含 operate/batchOperate，vdiServerApi 恰 6，类型导出块无 VMAccount |
| `xingran-react-frontend/src/lib/download.ts` | JSON 检测 helper + 双函数挂载 | ✓ VERIFIED | :74-93 helper，:103/:124 挂载，双函数 Promise<string> |
| `xingran-react-frontend/src/lib/api/networkApi.ts` | 零私有下载实现，薄壳化 | ✓ VERIFIED | 仅 3 个 import（post/download/类型），无 axios |
| `xingran-react-frontend/src/lib/apiFactory.invariants.test.ts` | keys 基线 + 递归 15 文件 | ✓ VERIFIED | POST_CLEANUP_BASELINE 6 对象 + EXPECTED_FILES 15 + HARD_ALLOWED vdiApi=0 |
| `xingran-react-frontend/src/lib/rpaApi.test.ts` | 17 方法契约基线 | ✓ VERIFIED | 151 行，四族 describe + keys 断言 |
| `xingran-react-frontend/src/lib/download.test.ts` | 摘 executionApi + 4 JSON 用例 | ✓ VERIFIED | 280 行，D-100-7 describe 4 用例 |
| `internal/api/v1/vdi/vm_router.go` | operate 带权限注册 | ✓ VERIFIED | :32 RequirePermissions vdi:vm:edit |
| `internal/api/v1/vdi/vm_router_test.go` | 新建：注册断言 + 401 断言 | ✓ VERIFIED | 2 用例，`go test -count=1` 实跑 PASS |
| `.planning/phases/100-frontend-contract-fixes/RECONCILIATION.md` | 116 方法六列对账 + vdi 段 + observed-debt | ✓ VERIFIED | rpa 十族 116 行 + vdi vmApi/vdiServerApi 段 + observed-debt 段（三页面内联 post 债登记不修） |
| `xingran-react-frontend/src/lib/__tests__/rpaApi.batch56.unit.test.ts` | 整文件删除 | ✓ VERIFIED | 文件不存在 |
| `CLAUDE.md` | 15 recursive + 硬档 0 同步 | ✓ VERIFIED | :456 |
| 已删死测试/空转 mock | rpa 页面 vi.mock 摘除 | ✓ VERIFIED | `vi.mock("@/lib/rpaApi")` 全 pages/ 零命中 |

### Key Link Verification

| From | To | Via | Status | Details |
| ---- | -- | --- | ------ | ------- |
| taskApi/executionApi 工厂方法 | 后端路由 | createResourceApi pick | ✓ WIRED | pick 引用的工厂方法 URL 与 rpa_router.go :48-53/:84-89 一致 |
| workerApi.statistics（无参收窄） | POST /rpa/workers/statistics | post 直调 | ✓ WIRED | rpa_router.go:66 注册；id 分支已删 |
| vmApi.batchOperate | POST /vdi/vms/operate | post 直调 | ✓ WIRED | vm_router.go:32 本相补注册；VirtualMachineList 活跃调用面恢复 |
| invariants keys 基线 | 4+2 导出对象运行时 keys | Object.keys 等值断言 | ✓ WIRED | :342-346 循环断言，rpaApi.test.ts/vdiApi.test.ts 互指 |
| batchExport | downloadFilePost 薄壳 | download.ts | ✓ WIRED | networkApi.ts:142 单行委托，9 页面消费者契约不变 |
| exportMACHistory | downloadFile 薄壳 | download.ts | ✓ WIRED | networkApi.ts:120；MACHistoryPage.tsx:374 await 全托管 |
| invariants 递归枚举 | api/networkApi.ts 进扫描面 | readdirSync recursive | ✓ WIRED | EXPECTED_FILES 含 api/ 前缀 2 文件，sep 归一（Windows 实跑绿） |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
| -------- | ------- | ------ | ------ |
| operate 路由注册 + 401 | `go test -count=1 ./internal/api/v1/vdi/... -run TestSetupVMRouter -v` | 2 用例 PASS | ✓ PASS |
| vdi 包回归 | `go test ./internal/api/v1/vdi/...` | ok | ✓ PASS |
| 全量前端测试 | `npx vitest run` | 553 files / 3792 tests / 0 failed, exit 0 | ✓ PASS |
| lint gate | `npm run lint` | 0 errors / 1378 warnings, exit 0 | ✓ PASS |
| type-check gate | `npm run type-check` | tsc 零输出, exit 0 | ✓ PASS |
| coverage ratchet | `bash .github/scripts/check-frontend-coverage.sh xingran-react-frontend/coverage/coverage-final.json .coverage-fe-floors` | PASS 45/45 dirs, exit 0 | ✓ PASS |

### Probe Execution

无 phase 声明 probe 脚本（`scripts/*/tests/probe-*.sh` 不适用本项目）；Gate 实跑（上表）即本相 probe 等价物，全部实跑非转述。

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
| ----------- | ----------- | ----------- | ------ | -------- |
| V130R-10 | 100-01 / 100-02 | rpaApi 8 处 + vdiApi 2 处 spread 幽灵方法处置 + 回归测试 | ✓ SATISFIED | pick 化端态 + keys 基线守卫 + rpaApi/vdiApi.test.ts（比 omit 方案更强的双向锁） |
| V130R-11 | 100-01 / 100-02 | rpaApi 全族契约对齐 + 对账清单落盘 + 守卫 | ✓ SATISFIED | RECONCILIATION.md 116 方法表 + 17 存活端 + invariants/rpaApi.test.ts 双守卫 |
| V130R-12 | 100-03 | networkApi 收敛 + downloadFilePost JSON 检测 + invariants 递归 + 回归测试 | ✓ SATISFIED | 薄壳化 + D-100-7 helper + 4 用例 + 递归 15 文件 |

无 ORPHANED requirement（REQUIREMENTS.md 中 Phase 100 映射仅 V130R-10..12，三 plan 全覆盖）。

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
| ---- | ---- | ------- | -------- | ------ |
| `src/pages/network/mac/history/MACHistoryPage.tsx` | :16, :724 | 预存 TODO（14-04/14-05 后续 plan 引用） | ℹ️ Info | git 追溯确认 2026-08（ea528c6）引入，非本相新增，且带正式工作项引用，不触发 debt-marker gate |
| RECONCILIATION.md（vdi 段头） | :182 | 「34 方法 = 22 alive + 6 ghost + 6 mismatch」算术双计 operate/batchOperate（同时计入 alive 与 mismatch），实际不同方法 32（vmApi 23 行 + vdiServer 9 行） | ℹ️ Info | 行级对账表完整无遗漏、不影响任何裁决；仅头部汇总数字口径瑕疵，建议下次触碰该文件时修正 |

本相新增/修改文件零 TBD/FIXME/XXX/HACK 命中；零 placeholder 返回值；零空实现。

### Human Verification Required

### 1. VDI 批量操作端到端（真实环境）

**Test:** VirtualMachineList 页面勾选多台虚拟机 → 批量开机/关机/重启操作
**Expected:** POST /api/v1/vdi/vms/operate 不再 404；vdi:vm:edit 权限账号操作成功、无权限账号 403
**Why human:** 需要运行中的后端 + 真实深信服 VDI 设备（外部服务集成，路由注册/权限链/前端接线已由测试锁定）

### 2. 浏览器下载体验（JSON 错误体防欺骗）

**Test:** 网络模块页面触发批量导出——正常导出 + 后端 200+JSON 错误体两种场景
**Expected:** 正常文件可打开；错误场景不再落地伪 .xlsx 且提示 message
**Why human:** 真实浏览器文件落地行为，单测 mock 无法覆盖端到端体验

### 3. VM 详情页 Tab 视觉走查

**Test:** 打开虚拟机详情页检查 Tab 结构
**Expected:** 「账号管理」Tab 消失，概览/操作记录/监控三 Tab 渲染正常
**Why human:** 视觉外观需人工目检

### Gaps Summary

无 gaps。6 项 SC 全部机器验证通过：三处代码级端态（rpaApi 17 方法 / vdiApi 16+6 方法 / 下载链单权威）与四道守卫（keys 基线 / 契约测试 / HARD_ALLOWED 双向锁 / 递归 invariants）实测在位；四道 gate（lint / type-check / vitest 全量 / coverage 45 dirs ratchet）由本验证独立实跑全绿，非转述 SUMMARY。RECONCILIATION.md 台账与后端路由真相源三方一致（台账 ↔ rpa_router.go ↔ rpaApi.ts）。

附注（非 gap）：
1. deferred-items.md 记录的 `internal/api/v1/network TestBackupHandler_Restore` 失败（归因 Phase 97 在途未提交改动）现已实测 PASS（`go test -count=1` ok）——该 deferred 项已自行消解，与 Phase 100 无涉。
2. ROADMAP 进度表 Phase 100 行仍为 "Not started"——工作实际已完成并入库（9 个 phase 100 commits，423484a..4341d35），进度表元数据待 orchestrator 回写。

---

_Verified: 2026-09-07T02:21:18Z_
_Verifier: Claude (gsd-verifier)_
