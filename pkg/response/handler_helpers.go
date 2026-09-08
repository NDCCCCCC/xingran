package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
	apperrors "github.com/xingran-next/xingran-go-backend/pkg/errors"
)

// HandleJSONBinding 统一处理 JSON 绑定
// 返回 true 表示绑定成功，false 表示绑定失败
//
// D-104-3: binding 错误透传 err.Error()（含 gin validator 字段名）
func HandleJSONBinding(c *gin.Context, obj interface{}) bool {
	if err := c.ShouldBindJSON(obj); err != nil {
		// D-104-3: pass through err.Error() — gin validator includes field name
		Error(c, apperrors.Wrap(err, apperrors.CodeParamError, err.Error()))
		return false
	}
	return true
}

// HandleServiceError 统一处理服务层错误
// 返回 true 表示没有错误，false 表示有错误
//
// D-104-4: apperrors.AppError 携带业务码(.Code)，使用其内置业务文案
// D-104-5: 裸内部错误使用 operation+"失败"，不泄露 err.Error() 到前端
func HandleServiceError(c *gin.Context, err error, operation string) bool {
	if err == nil {
		return true
	}
	// D-104-4: apperrors.AppError — use its built-in business message
	if apperrors.IsAppError(err) {
		Error(c, err)
	} else {
		// D-104-5: bare internal error — no err detail leaked to client
		Error(c, err, operation+"失败")
	}
	return false
}

// HandleIDParam 从路径参数中获取 ID
func HandleIDParam(c *gin.Context) (string, bool) {
	id := c.Param("id")
	if id == "" {
		Error(c, http.StatusBadRequest, "缺少 ID 参数")
		return "", false
	}
	return id, true
}

// HandleGetByID 通用的 GetByID 处理逻辑
// 需要提供一个函数来根据 ID 获取实体
func HandleGetByID(c *gin.Context, getter func(string) (interface{}, error), notFoundMessage string) bool {
	id, ok := HandleIDParam(c)
	if !ok {
		return false
	}

	entity, err := getter(id)
	if err != nil {
		Error(c, http.StatusNotFound, notFoundMessage)
		return false
	}

	Success(c, entity)
	return true
}
