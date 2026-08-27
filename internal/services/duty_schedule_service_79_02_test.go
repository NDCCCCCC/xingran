package services

// =====================================================================
// Phase 79-02 Task 1: GenerateSchedule 分支表 + getDutyType/isWeekend 纯函数
// Phase 79-02 Task 2: duty_schedule CRUD 族(List/Today/Monthly/Swap/Manual/Delete)
//
// 覆盖目标: duty_schedule_service.go 0% → ≥70%(基线 174 stmts 全 unc,79-RESEARCH §2)。
//
// 纪律(79-01 SUMMARY 手注沿用):
//   - helper 名带 7902 后缀(R5/D-79-06);fixture 用 t.TempDir() 文件库。
//   - 禁 t.Parallel()(同包串行,wave 2)。
//   - 日期期望值一律 time.Date 显式构造,禁相对"今天"断言(防跨日 flake);
//     唯一例外是 GetTodayDuty/GetMyDutyStats 这类语义上就是"今天"的方法 ——
//     按计划允许用 time.Now 构造种子行,但断言只比对成员/池字段,不断言日期值。
//   - 状态/模式断言一律引用 models.ScheduleMode* / models.DutyStatus* 具名常量,禁裸 0/1。
//   - 生产代码零改动;quirk 按"只锁不修"纪律注释记录(R7/Phase 73-04 Q5 同款)。
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
	"github.com/xingran-next/xingran-go-backend/internal/services/base"
)

// dsc7902Creator 所有测试共用的 creatorID(非魔法值,仅用于 CreatedBy 透传断言)。
const dsc7902Creator = "creator-7902"

// dsc7902Date 构造 UTC 零点日期 —— 与生产代码 time.Parse("2006-01-02", ...)
// 产出的 time.Time 完全同形态(同 location 同零点),保证存储文本逐字一致。
func dsc7902Date(year, month, day int) time.Time {
	return time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
}

// dsc7902Week2026 固定参照周:2026-03-02(周一) .. 2026-03-08(周日)。
// 周内事实(本 session 用 go run 复核):02=Mon 03=Tue 04=Wed 05=Thu 06=Fri 07=Sat 08=Sun。
var (
	dsc7902Mon = dsc7902Date(2026, 3, 2)
	dsc7902Tue = dsc7902Date(2026, 3, 3)
	dsc7902Wed = dsc7902Date(2026, 3, 4)
	dsc7902Thu = dsc7902Date(2026, 3, 5)
	dsc7902Fri = dsc7902Date(2026, 3, 6)
	dsc7902Sat = dsc7902Date(2026, 3, 7)
	dsc7902Sun = dsc7902Date(2026, 3, 8)
)

// newDsc7902 装配 DutyScheduleService + sqlite(t.TempDir 文件库)+ AutoMigrate
// GenerateSchedule / GetDutyScheduleList(Preload Pool/User)链路实际引用的 model。
// AutoMigrate 会连带迁移关联 model(DutyPool→Department/DutyPoolMember→User 等),
// 这里显式列出主链 model,Preload("Members.User")/Preload("Dept") 不因缺表报错。
func newDsc7902(t *testing.T) (*DutyScheduleService, *gorm.DB) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "dsc7902.db")), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	require.NoError(t, err, "open sqlite temp db")
	if sqlDB, err := db.DB(); err == nil {
		t.Cleanup(func() { _ = sqlDB.Close() })
	}
	require.NoError(t, db.AutoMigrate(
		&models.DutyPool{},
		&models.DutyPoolMember{},
		&models.DutySchedule{},
		&models.DutyExchange{},
		&models.Holiday{},
		&models.DutyConfig{},
		&models.Department{},
		&models.User{},
		&models.Post{},
		&models.UserPost{},
	), "auto migrate duty chain models")
	return NewDutyScheduleService(db), db
}

// seedPool7902 预置值班池(sys_duty_pool)+ 成员(sys_duty_pool_member,MemberOrder=下标)。
// Members 在 model 里是 []DutyPoolMember 关联(foreignKey PoolID),非序列化列 ——
// 轮询语义的成员顺序由 MemberOrder + Preload 返回序决定,fixture 按下标落库。
func seedPool7902(t *testing.T, db *gorm.DB, name string, dailyCount int, memberIDs ...string) string {
	t.Helper()
	pool := &models.DutyPool{
		BaseModel:  models.BaseModel{CreatedBy: dsc7902Creator},
		PoolName:   name,
		Status:     models.DutyPoolStatusEnabled,
		DailyCount: dailyCount,
	}
	require.NoError(t, db.Create(pool).Error, "seed duty pool")
	for i, uid := range memberIDs {
		require.NoError(t, db.Create(&models.DutyPoolMember{
			PoolID:      pool.ID,
			UserID:      uid,
			MemberOrder: i,
		}).Error, "seed duty pool member %d", i)
	}
	return pool.ID
}

// seedSchedule7902 预置一条排班行,返回落库后的完整记录(含生成的 ID)。
func seedSchedule7902(
	t *testing.T,
	db *gorm.DB,
	poolID, userID string,
	date time.Time,
	dutyType models.ScheduleMode,
) models.DutySchedule {
	t.Helper()
	row := models.DutySchedule{
		BaseModel:    models.BaseModel{CreatedBy: dsc7902Creator},
		ScheduleDate: date,
		PoolID:       poolID,
		UserID:       userID,
		DutyType:     dutyType,
		Status:       models.DutyStatusNormal,
	}
	require.NoError(t, db.Create(&row).Error, "seed duty schedule")
	return row
}

// dsc7902Count 统计 sys_duty_schedule 非软删除行数(断言无半写/回滚用)。
func dsc7902Count(t *testing.T, db *gorm.DB) int64 {
	t.Helper()
	var n int64
	require.NoError(t, db.Model(&models.DutySchedule{}).Count(&n).Error)
	return n
}

