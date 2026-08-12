package portcollection

import (
	"strconv"
	"strings"

	"github.com/xingran-next/xingran-go-backend/internal/device"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
	"github.com/xingran-next/xingran-go-backend/internal/templates"
)

// InterfaceInfo 接口信息
type InterfaceInfo struct {
	Name     string
	VLAN     *int
	Duplex   string
	Speed    string
	PortType string
}

// Dot1xInfo 802.1X信息
type Dot1xInfo struct {
	Enabled     bool
	PortStatus  string
	MaxUser     string
	CurrentUser string
}

// PortSecurityInfo 端口安全信息
type PortSecurityInfo struct {
	Enabled      bool
	SecurityMode string
	MaxMAC       int
	CurrentMAC   int
}

// InterfaceDescription 接口描述信息
type InterfaceDescription struct {
	InterfaceName string
	OriginalName  string
	AdminStatus   string
	OperStatus    string
	Description   string
}

// parseInterfaceList 解析接口列表
func parseInterfaceList(wrapper *device.ScrapliWrapper, vendor models.DeviceVendor) ([]InterfaceInfo, error) {
	command := getInterfaceCommand(vendor)

	resp, err := wrapper.SendCommand(command, true)
	if err != nil {
		return nil, err
	}

	templatePath := getInterfaceTemplate(vendor)
	fsm, err := templates.ParseTemplate(templatePath)
	if err != nil {
		return nil, err
	}

	records, err := fsm.ParseText(resp.Result)
	if err != nil {
		return nil, err
	}

	var interfaces []InterfaceInfo
	for _, record := range records {
		info := InterfaceInfo{}

		var name string
		if n, ok := record["INTERFACE"]; ok && n != "" {
			name = n
		} else if n, ok = record["Interface"]; ok && n != "" {
			name = n
		} else if n, ok = record["interface"]; ok && n != "" {
			name = n
		}

		if name == "" {
			continue
		}

		originalName := strings.ReplaceAll(name, " ", "")
		info.Name = NormalizeInterfaceName(originalName)

		if vlanStr, ok := record["VLAN"]; ok && vlanStr != "--" && vlanStr != "" {
			if vlan, err := strconv.Atoi(vlanStr); err == nil {
				info.VLAN = &vlan
			}
		}

		if duplex, ok := record["DUPLEX"]; ok && duplex != "--" && duplex != "" {
			info.Duplex = strings.ToLower(duplex)
		}

		if speed, ok := record["SPEED"]; ok && speed != "--" && speed != "" {
			info.Speed = speed
		}

		if portType, ok := record["TYPE"]; ok && portType != "--" && portType != "" {
			info.PortType = strings.ToLower(portType)
		}

		interfaces = append(interfaces, info)
	}

	return interfaces, nil
}

// parseInterfaceDescriptions 解析接口状态和描述信息
func parseInterfaceDescriptions(wrapper *device.ScrapliWrapper, vendor models.DeviceVendor) (map[string]InterfaceDescription, error) {
	command := getInterfaceDescriptionCommand(vendor)

	resp, err := wrapper.SendCommand(command, true)
	if err != nil {
		return nil, err
	}

	templatePath := getInterfaceDescriptionTemplate(vendor)
	fsm, err := templates.ParseTemplate(templatePath)
	if err != nil {
		return nil, err
	}

	records, err := fsm.ParseText(resp.Result)
	if err != nil {
		return nil, err
	}

	descriptions := make(map[string]InterfaceDescription)
	for _, record := range records {
		desc := InterfaceDescription{}

		if name, ok := record["INTERFACE"]; ok {
			trimmedName := strings.TrimSpace(name)
			desc.InterfaceName = NormalizeInterfaceName(trimmedName)
			desc.OriginalName = trimmedName
		}

		// 状态字段映射（厂商分支）：
		// 华为/H3C：PHY 带 `*` 前缀表示 administratively down。
		//   - AdminStatus 由 `*` 前缀推断：`*down`→down，其余→up
		//   - OperStatus = PHY 去掉 `*` 前缀的小写值（up/down）
		//   PROTOCOL（数据链路层协议状态）不参与 AdminStatus 计算，避免与锐捷语义错位。
		// 锐捷/迈普：STATUS→OperStatus，ADMINISTRATIVE→AdminStatus（已是正确语义，作为统一基准）。
		if vendor == models.VendorHuawei || vendor == models.VendorH3C {
			desc.AdminStatus, desc.OperStatus = normalizeHuaweiPHYStatus(record["PHY"])
		} else {
			if status, ok := record["STATUS"]; ok {
				desc.OperStatus = strings.ToLower(status)
			}
			if admin, ok := record["ADMINISTRATIVE"]; ok {
				desc.AdminStatus = strings.ToLower(admin)
			}
		}

		if description, ok := record["DESCRIPTION"]; ok {
			desc.Description = description
		}

		if desc.InterfaceName != "" {
			descriptions[desc.InterfaceName] = desc
			// 输出前3条调试信息
			if len(descriptions) <= 3 {
				applogger.Debugf("[parseInterfaceDescriptions] 解析到: interface=%s admin=%s oper=%s desc='%s'",
					desc.InterfaceName, desc.AdminStatus, desc.OperStatus, desc.Description)
			}
		}
	}

	applogger.Debugf("[parseInterfaceDescriptions] 共解析到 %d 条接口描述", len(descriptions))
	return descriptions, nil
}

