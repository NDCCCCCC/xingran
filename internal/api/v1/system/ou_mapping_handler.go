package system

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/xingran-next/xingran-go-backend/internal/core"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/services/addomain"
	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
	"github.com/xingran-next/xingran-go-backend/pkg/response"
	"gorm.io/gorm"
)

type OUMappingHandler struct {
	db     *gorm.DB
	mapper *addomain.DeptOUmapper
	core   *core.Core
}

func NewOUMappingHandler(db *gorm.DB, mapper *addomain.DeptOUmapper) *OUMappingHandler {
	return &OUMappingHandler{
		db:     db,
		mapper: mapper,
	}
}

// WithCore 注入 core 依赖（用于操作日志埋点），链式调用。Phase 34 Wave 7 新增。
func (h *OUMappingHandler) WithCore(core *core.Core) *OUMappingHandler {
	if h != nil {
		h.core = core
	}
	return h
}

// DeptMappingResponse 部门映射响应
type DeptMappingResponse struct {
	DeptID      string `json:"deptId"`
	DeptName    string `json:"deptName"`
	OUDn        string `json:"ouDn"`
	OUName      string `json:"ouName"`
	SyncEnabled bool   `json:"syncEnabled"`
	SyncStatus  string `json:"syncStatus"`
}

// GetOUDeptMapping 获取 OU 关联的部门信息
func (h *OUMappingHandler) GetOUDeptMapping(c *gin.Context) {
	ouDn := c.Param("ouDn")
	if ouDn == "" {
		response.Error(c, http.StatusBadRequest, "OU DN 不能为空")
		return
	}

	// 查询映射关系
	mapping, err := h.mapper.GetMappingByOU(c.Request.Context(), ouDn)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			// 没有映射关系，返回空结果
			response.Success(c, gin.H{
				"hasMapping": false,
				"message":    "该 OU 尚未关联部门",
			})
			return
		}
		applogger.Errorf("查询 OU 部门映射失败: %v", err)
		response.Error(c, http.StatusInternalServerError, "查询映射关系失败")
		return
	}

	// 查询部门名称
	var dept models.Department
	if err := h.db.WithContext(c.Request.Context()).Select("id, dept_name").Where("id = ?", mapping.DeptID).First(&dept).Error; err != nil {
		applogger.Errorf("查询部门信息失败: %v", err)
		response.Error(c, http.StatusInternalServerError, "查询部门信息失败")
		return
	}

	response.Success(c, gin.H{
		"hasMapping": true,
		"mapping": DeptMappingResponse{
			DeptID:      mapping.DeptID,
			DeptName:    dept.DeptName,
			OUDn:        mapping.OUDN,
			OUName:      mapping.OUName,
			SyncEnabled: mapping.SyncEnabled,
			SyncStatus:  mapping.SyncStatus,
		},
	})
}

// UpdateOUDeptMapping 更新 OU 部门关联
func (h *OUMappingHandler) UpdateOUDeptMapping(c *gin.Context) {
	ouDn := c.Param("ouDn")
	if ouDn == "" {
		response.Error(c, http.StatusBadRequest, "OU DN 不能为空")
		return
	}

	var req struct {
		DeptID string `json:"deptId" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "请求参数错误")
		return
	}

	// 验证部门是否存在
	var dept models.Department
	if err := h.db.WithContext(c.Request.Context()).Where("id = ?", req.DeptID).First(&dept).Error; err != nil {
		response.Error(c, http.StatusBadRequest, "部门不存在")
		return
	}

	// 查询现有的 AD 配置（取第一个启用的配置）
	var adConfig models.ADConfig
	if err := h.db.WithContext(c.Request.Context()).Where("sync_enabled = ? AND status = ?", true, models.ADConfigStatusEnabled).First(&adConfig).Error; err != nil {
		applogger.Errorf("查询 AD 配置失败: %v", err)
		response.Error(c, http.StatusInternalServerError, "查询 AD 配置失败")
		return
	}

	// 获取 OU 名称（从 ouDn 中解析）
	ouName := h.extractOUName(ouDn)
	parentOUDN := h.extractParentOUDN(ouDn)

	// 创建或更新映射关系
	var parentOUDNPtr *string
	if parentOUDN != "" {
		parentOUDNPtr = &parentOUDN
	}

	mapping := &models.DeptOUMapping{
		DeptID:      req.DeptID,
		ADConfigID:  adConfig.ID,
		OUDN:        ouDn,
		OUName:      ouName,
		ParentOUDN:  parentOUDNPtr,
		SyncEnabled: true,
		SyncStatus:  "pending",
	}

	if err := h.mapper.UpsertMapping(c.Request.Context(), mapping); err != nil {
		applogger.Errorf("更新 OU 部门映射失败: %v", err)
		response.Error(c, http.StatusInternalServerError, "更新映射关系失败")
		return
	}

	applogger.Infof("成功更新 OU 部门映射: ou=%s, dept=%s", ouDn, dept.DeptName)
	recordOperLog(c, h.core, "OU部门映射", OperTypeUpdate)
	response.Success(c, gin.H{
		"message": "关联成功",
		"mapping": DeptMappingResponse{
			DeptID:      req.DeptID,
			DeptName:    dept.DeptName,
			OUDn:        ouDn,
			OUName:      ouName,
			SyncEnabled: true,
			SyncStatus:  "pending",
		},
	})
}

// extractOUName 从 OU DN 中提取 OU 名称
// 例如: "OU=基础运维科,OU=科技创新部,DC=cpic,DC=com" -> "基础运维科"
func (h *OUMappingHandler) extractOUName(ouDn string) string {
	// 简单实现：提取第一个 OU= 的值
	// 实际应该使用 LDAP DN 解析库
	for i, r := range ouDn {
		if i > 3 && r == '=' {
			for j := i + 1; j < len(ouDn); j++ {
				if ouDn[j] == ',' || ouDn[j] == '\\' {
					return ouDn[i+1 : j]
				}
			}
			return ouDn[i+1:]
		}
	}
	return ouDn
}

// extractParentOUDN 从 OU DN 中提取父 OU DN
// 例如: "OU=基础运维科,OU=科技创新部,DC=cpic,DC=com" -> "OU=科技创新部,DC=cpic,DC=com"
func (h *OUMappingHandler) extractParentOUDN(ouDn string) string {
	// 查找第一个逗号，返回之后的部分
	for i, r := range ouDn {
		if r == ',' && i > 0 {
			return ouDn[i+2:] // 跳过逗号和可能的空格
		}
	}
	return ""
}
