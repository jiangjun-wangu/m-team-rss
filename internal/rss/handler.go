package rss

import (
	"net/http"

	"github.com/mmcdole/gofeed"
)

// SiteHandler 站点处理器接口
type SiteHandler interface {
	// GetSiteType 返回站点类型标识
	GetSiteType() string

	// ParseDownloadURL 解析并生成下载链接
	ParseDownloadURL(item *gofeed.Item, rssURL string) string

	// CheckEnclosure 检查是否优先使用enclosure
	CheckEnclosure() bool

	// BeforeDownload 下载前的处理（设置headers等）
	BeforeDownload(req *http.Request, task DownloadTask) error

	// CheckResponse 检查响应是否有效
	CheckResponse(resp *http.Response) error

	// AfterDownload 下载后的处理
	AfterDownload(filePath string, task DownloadTask) error
}

// DownloadTask 下载任务（定义在rss包中,避免循环依赖）
type DownloadTask struct {
	GUID    string
	Title   string
	URL     string
	PubDate interface{}
}
