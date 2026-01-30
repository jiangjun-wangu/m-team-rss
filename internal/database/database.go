package database

import (
	"database/sql"
	"encoding/json"
	"fmt"
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
		`CREATE INDEX IF NOT EXISTS idx_rss_cache_pub_date ON rss_cache(pub_date DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_rss_cache_cached_at ON rss_cache(cached_at)`,
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
		var filePath, downloadedAt sql.NullString
		err := rows.Scan(&item.GUID, &item.Title, &item.URL, &item.PubDate, &filePath, &downloadedAt)
		if err != nil {
			return nil, err
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
