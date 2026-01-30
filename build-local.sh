#!/bin/bash

# 本地Docker镜像构建脚本（不依赖Docker Hub）

set -e

echo "========================================="
echo "M-Team RSS下载器 - 本地Docker镜像构建"
echo "========================================="
echo ""

# 检查操作系统
OS_TYPE=$(uname -s)
ARCH_TYPE=$(uname -m)

echo "检测到系统: $OS_TYPE $ARCH_TYPE"

# 编译Go程序（本地编译，不依赖docker网络）
echo ""
echo "步骤 1: 编译Go程序..."
CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o mteam-downloader-linux-amd64 .

if [ $? -eq 0 ]; then
    echo "✓ 编译成功: mteam-downloader-linux-amd64"
else
    echo "✗ 编译失败"
    exit 1
fi

# 创建临时Dockerfile（使用本地编译的二进制）
echo ""
echo "步骤 2: 创建本地Dockerfile..."
cat > Dockerfile.localbuild << 'EOF'
FROM alpine:latest
RUN apk --no-cache add ca-certificates tzdata sqlite
ENV TZ=Asia/Shanghai
WORKDIR /app
COPY mteam-downloader-linux-amd64 .
COPY internal/web/templates ./web/templates
RUN mkdir -p /app/config /app/torrents /app/data
VOLUME ["/app/config", "/app/torrents", "/app/data"]
EXPOSE 8080
CMD ["./mteam-downloader-linux-amd64", "-config", "/app/config/config.yaml"]
EOF

# 构建Docker镜像
echo ""
echo "步骤 3: 构建Docker镜像（不依赖网络）..."
docker build -f Dockerfile.localbuild -t mteam-rss:latest .

# 显示镜像信息
echo ""
echo "步骤 4: 镜像构建完成！"
docker images | grep mteam-rss

# 清理临时文件
echo ""
echo "步骤 5: 清理临时文件..."
rm -f mteam-downloader-linux-amd64
# rm -f Dockerfile.localbuild  # 保留以便调试

echo ""
echo "========================================="
echo "本地构建完成！"
echo "镜像名称: mteam-rss:latest"
echo "镜像大小:"
docker images mteam-rss:latest --format "  {{.Size}} bytes"
echo ""
echo "运行命令:"
echo "  docker run -d -p 8080:8080 -v \$(pwd)/config:/app/config -v \$(pwd)/torrents:/app/torrents -v \$(pwd)/data:/app/data mteam-rss:latest"
echo "========================================="
