package v1

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/core"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/utils"
	"github.com/ruoyi-next/ruoyi-go-backend/pkg/response"
	"gorm.io/gorm"
)

const (
	OperTypeOther   = 0
	OperTypeCreate  = 1
	OperTypeUpdate  = 2
	OperTypeDelete  = 3
	OperTypeGrant   = 4
	OperTypeExport  = 5
	OperTypeImport  = 6
	OperTypeForce   = 7
	OperTypeGenCode = 8
	OperTypeClean   = 9
)

func recordOperLog(c *gin.Context, core *core.Core, module string, operType int) {
	clientIP := utils.GetClientIP(c)
	operURL := c.Request.URL.String()
	core.OperLogService.RecordAsync(
		core.GetDB(),
		module,
		operType,
		"",
		c.Request.Method,
		operURL,
		utils.GetUsernamePtr(c),
		utils.GetDeptNameFromDB(c, core.GetDB()),
		&clientIP,
		nil,
		nil,
		nil,
		0,
		0,
	)
}

func getRequiredParam(c *gin.Context, key, name string) (string, bool) {
	val := c.Param(key)
	if val == "" {
		response.Error(c, response.ErrBadRequest, name+"不能为空")
		return "", false
	}
	return val, true
}

// bindAndParseRoleRequest 绑定并解析角色请求参数
func bindAndParseRoleRequest(c *gin.Context) (*struct {
	RoleName  string
	RoleKey   string
	RoleSort  int
	DataScope int
	Status    int
	Remark    *string
	MenuIds   []string
	DeptIds   []string
}, bool) {
	var req struct {
		RoleName  string   `json:"roleName"`
		RoleKey   string   `json:"roleKey"`
		RoleSort  int      `json:"roleSort"`
		DataScope int      `json:"dataScope"`
		Status    int      `json:"status"`
		Remark    *string  `json:"remark"`
		MenuIds   []string `json:"menuIds"`
		DeptIds   []string `json:"deptIds"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		var rawReq map[string]interface{}
		if err2 := c.ShouldBindJSON(&rawReq); err2 != nil {
			response.Error(c, response.ErrBadRequest, "请求参数错误: "+err.Error())
			return nil, false
		}

		if name, ok := rawReq["roleName"]; ok {
			req.RoleName = fmt.Sprintf("%v", name)
		}
		if key, ok := rawReq["roleKey"]; ok {
			req.RoleKey = fmt.Sprintf("%v", key)
		}
		if sort, ok := rawReq["roleSort"]; ok {
			req.RoleSort = utils.ToInt(sort)
		}
		if dataScope, ok := rawReq["dataScope"]; ok {
			req.DataScope = utils.ToInt(dataScope)
		}
		if status, ok := rawReq["status"]; ok {
			req.Status = utils.ParseStatusToInt(status)
		}
		if remark, ok := rawReq["remark"]; ok {
			req.Remark = utils.ToStringPtr(remark)
		}
		if menuIds, ok := rawReq["menuIds"]; ok {
			req.MenuIds = utils.ToStringSlice(menuIds)
		}
		if deptIds, ok := rawReq["deptIds"]; ok {
			req.DeptIds = utils.ToStringSlice(deptIds)
		}
	}

	if req.RoleName == "" {
		response.Error(c, response.ErrBadRequest, "角色名称不能为空")
		return nil, false
	}
	if req.RoleKey == "" {
		response.Error(c, response.ErrBadRequest, "权限字符不能为空")
		return nil, false
	}

	if req.Status != 0 && req.Status != 1 {
		response.Error(c, response.ErrBadRequest, "状态值无效，必须是0(启用)或1(禁用)")
		return nil, false
	}

	result := struct {
		RoleName  string
		RoleKey   string
		RoleSort  int
		DataScope int
		Status    int
		Remark    *string
		MenuIds   []string
		DeptIds   []string
	}{
		RoleName:  req.RoleName,
		RoleKey:   req.RoleKey,
		RoleSort:  req.RoleSort,
		DataScope: req.DataScope,
		Status:    req.Status,
		Remark:    req.Remark,
		MenuIds:   req.MenuIds,
		DeptIds:   req.DeptIds,
	}

	return &result, true
}

// checkRoleNameExists 检查角色名称是否已存在
func checkRoleNameExists(db *gorm.DB, roleName, excludeID string) (bool, error) {
	return utils.CheckExistsExclude(db, &models.Role{}, excludeID, "role_name", roleName)
}

// checkRoleKeyExists 检查权限字符是否已存在
func checkRoleKeyExists(db *gorm.DB, roleKey, excludeID string) (bool, error) {
	return utils.CheckExistsExclude(db, &models.Role{}, excludeID, "role_key", roleKey)
}

// assignRoleMenusAndDepts 分配角色菜单和部门权限
func assignRoleMenusAndDepts(tx *gorm.DB, roleID string, menuIds, deptIds []string) error {
	// 关联菜单权限
	if len(menuIds) > 0 {
		for _, menuId := range menuIds {
			if err := tx.Create(&models.RoleMenu{
				RoleID: roleID,
				MenuID: menuId,
			}).Error; err != nil {
				return fmt.Errorf("关联菜单权限失败: %w", err)
			}
		}
	}

	// 关联部门权限
	if len(deptIds) > 0 {
		for _, deptId := range deptIds {
			if err := tx.Create(&models.RoleDept{
				RoleID: roleID,
				DeptID: deptId,
			}).Error; err != nil {
				return fmt.Errorf("关联部门权限失败: %w", err)
			}
		}
	}

	return nil
}

// updateRoleMenusAndDepts 更新角色菜单和部门权限
func updateRoleMenusAndDepts(tx *gorm.DB, roleID string, menuIds, deptIds []string) error {
	// 删除原有菜单权限关联
	if menuIds != nil {
		if err := tx.Where("role_id = ?", roleID).Delete(&models.RoleMenu{}).Error; err != nil {
			return fmt.Errorf("删除原菜单权限关联失败: %w", err)
		}

		// 重新关联菜单权限
		if len(menuIds) > 0 {
			for _, menuId := range menuIds {
				if err := tx.Create(&models.RoleMenu{
					RoleID: roleID,
					MenuID: menuId,
				}).Error; err != nil {
					return fmt.Errorf("关联菜单权限失败: %w", err)
				}
			}
		}
	}

	// 删除原有部门权限关联
	if deptIds != nil {
		if err := tx.Where("role_id = ?", roleID).Delete(&models.RoleDept{}).Error; err != nil {
			return fmt.Errorf("删除原部门权限关联失败: %w", err)
		}

		// 重新关联部门权限
		if len(deptIds) > 0 {
			for _, deptId := range deptIds {
				if err := tx.Create(&models.RoleDept{
					RoleID: roleID,
					DeptID: deptId,
				}).Error; err != nil {
					return fmt.Errorf("关联部门权限失败: %w", err)
				}
			}
		}
	}

	return nil
}
