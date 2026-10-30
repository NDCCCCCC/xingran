// Package middleware 提供请求解密中间件
package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/xingran-next/xingran-go-backend/pkg/crypto"
	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
	"github.com/xingran-next/xingran-go-backend/pkg/response"
	"gorm.io/gorm"
)

const (
	// maxBodySize 最大请求体大小（10MB）
	maxBodySize = 10 << 20
	// cacheTTL 配置缓存TTL（30秒）
	cacheTTL = 30 * time.Second
	// encryptionConfigKey 请求加密开关配置键
	encryptionConfigKey = "sys.request.encryption.enabled"
)

// RequestDecryptionConfig 解密配置
type RequestDecryptionConfig struct {
	Enabled           bool     // 是否启用
	ExcludePaths      []string // 排除路径（支持通配符）
	RequireEncryption bool     // 是否强制加密
}

// configCache 配置缓存
type configCache struct {
	value      bool
	lastUpdate time.Time
	mu         sync.RWMutex
}

// globalConfigCache 全局配置缓存实例
var globalConfigCache = &configCache{
	value:      true, // 默认启用
	lastUpdate: time.Time{},
}

// RequestDecryption 创建请求解密中间件
// 在 JWT 认证之前执行，解密请求体后替换原始请求体
// 支持从数据库动态读取加密开关配置
func RequestDecryption(encryptor *crypto.RequestEncryptor, staticConfig *RequestDecryptionConfig, db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 从数据库获取加密开关配置（带缓存）
		enabled := getConfigFromDB(c.Request.Context(), db, staticConfig.Enabled)

		// 检查是否启用加密
		if !enabled {
			c.Next()
			return
		}

		// 检查是否在排除列表中
		if isExcludedPath(c.Request.URL.Path, staticConfig.ExcludePaths) {
			c.Next()
			return
		}

		// 只处理 POST/PUT/PATCH
		if c.Request.Method != "POST" && c.Request.Method != "PUT" && c.Request.Method != "PATCH" {
			c.Next()
			return
		}

		// 跳过文件上传
		contentType := c.GetHeader("Content-Type")
		if strings.Contains(contentType, "multipart/form-data") {
			c.Next()
			return
		}

		// 检查请求体大小
		if c.Request.ContentLength > maxBodySize {
			response.Error(c, response.ErrBadRequest, "请求体过大")
			c.Abort()
			return
		}

		// 读取请求体（带大小限制）
		bodyBytes, err := io.ReadAll(io.LimitReader(c.Request.Body, maxBodySize+1))
		if err != nil {
			response.Error(c, response.ErrBadRequest, "读取请求体失败")
			c.Abort()
			return
		}

		// 检查是否超过限制
		if len(bodyBytes) > maxBodySize {
			response.Error(c, response.ErrBadRequest, "请求体过大")
			c.Abort()
			return
		}

		// 空请求体跳过
		if len(bodyBytes) == 0 {
			c.Next()
			return
		}

		// 检查是否为加密请求
		var rawMap map[string]interface{}
		if err := json.Unmarshal(bodyBytes, &rawMap); err != nil {
			// JSON 解析失败，可能不是加密请求，直接返回
			c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
			c.Next()
			return
		}

		// 检查 encrypted 标识
		encrypted, hasEncrypted := rawMap["encrypted"].(bool)
		if !hasEncrypted || !encrypted {
			// 未加密请求
			if staticConfig.RequireEncryption {
				response.Error(c, response.ErrBadRequest, "请求体必须加密")
				c.Abort()
				return
			}
			// 兼容模式：恢复请求体，继续处理
			c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
			c.Next()
			return
		}

		// 解析加密请求
		var encReq crypto.EncryptedRequest
		if err := json.Unmarshal(bodyBytes, &encReq); err != nil {
			applogger.WithFields(map[string]interface{}{
				"path":  c.Request.URL.Path,
				"error": err,
			}).Warn("解析加密请求失败")
			response.Error(c, response.ErrBadRequest, "加密请求格式错误")
			c.Abort()
			return
		}

		// 解密请求体（获取 SM4 密钥和 IV 用于响应加密）
		decryptedData, sm4Key, iv, err := encryptor.DecryptRequestWithKeyInfo(&encReq)
		if err != nil {
			applogger.WithFields(map[string]interface{}{
				"path":        c.Request.URL.Path,
				"method":      c.Request.Method,
				"timestamp":   encReq.Timestamp,
				"nonce":       encReq.Nonce,
				"data_length": len(encReq.Data),
				"error":       err,
			}).Warn("解密请求体失败")
			// F-06: 不向客户端泄露具体解密错误 (key/iv/格式细节会帮助攻击者
			// 探测密钥结构),改为返回通用错误消息;真实错误仅在 server 日志查询。
			response.Error(c, response.ErrBadRequest, "解密失败")
			c.Abort()
			return
		}

		// 验证解密后的数据是否为有效 JSON
		if !json.Valid(decryptedData) {
			applogger.WithField("path", c.Request.URL.Path).
				Warn("解密数据格式无效")
			response.Error(c, response.ErrBadRequest, "解密数据格式无效")
			c.Abort()
			return
		}

		// F-05: 已移除 [DECRYPTION DEBUG] 日志块 —
		// 原代码将 string(decryptedData) 作为 decrypted_data_preview 写入 INFO 日志,
		// 该数据是包含明文密码等敏感字段的完整解密请求体,会被持久化到日志文件,
		// 构成密码等敏感数据落盘的严重安全风险。
		// 下方 line 187+ 的"请求解密成功"日志只记录非敏感元数据,符合最小化原则。

		// 替换请求体
		c.Request.Body = io.NopCloser(bytes.NewBuffer(decryptedData))

		// 存储 SM4 密钥和 IV 到 gin.Context，供响应加密使用
		c.Set("sm4_key", sm4Key)
		c.Set("sm4_iv", iv)

		// 记录解密日志（不包含敏感数据）
		applogger.WithFields(map[string]interface{}{
			"path":                c.Request.URL.Path,
			"method":              c.Request.Method,
			"request_decrypted":   true,
			"decrypted_data_size": len(decryptedData),
		}).Info("请求解密成功")

		c.Next()
	}
}

