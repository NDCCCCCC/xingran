// Package operations: Phase 133 / DS-05「扩展实体」可见集收口基建。
//
// 设计契约（LOCKED 裁定，133 只消费不重新裁定）：
//   - asset：dept_id 直挂；
//   - serverRoom：楼宇链 building_id → ops_buildings.org_id；
//   - dedicatedLine：source_room_id / dest_room_id 双链 OR 命中
//     （2026-11-07 Lead 补充裁定：原裁定 1 假定存在 building_id 列与真实 schema
//     不符，见 .planning/phases/133-scope-final/133-SUMMARY.md §0.1）；
//   - roomDevice：两级链 room_id → ops_server_rooms.building_id → ops_buildings.org_id；
//   - workstation-device：跟随 workstation_id 所指工位可见集，Self 随工位 user_id。
//
// 实现口径（硬约束）：
//   - 读端点一律复用 Phase 131 的 orgScopeClause / wrapScopeExists /
//     applyScopeToSub / applyReadScope（EXISTS 子查询形态，链断天然 0 行 = fail-closed）；
//   - 写端点单条越界复用 Phase 132 的 checkWriteScope（ErrScopeDenied 显式 403）；
//   - 批量裁剪：因 132 filterVisibleIDs 的编译期白名单只含四域表（不含四实体表），
//     且不改 132 文件，这里用 131 的 orgScopeClause + effectiveOrgExpr 构造
//     `WHERE id IN ? AND CAST(<orgExpr> AS TEXT) IN ?` 的一次性查询（与
//     filterVisibleIDs 内部逐字同源：service 层、写 SQL 之前、DB 自筛、All 短路
//     零查询、FailClosed/空集 → 空集），不新增第 5 套实现、不放宽 132 白名单；
//   - 所有 uuid/varchar 比较双侧 CAST(... AS TEXT)（PG/SQLite 双方言）；
//   - All / 0 / unset（含未挂注入中间件）完全短路、零 DB 查询，超管零回归；
//   - 异常枚举 / Self 无用户列 / 可见集空 → 读空集、写 403。
package operations

import (
	"context"

	"github.com/xingran-next/xingran-go-backend/internal/services/base"
	"github.com/xingran-next/xingran-go-backend/pkg/datascope"
	apperrors "github.com/xingran-next/xingran-go-backend/pkg/errors"
	"gorm.io/gorm"
)

// ---------------------------------------------------------------------------
// 实体常量（编译期白名单，防 SQL 注入；table/列名不来自外部输入）
// ---------------------------------------------------------------------------

const (
	scopeEntityAsset             = "asset"
	scopeEntityServerRoom        = "serverRoom"
	scopeEntityDedicatedLine     = "dedicatedLine"
	scopeEntityRoomDevice        = "roomDevice"
	scopeEntityWorkstationDevice = "workstationDevice"
	// Phase 149 / SCOPE-01~04：5 类显式豁免面收口（方案 D 零迁移）。
	// walls/doors/floor-plan-texts 经 floor_id → ops_floors → ops_buildings 楼层归属链；
	// rooms-photos 经 room_id → ops_server_rooms → ops_buildings 两级链；
	// location-alias 按已有 dept_id ∈ 可见集直挂（不新增列）。
	scopeEntityWall          = "wall"
	scopeEntityDoor          = "door"
	scopeEntityFloorPlanText = "floorPlanText"
	scopeEntityRoomPhoto     = "roomPhoto"
	scopeEntityLocationAlias = "locationAlias"
)

// entityScopeTables 各实体主表名与「归属链有效机构」列所在的子查询表别名。
var entityScopeTables = map[string]string{
	scopeEntityAsset:             "ops_asset",
	scopeEntityServerRoom:        "ops_server_rooms",
	scopeEntityDedicatedLine:     "ops_dedicated_lines",
	scopeEntityRoomDevice:        "ops_room_devices",
	scopeEntityWorkstationDevice: "ops_workstation_device",
	// Phase 149：5 类新实体表名（编译期白名单，防 SQL 注入；table/列名不来自外部输入）。
	scopeEntityWall:          "ops_walls",
	scopeEntityDoor:          "ops_doors",
	scopeEntityFloorPlanText: "ops_floor_plan_texts",
	scopeEntityRoomPhoto:     "ops_room_photos",
	scopeEntityLocationAlias: "sys_dept_location_alias",
}

// entityBuildingOrgExpr 楼宇链「有效机构」表达式（单一真实源口径）。
//
// 四实体归属链终点都是 ops_buildings.org_id（无楼层覆盖列参与），故有效机构
// 就是 b.org_id 本身；orgScopeClause 会再包一层 CAST(... AS TEXT) 生成
// `CAST(b.org_id AS TEXT) IN ?`，与 131 读侧逐字同源。
const entityBuildingOrgExpr = "b.org_id"

// entityWorkstationOrgExpr 工位链「有效机构」表达式（楼层覆盖优先，单一真实源
// dept_filter.go::effectiveOrgExpr）；workstation-device 跟随工位可见集时使用。
func entityWorkstationOrgExpr() string {
	return effectiveOrgExpr("f.org_id", "b.org_id")
}

