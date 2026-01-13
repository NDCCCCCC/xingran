package v1

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/core"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
	"github.com/ruoyi-next/ruoyi-go-backend/pkg/response"
	"gorm.io/gorm"
)

// 操作类型常量
const (
	OperTypeOther   = 0 // 其他
	OperTypeCreate  = 1 // 新增
	OperTypeUpdate  = 2 // 修改
	OperTypeDelete  = 3 // 删除
	OperTypeGrant   = 4 // 授权
	OperTypeExport  = 5 // 导出
	OperTypeImport  = 6 // 导入
	OperTypeForce   = 7 // 强退
	OperTypeGenCode = 8 // 生成代码
	OperTypeClean   = 9 // 清空数据
)

// recordOperLog 记录操作日志（简化版，减少重复代码）
func recordOperLog(c *gin.Context, core *core.Core, module string, operType int) {
	clientIP := c.ClientIP()
	operURL := c.Request.URL.String()
	core.OperLogService.RecordAsync(
		core.GetDB(),
		module,
		operType,
		"",
		c.Request.Method,
		operURL,
		getUsernameFromContext(c),
		getDeptNameFromContext(c, core.GetDB()),
		&clientIP,
		nil,
		nil,
		nil,
		0,
		0,
	)
}

// getRequiredParam 获取必需的路径参数，如果为空则返回错误
func getRequiredParam(c *gin.Context, key, name string) (string, bool) {
	val := c.Param(key)
	if val == "" {
		response.Error(c, response.ErrBadRequest, name+"不能为空")
		return "", false
	}
	return val, true
}

// ========== 通用类型转换辅助函数 ==========

// toInt 从 interface{} 转换为 int
func toInt(v interface{}) int {
	switch val := v.(type) {
	case float64:
		return int(val)
	case int:
		return val
	case string:
		if i, err := strconv.Atoi(val); err == nil {
			return i
		}
	}
	return 0
}

// toBoolPtr 从 interface{} 转换为 *bool
func toBoolPtr(v interface{}) *bool {
	switch val := v.(type) {
	case bool:
		return &val
	case string:
		if b, err := strconv.ParseBool(val); err == nil {
			return &b
		}
	}
	return nil
}

// toIntPtr 从 interface{} 转换为 *int
func toIntPtr(v interface{}) *int {
	i := toInt(v)
	if i != 0 {
		return &i
	}
	return nil
}

// toStringPtr 从 interface{} 转换为 *string
func toStringPtr(v interface{}) *string {
	if v == nil {
		return nil
	}
	s := fmt.Sprintf("%v", v)
	if s != "" {
		return &s
	}
	return nil
}

// toStringSlice 从 interface{} 转换为 []string
func toStringSlice(v interface{}) []string {
	var result []string
	switch val := v.(type) {
	case []interface{}:
		for _, item := range val {
			result = append(result, fmt.Sprintf("%v", item))
		}
	case []string:
		return val
	case string:
		if val != "" {
			// 处理逗号分隔的字符串
			parts := strings.Split(val, ",")
			for _, part := range parts {
				if trimmed := strings.TrimSpace(part); trimmed != "" {
					result = append(result, trimmed)
				}
			}
		}
	}
	return result
}

// parseStatusToInt 从 interface{} 解析状态值
// 支持: float64, int, string, bool (true=0启用, false=1禁用)
func parseStatusToInt(v interface{}) int {
	switch val := v.(type) {
	case float64:
		return int(val)
	case int:
		return val
	case string:
		if i, err := strconv.Atoi(val); err == nil {
			return i
		}
	case bool:
		if val {
			return 0 // 启用
		}
		return 1 // 禁用
	}
	return 0
}

// getUsernameFromContext 从上下文获取当前用户名
func getUsernameFromContext(c *gin.Context) *string {
	if username, exists := c.Get("username"); exists {
		if usernameStr, ok := username.(string); ok && usernameStr != "" {
			return &usernameStr
		}
	}
	unknown := "unknown"
	return &unknown
}

