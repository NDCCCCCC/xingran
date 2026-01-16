package v1

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/robfig/cron/v3"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/core"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
	"github.com/ruoyi-next/ruoyi-go-backend/pkg/logger"
	"github.com/ruoyi-next/ruoyi-go-backend/pkg/response"
	"gorm.io/gorm"
)

// ==================== 请求/响应类型 ====================

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
	JobName   string  `json:"jobName,omitempty"`
	JobGroup  string  `json:"jobGroup,omitempty"`
	Status    *int    `json:"status,omitempty"`
	StartTime *string `json:"startTime,omitempty"`
	EndTime   *string `json:"endTime,omitempty"`
	Current   int     `json:"current"`
	PageSize  int     `json:"pageSize"`
}

// ==================== 基本 CRUD 操作 ====================

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

		// 记录操作日志
		recordOperLog(c, core, "定时任务", OperTypeCreate)

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

		// 记录操作日志
		recordOperLog(c, core, "定时任务", OperTypeUpdate)

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

		// 从调度器中移除任务（事务外操作）
		if core.Scheduler != nil {
			if err := core.Scheduler.RemoveJob(id); err != nil {
				// 忽略错误，继续删除数据库记录
			}
		}

		// 使用事务删除数据库记录
		if err := core.DB.GetDB().Transaction(func(tx *gorm.DB) error {
			// 删除任务
			if err := tx.Delete(&job).Error; err != nil {
				return fmt.Errorf("删除定时任务失败: %w", err)
			}

			// 删除任务执行日志
			if err := tx.Where("job_name = ? AND job_group = ?", job.JobName, job.JobGroup).
				Delete(&models.JobLog{}).Error; err != nil {
				return fmt.Errorf("删除任务日志失败: %w", err)
			}

			return nil
		}); err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		// 记录操作日志
		recordOperLog(c, core, "定时任务", OperTypeDelete)

		response.Success(c, gin.H{"message": "删除成功"})
	}
}
