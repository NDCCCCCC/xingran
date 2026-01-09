package v1

import (
	"github.com/gin-gonic/gin"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/core"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/services"
	"github.com/ruoyi-next/ruoyi-go-backend/pkg/middleware"
	"github.com/ruoyi-next/ruoyi-go-backend/pkg/response"
)

// SetupADDomainRouter 设置AD域管理路由
func SetupADDomainRouter(r *gin.RouterGroup, core *core.Core) {
	// AD配置管理
	adConfigs := r.Group("/configs")
	adConfigs.Use(middleware.RequirePermissions([]string{
		"ops:ad:config:list",
		"ops:ad:config:add",
		"ops:ad:config:edit",
		"ops:ad:config:delete",
		"ops:ad:config:test",
		"ops:ad:config:sync",
	}, core))
	{
		adConfigs.POST("/list", listADConfigs(core))
		adConfigs.POST("", createADConfig(core))
		adConfigs.GET("/:id", getADConfig(core))
		adConfigs.POST("/:id/update", updateADConfig(core))
		adConfigs.POST("/:id/delete", deleteADConfig(core))
		adConfigs.POST("/:id/test", testADConnection(core))
		adConfigs.POST("/:id/sync", syncADData(core))
	}

	// OU管理
	adOUs := r.Group("/ous")
	adOUs.Use(middleware.RequirePermissions([]string{
		"ops:ad:ou:view",
	}, core))
	{
		adOUs.POST("/tree", getADOUTree(core))
	}

	// 用户组管理
	adGroups := r.Group("/groups")
	adGroups.Use(middleware.RequirePermissions([]string{
		"ops:ad:group:view",
		"ops:ad:group:edit",
	}, core))
	{
		adGroups.POST("/list", listADGroups(core))
		adGroups.GET("/:id", getADGroupDetail(core))
		adGroups.POST("/:id/update", updateADGroup(core))
	}

	// 用户管理
	adUsers := r.Group("/users")
	adUsers.Use(middleware.RequirePermissions([]string{
		"ops:ad:user:view",
		"ops:ad:user:edit",
	}, core))
	{
		adUsers.POST("/list", listADUsers(core))
		adUsers.GET("/:id", getADUserDetail(core))
	}

	// 同步日志
	adLogs := r.Group("/logs")
	adLogs.Use(middleware.RequirePermissions([]string{
		"ops:ad:log:view",
	}, core))
	{
		adLogs.POST("/list", listADSyncLogs(core))
	}
}

// ==================== AD配置管理 ====================

// listADConfigs 查询AD配置列表
func listADConfigs(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Status   *int `json:"status,omitempty"`
			Current  int  `json:"current"`
			PageSize int  `json:"pageSize"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			// 允许空请求体
		}

		if req.Current == 0 {
			req.Current = 1
		}
		if req.PageSize == 0 {
			req.PageSize = 10
		}

		service := services.NewADDomainService(core.GetDB())
		list, total, err := service.GetADConfigList(c.Request.Context(), req.Status, req.Current, req.PageSize)
		if err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		response.Success(c, gin.H{
			"list":  list,
			"total": total,
		})
	}
}

// createADConfig 创建AD配置
func createADConfig(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req services.ADConfigCreateRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.Error(c, response.ErrBadRequest, "请求参数错误: "+err.Error())
			return
		}

		// 获取当前用户ID
		creatorID := c.GetString("user_id")

		service := services.NewADDomainService(core.GetDB())
		config, err := service.CreateADConfig(c.Request.Context(), &req, creatorID)
		if err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		// 记录操作日志
		clientIP := c.ClientIP()
		operURL := c.Request.URL.String()
		core.OperLogService.RecordAsync(
			core.GetDB(),
			"AD域配置",
			1, // 1=新增
			"",
			c.Request.Method,
			operURL,
			getUsernameFromContext(c),
			getDeptNameFromContext(c, core.GetDB()),
			&clientIP,
			nil,
			nil,
			nil,
			0,
			0,
		)

		// 隐藏密码
		config.AdminPassword = ""

		response.Success(c, config)
	}
}

// getADConfig 获取AD配置详情
func getADConfig(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")

		service := services.NewADDomainService(core.GetDB())
		config, err := service.GetADConfigByID(c.Request.Context(), id)
		if err != nil {
			response.Error(c, response.ErrNotFound, "AD配置不存在")
			return
		}

		// 隐藏密码
		config.AdminPassword = ""

		response.Success(c, config)
	}
}

// updateADConfig 更新AD配置
func updateADConfig(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")

		var req services.ADConfigUpdateRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.Error(c, response.ErrBadRequest, "请求参数错误: "+err.Error())
			return
		}

		req.ID = id

		// 获取当前用户ID
		updaterID := c.GetString("user_id")

		service := services.NewADDomainService(core.GetDB())
		if err := service.UpdateADConfig(c.Request.Context(), &req, updaterID); err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		// 记录操作日志
		clientIP := c.ClientIP()
		operURL := c.Request.URL.String()
		core.OperLogService.RecordAsync(
			core.GetDB(),
			"AD域配置",
			2, // 2=修改
			"",
			c.Request.Method,
			operURL,
			getUsernameFromContext(c),
			getDeptNameFromContext(c, core.GetDB()),
			&clientIP,
			nil,
			nil,
			nil,
			0,
			0,
		)

		response.Success(c, "更新成功")
	}
}

// deleteADConfig 删除AD配置
func deleteADConfig(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")

		service := services.NewADDomainService(core.GetDB())
		if err := service.DeleteADConfig(c.Request.Context(), id); err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		// 记录操作日志
		clientIP := c.ClientIP()
		operURL := c.Request.URL.String()
		core.OperLogService.RecordAsync(
			core.GetDB(),
			"AD域配置",
			3, // 3=删除
			"",
			c.Request.Method,
			operURL,
			getUsernameFromContext(c),
			getDeptNameFromContext(c, core.GetDB()),
			&clientIP,
			nil,
			nil,
			nil,
			0,
			0,
		)

		response.Success(c, "删除成功")
	}
}

// testADConnection 测试AD连接
func testADConnection(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")

		service := services.NewADDomainService(core.GetDB())
		err := service.TestADConnection(c.Request.Context(), id)
		if err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		response.Success(c, gin.H{
			"success": true,
			"message": "连接成功",
		})
	}
}

// syncADData 同步AD数据
func syncADData(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")

		var req struct {
			SyncType string `json:"syncType"`
		}
		c.ShouldBindJSON(&req)

		if req.SyncType == "" {
			req.SyncType = "full"
		}

		service := services.NewADDomainService(core.GetDB())
		result, err := service.SyncADData(c.Request.Context(), id, req.SyncType)
		if err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		// 记录操作日志
		clientIP := c.ClientIP()
		operURL := c.Request.URL.String()
		core.OperLogService.RecordAsync(
			core.GetDB(),
			"AD域配置",
			0, // 0=其它
			"",
			c.Request.Method,
			operURL,
			getUsernameFromContext(c),
			getDeptNameFromContext(c, core.GetDB()),
			&clientIP,
			nil,
			nil,
			nil,
			0,
			0,
		)

		response.Success(c, result)
	}
}

// ==================== OU管理 ====================

// getADOUTree 获取OU树形结构
func getADOUTree(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			ConfigID string `json:"configId" binding:"required"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			response.Error(c, response.ErrBadRequest, "请求参数错误: "+err.Error())
			return
		}

		service := services.NewADDomainService(core.GetDB())
		tree, err := service.GetOUTree(c.Request.Context(), req.ConfigID)
		if err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		response.Success(c, tree)
	}
}

