package services

// =====================================================================
// Phase 79-03: notice 簇测试(Task 1: notice_service.go CRUD/发布撤回/统计;
// Task 2 追加 notice_read/notice_target/notice_query 可见性链路)。
//
// 覆盖目标: notice_service.go 3.3% → ≥70%(基线 120 stmts / 116 unc),
// notice_read_service.go / notice_target_service.go / notice_query_service.go
// 各 0% → ≥70%(D-79-07: notice_query_service 从 79-01 移入本 plan,
// 因 buildUserVisibleQuery 是 *NoticeService 方法且 notice 表由本 plan 建)。
//
// 纪律:7903 后缀 helper、sqlite t.TempDir 文件库、禁 t.Parallel、
// 断言一律引用 models.PublishStatus*/models.NoticeStatus* 具名常量
// (E 簇反转语义:publish_status 1=已发布,Phase 69-04 判定;与 status
// 启停字段 0=正常 两类语义不可互换 —— notice_query_service.go:39 注释)。
// =====================================================================

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/xingran-next/xingran-go-backend/internal/models"
)

// newNtc7903 装配 NoticeService + sqlite(t.TempDir 文件库)+ AutoMigrate notice 链 model。
// GetNoticeList/GetNoticeByID 会 Preload("Channels") → sys_notification_channel 必须存在;
// buildTargets(dept 型)会走 getChildDeptIDs 的原生 SQL → sys_department 裸表见 QUIRK-79-03-A。
func newNtc7903(t *testing.T) (*NoticeService, *gorm.DB) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "ntc7903.db")), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	require.NoError(t, err, "open sqlite temp db")
	if sqlDB, err := db.DB(); err == nil {
		t.Cleanup(func() { _ = sqlDB.Close() })
	}
	require.NoError(t, db.AutoMigrate(
		&models.Notice{},
		&models.NoticeTarget{},
		&models.NoticeRead{},
		&models.NoticeIgnore{},
		&models.User{},
		&models.UserRole{},
		&models.Role{},
		&models.Department{},
	), "auto migrate notice chain models")
	// sys_notification_channel 不能走 AutoMigrate:NotificationChannel.ID 带
	// PG 专属 `default:gen_random_uuid()`(sqlite 缺该函数 → 语法错,sqlite 缺表
	// family pattern 的 PG-default 变体)。本 plan 只消费 Preload("Channels")
	// 的空集读,按 sanitize 形态手动建裸表。
	require.NoError(t, db.Exec(`CREATE TABLE IF NOT EXISTS sys_notification_channel (
		id TEXT PRIMARY KEY,
		notice_id TEXT NOT NULL,
		channel_type TEXT NOT NULL,
		email_config_id TEXT,
		api_config_id TEXT,
		custom_recipients TEXT,
		created_at DATETIME
	)`).Error, "create sanitized sys_notification_channel table")
	// QUIRK-79-03-A(只锁不修,零生产改动):getChildDeptIDs(notice_target_service.go:116)
	// 的原生 SQL 读取 `sys_department` 表,而 models.Department.TableName() = "sys_dept"
	// —— 两套部门表名并存(sys_department 仅存于 archive 的 init_data.sql,当前 GORM
	// 模型层不映射该表)。本 fixture 显式建 sys_department 裸表以驱动递归分支;
	// 生产语义是否统一属 escape hatch 范畴,不在本 plan 处置。
	require.NoError(t, db.Exec(`CREATE TABLE IF NOT EXISTS sys_department (
		id TEXT PRIMARY KEY,
		parent_id TEXT
	)`).Error, "create raw sys_department table for getChildDeptIDs")
	return NewNoticeService(db), db
}

// strPtr7903 / intPtr7903 / boolPtr7903 计划后缀指针构造(避免同包既有 helper 撞名)。
func strPtr7903(s string) *string { return &s }
func intPtr7903(i int) *int       { return &i }
func boolPtr7903(b bool) *bool    { return &b }

// ntc7903Dept 在 sys_department 裸表(QUirk-79-03-A 专用)落一行部门。
func ntc7903Dept(t *testing.T, db *gorm.DB, id, parentID string) {
	t.Helper()
	require.NoError(t, db.Exec(
		`INSERT INTO sys_department (id, parent_id) VALUES (?, ?)`, id, parentID,
	).Error, "seed raw dept %s", id)
}

