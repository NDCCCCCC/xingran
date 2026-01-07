package main

import (
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Holiday 节假日
type Holiday struct {
	ID          string     `gorm:"primaryKey;type:uuid" json:"id"`
	HolidayDate time.Time  `gorm:"type:date;not null;uniqueIndex:idx_holiday_date" json:"holidayDate"`
	HolidayName string     `gorm:"size:100;not null" json:"holidayName"`
	IsOffday    *bool      `gorm:"default:true" json:"isOffday"` // 使用指针避免零值被忽略，true=休息日 false=调休工作日
	HolidayType string     `gorm:"size:20;default:'custom'" json:"holidayType"`
	Year        int        `gorm:"not null;index:idx_holiday_year" json:"year"`
	Remark      string     `gorm:"size:500" json:"remark,omitempty"`
	CreatedAt   time.Time  `json:"createdAt"`
	UpdatedAt   time.Time  `json:"updatedAt"`
}

// TableName 指定表名
func (Holiday) TableName() string {
	return "sys_holiday"
}

// boolPtr 返回bool的指针辅助函数
func boolPtr(b bool) *bool {
	return &b
}

func main() {
	// 连接数据库
	dsn := "host=10.62.10.34 port=5432 user=postgres password=Cpic1234 dbname=ruoyi sslmode=disable TimeZone=Asia/Shanghai"
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		log.Fatal("连接数据库失败:", err)
	}

	// 2026年节假日数据
	holidays := []Holiday{
		// 元旦：1月1日-1月3日
		{ID: uuid.New().String(), HolidayDate: parseDate("2026-01-01"), HolidayName: "元旦", IsOffday: boolPtr(true), HolidayType: "legal", Year: 2026, Remark: "法定节假日"},
		{ID: uuid.New().String(), HolidayDate: parseDate("2026-01-02"), HolidayName: "元旦", IsOffday: boolPtr(true), HolidayType: "legal", Year: 2026, Remark: "法定节假日"},
		{ID: uuid.New().String(), HolidayDate: parseDate("2026-01-03"), HolidayName: "元旦", IsOffday: boolPtr(true), HolidayType: "legal", Year: 2026, Remark: "法定节假日"},

		// 春节：2月14日-2月20日（除夕至正月初六）
		{ID: uuid.New().String(), HolidayDate: parseDate("2026-02-14"), HolidayName: "春节", IsOffday: boolPtr(true), HolidayType: "legal", Year: 2026, Remark: "除夕"},
		{ID: uuid.New().String(), HolidayDate: parseDate("2026-02-15"), HolidayName: "春节", IsOffday: boolPtr(true), HolidayType: "legal", Year: 2026, Remark: "正月初一"},
		{ID: uuid.New().String(), HolidayDate: parseDate("2026-02-16"), HolidayName: "春节", IsOffday: boolPtr(true), HolidayType: "legal", Year: 2026, Remark: "正月初二"},
		{ID: uuid.New().String(), HolidayDate: parseDate("2026-02-17"), HolidayName: "春节", IsOffday: boolPtr(true), HolidayType: "legal", Year: 2026, Remark: "正月初三"},
		{ID: uuid.New().String(), HolidayDate: parseDate("2026-02-18"), HolidayName: "春节", IsOffday: boolPtr(true), HolidayType: "legal", Year: 2026, Remark: "正月初四"},
		{ID: uuid.New().String(), HolidayDate: parseDate("2026-02-19"), HolidayName: "春节", IsOffday: boolPtr(true), HolidayType: "legal", Year: 2026, Remark: "正月初五"},
		{ID: uuid.New().String(), HolidayDate: parseDate("2026-02-20"), HolidayName: "春节", IsOffday: boolPtr(true), HolidayType: "legal", Year: 2026, Remark: "正月初六"},

		// 春节调休工作日
		{ID: uuid.New().String(), HolidayDate: parseDate("2026-02-08"), HolidayName: "春节调休", IsOffday: boolPtr(false), HolidayType: "workday", Year: 2026, Remark: "春节假期调休补班"},
		{ID: uuid.New().String(), HolidayDate: parseDate("2026-02-22"), HolidayName: "春节调休", IsOffday: boolPtr(false), HolidayType: "workday", Year: 2026, Remark: "春节假期调休补班"},

		// 清明节：4月4日-4月6日
		{ID: uuid.New().String(), HolidayDate: parseDate("2026-04-04"), HolidayName: "清明节", IsOffday: boolPtr(true), HolidayType: "legal", Year: 2026, Remark: "法定节假日"},
		{ID: uuid.New().String(), HolidayDate: parseDate("2026-04-05"), HolidayName: "清明节", IsOffday: boolPtr(true), HolidayType: "legal", Year: 2026, Remark: "法定节假日"},
		{ID: uuid.New().String(), HolidayDate: parseDate("2026-04-06"), HolidayName: "清明节", IsOffday: boolPtr(true), HolidayType: "legal", Year: 2026, Remark: "法定节假日"},

		// 劳动节：5月1日-5月5日
		{ID: uuid.New().String(), HolidayDate: parseDate("2026-05-01"), HolidayName: "劳动节", IsOffday: boolPtr(true), HolidayType: "legal", Year: 2026, Remark: "法定节假日"},
		{ID: uuid.New().String(), HolidayDate: parseDate("2026-05-02"), HolidayName: "劳动节", IsOffday: boolPtr(true), HolidayType: "legal", Year: 2026, Remark: "法定节假日"},
		{ID: uuid.New().String(), HolidayDate: parseDate("2026-05-03"), HolidayName: "劳动节", IsOffday: boolPtr(true), HolidayType: "legal", Year: 2026, Remark: "法定节假日"},
		{ID: uuid.New().String(), HolidayDate: parseDate("2026-05-04"), HolidayName: "劳动节", IsOffday: boolPtr(true), HolidayType: "legal", Year: 2026, Remark: "法定节假日"},
		{ID: uuid.New().String(), HolidayDate: parseDate("2026-05-05"), HolidayName: "劳动节", IsOffday: boolPtr(true), HolidayType: "legal", Year: 2026, Remark: "法定节假日"},

		// 劳动节调休工作日
		{ID: uuid.New().String(), HolidayDate: parseDate("2026-05-09"), HolidayName: "劳动节调休", IsOffday: boolPtr(false), HolidayType: "workday", Year: 2026, Remark: "劳动节假期调休补班"},

		// 端午节：5月31日-6月2日
		{ID: uuid.New().String(), HolidayDate: parseDate("2026-05-31"), HolidayName: "端午节", IsOffday: boolPtr(true), HolidayType: "legal", Year: 2026, Remark: "法定节假日"},
		{ID: uuid.New().String(), HolidayDate: parseDate("2026-06-01"), HolidayName: "端午节", IsOffday: boolPtr(true), HolidayType: "legal", Year: 2026, Remark: "法定节假日"},
		{ID: uuid.New().String(), HolidayDate: parseDate("2026-06-02"), HolidayName: "端午节", IsOffday: boolPtr(true), HolidayType: "legal", Year: 2026, Remark: "法定节假日"},

		// 中秋节：9月19日-9月21日
		{ID: uuid.New().String(), HolidayDate: parseDate("2026-09-19"), HolidayName: "中秋节", IsOffday: boolPtr(true), HolidayType: "legal", Year: 2026, Remark: "法定节假日"},
		{ID: uuid.New().String(), HolidayDate: parseDate("2026-09-20"), HolidayName: "中秋节", IsOffday: boolPtr(true), HolidayType: "legal", Year: 2026, Remark: "法定节假日"},
		{ID: uuid.New().String(), HolidayDate: parseDate("2026-09-21"), HolidayName: "中秋节", IsOffday: boolPtr(true), HolidayType: "legal", Year: 2026, Remark: "法定节假日"},

		// 中秋节调休工作日（注意：9月20日既是节假日又是调休日，实际应该以节假日为准，这里取消调休）
		// 国庆节：10月1日-10月7日
		{ID: uuid.New().String(), HolidayDate: parseDate("2026-10-01"), HolidayName: "国庆节", IsOffday: boolPtr(true), HolidayType: "legal", Year: 2026, Remark: "法定节假日"},
		{ID: uuid.New().String(), HolidayDate: parseDate("2026-10-02"), HolidayName: "国庆节", IsOffday: boolPtr(true), HolidayType: "legal", Year: 2026, Remark: "法定节假日"},
		{ID: uuid.New().String(), HolidayDate: parseDate("2026-10-03"), HolidayName: "国庆节", IsOffday: boolPtr(true), HolidayType: "legal", Year: 2026, Remark: "法定节假日"},
		{ID: uuid.New().String(), HolidayDate: parseDate("2026-10-04"), HolidayName: "国庆节", IsOffday: boolPtr(true), HolidayType: "legal", Year: 2026, Remark: "法定节假日"},
		{ID: uuid.New().String(), HolidayDate: parseDate("2026-10-05"), HolidayName: "国庆节", IsOffday: boolPtr(true), HolidayType: "legal", Year: 2026, Remark: "法定节假日"},
		{ID: uuid.New().String(), HolidayDate: parseDate("2026-10-06"), HolidayName: "国庆节", IsOffday: boolPtr(true), HolidayType: "legal", Year: 2026, Remark: "法定节假日"},
		{ID: uuid.New().String(), HolidayDate: parseDate("2026-10-07"), HolidayName: "国庆节", IsOffday: boolPtr(true), HolidayType: "legal", Year: 2026, Remark: "法定节假日"},

		// 国庆节调休工作日
		{ID: uuid.New().String(), HolidayDate: parseDate("2026-10-10"), HolidayName: "国庆调休", IsOffday: boolPtr(false), HolidayType: "workday", Year: 2026, Remark: "国庆节假期调休补班"},
	}

	// 先删除2026年的旧数据
	fmt.Println("删除2026年的旧节假日数据...")
	db.Where("year = ?", 2026).Delete(&Holiday{})

	// 批量插入新数据
	fmt.Println("导入2026年节假日数据...")
	fmt.Println("\n导入前调休工作日数据预览：")
	for _, h := range holidays {
		if h.HolidayType == "workday" {
			fmt.Printf("  %s - IsOffday=%v (类型:%s)\n", h.HolidayDate.Format("2006-01-02"), *h.IsOffday, h.HolidayType)
		}
	}

	now := time.Now()
	for i := range holidays {
		holidays[i].CreatedAt = now
		holidays[i].UpdatedAt = now
	}

	if err := db.Create(&holidays).Error; err != nil {
		log.Fatal("导入失败:", err)
	}

	// 查询并显示导入结果
	var count int64
	db.Model(&Holiday{}).Where("year = ?", 2026).Count(&count)
	fmt.Printf("\n成功导入 %d 条2026年节假日数据！\n", count)

	// 使用SQL直接查询调休工作日的is_offday值
	fmt.Println("\n验证调休工作日数据（直接SQL查询）：")
	type SQLResult struct {
		HolidayDate string
		HolidayName string
		IsOffday    bool
		HolidayType string
	}
	var sqlResults []SQLResult
	db.Raw("SELECT holiday_date, holiday_name, is_offday, holiday_type FROM sys_holiday WHERE year = 2026 AND holiday_type = 'workday' ORDER BY holiday_date").Scan(&sqlResults)
	for _, r := range sqlResults {
		offday := "是"
		if !r.IsOffday {
			offday = "否（调休补班）"
		}
		fmt.Printf("  %s - %s - is_offday=%v - type=%s [%s]\n", r.HolidayDate, r.HolidayName, r.IsOffday, r.HolidayType, offday)
	}

	// 显示导入的节假日列表
	var result []Holiday
	db.Where("year = ?", 2026).Order("holiday_date").Find(&result)
	fmt.Println("\n2026年节假日列表：")
	fmt.Println("日期\t\t名称\t\t类型\t\t是否休息")
	fmt.Println("----------------------------------------------------------------")
	for _, h := range result {
		offday := "是"
		// IsOffday是指针类型，需要解引用并检查是否为nil
		if h.IsOffday != nil && !*h.IsOffday {
			offday = "否（调休补班）"
		}
		fmt.Printf("%s\t%s\t%s\t%s\n", h.HolidayDate.Format("2006-01-02"), h.HolidayName, h.HolidayType, offday)
	}
}

func parseDate(dateStr string) time.Time {
	t, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		log.Fatal("日期解析失败:", err)
	}
	return t
}
