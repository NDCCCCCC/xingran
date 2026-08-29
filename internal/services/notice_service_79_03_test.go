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
	_, total, err = svc.GetNoticeList(ctx, 1, 10, strPtr7903("开机公告"), nil, "", nil)
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

// =====================================================================
// Task 2: notice_read / notice_target / notice_query(可见性链路)
// =====================================================================

// ntc7903User 落 sys_user 行(+可选 sys_user_role 行),Password/Salt 满足 not null。
func ntc7903User(t *testing.T, db *gorm.DB, id string, deptID *string, roleIDs ...string) {
	t.Helper()
	require.NoError(t, db.Create(&models.User{
		BaseModel: models.BaseModel{ID: id},
		Username:  "user-" + id,
		Password:  "not-a-real-password",
		Salt:      "salt-7903",
		DeptID:    deptID,
	}).Error, "seed user %s", id)
	for _, roleID := range roleIDs {
		require.NoError(t, db.Create(&models.UserRole{UserID: id, RoleID: roleID}).Error,
			"seed user_role %s->%s", id, roleID)
	}
}

// ntc7903Publish 走 service 发布路径(草稿 → PublishStatusPublished)。
func ntc7903Publish(t *testing.T, s *NoticeService, noticeID string) {
	t.Helper()
	require.NoError(t, s.PublishNotice(context.Background(), noticeID), "publish notice %s", noticeID)
}

// ntc7903VisibleIDs 以 buildUserVisibleQuery 的产物实际执行查询,返回可见通知 ID 集。
func ntc7903VisibleIDs(t *testing.T, qctx *UserNoticeQueryContext) []string {
	t.Helper()
	var ids []string
	require.NoError(t, qctx.Query.Pluck("id", &ids).Error, "execute visible query")
	return ids
}

// TestNrd7903_ReadChain 已读链:标记已读落行 + ip 落库 + 重复读幂等 +
// GetUnreadCount 递减 + MarkAllNoticesRead 归零。
func TestNrd7903_ReadChain(t *testing.T) {
	svc, db := newNtc7903(t)
	ctx := context.Background()
	deptID := "dept-r1"
	ntc7903User(t, db, "reader-u1", &deptID, "role-r1")

	n1 := ntc7903Notice(t, svc, ntc7903CreateReq("已读甲", "c", models.PriorityNormal), "c", "创建人")
	n2 := ntc7903Notice(t, svc, ntc7903CreateReq("已读乙", "c", models.PriorityNormal), "c", "创建人")
	ntc7903Publish(t, svc, n1.ID)
	ntc7903Publish(t, svc, n2.ID)

	unread, err := svc.GetUnreadCount(ctx, "reader-u1")
	require.NoError(t, err)
	assert.Equal(t, 2, unread, "初始未读 2")

	// 标记已读:落 sys_notice_read 行 + ReadIP 透传
	require.NoError(t, svc.MarkNoticeRead(ctx, n1.ID, "reader-u1", "10.0.0.9"))
	var read models.NoticeRead
	require.NoError(t, db.Where("notice_id = ? AND user_id = ?", n1.ID, "reader-u1").First(&read).Error)
	assert.Equal(t, "10.0.0.9", read.ReadIP, "ip 参数落 ReadIP")
	assert.False(t, read.ReadAt.IsZero())

	unread, err = svc.GetUnreadCount(ctx, "reader-u1")
	require.NoError(t, err)
	assert.Equal(t, 1, unread, "读一条后未读递减")

	// 重复标记:实装先查已读计数,命中即跳过 → 不重复落行
	require.NoError(t, svc.MarkNoticeRead(ctx, n1.ID, "reader-u1", "10.0.0.9"))
	var readRows int64
	require.NoError(t, db.Model(&models.NoticeRead{}).Where("notice_id = ? AND user_id = ?", n1.ID, "reader-u1").Count(&readRows).Error)
	assert.Equal(t, int64(1), readRows, "重复已读不重复计")

	// QUIRK-79-03-D(锁定不修):MarkAllNoticesRead 的可见集口径 = 全部
	// 已发布+正常 通知(notice_read_service.go:39-41 不带 target/ignore 过滤),
	// 与 buildUserVisibleQuery 的 4-OR 口径不同;本用例两条通知均可见,两口径等价。
	// QUIRK-79-03-F(锁定不修):MarkAllNoticesRead 不校验既有已读行 → n1 重复
	// 落一条已读记录(共 3 行);GetUnreadCount 以「可见总数 - 已读行数」计,负值
	// 夹为 0,恰好掩盖了重复行 —— 两 quirk 叠加后未读数仍归 0。
	require.NoError(t, svc.MarkAllNoticesRead(ctx, "reader-u1"))
	unread, err = svc.GetUnreadCount(ctx, "reader-u1")
	require.NoError(t, err)
	assert.Equal(t, 0, unread, "全量已读后未读归零(含 QUIRK-79-03-F 的夹 0)")
	require.NoError(t, db.Model(&models.NoticeRead{}).Where("user_id = ?", "reader-u1").Count(&readRows).Error)
	assert.Equal(t, int64(3), readRows, "n1×2(重复)+ n2×1:全量已读不去重(QUIRK-79-03-F)")
}