// entityFloorOrgExpr 楼层归属链「有效机构」表达式（单一真实源）。
//
// Phase 149：walls / doors / floor-plan-texts 的 floor_id → ops_floors f →
// ops_buildings b 归属链终点与工位链完全相同（楼层 org_id 可空覆盖楼宇 org_id），
// 故直接复用 effectiveOrgExpr("f.org_id","b.org_id")，禁止内联复制 COALESCE/NULLIF。
func entityFloorOrgExpr() string {
	return effectiveOrgExpr("f.org_id", "b.org_id")
}

// ---------------------------------------------------------------------------
// 读端点：按归属链注入可见集（EXISTS 子查询形态，链断 → 0 行 → 空集）
// ---------------------------------------------------------------------------

// entityExistsSub 构造「归属链 EXISTS 子查询」骨架（不含可见集条件）。
//
// sub 内的表别名：sr=ops_server_rooms、b=ops_buildings、f=ops_floors、
// w=sys_workstation；主表列用 <主表>.<外键> 形式引用（由调用方在 mainAlias
// 指定），保证 EXISTS 子查询可与任意主查询组合。
//
// 断链处理：任一级 JOIN 不上 / 软删 → 子查询 0 行 → 外层 EXISTS=false → 空集，
// 天然 fail-closed，无需额外 1=0。
func entityExistsSub(db *gorm.DB, entity string, mainAlias string) *gorm.DB {
	sub := db.Session(&gorm.Session{NewDB: true})
	switch entity {
	case scopeEntityServerRoom:
		// serverRoom.building_id → ops_buildings.org_id（一级）
		return sub.Table("ops_buildings b").
			Where("CAST(b.id AS TEXT) = " + mainAlias + ".building_id").
			Where("b.deleted_at IS NULL").
			Select("1")

	case scopeEntityRoomDevice:
		// roomDevice.room_id → ops_server_rooms.building_id → ops_buildings.org_id（两级）
		return sub.Table("ops_server_rooms sr").
			Joins("JOIN ops_buildings b ON CAST(b.id AS TEXT) = sr.building_id").
			Where("CAST(sr.id AS TEXT) = " + mainAlias + ".room_id").
			Where("sr.deleted_at IS NULL AND b.deleted_at IS NULL").
			Select("1")

	case scopeEntityWorkstationDevice:
		// workstation-device → 工位可见集（工位 → 楼层 f → 楼宇 b 有效机构）。
		// sys_workstation.user_id 在 sub 作用域内可引用，供 Self 档分派。
		return sub.Table("sys_workstation w").
			Joins("JOIN ops_floors f ON CAST(f.id AS TEXT) = w.floor_id").
			Joins("JOIN ops_buildings b ON CAST(b.id AS TEXT) = f.building_id").
			Where("CAST(w.id AS TEXT) = CAST(" + mainAlias + ".workstation_id AS TEXT)").
			Where("w.deleted_at IS NULL AND f.deleted_at IS NULL AND b.deleted_at IS NULL").
			Select("1")

	case scopeEntityWall, scopeEntityDoor, scopeEntityFloorPlanText:
		// Phase 149 楼层归属链：主表.<floor_id> → ops_floors f → ops_buildings b。
		// 有效机构 = entityFloorOrgExpr() = effectiveOrgExpr("f.org_id","b.org_id")。
		// 断链（floor 软删 / 不存在 / building 软删）→ JOIN 不上 → EXISTS 0 行 → 空集。
		return sub.Table("ops_floors f").
			Joins("JOIN ops_buildings b ON CAST(b.id AS TEXT) = f.building_id").
			Where("CAST(f.id AS TEXT) = " + mainAlias + ".floor_id").
			Where("f.deleted_at IS NULL AND b.deleted_at IS NULL").
			Select("1")

	case scopeEntityRoomPhoto:
		// Phase 149 两级链：ops_room_photos.room_id → ops_server_rooms sr
		// → ops_buildings b，与 ds05 裁定 1 的 roomDevice 两级链逐字同款骨架，
		// 仅主表外键列改为 room_id；有效机构 = entityBuildingOrgExpr (b.org_id)。
		return sub.Table("ops_server_rooms sr").
			Joins("JOIN ops_buildings b ON CAST(b.id AS TEXT) = sr.building_id").
			Where("CAST(sr.id AS TEXT) = " + mainAlias + ".room_id").
			Where("sr.deleted_at IS NULL AND b.deleted_at IS NULL").
			Select("1")

	default:
		return nil
	}
}

// dedicatedLineEndSub 构造专线「单端」归属链 EXISTS 子查询骨架。
//
// 链：ops_dedicated_lines.<roomCol> → ops_server_rooms.building_id → ops_buildings.org_id。
// 两端各调一次，外层用 OR 组合（任一端命中可见集即放行）。
func dedicatedLineEndSub(db *gorm.DB, roomCol string) *gorm.DB {
	return db.Session(&gorm.Session{NewDB: true}).
		Table("ops_server_rooms sr").
		Joins("JOIN ops_buildings b ON CAST(b.id AS TEXT) = sr.building_id").
		Where("CAST(sr.id AS TEXT) = ops_dedicated_lines." + roomCol).
		Where("sr.deleted_at IS NULL AND b.deleted_at IS NULL").
		Select("1")
}