// normalizeHuaweiPHYStatus 把华为/H3C 的 PHY 字段（来自 display interface description）
// 归一化为统一的 (AdminStatus, OperStatus)，语义对齐锐捷基准：
//
//   - 手动 shutdown：PHY=`*down` → admin=down, oper=down
//   - 去 shutdown + 无网线：PHY=`down`  → admin=up,   oper=down
//   - 连网线：             PHY=`up`    → admin=up,   oper=up
//
// `*` 前缀表示 administratively down（管理关闭），由 AdminStatus 承载；
// 去掉 `*` 后的 up/down 为物理链路状态，由 OperStatus 承载。
// PROTOCOL 字段（数据链路层协议状态）不参与计算，避免与锐捷语义错位。
//
// 空 PHY 返回两个空串（与原行为一致，由调用方决定如何处理）。
func normalizeHuaweiPHYStatus(phy string) (adminStatus, operStatus string) {
	normalizedPHY := strings.ToLower(strings.TrimSpace(phy))
	if normalizedPHY == "" {
		return "", ""
	}
	if strings.HasPrefix(normalizedPHY, "*") {
		return "down", strings.TrimPrefix(normalizedPHY, "*")
	}
	return "up", normalizedPHY
}

// getAllDot1xStatus 批量获取所有端口的802.1X状态
func getAllDot1xStatus(wrapper *device.ScrapliWrapper, vendor models.DeviceVendor, cache *TemplateCache) (map[string]Dot1xInfo, error) {
	command := getDot1xBatchCommand(vendor)

	resp, err := wrapper.SendCommand(command, true)
	if err != nil {
		return nil, err
	}

	templatePath := getDot1xBatchTemplate(vendor)
	fsm, err := cache.Get(templatePath)
	if err != nil {
		return nil, err
	}

	records, err := fsm.ParseText(resp.Result)
	if err != nil {
		return nil, err
	}

	result := make(map[string]Dot1xInfo)
	for _, record := range records {
		ifaceName := NormalizeInterfaceName(record["INTERFACE"])

		if vendor == models.VendorHuawei || vendor == models.VendorH3C {
			// 从 display dot1x 命令解析
			dot1xEnabledStr := record["DOT1X_ENABLED"]
			dot1xEnabled := dot1xEnabledStr == "Enabled"

			// 如果 DOT1X_ENABLED 为空，默认为 true（因为模板只匹配 protocol is Enabled 的接口）
			if dot1xEnabledStr == "" {
				dot1xEnabled = true
			}

			currentUsersStr := record["CURRENT_USERS"]

			// 根据当前用户数确定端口状态
			var portStatus string = "Unauthorized"
			if dot1xEnabled && currentUsersStr != "" && currentUsersStr != "0" {
				portStatus = "Authorized"
			}

			result[ifaceName] = Dot1xInfo{
				Enabled:     dot1xEnabled,
				PortStatus:  portStatus,
				CurrentUser: currentUsersStr,
			}
		} else {
			// 锐捷/迈普等厂商的解析逻辑保持不变
			authened := record["AUTHENED"] == "yes"
			portStatus := "Unauthorized"
			if authened {
				portStatus = "Authorized"
			}

			result[ifaceName] = Dot1xInfo{
				Enabled:     true,
				PortStatus:  portStatus,
				MaxUser:     record["MAX_USER"], // 未采集(unlimited)留空,由 buildPortStatus 决定是否置 nil
				CurrentUser: record["DYNAMIC_USER"],
			}
		}
	}
	return result, nil
}

