package scheduler

import (
	"fmt"
	"log"
	"sync"
	"time"

	"mteam-rss/internal/database"
	"mteam-rss/internal/rss"
)

// MultiScheduler 多RSS源调度器
type MultiScheduler struct {
	db        *database.Database
	sources   map[int64]*SourceSchedule
	sourcesMu sync.RWMutex
	queue     chan *FetchTask
	workers   int
	stopChan  chan struct{}
	wg        sync.WaitGroup
	running   bool
}

// SourceSchedule 单个RSS源的调度状态
type SourceSchedule struct {
	SourceID     int64
	SourceName   string
	SiteType     string
	RSSURL       string
	PollInterval time.Duration
	MaxItems     int
	LastFetch    time.Time
	NextFetch    time.Time
	LastError    string
	ErrorCount   int
	IsRunning    bool
}

// FetchTask 抓取任务
type FetchTask struct {
	SourceID   int64
	SourceName string
	SiteType   string
	RSSURL     string
	MaxItems   int
}

// NewMultiScheduler 创建多源调度器
func NewMultiScheduler(db *database.Database, _ interface{}, workers int) *MultiScheduler {
	return &MultiScheduler{
		db:       db,
		sources:  make(map[int64]*SourceSchedule),
		queue:    make(chan *FetchTask, 100),
		workers:  workers,
		stopChan: make(chan struct{}),
		running:  false,
	}
}

// Start 启动调度器
func (ms *MultiScheduler) Start() error {
	ms.sourcesMu.Lock()
	defer ms.sourcesMu.Unlock()

	if ms.running {
		return fmt.Errorf("scheduler is already running")
	}

	// 从数据库加载启用的RSS源
	sources, err := ms.db.GetEnabledRSSSources()
	if err != nil {
		return fmt.Errorf("failed to load RSS sources: %w", err)
	}

	// 初始化每个源的调度状态
	for _, source := range sources {
		interval := time.Duration(source.PollInterval) * time.Second
		nextFetch := source.LastFetch.Add(interval)
		if source.LastFetch.IsZero() || nextFetch.Before(time.Now()) {
			nextFetch = time.Now()
		}

		ms.sources[source.ID] = &SourceSchedule{
			SourceID:     source.ID,
			SourceName:   source.Name,
			SiteType:     source.SiteType,
			RSSURL:       source.RSSURL,
			PollInterval: interval,
			MaxItems:     source.MaxItems,
			LastFetch:    source.LastFetch,
			NextFetch:    nextFetch,
			LastError:    source.LastError,
			ErrorCount:   source.ErrorCount,
			IsRunning:    false,
		}
	}

	// 启动worker
	for i := 0; i < ms.workers; i++ {
		ms.wg.Add(1)
		go ms.worker()
	}

	// 启动调度检查器
	ms.wg.Add(1)
	go ms.scheduler()

	ms.running = true
	log.Printf("MultiScheduler started with %d sources and %d workers", len(sources), ms.workers)
	return nil
}

// Stop 停止调度器
func (ms *MultiScheduler) Stop() {
	ms.sourcesMu.Lock()
	if !ms.running {
		ms.sourcesMu.Unlock()
		return
	}
	ms.running = false
	ms.sourcesMu.Unlock()

	close(ms.stopChan)
	ms.wg.Wait()
	log.Println("MultiScheduler stopped")
}

// scheduler 调度器主循环
func (ms *MultiScheduler) scheduler() {
	defer ms.wg.Done()

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ms.stopChan:
			return
		case <-ticker.C:
			ms.checkAndSchedule()
		}
	}
}

// checkAndSchedule 检查并调度任务
func (ms *MultiScheduler) checkAndSchedule() {
	ms.sourcesMu.RLock()
	defer ms.sourcesMu.RUnlock()

	now := time.Now()
	for _, schedule := range ms.sources {
		if !schedule.IsRunning && schedule.NextFetch.Before(now) && schedule.ErrorCount < 3 {
			// 提交抓取任务
			task := &FetchTask{
				SourceID:   schedule.SourceID,
				SourceName: schedule.SourceName,
				SiteType:   schedule.SiteType,
				RSSURL:     schedule.RSSURL,
				MaxItems:   schedule.MaxItems,
			}

			select {
			case ms.queue <- task:
				schedule.IsRunning = true
			default:
				log.Printf("Queue full, skipping source %s", schedule.SourceName)
			}
		}
	}
}

