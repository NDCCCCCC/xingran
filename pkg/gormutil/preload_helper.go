package gormutil

import (
	"strings"

	"gorm.io/gorm"
)

// PreloadConfig 预加载配置
type PreloadConfig struct {
	Path      string
	Condition string
	Args      []interface{}
}

// PreloadItem 预加载项，用于链式调用
type PreloadItem struct {
	builder *PreloadBuilder
	config  PreloadConfig
}

// PreloadBuilder 预加载构建器
type PreloadBuilder struct {
	configs []PreloadConfig
}

// NewPreloadBuilder 创建预加载构建器
func NewPreloadBuilder() *PreloadBuilder {
	return &PreloadBuilder{
		configs: make([]PreloadConfig, 0),
	}
}

// Add 添加预加载路径
func (b *PreloadBuilder) Add(path string) *PreloadBuilder {
	config := PreloadConfig{Path: path}
	b.configs = append(b.configs, config)
	return b
}

// WithCondition 添加预加载条件
func (i *PreloadItem) WithCondition(condition string, args ...interface{}) *PreloadItem {
	i.config.Condition = condition
	i.config.Args = args
	// 更新builder中的最后一个配置
	i.builder.configs[len(i.builder.configs)-1] = i.config
	return i
}

// Apply 应用所有预加载配置到DB
func (b *PreloadBuilder) Apply(db *gorm.DB) *gorm.DB {
	for _, config := range b.configs {
		if config.Condition != "" {
			// 构建预加载参数列表
			args := make([]interface{}, 0, len(config.Args)+1)
			args = append(args, config.Condition)
			args = append(args, config.Args...)
			db = db.Preload(config.Path, args...)
		} else {
			db = db.Preload(config.Path)
		}
	}
	return db
}

// GetConfigs 获取所有预加载配置
func (b *PreloadBuilder) GetConfigs() []PreloadConfig {
	return b.configs
}

// BuildPreloadPath 构建嵌套预加载路径
func BuildPreloadPath(paths ...string) string {
	return strings.Join(paths, ".")
}
