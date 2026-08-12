package services

import (
	"fmt"
	"time"
)

// ==================== Cron 表达式工具 ====================

// CalculateCronExpression 计算指定时间的Cron表达式
// 用于一次性定时任务：在指定时间执行一次
func CalculateCronExpression(t time.Time) string {
	// Cron表达式格式：秒 分 时 日 月 周
	// 例如：2024-01-15 14:30:00 -> "0 30 14 15 1 ?"
	return fmt.Sprintf("0 %d %d %d %d ?",
		t.Minute(),
		t.Hour(),
		t.Day(),
		int(t.Month()),
	)
}

// GenerateCronExpression 生成周期性任务的 Cron 表达式
// 根据 RecurrenceConfig 生成对应的 Cron 表达式
func GenerateCronExpression(recurrenceType string, executeTime string, weekDay, monthDay *int, customExpr *string) (string, error) {
	// 如果提供了自定义表达式，直接返回
	if recurrenceType == "custom" && customExpr != nil && *customExpr != "" {
		return *customExpr, nil
	}

	// 解析执行时间
	var hour, minute int
	if executeTime != "" {
		_, err := fmt.Sscanf(executeTime, "%d:%d", &hour, &minute)
		if err != nil || hour < 0 || hour > 23 || minute < 0 || minute > 59 {
			return "", fmt.Errorf("执行时间格式错误，应为 HH:mm 格式")
		}
	} else {
		// 默认上午9点
		hour = 9
		minute = 0
	}

	// 根据周期类型生成 Cron 表达式
	switch recurrenceType {
	case "daily":
		// 每天执行：0 0 9 * * ? （每天上午9点）
		return fmt.Sprintf("0 %d %d * * ?", minute, hour), nil

	case "weekly":
		// 每周执行：0 0 9 ? * 2 （周几上午9点）
		if weekDay == nil || *weekDay < 1 || *weekDay > 7 {
			return "", fmt.Errorf("周几参数错误，应为 1-7（1=周日，2=周一，...，7=周六）")
		}
		return fmt.Sprintf("0 %d %d ? * %d", minute, hour, *weekDay), nil

	case "monthly":
		// 每月执行：0 0 9 1 * ? （每月几号上午9点）
		if monthDay == nil || *monthDay < 1 || *monthDay > 31 {
			return "", fmt.Errorf("月份日期参数错误，应为 1-31")
		}
		return fmt.Sprintf("0 %d %d %d * ?", minute, hour, *monthDay), nil

	case "custom":
		// 自定义表达式但未提供
		return "", fmt.Errorf("自定义周期需要提供 Cron 表达式")

	default:
		return "", fmt.Errorf("不支持的周期类型: %s", recurrenceType)
	}
}

// GetNoticeJobName 获取通知定时任务的名称
func GetNoticeJobName(noticeID string) string {
	return fmt.Sprintf("notice_publish_%s", noticeID)
}

// CommonCronExpression 常用Cron表达式
type CommonCronExpression struct {
	Name        string `json:"name"`        // 名称
	Expression  string `json:"expression"`  // Cron表达式
	Description string `json:"description"` // 描述
}

// GetCommonCronExpressions 获取常用Cron表达式列表
func GetCommonCronExpressions() []CommonCronExpression {
	return []CommonCronExpression{
		{
			Name:        "每天凌晨0点",
			Expression:  "0 0 0 * * ?",
			Description: "每天00:00:00执行",
		},
		{
			Name:        "每天早上9点",
			Expression:  "0 0 9 * * ?",
			Description: "每天09:00:00执行",
		},
		{
			Name:        "每天中午12点",
			Expression:  "0 0 12 * * ?",
			Description: "每天12:00:00执行",
		},
		{
			Name:        "每天下午6点",
			Expression:  "0 0 18 * * ?",
			Description: "每天18:00:00执行",
		},
		{
			Name:        "每小时执行",
			Expression:  "0 0 * * * ?",
			Description: "每小时的0分0秒执行",
		},
		{
			Name:        "每30分钟执行",
			Expression:  "0 0,30 * * * ?",
			Description: "每小时的0分和30分执行",
		},
		{
			Name:        "每周一早上9点",
			Expression:  "0 0 9 ? * MON",
			Description: "每周一09:00:00执行",
		},
		{
			Name:        "每周一早上9点（备用）",
			Expression:  "0 0 9 ? * 2",
			Description: "每周一09:00:00执行（2表示周一）",
		},
		{
			Name:        "每月1号凌晨0点",
			Expression:  "0 0 0 1 * ?",
			Description: "每月1号00:00:00执行",
		},
		{
			Name:        "每月1号上午10点",
			Expression:  "0 0 10 1 * ?",
			Description: "每月1号10:00:00执行",
		},
		{
			Name:        "工作日早上9点",
			Expression:  "0 0 9 ? * MON-FRI",
			Description: "周一到周五09:00:00执行",
		},
		{
			Name:        "工作日早上9点（备用）",
			Expression:  "0 0 9 ? * 2-6",
			Description: "周一到周五09:00:00执行（2-6表示周一到周五）",
		},
	}
}