// getAllPortSecurity 批量获取所有端口的安全配置
func getAllPortSecurity(wrapper *device.ScrapliWrapper, vendor models.DeviceVendor, cache *TemplateCache) (map[string]PortSecurityInfo, error) {
	command := getPortSecurityBatchCommand(vendor)

	resp, err := wrapper.SendCommand(command, true)
	if err != nil {
		return nil, err
	}

	templatePath := getPortSecurityBatchTemplate(vendor)
	fsm, err := cache.Get(templatePath)
	if err != nil {
		return nil, err
	}

	records, err := fsm.ParseText(resp.Result)
	if err != nil {
		return nil, err
	}

	if len(records) == 0 {
		return make(map[string]PortSecurityInfo), nil
	}

	result := make(map[string]PortSecurityInfo)

	if vendor == models.VendorRuijie || vendor == models.VendorMaipu {
		portMACCount := make(map[string]int)
		for _, record := range records {
			portName := NormalizeInterfaceName(record["PORT"])
			portMACCount[portName]++
		}

		for portName, count := range portMACCount {
			result[portName] = PortSecurityInfo{
				Enabled:    true,
				CurrentMAC: count,
				MaxMAC:     0,
			}
		}
	} else {
		for _, record := range records {
			ifaceName := NormalizeInterfaceName(record["INTERFACE"])
			maxMAC, _ := strconv.Atoi(record["MAX_MAC"])
			currentMAC, _ := strconv.Atoi(record["CURRENT_MAC"])

			result[ifaceName] = PortSecurityInfo{
				Enabled:      record["ENABLED"] == "enabled",
				SecurityMode: record["SECURITY_MODE"],
				MaxMAC:       maxMAC,
				CurrentMAC:   currentMAC,
			}
		}
	}
	return result, nil
}

// getInterfaceCommand 根据厂商获取接口命令
func getInterfaceCommand(vendor models.DeviceVendor) string {
	commands := map[models.DeviceVendor]string{
		models.VendorHuawei: "display interface brief",
		models.VendorH3C:    "display interface",
		models.VendorRuijie: "show interfaces status",
		models.VendorMaipu:  "show interface",
	}

	if cmd, ok := commands[vendor]; ok {
		return cmd
	}
	return "show interface"
}

// getInterfaceTemplate 根据厂商获取接口解析模板
func getInterfaceTemplate(vendor models.DeviceVendor) string {
	templates := map[models.DeviceVendor]string{
		models.VendorHuawei: "templates/huawei_vrp_display_interface_brief.textfsm",
		models.VendorH3C:    "templates/huawei_vrp_display_interface.textfsm",
		models.VendorRuijie: "templates/ruijie_os_show_interfaces_status.textfsm",
		models.VendorMaipu:  "templates/ruijie_os_show_interfaces_status.textfsm",
	}

	if tmpl, ok := templates[vendor]; ok {
		return tmpl
	}
	return "templates/huawei_vrp_display_interface.textfsm"
}

// getInterfaceDescriptionCommand 根据厂商获取接口描述命令
func getInterfaceDescriptionCommand(vendor models.DeviceVendor) string {
	commands := map[models.DeviceVendor]string{
		models.VendorHuawei: "display interface description",
		models.VendorH3C:    "display interface description",
		models.VendorRuijie: "show int des",
		models.VendorMaipu:  "show int des",
	}

	if cmd, ok := commands[vendor]; ok {
		return cmd
	}
	return "show int des"
}

