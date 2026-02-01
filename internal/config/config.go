package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server       ServerConfig       `yaml:"server"`
	Database     DatabaseConfig     `yaml:"database"`
	Download     DownloadConfig     `yaml:"download"`
	RSSSources   []RSSSourceConfig  `yaml:"rss_sources"`
	Logging      LoggingConfig      `yaml:"logging"`

	// 兼容旧配置字段
	RSSURL        string        `yaml:"rss_url"`
	PollInterval  time.Duration `yaml:"poll_interval"`
	SavePath      string        `yaml:"save_path"`
	MaxConcurrent int           `yaml:"max_concurrent"`
	DBPath        string        `yaml:"db_path"`
	WebPort       int           `yaml:"web_port"`
	WebHost       string        `yaml:"web_host"`
}

type ServerConfig struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

type DatabaseConfig struct {
	Path string `yaml:"path"`
}

type DownloadConfig struct {
	SavePath     string        `yaml:"save_path"`
	MaxConcurrent int          `yaml:"max_concurrent"`
	Timeout       time.Duration `yaml:"timeout"`
	RetryLimit    int           `yaml:"retry_limit"`
}

type RSSSourceConfig struct {
	Name         string            `yaml:"name"`
	SiteType     string            `yaml:"site_type"`
	RSSURL       string            `yaml:"rss_url"`
	Enabled      bool              `yaml:"enabled"`
	PollInterval int               `yaml:"poll_interval"`
	MaxItems     int               `yaml:"max_items"`
	Filters      map[string]any    `yaml:"filters,omitempty"`
}

type LoggingConfig struct {
	Level     string `yaml:"level"`
	MaxSizeMB int    `yaml:"max_size_mb"`
	MaxFiles  int    `yaml:"max_files"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	// 兼容旧配置: 如果新配置未设置,从旧配置复制
	if cfg.Server.Host == "" && cfg.WebHost != "" {
		cfg.Server.Host = cfg.WebHost
	}
	if cfg.Server.Port == 0 && cfg.WebPort != 0 {
		cfg.Server.Port = cfg.WebPort
	}
	if cfg.Database.Path == "" && cfg.DBPath != "" {
		cfg.Database.Path = cfg.DBPath
	}
	if cfg.Download.SavePath == "" && cfg.SavePath != "" {
		cfg.Download.SavePath = cfg.SavePath
	}
	if cfg.Download.MaxConcurrent == 0 && cfg.MaxConcurrent != 0 {
		cfg.Download.MaxConcurrent = cfg.MaxConcurrent
	}

	// 设置默认值
	if cfg.Server.Host == "" {
		cfg.Server.Host = "0.0.0.0"
	}
	if cfg.Server.Port == 0 {
		cfg.Server.Port = 8080
	}
	if cfg.Database.Path == "" {
		cfg.Database.Path = "./data/downloads.db"
	}
	if cfg.Download.SavePath == "" {
		cfg.Download.SavePath = "./torrents"
	}
	if cfg.Download.MaxConcurrent == 0 {
		cfg.Download.MaxConcurrent = 3
	}
	if cfg.Download.Timeout == 0 {
		cfg.Download.Timeout = 60 * time.Second
	}
	if cfg.Download.RetryLimit == 0 {
		cfg.Download.RetryLimit = 3
	}
	if cfg.Logging.Level == "" {
		cfg.Logging.Level = "info"
	}
	if cfg.Logging.MaxSizeMB == 0 {
		cfg.Logging.MaxSizeMB = 100
	}
	if cfg.Logging.MaxFiles == 0 {
		cfg.Logging.MaxFiles = 10
	}

	// 设置默认RSS源轮询间隔
	for i := range cfg.RSSSources {
		if cfg.RSSSources[i].PollInterval == 0 {
			cfg.RSSSources[i].PollInterval = 15
		}
		if cfg.RSSSources[i].MaxItems == 0 {
			cfg.RSSSources[i].MaxItems = 50
		}
	}

	// 确保目录存在
	if err := os.MkdirAll(cfg.Download.SavePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create save path: %w", err)
	}
	dbDir := cfg.Database.Path
	if idx := lastIndexOf(dbDir, "/"); idx > 0 {
		dbDir = dbDir[:idx]
	} else if idx := lastIndexOf(dbDir, "\\"); idx > 0 {
		dbDir = dbDir[:idx]
	}
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	return &cfg, nil
}

func lastIndexOf(s, substr string) int {
	for i := len(s) - len(substr); i >= 0; i-- {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
