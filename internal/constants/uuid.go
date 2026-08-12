// UUID 相关常量:统一管理 UUID 验证相关的正则表达式,避免在多个包内重复定义。

package constants

import "regexp"

// UUIDPattern 匹配标准 8-4-4-4-12 格式的 UUID(十六进制小写)。
//
// 作为项目内 UUID 格式校验的唯一来源;任何新代码需要校验 UUID 格式
// 时都应使用本变量,而非自行重新编译正则。
//
// 使用方式:
//
//	if !constants.UUIDPattern.MatchString(id) {
//	    // id 不是合法 UUID
//	}
var UUIDPattern = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
