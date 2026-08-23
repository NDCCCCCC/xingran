package operations

import (
	"context"
	"net/http"
	"testing"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =====================================================================
// Phase 76-01: geocoding httpmock PoC（INFRA-01，go mod tidy 保活锚点）。
//
// 使用纪律（全仓 httpmock 活样板）：
//   1. 优先 httpmock.Activate(t) —— testing.TB 形态，测试结束自动恢复
//      DefaultTransport，不用手动 Deactivate 的旧形态；
//   2. 凡 httpClient / baseURL 可注入的场景一律优先标准库 httptest
//      （零依赖），见 geocoding_photo_floor_test.go 的假 RoundTripper 先例；
//   3. httpmock 专治 const-URL 无注入点场景——本 PoC 用生产构造器
//      NewGeocodingService("ak-test")，其内部 httpClient 为
//      &http.Client{Timeout:...} 且 Transport 为空 → 走 DefaultTransport，
//      Activate 后拦截即命中。禁止白盒替换 svc.httpClient（那会让 PoC
//      退化成 httptest 等价物，失去 const-URL 场景的示范意义）；
//   4. RegisterResponder 注册不带 query 的 URL 即可匹配任意 query。
// =====================================================================

// TestGeocodingService_HttpmockDefaultTransport 经 DefaultTransport 拦截
// const-URL（baiduGeocodingAPIURL）的百度地理编码调用。
// 复用同包 geocoding_photo_floor_test.go 的 geocodeOKBody fixture。
func TestGeocodingService_HttpmockDefaultTransport(t *testing.T) {
	httpmock.Activate(t) // testing.TB 形态：自动清理并恢复 DefaultTransport

	// 注册 URL 不带 query → 匹配任意 address/ak/output query 组合。
	httpmock.RegisterResponder(http.MethodGet, baiduGeocodingAPIURL,
		httpmock.NewStringResponder(http.StatusOK, geocodeOKBody))

	// 生产构造器，零白盒改动：Transport 为空 → DefaultTransport 被拦截。
	svc := NewGeocodingService("ak-test")

	lng, lat, err := svc.Geocode(context.Background(), "测试地址")
	require.NoError(t, err)
	assert.InDelta(t, 116.404, lng, 0.001)
	assert.InDelta(t, 39.915, lat, 0.001)

	// 拦截命中计数：内存缓存首次 miss，真实出站请求恰好 1 次。
	assert.Equal(t, 1, httpmock.GetTotalCallCount())
}
