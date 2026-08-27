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
