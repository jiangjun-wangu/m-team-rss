#!/bin/bash

# M-Team RSS下载器 - Docker镜像构建脚本

set -e

# 配置变量
IMAGE_NAME="mteam-rss"
IMAGE_TAG=${1:-latest}
REGISTRY="docker.io"  # 可改为其他registry如 registry.cn-hangzhou.aliyuncs.com
USERNAME=""  # Docker Hub用户名，填写后自动添加前缀

echo "========================================="
echo "M-Team RSS下载器 - Docker镜像构建"
echo "========================================="
echo ""

# 清理旧的构建缓存
echo "步骤 1: 清理旧的构建文件..."
rm -rf mteam-downloader

# 构建Docker镜像
echo ""
echo "步骤 2: 构建Docker镜像..."
if [ -z "$USERNAME" ]; then
    FULL_IMAGE_NAME="${IMAGE_NAME}:${IMAGE_TAG}"
else
    FULL_IMAGE_NAME="${USERNAME}/${IMAGE_NAME}:${IMAGE_TAG}"
fi

echo "构建镜像: ${FULL_IMAGE_NAME}"
docker build -t ${FULL_IMAGE_NAME} .

# 显示镜像信息
echo ""
echo "步骤 3: 镜像构建完成！"
docker images | grep ${IMAGE_NAME}

echo ""
echo "========================================="
echo "构建完成！"
echo "镜像名称: ${FULL_IMAGE_NAME}"
echo ""
echo "使用以下命令推送镜像:"
if [ -z "$USERNAME" ]; then
    echo "  docker push ${FULL_IMAGE_NAME}"
else
    echo "  docker push ${FULL_IMAGE_NAME}"
fi
echo ""
echo "使用以下命令运行:"
echo "  docker run -d -p 8080:8080 -v \$(pwd)/config:/app/config -v \$(pwd)/torrents:/app/torrents -v \$(pwd)/data:/app/data ${FULL_IMAGE_NAME}"
echo "========================================="
