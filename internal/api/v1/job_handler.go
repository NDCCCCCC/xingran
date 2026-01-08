package v1

import (
	"fmt"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/robfig/cron/v3"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/core"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
	"github.com/ruoyi-next/ruoyi-go-backend/pkg/logger"
	"github.com/ruoyi-next/ruoyi-go-backend/pkg/response"
	"gorm.io/gorm"
)

// JobListRequest 定时任务列表请求
type JobListRequest struct {
	JobName  string `json:"jobName,omitempty"`
	JobGroup string `json:"jobGroup,omitempty"`
	Status   *int   `json:"status,omitempty"`
	Current  int    `json:"current"`
	PageSize int    `json:"pageSize"`
}

// JobCreateRequest 创建定时任务请求
type JobCreateRequest struct {
	JobName        string `json:"jobName" binding:"required,max=64"`
	JobGroup       string `json:"jobGroup" binding:"required,max=64"`
	InvokeTarget   string `json:"invokeTarget" binding:"required,max=500"`
	CronExpression string `json:"cronExpression" binding:"required,max=255"`
	MisfirePolicy  int    `json:"misfirePolicy" binding:"oneof=0 1 2 3"`
	Concurrent     int    `json:"concurrent" binding:"oneof=0 1"`
	Remark         string `json:"remark"`
}

// JobUpdateRequest 更新定时任务请求
type JobUpdateRequest struct {
	ID             string `json:"id" binding:"required"`
	JobName        string `json:"jobName" binding:"required,max=64"`
	JobGroup       string `json:"jobGroup" binding:"required,max=64"`
	InvokeTarget   string `json:"invokeTarget" binding:"required,max=500"`
	CronExpression string `json:"cronExpression" binding:"required,max=255"`
	MisfirePolicy  int    `json:"misfirePolicy" binding:"oneof=0 1 2 3"`
	Concurrent     int    `json:"concurrent" binding:"oneof=0 1"`
	Remark         string `json:"remark"`
}

// JobStatusRequest 任务状态请求
type JobStatusRequest struct {
	ID     string `json:"id"`
	Status int    `json:"status" binding:"oneof=0 1"` // 移除required，因为0会被认为是零值
}

// JobExecuteRequest 执行任务请求
type JobExecuteRequest struct {
	ID string `json:"id" binding:"required"`
}

// JobLogListRequest 任务日志列表请求
type JobLogListRequest struct {
	JobName   string `json:"jobName,omitempty"`
	JobGroup  string `json:"jobGroup,omitempty"`
	Status    *int   `json:"status,omitempty"`
	StartTime *string `json:"startTime,omitempty"`
	EndTime   *string `json:"endTime,omitempty"`
	Current   int    `json:"current"`
	PageSize  int    `json:"pageSize"`
}

// listJobs 查询定时任务列表
// @Summary 查询定时任务列表
// @Description 分页查询定时任务列表
// @Tags 定时任务管理
// @Accept json
// @Produce json
// @Param request body JobListRequest false "查询参数"
// @Success 200 {object} response.Response{data=response.PageData}
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /monitor/jobs/list [post]
func listJobs(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req JobListRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			req.Current = 1
			req.PageSize = 10
		}

		// 设置默认值
		if req.Current <= 0 {
			req.Current = 1
		}
		if req.PageSize <= 0 {
			req.PageSize = 10
		}

		// 构建查询条件
		db := core.DB.GetDB().Model(&models.Job{})

		// 调试日志
		logger.Debugf("[JobList] 请求参数: jobName=%s, jobGroup=%s, status=%v, current=%d, pageSize=%d",
			req.JobName, req.JobGroup, req.Status, req.Current, req.PageSize)

		if req.JobName != "" {
			db = db.Where("job_name LIKE ?", "%"+req.JobName+"%")
		}
		if req.JobGroup != "" {
			db = db.Where("job_group = ?", req.JobGroup)
		}
		if req.Status != nil {
			db = db.Where("status = ?", *req.Status)
			logger.Debugf("[JobList] 添加状态筛选: status = %d", *req.Status)
		} else {
			logger.Debugf("[JobList] 无状态筛选，将返回所有状态的任务")
		}

		// 统计总数
		var total int64
		if err := db.Count(&total).Error; err != nil {
			response.Error(c, response.ErrServerError, "查询定时任务总数失败")
			return
		}

		// 查询分页数据
		var jobs []models.Job
		offset := (req.Current - 1) * req.PageSize
		if err := db.Offset(offset).Limit(req.PageSize).Order("created_at DESC").Find(&jobs).Error; err != nil {
			response.Error(c, response.ErrServerError, "查询定时任务列表失败")
			return
		}

		response.Page(c, jobs, total, req.Current, req.PageSize)
	}
}

