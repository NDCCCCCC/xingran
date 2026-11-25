package rpa

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/xingran-next/xingran-go-backend/pkg/query"
	"github.com/xingran-next/xingran-go-backend/pkg/response"
)

// bindAndValidate 绑定并验证请求参数。
//
// CONTRACT-02/03：错误消息必须可读 —— 旧实现固定返回「请求参数错误」，把
// "cannot unmarshal string into ... status of type int" 这类字段级原因吞掉，
// 前端与排障者无从定位。现在把底层绑定错误拼进 message（保留原前缀以兼容
// 既有断言），并追加字段名提示。
func bindAndValidate(c *gin.Context, v interface{}) bool {
	if err := c.ShouldBindJSON(v); err != nil {
		response.Error(c, http.StatusBadRequest, "请求参数错误: "+describeBindError(err))
		return false
	}
	return true
}

// describeBindError 把 encoding/json 的绑定错误翻译成字段级可读中文。
//
// CONTRACT-02 的验收要求「非法枚举 400 且错误可读」：JSON 类型不匹配
// （status:"pending" 传给 *int 形状的字段）发生在 binding 阶段，比 Validate
// 更早。原样透出 "json: cannot unmarshal string into Go struct field
// TaskListParams.status of type rpa.TaskStatus" 虽然含字段名，但对不熟悉 Go
// 类型名的人不可读，故在此追加中文解释。
func describeBindError(err error) string {
	var typeErr *json.UnmarshalTypeError
	if errors.As(err, &typeErr) {
		field := typeErr.Field
		if field == "" {
			field = "(根对象)"
		}
		return fmt.Sprintf("字段 %s 类型不匹配：期望 %s，收到 %s（%s）",
			field, typeErr.Type.String(), typeErr.Value, typeErr.Error())
	}
	return err.Error()
}

// validateListParams 执行 ListParams 的取值域校验（CONTRACT-02）。
//
// 与 bindAndValidate 分离的原因：binding 管「类型/必填」，本函数管「枚举域」。
// 非法枚举（status:"pending" 传给任务列表、status:"x" 传给 Worker 列表等）
// 必须 400 且消息指明合法取值，而不是让 SQL 静默过滤出空列表。
func validateListParams(c *gin.Context, err error) bool {
	if err != nil {
		response.Error(c, http.StatusBadRequest, "请求参数错误: "+err.Error())
		return false
	}
	return true
}

// getIDParam 获取路径参数 ID
func getIDParam(c *gin.Context) string {
	id := c.Param("id")
	if id == "" {
		response.Error(c, http.StatusBadRequest, "ID不能为空")
	}
	return id
}

// success 返回成功响应
func success(c *gin.Context, data interface{}) {
	response.Success(c, data)
}

// successMsg 返回成功消息
func successMsg(c *gin.Context, msg string) {
	response.Success(c, gin.H{"message": msg})
}

// handleError 处理错误
//
// 注意（NOCODE-03 踩坑）：本函数只用于「统一 500」的场景。它内部调用的
// response.Error(c, statusCode, msg) 传 int 时，pkg/response.toAppError 的
// `case int` 分支会把 HTTPStatus 硬编码为 400 —— 即传 500 实际返回 400。
// 需要**精确控制 HTTPStatus**（例如区分「用户可修正的 400」与「系统侧 500」）
// 时必须用 writeRPAError，不能依赖本函数。
func handleError(c *gin.Context, err error, statusCode int, msg string) bool {
	if err != nil {
		response.Error(c, statusCode, msg+": "+err.Error())
		return true
	}
	return false
}

// writeRPAError 写一个可精确指定 HTTP 状态的错误响应。
//
// 与 handleError 的区别：走 *response.AppError 分支（该分支尊重 HTTPStatus），
// 避免 `case int` 分支把状态码硬编码成 400 的陷阱。
//
// NOCODE-03 的三态引导依赖本函数：AI 不可用/非法动作 → 400，超时 → 500，
// 前端据此分流「改用录制」与「稍后重试」两条完全不同的引导路径。
func writeRPAError(c *gin.Context, httpStatus int, message string) {
	response.Error(c, &response.AppError{
		Code:       httpStatus,
		Message:    message,
		HTTPStatus: httpStatus,
	})
}

// setPaginationDefaults 设置分页默认值
func setPaginationDefaults(current, pageSize *int) {
	c, ps := query.NormalizePagination(*current, *pageSize)
	*current = c
	*pageSize = ps
}

// normalizeOptionalStringFilter 把「清空筛选」产生的空串归一化为 nil。
//
// CONTRACT-02：前端 AppSelect allowClear 在用户清空后可能提交 status:""，
// Gin 会把它绑定成指针指向空串（不是 nil）。service 层以 `params.Status != nil`
// 判定是否过滤，于是空串会变成 `WHERE status = ''` 静默返回空列表。
// 归属协议要求不改 worker_service / execution_service 的 List，故在 handler
// 入参侧归一化，保持「空串 = 不过滤」的前端语义。
func normalizeOptionalStringFilter(p *string) *string {
	if p != nil && strings.TrimSpace(*p) == "" {
		return nil
	}
	return p
}

// currentUserID 读取当前登录用户 ID。
//
// CONTRACT-01：JWT 中间件（pkg/middleware/auth.go:setUserContext）写入的键是
// `user_id`，而 RPA 各 handler 历史读的是 `userId` —— 恒取到空串，导致凭证
// 归属、任务 created_by、部门共享过滤全部失真。此处优先读 `user_id`，回落
// `userId`（兼容既有测试与潜在旧中间件）。
func currentUserID(c *gin.Context) string {
	if id := c.GetString("user_id"); id != "" {
		return id
	}
	return c.GetString("userId")
}

// currentDeptID 读取当前登录用户部门 ID（`dept_id`，回落 `deptId`）。
func currentDeptID(c *gin.Context) string {
	if id := c.GetString("dept_id"); id != "" {
		return id
	}
	return c.GetString("deptId")
}
