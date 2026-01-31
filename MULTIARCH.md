# M-Team RSS下载器 - 多架构部署指南

## 问题说明

由于你的设备使用 `linux/arm64` 架构（如树莓派、Apple Silicon等），而当前Docker镜像只支持 `linux/amd64`，因此需要构建多架构镜像。

## 解决方案

### 方案一：使用平台指定（推荐，快速）

修改 `docker-compose.yml` 添加平台指定：

```yaml
services:
  mteam-downloader:
    image: renjianhuashui/mteam-rss:latest
    platform: linux/amd64  # 强制使用amd64镜像
    # ... 其他配置
```

**注意**: 这会使用模拟运行，性能会有所降低。

### 方案二：构建ARM64专用镜像

在ARM64设备上直接构建：

```bash
# 1. 拉取代码
git clone <your-repo>
cd m-team-rss

# 2. 构建ARM64镜像
docker build -t renjianhuashui/mteam-rss:arm64 .

# 3. 推送
docker push renjianhuashui/mteam-rss:arm64

# 4. 修改docker-compose.yml使用arm64镜像
# image: renjianhuashui/mteam-rss:arm64
```

### 方案三：构建多架构镜像（最佳）

使用 `buildx` 构建同时支持 amd64 和 arm64 的镜像：

#### Windows系统

```cmd
build-multiarch.bat 1.0.0
```

#### Linux/macOS系统

```bash
chmod +x build-multiarch.sh
./build-multiarch.sh 1.0.0
```

#### 手动构建

```bash
# 创建并使用buildx构建器
docker buildx create --name multiarch --use
docker buildx inspect --bootstrap

# 构建并推送多架构镜像
docker buildx build \
  --platform linux/amd64,linux/arm64 \
  --tag renjianhuashui/mteam-rss:latest \
  --tag renjianhuashui/mteam-rss:amd64 \
  --tag renjianhuashui/mteam-rss:arm64 \
  --push .
```

## 验证架构支持

查看镜像支持的架构：

```bash
docker manifest inspect renjianhuashui/mteam-rss:latest
```

## 运行ARM64镜像

### 直接运行

```bash
docker run -d --name mteam-rss \
  -p 8080:8080 \
  -v $(pwd)/config:/app/config \
  -v $(pwd)/torrents:/app/torrents \
  -v $(pwd)/data:/app/data \
  -e TZ=Asia/Shanghai \
  renjianhuashui/mteam-rss:arm64
```

### 使用docker-compose

更新后的 `docker-compose.yml`（已移除version）：

```yaml
services:
  mteam-downloader:
    image: renjianhuashui/mteam-rss:arm64
    container_name: mteam-downloader
    restart: unless-stopped
    ports:
      - "8080:8080"
    volumes:
      - ./config:/app/config
      - ./torrents:/app/torrents
      - ./data:/app/data
    environment:
      - TZ=Asia/Shanghai
```

运行：

```bash
docker-compose up -d
```

## 构建要求

- Docker 19.03+
- buildx插件（默认包含）
- 对于ARM64构建，建议在ARM64设备上使用交叉编译
- 推送到Docker Hub需要先登录

## 性能对比

| 架构 | 运行方式 | 性能 |
|------|---------|------|
| amd64 | 原生 | 100% |
| amd64 | ARM64模拟 | ~50-70% |
| arm64 | ARM64原生 | 100% |
| arm64 | x86_64模拟 | ~30-50% |

## 总结

推荐使用**方案三**构建多架构镜像，这样：
- ✓ 支持所有主流平台
- ✓ 无需模拟，性能最优
- ✓ 一次构建，多处使用
- ✓ 自动选择最优架构
