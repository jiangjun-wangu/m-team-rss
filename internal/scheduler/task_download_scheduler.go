package scheduler

import (
	"fmt"
	"html"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"mteam-rss/internal/database"
	"mteam-rss/internal/rss"

	"github.com/mmcdole/gofeed"
)

// TaskDownloadScheduler 任务下载调度器 - 处理pending任务的下载
type TaskDownloadScheduler struct {
	db         *database.Database
	httpClient *http.Client
	maxWorkers int
	semaphore  chan struct{}
	stopChan   chan struct{}
	wg         sync.WaitGroup
	running    bool
}

// NewTaskDownloadScheduler 创建任务下载调度器
func NewTaskDownloadScheduler(db *database.Database, maxWorkers int) *TaskDownloadScheduler {
	if maxWorkers < 1 {
		maxWorkers = 3
	}
	return &TaskDownloadScheduler{
		db: db,
		httpClient: &http.Client{
			Timeout: 300 * time.Second,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= 10 {
					return fmt.Errorf("stopped after 10 redirects")
				}
				if len(via) > 0 {
					req.Header.Set("User-Agent", via[0].Header.Get("User-Agent"))
				}
				return nil
			},
		},
		maxWorkers: maxWorkers,
		semaphore:  make(chan struct{}, maxWorkers),
		stopChan:   make(chan struct{}),
		running:    false,
	}
}

// Start 启动调度器
func (tds *TaskDownloadScheduler) Start() error {
	tds.running = true

	// 启动多个worker
	for i := 0; i < tds.maxWorkers; i++ {
		tds.wg.Add(1)
		go tds.worker()
	}

	log.Printf("TaskDownloadScheduler started with %d workers", tds.maxWorkers)
	return nil
}

// Stop 停止调度器
func (tds *TaskDownloadScheduler) Stop() {
	if !tds.running {
		return
	}
	tds.running = false
	close(tds.stopChan)
	tds.wg.Wait()
	log.Println("TaskDownloadScheduler stopped")
}

// worker Worker处理下载任务
func (tds *TaskDownloadScheduler) worker() {
	defer tds.wg.Done()

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-tds.stopChan:
			return
		case <-ticker.C:
			tds.processNextTask()
		}
	}
}

// processNextTask 处理下一个待下载任务
func (tds *TaskDownloadScheduler) processNextTask() {
	// 获取信号量
	tds.semaphore <- struct{}{}
	defer func() { <-tds.semaphore }()

	// 查询pending任务
	tasks, _, err := tds.db.GetDownloadTasks(0, "pending", 1, 1)
	if err != nil {
		log.Printf("Failed to query pending tasks: %v", err)
		return
	}

	if len(tasks) == 0 {
		return
	}

	task := tasks[0]
	log.Printf("[Task %d] Starting download: %s", task.ID, task.Title)

	// 更新状态为downloading
	tds.db.UpdateTaskStatus(task.ID, "downloading", "", "")

	// 执行下载
	success, errorMsg, filePath := tds.downloadTorrent(task)

	// 更新任务状态
	if !success {
		log.Printf("[Task %d] Download failed: %v", task.ID, errorMsg)

		// 增加重试次数
		task.RetryCount++

		// 检查是否超过最大重试次数
		if task.RetryCount >= 3 {
			tds.db.UpdateTaskStatus(task.ID, "failed", "", errorMsg)
			tds.db.AddLog("error", &task.SourceID, fmt.Sprintf("任务下载失败(%d次): %s", task.RetryCount, task.Title), errorMsg)
		} else {
			tds.db.UpdateTaskStatus(task.ID, "pending", "", errorMsg)
			tds.db.AddLog("warning", &task.SourceID, fmt.Sprintf("任务下载失败,将重试(%d/3): %s", task.RetryCount, task.Title), errorMsg)
		}
	} else {
		log.Printf("[Task %d] Download succeeded: %s", task.ID, task.Title)
		tds.db.UpdateTaskStatus(task.ID, "completed", filePath, "")
		tds.db.AddLog("info", &task.SourceID, fmt.Sprintf("任务下载成功: %s", task.Title), "")

		// 记录到旧版数据库(保持兼容)
		tds.db.RecordDownload(task.GUID, task.Title, task.URL, filePath, task.PubDate)
	}

	// 更新系统统计
	tds.db.UpdateSystemStats()
}

