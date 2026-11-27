package rpa

// CONTRACT-06（Phase 152）: P0 契约回归闸门。
//
// 本文件覆盖 CONTRACT-01~05 的 HTTP 契约：
//   - 走 httptest + **真实 sqlite 文件库 + 真实 db.Database.AutoMigrate()**
//     （与生产 sqlite 启动路径同一入口，不是手写 CREATE TABLE）
//   - 走真实 service 实现（rpa.NewServiceGroup + 真实 core.SM4Cipher），
//     不使用 mock service 顶替
//   - 只对**外部** AI provider 用 httptest（那是外部 HTTP 依赖，不是内部服务替身）
//
// 禁止事项（已写入 AGENTS.md 测试约束段）：
//   1. 禁止在本文件里 db.Exec("CREATE TABLE ...") 自建表掩盖真实 schema；
//   2. 禁止用 mockTaskService / mockCredentialService 等替身跑这些用例。
// 违反任一条，本闸门即失效 —— 它的存在意义就是把 D1~D5 的 500/panic 挡在合并前。

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xingran-next/xingran-go-backend/internal/config"
	"github.com/xingran-next/xingran-go-backend/internal/core"
	"github.com/xingran-next/xingran-go-backend/internal/core/db"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	rpamodels "github.com/xingran-next/xingran-go-backend/internal/models/rpa"
	"github.com/xingran-next/xingran-go-backend/internal/services/rpa"
	"github.com/xingran-next/xingran-go-backend/pkg/crypto"
)

const contractP0SM4Key = "dGVzdC1zZWNyZXQxNiEhIQ==" // 仓库默认测试 key（16 字节 base64）

// contractP0AIResponse 是「超集」AI 响应：所有 AI 端点的 unmarshal 目标
// 都能从中取到自己要的字段（多余字段被忽略）。
const contractP0AIResponse = `{
  "actions": [{"type":"navigate","value":"https://example.com/login","attributes":{"description":"打开登录页"}}],
  "explanation": "生成 1 个导航动作",
  "confidence": 0.92,
  "changes": ["增加显式等待"],
  "errorType": "timing",
  "severity": "medium",
  "rootCause": "页面加载未完成",
  "suggestedActions": ["增加显式等待"],
  "preventionTips": ["使用稳定选择器"],
  "canAutoFix": true,
  "type": "click",
  "selector": "#submit",
  "coordinates": [10, 20],
  "value": "",
  "reasoning": "按钮已可见",
  "suggestedFix": "改用 data-testid",
  "originalAction": {"type":"click"},
  "fixedAction": {"type":"click","selector":"[data-testid=submit]"},
  "reason": "选择器不稳定"
}`

// newContractP0AIProvider 外部 AI provider 替身（httptest，不是内部服务 mock）。
func newContractP0AIProvider(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"choices": []interface{}{
				map[string]interface{}{
					"message": map[string]interface{}{"content": contractP0AIResponse},
				},
			},
		})
	}))
	t.Cleanup(srv.Close)
	return srv
}

// contractP0Core 组装「真实 DB + 真实 cipher」的 core。
//
// DB 走 db.NewDatabase(sqlite 文件) + AutoMigrate()，与生产 sqlite 启动路径
// 同一入口：任何模型缺 TableName / 缺注册 / PG-only 函数式默认，都会在
// AutoMigrate 处直接炸。
func contractP0Core(t *testing.T) (*core.Core, *db.Database) {
	t.Helper()

	database, err := db.NewDatabase(&config.DatabaseConfig{
		Type: "sqlite",
		Path: filepath.Join(t.TempDir(), "contract-p0.db"),
	})
	require.NoError(t, err, "创建 sqlite 测试库失败")
	require.NoError(t, database.AutoMigrate(), "真实 AutoMigrate 必须成功（sqlite 全新库建全表）")
	t.Cleanup(func() {
		if sqlDB, err := database.DB.DB(); err == nil {
			_ = sqlDB.Close()
		}
	})

	cipher, err := crypto.NewSM4Cipher(contractP0SM4Key)
	require.NoError(t, err, "SM4 cipher 构造失败")

	aiServer := newContractP0AIProvider(t)
	cfg := &config.Config{}
	cfg.RPA.AI.Generator = config.RPAAIGeneratorConfig{
		Enabled:   true,
		APIKey:    "test-key",
		BaseURL:   aiServer.URL,
		Model:     "test-model",
		MaxTokens: 256,
	}
	cfg.RPA.AI.Agent = config.RPAAIAgentConfig{
		Enabled:   true,
		APIKey:    "test-key",
		BaseURL:   aiServer.URL,
		Model:     "test-model",
		MaxTokens: 256,
	}
	cfg.Security.SM4Key = contractP0SM4Key

	return &core.Core{
		CoreInfra: &core.CoreInfra{
			Config:    cfg,
			DB:        database,
			SM4Cipher: cipher, // ← CONTRACT-01 的真实注入值
		},
		CoreServices: &core.CoreServices{},
	}, database
}

