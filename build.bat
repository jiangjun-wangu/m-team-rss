@echo off
REM M-Team RSS下载器 - Docker镜像构建脚本 (Windows)

setlocal EnableDelayedExpansion

REM 配置变量
set IMAGE_NAME=mteam-rss
set IMAGE_TAG=%1
if "%IMAGE_TAG%"=="" set IMAGE_TAG=latest
set REGISTRY=docker.io
set USERNAME=

echo =========================================
echo M-Team RSS下载器 - Docker镜像构建
echo =========================================
echo.

REM 清理旧的构建文件
echo 步骤 1: 清理旧的构建文件...
if exist mteam-downloader del /F /Q mteam-downloader.exe 2>nul
if exist mteam-downloader del /F /Q mteam-downloader 2>nul

REM 构建Docker镜像
echo.
echo 步骤 2: 构建Docker镜像...
if "%USERNAME%"=="" (
    set FULL_IMAGE_NAME=%IMAGE_NAME%:%IMAGE_TAG%
) else (
    set FULL_IMAGE_NAME=%USERNAME%/%IMAGE_NAME%:%IMAGE_TAG%
)

echo 构建镜像: %FULL_IMAGE_NAME%
docker build -t %FULL_IMAGE_NAME% .

REM 显示镜像信息
echo.
echo 步骤 3: 镜像构建完成！
docker images | findstr %IMAGE_NAME%

echo.
echo =========================================
echo 构建完成！
echo 镜像名称: %FULL_IMAGE_NAME%
echo.
echo 使用以下命令推送镜像:
if "%USERNAME%"=="" (
    echo   docker push %FULL_IMAGE_NAME%
) else (
    echo   docker push %FULL_IMAGE_NAME%
)
echo.
echo 使用以下命令运行:
echo   docker run -d -p 8080:8080 -v %cd%/config:/app/config -v %cd%/torrents:/app/torrents -v %cd%/data:/app/data %FULL_IMAGE_NAME%
echo =========================================

pause
