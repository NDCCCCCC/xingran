package operations

import "github.com/xingran-next/xingran-go-backend/internal/models"

// workstation_physical_diagnosis.go — Phase 73 / #7（DBG-07）工位物理链路「空因可诊断」
//
// 背景（2026-10-27 根因确认，见 .planning/debug/workstation-3f130-physical-link-empty.md）：
//   工位 3f130 的「物理链路设备」面板为空，实测四项读数：
//     ws_user_id=NULL / active_info_points=1 / ip_bound_to_valid_device=1 / mac_match_count=1
//   → 数据链路（信息点→端口→MAC）完全通，空因是**工位未绑定 user_id**（该面板按
//     「用户→物理链路」维度呈现）。旧实现只有裸 `[]`，前端无法区分「没绑用户」
//     与「真没设备」，用户只能看到空白。
//
// 设计原则（契约，重要）：
//   - **向后兼容**：`GetPhysicalDevices` 的签名/返回值零改动；诊断走新增方法 +
//     新增端点，前端不接也不影响现有行为。
//   - **四类空因必须互斥且可判定**（判定顺序即下面的常量顺序，先命中先返回）：
//       1. no_user_bound            未绑定用户（**压过「有数据」**——见下）
//       2. info_point_misbound      信息点绑定错位（未绑定 / 已软删 / 状态非正常）
//       3. device_or_mac_missing    设备或 MAC 数据缺失（信息点/端口/MAC/设备 orphan）
//       4. has_devices              正常有数据（EmptyReason 为空）
//
//     ⚠️ 第 1 条「压过有数据」是**有意设计**（勿改回「数据优先」）：
//     3f130 实测 mac_match_count=1（链路通）但前端面板空白——面板按「用户→物理链路」
//     维度呈现，未绑用户时召回结果无从归属。故「未绑用户」就是该现象的真实空因，
//     必须报 no_user_bound；设备数据本身不丢，仍在 Devices 里。
//     实现见 classifyPhysicalEmptyReason（workstation_device_service.go）。
//
//   - **nil 与空串语义**：EmptyReason 用空串 "" 表示「已绑用户且有数据」，
//     用 *string 会让前端多一层判空；这里选 string + `omitempty`，
//     JSON 里有数据时该字段直接不出现。

// 物理链路空因枚举（可诊断状态机）。取值固定，前端可按字面量分支展示。
const (
	// PhysicalEmptyReasonNone 已绑定用户且召回到设备，不是空因（EmptyReason 的正常值）
	PhysicalEmptyReasonNone = ""
	// PhysicalEmptyReasonNoUserBound 工位未绑定 user_id —— 3f130 的实际命中项。
	// 物理链路本身可由「工位→信息点」推出，但页面呈现维度是用户，
	// 因此该状态下仍可能召回设备；若召回为空则判定为其根因。
	PhysicalEmptyReasonNoUserBound = "no_user_bound"
	// PhysicalEmptyReasonInfoPointMisbound 信息点绑定错位：
	// 该工位下没有 active 信息点（未绑定 / 已软删 / status≠0），
	// 或信息点未绑定端口，导致链路在第一步就断。
	PhysicalEmptyReasonInfoPointMisbound = "info_point_misbound"
	// PhysicalEmptyReasonDeviceOrMACMissing 设备/MAC 数据缺失：
	// 信息点与端口在，但端口所指设备在 sys_network_device 中不存在（orphan），
	// 或该端口上没有任何 MAC 采集记录（含历史），链路在 MAC 环节断开。
	PhysicalEmptyReasonDeviceOrMACMissing = "device_or_mac_missing"
	// PhysicalEmptyReasonHasDevices 正常有数据（= PhysicalEmptyReasonNone 的显式别名，
	// 供调用方做穷举 switch 时表达意图，值必须与 None 一致）。
	PhysicalEmptyReasonHasDevices = ""
)

