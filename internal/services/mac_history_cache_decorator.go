package services

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
)

// 缓存键前缀常量 (D-13 锁定)
// 注意: Core 在缓存层自动加 xingran: 前缀, 这里返回的 key 不带前缀
const (
	cacheKeyPrefixPortHistory   = "mac:query:port-history"
	cacheKeyPrefixDeviceHistory = "mac:query:device-history"
	cacheKeyPrefixStats         = "mac:query:stats"
	cacheKeyPrefixHeatmap       = "mac:query:heatmap"
)

// BuildMACQueryCacheKey 构造缓存键: <prefix>:<sha256(params)>
// params 必须 JSON 序列化稳定 (struct 字段顺序稳定, Go encoding/json 保证)
func BuildMACQueryCacheKey(method string, params interface{}) (string, error) {
	var prefix string
	switch method {
	case "port-history":
		prefix = cacheKeyPrefixPortHistory
	case "device-history":
		prefix = cacheKeyPrefixDeviceHistory
	case "stats":
		prefix = cacheKeyPrefixStats
	case "heatmap":
		prefix = cacheKeyPrefixHeatmap
	default:
		return "", fmt.Errorf("未知方法: %s", method)
	}

	data, err := json.Marshal(params)
	if err != nil {
		return "", fmt.Errorf("序列化缓存参数失败: %w", err)
	}
	sum := sha256.Sum256(data)
	return fmt.Sprintf("%s:%s", prefix, hex.EncodeToString(sum[:])), nil
}
