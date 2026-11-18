package models

import "testing"

// TestIsValidAnchorMode 锁定锚点模式白名单（0/2/3/5），弃用编号 1/4 与未知值必须拒绝。
func TestIsValidAnchorMode(t *testing.T) {
	cases := []struct {
		mode AnchorMode
		want bool
	}{
		{AnchorModeNone, true},
		{AnchorModeDept, true},
		{AnchorModeExternalOrg, true},
		{AnchorModeLevel, true},
		{AnchorMode(1), false}, // 弃用：相对层深
		{AnchorMode(4), false}, // 弃用：经理机构
		{AnchorMode(-1), false},
		{AnchorMode(6), false},
		{AnchorMode(99), false},
	}
	for _, c := range cases {
		if got := IsValidAnchorMode(c.mode); got != c.want {
			t.Errorf("IsValidAnchorMode(%d) = %v, want %v", c.mode, got, c.want)
		}
	}
}
