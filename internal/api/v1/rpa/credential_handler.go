package rpa

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/xingran-next/xingran-go-backend/internal/core"
	rpamodels "github.com/xingran-next/xingran-go-backend/internal/models/rpa"
	"github.com/xingran-next/xingran-go-backend/internal/services/rpa"
	"github.com/xingran-next/xingran-go-backend/internal/utils/operlog"
	"github.com/xingran-next/xingran-go-backend/pkg/response"
)

// CredentialHandler 凭证处理器
type CredentialHandler struct {
	credentialService rpa.CredentialService
	sessionService    rpa.CredentialService
	core              *core.Core
}

// NewCredentialHandler 创建凭证处理器
func NewCredentialHandler(credService rpa.CredentialService) *CredentialHandler {
	return &CredentialHandler{
		credentialService: credService,
		sessionService:    credService,
	}
}

// WithCore 注入 core 依赖（用于操作日志埋点），链式调用
func (h *CredentialHandler) WithCore(core *core.Core) *CredentialHandler {
	h.core = core
	return h
}

// List 列出凭证
// @Summary 查询RPA凭据列表
// @Description 分页查询RPA凭据列表（凭据值脱敏）
// @Tags RPA管理
// @Accept json
// @Produce json
// @Param request body object true "查询参数"
// @Success 200 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /rpa/credentials/list [post]
func (h *CredentialHandler) List(c *gin.Context) {
	var params rpamodels.CredentialListParams
	if !bindAndValidate(c, &params) {
		return
	}

	setPaginationDefaults(&params.Current, &params.PageSize)

	// 获取用户信息
	userID := c.GetString("userId")
	deptID := c.GetString("deptId")

	result, total, err := h.credentialService.ListCredentials(c.Request.Context(), &params, userID, deptID)
	if handleError(c, err, http.StatusInternalServerError, "查询失败") {
		return
	}

	success(c, gin.H{
		"list":  result,
		"total": total,
	})
}

// Create 创建凭证
// @Summary 创建RPA凭据
// @Description 创建新的RPA凭据
// @Tags RPA管理
// @Accept json
// @Produce json
// @Param request body object true "凭据信息"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /rpa/credentials [post]
func (h *CredentialHandler) Create(c *gin.Context) {
	var req rpamodels.CredentialCreateRequest
	if !bindAndValidate(c, &req) {
		return
	}

	userID := c.GetString("userId")
	cred, err := h.credentialService.CreateCredential(c.Request.Context(), &req, userID)
	if handleError(c, err, http.StatusInternalServerError, "创建失败") {
		return
	}

	// T-34-W6-02 缓解：RecordWithBody 屏蔽 password/secret/key/token 等敏感字段
	operlog.RecordWithBody(c, h.core.OperLogService, h.core.GetDB(), "RPA凭据", operlog.OperTypeCreate)

	success(c, cred)
}

// GetByID 获取凭证详情
// @Summary 查询RPA凭据详情
// @Description 按ID查询RPA凭据详情（凭据值脱敏）
// @Tags RPA管理
// @Produce json
// @Param id path string true "凭据ID"
// @Success 200 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /rpa/credentials/{id} [post]
func (h *CredentialHandler) GetByID(c *gin.Context) {
	id := getIDParam(c)
	if id == "" {
		return
	}

	userID := c.GetString("userId")
	cred, err := h.credentialService.GetCredential(c.Request.Context(), id, userID)
	if err != nil {
		response.Error(c, http.StatusNotFound, "凭证不存在")
		return
	}

	success(c, cred)
}

// Update 更新凭证
// @Summary 更新RPA凭据
// @Description 按 ID 更新RPA凭据
// @Tags RPA管理
// @Accept json
// @Produce json
// @Param id path string true "凭据ID"
// @Param request body object true "更新内容"
// @Success 200 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /rpa/credentials/{id}/update [post]
func (h *CredentialHandler) Update(c *gin.Context) {
	id := getIDParam(c)
	if id == "" {
		return
	}

	var req rpamodels.CredentialUpdateRequest
	if !bindAndValidate(c, &req) {
		return
	}

	userID := c.GetString("userId")
	if handleError(c, h.credentialService.UpdateCredential(c.Request.Context(), id, &req, userID), http.StatusInternalServerError, "更新失败") {
		return
	}

	// T-34-W6-02 缓解：RecordWithBody 屏蔽敏感字段
	operlog.RecordWithBody(c, h.core.OperLogService, h.core.GetDB(), "RPA凭据", operlog.OperTypeUpdate)

	successMsg(c, "更新成功")
}

// Delete 删除凭证
// @Summary 删除RPA凭据
// @Description 删除指定RPA凭据
// @Tags RPA管理
// @Produce json
// @Param id path string true "凭据ID"
// @Success 200 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /rpa/credentials/{id}/delete [post]
func (h *CredentialHandler) Delete(c *gin.Context) {
	id := getIDParam(c)
	if id == "" {
		return
	}

	userID := c.GetString("userId")
	if handleError(c, h.credentialService.DeleteCredential(c.Request.Context(), id, userID), http.StatusInternalServerError, "删除失败") {
		return
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "RPA凭据", operlog.OperTypeDelete)

	successMsg(c, "删除成功")
}

// ===== 会话管理 =====

// ListSessions 列出会话
// @Summary 查询RPA凭据会话列表
// @Description 查询凭据关联的会话列表
// @Tags RPA管理
// @Produce json
// @Success 200 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /rpa/credentials/sessions/list [post]
func (h *CredentialHandler) ListSessions(c *gin.Context) {
	credentialID := c.Query("credentialId")
	if credentialID == "" {
		response.Error(c, http.StatusBadRequest, "缺少凭证ID")
		return
	}

	var params rpamodels.SessionListParams
	if !bindAndValidate(c, &params) {
		return
	}

	setPaginationDefaults(&params.Current, &params.PageSize)
	params.CredentialID = credentialID

	// 使用 sessionService 查询
	success(c, gin.H{
		"list":  []interface{}{},
		"total": 0,
	})
}

// InvalidateSession 使会话失效
// @Summary 使凭据会话失效
// @Description 使指定凭据会话失效
// @Tags RPA管理
// @Produce json
// @Param id path string true "会话ID"
// @Success 200 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /rpa/credentials/sessions/{id}/invalidate [post]
func (h *CredentialHandler) InvalidateSession(c *gin.Context) {
	id := getIDParam(c)
	if id == "" {
		return
	}

	reason := c.DefaultQuery("reason", "用户手动失效")

	if handleError(c, h.sessionService.InvalidateSession(c.Request.Context(), id, reason), http.StatusInternalServerError, "操作失败") {
		return
	}

	// T-34-W6-02 缓解：失效会话属于凭据/令牌生命周期变更，使用 OperTypeStatus
	operlog.RecordWithBody(c, h.core.OperLogService, h.core.GetDB(), "RPA凭据", operlog.OperTypeStatus)

	successMsg(c, "会话已失效")
}
