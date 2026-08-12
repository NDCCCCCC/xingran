// Package middleware 提供响应加密中间件
package middleware

import (
	"bytes"
	"encoding/json"
	"strconv"
	"sync/atomic"

	"github.com/gin-gonic/gin"
	"github.com/xingran-next/xingran-go-backend/pkg/crypto"
	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
	"gorm.io/gorm"
)

// ResponseEncryptionConfig 响应加密配置
type ResponseEncryptionConfig struct {
	Enabled      bool     // 是否启用
	ExcludePaths []string // 排除路径（支持通配符）
}

// responseWriter 自定义响应Writer，用于捕获响应内容
type responseWriter struct {
	gin.ResponseWriter
	buffer  *bytes.Buffer
	written int32 // 原子操作标记：0=未写入，1=已写入
}

// Write 拦截写入操作，只写入buffer，不写入原始Writer
// 在最后加密完成后，统一写入加密后的响应
func (w *responseWriter) Write(b []byte) (int, error) {
	return w.buffer.Write(b)
}

// WriteString 拦截字符串写入
func (w *responseWriter) WriteString(s string) (int, error) {
	return w.buffer.WriteString(s)
}

// writeBuffer 原子性地写入缓冲区内容（只执行一次）
func (w *responseWriter) writeBuffer() {
	if atomic.CompareAndSwapInt32(&w.written, 0, 1) {
		if _, err := w.ResponseWriter.Write(w.buffer.Bytes()); err != nil {
			applogger.Warnf("写入响应缓冲区失败: %v", err)
		}
	}
}

// ResponseEncryption 创建响应加密中间件
// 在响应返回前加密响应体
// 使用与请求解密相同的数据库配置，实现统一控制
func ResponseEncryption(encryptor *crypto.RequestEncryptor, config *ResponseEncryptionConfig, db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 从数据库获取加密开关配置（与请求解密共享配置）
		// 使用 false 作为回退值（响应加密默认禁用，向后兼容）
		enabled := getConfigFromDB(c.Request.Context(), db, false)

		// 检查是否启用加密
		if !enabled {
			c.Next()
			return
		}

		// Phase 40 修复（api-key-data-not-displaying）：
		// 即使响应加密开关为 true，也必须确认当前请求确实走过请求解密中间件
		// 并往 context 注入了 sm4_key（GET 类、未加密 POST 等路径不会注入）。
		// 若 context 中无 sm4_key，仍走加密路径会生成 sm4Key="" timestamp=0 的
		// 无效加密响应，前端无法解密，导致 API 密钥列表等页面数据不显示。
		if _, hasKey := c.Get("sm4_key"); !hasKey {
			c.Next()
			return
		}

		// 检查是否在排除列表中
		if isExcludedPath(c.Request.URL.Path, config.ExcludePaths) {
			c.Next()
			return
		}

		// 调试日志：记录所有请求路径（用于排查加密问题）
		if c.Request.URL.Path != "/api/v1/ad-domain/configs/list" {
			applogger.WithFields(map[string]interface{}{
				"path":     c.Request.URL.Path,
				"method":   c.Request.Method,
				"excluded": isExcludedPath(c.Request.URL.Path, config.ExcludePaths),
			}).Debug("响应加密中间件：处理请求")
		}

		// 只加密成功响应（状态码 200-299）
		// 使用自定义 Writer 捕获响应
		blw := &responseWriter{
			ResponseWriter: c.Writer,
			buffer:         bytes.NewBufferString(""),
		}
		c.Writer = blw

		// 执行后续处理
		c.Next()

		// 检查响应状态码
		statusCode := c.Writer.Status()
		if statusCode < 200 || statusCode >= 300 {
			// 错误响应不加密，直接返回原始响应
			blw.writeBuffer()
			return
		}

		// 检查是否已经发生错误
		if len(c.Errors) > 0 {
			// 有错误，不加密，直接返回原始响应
			blw.writeBuffer()
			return
		}

		// 检查 Content-Type，只加密 JSON 响应
		contentType := c.Writer.Header().Get("Content-Type")
		if contentType != "" && contentType != "application/json" && contentType != "application/json; charset=utf-8" {
			// 非 JSON 响应不加密，直接返回原始响应
			blw.writeBuffer()
			return
		}

		// 获取响应体
		responseBody := blw.buffer.Bytes()

		// 空响应体跳过
		if len(responseBody) == 0 {
			return
		}

		// 验证是否为有效 JSON
		if !json.Valid(responseBody) {
			// 不是有效 JSON，不加密，直接返回原始响应
			blw.writeBuffer()
			return
		}

		// 检查响应是否已经是加密格式
		var rawMap map[string]interface{}
		if json.Unmarshal(responseBody, &rawMap) == nil {
			if _, hasEncrypted := rawMap["encrypted"].(bool); hasEncrypted {
				// 已经是加密格式，直接返回原始响应
				blw.writeBuffer()
				return
			}
		}

		// 尝试从 gin.Context 获取 SM4 密钥和 IV（由请求解密中间件存储）
		var encryptedResp *crypto.EncryptedRequest
		if sm4Key, exists := c.Get("sm4_key"); exists && sm4Key != nil {
			if sm4Iv, exists := c.Get("sm4_iv"); exists && sm4Iv != nil {
				// 使用请求中的 SM4 密钥和 IV 加密响应
				keyBytes := sm4Key.([]byte)
				ivBytes := sm4Iv.([]byte)
				var err error
				encryptedResp, err = encryptor.EncryptResponseWithKey(responseBody, keyBytes, ivBytes)
				if err != nil {
					applogger.WithFields(map[string]interface{}{
						"path":   c.Request.URL.Path,
						"method": c.Request.Method,
					}).WithError(err).Warn("使用已有密钥加密响应失败，返回原始响应")
					blw.writeBuffer()
					return
				}
			}
		}

		// 如果没有存储的密钥，直接返回原始响应
		if encryptedResp == nil {
			blw.writeBuffer()
			return
		}

		// 将加密响应转为 JSON
		encryptedJSON, err := json.Marshal(encryptedResp)
		if err != nil {
			applogger.WithFields(map[string]interface{}{
				"path":   c.Request.URL.Path,
				"method": c.Request.Method,
			}).WithError(err).Warn("序列化加密响应失败，返回原始响应")
			blw.writeBuffer()
			return
		}

		// 设置响应头
		c.Writer.Header().Set("Content-Type", "application/json; charset=utf-8")
		c.Writer.Header().Set("X-Response-Encrypted", "true")

		// 清空原始响应并写入加密响应
		c.Writer.Header().Set("Content-Length", strconv.Itoa(len(encryptedJSON)))
		blw.buffer.Reset()
		if _, err := blw.ResponseWriter.Write(encryptedJSON); err != nil {
			applogger.Warnf("写入加密响应失败: %v", err)
		}

		// 记录加密日志
		applogger.WithFields(map[string]interface{}{
			"path":               c.Request.URL.Path,
			"method":             c.Request.Method,
			"response_encrypted": true,
			"encryption_success": true,
		}).Info("响应加密成功")
	}
}