// applyAssetReadScope 资产读端点：dept_id 直挂，本表列可直接引用。
// Self 档（无用户列）由 applyReadScope 的 userCol="" 分支返回 1=0 空集。
func applyAssetReadScope(query *gorm.DB, ctx context.Context) *gorm.DB {
	return applyReadScope(query, ctx, "dept_id", "")
}

// applyEntityReadScope 统一读端点分派：按实体归属链注入可见集。
//
// 返回 *gorm.DB，可直接作为 base.Scope 闭包返回值，或由调用方在
// repo.List / repo.GetByID 的 scopes 变参中使用（见各 service）。
//
// All / 未注入 → 原样返回（零 DB 查询、零额外 SQL，超管零回归）。
func applyEntityReadScope(query *gorm.DB, ctx context.Context, entity, mainAlias string) *gorm.DB {
	// Phase 149：location-alias 按已有 dept_id ∈ 可见集直挂（不走 EXISTS）。
	// 与 asset 的 applyAssetReadScope 逐字同款（本表列可直引用 + Self 无用户列空集）。
	if entity == scopeEntityAsset || entity == scopeEntityLocationAlias {
		return applyAssetReadScope(query, ctx)
	}

	scope, ok := scopeFromContext(ctx)
	if scopeIsAll(scope, ok) {
		return query
	}

	switch entity {
	case scopeEntityServerRoom, scopeEntityRoomDevice:
		sub := entityExistsSub(query, entity, mainAlias)
		return wrapScopeExists(query, ctx, sub, entityBuildingOrgExpr, "")

	case scopeEntityWall, scopeEntityDoor, scopeEntityFloorPlanText:
		// Phase 149 楼层归属链：EXISTS 子查询 + 楼层覆盖有效机构单一真实源。
		sub := entityExistsSub(query, entity, mainAlias)
		return wrapScopeExists(query, ctx, sub, entityFloorOrgExpr(), "")

	case scopeEntityRoomPhoto:
		// Phase 149 两级链（room → serverRoom → building），取 b.org_id。
		sub := entityExistsSub(query, entity, mainAlias)
		return wrapScopeExists(query, ctx, sub, entityBuildingOrgExpr, "")

	case scopeEntityDedicatedLine:
		// 双链 OR：两端各自挂可见集条件的 EXISTS，任一命中即放行；
		// 两端都断链 / 都越界 → 两个 EXISTS 都为 false → 空集（fail-closed）。
		src := dedicatedLineEndSub(query, "source_room_id")
		dst := dedicatedLineEndSub(query, "dest_room_id")
		scopedSrc, skipSrc := applyScopeToSub(src, ctx, entityBuildingOrgExpr, "")
		scopedDst, skipDst := applyScopeToSub(dst, ctx, entityBuildingOrgExpr, "")
		// 非 All 档下 skip 恒为 false（Self 无用户列 / FailClosed / 空集都被
		// applyScopeToSub 转成 1=0 并 applied=true）；防御性兜底仍按空集处理。
		if skipSrc || skipDst {
			return query.Where("1=0")
		}
		return query.Where("(EXISTS (?) OR EXISTS (?))", scopedSrc, scopedDst)

	case scopeEntityWorkstationDevice:
		sub := entityExistsSub(query, entity, mainAlias)
		// Self 档按工位 user_id 收窄（subUserCol = "w.user_id"）；
		// 其余档按工位有效机构（楼层覆盖优先）∈ 可见集。
		return wrapScopeExists(query, ctx, sub, entityWorkstationOrgExpr(), "w.user_id")

	default:
		return query.Where("1=0")
	}
}

// entityReadScope 把 applyEntityReadScope 包装成 base.Scope，供 repo.List /
// repo.GetByID 的变参 scopes 直接使用。
func entityReadScope(ctx context.Context, entity, mainAlias string) base.Scope {
	return func(db *gorm.DB) *gorm.DB {
		return applyEntityReadScope(db, ctx, entity, mainAlias)
	}
}

// ---------------------------------------------------------------------------
// 写端点：单条越界 403 + 批量按可见集裁剪
// ---------------------------------------------------------------------------

