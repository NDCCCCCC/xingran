package portwrite

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/services/portcollection"
)

// TestCheckDot1xHybridConflict 802.1X ↔ hybrid 互斥预检的判据矩阵。
//
// 真机依据（2026-11-02，锐捷 RG8607E / RS8607E）：
//
//	interface GigabitEthernet 1/25
//	 switchport mode hybrid
//	 Can not operate on interface Gi1/25. Conflict with IEEE 802.1X.
func TestCheckDot1xHybridConflict(t *testing.T) {
	collected := time.Date(2026, 11, 2, 10, 30, 0, 0, time.Local)
	dot1xPort := &models.DevicePortStatus{
		ID:            "p1",
		InterfaceName: "GE1/25",
		Dot1xEnabled:  true,
		CollectedAt:   collected,
	}
	plainPort := &models.DevicePortStatus{
		ID:            "p2",
		InterfaceName: "GE1/26",
		Dot1xEnabled:  false,
		CollectedAt:   collected,
	}

	// ① 锐捷 + dot1x 开启 + hybrid ⇒ 拒绝，且文案带真机原句与采集时间
	err := checkDot1xHybridConflict(dot1xPort, portcollection.TrunkSetHybrid, models.VendorRuijie)
	require.Error(t, err, "端口启用 802.1X 时必须拒绝 hybrid（真机设备会拒绝）")
	assert.True(t, errors.Is(err, portcollection.ErrActionNotSupported),
		"必须复用 ErrActionNotSupported ⇒ handler 映射 400")
	assert.Contains(t, err.Error(), "GE1/25", "文案必须指出端口")
	assert.Contains(t, err.Error(), "802.1X", "文案必须说明 802.1X 互斥")
	assert.Contains(t, err.Error(), "Conflict with IEEE 802.1X", "文案必须带真机原句")
	assert.Contains(t, err.Error(), "2026-11-02 10:30:00", "文案必须带端口采集时间")
	assert.Contains(t, err.Error(), "请先停用 802.1X", "文案必须给出可执行的下一步")

	// ② dot1x 未开启 ⇒ 放行
	assert.NoError(t, checkDot1xHybridConflict(plainPort, portcollection.TrunkSetHybrid, models.VendorRuijie),
		"未启用 802.1X 的端口必须放行")

	// ③ 只拦 hybrid：其它动作不受影响
	assert.NoError(t, checkDot1xHybridConflict(dot1xPort, portcollection.ActionShutdown, models.VendorRuijie),
		"802.1X 只与 hybrid 互斥，不得影响其它动作")
	assert.NoError(t, checkDot1xHybridConflict(dot1xPort, portcollection.ActionDot1xDisable, models.VendorRuijie),
		"停用 802.1X 的动作本身必须放行")

	// ④ 只拦锐捷：其它厂商的互斥关系未取证 ⇒ 不凭猜测拦
	assert.NoError(t, checkDot1xHybridConflict(dot1xPort, portcollection.TrunkSetHybrid, models.VendorHuawei),
		"华为的 hybrid × 802.1X 互斥未取证，不得拦截")

	// ⑤ trunk 类动作未取证 ⇒ 不拦
	assert.NoError(t, checkDot1xHybridConflict(dot1xPort, portcollection.TrunkSetAllowed, models.VendorRuijie),
		"trunk 类动作的 802.1X 互斥未取证，不得拦截")
	assert.NoError(t, checkDot1xHybridConflict(dot1xPort, portcollection.TrunkSetNative, models.VendorRuijie),
		"trunk native 的 802.1X 互斥未取证，不得拦截")

	// ⑥ 采集时间零值 ⇒ 明示"未采集"，不显示 0001-01-01
	zeroTimePort := &models.DevicePortStatus{InterfaceName: "GE1/27", Dot1xEnabled: true}
	err = checkDot1xHybridConflict(zeroTimePort, portcollection.TrunkSetHybrid, models.VendorRuijie)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "未采集", "采集时间缺失时必须明示")
}

