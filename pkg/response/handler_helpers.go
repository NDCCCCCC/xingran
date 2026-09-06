package response

import (
	"net/http"
	"time"

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
//
// D-04: BusinessError 携带语义化 HTTP status，业务冲突返回 409/400，
// 不再统一返回 500。
func HandleServiceError(c *gin.Context, err error, operation string) bool {
	if err != nil {
		// D-04: BusinessError carries semantic HTTP status.
		// 直接调用 c.JSON 以保留语义化 HTTPStatus，避免 Error() -> toAppError 把 int 误当 error code。
		if be, ok := err.(*BusinessError); ok {
			now := time.Now().Unix()
			var requestID string
			if rid, ok := c.Get("request_id"); ok {
				if s, ok := rid.(string); ok {
					requestID = s
				}
			}
			// D-04: BusinessError → HTTPStatus (409/400) + 业务码放 Data
			c.JSON(be.HTTPStatus, Response{
				Code:      be.HTTPStatus, // HTTP status 作为 code（success=0，error=HTTP status）
				Message:   operation + "失败: " + be.Message,
				Data:      map[string]int{"bizCode": be.Code}, // 业务码放 Data 里
				RequestID: requestID,
				Timestamp: now,
			})
			return false
		}
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
