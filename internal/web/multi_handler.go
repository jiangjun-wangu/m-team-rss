package web

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"mteam-rss/internal/database"
	"mteam-rss/internal/scheduler"
)

// MultiHandler 多RSS源API处理器
type MultiHandler struct {
	db        *database.Database
	scheduler *scheduler.MultiScheduler
}

func NewMulti(db *database.Database, scheduler *scheduler.MultiScheduler) *MultiHandler {
	return &MultiHandler{
		db:        db,
		scheduler: scheduler,
	}
}

// RegisterRoutes 注册路由
func (h *MultiHandler) RegisterRoutes(router *gin.Engine) {
	api := router.Group("/api")
	{
		// RSS源管理
		api.GET("/rss-sources", h.getRSSSources)
		api.POST("/rss-sources", h.createRSSSource)
		api.PUT("/rss-sources/:id", h.updateRSSSource)
		api.DELETE("/rss-sources/:id", h.deleteRSSSource)
		api.PATCH("/rss-sources/:id/toggle", h.toggleRSSSource)
		api.POST("/rss-sources/:id/fetch", h.triggerSourceFetch)

		// 下载任务管理
		api.GET("/tasks", h.getTasks)
		api.POST("/tasks/batch", h.batchTasks)
		api.DELETE("/tasks/cleanup", h.cleanupTasks)
		api.PATCH("/tasks/:id/status", h.updateTaskStatus)

		// 仪表盘
		api.GET("/dashboard/stats", h.getDashboardStats)
		api.GET("/dashboard/logs", h.getDashboardLogs)
	}

	// 页面路由
	router.GET("/", h.dashboardPage)
	router.GET("/rss-sources", h.rssSourcesPage)
	router.GET("/tasks", h.tasksPage)
}

// ========== RSS源管理 API ==========

type CreateRSSSourceRequest struct {
	Name         string `json:"name" binding:"required"`
	SiteType     string `json:"site_type" binding:"required"`
	RSSURL       string `json:"rss_url" binding:"required,url"`
	Enabled      bool   `json:"enabled"`
	PollInterval int    `json:"poll_interval"`
	MaxItems     int    `json:"max_items"`
}

type UpdateRSSSourceRequest struct {
	Name         string `json:"name"`
	SiteType     string `json:"site_type"`
	RSSURL       string `json:"rss_url" binding:"omitempty,url"`
	Enabled      *bool  `json:"enabled"`
	PollInterval int    `json:"poll_interval"`
	MaxItems     int    `json:"max_items"`
}

// getRSSSources 获取RSS源列表
func (h *MultiHandler) getRSSSources(c *gin.Context) {
	sources, err := h.db.GetRSSSources()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"sources": sources,
		},
	})
}

// createRSSSource 创建RSS源
func (h *MultiHandler) createRSSSource(c *gin.Context) {
	var req CreateRSSSourceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": -1, "message": err.Error()})
		return
	}

	source := database.RSSSource{
		Name:         req.Name,
		SiteType:     req.SiteType,
		RSSURL:       req.RSSURL,
		Enabled:      req.Enabled,
		PollInterval: req.PollInterval,
		MaxItems:     req.MaxItems,
	}

	id, err := h.db.CreateRSSSource(source)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "message": err.Error()})
		return
	}

	// 重新加载调度器
	h.scheduler.ReloadSources()
	h.db.UpdateSystemStats()

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "RSS源创建成功",
		"data": gin.H{
			"id": id,
		},
	})
}

// updateRSSSource 更新RSS源
func (h *MultiHandler) updateRSSSource(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": -1, "message": "无效的ID"})
		return
	}

	var req UpdateRSSSourceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": -1, "message": err.Error()})
		return
	}

	// 获取现有源
	source, err := h.db.GetRSSSource(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": -1, "message": "RSS源不存在"})
		return
	}

	// 更新字段
	if req.Name != "" {
		source.Name = req.Name
	}
	if req.SiteType != "" {
		source.SiteType = req.SiteType
	}
	if req.RSSURL != "" {
		source.RSSURL = req.RSSURL
	}
	if req.Enabled != nil {
		source.Enabled = *req.Enabled
	}
	if req.PollInterval > 0 {
		source.PollInterval = req.PollInterval
	}
	if req.MaxItems > 0 {
		source.MaxItems = req.MaxItems
	}

	if err := h.db.UpdateRSSSource(*source); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "message": err.Error()})
		return
	}

	// 重新加载调度器
	h.scheduler.ReloadSources()

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "RSS源更新成功",
	})
}

// deleteRSSSource 删除RSS源
func (h *MultiHandler) deleteRSSSource(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": -1, "message": "无效的ID"})
		return
	}

	if err := h.db.DeleteRSSSource(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "message": err.Error()})
		return
	}

	// 重新加载调度器
	h.scheduler.ReloadSources()
	h.db.UpdateSystemStats()

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "RSS源删除成功",
	})
}

