package v1

import (
	"fmt"
	"os"
	"runtime"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/core"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/pkg/system"
	"github.com/ruoyi-next/ruoyi-go-backend/pkg/response"
)

// ServerInfoRequest 服务器信息请求
type ServerInfoRequest struct {
	Current  int    `json:"current"`
	PageSize int    `json:"pageSize"`
	HostName string `json:"hostName,omitempty"`
	Status   *int   `json:"status,omitempty"`
}

// SystemMetricsRequest 系统指标请求
type SystemMetricsRequest struct {
	ServerID  string    `json:"serverId,omitempty"`
	StartTime *string   `json:"startTime,omitempty"`
	EndTime   *string   `json:"endTime,omitempty"`
	Current   int       `json:"current"`
	PageSize  int       `json:"pageSize"`
}

// OperLogRequest 操作日志请求
type OperLogRequest struct {
	Title       string `json:"title,omitempty"`
	BusinessType *int  `json:"businessType,omitempty"`
	Status      *int   `json:"status,omitempty"`
	OperName    string `json:"operName,omitempty"`
	StartTime   *string `json:"startTime,omitempty"`
	EndTime     *string `json:"endTime,omitempty"`
	Current     int    `json:"current"`
	PageSize    int    `json:"pageSize"`
}

// LoginLogRequest 登录日志请求
type LoginLogRequest struct {
	UserName string `json:"userName,omitempty"`
	IPAddr   string `json:"ipAddr,omitempty"`
	Status   *int   `json:"status,omitempty"`
	StartTime *string `json:"startTime,omitempty"`
	EndTime   *string `json:"endTime,omitempty"`
	Current  int    `json:"current"`
	PageSize int    `json:"pageSize"`
}

// getServerInfo 获取服务器信息
// @Summary 获取服务器信息
// @Description 获取服务器基本信息和当前状态（优先从缓存获取）
// @Tags 系统监控
// @Accept json
// @Produce json
// @Param request body ServerInfoRequest false "查询参数"
// @Success 200 {object} response.Response{data=[]models.ServerInfo}
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /monitor/server-info/list [post]
func getServerInfo(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req ServerInfoRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			req.Current = 1
			req.PageSize = 10
		}

		// 设置默认值
		if req.Current <= 0 {
			req.Current = 1
		}
		if req.PageSize <= 0 {
			req.PageSize = 10
		}

		// 优先从缓存服务获取服务器信息
		var currentServer *models.ServerInfo
		var err error

		// 尝试从缓存获取服务器信息
		if core.MetricsCacheService != nil {
			info, cacheErr := core.MetricsCacheService.GetServerInfo(c.Request.Context())
			if cacheErr == nil && info != nil {
				// 将缓存信息转换为 ServerInfo 结构
				currentServer = &models.ServerInfo{
					HostName:     info["hostname"].(string),
					OS:           info["os"].(string),
					Arch:         info["arch"].(string),
					CPUCount:     int(info["cpu_count"].(float64)),
					TotalMemory:  uint64(info["total_memory"].(float64)),
					DiskTotal:    uint64(info["disk_total"].(float64)),
					DiskAvailable: uint64(info["disk_available"].(float64)),
					Status:       0,
				}
				// 设置可用内存（如果有的话）
				if availMem, ok := info["available_memory"].(float64); ok {
					currentServer.AvailableMemory = uint64(availMem)
				}
				// 设置最后活跃时间
				now := time.Now()
				currentServer.LastActiveAt = &now
			}
		}

		// 如果缓存没有数据，降级到实时获取
		if currentServer == nil {
			currentServer, err = getCurrentServerInfo()
			if err != nil {
				response.Error(c, response.ErrServerError, "获取服务器信息失败: "+err.Error())
				return
			}
		}

		// 直接返回当前服务器的信息（来自缓存或实时）
		servers := []models.ServerInfo{*currentServer}
		total := int64(1)

		response.Page(c, servers, total, req.Current, req.PageSize)
	}
}

