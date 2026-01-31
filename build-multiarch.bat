@echo off
REM 多架构Docker镜像构建脚本 (Windows)

setlocal enabledelayedexpansion

if "%~1"=="" (
    set VERSION=latest
) else (
    set VERSION=%~1
)

echo === 构建多架构Docker镜像 ===
echo 镜像名称: renjianhuashui/mteam-rss
echo 版本: %VERSION%
echo.

REM 构建多架构镜像
echo 正在构建支持 amd64 和 arm64 的镜像...
docker buildx build ^
  --platform linux/amd64,linux/arm64 ^
  --tag renjianhuashui/mteam-rss:%VERSION% ^
  --tag renjianhuashui/mteam-rss:amd64 ^
  --tag renjianhuashui/mteam-rss:arm64 ^
  --push .

echo.
echo === 构建完成 ===
echo 已构建的架构:
echo   - linux/amd64
echo   - linux/arm64
echo.
echo 推送的标签:
echo   - renjianhuashui/mteam-rss:%VERSION%
echo   - renjianhuashui/mteam-rss:amd64
echo   - renjianhuashui/mteam-rss:arm64

endlocal