// worker Worker处理抓取任务
func (ms *MultiScheduler) worker() {
	defer ms.wg.Done()

	for {
		select {
		case <-ms.stopChan:
			return
		case task, ok := <-ms.queue:
			if !ok {
				return
			}
			ms.processTask(task)
		}
	}
}

// processTask 处理单个抓取任务
func (ms *MultiScheduler) processTask(task *FetchTask) {
	sourceID := task.SourceID
	log.Printf("[Source %d] Starting fetch: %s (type: %s)", sourceID, task.SourceName, task.SiteType)

	// 更新状态为正在运行
	ms.updateSourceStatus(sourceID, time.Now(), "", true)

	// 根据站点类型获取对应的handler
	handler, _, err := rss.GetHandlerByURL(task.RSSURL)
	if err != nil {
		log.Printf("[Source %d] No handler found for RSS URL: %s", sourceID, task.RSSURL)
		ms.updateSourceStatus(sourceID, time.Now(), "未找到对应的站点处理器", false)
		ms.db.AddLog("error", &sourceID, fmt.Sprintf("未找到RSS URL对应的处理器: %s", task.RSSURL), "")
		return
	}

	if handler == nil {
		log.Printf("[Source %d] Handler is nil for site type: %s", sourceID, task.SiteType)
		ms.updateSourceStatus(sourceID, time.Now(), "站点处理器为空", false)
		return
	}

	// 创建RSS客户端并抓取
	rssClient := rss.New(task.RSSURL)
	items, err := rssClient.FetchWithLimit(task.MaxItems)

	if err != nil {
		log.Printf("[Source %d] Fetch failed: %v", sourceID, err)
		ms.updateSourceStatus(sourceID, time.Now(), err.Error(), false)
		ms.db.AddLog("error", &sourceID, fmt.Sprintf("RSS抓取失败: %v", err), "")
		return
	}

	log.Printf("[Source %d] Fetched %d items using handler: %s", sourceID, len(items), task.SiteType)

	// 准备下载任务
	var tasks []database.DownloadTask
	for _, item := range items {
		tasks = append(tasks, database.DownloadTask{
			SourceID: sourceID,
			GUID:     item.GUID,
			Title:    item.Title,
			URL:      item.URL,
			Size:     item.Size,
			Category: item.Category,
			PubDate:  item.PubDate,
			Status:   "pending",
		})
	}

	// 创建下载任务
	if len(tasks) > 0 {
		createdCount := 0
		for _, task := range tasks {
			exists, _ := ms.db.TaskExists(task.GUID)
			if !exists {
				_, err := ms.db.CreateDownloadTask(task)
				if err == nil {
					createdCount++
				}
			}
		}

		log.Printf("[Source %d] Created %d new tasks", sourceID, createdCount)
		ms.db.AddLog("info", &sourceID, fmt.Sprintf("抓取完成，新增%d个任务", createdCount), "")
	}

	// 更新状态
	nextFetch := time.Now().Add(ms.getPollInterval(sourceID))
	ms.updateSourceStatus(sourceID, time.Now(), "", false)
	ms.updateSourceNextFetch(sourceID, nextFetch)

	// 更新系统统计
	ms.db.UpdateSystemStats()
}

// updateSourceStatus 更新RSS源状态
func (ms *MultiScheduler) updateSourceStatus(sourceID int64, lastFetch time.Time, lastError string, isRunning bool) {
	ms.sourcesMu.Lock()
	defer ms.sourcesMu.Unlock()

	if schedule, exists := ms.sources[sourceID]; exists {
		schedule.LastFetch = lastFetch
		schedule.IsRunning = isRunning
		if lastError != "" {
			schedule.LastError = lastError
			schedule.ErrorCount++
		} else {
			schedule.LastError = ""
			schedule.ErrorCount = 0
		}
	}

	// 更新数据库
	var dbLastError string
	if isRunning {
		dbLastError = lastError
	}
	ms.db.UpdateRSSSourceFetchStatus(sourceID, lastFetch, dbLastError)
}

