package v1

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/core"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
	"github.com/ruoyi-next/ruoyi-go-backend/pkg/logger"
	"github.com/ruoyi-next/ruoyi-go-backend/pkg/response"
	"gorm.io/gorm"
)

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

		// 记录操作日志
		recordOperLog(c, core, "定时任务", OperTypeUpdate)

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
