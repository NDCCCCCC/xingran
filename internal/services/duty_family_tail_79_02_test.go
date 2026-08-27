package services

// =====================================================================
// Phase 79-02 Task 4/5: duty_stats + duty_holiday + duty_config 三小文件 + duty_service 门面
//
// 覆盖目标(基线全 0%,79-RESEARCH §2):
//   duty_stats_service.go(32)/ duty_holiday_service.go(29)/
//   duty_config_service.go(24)/ duty_service.go(24 门面委托)。
//
// 纪律同前:7902 后缀、t.TempDir 文件库、禁 t.Parallel、具名常量、零生产改动。
// 复用 Task 1/2 的同包 helper(seedPool7902/seedSchedule7902/dsc7902User)。
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

// newDtx7902 一个库装配三个小 service(stats/holiday/config)。
func newDtx7902(t *testing.T) (*gorm.DB, *DutyStatsService, *DutyHolidayService, *DutyConfigService) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "dtx7902.db")), &gorm.Config{
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
	return db, NewDutyStatsService(db), NewDutyHolidayService(db), NewDutyConfigService(db)
}

// dtx7902ShiftOffMonthStart 边界自适应 fixture:统计类日期行不能落在本地月初 00:00。
// 原因:行存储为 UTC 零点文本('...01 00:00:00+00:00'),而服务绑定的本月起点是
// 本地零点('...01 00:00:00+08:00'),字符串比较时 '+00' < '+08' 会使恰在月初的行
// 被 SQL 排除而 Go 时间比较包含 —— 同一 fixture 两端口径漂移即 flake,故统一避开。
func dtx7902ShiftOffMonthStart(base time.Time, daysBack int) time.Time {
	c := base.AddDate(0, 0, -daysBack)
	if c.Day() == 1 {
		c = c.AddDate(0, 0, -1)
	}
	return c
}

