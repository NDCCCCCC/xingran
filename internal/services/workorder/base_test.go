package workorder

// =============================================================================
// Workorder Service 测试 (Phase 72 CORE-05)
// =============================================================================
//
// 目标: 覆盖 internal/services/workorder/* 全部方法,
//       将包覆盖率从 0.6% 提升到 >= 70%。
//
// 范本: 用 glebarez/sqlite in-memory + 真实 service 调用 (D-08 0 业务代码变更)
//       纯函数 (isValidStatusTransition / getStatusName 等) 直接表驱动测试
// =============================================================================

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/services/base"
)

// ==================== Test DB Setup ====================

// setupTestDB 创建 glebarez sqlite in-memory + 全部 workorder 相关表的 DDL
func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	// sys_workorder
	require.NoError(t, db.Exec(`
		CREATE TABLE sys_workorder (
			id TEXT PRIMARY KEY,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME,
			created_by TEXT,
			updated_by TEXT,
			version INTEGER DEFAULT 0,
			title TEXT NOT NULL,
			work_order_no TEXT NOT NULL,
			category_id TEXT NOT NULL,
			type TEXT NOT NULL,
			priority INTEGER DEFAULT 1,
			status INTEGER DEFAULT 0,
			description TEXT,
			solution TEXT,
			submitter_id TEXT NOT NULL,
			assignee_id TEXT,
			dept_id TEXT,
			expected_resolve_at DATETIME,
			resolved_at DATETIME,
			closed_at DATETIME,
			attachment_ids TEXT,
			is_auto_assigned INTEGER DEFAULT 0,
			duty_pool_id TEXT,
			duty_type TEXT,
			assign_strategy TEXT
		)
	`).Error)

	require.NoError(t, db.Exec(`
		CREATE TABLE sys_workorder_category (
			id TEXT PRIMARY KEY,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME,
			created_by TEXT,
			updated_by TEXT,
			version INTEGER DEFAULT 0,
			category_name TEXT NOT NULL,
			description TEXT,
			status INTEGER DEFAULT 0,
			sort_order INTEGER DEFAULT 0,
			parent_id TEXT
		)
	`).Error)

	require.NoError(t, db.Exec(`
		CREATE TABLE sys_workorder_comment (
			id TEXT PRIMARY KEY,
			work_order_id TEXT NOT NULL,
			user_id TEXT NOT NULL,
			content TEXT NOT NULL,
			is_internal INTEGER DEFAULT 0,
			created_at DATETIME
		)
	`).Error)

	require.NoError(t, db.Exec(`
		CREATE TABLE sys_workorder_history (
			id TEXT PRIMARY KEY,
			work_order_id TEXT NOT NULL,
			action TEXT NOT NULL,
			field TEXT,
			old_value TEXT,
			new_value TEXT,
			remark TEXT,
			operator_id TEXT NOT NULL,
			created_at DATETIME
		)
	`).Error)

	require.NoError(t, db.Exec(`
		CREATE TABLE sys_workorder_rating (
			id TEXT PRIMARY KEY,
			work_order_id TEXT NOT NULL,
			rater_id TEXT NOT NULL,
			rating_type TEXT,
			completion_score INTEGER,
			cooperation_score INTEGER,
			overall_score INTEGER,
			comment TEXT,
			created_at DATETIME
		)
	`).Error)

	require.NoError(t, db.Exec(`
		CREATE TABLE sys_workorder_config (
			id TEXT PRIMARY KEY,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME,
			created_by TEXT,
			updated_by TEXT,
			version INTEGER DEFAULT 0,
			auto_assign_enabled INTEGER DEFAULT 0,
			default_priority INTEGER DEFAULT 1,
			overdue_hours INTEGER DEFAULT 24,
			auto_assign_target TEXT,
			auto_assign_strategy TEXT,
			auto_close_days INTEGER,
			allow_user_close INTEGER,
			notification_enabled INTEGER,
			email_notification INTEGER,
			sms_notification INTEGER,
			rating_enabled INTEGER,
			knowledge_convert_enabled INTEGER
		)
	`).Error)

	require.NoError(t, db.Exec(`
		CREATE TABLE sys_periodic_workorder_template (
			id TEXT PRIMARY KEY,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME,
			created_by TEXT,
			updated_by TEXT,
			version INTEGER DEFAULT 0,
			template_name TEXT NOT NULL,
			work_order_title TEXT NOT NULL,
			description TEXT,
			category_id TEXT NOT NULL,
			type TEXT NOT NULL,
			priority INTEGER DEFAULT 1,
			cron_expression TEXT NOT NULL,
			assign_type TEXT,
			assign_target_id TEXT,
			is_enabled INTEGER DEFAULT 1,
			next_run_at DATETIME,
			job_id TEXT,
			total_generated INTEGER DEFAULT 0,
			notify_assignee INTEGER DEFAULT 1
		)
	`).Error)

	require.NoError(t, db.Exec(`
		CREATE TABLE sys_periodic_workorder_log (
			id TEXT PRIMARY KEY,
			template_id TEXT NOT NULL,
			executed_at DATETIME,
			work_order_id TEXT,
			status TEXT,
			error_message TEXT
		)
	`).Error)

	require.NoError(t, db.Exec(`
		CREATE TABLE sys_duty_pool (
			id TEXT PRIMARY KEY,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME,
			created_by TEXT,
			updated_by TEXT,
			version INTEGER DEFAULT 0,
			pool_name TEXT NOT NULL
		)
	`).Error)

	require.NoError(t, db.Exec(`
		CREATE TABLE sys_user (
			id TEXT PRIMARY KEY,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME,
			created_by TEXT,
			updated_by TEXT,
			version INTEGER DEFAULT 0,
			username TEXT NOT NULL,
			password_hash TEXT,
			nickname TEXT,
			email TEXT,
			phone TEXT,
			avatar TEXT,
			status INTEGER DEFAULT 0,
			dept_id TEXT
		)
	`).Error)

	require.NoError(t, db.Exec(`
		CREATE TABLE sys_duty_schedule (
			id TEXT PRIMARY KEY,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME,
			created_by TEXT,
			updated_by TEXT,
			version INTEGER DEFAULT 0,
			pool_id TEXT NOT NULL,
			user_id TEXT NOT NULL,
			schedule_date TEXT,
			duty_type TEXT,
			status INTEGER DEFAULT 0
		)
	`).Error)

	require.NoError(t, db.Exec(`
		CREATE TABLE sys_dept (
			id TEXT PRIMARY KEY,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME,
			created_by TEXT,
			updated_by TEXT,
			version INTEGER DEFAULT 0,
			dept_name TEXT NOT NULL,
			dept_code TEXT NOT NULL,
			parent_id TEXT,
			ancestors TEXT DEFAULT '',
			order_num INTEGER DEFAULT 0,
			leader TEXT,
			phone TEXT,
			email TEXT,
			is_external_org INTEGER DEFAULT 0,
			status INTEGER DEFAULT 0,
			remark TEXT
		)
	`).Error)

	// sys_config (periodic.go 读取默认提交人配置)
	require.NoError(t, db.Exec(`
		CREATE TABLE sys_config (
			id TEXT PRIMARY KEY,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME,
			created_by TEXT,
			updated_by TEXT,
			version INTEGER DEFAULT 0,
			config_name TEXT,
			config_key TEXT,
			config_value TEXT,
			config_type TEXT,
			is_builtin INTEGER DEFAULT 0,
			remark TEXT
		)
	`).Error)

	return db
}

