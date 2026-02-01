package downloader

import (
	"strings"
	"sync"
)

// handlerRegistry 下载处理器注册表
type handlerRegistry struct {
	handlers map[string]DownloadHandler
	mu       sync.RWMutex
}

var (
	registry = &handlerRegistry{
		handlers: make(map[string]DownloadHandler),
	}
)

// RegisterHandler 注册下载处理器
func RegisterHandler(siteType string, handler DownloadHandler) {
	registry.mu.Lock()
	defer registry.mu.Unlock()
	registry.handlers[siteType] = handler
}

// GetHandler 获取下载处理器
func GetHandler(siteType string) DownloadHandler {
	registry.mu.RLock()
	defer registry.mu.RUnlock()

	handler, ok := registry.handlers[siteType]
	if !ok {
		// 返回通用处理器作为默认
		return NewGenericHandler()
	}
	return handler
}

// GetHandlerByURL 根据URL获取下载处理器
func GetHandlerByURL(url string) (DownloadHandler, string, error) {
	registry.mu.RLock()
	defer registry.mu.RUnlock()

	// 根据URL特征判断站点类型
	siteType := "generic"
	if strings.Contains(url, "m-team") || strings.Contains(url, "mt-team") {
		siteType = "mteam"
	} else if strings.Contains(url, "ptzone") {
		siteType = "ptzone"
	}

	handler, ok := registry.handlers[siteType]
	if !ok {
		handler = NewGenericHandler()
	}

	return handler, siteType, nil
}

// init 初始化默认处理器
func init() {
	// 注册默认处理器
	RegisterHandler("generic", NewGenericHandler())
	RegisterHandler("mteam", NewMTeamHandler())
	RegisterHandler("ptzone", NewPTZoneHandler())
}
