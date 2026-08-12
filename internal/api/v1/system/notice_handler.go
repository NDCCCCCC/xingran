package system

import (
	"time"

	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/xingran-next/xingran-go-backend/internal/core"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/models/system/requests"
	"github.com/xingran-next/xingran-go-backend/internal/services"
	systemServices "github.com/xingran-next/xingran-go-backend/internal/services/system"
	"github.com/xingran-next/xingran-go-backend/internal/utils/operlog"
	apperrors "github.com/xingran-next/xingran-go-backend/pkg/errors"
	"github.com/xingran-next/xingran-go-backend/pkg/logger"
	"github.com/xingran-next/xingran-go-backend/pkg/response"
)

// NoticeHandler 通知公告处理器
type NoticeHandler struct {
	service          systemServices.NoticeCacheService
	channelService   systemServices.NotificationChannelService
	schedulerService SchedulerService
	core             *core.Core
}

// SchedulerService 调度器服务接口
type SchedulerService interface {
	AddJob(job *models.Job) error
	RemoveJob(jobID string) error
}

// NewNoticeHandler 创建通知公告处理器实例
func NewNoticeHandler(
	service systemServices.NoticeCacheService,
	channelService systemServices.NotificationChannelService,
	schedulerSvc SchedulerService,
) *NoticeHandler {
	return &NoticeHandler{
		service:          service,
		channelService:   channelService,
		schedulerService: schedulerSvc,
	}
}

// WithCore 注入 core 依赖（操作日志记录所需）。返回 receiver 自身以支持链式调用。
// Phase 34 操作日志全模块覆盖新增，用于 operlog.Record 访问 core.OperLogService 与
// core.GetDB()。不改写 NewNoticeHandler 三参构造器签名，避免破坏既有调用点。
func (h *NoticeHandler) WithCore(core *core.Core) *NoticeHandler {
	if h != nil {
		h.core = core
	}
	return h
}

// Statistics 通知状态统计(总数/已发布/草稿/定时发布)
// @Summary 通知状态统计
// @Description 返回通知总数及各发布状态计数,供统计卡片使用;用 COUNT 聚合而非加载全量行
// @Tags 通知公告
// @Produce json
// @Success 200 {object} response.Response
// @Router /system/notices/statistics [post]
func (h *NoticeHandler) Statistics(c *gin.Context) {
	result, err := h.service.GetStatusStatistics(c.Request.Context())
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, result)
}

// Create 创建通知公告
// @Summary 创建通知公告
// @Description 创建新的通知公告
// @Tags 通知公告
// @Accept json
// @Produce json
// @Param request body object{noticeTitle=string,noticeType=string,noticeContent=string,status=int,publishTime=string,executionType=string,recurrenceConfig=object{cronExpression=string,endDate=string},channels=[]object{channelType=string,emailConfigId=string,apiConfigId=string,customRecipients=[]string}} true "通知公告信息"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /system/notices [post]
func (h *NoticeHandler) Create(c *gin.Context) {
	var req requests.NoticeCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperrors.Wrap(err, apperrors.CodeParamError, "请求参数错误"))
		return
	}

	// 获取当前用户信息
	userID, exists := c.Get("user_id")
	if !exists {
		response.Error(c, apperrors.Unauthorized())
		return
	}
	username, _ := c.Get("username")
	usernameStr, _ := username.(string)

	notice, err := h.service.Create(c.Request.Context(), &req, userID.(string), usernameStr)
	if err != nil {
		response.Error(c, err)
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
			response.Error(c, apperrors.InternalServerErrorWithMsg("设置发送渠道失败"))
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
				response.Error(c, apperrors.ParamMissing("Cron表达式"))
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
			operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "通知公告", operlog.OperTypeCreate)
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
			response.Error(c, apperrors.InternalServerErrorWithMsg("创建定时任务失败"))
			return
		}

		// 定时任务创建成功后，更新发布状态为"定时发布中"，并保存结束时间。
		// Phase 34 WR-003 修复：原实现只有 logger.Debugf 占位，未真正写入 sys_notice.publish_status，
		// 产生"幻影成功"审计行（operlog 记录创建成功，但 DB 中状态未更新）。
		if shouldUpdateToScheduled {
			publishStatus := models.PublishStatusScheduled
			updateReq := &requests.NoticeUpdateRequest{
				PublishStatus: &publishStatus,
			}
			// 如果是周期性任务且有结束时间，保存结束时间
			if executionType == "recurring" && req.RecurrenceConfig != nil && req.RecurrenceConfig.EndDate != nil {
				// 解析 ISO 格式的时间字符串
				if endTime, err := time.Parse(time.RFC3339, *req.RecurrenceConfig.EndDate); err == nil {
					updateReq.EndDate = &endTime
				}
			}
			if err := h.service.Update(c.Request.Context(), notice.ID, updateReq); err != nil {
				response.Error(c, apperrors.InternalServerErrorWithMsg("更新发布状态失败"))
				return
			}
		}
	} else if executionType == "recurring" || (req.PublishTime != nil && req.PublishTime.After(time.Now())) {
		// 需要定时任务但调度器不可用
		response.Error(c, apperrors.InternalServerErrorWithMsg("调度器不可用，无法创建定时任务"))
		return
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "通知公告", operlog.OperTypeCreate)
	response.Success(c, gin.H{"id": notice.ID, "message": "创建成功"})
}

