#!/bin/bash

# M-Team RSS下载器 - Docker镜像发布脚本

set -e

# 配置变量 - 请修改这些值
USERNAME="your-dockerhub-username"  # Docker Hub用户名
IMAGE_NAME="mteam-rss"
IMAGE_TAG=${1:-latest}
REGISTRY="docker.io"

echo "========================================="
echo "M-Team RSS下载器 - Docker镜像发布"
echo "========================================="
echo ""

# 检查是否配置了用户名
if [ "$USERNAME" = "your-dockerhub-username" ]; then
    echo "错误: 请先设置USERNAME变量为你的Docker Hub用户名"
    echo "编辑此文件，将 USERNAME=\"your-dockerhub-username\" 改为实际的用户名"
    exit 1
fi

FULL_IMAGE_NAME="${USERNAME}/${IMAGE_NAME}:${IMAGE_TAG}"

# 登录Docker Hub
echo "步骤 1: 登录Docker Hub..."
echo "如果已登录可跳过，或按Ctrl+C取消"
docker login

# 构建镜像
echo ""
echo "步骤 2: 构建Docker镜像..."
docker build -t ${FULL_IMAGE_NAME} .

# 标记多个tag（可选）
echo ""
echo "步骤 3: 添加额外标签..."
docker tag ${FULL_IMAGE_NAME} ${USERNAME}/${IMAGE_NAME}:latest

# 推送镜像
echo ""
echo "步骤 4: 推送镜像到Docker Hub..."
docker push ${FULL_IMAGE_NAME}
docker push ${USERNAME}/${IMAGE_NAME}:latest

# 显示发布信息
echo ""
echo "========================================="
echo "发布完成！"
echo ""
echo "镜像地址:"
echo "  ${FULL_IMAGE_NAME}"
echo "  ${USERNAME}/${IMAGE_NAME}:latest"
echo ""
echo "Docker Hub页面:"
echo "  https://hub.docker.com/r/${USERNAME}/${IMAGE_NAME}"
echo ""
echo "使用示例:"
echo "  docker pull ${FULL_IMAGE_NAME}"
echo "  docker run -d -p 8080:8080 -v ./config:/app/config -v ./torrents:/app/torrents ${FULL_IMAGE_NAME}"
echo "========================================="