// createJob 创建定时任务
// @Summary 创建定时任务
// @Description 创建新的定时任务
// @Tags 定时任务管理
// @Accept json
// @Produce json
// @Param request body JobCreateRequest true "任务信息"
// @Success 200 {object} response.Response{data=models.Job}
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /monitor/jobs [post]
func createJob(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req JobCreateRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.Error(c, response.ErrBadRequest, "请求参数错误")
			return
		}

		// 检查任务名称是否已存在
		var count int64
		if err := core.DB.GetDB().Model(&models.Job{}).
			Where("job_name = ? AND job_group = ?", req.JobName, req.JobGroup).
			Count(&count).Error; err != nil {
			response.Error(c, response.ErrServerError, "检查任务名称失败")
			return
		}
		if count > 0 {
			response.Error(c, response.ErrRecordExists, "任务名称在该组中已存在")
			return
		}

		// 验证Cron表达式
		if _, err := cron.NewParser(cron.Second | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor).Parse(req.CronExpression); err != nil {
			response.Error(c, response.ErrBadRequest, "无效的Cron表达式: "+err.Error())
			return
		}

		// 创建任务
		job := models.Job{
			JobName:        req.JobName,
			JobGroup:       req.JobGroup,
			InvokeTarget:   req.InvokeTarget,
			CronExpression: req.CronExpression,
			MisfirePolicy:  models.MisfirePolicy(req.MisfirePolicy),
			Concurrent:     req.Concurrent == 1,
			Status:         0, // 默认正常状态
		}

		if req.Remark != "" {
			job.Remark = &req.Remark
		}

		if err := core.DB.GetDB().Create(&job).Error; err != nil {
			response.Error(c, response.ErrServerError, "创建定时任务失败")
			return
		}

		// 如果调度器已启动，添加任务到调度器
		if core.Scheduler != nil {
			if err := core.Scheduler.AddJob(&job); err != nil {
				response.Error(c, response.ErrServerError, "添加任务到调度器失败: "+err.Error())
				return
			}
		}

		response.Success(c, job)
	}
}

// getJob 获取定时任务详情
// @Summary 获取定时任务详情
// @Description 根据任务ID获取定时任务详细信息
// @Tags 定时任务管理
// @Accept json
// @Produce json
// @Param id path string true "任务ID"
// @Success 200 {object} response.Response{data=models.Job}
// @Failure 400 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /monitor/jobs/{id} [get]
func getJob(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if id == "" {
			response.Error(c, response.ErrBadRequest, "任务ID不能为空")
			return
		}

		var job models.Job
		if err := core.DB.GetDB().First(&job, "id = ?", id).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				response.Error(c, response.ErrRecordNotFound, "定时任务不存在")
			} else {
				response.Error(c, response.ErrServerError, "查询定时任务失败")
			}
			return
		}

		response.Success(c, job)
	}
}

// updateJob 更新定时任务
// @Summary 更新定时任务
// @Description 更新定时任务信息
// @Tags 定时任务管理
// @Accept json
// @Produce json
// @Param id path string true "任务ID"
// @Param request body JobUpdateRequest true "任务信息"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /monitor/jobs/{id} [put]
func updateJob(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if id == "" {
			response.Error(c, response.ErrBadRequest, "任务ID不能为空")
			return
		}

		var req JobUpdateRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.Error(c, response.ErrBadRequest, "请求参数错误")
			return
		}

		// 检查任务是否存在
		var job models.Job
		if err := core.DB.GetDB().First(&job, "id = ?", id).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				response.Error(c, response.ErrRecordNotFound, "定时任务不存在")
			} else {
				response.Error(c, response.ErrServerError, "查询定时任务失败")
			}
			return
		}

		// 检查任务名称是否与其他任务重复
		var count int64
		if err := core.DB.GetDB().Model(&models.Job{}).
			Where("job_name = ? AND job_group = ? AND id != ?", req.JobName, req.JobGroup, id).
			Count(&count).Error; err != nil {
			response.Error(c, response.ErrServerError, "检查任务名称失败")
			return
		}
		if count > 0 {
			response.Error(c, response.ErrRecordExists, "任务名称在该组中已存在")
			return
		}

		// 验证Cron表达式
		if _, err := cron.NewParser(cron.Second | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor).Parse(req.CronExpression); err != nil {
			response.Error(c, response.ErrBadRequest, "无效的Cron表达式: "+err.Error())
			return
		}

		// 更新任务信息
		job.JobName = req.JobName
		job.JobGroup = req.JobGroup
		job.InvokeTarget = req.InvokeTarget
		job.CronExpression = req.CronExpression
		job.MisfirePolicy = models.MisfirePolicy(req.MisfirePolicy)
		job.Concurrent = req.Concurrent == 1

		if req.Remark != "" {
			job.Remark = &req.Remark
		} else {
			job.Remark = nil
		}

		if err := core.DB.GetDB().Save(&job).Error; err != nil {
			response.Error(c, response.ErrServerError, "更新定时任务失败")
			return
		}

		// 更新调度器中的任务
		if core.Scheduler != nil {
			if job.Status == 0 { // 任务处于运行状态
				if err := core.Scheduler.UpdateJob(&job); err != nil {
					response.Error(c, response.ErrServerError, "更新调度器中的任务失败: "+err.Error())
					return
				}
			}
		}

		response.Success(c, gin.H{"message": "更新成功"})
	}
}