// entityEffectiveOrgs 按归属链解析「已存在资源」的有效机构候选集。
//
// 写侧 Delete/Update 的「改前」校验：把资源 ID 经归属链回溯到
// ops_buildings.org_id（asset 为 ops_asset.dept_id）。断链 / 软删 / 无行 →
// 返回空切片，调用方按 fail-closed 拒绝。
//
// 返回多个 orgID 仅 dedicatedLine（source/dest 两端）会出现；其余实体 0~1 个。
func entityEffectiveOrgs(ctx context.Context, db *gorm.DB, entity, id string) ([]string, error) {
	if id == "" {
		return nil, nil
	}
	var orgs []string
	var err error
	switch entity {
	case scopeEntityAsset:
		err = db.WithContext(ctx).Table("ops_asset").
			Where("CAST(id AS TEXT) = ? AND deleted_at IS NULL", id).
			Pluck("CAST(dept_id AS TEXT)", &orgs).Error

	case scopeEntityServerRoom:
		// serverRoom.building_id → b.org_id
		err = db.WithContext(ctx).Table("ops_server_rooms sr").
			Joins("JOIN ops_buildings b ON CAST(b.id AS TEXT) = sr.building_id").
			Where("CAST(sr.id AS TEXT) = ? AND sr.deleted_at IS NULL AND b.deleted_at IS NULL", id).
			Pluck("CAST(b.org_id AS TEXT)", &orgs).Error

	case scopeEntityRoomDevice:
		// roomDevice.room_id → sr.building_id → b.org_id（两级）
		err = db.WithContext(ctx).Table("ops_room_devices rd").
			Joins("JOIN ops_server_rooms sr ON CAST(sr.id AS TEXT) = rd.room_id").
			Joins("JOIN ops_buildings b ON CAST(b.id AS TEXT) = sr.building_id").
			Where("CAST(rd.id AS TEXT) = ? AND rd.deleted_at IS NULL AND sr.deleted_at IS NULL AND b.deleted_at IS NULL", id).
			Pluck("CAST(b.org_id AS TEXT)", &orgs).Error

	case scopeEntityDedicatedLine:
		// 双链：source_room_id 与 dest_room_id 各自回溯，两个结果都进候选集
		// （OR 语义由 checkEntityWriteOrgs「任一命中即放行」表达）。
		for _, col := range []string{"source_room_id", "dest_room_id"} {
			var one []string
			q := db.WithContext(ctx).Table("ops_dedicated_lines dl").
				Joins("JOIN ops_server_rooms sr ON CAST(sr.id AS TEXT) = dl."+col).
				Joins("JOIN ops_buildings b ON CAST(b.id AS TEXT) = sr.building_id").
				Where("CAST(dl.id AS TEXT) = ? AND dl.deleted_at IS NULL AND sr.deleted_at IS NULL AND b.deleted_at IS NULL", id).
				Pluck("CAST(b.org_id AS TEXT)", &one)
			if q.Error != nil {
				return nil, q.Error
			}
			orgs = append(orgs, one...)
		}

	case scopeEntityWorkstationDevice:
		// 跟随工位可见集：ops_workstation_device.workstation_id → sys_workstation
		// → ops_floors f → ops_buildings b 有效机构（楼层覆盖优先）。
		orgExpr := entityWorkstationOrgExpr()
		err = db.WithContext(ctx).Table("ops_workstation_device wd").
			Joins("JOIN sys_workstation w ON CAST(w.id AS TEXT) = CAST(wd.workstation_id AS TEXT)").
			Joins("JOIN ops_floors f ON CAST(f.id AS TEXT) = w.floor_id").
			Joins("JOIN ops_buildings b ON CAST(b.id AS TEXT) = f.building_id").
			Where("CAST(wd.id AS TEXT) = ? AND wd.deleted_at IS NULL AND w.deleted_at IS NULL AND f.deleted_at IS NULL AND b.deleted_at IS NULL", id).
			Pluck("CAST("+orgExpr+" AS TEXT)", &orgs).Error

	case scopeEntityWall, scopeEntityDoor, scopeEntityFloorPlanText:
		// Phase 149 楼层归属链（改前）：主表 → ops_floors f → ops_buildings b，
		// 取 CAST(entityFloorOrgExpr() AS TEXT)；断链 / 软删 → 空切片 → fail-closed。
		orgExpr := entityFloorOrgExpr()
		err = db.WithContext(ctx).Table(entityScopeTables[entity]+" e").
			Joins("JOIN ops_floors f ON CAST(f.id AS TEXT) = e.floor_id").
			Joins("JOIN ops_buildings b ON CAST(b.id AS TEXT) = f.building_id").
			Where("CAST(e.id AS TEXT) = ? AND e.deleted_at IS NULL AND f.deleted_at IS NULL AND b.deleted_at IS NULL", id).
			Pluck("CAST("+orgExpr+" AS TEXT)", &orgs).Error

	case scopeEntityRoomPhoto:
		// Phase 149 两级链（改前）：ops_room_photos → ops_server_rooms sr
		// → ops_buildings b，取 b.org_id。
		err = db.WithContext(ctx).Table("ops_room_photos e").
			Joins("JOIN ops_server_rooms sr ON CAST(sr.id AS TEXT) = e.room_id").
			Joins("JOIN ops_buildings b ON CAST(b.id AS TEXT) = sr.building_id").
			Where("CAST(e.id AS TEXT) = ? AND e.deleted_at IS NULL AND sr.deleted_at IS NULL AND b.deleted_at IS NULL", id).
			Pluck("CAST(b.org_id AS TEXT)", &orgs).Error

	case scopeEntityLocationAlias:
		// Phase 149 dept_id 直挂：本表列直接取；软删 / 不存在 → 空切片。
		err = db.WithContext(ctx).Table("sys_dept_location_alias").
			Where("CAST(id AS TEXT) = ? AND deleted_at IS NULL", id).
			Pluck("CAST(dept_id AS TEXT)", &orgs).Error

	default:
		return nil, nil
	}
	if err != nil {
		return nil, apperrors.Wrap(err, apperrors.CodeDatabaseError, "解析实体归属失败")
	}
	// 去空 + 去重，避免断链产生的空串被误当作候选机构。
	seen := make(map[string]struct{}, len(orgs))
	out := make([]string, 0, len(orgs))
	for _, o := range orgs {
		if o == "" {
			continue
		}
		if _, dup := seen[o]; dup {
			continue
		}
		seen[o] = struct{}{}
		out = append(out, o)
	}
	return out, nil
}

