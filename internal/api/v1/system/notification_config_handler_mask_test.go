package system

// Phase 144 · webhook 配置读响应脱敏单测（ALRT-02 安全红线）。
//
// 覆盖：
//  1. Get 单条配置：authConfig / headers 中 token/key/secret/Authorization 值
//     被置 "******"，非敏感字段保留；
//  2. APIURL 的 ?key= / access_token= 查询凭据被遮；
//  3. List 列表逐条脱敏（[]models.APINotificationConfig 形态）；
//  4. 原 DB 行不被修改（深拷贝）；
//  5. List/Get 端点响应体不含明文 secret。

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/xingran-next/xingran-go-backend/internal/models"
)

func TestPhase144_MaskAPINotificationConfig_MasksSecrets(t *testing.T) {
	cfg := &models.APINotificationConfig{
		ID:     "cfg-1",
		APIURL: "https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=SECRET123&access_token=TOK456",
		Headers: models.MapFields{
			"Content-Type":  "application/json",
			"Authorization": "Bearer sk-live-abcdef",
			"X-Api-Key":     "AK-plain",
		},
		AuthConfig: models.MapFields{
			"token":      "bot-secret-token",
			"appSecret":  "s3cr3t",
			"timeout":    30,
			"retryCount": 3,
		},
	}
	got := maskAPINotificationConfig(cfg)

	assert.Contains(t, got.APIURL, "key=******")
	assert.Contains(t, got.APIURL, "access_token=******")
	assert.NotContains(t, got.APIURL, "SECRET123")
	assert.NotContains(t, got.APIURL, "TOK456")

	assert.Equal(t, "application/json", got.Headers["Content-Type"])
	assert.Equal(t, "******", got.Headers["Authorization"])
	assert.Equal(t, "******", got.Headers["X-Api-Key"])
	assert.Equal(t, "******", got.AuthConfig["token"])
	assert.Equal(t, "******", got.AuthConfig["appSecret"])
	assert.Equal(t, 30, got.AuthConfig["timeout"], "非敏感字段必须保留")

	// 原对象（DB 行）未被改动
	assert.Equal(t, "Bearer sk-live-abcdef", cfg.Headers["Authorization"])
	assert.Equal(t, "bot-secret-token", cfg.AuthConfig["token"])
}

func TestPhase144_MaskAPINotificationConfigList_MasksAllRows(t *testing.T) {
	list := []models.APINotificationConfig{
		{ID: "a", Headers: models.MapFields{"token": "T1"}, AuthConfig: models.MapFields{"secret": "S1"}},
		{ID: "b", Headers: models.MapFields{"token": "T2"}, AuthConfig: models.MapFields{"secret": "S2"}},
	}
	out, ok := maskAPINotificationConfigList(list).([]models.APINotificationConfig)
	require.True(t, ok)
	require.Len(t, out, 2)
	for _, c := range out {
		assert.Equal(t, "******", c.Headers["token"])
		assert.Equal(t, "******", c.AuthConfig["secret"])
	}
	assert.NotContains(t, list[0].Headers["token"], "******", "源 slice 不被修改")
}

// TestPhase144_APINotificationGetAndList_NoPlainSecretInResponse 端点级断言：
// 读响应体不含明文 webhook secret。
func TestPhase144_APINotificationGetAndList_NoPlainSecretInResponse(t *testing.T) {
	h, db := setupNotifConfigHandler(t)
	id := seedAPINotificationSecretConfig(t, db)

	// GET /:id
	w := recoverDoJSON(t, h.GetAPINotificationConfig, "GET", "/system/api-notification-configs/"+id,
		nil, map[string]string{"id": id})
	require.Equal(t, http.StatusOK, w.Code)
	assert.NotContains(t, w.Body.String(), "PLAIN-BOT-TOKEN")
	assert.NotContains(t, w.Body.String(), "PLAIN-APP-SECRET")

	// POST /list
	w2 := recoverDoJSON(t, h.ListAPINotificationConfigs, "POST", "/system/api-notification-configs/list",
		map[string]interface{}{"current": 1, "pageSize": 10}, nil)
	require.Equal(t, http.StatusOK, w2.Code)
	assert.NotContains(t, w2.Body.String(), "PLAIN-BOT-TOKEN")
	assert.NotContains(t, w2.Body.String(), "key=PLAIN-KEY")
}

// seedAPINotificationSecretConfig 直插一条含明文凭据的配置（脱敏前的真实形态）。
func seedAPINotificationSecretConfig(t *testing.T, db *gorm.DB) string {
	t.Helper()
	id := "phase144-secret-config"
	headers, _ := json.Marshal(models.MapFields{"Authorization": "Bearer PLAIN-BOT-TOKEN"})
	auth, _ := json.Marshal(models.MapFields{"token": "PLAIN-BOT-TOKEN", "appSecret": "PLAIN-APP-SECRET"})
	require.NoError(t, db.Exec(`
		INSERT INTO sys_api_notification_config
		(id, config_name, config_type, api_url, api_method, headers, auth_config, status, retry_count, timeout, is_default, del_flag, created_at, updated_at)
		VALUES (?, 'phase144', 'webhook', ?, 'POST', ?, ?, 0, 3, 30, 1, 0, datetime('now'), datetime('now'))`,
		id,
		"https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=PLAIN-KEY",
		string(headers), string(auth)).Error)
	return id
}