// deleteJob 删除定时任务
// @Summary 删除定时任务
// @Description 删除指定定时任务
// @Tags 定时任务管理
// @Accept json
// @Produce json
// @Param id path string true "任务ID"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /monitor/jobs/{id} [delete]
func deleteJob(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if id == "" {
			response.Error(c, response.ErrBadRequest, "任务ID不能为空")
			return
		}

		// 检查任务是否存在
		var job models.Job
		if err := core.DB.GetDB().First(&job, "id = ?", id).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				response.Error(c, response.ErrRecordNotFound, "定时任务不存在")
			} else {
				response.Error(c, response.ErrServerError, "查询定时任务失败")
			}
			return
		}

		// 开始事务
		tx := core.DB.GetDB().Begin()
		defer func() {
			if r := recover(); r != nil {
				tx.Rollback()
			}
		}()

		// 从调度器中移除任务
		if core.Scheduler != nil {
			if err := core.Scheduler.RemoveJob(id); err != nil {
				// 忽略错误，继续删除数据库记录
			}
		}

		// 删除任务
		if err := tx.Delete(&job).Error; err != nil {
			tx.Rollback()
			response.Error(c, response.ErrServerError, "删除定时任务失败")
			return
		}

		// 删除任务执行日志
		if err := tx.Where("job_name = ? AND job_group = ?", job.JobName, job.JobGroup).
			Delete(&models.JobLog{}).Error; err != nil {
			tx.Rollback()
			response.Error(c, response.ErrServerError, "删除任务日志失败")
			return
		}

		// 提交事务
		if err := tx.Commit().Error; err != nil {
			response.Error(c, response.ErrServerError, "删除定时任务失败")
			return
		}

		response.Success(c, gin.H{"message": "删除成功"})
	}
}

// updateJobStatus 更改任务状态
// @Summary 更改任务状态
// @Description 启用或禁用定时任务
// @Tags 定时任务管理
// @Accept json
// @Produce json
// @Param id path string true "任务ID"
// @Param request body JobStatusRequest true "状态信息"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /monitor/jobs/{id}/status [put]
func updateJobStatus(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		logger.Debugf("[JOB] updateJobStatus 开始, ID=%s", id)

		if id == "" {
			logger.Warnf("[JOB] ID为空")
			response.Error(c, response.ErrBadRequest, "任务ID不能为空")
			return
		}

		var req JobStatusRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			logger.Debugf("[JOB] 绑定错误: ID=%s, error=%v", id, err)
			response.Error(c, response.ErrBadRequest, "请求参数错误: "+err.Error())
			return
		}

		logger.Debugf("[JOB] 绑定成功: URL ID=%s, Body ID=%s, Status=%d", id, req.ID, req.Status)

		// 检查任务是否存在
		var job models.Job
		if err := core.DB.GetDB().First(&job, "id = ?", id).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				response.Error(c, response.ErrRecordNotFound, "定时任务不存在")
			} else {
				response.Error(c, response.ErrServerError, "查询定时任务失败")
			}
			return
		}

		// 更新状态
		job.Status = models.JobStatus(req.Status)

		if err := core.DB.GetDB().Save(&job).Error; err != nil {
			response.Error(c, response.ErrServerError, "更新任务状态失败")
			return
		}

		// 根据状态启动或停止任务
		if core.Scheduler != nil {
			if req.Status == 0 { // 启用
				if err := core.Scheduler.StartJob(id); err != nil {
					response.Error(c, response.ErrServerError, "启动任务失败: "+err.Error())
					return
				}
			} else { // 禁用
				if err := core.Scheduler.StopJob(id); err != nil {
					response.Error(c, response.ErrServerError, "停止任务失败: "+err.Error())
					return
				}
			}
		}

		statusText := "启用"
		if req.Status == 1 {
			statusText = "暂停"
		}

		response.Success(c, gin.H{
			"message": fmt.Sprintf("%s成功", statusText),
		})
	}
}

