package downloader

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"mteam-rss/internal/database"
)

type Downloader struct {
	savePath      string
	maxConcurrent int
	semaphore     chan struct{}
	client        *http.Client
}

func New(savePath string, maxConcurrent int) *Downloader {
	return &Downloader{
		savePath:      savePath,
		maxConcurrent: maxConcurrent,
		semaphore:     make(chan struct{}, maxConcurrent),
		client: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

type DownloadTask struct {
	GUID    string
	Title   string
	URL     string
	PubDate time.Time
}

type DownloadResult struct {
	Task    DownloadTask
	Success bool
	Error   error
	Bytes   int64
}

func (d *Downloader) Download(task DownloadTask) (*DownloadResult, error) {
	// 获取信号量（限流）
	d.semaphore <- struct{}{}
	defer func() { <-d.semaphore }()

	// 生成安全的文件名
	safeName := sanitizeFilename(task.Title)
	if !strings.HasSuffix(safeName, ".torrent") {
		safeName += ".torrent"
	}

	filePath := filepath.Join(d.savePath, safeName)

	// 检查文件是否已存在
	if _, err := os.Stat(filePath); err == nil {
		return &DownloadResult{
			Task:    task,
			Success: true,
			Bytes:   0,
		}, nil
	}

	// 下载文件
	resp, err := d.client.Get(task.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to download: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// 创建文件
	file, err := os.Create(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	// 写入文件
	bytes, err := io.Copy(file, resp.Body)
	if err != nil {
		os.Remove(filePath) // 清理失败的下载
		return nil, fmt.Errorf("failed to write file: %w", err)
	}

	return &DownloadResult{
		Task:    task,
		Success: true,
		Error:   nil,
		Bytes:   bytes,
	}, nil
}

func (d *Downloader) DownloadBatch(tasks []DownloadTask, db *database.Database) []DownloadResult {
	var wg sync.WaitGroup
	results := make([]DownloadResult, len(tasks))
	resultChan := make(chan DownloadResult, len(tasks))

	for i, task := range tasks {
		wg.Add(1)
		go func(idx int, t DownloadTask) {
			defer wg.Done()

			result := &DownloadResult{Task: t}

			// 检查是否已下载
			downloaded, err := db.IsDownloaded(t.GUID)
			if err != nil {
				result.Error = fmt.Errorf("failed to check download status: %w", err)
				resultChan <- *result
				return
			}

			if downloaded {
				result.Success = true
				resultChan <- *result
				return
			}

			// 执行下载
			downloadResult, err := d.Download(t)
			if err != nil {
				result.Error = err
				result.Success = false
				resultChan <- *result
				return
			}

			// 记录到数据库
			safeName := sanitizeFilename(t.Title)
			if !strings.HasSuffix(safeName, ".torrent") {
				safeName += ".torrent"
			}
			filePath := filepath.Join(d.savePath, safeName)

			if err := db.RecordDownload(t.GUID, t.Title, t.URL, filePath, t.PubDate); err != nil {
				result.Error = fmt.Errorf("downloaded but failed to record: %w", err)
				result.Success = false
				resultChan <- *result
				return
			}

			result = downloadResult
			resultChan <- *result
		}(i, task)
	}

	// 等待所有下载完成
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// 收集结果
	i := 0
	for result := range resultChan {
		results[i] = result
		i++
	}

	return results
}

func sanitizeFilename(name string) string {
	// 移除或替换不安全的字符
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

	// 限制文件名长度
	if len(safe) > 200 {
		safe = safe[:200]
	}

	return strings.TrimSpace(safe)
}
