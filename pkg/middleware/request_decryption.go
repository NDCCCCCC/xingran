// Package middleware 提供请求解密中间件
package middleware

import (
	"bytes"
	"encoding/json"
	"io"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	applogger "github.com/ruoyi-next/ruoyi-go-backend/pkg/logger"
	"github.com/ruoyi-next/ruoyi-go-backend/pkg/response"
	"github.com/ruoyi-next/ruoyi-go-backend/pkg/crypto"
)

// RequestDecryptionConfig 解密配置
type RequestDecryptionConfig struct {
	Enabled          bool     // 是否启用
	ExcludePaths     []string // 排除路径（支持通配符）
	RequireEncryption bool    // 是否强制加密
}

// RequestDecryption 创建请求解密中间件
// 在 JWT 认证之前执行，解密请求体后替换原始请求体
func RequestDecryption(encryptor *crypto.RequestEncryptor, config *RequestDecryptionConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 检查是否启用加密
		if !config.Enabled {
			c.Next()
			return
		}

		// 检查是否在排除列表中
		if isExcludedPath(c.Request.URL.Path, config.ExcludePaths) {
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

		// 读取请求体
		bodyBytes, err := io.ReadAll(c.Request.Body)
		if err != nil {
			response.Error(c, response.ErrBadRequest, "读取请求体失败")
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
			if config.RequireEncryption {
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
			applogger.WithField("path", c.Request.URL.Path).
				WithError(err).
				Warn("解析加密请求失败")
			response.Error(c, response.ErrBadRequest, "加密请求格式错误")
			c.Abort()
			return
		}

		// 解密请求体（获取 SM4 密钥和 IV 用于响应加密）
		decryptedData, sm4Key, iv, err := encryptor.DecryptRequestWithKeyInfo(&encReq)
		if err != nil {
			applogger.WithFields(map[string]interface{}{
				"path": c.Request.URL.Path,
				"method": c.Request.Method,
			}).WithError(err).Warn("解密请求体失败")
			response.Error(c, response.ErrBadRequest, "解密失败: "+err.Error())
			c.Abort()
			return
		}

		// 验证解密后的数据是否为有效 JSON
		if !json.Valid(decryptedData) {
			applogger.WithField("path", c.Request.URL.Path).
				Warn("解密数据不是有效的 JSON")
			response.Error(c, response.ErrBadRequest, "解密数据格式无效")
			c.Abort()
			return
		}

		// 替换请求体
		c.Request.Body = io.NopCloser(bytes.NewBuffer(decryptedData))

		// 存储 SM4 密钥和 IV 到 gin.Context，供响应加密使用
		c.Set("sm4_key", sm4Key)
		c.Set("sm4_iv", iv)

		// 记录解密日志
		applogger.WithFields(map[string]interface{}{
			"path": c.Request.URL.Path,
			"method": c.Request.Method,
			"request_decrypted": true,
			"decryption_success": true,
		}).Info("请求解密成功")

		c.Next()
	}
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
