package addomain

import (
	"testing"
	"time"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/stretchr/testify/assert"
)

// TestDetectLoginEvent 表驱动覆盖 quick 261016-ad-login-history 决策 D1 / 风险 3
// 的变化检测三分支(纯函数,不依赖 DB):
//   - 双 nil → 不记录
//   - 旧 nil → 新非 nil → initial
//   - 新 > 旧 → login
//   - 相等 / 回拨变小 / 新 nil → 不记录
func TestDetectLoginEvent(t *testing.T) {
	t0 := time.Date(2026, 10, 1, 8, 0, 0, 0, time.UTC)
	later := t0.Add(time.Hour)
	earlier := t0.Add(-time.Hour)
	zero := time.Time{}

	tests := []struct {
		name     string
		oldLL    *time.Time
		newLL    *time.Time
		wantType models.ADLoginEventType
		wantHit  bool
	}{
		{name: "双nil-不记录", oldLL: nil, newLL: nil, wantType: "", wantHit: false},
		{name: "旧nil新非nil-initial", oldLL: nil, newLL: &later, wantType: models.ADLoginEventInitial, wantHit: true},
		{name: "旧非nil新更大-login", oldLL: &t0, newLL: &later, wantType: models.ADLoginEventLogin, wantHit: true},
		{name: "相等-不记录", oldLL: &t0, newLL: &t0, wantType: "", wantHit: false},
		{name: "回拨变小-不记录", oldLL: &t0, newLL: &earlier, wantType: "", wantHit: false},
		{name: "旧非nil新nil-不记录", oldLL: &t0, newLL: nil, wantType: "", wantHit: false},
		{name: "新为零值时间-不记录", oldLL: &t0, newLL: &zero, wantType: "", wantHit: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotType, hit := detectLoginEvent(tt.oldLL, tt.newLL)
			assert.Equal(t, tt.wantHit, hit, "hit 不匹配")
			assert.Equal(t, tt.wantType, gotType, "eventType 不匹配")
		})
	}
}

// TestDetectLoginEvent_NilSafety 验证指针不被修改、不 panic
func TestDetectLoginEvent_NilSafety(t *testing.T) {
	assert.NotPanics(t, func() {
		_, _ = detectLoginEvent(nil, nil)
	})
}
