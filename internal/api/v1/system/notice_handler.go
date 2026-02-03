package system

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/models/system/requests"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/services"
	systemServices "github.com/ruoyi-next/ruoyi-go-backend/internal/services/system"
	"github.com/ruoyi-next/ruoyi-go-backend/pkg/logger"
	"github.com/ruoyi-next/ruoyi-go-backend/pkg/response"
	"fmt"
)

// NoticeHandler 通知公告处理器
type NoticeHandler struct {
	service          systemServices.NoticeService
	channelService   systemServices.NotificationChannelService
	schedulerService SchedulerService
}

// SchedulerService 调度器服务接口
type SchedulerService interface {
	AddJob(job *models.Job) error
	RemoveJob(jobID string) error
}

// NewNoticeHandler 创建通知公告处理器实例
func NewNoticeHandler(
	service systemServices.NoticeService,
	channelService systemServices.NotificationChannelService,
	schedulerSvc SchedulerService,
) *NoticeHandler {
	return &NoticeHandler{
		service:          service,
		channelService:   channelService,
		schedulerService: schedulerSvc,
	}
}

// Create 创建通知公告
func (h *NoticeHandler) Create(c *gin.Context) {
	var req requests.NoticeCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "请求参数错误: "+err.Error())
		return
	}

	// 获取当前用户信息
	userID, exists := c.Get("user_id")
	if !exists {
		response.Error(c, http.StatusUnauthorized, "未授权")
		return
	}
	username, _ := c.Get("username")
	usernameStr, _ := username.(string)

	notice, err := h.service.Create(c.Request.Context(), &req, userID.(string), usernameStr)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	// 判断执行类型，用于后续处理
	executionType := "once" // 默认一次性
	if req.ExecutionType != nil {
		executionType = *req.ExecutionType
	}

	// 标记是否需要更新发布状态为"定时发布中"
	shouldUpdateToScheduled := false

	// 设置发送渠道
	if len(req.Channels) > 0 {
		channels := make([]models.NotificationChannel, len(req.Channels))
		for i, ch := range req.Channels {
			channels[i] = models.NotificationChannel{
				ChannelType:      ch.ChannelType,
				EmailConfigID:    ch.EmailConfigID,
				APIConfigID:      ch.APIConfigID,
				CustomRecipients: ch.CustomRecipients,
			}
		}
		if err := h.channelService.SetNotificationChannels(c.Request.Context(), notice.ID, channels); err != nil {
			response.Error(c, http.StatusInternalServerError, "设置发送渠道失败: "+err.Error())
			return
		}
	}

	// 处理定时任务
	if h.schedulerService != nil {
		var cronExpr string
		var misfirePolicy models.MisfirePolicy

		logger.Debugf("[NOTICE] executionType=%s, RecurrenceConfig=%v", executionType, req.RecurrenceConfig != nil)

		if executionType == "recurring" && req.RecurrenceConfig != nil {
			// 周期性执行，直接使用 Cron 表达式
			if req.RecurrenceConfig.CronExpression == nil || *req.RecurrenceConfig.CronExpression == "" {
				response.Error(c, http.StatusBadRequest, "Cron表达式不能为空")
				return
			}
			cronExpr = *req.RecurrenceConfig.CronExpression
			misfirePolicy = models.MisfirePolicyDefault // 周期性任务使用默认策略，不自动删除
			shouldUpdateToScheduled = true              // 标记需要更新状态
		} else if req.PublishTime != nil && req.PublishTime.After(time.Now()) {
			// 一次性定时任务
			cronExpr = services.CalculateCronExpression(*req.PublishTime)
			misfirePolicy = models.MisfirePolicyExecuteOnce // 执行后删除
			shouldUpdateToScheduled = true                  // 标记需要更新状态
		} else {
			// 没有定时任务，直接返回
			response.Success(c, gin.H{"id": notice.ID, "message": "创建成功"})
			return
		}

		job := &models.Job{
			JobName:        services.GetNoticeJobName(notice.ID),
			JobGroup:       "notice",
			InvokeTarget:   "notice_publish:" + notice.ID,
			CronExpression: cronExpr,
			MisfirePolicy:  misfirePolicy,
			Concurrent:     false,
			Status:         0, // 正常状态
			Remark:         strPtr(fmt.Sprintf("通知定时发布: %s", req.NoticeTitle)),
		}

		if err := h.schedulerService.AddJob(job); err != nil {
			// 任务创建失败，返回错误
			response.Error(c, http.StatusInternalServerError, "创建定时任务失败: "+err.Error())
			return
		}

		// 定时任务创建成功后，更新发布状态为"定时发布中"，并保存结束时间
		if shouldUpdateToScheduled {
			updates := map[string]interface{}{
				"publish_status": models.PublishStatusScheduled,
			}
			// 如果是周期性任务且有结束时间，保存结束时间
			if executionType == "recurring" && req.RecurrenceConfig != nil && req.RecurrenceConfig.EndDate != nil {
				// 解析 ISO 格式的时间字符串
				if endTime, err := time.Parse(time.RFC3339, *req.RecurrenceConfig.EndDate); err == nil {
					updates["end_date"] = endTime
				}
			}
			// 这里需要DB访问来更新，暂时记录日志
			logger.Debugf("[NOTICE] 需要更新发布状态为定时发布中，通知ID: %s", notice.ID)
		}
	} else if executionType == "recurring" || (req.PublishTime != nil && req.PublishTime.After(time.Now())) {
		// 需要定时任务但调度器不可用
		response.Error(c, http.StatusInternalServerError, "调度器不可用，无法创建定时任务")
		return
	}

	response.Success(c, gin.H{"id": notice.ID, "message": "创建成功"})
}

