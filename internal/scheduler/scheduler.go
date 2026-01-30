package scheduler

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
	"mteam-rss/internal/database"
	"mteam-rss/internal/downloader"
	"mteam-rss/internal/rss"
)

type Scheduler struct {
	cron       *cron.Cron
	rssClient  *rss.Client
	downloader *downloader.Downloader
	db         *database.Database
	mu         sync.RWMutex
	running    bool
	lastFetch  time.Time
	nextFetch  time.Time
	lastError  string
}

func New(rssURL string, pollInterval time.Duration, savePath string, maxConcurrent int, db *database.Database) *Scheduler {
	return &Scheduler{
		cron:       cron.New(),
		rssClient:  rss.New(rssURL),
		downloader: downloader.New(savePath, maxConcurrent),
		db:         db,
		running:    false,
	}
}

func (s *Scheduler) Start(pollInterval time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return fmt.Errorf("scheduler is already running")
	}

	// 添加定时任务
	schedule := fmt.Sprintf("@every %s", pollInterval.String())
	_, err := s.cron.AddFunc(schedule, s.runJob)
	if err != nil {
		return fmt.Errorf("failed to add cron job: %w", err)
	}

	s.running = true
	s.cron.Start()
	s.nextFetch = time.Now().Add(pollInterval)

	log.Printf("Scheduler started with interval: %v", pollInterval)
	return nil
}

func (s *Scheduler) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return
	}

	s.cron.Stop()
	s.running = false
	log.Println("Scheduler stopped")
}

func (s *Scheduler) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.running
}

func (s *Scheduler) GetLastFetch() time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.lastFetch
}

func (s *Scheduler) GetNextFetch() time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.nextFetch
}

func (s *Scheduler) GetLastError() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.lastError
}

func (s *Scheduler) runJob() {
	log.Println("Starting RSS fetch job...")

	// 获取RSS项目
	items, err := s.rssClient.Fetch()
	if err != nil {
		s.mu.Lock()
		s.lastError = err.Error()
		s.mu.Unlock()

		// 更新数据库状态
		s.db.UpdateSystemStatus(s.GetLastFetch(), time.Now().Add(5*time.Minute), err.Error())
		log.Printf("Failed to fetch RSS: %v", err)
		return
	}

	log.Printf("Fetched %d RSS items", len(items))

	// 准备下载任务
	var tasks []downloader.DownloadTask
	var cacheItems []database.RSSItemCache

	for _, item := range items {
		tasks = append(tasks, downloader.DownloadTask{
			GUID:    item.GUID,
			Title:   item.Title,
			URL:     item.URL,
			PubDate: item.PubDate,
		})

		cacheItems = append(cacheItems, database.RSSItemCache{
			GUID:     item.GUID,
			Title:    item.Title,
			URL:      item.URL,
			PubDate:  item.PubDate,
			Category: item.Category,
			Size:     item.Size,
		})
	}

	// 批量下载
	results := s.downloader.DownloadBatch(tasks, s.db)

	// 统计结果
	successCount := 0
	skipCount := 0
	failCount := 0
	var firstError error

	for _, result := range results {
		if result.Success {
			if result.Bytes > 0 {
				successCount++
			} else {
				skipCount++
			}
		} else {
			failCount++
			if firstError == nil {
				firstError = result.Error
			}
		}
	}

	// 缓存RSS项目
	if err := s.db.CacheRSSItems(cacheItems); err != nil {
		log.Printf("Failed to cache RSS items: %v", err)
	}

	// 清理过期缓存（24小时前）
	if err := s.db.CleanExpiredCache(time.Now().Add(-24 * time.Hour)); err != nil {
		log.Printf("Failed to clean expired cache: %v", err)
	}

	// 更新状态
	s.mu.Lock()
	s.lastFetch = time.Now()
	if firstError != nil {
		s.lastError = firstError.Error()
	} else {
		s.lastError = ""
	}
	s.mu.Unlock()

	// 更新数据库状态
	var errorMsg string
	if firstError != nil {
		errorMsg = firstError.Error()
	}
	if err := s.db.UpdateSystemStatus(s.GetLastFetch(), time.Now().Add(5*time.Minute), errorMsg); err != nil {
		log.Printf("Failed to update system status: %v", err)
	}

	log.Printf("Job completed: %d downloaded, %d skipped, %d failed", successCount, skipCount, failCount)
}

func (s *Scheduler) TriggerNow() error {
	go s.runJob()
	return nil
}
