package system

import (
	"context"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/models/system/requests"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/services/addomain"
	systemServices "github.com/ruoyi-next/ruoyi-go-backend/internal/services/system"
	apperrors "github.com/ruoyi-next/ruoyi-go-backend/pkg/errors"
	applogger "github.com/ruoyi-next/ruoyi-go-backend/pkg/logger"
	"github.com/ruoyi-next/ruoyi-go-backend/pkg/response"
)

// UserHandler 用户处理器
type UserHandler struct {
	service           systemServices.UserService
	userADSyncService *addomain.UserADSyncService
	// adSyncInFlight P0 #10: 按用户 ID 去重 AD 同步，避免短时间内
	// 多次更新同一用户触发并发 LDAP 连接。sync.Map 零值可用，无需初始化。
	adSyncInFlight sync.Map
}

// NewUserHandler 创建用户处理器实例
func NewUserHandler(service systemServices.UserService, userADSyncService ...*addomain.UserADSyncService) *UserHandler {
	handler := &UserHandler{service: service}
	if len(userADSyncService) > 0 && userADSyncService[0] != nil {
		handler.userADSyncService = userADSyncService[0]
	}
	return handler
}

// Create 创建用户
// @Summary 创建用户
// @Description 创建新用户
// @Tags 用户管理
// @Accept json
// @Produce json
// @Param request body requests.UserCreateRequest true "用户信息"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /system/users [post]
func (h *UserHandler) Create(c *gin.Context) {
	var req requests.UserCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperrors.Wrap(err, apperrors.CodeParamError, "请求参数错误"))
		return
	}

	if err := h.service.Create(c.Request.Context(), &req); err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, gin.H{"message": "创建成功"})
}

// List 查询用户列表（类型安全版本）
// @Summary 查询用户列表
// @Description 分页查询用户列表，支持多条件过滤
// @Tags 用户管理
// @Accept json
// @Produce json
// @Param request body requests.UserListParams true "查询条件"
// @Success 200 {object} response.Response{data=response.PageResponse}
// @Router /system/users/list [post]
func (h *UserHandler) List(c *gin.Context) {
	// 直接绑定到类型安全的请求结构体
	var params requests.UserListParams
	if err := c.ShouldBindJSON(&params); err != nil {
		// 如果是空请求体或解析失败，使用默认值
		params = requests.DefaultUserListParams()
	}

	// 应用分页限制
	current, pageSize := params.GetPagination()
	params.Current = current
	params.PageSize = pageSize

	result, err := h.service.List(c.Request.Context(), params)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Page(c, result.List, result.Total, result.Current, result.PageSize)
}

// GetByID 获取用户详情
// @Summary 获取用户详情
// @Description 根据用户ID获取用户详细信息
// @Tags 用户管理
// @Accept json
// @Produce json
// @Param id path string true "用户ID"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /system/users/:id [post]
func (h *UserHandler) GetByID(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, apperrors.ParamMissing("用户ID"))
		return
	}

	user, err := h.service.GetByID(c.Request.Context(), id)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, user)
}