// TestNrd7903_IgnoreChain 忽略链:忽略出现 → 幂等 → 取消消失 → 重复取消报错。
func TestNrd7903_IgnoreChain(t *testing.T) {
	svc, db := newNtc7903(t)
	ctx := context.Background()
	ntc7903User(t, db, "ignorer-u1", nil)

	n1 := ntc7903Notice(t, svc, ntc7903CreateReq("忽略甲", "c", models.PriorityNormal), "c", "创建人")
	n2 := ntc7903Notice(t, svc, ntc7903CreateReq("忽略乙", "c", models.PriorityNormal), "c", "创建人")
	ntc7903Publish(t, svc, n1.ID)
	ntc7903Publish(t, svc, n2.ID)

	// 空忽略集:空切片 + total 0(不查库分支)
	notices, total, err := svc.GetIgnoredNotices(ctx, "ignorer-u1", 1, 10)
	require.NoError(t, err)
	assert.Empty(t, notices)
	assert.Zero(t, total)

	require.NoError(t, svc.IgnoreNotice(ctx, n1.ID, "ignorer-u1"))
	notices, total, err = svc.GetIgnoredNotices(ctx, "ignorer-u1", 1, 10)
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	require.Len(t, notices, 1)
	assert.Equal(t, n1.ID, notices[0].ID)

	// 重复忽略:实装先查计数命中即跳过 → 幂等
	require.NoError(t, svc.IgnoreNotice(ctx, n1.ID, "ignorer-u1"))
	var ignoreRows int64
	require.NoError(t, db.Model(&models.NoticeIgnore{}).Where("user_id = ?", "ignorer-u1").Count(&ignoreRows).Error)
	assert.Equal(t, int64(1), ignoreRows, "重复忽略幂等")

	// QUIRK-79-03-E(锁定不修,⚠️ 现网可见):models.NoticeIgnore 的唯一索引
	// `idx_notice_ignore_user_notice` 标签只含 UserID 一个成员(NoticeID 仅挂在
	// 非 unique 的 idx_notice_ignore_notice_id)→ 唯一索引实为 user_id 单列,
	// 一个用户至多忽略一条通知;忽略第二条直接撞 UNIQUE 约束。
	// 修复需动 model 标签(生产 schema 变更),属 escape hatch 范畴,本 plan 只锁。
	err = svc.IgnoreNotice(ctx, n2.ID, "ignorer-u1")
	require.Error(t, err, "第二忽略命中 user_id 单列唯一索引")
	assert.Contains(t, err.Error(), "UNIQUE constraint failed")

	// 分页:size=1 单行 → 1 行、total 1
	notices, total, err = svc.GetIgnoredNotices(ctx, "ignorer-u1", 1, 1)
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Len(t, notices, 1)

	// 取消忽略 → 从列表消失;再次取消 → "该通知未被忽略"
	require.NoError(t, svc.UnignoreNotice(ctx, n1.ID, "ignorer-u1"))
	_, total, err = svc.GetIgnoredNotices(ctx, "ignorer-u1", 1, 10)
	require.NoError(t, err)
	assert.Zero(t, total, "取消后忽略集为空")
	require.ErrorContains(t, svc.UnignoreNotice(ctx, n1.ID, "ignorer-u1"), "该通知未被忽略")
}

