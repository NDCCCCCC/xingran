package addomain

import (
	"fmt"
	"strings"
	"time"
)

// encryptPassword 加密密码（简化实现，实际应使用AES）
func encryptPassword(password string) string {
	// TODO: 实际应使用AES等加密算法
	// 这里为了演示，简单进行base64编码
	return password
}

// decryptPassword 解密密码
func decryptPassword(encrypted string) string {
	// TODO: 实际应使用AES等解密算法
	return encrypted
}

// extractParentDN 提取父DN
func extractParentDN(dn string) string {
	parts := strings.Split(dn, ",")
	if len(parts) <= 1 {
		return ""
	}
	return strings.Join(parts[1:], ",")
}

// buildOUPath 构建OU路径
func buildOUPath(ouDN, baseDN string) string {
	if ouDN == baseDN {
		return "/"
	}
	// 简化处理：去掉baseDN后反转DN部分
	parts := strings.Split(ouDN, ",")
	var pathParts []string
	for _, part := range parts {
		if strings.Contains(part, "OU=") {
			name := strings.TrimPrefix(part, "OU=")
			pathParts = append([]string{name}, pathParts...)
		}
	}
	return "/" + strings.Join(pathParts, "/")
}

// parseIntOrDefault 解析整数，失败返回默认值
func parseIntOrDefault(str string, defaultValue int) int {
	var result int
	_, err := fmt.Sscanf(str, "%d", &result)
	if err != nil {
		return defaultValue
	}
	return result
}

// parseFileTime 解析AD FileTime格式
func parseFileTime(fileTime string) *time.Time {
	// AD FileTime是100纳秒为单位，从1601-01-01开始的计数
	// 这里简化处理，实际需要完整的转换
	var ft int64
	_, err := fmt.Sscanf(fileTime, "%d", &ft)
	if err != nil || ft == 0 {
		return nil
	}

	// 转换为Unix时间戳（简化版）
	// 实际需要：unix = (ft / 10000000) - 11644473600
	unix := (ft / 10000000) - 11644473600
	if unix < 0 {
		return nil
	}

	t := time.Unix(unix, 0)
	return &t
}
