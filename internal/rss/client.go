package rss

import (
	"fmt"
	"time"

	"github.com/mmcdole/gofeed"
)

type Client struct {
	rssURL string
}

func New(rssURL string) *Client {
	return &Client{rssURL: rssURL}
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
	fp := gofeed.NewParser()
	feed, err := fp.ParseURL(c.rssURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse RSS feed: %w", err)
	}

	var items []Item
	for _, item := range feed.Items {
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

		// 尝试从enclosure获取文件大小
		if len(item.Enclosures) > 0 {
			// Length是string类型，这里不做转换，Size默认为0
			if rssItem.URL == "" {
				rssItem.URL = item.Enclosures[0].URL
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
