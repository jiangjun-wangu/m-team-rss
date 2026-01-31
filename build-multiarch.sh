#!/bin/bash
# 多架构Docker镜像构建脚本

set -e

IMAGE_NAME="renjianhuashui/mteam-rss"
VERSION=${1:-latest}

echo "=== 构建多架构Docker镜像 ==="
echo "镜像名称: $IMAGE_NAME"
echo "版本: $VERSION"
echo ""

# 构建多架构镜像
echo "正在构建支持 amd64 和 arm64 的镜像..."
docker buildx build \
  --platform linux/amd64,linux/arm64 \
  --tag "$IMAGE_NAME:$VERSION" \
  --tag "$IMAGE_NAME:amd64" \
  --tag "$IMAGE_NAME:arm64" \
  --push \
  .

echo ""
echo "=== 构建完成 ==="
echo "已构建的架构:"
echo "  - linux/amd64"
echo "  - linux/arm64"
echo ""
echo "推送的标签:"
echo "  - $IMAGE_NAME:$VERSION"
echo "  - $IMAGE_NAME:amd64"
echo "  - $IMAGE_NAME:arm64"