// ntc7903Notice 以 service 路径建一条通知,失败即终止测试。
func ntc7903Notice(t *testing.T, s *NoticeService, req *CreateNoticeRequest, creatorID, creatorName string) *models.Notice {
	t.Helper()
	notice, err := s.CreateNoticeWithTargets(context.Background(), req, creatorID, creatorName)
	require.NoError(t, err, "create notice %s", req.NoticeTitle)
	return notice
}

// ntc7903CreateReq 构造最小合法创建请求(noticeType 1=公告)。
func ntc7903CreateReq(title, content string, priority models.NoticePriority) *CreateNoticeRequest {
	return &CreateNoticeRequest{
		NoticeTitle:   title,
		NoticeType:    "1",
		NoticeContent: content,
		Priority:      priority,
	}
}

// TestNtc7903_CreateWithTargets_AllTargetTypes
// target_type ∈ {0 全体, 1 部门, 2 角色, 3 指定用户} 四类请求各建一次,
// 断言 sys_notice 行 + sys_notice_target 行数与 target_type/target_id 形态。
func TestNtc7903_CreateWithTargets_AllTargetTypes(t *testing.T) {
	svc, db := newNtc7903(t)

	// dept 型前置:sys_department 裸表父子两层(buildTargets 递归展开子部门)
	ntc7903Dept(t, db, "dept-parent", "")
	ntc7903Dept(t, db, "dept-child", "dept-parent")
	ntc7903Dept(t, db, "dept-grandchild", "dept-child")

	// (a) target_type = 0 全体:不写 sys_notice_target 行
	all := ntc7903Notice(t, svc, func() *CreateNoticeRequest {
		r := ntc7903CreateReq("全体通知", "content-all", models.PriorityNormal)
		r.TargetType = models.TargetAll
		return r
	}(), "creator-1", "张创建")
	assert.NotEmpty(t, all.ID, "BaseModel.BeforeCreate 生成 UUID")
	assert.Equal(t, "张创建", all.CreatedByName, "creatorName 落 CreatedByName")
	// creatorID 实装不落 sys_notice(仅 CreatedByName 持久化),锁定现行为
	assert.Empty(t, all.CreatedBy, "creatorID 不落库(实装仅透传 CreatedByName)")
	assert.Equal(t, models.PublishStatusDraft, all.PublishStatus, "无定时发布时间 → 草稿")
	assert.Equal(t, int(models.NoticeStatusNormal), all.Status, "新建即启停字段=正常")
	var allTargets int64
	require.NoError(t, db.Model(&models.NoticeTarget{}).Where("notice_id = ?", all.ID).Count(&allTargets).Error)
	assert.Zero(t, allTargets, "TargetAll 不写目标行")

	// (b) target_type = 1 部门:父部门 + 递归子/孙部门各一行(全为 "dept" 形态)
	dept := ntc7903Notice(t, svc, func() *CreateNoticeRequest {
		r := ntc7903CreateReq("部门通知", "content-dept", models.PriorityImportant)
		r.TargetType = models.TargetDept
		r.TargetDepts = []string{"dept-parent"}
		return r
	}(), "creator-2", "李创建")
	var deptTargets []models.NoticeTarget
	require.NoError(t, db.Where("notice_id = ?", dept.ID).Find(&deptTargets).Error)
	require.Len(t, deptTargets, 3, "父+子+孙三层递归展开(见 QUIRK-79-03-A)")
	deptIDs := make([]string, 0, len(deptTargets))
	for _, tg := range deptTargets {
		assert.Equal(t, "dept", tg.TargetType)
		assert.Equal(t, dept.ID, tg.NoticeID)
		deptIDs = append(deptIDs, tg.TargetID)
	}
	assert.ElementsMatch(t, []string{"dept-parent", "dept-child", "dept-grandchild"}, deptIDs)

	// (c) target_type = 2 角色:每个角色一行 "role" 形态
	role := ntc7903Notice(t, svc, func() *CreateNoticeRequest {
		r := ntc7903CreateReq("角色通知", "content-role", models.PriorityNormal)
		r.TargetType = models.TargetRole
		r.TargetRoles = []string{"role-a", "role-b"}
		return r
	}(), "creator-3", "王创建")
	var roleTargets []models.NoticeTarget
	require.NoError(t, db.Where("notice_id = ?", role.ID).Find(&roleTargets).Error)
	require.Len(t, roleTargets, 2)
	roleIDs := make([]string, 0, len(roleTargets))
	for _, tg := range roleTargets {
		assert.Equal(t, "role", tg.TargetType)
		roleIDs = append(roleIDs, tg.TargetID)
	}
	assert.ElementsMatch(t, []string{"role-a", "role-b"}, roleIDs)

	// (d) target_type = 3 指定用户:每个用户一行 "user" 形态
	usr := ntc7903Notice(t, svc, func() *CreateNoticeRequest {
		r := ntc7903CreateReq("用户通知", "content-user", models.PriorityUrgent)
		r.TargetType = models.TargetUser
		r.TargetUsers = []string{"user-1", "user-2"}
		return r
	}(), "creator-4", "赵创建")
	var userTargets []models.NoticeTarget
	require.NoError(t, db.Where("notice_id = ?", usr.ID).Find(&userTargets).Error)
	require.Len(t, userTargets, 2)
	userIDs := make([]string, 0, len(userTargets))
	for _, tg := range userTargets {
		assert.Equal(t, "user", tg.TargetType)
		userIDs = append(userIDs, tg.TargetID)
	}
	assert.ElementsMatch(t, []string{"user-1", "user-2"}, userIDs)

	// (e) 定时发布:未来 PublishTime → PublishStatusScheduled(E 簇 2=定时发布中)
	future := time.Now().Add(24 * time.Hour).UTC().Truncate(time.Second)
	scheduled := ntc7903Notice(t, svc, func() *CreateNoticeRequest {
		r := ntc7903CreateReq("定时通知", "content-sched", models.PriorityNormal)
		r.PublishTime = &future
		return r
	}(), "creator-5", "钱创建")
	assert.Equal(t, models.PublishStatusScheduled, scheduled.PublishStatus, "未来时间 → 定时发布中")
	require.NotNil(t, scheduled.PublishTime)
	assert.True(t, scheduled.PublishTime.Equal(future), "PublishTime 透传落库")

	// (f) 过去 PublishTime → 仍为草稿(实装语义:非未来时间不进入定时态)
	past := time.Now().Add(-24 * time.Hour)
	pastNotice := ntc7903Notice(t, svc, func() *CreateNoticeRequest {
		r := ntc7903CreateReq("过去时间通知", "content-past", models.PriorityNormal)
		r.PublishTime = &past
		return r
	}(), "creator-6", "孙创建")
	assert.Equal(t, models.PublishStatusDraft, pastNotice.PublishStatus, "过去时间保持草稿(锁定现行为)")

	// (g) 非法 target_type(越界值)不写目标行(switch 无 default 分支)
	weird := ntc7903Notice(t, svc, func() *CreateNoticeRequest {
		r := ntc7903CreateReq("越界目标类型", "content-weird", models.PriorityNormal)
		r.TargetType = models.TargetType(9)
		return r
	}(), "creator-7", "周创建")
	var weirdTargets int64
	require.NoError(t, db.Model(&models.NoticeTarget{}).Where("notice_id = ?", weird.ID).Count(&weirdTargets).Error)
	assert.Zero(t, weirdTargets, "switch 未命中的 target_type 不写目标行")
}

