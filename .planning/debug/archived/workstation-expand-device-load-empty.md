---
slug: workstation-expand-device-load-empty
status: resolved
deferred_to: v1.16-tech-debt
trigger: "工位管理页面点击展开子列表展示设备提示加载设备失败，控制台及后端日志没有报错"
created: 2026-06-13
updated: 2026-06-25
---

# Debug Session: workstation-expand-device-load-empty

## Current Focus

- **root cause (FINAL, 2 layers)**:
  1. **Layer 1** (frontend, commit `8ad0f4a`): `WorkstationDeviceTable/index.tsx` 调用的 `workstationDeviceApi.getManual/getAD/getAsset` 不存在 → `undefined(workstationId)` 抛 TypeError 被外层 catch 静默吞掉 → 用户看到的"无声失败"
  2. **Layer 2** (backend, commit `16372f5`): router 把 `GET /:workstationId` 改成 `POST /:id`，**handler 里的 `c.Param("workstationId")` 没有同步改成 `c.Param("id")`** → 后端永远拿到空字符串 → 400 "参数缺失: 工位ID"。Layer 1 修复后 Layer 2 才暴露。
- **fix**: 
  - 前端：`fetchAllDevices` 改回单次 `getByWorkstation(workstationId)`，按 `deviceSource` 字段分发；line 181 `setPrimaryAndSave` 替换为 `syncAD/syncAsset + setPrimary` 组合；catch 块加 `console.error`
  - 后端：handler `c.Param("workstationId")` → `c.Param("id")`；同步 Swagger 注释 `workstationId` → `id`，`[get]` → `[post]`
- **verification**: `go build ./...` 通过；浏览器 console 现在能捕获真实错误（修复了"无声失败"问题）

## Symptoms

- **Expected**: 点击工位行展开子列表，展示该工位下所有关联设备列表
- **Actual**: 子列表空白，控制台静默提示"加载设备失败"
- **Error messages**: 用户报告"提示加载设备失败"，DevTools console 与后端 logrus 均**完全干净**
- **Reproduction**: 进入工位管理页面 → 点击工位行展开子表 → 立即弹出 `message.error('加载设备失败')`，无网络请求
- **Scope**: 仅"展开设备子表"场景

## Evidence

- timestamp: 2026-06-13
  checked: `git show 2c407c4` (用户归因的提交)
  found: 该提交仅删除自定义 `expandIcon` Button，与设备加载逻辑无关
  implication: 用户的归因不准确；真正的破坏者在更早的提交
- timestamp: 2026-06-13
  checked: `git show 8ad0f4a` (引入三段查询的提交)
  found: 此提交把 `fetchDevices` 从 `workstationDeviceApi.getByWorkstation(workstationId)` 改为 `Promise.allSettled([getManual, getAD, getAsset])`，但这三个方法在 `opsApi.ts` 中**从未定义**
  implication: **Layer 1 root cause**
- timestamp: 2026-06-13
  checked: `lib/opsApi.ts:594-629` `workstationDeviceApi` 当前定义
  found: 仅有 `getByWorkstation / addManual / syncAD / syncAsset / update / delete / setPrimary` —— **无** `getManual`/`getAD`/`getAsset`/`setPrimaryAndSave`
  implication: `workstationDeviceApi.getManual` 求值为 `undefined`，`undefined(workstationId)` 抛 TypeError
- timestamp: 2026-06-13
  checked: `WorkstationDeviceTable/index.tsx:46-80` `fetchAllDevices`
  found: 外层 `try`/`catch (error) { message.error('加载设备失败'); }` —— catch 块**不调用** `console.error`，不记录 `error.message`
  implication: TypeError 被静默吞掉，UI 只见静态文案
- timestamp: 2026-06-13 (after Layer 1 fix)
  checked: 浏览器 console 错误
  found: `POST /api/v1/ops/workstation-device/8fe705e1-ba49-4b7a-a8f1-df10ed63aae0 400 (Bad Request)` + `Error: 参数缺失: 工位ID`
  implication: 请求成功发出但后端拒绝 — **存在 Layer 2 bug**
- timestamp: 2026-06-13
  checked: `internal/api/router.go:595`
  found: `workstationDevices.POST(":id", workstationDeviceHandler.GetByWorkstation)` —— **路径参数名是 `id`**
  implication: handler 用的参数名与路由不一致
- timestamp: 2026-06-13
  checked: `workstation_device_handler.go:32` `c.Param("workstationId")`
  found: 期望参数名 `workstationId`，但路由注册为 `id` → 永远拿空字符串 → 触发 `apperrors.ParamMissing("工位ID")` → 400
  implication: **Layer 2 root cause**
- timestamp: 2026-06-13
  checked: `git log -S 'workstationDevices.GET' -- internal/api/router.go`
  found: 提交 `16372f5`（feat(28-06): add GetByDeviceSN and SearchBySerial）把 `workstationDevices.GET("/:workstationId", ...)` 改成 `workstationDevices.POST("/:id", ...)`，**但忘了同步 handler 里的参数名**
  implication: Layer 2 是 `16372f5` 引入的不完整修改

## Eliminated

