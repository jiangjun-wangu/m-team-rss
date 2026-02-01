@echo off
REM Windows启动脚本示例

REM 方式1: 使用默认配置文件
REM mteam-rss.exe

REM 方式2: 使用命令行参数（不依赖配置文件）
mteam-rss.exe ^
  -db "./data/downloads.db" ^
  -save-path "./torrents" ^
  -host "0.0.0.0" ^
  -port 8081

REM 方式3: 使用环境变量
REM set DB_PATH=./data/downloads.db
REM set SAVE_PATH=./torrents
REM set HOST=0.0.0.0
REM set PORT=8081
REM mteam-rss.exe

pause
