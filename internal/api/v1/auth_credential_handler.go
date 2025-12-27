package v1

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/core"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/services"
	"github.com/ruoyi-next/ruoyi-go-backend/pkg/response"
)

// AuthCredentialListRequest 授权凭证列表请求
type AuthCredentialListRequest struct {
	CredentialName *string                `json:"credentialName,omitempty"`
	ProtocolType   *models.ProtocolType   `json:"protocolType,omitempty"` // 改为协议类型过滤
	Current        int                    `json:"current" binding:"min=1"`
	PageSize       int                    `json:"pageSize" binding:"min=1,max=100"`
}

// AuthCredentialCreateRequest 创建授权凭证请求
type AuthCredentialCreateRequest struct {
	CredentialName  string               `json:"credentialName" binding:"required,min=2,max=100"`
	ProtocolType    models.ProtocolType  `json:"protocolType" binding:"required,oneof=ssh telnet"` // 改为协议类型
	Username        string               `json:"username,omitempty" binding:"max=100"`
	Password        string               `json:"password,omitempty" binding:"max=255"`
	EnablePassword  string               `json:"enablePassword,omitempty" binding:"max=255"`
	SNMPCommunities []string             `json:"snmpCommunities,omitempty"` // 改为数组
	SNMPVersion     models.SNMPVersion   `json:"snmpVersion,omitempty" binding:"omitempty,oneof=v1 v2c v3"`
	Description     string               `json:"description,omitempty"`
	IsDefault       bool                 `json:"isDefault"`
}

// AuthCredentialUpdateRequest 更新授权凭证请求
type AuthCredentialUpdateRequest struct {
	ID              string               `json:"id"` // ID 从 URL 参数获取，不需要验证
	CredentialName  string               `json:"credentialName" binding:"required,min=2,max=100"`
	ProtocolType    models.ProtocolType  `json:"protocolType" binding:"required,oneof=ssh telnet"` // 改为协议类型
	Username        string               `json:"username,omitempty" binding:"max=100"`
	Password        string               `json:"password,omitempty" binding:"max=255"`
	EnablePassword  string               `json:"enablePassword,omitempty" binding:"max=255"`
	SNMPCommunities []string             `json:"snmpCommunities,omitempty"` // 改为数组
	SNMPVersion     models.SNMPVersion   `json:"snmpVersion,omitempty" binding:"omitempty,oneof=v1 v2c v3"`
	Description     string               `json:"description,omitempty"`
	IsDefault       bool                 `json:"isDefault"`
}

// listAuthCredentials 查询授权凭证列表
// @Summary 查询授权凭证列表
// @Description 分页查询授权凭证列表
// @Tags 授权凭证管理
// @Accept json
// @Produce json
// @Param request body AuthCredentialListRequest true "查询条件"
// @Success 200 {object} response.Response{data=response.PageResponse}
// @Router /network/credentials/list [post]
func listAuthCredentials(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		var rawReq map[string]interface{}
		if err := c.ShouldBindJSON(&rawReq); err != nil {
			rawReq = make(map[string]interface{})
		}

		req := AuthCredentialListRequest{
			Current:  getIntField(rawReq, "current", 1),
			PageSize: getIntField(rawReq, "pageSize", 10),
		}

		if val, ok := rawReq["credentialName"].(string); ok && val != "" {
			req.CredentialName = &val
		}
		if val, ok := rawReq["protocolType"].(string); ok && val != "" {
			pt := models.ProtocolType(val)
			req.ProtocolType = &pt
		}

		svc := services.NewAuthCredentialService(core.DB.GetDB(), core.SM4Cipher)
		credentials, total, err := svc.List(c.Request.Context(), &services.ListCredentialRequest{
			CredentialName: req.CredentialName,
			ProtocolType:   req.ProtocolType,
			Current:        req.Current,
			PageSize:       req.PageSize,
		})
		if err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		pageResp := response.PageResponse{
			List:     credentials,
			Total:    total,
			Current:  req.Current,
			PageSize: req.PageSize,
		}

		response.Success(c, pageResp)
	}
}

// createAuthCredential 创建授权凭证
// @Summary 创建授权凭证
// @Description 创建新授权凭证
// @Tags 授权凭证管理
// @Accept json
// @Produce json
// @Param request body AuthCredentialCreateRequest true "凭证信息"
// @Success 200 {object} response.Response
// @Router /network/credentials [post]
func createAuthCredential(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req AuthCredentialCreateRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.Error(c, response.ErrBadRequest, "请求参数错误")
			return
		}

		svc := services.NewAuthCredentialService(core.DB.GetDB(), core.SM4Cipher)
		credential, err := svc.Create(c.Request.Context(), &services.CreateCredentialRequest{
			CredentialName:  req.CredentialName,
			ProtocolType:    req.ProtocolType,
			Username:        req.Username,
			Password:        req.Password,
			EnablePassword:  req.EnablePassword,
			SNMPCommunities: req.SNMPCommunities,
			SNMPVersion:     req.SNMPVersion,
			Description:     req.Description,
			IsDefault:       req.IsDefault,
			CreatedBy:       c.GetString("userId"),
		})
		if err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		response.Success(c, credential)
	}
}

