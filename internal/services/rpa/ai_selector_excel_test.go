package rpa

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/xuri/excelize/v2"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/xingran-next/xingran-go-backend/internal/config"
	rpamodels "github.com/xingran-next/xingran-go-backend/internal/models/rpa"
	"github.com/xingran-next/xingran-go-backend/pkg/cache"
)

// =====================================================================
// Phase 74-05: ai_client.go + ai_service.go + ai_analyzer.go +
// selector_learner.go + excel_service.go + service.go 测试。
// =====================================================================

// newAICompatServer 返回 OpenAI 兼容 API 的 httptest 服务器，
// content 即 Call() 解析出的 message.content。
func newAICompatServer(t *testing.T, content string, status int) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/chat/completions") {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if status != http.StatusOK {
			w.WriteHeader(status)
			_, _ = w.Write([]byte(`{"error":"boom"}`))
			return
		}
		resp := map[string]interface{}{
			"choices": []interface{}{
				map[string]interface{}{
					"message": map[string]interface{}{"content": content},
				},
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestAIClient_Call(t *testing.T) {
	srv := newAICompatServer(t, "hello", http.StatusOK)
	client := NewAIClient(srv.URL, "key", "m", 100, time.Second)

	got, err := client.Call(context.Background(), []Message{{Role: "user", Content: "hi"}})
	require.NoError(t, err)
	assert.Equal(t, "hello", got)

	// 非 200
	errSrv := newAICompatServer(t, "", http.StatusInternalServerError)
	_, err = NewAIClient(errSrv.URL, "k", "m", 1, time.Second).Call(context.Background(), []Message{{Role: "user"}})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "API 返回错误 500")

	// 响应缺少 choices
	badSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{}`))
	}))
	t.Cleanup(badSrv.Close)
	_, err = NewAIClient(badSrv.URL, "k", "m", 1, time.Second).Call(context.Background(), []Message{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "缺少 choices")

	// 非 JSON 响应
	notJSON := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`not-json`))
	}))
	t.Cleanup(notJSON.Close)
	_, err = NewAIClient(notJSON.URL, "k", "m", 1, time.Second).Call(context.Background(), []Message{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "解析响应失败")

	// 无效 URL
	_, err = NewAIClient("http://127.0.0.1:1", "k", "m", 1, 100*time.Millisecond).Call(context.Background(), []Message{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "API 请求失败")

	// content 非字符串
	badContent := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":123}}]}`))
	}))
	t.Cleanup(badContent.Close)
	_, err = NewAIClient(badContent.URL, "k", "m", 1, time.Second).Call(context.Background(), []Message{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "无效的响应内容")

	// ConvertToMessages
	msgs := ConvertToMessages([]map[string]interface{}{
		{"role": "system", "content": "s"},
		{"role": "user", "content": "u"},
	})
	require.Len(t, msgs, 2)
	assert.Equal(t, "u", msgs[1].Content)
}

// newAICfg 构造指向 server 的 RPA AI 配置。
func newAICfg(genSrv, agentSrv string, genOn, agentOn bool) *config.Config {
	return &config.Config{RPA: config.RPAConfig{AI: config.RPAAIConfig{
		Generator: config.RPAAIGeneratorConfig{Enabled: genOn, BaseURL: genSrv, APIKey: "k", Model: "g", MaxTokens: 16},
		Agent:     config.RPAAIAgentConfig{Enabled: agentOn, BaseURL: agentSrv, APIKey: "k", Model: "a", MaxTokens: 16},
	}}}
}