// seedContractP0Admin 播种 role_key='admin' 超管让权限中间件放行。
// 权限缓存是进程级 sync.Map（30s TTL），每个用例用**唯一 userID** 避免串扰。
func seedContractP0Admin(t *testing.T, c *core.Core, userID string) {
	t.Helper()
	role := &models.Role{
		BaseModel: models.BaseModel{ID: "role-" + userID},
		RoleName:  "契约测试超管",
		RoleKey:   "admin",
	}
	require.NoError(t, c.GetDB().Create(role).Error)
	require.NoError(t, c.GetDB().Create(&models.UserRole{UserID: userID, RoleID: role.ID}).Error)
}

// contractP0Engine 用**真实** rpa.NewServiceGroup 装配整套 RPA 路由。
//
// CONTRACT-01 的关键断言点：SetupRPARouter 内部必须把 core.SM4Cipher 作为
// 第 5 参传给 NewServiceGroup。若仍传 nil，凭证 Create 在本文件里必然 500。
func contractP0Engine(t *testing.T, c *core.Core, userID string) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(func(ctx *gin.Context) {
		ctx.Set("user_id", userID)
		ctx.Set("dept_id", "")
		ctx.Next()
	})
	group := engine.Group("/api/v1/rpa")
	SetupRPARouter(group, c)
	return engine
}

// doContractP0 发 JSON POST 并返回状态码 + 解码响应体。
func doContractP0(t *testing.T, engine *gin.Engine, path string, body interface{}) (int, map[string]interface{}) {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		raw, err := json.Marshal(body)
		require.NoError(t, err)
		buf.Write(raw)
	}
	req := httptest.NewRequest(http.MethodPost, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	var resp map[string]interface{}
	if w.Body.Len() > 0 {
		_ = json.Unmarshal(w.Body.Bytes(), &resp)
	}
	return w.Code, resp
}

func respMessage(resp map[string]interface{}) string {
	if resp == nil {
		return ""
	}
	if s, ok := resp["message"].(string); ok {
		return s
	}
	return ""
}

func respData(resp map[string]interface{}) map[string]interface{} {
	if resp == nil {
		return nil
	}
	if m, ok := resp["data"].(map[string]interface{}); ok {
		return m
	}
	return nil
}

func respTotal(t *testing.T, resp map[string]interface{}) int64 {
	t.Helper()
	data := respData(resp)
	if data == nil {
		return 0
	}
	switch v := data["total"].(type) {
	case float64:
		return int64(v)
	case int64:
		return v
	}
	return 0
}

// =====================================================================
// CONTRACT-01: 凭证链路端到端（真实 SM4 cipher + 真实 sqlite 表）
// =====================================================================

