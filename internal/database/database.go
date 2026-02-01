package database

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

type Database struct {
	db *sql.DB
}

func New(path string) (*Database, error) {
	// 使用modernc.org/sqlite驱动
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// 设置连接池
	db.SetMaxOpenConns(1) // SQLite是文件数据库，单连接
	db.SetMaxIdleConns(1)

	d := &Database{db: db}
	if err := d.initTables(); err != nil {
		db.Close()
		return nil, err
	}

	return d, nil
}

func (d *Database) initTables() error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS downloads (
			guid TEXT PRIMARY KEY,
			title TEXT NOT NULL,
			url TEXT NOT NULL,
			file_path TEXT,
			downloaded_at DATETIME,
			pub_date DATETIME,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS system_status (
			id INTEGER PRIMARY KEY CHECK (id = 1),
			last_fetch DATETIME,
			next_fetch DATETIME,
			last_error TEXT,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS rss_cache (
			guid TEXT PRIMARY KEY,
			title TEXT NOT NULL,
			url TEXT NOT NULL,
			pub_date DATETIME,
			category TEXT,
			size INTEGER,
			json_data TEXT,
			cached_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS rss_sources (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			site_type TEXT NOT NULL,
			rss_url TEXT NOT NULL UNIQUE,
			enabled INTEGER DEFAULT 1,
			poll_interval INTEGER DEFAULT 300,
			max_items INTEGER DEFAULT 50,
			filters TEXT,
			last_fetch DATETIME,
			last_error TEXT,
			error_count INTEGER DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS download_tasks (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			source_id INTEGER NOT NULL,
			guid TEXT NOT NULL,
			title TEXT NOT NULL,
			url TEXT NOT NULL,
			size INTEGER DEFAULT 0,
			category TEXT,
			pub_date DATETIME,
			status TEXT DEFAULT 'pending',
			file_path TEXT,
			error_msg TEXT,
			retry_count INTEGER DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(source_id, guid),
			FOREIGN KEY (source_id) REFERENCES rss_sources(id) ON DELETE CASCADE
		)`,
		`CREATE TABLE IF NOT EXISTS system_logs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			level TEXT NOT NULL,
			source_id INTEGER,
			message TEXT NOT NULL,
			details TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (source_id) REFERENCES rss_sources(id) ON DELETE SET NULL
		)`,
		`CREATE TABLE IF NOT EXISTS system_stats (
			id INTEGER PRIMARY KEY CHECK (id = 1),
			total_sources INTEGER DEFAULT 0,
			enabled_sources INTEGER DEFAULT 0,
			total_tasks INTEGER DEFAULT 0,
			pending_tasks INTEGER DEFAULT 0,
			downloading_tasks INTEGER DEFAULT 0,
			completed_tasks INTEGER DEFAULT 0,
			failed_tasks INTEGER DEFAULT 0,
			total_downloads INTEGER DEFAULT 0,
			last_fetch_time DATETIME,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_rss_cache_pub_date ON rss_cache(pub_date DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_rss_cache_cached_at ON rss_cache(cached_at)`,
		`CREATE INDEX IF NOT EXISTS idx_download_tasks_source_id ON download_tasks(source_id)`,
		`CREATE INDEX IF NOT EXISTS idx_download_tasks_status ON download_tasks(status)`,
		`CREATE INDEX IF NOT EXISTS idx_download_tasks_guid ON download_tasks(guid)`,
		`CREATE INDEX IF NOT EXISTS idx_system_logs_created_at ON system_logs(created_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_system_logs_level ON system_logs(level)`,
	}

	for _, q := range queries {
		if _, err := d.db.Exec(q); err != nil {
			return fmt.Errorf("failed to create table: %w", err)
		}
	}

	// 初始化系统状态记录
	_, err := d.db.Exec(`
		INSERT OR IGNORE INTO system_status (id, last_fetch, next_fetch, last_error)
		VALUES (1, NULL, NULL, NULL)
	`)
	if err != nil {
		return err
	}

	// 初始化系统统计记录
	_, err = d.db.Exec(`
		INSERT OR IGNORE INTO system_stats (id, total_sources, enabled_sources, total_tasks, pending_tasks, downloading_tasks, completed_tasks, failed_tasks, total_downloads, last_fetch_time)
		VALUES (1, 0, 0, 0, 0, 0, 0, 0, 0, NULL)
	`)

	return err
}

// 检查GUID是否已下载
func (d *Database) IsDownloaded(guid string) (bool, error) {
	var exists bool
	err := d.db.QueryRow("SELECT EXISTS(SELECT 1 FROM downloads WHERE guid = ?)", guid).Scan(&exists)
	return exists, err
}

// 记录下载
func (d *Database) RecordDownload(guid, title, url, filePath string, pubDate time.Time) error {
	_, err := d.db.Exec(`
		INSERT INTO downloads (guid, title, url, file_path, downloaded_at, pub_date)
		VALUES (?, ?, ?, ?, ?, ?)
	`, guid, title, url, filePath, time.Now(), pubDate)
	return err
}

// 获取下载统计
func (d *Database) GetStats() (total, downloaded int, err error) {
	err = d.db.QueryRow("SELECT COUNT(*) FROM rss_cache").Scan(&total)
	if err != nil {
		return 0, 0, err
	}
	err = d.db.QueryRow("SELECT COUNT(*) FROM downloads").Scan(&downloaded)
	return total, downloaded, err
}

// 更新系统状态
func (d *Database) UpdateSystemStatus(lastFetch, nextFetch time.Time, lastError string) error {
	_, err := d.db.Exec(`
		UPDATE system_status
		SET last_fetch = ?, next_fetch = ?, last_error = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = 1
	`, lastFetch, nextFetch, lastError)
	return err
}

// 获取系统状态
func (d *Database) GetSystemStatus() (lastFetch, nextFetch time.Time, lastError string, err error) {
	err = d.db.QueryRow(`
		SELECT last_fetch, next_fetch, last_error
		FROM system_status WHERE id = 1
	`).Scan(&lastFetch, &nextFetch, &lastError)
	return lastFetch, nextFetch, lastError, err
}

// 缓存RSS项目
func (d *Database) CacheRSSItems(items []RSSItemCache) error {
	tx, err := d.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
		INSERT OR REPLACE INTO rss_cache (guid, title, url, pub_date, category, size, json_data, cached_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, item := range items {
		jsonData, _ := json.Marshal(item)
		_, err := stmt.Exec(item.GUID, item.Title, item.URL, item.PubDate, item.Category, item.Size, jsonData, time.Now())
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

// 获取缓存的RSS项目
func (d *Database) GetCachedRSSItems(limit int) ([]RSSItemDisplay, error) {
	query := `
		SELECT r.guid, r.title, r.url, r.pub_date, d.file_path, d.downloaded_at
		FROM rss_cache r
		LEFT JOIN downloads d ON r.guid = d.guid
		ORDER BY r.pub_date DESC
		LIMIT ?
	`
	if limit <= 0 {
		query = `
			SELECT r.guid, r.title, r.url, r.pub_date, d.file_path, d.downloaded_at
			FROM rss_cache r
			LEFT JOIN downloads d ON r.guid = d.guid
			ORDER BY r.pub_date DESC
		`
	}

	rows, err := d.db.Query(query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []RSSItemDisplay
	for rows.Next() {
		var item RSSItemDisplay
		var pubDate sql.NullTime
		var filePath, downloadedAt sql.NullString
		err := rows.Scan(&item.GUID, &item.Title, &item.URL, &pubDate, &filePath, &downloadedAt)
		if err != nil {
			return nil, err
		}
		if pubDate.Valid {
			item.PubDate = pubDate.Time
		}
		item.IsDownloaded = filePath.Valid
		if downloadedAt.Valid {
			item.DownloadedAt, _ = time.Parse(time.RFC3339Nano, downloadedAt.String)
		}
		if filePath.Valid {
			item.FilePath = filePath.String
		}
		items = append(items, item)
	}

	return items, nil
}

// 清理过期缓存
func (d *Database) CleanExpiredCache(before time.Time) error {
	_, err := d.db.Exec("DELETE FROM rss_cache WHERE cached_at < ?", before)
	return err
}

func (d *Database) Close() error {
	return d.db.Close()
}

// ===== RSS源相关方法 =====

// GetRSSSources 获取所有RSS源
func (d *Database) GetRSSSources() ([]RSSSource, error) {
	query := `
		SELECT id, name, site_type, rss_url, enabled, poll_interval, max_items, filters,
			   last_fetch, last_error, error_count
		FROM rss_sources
		ORDER BY id ASC
	`

	rows, err := d.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sources []RSSSource
	for rows.Next() {
		var s RSSSource
		var lastFetch sql.NullTime
		var lastError sql.NullString
		err := rows.Scan(&s.ID, &s.Name, &s.SiteType, &s.RSSURL, &s.Enabled,
			&s.PollInterval, &s.MaxItems, &s.Filters,
			&lastFetch, &lastError, &s.ErrorCount)
		if err != nil {
			return nil, err
		}
		if lastFetch.Valid {
			s.LastFetch = lastFetch.Time
		}
		if lastError.Valid {
			s.LastError = lastError.String
		}
		sources = append(sources, s)
	}

	return sources, nil
}

// GetEnabledRSSSources 获取启用的RSS源
func (d *Database) GetEnabledRSSSources() ([]RSSSource, error) {
	query := `
		SELECT id, name, site_type, rss_url, enabled, poll_interval, max_items, filters,
			   last_fetch, last_error, error_count
		FROM rss_sources
		WHERE enabled = 1
		ORDER BY id ASC
	`

	rows, err := d.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sources []RSSSource
	for rows.Next() {
		var s RSSSource
		var lastFetch sql.NullTime
		var lastError sql.NullString
		err := rows.Scan(&s.ID, &s.Name, &s.SiteType, &s.RSSURL, &s.Enabled,
			&s.PollInterval, &s.MaxItems, &s.Filters,
			&lastFetch, &lastError, &s.ErrorCount)
		if err != nil {
			return nil, err
		}
		if lastFetch.Valid {
			s.LastFetch = lastFetch.Time
		}
		if lastError.Valid {
			s.LastError = lastError.String
		}
		sources = append(sources, s)
	}

	return sources, nil
}

// GetRSSSource 根据ID获取RSS源
func (d *Database) GetRSSSource(id int64) (*RSSSource, error) {
	query := `
		SELECT id, name, site_type, rss_url, enabled, poll_interval, max_items, filters,
			   last_fetch, last_error, error_count
		FROM rss_sources
		WHERE id = ?
	`

	var s RSSSource
	var lastFetch sql.NullTime
	var lastError sql.NullString
	err := d.db.QueryRow(query, id).Scan(&s.ID, &s.Name, &s.SiteType, &s.RSSURL, &s.Enabled,
		&s.PollInterval, &s.MaxItems, &s.Filters,
		&lastFetch, &lastError, &s.ErrorCount)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	if lastFetch.Valid {
		s.LastFetch = lastFetch.Time
	}
	if lastError.Valid {
		s.LastError = lastError.String
	}

	return &s, nil
}

// CreateRSSSource 创建RSS源
func (d *Database) CreateRSSSource(source RSSSource) (int64, error) {
	result, err := d.db.Exec(`
		INSERT INTO rss_sources (name, site_type, rss_url, enabled, poll_interval, max_items, filters)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, source.Name, source.SiteType, source.RSSURL, source.Enabled,
		source.PollInterval, source.MaxItems, source.Filters)

	if err != nil {
		return 0, err
	}

	return result.LastInsertId()
}

// UpdateRSSSource 更新RSS源
func (d *Database) UpdateRSSSource(source RSSSource) error {
	_, err := d.db.Exec(`
		UPDATE rss_sources
		SET name = ?, site_type = ?, rss_url = ?, enabled = ?,
		    poll_interval = ?, max_items = ?, filters = ?,
		    updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, source.Name, source.SiteType, source.RSSURL, source.Enabled,
		source.PollInterval, source.MaxItems, source.Filters, source.ID)

	return err
}

// DeleteRSSSource 删除RSS源
func (d *Database) DeleteRSSSource(id int64) error {
	_, err := d.db.Exec("DELETE FROM rss_sources WHERE id = ?", id)
	return err
}

// ToggleRSSSource 切换RSS源的启用状态
func (d *Database) ToggleRSSSource(id int64, enabled bool) error {
	_, err := d.db.Exec("UPDATE rss_sources SET enabled = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?", enabled, id)
	return err
}

// UpdateRSSSourceFetchStatus 更新RSS源的抓取状态
func (d *Database) UpdateRSSSourceFetchStatus(id int64, lastFetch time.Time, lastError string) error {
	_, err := d.db.Exec(`
		UPDATE rss_sources
		SET last_fetch = ?, last_error = ?,
		    error_count = CASE WHEN ? != '' THEN error_count + 1 ELSE 0 END,
		    updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, lastFetch, lastError, lastError, id)

	return err
}

// ===== 下载任务相关方法 =====

// GetDownloadTasks 获取下载任务
func (d *Database) GetDownloadTasks(sourceID int64, status string, page, pageSize int) ([]DownloadTask, int, error) {
	// 构建查询条件
	where := "WHERE 1=1"
	args := []any{}
	argIndex := 1

	if sourceID > 0 {
		where += fmt.Sprintf(" AND source_id = $%d", argIndex)
		args = append(args, sourceID)
		argIndex++
	}

	if status != "" {
		where += fmt.Sprintf(" AND status = $%d", argIndex)
		args = append(args, status)
		argIndex++
	}

	// 获取总数
	countQuery := "SELECT COUNT(*) FROM download_tasks " + where
	var total int
	err := d.db.QueryRow(countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	// 分页查询
	query := `
		SELECT id, source_id, guid, title, url, size, category, pub_date, status,
		       file_path, error_msg, retry_count, created_at, updated_at
		FROM download_tasks
	` + where + `
		ORDER BY created_at DESC
	`
	if pageSize > 0 {
		offset := (page - 1) * pageSize
		query += fmt.Sprintf(" LIMIT %d OFFSET %d", pageSize, offset)
	}

	// SQLite使用?占位符,需要替换$1, $2等为?
	for i := range args {
		query = strings.Replace(query, fmt.Sprintf("$%d", i+1), "?", 1)
	}

	rows, err := d.db.Query(query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var tasks []DownloadTask
	for rows.Next() {
		var t DownloadTask
		var pubDate sql.NullTime
		var filePath sql.NullString
		var errorMsg sql.NullString
		err := rows.Scan(&t.ID, &t.SourceID, &t.GUID, &t.Title, &t.URL, &t.Size,
			&t.Category, &pubDate, &t.Status, &filePath, &errorMsg,
			&t.RetryCount, &t.CreatedAt, &t.UpdatedAt)
		if err != nil {
			return nil, 0, err
		}
		if pubDate.Valid {
			t.PubDate = pubDate.Time
		}
		if filePath.Valid {
			t.FilePath = filePath.String
		}
		if errorMsg.Valid {
			t.ErrorMsg = errorMsg.String
		}
		tasks = append(tasks, t)
	}

	return tasks, total, nil
}

// TaskExists 检查任务是否已存在
func (d *Database) TaskExists(guid string) (bool, error) {
	var exists bool
	err := d.db.QueryRow("SELECT EXISTS(SELECT 1 FROM download_tasks WHERE guid = ?)", guid).Scan(&exists)
	return exists, err
}

// CreateDownloadTask 创建下载任务
func (d *Database) CreateDownloadTask(task DownloadTask) (int64, error) {
	result, err := d.db.Exec(`
		INSERT INTO download_tasks (source_id, guid, title, url, size, category, pub_date, status)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, task.SourceID, task.GUID, task.Title, task.URL, task.Size, task.Category, task.PubDate, task.Status)

	if err != nil {
		return 0, err
	}

	return result.LastInsertId()
}

// UpdateTaskStatus 更新任务状态
func (d *Database) UpdateTaskStatus(id int64, status, filePath, errorMsg string) error {
	_, err := d.db.Exec(`
		UPDATE download_tasks
		SET status = ?, file_path = ?, error_msg = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, status, filePath, errorMsg, id)

	return err
}

// UpdateTasksStatus 批量更新任务状态
func (d *Database) UpdateTasksStatus(ids []int64, status string) error {
	if len(ids) == 0 {
		return nil
	}

	placeholders := strings.Repeat("?,", len(ids)-1) + "?"
	args := make([]any, len(ids)+1)
	for i, id := range ids {
		args[i] = id
	}
	args[len(ids)] = status

	_, err := d.db.Exec(`
		UPDATE download_tasks
		SET status = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id IN (`+placeholders+`)
	`, args...)

	return err
}

// DeleteTasks 删除任务
func (d *Database) DeleteTasks(ids []int64) error {
	if len(ids) == 0 {
		return nil
	}

	placeholders := strings.Repeat("?,", len(ids)-1) + "?"
	args := make([]any, len(ids))
	for i, id := range ids {
		args[i] = id
	}

	_, err := d.db.Exec(`DELETE FROM download_tasks WHERE id IN (`+placeholders+`)`, args...)
	return err
}

// CleanupCompletedTasks 清理已完成任务
func (d *Database) CleanupCompletedTasks(olderThan time.Time) error {
	_, err := d.db.Exec(`
		DELETE FROM download_tasks
		WHERE status = 'completed' AND updated_at < ?
	`, olderThan)
	return err
}

// ===== 日志相关方法 =====

// AddLog 添加日志
func (d *Database) AddLog(level string, sourceID *int64, message, details string) error {
	_, err := d.db.Exec(`
		INSERT INTO system_logs (level, source_id, message, details)
		VALUES (?, ?, ?, ?)
	`, level, sourceID, message, details)

	return err
}

// GetLogs 获取日志
func (d *Database) GetLogs(level string, limit int) ([]SystemLog, error) {
	query := `
		SELECT id, level, source_id, message, details, created_at
		FROM system_logs
		WHERE 1=1
	`
	args := []any{}

	if level != "" {
		query += " AND level = ?"
		args = append(args, level)
	}

	query += " ORDER BY created_at DESC"
	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)
	}

	rows, err := d.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []SystemLog
	for rows.Next() {
		var l SystemLog
		var sourceID sql.NullInt64
		err := rows.Scan(&l.ID, &l.Level, &sourceID, &l.Message, &l.Details, &l.CreatedAt)
		if err != nil {
			return nil, err
		}
		if sourceID.Valid {
			l.SourceID = &sourceID.Int64
		}
		logs = append(logs, l)
	}

	return logs, nil
}

// ===== 系统统计相关方法 =====

// GetSystemStats 获取系统统计信息
func (d *Database) GetSystemStats() (*SystemStats, error) {
	var stats SystemStats

	// 获取RSS源统计
	d.db.QueryRow("SELECT COUNT(*), SUM(enabled) FROM rss_sources").Scan(&stats.TotalSources, &stats.EnabledSources)

	// 获取任务统计
	d.db.QueryRow(`
		SELECT
			COUNT(*),
			SUM(CASE WHEN status = 'pending' THEN 1 ELSE 0 END),
			SUM(CASE WHEN status = 'downloading' THEN 1 ELSE 0 END),
			SUM(CASE WHEN status = 'completed' THEN 1 ELSE 0 END),
			SUM(CASE WHEN status = 'failed' THEN 1 ELSE 0 END)
		FROM download_tasks
	`).Scan(&stats.TotalTasks, &stats.PendingTasks, &stats.DownloadingTasks,
		&stats.CompletedTasks, &stats.FailedTasks)

	// 获取下载数统计
	d.db.QueryRow("SELECT COUNT(*) FROM downloads").Scan(&stats.TotalDownloads)

	// 获取最后抓取时间
	var lastFetch sql.NullTime
	d.db.QueryRow("SELECT MAX(last_fetch) FROM rss_sources").Scan(&lastFetch)
	if lastFetch.Valid {
		stats.LastFetchTime = lastFetch.Time
	}

	return &stats, nil
}

// UpdateSystemStats 更新系统统计
func (d *Database) UpdateSystemStats() error {
	stats, err := d.GetSystemStats()
	if err != nil {
		return err
	}

	_, err = d.db.Exec(`
		UPDATE system_stats
		SET total_sources = ?, enabled_sources = ?, total_tasks = ?,
		    pending_tasks = ?, downloading_tasks = ?, completed_tasks = ?,
		    failed_tasks = ?, total_downloads = ?, last_fetch_time = ?,
		    updated_at = CURRENT_TIMESTAMP
		WHERE id = 1
	`, stats.TotalSources, stats.EnabledSources, stats.TotalTasks,
		stats.PendingTasks, stats.DownloadingTasks, stats.CompletedTasks,
		stats.FailedTasks, stats.TotalDownloads, stats.LastFetchTime)

	return err
}

type RSSItemCache struct {
	GUID     string    `json:"guid"`
	Title    string    `json:"title"`
	URL      string    `json:"url"`
	PubDate  time.Time `json:"pub_date"`
	Category string    `json:"category,omitempty"`
	Size     int64     `json:"size,omitempty"`
}

type RSSItemDisplay struct {
	GUID         string    `json:"guid"`
	Title        string    `json:"title"`
	URL          string    `json:"url"`
	PubDate      time.Time `json:"pub_date"`
	IsDownloaded bool      `json:"is_downloaded"`
	DownloadedAt time.Time `json:"downloaded_at,omitempty"`
	FilePath     string    `json:"file_path,omitempty"`
}

// RSSSource RSS源配置
type RSSSource struct {
	ID           int64  `json:"id" db:"id"`
	Name         string `json:"name" db:"name"`
	SiteType     string `json:"site_type" db:"site_type"`
	RSSURL       string `json:"rss_url" db:"rss_url"`
	Enabled      bool   `json:"enabled" db:"enabled"`
	PollInterval int    `json:"poll_interval" db:"poll_interval"`
	MaxItems     int    `json:"max_items" db:"max_items"`
	Filters      string `json:"filters,omitempty" db:"filters"`
	LastFetch    time.Time `json:"last_fetch" db:"last_fetch"`
	LastError    string `json:"last_error" db:"last_error"`
	ErrorCount   int    `json:"error_count" db:"error_count"`
}

// DownloadTask 下载任务
type DownloadTask struct {
	ID          int64     `json:"id" db:"id"`
	SourceID    int64     `json:"source_id" db:"source_id"`
	GUID        string    `json:"guid" db:"guid"`
	Title       string    `json:"title" db:"title"`
	URL         string    `json:"url" db:"url"`
	Size        int64     `json:"size" db:"size"`
	Category    string    `json:"category" db:"category"`
	PubDate     time.Time `json:"pub_date" db:"pub_date"`
	Status      string    `json:"status" db:"status"` // pending, downloading, completed, failed
	FilePath    string    `json:"file_path" db:"file_path"`
	ErrorMsg    string    `json:"error_msg" db:"error_msg"`
	RetryCount  int       `json:"retry_count" db:"retry_count"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

// SystemLog 系统日志
type SystemLog struct {
	ID        int64      `json:"id" db:"id"`
	Level     string     `json:"level" db:"level"` // info, warning, error
	SourceID  *int64     `json:"source_id,omitempty" db:"source_id"`
	Message   string     `json:"message" db:"message"`
	Details   string     `json:"details,omitempty" db:"details"`
	CreatedAt time.Time  `json:"created_at" db:"created_at"`
}

// SystemStats 系统统计信息
type SystemStats struct {
	TotalSources     int       `json:"total_sources"`
	EnabledSources   int       `json:"enabled_sources"`
	TotalTasks       int       `json:"total_tasks"`
	PendingTasks     int       `json:"pending_tasks"`
	DownloadingTasks int       `json:"downloading_tasks"`
	CompletedTasks   int       `json:"completed_tasks"`
	FailedTasks      int       `json:"failed_tasks"`
	TotalDownloads   int       `json:"total_downloads"`
	LastFetchTime    time.Time `json:"last_fetch_time"`
}
