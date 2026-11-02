package portwrite

// PWX-05（v1.51 Phase 127）：3 厂商 e2e fixture 全覆盖补缺 + 覆盖矩阵报告。
//
// 既有覆盖（本文件不重复，只补缺）：
//   - dry_run_test.go  TestPreview_AllVendors_AllActions —— 基础 7 动作 × 3 厂商（21 case）
//   - vlan_trunk_test.go TestTrunkRender_AllVendors —— trunk 12 case（含锐捷 replace、hybrid 二家）
//   - portsec_test.go  TestPortSecRender_AllVendors —— portsec 14 case（含 sticky 二家、static_bind critical）
//
// 本文件补四处缺口（125 契约注释里的厂商差异依据逐条断言）：
//   1. hybrid tagged/untagged 重叠 → ErrHybridOverlap（校验 + preview 双层）
//   2. trunk replace 华为/H3C 退化叠加语义（VRP allow-pass 天然叠加；H3C permit 同理）
//   3. portsec_sticky H3C 分支 + 三家 sticky「先 enable 前置」差异断言
//   4. 15 动作 × 3 厂商全矩阵跑通 + 覆盖报告输出（t.Logf 报告表）

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/services/portcollection"
)

// TestPWX05_HybridOverlapSentinel hybrid tagged/untagged 重叠 → 校验层拒绝，
// 且 dry-run preview 同样拦截（真相源同源，不产生第二套校验）。
func TestPWX05_HybridOverlapSentinel(t *testing.T) {
	overlap := TrunkParams{HybridTagged: []int{30}, HybridUntagged: []int{30, 40}}

	// 校验层（真相源）
	err := validateTrunkRequest(portcollection.TrunkSetHybrid, overlap)
	assert.ErrorIs(t, err, ErrHybridOverlap, "重叠列表必须被校验层拒绝")

	// preview 层（同源入口）
	db := newTestDB(t)
	seedPreviewDevice(t, db, models.VendorHuawei)
	svc := newTestService(&countingExecutor{}, nil, db)
	_, err = svc.PreviewPortWrite(context.Background(), PreviewRequest{
		DeviceID:       "device-" + string(models.VendorHuawei),
		Action:         portcollection.TrunkSetHybrid,
		PortIDs:        []string{"port-" + string(models.VendorHuawei)},
		HybridTagged:   overlap.HybridTagged,
		HybridUntagged: overlap.HybridUntagged,
	}, "tester")
	assert.ErrorIs(t, err, ErrHybridOverlap, "preview 必须透出同一 sentinel")
}

// TestPWX05_TrunkReplaceDegradation replace 仅锐捷原生为**全量覆盖**（switchport trunk allowed
// vlan only <list>，★ only 本身即全量语义；2026-11-02 真机定稿）；华为 VRP allow-pass 天然叠加、
// H3C permit 同理 —— 语义上退化为叠加，命令形态与 add 一致（依据见 vlan_trunk_template.go 厂商注释）。
func TestPWX05_TrunkReplaceDegradation(t *testing.T) {
	cases := []struct {
		vendor     models.DeviceVendor
		wantSub    []string
		notWantSub []string
	}{
		{models.VendorHuawei, []string{"port trunk allow-pass vlan 10 20"}, []string{"undo port trunk allow-pass"}},
		{models.VendorH3C, []string{"port trunk permit vlan 10 20"}, []string{"undo port trunk permit"}},
		// ★ 2026-11-02 真机定稿：锐捷 replace 的形态是 `only <list>`（only 本身即全量语义）；
		//   无关键字形态 `switchport trunk allowed vlan <list>` 真机 Invalid input ⇒ 非法命令。
		{models.VendorRuijie, []string{"switchport trunk allowed vlan only 10,20"},
			[]string{"switchport trunk allowed vlan add", "switchport trunk allowed vlan remove"}},
	}
	for _, tc := range cases {
		t.Run(string(tc.vendor), func(t *testing.T) {
			db := newTestDB(t)
			seedPreviewDevice(t, db, tc.vendor)
			svc := newTestService(&countingExecutor{}, nil, db)
			result, err := svc.PreviewPortWrite(context.Background(), PreviewRequest{
				DeviceID:     "device-" + string(tc.vendor),
				Action:       portcollection.TrunkSetAllowed,
				PortIDs:      []string{"port-" + string(tc.vendor)},
				AllowedVLANs: []int{10, 20},
				AllowedMode:  portcollection.AllowedModeReplace,
			}, "tester")
			require.NoError(t, err)
			joined := strings.Join(previewCommandsOf(result), "\n")
			for _, want := range tc.wantSub {
				assert.Contains(t, joined, want)
			}
			for _, notWant := range tc.notWantSub {
				assert.NotContains(t, joined, notWant, "%s replace 不得出现 add/remove 关键字（退化语义）", tc.vendor)
			}
			assert.Equal(t, "critical", result.RiskLevel)
		})
	}
}