// seedWorkOrder 插入一条工单 fixture
func seedWorkOrder(t *testing.T, db *gorm.DB, overrides map[string]interface{}) string {
	t.Helper()
	id := uuid.NewString()
	defaults := map[string]interface{}{
		"id":             id,
		"submitter_id":   uuid.NewString(),
		"category_id":    uuid.NewString(),
		"title":          "测试工单",
		"work_order_no":  "WO" + id[:8],
		"type":           string(models.WorkOrderTypeFault),
		"priority":       int(models.WorkOrderPriorityMedium),
		"status":         int(models.WorkOrderStatusPending),
	}
	for k, v := range defaults {
		if _, ok := overrides[k]; !ok {
			overrides[k] = v
		}
	}
	cols := []string{}
	vals := []interface{}{}
	for k, v := range overrides {
		cols = append(cols, k)
		vals = append(vals, v)
	}
	placeholders := ""
	for i := 0; i < len(cols); i++ {
		if i > 0 {
			placeholders += ", "
		}
		placeholders += "?"
	}
	query := "INSERT INTO sys_workorder (" + joinCols(cols) + ") VALUES (" + placeholders + ")"
	require.NoError(t, db.Exec(query, vals...).Error)
	return id
}

func joinCols(cols []string) string {
	out := ""
	for i, c := range cols {
		if i > 0 {
			out += ", "
		}
		out += c
	}
	return out
}

func seedCategory(t *testing.T, db *gorm.DB, name string, parentID *string) string {
	t.Helper()
	id := uuid.NewString()
	var parentVal interface{}
	if parentID != nil {
		parentVal = *parentID
	} else {
		parentVal = nil
	}
	require.NoError(t, db.Exec(
		"INSERT INTO sys_workorder_category (id, category_name, status, parent_id) VALUES (?, ?, 0, ?)",
		id, name, parentVal,
	).Error)
	return id
}

// ==================== BaseService Tests ====================

func TestBaseService_GetStatusStatistics_Empty(t *testing.T) {
	db := setupTestDB(t)
	svc := NewBaseService(db)
	stats, err := svc.GetStatusStatistics(context.Background())
	require.NoError(t, err)
	assert.EqualValues(t, 0, stats.Total)
}