// GetMyDutyStats:今日在岗 + 本月/总计计数 + 下次值班;零记录用户回零值结构。
func TestDst7902_GetMyDutyStats(t *testing.T) {
	db, statsSvc, _, _ := newDtx7902(t)
	pool := &models.DutyPool{
		BaseModel:  models.BaseModel{CreatedBy: dsc7902Creator},
		PoolName:   "stats-pool",
		Status:     models.DutyPoolStatusEnabled,
		DailyCount: 1,
	}
	require.NoError(t, db.Create(pool).Error)
	pool2 := &models.DutyPool{
		BaseModel:  models.BaseModel{CreatedBy: dsc7902Creator},
		PoolName:   "next-pool",
		Status:     models.DutyPoolStatusEnabled,
		DailyCount: 1,
	}
	require.NoError(t, db.Create(pool2).Error)

	// 与 duty_stats_service.go:24 同一构造 —— 绑定参数与存储文本逐字一致,
	// 这是"today"语义方法;断言只比对统计字段与池名,不断言日期字面值。
	today := time.Now().Truncate(24 * time.Hour)

	type fixture struct {
		userID string
		date   time.Time
		status models.DutyStatus
	}
	fixtures := []fixture{
		// member-a:今天在岗
		{"member-a", today, models.DutyStatusNormal},
		// member-a:本月两条历史(DutyType 不同)
		{"member-a", dtx7902ShiftOffMonthStart(today, 3), models.DutyStatusNormal},
		{"member-a", dtx7902ShiftOffMonthStart(today, 8), models.DutyStatusNormal},
		// member-a:约 40 天前的旧行(计入 total,本月与否由 oracle 判定)
		{"member-a", dtx7902ShiftOffMonthStart(today, 40), models.DutyStatusNormal},
		// member-a:已取消行(所有计数排除)
		{"member-a", dtx7902ShiftOffMonthStart(today, 5), models.DutyStatusCancelled},
		// member-a:未来两行(+5 最近,应为 NextDuty;+9 更远)
		{"member-a", today.AddDate(0, 0, 5), models.DutyStatusNormal},
		{"member-a", today.AddDate(0, 0, 9), models.DutyStatusNormal},
		// member-b:他人行,不进 member-a 统计
		{"member-b", today, models.DutyStatusNormal},
		{"member-b", dtx7902ShiftOffMonthStart(today, 4), models.DutyStatusNormal},
	}
	for i, f := range fixtures {
		seed := seedSchedule7902(t, db, pool.ID, f.userID, f.date, models.ScheduleModeWeekday)
		if f.status != models.DutyStatusNormal {
			require.NoError(t, db.Model(&models.DutySchedule{}).Where("id = ?", seed.ID).
				Update("status", int(f.status)).Error)
		}
		_ = i
	}
	// 未来行归属 next-pool(校验 NextDutyPoolName)
	require.NoError(t, db.Model(&models.DutySchedule{}).
		Where("user_id = ? AND pool_id = ?", "member-a", pool.ID).
		Update("pool_id", pool.ID).Error)
	var futureRows []models.DutySchedule
	require.NoError(t, db.Where("user_id = ? AND schedule_date > ?", "member-a", today).
		Order("schedule_date ASC").Find(&futureRows).Error)
	require.Len(t, futureRows, 2)
	require.NoError(t, db.Model(&models.DutySchedule{}).Where("id = ?", futureRows[0].ID).
		Update("pool_id", pool2.ID).Error)

	// fixture oracle:统计语义相对"今天",期望值由种子数据推导(非镜像实现 SQL)
	startOfMonth := time.Date(today.Year(), today.Month(), 1, 0, 0, 0, 0, time.Local)
	wantMonth, wantTotal := 0, 0
	for _, f := range fixtures {
		if f.userID != "member-a" || f.status != models.DutyStatusNormal {
			continue
		}
		wantTotal++
		if f.date.After(startOfMonth) && (f.date.Before(today) || f.date.Equal(today)) {
			wantMonth++
		}
	}

	stats, err := statsSvc.GetMyDutyStats(context.Background(), "member-a")
	require.NoError(t, err)
	require.NotNil(t, stats)
	assert.True(t, stats.IsOnDutyToday, "今天有 normal 排班 → 在岗")
	require.Len(t, stats.TodayDutyRecords, 1, "今日值班记录恰一条")
	assert.Equal(t, "member-a", stats.TodayDutyRecords[0].UserID)
	assert.Equal(t, "stats-pool", stats.TodayDutyRecords[0].PoolName)
	assert.Equal(t, wantMonth, stats.ThisMonthCount, "本月计数与 fixture oracle 一致")
	assert.Equal(t, wantTotal, stats.TotalCount, "总计数与 fixture oracle 一致(取消行排除)")

	// 下次值班 = 最近的未来行(+5,非 +9),池名取该行归属
	require.NotNil(t, stats.NextDutyDate, "存在未来排班 → NextDutyDate 非 nil")
	assert.Equal(t, futureRows[0].ScheduleDate.Format("2006-01-02"), *stats.NextDutyDate)
	require.NotNil(t, stats.NextDutyPoolName)
	assert.Equal(t, "next-pool", *stats.NextDutyPoolName)

	// 无记录用户 → 零值结构非 nil、不报错、指针字段为 nil
	empty, err := statsSvc.GetMyDutyStats(context.Background(), "nobody")
	require.NoError(t, err, "无记录不算错误")
	require.NotNil(t, empty)
	assert.False(t, empty.IsOnDutyToday)
	assert.Empty(t, empty.TodayDutyRecords)
	assert.Zero(t, empty.ThisMonthCount)
	assert.Zero(t, empty.TotalCount)
	assert.Nil(t, empty.NextDutyDate)
	assert.Nil(t, empty.NextDutyPoolName)
}

