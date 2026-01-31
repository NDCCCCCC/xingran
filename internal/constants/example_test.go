/**
 * 常量使用示例
 *
 * 展示如何在项目中使用定义的常量
 */
package constants

import (
	"fmt"
	"time"
)

// ExampleDefaultCacheExpire 展示缓存过期时间常量的使用
func ExampleDefaultCacheExpire() {
	// 使用缓存过期时间常量
	cacheExpire := DefaultCacheExpire // 5分钟
	_ = cacheExpire
}

// ExampleDefaultHTTPTimeout 展示HTTP超时时间常量的使用
func ExampleDefaultHTTPTimeout() {
	// 使用超时时间常量
	timeout := DefaultHTTPTimeout // 30秒
	_ = timeout
}

// ExampleOneDay 展示一天时间常量的使用
func ExampleOneDay() {
	// 使用一天时间常量
	today := time.Now().Truncate(OneDay)
	_ = today
}

// ExampleTokenBlacklistKeyFormat 展示缓存键格式常量的使用
func ExampleTokenBlacklistKeyFormat() {
	token := "abc123"
	// 使用常量格式化缓存键
	blacklistKey := fmt.Sprintf(TokenBlacklistKeyFormat, token)
	_ = blacklistKey
}

// ExampleDefaultPageSize 展示默认分页大小常量的使用
func ExampleDefaultPageSize() {
	// 使用默认分页大小
	pageSize := DefaultPageSize // 10
	_ = pageSize
}

// ExampleMaxPageSize 展示最大分页大小常量的使用
func ExampleMaxPageSize() {
	// 使用最大分页大小
	maxPageSize := MaxPageSize // 100
	_ = maxPageSize
}

// ExampleUserStatusEnabled 展示用户状态常量的使用
func ExampleUserStatusEnabled() {
	// 使用用户状态常量
	userStatus := UserStatusEnabled // 0
	_ = userStatus
}

// ExampleRoleStatusDisabled 展示角色状态常量的使用
func ExampleRoleStatusDisabled() {
	// 使用角色状态常量
	roleStatus := RoleStatusDisabled // 1
	_ = roleStatus
}