// getConfigFromDB 从数据库获取加密开关配置（带缓存）
// 如果数据库查询失败，回退到静态配置值
func getConfigFromDB(ctx context.Context, db *gorm.DB, fallbackValue bool) bool {
	// 检查缓存是否有效
	globalConfigCache.mu.RLock()
	if time.Since(globalConfigCache.lastUpdate) < cacheTTL {
		value := globalConfigCache.value
		globalConfigCache.mu.RUnlock()
		return value
	}
	globalConfigCache.mu.RUnlock()

	// 缓存过期，从数据库重新读取
	globalConfigCache.mu.Lock()
	defer globalConfigCache.mu.Unlock()

	// 双重检查：避免并发情况下重复查询
	if time.Since(globalConfigCache.lastUpdate) < cacheTTL {
		return globalConfigCache.value
	}

	// 从数据库查询配置
	var configValue string
	err := db.WithContext(ctx).
		Table("sys_config").
		Select("config_value").
		Where("config_key = ?", encryptionConfigKey).
		Pluck("config_value", &configValue).Error

	if err != nil {
		// 数据库查询失败，使用缓存值或静态配置
		applogger.WithField("error", err).Warn("从数据库读取请求加密开关配置失败，使用静态配置")
		if !globalConfigCache.lastUpdate.IsZero() {
			// 使用缓存值
			return globalConfigCache.value
		}
		// 使用静态配置作为回退
		return fallbackValue
	}

	// 解析配置值
	var enabled bool
	if strings.ToLower(configValue) == "true" || configValue == "1" {
		enabled = true
	} else if strings.ToLower(configValue) == "false" || configValue == "0" {
		enabled = false
	} else {
		// 无效值，使用静态配置
		applogger.WithField("config_value", configValue).
			Warn("请求加密开关配置值无效，使用静态配置")
		return fallbackValue
	}

	// 记录配置变化（仅在值真正变化时）
	if globalConfigCache.value != enabled {
		applogger.WithFields(map[string]interface{}{
			"old_value": globalConfigCache.value,
			"new_value": enabled,
		}).Info("请求加密开关配置已更新")
	}

	// 更新缓存
	globalConfigCache.value = enabled
	globalConfigCache.lastUpdate = time.Now()

	return enabled
}

// RefreshEncryptionConfigCache 刷新加密配置缓存
// 供配置更新接口调用，确保配置更改立即生效
func RefreshEncryptionConfigCache() {
	globalConfigCache.mu.Lock()
	defer globalConfigCache.mu.Unlock()

	globalConfigCache.lastUpdate = time.Time{}
	applogger.Info("请求加密配置缓存已标记为过期，下次请求将从数据库重新读取")
}

// GetEncryptionConfigFromCache 获取当前加密配置缓存值（用于公共端点）
// 返回缓存的配置值，不触发数据库查询
// 如果缓存从未初始化，返回默认值（启用）
func GetEncryptionConfigFromCache() bool {
	globalConfigCache.mu.RLock()
	defer globalConfigCache.mu.RUnlock()

	// 如果缓存从未初始化，返回默认值（启用）
	if globalConfigCache.lastUpdate.IsZero() {
		return true
	}

	return globalConfigCache.value
}

// isExcludedPath 检查路径是否在排除列表中
// 支持通配符匹配，如 "/api/v1/upload/*"
func isExcludedPath(path string, excludePatterns []string) bool {
	for _, pattern := range excludePatterns {
		// 尝试 filepath.Match 通配符匹配
		matched, _ := filepath.Match(pattern, path)
		if matched {
			return true
		}

		// 支持前缀匹配（如 "/api/v1/system/auth/*"）
		if strings.HasSuffix(pattern, "/*") {
			prefix := strings.TrimSuffix(pattern, "/*")
			if strings.HasPrefix(path, prefix) {
				// 检查后续部分是否以 / 开头或者是完全匹配
				rest := path[len(prefix):]
				if rest == "" || strings.HasPrefix(rest, "/") {
					return true
				}
			}
		}
	}
	return false
}