// List 查询通知公告列表
// @Summary 查询通知公告列表
// @Description 查询通知公告列表，支持多条件筛选和分页
// @Tags 通知公告
// @Accept json
// @Produce json
// @Param request body object{noticeTitle=string,noticeType=string,createTime=string,current=int,pageSize=int} true "查询条件"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /system/notices/list [post]
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

	// 服务端排序参数（透传给 GetNoticeList → base.ApplySort 白名单）
	if val, ok := rawReq["orderByColumn"].(string); ok && val != "" {
		params.OrderByColumn = val
	}
	if val, ok := rawReq["isAsc"]; ok {
		if b, ok := val.(bool); ok {
			params.IsAsc = &b
		}
	}

	result, err := h.service.List(c.Request.Context(), params)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Page(c, result.List, result.Total, result.Current, result.PageSize)
}

// GetByID 获取通知公告详情
// @Summary 获取通知公告详情
// @Description 根据通知公告ID获取详细信息
// @Tags 通知公告
// @Accept json
// @Produce json
// @Param id path string true "通知公告ID"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /system/notices/:id [post]
func (h *NoticeHandler) GetByID(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, apperrors.ParamMissing("通知公告ID"))
		return
	}

	notice, err := h.service.GetNoticeByID(c.Request.Context(), id)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, notice)
}

// Update 更新通知公告
// @Summary 更新通知公告
// @Description 更新通知公告信息
// @Tags 通知公告
// @Accept json
// @Produce json
// @Param id path string true "通知公告ID"
// @Param request body object{noticeTitle=string,noticeType=string,noticeContent=string,status=int,publishTime=string,channels=[]object{channelType=string,emailConfigId=string,apiConfigId=string,customRecipients=[]string}} true "通知公告信息"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /system/notices/:id/update [post]
func (h *NoticeHandler) Update(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, apperrors.ParamMissing("通知公告ID"))
		return
	}

	var req requests.NoticeUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperrors.Wrap(err, apperrors.CodeParamError, "请求参数错误"))
		return
	}

	if err := h.service.Update(c.Request.Context(), id, &req); err != nil {
		response.Error(c, err)
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
			response.Error(c, apperrors.InternalServerErrorWithMsg("设置发送渠道失败"))
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
			response.Error(c, apperrors.InternalServerErrorWithMsg("创建定时任务失败"))
			return
		}
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "通知公告", operlog.OperTypeUpdate)
	response.Success(c, gin.H{"message": "更新成功"})
}

// Delete 删除通知公告
// @Summary 删除通知公告
// @Description 删除指定的通知公告
// @Tags 通知公告
// @Accept json
// @Produce json
// @Param id path string true "通知公告ID"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /system/notices/:id/delete [post]
func (h *NoticeHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, apperrors.ParamMissing("通知公告ID"))
		return
	}

	if err := h.service.Delete(c.Request.Context(), id); err != nil {
		response.Error(c, err)
		return
	}

	// 删除关联的定时任务（简化处理）
	if h.schedulerService != nil {
		jobName := services.GetNoticeJobName(id)
		// 这里需要DB访问来查找任务ID，简化处理
		logger.Debugf("[NOTICE] 需要删除定时任务: %s", jobName)
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "通知公告", operlog.OperTypeDelete)
	response.Success(c, gin.H{"message": "删除成功"})
}

