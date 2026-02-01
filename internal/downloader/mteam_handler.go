package downloader

import (
	"fmt"
	"net/http"
	"strings"
)

// MTeamHandler M-Team专用下载处理器
type MTeamHandler struct{}

// NewMTeamHandler 创建M-Team下载处理器
func NewMTeamHandler() *MTeamHandler {
	return &MTeamHandler{}
}

// GetHandlerType 返回处理器类型
func (h *MTeamHandler) GetHandlerType() string {
	return "mteam"
}

// BeforeDownload 下载前的预处理
func (h *MTeamHandler) BeforeDownload(req *http.Request, task DownloadTask) error {
	// M-Team需要特定的 User-Agent
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Referer", "https://kp.m-team.cc/")
	req.Header.Set("Accept", "application/x-bittorrent, */*")
	return nil
}

// AfterDownload 下载后的后处理
func (h *MTeamHandler) AfterDownload(filePath string, task DownloadTask) error {
	// M-Team不需要特殊后处理
	return nil
}

// CheckResponse 检查响应是否有效
func (h *MTeamHandler) CheckResponse(resp *http.Response) error {
	// M-Team可能返回302重定向到错误页面
	if resp.StatusCode == http.StatusFound || resp.StatusCode == http.StatusMovedPermanently {
		location := resp.Header.Get("Location")
		if strings.Contains(location, "google.com") {
			return fmt.Errorf("M-Team authentication failed, redirecting to Google")
		}
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	return nil
}
