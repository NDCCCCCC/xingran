package response

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// Response 统一响应结构
type Response struct {
	Code      int         `json:"code"`      // 业务状态码
	Message   string      `json:"message"`   // 消息
	Data      interface{} `json:"data"`      // 数据
	Timestamp int64       `json:"timestamp"` // 时间戳
	RequestID string      `json:"request_id"` // 请求ID
}

// PageResponse 分页响应结构
type PageResponse struct {
	List     interface{} `json:"list"`      // 数据列表
	Total    int64       `json:"total"`     // 总数
	Current  int         `json:"current"`   // 当前页
	PageSize int         `json:"page_size"` // 每页大小
}

// 错误定义
var (
	ErrSuccess           = &AppError{Code: 0, Message: "成功"}
	ErrServerError       = &AppError{Code: 500, Message: "服务器内部错误", HTTPStatus: 500}
	ErrBadRequest        = &AppError{Code: 400, Message: "请求参数错误", HTTPStatus: 400}
	ErrUnauthorized      = &AppError{Code: 401, Message: "未授权", HTTPStatus: 401}
	ErrForbidden         = &AppError{Code: 403, Message: "禁止访问", HTTPStatus: 403}
	ErrNotFound          = &AppError{Code: 404, Message: "资源不存在", HTTPStatus: 404}
	ErrMethodNotAllowed  = &AppError{Code: 405, Message: "请求方法不允许", HTTPStatus: 405}
	ErrTokenExpired      = &AppError{Code: 1006, Message: "令牌已过期", HTTPStatus: 401}
	ErrTokenInvalid      = &AppError{Code: 1007, Message: "令牌无效", HTTPStatus: 401}
	ErrTokenNotValidYet  = &AppError{Code: 1008, Message: "令牌尚未生效", HTTPStatus: 401}
	ErrUserNotFound      = &AppError{Code: 1002, Message: "用户不存在", HTTPStatus: 404}
	ErrPasswordError     = &AppError{Code: 1003, Message: "密码错误", HTTPStatus: 401}
	ErrUserDisabled      = &AppError{Code: 1004, Message: "用户已禁用", HTTPStatus: 401}
	ErrCaptchaError      = &AppError{Code: 1005, Message: "验证码错误", HTTPStatus: 400}
	ErrParamError        = &AppError{Code: 1001, Message: "参数错误", HTTPStatus: 400}
	ErrRecordExists      = &AppError{Code: 1009, Message: "记录已存在", HTTPStatus: 400}
	ErrRecordNotFound    = &AppError{Code: 1010, Message: "记录不存在", HTTPStatus: 404}
)

// AppError 应用错误
type AppError struct {
	Code       int    `json:"code"`       // 业务错误码
	Message    string `json:"message"`    // 错误信息
	HTTPStatus int    `json:"http_status"` // HTTP状态码
}

// Error 实现error接口
func (e *AppError) Error() string {
	return e.Message
}

// Success 成功响应
func Success(c *gin.Context, data ...interface{}) {
	var responseData interface{}
	if len(data) > 0 {
		responseData = data[0]
	}

	requestID, _ := c.Get("request_id")
	c.JSON(http.StatusOK, Response{
		Code:      0,
		Message:   "success",
		Data:      responseData,
		Timestamp: time.Now().Unix(),
		RequestID: toString(requestID),
	})
}

// Error 错误响应
func Error(c *gin.Context, err interface{}, message ...string) {
	var appErr *AppError
	switch e := err.(type) {
	case *AppError:
		appErr = e
	case int:
		appErr = &AppError{Code: e, Message: "操作失败", HTTPStatus: http.StatusBadRequest}
	case string:
		appErr = &AppError{Code: 500, Message: e, HTTPStatus: http.StatusInternalServerError}
	case error:
		appErr = &AppError{Code: 500, Message: e.Error(), HTTPStatus: http.StatusInternalServerError}
	default:
		appErr = ErrServerError
	}

	if len(message) > 0 {
		appErr.Message = message[0]
	}

	requestID, _ := c.Get("request_id")
	c.JSON(appErr.HTTPStatus, Response{
		Code:      appErr.Code,
		Message:   appErr.Message,
		Data:      nil,
		Timestamp: time.Now().Unix(),
		RequestID: toString(requestID),
	})
}

// Page 分页响应
func Page(c *gin.Context, list interface{}, total int64, current, pageSize int) {
	requestID, _ := c.Get("request_id")
	c.JSON(http.StatusOK, Response{
		Code:      0,
		Message:   "success",
		Data: PageResponse{
			List:     list,
			Total:    total,
			Current:  current,
			PageSize: pageSize,
		},
		Timestamp: time.Now().Unix(),
		RequestID: toString(requestID),
	})
}

// toString 转换为字符串
func toString(v interface{}) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}