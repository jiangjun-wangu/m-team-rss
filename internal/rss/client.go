package rss

import (
	"fmt"
	"strconv"
	"time"

	"github.com/mmcdole/gofeed"
)

type Client struct {
	rssURL string
}

func New(rssURL string) *Client {
	return &Client{rssURL: rssURL}
}

func NewClient() *Client {
	return &Client{rssURL: ""}
}


type Item struct {
	GUID     string
	Title    string
	URL      string
	PubDate  time.Time
	Category string
	Size     int64
}

func (c *Client) Fetch() ([]Item, error) {
	return c.FetchWithLimit(0)
}

func (c *Client) FetchWithLimit(limit int) ([]Item, error) {
	fp := gofeed.NewParser()
	feed, err := fp.ParseURL(c.rssURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse RSS feed: %w", err)
	}

	var items []Item
	maxItems := len(feed.Items)
	if limit > 0 && limit < maxItems {
		maxItems = limit
	}

	for i := 0; i < maxItems; i++ {
		item := feed.Items[i]

		guid := item.GUID
		if guid == "" {
			guid = item.Link
		}

		var pubDate time.Time
		if item.PublishedParsed != nil {
			pubDate = *item.PublishedParsed
		}

		rssItem := Item{
			GUID:    guid,
			Title:   item.Title,
			URL:     item.Link,
			PubDate: pubDate,
		}

		// 优先使用 enclosure URL（直接下载链接），而不是 link（详情页链接）
		if len(item.Enclosures) > 0 && item.Enclosures[0].URL != "" {
			rssItem.URL = item.Enclosures[0].URL
		}

		// 获取文件大小
		if len(item.Enclosures) > 0 && item.Enclosures[0].Length != "" {
			if size, err := parseSize(item.Enclosures[0].Length); err == nil {
				rssItem.Size = size
			}
		}

		// 获取分类信息
		if len(item.Categories) > 0 {
			rssItem.Category = item.Categories[0]
		}

		items = append(items, rssItem)
	}

	return items, nil
}

func parseSize(sizeStr string) (int64, error) {
	return strconv.ParseInt(sizeStr, 10, 64)
}