// toggleRSSSource 切换RSS源启用状态
func (h *MultiHandler) toggleRSSSource(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": -1, "message": "无效的ID"})
		return
	}

	// 获取当前状态
	source, err := h.db.GetRSSSource(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "message": err.Error()})
		return
	}
	if source == nil {
		c.JSON(http.StatusNotFound, gin.H{"code": -1, "message": "RSS源不存在"})
		return
	}

	enabled := !source.Enabled
	err = h.db.ToggleRSSSource(id, enabled)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "message": err.Error()})
		return
	}

	// 重新加载调度器
	h.scheduler.ReloadSources()
	h.db.UpdateSystemStats()

	status := "已禁用"
	if enabled {
		status = "已启用"
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "RSS源" + status,
		"data": gin.H{
			"enabled": enabled,
		},
	})
}

// triggerSourceFetch 手动触发RSS源抓取
func (h *MultiHandler) triggerSourceFetch(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": -1, "message": "无效的ID"})
		return
	}

	if err := h.scheduler.TriggerFetch(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "抓取任务已启动",
	})
}

// ========== 下载任务管理 API ==========

type BatchTasksRequest struct {
	Action  string  `json:"action" binding:"required"` // retry, delete, pause
	TaskIDs []int64 `json:"task_ids" binding:"required"`
}

// getTasks 获取任务列表
func (h *MultiHandler) getTasks(c *gin.Context) {
	sourceID, _ := strconv.ParseInt(c.Query("source_id"), 10, 64)
	status := c.Query("status")
	page, _ := strconv.Atoi(c.Query("page"))
	pageSize, _ := strconv.Atoi(c.Query("page_size"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	tasks, total, err := h.db.GetDownloadTasks(sourceID, status, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "message": err.Error()})
		return
	}

	totalPages := (total + pageSize - 1) / pageSize

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"tasks": tasks,
			"pagination": gin.H{
				"total":       total,
				"page":        page,
				"page_size":   pageSize,
				"total_pages": totalPages,
			},
		},
	})
}

// batchTasks 批量操作任务
func (h *MultiHandler) batchTasks(c *gin.Context) {
	var req BatchTasksRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": -1, "message": err.Error()})
		return
	}

	successCount := 0
	var err error

	switch req.Action {
	case "retry":
		// 重试失败任务
		err = h.db.UpdateTasksStatus(req.TaskIDs, "pending")
	case "delete":
		// 删除任务
		err = h.db.DeleteTasks(req.TaskIDs)
	case "pause":
		// 暂停任务
		err = h.db.UpdateTasksStatus(req.TaskIDs, "paused")
	default:
		c.JSON(http.StatusBadRequest, gin.H{"code": -1, "message": "无效的操作"})
		return
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "message": err.Error()})
		return
	}

	successCount = len(req.TaskIDs)
	h.db.UpdateSystemStats()

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "操作成功",
		"data": gin.H{
			"success_count": successCount,
			"failed_count":  0,
		},
	})
}

// cleanupTasks 清理已完成任务
func (h *MultiHandler) cleanupTasks(c *gin.Context) {
	olderThanStr := c.Query("older_than")
	if olderThanStr == "" {
		olderThanStr = "7d" // 默认7天
	}

	duration, err := time.ParseDuration(olderThanStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": -1, "message": "无效的时间格式"})
		return
	}

	cutoffTime := time.Now().Add(-duration)
	if err := h.db.CleanupCompletedTasks(cutoffTime); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "message": err.Error()})
		return
	}

	h.db.UpdateSystemStats()

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "清理完成",
	})
}

// updateTaskStatus 更新单个任务状态
func (h *MultiHandler) updateTaskStatus(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": -1, "message": "无效的ID"})
		return
	}

	var req struct {
		Status string `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": -1, "message": err.Error()})
		return
	}

	if err := h.db.UpdateTaskStatus(id, req.Status, "", ""); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "message": err.Error()})
		return
	}

	h.db.UpdateSystemStats()

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "状态更新成功",
	})
}

// ========== 仪表盘 API ==========

// getDashboardStats 获取系统统计
func (h *MultiHandler) getDashboardStats(c *gin.Context) {
	stats, err := h.db.GetSystemStats()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "message": err.Error()})
		return
	}

	// 获取RSS源状态
	sourcesStatus := h.scheduler.GetSourcesStatus()

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"stats":          stats,
			"sources_status": sourcesStatus,
		},
	})
}

// getDashboardLogs 获取系统日志
func (h *MultiHandler) getDashboardLogs(c *gin.Context) {
	limit := 50
	if l, err := strconv.Atoi(c.Query("limit")); err == nil && l > 0 {
		limit = l
	}

	level := c.Query("level")
	logs, err := h.db.GetLogs(level, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"logs": logs,
		},
	})
}

// ========== 页面路由 ==========

func (h *MultiHandler) dashboardPage(c *gin.Context) {
	c.HTML(http.StatusOK, "dashboard.html", gin.H{
		"title": "仪表盘 - PT种子下载器",
	})
}

func (h *MultiHandler) rssSourcesPage(c *gin.Context) {
	c.HTML(http.StatusOK, "rss-sources.html", gin.H{
		"title": "RSS源管理 - PT种子下载器",
	})
}

func (h *MultiHandler) tasksPage(c *gin.Context) {
	c.HTML(http.StatusOK, "tasks.html", gin.H{
		"title": "任务列表 - PT种子下载器",
	})
}