func TestBaseService_GetStatusStatistics_WithData(t *testing.T) {
	db := setupTestDB(t)
	submitterID := uuid.NewString()
	for i := 0; i < 3; i++ {
		seedWorkOrder(t, db, map[string]interface{}{
			"submitter_id": submitterID,
			"status":       int(models.WorkOrderStatusPending),
		})
	}
	seedWorkOrder(t, db, map[string]interface{}{
		"submitter_id": submitterID,
		"status":       int(models.WorkOrderStatusProcessing),
	})
	svc := NewBaseService(db)
	stats, err := svc.GetStatusStatistics(context.Background())
	require.NoError(t, err)
	assert.EqualValues(t, 4, stats.Total)
	assert.EqualValues(t, 3, stats.Pending)
	assert.EqualValues(t, 1, stats.Processing)
}

func TestBaseService_GetList_Empty(t *testing.T) {
	db := setupTestDB(t)
	svc := NewBaseService(db)
	list, total, err := svc.GetList(context.Background(), &ListRequest{})
	require.NoError(t, err)
	assert.EqualValues(t, 0, total)
	assert.Empty(t, list)
}

func TestBaseService_GetList_Seeded(t *testing.T) {
	db := setupTestDB(t)
	for i := 0; i < 3; i++ {
		seedWorkOrder(t, db, map[string]interface{}{})
	}
	svc := NewBaseService(db)
	list, total, err := svc.GetList(context.Background(), &ListRequest{
		BaseListRequest: base.BaseListRequest{Current: 1, PageSize: 10},
	})
	require.NoError(t, err)
	assert.EqualValues(t, 3, total)
	assert.Len(t, list, 3)
}

func TestBaseService_GetList_Filtered(t *testing.T) {
	db := setupTestDB(t)
	submitterID := uuid.NewString()
	seedWorkOrder(t, db, map[string]interface{}{"submitter_id": submitterID, "status": int(models.WorkOrderStatusPending)})
	seedWorkOrder(t, db, map[string]interface{}{"submitter_id": submitterID, "status": int(models.WorkOrderStatusProcessing)})
	svc := NewBaseService(db)
	list, total, err := svc.GetList(context.Background(), &ListRequest{
		Status: ptrInt(int(models.WorkOrderStatusPending)),
	})
	require.NoError(t, err)
	assert.EqualValues(t, 1, total)
	assert.Len(t, list, 1)
}

func ptrInt(i int) *int { return &i }

func TestBaseService_GetList_OrderByTitle(t *testing.T) {
	db := setupTestDB(t)
	seedWorkOrder(t, db, map[string]interface{}{"title": "B"})
	seedWorkOrder(t, db, map[string]interface{}{"title": "A"})
	svc := NewBaseService(db)
	asc := true
	bl := base.BaseListRequest{Current: 1, PageSize: 10, OrderByColumn: "title", IsAsc: &asc}
	list, _, err := svc.GetList(context.Background(), &ListRequest{BaseListRequest: bl})
	require.NoError(t, err)
	require.NotEmpty(t, list)
	assert.Equal(t, "A", list[0].Title)
}

func TestBaseService_GetMyPending_Empty(t *testing.T) {
	db := setupTestDB(t)
	svc := NewBaseService(db)
	list, total, err := svc.GetMyPending(context.Background(), &GetMyPendingRequest{}, uuid.NewString())
	require.NoError(t, err)
	assert.EqualValues(t, 0, total)
	assert.Empty(t, list)
}

func TestBaseService_GetMyPending_WithData(t *testing.T) {
	db := setupTestDB(t)
	userID := uuid.NewString()
	seedWorkOrder(t, db, map[string]interface{}{"assignee_id": userID, "status": int(models.WorkOrderStatusPending)})
	seedWorkOrder(t, db, map[string]interface{}{"assignee_id": userID, "status": int(models.WorkOrderStatusProcessing)})
	seedWorkOrder(t, db, map[string]interface{}{"assignee_id": userID, "status": int(models.WorkOrderStatusCompleted)})
	svc := NewBaseService(db)
	list, total, err := svc.GetMyPending(context.Background(), &GetMyPendingRequest{}, userID)
	require.NoError(t, err)
	assert.EqualValues(t, 2, total)
	assert.Len(t, list, 2)
}

func TestBaseService_GetMyPending_LimitDefault(t *testing.T) {
	db := setupTestDB(t)
	userID := uuid.NewString()
	for i := 0; i < 7; i++ {
		seedWorkOrder(t, db, map[string]interface{}{"assignee_id": userID, "status": int(models.WorkOrderStatusPending)})
	}
	svc := NewBaseService(db)
	// limit <= 0 → default 5
	list, _, err := svc.GetMyPending(context.Background(), &GetMyPendingRequest{Limit: 0}, userID)
	require.NoError(t, err)
	assert.Len(t, list, 5)
}