func TestContractP0_CredentialCRUD_SQLiteRealTables(t *testing.T) {
	c, database := contractP0Core(t)
	const userID = "contract-p0-cred-user"
	seedContractP0Admin(t, c, userID)
	engine := contractP0Engine(t, c, userID)

	// --- Create ---
	code, resp := doContractP0(t, engine, "/api/v1/rpa/credentials", map[string]interface{}{
		"name":         "ERP 主账号",
		"targetSystem": "erp",
		"targetUrl":    "https://erp.example.com",
		"username":     "alice",
		"password":     "s3cret",
		"extraData":    map[string]interface{}{"tenant": "t1"},
	})
	require.Equal(t, http.StatusOK, code, "凭证创建必须 200（D1 修复前是 500/panic）: %s", respMessage(resp))
	data := respData(resp)
	require.NotNil(t, data, "创建响应必须带 data")
	credID, _ := data["id"].(string)
	require.NotEmpty(t, credID, "必须返回真实落库 ID")

	// 真实列里必须是密文，不能是明文
	var stored rpamodels.RPACredential
	require.NoError(t, database.DB.Where("id = ?", credID).First(&stored).Error)
	assert.NotEqual(t, "alice", stored.UsernameEncrypted, "username 必须已 SM4 加密")
	assert.NotEqual(t, "s3cret", stored.PasswordEncrypted, "password 必须已 SM4 加密")
	assert.NotEmpty(t, stored.ExtraDataEncrypted, "extraData 必须已加密落库")

	// --- Get ---
	code, resp = doContractP0(t, engine, "/api/v1/rpa/credentials/"+credID, nil)
	require.Equal(t, http.StatusOK, code, "凭证详情必须 200: %s", respMessage(resp))
	assert.Equal(t, "alice", respData(resp)["username"], "详情必须返回解密后的用户名")

	// --- List ---
	code, resp = doContractP0(t, engine, "/api/v1/rpa/credentials/list", map[string]interface{}{
		"current": 1, "pageSize": 10,
	})
	require.Equal(t, http.StatusOK, code, "凭证列表必须 200: %s", respMessage(resp))
	assert.EqualValues(t, 1, respTotal(t, resp))

	// --- Update ---
	code, resp = doContractP0(t, engine, "/api/v1/rpa/credentials/"+credID+"/update", map[string]interface{}{
		"name":     "ERP 主账号(改名)",
		"username": "alice2",
		"password": "n3w-secret",
	})
	require.Equal(t, http.StatusOK, code, "凭证更新必须 200: %s", respMessage(resp))
	require.NoError(t, database.DB.Where("id = ?", credID).First(&stored).Error)
	assert.Equal(t, "ERP 主账号(改名)", stored.Name)
	assert.NotEqual(t, "alice2", stored.UsernameEncrypted, "更新后必须重新加密")

	code, resp = doContractP0(t, engine, "/api/v1/rpa/credentials/"+credID, nil)
	require.Equal(t, http.StatusOK, code)
	assert.Equal(t, "alice2", respData(resp)["username"])

	// --- 会话创建（真实 service）+ 会话失效（HTTP）---
	credSvc := rpa.NewCredentialService(database.DB, c.SM4Cipher, nil)
	session, err := credSvc.CreateSession(t.Context(), &rpamodels.SessionCreateRequest{
		CredentialID: credID,
		TargetSystem: "erp",
		AccessToken:  "at-token",
		SessionData:  map[string]interface{}{"loggedIn": true},
	})
	require.NoError(t, err, "会话创建必须成功（cipher 非 nil）")
	assert.NotEqual(t, "at-token", session.AccessTokenEncrypted, "session token 必须加密落库")

	code, resp = doContractP0(t, engine,
		"/api/v1/rpa/credentials/sessions/"+session.ID+"/invalidate?reason=contract-p0", nil)
	require.Equal(t, http.StatusOK, code, "会话失效必须 200: %s", respMessage(resp))

	// --- Delete ---
	code, resp = doContractP0(t, engine, "/api/v1/rpa/credentials/"+credID+"/delete", nil)
	require.Equal(t, http.StatusOK, code, "凭证删除必须 200: %s", respMessage(resp))
	// 注：response.Error(c, http.StatusNotFound, ...) 在本仓 pkg/response 里把
	// int 形态的错误码映射成 HTTP 400（toAppError 的 int 分支恒 HTTPStatus=400，
	// 见 pkg/response/response.go:157-162）。这是全局既有行为，不在本 Phase 修复
	// （改动会影响全部模块），故这里断言「非 200 且 message 明确是凭证不存在」。
	code, resp = doContractP0(t, engine, "/api/v1/rpa/credentials/"+credID, nil)
	assert.NotEqual(t, http.StatusOK, code, "删除后详情不得 200")
	assert.Contains(t, respMessage(resp), "凭证不存在",
		"删除后详情必须明确报不存在；实际 code=%d message=%s", code, respMessage(resp))
}

