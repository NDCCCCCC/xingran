package portcollection

import (
	"strconv"
	"strings"

	"github.com/ruoyi-next/ruoyi-go-backend/internal/device"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/templates"
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
		info.Name = normalizeInterfaceName(originalName)

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
			desc.InterfaceName = normalizeInterfaceName(trimmedName)
			desc.OriginalName = trimmedName
		}

		if status, ok := record["STATUS"]; ok {
			desc.OperStatus = strings.ToLower(status)
		}

		if admin, ok := record["ADMINISTRATIVE"]; ok {
			desc.AdminStatus = strings.ToLower(admin)
		}

		if description, ok := record["DESCRIPTION"]; ok {
			desc.Description = description
		}

		if desc.InterfaceName != "" {
			descriptions[desc.InterfaceName] = desc
		}
	}

	return descriptions, nil
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
		ifaceName := normalizeInterfaceName(record["INTERFACE"])

		if vendor == models.VendorHuawei || vendor == models.VendorH3C {
			enabled := record["ENABLED"] == "Enabled"
			portStatus := "Unknown"
			if record["CURRENT_USERS"] != "" && record["CURRENT_USERS"] != "0" {
				portStatus = "Authorized"
			} else if record["STATUS"] == "UP" {
				portStatus = "Unauthorized"
			}

			result[ifaceName] = Dot1xInfo{
				Enabled:     enabled,
				PortStatus:  portStatus,
				CurrentUser: record["CURRENT_USERS"],
			}
		} else {
			authened := record["AUTHENED"] == "yes"
			portStatus := "Unauthorized"
			if authened {
				portStatus = "Authorized"
			}

			result[ifaceName] = Dot1xInfo{
				Enabled:     true,
				PortStatus:  portStatus,
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
			portName := normalizeInterfaceName(record["PORT"])
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
			ifaceName := normalizeInterfaceName(record["INTERFACE"])
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
