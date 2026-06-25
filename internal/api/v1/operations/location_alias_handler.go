package operations

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/core"
	opsServices "github.com/ruoyi-next/ruoyi-go-backend/internal/services/operations"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/utils/operlog"
	apperrors "github.com/ruoyi-next/ruoyi-go-backend/pkg/errors"
	"github.com/ruoyi-next/ruoyi-go-backend/pkg/response"
)

// LocationAliasHandler 工位部门 ↔ 物理位置部门 映射(别名)handler
//
// 4 个端点 (REQ-39-03):
//  - POST /ops/location-alias/list      分页查询(含 sys_dept JOIN)
//  - POST /ops/location-alias           新建
//  - POST /ops/location-alias/:id/update 更新
//  - POST /ops/location-alias/:id/delete 删除
//
// 3 个写方法(Create/Update/Delete)遵循 CLAUDE.md operlog 约定,模块名用 "工位管理"
// (与 workstation_handler.go 一致 — 工位部门下拉 union 是工位域功能)。
type LocationAliasHandler struct {
	service opsServices.LocationAliasService
	core    *core.Core
}

// NewLocationAliasHandler 构造 LocationAliasHandler 实例
func NewLocationAliasHandler(service opsServices.LocationAliasService) *LocationAliasHandler {
	return &LocationAliasHandler{service: service}
}

// WithCore 注入 core 依赖(操作日志记录所需)。链式注入而非改写构造器签名,
// 与 BuildingHandler/WorkstationHandler 保持调用点兼容(router.go)。
func (h *LocationAliasHandler) WithCore(core *core.Core) *LocationAliasHandler {
	if h != nil {
		h.core = core
	}
	return h
}

// aliasListQuery alias 列表查询请求体(分页参数)
type aliasListQuery struct {
	Current  int `json:"current" form:"current"`
	PageSize int `json:"pageSize" form:"pageSize"`
}

// List 查询 alias 列表(读操作,不写 operlog)
func (h *LocationAliasHandler) List(c *gin.Context) {
	pageNum := 1
	pageSize := 10

	if v := c.Query("current"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			pageNum = n
		}
	}
	if v := c.Query("pageSize"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			pageSize = n
		}
	}

	result, err := h.service.List(c.Request.Context(), pageNum, pageSize)
	if err != nil {
		response.Error(c, apperrors.Wrap(err, apperrors.CodeServerError, "查询别名映射失败"))
		return
	}
	response.Success(c, result)
}

// Create 新建 alias(写操作,触发 validateAlias 三级校验)
func (h *LocationAliasHandler) Create(c *gin.Context) {
	var req opsServices.LocationAliasCreateRequest
	if !handleJSONBinding(c, &req) {
		return
	}

	alias, err := h.service.Create(c.Request.Context(), &req)
	if err != nil {
		// validateAlias 三级校验失败:中文错误信息直接透传给前端
		// HTTP 400 + 明确错误信息(自映射/外部机构/后代)
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "工位管理", operlog.OperTypeCreate)
	response.Success(c, alias)
}

// Update 更新 alias(dept_id/location_id 变更时 service 层重新跑 validateAlias)
func (h *LocationAliasHandler) Update(c *gin.Context) {
	id := c.Param("id")
	var req opsServices.LocationAliasUpdateRequest
	if !handleJSONBinding(c, &req) {
		return
	}

	if err := h.service.Update(c.Request.Context(), id, &req); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "工位管理", operlog.OperTypeUpdate)
	response.Success(c, gin.H{"id": id})
}

// Delete 删除 alias(软删除)
func (h *LocationAliasHandler) Delete(c *gin.Context) {
	id := c.Param("id")

	if err := h.service.Delete(c.Request.Context(), id); err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "工位管理", operlog.OperTypeDelete)
	response.Success(c, gin.H{"id": id})
}