// Holiday 六方法:Create → List(按年) → Update → Delete;跨两年数据;
// GetHolidayYears 去重 + 降序(QUIRK-79-02-H:计划文案写"升序",实装 sort.Reverse → 降序,锁定)。
func TestDhd7902_HolidayCRUD(t *testing.T) {
	db, _, holidaySvc, _ := newDtx7902(t)

	// Create(2026 两条 + 2025 一条)
	h1 := &models.Holiday{
		BaseModel:   models.BaseModel{CreatedBy: dsc7902Creator},
		HolidayDate: dsc7902Tue,
		HolidayName: "元旦",
		IsOffday:    true,
		HolidayType: models.HolidayTypeLegal,
		Year:        2026,
	}
	require.NoError(t, holidaySvc.CreateHoliday(context.Background(), h1, dsc7902Creator))
	h2 := &models.Holiday{
		BaseModel:   models.BaseModel{CreatedBy: dsc7902Creator},
		HolidayDate: dsc7902Mon,
		HolidayName: "调休上班",
		IsOffday:    false,
		HolidayType: models.HolidayTypeWorkday,
		Year:        2026,
	}
	require.NoError(t, holidaySvc.CreateHoliday(context.Background(), h2, "creator-2"))

	// List 按年过滤 + 升序
	list2026, err := holidaySvc.GetHolidayList(context.Background(), 2026)
	require.NoError(t, err)
	require.Len(t, list2026, 2, "只返回 2026 年")
	assert.Equal(t, "2026-03-02", list2026[0].HolidayDate.Format("2006-01-02"), "holiday_date ASC")
	assert.Equal(t, "2026-03-03", list2026[1].HolidayDate.Format("2006-01-02"))
	assert.Equal(t, "creator-2", list2026[0].CreatedBy, "CreatedBy 透传")

	list2025, err := holidaySvc.GetHolidayList(context.Background(), 2025)
	require.NoError(t, err)
	assert.Empty(t, list2025, "无 2025 数据")

	// Update(Save 全量)
	h1.HolidayName = "元旦(改)"
	h1.IsOffday = false
	require.NoError(t, holidaySvc.UpdateHoliday(context.Background(), h1, "updater-7902"))
	after, err := holidaySvc.GetHolidayList(context.Background(), 2026)
	require.NoError(t, err)
	require.Len(t, after, 2)
	updated := after[1]
	assert.Equal(t, "元旦(改)", updated.HolidayName)
	assert.False(t, updated.IsOffday)
	assert.Equal(t, "updater-7902", updated.UpdatedBy)

	// GetHolidayYears:去重 + 降序(实装 sort.Reverse(sort.IntSlice),锁定现行为)
	require.NoError(t, db.Create(&models.Holiday{
		BaseModel:   models.BaseModel{CreatedBy: dsc7902Creator},
		HolidayDate: dsc7902Date(2025, 5, 1),
		HolidayName: "劳动节",
		IsOffday:    true,
		HolidayType: models.HolidayTypeLegal,
		Year:        2025,
	}).Error)
	years, err := holidaySvc.GetHolidayYears(context.Background())
	require.NoError(t, err)
	assert.Equal(t, []int{2026, 2025}, years, "去重 + 降序(最近年份在前)")

	// BatchCreateHolidays:2027 两条 → 年份列表追加。
	// 日期避开 1 月 1 日 UTC 零点(该形态会被年过滤下界排除,见 DateShape 的 QUIRK-79-02-J)。
	batch := []models.Holiday{
		{
			BaseModel: models.BaseModel{CreatedBy: dsc7902Creator}, HolidayDate: dsc7902Date(2027, 1, 2),
			HolidayName: "春节", IsOffday: true, HolidayType: models.HolidayTypeLegal, Year: 2027,
		},
		{
			BaseModel: models.BaseModel{CreatedBy: dsc7902Creator}, HolidayDate: dsc7902Date(2027, 1, 3),
			HolidayName: "初二", IsOffday: true, HolidayType: models.HolidayTypeLegal, Year: 2027,
		},
	}
	require.NoError(t, holidaySvc.BatchCreateHolidays(context.Background(), batch, "batch-creator"))
	list2027, err := holidaySvc.GetHolidayList(context.Background(), 2027)
	require.NoError(t, err)
	require.Len(t, list2027, 2)
	assert.Equal(t, "batch-creator", list2027[0].CreatedBy, "批量路径也逐条回填 CreatedBy")

	// QUIRK-79-02-I(只锁不修):空切片 → GORM ErrEmptySlice,实装包装为
	// "批量创建节假日失败"(handler binding 层负责非空,service 层透传驱动错误)。
	err = holidaySvc.BatchCreateHolidays(context.Background(), []models.Holiday{}, dsc7902Creator)
	require.Error(t, err, "空批量按 GORM 语义报错(锁定现行为)")
	assert.Contains(t, err.Error(), "批量创建节假日失败")

	// Delete
	require.NoError(t, holidaySvc.DeleteHoliday(context.Background(), h2.ID))
	listAfter, err := holidaySvc.GetHolidayList(context.Background(), 2026)
	require.NoError(t, err)
	assert.Len(t, listAfter, 1, "删除后 2026 只剩一条")
	// 删除不存在的 ID → 不报错(软删 0 行)
	require.NoError(t, holidaySvc.DeleteHoliday(context.Background(), "no-such-holiday"))
}

