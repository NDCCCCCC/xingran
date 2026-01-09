package v1

import (
	"github.com/gin-gonic/gin"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
	"gorm.io/gorm"
)

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