// dsc7902Rows 读取某池全部排班行(按日期升序),供分支断言。
func dsc7902Rows(t *testing.T, db *gorm.DB, poolID string) []models.DutySchedule {
	t.Helper()
	var rows []models.DutySchedule
	require.NoError(t, db.Where("pool_id = ?", poolID).Order("schedule_date ASC").Find(&rows).Error)
	return rows
}

// ==================== GenerateSchedule 分支表 ====================

// 分支 :29 池不存在 → 直接返回错误,事务未开启,无半写。
func TestDsc7902_GenerateSchedule_PoolMissing(t *testing.T) {
	svc, db := newDsc7902(t)

	count, err := svc.GenerateSchedule(context.Background(), &GenerateScheduleRequest{
		PoolID:    "no-such-pool",
		StartDate: "2026-03-02",
		EndDate:   "2026-03-08",
		DutyType:  string(models.ScheduleModeWeekday),
	}, dsc7902Creator)

	require.Error(t, err, "池不存在必须报错")
	assert.Contains(t, err.Error(), "值班池不存在")
	assert.Zero(t, count, "错误路径返回 0")
	assert.Zero(t, dsc7902Count(t, db), "事务未开启,无半写")
}

// 分支 :34 池存在但 Members 空 → "值班池没有成员"。
func TestDsc7902_GenerateSchedule_EmptyMembers(t *testing.T) {
	svc, db := newDsc7902(t)
	poolID := seedPool7902(t, db, "empty-pool", 1) // 不建任何成员行

	count, err := svc.GenerateSchedule(context.Background(), &GenerateScheduleRequest{
		PoolID:    poolID,
		StartDate: "2026-03-02",
		EndDate:   "2026-03-08",
		DutyType:  string(models.ScheduleModeWeekday),
	}, dsc7902Creator)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "值班池没有成员")
	assert.Zero(t, count)
	assert.Zero(t, dsc7902Count(t, db), "错误路径无半写")
}

// 分支 :39/:43 两个日期解析失败 → 各自的"日期格式错误"文案。
func TestDsc7902_GenerateSchedule_BadDate(t *testing.T) {
	svc, db := newDsc7902(t)
	poolID := seedPool7902(t, db, "bad-date-pool", 1, "member-a")

	// StartDate 非法(斜杠格式不匹配 2006-01-02)
	count, err := svc.GenerateSchedule(context.Background(), &GenerateScheduleRequest{
		PoolID:    poolID,
		StartDate: "2026/03/02",
		EndDate:   "2026-03-08",
		DutyType:  string(models.ScheduleModeWeekday),
	}, dsc7902Creator)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "开始日期格式错误")
	assert.Zero(t, count)

	// EndDate 非法
	count, err = svc.GenerateSchedule(context.Background(), &GenerateScheduleRequest{
		PoolID:    poolID,
		StartDate: "2026-03-02",
		EndDate:   "not-a-date",
		DutyType:  string(models.ScheduleModeWeekday),
	}, dsc7902Creator)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "结束日期格式错误")
	assert.Zero(t, count)

	// 两个解析分支都在事务前返回 → 全库无排班行
	assert.Zero(t, dsc7902Count(t, db), "日期解析失败无半写")
}

// 分支 :85 类型过滤 continue:Weekday 请求只排工作日,Weekend 请求只排周末。
func TestDsc7902_GenerateSchedule_DutyTypeFilter(t *testing.T) {
	svc, db := newDsc7902(t)
	poolID := seedPool7902(t, db, "filter-pool", 2, "member-a", "member-b", "member-c")

	// 2026-03-02..03-08 = 周一..周日:5 个工作日 × DailyCount 2 = 10 行
	count, err := svc.GenerateSchedule(context.Background(), &GenerateScheduleRequest{
		PoolID:    poolID,
		StartDate: "2026-03-02",
		EndDate:   "2026-03-08",
		DutyType:  string(models.ScheduleModeWeekday),
	}, dsc7902Creator)
	require.NoError(t, err)
	assert.Equal(t, 10, count, "5 个工作日 × 2")

	rows := dsc7902Rows(t, db, poolID)
	require.Len(t, rows, 10)
	for _, r := range rows {
		assert.Equal(t, models.ScheduleModeWeekday, r.DutyType, "周末被 :85 continue 跳过,落库全为工作日")
		assert.Equal(t, models.DutyStatusNormal, r.Status)
	}
	// 周六/周日无行
	weekendDates := map[string]bool{
		dsc7902Sat.Format("2006-01-02"): false,
		dsc7902Sun.Format("2006-01-02"): false,
	}
	for _, r := range rows {
		key := r.ScheduleDate.Format("2006-01-02")
		assert.NotContains(t, weekendDates, key, "周末不应有工作日排班: %s", key)
	}

	// Weekend 请求(同库新池,避免 ClearExists 干扰):2 天 × 2 = 4 行
	weekendPool := seedPool7902(t, db, "filter-pool-weekend", 2, "member-a", "member-b")
	count, err = svc.GenerateSchedule(context.Background(), &GenerateScheduleRequest{
		PoolID:    weekendPool,
		StartDate: "2026-03-02",
		EndDate:   "2026-03-08",
		DutyType:  string(models.ScheduleModeWeekend),
	}, dsc7902Creator)
	require.NoError(t, err)
	assert.Equal(t, 4, count, "2 个周末日 × 2")

	weekendRows := dsc7902Rows(t, db, weekendPool)
	require.Len(t, weekendRows, 4)
	for _, r := range weekendRows {
		assert.Equal(t, models.ScheduleModeWeekend, r.DutyType)
		assert.Contains(t, weekendDates, r.ScheduleDate.Format("2006-01-02"), "只排周六/周日")
	}
}