// HolidayDate 形态锁定:读回的日期部分与写入一致(GetHolidayList 年过滤与
// GenerateSchedule holidayMap 键都依赖该形态),防止时区/时分污染引发 flake。
func TestDhd7902_Holiday_DateShape(t *testing.T) {
	db, _, holidaySvc, _ := newDtx7902(t)

	h := &models.Holiday{
		BaseModel:   models.BaseModel{CreatedBy: dsc7902Creator},
		HolidayDate: dsc7902Tue, // UTC 零点,与生产 time.Parse("2006-01-02") 同形态
		HolidayName: "形态锁定",
		IsOffday:    true,
		HolidayType: models.HolidayTypeCustom,
		Year:        2026,
	}
	require.NoError(t, holidaySvc.CreateHoliday(context.Background(), h, dsc7902Creator))

	var stored models.Holiday
	require.NoError(t, db.Where("id = ?", h.ID).First(&stored).Error)
	assert.Equal(t, "2026-03-03", stored.HolidayDate.Format("2006-01-02"),
		"日期部分不漂移(年过滤与 holidayMap 键的形态依赖)")
	assert.Zero(t, stored.HolidayDate.Hour()+stored.HolidayDate.Minute(),
		"UTC 零点形态:时/分为 0(drv 存取同值)")

	// 形态的可观察后果:2026 年过滤命中;2027 不命中
	in2026, err := holidaySvc.GetHolidayList(context.Background(), 2026)
	require.NoError(t, err)
	assert.Len(t, in2026, 1)
	in2027, err := holidaySvc.GetHolidayList(context.Background(), 2027)
	require.NoError(t, err)
	assert.Empty(t, in2027)

	// QUIRK-79-02-J(只锁不修):恰好落在年初 UTC 零点('2027-01-01 00:00:00+00:00')
	// 的行会被 GetHolidayList(2027) 下界排除 —— 绑定参数是本地零点
	// ('2027-01-01 00:00:00+08:00'),sqlite 按 TEXT 逐字比较时 '+00' < '+08'。
	// 生产侧 time.Parse("2006-01-02") 同样产出 UTC 零点,即元旦(1/1)节假日
	// 在 +08 时区不会出现在当年列表。属现网行为,修需动绑定/存储形态(生产改动,
	// 走 escape hatch),此处仅锁定证据。
	jan1 := &models.Holiday{
		BaseModel:   models.BaseModel{CreatedBy: dsc7902Creator},
		HolidayDate: dsc7902Date(2027, 1, 1),
		HolidayName: "元旦(边界形态)",
		IsOffday:    true,
		HolidayType: models.HolidayTypeLegal,
		Year:        2027,
	}
	require.NoError(t, holidaySvc.CreateHoliday(context.Background(), jan1, dsc7902Creator))
	in2027Again, err := holidaySvc.GetHolidayList(context.Background(), 2027)
	require.NoError(t, err)
	assert.Empty(t, in2027Again, "年初 UTC 零点行被年过滤排除(QUIRK-79-02-J 锁定)")
}