// TestContractP0_CredentialNilCipher_ReadableError 负向用例：
// cipher 未配置时必须返回可读错误（不是 panic / 裸 500）。
// 刻意构造 nil cipher 的**真实** service（不是 mock service）。
func TestContractP0_CredentialNilCipher_ReadableError(t *testing.T) {
	_, database := contractP0Core(t)

	// 先直接落一行「看起来已加密」的凭证，让 List/Get 的解密分支真的被执行
	// （空表下 List 不会进解密循环，测不出 nil cipher 缺陷）。
	require.NoError(t, database.DB.Create(&rpamodels.RPACredential{
		Name: "nil-cipher-row", TargetSystem: "erp",
		UsernameEncrypted: "ciphertext-placeholder",
		PasswordEncrypted: "ciphertext-placeholder",
		UserID:            "contract-p0-nil-cipher",
	}).Error)

	nilCipherSvc := rpa.NewCredentialService(database.DB, nil, nil)
	handler := NewCredentialHandler(nilCipherSvc).WithCore(&core.Core{
		CoreInfra:    &core.CoreInfra{DB: database},
		CoreServices: &core.CoreServices{},
	})

	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(func(ctx *gin.Context) { ctx.Set("user_id", "contract-p0-nil-cipher") })
	engine.POST("/credentials", handler.Create)
	engine.POST("/credentials/list", handler.List)

	// Create：nil cipher → 非 200 且 message 含「凭证加密器未配置」
	code, resp := doContractP0(t, engine, "/credentials", map[string]interface{}{
		"name": "x", "targetSystem": "erp", "username": "u", "password": "p",
	})
	assert.NotEqual(t, http.StatusOK, code, "nil cipher 下创建不得成功")
	assert.Contains(t, respMessage(resp), "凭证加密器未配置",
		"必须返回可读原因而不是 panic；实际: %s", respMessage(resp))

	// List：nil cipher 也必须返回可读错误（不是 panic）
	code, resp = doContractP0(t, engine, "/credentials/list", map[string]interface{}{
		"current": 1, "pageSize": 10,
	})
	assert.NotEqual(t, http.StatusOK, code)
	assert.Contains(t, respMessage(resp), "凭证加密器未配置",
		"列表解密用户名时也必须明确报错；实际: %s", respMessage(resp))
}

// =====================================================================
// CONTRACT-02: 三张列表状态筛选前后端同构
// =====================================================================

func TestContractP0_TaskListStatusBinding(t *testing.T) {
	c, database := contractP0Core(t)
	const userID = "contract-p0-task-list"
	seedContractP0Admin(t, c, userID)
	engine := contractP0Engine(t, c, userID)

	for _, st := range []rpamodels.TaskStatus{rpamodels.TaskStatusEnabled, rpamodels.TaskStatusDisabled} {
		require.NoError(t, database.DB.Create(&rpamodels.Task{
			TaskName: fmt.Sprintf("task-%d", int(st)),
			Script:   json.RawMessage(`[{"type":"click","selector":"#a"}]`),
			Status:   st,
			Priority: 50,
		}).Error)
	}

	// status: 0（启用）
	code, resp := doContractP0(t, engine, "/api/v1/rpa/tasks/list", map[string]interface{}{
		"current": 1, "pageSize": 10, "status": 0,
	})
	require.Equal(t, http.StatusOK, code, "status=0 必须绑定成功: %s", respMessage(resp))
	assert.EqualValues(t, 1, respTotal(t, resp), "status=0 应只命中启用任务")

	// status: 1（停用）
	code, resp = doContractP0(t, engine, "/api/v1/rpa/tasks/list", map[string]interface{}{
		"current": 1, "pageSize": 10, "status": 1,
	})
	require.Equal(t, http.StatusOK, code)
	assert.EqualValues(t, 1, respTotal(t, resp), "status=1 应只命中停用任务")

	// status: null → 不过滤
	code, resp = doContractP0(t, engine, "/api/v1/rpa/tasks/list", map[string]interface{}{
		"current": 1, "pageSize": 10, "status": nil,
	})
	require.Equal(t, http.StatusOK, code)
	assert.EqualValues(t, 2, respTotal(t, resp), "status=null 必须视为不过滤")

	// status: "pending"（旧前端误传执行态字符串）→ 400 且错误可读
	code, resp = doContractP0(t, engine, "/api/v1/rpa/tasks/list", map[string]interface{}{
		"current": 1, "pageSize": 10, "status": "pending",
	})
	assert.Equal(t, http.StatusBadRequest, code, "执行态字符串传给任务列表必须 400")
	assert.Contains(t, respMessage(resp), "status", "400 必须指明出问题的字段；实际: %s", respMessage(resp))
	assert.Contains(t, respMessage(resp), "类型不匹配",
		"400 必须给出可读原因（不是裸 JSON 错误）；实际: %s", respMessage(resp))
	assert.Contains(t, respMessage(resp), "string",
		"400 必须指出实际收到的类型；实际: %s", respMessage(resp))

	// status: 9（枚举域外）→ 400 可读
	code, resp = doContractP0(t, engine, "/api/v1/rpa/tasks/list", map[string]interface{}{
		"current": 1, "pageSize": 10, "status": 9,
	})
	assert.Equal(t, http.StatusBadRequest, code)
	assert.Contains(t, respMessage(resp), "0（启用）", "非法枚举必须列出合法值；实际: %s", respMessage(resp))
}