// entitySubmittedOrgs 解析「提交的资源」的有效机构候选集（Create/Update 提交侧）。
//
// 参数 fkIDs 为归属链外键候选（各实体含义不同）：
//   - asset：[0] = dept_id（直挂，无需查库）；
//   - serverRoom：[0] = building_id → 查 b.org_id；
//   - roomDevice：[0] = room_id → 查 sr.building_id → b.org_id；
//   - dedicatedLine：[0]=source_room_id、[1]=dest_room_id（双链 OR）；
//   - workstationDevice：[0] = workstation_id → 工位有效机构（+ user_id 供 Self）。
//
// 空 / 断链 / 查不到 → 该候选不加入结果；结果为空时调用方 fail-closed。
func entitySubmittedOrgs(ctx context.Context, db *gorm.DB, entity string, fkIDs ...string) ([]string, error) {
	pick := func(i int) string {
		if i < len(fkIDs) {
			return fkIDs[i]
		}
		return ""
	}
	var orgs []string
	var err error
	switch entity {
	case scopeEntityAsset:
		if o := pick(0); o != "" {
			orgs = append(orgs, o)
		}

	case scopeEntityServerRoom:
		if bid := pick(0); bid != "" {
			var one []string
			err = db.WithContext(ctx).Table("ops_buildings b").
				Where("CAST(b.id AS TEXT) = ? AND b.deleted_at IS NULL", bid).
				Pluck("CAST(b.org_id AS TEXT)", &one).Error
			orgs = append(orgs, one...)
		}

	case scopeEntityRoomDevice:
		if rid := pick(0); rid != "" {
			var one []string
			err = db.WithContext(ctx).Table("ops_server_rooms sr").
				Joins("JOIN ops_buildings b ON CAST(b.id AS TEXT) = sr.building_id").
				Where("CAST(sr.id AS TEXT) = ? AND sr.deleted_at IS NULL AND b.deleted_at IS NULL", rid).
				Pluck("CAST(b.org_id AS TEXT)", &one).Error
			orgs = append(orgs, one...)
		}

	case scopeEntityDedicatedLine:
		for _, rid := range []string{pick(0), pick(1)} {
			if rid == "" {
				continue
			}
			var one []string
			q := db.WithContext(ctx).Table("ops_server_rooms sr").
				Joins("JOIN ops_buildings b ON CAST(b.id AS TEXT) = sr.building_id").
				Where("CAST(sr.id AS TEXT) = ? AND sr.deleted_at IS NULL AND b.deleted_at IS NULL", rid).
				Pluck("CAST(b.org_id AS TEXT)", &one)
			if q.Error != nil {
				return nil, apperrors.Wrap(q.Error, apperrors.CodeDatabaseError, "解析专线端点归属失败")
			}
			orgs = append(orgs, one...)
		}

	case scopeEntityWorkstationDevice:
		if wid := pick(0); wid != "" {
			orgExpr := entityWorkstationOrgExpr()
			var one []string
			err = db.WithContext(ctx).Table("sys_workstation w").
				Joins("JOIN ops_floors f ON CAST(f.id AS TEXT) = w.floor_id").
				Joins("JOIN ops_buildings b ON CAST(b.id AS TEXT) = f.building_id").
				Where("CAST(w.id AS TEXT) = CAST(? AS TEXT) AND w.deleted_at IS NULL AND f.deleted_at IS NULL AND b.deleted_at IS NULL", wid).
				Pluck("CAST("+orgExpr+" AS TEXT)", &one).Error
			orgs = append(orgs, one...)
		}

	case scopeEntityWall, scopeEntityDoor, scopeEntityFloorPlanText:
		// Phase 149 提交侧：fkIDs[0] = floorID → 查 ops_floors f JOIN ops_buildings b
		// 取楼层覆盖有效机构；断链 / 空 → 候选为空 → fail-closed。
		if fid := pick(0); fid != "" {
			orgExpr := entityFloorOrgExpr()
			var one []string
			err = db.WithContext(ctx).Table("ops_floors f").
				Joins("JOIN ops_buildings b ON CAST(b.id AS TEXT) = f.building_id").
				Where("CAST(f.id AS TEXT) = CAST(? AS TEXT) AND f.deleted_at IS NULL AND b.deleted_at IS NULL", fid).
				Pluck("CAST("+orgExpr+" AS TEXT)", &one).Error
			orgs = append(orgs, one...)
		}

	case scopeEntityRoomPhoto:
		// Phase 149 提交侧：fkIDs[0] = roomID → 查 ops_server_rooms sr
		// JOIN ops_buildings b 取 b.org_id；断链 → 候选为空。
		if rid := pick(0); rid != "" {
			var one []string
			err = db.WithContext(ctx).Table("ops_server_rooms sr").
				Joins("JOIN ops_buildings b ON CAST(b.id AS TEXT) = sr.building_id").
				Where("CAST(sr.id AS TEXT) = CAST(? AS TEXT) AND sr.deleted_at IS NULL AND b.deleted_at IS NULL", rid).
				Pluck("CAST(b.org_id AS TEXT)", &one).Error
			orgs = append(orgs, one...)
		}

	case scopeEntityLocationAlias:
		// Phase 149 dept_id 直挂（提交侧）：[0] = deptID，无需查库。
		if o := pick(0); o != "" {
			orgs = append(orgs, o)
		}

	default:
		return nil, nil
	}
	if err != nil {
		return nil, apperrors.Wrap(err, apperrors.CodeDatabaseError, "解析提交归属失败")
	}
	seen := make(map[string]struct{}, len(orgs))
	out := make([]string, 0, len(orgs))
	for _, o := range orgs {
		if o == "" {
			continue
		}
		if _, dup := seen[o]; dup {
			continue
		}
		seen[o] = struct{}{}
		out = append(out, o)
	}
	return out, nil
}

