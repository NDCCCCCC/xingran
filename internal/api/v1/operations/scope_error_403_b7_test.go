package operations

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	opsServices "github.com/xingran-next/xingran-go-backend/internal/services/operations"
)

// B-7 / quick 261110-161：room_photo_handler.go 的 8 个 service 错误分支统一走
// writeScopeAwareServiceError。该 helper 必须把 ErrScopeDenied 显式透传 403，
// 普通 error 保留存量 response.ErrServerError(500) 形态（零回归）。
//
// 用 gin context + httptest 直接测 helper，避免为闭包式 room_photo handler
// 引入整套 service mock 基建；8 个调用点共享同一个 helper，守护等价。
func TestWriteScopeAwareServiceError_ScopeDenied(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cases := []struct {
		name string
		err  error
		want int
	}{
		{"ErrScopeDenied 直接", opsServices.ErrScopeDenied, http.StatusForbidden},
		{"ErrScopeDenied 被 fmt.Errorf 包裹", errors.Join(errors.New("wrap"), opsServices.ErrScopeDenied), http.StatusForbidden},
		{"普通 error 保留 500", errors.New("db busy"), http.StatusInternalServerError},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			writeScopeAwareServiceError(c, tc.err)
			assert.Equal(t, tc.want, w.Code,
				"越界必须 403、普通错误必须保持 500（149-UAT-RECHECK.md:43 口径）")
		})
	}
}