// getDeptNameFromContext 从上下文获取当前用户部门名称
// 如果上下文中没有部门信息，则从数据库查询
func getDeptNameFromContext(c *gin.Context, db *gorm.DB) *string {
	// 首先尝试从上下文获取
	if deptName, exists := c.Get("dept_name"); exists {
		if deptNameStr, ok := deptName.(string); ok && deptNameStr != "" {
			return &deptNameStr
		}
	}

	// 从数据库查询用户部门
	if userID, exists := c.Get("user_id"); exists {
		if userIDStr, ok := userID.(string); ok && userIDStr != "" {
			var user models.User
			if err := db.Select("dept_name").Where("id = ?", userIDStr).First(&user).Error; err == nil {
				if user.DeptName != nil && *user.DeptName != "" {
					return user.DeptName
				}
			}
		}
	}

	return nil
}

// ========== 通用切片/数组转换辅助函数 ==========

// ToSlicePtr 将值切片转换为指针切片（泛型版本）
// 示例: ToSlicePtr([]models.Role{...}) => []*models.Role{...}
func ToSlicePtr[T any](slice []T) []*T {
	result := make([]*T, len(slice))
	for i := range slice {
		result[i] = &slice[i]
	}
	return result
}

// ToSlice 从指针切片转换为值切片（泛型版本）
// 示例: ToSlice([]*models.Role{...}) => []models.Role{...}
func ToSlice[T any](ptrs []*T) []T {
	result := make([]T, 0, len(ptrs))
	for _, ptr := range ptrs {
		if ptr != nil {
			result = append(result, *ptr)
		}
	}
	return result
}

// derefString 安全地解引用字符串指针
func derefString(s *string) string {
	if s != nil {
		return *s
	}
	return ""
}

// bindAndParseRoleRequest 绑定并解析角色请求参数
func bindAndParseRoleRequest(c *gin.Context) (*struct {
	RoleName string
	RoleKey  string
	RoleSort int
	DataScope int
	Status    int
	Remark    *string
	MenuIds   []string
	DeptIds   []string
}, bool) {
	var req struct {
		RoleName string   `json:"roleName"`
		RoleKey  string   `json:"roleKey"`
		RoleSort int      `json:"roleSort"`
		DataScope int     `json:"dataScope"`
		Status   int      `json:"status"`
		Remark   *string  `json:"remark"`
		MenuIds  []string `json:"menuIds"`
		DeptIds  []string `json:"deptIds"`
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
			req.RoleSort = toInt(sort)
		}
		if dataScope, ok := rawReq["dataScope"]; ok {
			req.DataScope = toInt(dataScope)
		}
		if status, ok := rawReq["status"]; ok {
			req.Status = parseStatusToInt(status)
		}
		if remark, ok := rawReq["remark"]; ok {
			req.Remark = toStringPtr(remark)
		}
		if menuIds, ok := rawReq["menuIds"]; ok {
			req.MenuIds = toStringSlice(menuIds)
		}
		if deptIds, ok := rawReq["deptIds"]; ok {
			req.DeptIds = toStringSlice(deptIds)
		}
	}

	// 验证必填字段
	if req.RoleName == "" {
		response.Error(c, response.ErrBadRequest, "角色名称不能为空")
		return nil, false
	}
	if req.RoleKey == "" {
		response.Error(c, response.ErrBadRequest, "权限字符不能为空")
		return nil, false
	}

	// 验证状态值
	if req.Status != 0 && req.Status != 1 {
		response.Error(c, response.ErrBadRequest, "状态值无效，必须是0(启用)或1(禁用)")
		return nil, false
	}

	result := struct {
		RoleName string
		RoleKey  string
		RoleSort int
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
	var count int64
	query := db.Model(&models.Role{}).Where("role_name = ?", roleName)
	if excludeID != "" {
		query = query.Where("id != ?", excludeID)
	}
	if err := query.Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

// checkRoleKeyExists 检查权限字符是否已存在
func checkRoleKeyExists(db *gorm.DB, roleKey, excludeID string) (bool, error) {
	var count int64
	query := db.Model(&models.Role{}).Where("role_key = ?", roleKey)
	if excludeID != "" {
		query = query.Where("id != ?", excludeID)
	}
	if err := query.Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
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