// TestNtc7903_GetList_PaginationFilterSort
// 预置 5 行 → 分页 + title 模糊 + noticeType 精确 + 白名单排序/非法列回退。
func TestNtc7903_GetList_PaginationFilterSort(t *testing.T) {
	svc, _ := newNtc7903(t)
	ctx := context.Background()

	// 5 行:3 条 type=1、2 条 type=2;优先级与创建序可控
	type spec struct {
		title    string
		ntype    string
		priority models.NoticePriority
	}
	specs := []spec{
		{"开机公告甲", "1", models.PriorityNormal},
		{"停电通知乙", "2", models.PriorityUrgent},
		{"开机公告丙", "1", models.PriorityImportant},
		{"巡检通知丁", "2", models.PriorityNormal},
		{"开机公告戊", "1", models.PriorityUrgent},
	}
	for _, sp := range specs {
		req := ntc7903CreateReq(sp.title, "c-"+sp.title, sp.priority)
		req.NoticeType = sp.ntype
		ntc7903Notice(t, svc, req, "creator", "创建人")
	}
	time.Sleep(2 * time.Millisecond) // 保证 created_at 可区分(sqlite 文本时间精度)

	asc := true
	// 分页:page=1/size=2 与 page=3/size=2(尾页 1 行)
	list, total, err := svc.GetNoticeList(ctx, 1, 2, nil, nil, "", nil)
	require.NoError(t, err)
	assert.Equal(t, int64(5), total)
	require.Len(t, list, 2)
	list, total, err = svc.GetNoticeList(ctx, 3, 2, nil, nil, "", nil)
	require.NoError(t, err)
	assert.Equal(t, int64(5), total)
	require.Len(t, list, 1)

	// title 模糊
	list, total, err = svc.GetNoticeList(ctx, 1, 10, strPtr7903("开机公告"), nil, "", nil)
	require.NoError(t, err)
	assert.Equal(t, int64(3), total, "title LIKE %开机公告%")

	// noticeType 精确
	list, total, err = svc.GetNoticeList(ctx, 1, 10, nil, strPtr7903("2"), "", nil)
	require.NoError(t, err)
	assert.Equal(t, int64(2), total, "notice_type = 2")
	for _, n := range list {
		assert.Equal(t, "2", n.NoticeType)
	}

	// 组合过滤:title + noticeType
	_, total, err = svc.GetNoticeList(ctx, 1, 10, strPtr7903("停电"), strPtr7903("2"), "", nil)
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)

	// 白名单排序:orderByColumn=priority 升序 → 首行 priority=Normal
	list, total, err = svc.GetNoticeList(ctx, 1, 10, nil, nil, "priority", &asc)
	require.NoError(t, err)
	assert.Equal(t, int64(5), total)
	require.NotEmpty(t, list)
	assert.Equal(t, models.PriorityNormal, list[0].Priority, "priority ASC 首行必为普通优先级")

	// 白名单排序:orderByColumn=priority 降序 → 首行 priority=Urgent
	list, _, err = svc.GetNoticeList(ctx, 1, 10, nil, nil, "priority", boolPtr7903(false))
	require.NoError(t, err)
	require.NotEmpty(t, list)
	assert.Equal(t, models.PriorityUrgent, list[0].Priority, "priority DESC 首行必为紧急优先级")

	// QUIRK-79-03-B(锁定不修):非法 orderByColumn 走 base.ApplySort 白名单回退
	// (仅 warn 日志、不追加 Order);又因 GetNoticeList 仅在 orderByColumn==""
	// 时补默认 "priority DESC, created_at DESC" → 非法列时退化为 sqlite 自然序。
	// 断言:无错误 + 总数正确,不锁具体顺序(实现细节)。
	list, total, err = svc.GetNoticeList(ctx, 1, 10, nil, nil, "1; DROP TABLE sys_notice", &asc)
	require.NoError(t, err, "非法列必须被白名单拦截且不报错")
	assert.Equal(t, int64(5), total)
	assert.Len(t, list, 5)
	var noticesLeft int64
	require.NoError(t, svc.db.Model(&models.Notice{}).Count(&noticesLeft).Error)
	assert.Equal(t, int64(5), noticesLeft, "非法排序输入不得影响表数据")

	// 空 orderByColumn → 默认排序(priority DESC, created_at DESC)
	list, _, err = svc.GetNoticeList(ctx, 1, 10, nil, nil, "", nil)
	require.NoError(t, err)
	require.Len(t, list, 5)
	assert.Equal(t, models.PriorityUrgent, list[0].Priority, "默认排序首行 priority DESC")
}

