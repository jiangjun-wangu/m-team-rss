# Docker镜像发布指南

## 前提条件

1. 已安装Docker Desktop
2. 已有Docker Hub账号: https://hub.docker.com/
3. 已登录Docker Hub

## 快速开始

### Windows系统

#### 1. 配置用户名

编辑 `publish.bat` 文件，修改第6行：

```bat
set USERNAME=your-dockerhub-username
```

改为你的Docker Hub用户名：

```bat
set USERNAME=myusername
```

#### 2. 登录Docker Hub

```bash
docker login
```

#### 3. 构建并发布镜像

```bash
# 发布latest版本
publish.bat

# 发布指定版本
publish.bat 1.0.0
```

### Linux/macOS系统

#### 1. 配置用户名

编辑 `publish.sh` 文件，修改第6行：

```bash
USERNAME="your-dockerhub-username"
```

改为你的Docker Hub用户名：

```bash
USERNAME="myusername"
```

#### 2. 添加执行权限

```bash
chmod +x publish.sh
```

#### 3. 登录Docker Hub

```bash
docker login
```

#### 4. 构建并发布镜像

```bash
# 发布latest版本
./publish.sh

# 发布指定版本
./publish.sh 1.0.0
```

## 手动构建和发布

### 构建镜像

```bash
# 基础构建
docker build -t mteam-rss:latest .

# 指定版本
docker build -t mteam-rss:1.0.0 .

# 添加用户名前缀
docker build -t username/mteam-rss:latest .
```

### 推送镜像

```bash
# 先登录
docker login

# 推送到Docker Hub
docker push username/mteam-rss:latest
docker push username/mteam-rss:1.0.0
```

## 国内镜像加速

如果Docker Hub访问缓慢，可以使用国内镜像源：

### 阿里云容器镜像服务

```bash
# 登录阿里云容器镜像服务
docker login registry.cn-hangzhou.aliyuncs.com

# 重新标记镜像
docker tag mteam-rss:latest registry.cn-hangzhou.aliyuncs.com/username/mteam-rss:latest

# 推送到阿里云
docker push registry.cn-hangzhou.aliyuncs.com/username/mteam-rss:latest
```

### 腾讯云容器镜像服务

```bash
# 登录腾讯云
docker login ccr.ccs.tencentyun.com

# 重新标记镜像
docker tag mteam-rss:latest ccr.ccs.tencentyun.com/username/mteam-rss:latest

# 推送到腾讯云
docker push ccr.ccs.tencentyun.com/username/mteam-rss:latest
```

## 使用已发布的镜像

### 拉取镜像

```bash
# 从Docker Hub
docker pull username/mteam-rss:latest

# 从阿里云
docker pull registry.cn-hangzhou.aliyuncs.com/username/mteam-rss:latest

# 从腾讯云
docker pull ccr.ccs.tencentyun.com/username/mteam-rss:latest
```

### 运行容器

```bash
docker run -d \
  --name mteam-rss \
  -p 8080:8080 \
  -v $(pwd)/config:/app/config \
  -v $(pwd)/torrents:/app/torrents \
  -v $(pwd)/data:/app/data \
  -e TZ=Asia/Shanghai \
  username/mteam-rss:latest
```

## Docker Compose部署

修改 `docker-compose.yml` 中的镜像地址：

```yaml
version: '3.8'

services:
  mteam-downloader:
    image: username/mteam-rss:latest  # 修改这里
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
    healthcheck:
      test: ["CMD", "wget", "--quiet", "--tries=1", "--spider", "http://localhost:8080/api/status"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 40s
```

运行：

```bash
docker-compose up -d
```

## 镜像优化建议

1. **减小镜像大小**: 使用多阶段构建（已实现）
2. **选择合适的基础镜像**: alpine比debian更小
3. **清理缓存**: 删除不必要的文件和缓存
4. **使用.dockerignore**: 排除不需要的文件

## 安全建议

1. **不要在镜像中包含敏感信息**（如配置文件）
2. **使用环境变量**传递配置
3. **定期更新基础镜像**
4. **扫描镜像漏洞**: 使用 `docker scan`

```bash
docker scan username/mteam-rss:latest
```

## 故障排除

### 构建失败

- 检查Dockerfile语法
- 确保网络连接正常
- 清理Docker缓存: `docker builder prune`

### 推送失败

- 确认已登录Docker Hub
- 检查镜像名称格式
- 确认Docker Hub账号正常

### 运行失败

- 检查端口是否被占用
- 确认配置文件存在
- 查看容器日志: `docker logs mteam-rss`

## 版本管理建议

推荐使用语义化版本号：

- `1.0.0` - 主版本号.次版本号.修订号
- `1.0.1` - Bug修复
- `1.1.0` - 新功能
- `2.0.0` - 重大变更

发布流程：

```bash
# 发布v1.0.0
publish.sh 1.0.0

# 发布v1.0.1（修复）
publish.sh 1.0.1

# 发布v1.1.0（新功能）
publish.sh 1.1.0

# 更新latest标签
publish.sh latest
```