func TestContractP0_WorkerListStatusBinding(t *testing.T) {
	c, database := contractP0Core(t)
	const userID = "contract-p0-worker-list"
	seedContractP0Admin(t, c, userID)
	engine := contractP0Engine(t, c, userID)

	require.NoError(t, database.DB.Create(&rpamodels.Worker{
		WorkerName: "w-online", WorkerID: "w-online", Status: rpamodels.WorkerStatusOnline,
	}).Error)
	require.NoError(t, database.DB.Create(&rpamodels.Worker{
		WorkerName: "w-offline", WorkerID: "w-offline", Status: rpamodels.WorkerStatusOffline,
	}).Error)

	// status:"online" —— 旧实现 *int 会 400 cannot unmarshal string into int
	code, resp := doContractP0(t, engine, "/api/v1/rpa/workers/list", map[string]interface{}{
		"current": 1, "pageSize": 10, "status": "online",
	})
	require.Equal(t, http.StatusOK, code, "status=\"online\" 必须绑定成功: %s", respMessage(resp))
	assert.EqualValues(t, 1, respTotal(t, resp), "应只命中在线 Worker")

	// status:"" （前端 allowClear 清空）→ 不过滤
	code, resp = doContractP0(t, engine, "/api/v1/rpa/workers/list", map[string]interface{}{
		"current": 1, "pageSize": 10, "status": "",
	})
	require.Equal(t, http.StatusOK, code)
	assert.EqualValues(t, 2, respTotal(t, resp), "空串必须视为不过滤（不能变成 WHERE status=''）")

	// status:null
	code, resp = doContractP0(t, engine, "/api/v1/rpa/workers/list", map[string]interface{}{
		"current": 1, "pageSize": 10, "status": nil,
	})
	require.Equal(t, http.StatusOK, code)
	assert.EqualValues(t, 2, respTotal(t, resp))

	// 非法枚举 → 400 可读
	code, resp = doContractP0(t, engine, "/api/v1/rpa/workers/list", map[string]interface{}{
		"current": 1, "pageSize": 10, "status": "yelling",
	})
	assert.Equal(t, http.StatusBadRequest, code)
	assert.Contains(t, respMessage(resp), "online", "错误消息必须列出合法枚举；实际: %s", respMessage(resp))
}

func TestContractP0_ExecutionListStatusBinding(t *testing.T) {
	c, database := contractP0Core(t)
	const userID = "contract-p0-exec-list"
	seedContractP0Admin(t, c, userID)
	engine := contractP0Engine(t, c, userID)

	for _, st := range []string{"pending", "timeout"} {
		require.NoError(t, database.DB.Create(&rpamodels.Execution{
			TaskID: "t-1", TaskName: "t", Status: st,
		}).Error)
	}

	code, resp := doContractP0(t, engine, "/api/v1/rpa/executions/list", map[string]interface{}{
		"current": 1, "pageSize": 10, "status": "pending",
	})
	require.Equal(t, http.StatusOK, code, "status=\"pending\" 必须绑定成功: %s", respMessage(resp))
	assert.EqualValues(t, 1, respTotal(t, resp))

	// timeout 是本次补进前端枚举的取值，必须能筛
	code, resp = doContractP0(t, engine, "/api/v1/rpa/executions/list", map[string]interface{}{
		"current": 1, "pageSize": 10, "status": "timeout",
	})
	require.Equal(t, http.StatusOK, code)
	assert.EqualValues(t, 1, respTotal(t, resp), "timeout 状态必须能被筛选")

	code, resp = doContractP0(t, engine, "/api/v1/rpa/executions/list", map[string]interface{}{
		"current": 1, "pageSize": 10, "status": "",
	})
	require.Equal(t, http.StatusOK, code)
	assert.EqualValues(t, 2, respTotal(t, resp), "空串必须视为不过滤")

	code, resp = doContractP0(t, engine, "/api/v1/rpa/executions/list", map[string]interface{}{
		"current": 1, "pageSize": 10, "status": nil,
	})
	require.Equal(t, http.StatusOK, code)
	assert.EqualValues(t, 2, respTotal(t, resp))

	code, resp = doContractP0(t, engine, "/api/v1/rpa/executions/list", map[string]interface{}{
		"current": 1, "pageSize": 10, "status": "completed",
	})
	assert.Equal(t, http.StatusBadRequest, code, "completed 不是后端执行态枚举，必须 400")
	assert.Contains(t, respMessage(resp), "pending", "错误消息必须列出合法枚举；实际: %s", respMessage(resp))
}