// TestNtc7903_GetByID_Miss 不存在 → 错误;存在 → 字段与目标行一致。
func TestNtc7903_GetByID_Miss(t *testing.T) {
	svc, _ := newNtc7903(t)
	ctx := context.Background()

	_, err := svc.GetNoticeByID(ctx, "no-such-notice")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "通知不存在")

	req := ntc7903CreateReq("查询目标通知", "全文内容", models.PriorityImportant)
	req.TargetType = models.TargetUser
	req.TargetUsers = []string{"reader-1"}
	created := ntc7903Notice(t, svc, req, "creator-x", "创建人甲")

	got, err := svc.GetNoticeByID(ctx, created.ID)
	require.NoError(t, err)
	assert.Equal(t, created.ID, got.ID)
	assert.Equal(t, "查询目标通知", got.NoticeTitle)
	assert.Equal(t, "全文内容", got.NoticeContent)
	assert.Equal(t, models.PriorityImportant, got.Priority)
	assert.Equal(t, models.PublishStatusDraft, got.PublishStatus)
	require.Len(t, got.Targets, 1, "Preload(Targets) 装载目标行")
	assert.Equal(t, "reader-1", got.Targets[0].TargetID)
	assert.Empty(t, got.Channels, "Preload(Channels) 无行时空切片")
}

