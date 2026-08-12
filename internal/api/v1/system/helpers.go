package system

import "strconv"

// parseInt 安全地将字符串转换为整数
func parseInt(s string) int {
	if i, err := strconv.Atoi(s); err == nil {
		return i
	}
	return 0
}
