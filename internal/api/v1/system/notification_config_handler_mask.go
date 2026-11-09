package system

// Phase 144 · webhook 配置密钥脱敏补齐。
//
// 复用 sys_api_notification_config 作为漂移告警 webhook 配置前，
// GetAPINotificationConfig / ListAPINotificationConfigs 原样回显 authConfig /
// headers 中的 Bearer token / API key / HMAC secret（Create/Update 已有
// RecordWithBody 落库脱敏，但读端点此前无遮蔽）。本文件对读响应主动遮蔽，
// 与 email_config 的 Password = "******" 惯例（:98-104）一致。
//
// 遮蔽口径：对 authConfig / headers 的 JSON key 名做大小写不敏感子串匹配，
// 命中 token / secret / key / authorization / password / sign / credential
// 等关键词的值置 "******"；整段 MapFields 深拷贝后再改，绝不改动 DB 行。

import (
	"strings"

	"github.com/xingran-next/xingran-go-backend/internal/models"
)

// apiConfigSensitiveKeyParts 命中即遮蔽的 key 名子串。
var apiConfigSensitiveKeyParts = []string{
	"token", "secret", "key", "authorization", "auth", "password",
	"pwd", "sign", "credential", "access", "bearer", "apikey", "hmac",
}

func isAPISensitiveKey(k string) bool {
	lower := strings.ToLower(k)
	for _, part := range apiConfigSensitiveKeyParts {
		if strings.Contains(lower, part) {
			return true
		}
	}
	return false
}

// maskAPIMapFields 深拷贝并遮蔽敏感键；nil 返回 nil。
func maskAPIMapFields(src models.MapFields) models.MapFields {
	if src == nil {
		return nil
	}
	out := make(models.MapFields, len(src))
	for k, v := range src {
		if isAPISensitiveKey(k) {
			out[k] = "******"
		} else {
			out[k] = v
		}
	}
	return out
}

// maskAPINotificationConfig 遮蔽单条配置（Get 端点）。返回脱敏副本。
func maskAPINotificationConfig(c *models.APINotificationConfig) *models.APINotificationConfig {
	if c == nil {
		return nil
	}
	copied := *c
	copied.Headers = maskAPIMapFields(c.Headers)
	copied.AuthConfig = maskAPIMapFields(c.AuthConfig)
	// APIURL 的 ?key= / access_token= 查询凭据也遮蔽（PLAN 安全红线）
	copied.APIURL = maskURLQuerySecrets(c.APIURL)
	return &copied
}

// maskAPINotificationConfigList 遮蔽列表（List 端点）。支持 []models.APINotificationConfig
// 与指针切片两种形态；未知类型原样返回（不 panic，读端点不因脱敏失败）。
func maskAPINotificationConfigList(list interface{}) interface{} {
	switch v := list.(type) {
	case []models.APINotificationConfig:
		out := make([]models.APINotificationConfig, len(v))
		for i := range v {
			out[i] = *maskAPINotificationConfig(&v[i])
		}
		return out
	case []*models.APINotificationConfig:
		out := make([]*models.APINotificationConfig, len(v))
		for i := range v {
			out[i] = maskAPINotificationConfig(v[i])
		}
		return out
	default:
		return list
	}
}

// maskURLQuerySecrets 把 URL 查询串里敏感参数值遮成 ******。
func maskURLQuerySecrets(raw string) string {
	q := strings.Index(raw, "?")
	if q < 0 {
		return raw
	}
	base := raw[:q+1]
	query := raw[q+1:]
	parts := strings.Split(query, "&")
	masked := make([]string, 0, len(parts))
	for _, p := range parts {
		eq := strings.Index(p, "=")
		if eq < 0 {
			masked = append(masked, p)
			continue
		}
		name := p[:eq]
		if isAPISensitiveKey(name) {
			masked = append(masked, name+"=******")
		} else {
			masked = append(masked, p)
		}
	}
	return base + strings.Join(masked, "&")
}
