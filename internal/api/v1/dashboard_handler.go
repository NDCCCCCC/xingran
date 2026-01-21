package v1

import (
	"github.com/gin-gonic/gin"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/core"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/services"
	"github.com/ruoyi-next/ruoyi-go-backend/pkg/response"
)

// ==================== 仪表盘 CRUD 操作 ====================

// getDashboardList 获取仪表盘列表
func getDashboardList(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req services.DashboardListParams
		if err := c.ShouldBindJSON(&req); err != nil {
			response.Error(c, response.ErrParamError)
			return
		}

		svc := services.NewDashboardService(core.DB.GetDB(), core.Cache)
		result, err := svc.GetDashboards(c.Request.Context(), req)
		if err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		response.Success(c, result)
	}
}

// getDashboard 获取仪表盘详情
func getDashboard(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if id == "" {
			response.Error(c, response.ErrParamError)
			return
		}

		svc := services.NewDashboardService(core.DB.GetDB(), core.Cache)
		dashboard, err := svc.GetDashboard(c.Request.Context(), id)
		if err != nil {
			response.Error(c, response.ErrNotFound)
			return
		}

		response.Success(c, gin.H{"dashboard": dashboard})
	}
}

// createDashboard 创建仪表盘
func createDashboard(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists {
			response.Error(c, response.ErrUnauthorized)
			return
		}

		var req services.CreateDashboardRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.Error(c, response.ErrParamError)
			return
		}

		svc := services.NewDashboardService(core.DB.GetDB(), core.Cache)
		dashboard, err := svc.CreateDashboard(c.Request.Context(), userID.(string), req)
		if err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		response.Success(c, gin.H{"dashboard": dashboard})
	}
}

// updateDashboard 更新仪表盘
func updateDashboard(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists {
			response.Error(c, response.ErrUnauthorized)
			return
		}

		id := c.Param("id")
		if id == "" {
			response.Error(c, response.ErrParamError)
			return
		}

		var req services.UpdateDashboardRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.Error(c, response.ErrParamError)
			return
		}

		svc := services.NewDashboardService(core.DB.GetDB(), core.Cache)
		if err := svc.UpdateDashboard(c.Request.Context(), userID.(string), id, req); err != nil {
			if err.Error() == "dashboard not found" {
				response.Error(c, response.ErrNotFound)
				return
			}
			if err.Error() == "permission denied: not the owner" {
				response.Error(c, response.ErrForbidden)
				return
			}
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		response.Success(c, gin.H{"message": "更新成功"})
	}
}

// deleteDashboard 删除仪表盘
func deleteDashboard(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists {
			response.Error(c, response.ErrUnauthorized)
			return
		}

		id := c.Param("id")
		if id == "" {
			response.Error(c, response.ErrParamError)
			return
		}

		svc := services.NewDashboardService(core.DB.GetDB(), core.Cache)
		if err := svc.DeleteDashboard(c.Request.Context(), userID.(string), id); err != nil {
			if err.Error() == "dashboard not found" {
				response.Error(c, response.ErrNotFound)
				return
			}
			if err.Error() == "permission denied: not the owner" {
				response.Error(c, response.ErrForbidden)
				return
			}
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		response.Success(c, gin.H{"message": "删除成功"})
	}
}

// duplicateDashboard 复制仪表盘
func duplicateDashboard(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists {
			response.Error(c, response.ErrUnauthorized)
			return
		}

		id := c.Param("id")
		if id == "" {
			response.Error(c, response.ErrParamError)
			return
		}

		svc := services.NewDashboardService(core.DB.GetDB(), core.Cache)
		dashboard, err := svc.DuplicateDashboard(c.Request.Context(), userID.(string), id)
		if err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		response.Success(c, gin.H{"dashboard": dashboard})
	}
}

// setDefaultDashboard 设置默认仪表盘
func setDefaultDashboard(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists {
			response.Error(c, response.ErrUnauthorized)
			return
		}

		id := c.Param("id")
		if id == "" {
			response.Error(c, response.ErrParamError)
			return
		}

		svc := services.NewDashboardService(core.DB.GetDB(), core.Cache)
		if err := svc.SetDefaultDashboard(c.Request.Context(), userID.(string), id); err != nil {
			if err.Error() == "dashboard not found" {
				response.Error(c, response.ErrNotFound)
				return
			}
			if err.Error() == "permission denied: not the owner" {
				response.Error(c, response.ErrForbidden)
				return
			}
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		response.Success(c, gin.H{"message": "设置成功"})
	}
}

// ==================== 仪表盘模板操作 ====================