// executeJob 立即执行任务
// @Summary 立即执行任务
// @Description 立即执行指定的定时任务
// @Tags 定时任务管理
// @Accept json
// @Produce json
// @Param id path string true "任务ID"
// @Param request body JobExecuteRequest false "执行参数"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /monitor/jobs/{id}/execute [post]
func executeJob(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if id == "" {
			response.Error(c, response.ErrBadRequest, "任务ID不能为空")
			return
		}

		// 检查任务是否存在
		var job models.Job
		if err := core.DB.GetDB().First(&job, "id = ?", id).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				response.Error(c, response.ErrRecordNotFound, "定时任务不存在")
			} else {
				response.Error(c, response.ErrServerError, "查询定时任务失败")
			}
			return
		}

		// 执行任务
		if core.Scheduler != nil {
			if err := core.Scheduler.ExecuteJob(id); err != nil {
				response.Error(c, response.ErrServerError, "执行任务失败: "+err.Error())
				return
			}
		} else {
			// 如果调度器未启动，记录简单日志
			startTime := time.Now()
			jobLog := models.JobLog{
				JobName:      job.JobName,
				JobGroup:     job.JobGroup,
				InvokeTarget: job.InvokeTarget,
				JobMessage:   "手动执行",
				Status:       0,
				StartTime:    &startTime,
			}

			endTime := time.Now()
			jobLog.EndTime = &endTime
			jobLog.Duration = endTime.Sub(startTime).Milliseconds()
			jobLog.JobMessage = "任务执行成功(调度器未启动)"

			if err := core.DB.GetDB().Create(&jobLog).Error; err != nil {
				response.Error(c, response.ErrServerError, "记录任务日志失败")
				return
			}
		}

		response.Success(c, gin.H{
			"message": "任务执行成功",
		})
	}
}

// getJobLogs 获取任务执行日志
// @Summary 获取任务执行日志
// @Description 分页查询任务执行日志列表
// @Tags 定时任务管理
// @Accept json
// @Produce json
// @Param request body JobLogListRequest false "查询参数"
// @Success 200 {object} response.Response{data=response.PageData}
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /monitor/jobs/logs/list [post]
func getJobLogs(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req JobLogListRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			req.Current = 1
			req.PageSize = 10
		}

		// 设置默认值
		if req.Current <= 0 {
			req.Current = 1
		}
		if req.PageSize <= 0 {
			req.PageSize = 10
		}

		// 构建查询条件
		db := core.DB.GetDB().Model(&models.JobLog{})

		if req.JobName != "" {
			db = db.Where("job_name LIKE ?", "%"+req.JobName+"%")
		}
		if req.JobGroup != "" {
			db = db.Where("job_group = ?", req.JobGroup)
		}
		if req.Status != nil {
			db = db.Where("status = ?", *req.Status)
		}
		if req.StartTime != nil && *req.StartTime != "" {
			if t, err := time.Parse("2006-01-02 15:04:05", *req.StartTime); err == nil {
				db = db.Where("start_time >= ?", t)
			}
		}
		if req.EndTime != nil && *req.EndTime != "" {
			if t, err := time.Parse("2006-01-02 15:04:05", *req.EndTime); err == nil {
				db = db.Where("start_time <= ?", t)
			}
		}

		// 统计总数
		var total int64
		if err := db.Count(&total).Error; err != nil {
			response.Error(c, response.ErrServerError, "查询任务日志总数失败")
			return
		}

		// 查询分页数据
		var logs []models.JobLog
		offset := (req.Current - 1) * req.PageSize
		if err := db.Offset(offset).Limit(req.PageSize).Order("created_at DESC").Find(&logs).Error; err != nil {
			response.Error(c, response.ErrServerError, "查询任务日志失败")
			return
		}

		response.Page(c, logs, total, req.Current, req.PageSize)
	}
}

// getCurrentUser 获取当前用户
func getCurrentUser(c *gin.Context) string {
	if username, exists := c.Get("username"); exists {
		return username.(string)
	}
	return "system"
}

// Cron表达式验证辅助函数
func validateCronExpression(expr string) bool {
	_, err := cron.NewParser(cron.Second | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor).Parse(expr)
	return err == nil
}

// calculateNextRunTime 计算下次执行时间
func calculateNextRunTime(expr string) time.Time {
	schedule, err := cron.NewParser(cron.Second | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor).Parse(expr)
	if err != nil {
		return time.Time{}
	}
	return schedule.Next(time.Now())
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
	schedule, err := cron.NewParser(cron.Second | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor).Parse(expr)
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