// List 查询通知公告列表
func (h *NoticeHandler) List(c *gin.Context) {
	var rawReq map[string]interface{}
	if err := c.ShouldBindJSON(&rawReq); err != nil {
		rawReq = make(map[string]interface{})
	}

	params := requests.DefaultNoticeListParams()

	// 处理分页参数
	if val, ok := rawReq["current"]; ok {
		switch v := val.(type) {
		case float64:
			params.Current = int(v)
		case int:
			params.Current = v
		}
	}
	if val, ok := rawReq["pageSize"]; ok {
		switch v := val.(type) {
		case float64:
			params.PageSize = int(v)
		case int:
			params.PageSize = v
		}
	}

	// 处理字符串字段
	if val, ok := rawReq["noticeTitle"].(string); ok && val != "" {
		params.NoticeTitle = &val
	}
	if val, ok := rawReq["noticeType"].(string); ok && val != "" {
		params.NoticeType = &val
	}
	if val, ok := rawReq["createTime"].(string); ok && val != "" {
		params.CreateTime = &val
	}

	result, err := h.service.List(c.Request.Context(), params)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Page(c, result.List, result.Total, result.Current, result.PageSize)
}

// GetByID 获取通知公告详情
func (h *NoticeHandler) GetByID(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, http.StatusBadRequest, "通知公告ID不能为空")
		return
	}

	notice, err := h.service.GetByID(c.Request.Context(), id)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, notice)
}

// Update 更新通知公告
func (h *NoticeHandler) Update(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, http.StatusBadRequest, "通知公告ID不能为空")
		return
	}

	var req requests.NoticeUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "请求参数错误: "+err.Error())
		return
	}

	if err := h.service.Update(c.Request.Context(), id, &req); err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	// 设置发送渠道（如果提供了）
	if len(req.Channels) > 0 {
		channels := make([]models.NotificationChannel, len(req.Channels))
		for i, ch := range req.Channels {
			channels[i] = models.NotificationChannel{
				ChannelType:      ch.ChannelType,
				EmailConfigID:    ch.EmailConfigID,
				APIConfigID:      ch.APIConfigID,
				CustomRecipients: ch.CustomRecipients,
			}
		}
		if err := h.channelService.SetNotificationChannels(c.Request.Context(), id, channels); err != nil {
			response.Error(c, http.StatusInternalServerError, "设置发送渠道失败: "+err.Error())
			return
		}
	}

	// 处理定时任务更新（简化处理，不包含完整逻辑）
	if h.schedulerService != nil && req.PublishTime != nil && req.PublishTime.After(time.Now()) {
		jobName := services.GetNoticeJobName(id)
		cronExpr := services.CalculateCronExpression(*req.PublishTime)
		title := "通知定时发布"
		if req.NoticeTitle != nil {
			title = *req.NoticeTitle
		}
		job := &models.Job{
			JobName:        jobName,
			JobGroup:       "notice",
			InvokeTarget:   "notice_publish:" + id,
			CronExpression: cronExpr,
			MisfirePolicy:  models.MisfirePolicyExecuteOnce,
			Concurrent:     false,
			Status:         0,
			Remark:         strPtr(fmt.Sprintf("通知定时发布: %s", title)),
		}
		if err := h.schedulerService.AddJob(job); err != nil {
			response.Error(c, http.StatusInternalServerError, "创建定时任务失败: "+err.Error())
			return
		}
	}

	response.Success(c, gin.H{"message": "更新成功"})
}

