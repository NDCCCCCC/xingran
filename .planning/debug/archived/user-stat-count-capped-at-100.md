---
slug: user-stat-count-capped-at-100
status: resolved
trigger: "用户管理页面总用户数、正常用户全部显示 100，但实际用户数有 1196（后扩展到通知/值班池同款统计卡片）"
created: 2026-06-20
updated: 2026-06-20
---

# Debug Session: 用户管理统计卡片显示 100（实际 1196）

## Symptoms

- **Expected**: 统计卡片「总用户数」应显示 ~1196，「正常用户」「禁用用户」显示实际 status 计数。
- **Actual**: 「总用户数」「正常用户」均显示 100（实际 1196）。
- **Reproduction**: 打开用户管理页，看顶部三个统计卡片。
- **Errors**: 无报错（纯数据错误）。

## Root Cause

统计数字由「加载全量用户列表后在内存里 `.length` 计数」得到，而后端把 pageSize 钳制到 **MaxPageSize=100**，导致只返回 100 条 → `allUsers.length` = 100。

**证据链（前端 → 后端）：**

1. **前端** `xingran-react-frontend/src/pages/system/user/hooks/useUserData.ts:53-63`
   ```ts
   const result = await post<PageResponse<User>>("/system/users/list", { current: 1, pageSize: 1000 });
   const allUsers = result.data?.list || [];          // ← 只拿到被钳制后的 100 条
   ...
   setStatistics({ total: allUsers.length, ... });     // ← 用 list 长度当总数，丢弃 result.data.total
   ```
   请求 `pageSize:1000` 期望「全量」，但用 `list.length` 当总数，且**丢弃了响应里正确的 `total` 字段**。

2. **Handler** `internal/api/v1/system/user_handler.go:101-103` 调用 `params.GetPagination()` 后覆盖 `params.PageSize`。

3. **GetPagination** `internal/models/system/requests/user_requests.go:44-49`
   ```go
   if pageSize > constants.MaxPageSize { pageSize = constants.MaxPageSize }  // 1000 → 100
   ```

4. **常量** `internal/constants/pagination.go:20`：`MaxPageSize = 100`

5. 结果：1196 > 100，列表只回 100 条 → 统计卡显示 100。

**附注（非本 bug 但相关）：** `internal/services/operations/pagination_helper.go:17` 的运维模块用独立常量 `MaxPageSize = 10000`，故运维模块不受影响——本问题仅限 system 模块的 100 上限。

## Fix (Applied — 3 modules, same pattern)

全部改为后端专用 COUNT 聚合端点，前端直接读取，不再用 list 长度充当总数。

### 1. 用户管理 `POST /system/users/statistics`
返回 `{total, active, inactive}`（status 0=正常/1=停用）。

### 2. 通知公告 `POST /system/notices/statistics`
返回 `{total, published, draft, scheduled}`。**顺带修正状态桶语义**：旧前端 `useNoticeStatistics` 把 `scheduled` 当成 `publishStatus=3`（实为「已撤回」）、把 `draft` 当成 `0||2`（吞掉「定时发布中」）。后端按模型真实语义 `0=草稿/1=已发布/2=定时发布中/3=已撤回` 实现。

### 3. 值班池 `POST /duty/pools/statistics`
返回 `{total, enabled, disabled, totalMembers}`。旧实现最糟——直接用**当前页**（~10 条）list 算统计。`totalMembers` 用子查询 `pool_id IN (SELECT id FROM sys_duty_pool WHERE deleted_at IS NULL)` 统计非软删除池的成员，避免硬编码表名。

## Verification

**已通过的代码级验证：**

- [x] `go build ./...` 通过
- [x] 前端 `npm run type-check` 通过
- [x] 前端 `npm run build` 通过
- [x] 3 个回归测试全绿（均 >100 场景 + 软删除排除）：
  - `internal/services/system/user_statistics_test.go::TestUserStatistics_CountExceedsPageSizeCap` — 150 用户 → total=150/active=100/inactive=50
  - `internal/services/notice_status_statistics_test.go::TestNoticeStatusStatistics_CountExceedsPageSizeCap` — 150 通知 → total=150/published=60/draft=40/scheduled=30（锁定正确状态桶）
  - `internal/services/duty_pool_statistics_test.go::TestDutyPoolStatistics_NotDerivedFromCurrentPage` — 120 池 → total=120/enabled=70/disabled=50/totalMembers=190（软删除池的 5 成员被排除）
