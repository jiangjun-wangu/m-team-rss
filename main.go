package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"

	"github.com/gin-gonic/gin"
	"mteam-rss/internal/config"
	"mteam-rss/internal/configimport"
	"mteam-rss/internal/database"
	"mteam-rss/internal/scheduler"
	"mteam-rss/internal/web"
)

func main() {
	// 解析命令行参数
	configPath := flag.String("config", "", "配置文件路径 (默认: config/config.yaml)")
	dbPath := flag.String("db", "", "数据库路径 (默认: ./data/downloads.db)")
	savePath := flag.String("save-path", "", "下载保存路径 (默认: ./torrents)")
	host := flag.String("host", "", "Web服务器监听地址 (默认: 0.0.0.0)")
	port := flag.Int("port", 0, "Web服务器端口 (默认: 8080 或配置文件中的值)")
	flag.Parse()

	// 支持环境变量 (优先级低于命令行参数)
	if *configPath == "" {
		if envPath := os.Getenv("CONFIG_PATH"); envPath != "" {
			*configPath = envPath
		} else {
			// 默认使用 config/config.yaml
			*configPath = "config/config.yaml"
		}
	}

	if *dbPath == "" {
		if envPath := os.Getenv("DB_PATH"); envPath != "" {
			*dbPath = envPath
		}
	}

	if *savePath == "" {
		if envPath := os.Getenv("SAVE_PATH"); envPath != "" {
			*savePath = envPath
		}
	}

	if *host == "" {
		if envHost := os.Getenv("HOST"); envHost != "" {
			*host = envHost
		}
	}

	if *port == 0 {
		if envPort := os.Getenv("PORT"); envPort != "" {
			if p, err := strconv.Atoi(envPort); err == nil {
				*port = p
			}
		}
	}

	// 加载配置文件 (如果存在)
	var cfg *config.Config
	if _, err := os.Stat(*configPath); err == nil {
		log.Printf("加载配置文件: %s", *configPath)
		cfg, err = config.Load(*configPath)
		if err != nil {
			log.Fatalf("Failed to load config: %v", err)
		}
		log.Printf("配置文件加载成功: %d个RSS源", len(cfg.RSSSources))
	} else {
		log.Printf("配置文件不存在: %s, 使用默认配置", *configPath)
		cfg = &config.Config{}
	}

	// 命令行参数覆盖配置文件
	if *dbPath != "" {
		cfg.Database.Path = *dbPath
	}
	if cfg.Database.Path == "" {
		cfg.Database.Path = "./data/downloads.db"
	}

	if *savePath != "" {
		cfg.Download.SavePath = *savePath
	}
	if cfg.Download.SavePath == "" {
		cfg.Download.SavePath = "./torrents"
	}

	if *host != "" {
		cfg.Server.Host = *host
	}
	if cfg.Server.Host == "" {
		cfg.Server.Host = "0.0.0.0"
	}

	if *port != 0 {
		cfg.Server.Port = *port
	}
	if cfg.Server.Port == 0 {
		cfg.Server.Port = 8080
	}

	// 确保目录存在
	if err := os.MkdirAll(cfg.Download.SavePath, 0755); err != nil {
		log.Fatalf("Failed to create save path: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(cfg.Database.Path), 0755); err != nil {
		log.Fatalf("Failed to create data directory: %v", err)
	}

	// 初始化数据库
	db, err := database.New(cfg.Database.Path)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()
	log.Printf("数据库初始化成功: %s", cfg.Database.Path)

	// 从配置文件导入RSS源
	if len(cfg.RSSSources) > 0 {
		if err := configimport.ImportConfigSources(cfg, db); err != nil {
			log.Printf("Warning: Failed to import config sources: %v", err)
		}
	}

	// 创建多RSS源调度器
	multiSched := scheduler.NewMultiScheduler(db, nil, 3)

	// 启动调度器
	if err := multiSched.Start(); err != nil {
		log.Fatalf("Failed to start multi scheduler: %v", err)
	}
	log.Println("多RSS源调度器启动成功")

	// 创建任务下载调度器
	taskSched := scheduler.NewTaskDownloadScheduler(db, cfg.Download.MaxConcurrent)

	// 启动任务下载调度器
	if err := taskSched.Start(); err != nil {
		log.Fatalf("Failed to start task download scheduler: %v", err)
	}
	log.Printf("任务下载调度器启动成功: %d workers", cfg.Download.MaxConcurrent)

	// 创建Web处理器
	handler := web.NewMulti(db, multiSched)

	// 配置Gin
	ginMode := os.Getenv("GIN_MODE")
	if ginMode == "" {
		ginMode = gin.ReleaseMode
	}
	gin.SetMode(ginMode)
	router := gin.Default()

	// 加载HTML模板
	templatePaths := []string{
		"internal/web/templates/*",
		"web/templates/*",
		"/app/web/templates/*", // Docker环境
	}
	var templatePath string
	for _, p := range templatePaths {
		if _, err := os.Stat(filepath.Dir(p)); err == nil {
			// 检查目录是否存在
			files, _ := filepath.Glob(p)
			if len(files) > 0 {
				templatePath = p
				break
			}
		}
	}
	if templatePath != "" {
		router.LoadHTMLGlob(templatePath)
		log.Printf("加载HTML模板: %s", templatePath)
	}

	// 静态文件路由
	router.Static("/static", "./internal/web/static")
	router.StaticFile("/favicon.ico", "./internal/web/static/favicon.ico")

	// 注册API路由
	handler.RegisterRoutes(router)

	// 启动Web服务器
	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	log.Printf("Web服务器启动在: http://%s", addr)

	go func() {
		if err := router.Run(addr); err != nil {
			log.Fatalf("Failed to start web server: %v", err)
		}
	}()

	// 等待中断信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	log.Println("正在关闭...")
	taskSched.Stop()
	multiSched.Stop()
	log.Println("程序已退出")
}

