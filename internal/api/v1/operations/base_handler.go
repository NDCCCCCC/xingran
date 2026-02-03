package operations

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ruoyi-next/ruoyi-go-backend/pkg/response"
)

// handleJSONBinding 统一处理 JSON 绑定
func handleJSONBinding(c *gin.Context, v interface{}) bool {
	if err := c.ShouldBindJSON(v); err != nil {
		response.Error(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return false
	}
	return true
}

// handleServiceError 统一处理服务层错误
func handleServiceError(c *gin.Context, err error, action string) bool {
	if err != nil {
		response.Error(c, http.StatusInternalServerError, action+"失败: "+err.Error())
		return false
	}
	return true
}
