package vdi

import (
	"net/http"

	"github.com/gin-gonic/gin"
	apperrors "github.com/xingran-next/xingran-go-backend/pkg/errors"
	"github.com/xingran-next/xingran-go-backend/pkg/response"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
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
		// 记录详细的错误信息到日志
		logrus.WithFields(logrus.Fields{
			"action": action,
			"path":   c.Request.URL.Path,
			"error":  err.Error(),
		}).Errorf("VDI %s failed", action)

		response.Error(c, apperrors.Wrap(err, apperrors.CodeServerError, action+"失败"))
		return false
	}
	return true
}

// verifyVDIServerExists 验证VDI服务器是否存在
func verifyVDIServerExists(db *gorm.DB, serverID string) *models.VDIServer {
	var server models.VDIServer
	if err := db.Where("id = ?", serverID).First(&server).Error; err != nil {
		return nil
	}
	return &server
}

// ensureVDIServer 验证VDI服务器并返回错误响应（如果不存在）
func ensureVDIServer(c *gin.Context, db *gorm.DB, serverID string) bool {
	if serverID == "" {
		response.Error(c, http.StatusBadRequest, "VDI服务器ID不能为空")
		return false
	}

	server := verifyVDIServerExists(db, serverID)
	if server == nil {
		response.Error(c, http.StatusNotFound, "VDI服务器不存在")
		return false
	}

	return true
}
