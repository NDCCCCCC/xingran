package operations

import (
	"context"
	"strings"
	"testing"
)

// TestGeocode_WithoutBaiduAPIKey 覆盖未配置百度地图 AK 的分支。
//
// 未配置 AK 时若继续往下走，会带着空 ak 去请求百度接口，只换来
// 难以定位的「ak 无效」鉴权错误，排查方向容易跑偏。这里要求直接
// 返回明确的配置错误。
func TestGeocode_WithoutBaiduAPIKey(t *testing.T) {
	svc := NewGeocodingService("")

	_, _, err := svc.Geocode(context.Background(), "北京市海淀区中关村大街1号")
	if err == nil {
		t.Fatal("Geocode() err = nil, want 未配置百度地图 AK 的错误")
	}
	if !strings.Contains(err.Error(), "未配置百度地图 AK") {
		t.Fatalf("Geocode() err = %v, want 含「未配置百度地图 AK」", err)
	}
}

// TestGeocode_EmptyAddress 覆盖空地址的早退分支。
func TestGeocode_EmptyAddress(t *testing.T) {
	svc := NewGeocodingService("ak-for-test")

	_, _, err := svc.Geocode(context.Background(), "")
	if err == nil {
		t.Fatal("Geocode() err = nil, want 地址为空的错误")
	}
	if !strings.Contains(err.Error(), "地址不能为空") {
		t.Fatalf("Geocode() err = %v, want 含「地址不能为空」", err)
	}
}