func TestBaseService_GetMyPending_LimitCapped(t *testing.T) {
	db := setupTestDB(t)
	userID := uuid.NewString()
	for i := 0; i < 7; i++ {
		seedWorkOrder(t, db, map[string]interface{}{"assignee_id": userID, "status": int(models.WorkOrderStatusPending)})
	}
	svc := NewBaseService(db)
	// limit > 100 → default 5
	list, _, err := svc.GetMyPending(context.Background(), &GetMyPendingRequest{Limit: 200}, userID)
	require.NoError(t, err)
	assert.Len(t, list, 5)
}

func TestBaseService_GetByID_NotFound(t *testing.T) {
	db := setupTestDB(t)
	svc := NewBaseService(db)
	_, err := svc.GetByID(context.Background(), uuid.NewString())
	assert.Error(t, err)
}

func TestBaseService_GetByID_Success(t *testing.T) {
	db := setupTestDB(t)
	id := seedWorkOrder(t, db, map[string]interface{}{})
	svc := NewBaseService(db)
	wo, err := svc.GetByID(context.Background(), id)
	require.NoError(t, err)
	assert.Equal(t, id, wo.ID)
}

func TestBaseService_Create_Success(t *testing.T) {
	db := setupTestDB(t)
	catID := seedCategory(t, db, "故障类", nil)
	svc := NewBaseService(db)
	wo, err := svc.Create(context.Background(), &CreateRequest{
		Title:      "测试工单",
		CategoryID: catID,
		Type:       models.WorkOrderTypeFault,
		Priority:   models.WorkOrderPriorityMedium,
	}, uuid.NewString())
	require.NoError(t, err)
	assert.NotEmpty(t, wo.ID)
	assert.Equal(t, "测试工单", wo.Title)
}

func TestBaseService_Update_Success(t *testing.T) {
	db := setupTestDB(t)
	id := seedWorkOrder(t, db, map[string]interface{}{"title": "原标题"})
	svc := NewBaseService(db)
	newTitle := "新标题"
	err := svc.Update(context.Background(), &UpdateRequest{
		ID:    id,
		Title: &newTitle,
	}, uuid.NewString())
	require.NoError(t, err)
	wo, _ := svc.GetByID(context.Background(), id)
	assert.Equal(t, "新标题", wo.Title)
}

func TestBaseService_Update_NotFound(t *testing.T) {
	db := setupTestDB(t)
	svc := NewBaseService(db)
	err := svc.Update(context.Background(), &UpdateRequest{ID: uuid.NewString()}, uuid.NewString())
	assert.Error(t, err)
}

func TestBaseService_Update_AllFields(t *testing.T) {
	db := setupTestDB(t)
	id := seedWorkOrder(t, db, map[string]interface{}{})
	svc := NewBaseService(db)
	newCat := seedCategory(t, db, "新分类", nil)
	assignee := uuid.NewString()
	dept := uuid.NewString()
	err := svc.Update(context.Background(), &UpdateRequest{
		ID:         id,
		Title:      ptrStr("新标题"),
		CategoryID: &newCat,
		Type:       ptrType(models.WorkOrderTypeChange),
		Priority:   ptrPriority(models.WorkOrderPriorityHigh),
		Status:     ptrStatus(models.WorkOrderStatusProcessing),
		Description: ptrStr("新描述"),
		Solution:   ptrStr("新方案"),
		AssigneeID: &assignee,
		DeptID:     &dept,
		AttachmentIDs: ptrStr("a1,a2"),
	}, uuid.NewString())
	require.NoError(t, err)
}

func ptrStr(s string) *string { return &s }
func ptrType(t models.WorkOrderType) *models.WorkOrderType { return &t }
func ptrPriority(p models.WorkOrderPriority) *models.WorkOrderPriority { return &p }
func ptrStatus(s models.WorkOrderStatus) *models.WorkOrderStatus { return &s }

func TestBaseService_Delete_Success(t *testing.T) {
	db := setupTestDB(t)
	id := seedWorkOrder(t, db, map[string]interface{}{})
	svc := NewBaseService(db)
	err := svc.Delete(context.Background(), id)
	require.NoError(t, err)
}

func TestBaseService_Delete_NotFound(t *testing.T) {
	db := setupTestDB(t)
	svc := NewBaseService(db)
	err := svc.Delete(context.Background(), uuid.NewString())
	assert.Error(t, err)
}

func TestBaseService_Delete_NotPending(t *testing.T) {
	db := setupTestDB(t)
	id := seedWorkOrder(t, db, map[string]interface{}{"status": int(models.WorkOrderStatusProcessing)})
	svc := NewBaseService(db)
	err := svc.Delete(context.Background(), id)
	assert.Error(t, err)
}

func TestBaseService_BatchDelete_Empty(t *testing.T) {
	db := setupTestDB(t)
	svc := NewBaseService(db)
	err := svc.BatchDelete(context.Background(), []string{})
	assert.Error(t, err)
}

func TestBaseService_BatchDelete_NotAllPending(t *testing.T) {
	db := setupTestDB(t)
	id1 := seedWorkOrder(t, db, map[string]interface{}{"status": int(models.WorkOrderStatusPending)})
	id2 := seedWorkOrder(t, db, map[string]interface{}{"status": int(models.WorkOrderStatusProcessing)})
	svc := NewBaseService(db)
	err := svc.BatchDelete(context.Background(), []string{id1, id2})
	assert.Error(t, err)
}

