package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/gin-gonic/gin"
	"mteam-rss/internal/config"
	"mteam-rss/internal/database"
	"mteam-rss/internal/scheduler"
	"mteam-rss/internal/web"
)

func main() {
	// 解析命令行参数
	configPath := flag.String("config", "config.yaml", "配置文件路径")
	flag.Parse()

	// 加载配置
	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	log.Printf("配置加载成功: RSS URL=%s, SavePath=%s, WebPort=%d", cfg.RSSURL, cfg.SavePath, cfg.WebPort)

	// 初始化数据库
	db, err := database.New(cfg.DBPath)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()
	log.Println("数据库初始化成功")

	// 创建调度器
	sched := scheduler.New(cfg.RSSURL, cfg.PollInterval, cfg.SavePath, cfg.MaxConcurrent, db)

	// 启动调度器
	if err := sched.Start(cfg.PollInterval); err != nil {
		log.Fatalf("Failed to start scheduler: %v", err)
	}
	log.Println("调度器启动成功")

	// 创建Web处理器
	handler := web.New(db, sched, cfg.RSSURL)

	// 配置Gin
	gin.SetMode(gin.ReleaseMode)
	router := gin.Default()

	// 加载HTML模板
	router.LoadHTMLGlob("internal/web/templates/*")

	// 静态文件
	router.Static("/static", "internal/web/static")

	// 注册路由
	handler.RegisterRoutes(router)

	// 启动Web服务器
	addr := fmt.Sprintf("%s:%d", cfg.WebHost, cfg.WebPort)
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
	sched.Stop()
	log.Println("程序已退出")
}