// checkEntityWriteOrgs 统一「单条写越界」判定：候选有效机构任一落在可见集内即放行。
//
// 复用 Phase 132 的 checkWriteScope（同一 ErrScopeDenied 显式 403）：
//   - All / 未注入 → 任一候选首次调用即 nil（放行，超管零回归）；
//   - FailClosed / Self 无用户列 / 空集 → 每次都 ErrScopeDenied；
//   - DeptIDs 档 → 任一候选命中 IN 集合即放行（dedicatedLine 双链 OR 语义天然成立）；
//   - 候选为空（归属链断链）→ 直接 ErrScopeDenied（fail-closed，不回退 All、不猜测）。
//
// workstation-device 的 Self 档需按工位 user_id 判定，由调用方额外使用
// checkEntityWorkstationDeviceSelfUser 处理；此处保持四实体通用。
func checkEntityWriteOrgs(ctx context.Context, orgs []string) error {
	// All / 0 / unset（含 FromContext ok=false 未挂注入中间件）→ 完全放行。
	// 这是 131/132 的 scopeIsAll(scope, ok) = !ok || scope.All 契约（超管 /
	// 存量直接以 context.Background() 调用的单测与内部任务零回归硬约束）：
	// 未注入档不得做任何归属解析或空集拒绝。
	scope, ok := scopeFromContext(ctx)
	if scopeIsAll(scope, ok) {
		return nil
	}
	if len(orgs) == 0 {
		return ErrScopeDenied
	}
	var last error
	for _, o := range orgs {
		if err := checkWriteScope(ctx, o); err == nil {
			return nil
		} else {
			last = err
		}
	}
	if last != nil {
		return last
	}
	return ErrScopeDenied
}

// checkEntityWriteScope 单条写越界校验（Delete / Update 改前侧）：
// 按实体归属链解析已存在资源的有效机构后统一判定，越界 → ErrScopeDenied(403)。
func checkEntityWriteScope(ctx context.Context, db *gorm.DB, entity, id string) error {
	// All / 未注入：零 DB 查询直接放行（131/132 scopeIsAll 契约，存量零回归）。
	scope, ok := scopeFromContext(ctx)
	if scopeIsAll(scope, ok) {
		return nil
	}
	orgs, err := entityEffectiveOrgs(ctx, db, entity, id)
	if err != nil {
		return err
	}
	return checkEntityWriteOrgs(ctx, orgs)
}

// checkEntitySubmittedScope 单条写越界校验（Create / Update 提交侧）。
func checkEntitySubmittedScope(ctx context.Context, db *gorm.DB, entity string, fkIDs ...string) error {
	// All / 未注入：零 DB 查询直接放行（131/132 scopeIsAll 契约，存量零回归）。
	scope, ok := scopeFromContext(ctx)
	if scopeIsAll(scope, ok) {
		return nil
	}
	orgs, err := entitySubmittedOrgs(ctx, db, entity, fkIDs...)
	if err != nil {
		return err
	}
	return checkEntityWriteOrgs(ctx, orgs)
}

// ---------------------------------------------------------------------------
// workstation-device 前置守卫（裁定 2：跟随工位可见集，无独立机构语义）
// ---------------------------------------------------------------------------

// workstationScopeDecision 判定「指定工位」是否落在可见集内。
//
// 口径与四实体一致（All 短路 / FailClosed 空 / DeptIDs 按有效机构 / Self 按
// 工位 user_id），但按**工位 ID**（而非设备 ID）解析，供 workstation-device
// 的读/写端点在任何 SQL 之前做前置校验（裁定 2）。
//
// 返回 (allowed, error)：allowed=false 表示越界（读返回空 / 写 403）；
// error 仅表示解析过程 DB 出错。
func workstationScopeDecision(ctx context.Context, db *gorm.DB, workstationID string) (bool, error) {
	if workstationID == "" {
		return false, nil
	}
	scope, ok := scopeFromContext(ctx)
	if scopeIsAll(scope, ok) {
		return true, nil // 超管零回归、零 DB 查询
	}
	if scope.FailClosed {
		return false, nil
	}

	// 工位归属链：sys_workstation → ops_floors f → ops_buildings b，
	// 有效机构走 effectiveOrgExpr 单一真实源；Self 档另取 w.user_id。
	orgExpr := entityWorkstationOrgExpr()
	var row struct {
		EffOrg string
		UserID string
	}
	err := db.WithContext(ctx).Table("sys_workstation w").
		Joins("JOIN ops_floors f ON CAST(f.id AS TEXT) = w.floor_id").
		Joins("JOIN ops_buildings b ON CAST(b.id AS TEXT) = f.building_id").
		Where("CAST(w.id AS TEXT) = CAST(? AS TEXT) AND w.deleted_at IS NULL AND f.deleted_at IS NULL AND b.deleted_at IS NULL", workstationID).
		Select("CAST(" + orgExpr + " AS TEXT) AS eff_org, CAST(w.user_id AS TEXT) AS user_id").
		Scan(&row).Error
	if err != nil {
		return false, apperrors.Wrap(err, apperrors.CodeDatabaseError, "解析工位归属失败")
	}
	if scope.ResourceHook == datascope.HookSelfUser {
		// Self 档：随工位 user_id 收窄（裁定 2）；UserID 缺失 → fail-closed。
		if scope.UserID == "" || row.UserID == "" || row.UserID != scope.UserID {
			return false, nil
		}
		return true, nil
	}
	if len(scope.DeptIDs) == 0 || row.EffOrg == "" {
		return false, nil
	}
	return orgInVisibleSet(row.EffOrg, scope.DeptIDs), nil
}