// TestNtc7903_UpdateNotice_FullAndMissing 合法更新读回一致;不存在 ID → 错误;
// 空更新集 → 错误;PublishStatus 显式覆盖优先于 PublishTime 推算。
func TestNtc7903_UpdateNotice_FullAndMissing(t *testing.T) {
	svc, _ := newNtc7903(t)
	ctx := context.Background()

	created := ntc7903Notice(t, svc, ntc7903CreateReq("更新前标题", "更新前内容", models.PriorityNormal), "c", "创建人")

	newTitle := "更新后标题"
	newType := "2"
	newContent := "更新后内容"
	newPriority := models.PriorityUrgent
	closed := int(models.NoticeStatusClosed)
	require.NoError(t, svc.UpdateNotice(ctx, created.ID, &UpdateNoticeRequest{
		NoticeTitle:   &newTitle,
		NoticeType:    &newType,
		NoticeContent: &newContent,
		Priority:      &newPriority,
		Status:        &closed,
	}))

	got, err := svc.GetNoticeByID(ctx, created.ID)
	require.NoError(t, err)
	assert.Equal(t, "更新后标题", got.NoticeTitle)
	assert.Equal(t, "2", got.NoticeType)
	assert.Equal(t, "更新后内容", got.NoticeContent)
	assert.Equal(t, models.PriorityUrgent, got.Priority)
	assert.Equal(t, closed, got.Status, "启停字段更新为关闭(NoticeStatusClosed=1)")

	// PublishTime 未来时间 → 推算为定时发布中
	future := time.Now().Add(48 * time.Hour)
	require.NoError(t, svc.UpdateNotice(ctx, created.ID, &UpdateNoticeRequest{PublishTime: &future}))
	got, err = svc.GetNoticeByID(ctx, created.ID)
	require.NoError(t, err)
	assert.Equal(t, models.PublishStatusScheduled, got.PublishStatus, "未来 PublishTime → 定时发布中")

	// ClearPublishTime → 清时间并回草稿
	require.NoError(t, svc.UpdateNotice(ctx, created.ID, &UpdateNoticeRequest{ClearPublishTime: true}))
	got, err = svc.GetNoticeByID(ctx, created.ID)
	require.NoError(t, err)
	assert.Nil(t, got.PublishTime, "ClearPublishTime 清空 publish_time")
	assert.Equal(t, models.PublishStatusDraft, got.PublishStatus)

	// 显式 PublishStatus 覆盖优先级高于推算(E 簇 1=已发布)
	pub := models.PublishStatusPublished
	require.NoError(t, svc.UpdateNotice(ctx, created.ID, &UpdateNoticeRequest{PublishStatus: &pub}))
	got, err = svc.GetNoticeByID(ctx, created.ID)
	require.NoError(t, err)
	assert.Equal(t, models.PublishStatusPublished, got.PublishStatus, "显式覆盖直接写入")

	// EndDate 更新(Phase 34 WR-003 字段)
	end := time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)
	require.NoError(t, svc.UpdateNotice(ctx, created.ID, &UpdateNoticeRequest{EndDate: &end}))
	got, err = svc.GetNoticeByID(ctx, created.ID)
	require.NoError(t, err)
	require.NotNil(t, got.EndDate)
	assert.True(t, got.EndDate.Equal(end), "end_date 落库读回")

	// 空 updates → 明确报错
	require.ErrorContains(t, svc.UpdateNotice(ctx, created.ID, &UpdateNoticeRequest{}), "没有需要更新的字段")
	// 不存在 ID(有合法字段)→ RowsAffected==0 分支
	require.ErrorContains(t, svc.UpdateNotice(ctx, "ghost-id", &UpdateNoticeRequest{
		NoticeTitle: &newTitle,
	}), "通知不存在")
}