// downloadTorrent 下载单个种子文件
func (tds *TaskDownloadScheduler) downloadTorrent(task database.DownloadTask) (bool, string, string) {
	// 获取RSS源信息
	source, err := tds.db.GetRSSSource(task.SourceID)
	if err != nil {
		return false, fmt.Sprintf("failed to get RSS source: %v", err), ""
	}

	// 生成安全的文件名
	safeName := sanitizeFilename(task.Title)
	if !strings.HasSuffix(safeName, ".torrent") {
		safeName += ".torrent"
	}

	filePath := filepath.Join("./torrents", safeName)

	// 检查文件是否已存在
	if _, err := os.Stat(filePath); err == nil {
		return true, "", filePath
	}

	// 获取站点handler
	handler, siteType, err := rss.GetHandlerByURL(source.RSSURL)
	if err != nil || handler == nil {
		log.Printf("[Task %d] Using generic download for %s", task.ID, source.SiteType)
		return tds.downloadGeneric(task, filePath)
	}

	log.Printf("[Task %d] Using handler %s for %s", task.ID, source.SiteType, task.Title)

	// 对于PT站点，重新获取RSS来获取最新的下载URL
	var downloadURL string
	if siteType == "ptzone" {
		// 重新获取RSS获取最新下载URL
		downloadURL, err = tds.getLatestDownloadURL(source.RSSURL, task.GUID)
		if err != nil {
			log.Printf("[Task %d] Failed to get fresh download URL from RSS: %v", task.ID, err)
			return false, fmt.Sprintf("failed to get fresh download URL: %v", err), ""
		}
		log.Printf("[Task %d] Got fresh download URL: %s", task.ID, downloadURL)
	} else {
		// 其他站点使用数据库中存储的URL
		downloadURL = task.URL
	}

	log.Printf("[Task %d] Download URL: %s", task.ID, downloadURL)

	// 使用handler下载
	return tds.downloadWithHandler(task, filePath, handler, downloadURL)
}

// downloadWithHandler 使用站点handler下载
func (tds *TaskDownloadScheduler) downloadWithHandler(task database.DownloadTask, filePath string, handler rss.SiteHandler, downloadURL string) (bool, string, string) {
	// 创建HTTP请求
	req, err := http.NewRequest("GET", downloadURL, nil)
	if err != nil {
		return false, fmt.Sprintf("failed to create request: %v", err), ""
	}

	// 调用handler的BeforeDownload设置请求头
	downloadTask := rss.DownloadTask{
		GUID:    task.GUID,
		Title:   task.Title,
		URL:     downloadURL,
		PubDate: task.PubDate,
	}
	if err := handler.BeforeDownload(req, downloadTask); err != nil {
		return false, fmt.Sprintf("BeforeDownload failed: %v", err), ""
	}

	// 执行请求
	resp, err := tds.httpClient.Do(req)
	if err != nil {
		return false, fmt.Sprintf("failed to download: %v", err), ""
	}
	defer resp.Body.Close()

	// 调用handler的CheckResponse检查响应
	if err := handler.CheckResponse(resp); err != nil {
		return false, err.Error(), ""
	}

	// 确保目录存在
	os.MkdirAll(filepath.Dir(filePath), 0755)

	// 创建文件
	file, err := os.Create(filePath)
	if err != nil {
		return false, fmt.Sprintf("failed to create file: %v", err), ""
	}
	defer file.Close()

	// 写入文件
	bytes, err := io.Copy(file, resp.Body)
	if err != nil {
		os.Remove(filePath)
		return false, fmt.Sprintf("failed to write file: %v", err), ""
	}

	log.Printf("[Task] Downloaded %s (%d bytes)", task.Title, bytes)

	// 调用handler的AfterDownload
	if err := handler.AfterDownload(filePath, downloadTask); err != nil {
		log.Printf("[Task] AfterDownload warning: %v", err)
	}

	return true, "", filePath
}