// checkWorkstationDeviceWriteScope 写端点前置守卫：目标工位越界 → ErrScopeDenied(403)。
func checkWorkstationDeviceWriteScope(ctx context.Context, db *gorm.DB, workstationID string) error {
	// All / 0 / unset（含 FromContext ok=false 未挂注入中间件）→ 完全放行、
	// 零 DB 查询。与四实体写侧一致（131/132 scopeIsAll 契约，存量单测零回归）：
	// 未注入档不得做任何工位归属解析，也不得因「不存在 id / 空 workstationID」
	// 误判为越界 403（例：DeleteDevice 传不存在 id 的存量行为不报错）。
	scope, ok := scopeFromContext(ctx)
	if scopeIsAll(scope, ok) {
		return nil
	}
	allowed, err := workstationScopeDecision(ctx, db, workstationID)
	if err != nil {
		return err
	}
	if !allowed {
		return ErrScopeDenied
	}
	return nil
}

// workstationDeviceReadAllowed 读端点前置守卫：越界工位返回 false（→ 空集，不报错）。
func workstationDeviceReadAllowed(ctx context.Context, db *gorm.DB, workstationID string) bool {
	allowed, err := workstationScopeDecision(ctx, db, workstationID)
	return err == nil && allowed
}

// visibleWorkstationIDs 批量读前置裁剪：只保留落在可见集内的工位 ID。
//
// 供 GetADDevicesByWorkstations / GetAssetDevicesByWorkstations /
// GetPhysicalDevicesByWorkstations 三个导出优化批量 helper 使用（裁定 2：
// 工位 ∈ 可见集否则不返回）。All / 未注入短路：零 DB 查询、原样返回。
func visibleWorkstationIDs(ctx context.Context, db *gorm.DB, ids []string) []string {
	scope, ok := scopeFromContext(ctx)
	if scopeIsAll(scope, ok) {
		return append([]string(nil), ids...)
	}
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		allowed, err := workstationScopeDecision(ctx, db, id)
		if err == nil && allowed {
			out = append(out, id)
		}
	}
	return out
}

// deviceWorkstationID 从 ops_workstation_device 取指定设备关联的工位 ID。
// 供 UpdateDevice / DeleteDevice / SetPrimaryDevice 这类「按设备 ID」的写端点
// 在校验前解析目标工位（不存在/软删 → 空串 → 后续 fail-closed 403）。
func deviceWorkstationID(ctx context.Context, db *gorm.DB, deviceID string) (string, error) {
	if deviceID == "" {
		return "", nil
	}
	var ids []string
	err := db.WithContext(ctx).Table("ops_workstation_device").
		Where("CAST(id AS TEXT) = ? AND deleted_at IS NULL", deviceID).
		Pluck("CAST(workstation_id AS TEXT)", &ids).Error
	if err != nil {
		return "", apperrors.Wrap(err, apperrors.CodeDatabaseError, "解析设备关联工位失败")
	}
	if len(ids) == 0 {
		return "", nil
	}
	return ids[0], nil
}

// ---------------------------------------------------------------------------
// 批量裁剪：与 132 filterVisibleIDs 同源 SQL（不改 132 白名单）
// ---------------------------------------------------------------------------

