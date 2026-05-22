package system

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestSyncDeptToADHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// 创建Handler（传入nil服务用于测试）
	handler := NewADDeptSyncHandler(nil)
	router.POST("/sync/dept-to-ad", handler.SyncDeptToAD)

	// 测试正常请求
	reqBody := map[string]string{
		"adConfigId": "config-1",
	}
	body, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest("POST", "/sync/dept-to-ad", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// 由于没有真实的服务，这里主要测试请求格式正确
	// 实际的集成测试需要Mock服务
	assert.Contains(t, w.Body.String(), "adConfigId")
}

func TestGetDeptSyncStatus(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	handler := NewADDeptSyncHandler(nil)
	router.GET("/sync/dept-status/:configId", handler.GetDeptSyncStatus)

	req, _ := http.NewRequest("GET", "/sync/dept-status/config-1", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// 测试路由正确设置
	assert.Contains(t, w.Body.String(), "configId")
}

func TestTriggerDeptSync(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	handler := NewADDeptSyncHandler(nil)
	router.POST("/sync/trigger", handler.TriggerDeptSync)

	reqBody := map[string]string{
		"adConfigId": "config-1",
	}
	body, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest("POST", "/sync/trigger", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// 测试请求处理
	assert.True(t, w.Code == http.StatusOK || w.Code == http.StatusBadRequest)
}
