package vdi

import (
	"github.com/xingran-next/xingran-go-backend/internal/constants"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
	"gorm.io/gorm"
)

// isValidUUID 验证字符串是否符合UUID格式
func isValidUUID(id string) bool {
	return constants.UuidPattern.MatchString(id)
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