// Delete 删除通知公告
func (h *NoticeHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, http.StatusBadRequest, "通知公告ID不能为空")
		return
	}

	if err := h.service.Delete(c.Request.Context(), id); err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	// 删除关联的定时任务（简化处理）
	if h.schedulerService != nil {
		jobName := services.GetNoticeJobName(id)
		// 这里需要DB访问来查找任务ID，简化处理
		logger.Debugf("[NOTICE] 需要删除定时任务: %s", jobName)
	}

	response.Success(c, gin.H{"message": "删除成功"})
}

// BatchDelete 批量删除通知公告
func (h *NoticeHandler) BatchDelete(c *gin.Context) {
	var req struct {
		IDs []string `json:"ids" binding:"required,min=1"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "请求参数错误: "+err.Error())
		return
	}

	if err := h.service.BatchDelete(c.Request.Context(), req.IDs); err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	// 删除关联的定时任务（简化处理）
	if h.schedulerService != nil {
		for _, id := range req.IDs {
			jobName := services.GetNoticeJobName(id)
			logger.Debugf("[NOTICE] 需要删除定时任务: %s", jobName)
		}
	}

	response.Success(c, gin.H{"message": "批量删除成功"})
}

// GetStatistics 获取通知阅读统计
func (h *NoticeHandler) GetStatistics(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, http.StatusBadRequest, "通知公告ID不能为空")
		return
	}

	stats, err := h.service.GetStatistics(c.Request.Context(), id)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, stats)
}

// Publish 发布通知
func (h *NoticeHandler) Publish(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, http.StatusBadRequest, "通知公告ID不能为空")
		return
	}

	if err := h.channelService.PublishAndSendNotice(c.Request.Context(), id); err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, gin.H{"message": "发布成功"})
}

// Withdraw 撤回/取消发布通知
func (h *NoticeHandler) Withdraw(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, http.StatusBadRequest, "通知公告ID不能为空")
		return
	}

	if err := h.service.Withdraw(c.Request.Context(), id); err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, gin.H{"message": "撤回成功"})
}

// GetChannels 获取通知的发送渠道配置
func (h *NoticeHandler) GetChannels(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, http.StatusBadRequest, "通知公告ID不能为空")
		return
	}

	channels, err := h.channelService.GetNotificationChannels(c.Request.Context(), id)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, channels)
}

// SetChannels 设置通知的发送渠道配置
func (h *NoticeHandler) SetChannels(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, http.StatusBadRequest, "通知公告ID不能为空")
		return
	}

	var req []requests.NotificationChannelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "请求参数错误: "+err.Error())
		return
	}

	channels := make([]models.NotificationChannel, len(req))
	for i, ch := range req {
		channels[i] = models.NotificationChannel{
			ChannelType:      ch.ChannelType,
			EmailConfigID:    ch.EmailConfigID,
			APIConfigID:      ch.APIConfigID,
			CustomRecipients: ch.CustomRecipients,
		}
	}

	if err := h.channelService.SetNotificationChannels(c.Request.Context(), id, channels); err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, gin.H{"message": "设置成功"})
}

// GetCronExpressions 获取常用Cron表达式列表
func (h *NoticeHandler) GetCronExpressions(c *gin.Context) {
	expressions := services.GetCommonCronExpressions()
	response.Success(c, expressions)
}

// strPtr 返回字符串指针的辅助函数
func strPtr(s string) *string {
	return &s
}
