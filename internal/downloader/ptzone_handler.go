package downloader

import (
	"errors"
	"net/http"
)

// PTZoneHandler PTZone专用下载处理器
type PTZoneHandler struct{}

// NewPTZoneHandler 创建PTZone下载处理器
func NewPTZoneHandler() *PTZoneHandler {
	return &PTZoneHandler{}
}

// GetHandlerType 返回处理器类型
func (h *PTZoneHandler) GetHandlerType() string {
	return "ptzone"
}

// BeforeDownload 下载前的预处理
func (h *PTZoneHandler) BeforeDownload(req *http.Request, task DownloadTask) error {
	// PTZone需要特定的 User-Agent
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	req.Header.Set("Referer", "https://ptzone.xyz/")
	return nil
}

// AfterDownload 下载后的后处理
func (h *PTZoneHandler) AfterDownload(filePath string, task DownloadTask) error {
	// PTZone不需要特殊后处理
	return nil
}

// CheckResponse 检查响应是否有效
func (h *PTZoneHandler) CheckResponse(resp *http.Response) error {
	if resp.StatusCode != http.StatusOK {
		return errors.New("unexpected status code")
	}
	return nil
}
