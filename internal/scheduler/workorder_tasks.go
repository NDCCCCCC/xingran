package scheduler

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/robfig/cron/v3"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
	"gorm.io/gorm"
)

// ==================== 周期性工单任务注册函数 ====================

// RegisterWorkOrderTasks 注册运维工单相关定时任务
func RegisterWorkOrderTasks(scheduler *Scheduler) {
	// 周期性工单生成任务 - 根据模板自动创建工单
	scheduler.RegisterTask("periodic_workorder_create", func(ctx context.Context, params map[string]interface{}) error {
		return executePeriodicWorkOrderCreateTask(ctx, params)
	})
}

// executePeriodicWorkOrderCreateTask 执行周期性工单创建任务
// 根据模板ID创建工单，并自动分配给值班人员
func executePeriodicWorkOrderCreateTask(ctx context.Context, params map[string]interface{}) error {
	if GlobalDB == nil {
		return fmt.Errorf("数据库未初始化")
	}

	// 从参数中获取模板ID（统一使用 "param" key）
	templateID, ok := params["param"].(string)
	if !ok || templateID == "" {
		return fmt.Errorf("模板ID参数无效")
	}

	db := GlobalDB.GetDB()

	// 查询周期性工单模板
	var template models.PeriodicWorkOrderTemplate
	if err := db.Where("id = ? AND is_enabled = ?", templateID, true).First(&template).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			applogger.Infof("周期性工单模板不存在或已禁用: %s", templateID)
			return nil // 不是错误，模板已被禁用
		}
		return fmt.Errorf("查询周期性工单模板失败: %w", err)
	}

	// 在事务外声明变量，以便后续使用
	var workOrderNo string
	var title string
	var workOrder *models.WorkOrder

	// 获取 system 用户的 ID
	var systemUser struct {
		ID string
	}
	if err := db.Table("sys_user").Select("id").Where("username = ?", "system").First(&systemUser).Error; err != nil {
		return fmt.Errorf("查询系统用户失败，请确保存在 username='system' 的用户: %w", err)
	}

	// 开始事务
	if err := db.Transaction(func(tx *gorm.DB) error {
		// 生成工单编号
		workOrderNo = generateWorkOrderNo()

		// 替换标题中的变量
		title = replaceVariables(template.WorkOrderTitle, time.Now())

		// 创建工单
		workOrder = &models.WorkOrder{
			Title:          title,
			WorkOrderNo:    workOrderNo,
			CategoryID:     template.CategoryID,
			Type:           template.Type,
			Priority:       template.Priority,
			Status:         models.WorkOrderStatusPending, // 待处理
			Description:    template.Description,
			SubmitterID:    systemUser.ID, // 系统创建用户ID
			IsAutoAssigned: true,          // 标记为自动分配
			AssignStrategy: "assign_one",  // 默认分配策略
		}

		// 根据分配类型获取处理人
		assigneeID, err := assignWorkOrderHandler(tx, &template, workOrder)
		if err != nil {
			applogger.Infof("分配处理人失败: %v，将创建工单但不分配", err)
		}
		workOrder.AssigneeID = assigneeID

		// 保存工单
		if err := tx.Create(workOrder).Error; err != nil {
			return fmt.Errorf("创建工单失败: %w", err)
		}

		// 记录执行日志
		execLog := &models.PeriodicWorkOrderLog{
			TemplateID:  template.ID,
			WorkOrderID: workOrder.ID,
			ExecutedAt:  time.Now(),
			JobID:       template.JobID,
			Status:      "success",
			Result:      fmt.Sprintf("成功创建工单: %s (%s)", workOrderNo, title),
		}

		if err := tx.Create(execLog).Error; err != nil {
			return fmt.Errorf("记录执行日志失败: %w", err)
		}

		// 更新模板的已生成工单数量
		if err := tx.Model(&template).UpdateColumn("total_generated", gorm.Expr("total_generated + 1")).Error; err != nil {
			applogger.Infof("更新模板生成数量失败: %v", err)
		}

		return nil
	}); err != nil {
		return err
	}

	// 如果启用了通知且分配了处理人，发送通知
	if template.NotifyAssignee && workOrder.AssigneeID != nil {
		sendWorkOrderNotification(workOrder.ID, *workOrder.AssigneeID, title)
	}

	applogger.Infof("周期性工单创建成功: %s - %s", workOrderNo, title)
	return nil
}

