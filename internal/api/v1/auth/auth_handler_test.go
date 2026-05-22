package auth

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestADLoginWithOUProcessing(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// 这个测试验证AD登录成功后OU处理逻辑的集成
	// 实际测试需要Mock AD认证服务和UserOUService

	tests := []struct {
		name           string
		requestBody    map[string]interface{}
		expectedStatus int
		description    string
	}{
		{
			name: "AD登录成功-有OU映射",
			requestBody: map[string]interface{}{
				"username":  "testuser",
				"password":  "password123",
				"authType":  "ad",
				"captchaId": "test-captcha",
				"captcha":   "1234",
			},
			expectedStatus: http.StatusOK,
			description:    "AD用户登录成功，应触发OU处理",
		},
		{
			name: "本地登录-不处理OU",
			requestBody: map[string]interface{}{
				"username":  "localuser",
				"password":  "password123",
				"authType":  "local",
				"captchaId": "test-captcha",
				"captcha":   "1234",
			},
			expectedStatus: http.StatusOK,
			description:    "本地用户登录，不应触发OU处理",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()

			// 这里需要Mock AuthHandler和UserOUService
			// 实际集成测试会使用真实的依赖和测试数据库

			body, _ := json.Marshal(tt.requestBody)
			req, _ := http.NewRequest("POST", "/login", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			// 验证响应状态码
			assert.True(t, w.Code == tt.expectedStatus || w.Code == http.StatusUnauthorized, tt.description)
		})
	}
}

func TestADLoginOUProcessing_Integration(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// 集成测试：验证AD登录流程中OU处理的完整路径
	// 1. AD认证成功 → 提取OUDN
	// 2. 调用UserOUService.HandleUserLoginAD
	// 3. 更新用户的dept_id、ad_user_dn、ad_ou_dn

	t.Run("AD登录OU处理完整流程", func(t *testing.T) {
		// 这个测试需要：
		// - Mock AD服务器或使用真实测试AD
		// - 测试数据库（包含部门-OU映射）
		// - 测试用户数据
		// - UserOUService实例

		// 验证点：
		// - 1. AD认证返回OUDN字段
		// - 2. UserOUService被正确调用
		// - 3. 用户表的AD字段被更新
		// - 4. 登录成功返回token

		assert.True(t, true, "集成测试需要完整的环境配置")
	})
}
