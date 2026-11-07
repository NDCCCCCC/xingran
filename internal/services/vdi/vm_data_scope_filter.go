package vdi

import (
	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/pkg/constants"
	"github.com/xingran-next/xingran-go-backend/pkg/datascope"
	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
	"gorm.io/gorm"
)

// ApplyVMScopeResult 是 v1.52 Phase 133 迁移后的「ScopeResult 单一真实源」过滤入口。
//
// 与旧 ApplyVMDataScopeFilter 的语义逐档对齐（LOCKED 裁定
// .planning/notes/261107-v152-self-scope-decision.md §4）：
//   - All / 0 / unset：不过滤（超管零回归；DataScopePermission 对 0 注入 AllScope）。
//   - Custom / Dept / DeptChild：DeptIDs 已由 datascope.Resolve 展开，按
//     bound_user_id IN (SELECT id FROM sys_user WHERE dept_id IN (?)) 过滤；
//     空 DeptIDs → 1=0（fail-closed）。
//   - Self（HookSelfUser）：bound_user_id = UserID（ResourceHook 语义，与部门集不等价）。
//   - FailClosed（异常枚举）：1=0。
//   - 非 All 档统一追加 bound_user_id IS NOT NULL（等价旧 ApplyBoundUserFilter）。
//
// 参数 scope 来自 datascope.FromContext(ctx)；ok=false（未挂注入中间件）时调用方
// 应回落旧 ApplyVMDataScopeFilter 显式参数路径，本函数不处理 ok=false。
// 本函数为纯函数（不查库、不读 ctx），便于「新旧实现迁移对照测试」逐档比对。
func ApplyVMScopeResult(query *gorm.DB, scope datascope.ScopeResult) *gorm.DB {
	// All 档特判短路：不过滤、不追加 IS NOT NULL（超管/0/unset 零回归）。
	if scope.All {
		return query
	}

	// 非 All 档：先按档位挂主体条件，再统一追加「排除未绑定用户」。
	bounded := func(q *gorm.DB) *gorm.DB {
		switch {
		case scope.FailClosed:
			return q.Where("1=0")
		case scope.ResourceHook == datascope.HookSelfUser:
			// Self：部门集为空，按 bound_user_id = 登录人（UserID 缺失则 fail-closed）。
			if scope.UserID == "" {
				return q.Where("1=0")
			}
			return q.Where("bound_user_id = ?", scope.UserID)
		case len(scope.DeptIDs) == 0:
			// Custom/Dept/DeptChild 但无可见部门 → fail-closed 空集。
			return q.Where("1=0")
		default:
			return q.Where(
				"bound_user_id IN (SELECT id FROM sys_user WHERE dept_id IN (?))",
				scope.DeptIDs,
			)
		}
	}(query)

	return bounded.Where("bound_user_id IS NOT NULL")
}

// isValidUUID 验证字符串是否符合UUID格式
func isValidUUID(id string) bool {
	return constants.UUIDPattern.MatchString(id)
}

// ApplyVMDataScopeFilter 应用虚拟机数据范围过滤
// 根据 bound_user_id 字段过滤虚拟机数据，支持5种数据范围
func ApplyVMDataScopeFilter(query *gorm.DB, userID string, dataScope models.DataScope, db *gorm.DB) *gorm.DB {
	// Validate userID format
	if userID == "" || !isValidUUID(userID) {
		applogger.Errorf("Invalid userID format for data scope filtering: %s", userID)
		return query.Where("1=0")
	}
	switch dataScope {
	case models.DataScopeAll:
		// 全部数据权限，不做过滤
		return query

	case models.DataScopeCustom:
		// 自定义数据权限：通过 sys_role_dept 表过滤
		// 查询用户可访问的部门，然后过滤 bound_user_id 属于这些部门的虚拟机
		var deptIds []string
		err := db.Raw(`
			SELECT DISTINCT rd.dept_id
			FROM sys_user_role ur
			INNER JOIN sys_role_dept rd ON ur.role_id = rd.role_id
			WHERE ur.user_id = ?
		`, userID).Scan(&deptIds).Error

		if err != nil {
			applogger.Errorf("Failed to query custom departments for data scope filtering (user_id=%s): %v", userID, err)
			return query.Where("1=0")
		}

		if len(deptIds) > 0 {
			// 过滤 bound_user_id 属于这些部门的虚拟机
			return query.Where("bound_user_id IN (SELECT id FROM sys_user WHERE dept_id IN (?))", deptIds)
		}
		return query.Where("1=0")

	case models.DataScopeDept:
		// 本部门数据权限：过滤 bound_user_id 所属部门与当前用户相同的虚拟机
		var deptId string
		err := db.Raw("SELECT dept_id FROM sys_user WHERE id = ?", userID).Scan(&deptId).Error
		if err != nil {
			if err != gorm.ErrRecordNotFound {
				applogger.Errorf("Failed to query user dept for data scope filtering (user_id=%s): %v", userID, err)
			}
			return query.Where("1=0")
		}

		if deptId != "" {
			return query.Where("bound_user_id IN (SELECT id FROM sys_user WHERE dept_id = ?)", deptId)
		}
		return query.Where("1=0")

	case models.DataScopeDeptChild:
		// 本部门及子部门数据权限：过滤 bound_user_id 属于当前部门及其子部门的虚拟机
		var deptId string
		err := db.Raw("SELECT dept_id FROM sys_user WHERE id = ?", userID).Scan(&deptId).Error
		if err != nil {
			if err != gorm.ErrRecordNotFound {
				applogger.Errorf("Failed to query user dept for data scope filtering (user_id=%s): %v", userID, err)
			}
			return query.Where("1=0")
		}

		if deptId != "" {
			// 查询本部门及所有子部门
			var childDeptIds []string
			childDeptIds = append(childDeptIds, deptId)
			getChildDepts(db, deptId, &childDeptIds)

			return query.Where("bound_user_id IN (SELECT id FROM sys_user WHERE dept_id IN (?))", childDeptIds)
		}
		return query.Where("1=0")

	case models.DataScopeSelf:
		// 仅本人数据权限：只过滤 bound_user_id 等于当前用户的虚拟机
		return query.Where("bound_user_id = ?", userID)

	default:
		return query.Where("1=0")
	}
}

// ApplyBoundUserFilter 应用无绑定用户过滤规则
// 无绑定用户的虚拟机（bound_user_id IS NULL）仅对 DataScopeAll 可见
func ApplyBoundUserFilter(query *gorm.DB, dataScope models.DataScope) *gorm.DB {
	if dataScope != models.DataScopeAll {
		// 非 DataScopeAll 的查询自动过滤掉无绑定用户的虚拟机
		return query.Where("bound_user_id IS NOT NULL")
	}
	return query
}

// getChildDepts 递归查询所有子部门（复用自 permission.go 的逻辑）
func getChildDepts(db *gorm.DB, parentId string, deptIds *[]string) {
	var depts []struct {
		ID string
	}

	err := db.Raw("SELECT id FROM sys_dept WHERE parent_id = ? AND status = ?", parentId, 0).Scan(&depts).Error
	if err != nil {
		return
	}

	for _, dept := range depts {
		*deptIds = append(*deptIds, dept.ID)
		getChildDepts(db, dept.ID, deptIds)
	}
}
