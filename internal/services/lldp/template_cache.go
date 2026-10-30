package lldp

import (
	"sync"

	"github.com/xingran-next/xingran-go-backend/internal/templates"
)

// TemplateCache 模板缓存
type TemplateCache struct {
	cache map[string]*templates.FSM
	mu    sync.RWMutex
}

// NewTemplateCache 创建模板缓存
func NewTemplateCache() *TemplateCache {
	return &TemplateCache{
		cache: make(map[string]*templates.FSM),
	}
}

// Get 获取缓存中的模板
func (c *TemplateCache) Get(templatePath string) (*templates.FSM, error) {
	c.mu.RLock()
	fsm, ok := c.cache[templatePath]
	c.mu.RUnlock()

	if ok {
		return fsm, nil
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// 双重检查
	fsm, ok = c.cache[templatePath]
	if ok {
		return fsm, nil
	}

	// 加载并缓存模板
	fsm, err := templates.ParseTemplate(templatePath)
	if err != nil {
		return nil, err
	}

	c.cache[templatePath] = fsm
	return fsm, nil
}
