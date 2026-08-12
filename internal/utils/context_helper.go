package utils

import (
	"database/sql"

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

// GetNicknamePtr 从上下文获取用户昵称指针
// 优先从 context("nickname") 读(由 auth 中间件 c.Set),如果 context 缺值或为空,
// 调用方应使用 GetNicknamePtrWithDB(c, db) 走 DB 兜底(常见于老 token / 跨服务调用场景)
func GetNicknamePtr(c *gin.Context) *string {
	if nickname, exists := c.Get("nickname"); exists {
		if nicknameStr, ok := nickname.(string); ok && nicknameStr != "" {
			return &nicknameStr
		}
	}
	return nil
}

// GetNicknamePtrWithDB 从上下文获取用户昵称指针,context 缺值时回查 sys_user.nickname
// 设计原因:老 token / 跨服务调用场景下 c.Get("nickname") 可能为 ""(老 token 没存该字段),
// 直接写库会存 NULL,日志页面就只显示 username。DB 兜底保证 nickname 链路对历史 token 也生效。
func GetNicknamePtrWithDB(c *gin.Context, db *gorm.DB) *string {
	if ptr := GetNicknamePtr(c); ptr != nil {
		return ptr
	}
	if db == nil {
		return nil
	}
	userID := GetUserID(c)
	if userID == "" {
		return nil
	}
	var nickname sql.NullString
	if err := db.Table("sys_user").
		Select("nickname").
		Where("id = ?", userID).
		Scan(&nickname).Error; err == nil && nickname.Valid && nickname.String != "" {
		return &nickname.String
	}
	return nil
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
//
// 关键修复:F-03 — 必须使用 sql.NullString 而不是 string。
// 真实崩溃:PostgreSQL 返回 NULL 时(例如 sys_user.dept_name 是 NULL 字段,
// 或系统/超管用户没有部门),Go database/sql 无法把 NULL 转换给 string,
// 抛 "converting NULL to string is unsupported" 错误。
// 复现:userID='8bd62962-...' 是个 NULL dept_name 的用户,导致后续
// operlog middleware 写入失败 → handler 返回 500。
//
// 此函数在 oper_log middleware + operlog.Record 热路径上被调用,
// 每次写请求都会触发,所以 NULL 兼容是必需的(系统用户/超管用户常见)。
func GetDeptNameFromDB(c *gin.Context, db *gorm.DB) *string {
	if deptName := GetDeptName(c); deptName != nil {
		return deptName
	}

	userID := GetUserID(c)
	if userID == "" {
		return nil
	}

	// 关键:必须能接受 NULL。sys_user.dept_name 是 *string(nullable),
	// 某些用户(系统/超管/未分配部门的用户)的 dept_name 为 NULL。
	var deptName sql.NullString
	if err := db.Table("sys_user").
		Select("dept_name").
		Where("id = ?", userID).
		Scan(&deptName).Error; err == nil && deptName.Valid && deptName.String != "" {
		// .Valid 区分"SQL 返回 NULL"和"SQL 返回空字符串"
		// .String 为 "" 也视为没有值,避免上游出现 "未知部门" 空标签
		return &deptName.String
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
