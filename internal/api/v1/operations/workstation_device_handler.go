package operations

import (
	"github.com/gin-gonic/gin"
	opsServices "github.com/ruoyi-next/ruoyi-go-backend/internal/services/operations"
	apperrors "github.com/ruoyi-next/ruoyi-go-backend/pkg/errors"
	"github.com/ruoyi-next/ruoyi-go-backend/pkg/response"
)

// WorkstationDeviceHandler 工位设备关联处理器
type WorkstationDeviceHandler struct {
	service opsServices.WorkstationDeviceService
}

// NewWorkstationDeviceHandler 创建工位设备关联处理器实例
func NewWorkstationDeviceHandler(service opsServices.WorkstationDeviceService) *WorkstationDeviceHandler {
	return &WorkstationDeviceHandler{service: service}
}

// GetByWorkstation 获取工位关联设备列表
// @Summary 获取工位关联设备列表
// @Description 根据工位ID获取该工位关联的所有设备
// @Tags 运维管理
// @Accept json
// @Produce json
// @Param workstationId path string true "工位ID"
// @Success 200 {object} response.Response{data=[]models.WorkstationDevice}
// @Failure 400 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /ops/workstation-device/{workstationId} [get]
func (h *WorkstationDeviceHandler) GetByWorkstation(c *gin.Context) {
	workstationID := c.Param("workstationId")
	if workstationID == "" {
		response.Error(c, apperrors.ParamMissing("工位ID"))
		return
	}

	devices, err := h.service.GetDevicesByWorkstation(c.Request.Context(), workstationID)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, devices)
}

// AddManual 手动添加设备
// @Summary 手动添加设备
// @Description 通过序列号手动添加设备到工位
// @Tags 运维管理
// @Accept json
// @Produce json
// @Param request body opsServices.AddDeviceRequest true "设备信息"
// @Success 200 {object} response.Response{data=models.WorkstationDevice}
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /ops/workstation-device/manual [post]
func (h *WorkstationDeviceHandler) AddManual(c *gin.Context) {
	var req opsServices.AddDeviceRequest
	if !handleJSONBinding(c, &req) {
		return
	}

	device, err := h.service.AddDeviceManual(c.Request.Context(), &req)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, device)
}

// SyncAD 同步域控设备
// @Summary 同步域控设备
// @Description 从域控同步设备信息到工位
// @Tags 运维管理
// @Accept json
// @Produce json
// @Param request body object{workstation_id=string} true "工位ID"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /ops/workstation-device/sync-ad [post]
func (h *WorkstationDeviceHandler) SyncAD(c *gin.Context) {
	var req struct {
		WorkstationID string `json:"workstation_id" binding:"required"`
	}
	if !handleJSONBinding(c, &req) {
		return
	}

	if err := h.service.SyncFromAD(c.Request.Context(), req.WorkstationID); err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, gin.H{"message": "同步成功"})
}

// SyncAsset 同步资产设备
// @Summary 同步资产设备
// @Description 从资产系统同步设备信息到工位
// @Tags 运维管理
// @Accept json
// @Produce json
// @Param request body object{workstation_id=string} true "工位ID"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /ops/workstation-device/sync-asset [post]
func (h *WorkstationDeviceHandler) SyncAsset(c *gin.Context) {
	var req struct {
		WorkstationID string `json:"workstation_id" binding:"required"`
	}
	if !handleJSONBinding(c, &req) {
		return
	}

	if err := h.service.SyncFromAsset(c.Request.Context(), req.WorkstationID); err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, gin.H{"message": "同步成功"})
}

// Update 更新设备信息
// @Summary 更新设备信息
// @Description 更新工位设备的详细信息
// @Tags 运维管理
// @Accept json
// @Produce json
// @Param id path string true "设备ID"
// @Param request body opsServices.UpdateDeviceRequest true "更新信息"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /ops/workstation-device/{id}/update [post]
func (h *WorkstationDeviceHandler) Update(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, apperrors.ParamMissing("设备ID"))
		return
	}

	var req opsServices.UpdateDeviceRequest
	if !handleJSONBinding(c, &req) {
		return
	}

	if err := h.service.UpdateDevice(c.Request.Context(), id, &req); err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, nil)
}

// Delete 删除设备关联
// @Summary 删除设备关联
// @Description 删除工位与设备的关联关系
// @Tags 运维管理
// @Accept json
// @Produce json
// @Param id path string true "设备ID"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /ops/workstation-device/{id}/delete [post]
func (h *WorkstationDeviceHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, apperrors.ParamMissing("设备ID"))
		return
	}

	if err := h.service.DeleteDevice(c.Request.Context(), id); err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, nil)
}

// SetPrimary 设置主设备
// @Summary 设置主设备
// @Description 将指定设备设置为工位的主设备
// @Tags 运维管理
// @Accept json
// @Produce json
// @Param id path string true "设备ID"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /ops/workstation-device/{id}/set-primary [post]
func (h *WorkstationDeviceHandler) SetPrimary(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, apperrors.ParamMissing("设备ID"))
		return
	}

	if err := h.service.SetPrimaryDevice(c.Request.Context(), id); err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, nil)
}
