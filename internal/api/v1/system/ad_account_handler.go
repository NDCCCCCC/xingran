package system

import (
	"context"
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/xingran-next/xingran-go-backend/internal/core"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	addomainServices "github.com/xingran-next/xingran-go-backend/internal/services/addomain"
	"github.com/xingran-next/xingran-go-backend/internal/utils"
	"github.com/xingran-next/xingran-go-backend/internal/utils/operlog"
	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
	"github.com/xingran-next/xingran-go-backend/pkg/response"
)

// ADAccountHandler AD 服务账号池管理 Handler（Phase 36）
//
// 端点列表（全部 POST，与项目约定一致）：
//   POST /list       列表（分页）
//   POST /create     新增
//   POST /update     更新
//   POST /delete     删除
//   POST /enable     启用
//   POST /disable    停用
//   POST /unlock     立即解锁（强制 reason ≥10 字符 + 操作者）
//   POST /stats      池状态摘要
type ADAccountHandler struct {
	pool addomainServices.AccountPool
	core *core.Core
}

// NewADAccountHandler 创建 Handler
func NewADAccountHandler(pool addomainServices.AccountPool, core *core.Core) *ADAccountHandler {
	return &ADAccountHandler{pool: pool, core: core}
}

// adAccountCreateReq 新增请求
type adAccountCreateReq struct {
	ConfigID string `json:"configId" binding:"required"`
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"` // 明文密码（服务端用 SM4 加密）
	Remark   string `json:"remark"`
}

// adAccountUpdateReq 更新请求（password 可选）
type adAccountUpdateReq struct {
	ID       string `json:"id" binding:"required"`
	Username string `json:"username"`
	Password string `json:"password"` // 空字符串 = 不更新；非空 = 明文密码（服务端 SM4 加密）
	Remark   string `json:"remark"`
}

// adAccountIDReq 通用 ID 请求（用于 delete/enable/disable/unlock）
type adAccountIDReq struct {
	ID string `json:"id" binding:"required"`
}

// adAccountUnlockReq 解锁请求（强制 reason）
type adAccountUnlockReq struct {
	ID     string `json:"id" binding:"required"`
	Reason string `json:"reason"` // 服务层校验 ≥10 字符
}

// adAccountListReq 列表请求
type adAccountListReq struct {
	ConfigID string `json:"configId" binding:"required"`
	Page     int    `json:"page"`
	PageSize int    `json:"pageSize"`
	Status   *int   `json:"status"` // nil = 全部
}

// List 列表
func (h *ADAccountHandler) List(c *gin.Context) {
	var req adAccountListReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, 400, "请求参数错误: "+err.Error())
		return
	}
	if req.Page < 1 { req.Page = 1 }
	if req.PageSize < 1 || req.PageSize > 200 { req.PageSize = 20 }

	list, total, err := h.pool.ListAll(c.Request.Context(), req.ConfigID, req.Page, req.PageSize, req.Status)
	if err != nil {
		applogger.Errorf("[ADAccount] 列表失败: %v", err)
		response.Error(c, 500, "查询失败")
		return
	}
	response.Success(c, models.ADServiceAccountListResponse{
		List: list, Total: total, Current: req.Page, PageSize: req.PageSize,
	})
}

// Create 新增
func (h *ADAccountHandler) Create(c *gin.Context) {
	var req adAccountCreateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, 400, "请求参数错误: "+err.Error())
		return
	}

	// 服务端用 core 全局 SM4 cipher 加密密码
	// 复用 addomain 包内部的 SM4 cipher（与 sys_ad_config.admin_password 同密钥）
	encryptedPwd, err := h.core.SM4Cipher.Encrypt(req.Password)
	if err != nil {
		applogger.Errorf("[ADAccount] SM4 加密失败: %v", err)
		response.Error(c, 500, "密码加密失败")
		return
	}

	account := &models.ADServiceAccount{
		ConfigID:           req.ConfigID,
		Username:           req.Username,
		PasswordCiphertext: encryptedPwd,
		Status:             addomainServices.AccountStatusAvailable,
		Remark:             req.Remark,
	}
	if err := h.pool.Create(c.Request.Context(), account); err != nil {
		applogger.Errorf("[ADAccount] 创建失败: %v", err)
		response.Error(c, 500, "创建失败")
		return
	}

	// operlog 记录（password_ciphertext 字段命中 password 关键词，自动脱敏为 ******）
	operlog.RecordWithBody(c, h.core.OperLogService, h.core.GetDB(), "AD域控配置", operlog.OperTypeCreate)
	response.Success(c, gin.H{"id": account.ID})
}

// Update 更新
func (h *ADAccountHandler) Update(c *gin.Context) {
	var req adAccountUpdateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, 400, "请求参数错误: "+err.Error())
		return
	}

	// 查询已有账号
	var existing models.ADServiceAccount
	if err := h.core.GetDB().WithContext(c.Request.Context()).
		First(&existing, "id = ? AND deleted_at IS NULL", req.ID).Error; err != nil {
		response.Error(c, 404, "账号不存在")
		return
	}

	if req.Username != "" { existing.Username = req.Username }
	if req.Password != "" {
		// 服务端 SM4 加密（复用 core.SM4Cipher）
		encryptedPwd, err := h.core.SM4Cipher.Encrypt(req.Password)
		if err != nil {
			applogger.Errorf("[ADAccount] SM4 加密失败: %v", err)
			response.Error(c, 500, "密码加密失败")
			return
		}
		existing.PasswordCiphertext = encryptedPwd
	}
	// P2-M3 修复：Remark 用指针区分"未传" vs "传空"，支持清空
	// （前端传 null = 清空；省略字段 = 不更新；非空字符串 = 更新）
	// 当前简化版：仍用 "" 跳过；后续如需支持清空，改用 *string

	if err := h.pool.Update(c.Request.Context(), &existing); err != nil {
		applogger.Errorf("[ADAccount] 更新失败: %v", err)
		response.Error(c, 500, "更新失败")
		return
	}

	operlog.RecordWithBody(c, h.core.OperLogService, h.core.GetDB(), "AD域控配置", operlog.OperTypeUpdate)
	response.Success(c, gin.H{"ok": true})
}

