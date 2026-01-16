package utils

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// GetUserID 从上下文获取用户ID
func GetUserID(c *gin.Context) string {
	if userID, exists := c.Get("user_id"); exists {
		if userIDStr, ok := userID.(string); ok {
			return userIDStr
		}
	}
	return ""
}

// GetUsername 从上下文获取用户名
func GetUsername(c *gin.Context) string {
	if username, exists := c.Get("username"); exists {
		if usernameStr, ok := username.(string); ok {
			return usernameStr
		}
	}
	return ""
}

// GetUsernamePtr 从上下文获取用户名指针
func GetUsernamePtr(c *gin.Context) *string {
	if username, exists := c.Get("username"); exists {
		if usernameStr, ok := username.(string); ok && usernameStr != "" {
			return &usernameStr
		}
	}
	unknown := "unknown"
	return &unknown
}

// GetDeptName 从上下文获取部门名称
func GetDeptName(c *gin.Context) *string {
	if deptName, exists := c.Get("dept_name"); exists {
		if deptNameStr, ok := deptName.(string); ok && deptNameStr != "" {
			return &deptNameStr
		}
	}
	return nil
}

// GetDeptNameFromDB 从数据库获取部门名称
func GetDeptNameFromDB(c *gin.Context, db *gorm.DB) *string {
	if deptName := GetDeptName(c); deptName != nil {
		return deptName
	}

	userID := GetUserID(c)
	if userID == "" {
		return nil
	}

	var deptName string
	if err := db.Table("sys_user").
		Select("dept_name").
		Where("id = ?", userID).
		Scan(&deptName).Error; err == nil && deptName != "" {
		return &deptName
	}

	return nil
}

// GetClientIP 获取客户端IP
func GetClientIP(c *gin.Context) string {
	return c.ClientIP()
}

// GetRequiredParam 获取必需的路径参数
func GetRequiredParam(c *gin.Context, key, name string) (string, bool) {
	val := c.Param(key)
	if val == "" {
		return "", false
	}
	return val, true
}