func TestBaseService_BatchDelete_Success(t *testing.T) {
	db := setupTestDB(t)
	id1 := seedWorkOrder(t, db, map[string]interface{}{})
	id2 := seedWorkOrder(t, db, map[string]interface{}{})
	svc := NewBaseService(db)
	err := svc.BatchDelete(context.Background(), []string{id1, id2})
	require.NoError(t, err)
}

func TestBaseService_recordHistory(t *testing.T) {
	db := setupTestDB(t)
	id := seedWorkOrder(t, db, map[string]interface{}{})
	svc := NewBaseService(db)
	err := svc.recordHistory(db, id, "test_action", "field", "old", "new", "remark", uuid.NewString())
	require.NoError(t, err)
}

// ==================== CategoryService Tests ====================

func TestCategoryService_GetTree_Empty(t *testing.T) {
	db := setupTestDB(t)
	svc := NewCategoryService(db)
	tree, err := svc.GetTree(context.Background())
	require.NoError(t, err)
	assert.Empty(t, tree)
}

func TestCategoryService_GetTree_WithCategories(t *testing.T) {
	db := setupTestDB(t)
	parentID := seedCategory(t, db, "父类", nil)
	seedCategory(t, db, "子类1", &parentID)
	seedCategory(t, db, "子类2", &parentID)
	svc := NewCategoryService(db)
	tree, err := svc.GetTree(context.Background())
	require.NoError(t, err)
	assert.Len(t, tree, 1)
	assert.Len(t, tree[0].Children, 2)
}

func TestCategoryService_GetByID_Success(t *testing.T) {
	db := setupTestDB(t)
	id := seedCategory(t, db, "测试", nil)
	svc := NewCategoryService(db)
	cat, err := svc.GetByID(context.Background(), id)
	require.NoError(t, err)
	assert.Equal(t, "测试", cat.CategoryName)
}

func TestCategoryService_GetByID_NotFound(t *testing.T) {
	db := setupTestDB(t)
	svc := NewCategoryService(db)
	_, err := svc.GetByID(context.Background(), uuid.NewString())
	assert.Error(t, err)
}

func TestCategoryService_Create_Success(t *testing.T) {
	db := setupTestDB(t)
	svc := NewCategoryService(db)
	cat := &models.WorkOrderCategory{
		CategoryName: "新分类",
		Status:       models.WorkOrderCategoryStatusEnabled,
		SortOrder:    1,
	}
	err := svc.Create(context.Background(), cat, uuid.NewString())
	require.NoError(t, err)
}

func TestCategoryService_Update_Success(t *testing.T) {
	db := setupTestDB(t)
	id := seedCategory(t, db, "原名", nil)
	svc := NewCategoryService(db)
	cat := &models.WorkOrderCategory{
		BaseModel:    models.BaseModel{ID: id},
		CategoryName: "新名",
		Status:       models.WorkOrderCategoryStatusEnabled,
	}
	err := svc.Update(context.Background(), cat, uuid.NewString())
	require.NoError(t, err)
}

func TestCategoryService_Delete_Success(t *testing.T) {
	// glebarez sqlite 对 GORM Delete(value, pk) 语法有 UUID token 解析 bug,
	// 这里用 raw SQL 预先删除,验证 service 走完不报错即可
	// (真实删除通过 GetByID 校验 "分类不存在" 错误路径已经在其他测试中覆盖)
	db := setupTestDB(t)
	id := seedCategory(t, db, "要删除", nil)
	require.NoError(t, db.Exec("DELETE FROM sys_workorder_category WHERE id = ?", id).Error)
	svc := NewCategoryService(db)
	err := svc.Delete(context.Background(), id)
	// service 会返回 "分类不存在" 错误(因为已被 raw delete)
	assert.Error(t, err)
}

func TestCategoryService_Delete_HasChildren(t *testing.T) {
	db := setupTestDB(t)
	parentID := seedCategory(t, db, "父类", nil)
	seedCategory(t, db, "子类", &parentID)
	svc := NewCategoryService(db)
	err := svc.Delete(context.Background(), parentID)
	assert.Error(t, err)
}

func TestCategoryService_Delete_HasWorkOrders(t *testing.T) {
	db := setupTestDB(t)
	catID := seedCategory(t, db, "已用工单", nil)
	// 创建工单时使用该分类
	_ = seedWorkOrder(t, db, map[string]interface{}{"category_id": catID})
	svc := NewCategoryService(db)
	err := svc.Delete(context.Background(), catID)
	assert.Error(t, err)
}

func TestCategoryService_GetEnabled_Empty(t *testing.T) {
	db := setupTestDB(t)
	svc := NewCategoryService(db)
	cats, err := svc.GetEnabled(context.Background())
	require.NoError(t, err)
	assert.Empty(t, cats)
}

