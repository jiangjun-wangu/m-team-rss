package downloader

import (
	"fmt"
	"net/http"
	"time"
)

// DownloadHandler 下载处理器接口
type DownloadHandler interface {
	// BeforeDownload 下载前的预处理,可以添加自定义请求头等
	BeforeDownload(req *http.Request, task DownloadTask) error

	// AfterDownload 下载后的后处理,可以验证文件内容等
	AfterDownload(filePath string, task DownloadTask) error

	// CheckResponse 检查响应是否有效
	CheckResponse(resp *http.Response) error
}

// BaseDownloader 基础下载器
type BaseDownloader struct {
	savePath  string
	semaphore chan struct{}
	client    *http.Client
}

// NewBaseDownloader 创建基础下载器
func NewBaseDownloader(savePath string, maxConcurrent int) *BaseDownloader {
	return &BaseDownloader{
		savePath:  savePath,
		semaphore: make(chan struct{}, maxConcurrent),
		client: &http.Client{
			Timeout: 60 * time.Second,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				// 允许最多10次重定向
				if len(via) >= 10 {
					return fmt.Errorf("stopped after 10 redirects")
				}
				// 复制原始请求的 User-Agent
				if len(via) > 0 {
					req.Header.Set("User-Agent", via[0].Header.Get("User-Agent"))
				}
				return nil
			},
		},
	}
}