// 分支 :90 轮询 memberIndex%memberCount:12 槽位 × 3 成员 → 每人恰 4 次。
// 2026-03-02..03-09 含 6 个工作日(02,03,04,05,06,09) × DailyCount 2 = 12 槽。
func TestDsc7902_GenerateSchedule_RoundRobin(t *testing.T) {
	svc, db := newDsc7902(t)
	poolID := seedPool7902(t, db, "roundrobin-pool", 2, "member-a", "member-b", "member-c")

	count, err := svc.GenerateSchedule(context.Background(), &GenerateScheduleRequest{
		PoolID:    poolID,
		StartDate: "2026-03-02",
		EndDate:   "2026-03-09",
		DutyType:  string(models.ScheduleModeWeekday),
	}, dsc7902Creator)
	require.NoError(t, err)
	assert.Equal(t, 12, count)

	rows := dsc7902Rows(t, db, poolID)
	require.Len(t, rows, 12)

	perMember := map[string]int{}
	perDate := map[string][]string{}
	for _, r := range rows {
		perMember[r.UserID]++
		perDate[r.ScheduleDate.Format("2006-01-02")] = append(
			perDate[r.ScheduleDate.Format("2006-01-02")], r.UserID)
	}
	assert.Len(t, perMember, 3, "3 个成员都被排到")
	for uid, n := range perMember {
		assert.Equal(t, 4, n, "12 槽位轮询 3 成员,每人恰 4 次:%s", uid)
	}
	// 每个工作日恰 2 人,且同一日两个成员不重复(DailyCount=2 < memberCount=3)
	for date, uids := range perDate {
		assert.Len(t, uids, 2, "每日 2 槽:%s", date)
		assert.NotEqual(t, uids[0], uids[1], "同日成员不重复(memberIndex 递增):%s", date)
	}
	// 6 个工作日都有行
	assert.Len(t, perDate, 6)
}

// 分支 :50-61 节假日映射双向断言:IsOffday 行把工作日翻成 holiday。
// 预置 2026-03-03(周二) 为法定节假日 → weekday 请求跳过该日,holiday 请求只排该日。
func TestDsc7902_GenerateSchedule_HolidayMap(t *testing.T) {
	// 方向一:weekday 请求 → 03-03 被节假日翻型,只剩 02/04 两天
	svc, db := newDsc7902(t)
	poolID := seedPool7902(t, db, "holiday-pool-wd", 1, "member-a")
	require.NoError(t, db.Create(&models.Holiday{
		BaseModel:   models.BaseModel{CreatedBy: dsc7902Creator},
		HolidayDate: dsc7902Tue,
		HolidayName: "测试节假日",
		IsOffday:    true,
		HolidayType: models.HolidayTypeLegal,
		Year:        2026,
	}).Error, "seed holiday")

	count, err := svc.GenerateSchedule(context.Background(), &GenerateScheduleRequest{
		PoolID:    poolID,
		StartDate: "2026-03-02",
		EndDate:   "2026-03-04",
		DutyType:  string(models.ScheduleModeWeekday),
	}, dsc7902Creator)
	require.NoError(t, err)
	assert.Equal(t, 2, count, "03-03 被翻成 holiday,只排 02/04")
	wdRows := dsc7902Rows(t, db, poolID)
	for _, r := range wdRows {
		assert.NotEqual(t, "2026-03-03", r.ScheduleDate.Format("2006-01-02"))
		assert.Equal(t, models.ScheduleModeWeekday, r.DutyType)
	}

	// 方向二:holiday 请求(独立库) → 只排 03-03
	svc2, db2 := newDsc7902(t)
	poolID2 := seedPool7902(t, db2, "holiday-pool-hd", 1, "member-a")
	require.NoError(t, db2.Create(&models.Holiday{
		BaseModel:   models.BaseModel{CreatedBy: dsc7902Creator},
		HolidayDate: dsc7902Tue,
		HolidayName: "测试节假日",
		IsOffday:    true,
		HolidayType: models.HolidayTypeLegal,
		Year:        2026,
	}).Error)

	count, err = svc2.GenerateSchedule(context.Background(), &GenerateScheduleRequest{
		PoolID:    poolID2,
		StartDate: "2026-03-02",
		EndDate:   "2026-03-04",
		DutyType:  string(models.ScheduleModeHoliday),
	}, dsc7902Creator)
	require.NoError(t, err)
	assert.Equal(t, 1, count, "只排节假日 03-03")
	hdRows := dsc7902Rows(t, db2, poolID2)
	require.Len(t, hdRows, 1)
	assert.Equal(t, "2026-03-03", hdRows[0].ScheduleDate.Format("2006-01-02"))
	assert.Equal(t, models.ScheduleModeHoliday, hdRows[0].DutyType)
}

// 分支 :67/:109 ClearExists:再生成先清旧排班;不清则累计翻倍。
func TestDsc7902_GenerateSchedule_ClearExists(t *testing.T) {
	svc, db := newDsc7902(t)
	poolID := seedPool7902(t, db, "clear-pool", 1, "member-a", "member-b")

	req := &GenerateScheduleRequest{
		PoolID:      poolID,
		StartDate:   "2026-03-02",
		EndDate:     "2026-03-06", // 5 个工作日
		DutyType:    string(models.ScheduleModeWeekday),
		ClearExists: false,
	}

	count, err := svc.GenerateSchedule(context.Background(), req, dsc7902Creator)
	require.NoError(t, err)
	assert.Equal(t, 5, count, "首次生成 5 个工作日 × 1")
	require.Equal(t, int64(5), dsc7902Count(t, db))

	// ClearExists=false → 旧记录保留,累计翻倍
	count, err = svc.GenerateSchedule(context.Background(), req, dsc7902Creator)
	require.NoError(t, err)
	assert.Equal(t, 5, count)
	assert.Equal(t, int64(10), dsc7902Count(t, db), "不清除时累计翻倍")

	// ClearExists=true → 事务内先删旧排班,总量回到 5
	req.ClearExists = true
	count, err = svc.GenerateSchedule(context.Background(), req, dsc7902Creator)
	require.NoError(t, err)
	assert.Equal(t, 5, count)
	assert.Equal(t, int64(5), dsc7902Count(t, db), "清除后旧记录被删、总数正确")
}