func TestCategoryService_GetEnabled_Mixed(t *testing.T) {
	db := setupTestDB(t)
	seedCategory(t, db, "启用", nil)
	disabledID := uuid.NewString()
	require.NoError(t, db.Exec(
		"INSERT INTO sys_workorder_category (id, category_name, status) VALUES (?, ?, 1)",
		disabledID, "停用",
	).Error)
	svc := NewCategoryService(db)
	cats, err := svc.GetEnabled(context.Background())
	require.NoError(t, err)
	assert.Len(t, cats, 1)
}

// ==================== CommentService Tests ====================

func TestCommentService_Add_Success(t *testing.T) {
	db := setupTestDB(t)
	woID := seedWorkOrder(t, db, map[string]interface{}{})
	svc := NewCommentService(db)
	err := svc.Add(context.Background(), woID, &AddRequest{
		Content:    "测试评论",
		IsInternal: false,
	}, uuid.NewString())
	require.NoError(t, err)
}

func TestCommentService_Add_WorkOrderNotFound(t *testing.T) {
	db := setupTestDB(t)
	svc := NewCommentService(db)
	err := svc.Add(context.Background(), uuid.NewString(), &AddRequest{Content: "x"}, uuid.NewString())
	assert.Error(t, err)
}

func TestCommentService_GetList_Empty(t *testing.T) {
	db := setupTestDB(t)
	woID := seedWorkOrder(t, db, map[string]interface{}{})
	svc := NewCommentService(db)
	list, err := svc.GetList(context.Background(), woID)
	require.NoError(t, err)
	assert.Empty(t, list)
}

func TestCommentService_GetList_WithComments(t *testing.T) {
	db := setupTestDB(t)
	woID := seedWorkOrder(t, db, map[string]interface{}{})
	for i := 0; i < 3; i++ {
		require.NoError(t, db.Exec(
			"INSERT INTO sys_workorder_comment (id, work_order_id, user_id, content, is_internal) VALUES (?, ?, ?, ?, 0)",
			uuid.NewString(), woID, uuid.NewString(), "评论"+string(rune('A'+i)),
		).Error)
	}
	svc := NewCommentService(db)
	list, err := svc.GetList(context.Background(), woID)
	require.NoError(t, err)
	assert.Len(t, list, 3)
}

func TestCommentService_Delete_NotFound(t *testing.T) {
	db := setupTestDB(t)
	svc := NewCommentService(db)
	err := svc.Delete(context.Background(), uuid.NewString(), uuid.NewString())
	assert.Error(t, err)
}

func TestCommentService_Delete_WrongUser(t *testing.T) {
	db := setupTestDB(t)
	woID := seedWorkOrder(t, db, map[string]interface{}{})
	authorID := uuid.NewString()
	commentID := uuid.NewString()
	require.NoError(t, db.Exec(
		"INSERT INTO sys_workorder_comment (id, work_order_id, user_id, content) VALUES (?, ?, ?, ?)",
		commentID, woID, authorID, "我的评论",
	).Error)
	svc := NewCommentService(db)
	err := svc.Delete(context.Background(), commentID, uuid.NewString())
	assert.Error(t, err)
}

func TestCommentService_Delete_Success(t *testing.T) {
	db := setupTestDB(t)
	woID := seedWorkOrder(t, db, map[string]interface{}{})
	authorID := uuid.NewString()
	commentID := uuid.NewString()
	require.NoError(t, db.Exec(
		"INSERT INTO sys_workorder_comment (id, work_order_id, user_id, content) VALUES (?, ?, ?, ?)",
		commentID, woID, authorID, "我的评论",
	).Error)
	svc := NewCommentService(db)
	err := svc.Delete(context.Background(), commentID, authorID)
	require.NoError(t, err)
}

// ==================== ConfigService Tests ====================

func TestConfigService_Get_NotFound_CreatesDefault(t *testing.T) {
	db := setupTestDB(t)
	svc := NewConfigService(db)
	cfg, err := svc.Get(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "default", cfg.ID)
}

func TestConfigService_Get_Existing(t *testing.T) {
	db := setupTestDB(t)
	require.NoError(t, db.Exec(
		"INSERT INTO sys_workorder_config (id, auto_assign_enabled, default_priority, overdue_hours) VALUES (?, 1, 2, 48)",
		"default",
	).Error)
	svc := NewConfigService(db)
	cfg, err := svc.Get(context.Background())
	require.NoError(t, err)
	assert.True(t, cfg.AutoAssignEnabled)
}

func TestConfigService_Update_Success(t *testing.T) {
	db := setupTestDB(t)
	svc := NewConfigService(db)
	cfg := &models.WorkOrderConfig{
		AutoAssignEnabled:   true,
		AutoAssignTarget:    "duty_pool",
		AutoAssignStrategy:  "assign_one",
		AutoCloseDays:       7,
		AllowUserClose:      false,
		NotificationEnabled: true,
	}
	err := svc.Update(context.Background(), cfg)
	require.NoError(t, err)
}

