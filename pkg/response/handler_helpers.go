package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// HandleJSONBinding 统一处理 JSON 绑定
// 返回 true 表示绑定成功，false 表示绑定失败
func HandleJSONBinding(c *gin.Context, obj interface{}) bool {
	if err := c.ShouldBindJSON(obj); err != nil {
		Error(c, http.StatusBadRequest, "请求参数错误: "+err.Error())
		return false
	}
	return true
}

// HandleServiceError 统一处理服务层错误
// 返回 true 表示没有错误，false 表示有错误
func HandleServiceError(c *gin.Context, err error, operation string) bool {
	if err != nil {
		Error(c, http.StatusInternalServerError, operation+"失败: "+err.Error())
		return false
	}
	return true
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