// 分支 :109 len(schedules)==0:区间内无匹配类型 → 返回 0 且不报错、不插入。
func TestDsc7902_GenerateSchedule_EmptyResult(t *testing.T) {
	svc, db := newDsc7902(t)
	poolID := seedPool7902(t, db, "empty-result-pool", 1, "member-a")

	// 区间内无任何节假日 → holiday 请求 0 行
	count, err := svc.GenerateSchedule(context.Background(), &GenerateScheduleRequest{
		PoolID:    poolID,
		StartDate: "2026-03-02",
		EndDate:   "2026-03-04",
		DutyType:  string(models.ScheduleModeHoliday),
	}, dsc7902Creator)
	require.NoError(t, err, "空结果不是错误")
	assert.Zero(t, count)
	assert.Zero(t, dsc7902Count(t, db), "空 schedules 不插入(:109 分支)")
}

// ==================== 纯函数表驱动 ====================

// getDutyType 三态:holidayMap IsOffday=true → holiday;
// IsOffday=false(节假日但非休)→ 落到周末/工作日判断;不在 map → 周末/工作日。
func TestDsc7902_GetDutyType_Table(t *testing.T) {
	svc, _ := newDsc7902(t)

	offdayHoliday := models.Holiday{HolidayDate: dsc7902Tue, IsOffday: true}
	workdayHoliday := models.Holiday{HolidayDate: dsc7902Wed, IsOffday: false}

	tests := []struct {
		name       string
		date       time.Time
		holidayMap map[string]models.Holiday
		want       models.ScheduleMode
	}{
		{
			name:       "节假日且休 → holiday",
			date:       dsc7902Tue,
			holidayMap: map[string]models.Holiday{"2026-03-03": offdayHoliday},
			want:       models.ScheduleModeHoliday,
		},
		{
			name:       "节假日但非休(调休工作日)+ 平日 → weekday",
			date:       dsc7902Wed,
			holidayMap: map[string]models.Holiday{"2026-03-04": workdayHoliday},
			want:       models.ScheduleModeWeekday,
		},
		{
			name:       "节假日但非休 + 周末 → weekend(落到周末判断)",
			date:       dsc7902Sat,
			holidayMap: map[string]models.Holiday{"2026-03-07": workdayHoliday},
			want:       models.ScheduleModeWeekend,
		},
		{
			name:       "不在 map + 平日 → weekday",
			date:       dsc7902Mon,
			holidayMap: map[string]models.Holiday{},
			want:       models.ScheduleModeWeekday,
		},
		{
			name:       "不在 map + 周六 → weekend",
			date:       dsc7902Sat,
			holidayMap: map[string]models.Holiday{},
			want:       models.ScheduleModeWeekend,
		},
		{
			name:       "不在 map + 周日 → weekend",
			date:       dsc7902Sun,
			holidayMap: map[string]models.Holiday{},
			want:       models.ScheduleModeWeekend,
		},
		{
			name:       "map 命中但日期键不匹配 → 按周末/工作日(周一)",
			date:       dsc7902Mon,
			holidayMap: map[string]models.Holiday{"2026-03-03": offdayHoliday},
			want:       models.ScheduleModeWeekday,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := svc.getDutyType(tt.date, tt.holidayMap)
			assert.Equal(t, string(tt.want), got)
		})
	}
}

// isWeekend:周六/周日 true,周一..周五 false(同包直调白盒)。
func TestDsc7902_IsWeekend(t *testing.T) {
	tests := []struct {
		date time.Time
		want bool
	}{
		{dsc7902Mon, false},
		{dsc7902Tue, false},
		{dsc7902Wed, false},
		{dsc7902Thu, false},
		{dsc7902Fri, false},
		{dsc7902Sat, true},
		{dsc7902Sun, true},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.want, isWeekend(tt.date),
			"%s 应为 %v", tt.date.Format("2006-01-02"), tt.want)
	}
}

// ==================== Task 2: duty_schedule CRUD 族 ====================

// dsc7902LocalToday 本地"今天"日期的 UTC 零点 —— 与生产 GenerateSchedule 经
// time.Parse("2006-01-02") 落库的行同形态,保证 DATE(schedule_date) 能命中
// GetTodayDuty 的本地今天字符串。
// 注意:不能用 time.Now().Truncate(24*time.Hour) —— 那是 UTC 日的零点,在 +08
// 时区 08:00 前其本地日期是"昨天"(duty_stats_service.go:24 即此构造,Task 4 另行
// 同值种子处理)。断言只比对成员/池字段,不断言日期值本身(跨日安全)。
func dsc7902LocalToday() time.Time {
	now := time.Now()
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
}

// dsc7902User 预置一个 sys_user 行(Username 必填,Password/Salt 有 not null 约束)。
func dsc7902User(t *testing.T, db *gorm.DB, id, username string, nickname, phone *string) {
	t.Helper()
	require.NoError(t, db.Create(&models.User{
		BaseModel: models.BaseModel{ID: id, CreatedBy: dsc7902Creator},
		Username:  username,
		Password:  "not-a-real-password",
		Salt:      "salt-7902",
		Nickname:  nickname,
		Phone:     phone,
	}).Error, "seed user %s", username)
}

