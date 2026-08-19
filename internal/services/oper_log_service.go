package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/utils/operlog"
	"gorm.io/gorm"
)

// OperLogService 操作日志服务接口
type OperLogService interface {
	// RecordOperLog 记录操作日志
	RecordOperLog(ctx context.Context, db *gorm.DB, operLog *models.OperLog) error
	// RecordFromGinContext 从 Gin 上下文记录操作日志
	RecordFromGinContext(c *gin.Context, db *gorm.DB, title string, businessType int, method string)
	// RecordAsync 异步记录操作日志（简化版）
	RecordAsync(db *gorm.DB, title string, businessType int, method, requestMethod, operUrl string,
		operatorName, operatorNickname, deptName *string, operIP *string, operParam, jsonResult, errorMsg *string, status int, costTime int64)
}

// operLogService 操作日志服务实现
type operLogService struct{}

// NewOperLogService 创建操作日志服务
func NewOperLogService() OperLogService {
	return &operLogService{}
}

// RecordOperLog 记录操作日志
func (s *operLogService) RecordOperLog(ctx context.Context, db *gorm.DB, operLog *models.OperLog) error {
	if err := db.Create(operLog).Error; err != nil {
		return fmt.Errorf("记录操作日志失败: %w", err)
	}
	return nil
}

// RecordAsync 异步记录操作日志（简化版，供handler直接调用）
func (s *operLogService) RecordAsync(db *gorm.DB, title string, businessType int, method, requestMethod, operUrl string,
	operatorName, operatorNickname, deptName *string, operIP *string, operParam, jsonResult, errorMsg *string, status int, costTime int64) {

	operLog := &models.OperLog{
		Title:         title,
		BusinessType:  businessType,
		Method:        method,
		RequestMethod: requestMethod,
		OperatorType:  1, // 1=后台用户
		OperatorName:  operatorName,
		Nickname:      operatorNickname,
		DeptName:      deptName,
		OperUrl:       &operUrl,
		OperIP:        operIP,
		OperLocation:  nil, // 可以添加 IP 地理位置解析
		OperParam:     operParam,
		JsonResult:    jsonResult,
		Status:        status,
		ErrorMsg:      errorMsg,
		OperTime:      time.Now(),
		CostTime:      costTime,
	}

	// 异步写入数据库，不影响响应速度
	go func() {
		if err := db.Create(operLog).Error; err != nil {
			// 静默处理日志记录失败
			_ = err
		}
	}()
}

// RecordFromGinContext 从 Gin 上下文记录操作日志
func (s *operLogService) RecordFromGinContext(c *gin.Context, db *gorm.DB, title string, businessType int, method string) {
	// 获取当前用户信息
	username := "unknown"
	operatorName := &username
	deptName := (*string)(nil)

	// 从 JWT claims 中获取用户信息
	if claims, exists := c.Get("claims"); exists {
		if claimData, ok := claims.(map[string]interface{}); ok {
			if uname, ok := claimData["username"].(string); ok {
				username = uname
				operatorName = &username
			}
		}
	}

	// 获取请求信息
	requestMethod := c.Request.Method
	operUrl := c.Request.URL.String()
	operIP := c.ClientIP()

	// 获取请求参数（排除敏感信息）
	operParam := s.getRequestParams(c)

	// 获取响应结果
	jsonResult := s.getResponseResult(c)

	// 计算耗时
	costTime := int64(0)
	if startTime, exists := c.Get("start_time"); exists {
		if start, ok := startTime.(time.Time); ok {
			costTime = time.Since(start).Milliseconds()
		}
	}

	// 确定状态
	status := models.OperLogStatusSuccess
	errorMsg := (*string)(nil)
	if len(c.Errors) > 0 {
		status = models.OperLogStatusFailure
		errMsg := c.Errors.String()
		errorMsg = &errMsg
	}

	operLog := &models.OperLog{
		Title:         title,
		BusinessType:  businessType,
		Method:        method,
		RequestMethod: requestMethod,
		OperatorType:  1, // 1=后台用户
		OperatorName:  operatorName,
		DeptName:      deptName,
		OperUrl:       &operUrl,
		OperIP:        &operIP,
		OperLocation:  nil, // 可以添加 IP 地理位置解析
		OperParam:     operParam,
		JsonResult:    jsonResult,
		Status:        int(status),
		ErrorMsg:      errorMsg,
		OperTime:      time.Now(),
		CostTime:      costTime,
	}

	// 异步写入数据库，不影响响应速度
	go func() {
		if err := s.RecordOperLog(context.Background(), db, operLog); err != nil {
			// 静默处理日志记录失败
			_ = err
		}
	}()
}

// getRequestParams 获取请求参数（排除敏感信息）
func (s *operLogService) getRequestParams(c *gin.Context) *string {
	// 只记录 POST/PUT/DELETE 请求的参数
	if c.Request.Method != "POST" && c.Request.Method != "PUT" && c.Request.Method != "DELETE" {
		return nil
	}

	// 读取查询参数
	queryParams := c.Request.URL.Query()
	if len(queryParams) > 0 {
		params := make(map[string]string)
		for key, values := range queryParams {
			if len(values) > 0 {
				params[key] = values[0]
			}
		}
		if jsonBytes, err := json.Marshal(params); err == nil {
			result := string(jsonBytes)
			return &result
		}
	}

	return nil
}

// getResponseResult 获取响应结果
func (s *operLogService) getResponseResult(c *gin.Context) *string {
	// 只对成功的请求记录响应结果，且限制大小
	if c.Writer.Status() >= 400 {
		return nil
	}

	// 获取响应体（如果设置了）
	if responseBody, exists := c.Get("response_body"); exists {
		if body, ok := responseBody.(string); ok {
			// 限制响应体大小
			if len(body) > 2000 {
				body = body[:2000] + "...（省略）"
			}
			return &body
		}
	}

	return nil
}

// FilterSensitiveParams 过滤敏感参数。Delegates to the shared
// internal/utils/operlog.FilterSensitiveParams implementation which masks a
// 17-keyword case-insensitive set and replaces every occurrence per keyword.
func (s *operLogService) FilterSensitiveParams(params string) string {
	return operlog.FilterSensitiveParams(params)
}