// generateWorkOrderNo 生成工单编号
// 格式: WO + 年月日 + 12位 UUID hex 后缀 (高碰撞概率下 < 10^-12)
//
// F-11: 原实现用 Count+1 生成序号,在并发场景下两个 goroutine 同时
// Count(...) 得到相同值,生成完全相同的工单号,触发 UNIQUE 约束错误
// 或更糟的数据冲突。改用 UUID v4 后缀彻底消除该竞态。
//
// 工单号示例: WO20260612a1b2c3d4e5f6 (人类仍可识别日期前缀)
func generateWorkOrderNo() string {
	now := time.Now()
	dateStr := now.Format("20060102")
	suffix := strings.ReplaceAll(uuid.New().String(), "-", "")[:12]
	return fmt.Sprintf("WO%s%s", dateStr, suffix)
}

// replaceVariables 替换标题中的变量
// 支持的变量: {date}, {year}, {month}, {day}, {weekday}, {hour}, {minute}
func replaceVariables(title string, t time.Time) string {
	// 定义变量映射
	variables := map[string]string{
		"{date}":    t.Format("2006-01-02"),
		"{year}":    t.Format("2006"),
		"{month}":   t.Format("01"),
		"{day}":     t.Format("02"),
		"{weekday}": getWeekdayName(t.Weekday()),
		"{hour}":    t.Format("15"),
		"{minute}":  t.Format("04"),
	}

	// 替换变量
	result := title
	for key, value := range variables {
		result = strings.ReplaceAll(result, key, value)
	}

	return result
}

// getWeekdayName 获取星期名称
func getWeekdayName(wd time.Weekday) string {
	weekdays := map[time.Weekday]string{
		time.Sunday:    "周日",
		time.Monday:    "周一",
		time.Tuesday:   "周二",
		time.Wednesday: "周三",
		time.Thursday:  "周四",
		time.Friday:    "周五",
		time.Saturday:  "周六",
	}
	return weekdays[wd]
}

// getTodayDutyPerson 获取当天值班人员
// 返回值班人员ID，如果有多个则返回第一个
func getTodayDutyPerson(db *gorm.DB) (*string, error) {
	today := time.Now().Format("2006-01-02")

	type DutyPerson struct {
		UserID string
	}

	var dutyPerson DutyPerson
	err := db.Table("sys_duty_schedule").
		Select("user_id").
		Where("schedule_date = ? AND status = ?", today, 0). // DutyStatusNormal = 0
		First(&dutyPerson).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("今日暂无值班人员")
		}
		return nil, fmt.Errorf("查询值班人员失败: %w", err)
	}

	return &dutyPerson.UserID, nil
}

// assignWorkOrderHandler 根据分配类型分配工单处理人
func assignWorkOrderHandler(db *gorm.DB, template *models.PeriodicWorkOrderTemplate, workOrder *models.WorkOrder) (*string, error) {
	switch template.AssignType {
	case models.PeriodicAssignTypeManual:
		// 手动指定处理人
		if template.AssignTargetID != nil && *template.AssignTargetID != "" {
			workOrder.IsAutoAssigned = false
			return template.AssignTargetID, nil
		}
		return nil, nil

	case models.PeriodicAssignTypeDutyPool:
		// 分配给当天值班人员
		assigneeID, err := getTodayDutyPerson(db)
		if err != nil {
			return nil, err
		}
		if assigneeID != nil {
			workOrder.DutyType = "duty_pool"
			workOrder.AssignStrategy = "assign_one"
		}
		return assigneeID, nil

	case models.PeriodicAssignTypeRotation:
		// 按轮询分配（暂未实现）
		applogger.Infof("轮询分配暂未实现")
		return nil, nil

	default:
		return nil, nil
	}
}

// sendWorkOrderNotification 发送工单通知
func sendWorkOrderNotification(workOrderID, assigneeID, title string) {
	if GlobalNoticeHub == nil {
		applogger.Infof("通知中心未初始化，跳过工单通知")
		return
	}

	message := map[string]interface{}{
		"type":        "new_workorder",
		"workOrderId": workOrderID,
		"title":       "新工单通知",
		"content":     fmt.Sprintf("您有一个新的周期性工单待处理：%s", title),
		"priority":    1,
		"timestamp":   time.Now().Unix(),
	}

	GlobalNoticeHub.BroadcastToUsers([]string{assigneeID}, message)
	applogger.Infof("工单通知已发送给用户: %s", assigneeID)
}