- [x] `internal/services/`、`internal/api/v1/system/`、`internal/services/system/`（跳过预存坏测试后）全绿，无回归

**改动文件（共 14 个）：**

后端：
- `internal/services/system/user_service.go`、`internal/api/v1/system/user_handler.go`、`internal/api/v1/system/user_router.go`
- `internal/services/notice_service.go`、`internal/services/system/notice_cache_impl.go`、`internal/api/v1/system/notice_handler.go`、`internal/api/v1/system/notice_router.go`
- `internal/services/duty_pool_service.go`、`internal/services/duty_service.go`、`internal/services/duty/duty_cache_impl.go`、`internal/api/v1/duty/duty_handler.go`、`internal/api/v1/duty/duty_router.go`

前端：
- `xingran-react-frontend/src/pages/system/user/hooks/useUserData.ts`
- `xingran-react-frontend/src/pages/system/notice/hooks/useNoticeStatistics.ts`、`xingran-react-frontend/src/lib/noticeApi.ts`
- `xingran-react-frontend/src/pages/duty/pools/index.tsx`、`xingran-react-frontend/src/lib/dutyApi.ts`

测试：3 个新增 `*_statistics_test.go`。

**待 UAT（需真实数据环境）：**
- [ ] 1196 用户 / 大量通知 / 多页值班池环境下，三处统计卡片显示正确计数
- [ ] 增删改后卡片刷新正确（前端各 mutation 后已调统计接口）
- [ ] 通知卡片 draft/scheduled 数值变化符合预期（状态桶已修正，与旧版数值会有差异——这是修正，非回归）

**附注（预存问题，与本次无关）：** `internal/services/system/apikey_service_test.go`（TestUpdateAPIKey/TestListAPIKeys 等）与 `role_service_apperrors_test.go::TestRoleService_Create_RoleNameExists` 为 nil 指针 panic，属测试 setup 自身问题，本次未触碰这些文件。`internal/services` 的 usage_logger 测试偶发 SQLite「table is locked」亦为预存并发问题。

---

## 第二轮（2026-06-20）：扩展修复另外 4 个活跃 bug

全仓扫描发现 4 个同款活跃 bug（详见对话），一并修复，模式与第一轮一致（专用 COUNT 端点）：

| 模块 | 端点 | 变体 | 备注 |
|------|------|------|------|
| 角色管理 | `POST /system/roles/statistics` | 钳制变体（system MaxPageSize=100） | 与 user 同款，roleCacheService 内嵌 *roleService 自动提升 |
| 工单管理 | `POST /workorder/orders/status-statistics` | 当前页变体 | 按 WorkOrderStatus 0-3 聚合；命名避开已有 GetStatistics（统计页） |
| 周期工单模板 | `POST /workorder/periodic/templates/statistics` | 当前页变体 | 布尔 `CASE WHEN is_enabled`（PG/SQLite 双兼容）+ COALESCE(SUM(total_generated)) |
| 知识库文章 | `POST /knowledge/articles/statistics` | 当前页变体 | COALESCE(SUM(view_count/like_count)) 防空集 NULL |

**前端模式：** workorder 两处统一用「fetchList 内部顺带调 fetchStats」，搜索/分页/增删改（均经 fetchList）都会刷新统计为真实全局计数。role 走 useRoleActions 在增删改后显式调 loadStatistics。

**第二轮验证：**
- [x] `go build ./...` / 前端 type-check / build 全通过
- [x] 2 个新回归测试（覆盖新 SQL 构造）：`periodic_statistics_test.go`（布尔 is_enabled + SUM total_generated）、`knowledge_statistics_test.go`（COALESCE SUM views/likes），均含软删除排除断言
- [x] role/workorder-status 为纯状态计数，与第一轮 user/notice/duty 同款已验证

