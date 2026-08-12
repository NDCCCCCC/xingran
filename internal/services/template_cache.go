package services

import (
	"sync"

	"github.com/xingran-next/xingran-go-backend/internal/templates"
)

// TemplateCache TextFSM模板缓存
type TemplateCache struct {
	sync.RWMutex
	templates map[string]*templates.FSM
}

// NewTemplateCache 创建模板缓存
func NewTemplateCache() *TemplateCache {
	return &TemplateCache{
		templates: make(map[string]*templates.FSM),
	}
}

// Get 获取缓存的模板（线程安全，双重检查锁定）
func (c *TemplateCache) Get(templatePath string) (*templates.FSM, error) {
	// 读锁检查
	c.RLock()
	if fsm, ok := c.templates[templatePath]; ok {
		c.RUnlock()
		return fsm, nil
	}
	c.RUnlock()

	// 写锁加载
	c.Lock()
	defer c.Unlock()

	// 双重检查
	if fsm, ok := c.templates[templatePath]; ok {
		return fsm, nil
	}

	// 加载并缓存
	fsm, err := templates.ParseTemplate(templatePath)
	if err != nil {
		return nil, err
	}
	c.templates[templatePath] = fsm
	return fsm, nil
}

// Clear 清空缓存
func (c *TemplateCache) Clear() {
	c.Lock()
	defer c.Unlock()
	c.templates = make(map[string]*templates.FSM)
}