// SyncPeriodicWorkOrderJobs 同步周期性工单任务到调度器
// 在调度器启动后调用，将所有启用的周期性工单模板注册为定时任务
func SyncPeriodicWorkOrderJobs(scheduler *Scheduler) error {
	if GlobalDB == nil {
		return fmt.Errorf("数据库未初始化")
	}

	db := GlobalDB.GetDB()

	// 查询所有已启用的周期性工单模板
	var templates []models.PeriodicWorkOrderTemplate
	if err := db.Where("is_enabled = ?", true).Find(&templates).Error; err != nil {
		return fmt.Errorf("查询周期性工单模板失败: %w", err)
	}

	applogger.Infof("开始同步周期性工单任务，共 %d 个模板", len(templates))

	successCount := 0
	failCount := 0

	for _, template := range templates {
		if err := syncWorkOrderJob(db, scheduler, &template); err != nil {
			applogger.Infof("同步工单任务失败 [%s]: %v", template.TemplateName, err)
			failCount++
		} else {
			successCount++
		}
	}

	applogger.Infof("周期性工单任务同步完成: 成功=%d, 失败=%d", successCount, failCount)
	return nil
}

// syncWorkOrderJob 同步单个工单任务
func syncWorkOrderJob(db *gorm.DB, scheduler *Scheduler, template *models.PeriodicWorkOrderTemplate) error {
	// 计算下次执行时间
	nextRunTime, err := calculateNextRunTimeCron(template.CronExpression)
	if err != nil {
		return fmt.Errorf("解析Cron表达式失败: %w", err)
	}

	jobName := getPeriodicWorkOrderJobName(template.ID)

	// 查找已存在的任务
	var job models.Job
	err = db.Where("job_name = ?", jobName).First(&job).Error

	if err == gorm.ErrRecordNotFound {
		return createWorkOrderJob(db, scheduler, template, jobName, nextRunTime)
	}

	if err != nil {
		return fmt.Errorf("查询定时任务失败: %w", err)
	}

	return updateExistingWorkOrderJob(db, template, &job, nextRunTime)
}

// createWorkOrderJob 创建新的工单任务
func createWorkOrderJob(db *gorm.DB, scheduler *Scheduler, template *models.PeriodicWorkOrderTemplate, jobName string, nextRunTime time.Time) error {
	remark := fmt.Sprintf("周期性工单模板: %s", template.TemplateName)
	newJob := &models.Job{
		JobName:        jobName,
		JobGroup:       "PERIODIC_WORKORDER",
		InvokeTarget:   "periodic_workorder_create:" + template.ID,
		CronExpression: template.CronExpression,
		MisfirePolicy:  models.MisfirePolicyExecuteOnce,
		Status:         0,
		NextRunTime:    &nextRunTime,
		Remark:         &remark,
	}

	if err := scheduler.AddJob(newJob); err != nil {
		return fmt.Errorf("创建定时任务失败: %w", err)
	}

	if err := db.Model(&models.PeriodicWorkOrderTemplate{}).Where("id = ?", template.ID).Update("job_id", newJob.ID).Error; err != nil {
		applogger.Infof("更新模板JobID失败 [%s]: %v", template.TemplateName, err)
	}

	applogger.Infof("周期性工单任务已创建: %s (下次执行: %v)", template.TemplateName, nextRunTime)
	return nil
}

// updateExistingWorkOrderJob 更新已存在的工单任务
func updateExistingWorkOrderJob(db *gorm.DB, template *models.PeriodicWorkOrderTemplate, job *models.Job, nextRunTime time.Time) error {
	// 更新模板的 JobID
	if template.JobID == nil || *template.JobID != job.ID {
		if err := db.Model(&models.PeriodicWorkOrderTemplate{}).Where("id = ?", template.ID).Update("job_id", job.ID).Error; err != nil {
			applogger.Infof("更新模板JobID失败 [%s]: %v", template.TemplateName, err)
		}
	}

	// 更新下次执行时间
	if err := db.Model(job).Update("next_run_time", nextRunTime).Error; err != nil {
		return fmt.Errorf("更新下次执行时间失败: %w", err)
	}

	applogger.Infof("周期性工单任务已同步: %s (下次执行: %v)", template.TemplateName, nextRunTime)
	return nil
}

