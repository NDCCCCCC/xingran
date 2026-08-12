package operations

import (
	"github.com/gin-gonic/gin"
	apperrors "github.com/xingran-next/xingran-go-backend/pkg/errors"
	"github.com/xingran-next/xingran-go-backend/pkg/response"
)

// handleJSONBinding 统一处理 JSON 绑定
func handleJSONBinding(c *gin.Context, v interface{}) bool {
	if err := c.ShouldBindJSON(v); err != nil {
		response.Error(c, apperrors.Wrap(err, apperrors.CodeParamError, "参数错误"))
		return false
	}
	return true
}

// handleServiceError 统一处理服务层错误
func handleServiceError(c *gin.Context, err error, action string) bool {
	if err != nil {
		response.Error(c, apperrors.Wrap(err, apperrors.CodeServerError, action+"失败"))
		return false
	}
	return true
}