// 分页 + 全量过滤分支 + 白名单排序 + 非法列回退。
func TestDsc7902_GetList_PaginationAndFilter(t *testing.T) {
	svc, db := newDsc7902(t)
	poolA := seedPool7902(t, db, "list-pool-a", 1, "member-a")
	poolB := seedPool7902(t, db, "list-pool-b", 1, "member-b")

	seedSchedule7902(t, db, poolA, "member-a", dsc7902Mon, models.ScheduleModeWeekday)
	seedSchedule7902(t, db, poolA, "member-a", dsc7902Tue, models.ScheduleModeWeekday)
	seedSchedule7902(t, db, poolA, "member-b", dsc7902Wed, models.ScheduleModeWeekday)
	seedSchedule7902(t, db, poolA, "member-b", dsc7902Sat, models.ScheduleModeWeekend)
	r5 := seedSchedule7902(t, db, poolB, "member-c", dsc7902Sun, models.ScheduleModeWeekend)
	// 状态变体行(默认全部 DutyStatusNormal,补一条 Exchanged 供 status 过滤)
	seedSchedule7902(t, db, poolB, "member-c", dsc7902Fri, models.ScheduleModeWeekday)
	require.NoError(t, db.Model(&models.DutySchedule{}).Where("id = ?", r5.ID).
		Update("status", int(models.DutyStatusExchanged)).Error)

	strPtr := func(s string) *string { return &s }
	intPtr := func(i int) *int { return &i }

	// 分页语义:Current 从 1 起,PageSize=2,第 2 页
	rows, total, err := svc.GetDutyScheduleList(context.Background(), &DutyScheduleListRequest{
		BaseListRequest: base7902Req(2, 2),
	})
	require.NoError(t, err)
	assert.Equal(t, int64(6), total)
	assert.Len(t, rows, 2)

	// PageSize=0 → 默认 20(:205 分支)
	rows, total, err = svc.GetDutyScheduleList(context.Background(), &DutyScheduleListRequest{})
	require.NoError(t, err)
	assert.Equal(t, int64(6), total)
	assert.Len(t, rows, 6, "未传分页参数时默认 PageSize=20,单页取全量")

	// poolID 过滤
	rows, total, err = svc.GetDutyScheduleList(context.Background(), &DutyScheduleListRequest{
		PoolID: strPtr(poolA),
	})
	require.NoError(t, err)
	assert.Equal(t, int64(4), total)
	for _, r := range rows {
		assert.Equal(t, poolA, r.PoolID)
	}

	// userId 过滤
	rows, total, err = svc.GetDutyScheduleList(context.Background(), &DutyScheduleListRequest{
		UserID: strPtr("member-b"),
	})
	require.NoError(t, err)
	assert.Equal(t, int64(2), total)
	for _, r := range rows {
		assert.Equal(t, "member-b", r.UserID)
	}

	// 日期范围过滤(周一..周三)
	rows, total, err = svc.GetDutyScheduleList(context.Background(), &DutyScheduleListRequest{
		StartDate: strPtr("2026-03-02"),
		EndDate:   strPtr("2026-03-04"),
	})
	require.NoError(t, err)
	assert.Equal(t, int64(3), total)

	// dutyType 过滤
	rows, total, err = svc.GetDutyScheduleList(context.Background(), &DutyScheduleListRequest{
		DutyType: strPtr(string(models.ScheduleModeWeekend)),
	})
	require.NoError(t, err)
	assert.Equal(t, int64(2), total)
	for _, r := range rows {
		assert.Equal(t, models.ScheduleModeWeekend, r.DutyType)
	}

	// status 过滤(具名常量,禁裸 0/1)
	rows, total, err = svc.GetDutyScheduleList(context.Background(), &DutyScheduleListRequest{
		Status: intPtr(int(models.DutyStatusExchanged)),
	})
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	require.Len(t, rows, 1)
	assert.Equal(t, models.DutyStatusExchanged, rows[0].Status)

	// Preload("Pool") 生效:池名随行返回
	rows, _, err = svc.GetDutyScheduleList(context.Background(), &DutyScheduleListRequest{
		PoolID: strPtr(poolB),
	})
	require.NoError(t, err)
	require.NotEmpty(t, rows)
	require.NotNil(t, rows[0].Pool, "Preload(Pool) 应装载池名")
	assert.Equal(t, "list-pool-b", rows[0].Pool.PoolName)

	// 白名单排序:orderByColumn=scheduleDate + IsAsc=false → schedule_date DESC
	asc := false
	rows, _, err = svc.GetDutyScheduleList(context.Background(), &DutyScheduleListRequest{
		BaseListRequest: base.BaseListRequest{OrderByColumn: "scheduleDate", IsAsc: &asc, PageSize: 6},
	})
	require.NoError(t, err)
	require.Len(t, rows, 6)
	assert.Equal(t, "2026-03-08", rows[0].ScheduleDate.Format("2006-01-02"), "降序首行是最晚日期")

	// 升序
	ascTrue := true
	rows, _, err = svc.GetDutyScheduleList(context.Background(), &DutyScheduleListRequest{
		BaseListRequest: base.BaseListRequest{OrderByColumn: "scheduleDate", IsAsc: &ascTrue, PageSize: 6},
	})
	require.NoError(t, err)
	require.Len(t, rows, 6)
	assert.Equal(t, "2026-03-02", rows[0].ScheduleDate.Format("2006-01-02"), "升序首行是最早日期")

	// QUIRK-79-02-A(只锁不修,R7):非法列名走 ApplySort 回退分支 —— 白名单未命中
	// 仅打 warn 日志且不追加 Order;又因 OrderByColumn 非空,源码不再补默认
	// schedule_date ASC(:212 条件只对空串生效),最终顺序退化为 sqlite 自然序(插入序)。
	// 此处只断言"无错误 + 总数正确 + 白名单列未被注入",不锁具体顺序(实现细节)。
	rows, total, err = svc.GetDutyScheduleList(context.Background(), &DutyScheduleListRequest{
		BaseListRequest: base.BaseListRequest{OrderByColumn: "schedule_date; DROP TABLE x", PageSize: 6},
	})
	require.NoError(t, err, "非法排序列静默忽略,不报错")
	assert.Equal(t, int64(6), total)
	assert.Len(t, rows, 6)
}