// TestNtc7903_DeleteAndBatchDelete 单删/批删后计数;软删语义(Unscoped 可见)。
func TestNtc7903_DeleteAndBatchDelete(t *testing.T) {
	svc, db := newNtc7903(t)
	ctx := context.Background()

	n1 := ntc7903Notice(t, svc, ntc7903CreateReq("待删甲", "c", models.PriorityNormal), "c", "创建人")
	n2 := ntc7903Notice(t, svc, ntc7903CreateReq("待删乙", "c", models.PriorityNormal), "c", "创建人")
	n3 := ntc7903Notice(t, svc, ntc7903CreateReq("留存丙", "c", models.PriorityNormal), "c", "创建人")

	// 不存在 → 明确报错
	require.ErrorContains(t, svc.DeleteNotice(ctx, "ghost-id"), "通知不存在")

	require.NoError(t, svc.DeleteNotice(ctx, n1.ID))
	var live int64
	require.NoError(t, db.Model(&models.Notice{}).Count(&live).Error)
	assert.Equal(t, int64(2), live, "单删后可见计数 2")

	// 软删语义:BaseModel.DeletedAt 存在 → Unscoped 仍可见
	var softDeleted int64
	require.NoError(t, db.Unscoped().Model(&models.Notice{}).Where("id = ?", n1.ID).Count(&softDeleted).Error)
	assert.Equal(t, int64(1), softDeleted, "软删行 Unscoped 可见(glebarez sqlite deleted_at 非空)")

	// 批删:2 条全删;空列表 → 明确报错
	require.NoError(t, svc.BatchDeleteNotices(ctx, []string{n2.ID, n3.ID}))
	require.NoError(t, db.Model(&models.Notice{}).Count(&live).Error)
	assert.Equal(t, int64(0), live, "批删后可见计数 0")
	require.ErrorContains(t, svc.BatchDeleteNotices(ctx, []string{}), "通知ID列表不能为空")

	// 批删含不存在 ID → 实装不校验 RowsAffected,静默成功(锁定现行为)
	require.NoError(t, svc.BatchDeleteNotices(ctx, []string{"ghost-id"}))
}

