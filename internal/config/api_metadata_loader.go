package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// APIMetadataConfig API元数据配置
type APIMetadataConfig struct {
	Version  string           `yaml:"version"`
	Metadata []ModuleMetadata `yaml:"metadata"`
}

// ModuleMetadata 模块元数据
type ModuleMetadata struct {
	Module    string         `yaml:"module"`
	Category  string         `yaml:"category"`
	Icon      string         `yaml:"icon"`
	Endpoints []EndpointMeta `yaml:"endpoints"`
}

// EndpointMeta 端点元数据
type EndpointMeta struct {
	Route            string            `yaml:"route"`
	Method           string            `yaml:"method"`
	DisplayName      string            `yaml:"displayName"`
	Description      string            `yaml:"description"`
	DataType         string            `yaml:"dataType"` // paginated/single
	DataPath         string            `yaml:"dataPath"`
	SupportedWidgets []string          `yaml:"supportedWidgets"`
	Permissions      []string          `yaml:"permissions"`
	ExampleParams    map[string]string `yaml:"exampleParams"`
}

// LoadAPIMetadata 从YAML文件加载API元数据
func LoadAPIMetadata(path string) (*APIMetadataConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("读取API元数据配置失败: %w", err)
	}

	var config APIMetadataConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("解析API元数据配置失败: %w", err)
	}

	return &config, nil
}

// GetEndpointByRoute 根据路由和方法获取端点元数据
func (c *APIMetadataConfig) GetEndpointByRoute(route, method string) *EndpointMeta {
	for _, module := range c.Metadata {
		for _, endpoint := range module.Endpoints {
			if endpoint.Route == route && endpoint.Method == method {
				return &endpoint
			}
		}
	}
	return nil
}

// GetAllEndpoints 获取所有端点元数据（按模块分组）
func (c *APIMetadataConfig) GetAllEndpoints() []ModuleMetadata {
	return c.Metadata
}
