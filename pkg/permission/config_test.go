package permission

// Phase 144 · 权限常量 + 路由注册表断言（ALRT-01/02/03 + Phase 143 移交项）。
//
// 守护：
//  1. 5 个漂移告警权限码逐字一致（与 sys_menu seed / 前端 permissionCodes.ts）；
//  2. 5 条漂移告警端点 GetPermissionByPath 命中且权限分组正确；
//  3. Phase 143 移交的 POST /network/ports/write/physical 已补登 NetworkPortWrite。

import "testing"

func TestPhase144_ADDriftAlertPermissionCodes(t *testing.T) {
	want := map[PermissionCode]string{
		ADDriftAlert:        "system:adDriftAlert",
		ADDriftAlertList:    "system:adDriftAlert:list",
		ADDriftAlertConfig:  "system:adDriftAlert:config",
		ADDriftAlertEdit:    "system:adDriftAlert:edit",
		ADDriftAlertTrigger: "system:adDriftAlert:trigger",
	}
	for code, literal := range want {
		if string(code) != literal {
			t.Errorf("权限码 %q 漂移: got %q", literal, string(code))
		}
	}
}

func TestPhase144_ADDriftAlertRoutePermissions(t *testing.T) {
	cases := []struct {
		path string
		want PermissionCode
	}{
		{"/system/ad-drift-alert/status", ADDriftAlertList},
		{"/system/ad-drift-alert/logs", ADDriftAlertList},
		{"/system/ad-drift-alert/config", ADDriftAlertConfig},
		{"/system/ad-drift-alert/config/update", ADDriftAlertEdit},
		{"/system/ad-drift-alert/trigger", ADDriftAlertTrigger},
		{"/system/ad-drift-alert/logs/read", ADDriftAlertList},
	}
	for _, c := range cases {
		got := GetPermissionByPath(c.path, "POST")
		if got != c.want {
			t.Errorf("POST %s 死表权限码 = %q, want %q", c.path, got, c.want)
		}
	}
}

// TestPhase144_NetworkPortWritePhysicalRegistered Phase 143 移交补登。
func TestPhase144_NetworkPortWritePhysicalRegistered(t *testing.T) {
	got := GetPermissionByPath("/network/ports/write/physical", "POST")
	if got != NetworkPortWrite {
		t.Errorf("POST /network/ports/write/physical 死表权限码 = %q, want %q", got, NetworkPortWrite)
	}
}
