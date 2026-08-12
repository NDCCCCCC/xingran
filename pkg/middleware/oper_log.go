package middleware

import (
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/xingran-next/xingran-go-backend/internal/core"
	"github.com/xingran-next/xingran-go-backend/internal/services"
	"github.com/xingran-next/xingran-go-backend/internal/utils"
)

// OperLogConfig 操作日志配置
type OperLogConfig struct {
	// 需要记录操作日志的路径前缀
	LogPaths []string
	// 不需要记录日志的路径（排除）
	ExcludePaths []string
}

// DefaultOperLogConfig 默认操作日志配置
func DefaultOperLogConfig() *OperLogConfig {
	return &OperLogConfig{
		LogPaths: []string{
			"/system/user",
			"/system/role",
			"/system/menu",
			"/system/dept",
			"/system/post",
			"/system/dict",
			"/system/config",
			"/system/notice",
			"/system/log",
			"/monitor/job",
			"/network/device",
			"/workorder",
			"/knowledge",
			"/ad-domain",
			// 运维管理模块
			"/ops/building",
			"/ops/floor",
			"/ops/workstation",
			"/ops/server-room",
			"/ops/room-device",
			"/ops/dedicated-line",
			"/ops/info-point",
			"/ops/wall",
			"/ops/door",
			"/ops/text",
			// 资产对账模块 (Phase 42 R1)
			"/asset/reconciliation",
		},
		ExcludePaths: []string{
			"/list",     // 列表查询不记录
			"/get",      // 获取详情不记录
			"/export",   // 导出操作单独处理
			"/import",   // 导入操作单独处理
			"/download", // 下载操作不记录
			"/tree",     // 树形结构查询不记录
		},
	}
}

// OperLogMiddleware 操作日志中间件
// 自动记录符合条件的操作到数据库
func OperLogMiddleware(operLogSvc services.OperLogService, core *core.Core) gin.HandlerFunc {
	config := DefaultOperLogConfig()

	return func(c *gin.Context) {
		// 记录开始时间
		startTime := time.Now()
		c.Set("start_time", startTime)

		// 处理请求
		c.Next()

		// 判断是否需要记录操作日志
		if !shouldLogOperation(c.Request.URL.Path, c.Request.Method, config) {
			return
		}

		// 从上下文获取操作日志信息（由各个 handler 设置）
		title, hasTitle := c.Get("oper_log_title")
		businessType, hasBusinessType := c.Get("oper_log_business_type")

		// 如果 handler 没有设置，使用默认值
		if !hasTitle || !hasBusinessType {
			return
		}

		// 获取操作信息
		titleStr, _ := title.(string)
		businessTypeInt, _ := businessType.(int)

		// 构建方法描述
		method := getMethodDescription(c.Request.URL.Path, c.Request.Method)

		// 计算耗时
		costTime := time.Since(startTime).Milliseconds()

		// 获取用户信息
		username := utils.GetUsernamePtr(c)
		deptName := utils.GetDeptNameFromDB(c, core.GetDB())
		clientIP := utils.GetClientIP(c)
		nickname := utils.GetNicknamePtrWithDB(c, core.GetDB())

		// 确定状态和错误信息
		status := 0
		var errorMsg *string
		if len(c.Errors) > 0 {
			status = 1
			errStr := c.Errors.String()
			errorMsg = &errStr
		}

		// 异步记录操作日志
		operLogSvc.RecordAsync(
			core.GetDB(),
			titleStr,
			businessTypeInt,
			method,
			c.Request.Method,
			c.Request.URL.String(),
			username,
		nickname,
			deptName,
			&clientIP,
			nil, // operParam - 可以后续添加请求参数过滤
			nil, // jsonResult - 可以后续添加响应结果
			errorMsg,
			status,
			costTime,
		)
	}
}

// shouldLogOperation 判断是否需要记录操作日志
func shouldLogOperation(path, method string, config *OperLogConfig) bool {
	// 只记录 POST, PUT, DELETE 操作
	if method != "POST" && method != "PUT" && method != "DELETE" {
		return false
	}

	// 检查是否在排除列表中
	for _, excludePath := range config.ExcludePaths {
		if strings.Contains(path, excludePath) {
			return false
		}
	}

	// 检查是否在需要记录的路径列表中
	for _, logPath := range config.LogPaths {
		if strings.HasPrefix(path, logPath) {
			return true
		}
	}

	return false
}

// SetOperLogInfo 设置操作日志信息（供各个 handler 使用）
func SetOperLogInfo(c *gin.Context, title string, businessType int, method string) {
	c.Set("oper_log_title", title)
	c.Set("oper_log_business_type", businessType)
	c.Set("oper_log_method", method)
}

// GetBusinessType 获取业务类型
func GetBusinessType(path, method string) int {
	// 根据路径和方法判断业务类型
	// 0=其它 1=新增 2=修改 3=删除 4=授权 5=导出 6=导入 7=强退 8=生成代码 9=清空数据
	switch {
	case strings.Contains(path, "/add") || strings.Contains(path, "/create"):
		return 1 // 新增
	case strings.Contains(path, "/edit") || strings.Contains(path, "/update"):
		return 2 // 修改
	case strings.Contains(path, "/delete") || strings.Contains(path, "/remove"):
		return 3 // 删除
	case strings.Contains(path, "/export"):
		return 5 // 导出
	case strings.Contains(path, "/import"):
		return 6 // 导入
	default:
		return 0 // 其它
	}
}

// getMethodDescription 获取方法描述
func getMethodDescription(path, method string) string {
	var action string
	switch {
	case strings.Contains(path, "/add") || strings.Contains(path, "/create"):
		action = "新增"
	case strings.Contains(path, "/edit") || strings.Contains(path, "/update"):
		action = "修改"
	case strings.Contains(path, "/delete") || strings.Contains(path, "/remove"):
		action = "删除"
	case strings.Contains(path, "/export"):
		action = "导出"
	case strings.Contains(path, "/import"):
		action = "导入"
	default:
		action = method
	}
	return action
}
