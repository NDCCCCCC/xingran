---
one-liner: 接通用户导出/工单评价端点与前端入口，用 migration 236 seed 闭环 OQ-7 五个读权限码恒 403
---

# Phase 136 — 接线型端点缺口（v1.53 / FIX-03 + FIX-06 + FIX-13）

**一行摘要**：接通用户导出与工单评价两组缺失端点，并以 migration seed + 对照证据闭环 OQ-7「5 个读权限码恒 403」。

## 交付清单

| REQ | 文件 | 内容 |
|---|---|---|
| FIX-03 | `internal/api/v1/system/user_export_handler.go`（新建）、`user_router.go`、`internal/api/router.go` | `POST /system/users/export`，复用 `ExcelService.ExportData("user")`，严格单一权限点 + operlog `OperTypeExport` |
| FIX-03 | `xingran-react-frontend/src/pages/system/user/index.tsx` | 导出按钮 + `PERMISSION_CODES.systemUser.export` 三态门控，复用 `downloadFilePost` |
| FIX-06 | `internal/api/v1/workorder/rating_handler.go`（新建）、`workorder_router.go:62-92`、`router.go:583,589-592` | 空函数 `SetupWorkOrderRatingsRouter` 接入 `RatingService`，读/写兄弟组分离，写组挂严格权限 |
| FIX-06 | `WorkOrderRatingModal.tsx`（新建）、`workorderApi.ts:641-695`、`permissionCodes.ts` | 最小可用评价入口 + V2 命名包装 API |
| FIX-06 | `migration_237_workorder_rating_perm_seed.go`（新建）+ `database.go` 双分支注册 | F 型按钮菜单 seed |
| FIX-13 | `migration_236_oq7_read_perm_seed.go`（新建）+ 测试 | 5 个读码（`ops:floor-plan:*` / `ops:walls:list` / `ops:doors:list`）幂等 seed 到 F 型按钮，精准授权给持父菜单角色 |
| FIX-13 | `pkg/permission/config.go`、`internal/api/router.go`、`scripts/route-registry.json`、`permissionCodes.ts` | 字面量升格常量（单一真相源）、TS 声明、路由注册表重生成 |
| 测试 | `rating_handler_test.go`、`pkg/middleware/oq7_read_perm_test.go`、`migration_236_..._test.go` | Create/List/Statistics 业务规则、403→200 对照、严格权限零回归、seed 幂等 |

## OQ-7 根因与证据
- **根因 = 数据侧**：5 个读码在 `sys_menu` 无 live 菜单行承载，Step 1 精确匹配 / Step 2 F 型继承 / Step 3 模块降级全落空 → 除 admin 外恒 403；非中间件代码缺陷。
- `TestOQ7_ReadRoutes_403BeforeSeed_200AfterSeed`：Before（无菜单行）403 → After（播种+授权）200，逐码验证。
- `TestOQ7_StrictPermissionZeroRegression`：只读角色调严格写路由仍 403，`RequireStrictPermissions` 语义零改动。

## 验证
- `go build ./...` exit 0；`go test ./...` exit 0 全绿
- `npx tsc --noEmit` exit 0
- `node scripts/check-permission-codes.mjs` 双向零漂移 ✓；`scripts/check-route-registry.mjs` OK（702→707 路由）
- 注：ratingsWrite 组一度误挂 `DataScopePermission` 触发挂载数 31→32 断言失败，评估后移除该非必要挂载（workorder 域不在 DS-02 收口内），全量恢复绿灯，未改其它 Phase 测试。

## 边界
- 生产库需执行 migration 236/237（随启动自动 runAndRecord）后受限角色才真正 403→200；本 Phase 未连生产 PG 写数据。
- 评价前端为最小可用弹窗入口，未新增独立菜单/路由页。