// getInterfaceDescriptionTemplate 根据厂商获取接口描述解析模板
func getInterfaceDescriptionTemplate(vendor models.DeviceVendor) string {
	templates := map[models.DeviceVendor]string{
		models.VendorHuawei: "templates/huawei_vrp_display_interface_description.textfsm",
		models.VendorH3C:    "templates/huawei_vrp_display_interface_description.textfsm",
		models.VendorRuijie: "templates/ruijie_os_show_interface_description.textfsm",
		models.VendorMaipu:  "templates/ruijie_os_show_interface_description.textfsm",
	}

	if tmpl, ok := templates[vendor]; ok {
		return tmpl
	}
	return "templates/ruijie_os_show_interface_description.textfsm"
}

// getDot1xBatchCommand 获取批量802.1X命令
func getDot1xBatchCommand(vendor models.DeviceVendor) string {
	commands := map[models.DeviceVendor]string{
		models.VendorHuawei: "display dot1x",
		models.VendorH3C:    "display dot1x",
		models.VendorRuijie: "show dot1x port-control",
		models.VendorMaipu:  "show dot1x port-control",
	}
	if cmd, ok := commands[vendor]; ok {
		return cmd
	}
	return "display dot1x"
}

// getPortSecurityBatchCommand 获取批量端口安全命令
func getPortSecurityBatchCommand(vendor models.DeviceVendor) string {
	commands := map[models.DeviceVendor]string{
		models.VendorHuawei: "display port-security",
		models.VendorH3C:    "display port-security",
		models.VendorRuijie: "show port-security all",
		models.VendorMaipu:  "show port-security all",
	}
	if cmd, ok := commands[vendor]; ok {
		return cmd
	}
	return "display port-security"
}

// getDot1xBatchTemplate 获取批量802.1X模板路径
func getDot1xBatchTemplate(vendor models.DeviceVendor) string {
	templates := map[models.DeviceVendor]string{
		models.VendorHuawei: "templates/huawei_vrp_display_dot1x.textfsm",
		models.VendorH3C:    "templates/huawei_vrp_display_dot1x.textfsm",
		models.VendorRuijie: "templates/ruijie_os_show_dot1x.textfsm",
		models.VendorMaipu:  "templates/ruijie_os_show_dot1x.textfsm",
	}
	if tmpl, ok := templates[vendor]; ok {
		return tmpl
	}
	return "templates/huawei_vrp_display_dot1x.textfsm"
}

// getPortSecurityBatchTemplate 获取批量端口安全模板路径
func getPortSecurityBatchTemplate(vendor models.DeviceVendor) string {
	templates := map[models.DeviceVendor]string{
		models.VendorHuawei: "templates/huawei_vrp_display_port-security.textfsm",
		models.VendorH3C:    "templates/huawei_vrp_display_port-security.textfsm",
		models.VendorRuijie: "templates/ruijie_os_show_port-security.textfsm",
		models.VendorMaipu:  "templates/ruijie_os_show_port-security.textfsm",
	}
	if tmpl, ok := templates[vendor]; ok {
		return tmpl
	}
	return "templates/huawei_vrp_display_port-security.textfsm"
}

// parseInterfaceVLANInfo 解析华为/H3C设备的接口VLAN信息
func parseInterfaceVLANInfo(wrapper *device.ScrapliWrapper, vendor models.DeviceVendor) (map[string]*int, error) {
	// 只有华为和H3C设备需要额外的VLAN解析
	if vendor != models.VendorHuawei && vendor != models.VendorH3C {
		return make(map[string]*int), nil
	}

	command := "display port vlan"
	resp, err := wrapper.SendCommand(command, true)
	if err != nil {
		// 如果命令失败，返回空map而不是错误（VLAN信息可选）
		return make(map[string]*int), nil
	}

	templatePath := "templates/huawei_vrp_display_port_vlan.textfsm"
	fsm, err := templates.ParseTemplate(templatePath)
	if err != nil {
		return make(map[string]*int), nil
	}

	records, err := fsm.ParseText(resp.Result)
	if err != nil {
		return make(map[string]*int), nil
	}

	vlanMap := make(map[string]*int)
	for _, record := range records {
		ifaceName := NormalizeInterfaceName(record["INTERFACE"])
		
		// 优先使用PVID (VLAN_ID)
		if vlanStr, ok := record["VLAN_ID"]; ok && vlanStr != "" && vlanStr != "0" {
			if vlan, err := strconv.Atoi(vlanStr); err == nil {
				vlanMap[ifaceName] = &vlan
			}
		}
	}

	return vlanMap, nil
}