// Update 更新用户
// @Summary 更新用户
// @Description 更新用户信息
// @Tags 用户管理
// @Accept json
// @Produce json
// @Param id path string true "用户ID"
// @Param request body requests.UserUpdateRequest true "用户信息"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /system/users/:id/update [post]
func (h *UserHandler) Update(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, apperrors.ParamMissing("用户ID"))
		return
	}

	var req requests.UserUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperrors.Wrap(err, apperrors.CodeParamError, "请求参数错误"))
		return
	}

	req.ID = id
	if err := h.service.Update(c.Request.Context(), &req); err != nil {
		response.Error(c, err)
		return
	}

	// 系统更新成功后，异步同步到AD域控（降级处理：不阻断系统更新）
	// P0 #10: 在原有 panic recover 基础上增加 (1) dedupe 防止短时间内对同一
	// 用户的多次更新触发并发 LDAP 连接；(2) 有限重试 + 指数退避，吸收 AD
	// 连接/网络的瞬时抖动。SyncUserUpdateToAD 每次调用都新建 LDAP 连接，
	// 故重试在 handler 层做最稳妥。
	if h.userADSyncService != nil {
		adSyncReq := h.buildADSyncMap(&req)
		// dedupe：若该用户的同步仍在飞行中，跳过本次重复触发
		if _, inFlight := h.adSyncInFlight.LoadOrStore(id, struct{}{}); inFlight {
			applogger.Infof("[AD-SYNC] 用户 %s 的 AD 同步仍在进行中，跳过本次重复触发", id)
		} else {
			go func() {
				defer func() {
					h.adSyncInFlight.Delete(id) // 释放去重标记（无论成功/失败/panic）
					if r := recover(); r != nil {
						applogger.Errorf("[AD-SYNC] SyncUserUpdateToAD panic 已恢复 [ID=%s]: panic=%v", id, r)
					}
				}()
				ctx := context.Background()
				const maxRetries = 3
				var lastErr error
				for attempt := 0; attempt < maxRetries; attempt++ {
					if attempt > 0 {
						// 指数退避：1s, 2s（仅阻塞本 goroutine，不影响请求）
						time.Sleep(time.Duration(1<<uint(attempt-1)) * time.Second)
					}
					if err := h.userADSyncService.SyncUserUpdateToAD(ctx, id, adSyncReq); err != nil {
						lastErr = err
						applogger.Warnf("[AD-SYNC] 同步失败 [ID=%s] 第%d/%d次: %v", id, attempt+1, maxRetries, err)
						continue
					}
					lastErr = nil
					break
				}
				if lastErr != nil {
					applogger.Errorf("[AD-SYNC] 同步用户到AD最终失败（已重试%d次）[ID=%s]: %v", maxRetries, id, lastErr)
				}
			}()
		}
	}

	response.Success(c, gin.H{"message": "更新成功"})
}

// buildADSyncMap 将UserUpdateRequest转换为AD同步用的map
func (h *UserHandler) buildADSyncMap(req *requests.UserUpdateRequest) map[string]interface{} {
	m := make(map[string]interface{})
	if req.Nickname != nil {
		m["nickname"] = *req.Nickname
	}
	if req.Email != nil {
		m["email"] = *req.Email
	}
	if req.Phone != nil {
		m["phone"] = *req.Phone
	}
	if req.DeptID != nil {
		m["deptId"] = *req.DeptID
	}
	return m
}

// Delete 删除用户
// @Summary 删除用户
// @Description 删除指定用户
// @Tags 用户管理
// @Accept json
// @Produce json
// @Param id path string true "用户ID"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Router /system/users/:id/delete [post]
func (h *UserHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, apperrors.ParamMissing("用户ID"))
		return
	}

	if err := h.service.Delete(c.Request.Context(), id); err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, nil)
}

// BatchDelete 批量删除用户
// @Summary 批量删除用户
// @Description 批量删除多个用户
// @Tags 用户管理
// @Accept json
// @Produce json
// @Param request body object{ids=[]string} true "用户ID列表"
// @Success 200 {object} response.Response
// @Router /system/users/batch [post]
func (h *UserHandler) BatchDelete(c *gin.Context) {
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

	response.Success(c, nil)
}

// UpdateStatus 更新用户状态
// @Summary 更新用户状态
// @Description 更新用户的启用/禁用状态
// @Tags 用户管理
// @Accept json
// @Produce json
// @Param id path string true "用户ID"
// @Param request body object{status=int} true "状态"
// @Success 200 {object} response.Response
// @Router /system/users/:id/status [post]
func (h *UserHandler) UpdateStatus(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, apperrors.ParamMissing("用户ID"))
		return
	}

	var req struct {
		Status int `json:"status" binding:"min=0,max=1"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperrors.Wrap(err, apperrors.CodeParamError, "请求参数错误"))
		return
	}

	if err := h.service.UpdateStatus(c.Request.Context(), id, req.Status); err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, gin.H{"message": "状态更新成功"})
}

// ResetPassword 重置用户密码
// @Summary 重置用户密码
// @Description 管理员重置指定用户的密码
// @Tags 用户管理
// @Accept json
// @Produce json
// @Param id path string true "用户ID"
// @Success 200 {object} response.Response
// @Router /system/users/:id/reset-password [post]
func (h *UserHandler) ResetPassword(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, apperrors.ParamMissing("用户ID"))
		return
	}

	// 默认密码为123456
	defaultPassword := "123456"
	if err := h.service.ResetPassword(c.Request.Context(), id, defaultPassword); err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, gin.H{"message": "重置成功，新密码为：123456"})
}
