package operations

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/xingran-next/xingran-go-backend/internal/constants"
	"github.com/xingran-next/xingran-go-backend/internal/core"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/pkg/response"
)

// AssetComponentHandler exposes the read-side component listing endpoint
// for Phase 48 Wave 3 (D-07 frontend view).
//
// Unlike assetService.List (which default-filters component_type IS NULL
// per D-07), this handler returns ONLY component rows (component_type IS
// NOT NULL) under a given parent_asset_id. It is mounted at
// GET /ops/asset/components and reuses the existing ops:asset:list perm
// namespace per MEMORY xingran-perm-namespace-split-readonly-page (no new
// perm introduced).
type AssetComponentHandler struct {
	core *core.Core
}

// NewAssetComponentHandler constructs the handler. core must be non-nil
// before any ListComponents call (handler uses core.GetDB()).
func NewAssetComponentHandler(core *core.Core) *AssetComponentHandler {
	return &AssetComponentHandler{core: core}
}

// WithCore is a chainable setter satisfying the router's inline-construction
// pattern (mirrors AssetHandler.WithCore).
func (h *AssetComponentHandler) WithCore(core *core.Core) *AssetComponentHandler {
	if h != nil {
		h.core = core
	}
	return h
}

// ListComponents returns the subordinate component list for one parent
// asset. Query parameter parentAssetId is required and must be a valid
// UUID (constants.UUIDPattern).
//
// Response shape (response.Success wrapper):
//
//	{
//	  "code": 0,
//	  "data": { "list": Asset[], "total": N }
//	}
//
// Rows are ordered by component_type, then component_slot so the frontend
// Table renders a stable, grouped view.
func (h *AssetComponentHandler) ListComponents(c *gin.Context) {
	if h == nil || h.core == nil {
		response.Error(c, http.StatusInternalServerError, "core 未初始化")
		return
	}
	parentAssetID := c.Query("parentAssetId")
	if parentAssetID == "" {
		response.Error(c, http.StatusBadRequest, "parentAssetId 参数必填")
		return
	}
	if !constants.UUIDPattern.MatchString(parentAssetID) {
		response.Error(c, http.StatusBadRequest, "parentAssetId 必须是有效的 UUID")
		return
	}

	var list []models.Asset
	err := h.core.GetDB().WithContext(c.Request.Context()).
		Table("ops_asset").
		Where("parent_asset_id = ? AND deleted_at IS NULL AND component_type IS NOT NULL", parentAssetID).
		Order("component_type, component_slot").
		Find(&list).Error
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, gin.H{
		"list":  list,
		"total": len(list),
	})
}
