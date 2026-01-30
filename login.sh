#!/bin/bash
# Docker登录脚本（Linux/macOS）

echo "正在登录Docker..."
docker login $1 -u $2 -p $3

echo "登录完成！"
