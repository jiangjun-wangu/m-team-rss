@echo off
REM 本地Docker镜像构建脚本（Windows，不依赖Docker Hub）

setlocal EnableDelayedExpansion

echo =========================================
echo M-Team RSS下载器 - 本地Docker镜像构建
echo =========================================
echo.

REM 编译Go程序（交叉编译为Linux）
echo 步骤 1: 编译Go程序...
set GOOS=linux
set GOARCH=amd64
set CGO_ENABLED=1

go build -ldflags="-w -s" -o mteam-downloader-linux-amd64.exe .

if %ERRORLEVEL% NEQ 0 (
    echo ✗ 编译失败
    pause
    exit /b 1
)

echo ✓ 编译成功: mteam-downloader-linux-amd64.exe

REM 创建本地Dockerfile
echo.
echo 步骤 2: 创建本地Dockerfile...
(
echo FROM alpine:latest
echo RUN apk --no-cache add ca-certificates tzdata sqlite
echo ENV TZ=Asia/Shanghai
echo WORKDIR /app
echo COPY mteam-downloader-linux-amd64.exe ./mteam-downloader
echo COPY internal/web/templates ./web/templates
echo RUN mkdir -p /app/config /app/torrents /app/data
echo VOLUME ["/app/config", "/app/torrents", "/app/data"^)
echo EXPOSE 8080
echo CMD ["./mteam-downloader", "-config", "/app/config/config.yaml"]
) > Dockerfile.localbuild

REM 构建Docker镜像
echo.
echo 步骤 3: 构建Docker镜像...
docker build -f Dockerfile.localbuild -t mteam-rss:latest .

if %ERRORLEVEL% NEQ 0 (
    echo ✗ Docker构建失败
    pause
    exit /b 1
)

REM 显示镜像信息
echo.
echo 步骤 4: 镜像构建完成！
docker images | findstr mteam-rss

REM 清理临时文件
echo.
echo 步骤 5: 清理临时文件...
del /F /Q mteam-downloader-linux-amd64.exe
REM del /F /Q Dockerfile.localbuild

echo.
echo =========================================
echo 本地构建完成！
echo 镜像名称: mteam-rss:latest
echo.
echo 运行命令:
echo   docker run -d -p 8080:8080 -v %cd%\config:/app/config -v %cd%\torrents:/app/torrents -v %cd%\data:/app/data mteam-rss:latest
echo =========================================

pause