// TestNrd7903_GetUserNotices_StatusFilter 用户可见列表:status 过滤 nil/read/unread
// + 未知值不过滤 + 分页正确 + 用户缺失报错。
func TestNrd7903_GetUserNotices_StatusFilter(t *testing.T) {
	svc, db := newNtc7903(t)
	ctx := context.Background()
	deptD1, deptD2 := "dept-d1", "dept-d2"
	ntc7903User(t, db, "list-u1", &deptD1, "role-r1")
	ntc7903User(t, db, "list-u2", &deptD2)

	// 6 条已发布 + 1 草稿;target 形态各异(可见集口径见矩阵用例)
	seed := func(title string, targetType models.TargetType, ids []string) *models.Notice {
		t.Helper()
		req := ntc7903CreateReq(title, "c", models.PriorityNormal)
		req.TargetType = targetType
		switch targetType {
		case models.TargetDept:
			req.TargetDepts = ids
		case models.TargetRole:
			req.TargetRoles = ids
		case models.TargetUser:
			req.TargetUsers = ids
		}
		n := ntc7903Notice(t, svc, req, "c", "创建人")
		ntc7903Publish(t, svc, n.ID)
		return n
	}
	nAll := seed("列表-全体", models.TargetAll, nil)
	seed("列表-部门命中", models.TargetDept, []string{deptD1})
	seed("列表-部门未命中", models.TargetDept, []string{deptD2})
	seed("列表-角色命中", models.TargetRole, []string{"role-r1"})
	seed("列表-角色未命中", models.TargetRole, []string{"role-rx"})
	seed("列表-用户命中", models.TargetUser, []string{"list-u1"})
	draft := ntc7903Notice(t, svc, ntc7903CreateReq("列表-草稿", "c", models.PriorityNormal), "c", "创建人")
	_ = draft // 保持草稿态,不发布
	_ = nAll

	// nil status → 不过滤已读,只走可见性
	list, total, err := svc.GetUserNotices(ctx, "list-u1", 1, 10, nil)
	require.NoError(t, err)
	assert.Equal(t, int64(4), total, "可见 = 全体 + 部门命中 + 角色命中 + 用户命中")
	require.Len(t, list, 4)

	// unread:全部未读 → 4
	unreadStatus := "unread"
	_, total, err = svc.GetUserNotices(ctx, "list-u1", 1, 10, &unreadStatus)
	require.NoError(t, err)
	assert.Equal(t, int64(4), total)

	// 读一条 → read=1 / unread=3
	require.NoError(t, svc.MarkNoticeRead(ctx, nAll.ID, "list-u1", "127.0.0.1"))
	readStatus := "read"
	_, total, err = svc.GetUserNotices(ctx, "list-u1", 1, 10, &readStatus)
	require.NoError(t, err)
	assert.Equal(t, int64(1), total, "read 子查询仅命中已读行")
	_, total, err = svc.GetUserNotices(ctx, "list-u1", 1, 10, &unreadStatus)
	require.NoError(t, err)
	assert.Equal(t, int64(3), total)

	// 未知 status 值:switch 无分支 → 不过滤(锁定现行为)
	weird := "weird"
	_, total, err = svc.GetUserNotices(ctx, "list-u1", 1, 10, &weird)
	require.NoError(t, err)
	assert.Equal(t, int64(4), total, "未知 status 值等价于不过滤")

	// 分页:size=3 → 第 1 页 3 行、第 2 页 1 行,total 恒 4
	_, total, err = svc.GetUserNotices(ctx, "list-u1", 1, 3, nil)
	require.NoError(t, err)
	assert.Equal(t, int64(4), total)
	list, total, err = svc.GetUserNotices(ctx, "list-u1", 2, 3, nil)
	require.NoError(t, err)
	assert.Equal(t, int64(4), total)
	assert.Len(t, list, 1, "尾页余 1 行")

	// 用户缺失 → buildUserVisibleQuery 报"获取用户信息失败"
	_, _, err = svc.GetUserNotices(ctx, "ghost-user", 1, 10, nil)
	require.ErrorContains(t, err, "获取用户信息失败")
}

