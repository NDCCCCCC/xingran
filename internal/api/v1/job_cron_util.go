package v1

import (
	"fmt"
	"time"

	"github.com/robfig/cron/v3"
)

// ==================== Cron 表达式处理工具 ====================

// validateCronExpression 验证Cron表达式
func validateCronExpression(expr string) bool {
	_, err := cron.NewParser(cron.Second | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor).Parse(expr)
	return err == nil
}

// calculateNextRunTime 计算下次执行时间
func calculateNextRunTime(expr string) time.Time {
	// 使用 cron 实例计算下次执行时间（带时区支持）
	// time.Local 已在 main.go 中设置为 Asia/Shanghai
	c := cron.New(cron.WithSeconds(), cron.WithLocation(time.Local))
	entryID, err := c.AddFunc(expr, func() {})
	if err != nil {
		return time.Time{}
	}
	// 必须启动 cron 才能获取正确的 Next 时间
	c.Start()
	defer c.Stop()
	return c.Entry(entryID).Next
}

// GetCronDescription 获取Cron表达式描述
func GetCronDescription(expr string) string {
	// 简化实现，返回常见表达式的描述
	descriptions := map[string]string{
		"0 * * * * ?":     "每分钟执行",
		"0 0 * * * ?":     "每小时执行",
		"0 0 0 * * ?":     "每天0点执行",
		"0 0 0 * * MON ?": "每周一0点执行",
		"0 0 0 1 * ?":     "每月1号0点执行",
	}

	if desc, ok := descriptions[expr]; ok {
		return desc
	}

	// 解析表达式
	if _, err := cron.NewParser(cron.Second | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor).Parse(expr); err != nil {
		return "无效的Cron表达式"
	}

	nextTime := calculateNextRunTime(expr)
	return "下次执行: " + nextTime.Format("2006-01-02 15:04:05")
}

// ParseCronExpression 解析Cron表达式并返回各部分
func ParseCronExpression(expr string) (seconds, minutes, hours, dayOfMonth, month, dayOfWeek string, err error) {
	parts := make([]string, 6)
	if expr == "" {
		return "", "", "", "", "", "", fmt.Errorf("表达式不能为空")
	}

	// 处理6位或7位Cron表达式
	if len(expr) > 0 {
		idx := 0
		for _, part := range expr {
			if part == ' ' {
				idx++
				if idx > 5 {
					break
				}
				continue
			}
			parts[idx] += string(part)
		}
	}

	if len(parts) < 6 {
		return "", "", "", "", "", "", fmt.Errorf("Cron表达式格式错误")
	}

	return parts[0], parts[1], parts[2], parts[3], parts[4], parts[5], nil
}

// ValidateCronExpression 验证并返回详细的Cron表达式信息
func ValidateCronExpression(expr string) (valid bool, nextRunTime string, description string, err error) {
	if !validateCronExpression(expr) {
		return false, "", "", fmt.Errorf("无效的Cron表达式")
	}

	nextTime := calculateNextRunTime(expr)
	return true, nextTime.Format("2006-01-02 15:04:05"), GetCronDescription(expr), nil
}

// generateCronExpression 生成Cron表达式
func generateCronExpression(seconds, minutes, hours, dayOfMonth, month, dayOfWeek string) string {
	return fmt.Sprintf("%s %s %s %s %s %s", seconds, minutes, hours, dayOfMonth, month, dayOfWeek)
}

// CronExpressionBuilder Cron表达式构建器
type CronExpressionBuilder struct {
	seconds   string
	minutes   string
	hours     string
	day       string
	month     string
	dayOfWeek string
}

// NewCronExpressionBuilder 创建Cron表达式构建器
func NewCronExpressionBuilder() *CronExpressionBuilder {
	return &CronExpressionBuilder{
		seconds:   "0",
		minutes:   "*",
		hours:     "*",
		day:       "*",
		month:     "*",
		dayOfWeek: "?",
	}
}

// SetSeconds 设置秒
func (b *CronExpressionBuilder) SetSeconds(seconds string) *CronExpressionBuilder {
	b.seconds = seconds
	return b
}

// SetMinutes 设置分钟
func (b *CronExpressionBuilder) SetMinutes(minutes string) *CronExpressionBuilder {
	b.minutes = minutes
	return b
}

// SetHours 设置小时
func (b *CronExpressionBuilder) SetHours(hours string) *CronExpressionBuilder {
	b.hours = hours
	return b
}

// SetDay 设置日
func (b *CronExpressionBuilder) SetDay(day string) *CronExpressionBuilder {
	b.day = day
	return b
}

// SetMonth 设置月
func (b *CronExpressionBuilder) SetMonth(month string) *CronExpressionBuilder {
	b.month = month
	return b
}

// SetDayOfWeek 设置星期
func (b *CronExpressionBuilder) SetDayOfWeek(dayOfWeek string) *CronExpressionBuilder {
	b.dayOfWeek = dayOfWeek
	return b
}

// Build 构建Cron表达式
func (b *CronExpressionBuilder) Build() string {
	return generateCronExpression(b.seconds, b.minutes, b.hours, b.day, b.month, b.dayOfWeek)
}

// EveryMinute 每分钟执行
func EveryMinute() string {
	return NewCronExpressionBuilder().Build()
}

// EveryHour 每小时执行
func EveryHour() string {
	return NewCronExpressionBuilder().SetMinutes("0").Build()
}

// EveryDay 每天0点执行
func EveryDay() string {
	return NewCronExpressionBuilder().SetMinutes("0").SetHours("0").Build()
}

// EveryWeek 每周一0点执行
func EveryWeek() string {
	return NewCronExpressionBuilder().SetMinutes("0").SetHours("0").SetDayOfWeek("MON").Build()
}

// EveryMonth 每月1号0点执行
func EveryMonth() string {
	return NewCronExpressionBuilder().SetMinutes("0").SetHours("0").SetDay("1").Build()
}

// Custom 自定义Cron表达式
func Custom(seconds, minutes, hours, day, month, dayOfWeek string) string {
	return generateCronExpression(seconds, minutes, hours, day, month, dayOfWeek)
}

// GetCommonCronExpressions 获取常用Cron表达式
func GetCommonCronExpressions() []map[string]string {
	return []map[string]string{
		{"value": "0 * * * * ?", "label": "每分钟"},
		{"value": "0 0 * * * ?", "label": "每小时"},
		{"value": "0 0 0 * * ?", "label": "每天0点"},
		{"value": "0 0 0 * * MON ?", "label": "每周一0点"},
		{"value": "0 0 0 1 * ?", "label": "每月1号0点"},
		{"value": "0 */5 * * * ?", "label": "每5分钟"},
		{"value": "0 */10 * * * ?", "label": "每10分钟"},
		{"value": "0 */30 * * * ?", "label": "每30分钟"},
		{"value": "0 0 */2 * * ?", "label": "每2小时"},
		{"value": "0 0 8,12,18 * * ?", "label": "每天8点、12点、18点"},
	}
}

// ParseNextRunTimes 解析并返回接下来几次的执行时间
func ParseNextRunTimes(expr string, count int) ([]string, error) {
	// 使用6字段格式解析器
	parser := cron.NewParser(cron.Second | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)
	schedule, err := parser.Parse(expr)
	if err != nil {
		return nil, err
	}

	var times []string
	nextTime := time.Now()
	for i := 0; i < count; i++ {
		nextTime = schedule.Next(nextTime)
		times = append(times, nextTime.Format("2006-01-02 15:04:05"))
	}

	return times, nil
}