- hypothesis: "事件冒泡导致 fetch 没被调用"
  evidence: 展开机制本身工作正常（`expandedRowRender` 触发，`fetchAllDevices` 被调用），失败发生在 fetch 内部
  timestamp: 2026-06-13
- hypothesis: "commit 2c407c4 的改动引入了 bug"
  evidence: 该提交只删了 17 行自定义 expandIcon 按钮，与设备加载 API 调用完全无关
  timestamp: 2026-06-13
- hypothesis: "axios 拦截器吞掉了请求"
  evidence: 错误来自调用 `undefined`，请求根本未构造
  timestamp: 2026-06-13
- hypothesis: "请求被路由/中间件过滤"
  evidence: 后端 logrus 无记录 —— 因为根本没有请求发出
  timestamp: 2026-06-13

## Resolution

- **root_cause (2 layers, 加上完整 260610 refactor 补齐)**:
  1. **Layer 1** (frontend, commit `8ad0f4a`): `WorkstationDeviceTable` 调用 `workstationDeviceApi` 上不存在的 `getManual`/`getAD`/`getAsset`/`setPrimaryAndSave`，TypeError 被外层 catch 静默吞掉 → 用户看到的"无声失败"
  2. **Layer 2** (backend, commit `16372f5`): router 把路径参数从 `workstationId` 改为 `id`，但 handler 的 `c.Param("workstationId")` 没同步更新 → 400
  3. **260610 refactor 半成品** (`.planning/quick/260610-device-source-refactor/`): 服务层 `GetADDevices/GetAssetDevices/SetPrimaryAndSave` 已实现（commit `dcf1fb8`），但 handler、router、frontend opsApi 从未跟进。**设计意图**："AD 设备和资产设备实时查询并显示在工位子表格里"，由用户最终确认。

- **fix (4 files — 完整 260610 refactor + Layer 1 + Layer 2)**:
  - `internal/api/v1/operations/workstation_device_handler.go`:
    - 新增 `GetADDevices` handler（POST `/:id/ad`）
    - 新增 `GetAssetDevices` handler（POST `/:id/asset`）
    - 新增 `SetPrimaryAndSave` handler（POST `/:id/set-primary-and-save`）
    - 修复 Layer 2：`c.Param("workstationId")` → `c.Param("id")`
    - 同步 Swagger 注释
  - `internal/api/router.go`:
    - 注册 3 个新路由：`/:id/ad`、`/:id/asset`、`/:id/set-primary-and-save`
  - `xingran-react-frontend/src/lib/opsApi.ts`:
    - 新增 `getAD(workstationId)` → `POST /ops/workstation-device/{id}/ad`
    - 新增 `getAsset(workstationId)` → `POST /ops/workstation-device/{id}/asset`
    - 新增 `setPrimaryAndSave(deviceId, data)` → `POST /ops/workstation-device/{id}/set-primary-and-save`
    - 把 `getByWorkstation` 改名为更准确的 `getManual`，保留 `getByWorkstation` 为兼容别名
  - `xingran-react-frontend/src/components/operations/WorkstationDeviceTable/index.tsx`:
    - 恢复三段并行查询：`Promise.allSettled([getManual, getAD, getAsset])`，手动失败要提示、AD/资产失败只 warn 不阻塞（设计要求：实时查询不应阻塞主流程）
    - 恢复 `setPrimaryAndSave` 调用（line 181 替代了之前的 sync+setPrimary hack）
    - catch 块增加 `console.error` 防止再次"无声失败"

- **verification**:
  - 后端 `go build ./...` 通过
  - 前端 `npx tsc --noEmit` 通过
  - 等待用户在浏览器验证：
    1. 展开工位行 → 子表格显示"手动设备"列表 + 折叠面板里"域控设备"、"资产设备"
    2. 点击 AD/资产设备的"设为主设备" → 弹窗确认 → 弹窗消失、设备转为手动设备并设为主

- **files_changed**:
  - `xingran-react-frontend/src/components/operations/WorkstationDeviceTable/index.tsx`
  - `xingran-react-frontend/src/lib/opsApi.ts`
  - `internal/api/v1/operations/workstation_device_handler.go`
  - `internal/api/router.go`

## Phase 40 Closure (2026-06-25)

复测确认两层修复均已就位:
- Layer 1 前端:`opsApi.ts:657-672` 已定义 `getManual`/`getAD`/`getAsset`,`getByWorkstation` 保留为兼容别名;`WorkstationDeviceTable/index.tsx` 三段并行查询 + catch `console.error` 就位
- Layer 2 后端:`workstation_device_handler.go` 全部用 `c.Param("id")`(line 43/97/124),路由 `POST /:id` 一致

**dev 浏览器验证通过(用户操作)**:登录 → 工位管理 → 展开有设备的工位行 → 子表格成功加载手动/域控/资产设备列表;Network `workstation-device/{id}/...` 返回 200;console 无 TypeError / 无 "加载设备失败"。frontmatter 翻 `resolved`(D-05 + D-07)。

verification: 2026-06-25 dev 浏览器验证通过,展开工位加载设备成功,Network 200,console 无错误
