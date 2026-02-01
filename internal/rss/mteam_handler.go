package rss

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/mmcdole/gofeed"
)

// MTeamHandler M-Team 站点处理器
type MTeamHandler struct {
}

func NewMTeamHandler() *MTeamHandler {
	return &MTeamHandler{}
}

func (h *MTeamHandler) GetSiteType() string {
	return "mteam"
}

func (h *MTeamHandler) Match(url string) bool {
	return strings.Contains(url, "m-team") ||
		strings.Contains(url, "kp.m-team") ||
		strings.Contains(url, "tp.m-team")
}

// CheckEnclosure M-Team优先使用enclosure
func (h *MTeamHandler) CheckEnclosure() bool {
	return true
}

// ParseDownloadURL 解析并生成下载链接
func (h *MTeamHandler) ParseDownloadURL(item *gofeed.Item, rssURL string) string {
	// 如果有 enclosure URL，使用 RSS 源参数重新构造下载链接
	if len(item.Enclosures) > 0 && item.Enclosures[0].URL != "" {
		enclosureURL := item.Enclosures[0].URL
		
		// 检查是否是新的 dlv2 API
		if strings.Contains(enclosureURL, "/api/rss/dlv2") {
			// 从 RSS URL 提取参数
			sign := extractParam(rssURL, "sign")
			uid := extractParam(rssURL, "uid")
			
			// 从 enclosure URL 提取 tid
			tid := extractParam(enclosureURL, "tid")
			
			// 如果有完整参数，使用 RSS 源的签名重新构造 URL（避免签名过期/错误）
			if sign != "" && uid != "" && tid != "" {
				// 使用当前时间戳，避免链接过期
				currentTime := time.Now().Unix()
				return fmt.Sprintf("https://rss.m-team.io/api/rss/dlv2?sign=%s&tid=%s&t=%d&uid=%s", sign, tid, currentTime, uid)
			}
		}
		// 否则直接使用 enclosure URL
		return enclosureURL
	}

	// 没有 enclosure，尝试从详情页链接构造
	downloadURL := item.Link

	// 从 RSS URL 提取参数
	sign := extractParam(rssURL, "sign")
	uid := extractParam(rssURL, "uid")

	// 提取torrent ID
	id := extractIDFromURL(downloadURL)

	// 如果有 sign 和 uid，使用 RSS API 下载链接（使用当前时间戳）
	if sign != "" && uid != "" && id != "" {
		currentTime := time.Now().Unix()
		return fmt.Sprintf("https://rss.m-team.io/api/rss/dlv2?sign=%s&tid=%s&t=%d&uid=%s", sign, id, currentTime, uid)
	}

	// 否则返回原始链接（详情页）
	return downloadURL
}

// BeforeDownload 下载前处理
func (h *MTeamHandler) BeforeDownload(req *http.Request, task DownloadTask) error {
	// 设置标准的浏览器User-Agent
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "application/x-bittorrent, application/octet-stream, */*")

	// 设置Referer，根据URL类型判断
	if strings.Contains(task.URL, "rss.m-team.io") {
		req.Header.Set("Referer", "https://rss.m-team.io")
	} else if strings.Contains(task.URL, "kp.m-team.cc") {
		req.Header.Set("Referer", "https://kp.m-team.cc")
	}

	return nil
}

// CheckResponse 检查响应是否有效
func (h *MTeamHandler) CheckResponse(resp *http.Response) error {
	// M-Team的API可能返回302重定向，这是正常的
	if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusFound || resp.StatusCode == http.StatusMovedPermanently {
		return nil
	}
	return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
}

// AfterDownload 下载后处理
func (h *MTeamHandler) AfterDownload(filePath string, task DownloadTask) error {
	return nil
}

// extractParam 从URL中提取指定参数值
func extractParam(rawURL, param string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	return u.Query().Get(param)
}

// extractIDFromURL 从URL中提取ID
func extractIDFromURL(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	path := u.Path
	parts := strings.Split(path, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return ""
}