// PhysicalDevicesResult 物理链路设备查询结果（含空因诊断）。
//
// 与 `[]*models.WorkstationDevice` 的关系：`Devices` 恒等于 `GetPhysicalDevices` 的返回值，
// 旧调用方可以继续用 `result.Devices` 走老路径，不需要感知诊断字段。
type PhysicalDevicesResult struct {
	// Devices 物理链路设备列表（语义与 GetPhysicalDevices 完全一致，可为空切片，不为 nil）
	Devices []*models.WorkstationDevice `json:"devices"`
	// Total 设备数量（= len(Devices)），方便前端直接显示「N 台」
	Total int `json:"total"`
	// EmptyReason 空因代码。
	//
	// ⚠️ 取值口径（v1.24 Phase 80 修正旧注释的表述矛盾）：
	// **「未绑定用户」压过「有数据」** —— 工位未绑 user_id 时，
	// 即使物理链路能召回设备，EmptyReason 仍为 no_user_bound，
	// 因为该面板按「用户→物理链路」维度呈现，未绑用户时召回结果无从归属。
	// 只有「已绑定用户 + 召回到设备」时 EmptyReason 才是空串（JSON 中省略）。
	// 判定实现见 classifyPhysicalEmptyReason（workstation_device_service.go）。
	//
	// 取值见 PhysicalEmptyReason* 常量。
	EmptyReason string `json:"emptyReason,omitempty"`
	// EmptyMessage 面向运维人员的中文可读结论（有数据时省略）。
	// 例："该工位未绑定用户（user_id 为空），物理链路面板按用户维度展示，因此为空。"
	EmptyMessage string `json:"emptyMessage,omitempty"`
	// Warnings 非致命告警（**任何情况下都可能有**，包括有数据时）。
	//
	// 典型场景（3f130）：物理链路能召回设备，但工位未绑定 user_id —— 此时
	// emptyReason 已是 no_user_bound（同上，根因可见），Warnings 仍单独承载该告警，
	// 两个通道语义互补：emptyReason 给「结论」，warnings 给「逐条提示」。
	Warnings []string `json:"warnings,omitempty"`
	// Diagnosis 逐段链路计数，用于区分「数据缺失」与「绑定错位」——
	// 即使有数据也返回，便于排查「有 1 台但少了几台」。
	Diagnosis PhysicalChainDiagnosis `json:"diagnosis"`
}

// PhysicalChainDiagnosis 物理链路逐段计数（只读诊断，不参与业务判定）。
//
// 计数口径严格对齐 GetPhysicalDevices 的 CTE，任一段为 0 即链路断开处：
//
//	sys_workstation → ops_info_points → sys_device_port_status → sys_device_mac_address
//	                → ops_asset / sys_ad_computer
type PhysicalChainDiagnosis struct {
	// WorkstationExists 工位是否存在于 sys_workstation（且未软删）
	WorkstationExists bool `json:"workstationExists"`
	// UserBound 工位是否绑定了 user_id（3f130 的断点就在此）
	UserBound bool `json:"userBound"`
	// InfoPointTotal 该工位关联的信息点总数（含软删/停用）
	InfoPointTotal int `json:"infoPointTotal"`
	// InfoPointActive 其中 status=0 且未软删的数量（CTE 实际用到的部分）
	InfoPointActive int `json:"infoPointActive"`
	// InfoPointWithPort active 信息点中已绑定 port_id 的数量
	InfoPointWithPort int `json:"infoPointWithPort"`
	// PortResolved active 信息点的 port_id 在 sys_device_port_status 中真实存在的数量
	PortResolved int `json:"portResolved"`
	// DeviceResolved port 所指设备（ip.device_id）在 sys_network_device 中存在的数量
	DeviceResolved int `json:"deviceResolved"`
	// MACMatched 该链路上命中 MAC（实测或历史）的设备条数
	MACMatched int `json:"macMatched"`
	// AssetMatched 其中还命中 ops_asset（有资产详情）的条数
	AssetMatched int `json:"assetMatched"`
	// ADMatched 其中还命中 sys_ad_computer（有域控详情）的条数
	ADMatched int `json:"adMatched"`
}