// =====================================================================
// CONTRACT-03: 任务搜索与字段无损（建→查→改→查 round-trip）
// =====================================================================

func TestContractP0_TaskRoundTrip_TagsAndZeroStatus(t *testing.T) {
	c, database := contractP0Core(t)
	const userID = "contract-p0-task-roundtrip"
	seedContractP0Admin(t, c, userID)
	engine := contractP0Engine(t, c, userID)

	// --- Create（tags + priority 0 + status 0）---
	code, resp := doContractP0(t, engine, "/api/v1/rpa/tasks", map[string]interface{}{
		"name":        "round-trip-task",
		"description": "契约回归",
		"script":      []interface{}{map[string]interface{}{"type": "click", "selector": "#go"}},
		"timeout":     300,
		"maxRetry":    1,
		"priority":    0,
		"status":      0,
		"tags":        "alpha,beta",
	})
	require.Equal(t, http.StatusOK, code, "任务创建必须 200: %s", respMessage(resp))
	taskID, _ := respData(resp)["id"].(string)
	require.NotEmpty(t, taskID)

	// --- 查：tags 必须已落库（旧实现 req.Tags 从不赋值 → 死字段）---
	code, resp = doContractP0(t, engine, "/api/v1/rpa/tasks/"+taskID, nil)
	require.Equal(t, http.StatusOK, code)
	assert.Equal(t, "alpha,beta", respData(resp)["tags"], "Create 必须保存 tags")
	assert.EqualValues(t, 0, respData(resp)["priority"])

	// --- 搜索：任务名过滤走真实列 name（旧 task_name → no such column 500）---
	code, resp = doContractP0(t, engine, "/api/v1/rpa/tasks/list", map[string]interface{}{
		"current": 1, "pageSize": 10, "name": "round-trip",
	})
	require.Equal(t, http.StatusOK, code,
		"任务名搜索必须 200（D3 修复前是 500 no such column: task_name）: %s", respMessage(resp))
	assert.EqualValues(t, 1, respTotal(t, resp), "任务名搜索必须命中真实 name 列")

	// --- 改：status=1 + priority=7 + tags 变更 ---
	code, resp = doContractP0(t, engine, "/api/v1/rpa/tasks/"+taskID+"/update", map[string]interface{}{
		"status":   1,
		"priority": 7,
		"tags":     "gamma",
	})
	require.Equal(t, http.StatusOK, code, "更新必须 200: %s", respMessage(resp))

	code, resp = doContractP0(t, engine, "/api/v1/rpa/tasks/"+taskID, nil)
	require.Equal(t, http.StatusOK, code)
	assert.EqualValues(t, 1, respData(resp)["status"], "status 必须改为 1")
	assert.EqualValues(t, 7, respData(resp)["priority"])
	assert.Equal(t, "gamma", respData(resp)["tags"], "Update 必须保存 tags")

	// --- 再改：status=0 + priority=0 —— 旧 `if req.Status > 0` 永远写不回 0 ---
	code, resp = doContractP0(t, engine, "/api/v1/rpa/tasks/"+taskID+"/update", map[string]interface{}{
		"status":   0,
		"priority": 0,
	})
	require.Equal(t, http.StatusOK, code, "status=0 更新必须 200: %s", respMessage(resp))
	code, resp = doContractP0(t, engine, "/api/v1/rpa/tasks/"+taskID, nil)
	require.Equal(t, http.StatusOK, code)
	assert.EqualValues(t, 0, respData(resp)["status"], "status 必须能改回 0（指针语义：显式 0 ≠ 未传）")
	assert.EqualValues(t, 0, respData(resp)["priority"], "priority 必须能改回 0")

	// --- 未传 status 的更新不得冲掉既有值（nil ≠ 0）---
	code, _ = doContractP0(t, engine, "/api/v1/rpa/tasks/"+taskID+"/update", map[string]interface{}{
		"name": "round-trip-task-renamed",
	})
	require.Equal(t, http.StatusOK, code)
	code, resp = doContractP0(t, engine, "/api/v1/rpa/tasks/"+taskID, nil)
	require.Equal(t, http.StatusOK, code)
	assert.Equal(t, "round-trip-task-renamed", respData(resp)["taskName"])
	assert.EqualValues(t, 0, respData(resp)["status"], "未传 status 必须保留原值 0")

	// --- 优先级域统一（裁定 0-100）：101 → 400 ---
	code, resp = doContractP0(t, engine, "/api/v1/rpa/tasks/"+taskID+"/update", map[string]interface{}{
		"priority": 101,
	})
	assert.Equal(t, http.StatusBadRequest, code, "priority=101 超出 0-100 必须 400")
	assert.Contains(t, respMessage(resp), "0-100", "错误消息必须给出取值域；实际: %s", respMessage(resp))

	// --- 清理：软删除后列表不再命中 ---
	code, resp = doContractP0(t, engine, "/api/v1/rpa/tasks/"+taskID+"/delete", nil)
	require.Equal(t, http.StatusOK, code, "软删除必须 200: %s", respMessage(resp))
	code, resp = doContractP0(t, engine, "/api/v1/rpa/tasks/list", map[string]interface{}{
		"current": 1, "pageSize": 10, "name": "round-trip-task-renamed",
	})
	require.Equal(t, http.StatusOK, code)
	assert.EqualValues(t, 0, respTotal(t, resp), "软删除后不得再出现在列表")
	_ = database
}

