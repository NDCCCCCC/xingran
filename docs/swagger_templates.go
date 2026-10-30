package docs

// swagger_templates.go
// 此文件包含Swagger文档的通用模板和模型定义
// 用于为API处理器提供标准的响应结构和文档注解模板

// 通用Swagger注解模板

// 分页请求模板
// @Param current body int false "当前页码" default(1)
// @Param pageSize body int false "每页大小" default(10)

// 通用成功响应模板
// @Success 200 {object} Response{data=object} "成功响应"

// 通用分页响应模板
// @Success 200 {object} Response{data=PageResponse} "分页成功响应"

// 通用错误响应模板
// @Failure 400 {object} Response "请求参数错误"
// @Failure 401 {object} Response "未授权"
// @Failure 403 {object} Response "禁止访问"
// @Failure 404 {object} Response "资源不存在"
// @Failure 500 {object} Response "服务器内部错误"

// 通用安全注解
// @Security ApiKeyAuth

// 通用请求体注解
// @Accept json
// @Produce json

// Response 通用API响应结构
type Response struct {
	Code      int         `json:"code" example:"0"`                // 业务状态码，0表示成功
	Message   string      `json:"message" example:"success"`       // 响应消息
	Data      interface{} `json:"data"`                            // 响应数据
	Timestamp int64       `json:"timestamp" example:"1766380800"`  // Unix时间戳
	RequestID string      `json:"request_id" example:"uuid-string"` // 请求ID
}

// PageResponse 分页响应结构
type PageResponse struct {
	List     interface{} `json:"list"`     // 数据列表
	Total    int64       `json:"total" example:"100"`    // 总记录数
	Current  int         `json:"current" example:"1"`    // 当前页码
	PageSize int         `json:"pageSize" example:"10"`  // 每页大小
}

// ErrorResponse 错误响应详情
type ErrorResponse struct {
	Code      int    `json:"code" example:"1001"`               // 错误码
	Message   string `json:"message" example:"参数错误"`         // 错误消息
	Timestamp int64  `json:"timestamp" example:"1766380800"`    // Unix时间戳
	RequestID string `json:"request_id" example:"uuid-string"` // 请求ID
}

// PaginationParams 分页请求参数
type PaginationParams struct {
	Current  int `json:"current" example:"1"`   // 当前页码，从1开始
	PageSize int `json:"pageSize" example:"10"` // 每页大小
}

// BatchDeleteRequest 批量删除请求
type BatchDeleteRequest struct {
	IDs []string `json:"ids" binding:"required,min=1" example:"uuid1,uuid2"` // 要删除的ID列表
}

// UpdateStatusRequest 更新状态请求
type UpdateStatusRequest struct {
	Status int `json:"status" binding:"min=0,max=1" example:"0"` // 状态：0=启用/正常，1=禁用/停用
}

// 通用业务错误码定义
// const (
// 	CodeSuccess           = 0     // 成功
// 	CodeBadRequest        = 400   // 请求参数错误
// 	CodeUnauthorized      = 401   // 未授权
// 	CodeForbidden         = 403   // 禁止访问
// 	CodeNotFound          = 404   // 资源不存在
// 	CodeMethodNotAllowed  = 405   // 请求方法不允许
// 	CodeServerError       = 500   // 服务器内部错误
// 	CodeTokenExpired      = 1006  // 令牌已过期
// 	CodeTokenInvalid      = 1007  // 令牌无效
// 	CodeTokenNotValidYet  = 1008  // 令牌尚未生效
// 	CodeUserNotFound      = 1002  // 用户不存在
// 	CodePasswordError     = 1003  // 密码错误
// 	CodeUserDisabled      = 1004  // 用户已禁用
// 	CodeCaptchaError      = 1005  // 验证码错误
// 	CodeParamError        = 1001  // 参数错误
// 	CodeRecordExists      = 1009  // 记录已存在
// 	CodeRecordNotFound    = 1010  // 记录不存在
// 	CodeNotImplemented    = 501   // 功能暂未实现
// 	CodePasswordHashFailed = 1011 // 密码哈希格式不支持
// 	CodePasswordDecrypt   = 1012  // 密码解析失败
// 	CodeCredentialInvalid = 1013  // 用户名或密码错误
// 	CodeDatabaseError     = 1014  // 数据库操作失败
// )
