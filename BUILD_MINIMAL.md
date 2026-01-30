# 最小化Docker镜像构建指南

## 说明

由于Docker镜像源无法访问，使用scratch基础镜像构建最小化镜像。

## 前提条件

- 需要手动准备alpine镜像（或从其他来源获取）
- 需要CA证书文件
- 需要预编译的Linux二进制文件

## 准备工作

### 1. 编译Linux二进制文件

```bash
CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o mteam-downloader .
```

### 2. 从Alpine Linux获取CA证书

如果无法下载Alpine镜像，可以从以下方式获取：

#### 方式A：从现有Alpine容器提取

```bash
# 如果有其他机器的Alpine镜像
docker run --rm alpine:latest cat /etc/ssl/certs/ca-certificates.crt > ca-certificates.crt
```

#### 方式B：手动下载证书

从Mozilla证书库下载：
https://curl.se/docs/caextract.html

#### 方式C：使用空镜像

如果只是测试，可以暂时忽略证书（不推荐生产使用）。

### 3. 构建最小化镜像

```bash
docker build -f Dockerfile.minimal -t mteam-rss:minimal .
```

## 使用scratch镜像的问题

scratch是最小的Docker基础镜像，但是：

- ✗ 没有任何包管理器
- ✗ 没有shell（需要手动复制sh）
- ✗ 没有网络工具
- ✗ 需要手动处理所有依赖

因此生产环境仍建议使用alpine镜像。

## 推荐方案：等待网络恢复

当前最佳方案是：

1. **等待网络恢复**，然后使用原始Dockerfile构建
2. **配置Docker镜像源**，使用可访问的镜像站
3. **使用离线缓存**，如果本地有缓存的镜像

### 配置镜像源

Windows Docker Desktop配置镜像源：

1. 打开Docker Desktop
2. 进入Settings -> Docker Engine
3. 添加镜像源配置：

```json
{
  "registry-mirrors": [
    "https://docker.mirrors.ustc.edu.cn"
  ]
}
```

4. 点击"Apply & Restart"

### 常用国内镜像源

```json
{
  "registry-mirrors": [
    "https://docker.mirrors.ustc.edu.cn",
    "https://dockerhub.azk8s.cn",
    "https://dockerproxy.com"
  ]
}
```

## 临时解决方案

如果急需构建Docker镜像，可以考虑：

### 方案一：直接编译运行（不使用Docker）

```bash
# 本地编译Linux版本
CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -o mteam-downloader .

# 直接运行（需要Linux环境）
./mteam-downloader -config config.yaml
```

### 方案二：使用WSL2 + Docker

如果你在Windows上，可以在WSL2中使用Docker：

```bash
# 在WSL2中
sudo service docker start
docker build -t mteam-rss:latest .
```

### 方案三：使用其他容器运行时

如Podman、containerd等：

```bash
podman build -t mteam-rss:latest .
```

## 总结

当前由于Docker Hub镜像源访问受限，建议：

1. ✓ 等待网络恢复或配置镜像源
2. ✓ 重新执行 `docker build -t mteam-rss:latest .`
3. ✓ 或使用本地编译方式直接运行Go程序

所有必要的构建脚本和文档已准备就绪。