// GetDutyConfig 空表回默认(:26-34):ReminderEnabled=true / "08:00" / "websocket" / nil。
func TestDcf7902_GetConfig_EmptyBackfillsDefault(t *testing.T) {
	_, _, _, configSvc := newDtx7902(t)

	cfg, err := configSvc.GetDutyConfig(context.Background())
	require.NoError(t, err)
	require.NotNil(t, cfg)
	assert.True(t, cfg.ReminderEnabled, "默认启用提醒")
	assert.Equal(t, "08:00", cfg.ReminderTime, "默认提醒时间(HH:mm)")
	assert.Equal(t, "websocket", cfg.ReminderChannels, "默认提醒渠道")
	assert.Nil(t, cfg.BeforeReminderMinutes, "默认无提前提醒分钟数")
}

// UpdateDutyConfig 往返:空表首次 Update 走创建分支;再 Update 走更新分支(行数不变)。
func TestDcf7902_UpdateConfig_RoundTrip(t *testing.T) {
	db, _, _, configSvc := newDtx7902(t)

	minutes := 15
	toCreate := &models.DutyConfig{
		ReminderEnabled:       true,
		ReminderTime:          "09:30",
		ReminderChannels:      "websocket,email",
		BeforeReminderMinutes: &minutes,
	}
	// 首次 Update:空表 → 创建分支(Created/UpdatedBy=updater)
	require.NoError(t, configSvc.UpdateDutyConfig(context.Background(), toCreate, "cfg-admin"))

	var rows []models.DutyConfig
	require.NoError(t, db.Find(&rows).Error)
	require.Len(t, rows, 1, "空表首次 Update 创建唯一一行")
	createdID := rows[0].ID
	assert.Equal(t, "cfg-admin", rows[0].CreatedBy)
	assert.Equal(t, "cfg-admin", rows[0].UpdatedBy)

	// Get 读回一致
	cfg, err := configSvc.GetDutyConfig(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "09:30", cfg.ReminderTime)
	assert.Equal(t, "websocket,email", cfg.ReminderChannels)
	require.NotNil(t, cfg.BeforeReminderMinutes)
	assert.Equal(t, 15, *cfg.BeforeReminderMinutes)

	// 再 Update:走更新分支 —— ID/CreatedBy 保留,行数不变
	newMinutes := 30
	toUpdate := &models.DutyConfig{
		ReminderEnabled:       false,
		ReminderTime:          "07:45",
		ReminderChannels:      "sms",
		BeforeReminderMinutes: &newMinutes,
	}
	require.NoError(t, configSvc.UpdateDutyConfig(context.Background(), toUpdate, "cfg-admin-2"))

	require.NoError(t, db.Find(&rows).Error)
	require.Len(t, rows, 1, "更新分支不新建行")
	assert.Equal(t, createdID, rows[0].ID, "Save 复用原 ID")
	assert.Equal(t, "cfg-admin", rows[0].CreatedBy, "CreatedBy 保留原值")
	assert.Equal(t, "cfg-admin-2", rows[0].UpdatedBy)
	assert.False(t, rows[0].ReminderEnabled)
	assert.Equal(t, "07:45", rows[0].ReminderTime)
	assert.Equal(t, "sms", rows[0].ReminderChannels)
	require.NotNil(t, rows[0].BeforeReminderMinutes)
	assert.Equal(t, 30, *rows[0].BeforeReminderMinutes)
}

// ==================== Task 5: duty_service 门面委托链 ====================

