package scheduler

import (
	"context"
	"time"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/services"
	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
	"gorm.io/gorm"
)

// RegisterMACHistoryMatViewTasks 注册 MAC 历史物化视图刷新定时任务
//
// Phase 15 PERF-02 (D-11 锁定):
//   - 任务 ID: mac_history_matview_refresh
//   - Cron 表达式: 0 */5 * * * * (6 字段格式, 每 5 分钟)
//   - 复用 Phase 12 RegisterMACHistoryTasks 模式 (handler 注册 + Job 记录双注册)
func RegisterMACHistoryMatViewTasks(s *Scheduler, db *gorm.DB, matViewSvc services.MACHistoryMatViewService) {
	// 注册任务处理器
	s.RegisterTask("mac_history_matview_refresh", func(ctx context.Context, params map[string]interface{}) error {
		return matViewSvc.RefreshAllMaterializedViews(ctx)
	})

	applogger.Infof("MAC历史物化视图刷新任务处理器已注册")

	// 向数据库添加任务记录
	if db == nil {
		applogger.Warnf("MAC历史物化视图刷新任务数据库注册跳过（数据库未设置）")
		return
	}

	var existingJob models.Job
	err := db.Where("job_name = ?", "MAC历史物化视图刷新").First(&existingJob).Error
	if err == nil {
		applogger.Infof("MAC历史物化视图刷新任务已存在于数据库，跳过创建")
		return
	}
	if err != gorm.ErrRecordNotFound {
		applogger.Warnf("查询MAC历史物化视图刷新任务失败: %v", err)
		return
	}

	// 计算下次执行时间（下一个 5 分钟整点）
	now := time.Now()
	nextMinute := (now.Minute()/5+1)*5
	nextRunTime := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), nextMinute, 0, 0, now.Location())
	remark := "MAC地址历史物化视图定时刷新任务，每5分钟执行一次，REFRESH MATERIALIZED VIEW CONCURRENTLY 避免锁表"

	newJob := &models.Job{
		JobName:        "MAC历史物化视图刷新",
		JobGroup:       "SYSTEM",
		InvokeTarget:   "mac_history_matview_refresh",
		CronExpression: "0 */5 * * * *",
		MisfirePolicy:  models.MisfirePolicyExecuteOnce,
		Status:         0,
		NextRunTime:    &nextRunTime,
		Remark:         &remark,
	}

	if err := s.AddJob(newJob); err != nil {
		applogger.Warnf("创建MAC历史物化视图刷新任务失败: %v", err)
		return
	}

	applogger.Infof("MAC历史物化视图刷新任务已添加到数据库（下次执行: %v）", nextRunTime.Format("2006-01-02 15:04:05"))
}
