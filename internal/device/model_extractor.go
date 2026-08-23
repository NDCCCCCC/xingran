package device

import (
	"regexp"
	"strings"

	"github.com/xingran-next/xingran-go-backend/internal/models"
)

// ModelExtractor 型号提取器
type ModelExtractor struct {
	vendor   models.DeviceVendor
	sysDescr string
}

// NewModelExtractor 创建型号提取器
func NewModelExtractor(sysDescr string, vendor models.DeviceVendor) *ModelExtractor {
	return &ModelExtractor{
		vendor:   vendor,
		sysDescr: sysDescr,
	}
}

// Extract 提取设备型号
func (e *ModelExtractor) Extract() string {
	if e.sysDescr == "" {
		return ""
	}

	switch e.vendor {
	case models.VendorHuawei:
		return e.extractHuaweiModel()
	case models.VendorH3C:
		return e.extractH3CModel()
	case models.VendorRuijie:
		return e.extractRuijieModel()
	case models.VendorMaipu:
		return e.extractMaipuModel()
	default:
		return e.extractGenericModel()
	}
}

// trimAnchorPrefix 去掉正则匹配结果中来自锚点 `(?:^|[\s\r\n])` 的前导空白字符。
func trimAnchorPrefix(match string) string {
	return strings.TrimLeft(match, " \t\r\n\f")
}

// extractHuaweiModel 提取华为设备型号
// 华为设备型号通常如: S5735-L48T4X-A, S5700-28P-LI-AC, AR2220, USG6000等
func (e *ModelExtractor) extractHuaweiModel() string {
	// 华为常见型号正则模式（不使用\b，Go的正则引擎不支持）
	patterns := []string{
		// S系列交换机: S5735-L48T4X-A, S5700-28P-LI-AC, S6720-30C-EI-24S-DC等
		`(?:^|[\s\r\n])S[0-9]{4,5}[A-Z0-9\-]+`,
		// AR系列路由器: AR2220, AR1220E, AR1220V等
		`(?:^|[\s\r\n])AR[0-9]{3,4}[A-Z]*`,
		// USG系列防火墙: USG6000, USG6680, USG67xx等
		`(?:^|[\s\r\n])USG[0-9]{4,5}(?:-[A-Z0-9]+)?`,
		// AirEngine系列AP: AirEngine5760-10, AirEngine5761-21等
		`(?:^|[\s\r\n])AirEngine[0-9]{4,6}(?:-[A-Z0-9]+)?`,
		// AP系列: AP4030DN, AP5030DN, AP6010SN等
		`(?:^|[\s\r\n])AP[0-9]{4}[A-Z]{2}`,
		// NetEngine系列路由器: NetEngine8000, NetEngine40等
		`(?:^|[\s\r\n])NetEngine[0-9]{4}`,
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		if match := re.FindString(e.sysDescr); match != "" {
			// 移除前导空白
			model := regexp.MustCompile(`^[A-Z0-9\-]+`).FindString(trimAnchorPrefix(match))
			if len(model) > 0 && len(model) <= 50 {
				return model
			}
		}
	}

	return ""
}

// extractH3CModel 提取H3C设备型号
// H3C设备型号通常如: S5120-28P-SI, S12508, MSR3640, F1000等
func (e *ModelExtractor) extractH3CModel() string {
	patterns := []string{
		// S系列交换机: S5120-28P-SI, S12508, S10500等
		`(?:^|[\s\r\n])S[0-9]{4,5}[A-Z0-9\-]+`,
		// MSR系列路由器: MSR3640, MSR3020, MSR810等
		`(?:^|[\s\r\n])MSR[0-9]{3,4}`,
		// F系列防火墙: F1000, F5000-A, F5040等
		`(?:^|[\s\r\n])F[0-9]{4,5}(?:-[A-Z])?`,
		// WA系列AP: WA4320, WA5320, WA6620等
		`(?:^|[\s\r\n])WA[0-9]{4,5}`,
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		if match := re.FindString(e.sysDescr); match != "" {
			model := regexp.MustCompile(`^[A-Z0-9\-]+`).FindString(trimAnchorPrefix(match))
			if len(model) > 0 && len(model) <= 50 {
				return model
			}
		}
	}

	return ""
}