// TestNtg7903_GetTargetUsers_FourTypes 四类目标的用户解析 + getTargetIDsByType/unique 直调。
func TestNtg7903_GetTargetUsers_FourTypes(t *testing.T) {
	svc, db := newNtc7903(t)
	ctx := context.Background()

	// 部门树(sys_department 裸表,QUIRK-79-03-A)
	ntc7903Dept(t, db, "dept-t1", "")
	ntc7903Dept(t, db, "dept-t1-child", "dept-t1")
	// 用户:d1 主部门 / child 子部门 / 双角色 / 仅 r2 / 无归属
	ntc7903User(t, db, "u-d1", strPtr7903("dept-t1"))
	ntc7903User(t, db, "u-child", strPtr7903("dept-t1-child"))
	ntc7903User(t, db, "u-role1", strPtr7903("dept-t1"), "role-r1")
	ntc7903User(t, db, "u-role1b", strPtr7903("dept-t1-child"), "role-r1")
	ntc7903User(t, db, "u-role2", nil, "role-r2")
	ntc7903User(t, db, "u-none", nil)

	build := func(title string, targetType models.TargetType) *models.Notice {
		t.Helper()
		req := ntc7903CreateReq(title, "c", models.PriorityNormal)
		req.TargetType = targetType
		return ntc7903Notice(t, svc, req, "c", "创建人")
	}
	load := func(id string) *models.Notice {
		t.Helper()
		n, err := svc.GetNoticeByID(ctx, id)
		require.NoError(t, err)
		return n
	}

	// (0) 全体 → 全部用户(实装上限 10000)
	nAll := build("目标-全体", models.TargetAll)
	users, err := svc.GetTargetUsers(ctx, load(nAll.ID))
	require.NoError(t, err)
	assert.ElementsMatch(t,
		[]string{"u-d1", "u-child", "u-role1", "u-role1b", "u-role2", "u-none"}, users)

	// (1) 部门 → target 行消费(GetTargetUsers 不做递归,buildTargets 已展开子部门)
	deptReq := ntc7903CreateReq("目标-部门", "c", models.PriorityNormal)
	deptReq.TargetType = models.TargetDept
	deptReq.TargetDepts = []string{"dept-t1"}
	withTargets := ntc7903Notice(t, svc, deptReq, "c", "创建人")
	users, err = svc.GetTargetUsers(ctx, load(withTargets.ID))
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"u-d1", "u-child", "u-role1", "u-role1b"}, users,
		"dept_t1 及其子部门的用户集")

	// 空 dept target 行 → 空集(非 nil)
	emptyDept := build("目标-部门空", models.TargetDept)
	users, err = svc.GetTargetUsers(ctx, load(emptyDept.ID))
	require.NoError(t, err)
	assert.NotNil(t, users)
	assert.Empty(t, users)

	// (2) 角色 → sys_user_role DISTINCT user_id
	roleReq := ntc7903CreateReq("目标-角色", "c", models.PriorityNormal)
	roleReq.TargetType = models.TargetRole
	roleReq.TargetRoles = []string{"role-r1", "role-r2"}
	roleNotice := ntc7903Notice(t, svc, roleReq, "c", "创建人")
	users, err = svc.GetTargetUsers(ctx, load(roleNotice.ID))
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"u-role1", "u-role1b", "u-role2"}, users,
		"角色目标经 sys_user_role DISTINCT 解析")

	// (3) 指定用户 → target 行直取(不查 sys_user)+ unique 去重保持首现序
	dupReq := ntc7903CreateReq("目标-用户重复", "c", models.PriorityNormal)
	dupReq.TargetType = models.TargetUser
	dupReq.TargetUsers = []string{"u-none", "u-none", "u-d1"}
	dupNotice := ntc7903Notice(t, svc, dupReq, "c", "创建人")
	users, err = svc.GetTargetUsers(ctx, load(dupNotice.ID))
	require.NoError(t, err)
	assert.Equal(t, []string{"u-none", "u-d1"}, users, "unique 去重且保持首现序")

	// 包级直调:getTargetIDsByType 过滤 + unique 去重(表驱动)
	targets := []models.NoticeTarget{
		{TargetType: "dept", TargetID: "d1"},
		{TargetType: "user", TargetID: "u1"},
		{TargetType: "user", TargetID: "u1"},
		{TargetType: "role", TargetID: "r1"},
		{TargetType: "user", TargetID: "u2"},
	}
	assert.Equal(t, []string{"d1"}, getTargetIDsByType(targets, "dept"))
	assert.Equal(t, []string{"u1", "u1", "u2"}, getTargetIDsByType(targets, "user"))
	assert.Empty(t, getTargetIDsByType(targets, "api"))
	assert.Equal(t, []string{"u1", "u2"}, unique([]string{"u1", "u1", "u2", "u1"}))
	assert.Empty(t, unique(nil))
}