// TestNtc7903_PublishAndWithdraw 草稿 → 发布(E 簇 1=已发布)→ 撤回;
// 重复发布/撤回未发布/不存在 ID 分支锁定。
func TestNtc7903_PublishAndWithdraw(t *testing.T) {
	svc, _ := newNtc7903(t)
	ctx := context.Background()

	draft := ntc7903Notice(t, svc, ntc7903CreateReq("发布流通知", "c", models.PriorityNormal), "c", "创建人")

	// 草稿 → 发布
	require.NoError(t, svc.PublishNotice(ctx, draft.ID))
	got, err := svc.GetNoticeByID(ctx, draft.ID)
	require.NoError(t, err)
	// E 簇反转语义:publish_status 1=已发布(models.PublishStatusPublished),
	// 与启停字段 status 0=正常 两套语义不可互换(Phase 69-04 判定)。
	assert.Equal(t, models.PublishStatusPublished, got.PublishStatus, "发布后 = PublishStatusPublished(E 簇 1=已发布)")
	require.NotNil(t, got.PublishTime, "手动发布(publish_time 为空)→ 写入当前时间")

	// 重复发布 → 明确报错
	require.ErrorContains(t, svc.PublishNotice(ctx, draft.ID), "通知已经发布")

	// 已发布 → 撤回(实装:回草稿态,QUIRK-79-03-C)
	require.NoError(t, svc.WithdrawNotice(ctx, draft.ID))
	got, err = svc.GetNoticeByID(ctx, draft.ID)
	require.NoError(t, err)
	assert.Equal(t, models.PublishStatusDraft, got.PublishStatus,
		"撤回后回草稿态(QUIRK-79-03-C:实装写 PublishStatusDraft 而非 PublishStatusWithdrawn)")

	// 撤回未发布 → 明确报错
	require.ErrorContains(t, svc.WithdrawNotice(ctx, draft.ID), "只有已发布的通知可以撤回")

	// 定时发布中的通知也可发布,且保留原 publish_time
	future := time.Now().Add(72 * time.Hour).UTC().Truncate(time.Second)
	scheduled := ntc7903Notice(t, svc, func() *CreateNoticeRequest {
		r := ntc7903CreateReq("定时发布通知", "c", models.PriorityNormal)
		r.PublishTime = &future
		return r
	}(), "c", "创建人")
	require.NoError(t, svc.PublishNotice(ctx, scheduled.ID))
	got, err = svc.GetNoticeByID(ctx, scheduled.ID)
	require.NoError(t, err)
	assert.Equal(t, models.PublishStatusPublished, got.PublishStatus)
	require.NotNil(t, got.PublishTime)
	assert.True(t, got.PublishTime.Equal(future), "已有 publish_time 的定时通知发布时保留原时间")

	// PublishStatusWithdrawn(3)仅可由 UpdateNotice 显式写入;此态下发布被拒
	withdrawn := ntc7903Notice(t, svc, ntc7903CreateReq("已撤回态通知", "c", models.PriorityNormal), "c", "创建人")
	wStat := models.PublishStatusWithdrawn
	require.NoError(t, svc.UpdateNotice(ctx, withdrawn.ID, &UpdateNoticeRequest{PublishStatus: &wStat}))
	require.ErrorContains(t, svc.PublishNotice(ctx, withdrawn.ID), "通知已撤回，无法重新发布")

	// 不存在 ID:发布/撤回均报"通知不存在"
	require.ErrorContains(t, svc.PublishNotice(ctx, "ghost-id"), "通知不存在")
	require.ErrorContains(t, svc.WithdrawNotice(ctx, "ghost-id"), "通知不存在")
}

// TestNtc7903_StatusStatistics 预置各发布状态行 → 各计数与手算一致
// (口径对照既有 notice_status_statistics_test.go,该文件不动,本用例走 GORM 建行路径)。
func TestNtc7903_StatusStatistics(t *testing.T) {
	svc, _ := newNtc7903(t)
	ctx := context.Background()

	seed := func(title string, ps models.PublishStatus) {
		t.Helper()
		require.NoError(t, svc.db.Create(&models.Notice{
			NoticeTitle:   title,
			NoticeType:    "1",
			NoticeContent: "content",
			PublishStatus: ps,
			Status:        int(models.NoticeStatusNormal),
		}).Error, "seed notice %s", title)
	}
	// 2 草稿(0) + 3 已发布(1) + 1 定时(2) + 1 撤回(3) = 7
	seed("s-draft-1", models.PublishStatusDraft)
	seed("s-draft-2", models.PublishStatusDraft)
	seed("s-pub-1", models.PublishStatusPublished)
	seed("s-pub-2", models.PublishStatusPublished)
	seed("s-pub-3", models.PublishStatusPublished)
	seed("s-sched-1", models.PublishStatusScheduled)
	seed("s-wdraw-1", models.PublishStatusWithdrawn)

	result, err := svc.GetNoticeStatusStatistics(ctx)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, int64(7), result.Total)
	assert.Equal(t, int64(3), result.Published, "published = PublishStatusPublished 计数")
	assert.Equal(t, int64(2), result.Draft, "draft = PublishStatusDraft 计数(不含定时发布)")
	assert.Equal(t, int64(1), result.Scheduled, "scheduled = PublishStatusScheduled 计数")

	// 软删行必须排除(BaseModel.DeletedAt)
	var ghost models.Notice
	require.NoError(t, svc.db.First(&ghost, "notice_title = ?", "s-pub-1").Error)
	require.NoError(t, svc.db.Delete(&ghost).Error)
	result, err = svc.GetNoticeStatusStatistics(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(6), result.Total, "软删行排除")
	assert.Equal(t, int64(2), result.Published, "软删的已发布行不再计入")
}
