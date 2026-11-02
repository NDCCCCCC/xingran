package portwrite

import (
	"fmt"
	"time"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/services/portcollection"
)

// ====================== 802.1X ↔ hybrid 互斥预检 ======================
//
// 真机事实（2026-11-02，锐捷 RG8607E 10.62.63.47 GE1/25 / RS8607E 10.62.63.21 GE1/25）：
//
//	interface GigabitEthernet 1/25
//	 switchport mode hybrid
//	 Can not operate on interface Gi1/25. Conflict with IEEE 802.1X.
//
// ⇒ 端口配了 802.1X（dot1x port-control auto）时，锐捷设备**不允许**把口切到
//   hybrid 模式（设备级互斥）。真实网络里端口默认带 dot1x（实测该批设备 100% 命中）
//   ⇒ 不在预检阶段拦截的话，下发必然失败，且用户只看到一句英文报错。
//
// 判据用**采集快照**里的 Dot1xEnabled：
//   - 该字段按接口粒度采集（锐捷 `show dot1x port-control` → portcollection/utils.go
//     填入 Dot1xEnabled），**零设备开销**；
//   - 与既有口径一致：pre_state_check.go 里 ActionPortBinding 已论证「写路径实时 SSH
//     查端口信息 3-5s/port 不可接受」⇒ 这里同样不做实时读配置。
//
// 口径（lead 裁定 2026-11-02）：**拒绝 + 错误文案带采集时间**，把判断权交回操作者
//   （不静默放行，也不因快照过期就把所有 hybrid 操作锁死）。
//
// 只拦 TrunkSetHybrid 且只拦锐捷：真机只证实了「锐捷 hybrid × 802.1X」互斥，
//   trunk 类动作与其它厂商的互斥关系**未取证** ⇒ 不凭猜测扩大拦截面。

// checkDot1xHybridConflict 端口启用 802.1X 时拒绝 hybrid 动作（预检层）。
//
// 返回的错误包装 portcollection.ErrActionNotSupported ⇒ handler 已映射 400，
// 无需改 handler。
func checkDot1xHybridConflict(port *models.DevicePortStatus, action Action, vendor models.DeviceVendor) error {
	if action != portcollection.TrunkSetHybrid {
		return nil
	}
	if vendor != models.VendorRuijie {
		return nil
	}
	if port == nil || !port.Dot1xEnabled {
		return nil
	}
	return fmt.Errorf("%w: 端口 %s 当前启用了 802.1X（dot1x port-control auto），"+
		"锐捷设备不允许在启用 802.1X 的端口上切换 hybrid 模式"+
		"（真机依据 2026-11-02：Can not operate on interface %s. Conflict with IEEE 802.1X.；"+
		"端口采集时间 %s）请先停用 802.1X 再切换端口模式",
		portcollection.ErrActionNotSupported,
		port.InterfaceName,
		port.InterfaceName,
		formatCollectedAt(port.CollectedAt),
	)
}

// formatCollectedAt 端口采集时间的展示（零值 ⇒ 明示"未采集"）。
func formatCollectedAt(t time.Time) string {
	if t.IsZero() {
		return "（未采集）"
	}
	return t.Format("2006-01-02 15:04:05")
}
