package rss

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/mmcdole/gofeed"
)

// PTZoneHandler PTzone站点处理器
type PTZoneHandler struct{}

// NewPTZoneHandler 创建PTzone处理器
func NewPTZoneHandler() *PTZoneHandler {
	return &PTZoneHandler{}
}

// GetSiteType 返回站点类型
func (h *PTZoneHandler) GetSiteType() string {
	return "ptzone"
}

// CheckEnclosure PTzone优先使用enclosure
func (h *PTZoneHandler) CheckEnclosure() bool {
	return true
}

// ParseDownloadURL 解析并生成下载链接
func (h *PTZoneHandler) ParseDownloadURL(item *gofeed.Item, rssURL string) string {
	// 如果有enclosure，优先使用
	if len(item.Enclosures) > 0 && item.Enclosures[0].URL != "" {
		downloadURL := item.Enclosures[0].URL

		// 从 RSS URL 提取 passkey
		passkey := extractPasskey(rssURL)

		// PTzone：如果RSS URL中有passkey，确保下载链接也有passkey
		if passkey != "" && !strings.Contains(downloadURL, "passkey=") {
			if strings.Contains(downloadURL, "?") {
				downloadURL += "&passkey=" + passkey
			} else {
				downloadURL += "?passkey=" + passkey
			}
		}

		return downloadURL
	}

	// 没有enclosure，从link生成下载链接
	downloadURL := item.Link

	// 从 RSS URL 提取 passkey
	passkey := extractPasskey(rssURL)

	// 提取ID并生成下载链接
	id := extractIDFromURL(downloadURL)
	downloadURL = fmt.Sprintf("https://ptzone.xyz/download.php?id=%s", id)

	// 添加passkey
	if passkey != "" {
		downloadURL += "&passkey=" + passkey
	}

	return downloadURL
}

// BeforeDownload 下载前处理
func (h *PTZoneHandler) BeforeDownload(req *http.Request, task DownloadTask) error {
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	req.Header.Set("Accept", "application/x-bittorrent, application/octet-stream")
	return nil
}

// CheckResponse 检查响应是否有效
func (h *PTZoneHandler) CheckResponse(resp *http.Response) error {
	if resp.StatusCode != 200 {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	return nil
}

// AfterDownload 下载后处理
func (h *PTZoneHandler) AfterDownload(filePath string, task DownloadTask) error {
	return nil
}

// extractPasskey 从RSS URL提取passkey
func extractPasskey(rssURL string) string {
	u, err := url.Parse(rssURL)
	if err != nil {
		return ""
	}
	return u.Query().Get("passkey")
}