// =====================================================================
// CONTRACT-04: 选择器学习表打通（真实迁移后可写入 + 11 个 AI/selector 端点）
// =====================================================================

func TestContractP0_SelectorTables_RealMigrationThenWrite(t *testing.T) {
	_, database := contractP0Core(t)

	require.True(t, database.DB.Migrator().HasTable(&rpa.SelectorSuccessRecord{}))
	require.True(t, database.DB.Migrator().HasTable(&rpa.SelectorFailureRecord{}))
	require.Equal(t, "sys_rpa_selector_success", (&rpa.SelectorSuccessRecord{}).TableName())
	require.Equal(t, "sys_rpa_selector_failure", (&rpa.SelectorFailureRecord{}).TableName())

	var legacyCount int64
	require.NoError(t, database.DB.Raw(
		"SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name = 'selector_success_records'",
	).Scan(&legacyCount).Error)
	assert.EqualValues(t, 0, legacyCount, "不得存在 GORM 推导名的影子表")

	// 直接写入（不经 service）验证 schema 与 model tag 一致
	require.NoError(t, database.DB.Create(&rpa.SelectorSuccessRecord{
		PageURL: "https://p.example.com", ElementID: "btn", Selector: "#btn",
		SelectorType: "css", SuccessCount: 1,
	}).Error)
	require.NoError(t, database.DB.Create(&rpa.SelectorFailureRecord{
		PageURL: "https://p.example.com", ElementID: "btn", Selector: "#btn",
		ErrorType: "timeout", FailureCount: 1,
	}).Error)

	// BeforeCreate 必须补 UUID 主键（PG-only 默认已移除，sqlite 才能建表）
	var success rpa.SelectorSuccessRecord
	require.NoError(t, database.DB.Where("selector = ?", "#btn").First(&success).Error)
	assert.NotEmpty(t, success.ID, "应用层必须补 UUID 主键")
	_ = database
}