// extractRuijieModel 提取锐捷设备型号
// 锐捷设备型号通常如: RG-S5750-28GT-P-S, RSR20-04, RG-AP640等
func (e *ModelExtractor) extractRuijieModel() string {
	patterns := []string{
		// RG-S系列交换机: RG-S5750-28GT-P-S, RG-S6220-48GT4XS-P-S等
		`(?:^|[\s\r\n])RG-S[0-9]{4,5}[A-Z0-9\-]+`,
		// RSR系列路由器: RSR20-04, RSR50-04, RSR30-44等
		`(?:^|[\s\r\n])RSR[0-9]{2,4}(?:-[0-9]{2})?`,
		// RG-WALL防火墙: RG-WALL1600, RG-WALL2600等
		`(?:^|[\s\r\n])RG-WALL[0-9]{4}`,
		// RG-AP系列AP: RG-AP640, RG-AP720, RG-AP840等
		`(?:^|[\s\r\n])RG-AP[0-9]{3,4}`,
		// RG-EG系列网关: RG-EG2000, RG-EG3000等
		`(?:^|[\s\r\n])RG-EG[0-9]{4}`,
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		if match := re.FindString(e.sysDescr); match != "" {
			model := regexp.MustCompile(`^[A-Z0-9\-]+`).FindString(trimAnchorPrefix(match))
			if len(model) > 0 && len(model) <= 50 {
				return model
			}
		}
	}

	return ""
}

// extractMaipuModel 提取迈普设备型号
// 迈普设备型号通常如: Secure, MyPower, SM等系列
func (e *ModelExtractor) extractMaipuModel() string {
	patterns := []string{
		// Secure系列防火墙: Secure2000, Secure6000等
		`(?:^|[\s\r\n])Secure[0-9]{4}`,
		// MyPower系列: MyPower S3000, MyPower S4000等
		`(?:^|[\s\r\n])MyPower\s*[A-Z0-9\-]+`,
		// SM系列交换机: SM2800, SM3900, SM4900等
		`(?:^|[\s\r\n])SM[0-9]{4}`,
		// MP系列路由器: MP2800, MP3800, MP2815等
		`(?:^|[\s\r\n])MP[0-9]{4}`,
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		if match := re.FindString(e.sysDescr); match != "" {
			model := regexp.MustCompile(`^[A-Z0-9\s\-]+`).FindString(trimAnchorPrefix(match))
			model = strings.TrimRight(model, " \t\r\n\f")
			if len(model) > 0 && len(model) <= 50 {
				return model
			}
		}
	}

	return ""
}

// extractGenericModel 通用型号提取
// 适用于未知厂商，尝试提取类似设备型号的字符串
func (e *ModelExtractor) extractGenericModel() string {
	patterns := []string{
		// 大写字母+数字组合: S5700, AR2220, RG-S5750等
		`(?:^|[\s\r\n])[A-Z]{1,4}[0-9]{3,6}[A-Z0-9\-]+`,
		// XX-XXXX格式: RG-S5750, H3C-S5120等
		`(?:^|[\s\r\n])[A-Z]{2,}-[A-Z0-9\-]+`,
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		if match := re.FindString(e.sysDescr); match != "" {
			model := regexp.MustCompile(`^[A-Z0-9\-]+`).FindString(trimAnchorPrefix(match))
			if len(model) > 0 && len(model) <= 50 {
				return model
			}
		}
	}

	return ""
}

// IdentifyVendor 从sysDescr识别设备厂商
// 只返回已定义的4个厂商，其他情况返回空字符串
func IdentifyVendor(sysDescr string) models.DeviceVendor {
	descr := toUpper(sysDescr)

	vendors := []struct {
		name   string
		vendor models.DeviceVendor
	}{
		{"HUAWEI", models.VendorHuawei},
		{"H3C", models.VendorH3C},
		{"RUIJIE", models.VendorRuijie},
		{"MAIPU", models.VendorMaipu},
	}

	for _, v := range vendors {
		if contains(descr, v.name) {
			return v.vendor
		}
	}

	return "" // 未识别的厂商返回空字符串
}

// IdentifyDeviceType 从sysDescr识别设备类型
func IdentifyDeviceType(sysDescr string) models.DeviceType {
	descr := string(sysDescr)

	switch {
	case containsIgnoreCase(descr, "Router"):
		return models.DeviceTypeRouter
	case containsIgnoreCase(descr, "Switch"):
		return models.DeviceTypeSwitch
	case containsIgnoreCase(descr, "Firewall"), containsIgnoreCase(descr, "USG"):
		return models.DeviceTypeFirewall
	case containsIgnoreCase(descr, "AP"), containsIgnoreCase(descr, "Fat AP"), containsIgnoreCase(descr, "Fit AP"), containsIgnoreCase(descr, "Access Point"):
		return models.DeviceTypeAP
	case containsIgnoreCase(descr, "Load Balancer"):
		return models.DeviceTypeLoadBalancer
	default:
		return models.DeviceTypeSwitch // 默认为交换机
	}
}