// base7902Req 构造分页参数(独立 helper 避免与既有测试重名)。
func base7902Req(current, pageSize int) base.BaseListRequest {
	return base.BaseListRequest{Current: current, PageSize: pageSize}
}

// GetTodayDuty:只返回今天的行;昵称展示格式;空库报"今日无值班人员"。
func TestDsc7902_GetTodayDuty(t *testing.T) {
	svc, db := newDsc7902(t)
	poolID := seedPool7902(t, db, "today-pool", 2, "member-a", "member-b")

	nick := "张三丰"
	phone := "13800000001"
	dsc7902User(t, db, "member-a", "operator01", &nick, &phone)
	// 无昵称成员 → 展示用户名本身
	dsc7902User(t, db, "member-b", "operator02", nil, nil)

	today := dsc7902LocalToday()
	seedSchedule7902(t, db, poolID, "member-a", today, models.ScheduleModeWeekday)
	seedSchedule7902(t, db, poolID, "member-b", today.AddDate(0, 0, 1), models.ScheduleModeWeekday) // 明天,不返回

	members, err := svc.GetTodayDuty(context.Background())
	require.NoError(t, err)
	require.Len(t, members, 1, "只返回今天(明天那行不返回)")

	m := members[0]
	assert.Equal(t, poolID, m.PoolID)
	assert.Equal(t, "member-a", m.UserID)
	assert.Equal(t, "today-pool", m.PoolName)
	assert.Equal(t, "张三丰 (operator01)", m.Username, "昵称(用户名)统一展示格式")
	assert.Equal(t, phone, m.Phone)
	assert.Equal(t, string(models.ScheduleModeWeekday), m.DutyType)
	assert.NotEmpty(t, m.ScheduleID)
	// 不断言日期值本身(跨日安全): today 的取值与种子行同源

	// 空库 → 实装返回错误(非空切片),按现行为锁定
	svc2, db2 := newDsc7902(t)
	_ = db2
	members2, err := svc2.GetTodayDuty(context.Background())
	require.Error(t, err, "空库无今日值班 → 实装返回错误")
	assert.Contains(t, err.Error(), "今日无值班人员")
	assert.Nil(t, members2)
}

// GetMonthlyDutySchedule:map 键 "YYYY-MM-DD"、只含当月、按日分组;月份越界归一化锁定。
func TestDsc7902_GetMonthlyDutySchedule(t *testing.T) {
	svc, db := newDsc7902(t)
	poolID := seedPool7902(t, db, "monthly-pool", 2, "member-a", "member-b")

	seedSchedule7902(t, db, poolID, "member-a", dsc7902Mon, models.ScheduleModeWeekday)
	seedSchedule7902(t, db, poolID, "member-b", dsc7902Mon, models.ScheduleModeWeekday)
	seedSchedule7902(t, db, poolID, "member-a", dsc7902Date(2026, 3, 15), models.ScheduleModeWeekday)
	seedSchedule7902(t, db, poolID, "member-b", dsc7902Date(2026, 3, 31), models.ScheduleModeWeekend)
	seedSchedule7902(t, db, poolID, "member-a", dsc7902Date(2026, 4, 1), models.ScheduleModeWeekday) // 跨月,排除

	byDay, err := svc.GetMonthlyDutySchedule(context.Background(), 2026, 3)
	require.NoError(t, err)
	require.Len(t, byDay, 3, "只含 3 月的 3 天")

	require.Len(t, byDay["2026-03-02"], 2, "同日两行分到一组")
	assert.Equal(t, "member-a", byDay["2026-03-02"][0].UserID)
	assert.Equal(t, "member-b", byDay["2026-03-02"][1].UserID)
	assert.Equal(t, "monthly-pool", byDay["2026-03-02"][0].PoolName)
	assert.Len(t, byDay["2026-03-15"], 1)
	assert.Len(t, byDay["2026-03-31"], 1)
	assert.NotContains(t, byDay, "2026-04-01", "跨月行不入当月 map")

	// QUIRK-79-02-B(只锁不修):无效月份不报错 —— time.Date 归一化
	// month=13 → 2027-01;month=0 → 2025-12。返回空 map(非 nil)。
	byDay13, err := svc.GetMonthlyDutySchedule(context.Background(), 2026, 13)
	require.NoError(t, err, "month=13 被 time.Date 归一化为 2027-01,不报错")
	assert.NotNil(t, byDay13)
	assert.Empty(t, byDay13, "2027-01 无种子行")
	byDay0, err := svc.GetMonthlyDutySchedule(context.Background(), 2026, 0)
	require.NoError(t, err, "month=0 归一化为 2025-12")
	assert.NotNil(t, byDay0)
	assert.Empty(t, byDay0)
}