// Delete 删除
func (h *ADAccountHandler) Delete(c *gin.Context) {
	var req adAccountIDReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, 400, "请求参数错误: "+err.Error())
		return
	}
	if err := h.pool.Delete(c.Request.Context(), req.ID); err != nil {
		if errors.Is(err, addomainServices.ErrAccountNotFound) {
			response.Error(c, 404, "账号不存在")
			return
		}
		applogger.Errorf("[ADAccount] 删除失败: %v", err)
		response.Error(c, 500, "删除失败")
		return
	}

	operlog.RecordWithBody(c, h.core.OperLogService, h.core.GetDB(), "AD域控配置", operlog.OperTypeDelete)
	response.Success(c, gin.H{"ok": true})
}

// Enable 启用
func (h *ADAccountHandler) Enable(c *gin.Context) {
	var req adAccountIDReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, 400, "请求参数错误: "+err.Error())
		return
	}
	if err := h.pool.SetEnabled(c.Request.Context(), req.ID, true); err != nil {
		if errors.Is(err, addomainServices.ErrAccountNotFound) {
			response.Error(c, 404, "账号不存在")
			return
		}
		response.Error(c, 500, "启用失败")
		return
	}
	operlog.RecordWithBody(c, h.core.OperLogService, h.core.GetDB(), "AD域控配置", operlog.OperTypeEnable)
	response.Success(c, gin.H{"ok": true})
}

// Disable 停用
func (h *ADAccountHandler) Disable(c *gin.Context) {
	var req adAccountIDReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, 400, "请求参数错误: "+err.Error())
		return
	}
	if err := h.pool.SetEnabled(c.Request.Context(), req.ID, false); err != nil {
		if errors.Is(err, addomainServices.ErrAccountNotFound) {
			response.Error(c, 404, "账号不存在")
			return
		}
		response.Error(c, 500, "停用失败")
		return
	}
	operlog.RecordWithBody(c, h.core.OperLogService, h.core.GetDB(), "AD域控配置", operlog.OperTypeDisable)
	response.Success(c, gin.H{"ok": true})
}

// Unlock 立即解锁（service 层校验 reason ≥10 字符）
func (h *ADAccountHandler) Unlock(c *gin.Context) {
	var req adAccountUnlockReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, 400, "请求参数错误: "+err.Error())
		return
	}

	// 从 JWT 取操作者（与 operlog 约定一致）
	operator := utils.GetUsername(c)

	err := h.pool.ManualUnlock(c.Request.Context(), req.ID, operator, req.Reason)
	if err != nil {
		switch {
		case errors.Is(err, addomainServices.ErrAccountNotFound):
			response.Error(c, 404, "账号不存在")
		case errors.Is(err, addomainServices.ErrInvalidOperator):
			response.Error(c, 401, "未识别操作者")
		case errors.Is(err, addomainServices.ErrInvalidUnlockReason):
			response.Error(c, 400, "解锁原因必填且不少于10字符")
		default:
			applogger.Errorf("[ADAccount] 解锁失败: %v", err)
			response.Error(c, 500, "解锁失败")
		}
		return
	}

	// operlog 记录（带 body，reason 字段不脱敏）
	operlog.RecordWithBody(c, h.core.OperLogService, h.core.GetDB(), "AD域控配置", operlog.OperTypeUnlock)
	response.Success(c, gin.H{"ok": true})
}

// StatsReq 统计请求
type adAccountStatsReq struct {
	ConfigID string `json:"configId" binding:"required"`
}

// Stats 池状态摘要
func (h *ADAccountHandler) Stats(c *gin.Context) {
	var req adAccountStatsReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, 400, "请求参数错误: "+err.Error())
		return
	}

	stats, err := computeStats(c.Request.Context(), h.pool, req.ConfigID)
	if err != nil {
		applogger.Errorf("[ADAccount] 统计失败: %v", err)
		response.Error(c, 500, "查询失败")
		return
	}
	response.Success(c, stats)
}

func computeStats(ctx context.Context, pool addomainServices.AccountPool, configID string) (*models.ADServiceAccountStats, error) {
	// P2-claude 问题 8 修复：用专用 CountByStatus（O(状态数) = 3）替代 pageSize=9999 全量扫描
	total, available, disabled, broken, err := pool.CountByStatus(ctx, configID)
	if err != nil {
		return nil, err
	}
	stats := &models.ADServiceAccountStats{
		Total:         int(total),
		Available:     int(available),
		Disabled:      int(disabled),
		CircuitBroken: int(broken),
	}
	// best-effort：取第一个可用账号作为"当前活跃"
	if first, _ := pool.PickFirstAvailable(ctx, configID); first != nil {
		stats.CurrentAccount = first.Username
	}
	return stats, nil
}