// ==================== 用户组管理 ====================

// listADGroups 查询AD用户组列表
func listADGroups(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req services.ADGroupListRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			// 允许空请求体，设置默认值
			if req.Current == 0 {
				req.Current = 1
			}
			if req.PageSize == 0 {
				req.PageSize = 10
			}
		}

		service := services.NewADDomainService(core.GetDB())
		list, total, err := service.GetADGroupList(c.Request.Context(), &req)
		if err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		response.Success(c, gin.H{
			"list":  list,
			"total": total,
		})
	}
}

// getADGroupDetail 获取AD用户组详情
func getADGroupDetail(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")

		service := services.NewADDomainService(core.GetDB())
		// 先获取配置列表找到对应的configID，然后使用DN查询
		// 简化实现：直接返回ID
		response.Success(c, gin.H{
			"id": id,
			"message": "功能开发中",
		})
		_ = service
	}
}

// updateADGroup 更新AD用户组
func updateADGroup(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")

		var req struct {
			GroupName   *string `json:"groupName,omitempty"`
			Description *string `json:"description,omitempty"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			response.Error(c, response.ErrBadRequest, "请求参数错误: "+err.Error())
			return
		}

		service := services.NewADDomainService(core.GetDB())
		// 这个方法可能需要在 service 中实现
		response.Error(c, response.ErrNotImplemented, "功能暂未实现")
		_ = id
		_ = service
		_ = req.GroupName
		_ = req.Description
	}
}

// ==================== 用户管理 ====================

// listADUsers 查询AD用户列表
func listADUsers(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req services.ADUserListRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			// 允许空请求体，设置默认值
		}

		service := services.NewADDomainService(core.GetDB())
		list, total, err := service.GetADUserList(c.Request.Context(), &req)
		if err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		response.Success(c, gin.H{
			"list":  list,
			"total": total,
		})
	}
}

// getADUserDetail 获取AD用户详情
func getADUserDetail(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")

		service := services.NewADDomainService(core.GetDB())
		// 简化实现：直接返回ID
		response.Success(c, gin.H{
			"id": id,
			"message": "功能开发中",
		})
		_ = service
	}
}

// ==================== 同步日志 ====================

// listADSyncLogs 查询AD同步日志列表
func listADSyncLogs(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			ConfigID string `json:"configId,omitempty"`
			Current  int    `json:"current"`
			PageSize  int    `json:"pageSize"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			// 允许空请求体
		}

		if req.Current == 0 {
			req.Current = 1
		}
		if req.PageSize == 0 {
			req.PageSize = 10
		}

		service := services.NewADDomainService(core.GetDB())
		list, total, err := service.GetADSyncLogList(c.Request.Context(), req.ConfigID, req.Current, req.PageSize)
		if err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		response.Success(c, gin.H{
			"list":  list,
			"total": total,
		})
	}
}
