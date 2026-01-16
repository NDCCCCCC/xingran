package v1

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/core"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
	"github.com/ruoyi-next/ruoyi-go-backend/pkg/response"
)

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
