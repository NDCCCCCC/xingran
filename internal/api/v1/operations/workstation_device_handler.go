package operations

import (
	"github.com/gin-gonic/gin"
	"github.com/xingran-next/xingran-go-backend/internal/core"
	opsServices "github.com/xingran-next/xingran-go-backend/internal/services/operations"
	"github.com/xingran-next/xingran-go-backend/internal/utils/operlog"
	apperrors "github.com/xingran-next/xingran-go-backend/pkg/errors"
	"github.com/xingran-next/xingran-go-backend/pkg/response"
)

// WorkstationDeviceHandler 工位设备关联处理器
type WorkstationDeviceHandler struct {
	service opsServices.WorkstationDeviceService
	core    *core.Core
}

// NewWorkstationDeviceHandler 创建工位设备关联处理器实例
func NewWorkstationDeviceHandler(service opsServices.WorkstationDeviceService) *WorkstationDeviceHandler {
	return &WorkstationDeviceHandler{service: service}
}

// WithCore 注入 core 依赖（操作日志记录所需）。Phase 34 操作日志全模块覆盖。
func (h *WorkstationDeviceHandler) WithCore(core *core.Core) *WorkstationDeviceHandler {
	if h != nil {
		h.core = core
	}
	return h
}

// GetByWorkstation 获取工位关联设备列表
// @Summary 获取工位关联设备列表
// @Description 根据工位ID获取该工位关联的所有设备
// @Tags 运维管理
// @Accept json
// @Produce json
// @Param id path string true "工位ID"
// @Success 200 {object} response.Response{data=[]models.WorkstationDevice}
// @Failure 400 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /ops/workstation-device/{id} [post]
func (h *WorkstationDeviceHandler) GetByWorkstation(c *gin.Context) {
	workstationID := c.Param("id")
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

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "工位设备", operlog.OperTypeCreate)
	response.Success(c, device)
}

// GetADDevices 实时查询域控设备
// @Summary 实时查询工位关联的域控设备
// @Description 通过工位绑定的用户实时查询其域控管理的设备（不保存到数据库）
// @Tags 运维管理
// @Accept json
// @Produce json
// @Param id path string true "工位ID"
// @Success 200 {object} response.Response{data=[]models.WorkstationDevice}
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /ops/workstation-device/{id}/ad [post]
func (h *WorkstationDeviceHandler) GetADDevices(c *gin.Context) {
	workstationID := c.Param("id")
	if workstationID == "" {
		response.Error(c, apperrors.ParamMissing("工位ID"))
		return
	}

	devices, err := h.service.GetADDevices(c.Request.Context(), workstationID)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, devices)
}

// GetAssetDevices 实时查询资产设备
// @Summary 实时查询工位关联的资产设备
// @Description 通过工位绑定的用户实时查询其名下的资产设备（不保存到数据库）
// @Tags 运维管理
// @Accept json
// @Produce json
// @Param id path string true "工位ID"
// @Success 200 {object} response.Response{data=[]models.WorkstationDevice}
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /ops/workstation-device/{id}/asset [post]
func (h *WorkstationDeviceHandler) GetAssetDevices(c *gin.Context) {
	workstationID := c.Param("id")
	if workstationID == "" {
		response.Error(c, apperrors.ParamMissing("工位ID"))
		return
	}

	devices, err := h.service.GetAssetDevices(c.Request.Context(), workstationID)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, devices)
}

// GetPhysicalDevices 实时查询物理链路设备
// @Summary 实时查询工位关联的物理链路设备
// @Description 通过工位绑定的信息点反查物理链路 MAC→port→infoPoint→workstation
// 解析出该工位物理接入的设备（与工位是否绑定用户无关），资产字段与域控字段合并时资产优先
// （不保存到数据库）。
// @Tags 运维管理
// @Accept json
// @Produce json
// @Param id path string true "工位ID"
// @Success 200 {object} response.Response{data=[]models.WorkstationDevice}
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /ops/workstation-device/{id}/physical [post]
func (h *WorkstationDeviceHandler) GetPhysicalDevices(c *gin.Context) {
	workstationID := c.Param("id")
	if workstationID == "" {
		response.Error(c, apperrors.ParamMissing("工位ID"))
		return
	}

	devices, err := h.service.GetPhysicalDevices(c.Request.Context(), workstationID)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, devices)
}

// SetPrimaryAndSave 设置主设备并保存
// @Summary 将 AD/资产设备设为主设备并保存为手动设备
// @Description 用于将实时查询到的 AD 或资产设备持久化到数据库并标记为主设备。
// 后端在保存前会以 device_serial 为键实时拉取 AD 与资产两侧设备列表并按字段优先级
// 合并（deviceName 优先 AD；deviceModel/deviceType/responsibleUser 优先资产；
// macAddress 优先 AD 再资产再请求；ipAddress 优先 AD），并清理该工位下旧的
// AD/资产来源记录，最终写入 IsPrimary=true 的 manual 记录。
// @Tags 运维管理
// @Accept json
// @Produce json
// @Param id path string true "设备临时ID（ad-{index} 或 asset-{index}，仅用于前端标识）"
// @Param request body opsServices.SetPrimaryAndSaveRequest true "主设备信息"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /ops/workstation-device/{id}/set-primary-and-save [post]
func (h *WorkstationDeviceHandler) SetPrimaryAndSave(c *gin.Context) {
	var req opsServices.SetPrimaryAndSaveRequest
	if !handleJSONBinding(c, &req) {
		return
	}

	if err := h.service.SetPrimaryAndSave(c.Request.Context(), c.Param("id"), &req); err != nil {
		response.Error(c, err)
		return
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "工位设备", operlog.OperTypeUpdate)
	response.Success(c, gin.H{"message": "设置成功"})
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

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "工位设备", operlog.OperTypeSync)
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

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "工位设备", operlog.OperTypeSync)
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

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "工位设备", operlog.OperTypeUpdate)
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

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "工位设备", operlog.OperTypeDelete)
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

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "工位设备", operlog.OperTypeGrant)
	response.Success(c, nil)
}