// updateSourceNextFetch 更新下次抓取时间
func (ms *MultiScheduler) updateSourceNextFetch(sourceID int64, nextFetch time.Time) {
	ms.sourcesMu.Lock()
	defer ms.sourcesMu.Unlock()

	if schedule, exists := ms.sources[sourceID]; exists {
		schedule.NextFetch = nextFetch
	}
}

// getPollInterval 获取RSS源的轮询间隔
func (ms *MultiScheduler) getPollInterval(sourceID int64) time.Duration {
	ms.sourcesMu.RLock()
	defer ms.sourcesMu.RUnlock()

	if schedule, exists := ms.sources[sourceID]; exists {
		return schedule.PollInterval
	}
	return 5 * time.Minute
}

// GetSourcesStatus 获取所有RSS源的状态
func (ms *MultiScheduler) GetSourcesStatus() []SourceStatus {
	ms.sourcesMu.RLock()
	defer ms.sourcesMu.RUnlock()

	var status []SourceStatus
	for _, schedule := range ms.sources {
		status = append(status, SourceStatus{
			SourceID:   schedule.SourceID,
			SourceName: schedule.SourceName,
			LastFetch:  schedule.LastFetch,
			NextFetch:  schedule.NextFetch,
			LastError:  schedule.LastError,
			ErrorCount: schedule.ErrorCount,
			IsRunning:  schedule.IsRunning,
		})
	}
	return status
}

// TriggerFetch 手动触发RSS源抓取
func (ms *MultiScheduler) TriggerFetch(sourceID int64) error {
	ms.sourcesMu.RLock()
	schedule, exists := ms.sources[sourceID]
	ms.sourcesMu.RUnlock()

	if !exists {
		return fmt.Errorf("source %d not found", sourceID)
	}

	task := &FetchTask{
		SourceID:   schedule.SourceID,
		SourceName: schedule.SourceName,
		SiteType:   schedule.SiteType,
		RSSURL:     schedule.RSSURL,
		MaxItems:   schedule.MaxItems,
	}

	select {
	case ms.queue <- task:
		return nil
	default:
		return fmt.Errorf("queue is full")
	}
}

// ReloadSources 重新加载RSS源配置
func (ms *MultiScheduler) ReloadSources() error {
	sources, err := ms.db.GetEnabledRSSSources()
	if err != nil {
		return err
	}

	ms.sourcesMu.Lock()
	defer ms.sourcesMu.Unlock()

	// 清空现有调度
	ms.sources = make(map[int64]*SourceSchedule)

	// 重新初始化
	for _, source := range sources {
		interval := time.Duration(source.PollInterval) * time.Second
		nextFetch := source.LastFetch.Add(interval)
		if source.LastFetch.IsZero() || nextFetch.Before(time.Now()) {
			nextFetch = time.Now()
		}

		ms.sources[source.ID] = &SourceSchedule{
			SourceID:     source.ID,
			SourceName:   source.Name,
			SiteType:     source.SiteType,
			RSSURL:       source.RSSURL,
			PollInterval: interval,
			MaxItems:     source.MaxItems,
			LastFetch:    source.LastFetch,
			NextFetch:    nextFetch,
			LastError:    source.LastError,
			ErrorCount:   source.ErrorCount,
			IsRunning:    false,
		}
	}

	log.Printf("Reloaded %d RSS sources", len(sources))
	return nil
}

// SourceStatus RSS源状态信息
type SourceStatus struct {
	SourceID   int64     `json:"source_id"`
	SourceName string    `json:"source_name"`
	LastFetch  time.Time `json:"last_fetch"`
	NextFetch  time.Time `json:"next_fetch"`
	LastError  string    `json:"last_error"`
	ErrorCount int       `json:"error_count"`
	IsRunning  bool      `json:"is_running"`
}