// getCurrentServerMetrics 获取当前服务器指标
// @Summary 获取当前服务器指标
// @Description 获取当前服务器的实时性能指标（优先从缓存获取）
// @Tags 系统监控
// @Accept json
// @Produce json
// @Success 200 {object} response.Response{data=map[string]interface{}}
// @Failure 500 {object} response.Response
// @Router /monitor/server-metrics/current [post]
func getCurrentServerMetrics(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 优先从缓存服务获取系统指标
		if core.MetricsCacheService != nil {
			metrics, err := core.MetricsCacheService.GetCurrentMetrics(c.Request.Context())
			if err != nil {
				response.Error(c, response.ErrServerError, "获取系统指标失败: "+err.Error())
				return
			}

			// 转换为响应格式
			responseData := map[string]interface{}{
				"cpuUsage":     metrics.CPUUsage,
				"memoryUsage":  metrics.MemoryUsage,
				"diskUsage":    metrics.DiskUsage,
				"networkRx":    metrics.NetworkRx,
				"networkTx":    metrics.NetworkTx,
				"processCount": metrics.ProcessNum,
				"totalMemory":  metrics.TotalMemory,
				"usedMemory":   metrics.UsedMemory,
				"timestamp":    metrics.Timestamp,
				"fromCache":    true, // 标识数据来自缓存
			}

			response.Success(c, responseData)
			return
		}

		// 降级到实时获取
		metrics, err := system.GetSystemMetrics()
		if err != nil {
			response.Error(c, response.ErrServerError, "获取系统指标失败: "+err.Error())
			return
		}

		// 转换为响应格式
		responseData := map[string]interface{}{
			"cpuUsage":     metrics.CPUUsage,
			"memoryUsage":  metrics.MemoryUsage,
			"diskUsage":    metrics.DiskUsage,
			"networkRx":    metrics.NetworkRx,
			"networkTx":    metrics.NetworkTx,
			"processCount": metrics.ProcessNum,
			"totalMemory":  metrics.TotalMemory,
			"usedMemory":   metrics.UsedMemory,
			"timestamp":    metrics.Timestamp,
			"fromCache":    false, // 标识数据来自实时计算
		}

		response.Success(c, responseData)
	}
}

