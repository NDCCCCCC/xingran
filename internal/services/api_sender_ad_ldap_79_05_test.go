package services

import (
	"context"
	"encoding/base64"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	"gorm.io/gorm"
)

// -------------------------------------------------------------------------
// 79-05 Task 7: api_sender_service.go(119 stmts)+ ad_ldap_client.go(179 stmts)
//
// api_sender: APISenderService.client 为同包可及 *http.Client → httpmock.ActivateNonDefault
// (research §8 认可;httptest.NewServer 为 fallback,此处不需要 —— 无真实 socket)。
//
// ad_ldap(D-79-04 Tier-1): param-assembly + dial-error + 纯 helper + wire ops 未连接/
// 连接已闭守卫分支;wire 真路径不在本 phase(78-07 Conclusion B:BER fake 不兼容,不再重试)。
// -------------------------------------------------------------------------

// newApd7905 sqlite + APISenderService + httpmock 挂到其内部 client。
func newApd7905(t *testing.T) (*APISenderService, *gorm.DB) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(strings.ReplaceAll(t.TempDir(), `\`, "/")+"/apd7905.db"), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	require.NoError(t, err, "open sqlite temp db")
	if sqlDB, err := db.DB(); err == nil {
		t.Cleanup(func() { _ = sqlDB.Close() })
	}
	require.NoError(t, db.AutoMigrate(&models.APINotificationConfig{}))

	svc := NewAPISenderService(db)
	require.NotNil(t, svc.client, "装配前提:内部 *http.Client 可注入 httpmock")
	// httpmock 需先 Activate(DefaultTransport 全局激活),ActivateNonDefault 只做 transport 挂载
	httpmock.Activate()
	httpmock.ActivateNonDefault(svc.client)
	t.Cleanup(httpmock.DeactivateAndReset)
	return svc, db
}

// apd7905SeedConfig 落一条 API 通知配置。
func apd7905SeedConfig(t *testing.T, db *gorm.DB, name, url, method string, cfgType models.APIConfigType, authType models.AuthType, authCfg models.MapFields, headers models.MapFields) *models.APINotificationConfig {
	t.Helper()
	cfg := &models.APINotificationConfig{
		ID:         "apd7905-" + name,
		ConfigName: name,
		ConfigType: cfgType,
		APIURL:     url,
		APIMethod:  method,
		Headers:    headers,
		AuthType:   authType,
		AuthConfig: authCfg,
		Status:     int(models.NotificationConfigStatusNormal),
		DelFlag:    0,
	}
	require.NoError(t, db.Create(cfg).Error)
	return cfg
}

// apd7905StubResponder 记录请求并返回 200。
func apd7905StubResponder(seen *apd7905Seen) httpmock.Responder {
	return func(req *http.Request) (*http.Response, error) {
		body, _ := io.ReadAll(req.Body)
		seen.mu.Lock()
		seen.body = string(body)
		seen.auth = req.Header.Get("Authorization")
		seen.contentType = req.Header.Get("Content-Type")
		seen.userAgent = req.Header.Get("User-Agent")
		seen.custom = req.Header.Get("X-Custom-7905")
		seen.apiKey = req.Header.Get("X-Api-Key-7905")
		seen.mu.Unlock()
		return httpmock.NewStringResponse(200, `{"ok":true}`), nil
	}
}

// apd7905Seen 记录服务端看到的请求形态。
type apd7905Seen struct {
	mu          sync.Mutex
	body        string
	auth        string
	contentType string
	userAgent   string
	custom      string
	apiKey      string
}

const apd7905URL = "http://fake7905.internal/api"

// -------------------------------------------------------------------------
// api_sender
// -------------------------------------------------------------------------

// TestApd7905_SendRequest_Happy sendRequest 回环成功:断言服务端收到的 body 与 header。
func TestApd7905_SendRequest_Happy(t *testing.T) {
	svc, db := newApd7905(t)
	seen := &apd7905Seen{}
	httpmock.RegisterResponder("POST", apd7905URL, apd7905StubResponder(seen))

	cfg := apd7905SeedConfig(t, db, "happy7905", apd7905URL, "POST", models.APIConfigTypeWebhook,
		models.AuthTypeBearer, models.MapFields{"token": "tok7905"}, models.MapFields{"X-Custom-7905": "custom-value"})

	result := svc.Send(context.Background(), cfg.ID, &APIMessage{
		Recipients: []string{"u1", "u2"},
		Title:      "title7905",
		Content:    "content7905",
	})
	require.NotNil(t, result)
	assert.True(t, result.Success, "message=%v err=%v", result.Message, result.Error)
	assert.Equal(t, 200, result.HTTPCode)
	assert.Contains(t, result.ResponseBody, "ok")
	assert.Equal(t, 0, result.RetryCount, "首次成功 RetryCount=0")

	seen.mu.Lock()
	defer seen.mu.Unlock()
	assert.Equal(t, "Bearer tok7905", seen.auth, "Bearer 认证头")
	assert.Equal(t, "custom-value", seen.custom, "自定义头透传")
	assert.Equal(t, "application/json", seen.contentType, "默认 Content-Type")
	assert.Contains(t, seen.body, `"title":"title7905"`)
	assert.Contains(t, seen.body, `"content":"content7905"`)
	assert.Contains(t, seen.body, `"recipients":["u1","u2"]`)
	assert.Equal(t, 1, httpmock.GetTotalCallCount(), "首次成功只发一次")
}

// TestApd7905_SendRequest_HTTPError HTTP 500 与连接拒绝两个失败分支。
func TestApd7905_SendRequest_HTTPError(t *testing.T) {
	svc, db := newApd7905(t)

	t.Run("http_500", func(t *testing.T) {
		httpmock.RegisterResponder("POST", apd7905URL, httpmock.NewStringResponder(500, "boom7905"))
		cfg := apd7905SeedConfig(t, db, "err7905", apd7905URL, "POST", models.APIConfigTypeWebhook, models.AuthTypeNone, nil, nil)

		result := svc.Send(context.Background(), cfg.ID, &APIMessage{Title: "t"})
		require.NotNil(t, result)
		assert.False(t, result.Success)
		//QUIRK-79-05-M(锁定): Send 的重试聚合返回值只带 Message/Error/RetryCount,
		// 末次尝试的 HTTPCode/ResponseBody 被丢弃(与单次 sendRequest 直返形态不同)。
		assert.Zero(t, result.HTTPCode, "聚合结果不带末次 HTTPCode(QUIRK-79-05-M)")
		assert.Empty(t, result.ResponseBody)
		require.Error(t, result.Error)
		assert.Contains(t, result.Error.Error(), "HTTP status code: 500")
		assert.Contains(t, result.Message, "API发送失败，已重试")
		assert.Equal(t, 3, result.RetryCount, "RetryCount 走列 default:3(零值跳过)")
		// 重试机制:默认 3 次 → 共 4 次请求
		assert.Equal(t, 4, httpmock.GetTotalCallCount(), "1 次首发 + 3 次重试")
	})

	t.Run("connection_refused", func(t *testing.T) {
		ln, err := net.Listen("tcp", "127.0.0.1:0")
		require.NoError(t, err)
		deadAddr := ln.Addr().String()
		require.NoError(t, ln.Close())

		// 同包换 client 字段绕开 httpmock:必须显式给一个全新真实 Transport,
		// 因为 httpmock.Activate() 会替换 http.DefaultTransport,Transport=nil 的新 client
		// 仍会被 mock 截获(报 no responder found)。
		deadClient := &http.Client{Timeout: 5 * time.Second, Transport: &http.Transport{}}
		orig := svc.client
		svc.client = deadClient
		t.Cleanup(func() { svc.client = orig })

		cfg := apd7905SeedConfig(t, db, "dead7905", "http://"+deadAddr+"/x", "POST", models.APIConfigTypeWebhook, models.AuthTypeNone, nil, nil)
		result := svc.Send(context.Background(), cfg.ID, &APIMessage{Title: "t"})
		require.NotNil(t, result)
		assert.False(t, result.Success)
		assert.Contains(t, result.Message, "API发送失败，已重试")
		require.Error(t, result.Error)
		// Windows 报 "connectex: No connection could be made because the target machine
		// actively refused it.",Linux 报 "connection refused" → 取共同子串
		assert.Contains(t, result.Error.Error(), "refused", "直连已关闭端口应产生连接拒绝错误")
	})
}

// TestApd7905_BuildBody_Trio buildRequestBody/buildFromTemplate/buildDefaultBody(:226-277)。
func TestApd7905_BuildBody_Trio(t *testing.T) {
	svc, _ := newApd7905(t)
	msg := &APIMessage{
		Recipients: []string{"a", "b"},
		Title:      "标题7905",
		Content:    "内容7905",
		Data:       map[string]interface{}{"level": 3},
	}

	t.Run("default_body_json", func(t *testing.T) {
		reader, err := svc.buildDefaultBody(msg)
		require.NoError(t, err)
		body, err := io.ReadAll(reader)
		require.NoError(t, err)
		assert.Contains(t, string(body), `"title":"标题7905"`)
		assert.Contains(t, string(body), `"content":"内容7905"`)
		assert.Contains(t, string(body), `"recipients":["a","b"]`)
		assert.Contains(t, string(body), `"level":3`, "Data 合并进顶层 JSON")
	})

	t.Run("template_body_substitution", func(t *testing.T) {
		tpl := `{"text":"{{title}}/{{content}}","to":"{{recipients}}","lvl":{{level}}}`
		reader, err := svc.buildFromTemplate(tpl, msg)
		require.NoError(t, err)
		body, err := io.ReadAll(reader)
		require.NoError(t, err)
		assert.Equal(t, `{"text":"标题7905/内容7905","to":"a,b","lvl":3}`, string(body))
	})

	t.Run("template_without_recipients_keeps_placeholder", func(t *testing.T) {
		//QUIRK-79-05-J(锁定): Recipients 为空时跳过 {{recipients}} 替换,占位符原样保留
		reader, err := svc.buildFromTemplate(`{"to":"{{recipients}}"}`, &APIMessage{Title: "t", Content: "c"})
		require.NoError(t, err)
		body, _ := io.ReadAll(reader)
		assert.Equal(t, `{"to":"{{recipients}}"}`, string(body))
	})

	t.Run("buildRequestBody_selects_template", func(t *testing.T) {
		cfg := &models.APINotificationConfig{TemplateBody: `{"t":"{{title}}"}`}
		reader, err := svc.buildRequestBody(cfg, msg)
		require.NoError(t, err)
		body, _ := io.ReadAll(reader)
		assert.Equal(t, `{"t":"标题7905"}`, string(body))
	})

	t.Run("buildRequestBody_selects_default", func(t *testing.T) {
		cfg := &models.APINotificationConfig{}
		reader, err := svc.buildRequestBody(cfg, msg)
		require.NoError(t, err)
		body, _ := io.ReadAll(reader)
		assert.Contains(t, string(body), `"title":"标题7905"`)
	})
}

// TestApd7905_Headers_And_Auth setRequestHeaders(:280-302)与 setAuthentication(:305-348)。
func TestApd7905_Headers_And_Auth(t *testing.T) {
	svc, _ := newApd7905(t)
	newReq := func() *http.Request {
		req, err := http.NewRequestWithContext(context.Background(), "POST", apd7905URL, strings.NewReader("{}"))
		require.NoError(t, err)
		return req
	}

	t.Run("headers_defaults_and_merge", func(t *testing.T) {
		req := newReq()
		svc.setRequestHeaders(req, &models.APINotificationConfig{})
		assert.Equal(t, "application/json", req.Header.Get("Content-Type"), "默认 Content-Type")
		assert.Equal(t, "Xingran-Notification/1.0", req.Header.Get("User-Agent"), "默认 UA")

		req2 := newReq()
		req2.Header.Set("User-Agent", "custom-ua")
		svc.setRequestHeaders(req2, &models.APINotificationConfig{Headers: models.MapFields{
			"X-Custom-7905": "v1",
			"X-Number":      42,
			"Content-Type":  "text/plain",
		}})
		assert.Equal(t, "custom-ua", req2.Header.Get("User-Agent"), "已有 UA 不覆盖")
		assert.Equal(t, "v1", req2.Header.Get("X-Custom-7905"))
		assert.Equal(t, "text/plain", req2.Header.Get("Content-Type"), "配置头可覆盖默认")
		assert.Equal(t, "42", req2.Header.Get("X-Number"), "非字符串值经 JSON 序列化")
	})

	authCases := []struct {
		name      string
		authType  models.AuthType
		authCfg   models.MapFields
		wantAuth  string
		wantKey   string
		wantKeyV  string
		wantError string
	}{
		{name: "none", authType: models.AuthTypeNone},
		{name: "basic", authType: models.AuthTypeBasic, authCfg: models.MapFields{"username": "u7905", "password": "p7905"},
			wantAuth: "Basic " + base64.StdEncoding.EncodeToString([]byte("u7905:p7905"))},
		{name: "bearer", authType: models.AuthTypeBearer, authCfg: models.MapFields{"token": "tok"}},
		{name: "apikey_default_header", authType: models.AuthTypeAPIKey, authCfg: models.MapFields{"key": "X-Api-Key-7905", "value": "secret"}},
		{name: "apikey_custom_header", authType: models.AuthTypeAPIKey, authCfg: models.MapFields{"key": "ignored", "value": "v", "header_name": "X-Real"}},
		{name: "basic_missing_config", authType: models.AuthTypeBasic, wantError: "Basic认证需要配置用户名和密码"},
		{name: "bearer_missing_config", authType: models.AuthTypeBearer, wantError: "Bearer认证需要配置Token"},
		{name: "apikey_missing_config", authType: models.AuthTypeAPIKey, wantError: "API Key认证需要配置Key"},
		{name: "unknown_type", authType: models.AuthType("oauth2"), wantError: "不支持的认证类型"},
	}
	for _, tc := range authCases {
		t.Run(tc.name, func(t *testing.T) {
			req := newReq()
			err := svc.setAuthentication(req, &models.APINotificationConfig{AuthType: tc.authType, AuthConfig: tc.authCfg})
			if tc.wantError != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.wantError)
				return
			}
			require.NoError(t, err)
			switch tc.authType {
			case models.AuthTypeBasic:
				assert.Equal(t, tc.wantAuth, req.Header.Get("Authorization"))
			case models.AuthTypeBearer:
				assert.Equal(t, "Bearer tok", req.Header.Get("Authorization"))
			case models.AuthTypeAPIKey:
				if tc.name == "apikey_custom_header" {
					assert.Equal(t, "v", req.Header.Get("X-Real"), "header_name 覆盖 key 字段")
				} else {
					assert.Equal(t, "secret", req.Header.Get("X-Api-Key-7905"))
				}
			case models.AuthTypeNone:
				assert.Empty(t, req.Header.Get("Authorization"))
			}
		})
	}
}

// TestApd7905_SendVariants Send 族六个入口(Send/SendWithDefaultConfig/SendNoticeAPI/
// SendSMS/SendWebhook/TestAPIConfig)成功与配置缺失两态。
func TestApd7905_SendVariants(t *testing.T) {
	ctx := context.Background()
	svc, db := newApd7905(t)

	// 必须最先跑:此时 db 尚无任何配置(嵌套 newApd7905 的 t.Cleanup 会 DeactivateAndReset
	// 全局 transport,故本测试内禁止再起新的 mock 环境)。
	t.Run("send_with_default_config_no_config_at_all", func(t *testing.T) {
		result := svc.SendWithDefaultConfig(ctx, models.APIConfigTypeSMS, &APIMessage{Title: "t"})
		require.NotNil(t, result)
		assert.False(t, result.Success)
		assert.Equal(t, "未找到默认API配置", result.Message)
	})

	t.Run("send_config_missing", func(t *testing.T) {
		result := svc.Send(ctx, "no-such-7905", &APIMessage{Title: "t"})
		require.NotNil(t, result)
		assert.False(t, result.Success)
		assert.Equal(t, "获取API配置失败", result.Message)
	})

	t.Run("send_config_disabled", func(t *testing.T) {
		cfg := apd7905SeedConfig(t, db, "disabled7905", apd7905URL, "POST", models.APIConfigTypeWebhook, models.AuthTypeNone, nil, nil)
		require.NoError(t, db.Model(cfg).Update("status", int(models.NotificationConfigStatusStopped)).Error)
		result := svc.Send(ctx, cfg.ID, &APIMessage{Title: "t"})
		require.NotNil(t, result)
		assert.False(t, result.Success)
		assert.Equal(t, "API配置未启用", result.Message)
		assert.Zero(t, httpmock.GetTotalCallCount(), "停用配置不应发请求")
	})

	t.Run("send_with_default_config", func(t *testing.T) {
		seen := &apd7905Seen{}
		httpmock.RegisterResponder("POST", apd7905URL, apd7905StubResponder(seen))
		cfg := apd7905SeedConfig(t, db, "default7905", apd7905URL, "POST", models.APIConfigTypePush, models.AuthTypeNone, nil, nil)
		require.NoError(t, db.Model(cfg).Update("is_default", true).Error)

		webhookType := models.APIConfigTypeWebhook
		pushType := models.APIConfigTypePush
		result := svc.SendWithDefaultConfig(ctx, pushType, &APIMessage{Title: "push7905"})
		require.NotNil(t, result)
		assert.True(t, result.Success, "message=%v err=%v", result.Message, result.Error)

		// 同类型但无默认配置 → 失败
		result2 := svc.SendWithDefaultConfig(ctx, webhookType, &APIMessage{Title: "t"})
		require.NotNil(t, result2)
		assert.False(t, result2.Success)
	})

	t.Run("send_notice_api", func(t *testing.T) {
		seen := &apd7905Seen{}
		httpmock.RegisterResponder("POST", apd7905URL, apd7905StubResponder(seen))
		cfg := apd7905SeedConfig(t, db, "notice7905", apd7905URL, "POST", models.APIConfigTypeWebhook, models.AuthTypeNone, nil, nil)

		publishAt := mhq7905Time(10, 8, 0)
		result := svc.SendNoticeAPI(ctx, cfg.ID, &models.Notice{
			NoticeTitle:   "通知7905",
			NoticeContent: "内容7905",
			NoticeType:    "2",
			Priority:      models.PriorityUrgent,
			PublishTime:   &publishAt,
			IsMarkdown:    true,
		}, []string{"u1"})
		require.NotNil(t, result)
		assert.True(t, result.Success, "message=%v err=%v", result.Message, result.Error)

		seen.mu.Lock()
		body := seen.body
		seen.mu.Unlock()
		assert.Contains(t, body, `"title":"通知7905"`)
		assert.Contains(t, body, `"noticeId":`)
		assert.Contains(t, body, `"priority":2`)
		assert.Contains(t, body, `"isMarkdown":true`)
	})

	t.Run("send_sms_and_webhook", func(t *testing.T) {
		seen := &apd7905Seen{}
		httpmock.RegisterResponder("POST", apd7905URL, apd7905StubResponder(seen))
		smsCfg := apd7905SeedConfig(t, db, "sms7905", apd7905URL, "POST", models.APIConfigTypeSMS, models.AuthTypeNone, nil, nil)

		result := svc.SendSMS(ctx, smsCfg.ID, []string{"13800007905"}, "短信7905")
		require.NotNil(t, result)
		assert.True(t, result.Success)
		seen.mu.Lock()
		assert.Contains(t, seen.body, `"type":"sms"`)
		assert.Contains(t, seen.body, `"content":"短信7905"`)
		seen.mu.Unlock()

		webhookCfg := apd7905SeedConfig(t, db, "hook7905", apd7905URL, "POST", models.APIConfigTypeWebhook, models.AuthTypeNone, nil, nil)
		result2 := svc.SendWebhook(ctx, webhookCfg.ID, map[string]interface{}{"k": "v7905"})
		require.NotNil(t, result2)
		assert.True(t, result2.Success)
		seen.mu.Lock()
		assert.Contains(t, seen.body, `"title":"Webhook通知"`)
		assert.Contains(t, seen.body, `"k":"v7905"`)
		seen.mu.Unlock()
	})

	t.Run("test_api_config", func(t *testing.T) {
		seen := &apd7905Seen{}
		httpmock.RegisterResponder("POST", apd7905URL, apd7905StubResponder(seen))
		cfg := apd7905SeedConfig(t, db, "test7905", apd7905URL, "POST", models.APIConfigTypeWebhook, models.AuthTypeNone, nil, nil)

		result := svc.TestAPIConfig(ctx, cfg.ID)
		require.NotNil(t, result)
		assert.True(t, result.Success)
		seen.mu.Lock()
		assert.Contains(t, seen.body, `"title":"测试消息"`)
		seen.mu.Unlock()

		result2 := svc.TestAPIConfig(ctx, "nope-7905")
		require.NotNil(t, result2)
		assert.False(t, result2.Success)
		assert.Equal(t, "获取API配置失败", result2.Message)
	})
}

// -------------------------------------------------------------------------
// ad_ldap_client(D-79-04 Tier-1)
// -------------------------------------------------------------------------

// newAdl7905 最小配置的 LDAPClient(未连接)。
func newAdl7905() *LDAPClient {
	return NewLDAPClient(&models.ADConfig{
		ServerAddress: "127.0.0.1",
		ServerPort:    389,
		DomainName:    "test.local",
		BaseDN:        "DC=test,DC=local",
		AdminUsername: "admin7905",
		AdminPassword: "pass7905",
	})
}

// TestAdl7905_Connect_DialError 指向已关闭端口 → Connect 报错;Close/IsConnected 幂等。
func TestAdl7905_Connect_DialError(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	deadPort := ln.Addr().(*net.TCPAddr).Port
	require.NoError(t, ln.Close())

	cfg := &models.ADConfig{
		ServerAddress: "127.0.0.1",
		ServerPort:    deadPort,
		DomainName:    "test.local",
		BaseDN:        "DC=test,DC=local",
		AdminUsername: "admin7905",
		AdminPassword: "pass7905",
	}
	client := NewLDAPClient(cfg)
	assert.False(t, client.IsConnected(), "连接前 IsConnected 应为 false")

	err = client.Connect()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "连接AD服务器失败")
	assert.False(t, client.IsConnected(), "dial 失败不建立连接")

	assert.NotPanics(t, func() { client.Close() }, "未连接时 Close 不应 panic")
	assert.False(t, client.IsConnected())
}

// TestAdl7905_Connect_BindFails_RealTCP 真实 TCP 监听(立即断开)→ Bind 失败分支。
//
// QUIRK-79-05-K(锁定): Connect 的 Bind 失败路径只调 c.conn.Close()(ldap.Conn 方法),
// 不把 c.conn 置 nil → IsConnected() 在 Connect 失败后仍返回 true。上层若以 IsConnected
// 判定可用性会误判;修复属生产改动,先立项。
func TestAdl7905_Connect_BindFails_RealTCP(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	var acceptWG sync.WaitGroup
	acceptWG.Add(1)
	stopAccept := make(chan struct{})
	go func() {
		defer acceptWG.Done()
		for {
			select {
			case <-stopAccept:
				return
			default:
			}
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			_ = conn.Close() // 立即断开 → Bind 失败
		}
	}()
	t.Cleanup(func() {
		close(stopAccept)
		_ = ln.Close()
		acceptWG.Wait()
	})

	cfg := &models.ADConfig{
		ServerAddress: "127.0.0.1",
		ServerPort:    ln.Addr().(*net.TCPAddr).Port,
		DomainName:    "test.local",
		BaseDN:        "DC=test,DC=local",
		AdminUsername: "admin7905",
		AdminPassword: "pass7905",
	}
	client := NewLDAPClient(cfg)
	err = client.Connect()
	require.Error(t, err, "立即断开的服务端 → Bind 必失败")
	t.Logf("connect error: %v", err)
	assert.True(t, client.IsConnected(), "QUIRK-79-05-K: Bind 失败后 conn 未置 nil")
	assert.NotPanics(t, func() { client.Close() })
	assert.False(t, client.IsConnected(), "显式 Close 后 conn 置 nil")
}

// TestAdl7905_LoadADTLSSkipVerify_EnvMatrix loadADTLSSkipVerify(:29-38)env 矩阵。
//
// 白盒:同包重置 adTLSSkipVerifyOnce 以驱动 sync.Once 多次执行(仅测试态,零生产改动)。
// 注意 loadADTLSSkipVerify → config.Load → Validate 要求 security.sm4_key 非空,
// 测试进程无 config.yaml → 必须 t.Setenv("SM4_KEY", ...) 否则 panic。
func TestAdl7905_LoadADTLSSkipVerify_EnvMatrix(t *testing.T) {
	// 恢复进程级默认(true):直接写缓存变量,不再触发 config.Load
	// (此时 t.Setenv 已按 LIFO 先还原 SM4_KEY,再 Load 会 panic)
	t.Cleanup(func() {
		_ = os.Unsetenv("XINGRAN_AD_TLS_SKIP_VERIFY")
		adTLSSkipVerifyOnce = sync.Once{}
		adTLSSkipVerify = true
	})

	t.Setenv("SM4_KEY", "MTIzNDU2Nzg5MDEyMzQ1Ng==") // config.Validate 必需

	cases := []struct {
		name   string
		value  string
		set    bool
		expect bool
	}{
		{name: "unset_defaults_true", set: false, expect: true},
		{name: "explicit_true", value: "true", set: true, expect: true},
		{name: "numeric_one_true", value: "1", set: true, expect: true},
		{name: "explicit_false", value: "false", set: true, expect: false},
		{name: "numeric_zero_false", value: "0", set: true, expect: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			adTLSSkipVerifyOnce = sync.Once{} // 重置 Once → 下次调用重新 Load
			if tc.set {
				t.Setenv("XINGRAN_AD_TLS_SKIP_VERIFY", tc.value)
			} else {
				require.NoError(t, os.Unsetenv("XINGRAN_AD_TLS_SKIP_VERIFY"))
			}
			assert.Equal(t, tc.expect, loadADTLSSkipVerify(), "ad.tls_skip_verify env=%q", tc.value)
		})
	}
}

// TestAdl7905_FormatUsername formatUsername(:101-116)三种形态。
func TestAdl7905_FormatUsername(t *testing.T) {
	client := newAdl7905()

	t.Run("plain_user_gets_domain_prefix", func(t *testing.T) {
		assert.Equal(t, `TEST\user7905`, client.formatUsername("user7905"))
	})

	t.Run("upn_passthrough", func(t *testing.T) {
		assert.Equal(t, "user7905@test.local", client.formatUsername("user7905@test.local"))
	})

	t.Run("netbios_passthrough", func(t *testing.T) {
		assert.Equal(t, `OTHER\user`, client.formatUsername(`OTHER\user`))
	})

	t.Run("empty_domain_leading_backslash", func(t *testing.T) {
		//QUIRK-79-05-L(锁定): DomainName 为空时 Split(".")[0] = "" → 返回 "\user"(前导反斜杠)
		empty := NewLDAPClient(&models.ADConfig{DomainName: ""})
		assert.Equal(t, `\user7905`, empty.formatUsername("user7905"))
	})
}

// TestAdl7905_PureHelpers extractRDN/parseIntOrDefault/encodePassword(:516-548)。
func TestAdl7905_PureHelpers(t *testing.T) {
	client := newAdl7905()

	t.Run("extract_rdn", func(t *testing.T) {
		assert.Equal(t, "CN=user1", client.extractRDN("CN=user1,OU=Sales,DC=example,DC=com"))
		assert.Equal(t, "CN=user1", client.extractRDN("CN=user1"))
		assert.Equal(t, "", client.extractRDN(""))
	})

	t.Run("parse_int_or_default", func(t *testing.T) {
		assert.Equal(t, 512, client.parseIntOrDefault("512", 999))
		assert.Equal(t, 66048, client.parseIntOrDefault("66048", 999))
		assert.Equal(t, 42, client.parseIntOrDefault("abc", 42))
		assert.Equal(t, 42, client.parseIntOrDefault("", 42))
		assert.Equal(t, -3, client.parseIntOrDefault("-3", 7), "负数可解析(Sscanf %d 接受负号,不走默认值)")
	})

	t.Run("encode_password_utf16le_quoted", func(t *testing.T) {
		// AD 要求:双引号包裹 + UTF-16LE
		assert.Equal(t,
			[]byte{0x22, 0x00, 0x41, 0x00, 0x22, 0x00},
			[]byte(client.encodePassword("A")), `"A" → UTF-16LE`)
		assert.Equal(t,
			[]byte{0x22, 0x00, 0x2D, 0x4E, 0x22, 0x00},
			[]byte(client.encodePassword("中")), "非 BMP 前字符按 rune 拆双字节(小端)")
		assert.Equal(t,
			[]byte{0x22, 0x00, 0x50, 0x00, 0x61, 0x00, 0x73, 0x00, 0x22, 0x00},
			[]byte(client.encodePassword("Pas")))
		assert.Len(t, client.encodePassword("pass7905"), 2*(len("pass7905")+2), "长度 = 2 × (字符数 + 2 引号)")
	})
}

// TestAdl7905_WireOps_NotConnected_GuardTable 16 个 wire 入口在连接不可用态全部报错。
//
// D-79-04: wire 真路径不在本 phase(78-07 Conclusion B:BER fake 不兼容),
// Tier-1 只收「连接不可用即错误」的守卫分支 + 参数组装段 + 纯 helper。
// 连接态来源:Connect 对立即断开的 TCP 监听 Bind 失败 → 底层 ldap.Conn 已进入 closing,
// 后续 Search/Modify/ModifyDN 一律立刻返回错误(不发网络包、不挂起)。
func TestAdl7905_WireOps_NotConnected_GuardTable(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { _ = ln.Close() })
	go func() {
		for {
			conn, aerr := ln.Accept()
			if aerr != nil {
				return
			}
			_ = conn.Close()
		}
	}()

	cfg := &models.ADConfig{
		ServerAddress: "127.0.0.1",
		ServerPort:    ln.Addr().(*net.TCPAddr).Port,
		DomainName:    "test.local",
		BaseDN:        "DC=test,DC=local",
		AdminUsername: "admin7905",
		AdminPassword: "pass7905",
	}
	client := NewLDAPClient(cfg)
	require.Error(t, client.Connect(), "前置:Bind 失败进入连接不可用态")

	ops := []struct {
		name string
		call func() error
	}{
		{"SearchOUs", func() error { _, err := client.SearchOUs("DC=test,DC=local"); return err }},
		{"SearchGroups", func() error { _, err := client.SearchGroups("DC=test,DC=local"); return err }},
		{"SearchUsers", func() error { _, err := client.SearchUsers("DC=test,DC=local"); return err }},
		{"SearchUsersByOU", func() error { _, err := client.SearchUsersByOU("DC=test,DC=local", "OU=Sales"); return err }},
		{"SearchGroupMembers", func() error { _, err := client.SearchGroupMembers("CN=g,DC=test,DC=local"); return err }},
		{"UpdateUserAttribute", func() error {
			return client.UpdateUserAttribute("CN=u,DC=test,DC=local", map[string]string{"mail": "x@y.z"})
		}},
		{"UpdateUserMultipleAttributes", func() error {
			return client.UpdateUserMultipleAttributes("CN=u,DC=test,DC=local", map[string][]string{"mail": {"x"}})
		}},
		{"UpdateGroupAttribute", func() error {
			return client.UpdateGroupAttribute("CN=g,DC=test,DC=local", map[string]string{"info": "x"})
		}},
		{"MoveUser", func() error { return client.MoveUser("CN=u,OU=a,DC=test,DC=local", "OU=b,DC=test,DC=local") }},
		{"EnableUser", func() error { return client.EnableUser("CN=u,DC=test,DC=local") }},
		{"DisableUser", func() error { return client.DisableUser("CN=u,DC=test,DC=local") }},
		{"UnlockUser", func() error { return client.UnlockUser("CN=u,DC=test,DC=local") }},
		{"ResetPassword", func() error { return client.ResetPassword("CN=u,DC=test,DC=local", "NewPass7905!") }},
		{"AddGroupMember", func() error { return client.AddGroupMember("CN=g,DC=test,DC=local", "CN=u,DC=test,DC=local") }},
		{"RemoveGroupMember", func() error { return client.RemoveGroupMember("CN=g,DC=test,DC=local", "CN=u,DC=test,DC=local") }},
		{"getUserByDN", func() error { _, err := client.getUserByDN("CN=u,DC=test,DC=local"); return err }},
	}

	for _, op := range ops {
		t.Run(op.name, func(t *testing.T) {
			var got error
			assert.NotPanics(t, func() { got = op.call() }, "连接不可用态应返回错误而非 panic")
			require.Error(t, got, "%s 在连接不可用时应返回 error", op.name)
		})
	}

	// 参数组装纯段在连接态之外的可达性:Ping 无连接报"连接未建立"
	assert.NotPanics(t, func() {
		pingErr := client.Ping()
		assert.Error(t, pingErr)
	})
	client.Close()
}