// BatchDelete 批量删除通知公告
// @Summary 批量删除通知公告
// @Description 批量删除多个通知公告
// @Tags 通知公告
// @Accept json
// @Produce json
// @Param request body object{ids=[]string} true "通知公告ID列表"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /system/notices/batch [post]
func (h *NoticeHandler) BatchDelete(c *gin.Context) {
	var req struct {
		IDs []string `json:"ids" binding:"required,min=1"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperrors.Wrap(err, apperrors.CodeParamError, "请求参数错误"))
		return
	}

	if err := h.service.BatchDelete(c.Request.Context(), req.IDs); err != nil {
		response.Error(c, err)
		return
	}

	// 删除关联的定时任务（简化处理）
	if h.schedulerService != nil {
		for _, id := range req.IDs {
			jobName := services.GetNoticeJobName(id)
			logger.Debugf("[NOTICE] 需要删除定时任务: %s", jobName)
		}
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "通知公告", operlog.OperTypeBatch)
	response.Success(c, gin.H{"message": "批量删除成功"})
}

// GetStatistics 获取通知阅读统计
// @Summary 获取通知阅读统计
// @Description 获取通知公告的阅读统计信息
// @Tags 通知公告
// @Accept json
// @Produce json
// @Param id path string true "通知公告ID"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /system/notices/:id/statistics [get]
func (h *NoticeHandler) GetStatistics(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, apperrors.ParamMissing("通知公告ID"))
		return
	}

	stats, err := h.service.GetStatistics(c.Request.Context(), id)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, stats)
}

// Publish 发布通知
// @Summary 发布通知
// @Description 发布通知公告
// @Tags 通知公告
// @Accept json
// @Produce json
// @Param id path string true "通知公告ID"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /system/notices/:id/publish [post]
func (h *NoticeHandler) Publish(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, apperrors.ParamMissing("通知公告ID"))
		return
	}

	if err := h.channelService.PublishAndSendNotice(c.Request.Context(), id); err != nil {
		response.Error(c, err)
		return
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "通知公告", operlog.OperTypeUpdate)
	response.Success(c, gin.H{"message": "发布成功"})
}

// Withdraw 撤回/取消发布通知
// @Summary 撤回通知
// @Description 撤回已发布的通知公告
// @Tags 通知公告
// @Accept json
// @Produce json
// @Param id path string true "通知公告ID"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /system/notices/:id/withdraw [post]
func (h *NoticeHandler) Withdraw(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, apperrors.ParamMissing("通知公告ID"))
		return
	}

	if err := h.service.Withdraw(c.Request.Context(), id); err != nil {
		response.Error(c, err)
		return
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "通知公告", operlog.OperTypeUpdate)
	response.Success(c, gin.H{"message": "撤回成功"})
}

// GetChannels 获取通知的发送渠道配置
// @Summary 获取通知渠道配置
// @Description 获取通知公告的发送渠道配置
// @Tags 通知公告
// @Accept json
// @Produce json
// @Param id path string true "通知公告ID"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /system/notices/:id/channels [get]
func (h *NoticeHandler) GetChannels(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, apperrors.ParamMissing("通知公告ID"))
		return
	}

	channels, err := h.channelService.GetNotificationChannels(c.Request.Context(), id)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, channels)
}

// SetChannels 设置通知的发送渠道配置
// @Summary 设置通知渠道配置
// @Description 设置通知公告的发送渠道配置
// @Tags 通知公告
// @Accept json
// @Produce json
// @Param id path string true "通知公告ID"
// @Param request body object{channelType=string,emailConfigId=string,apiConfigId=string,customRecipients=[]string} true "渠道配置列表"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /system/notices/:id/channels [post]
func (h *NoticeHandler) SetChannels(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, apperrors.ParamMissing("通知公告ID"))
		return
	}

	var req []requests.NotificationChannelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperrors.Wrap(err, apperrors.CodeParamError, "请求参数错误"))
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
		response.Error(c, err)
		return
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "通知公告", operlog.OperTypeUpdate)
	response.Success(c, gin.H{"message": "设置成功"})
}

// GetCronExpressions 获取常用Cron表达式列表
// @Summary 获取常用Cron表达式
// @Description 获取常用的Cron表达式列表
// @Tags 通知公告
// @Accept json
// @Produce json
// @Success 200 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /system/notices/cron-expressions [get]
func (h *NoticeHandler) GetCronExpressions(c *gin.Context) {
	expressions := services.GetCommonCronExpressions()
	response.Success(c, expressions)
}

// strPtr 返回字符串指针的辅助函数
func strPtr(s string) *string {
	return &s
}