func TestContractP0_AIAndSelectorEndpoints_RealDB(t *testing.T) {
	c, database := contractP0Core(t)
	const userID = "contract-p0-ai"
	seedContractP0Admin(t, c, userID)
	engine := contractP0Engine(t, c, userID)

	// 第 1 个：记录选择器成功（真库写）
	code, resp := doContractP0(t, engine, "/api/v1/rpa/ai/selector/record-success", map[string]interface{}{
		"pageUrl": "https://erp.example.com/login", "elementId": "btn-login",
		"selector": "#login", "selectorType": "css", "successCount": 1, "avgDuration": 12,
	})
	require.Equal(t, http.StatusOK, code,
		"record-success 必须 200（D4 修复前 no such table）: %s", respMessage(resp))
	var successCount int64
	require.NoError(t, database.DB.Model(&rpa.SelectorSuccessRecord{}).
		Where("page_url = ? AND selector = ?", "https://erp.example.com/login", "#login").
		Count(&successCount).Error)
	assert.EqualValues(t, 1, successCount, "record-success 必须真实落库")

	// 第 2 个：再次成功 → upsert 计数 +1（证明真表可更新）
	code, _ = doContractP0(t, engine, "/api/v1/rpa/ai/selector/record-success", map[string]interface{}{
		"pageUrl": "https://erp.example.com/login", "elementId": "btn-login",
		"selector": "#login", "selectorType": "css", "successCount": 1, "avgDuration": 12,
	})
	require.Equal(t, http.StatusOK, code)
	var rec rpa.SelectorSuccessRecord
	require.NoError(t, database.DB.Where("page_url = ? AND selector = ?",
		"https://erp.example.com/login", "#login").First(&rec).Error)
	assert.EqualValues(t, 2, rec.SuccessCount, "重复上报必须递增 success_count")

	// 第 3 个：记录失败（真库写）
	code, resp = doContractP0(t, engine, "/api/v1/rpa/ai/selector/record-failure", map[string]interface{}{
		"pageUrl": "https://erp.example.com/login", "elementId": "btn-login",
		"selector": "#login-old", "errorType": "not_found", "errorMessage": "element not found",
	})
	require.Equal(t, http.StatusOK, code, "record-failure 必须 200: %s", respMessage(resp))
	var failureCount int64
	require.NoError(t, database.DB.Model(&rpa.SelectorFailureRecord{}).
		Where("page_url = ? AND selector = ?", "https://erp.example.com/login", "#login-old").
		Count(&failureCount).Error)
	assert.EqualValues(t, 1, failureCount, "record-failure 必须真实落库")

	// 第 4 个：最佳选择器（真库读）
	code, resp = doContractP0(t, engine, "/api/v1/rpa/ai/selector/best", map[string]interface{}{
		"pageUrl": "https://erp.example.com/login", "elementId": "btn-login",
	})
	require.Equal(t, http.StatusOK, code, "best 必须 200: %s", respMessage(resp))
	best, _ := respData(resp)["selector"].(string)
	assert.Equal(t, "#login", best, "best 必须从真表选出成功率最高的选择器")

	// 第 5 个：评分（真库读）
	code, resp = doContractP0(t, engine, "/api/v1/rpa/ai/selector/score", map[string]interface{}{
		"selector": "#login", "pageUrl": "https://erp.example.com/login",
	})
	require.Equal(t, http.StatusOK, code, "score 必须 200: %s", respMessage(resp))
	assert.NotNil(t, respData(resp)["score"])

	// 第 6 个：备选选择器（真库读）
	code, resp = doContractP0(t, engine, "/api/v1/rpa/ai/selector/alternatives", map[string]interface{}{
		"selector": "#login", "pageUrl": "https://erp.example.com/login",
	})
	require.Equal(t, http.StatusOK, code, "alternatives 必须 200: %s", respMessage(resp))
	assert.NotNil(t, respData(resp)["alternatives"])

	// 第 7~11 个：其余 AI 端点走真实 AI provider（httptest 外部依赖）
	aiCases := []struct {
		path string
		body map[string]interface{}
	}{
		{"/api/v1/rpa/ai/generate", map[string]interface{}{
			"description": "打开登录页并填写用户名",
			"url":         "https://erp.example.com/login",
		}},
		{"/api/v1/rpa/ai/optimize", map[string]interface{}{
			"script":      []interface{}{map[string]interface{}{"type": "click", "selector": "#a"}},
			"description": "加等待",
			"goals":       []string{"reliability"},
		}},
		{"/api/v1/rpa/ai/decide", map[string]interface{}{
			"taskDescription": "登录 ERP",
			"currentStep":     2,
			"lastError":       "timeout",
		}},
		{"/api/v1/rpa/ai/analyze-failure", map[string]interface{}{
			"taskDescription": "登录 ERP",
			"errorMessage":    "timeout waiting for #login",
		}},
		{"/api/v1/rpa/ai/suggest-fix", map[string]interface{}{
			"taskDescription": "登录 ERP",
			"errorMessage":    "timeout waiting for #login",
			"selector":        "#login",
		}},
		{"/api/v1/rpa/ai/classify-error", map[string]interface{}{
			"errorMessage": "timeout waiting for #login",
		}},
	}
	for _, tc := range aiCases {
		code, resp := doContractP0(t, engine, tc.path, tc.body)
		require.Equal(t, http.StatusOK, code, "%s 必须 200: %s", tc.path, respMessage(resp))
		assert.NotNil(t, respData(resp), "%s 必须返回 data", tc.path)
	}

	// 11 个 AI/selector 端点已全部走过真实 DB / 真实 AI provider。
	// （路由注册数由 router_public_test.go TestSetupAIRouter_Smoke 守护。）
}
