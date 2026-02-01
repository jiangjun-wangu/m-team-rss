package rss

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/mmcdole/gofeed"
)

// GenericHandler 通用站点处理器
type GenericHandler struct {
	siteType string
}

// NewGenericHandler 创建通用处理器
func NewGenericHandler(siteType string) *GenericHandler {
	return &GenericHandler{
		siteType: siteType,
	}
}

// GetSiteType 返回站点类型
func (h *GenericHandler) GetSiteType() string {
	return h.siteType
}

// CheckEnclosure 通用站点优先使用enclosure
func (h *GenericHandler) CheckEnclosure() bool {
	return true
}

// ParseDownloadURL 解析并生成下载链接
func (h *GenericHandler) ParseDownloadURL(item *gofeed.Item, rssURL string) string {
	// 如果有enclosure，优先使用
	if len(item.Enclosures) > 0 && item.Enclosures[0].URL != "" {
		return item.Enclosures[0].URL
	}

	// 没有enclosure，尝试添加dl=1参数
	downloadURL := item.Link
	if strings.Contains(downloadURL, "?") {
		downloadURL += "&dl=1"
	} else {
		downloadURL += "?dl=1"
	}

	return downloadURL
}

// BeforeDownload 下载前处理
func (h *GenericHandler) BeforeDownload(req *http.Request, task DownloadTask) error {
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	return nil
}

// CheckResponse 检查响应是否有效
func (h *GenericHandler) CheckResponse(resp *http.Response) error {
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	return nil
}

// AfterDownload 下载后处理
func (h *GenericHandler) AfterDownload(filePath string, task DownloadTask) error {
	return nil
}
