package web

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"mteam-rss/internal/database"
	"mteam-rss/internal/scheduler"
)

type Handler struct {
	db        *database.Database
	scheduler *scheduler.Scheduler
	rssURL    string
}

type SystemStatusResponse struct {
	RSSURL     string    `json:"rss_url"`
	Status     string    `json:"status"`
	LastFetch  string    `json:"last_fetch"`
	NextFetch  string    `json:"next_fetch"`
	TotalItems int       `json:"total_items"`
	Downloaded int       `json:"downloaded"`
	Pending    int       `json:"pending"`
	LastError  string    `json:"last_error,omitempty"`
}

type RSSItemsResponse struct {
	Items []database.RSSItemDisplay `json:"items"`
}

type ConfigResponse struct {
	RSSURL       string `json:"rss_url"`
	PollInterval string `json:"poll_interval"`
	SavePath     string `json:"save_path"`
	MaxConcurrent int   `json:"max_concurrent"`
	WebPort      int    `json:"web_port"`
}

func New(db *database.Database, scheduler *scheduler.Scheduler, rssURL string) *Handler {
	return &Handler{
		db:        db,
		scheduler: scheduler,
		rssURL:    rssURL,
	}
}

func (h *Handler) RegisterRoutes(router *gin.Engine) {
	router.GET("/", h.indexPage)
	router.GET("/api/status", h.getStatus)
	router.GET("/api/rss-items", h.getRSSItems)
	router.GET("/api/config", h.getConfig)
	router.POST("/api/trigger", h.triggerFetch)
}

func (h *Handler) indexPage(c *gin.Context) {
	c.HTML(http.StatusOK, "index.html", gin.H{
		"title": "M-Team RSS下载器",
	})
}

func (h *Handler) getStatus(c *gin.Context) {
	status := "stopped"
	if h.scheduler.IsRunning() {
		status = "running"
	}

	lastFetch := h.scheduler.GetLastFetch()
	nextFetch := h.scheduler.GetNextFetch()
	lastError := h.scheduler.GetLastError()

	total, downloaded, err := h.db.GetStats()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	pending := total - downloaded
	if pending < 0 {
		pending = 0
	}

	response := SystemStatusResponse{
		RSSURL:     h.rssURL,
		Status:     status,
		LastFetch:  formatTime(lastFetch),
		NextFetch:  formatTime(nextFetch),
		TotalItems: total,
		Downloaded: downloaded,
		Pending:    pending,
		LastError:  lastError,
	}

	c.JSON(http.StatusOK, response)
}

func (h *Handler) getRSSItems(c *gin.Context) {
	limit := 100 // 默认限制返回数量
	items, err := h.db.GetCachedRSSItems(limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	response := RSSItemsResponse{
		Items: items,
	}

	c.JSON(http.StatusOK, response)
}

func (h *Handler) getConfig(c *gin.Context) {
	response := ConfigResponse{
		RSSURL:        h.rssURL,
		PollInterval:  "5m",
		SavePath:      "/app/torrents",
		MaxConcurrent: 3,
		WebPort:       8080,
	}

	c.JSON(http.StatusOK, response)
}

func (h *Handler) triggerFetch(c *gin.Context) {
	if err := h.scheduler.TriggerNow(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Fetch triggered successfully",
	})
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}
