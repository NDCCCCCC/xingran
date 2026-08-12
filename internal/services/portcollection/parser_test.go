package portcollection

import "testing"

// TestNormalizeHuaweiPHYStatus 锁定华为/H3C PHY 字段 → (AdminStatus, OperStatus) 归一化。
//
// 回归目标（统一端口状态语义，对齐锐捷基准）:
//   - 手动 shutdown：PHY=`*down` → admin=down, oper=down
//   - 去 shutdown + 无网线：PHY=`down`  → admin=up,   oper=down
//   - 连网线：             PHY=`up`    → admin=up,   oper=up
//
// 历史 bug（port-status-unify-vendors）: 旧代码把 PROTOCOL 当 AdminStatus、把 PHY(含 `*`)
// 原样塞 OperStatus，导致华为显示 `*DOWN` 字面且管理员激活含义错位。修复后 AdminStatus 由
// `*` 前缀推断，OperStatus = PHY 去掉 `*`，PROTOCOL 不再参与计算。
func TestNormalizeHuaweiPHYStatus(t *testing.T) {
	tests := []struct {
		name             string
		phy              string
		wantAdminStatus  string
		wantOperStatus   string
	}{
		{
			name:            "shutdown 场景 PHY=*down → admin=down oper=down",
			phy:             "*down",
			wantAdminStatus: "down",
			wantOperStatus:  "down",
		},
		{
			name:            "去shutdown无网线 PHY=down → admin=up oper=down",
			phy:             "down",
			wantAdminStatus: "up",
			wantOperStatus:  "down",
		},
		{
			name:            "连网线 PHY=up → admin=up oper=up",
			phy:             "up",
			wantAdminStatus: "up",
			wantOperStatus:  "up",
		},
		{
			name:            "大写 *DOWN 归一为小写 admin=down oper=down",
			phy:             "*DOWN",
			wantAdminStatus: "down",
			wantOperStatus:  "down",
		},
		{
			name:            "大写 UP 归一为小写 admin=up oper=up",
			phy:             "UP",
			wantAdminStatus: "up",
			wantOperStatus:  "up",
		},
		{
			name:            "PHY 带前后空白被 TrimSpace",
			phy:             "  *down  ",
			wantAdminStatus: "down",
			wantOperStatus:  "down",
		},
		{
			name:            "空 PHY 返回两个空串（与原行为一致）",
			phy:             "",
			wantAdminStatus: "",
			wantOperStatus:  "",
		},
		{
			name:            "纯空白 PHY 视为空",
			phy:             "   ",
			wantAdminStatus: "",
			wantOperStatus:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotAdmin, gotOper := normalizeHuaweiPHYStatus(tt.phy)
			if gotAdmin != tt.wantAdminStatus || gotOper != tt.wantOperStatus {
				t.Errorf("normalizeHuaweiPHYStatus(%q) = (admin=%q, oper=%q), want (admin=%q, oper=%q)",
					tt.phy, gotAdmin, gotOper, tt.wantAdminStatus, tt.wantOperStatus)
			}
		})
	}
}
