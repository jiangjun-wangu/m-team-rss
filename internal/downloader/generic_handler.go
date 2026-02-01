package downloader

import (
	"errors"
	"net/http"
)

// GenericHandler 通用下载处理器,适用于大多数站点
type GenericHandler struct{}

// NewGenericHandler 创建通用下载处理器
func NewGenericHandler() *GenericHandler {
	return &GenericHandler{}
}

// GetHandlerType 返回处理器类型
func (h *GenericHandler) GetHandlerType() string {
	return "generic"
}

// BeforeDownload 下载前的预处理
func (h *GenericHandler) BeforeDownload(req *http.Request, task DownloadTask) error {
	// 添加基本 User-Agent
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	return nil
}

// AfterDownload 下载后的后处理
func (h *GenericHandler) AfterDownload(filePath string, task DownloadTask) error {
	// 通用处理器不需要后处理
	return nil
}

// CheckResponse 检查响应是否有效
func (h *GenericHandler) CheckResponse(resp *http.Response) error {
	if resp.StatusCode != http.StatusOK {
		return errors.New("unexpected status code")
	}
	return nil
}
