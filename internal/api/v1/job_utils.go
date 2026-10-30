package v1

import (
	"strconv"
	"time"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	"gorm.io/gorm"
)

// ==================== 工具函数 ====================

// FormatDuration 格式化时长
func FormatDuration(milliseconds int64) string {
	if milliseconds < 1000 {
		return strconv.FormatInt(milliseconds, 10) + "ms"
	}

	seconds := milliseconds / 1000
	if seconds < 60 {
		return strconv.FormatInt(seconds, 10) + "s"
	}

	minutes := seconds / 60
	if minutes < 60 {
		return strconv.FormatInt(minutes, 10) + "m"
	}

	hours := minutes / 60
	return strconv.FormatInt(hours, 10) + "h"
}

// ==================== 统计功能 ====================

// GetJobStatistics 获取任务统计信息
func GetJobStatistics(db *gorm.DB) (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// 总任务数
	var total int64
	if err := db.Model(&models.Job{}).Count(&total).Error; err != nil {
		return nil, err
	}
	stats["total"] = total

	// 运行中任务数
	var running int64
	if err := db.Model(&models.Job{}).Where("status = ?", 0).Count(&running).Error; err != nil {
		return nil, err
	}
	stats["running"] = running

	// 暂停任务数
	stats["paused"] = total - running

	// 今日执行成功次数
	today := time.Now().Format("2006-01-02")
	var successCount int64
	if err := db.Model(&models.JobLog{}).
		Where("status = ? AND DATE(created_at) = ?", 0, today).
		Count(&successCount).Error; err != nil {
		return nil, err
	}
	stats["todaySuccess"] = successCount

	// 今日执行失败次数
	var failCount int64
	if err := db.Model(&models.JobLog{}).
		Where("status = ? AND DATE(created_at) = ?", 1, today).
		Count(&failCount).Error; err != nil {
		return nil, err
	}
	stats["todayFail"] = failCount

	return stats, nil
}