func TestAIService_GenerateAndOptimize(t *testing.T) {
	genSrv := newAICompatServer(t, `{"actions":[{"type":"click"}],"explanation":"ok","confidence":0.9}`, http.StatusOK)
	cfg := newAICfg(genSrv.URL, genSrv.URL, true, false)
	svc := NewAIService(cfg, nil, nil)
	ctx := context.Background()

	resp, err := svc.GenerateScript(ctx, &AIScriptGenerateRequest{Description: "点登录", URL: "http://x"})
	require.NoError(t, err)
	assert.Equal(t, "ok", resp.Explanation)

	// 优化: 响应带 changes
	optSrv := newAICompatServer(t, `{"actions":[],"explanation":"opt","changes":["加了等待"],"confidence":0.8}`, http.StatusOK)
	cfg2 := newAICfg(optSrv.URL, optSrv.URL, true, false)
	svc2 := NewAIService(cfg2, nil, nil)
	opt, err := svc2.OptimizeScript(ctx, &AIScriptOptimizeRequest{
		Script: []interface{}{map[string]interface{}{"type": "click"}},
		Goals:  []string{"speed", "可靠性", "可维护性", "可读性", "custom-goal"},
	})
	require.NoError(t, err)
	assert.Equal(t, "opt", opt.Explanation)

	// 响应非预期 JSON
	badSrv := newAICompatServer(t, "plain text", http.StatusOK)
	cfg3 := newAICfg(badSrv.URL, badSrv.URL, true, false)
	_, err = NewAIService(cfg3, nil, nil).GenerateScript(ctx, &AIScriptGenerateRequest{Description: "d"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "解析 AI 响应失败")

	// 未启用
	off := newAICfg("", "", false, false)
	_, err = NewAIService(off, nil, nil).GenerateScript(ctx, &AIScriptGenerateRequest{Description: "d"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "未启用")
	_, err = NewAIService(off, nil, nil).OptimizeScript(ctx, &AIScriptOptimizeRequest{Script: []interface{}{}})
	require.Error(t, err)

	// DecideNextAction
	agentSrv := newAICompatServer(t, `{"type":"click","selector":"#x","confidence":0.7}`, http.StatusOK)
	cfg4 := newAICfg(agentSrv.URL, agentSrv.URL, false, true)
	action, err := NewAIService(cfg4, nil, nil).DecideNextAction(ctx, &AIAgentDecisionRequest{
		TaskDescription: "d", LastError: "e",
	})
	require.NoError(t, err)
	assert.Equal(t, "#x", action.Selector)

	_, err = NewAIService(off, nil, nil).DecideNextAction(ctx, &AIAgentDecisionRequest{TaskDescription: "d"})
	require.Error(t, err)
}

func TestErrorAnalyzer_Local(t *testing.T) {
	cfg := newAICfg("", "", false, false)
	a := NewErrorAnalyzer(cfg)
	ctx := context.Background()

	// 未启用 → AnalyzeFailure/SuggestFix 报错
	_, err := a.AnalyzeFailure(ctx, &AnalyzeFailureRequest{TaskDescription: "d", ErrorMessage: "x"})
	require.Error(t, err)
	_, err = a.SuggestFix(ctx, &SuggestFixRequest{TaskDescription: "d", ErrorMessage: "x"})
	require.Error(t, err)

	// 本地分类（不调 AI）
	cases := []struct {
		msg, category, sub string
		recoverable        bool
	}{
		{"element timed out", "timing", "timeout", true},
		{"selector not found", "selector", "element_not_found", true},
		{"network connection refused", "network", "connection_error", true},
		{"blocked by captcha", "network", "access_denied", false},
		{"invalid value format", "content", "invalid_input", true},
		{"permission forbidden", "logic", "access_denied", false},
		{"something else", "unknown", "unknown", false},
	}
	for _, tc := range cases {
		c, err := a.ClassifyError(ctx, tc.msg)
		require.NoError(t, err)
		assert.Equal(t, tc.category, c.Category, tc.msg)
		assert.Equal(t, tc.sub, c.SubCategory, tc.msg)
		assert.Equal(t, tc.recoverable, c.Recoverable, tc.msg)
	}
}

func TestErrorAnalyzer_AnalyzeFailure_FallbackToLocal(t *testing.T) {
	// AI 启用但端点不可达 → 降级为本地增强分类
	cfg := newAICfg("http://127.0.0.1:1", "http://127.0.0.1:1", true, true)
	a := NewErrorAnalyzer(cfg)

	res, err := a.AnalyzeFailure(context.Background(), &AnalyzeFailureRequest{
		TaskDescription: "d", ErrorMessage: "timeout waiting for selector",
	})
	require.NoError(t, err)
	assert.Equal(t, "timing", res.ErrorType)
	assert.NotEmpty(t, res.RootCause)
	assert.Equal(t, "medium", res.Severity)
	assert.True(t, res.CanAutoFix)

	// SuggestFix: AI 失败直接报错（无本地降级）
	_, err = a.SuggestFix(context.Background(), &SuggestFixRequest{TaskDescription: "d", ErrorMessage: "x"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "AI 请求失败")

	// SuggestFix: 正常路径
	srv := newAICompatServer(t, `{"reason":"r","confidence":0.9}`, http.StatusOK)
	cfg2 := newAICfg(srv.URL, srv.URL, true, true)
	fix, err := NewErrorAnalyzer(cfg2).SuggestFix(context.Background(), &SuggestFixRequest{
		TaskDescription: "d", ErrorMessage: "x", ScreenshotBase64: "img",
	})
	require.NoError(t, err)
	assert.Equal(t, "r", fix.Reason)
}

// fakeSelectorCache selector_learner 用缓存假实现。
type fakeSelectorCache struct {
	cache.Cache
	get    func(ctx context.Context, key string) (string, error)
	set    func(ctx context.Context, key string, val interface{}, ttl time.Duration) error
	delete func(ctx context.Context, key string) error
}

func (f *fakeSelectorCache) Get(ctx context.Context, key string) (string, error) {
	if f.get != nil {
		return f.get(ctx, key)
	}
	return "", assert.AnError
}
func (f *fakeSelectorCache) Set(ctx context.Context, key string, val interface{}, ttl time.Duration) error {
	if f.set != nil {
		return f.set(ctx, key, val, ttl)
	}
	return nil
}
func (f *fakeSelectorCache) Delete(ctx context.Context, key string) error {
	if f.delete != nil {
		return f.delete(ctx, key)
	}
	return nil
}

func newSelectorTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Exec(`
		CREATE TABLE selector_success_records (
			id TEXT PRIMARY KEY NOT NULL DEFAULT (lower(hex(randomblob(16)))),
			page_url TEXT, element_id TEXT, selector TEXT, selector_type TEXT,
			success_count INTEGER DEFAULT 1, avg_duration INTEGER DEFAULT 0,
			last_used_at DATETIME, created_at DATETIME, metadata TEXT
		);
		CREATE TABLE selector_failure_records (
			id TEXT PRIMARY KEY NOT NULL DEFAULT (lower(hex(randomblob(16)))),
			page_url TEXT, element_id TEXT, selector TEXT, error_type TEXT,
			error_message TEXT, failure_count INTEGER DEFAULT 1,
			created_at DATETIME, resolved_at DATETIME, resolved_with TEXT
		)
	`).Error)
	return db
}

func TestSelectorLearner_RecordAndGetBest(t *testing.T) {
	db := newSelectorTestDB(t)
	var setKeys []string
	c := &fakeSelectorCache{set: func(ctx context.Context, k string, v interface{}, ttl time.Duration) error {
		setKeys = append(setKeys, k)
		return nil
	}}
	l := NewSelectorLearner(db, c, newAICfg("", "", false, false))
	ctx := context.Background()

	// 新增成功记录
	require.NoError(t, l.RecordSuccess(ctx, &SelectorSuccessRecord{
		PageURL: "http://p", ElementID: "btn", Selector: "#btn", SelectorType: "css",
		AvgDuration: 10,
	}))
	// 重复记录 → 更新 success_count
	require.NoError(t, l.RecordSuccess(ctx, &SelectorSuccessRecord{
		PageURL: "http://p", ElementID: "btn", Selector: "#btn", AvgDuration: 20,
	}))

	// 失败记录新增 + 重复（达到 3 次触发 markSelectorForUpdate 空实现）
	require.NoError(t, l.RecordFailure(ctx, &SelectorFailureRecord{
		PageURL: "http://p", ElementID: "btn", Selector: "#old", ErrorType: "not_found",
	}))
	require.NoError(t, l.RecordFailure(ctx, &SelectorFailureRecord{
		PageURL: "http://p", ElementID: "btn", Selector: "#old",
	}))
	require.NoError(t, l.RecordFailure(ctx, &SelectorFailureRecord{
		PageURL: "http://p", ElementID: "btn", Selector: "#old",
	}))

	// GetBestSelector
	best, err := l.GetBestSelector(ctx, "http://p", "btn")
	require.NoError(t, err)
	require.NotNil(t, best)
	assert.Equal(t, "#btn", best.Selector)
	assert.NotEmpty(t, setKeys, "结果应写缓存")

	// 无记录 → nil, nil
	best, err = l.GetBestSelector(ctx, "http://p", "none")
	require.NoError(t, err)
	assert.Nil(t, best)

	// 缓存命中 → 直接返回
	cached := SelectorRecommendation{Selector: "#cached", Score: 1}
	cachedJSON, _ := json.Marshal(cached)
	hitCache := &fakeSelectorCache{get: func(ctx context.Context, k string) (string, error) {
		return string(cachedJSON), nil
	}}
	l2 := NewSelectorLearner(db, hitCache, nil)
	best, err = l2.GetBestSelector(ctx, "http://p", "btn")
	require.NoError(t, err)
	require.NotNil(t, best)
	assert.Equal(t, "#cached", best.Selector)

	// LearnFromExecution 是 TODO 空实现
	require.NoError(t, l.LearnFromExecution(ctx, "e1"))
}

func TestSelectorLearner_ScoreAndAlternatives(t *testing.T) {
	db := newSelectorTestDB(t)
	now := time.Now()
	require.NoError(t, db.Exec(`INSERT INTO selector_success_records (id, page_url, element_id, selector, selector_type, success_count, avg_duration, last_used_at, created_at) VALUES
		('s1','http://p','btn','#btn','css',10,5,?,?),
		('s2','http://p','btn','[data-testid=btn]','data-testid',8,5,?,?),
		('s3','http://other','x','#x','css',5,5,?,?)`, now, now, now, now, now, now).Error)
	require.NoError(t, db.Exec(`INSERT INTO selector_failure_records (id, page_url, element_id, selector, failure_count, created_at) VALUES
		('f1','http://p','btn','#btn',2,?)`, now).Error)

	l := NewSelectorLearner(db, &fakeSelectorCache{}, newAICfg("", "", false, false))
	ctx := context.Background()

	score, err := l.ScoreSelector(ctx, "#btn", "http://p")
	require.NoError(t, err)
	// D-12 quirk: ScoreSelector 构造 SelectorStats 时不设置 SuccessRate(恒 0),
	// 且 sqlite TEXT 时间戳经 raw INSERT 回读解析为近零时间 → recency 也为 0,
	// 得分只剩 usage 项: 0.2 * (10/100) = 0.02。断言现状, 不修业务码。
	assert.InDelta(t, 0.02, score, 0.001)

	// 未找到
	_, err = l.ScoreSelector(ctx, "#missing", "http://p")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "选择器未找到")

	// 替代选择器（同元素不同选择器, 得分>0.5）
	alts, err := l.GetSelectorAlternatives(ctx, "#btn", "http://p")
	require.NoError(t, err)
	assert.Contains(t, alts, "[data-testid=btn]")

	_, err = l.GetSelectorAlternatives(ctx, "#missing", "http://p")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "未找到选择器记录")

	// 历史（impl 级方法, 不在接口上）
	impl := l.(*selectorLearnerImpl)
	history, err := impl.GetSelectorHistory(ctx, "http://p", "btn", 10)
	require.NoError(t, err)
	assert.Len(t, history, 2)

	// 趋势
	trend, err := impl.AnalyzeSelectorTrends(ctx, "http://p", 7)
	require.NoError(t, err)
	assert.Equal(t, 2, trend.TotalSuccesses)
	assert.Equal(t, 1, trend.TotalFailures)
	assert.InDelta(t, 2.0/3.0, trend.OverallSuccessRate, 0.001)
	assert.Len(t, trend.MostUsedTypes, 2)
}