// TestNtg7903_GetChildDeptIDs getChildDeptIDs 递归(含孙级)+ 叶子空集。
func TestNtg7903_GetChildDeptIDs(t *testing.T) {
	svc, db := newNtc7903(t)

	ntc7903Dept(t, db, "root", "")
	ntc7903Dept(t, db, "child-a", "root")
	ntc7903Dept(t, db, "grandchild", "child-a")
	ntc7903Dept(t, db, "child-b", "root")

	got := svc.getChildDeptIDs("root")
	assert.ElementsMatch(t, []string{"child-a", "grandchild", "child-b"}, got, "递归展开含孙级")

	got = svc.getChildDeptIDs("grandchild")
	assert.Empty(t, got, "叶子部门无子集")

	got = svc.getChildDeptIDs("ghost-dept")
	assert.Empty(t, got, "不存在部门 → 空集")
}

// TestNtg7903_NoticeStatistics 阅读统计:TotalTargets/ReadCount/UnreadCount/ReadRate 手算一致。
func TestNtg7903_NoticeStatistics(t *testing.T) {
	svc, db := newNtc7903(t)
	ctx := context.Background()

	ntc7903User(t, db, "stat-u1", nil)
	ntc7903User(t, db, "stat-u2", nil)
	ntc7903User(t, db, "stat-u3", nil)

	// TargetAll → 目标 = 全部 3 用户
	req := ntc7903CreateReq("统计通知", "c", models.PriorityNormal)
	notice := ntc7903Notice(t, svc, req, "c", "创建人")
	ntc7903Publish(t, svc, notice.ID)

	require.NoError(t, svc.MarkNoticeRead(ctx, notice.ID, "stat-u1", "127.0.0.1"))

	stats, err := svc.GetNoticeStatistics(ctx, notice.ID)
	require.NoError(t, err)
	assert.Equal(t, 3, stats.TotalTargets)
	assert.Equal(t, 1, stats.ReadCount)
	assert.Equal(t, 2, stats.UnreadCount)
	assert.InDelta(t, 33.33, stats.ReadRate, 0.01, "阅读率 = 1/3*100")

	// 零目标分支:dept 型且无 target 行 → 不执行已读计数,ReadRate=0
	emptyReq := ntc7903CreateReq("零目标通知", "c", models.PriorityNormal)
	emptyReq.TargetType = models.TargetDept
	emptyNotice := ntc7903Notice(t, svc, emptyReq, "c", "创建人")
	stats, err = svc.GetNoticeStatistics(ctx, emptyNotice.ID)
	require.NoError(t, err)
	assert.Equal(t, 0, stats.TotalTargets)
	assert.Equal(t, 0, stats.ReadCount)
	assert.Equal(t, 0, stats.UnreadCount)
	assert.Zero(t, stats.ReadRate, "零目标 → ReadRate 0(防除零分支)")

	// 通知不存在 → 错误
	_, err = svc.GetNoticeStatistics(ctx, "ghost-notice")
	require.ErrorContains(t, err, "查询通知失败")
}

// TestNqv7903_BuildVisibleQuery_UserMissing 不存在用户 → 明确报错。
func TestNqv7903_BuildVisibleQuery_UserMissing(t *testing.T) {
	svc, _ := newNtc7903(t)
	_, err := svc.buildUserVisibleQuery(context.Background(), "ghost-user")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "获取用户信息失败")
}

