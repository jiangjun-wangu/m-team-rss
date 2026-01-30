package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	RSSURL        string        `yaml:"rss_url"`
	PollInterval  time.Duration `yaml:"poll_interval"`
	SavePath      string        `yaml:"save_path"`
	MaxConcurrent int           `yaml:"max_concurrent"`
	DBPath        string        `yaml:"db_path"`
	WebPort       int           `yaml:"web_port"`
	WebHost       string        `yaml:"web_host"`
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

	// 设置默认值
	if cfg.WebPort == 0 {
		cfg.WebPort = 8080
	}
	if cfg.WebHost == "" {
		cfg.WebHost = "0.0.0.0"
	}
	if cfg.PollInterval == 0 {
		cfg.PollInterval = 5 * time.Minute
	}
	if cfg.MaxConcurrent == 0 {
		cfg.MaxConcurrent = 3
	}
	if cfg.SavePath == "" {
		cfg.SavePath = "./torrents"
	}
	if cfg.DBPath == "" {
		cfg.DBPath = "./data/downloads.db"
	}

	// 确保目录存在
	if err := os.MkdirAll(cfg.SavePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create save path: %w", err)
	}
	if err := os.MkdirAll(cfg.DBPath[:len(cfg.DBPath)-len("/downloads.db")], 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	return &cfg, nil
}
