# Phase 100: 前端契约修复 - Context

**Gathered:** 2026-09-06
**Status:** Ready for planning
**Mode:** Auto-generated（autonomous 模式 smart-discuss——phase 无 D-03 设计决策项，grey area 按用户长程标准自动裁决）

## Phase Boundary

前端 API 契约与后端真实路由对齐三专项：幽灵方法处置（V130R-10）、rpaApi 全族契约对账（V130R-11）、networkApi 下载链收敛（V130R-12）。全部为 `src/lib` 层契约修复，不涉及页面/组件重写；消费者调用点随对账结果同步处理。

## Implementation Decisions

### V130R-10 幽灵方法处置

- **D-100-1:** 用户标准「不考虑向后兼容」→ rpaApi 8 处 / vdiApi 2 处工厂 spread 幽灵方法**直接删除**（不保留空壳）。删除前逐一确认零调用方（grep 调用面）；有调用方的幽灵方法升格为 V130R-11 对账项处理。
- **D-100-2:** 守卫升级——`apiFactory.invariants.test.ts` 对 rpaApi/vdiApi 锁定清理后方法集（KEEP baseline 更新为「后端路由存在」的实测集合），防回增。

### V130R-11 rpaApi 全族契约对账

- **D-100-3:** 对账方法：scriptApi/scheduleApi/variableApi/templateApi/notificationApi/statisticsApi 全族逐一列清单——方法名/HTTP 动词/路径 → 对照后端 `internal/api/v1/` rpa 路由注册表 → 三分类：**alive**（路由存在，保留）/ **dead**（路由不存在且零调用方，删除）/ **mismatch**（路由不存在但有调用方——逐个裁决：调用方同删，或确有产品语义则后端补路由；默认删，补路由需在 RESEARCH 中给出明确证据）。
- **D-100-4:** 对账清单落盘为 phase 工件 `RECONCILIATION.md`（方法/动词/路径/后端路由/裁决/理由 六列），作为 SUMMARY 附件归档。
- **D-100-5:** 守卫：invariants 测试 rpaApi 档锁定清理后基线（warning tier 计数 = 清理后实测值），死方法回增即红。

### V130R-12 下载链收敛

- **D-100-6:** `src/lib/api/networkApi.ts` 内嵌 axios 下载链全部收敛到权威 `src/lib/download.ts`（blobAxios + downloadFile/downloadFilePost），禁止平行实现。
- **D-100-7:** `downloadFilePost` 补 JSON 错误体检测：响应 `content-type` 含 `application/json` 时解析错误体并 throw（不再把 200+JSON 错误响应存成 .xlsx）；附回归测试（mock blob/json 两种响应）。
- **D-100-8:** `apiFactory.invariants.test.ts` 的 `readdirSync` 改递归扫描（`readdirSync` with `recursive: true` 或手工栈），覆盖子目录中的 `*Api.ts`。

### 全局（用户长程标准适用）

- 死方法/死链一律删除，不保留兼容导出
- 前端三 gate 硬约束：`npm run lint` 0 errors（1389 warnings 存量不倒退）、`npm run type-check` 通过、`npm run test` 0 失败（554 文件 / 3800 tests 基线）；覆盖率 45/45 dirs gate 不倒退
- 后端若需补路由（D-100-3 mismatch 裁决为补），走后端既有 Handler-Service 模式 + operlog 写操作约定

## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### API 工厂与守卫
- `xingran-react-frontend/src/lib/apiFactory.ts` — createResourceApi 单一权威
- `xingran-react-frontend/src/lib/apiFactory.invariants.test.ts` — D-12 双档 AST 扫描防线（hard/warning tier + KEEP baseline）
- `xingran-react-frontend/src/types/apiFactory.ts` — 工厂类型

### 问题文件
- `xingran-react-frontend/src/lib/rpaApi.ts` — V130R-10/11 主战场
- `xingran-react-frontend/src/lib/vdiApi.ts` — 2 处幽灵方法
- `xingran-react-frontend/src/lib/api/networkApi.ts` — V130R-12 平行下载链
- `xingran-react-frontend/src/lib/download.ts` — 下载链权威

### 后端路由真相源
- `internal/api/v1/rpa/`（handler/router 注册）— rpaApi 对账基准
- `internal/api/v1/vdi/` — vdiApi 对账基准
- `internal/api/router.go` — 全量路由装配

### CLAUDE.md 约定
- Frontend API Factory Convention（KEEP 例外清单——本 phase 清理后需同步更新该清单）

## Existing Code Insights

- Phase 94 已建立 apiFactory 单一权威 + invariants 双档扫描；KEEP 例外：notificationConfigApi / assetApi / menuApi / profileApi / columnConfigApi
- `download.ts` 已有 blobAxios（5-min timeout）+ downloadFile / downloadFilePost
- 前端测试基线：554 文件 / 3800 tests（v1.29 收口实测）

## Specific Ideas

- rpaApi 对账是本 milestone 最大单项对齐工程——RESEARCH 必须产出完整方法清单再动手，避免边查边改
- mismatch 裁决宁删勿补：后端补路由会扩大攻击面与维护面，除非调用方是活跃产品功能

## Deferred Ideas

None — 三项全部在 scope 内。

---

*Phase: 100-frontend-contract-fixes*
*Context gathered: 2026-09-06 via autonomous smart-discuss*
