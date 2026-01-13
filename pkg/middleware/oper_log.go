package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
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
		},
		ExcludePaths: []string{
			"/list",      // 列表查询不记录
			"/get",       // 获取详情不记录
			"/export",    // 导出操作单独处理
			"/import",    // 导入操作单独处理
			"/download",  // 下载操作不记录
		},
	}
}

// OperLogMiddleware 操作日志中间件
// 自动记录符合条件的操作到数据库
func OperLogMiddleware(operLogService interface{}, core interface{}) gin.HandlerFunc {
	config := DefaultOperLogConfig()

	return func(c *gin.Context) {
		// 记录开始时间
		c.Set("start_time", c.Request.Context().Value("start_time"))

		// 处理请求
		c.Next()

		// 判断是否需要记录操作日志
		if !shouldLogOperation(c.Request.URL.Path, c.Request.Method, config) {
			return
		}

		// 从上下文获取操作日志信息（由各个 handler 设置）
		if title, exists := c.Get("oper_log_title"); exists {
			if businessType, exists := c.Get("oper_log_business_type"); exists {
				// 记录操作日志
				// 这里需要调用 operLogService.RecordFromGinContext
				// 由于类型转换问题，暂时使用反射或接口调用
				_ = title
				_ = businessType
				// TODO: 实现操作日志记录
			}
		}
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