// TestPWX05_StickyAllVendors sticky 三家补全（含既有用例缺失的 H3C），
// 并断言「sticky 前置 port-security enable」的依赖顺序（PWX-02 注明的
// 依赖顺序错误规避）。
func TestPWX05_StickyAllVendors(t *testing.T) {
	cases := []struct {
		vendor         models.DeviceVendor
		stickyCmd      string
		enableCmdNeed  bool // sticky 前置 enable 命令（锐捷/Cisco 风格需要显式 enable）
		enableCmdForms []string
	}{
		{models.VendorHuawei, "port-security mac-address sticky", true, []string{"port-security enable"}},
		{models.VendorH3C, "port-security mac-address sticky", true, []string{"port-security enable"}},
		{models.VendorRuijie, "switchport port-security mac-address sticky", true, []string{"switchport port-security"}},
	}
	for _, tc := range cases {
		t.Run(string(tc.vendor), func(t *testing.T) {
			db := newTestDB(t)
			seedPreviewDevice(t, db, tc.vendor)
			svc := newTestService(&countingExecutor{}, nil, db)
			result, err := svc.PreviewPortWrite(context.Background(), PreviewRequest{
				DeviceID: "device-" + string(tc.vendor),
				Action:   portcollection.PortSecSticky,
				PortIDs:  []string{"port-" + string(tc.vendor)},
			}, "tester")
			require.NoError(t, err)
			cmds := previewCommandsOf(result)
			joined := strings.Join(cmds, "\n")
			assert.Contains(t, joined, tc.stickyCmd, "%s sticky 命令缺失", tc.vendor)
			// 依赖顺序：enable 前置命令必须出现在 sticky 命令之前（若渲染层输出 enable）
			stickyIdx := -1
			for i, c := range cmds {
				if strings.Contains(c, "sticky") {
					stickyIdx = i
					break
				}
			}
			require.GreaterOrEqual(t, stickyIdx, 0)
			enableIdx := -1
			for _, form := range tc.enableCmdForms {
				for i, c := range cmds[:stickyIdx] {
					if strings.Contains(c, form) {
						enableIdx = i
						break
					}
				}
				if enableIdx >= 0 {
					break
				}
			}
			if enableIdx >= 0 {
				assert.Less(t, enableIdx, stickyIdx, "%s enable 命令必须先于 sticky（依赖顺序）", tc.vendor)
			}
			assert.Equal(t, "medium", result.RiskLevel)
		})
	}
}