// getTemplates 获取仪表盘模板列表
func getTemplates(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Scope *string `json:"scope"`
		}
		c.ShouldBindJSON(&req)

		var scope *models.TemplateScope
		if req.Scope != nil {
			ts := models.TemplateScope(*req.Scope)
			if ts == models.TemplateScopeGlobal || ts == models.TemplateScopeDept || ts == models.TemplateScopePersonal {
				scope = &ts
			}
		}

		svc := services.NewDashboardService(core.DB.GetDB(), core.Cache)
		templates, err := svc.GetTemplates(c.Request.Context(), scope)
		if err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		response.Success(c, gin.H{"templates": templates})
	}
}

// createFromTemplate 从模板创建仪表盘
func createFromTemplate(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists {
			response.Error(c, response.ErrUnauthorized)
			return
		}

		templateID := c.Param("id")
		if templateID == "" {
			response.Error(c, response.ErrParamError)
			return
		}

		var req struct {
			Name string `json:"name" binding:"required,min=1,max=100"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			response.Error(c, response.ErrParamError)
			return
		}

		svc := services.NewDashboardService(core.DB.GetDB(), core.Cache)
		dashboard, err := svc.CreateFromTemplate(c.Request.Context(), userID.(string), templateID, req.Name)
		if err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		response.Success(c, gin.H{"dashboard": dashboard})
	}
}

// ==================== 仪表盘版本操作 ====================

// getVersions 获取仪表盘版本历史
func getVersions(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if id == "" {
			response.Error(c, response.ErrParamError)
			return
		}

		svc := services.NewDashboardService(core.DB.GetDB(), core.Cache)
		versions, err := svc.GetVersions(c.Request.Context(), id)
		if err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		response.Success(c, gin.H{"versions": versions})
	}
}

// createVersion 创建版本快照
func createVersion(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists {
			response.Error(c, response.ErrUnauthorized)
			return
		}

		id := c.Param("id")
		if id == "" {
			response.Error(c, response.ErrParamError)
			return
		}

		var req struct {
			Comment string `json:"comment"`
		}
		c.ShouldBindJSON(&req)

		svc := services.NewDashboardService(core.DB.GetDB(), core.Cache)
		version, err := svc.CreateVersion(c.Request.Context(), userID.(string), id, req.Comment)
		if err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		response.Success(c, gin.H{"version": version})
	}
}

// restoreVersion 从版本恢复仪表盘
func restoreVersion(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists {
			response.Error(c, response.ErrUnauthorized)
			return
		}

		dashboardID := c.Param("id")
		versionID := c.Param("versionId")
		if dashboardID == "" || versionID == "" {
			response.Error(c, response.ErrParamError)
			return
		}

		svc := services.NewDashboardService(core.DB.GetDB(), core.Cache)
		if err := svc.RestoreVersion(c.Request.Context(), userID.(string), dashboardID, versionID); err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		response.Success(c, gin.H{"message": "恢复成功"})
	}
}

// ==================== 仪表盘导入导出 ====================

// exportDashboard 导出仪表盘配置
func exportDashboard(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if id == "" {
			response.Error(c, response.ErrParamError)
			return
		}

		svc := services.NewDashboardService(core.DB.GetDB(), core.Cache)
		config, err := svc.ExportDashboard(c.Request.Context(), id)
		if err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		response.Success(c, gin.H{"config": config})
	}
}

// importDashboard 导入仪表盘配置
func importDashboard(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists {
			response.Error(c, response.ErrUnauthorized)
			return
		}

		var req struct {
			Config string `json:"config" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			response.Error(c, response.ErrParamError)
			return
		}

		svc := services.NewDashboardService(core.DB.GetDB(), core.Cache)
		dashboard, err := svc.ImportDashboard(c.Request.Context(), userID.(string), req.Config)
		if err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		response.Success(c, gin.H{"dashboard": dashboard})
	}
}

// ==================== Widget 数据获取 ====================

// getWidgetData 获取 Widget 数据
func getWidgetData(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		widgetID := c.Param("id")
		if widgetID == "" {
			response.Error(c, response.ErrParamError)
			return
		}

		var req struct {
			Endpoint string                 `json:"endpoint"`
			Params   map[string]interface{} `json:"params"`
		}
		c.ShouldBindJSON(&req)

		svc := services.NewDashboardService(core.DB.GetDB(), core.Cache)
		data, err := svc.GetWidgetData(c.Request.Context(), widgetID, req.Endpoint, req.Params)
		if err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		response.Success(c, gin.H{"data": data})
	}
}

// getBatchWidgetData 批量获取 Widget 数据
func getBatchWidgetData(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			WidgetIds []string `json:"widgetIds" binding:"required,min=1"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			response.Error(c, response.ErrParamError)
			return
		}

		svc := services.NewDashboardService(core.DB.GetDB(), core.Cache)
		data, err := svc.GetBatchWidgetData(c.Request.Context(), req.WidgetIds)
		if err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		response.Success(c, gin.H{"data": data})
	}
}
