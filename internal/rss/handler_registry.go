package rss

import (
	"sync"
)

// HandlerRegistry 站点处理器注册表
type HandlerRegistry struct {
	handlers map[string]SiteHandler
	mu       sync.RWMutex
}

// NewHandlerRegistry 创建处理器注册表
func NewHandlerRegistry() *HandlerRegistry {
	registry := &HandlerRegistry{
		handlers: make(map[string]SiteHandler),
	}

	// 注册内置处理器 (使用统一的site_type命名)
	registry.Register(NewMTeamHandler())
	registry.Register(NewPTZoneHandler())
	registry.Register(NewGenericHandler("generic"))

	return registry
}

// Register 注册站点处理器
func (r *HandlerRegistry) Register(handler SiteHandler) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.handlers[handler.GetSiteType()] = handler
}

// Get 获取站点处理器
func (r *HandlerRegistry) Get(siteType string) (SiteHandler, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	handler, exists := r.handlers[siteType]
	if !exists {
		// 如果没有找到专用处理器，返回通用处理器
		return NewGenericHandler(siteType), nil
	}

	return handler, nil
}

// GetByURL 根据RSS URL自动识别并获取处理器
func (r *HandlerRegistry) GetByURL(rssURL string) (SiteHandler, string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// 识别站点类型
	siteType := identifySiteType(rssURL)

	// 获取处理器
	handler, exists := r.handlers[siteType]
	if !exists {
		// 返回通用处理器
		return NewGenericHandler(siteType), siteType, nil
	}

	return handler, siteType, nil
}

// identifySiteType 识别站点类型
func identifySiteType(rssURL string) string {
	// M-Team站点
	if contains(rssURL, "m-team.cc") || contains(rssURL, "kp.m-team.cc") || contains(rssURL, "rss.m-team.io") || contains(rssURL, "m-team") {
		return "mteam"
	}

	// PTzone站点
	if contains(rssURL, "ptzone.xyz") {
		return "ptzone"
	}

	// HDChina站点
	if contains(rssURL, "hdchina") {
		return "hdchina"
	}

	// CHDBits站点
	if contains(rssURL, "chdbits") {
		return "chdbits"
	}

	// HDBits站点
	if contains(rssURL, "hdbits") {
		return "hdbits"
	}

	// 未知站点，返回generic
	return "generic"
}

// contains 简单的字符串包含检查（不区分大小写）
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr ||
		(len(s) > len(substr) && containsSubstring(s, substr)))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// 全局注册表实例
var globalRegistry *HandlerRegistry

// GetGlobalRegistry 获取全局注册表
func GetGlobalRegistry() *HandlerRegistry {
	if globalRegistry == nil {
		globalRegistry = NewHandlerRegistry()
	}
	return globalRegistry
}

// RegisterHandler 注册站点处理器（便捷方法）
func RegisterHandler(handler SiteHandler) {
	GetGlobalRegistry().Register(handler)
}

// GetHandler 获取站点处理器（便捷方法）
func GetHandler(siteType string) (SiteHandler, error) {
	return GetGlobalRegistry().Get(siteType)
}

// GetHandlerByURL 根据URL获取处理器（便捷方法）
func GetHandlerByURL(rssURL string) (SiteHandler, string, error) {
	return GetGlobalRegistry().GetByURL(rssURL)
}