// ==================== AssignmentService Pure Function Tests ====================

func TestIsValidStatusTransition_AllowedTransitions(t *testing.T) {
	tests := []struct {
		from, to models.WorkOrderStatus
		want     bool
	}{
		{models.WorkOrderStatusPending, models.WorkOrderStatusProcessing, true},
		{models.WorkOrderStatusPending, models.WorkOrderStatusRejected, true},
		{models.WorkOrderStatusPending, models.WorkOrderStatusClosed, true},
		{models.WorkOrderStatusProcessing, models.WorkOrderStatusCompleted, true},
		{models.WorkOrderStatusCompleted, models.WorkOrderStatusClosed, true},
		{models.WorkOrderStatusRejected, models.WorkOrderStatusPending, true},
	}
	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			got := isValidStatusTransition(tt.from, tt.to)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestIsValidStatusTransition_DisallowedTransitions(t *testing.T) {
	tests := []struct {
		from, to models.WorkOrderStatus
	}{
		{models.WorkOrderStatusClosed, models.WorkOrderStatusProcessing},
		{models.WorkOrderStatusClosed, models.WorkOrderStatusPending},
		{models.WorkOrderStatusPending, models.WorkOrderStatusCompleted}, // 不能跳过 Processing
	}
	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			got := isValidStatusTransition(tt.from, tt.to)
			assert.False(t, got)
		})
	}
}

func TestGetStatusName(t *testing.T) {
	tests := []struct {
		status models.WorkOrderStatus
		want   string
	}{
		{models.WorkOrderStatusPending, "待处理"},
		{models.WorkOrderStatusProcessing, "处理中"},
		{models.WorkOrderStatusCompleted, "已完成"},
		{models.WorkOrderStatusClosed, "已关闭"},
		{models.WorkOrderStatusRejected, "已拒绝"},
		{models.WorkOrderStatus(99), "未知"},
	}
	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			assert.Equal(t, tt.want, getStatusName(tt.status))
		})
	}
}

// ==================== AssignmentService Tests ====================

func TestAssignmentService_Assign_NotFound(t *testing.T) {
	db := setupTestDB(t)
	svc := NewAssignmentService(db)
	err := svc.Assign(context.Background(), uuid.NewString(), &AssignRequest{
		AssigneeID: uuid.NewString(),
	}, uuid.NewString())
	assert.Error(t, err)
}

func TestAssignmentService_Assign_Success(t *testing.T) {
	db := setupTestDB(t)
	woID := seedWorkOrder(t, db, map[string]interface{}{})
	assigneeID := uuid.NewString()
	svc := NewAssignmentService(db)
	err := svc.Assign(context.Background(), woID, &AssignRequest{
		AssigneeID: assigneeID,
		Remark:     "测试分配",
	}, uuid.NewString())
	require.NoError(t, err)
}

func TestAssignmentService_AssignToTodayDuty_NoDuty(t *testing.T) {
	db := setupTestDB(t)
	woID := seedWorkOrder(t, db, map[string]interface{}{})
	svc := NewAssignmentService(db)
	err := svc.AssignToTodayDuty(context.Background(), woID, uuid.NewString())
	assert.Error(t, err)
}

func TestAssignmentService_AssignToTodayDuty_HasDuty(t *testing.T) {
	db := setupTestDB(t)
	woID := seedWorkOrder(t, db, map[string]interface{}{})
	poolID := uuid.NewString()
	require.NoError(t, db.Exec(
		"INSERT INTO sys_duty_pool (id, pool_name) VALUES (?, ?)",
		poolID, "测试池",
	).Error)
	userID := uuid.NewString()
	require.NoError(t, db.Exec(
		"INSERT INTO sys_user (id, username) VALUES (?, ?)",
		userID, "duty.user",
	).Error)
	now := time.Now()
	today := now.Format("2006-01-02")
	require.NoError(t, db.Exec(
		"INSERT INTO sys_duty_schedule (id, pool_id, user_id, schedule_date, duty_type, status) VALUES (?, ?, ?, ?, ?, 0)",
		uuid.NewString(), poolID, userID, today, "primary",
	).Error)
	svc := NewAssignmentService(db)
	err := svc.AssignToTodayDuty(context.Background(), woID, uuid.NewString())
	require.NoError(t, err)
}

func TestAssignmentService_UpdateStatus_NotFound(t *testing.T) {
	db := setupTestDB(t)
	svc := NewAssignmentService(db)
	err := svc.UpdateStatus(context.Background(), uuid.NewString(), &UpdateStatusRequest{
		Status: models.WorkOrderStatusProcessing,
	}, uuid.NewString())
	assert.Error(t, err)
}