// getAuthCredential 获取凭证详情
// @Summary 获取凭证详情
// @Description 根据ID获取凭证详情
// @Tags 授权凭证管理
// @Accept json
// @Produce json
// @Param id path string true "凭证ID"
// @Success 200 {object} response.Response{data=models.AuthCredential}
// @Router /network/credentials/:id [post]
func getAuthCredential(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if id == "" {
			response.Error(c, response.ErrBadRequest, "ID不能为空")
			return
		}

		svc := services.NewAuthCredentialService(core.DB.GetDB(), core.SM4Cipher)
		credential, err := svc.GetByID(c.Request.Context(), id)
		if err != nil {
			response.Error(c, response.ErrNotFound, "凭证不存在")
			return
		}

		response.Success(c, credential)
	}
}

// updateAuthCredential 更新授权凭证
// @Summary 更新授权凭证
// @Description 更新凭证信息
// @Tags 授权凭证管理
// @Accept json
// @Produce json
// @Param id path string true "凭证ID"
// @Param request body AuthCredentialUpdateRequest true "凭证信息"
// @Success 200 {object} response.Response
// @Router /network/credentials/:id/update [post]
func updateAuthCredential(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if id == "" {
			response.Error(c, response.ErrBadRequest, "ID不能为空")
			return
		}

		var req AuthCredentialUpdateRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.Error(c, response.ErrBadRequest, fmt.Sprintf("请求参数错误: %v", err))
			return
		}
		req.ID = id

		svc := services.NewAuthCredentialService(core.DB.GetDB(), core.SM4Cipher)
		if err := svc.Update(c.Request.Context(), &services.UpdateCredentialRequest{
			ID:              req.ID,
			CredentialName:  req.CredentialName,
			ProtocolType:    req.ProtocolType,
			Username:        req.Username,
			Password:        req.Password,
			EnablePassword:  req.EnablePassword,
			SNMPCommunities: req.SNMPCommunities,
			SNMPVersion:     req.SNMPVersion,
			Description:     req.Description,
			IsDefault:       req.IsDefault,
			UpdatedBy:       c.GetString("userId"),
		}); err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		response.Success(c, nil)
	}
}

// deleteAuthCredential 删除授权凭证
// @Summary 删除授权凭证
// @Description 删除指定凭证
// @Tags 授权凭证管理
// @Accept json
// @Produce json
// @Param id path string true "凭证ID"
// @Success 200 {object} response.Response
// @Router /network/credentials/:id/delete [post]
func deleteAuthCredential(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if id == "" {
			response.Error(c, response.ErrBadRequest, "ID不能为空")
			return
		}

		svc := services.NewAuthCredentialService(core.DB.GetDB(), core.SM4Cipher)
		if err := svc.Delete(c.Request.Context(), id); err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		response.Success(c, nil)
	}
}

// batchDeleteAuthCredentials 批量删除凭证
// @Summary 批量删除凭证
// @Description 批量删除多个凭证
// @Tags 授权凭证管理
// @Accept json
// @Produce json
// @Param request body map[string][]string true "ID列表"
// @Success 200 {object} response.Response
// @Router /network/credentials/batch-delete [post]
func batchDeleteAuthCredentials(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Ids []string `json:"ids" binding:"required,min=1"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			response.Error(c, response.ErrBadRequest, "请求参数错误")
			return
		}

		svc := services.NewAuthCredentialService(core.DB.GetDB(), core.SM4Cipher)
		if err := svc.BatchDelete(c.Request.Context(), req.Ids); err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		response.Success(c, nil)
	}
}

// setDefaultCredential 设置默认凭证
// @Summary 设置默认凭证
// @Description 设置指定凭证为默认凭证
// @Tags 授权凭证管理
// @Accept json
// @Produce json
// @Param id path string true "凭证ID"
// @Success 200 {object} response.Response
// @Router /network/credentials/:id/set-default [post]
func setDefaultCredential(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if id == "" {
			response.Error(c, response.ErrBadRequest, "ID不能为空")
			return
		}

		svc := services.NewAuthCredentialService(core.DB.GetDB(), core.SM4Cipher)
		if err := svc.SetDefaultCredential(c.Request.Context(), id, c.GetString("userId")); err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		response.Success(c, nil)
	}
}

// getDevicesByCredential 获取使用凭证的设备列表
// @Summary 获取使用凭证的设备列表
// @Description 获取使用指定凭证的所有设备
// @Tags 授权凭证管理
// @Accept json
// @Produce json
// @Param id path string true "凭证ID"
// @Success 200 {object} response.Response{data=[]models.NetworkDevice}
// @Router /network/credentials/:id/devices [post]
func getDevicesByCredential(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if id == "" {
			response.Error(c, response.ErrBadRequest, "ID不能为空")
			return
		}

		svc := services.NewAuthCredentialService(core.DB.GetDB(), core.SM4Cipher)
		devices, err := svc.GetDevicesByCredential(c.Request.Context(), id)
		if err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		response.Success(c, devices)
	}
}
