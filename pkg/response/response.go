package response

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	apperrors "github.com/xingran-next/xingran-go-backend/pkg/errors"
)

const (
	successCode               = 0
	successMessage            = "success"
	defaultServerErrorCode    = 500
	defaultServerErrorMessage = "服务器内部错误"
)

// Response 统一响应结构
type Response struct {
	Code      int         `json:"code"`       // 业务状态码
	Message   string      `json:"message"`    // 消息
	Data      interface{} `json:"data"`       // 数据
	Timestamp int64       `json:"timestamp"`  // 时间戳
	RequestID string      `json:"request_id"` // 请求ID
}

// PageResponse 分页响应结构
type PageResponse struct {
	List     interface{} `json:"list"`     // 数据列表
	Total    int64       `json:"total"`    // 总数
	Current  int         `json:"current"`  // 当前页
	PageSize int         `json:"pageSize"` // 每页大小
}

// 错误定义
var (
	ErrSuccess            = &AppError{Code: 0, Message: "成功"}
	ErrServerError        = &AppError{Code: 500, Message: "服务器内部错误", HTTPStatus: 500}
	ErrBadRequest         = &AppError{Code: 400, Message: "请求参数错误", HTTPStatus: 400}
	ErrUnauthorized       = &AppError{Code: 401, Message: "未授权", HTTPStatus: 401}
	ErrForbidden          = &AppError{Code: 403, Message: "禁止访问", HTTPStatus: 403}
	ErrNotFound           = &AppError{Code: 404, Message: "资源不存在", HTTPStatus: 404}
	ErrMethodNotAllowed   = &AppError{Code: 405, Message: "请求方法不允许", HTTPStatus: 405}
	ErrTokenExpired       = &AppError{Code: 1006, Message: "令牌已过期", HTTPStatus: 401}
	ErrTokenInvalid       = &AppError{Code: 1007, Message: "令牌无效", HTTPStatus: 401}
	ErrTokenNotValidYet   = &AppError{Code: 1008, Message: "令牌尚未生效", HTTPStatus: 401}
	ErrUserNotFound       = &AppError{Code: 1002, Message: "用户不存在", HTTPStatus: 404}
	ErrPasswordError      = &AppError{Code: 1003, Message: "密码错误", HTTPStatus: 401}
	ErrUserDisabled       = &AppError{Code: 1004, Message: "用户已禁用", HTTPStatus: 401}
	ErrCaptchaError       = &AppError{Code: 1005, Message: "验证码错误", HTTPStatus: 400}
	ErrParamError         = &AppError{Code: 1001, Message: "参数错误", HTTPStatus: 400}
	ErrRecordExists       = &AppError{Code: 1009, Message: "记录已存在", HTTPStatus: 400}
	ErrRecordNotFound     = &AppError{Code: 1010, Message: "记录不存在", HTTPStatus: 404}
	ErrNotImplemented     = &AppError{Code: 501, Message: "功能暂未实现", HTTPStatus: 501}
	ErrPasswordHashFailed = &AppError{Code: 1011, Message: "密码哈希格式不支持", HTTPStatus: 500}
	ErrPasswordDecrypt    = &AppError{Code: 1012, Message: "密码解析失败", HTTPStatus: 500}
	ErrCredentialInvalid  = &AppError{Code: 1013, Message: "用户名或密码错误", HTTPStatus: 401}
	ErrDatabaseError      = &AppError{Code: 1014, Message: "数据库操作失败", HTTPStatus: 500}
)

// AppError 应用错误
type AppError struct {
	Code       int    `json:"code"`        // 业务错误码
	Message    string `json:"message"`     // 错误信息
	HTTPStatus int    `json:"http_status"` // HTTP状态码
}

// Error 实现error接口
func (e *AppError) Error() string {
	return e.Message
}

// Success 成功响应
func Success(c *gin.Context, data ...interface{}) {
	responseData := getOptionalData(data)

	c.JSON(http.StatusOK, Response{
		Code:      successCode,
		Message:   successMessage,
		Data:      responseData,
		Timestamp: time.Now().Unix(),
		RequestID: getRequestID(c),
	})
}

// Error 错误响应
func Error(c *gin.Context, err interface{}, message ...string) {
	appErr := toAppError(err)

	// 如果提供了自定义消息，覆盖默认消息
	if len(message) > 0 {
		appErr.Message = message[0]
	}

	c.JSON(appErr.HTTPStatus, Response{
		Code:      appErr.Code,
		Message:   appErr.Message,
		Data:      nil,
		Timestamp: time.Now().Unix(),
		RequestID: getRequestID(c),
	})
}

// ErrorWithData 错误响应（带数据）
func ErrorWithData(c *gin.Context, err interface{}, data interface{}, message ...string) {
	appErr := toAppError(err)

	// 如果提供了自定义消息，覆盖默认消息
	if len(message) > 0 {
		appErr.Message = message[0]
	}

	c.JSON(appErr.HTTPStatus, Response{
		Code:      appErr.Code,
		Message:   appErr.Message,
		Data:      data,
		Timestamp: time.Now().Unix(),
		RequestID: getRequestID(c),
	})
}

// Page 分页响应
func Page(c *gin.Context, list interface{}, total int64, current, pageSize int) {
	c.JSON(http.StatusOK, Response{
		Code:    successCode,
		Message: successMessage,
		Data: PageResponse{
			List:     list,
			Total:    total,
			Current:  current,
			PageSize: pageSize,
		},
		Timestamp: time.Now().Unix(),
		RequestID: getRequestID(c),
	})
}

// toAppError 转换为AppError
func toAppError(err interface{}) *AppError {
	switch e := err.(type) {
	case *AppError:
		return e
	case *apperrors.AppError:
		// 新的错误类型转换
		return &AppError{
			Code:       int(e.Code),
			Message:    e.Message,
			HTTPStatus: e.GetHTTPStatus(),
		}
	case apperrors.ErrorCode:
		// 直接使用错误码
		return &AppError{
			Code:       int(e),
			Message:    e.DefaultMessage(),
			HTTPStatus: e.DefaultHTTPStatus(),
		}
	case int:
		return &AppError{
			Code:       e,
			Message:    "操作失败",
			HTTPStatus: http.StatusBadRequest,
		}
	case string:
		return &AppError{
			Code:       defaultServerErrorCode,
			Message:    e,
			HTTPStatus: http.StatusInternalServerError,
		}
	case error:
		// 检查是否是新的AppError类型
		if appErr := apperrors.GetAppError(e); appErr != nil {
			return &AppError{
				Code:       int(appErr.Code),
				Message:    appErr.Message,
				HTTPStatus: appErr.GetHTTPStatus(),
			}
		}
		// 普通错误
		return &AppError{
			Code:       defaultServerErrorCode,
			Message:    e.Error(),
			HTTPStatus: http.StatusInternalServerError,
		}
	default:
		return ErrServerError
	}
}

// getOptionalData 获取可选数据参数
func getOptionalData(data []interface{}) interface{} {
	if len(data) > 0 {
		return data[0]
	}
	return nil
}

// getRequestID 从上下文获取请求ID
func getRequestID(c *gin.Context) string {
	if requestID, exists := c.Get("request_id"); exists {
		if str, ok := requestID.(string); ok {
			return str
		}
	}
	return ""
}
