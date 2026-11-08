package workorder

// FIX-06 / Phase 136：工单评价 handler 的 HTTP 端点可用性测试。
//
// 覆盖 RatingService 接线的三个端点（Create / List / Statistics）的
// response 包装（code=0 / code!=0）与业务规则透传。
// 复用同包 workorder_handler_test.go 的 newTestContext / parseResponse 夹具；
// 但**自建专用内存库与正确列集的表**（既有 setupTestDB 的
// sys_workorder_rating 是旧列 score，与 models.WorkOrderRating 不一致，
// 直接复用会让 service.Create 写库失败 → 测试无法证明接线可用）。

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xingran-next/xingran-go-backend/internal/core"
	"github.com/xingran-next/xingran-go-backend/internal/core/db"
	"github.com/xingran-next/xingran-go-backend/internal/services/workorder"
	"gorm.io/gorm"
)

func newRatingTestHandler(t *testing.T) (*RatingHandler, *gorm.DB) {
	t.Helper()
	gormDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, gormDB.Exec(`CREATE TABLE sys_workorder (
		id TEXT PRIMARY KEY, work_order_no TEXT, title TEXT, status INTEGER DEFAULT 0, deleted_at DATETIME
	)`).Error)
	require.NoError(t, gormDB.Exec(`CREATE TABLE sys_workorder_rating (
		id TEXT PRIMARY KEY, work_order_id TEXT, rating_type TEXT, completion_score INTEGER DEFAULT 0,
		cooperation_score INTEGER DEFAULT 0, comment TEXT, rater_id TEXT, created_at DATETIME
	)`).Error)
	// RatingService.GetList 用 Preload("Rater") 关联 sys_user，需最小表避免报错
	require.NoError(t, gormDB.Exec(`CREATE TABLE sys_user (
		id TEXT PRIMARY KEY, username TEXT, nickname TEXT, deleted_at DATETIME
	)`).Error)
	return NewRatingHandler(workorder.NewRatingService(gormDB), &core.Core{
		CoreInfra:    &core.CoreInfra{DB: &db.Database{DB: gormDB, Type: "sqlite"}},
		CoreServices: &core.CoreServices{},
	}), gormDB
}

func responseCode(t *testing.T, w *httptest.ResponseRecorder) int {
	t.Helper()
	resp := parseResponse(t, w.Body.Bytes())
	return int(resp["code"].(float64))
}

func TestRatingHandler_CreateListStatistics(t *testing.T) {
	h, gormDB := newRatingTestHandler(t)

	// 已完成工单（service 只允许 Completed/Closed 评价）
	woID := "11111111-1111-4111-8111-111111111111"
	require.NoError(t, gormDB.Exec(
		`INSERT INTO sys_workorder (id, work_order_no, title, status) VALUES (?, ?, ?, ?)`,
		woID, "WO-001", "测试工单", 2, // WorkOrderStatusCompleted
	).Error)

	// ---- 1) Create（写端点；operlog.Record 对 nil OperLogService 优雅返回） ----
	c1, w1 := newTestContext(http.MethodPost, "/workorder/orders/ratings", "u-rater", map[string]interface{}{
		"workOrderId":     woID,
		"ratingType":      "user",
		"completionScore": 5,
		"comment":         "处理及时",
	})
	h.CreateRating(c1)
	assert.Equal(t, 0, responseCode(t, w1), "创建评价成功应返回 code=0")

	var count int64
	require.NoError(t, gormDB.Raw(`SELECT COUNT(*) FROM sys_workorder_rating WHERE work_order_id = ?`, woID).Scan(&count).Error)
	assert.Equal(t, int64(1), count, "评价应真实落库")

	// ---- 2) 重复评价被拒（service 规则透传） ----
	c2, w2 := newTestContext(http.MethodPost, "/workorder/orders/ratings", "u-rater", map[string]interface{}{
		"workOrderId": woID, "ratingType": "user", "completionScore": 4,
	})
	h.CreateRating(c2)
	assert.NotEqual(t, 0, responseCode(t, w2), "重复评价应被拒")

	// ---- 3) List（读端点） ----
	c3, w3 := newTestContext(http.MethodPost, "/workorder/orders/ratings/list", "u-rater", map[string]interface{}{
		"workOrderId": woID,
	})
	h.ListRatings(c3)
	assert.Equal(t, 0, responseCode(t, w3), "评价列表应 code=0")

	// ---- 4) Statistics（读端点；snake→camel 映射） ----
	c4, w4 := newTestContext(http.MethodPost, "/workorder/orders/ratings/statistics", "u-rater", map[string]interface{}{
		"workOrderId": woID,
	})
	h.GetRatingStatistics(c4)
	assert.Equal(t, 0, responseCode(t, w4), "评分统计应 code=0")

	// ---- 5) 缺 workOrderId 参数 → 非 0（ParamMissing） ----
	c5, w5 := newTestContext(http.MethodPost, "/workorder/orders/ratings/list", "u-rater", map[string]interface{}{})
	h.ListRatings(c5)
	assert.NotEqual(t, 0, responseCode(t, w5), "缺工单ID应报错")
}

func TestRatingHandler_CreateRejectsUnfinishedWorkOrder(t *testing.T) {
	h, gormDB := newRatingTestHandler(t)

	require.NoError(t, gormDB.Exec(
		`INSERT INTO sys_workorder (id, work_order_no, title, status) VALUES (?, ?, ?, ?)`,
		"22222222-2222-4222-8222-222222222222", "WO-002", "待处理工单", 0,
	).Error)

	c, w := newTestContext(http.MethodPost, "/workorder/orders/ratings", "u-rater", map[string]interface{}{
		"workOrderId": "22222222-2222-4222-8222-222222222222", "ratingType": "user", "completionScore": 5,
	})
	h.CreateRating(c)
	assert.NotEqual(t, 0, responseCode(t, w), "未完成工单不得评价")

	var count int64
	require.NoError(t, gormDB.Raw(`SELECT COUNT(*) FROM sys_workorder_rating`).Scan(&count).Error)
	assert.Equal(t, int64(0), count, "被拒的评价不得落库")
}