// saveSystemMetrics 保存系统指标
// @Summary 保存系统指标
// @Description 保存系统性能指标数据
// @Tags 系统监控
// @Accept json
// @Produce json
// @Param request body object true "指标数据"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /monitor/server-metrics/save [post]
func saveSystemMetrics(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			ServerID string  `json:"serverId"`
			CPUUsage float64 `json:"cpuUsage"`
			MemoryUsage float64 `json:"memoryUsage"`
			DiskUsage float64 `json:"diskUsage"`
			NetworkRx uint64 `json:"networkRx"`
			NetworkTx uint64 `json:"networkTx"`
			ProcessCount int `json:"processCount"`
			LoadAverage float64 `json:"loadAverage"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			response.Error(c, response.ErrBadRequest, "请求参数错误")
			return
		}

		metrics := models.SystemMetrics{
			ServerID:     req.ServerID,
			CPUUsage:     req.CPUUsage,
			MemoryUsage:  req.MemoryUsage,
			DiskUsage:    req.DiskUsage,
			NetworkRx:    req.NetworkRx,
			NetworkTx:    req.NetworkTx,
			ProcessCount: req.ProcessCount,
			LoadAverage:  req.LoadAverage,
			Timestamp:    time.Now(),
		}

		if err := core.DB.GetDB().Create(&metrics).Error; err != nil {
			response.Error(c, response.ErrServerError, "保存系统指标失败")
			return
		}

		response.Success(c, gin.H{"message": "保存成功"})
	}
}

// getSystemMetricsHistory 获取系统指标历史
// @Summary 获取系统指标历史
// @Description 获取指定时间范围内的系统性能指标历史数据
// @Tags 系统监控
// @Accept json
// @Produce json
// @Param request body SystemMetricsRequest false "查询参数"
// @Success 200 {object} response.Response{data=[]models.SystemMetrics}
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /monitor/server-metrics/history [post]
func getSystemMetricsHistory(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req SystemMetricsRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			req.Current = 1
			req.PageSize = 100
		}

		// 设置默认值
		if req.Current <= 0 {
			req.Current = 1
		}
		if req.PageSize <= 0 {
			req.PageSize = 100
		}

		// 构建查询条件
		db := core.DB.GetDB().Model(&models.SystemMetrics{})

		if req.ServerID != "" {
			db = db.Where("server_id = ?", req.ServerID)
		}
		if req.StartTime != nil && *req.StartTime != "" {
			db = db.Where("timestamp >= ?", *req.StartTime)
		}
		if req.EndTime != nil && *req.EndTime != "" {
			db = db.Where("timestamp <= ?", *req.EndTime)
		}

		// 统计总数
		var total int64
		if err := db.Count(&total).Error; err != nil {
			response.Error(c, response.ErrServerError, "查询系统指标总数失败")
			return
		}

		// 查询分页数据
		var metrics []models.SystemMetrics
		offset := (req.Current - 1) * req.PageSize
		if err := db.Offset(offset).Limit(req.PageSize).Order("timestamp DESC").Find(&metrics).Error; err != nil {
			response.Error(c, response.ErrServerError, "查询系统指标失败")
			return
		}

		response.Page(c, metrics, total, req.Current, req.PageSize)
	}
}

// getOperLogs 获取操作日志
// @Summary 获取操作日志
// @Description 分页查询操作日志列表
// @Tags 系统监控
// @Accept json
// @Produce json
// @Param request body OperLogRequest false "查询参数"
// @Success 200 {object} response.Response{data=[]models.OperLog}
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /monitor/oper-logs/list [post]
func getOperLogs(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req OperLogRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			req.Current = 1
			req.PageSize = 10
		}

		// 设置默认值
		if req.Current <= 0 {
			req.Current = 1
		}
		if req.PageSize <= 0 {
			req.PageSize = 10
		}

		// 构建查询条件
		db := core.DB.GetDB().Model(&models.OperLog{})

		if req.Title != "" {
			db = db.Where("title LIKE ?", "%"+req.Title+"%")
		}
		if req.BusinessType != nil {
			db = db.Where("business_type = ?", *req.BusinessType)
		}
		if req.Status != nil {
			db = db.Where("status = ?", *req.Status)
		}
		if req.OperName != "" {
			db = db.Where("oper_name LIKE ?", "%"+req.OperName+"%")
		}
		if req.StartTime != nil && *req.StartTime != "" {
			db = db.Where("oper_time >= ?", *req.StartTime)
		}
		if req.EndTime != nil && *req.EndTime != "" {
			db = db.Where("oper_time <= ?", *req.EndTime)
		}

		// 统计总数
		var total int64
		if err := db.Count(&total).Error; err != nil {
			response.Error(c, response.ErrServerError, "查询操作日志总数失败")
			return
		}

		// 查询分页数据
		var logs []models.OperLog
		offset := (req.Current - 1) * req.PageSize
		if err := db.Offset(offset).Limit(req.PageSize).Order("oper_time DESC").Find(&logs).Error; err != nil {
			response.Error(c, response.ErrServerError, "查询操作日志失败")
			return
		}

		
		response.Page(c, logs, total, req.Current, req.PageSize)
	}
}

// getLoginLogs 获取登录日志
// @Summary 获取登录日志
// @Description 分页查询登录日志列表
// @Tags 系统监控
// @Accept json
// @Produce json
// @Param request body LoginLogRequest false "查询参数"
// @Success 200 {object} response.Response{data=[]models.LoginLog}
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /monitor/login-logs/list [post]
func getLoginLogs(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req LoginLogRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			req.Current = 1
			req.PageSize = 10
		}

		// 设置默认值
		if req.Current <= 0 {
			req.Current = 1
		}
		if req.PageSize <= 0 {
			req.PageSize = 10
		}

		// 构建查询条件
		db := core.DB.GetDB().Model(&models.LoginLog{})

		if req.UserName != "" {
			db = db.Where("user_name LIKE ?", "%"+req.UserName+"%")
		}
		if req.IPAddr != "" {
			db = db.Where("ip_addr LIKE ?", "%"+req.IPAddr+"%")
		}
		if req.Status != nil {
			db = db.Where("status = ?", *req.Status)
		}
		if req.StartTime != nil && *req.StartTime != "" {
			db = db.Where("login_time >= ?", *req.StartTime)
		}
		if req.EndTime != nil && *req.EndTime != "" {
			db = db.Where("login_time <= ?", *req.EndTime)
		}

		// 统计总数
		var total int64
		if err := db.Count(&total).Error; err != nil {
			response.Error(c, response.ErrServerError, "查询登录日志总数失败")
			return
		}

		// 查询分页数据
		var logs []models.LoginLog
		offset := (req.Current - 1) * req.PageSize
		if err := db.Offset(offset).Limit(req.PageSize).Order("login_time DESC").Find(&logs).Error; err != nil {
			response.Error(c, response.ErrServerError, "查询登录日志失败")
			return
		}


		response.Page(c, logs, total, req.Current, req.PageSize)
	}
}

// cleanOperLogs 清空操作日志
// @Summary 清空操作日志
// @Description 清空所有操作日志
// @Tags 系统监控
// @Accept json
// @Produce json
// @Success 200 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /monitor/oper-logs/clean [post]
func cleanOperLogs(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := core.DB.GetDB().Exec("DELETE FROM sys_oper_log").Error; err != nil {
			response.Error(c, response.ErrServerError, "清空操作日志失败")
			return
		}

		response.Success(c, gin.H{"message": "清空成功"})
	}
}

// cleanLoginLogs 清空登录日志
// @Summary 清空登录日志
// @Description 清空所有登录日志
// @Tags 系统监控
// @Accept json
// @Produce json
// @Success 200 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /monitor/login-logs/clean [post]
func cleanLoginLogs(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := core.DB.GetDB().Exec("DELETE FROM sys_login_log").Error; err != nil {
			response.Error(c, response.ErrServerError, "清空登录日志失败")
			return
		}

		response.Success(c, gin.H{"message": "清空成功"})
	}
}

// getCurrentServerInfo 获取当前服务器的实时信息
func getCurrentServerInfo() (*models.ServerInfo, error) {
	// 获取系统指标
	metrics, err := system.GetSystemMetrics()
	if err != nil {
		return nil, err
	}

	// 获取所有磁盘信息，计算总容量（仅使用真实数据）
	var totalDiskSize, availableDiskSize uint64
	disks, err := system.GetAllDiskInfo()
	if err != nil {
		return nil, fmt.Errorf("获取磁盘信息失败: %w", err)
	}

	if len(disks) == 0 {
		return nil, fmt.Errorf("未找到任何磁盘信息")
	}

	for _, disk := range disks {
		// 数据验证：确保总容量大于0，可用容量不超过总容量
		if disk.Total > 0 && disk.Available <= disk.Total {
			totalDiskSize += disk.Total
			availableDiskSize += disk.Available
		}
	}

	// 最终数据验证
	if totalDiskSize == 0 {
		return nil, fmt.Errorf("获取到的磁盘总容量为0")
	}
	if availableDiskSize > totalDiskSize {
		return nil, fmt.Errorf("磁盘可用容量超过总容量")
	}

	// 获取主机信息
	hostname := getHostname()
	os := getOSInfo()
	arch := getArchInfo()

	// 构建服务器信息
	// 正确赋值内存和磁盘信息
	usedMemory := metrics.UsedMemory
	availableMemory := metrics.TotalMemory - usedMemory

	serverInfo := &models.ServerInfo{
		HostName:        hostname,
		OS:              os,
		Arch:            arch,
		CPUCount:        runtime.NumCPU(),
		TotalMemory:     metrics.TotalMemory,   // 总内存
		AvailableMemory: availableMemory,       // 可用内存
		DiskTotal:       totalDiskSize,         // 总磁盘容量
		DiskAvailable:   availableDiskSize,     // 可用磁盘容量
		Status:          0, // 正常状态
		LastActiveAt:    &metrics.Timestamp,
	}

	return serverInfo, nil
}

// getCurrentDiskPath 获取当前磁盘路径
func getCurrentDiskPath() string {
	// Windows和Unix-like系统的根目录不同
	if isWindows() {
		return "C:" // Windows系统盘
	}
	return "/" // Unix-like系统根目录
}

// getDiskInfo 获取磁盘详细信息（总容量和可用容量）
func getDiskInfo(path string) (uint64, uint64, error) {
	return system.GetDiskInfoDetailed(path)
}

// getHostname 获取主机名
func getHostname() string {
	if hostname, err := os.Hostname(); err == nil {
		return hostname
	}
	return "localhost"
}

// getOSInfo 获取操作系统信息
func getOSInfo() string {
	return runtime.GOOS
}

// getArchInfo 获取系统架构信息
func getArchInfo() string {
	return runtime.GOARCH
}

// isWindows 判断是否为Windows系统
func isWindows() bool {
	return runtime.GOOS == "windows"
}