package v1

import (
	"github.com/ruoyi-next/ruoyi-go-backend/pkg/response"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/core"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
)

// ==================== 通用请求类型 ====================

// UserStatusRequest 用户状态更新请求
type UserStatusRequest struct {
	ID     string `json:"id" binding:"required"`
	Status int    `json:"status" binding:"required"`
}

// ==================== 辅助函数 ====================

// getIntField 从map中获取整数字段，提供默认值
func getIntField(m map[string]interface{}, key string, defaultValue int) int {
	if val, ok := m[key]; ok {
		switch v := val.(type) {
		case float64:
			return int(v)
		case int:
			return v
		case string:
			return parseInt(v)
		}
	}
	return defaultValue
}

// getStringField 从map中获取字符串字段，提供默认值
func getStringField(m map[string]interface{}, key string, defaultValue string) string {
	if val, ok := m[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return defaultValue
}

// getBoolField 从map中获取布尔字段，提供默认值
func getBoolField(m map[string]interface{}, key string, defaultValue bool) bool {
	if val, ok := m[key]; ok {
		if b, ok := val.(bool); ok {
			return b
		}
	}
	return defaultValue
}

// parseInt 解析字符串为整数
func parseInt(s string) int {
	i, _ := strconv.Atoi(s)
	return i
}

// ==================== 用户相关辅助函数 ====================

// checkUserExists 检查用户是否存在
func checkUserExists(db *core.Core, userID string) (*models.User, bool) {
	var user models.User
	if err := db.GetDB().First(&user, "id = ?", userID).Error; err != nil {
		if err == response.ErrRecordNotFound {
			return nil, false
		}
		return nil, false
	}
	return &user, true
}

// getUserFromContext 从上下文获取用户信息
func getUserFromContext(c *gin.Context) (userID, username string, exists bool) {
	userIDVal, userIDExists := c.Get("user_id")
	usernameVal, usernameExists := c.Get("username")
	if !userIDExists || !usernameExists {
		return "", "", false
	}
	return userIDVal.(string), usernameVal.(string), true
}