// getPeriodicWorkOrderJobName 获取周期性工单定时任务的名称
func getPeriodicWorkOrderJobName(templateID string) string {
	return fmt.Sprintf("periodic_workorder_%s", templateID)
}

// calculateNextRunTimeCron 计算下次执行时间
func calculateNextRunTimeCron(cronExpression string) (time.Time, error) {
	// 使用 cron 实例计算下次执行时间（带时区支持）
	// time.Local 已在 main.go 中设置为 Asia/Shanghai
	c := cron.New(cron.WithSeconds(), cron.WithLocation(time.Local))
	entryID, err := c.AddFunc(cronExpression, func() {})
	if err != nil {
		return time.Time{}, fmt.Errorf("解析Cron表达式失败: %w", err)
	}
	// 必须启动 cron 才能获取正确的 Next 时间
	c.Start()
	defer c.Stop()
	entry := c.Entry(entryID)
	return entry.Next, nil
}

// DisablePeriodicWorkOrderJob 禁用周期性工单任务
func DisablePeriodicWorkOrderJob(scheduler *Scheduler, templateID string) error {
	if GlobalDB == nil {
		return fmt.Errorf("数据库未初始化")
	}

	db := GlobalDB.GetDB()

	// 查询模板
	var template models.PeriodicWorkOrderTemplate
	if err := db.Where("id = ?", templateID).First(&template).Error; err != nil {
		return fmt.Errorf("查询周期性工单模板失败: %w", err)
	}

	// 如果有关联的定时任务，停止任务
	if template.JobID != nil && *template.JobID != "" {
		if err := scheduler.StopJob(*template.JobID); err != nil {
			applogger.Infof("停止定时任务失败: %v", err)
			// 不中断流程，继续执行
		} else {
			applogger.Infof("定时任务已停止: %s", *template.JobID)
		}

		// 删除定时任务
		if err := scheduler.RemoveJob(*template.JobID); err != nil {
			applogger.Infof("删除定时任务失败: %v", err)
		}
	}

	// 更新模板的 JobID 为空
	if err := db.Model(&template).Update("job_id", nil).Error; err != nil {
		return fmt.Errorf("更新模板JobID失败: %w", err)
	}

	return nil
}

// EnablePeriodicWorkOrderJob 启用周期性工单任务
func EnablePeriodicWorkOrderJob(scheduler *Scheduler, templateID string) error {
	if GlobalDB == nil {
		return fmt.Errorf("数据库未初始化")
	}

	db := GlobalDB.GetDB()

	// 查询模板
	var template models.PeriodicWorkOrderTemplate
	if err := db.Where("id = ?", templateID).First(&template).Error; err != nil {
		return fmt.Errorf("查询周期性工单模板失败: %w", err)
	}

	jobName := getPeriodicWorkOrderJobName(template.ID)

	// 计算下次执行时间
	nextRunTime, err := calculateNextRunTimeCron(template.CronExpression)
	if err != nil {
		return fmt.Errorf("计算下次执行时间失败: %w", err)
	}

	// 创建新的定时任务
	remark := fmt.Sprintf("周期性工单模板: %s", template.TemplateName)
	newJob := &models.Job{
		JobName:        jobName,
		JobGroup:       "PERIODIC_WORKORDER",
		InvokeTarget:   "periodic_workorder_create:" + template.ID,
		CronExpression: template.CronExpression,
		MisfirePolicy:  models.MisfirePolicyExecuteOnce,
		Status:         0, // 启用
		NextRunTime:    &nextRunTime,
		Remark:         &remark,
	}

	if err := scheduler.AddJob(newJob); err != nil {
		return fmt.Errorf("创建定时任务失败: %w", err)
	}

	// 更新模板的 JobID
	if err := db.Model(&template).Update("job_id", newJob.ID).Error; err != nil {
		return fmt.Errorf("更新模板JobID失败: %w", err)
	}

	applogger.Infof("周期性工单任务已启用: %s (下次执行: %v)", template.TemplateName, nextRunTime)
	return nil
}
