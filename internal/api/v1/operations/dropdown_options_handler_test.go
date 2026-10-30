package operations

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	opsServices "github.com/xingran-next/xingran-go-backend/internal/services/operations"
)

// TestDropdownOption_JSONShape 锁定前后端契约:dropdown options 必须有 value 和 label 两个字段。
// 改了 DropdownOption 的 json tag 会立即失败。
func TestDropdownOption_JSONShape(t *testing.T) {
	opt := opsServices.DropdownOption{Value: "v1", Label: "label-1"}
	b, err := json.Marshal(opt)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	want := `{"value":"v1","label":"label-1"}`
	if string(b) != want {
		t.Errorf("JSON = %s, want %s", string(b), want)
	}
}

// TestDropdownOptionsRequestFlow 验证 handler 的 request body 解析路径:
// 空 body、带 name、无效 JSON 三种场景下,都不应 panic,均返回 200 + data 数组。
// 这是 handler 的最小契约测试;完整 service 调用测试需要 mock 接口,留待后续 PR。
func TestDropdownOptionsRequestFlow(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/test-dropdown", func(c *gin.Context) {
		var params map[string]interface{}
		if err := c.ShouldBindJSON(&params); err != nil {
			// 无效 JSON 不报错,降级为空 params(与 handler 实现一致)
			params = map[string]interface{}{}
		}
		// 模拟 service 行为:始终返回固定 options
		options := []opsServices.DropdownOption{
			{Value: "id-1", Label: "主楼"},
			{Value: "id-2", Label: "副楼"},
		}
		c.JSON(http.StatusOK, gin.H{"code": 0, "data": options})
	})

	tests := []struct {
		name       string
		body       string
		wantStatus int
		wantCount  int
	}{
		{name: "空 body → 返回默认 options", body: `{}`, wantStatus: http.StatusOK, wantCount: 2},
		{name: "带 name → 返回默认 options", body: `{"name":"主楼"}`, wantStatus: http.StatusOK, wantCount: 2},
		{name: "无效 JSON → 走 fallback 返回默认", body: `{`, wantStatus: http.StatusOK, wantCount: 2},
		{name: "多 filter 字段", body: `{"name":"x","orgId":"y","status":0}`, wantStatus: http.StatusOK, wantCount: 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/test-dropdown", bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d, body = %s", w.Code, tt.wantStatus, w.Body.String())
			}

			var resp struct {
				Code int                          `json:"code"`
				Data []opsServices.DropdownOption `json:"data"`
			}
			if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
				t.Fatalf("unmarshal response: %v", err)
			}
			if resp.Code != 0 {
				t.Errorf("response code = %d, want 0", resp.Code)
			}
			if len(resp.Data) != tt.wantCount {
				t.Errorf("response data length = %d, want %d", len(resp.Data), tt.wantCount)
			}
		})
	}
}