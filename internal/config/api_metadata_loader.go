package config

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"
)

// APIMetadataConfig API 元数据配置。
//
// 线程安全:本结构从磁盘加载后只读,所有方法可并发调用。
// Metadata 字段在初始化完成后不再修改;endpointIndex 由 sync.Once
// 保护懒加载,只构建一次。
type APIMetadataConfig struct {
	Version  string           `yaml:"version"`
	Metadata []ModuleMetadata `yaml:"metadata"`

	indexOnce sync.Once
	index     map[string]*EndpointMeta // key = "<METHOD> <ROUTE>",值指向 Metadata 中的原始指针
}

// ModuleMetadata 模块元数据。
type ModuleMetadata struct {
	Module    string         `yaml:"module"`
	Category  string         `yaml:"category"`
	Icon      string         `yaml:"icon"`
	Endpoints []EndpointMeta `yaml:"endpoints"`
}

// EndpointMeta 端点元数据。
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

// endpointIndexKey 生成索引键。
//
// HTTP method 统一大写(GET/POST);Route 由 normalize() 规范化后已带前导 /、
// 无尾部 /。 直接拼接即可。
func (c *APIMetadataConfig) endpointIndexKey(method, route string) string {
	return method + " " + route
}

// buildIndex 懒加载端点索引。
//
// 复杂度:O(N*M) 一次性构建 → 之后查询 O(1)。
// 对于 N=10 个模块 × M=20 端点 = 200 条记录,即使没有索引也很快;
// 但前端仪表盘高频调用 GetEndpointByRoute 时(每次路由匹配都查一次),
// 索引能显著降低延迟。
func (c *APIMetadataConfig) buildIndex() {
	c.index = make(map[string]*EndpointMeta, len(c.Metadata)*8)
	for i := range c.Metadata {
		for j := range c.Metadata[i].Endpoints {
			ep := &c.Metadata[i].Endpoints[j]
			c.index[c.endpointIndexKey(ep.Method, ep.Route)] = ep
		}
	}
}

// normalize 规范化所有端点的 Method / Route,消除大小写和前后缀差异,
//
//	避免索引键重复。
//
// 规则:
//   - Method:  trim 后 ToUpper(GET / POST / ...)
//   - Route:   trim 后剥离前后多余 /,再加单个前导 /
//
// 在 LoadAPIMetadata 解析完成后调用一次,buildIndex() 看到的就是规范化后的字段。
func (c *APIMetadataConfig) normalize() {
	for i := range c.Metadata {
		for j := range c.Metadata[i].Endpoints {
			ep := &c.Metadata[i].Endpoints[j]
			ep.Method = strings.ToUpper(strings.TrimSpace(ep.Method))
			route := strings.TrimSpace(ep.Route)
			route = strings.TrimLeft(route, "/")
			route = strings.TrimRight(route, "/")
			ep.Route = "/" + route
		}
	}
}

// LoadAPIMetadata 从 YAML 文件加载 API 元数据。
//
// 解析完成后会做一次字段规范化(WR-3),消除 Method / Route 的大小写与
// 前后斜杠差异,使索引键在数据源不一致时也能命中同一端点。
//
// ctx 用于取消读文件操作(网络挂载/慢盘场景)。
func LoadAPIMetadata(ctx context.Context, path string) (*APIMetadataConfig, error) {
	// os.ReadFile 不支持 ctx,包成 goroutine + select 让 ctx 能取消磁盘读。
	// 大部分配置文件 < 100KB,select 立即命中 readCh;ctx 取消才会走 ctx.Done 分支。
	type result struct {
		data []byte
		err  error
	}
	readCh := make(chan result, 1)
	go func() {
		data, err := os.ReadFile(path)
		readCh <- result{data, err}
	}()

	var data []byte
	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("读取API元数据配置已取消: %w", ctx.Err())
	case r := <-readCh:
		if r.err != nil {
			return nil, fmt.Errorf("读取API元数据配置失败: %w", r.err)
		}
		data = r.data
	}

	var cfg APIMetadataConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("解析API元数据配置失败: %w", err)
	}

	cfg.normalize()
	return &cfg, nil
}

// GetEndpointByRoute 根据路由和方法获取端点元数据。
//
// 返回 nil 表示未找到,调用方需要处理 nil(典型用法:降级到默认权限)。
//
// 性能:首次调用 O(N*M) 构建索引,之后每次 O(1) 哈希查找。
// 多 goroutine 并发安全。
func (c *APIMetadataConfig) GetEndpointByRoute(route, method string) *EndpointMeta {
	c.indexOnce.Do(c.buildIndex)
	return c.index[c.endpointIndexKey(strings.ToUpper(strings.TrimSpace(method)), route)]
}

// GetAllEndpoints 获取所有端点元数据(按模块分组)。
//
// 返回浅拷贝切片,调用方可以安全地 append 或修改元素,不会影响内部状态。
// 指针字段(*EndpointMeta)仍指向内部,不可修改其内容。
// nil Metadata 时返回空切片(而非 nil),便于调用方 len() == 0 判断。
func (c *APIMetadataConfig) GetAllEndpoints() []ModuleMetadata {
	snapshot := make([]ModuleMetadata, len(c.Metadata))
	copy(snapshot, c.Metadata)
	return snapshot
}