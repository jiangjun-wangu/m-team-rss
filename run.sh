#!/bin/bash

# Linux/Mac启动脚本示例

# 方式1: 使用默认配置文件
# ./mteam-rss

# 方式2: 使用命令行参数（不依赖配置文件）
./mteam-rss \
  -db "./data/downloads.db" \
  -save-path "./torrents" \
  -host "0.0.0.0" \
  -port 8080

# 方式3: 使用环境变量
# export DB_PATH="./data/downloads.db"
# export SAVE_PATH="./torrents"
# export HOST="0.0.0.0"
# export PORT="8080"
# ./mteam-rss
