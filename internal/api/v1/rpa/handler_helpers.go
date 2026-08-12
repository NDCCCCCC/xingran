package rpa

import (
	"github.com/gin-gonic/gin"
	"github.com/xingran-next/xingran-go-backend/pkg/response"
	"net/http"
)

// bindAndValidate 绑定并验证请求参数
func bindAndValidate(c *gin.Context, v interface{}) bool {
	if err := c.ShouldBindJSON(v); err != nil {
		response.Error(c, http.StatusBadRequest, "请求参数错误")
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
func handleError(c *gin.Context, err error, statusCode int, msg string) bool {
	if err != nil {
		response.Error(c, statusCode, msg+": "+err.Error())
		return true
	}
	return false
}

// setPaginationDefaults 设置分页默认值
func setPaginationDefaults(current, pageSize *int) {
	if *current <= 0 {
		*current = 1
	}
	if *pageSize <= 0 {
		*pageSize = 10
	}
}