// QUIRK 说明:门面 duty_service.go 共 21 个委托方法(池 6 + 排班 6 + stats 1 +
// holiday 6 + config 2),全部单行转发。按 D-79-07 轻量口径:池域 4 方法做
// 完整 round-trip,排班域 3 方法做主干链,其余以"调用形态正确 + 错误分支
// 透传"覆盖,不重复下层全量断言(下层已由 Task 1-4 覆盖)。
func TestDsv7902_FacadeDelegation(t *testing.T) {
	db, _, _, _ := newDtx7902(t)
	facade := NewDutyService(db)
	// 池域成员走存在性校验 → 先落 sys_user
	dsc7902User(t, db, "member-a", "user-member-a", nil, nil)
	dsc7902User(t, db, "member-b", "user-member-b", nil, nil)

	// ---- 池域 round-trip:Create → GetByID → Update → Delete ----
	created, err := facade.CreateDutyPool(context.Background(), &DutyPoolCreateRequest{
		PoolName:   "facade-pool",
		DailyCount: 2,
		MemberIDs:  []string{"member-a"},
	}, dsc7902Creator)
	require.NoError(t, err, "门面 CreateDutyPool 与直调子 service 等价")
	require.NotNil(t, created)

	byID, err := facade.GetDutyPoolByID(context.Background(), created.ID)
	require.NoError(t, err)
	require.Len(t, byID.Members, 1)
	assert.Equal(t, "member-a", byID.Members[0].UserID, "门面读回与直调一致")

	list, total, err := facade.GetDutyPoolList(context.Background(), &DutyPoolListRequest{
		PoolName: &[]string{"facade"}[0],
	})
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	require.Len(t, list, 1)

	require.NoError(t, facade.UpdateDutyPool(context.Background(), &DutyPoolUpdateRequest{
		ID:         created.ID,
		PoolName:   "facade-pool-v2",
		DailyCount: 3,
		MemberIDs:  []string{"member-a", "member-b"},
	}, "facade-updater"))
	afterUpdate, err := facade.GetDutyPoolByID(context.Background(), created.ID)
	require.NoError(t, err)
	assert.Equal(t, "facade-pool-v2", afterUpdate.PoolName)
	assert.Equal(t, 3, afterUpdate.DailyCount)

	// 错误分支透传:门面原样返回子 service 错误
	_, err = facade.GetDutyPoolByID(context.Background(), "no-such-pool")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "值班池不存在", "错误透传不变形")

	// ---- 排班域主干链:GenerateSchedule → GetDutyScheduleList → BatchDelete ----
	count, err := facade.GenerateSchedule(context.Background(), &GenerateScheduleRequest{
		PoolID:    created.ID,
		StartDate: "2026-03-02",
		EndDate:   "2026-03-04",
		DutyType:  string(models.ScheduleModeWeekday),
	}, dsc7902Creator)
	require.NoError(t, err)
	assert.Equal(t, 9, count, "3 个工作日 × DailyCount 3")

	schedules, total, err := facade.GetDutyScheduleList(context.Background(), &DutyScheduleListRequest{
		PoolID: &created.ID,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(9), total)
	require.Len(t, schedules, 9)

	// 手工排班 + 删除族
	require.NoError(t, facade.ManualDuty(context.Background(), &ManualDutyRequest{
		PoolID:   created.ID,
		DutyDate: "2026-03-07",
		UserIDs:  []string{"member-a"},
		DutyType: string(models.ScheduleModeWeekend),
		Reason:   "facade 手工",
	}, dsc7902Creator))
	require.NoError(t, facade.BatchDeleteDutySchedules(context.Background(),
		[]string{schedules[0].ID, schedules[1].ID}))
	_, totalAfter, err := facade.GetDutyScheduleList(context.Background(), &DutyScheduleListRequest{
		PoolID: &created.ID,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(8), totalAfter, "批删 2 行后剩 8 行(9 自动 -2 +1 手工)")
	require.NoError(t, facade.DeleteDutySchedule(context.Background(), schedules[2].ID))
	_, totalAfterSingle, err := facade.GetDutyScheduleList(context.Background(), &DutyScheduleListRequest{
		PoolID: &created.ID,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(7), totalAfterSingle, "单删后剩 7 行")

	// ---- 其余委托方法:调用形态正确 / 错误分支透传,不重复下层全量断言 ----

	// stats 域
	stats, err := facade.GetMyDutyStats(context.Background(), "member-a")
	require.NoError(t, err)
	require.NotNil(t, stats, "门面 GetMyDutyStats 返回形态正确")

	// holiday 域全链
	holiday := &models.Holiday{
		BaseModel:   models.BaseModel{CreatedBy: dsc7902Creator},
		HolidayDate: dsc7902Tue,
		HolidayName: "门面节假日",
		IsOffday:    true,
		HolidayType: models.HolidayTypeCustom,
		Year:        2026,
	}
	require.NoError(t, facade.CreateHoliday(context.Background(), holiday, dsc7902Creator))
	holidays, err := facade.GetHolidayList(context.Background(), 2026)
	require.NoError(t, err)
	require.Len(t, holidays, 1)
	years, err := facade.GetHolidayYears(context.Background())
	require.NoError(t, err)
	assert.Equal(t, []int{2026}, years)
	require.NoError(t, facade.UpdateHoliday(context.Background(), &holidays[0], "facade-updater"))
	require.NoError(t, facade.DeleteHoliday(context.Background(), holidays[0].ID))
	// 批量创建用新日期:删除是软删,holiday_date 的硬唯一索引仍被已删行占位
	// (QUIRK-79-02-K:同日期节假日"删后再建"会撞 UNIQUE 约束,现行为锁定)
	rebatch := &models.Holiday{
		BaseModel:   models.BaseModel{CreatedBy: dsc7902Creator},
		HolidayDate: dsc7902Thu,
		HolidayName: "门面批量节假日",
		IsOffday:    true,
		HolidayType: models.HolidayTypeCustom,
		Year:        2026,
	}
	require.NoError(t, facade.BatchCreateHolidays(context.Background(),
		[]models.Holiday{*rebatch}, dsc7902Creator))

	// config 域
	require.NoError(t, facade.UpdateDutyConfig(context.Background(), &models.DutyConfig{
		ReminderEnabled: true,
		ReminderTime:    "08:30",
	}, "facade-cfg"))
	cfg, err := facade.GetDutyConfig(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "08:30", cfg.ReminderTime, "门面 config 委托读写一致")

	// 池域收尾:statistics + delete(有排班记录 → 拒删透传)
	poolStats, err := facade.GetDutyPoolStatistics(context.Background())
	require.NoError(t, err)
	assert.Equal(t, int64(1), poolStats.Total)
	err = facade.DeleteDutyPool(context.Background(), created.ID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "该值班池存在排班记录，无法删除", "门面错误透传")

	// 换班:两行互换(错误分支传坏参数)
	var remain []models.DutySchedule
	require.NoError(t, db.Where("pool_id = ?", created.ID).Find(&remain).Error)
	require.Len(t, remain, 7, "9 自动 -2 批删 -1 单删 +1 手工")
	require.NoError(t, facade.SwapDuty(context.Background(), &SwapDutyRequest{
		FromScheduleID: remain[0].ID,
		ToScheduleID:   remain[1].ID,
		Reason:         "facade 换班",
	}, "facade-operator"))
	err = facade.SwapDuty(context.Background(), &SwapDutyRequest{
		FromScheduleID: "no-such-id",
		ToScheduleID:   remain[0].ID,
	}, "facade-operator")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "原排班记录不存在")

	// 今日值班:无今天行 → 错误透传(不断言日期值)
	_, err = facade.GetTodayDuty(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "今日无值班人员")
	monthly, err := facade.GetMonthlyDutySchedule(context.Background(), 2026, 3)
	require.NoError(t, err)
	assert.NotNil(t, monthly, "门面 GetMonthlyDutySchedule 返回 map 形态正确")
}