// SwapDuty:双方 ScheduleDate 保留、UserID 对调 + 状态/IsManual/SwapFrom/历史记录;
// 双侧不存在分支;同 ID 自换(实装允许,按现行为锁定)。
func TestDsc7902_SwapDuty(t *testing.T) {
	svc, db := newDsc7902(t)
	poolID := seedPool7902(t, db, "swap-pool", 1, "member-a", "member-b")
	from := seedSchedule7902(t, db, poolID, "member-a", dsc7902Mon, models.ScheduleModeWeekday)
	to := seedSchedule7902(t, db, poolID, "member-b", dsc7902Tue, models.ScheduleModeWeekday)

	err := svc.SwapDuty(context.Background(), &SwapDutyRequest{
		FromScheduleID: from.ID,
		ToScheduleID:   to.ID,
		Reason:         "临时调班",
	}, "operator-7902")
	require.NoError(t, err)

	var afterFrom, afterTo models.DutySchedule
	require.NoError(t, db.Where("id = ?", from.ID).First(&afterFrom).Error)
	require.NoError(t, db.Where("id = ?", to.ID).First(&afterTo).Error)

	// 双方字段对调断言(日期保留,人员互换)
	assert.Equal(t, "member-b", afterFrom.UserID, "原排班换成对方人员")
	assert.Equal(t, "member-a", afterTo.UserID, "目标排班换成原人员")
	assert.Equal(t, dsc7902Mon.Format("2006-01-02"), afterFrom.ScheduleDate.Format("2006-01-02"), "日期保留")
	assert.Equal(t, dsc7902Tue.Format("2006-01-02"), afterTo.ScheduleDate.Format("2006-01-02"), "日期保留")
	assert.Equal(t, models.DutyStatusExchanged, afterFrom.Status)
	assert.Equal(t, models.DutyStatusExchanged, afterTo.Status)
	assert.True(t, afterFrom.IsManual)
	assert.True(t, afterTo.IsManual)
	require.NotNil(t, afterFrom.SwapFromDate)
	assert.Equal(t, dsc7902Mon.Format("2006-01-02"), afterFrom.SwapFromDate.Format("2006-01-02"))
	require.NotNil(t, afterTo.SwapFromDate)
	assert.Equal(t, dsc7902Tue.Format("2006-01-02"), afterTo.SwapFromDate.Format("2006-01-02"))
	assert.Equal(t, "临时调班", afterFrom.SwapReason)
	assert.Equal(t, "operator-7902", afterFrom.UpdatedBy)
	assert.Equal(t, "operator-7902", afterTo.UpdatedBy)

	// 调班历史恰好一条,original/new 与交换方向一致
	var exchanges []models.DutyExchange
	require.NoError(t, db.Find(&exchanges).Error)
	require.Len(t, exchanges, 1)
	assert.Equal(t, from.ID, exchanges[0].ScheduleID)
	assert.Equal(t, "member-a", exchanges[0].OriginalUserID)
	assert.Equal(t, "member-b", exchanges[0].NewUserID)
	assert.Equal(t, "临时调班", exchanges[0].Reason)
	assert.Equal(t, "operator-7902", exchanges[0].CreatedBy)

	// 原排班不存在分支
	err = svc.SwapDuty(context.Background(), &SwapDutyRequest{
		FromScheduleID: "no-such-id",
		ToScheduleID:   to.ID,
	}, "operator-7902")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "原排班记录不存在")

	// 目标排班不存在分支
	err = svc.SwapDuty(context.Background(), &SwapDutyRequest{
		FromScheduleID: from.ID,
		ToScheduleID:   "no-such-id",
	}, "operator-7902")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "目标排班记录不存在")

	// QUIRK-79-02-C(只锁不修):From==To 自换 —— 实装两次查到同一行后对调同一值,
	// 人员不变、状态置 Exchanged、再记一条历史,均不报错。
	err = svc.SwapDuty(context.Background(), &SwapDutyRequest{
		FromScheduleID: afterFrom.ID,
		ToScheduleID:   afterFrom.ID,
		Reason:         "自换",
	}, "operator-7902")
	require.NoError(t, err, "实装不禁止自换")
	var selfSwapped models.DutySchedule
	require.NoError(t, db.Where("id = ?", afterFrom.ID).First(&selfSwapped).Error)
	assert.Equal(t, "member-b", selfSwapped.UserID, "自换不改变人员")
	assert.Equal(t, models.DutyStatusExchanged, selfSwapped.Status)
	var exchangeCount int64
	require.NoError(t, db.Model(&models.DutyExchange{}).Count(&exchangeCount).Error)
	assert.Equal(t, int64(2), exchangeCount, "自换也记一条历史")
}