// TestTrunkWrite_BlockedByDot1x 写路径级：端口启用 802.1X 时，锐捷 hybrid 必须在
// **下发前**被拒（且一条命令都不下发）—— 否则设备侧才报错，用户只看到英文原文。
//
// ★ 本用例**只带 untagged**（锐捷 hybrid 现在只接受 untagged，见 TestTrunkWrite_
//
//	RuijieHybridRejectsTagged）：带 tagged 会先被渲染层的 (i-a) 拦截命中，
//	压根走不到 802.1X 预检 —— 两个用例各测一件事，不要混。
func TestTrunkWrite_BlockedByDot1x(t *testing.T) {
	db := newTestDB(t)
	seedPreviewDevice(t, db, models.VendorRuijie)

	// 设备型号命中锐捷能力矩阵（RS8607E ⇒ hybrid 真机支持）
	require.NoError(t, db.Model(&models.NetworkDevice{}).
		Where("id = ?", "device-ruijie").
		Update("model", "RS8607E").Error)

	// 端口启用 802.1X（真机事实：这批设备 100% 命中）
	require.NoError(t, db.Model(&models.DevicePortStatus{}).
		Where("id = ?", "port-ruijie").
		Update("dot1x_enabled", true).Error)

	exec := &countingExecutor{}
	svc := newTestService(exec, nil, db)

	_, err := svc.TrunkWrite(context.Background(), "port-ruijie", portcollection.TrunkSetHybrid,
		TrunkParams{HybridUntagged: []int{3990}}, "tester")

	require.Error(t, err, "端口启用 802.1X 时必须拒绝 hybrid")
	assert.ErrorIs(t, err, portcollection.ErrActionNotSupported,
		"必须复用 ErrActionNotSupported ⇒ handler 映射 400")
	assert.Contains(t, err.Error(), "Conflict with IEEE 802.1X", "错误文案必须带真机原句")
	assert.Equal(t, 0, exec.calls, "预检必须发生在下发之前（一条命令都不应下发）")
}

// TestTrunkWrite_RuijieHybridRejectsTagged 写路径级：(i-a) 锐捷 hybrid 带 tagged
// 必须被拒（端口**无** dot1x ⇒ 命中的一定是 (i-a)，不会与 802.1X 预检混淆）。
//
// 真机依据（2026-11-02）：锐捷 hybrid 是自动补集语义 ⇒ tagged 在锐捷无设备侧
// 对应物 ⇒ 提交 tagged 必须明确报错，不静默丢弃。
func TestTrunkWrite_RuijieHybridRejectsTagged(t *testing.T) {
	db := newTestDB(t)
	seedPreviewDevice(t, db, models.VendorRuijie)
	require.NoError(t, db.Model(&models.NetworkDevice{}).
		Where("id = ?", "device-ruijie").
		Update("model", "RS8607E").Error)
	// 端口**不**启用 802.1X（seed 默认 false）⇒ 只可能命中 (i-a)

	exec := &countingExecutor{}
	svc := newTestService(exec, nil, db)

	_, err := svc.TrunkWrite(context.Background(), "port-ruijie", portcollection.TrunkSetHybrid,
		TrunkParams{HybridTagged: []int{3990}, HybridUntagged: []int{3991}}, "tester")

	require.Error(t, err, "锐捷 hybrid 带 tagged 必须被拒")
	assert.ErrorIs(t, err, portcollection.ErrActionNotSupported, "必须复用 ErrActionNotSupported ⇒ 400")
	assert.Contains(t, err.Error(), "自动补集语义", "错误文案必须是 (i-a) 的说明")
	assert.NotContains(t, err.Error(), "Conflict with IEEE 802.1X",
		"端口未启用 802.1X 时不得出现 802.1X 文案")
	assert.Equal(t, 0, exec.calls, "预检必须发生在下发之前（一条命令都不应下发）")
}

// TestTrunkWrite_HybridGuardOrder 锁定**两层拦截的顺序**：带 tagged **且**端口启用
// 802.1X 时，命中的必须是 (i-a)「自动补集语义」（渲染层在前），而不是 802.1X 文案。
//
// 为什么值得锁：双层拦截的顺序决定用户看到的错误文案 —— 顺序变了，用户看到的
// 提示就从"你的参数在这个厂商不被支持"变成"这个口开了 802.1X"，属于行为变更。
// 本用例让顺序被调整时立刻报错。
func TestTrunkWrite_HybridGuardOrder(t *testing.T) {
	db := newTestDB(t)
	seedPreviewDevice(t, db, models.VendorRuijie)
	require.NoError(t, db.Model(&models.NetworkDevice{}).
		Where("id = ?", "device-ruijie").
		Update("model", "RS8607E").Error)
	// 端口同时启用 802.1X
	require.NoError(t, db.Model(&models.DevicePortStatus{}).
		Where("id = ?", "port-ruijie").
		Update("dot1x_enabled", true).Error)

	exec := &countingExecutor{}
	svc := newTestService(exec, nil, db)

	_, err := svc.TrunkWrite(context.Background(), "port-ruijie", portcollection.TrunkSetHybrid,
		TrunkParams{HybridTagged: []int{3990}, HybridUntagged: []int{3991}}, "tester")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "自动补集语义",
		"两层拦截中渲染层在前 ⇒ 参数问题（tagged）优先于端口状态问题（802.1X）")
	assert.NotContains(t, err.Error(), "Conflict with IEEE 802.1X",
		"命中 (i-a) 时不得混入 802.1X 文案（顺序即契约）")
	assert.Equal(t, 0, exec.calls, "拦截必须发生在下发之前")
}