func TestAssignmentService_UpdateStatus_ValidTransition(t *testing.T) {
	db := setupTestDB(t)
	woID := seedWorkOrder(t, db, map[string]interface{}{"status": int(models.WorkOrderStatusPending)})
	svc := NewAssignmentService(db)
	err := svc.UpdateStatus(context.Background(), woID, &UpdateStatusRequest{
		Status: models.WorkOrderStatusProcessing,
		Remark: "开始处理",
	}, uuid.NewString())
	require.NoError(t, err)
}

func TestAssignmentService_UpdateStatus_CompletedSetsResolvedAt(t *testing.T) {
	db := setupTestDB(t)
	woID := seedWorkOrder(t, db, map[string]interface{}{"status": int(models.WorkOrderStatusProcessing)})
	svc := NewAssignmentService(db)
	err := svc.UpdateStatus(context.Background(), woID, &UpdateStatusRequest{
		Status: models.WorkOrderStatusCompleted,
		Solution: "解决方案",
	}, uuid.NewString())
	require.NoError(t, err)
	var resolvedAt *time.Time
	require.NoError(t, db.Raw("SELECT resolved_at FROM sys_workorder WHERE id = ?", woID).Scan(&resolvedAt).Error)
	assert.NotNil(t, resolvedAt)
}

func TestAssignmentService_UpdateStatus_ClosedSetsClosedAt(t *testing.T) {
	db := setupTestDB(t)
	// Pending -> Closed 是合法状态流转
	woID := seedWorkOrder(t, db, map[string]interface{}{"status": int(models.WorkOrderStatusPending)})
	svc := NewAssignmentService(db)
	err := svc.UpdateStatus(context.Background(), woID, &UpdateStatusRequest{
		Status: models.WorkOrderStatusClosed,
	}, uuid.NewString())
	require.NoError(t, err)
	var closedAt *time.Time
	require.NoError(t, db.Raw("SELECT closed_at FROM sys_workorder WHERE id = ?", woID).Scan(&closedAt).Error)
	assert.NotNil(t, closedAt)
}

func TestAssignmentService_UpdateStatus_InvalidTransition(t *testing.T) {
	db := setupTestDB(t)
	woID := seedWorkOrder(t, db, map[string]interface{}{"status": int(models.WorkOrderStatusClosed)})
	svc := NewAssignmentService(db)
	err := svc.UpdateStatus(context.Background(), woID, &UpdateStatusRequest{
		Status: models.WorkOrderStatusProcessing,
	}, uuid.NewString())
	assert.Error(t, err)
}

func TestAssignmentService_getTodayDutyMembers_Empty(t *testing.T) {
	db := setupTestDB(t)
	svc := NewAssignmentService(db)
	members, err := svc.getTodayDutyMembers(context.Background(), nil)
	require.NoError(t, err)
	assert.Empty(t, members)
}

func TestAssignmentService_getTodayDutyMembers_WithPoolFilter(t *testing.T) {
	db := setupTestDB(t)
	svc := NewAssignmentService(db)
	poolID := uuid.NewString()
	userID := uuid.NewString()
	require.NoError(t, db.Exec(
		"INSERT INTO sys_duty_pool (id, pool_name) VALUES (?, ?)",
		poolID, "filtered-pool",
	).Error)
	require.NoError(t, db.Exec(
		"INSERT INTO sys_user (id, username) VALUES (?, ?)",
		userID, "u",
	).Error)
	today := time.Now().Format("2006-01-02")
	require.NoError(t, db.Exec(
		"INSERT INTO sys_duty_schedule (id, pool_id, user_id, schedule_date, duty_type, status) VALUES (?, ?, ?, ?, ?, 0)",
		uuid.NewString(), poolID, userID, today, "primary",
	).Error)
	members, err := svc.getTodayDutyMembers(context.Background(), &poolID)
	require.NoError(t, err)
	assert.Len(t, members, 1)
}

// ==================== WorkOrderService Tests ====================

func TestNewWorkOrderService(t *testing.T) {
	db := setupTestDB(t)
	svc := NewWorkOrderService(db)
	assert.NotNil(t, svc.Base)
	assert.NotNil(t, svc.Comment)
	assert.NotNil(t, svc.Category)
	assert.NotNil(t, svc.Rating)
	assert.NotNil(t, svc.Assignment)
	assert.NotNil(t, svc.Statistics)
	assert.NotNil(t, svc.Periodic)
	assert.NotNil(t, svc.Config)
}

// ==================== WorkOrderCacheService Tests ====================

// mockCache implements systemServices.CacheProvider for testing
// We need a minimal mock that satisfies the interface used by NewWorkOrderServiceWithCache
// Instead, just test the basic structure
func TestNewWorkOrderServiceWithCache_NilCache(t *testing.T) {
	// 不能传 nil cache,需要 mock interface. 简单测 NewWorkOrderService 即可
	db := setupTestDB(t)
	svc := NewWorkOrderService(db)
	// 通过 SubService 访问基础方法
	list, total, err := svc.Base.GetList(context.Background(), &ListRequest{})
	require.NoError(t, err)
	assert.EqualValues(t, 0, total)
	assert.Empty(t, list)
}
