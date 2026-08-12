package system

import (
	"github.com/gin-gonic/gin"
	"github.com/xingran-next/xingran-go-backend/internal/core"
	apperrors "github.com/xingran-next/xingran-go-backend/pkg/errors"
	"github.com/xingran-next/xingran-go-backend/pkg/response"
)

// FixMenuPathsHandler 修复菜单路径的临时API
func FixMenuPathsHandler(c *gin.Context, core *core.Core) {
	db := core.DB.GetDB()

	// 1. 获取用户中心父菜单ID
	var userCenterID string
	err := db.Raw("SELECT id FROM sys_menu WHERE menu_name = '用户中心' AND parent_id IS NULL LIMIT 1").Scan(&userCenterID).Error
	if err != nil {
		response.Error(c, apperrors.InternalServerErrorWithMsg("用户中心菜单不存在"))
		return
	}

	if userCenterID == "" {
		response.Error(c, apperrors.NotFound("用户中心父菜单不存在"))
		return
	}

	// 2. 修复个人中心路径
	result1 := db.Exec(`
		UPDATE sys_menu
		SET path = 'user/profile'
		WHERE menu_name = '个人中心'
		  AND parent_id = ?
		  AND path = 'profile'
	`, userCenterID)

	// 3. 修复系统设置路径
	result2 := db.Exec(`
		UPDATE sys_menu
		SET path = 'user/settings'
		WHERE menu_name = '系统设置'
		  AND parent_id = ?
		  AND path = 'settings'
	`, userCenterID)

	// 4. 修复"我的通知"路径
	result3 := db.Exec(`
		UPDATE sys_menu
		SET path = 'user/my-notices'
		WHERE menu_name = '我的通知'
		  AND parent_id = ?
		  AND path = 'my-notices'
	`, userCenterID)

	// 5. 更新组件路径
	db.Exec(`
		UPDATE sys_menu
		SET component = 'profile/index'
		WHERE menu_name = '个人中心'
		  AND parent_id = ?
		  AND component IS NULL
	`, userCenterID)

	db.Exec(`
		UPDATE sys_menu
		SET component = 'system/settings-page/index'
		WHERE menu_name = '系统设置'
		  AND parent_id = ?
		  AND component IS NULL
	`, userCenterID)

	db.Exec(`
		UPDATE sys_menu
		SET component = 'my-notices/index'
		WHERE menu_name = '我的通知'
		  AND parent_id = ?
		  AND component IS NULL
	`, userCenterID)

	// 6. 查询修复结果
	type MenuInfo struct {
		MenuName  string `json:"menu_name"`
		Path      string `json:"path"`
		Component string `json:"component"`
		ParentID  string `json:"parent_id"`
	}

	var menus []MenuInfo
	db.Raw(`
		SELECT menu_name, path, component, parent_id
		FROM sys_menu
		WHERE menu_name IN ('用户中心', '个人中心', '系统设置', '我的通知')
		ORDER BY
			CASE WHEN menu_name = '用户中心' THEN 0
				 WHEN menu_name = '个人中心' THEN 1
					WHEN menu_name = '系统设置' THEN 2
					WHEN menu_name = '我的通知' THEN 3
			END
	`).Scan(&menus)

	response.Success(c, gin.H{
		"message": "菜单路径修复成功",
		"fixed": gin.H{
			"个人中心": result1.RowsAffected > 0,
			"系统设置": result2.RowsAffected > 0,
			"我的通知": result3.RowsAffected > 0,
		},
		"menus": menus,
	})
}