// downloadGeneric 使用通用方式下载
func (tds *TaskDownloadScheduler) downloadGeneric(task database.DownloadTask, filePath string) (bool, string, string) {
	// 创建HTTP请求
	req, err := http.NewRequest("GET", task.URL, nil)
	if err != nil {
		return false, fmt.Sprintf("failed to create request: %v", err), ""
	}

	// 设置通用请求头
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	req.Header.Set("Accept", "application/x-bittorrent, application/octet-stream")

	// 执行请求
	resp, err := tds.httpClient.Do(req)
	if err != nil {
		return false, fmt.Sprintf("failed to download: %v", err), ""
	}
	defer resp.Body.Close()

	// 检查状态码
	if resp.StatusCode != http.StatusOK {
		_, _ = io.ReadAll(resp.Body)
		return false, fmt.Sprintf("unexpected status code: %d, URL: %s", resp.StatusCode, task.URL), ""
	}

	// 检查Content-Type
	contentType := resp.Header.Get("Content-Type")
	if contentType != "" && !strings.Contains(contentType, "application/x-bittorrent") &&
		!strings.Contains(contentType, "application/octet-stream") {
		_, _ = io.ReadAll(resp.Body)
		return false, fmt.Sprintf("unexpected content type: %s", contentType), ""
	}

	// 确保目录存在
	os.MkdirAll(filepath.Dir(filePath), 0755)

	// 创建文件
	file, err := os.Create(filePath)
	if err != nil {
		return false, fmt.Sprintf("failed to create file: %v", err), ""
	}
	defer file.Close()

	// 写入文件
	bytes, err := io.Copy(file, resp.Body)
	if err != nil {
		os.Remove(filePath)
		return false, fmt.Sprintf("failed to write file: %v", err), ""
	}

	log.Printf("[Task] Downloaded %s (%d bytes)", task.Title, bytes)
	return true, "", filePath
}

// getLatestDownloadURL 从RSS获取最新的下载URL
func (tds *TaskDownloadScheduler) getLatestDownloadURL(rssURL string, guid string) (string, error) {
	// 创建RSS客户端
	rssClient := rss.New(rssURL)

	// 获取RSS内容（获取更多条目以增加匹配几率）
	items, err := rssClient.FetchWithLimit(100)
	if err != nil {
		return "", fmt.Errorf("failed to fetch RSS: %v", err)
	}

	// 查找匹配的item
	for _, item := range items {
		if item.GUID == guid {
			// 使用RSS handler解析下载URL
			handler, _, err := rss.GetHandlerByURL(rssURL)
			if err != nil || handler == nil {
				// 没有handler，直接返回URL
				return item.URL, nil
			}

			// 使用handler解析，需要转换为gofeed.Item
			fp := &gofeed.Item{
				GUID:       item.GUID,
				Title:      item.Title,
				Link:       item.URL,
				Enclosures: []*gofeed.Enclosure{{URL: item.URL}},
			}
			return handler.ParseDownloadURL(fp, rssURL), nil
		}
	}

	return "", fmt.Errorf("item with GUID %s not found in RSS", guid)
}

// sanitizeFilename 清理文件名
func sanitizeFilename(name string) string {
	// HTML实体解码
	name = html.UnescapeString(name)

	// 移除HTML标签
	re := `<[^>]*>`
	regex := regexp.MustCompile(re)
	name = regex.ReplaceAllString(name, "")

	replacer := strings.NewReplacer(
		"<", "_",
		">", "_",
		":", "_",
		"\"", "_",
		"/", "_",
		"\\", "_",
		"|", "_",
		"?", "_",
		"*", "_",
		"\n", "",
		"\r", "",
	)
	safe := replacer.Replace(name)

	if len(safe) > 200 {
		safe = safe[:200]
	}

	return strings.TrimSpace(safe)
}

// RetryFailedTasks 重试失败的任务
func (tds *TaskDownloadScheduler) RetryFailedTasks(limit int) error {
	tasks, _, err := tds.db.GetDownloadTasks(0, "failed", 1, limit)
	if err != nil {
		return err
	}

	var taskIDs []int64
	for _, task := range tasks {
		taskIDs = append(taskIDs, task.ID)
	}

	if len(taskIDs) > 0 {
		return tds.db.UpdateTasksStatus(taskIDs, "pending")
	}
	return nil
}