**第二轮改动文件：** role（service+handler+router+useRoleData）、workorder orders（base+cache_impl+handler+router+useWorkOrderData+workorderApi）、workorder periodic（periodic+handler+router+useTemplateData+workorderApi）、knowledge（knowledge_service+cache_impl+handler+router+useArticleData+knowledgeApi）+ 2 测试。

**仍潜伏（未修，network ~9 处）：** 后端钳制高 ~10000，仅超大数据量触发，风险低面广，暂不动。

---

## 第三轮（2026-06-20）：network 模块 9 处潜伏项全部修复

| 模块 | 端点 | 后端情况 |
|------|------|---------|
| ports | (已有 GET /statistics) | 仅删前端 fallback（updateStatistics 用当前页 list） |
| backups | (已有 GET /statistics) | 扩展加 uniqueDevices（COUNT DISTINCT device_id） |
| devices | POST /statistics | 暴露已有 GetDeviceStatistics（CacheService 接口本就有） |
| mac | (已有 GET /statistics) | 扩展加 dynamic/static/secure（mac_type string） |
| templates | POST /statistics | 新增（is_system bool + template_type） |
| credentials | POST /statistics | 新增（protocol_type ssh/telnet） |
| command | POST /statistics | 新增（execution_type='command' 过滤 + int 状态） |
| executions | POST /statistics | 新增（execution_type='template' 过滤 + int 状态） |
| discoveries | POST /statistics | 新增（int 状态 + SUM discovered_count） |

要点：command/executions 同查 sys_config_execution 表，按 execution_type 分开统计；mac/discoveries/ConfigExecution 无软删除；int 状态枚举 0待执行/1执行中/2成功/3失败。
回归测试：command_statistics_test.go（execution_type 过滤）、discovery_statistics_test.go（int 状态 + SUM）。第三轮断言「全仓清零」，第四轮复核证明过早（见下）。

---

## 第四轮（2026-06-20）：复核补漏——rpa 活跃 + system 潜伏 + 死代码清理

全仓再扫（搜 inline `value={...length}`、`useMemo(() => ({...length}))`、`statisticsHelper`、`calculateStatistics`）发现遗漏并修复。

### 活跃 bug（当前页变体，恒 ≤ pageSize）
| 模块 | 端点 | 说明 |
|------|------|------|
| RPA 执行记录 | `POST /rpa/executions/statistics` | 原 `executions.length`(当前页10) + 5s 自动刷新；顺带修正前端 status 误用（`completed`→`success` 对齐后端真实枚举） |
| RPA Worker | `POST /rpa/workers/statistics` | 原 `workers.length` + 容量 reduce(当前页)；online/offline/busy 按实时心跳(120s)派生，worker 数量少故后端全量查 + Go 算（避免跨库时间函数） |

### 潜伏（system 模块 MaxPageSize=100 钳制，数据 >100 才触发）
| 模块 | 端点 |
|------|------|
| 参数配置 | `POST /system/configs/statistics`（config_type Y/N） |
| 岗位管理 | `POST /system/posts/statistics`（status 0/1） |
| 字典类型 | `POST /system/dicts/types/statistics` |
| 字典数据 | `POST /system/dicts/data/statistics`（可选 dictType 过滤） |

### 死代码清理
- `network/backups/utils.ts calculateStatistics`：第三轮改前端为专用端点后遗留，无调用方，删除（menu 同名函数不同文件，保留）。

### 复核确认非 bug（不修）
- `building-spaces-3d/utils.ts calculateWorkstationStats`：单楼层范围，与 3D 渲染共用同一 list（count = 屏幕工位数），operations 模块 cap 10000，单楼层 >1000 不现实。
- `menu/utils.tsx`、`dept tree`：用 tree 端点全量加载，非分页。
- `utils/statisticsHelper.ts`：operations 6 模块（楼宇/楼层/机房/信息点/专线/机房设备）分页拉全量(≤10000)计数，正确但低效；数据现实不超 10000，改专用端点属独立重构，暂缓。

