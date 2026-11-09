// Package operations: Phase 151 / ARCH-01 资源归属注册表（ops 侧装配）。
//
// 分层契约（PLAN §3 F-4 / §4.2）：
//   - pkg/query 只存「表名 → 回调」，零业务表名、零 SQL、零档位判定；
//     本文件在 ops 包内把 149 的 entityExistsSub / effectiveOrgExpr /
//     wrapScopeExists / dept_id 直挂「注册为回调」，零 SQL 复制、零循环依赖。
//   - 谓词一律**引用既有单一真实源**（scope_filter.go / scope_entities.go /
//     dept_filter.go 的既有实现），不新增第 5 套归属链实现、不内联
//     COALESCE/NULLIF/EXISTS 骨架、不复制 scopeCacheKey 算法。
//   - 幂等守卫（scopeAlreadyApplied）：8 核心 service 已由 131/149 自觉挂
//     wrapScopeExists / applyBuildingScope / applyEntityReadScope，ScopedDB
//     的 Scopes 回调在 Count/Find 时才执行，检测到链上已有可见集谓词即
//     原样返回 → **SQL 零漂移（零回归硬约束）**；漏挂 / 新增端点由回调
//     自动补挂 → 「靠约定」升级为「靠机制」。
package operations

import (
	"context"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/xingran-next/xingran-go-backend/pkg/datascope"
	"github.com/xingran-next/xingran-go-backend/pkg/query"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// scopeOwnerTable 登记表：业务主表名 → 既有收口分派函数。
//
// 覆盖清单（PLAN §4.3 / T2 登记表，共 15 项）：
//
//	ops_buildings / ops_floors / sys_workstation / ops_asset /
//	ops_info_points / ops_server_rooms / ops_dedicated_lines /
//	ops_room_devices / ops_workstation_device / ops_walls / ops_doors /
//	ops_floor_plan_texts / ops_room_photos / sys_dept_location_alias
//
// 注：building_service_typesafe.go 旁路实现与 ops_buildings 共用同一条目，
// **不重复注册**（Register 同表重复会返回 error）。
type scopeOwnerTable struct {
	table string
	apply func(q *gorm.DB, ctx context.Context) *gorm.DB
}

// scopedOwnerTables 是唯一的登记清单（静态断言测试按本表逐项比对）。
var scopedOwnerTables = []scopeOwnerTable{
	// ---- 8 核心 service 主表（131 既有收口）----
	{table: "ops_buildings", apply: applyBuildingScope},
	{table: "ops_floors", apply: applyFloorRegistryScope},
	{table: "sys_workstation", apply: applyWorkstationRegistryScope},
	{table: "ops_asset", apply: applyAssetReadScope},
	{table: "ops_info_points", apply: applyInfoPointRegistryScope},
	// ---- 149 既有的 4 实体（entityExistsSub 归属链）----
	{table: "ops_server_rooms", apply: entityRegistryScope(scopeEntityServerRoom, "ops_server_rooms")},
	{table: "ops_dedicated_lines", apply: entityRegistryScope(scopeEntityDedicatedLine, "ops_dedicated_lines")},
	{table: "ops_room_devices", apply: entityRegistryScope(scopeEntityRoomDevice, "ops_room_devices")},
	{table: "ops_workstation_device", apply: entityRegistryScope(scopeEntityWorkstationDevice, "ops_workstation_device")},
	// ---- 149 的 5 类显式豁免面（保留文件，仅登记、不裸改起链）----
	{table: "ops_walls", apply: entityRegistryScope(scopeEntityWall, "ops_walls")},
	{table: "ops_doors", apply: entityRegistryScope(scopeEntityDoor, "ops_doors")},
	{table: "ops_floor_plan_texts", apply: entityRegistryScope(scopeEntityFloorPlanText, "ops_floor_plan_texts")},
	{table: "ops_room_photos", apply: entityRegistryScope(scopeEntityRoomPhoto, "ops_room_photos")},
	{table: "sys_dept_location_alias", apply: entityRegistryScope(scopeEntityLocationAlias, "sys_dept_location_alias")},
}

// entityRegistryScope 委托 149 的 applyEntityReadScope（entityExistsSub 归属链
// EXISTS 骨架 / dept_id 直挂 / wrapScopeExists，零 SQL 复制）。
func entityRegistryScope(entity, mainAlias string) func(*gorm.DB, context.Context) *gorm.DB {
	return func(q *gorm.DB, ctx context.Context) *gorm.DB {
		return applyEntityReadScope(q, ctx, entity, mainAlias)
	}
}

// applyFloorRegistryScope 楼层：引用 effectiveOrgExpr 单一真实源 +
// wrapScopeExists（EXISTS 骨架与 floor_service.go:62-66 同形，列引用复用既有
// orgExpr，不复制 COALESCE/NULLIF 表达式本身）。
func applyFloorRegistryScope(q *gorm.DB, ctx context.Context) *gorm.DB {
	orgExpr := effectiveOrgExpr("ops_floors.org_id", "b.org_id")
	sub := q.Session(&gorm.Session{NewDB: true}).
		Table("ops_buildings b").
		Where("CAST(b.id AS TEXT) = ops_floors.building_id").
		Where("b.deleted_at IS NULL").
		Select("1")
	return wrapScopeExists(q, ctx, sub, orgExpr, "")
}

// applyWorkstationRegistryScope 工位：引用 workstationOrgExistsSub（既有骨架）
// + wrapScopeExists + workstationOrgColumn（零 SQL 复制）。
func applyWorkstationRegistryScope(q *gorm.DB, ctx context.Context) *gorm.DB {
	sub := workstationOrgExistsSub(q, "CAST(f.id AS TEXT) = sys_workstation.floor_id")
	return wrapScopeExists(q, ctx, sub, workstationOrgColumn, "sys_workstation.user_id")
}

// applyInfoPointRegistryScope 信息点：引用 effectiveOrgExpr + wrapScopeExists
// （EXISTS 骨架与 infopoint_service.go:46-52 同形，w.user_id 供 Self 档分派）。
func applyInfoPointRegistryScope(q *gorm.DB, ctx context.Context) *gorm.DB {
	orgExpr := effectiveOrgExpr("f.org_id", "b.org_id")
	sub := q.Session(&gorm.Session{NewDB: true}).
		Table("sys_workstation w").
		Joins("JOIN ops_floors f ON CAST(f.id AS TEXT) = w.floor_id").
		Joins("JOIN ops_buildings b ON CAST(b.id AS TEXT) = f.building_id").
		Where("CAST(w.id AS TEXT) = ops_info_points.workstation_id").
		Where("w.deleted_at IS NULL AND f.deleted_at IS NULL AND b.deleted_at IS NULL").
		Select("1")
	return wrapScopeExists(q, ctx, sub, orgExpr, "w.user_id")
}

// scopeAlreadyApplied 检测链上是否**已**由 131/149 的既有收口挂了可见集谓词。
//
// GORM 的 Scopes 回调在 Count/Find 构建 SQL 时才执行，而 service 里的
// applyBuildingScope / wrapScopeExists / applyEntityReadScope 在起链后立即
// 同步挂 WHERE；故此处遍历 Statement.Clauses["WHERE"] 的 Exprs，命中
// 「EXISTS / CAST(... AS TEXT) IN / 1=0 / dept_id」任一特征即判定已挂 →
// 幂等跳过，保证 8 核心 service 的 SQL 逐字零漂移（零回归硬约束）。
func scopeAlreadyApplied(tx *gorm.DB) bool {
	if tx == nil || tx.Statement == nil {
		return false
	}
	w, ok := tx.Statement.Clauses["WHERE"]
	if !ok {
		return false
	}
	cw, ok := w.Expression.(clause.Where)
	if !ok {
		return false
	}
	for _, e := range cw.Exprs {
		expr, ok := e.(clause.Expr)
		if !ok {
			continue
		}
		sql := strings.ToUpper(expr.SQL)
		if strings.Contains(sql, "EXISTS") ||
			strings.Contains(sql, "AS TEXT) IN") ||
			strings.Contains(sql, "DEPT_ID") ||
			strings.HasPrefix(strings.TrimSpace(sql), "1=") {
			return true
		}
	}
	return false
}

// scopedPredicate 把既有收口函数包装成 query.ScopeFactory。
//   - 幂等：已有谓词 → 原样返回（零漂移）；
//   - 补挂：未挂 → 调用既有函数（机制化兜住漏挂 / 新增端点）；
//   - ctx 取自 tx.Statement.Context（Scopes 执行时已带请求 ctx）。
func scopedPredicate(apply func(*gorm.DB, context.Context) *gorm.DB) query.ScopeFactory {
	return func(tx *gorm.DB, _ datascope.ScopeResult) *gorm.DB {
		if scopeAlreadyApplied(tx) {
			return tx
		}
		ctx := tx.Statement.Context
		if ctx == nil {
			ctx = context.Background()
		}
		return apply(tx, ctx)
	}
}

// SetupScopedDBRegistry 装配资源归属注册表（供 router.go 启动期调用一次）。
//
// 返回 error（项目禁 panic）；幂等：注册表已装配（非空）时直接返回 nil，
// 不会触发「同表重复注册」——SetupRouter 在单元测试里会被逐用例多次调用
// （每个用例新建 gin Engine），进程级静态映射重复装配是错误语义。
// 单次装配内若清单里真有重复表，query.Register 仍返回 error（PLAN :276）。
// query.ResetRegistryForTest() 清空后本函数会重新装配（供就近单测隔离）。
func SetupScopedDBRegistry() error {
	if len(query.RegisteredTables()) > 0 {
		return nil
	}
	for _, item := range scopedOwnerTables {
		if err := query.Register(item.table, query.Entry{
			Predicate: scopedPredicate(item.apply),
			CacheKey:  ScopedCacheKey,
		}); err != nil {
			return err
		}
	}
	return nil
}

// MustSetupScopedDBRegistry 是 SetupScopedDBRegistry 的启动期包装：
// router.go 的 SetupRouter 无 error 返回时用它，内部 logrus.Fatalf。
func MustSetupScopedDBRegistry() {
	if err := SetupScopedDBRegistry(); err != nil {
		logrus.Fatalf("Phase 151 / ARCH-01 ScopedDB 资源归属注册表装配失败: %v", err)
	}
}