// TestPWX05_FullCoverageMatrix 15 动作 × 3 厂商全矩阵跑通 + 覆盖报告。
// 每个 cell 断言：preview 无错、零网络调用、cleanup 末尾命令（有命令时）。
// 覆盖报告输出到 t.Logf（go test -v 可见），作为验收 #1 的矩阵覆盖表。
func TestPWX05_FullCoverageMatrix(t *testing.T) {
	vendors := []models.DeviceVendor{models.VendorHuawei, models.VendorH3C, models.VendorRuijie}
	actions := []Action{
		// 基础 7
		portcollection.ActionShutdown,
		portcollection.ActionUndoShutdown,
		portcollection.ActionDescription,
		portcollection.ActionDot1xEnable,
		portcollection.ActionDot1xDisable,
		portcollection.ActionSetAccessVLAN,
		portcollection.ActionPortBinding,
		// trunk 3
		portcollection.TrunkSetNative,
		portcollection.TrunkSetAllowed,
		portcollection.TrunkSetHybrid,
		// portsec 5
		portcollection.PortSecEnable,
		portcollection.PortSecDisable,
		portcollection.PortSecMaxMac,
		portcollection.PortSecSticky,
		portcollection.PortSecStaticBind,
	}

	report := make([][]string, 0, len(vendors))
	for _, vendor := range vendors {
		row := []string{string(vendor)}
		for _, action := range actions {
			action := action
			t.Run(fmt.Sprintf("%s/%s", vendor, action), func(t *testing.T) {
				// 2026-11-02：锐捷 hybrid 由"渲染层拒绝"改为"真机支持"
				// （真机 `?` 帮助确认 switchport hybrid native/allowed 存在），
				// 因此锐捷 hybrid 走正常 preview 路径，与华为/H3C 同口径。
				db := newTestDB(t)
				seedPreviewDevice(t, db, vendor)
				// portsec_disable 需要"已启用"pre-state 才不 NoOp
				if action == portcollection.PortSecDisable {
					var port models.DevicePortStatus
					require.NoError(t, db.Where("id = ?", "port-"+string(vendor)).First(&port).Error)
					port.PortSecurityEnabled = true
					require.NoError(t, db.Save(&port).Error)
				}
				exec := &countingExecutor{}
				svc := newTestService(exec, nil, db)
				req := fullMatrixRequest(vendor, action)
				result, err := svc.PreviewPortWrite(context.Background(), req, "matrix-tester")
				require.NoError(t, err, "%s/%s preview 不应报错（NoOp 语义内化为 skipped）", vendor, action)
				assert.Equal(t, 0, exec.calls, "%s/%s dry-run 必须零网络调用", vendor, action)

				cmds := previewCommandsOf(result)
				cell := "cmd"
				if len(cmds) == 0 {
					cell = "noop"
				} else {
					last := cmds[len(cmds)-1]
					assert.True(t, last == "exit" || last == "quit",
						"%s/%s 末尾必须是 view 退出命令, got %q", vendor, action, last)
				}
				row = append(row, fmt.Sprintf("%s(%s)", action, cell))
			})
			_ = row
		}
		report = append(report, row)
	}

	// 覆盖报告（验收 #1：矩阵覆盖表）
	t.Logf("=== PWX-05 覆盖矩阵报告（%d 厂商 × %d 动作 = %d cells） ===", len(vendors), len(actions), len(vendors)*len(actions))
	t.Logf("动作: %v", actions)
	t.Logf("说明: 锐捷 trunk_set_hybrid 2026-11-02 起真机支持（`?` 帮助确认），" +
		"已注册渲染并走正常 preview 路径；" +
		"NoOp 语义 cell 标记 noop（pre-state 已匹配，skipped），其余标记 cmd。")
	for _, row := range report {
		t.Logf("vendor=%s cells=%d", row[0], len(row)-1)
	}
}

// fullMatrixRequest 按 action 组装全参数请求（参数间互不冲突，仅用于矩阵跑通）。
func fullMatrixRequest(vendor models.DeviceVendor, action Action) PreviewRequest {
	req := PreviewRequest{
		DeviceID:     "device-" + string(vendor),
		Action:       action,
		PortIDs:      []string{"port-" + string(vendor)},
		Description:  "matrix",
		VLANID:       100,
		BindOp:       "add",
		IPAddress:    "192.168.1.10",
		MACAddress:   "AA:BB:CC:DD:EE:FF",
		MaxMacCount:  5,
		NativeVLAN:   10,
		AllowedVLANs: []int{10, 20},
		AllowedMode:  portcollection.AllowedModeAdd,
		// hybrid 重叠会被 ErrHybridOverlap 拦截，矩阵用不重叠参数
		HybridTagged:   []int{30},
		HybridUntagged: []int{10, 20},
	}
	// ★ 2026-11-02 产品口径 (i)（lead 拍板，依据第三次真机取证 + 补验）：
	//   锐捷 hybrid 是自动补集语义 ⇒ **不下发 tagged**（tagged 在锐捷无设备侧
	//   对应物，提交 tagged 会被 ErrActionNotSupported 拒绝）⇒ 锐捷侧清空 tagged。
	//   untagged 允许多个（补验⑤：多 untagged 被设备接受，设备写成区间形式）。
	if vendor == models.VendorRuijie {
		req.HybridTagged = nil
	}
	// portsec_disable 需要"已启用"pre-state 才不 NoOp（与既有用例同口径）
	_ = action
	return req
}
