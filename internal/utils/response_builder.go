package utils

import (
	"github.com/gin-gonic/gin"
)

// BuildSuccessResponse 构建成功响应
func BuildSuccessResponse(data interface{}) gin.H {
	return gin.H{
		"data": data,
	}
}

// BuildListResponse 构建列表响应
func BuildListResponse(list interface{}, total int64, page, pageSize int) gin.H {
	return gin.H{
		"list":     list,
		"total":    total,
		"page":     page,
		"pageSize": pageSize,
	}
}

// BuildCountResponse 构建计数响应
func BuildCountResponse(count int) gin.H {
	return gin.H{
		"count": count,
	}
}

// BuildMessageResponse 构建消息响应
func BuildMessageResponse(message string) gin.H {
	return gin.H{
		"message": message,
	}
}

// BuildIDResponse 构建ID响应
func BuildIDResponse(id string) gin.H {
	return gin.H{
		"id": id,
	}
}