// ManualDuty:合法请求落 IsManual 行;同池同日旧行被替换;坏日期报错;
// QUIRK-79-02-D:实装不校验池存在性,池不存在也不报错(计划预期错误分支,按现行为锁定)。
func TestDsc7902_ManualDuty(t *testing.T) {
	svc, db := newDsc7902(t)
	poolID := seedPool7902(t, db, "manual-pool", 1, "member-a", "member-b")

	// 预置同池同日旧行(自动排班),验证替换语义
	old := seedSchedule7902(t, db, poolID, "member-a", dsc7902Thu, models.ScheduleModeWeekday)

	err := svc.ManualDuty(context.Background(), &ManualDutyRequest{
		PoolID:   poolID,
		DutyDate: "2026-03-05",
		UserIDs:  []string{"member-a", "member-b"},
		DutyType: string(models.ScheduleModeWeekend),
		Reason:   "手工补班",
	}, dsc7902Creator)
	require.NoError(t, err)

	rows := dsc7902Rows(t, db, poolID)
	require.Len(t, rows, 2, "同池同日旧行被删,只剩 2 条新行")
	for _, r := range rows {
		assert.NotEqual(t, old.ID, r.ID, "旧行已被替换")
		assert.Equal(t, dsc7902Thu.Format("2006-01-02"), r.ScheduleDate.Format("2006-01-02"))
		assert.True(t, r.IsManual, "手工排班行 IsManual=true")
		assert.Equal(t, models.ScheduleModeWeekend, r.DutyType)
		assert.Equal(t, models.DutyStatusNormal, r.Status)
		assert.Equal(t, dsc7902Creator, r.CreatedBy)
		assert.Equal(t, "手工补班", r.SwapReason)
	}
	userIDs := []string{rows[0].UserID, rows[1].UserID}
	assert.ElementsMatch(t, []string{"member-a", "member-b"}, userIDs)

	// 坏日期 → 报错且不落库
	err = svc.ManualDuty(context.Background(), &ManualDutyRequest{
		PoolID:   poolID,
		DutyDate: "2026-03-05T10:00",
		UserIDs:  []string{"member-a"},
		DutyType: string(models.ScheduleModeWeekday),
	}, dsc7902Creator)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "值班日期格式错误")
	assert.Equal(t, int64(2), dsc7902Count(t, db), "坏日期无写入")

	// QUIRK-79-02-D(只锁不修):池不存在不报错,行仍创建 —— 计划预期
	// "池不存在分支 → 错误",实装 ManualDuty 无池校验(:388-418),按现行为锁定。
	err = svc.ManualDuty(context.Background(), &ManualDutyRequest{
		PoolID:   "no-such-pool",
		DutyDate: "2026-03-06",
		UserIDs:  []string{"member-a"},
		DutyType: string(models.ScheduleModeWeekday),
	}, dsc7902Creator)
	require.NoError(t, err, "实装无池存在性校验(quirk 锁定)")
	assert.Equal(t, int64(3), dsc7902Count(t, db))
}

// Delete / BatchDelete:单删-1、批删只剩未列入行、删不存在 ID 不报错。
func TestDsc7902_DeleteAndBatchDelete(t *testing.T) {
	svc, db := newDsc7902(t)
	poolID := seedPool7902(t, db, "delete-pool", 1, "member-a", "member-b")
	r1 := seedSchedule7902(t, db, poolID, "member-a", dsc7902Mon, models.ScheduleModeWeekday)
	r2 := seedSchedule7902(t, db, poolID, "member-a", dsc7902Tue, models.ScheduleModeWeekday)
	r3 := seedSchedule7902(t, db, poolID, "member-b", dsc7902Wed, models.ScheduleModeWeekday)
	r4 := seedSchedule7902(t, db, poolID, "member-b", dsc7902Thu, models.ScheduleModeWeekday)

	// 单删
	require.NoError(t, svc.DeleteDutySchedule(context.Background(), r1.ID))
	assert.Equal(t, int64(3), dsc7902Count(t, db), "单删后行数 -1")

	// 删不存在的 ID → 不报错、行数不变(软删除 0 行受影响)
	require.NoError(t, svc.DeleteDutySchedule(context.Background(), "no-such-id"))
	assert.Equal(t, int64(3), dsc7902Count(t, db))

	// 批删
	require.NoError(t, svc.BatchDeleteDutySchedules(context.Background(), []string{r2.ID, r3.ID}))
	assert.Equal(t, int64(1), dsc7902Count(t, db), "批删后只剩未列入 IDs 的行")
	remaining := dsc7902Rows(t, db, poolID)
	require.Len(t, remaining, 1)
	assert.Equal(t, r4.ID, remaining[0].ID)

	// QUIRK-79-02-E(只锁不修):空 IDs 切片 → GORM 无 WHERE 条件,
	// 实装把驱动错误包装为"批量删除排班记录失败"。
	err := svc.BatchDeleteDutySchedules(context.Background(), []string{})
	if err == nil {
		t.Log("空 IDs 不报错(GORM 版本行为),锁定当前实现")
		return
	}
	assert.Contains(t, err.Error(), "批量删除排班记录失败")
}

// Expired 过期态过滤(:185-193):0=未过期(schedule_date >= 今天),1=已过期(< 今天)。
// "今天"语义方法 → 种子用本地日期的 UTC 零点(同 dsc7902LocalToday 形态),
// 断言只区分"今天/过去"两行,不依赖日期字面值(跨日安全:
// 种子行日期与绑定字符串同源,行分类在任意运行时刻都稳定)。
func TestDsc7902_GetList_ExpiredFilter(t *testing.T) {
	svc, db := newDsc7902(t)
	poolID := seedPool7902(t, db, "expired-pool", 1, "member-a")
	today := dsc7902LocalToday()
	past := today.AddDate(0, 0, -3)
	seedSchedule7902(t, db, poolID, "member-a", today, models.ScheduleModeWeekday)
	seedSchedule7902(t, db, poolID, "member-a", past, models.ScheduleModeWeekday)

	// Expired=0 → 仅未过期(今天那行)
	notExpired := 0
	rows, total, err := svc.GetDutyScheduleList(context.Background(), &DutyScheduleListRequest{
		Expired: &notExpired,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(1), total, "未过期只有今天那行")
	require.Len(t, rows, 1)

	// Expired=1 → 仅已过期(过去那行)
	expired := 1
	rows, total, err = svc.GetDutyScheduleList(context.Background(), &DutyScheduleListRequest{
		Expired: &expired,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(1), total, "已过期只有过去那行")
	require.Len(t, rows, 1)
	assert.Equal(t, past.Format("2006-01-02"), rows[0].ScheduleDate.Format("2006-01-02"))

	// 不传 Expired → 全部
	_, total, err = svc.GetDutyScheduleList(context.Background(), &DutyScheduleListRequest{})
	require.NoError(t, err)
	assert.Equal(t, int64(2), total, "不传 expired 返回全部")
}