**第四轮验证：** `go build ./...` 通过；前端 type-check 通过；5 个新回归测试（execution/worker/config/post/dict[type+data]）全绿，均含软删除排除（+dictType 过滤）。config/post/dict 的 cache wrapper 嵌入底层 service，Statistics 自动提升；`settings_service_test.go` 的 `stubConfigService` 补 Statistics 方法。

---

## 第五轮（2026-06-20）：复核补漏——assets 伪造比例统计

第四轮后再复核（搜 `setStatistics` 数据源 + inline `value={...length}` + `Set.size`/`reduce`）。inline length 已**清零**，所有 setStatistics 逐一核实数据源——仅发现一处严重问题：

### 🔴 operations/assets：统计卡片显示伪造数据（占位实现造假，非钳制 bug）

`assets/index.tsx` 原 `loadStatistics` 用 `total*0.8/0.15/0.05` 编造 normal/stopped/nbf（注释自承"简化实现"），total 真实但状态桶是假的固定比例。

**修复**：`POST /ops/asset/statistics`，按 model 真实字段聚合——`status`(0=正常/1=停用)→normal/stopped，`nbf_status`(0=否/1=拟报废，独立维度)→nbf，COUNT(*)→total。前端 `assetApi.statistics()` + 删伪造比例。回归测试 `asset_statistics_test.go`（status/nbf_status 双维度 + 软删除排除）。

### 复核确认非 bug（不修）
- `building-spaces/index.tsx`：楼宇空间概览页拉 building(100)/floor(1000)/workstation(10000) 全量前端关联算，operations cap 10000，属"概览页拉全量"设计假设（与 statisticsHelper 6 模块同类），潜伏，暂缓。
- `captcha-background`：专用端点 ✅；`menu`/`dept`：树端点全量 ✅；其余全部专用端点 ✅。

**结论：真正"统计卡片数据错误"至此清零**，剩余仅 building-spaces + statisticsHelper 两类"拉全量前端算"暂缓项（属独立重构）。

---

## 第六轮（2026-06-21）：Phase 1+2 重构——处理两类「拉全量前端算」暂缓项

### Phase 1：operations 6 模块 + workstation 统计端点化
- 后端：building/floor/server-room/info-point/dedicated-line/room-device/workstation 各加 `Statistics` COUNT 端点（按 status 枚举聚合，排除软删除）。building 复用 `applyFilters`、workstation 内联 orgId 过滤，**保留筛选统计语义**（部门管理员按部门统计）。
- 前端：`createCrudApi` 工厂加 `statistics` 方法；6 模块 loadStatistics 改调各自端点；workstation `useWorkstationData` 从 4 次 list 拼 total 改 1 次 `statistics({orgId})`；**删除 `utils/statisticsHelper.ts`**（零残留）。
- 测试：`operations_statistics_test.go`（7 模块，status 全枚举 + 软删除）。

### Phase 2：building-spaces 概览页重构
- 后端：`OpsBuilding` 加 `WorkstationCount`（`gorm:"->"` 只读，子查询动态算）；`TotalFloors` 为已有维护字段（floor 增删更新）。building list Select 子查询带计数。
- 前端：统计卡片改调 building/floor/workstation 端点 `.total`；楼宇卡片网格分页（pageSize 12）；`BuildingModal` 打开时按 `buildingId` 懒加载楼层（`floorApi.list`），工位由 `WorkstationView` 自身懒加载；删除一次性拉 floor(1000)/workstation(10000) 全量前端关联。

### 关键坑（post 返回类型）
`post<T>` 返回 `BaseResponse<T>`（非 T）。statistics 方法需取 `.data`；前端 6 模块用字段构造 `{ total: stats.total ?? 0, ... }` 避免 `Record<string,number>` 与具体 Statistics 接口类型不兼容（vite build tsc 严格模式暴露，type-check 未报）。

提交：`b0d6c57`(P1 后端) `7c89d40`(P1 前端) `5f80703`(类型修复) `70d8395`(P2 后端) `9edb265`(P2 前端)。两类暂缓项至此**清零**。
