package lldp

import (
	"fmt"
	"strings"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
)

// LLDPParser LLDP解析器
type LLDPParser struct {
	templateCache *TemplateCache
}

// NewLLDPParser 创建LLDP解析器
func NewLLDPParser() *LLDPParser {
	return &LLDPParser{
		templateCache: NewTemplateCache(),
	}
}

// ParseLLDPNeighbors 解析LLDP邻居信息输出
func (p *LLDPParser) ParseLLDPNeighbors(output string, vendor models.DeviceVendor) ([]*models.LLDPNeighborInfo, error) {
	if strings.TrimSpace(output) == "" {
		return []*models.LLDPNeighborInfo{}, nil
	}

	templatePath := p.getTemplatePath(vendor)
	fsm, err := p.templateCache.Get(templatePath)
	if err != nil {
		applogger.Errorf("Failed to get LLDP template %s: %v", templatePath, err)
		return nil, fmt.Errorf("failed to get LLDP template: %w", err)
	}

	records, err := fsm.ParseText(output)
	if err != nil {
		applogger.Errorf("Failed to parse LLDP output with template %s: %v", templatePath, err)
		return nil, fmt.Errorf("failed to parse LLDP output: %w", err)
	}

	// 2026-07-01: 初始化为空 slice(非 nil),让格式错误/无匹配的输入返回
	// 非 nil 空切片,匹配 TestParseMalformedLLDPOutput 的 NotNil 断言。
	neighbors := make([]*models.LLDPNeighborInfo, 0)
	for _, record := range records {
		neighbor := &models.LLDPNeighborInfo{}

		// 提取本地接口
		if localInterface, ok := record["LOCAL_INTERFACE"]; ok && localInterface != "" {
			neighbor.LocalInterface = strings.TrimSpace(localInterface)
		} else {
			// 尝试其他可能的键名
			if li, ok := record["LocalInterface"]; ok && li != "" {
				neighbor.LocalInterface = strings.TrimSpace(li)
			} else {
				continue // 跳过没有本地接口的记录
			}
		}

		// 提取邻居ID（Chassis ID）
		if neighborID, ok := record["NEIGHBOR_ID"]; ok && neighborID != "" {
			neighbor.NeighborID = strings.TrimSpace(neighborID)
		}

		// 提取邻居端口
		if neighborPort, ok := record["NEIGHBOR_PORT"]; ok && neighborPort != "" {
			neighbor.NeighborInterface = strings.TrimSpace(neighborPort)
		}

		// 提取邻居名称（可选）
		if neighborName, ok := record["NEIGHBOR_NAME"]; ok && neighborName != "" {
			neighbor.NeighborName = strings.TrimSpace(neighborName)
		}

		// 提取能力（可选）
		if capabilities, ok := record["CAPABILITIES"]; ok && capabilities != "" {
			neighbor.Capabilities = strings.TrimSpace(capabilities)
		}

		neighbors = append(neighbors, neighbor)
	}

	return neighbors, nil
}

// getTemplatePath 根据厂商获取模板路径
func (p *LLDPParser) getTemplatePath(vendor models.DeviceVendor) string {
	templates := map[models.DeviceVendor]string{
		models.VendorHuawei: "templates/lldp/lldp_huawei.textfsm",
		models.VendorH3C:    "templates/lldp/lldp_huawei.textfsm",
		models.VendorRuijie: "templates/lldp/lldp_ruijie.textfsm",
		models.VendorMaipu:  "templates/lldp/lldp_ruijie.textfsm",
	}

	if tmpl, ok := templates[vendor]; ok {
		return tmpl
	}
	return "templates/lldp/lldp_ruijie.textfsm" // 默认使用Ruijie模板
}