// TestNqv7903_BuildVisibleQuery_FourOrMatrix buildUserVisibleQuery 的 4-OR 权限过滤
// 逐支断言(全体体 / 部门命中与未命中 / 角色命中与未命中 / 指定用户命中与未命中 /
// 发布态+启停语义 / ignore 排除),断言走 ctx.Query 实际执行 Count/Find。
func TestNqv7903_BuildVisibleQuery_FourOrMatrix(t *testing.T) {
	svc, db := newNtc7903(t)
	ctx := context.Background()

	deptD1, deptD2 := "dept-q1", "dept-q2"
	ntc7903User(t, db, "q-user", &deptD1, "role-qr1", "role-qr2")
	ntc7903User(t, db, "q-user-other", &deptD2)

	seed := func(title string, targetType models.TargetType, ids []string, publish bool, status int) string {
		t.Helper()
		req := ntc7903CreateReq(title, "c", models.PriorityNormal)
		req.TargetType = targetType
		switch targetType {
		case models.TargetDept:
			req.TargetDepts = ids
		case models.TargetRole:
			req.TargetRoles = ids
		case models.TargetUser:
			req.TargetUsers = ids
		}
		n := ntc7903Notice(t, svc, req, "c", "创建人")
		if status != int(models.NoticeStatusNormal) {
			require.NoError(t, svc.UpdateNotice(ctx, n.ID, &UpdateNoticeRequest{Status: &status}))
		}
		if publish {
			ntc7903Publish(t, svc, n.ID)
		}
		return n.ID
	}
	nAll := seed("矩阵-全体", models.TargetAll, nil, true, int(models.NoticeStatusNormal))
	nDeptHit := seed("矩阵-部门命中", models.TargetDept, []string{deptD1}, true, int(models.NoticeStatusNormal))
	nDeptMiss := seed("矩阵-部门未命中", models.TargetDept, []string{deptD2}, true, int(models.NoticeStatusNormal))
	nRoleHit := seed("矩阵-角色命中", models.TargetRole, []string{"role-qr1"}, true, int(models.NoticeStatusNormal))
	nRoleMiss := seed("矩阵-角色未命中", models.TargetRole, []string{"role-qx"}, true, int(models.NoticeStatusNormal))
	nUserHit := seed("矩阵-用户命中", models.TargetUser, []string{"q-user"}, true, int(models.NoticeStatusNormal))
	nUserMiss := seed("矩阵-用户未命中", models.TargetUser, []string{"q-user-other"}, true, int(models.NoticeStatusNormal))
	nDraft := seed("矩阵-草稿", models.TargetAll, nil, false, int(models.NoticeStatusNormal))
	nClosed := seed("矩阵-停用", models.TargetAll, nil, true, int(models.NoticeStatusClosed))
	nIgnored := seed("矩阵-被忽略", models.TargetAll, nil, true, int(models.NoticeStatusNormal))

	qctx, err := svc.buildUserVisibleQuery(ctx, "q-user")
	require.NoError(t, err)
	assert.Equal(t, "q-user", qctx.User.ID, "上下文携带用户")
	assert.ElementsMatch(t, []string{"role-qr1", "role-qr2"}, qctx.RoleIDs, "上下文携带角色集")

	// 可见集 = 全体×2(nAll + nIgnored) + 部门命中 + 角色命中 + 用户命中;
	// 其余 5 条(部门/角色/用户未命中、草稿、停用)不可见
	visible := ntc7903VisibleIDs(t, qctx)
	assert.ElementsMatch(t, []string{nAll, nIgnored, nDeptHit, nRoleHit, nUserHit}, visible)

	var count int64
	require.NoError(t, qctx.Query.Count(&count).Error)
	assert.Equal(t, int64(5), count, "Count 与 Find 同口径")

	// (e) 反向证明:publish_status 非发布态 或 status 停用 → 不可见(E 簇语义)
	for _, id := range []string{nDraft, nClosed} {
		q2, err := svc.buildUserVisibleQuery(ctx, "q-user")
		require.NoError(t, err)
		var hit int64
		require.NoError(t, q2.Query.Where("id = ?", id).Count(&hit).Error)
		assert.Zero(t, hit, "id=%s 不可见(草稿/停用)", id)
	}

	// (f) ignore 排除:忽略 nIgnored 后其退出可见集
	require.NoError(t, svc.IgnoreNotice(ctx, nIgnored, "q-user"))
	q3, err := svc.buildUserVisibleQuery(ctx, "q-user")
	require.NoError(t, err)
	visible = ntc7903VisibleIDs(t, q3)
	assert.NotContains(t, visible, nIgnored, "被忽略行退出可见集")
	assert.Len(t, visible, 4, "其余可见集不变")

	// 未命中支逐条复核:部门未命中/角色未命中/用户未命中 3 条始终不可见
	for _, id := range []string{nDeptMiss, nRoleMiss, nUserMiss} {
		assert.NotContains(t, visible, id)
	}
}
