// 常量使用示例:展示如何在项目中使用定义的常量。

package constants

import (
	"fmt"
	"time"
)

// ExampleDefaultCacheExpire 展示缓存过期时间常量的使用
func ExampleDefaultCacheExpire() {
	cacheExpire := DefaultCacheExpire // 5分钟
	_ = cacheExpire
}

// ExampleDefaultHTTPTimeout 展示HTTP超时时间常量的使用
func ExampleDefaultHTTPTimeout() {
	timeout := DefaultHTTPTimeout // 30秒
	_ = timeout
}

// ExampleOneDay 展示一天时间常量的使用
func ExampleOneDay() {
	// OneDay 可用于计算"距今 N 天"的截止时间
	deadline := time.Now().Add(OneDay)
	_ = deadline
}

// ExampleTokenBlacklistKeyFormat 展示缓存键格式常量的使用
func ExampleTokenBlacklistKeyFormat() {
	token := "abc123"
	blacklistKey := fmt.Sprintf(TokenBlacklistKeyFormat, token)
	_ = blacklistKey
}

// ExampleDefaultPageSize 展示默认分页大小常量的使用
func ExampleDefaultPageSize() {
	pageSize := DefaultPageSize // 10
	_ = pageSize
}

// ExampleMaxListPageSize 展示列表分页上限常量的使用
func ExampleMaxListPageSize() {
	maxPageSize := MaxListPageSize // 100
	_ = maxPageSize
}