// buildXlsxFile 构造内存 xlsx 并包装为 multipart.FileHeader。
func buildXlsxFile(t *testing.T, headers []string, rows [][]string, name string) (*multipart.FileHeader, error) {
	t.Helper()
	f := excelize.NewFile()
	sheet := "Sheet1"
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		if err := f.SetCellValue(sheet, cell, h); err != nil {
			return nil, err
		}
	}
	for r, row := range rows {
		for i, v := range row {
			cell, _ := excelize.CoordinatesToCellName(i+1, r+2)
			if err := f.SetCellValue(sheet, cell, v); err != nil {
				return nil, err
			}
		}
	}
	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		return nil, err
	}

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", name)
	if err != nil {
		return nil, err
	}
	if _, err := part.Write(buf.Bytes()); err != nil {
		return nil, err
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}

	reader := multipart.NewReader(&body, writer.Boundary())
	form, err := reader.ReadForm(1 << 20)
	if err != nil {
		return nil, err
	}
	return form.File["file"][0], nil
}

func TestRPAExcelService_Parse(t *testing.T) {
	svc := NewRPAExcelService(nil)

	fh, err := buildXlsxFile(t,
		[]string{"name", "value"},
		[][]string{{"a", "1"}, {"b", "2"}, {"c", "3"}, {"d", "4"}, {"e", "5"},
			{"f", "6"}, {"g", "7"}, {"h", "8"}, {"i", "9"}, {"j", "10"}, {"k", "11"}, {"l", "12"}},
		"data.xlsx")
	require.NoError(t, err)

	result, err := svc.ParseExcelFile(fh)
	require.NoError(t, err)
	assert.Equal(t, "data.xlsx", result.FileName)
	assert.Equal(t, 1, result.SheetCount)
	assert.Equal(t, []string{"name", "value"}, result.Columns)
	assert.Equal(t, 12, result.RowCount)
	assert.Len(t, result.Preview, 10, "预览最多 10 行")

	// 全量数据行
	execData, err := svc.ParseExcelForExecution(fh)
	require.NoError(t, err)
	assert.Len(t, execData, 12)
	assert.Equal(t, "a", execData[0]["name"])

	// 空工作簿（无数据行）
	empty, err := buildXlsxFile(t, []string{"h"}, nil, "empty.xlsx")
	require.NoError(t, err)
	_, err = svc.ParseExcelForExecution(empty)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "没有数据行")

	// 非 Excel 内容
	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	part, _ := w.CreateFormFile("file", "bad.xlsx")
	_, _ = part.Write([]byte("not an excel"))
	_ = w.Close()
	rd := multipart.NewReader(&body, w.Boundary())
	form, err := rd.ReadForm(1 << 20)
	require.NoError(t, err)
	_, err = svc.ParseExcelFile(form.File["file"][0])
	require.Error(t, err)
	assert.Contains(t, err.Error(), "解析Excel失败")
}

func newInterventionTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Exec(`
		CREATE TABLE sys_rpa_human_interventions (
			id TEXT PRIMARY KEY, created_at DATETIME,
			execution_id TEXT, worker_id TEXT, action TEXT,
			message TEXT, input_data TEXT, reason TEXT,
			processed_at DATETIME, status TEXT DEFAULT 'pending'
		)
	`).Error)
	return db
}

func TestRPAExcelService_InterventionsAndReports(t *testing.T) {
	db := newInterventionTestDB(t)
	svc := NewRPAExcelService(db)
	ctx := context.Background()

	event, err := svc.CreateHumanInterventionEvent(ctx, &HumanInterventionRequest{
		ExecutionID: "e1", Action: "resume",
		Input:  map[string]interface{}{"k": "v"},
		Reason: "manual",
	}, "wk-1")
	require.NoError(t, err)
	assert.Equal(t, "pending", event.Status)
	assert.JSONEq(t, `{"k":"v"}`, event.InputData)

	// 待处理查询
	got, err := svc.GetPendingHumanIntervention(ctx, "e1")
	require.NoError(t, err)
	assert.Equal(t, event.ID, got.ID)

	// 不存在
	_, err = svc.GetPendingHumanIntervention(ctx, "missing")
	require.Error(t, err)

	// 处理
	require.NoError(t, svc.ProcessHumanIntervention(ctx, event.ID, true))
	var status string
	var processedAt *time.Time
	require.NoError(t, db.Raw(`SELECT status, processed_at FROM sys_rpa_human_interventions WHERE id = ?`, event.ID).Row().Scan(&status, &processedAt))
	assert.Equal(t, "processed", status)
	assert.NotNil(t, processedAt)

	// 批量报告（当前为存根实现）
	reportTask := rpamodels.Task{TaskName: "task"}
	reportTask.ID = "t1"
	require.NoError(t, svc.CreateBatchExecutionReport(ctx, "e1", &reportTask, []map[string]interface{}{{"a": 1}}))
	require.NoError(t, svc.UpdateBatchItemReport(ctx, "e1", 0, "success", nil, "", ""))
	report, err := svc.GetBatchExecutionReport(ctx, "e1")
	require.NoError(t, err)
	assert.Equal(t, "e1", report.ExecutionID)
}

func TestServiceGroup(t *testing.T) {
	db := newTaskTestDB(t)
	cfg := newAICfg("", "", false, false)
	sg := NewServiceGroup(db, cfg, nil, &fakePlainCache{}, &fakeRPACipher{})
	require.NotNil(t, sg.TaskService)
	require.NotNil(t, sg.WorkerService)
	require.NotNil(t, sg.ExecutionService)
	require.NotNil(t, sg.CredentialService)
	require.NotNil(t, sg.AIService)
	assert.Equal(t, db, sg.DB())
}

func TestFormatLogMessage(t *testing.T) {
	// 额外锁 SanitizeLogMessage 行为: pattern 是字面量替换
	out := SanitizeLogMessage("password=[^\\s]*")
	assert.Equal(t, "***", out, "当消息恰好等于 pattern 字面量时才替换")
	assert.Equal(t, "plain", SanitizeLogMessage("plain"))
}