// filterEntityVisibleIDs 按可见集裁剪目标 ID 集，返回 (可见 ID, 越界行数, error)。
//
// 语义与 132 filterVisibleIDs 逐字同源（service 层、任何写 SQL 之前、一次性
// `WHERE id IN ? AND <有效机构 ∈ 可见集>`、DB 自筛、禁止先取全量再内存过滤）：
//   - 用 131 orgScopeClause(scope, ok, orgExpr) 生成 `CAST(orgExpr AS TEXT) IN ?`
//     / `1=0` / skip 三态，不内联复制 COALESCE/NULLIF；
//   - orgExpr 由各实体归属链构造（EXISTS 子查询形态），链断天然不命中；
//   - All / 未注入 → 零 DB 查询、原样返回 ids、越界行数 0（超管零回归）；
//   - FailClosed / Self 无用户列 / 空集 → 返回空切片，越界行数 = len(ids)。
//
// 「不改 132 文件」：132 filterVisibleIDs 的 filterVisibleIDsAllowedTables 白名单
// 只含四域表，四实体表不在其中；本函数按 Lead 确认用 131 同源 SQL 自行裁剪，
// 不放宽该白名单（决策登记见 133-SUMMARY.md §0.3）。
func filterEntityVisibleIDs(ctx context.Context, db *gorm.DB, entity string, ids []string) ([]string, int, error) {
	if len(ids) == 0 {
		return []string{}, 0, nil
	}
	table, ok := entityScopeTables[entity]
	if !ok {
		return nil, 0, apperrors.New(apperrors.CodeParamError, "非法的批量裁剪实体")
	}
	scope, okScope := scopeFromContext(ctx)
	if scopeIsAll(scope, okScope) {
		return append([]string(nil), ids...), 0, nil
	}

	// 构造「有效机构 EXISTS 子查询」骨架（与读侧 applyEntityReadScope 同源）。
	var sub *gorm.DB
	var orgExpr string
	switch entity {
	case scopeEntityAsset, scopeEntityLocationAlias:
		// dept_id 直挂：本表列可直接引用，无需 EXISTS。
		// Phase 149：location-alias 已有 DeptID 列，与 asset 逐字同款；
		// table 由 entityScopeTables 映射到 sys_dept_location_alias。
		cond, args, skip := orgScopeClause(scope, okScope, "dept_id")
		if skip {
			return append([]string(nil), ids...), 0, nil
		}
		q := db.WithContext(ctx).Table(table).
			Select("DISTINCT CAST(id AS TEXT)").
			Where("id IN ?", ids).
			Where(cond, args...)
		var visible []string
		if err := q.Pluck("CAST(id AS TEXT)", &visible).Error; err != nil {
			return nil, 0, apperrors.Wrap(err, apperrors.CodeDatabaseError, "批量裁剪可见集失败")
		}
		return visible, len(ids) - len(visible), nil

	case scopeEntityServerRoom, scopeEntityRoomDevice, scopeEntityWorkstationDevice:
		sub = entityExistsSub(db, entity, table)
		if entity == scopeEntityWorkstationDevice {
			orgExpr = entityWorkstationOrgExpr()
		} else {
			orgExpr = entityBuildingOrgExpr
		}

	case scopeEntityWall, scopeEntityDoor, scopeEntityFloorPlanText:
		// Phase 149 楼层归属链批量裁剪：sub = 楼层链 EXISTS，orgExpr = 楼层覆盖
		// 有效机构单一真实源 entityFloorOrgExpr()（禁止内联 COALESCE/NULLIF）。
		sub = entityExistsSub(db, entity, table)
		orgExpr = entityFloorOrgExpr()

	case scopeEntityRoomPhoto:
		// Phase 149 两级链批量裁剪：room → serverRoom → building，取 b.org_id。
		sub = entityExistsSub(db, entity, table)
		orgExpr = entityBuildingOrgExpr

	case scopeEntityDedicatedLine:
		// 双链 OR：构造一个「两端任一命中」的 EXISTS 子查询，把可见集条件
		// 挂在楼宇层，两端用 OR 关联（任一端归属链命中即放行）。
		src := dedicatedLineEndSub(db, "source_room_id")
		dst := dedicatedLineEndSub(db, "dest_room_id")
		cond, args, skip := orgScopeClause(scope, okScope, entityBuildingOrgExpr)
		if skip {
			return append([]string(nil), ids...), 0, nil
		}
		// 两端子查询各自挂同一个可见集条件（cond 引用 b.org_id），再 OR 组合。
		scopedSrc := src.Where(cond, args...)
		scopedDst := dst.Where(cond, args...)
		q := db.WithContext(ctx).Table(table).
			Select("DISTINCT CAST(id AS TEXT)").
			Where("id IN ?", ids).
			Where("(EXISTS (?) OR EXISTS (?))", scopedSrc, scopedDst)
		var visible []string
		if err := q.Pluck("CAST(id AS TEXT)", &visible).Error; err != nil {
			return nil, 0, apperrors.Wrap(err, apperrors.CodeDatabaseError, "批量裁剪可见集失败")
		}
		return visible, len(ids) - len(visible), nil

	default:
		return nil, 0, apperrors.New(apperrors.CodeParamError, "非法的批量裁剪实体")
	}

	cond, args, skip := orgScopeClause(scope, okScope, orgExpr)
	if skip {
		return append([]string(nil), ids...), 0, nil
	}
	// Self 档 workstation-device：按工位 user_id 收窄（sub 内 w.user_id 可引用）；
	// 其余实体 Self 无用户列 → orgScopeClause 已返回 1=0（fail-closed）。
	if entity == scopeEntityWorkstationDevice &&
		scope.ResourceHook == datascope.HookSelfUser && !scope.FailClosed {
		if scope.UserID == "" {
			return []string{}, len(ids), nil
		}
		sub = sub.Where("CAST(w.user_id AS TEXT) = ?", scope.UserID)
	} else {
		sub = sub.Where(cond, args...)
	}
	q := db.WithContext(ctx).Table(table).
		Select("DISTINCT CAST(id AS TEXT)").
		Where("id IN ?", ids).
		Where("EXISTS (?)", sub)
	var visible []string
	if err := q.Pluck("CAST(id AS TEXT)", &visible).Error; err != nil {
		return nil, 0, apperrors.Wrap(err, apperrors.CodeDatabaseError, "批量裁剪可见集失败")
	}
	return visible, len(ids) - len(visible), nil
}
