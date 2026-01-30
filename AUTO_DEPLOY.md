# Docker镜像自动发布指南

## 方式一：使用Token登录（推荐）

### 1. 创建Docker Hub访问令牌

1. 访问 https://hub.docker.com/settings/security
2. 点击 "New Access Token"
3. 填写描述，选择 "Read & Write" 权限
4. 点击 "Generate" 复制生成的Token

### 2. 使用Token登录

#### Linux/macOS

```bash
docker login -u your-username --password-stdin < <(echo "your-docker-access-token")
```

或使用脚本：

```bash
echo "your-docker-access-token" | docker login -u your-username --password-stdin
```

#### Windows (PowerShell)

```powershell
"your-docker-access-token" | docker login -u your-username --password-stdin
```

#### Windows (CMD)

```cmd
docker login -u your-username --password-stdin < token.txt
```

其中 `token.txt` 文件内容为：
```
your-docker-access-token
```

## 方式二：使用CI/CD自动发布

### GitHub Actions

创建 `.github/workflows/docker-publish.yml`:

```yaml
name: Build and Push Docker Image

on:
  push:
    tags:
      - 'v*'
  workflow_dispatch:

jobs:
  build-and-push:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout
        uses: actions/checkout@v3

      - name: Login to Docker Hub
        uses: docker/login-action@v2
        with:
          username: ${{ secrets.DOCKER_USERNAME }}
          password: ${{ secrets.DOCKER_PASSWORD }}

      - name: Build and push
        uses: docker/build-push-action@v4
        with:
          context: .
          push: true
          tags: |
            username/mteam-rss:latest
            username/mteam-rss:${{ github.ref_name }}
```

### 配置GitHub Secrets

1. 访问 GitHub仓库的 Settings -> Secrets and variables -> Actions
2. 添加以下secrets：
   - `DOCKER_USERNAME`: Docker Hub用户名
   - `DOCKER_PASSWORD`: Docker Hub访问令牌

3. 推送标签触发自动构建：

```bash
git tag v1.0.0
git push origin v1.0.0
```

## 手动发布完整流程

### 1. 创建访问令牌

访问 https://hub.docker.com/settings/security 创建Access Token

### 2. 登录Docker Hub

```bash
# 方式一：使用Token（推荐）
echo "your-token" | docker login -u your-username --password-stdin

# 方式二：交互式登录（需要TTY）
docker login
```

### 3. 构建镜像

```bash
docker build -t your-username/mteam-rss:1.0.0 .
```

### 4. 推送镜像

```bash
docker push your-username/mteam-rss:1.0.0
docker tag your-username/mteam-rss:1.0.0 your-username/mteam-rss:latest
docker push your-username/mteam-rss:latest
```

## 国内镜像加速

如果推送到Docker Hub很慢，可以使用国内镜像源：

### 阿里云容器镜像服务

```bash
# 登录阿里云
docker login registry.cn-hangzhou.aliyuncs.com -u your-username --password-stdin

# 构建镜像
docker build -t registry.cn-hangzhou.aliyuncs.com/your-username/mteam-rss:1.0.0 .

# 推送镜像
docker push registry.cn-hangzhou.aliyuncs.com/your-username/mteam-rss:1.0.0
```

### 腾讯云容器镜像服务

```bash
# 登录腾讯云
docker login ccr.ccs.tencentyun.com -u your-username --password-stdin

# 构建镜像
docker build -t ccr.ccs.tencentyun.com/your-username/mteam-rss:1.0.0 .

# 推送镜像
docker push ccr.ccs.tencentyun.com/your-username/mteam-rss:1.0.0
```

## 使用已发布的镜像

### 拉取镜像

```bash
# 从Docker Hub
docker pull your-username/mteam-rss:latest

# 从阿里云
docker pull registry.cn-hangzhou.aliyuncs.com/your-username/mteam-rss:latest

# 从腾讯云
docker pull ccr.ccs.tencentyun.com/your-username/mteam-rss:latest
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
  your-username/mteam-rss:latest
```

## 故障排除

### 登录失败

1. 检查网络连接
2. 确认用户名和密码/Token正确
3. 尝试使用Token而非密码
4. 清理Docker缓存: `docker system prune`

### 推送失败

1. 确认已登录成功
2. 检查镜像名称格式
3. 确认Docker Hub账号正常
4. 查看详细错误日志

### 网络超时

1. 使用国内镜像加速
2. 重试多次推送
3. 检查防火墙设置
4. 使用代理或VPN

## 安全建议

1. **使用Access Token**而非密码
2. **定期轮换Token**
3. **不要在代码中硬编码凭证**
4. **使用GitHub Actions等CI/CD**避免手动操作
5. **限制Token权限**为最小必要范围

## 完整示例

### 使用Token发布到Docker Hub

```bash
# 1. 创建Token（在Docker Hub网站）
# 2. 使用Token登录
echo "dckr_xxxxxxxxxxxxxxxxxxxxxxxxxxxx" | docker login -u your-username --password-stdin

# 3. 构建镜像
docker build -t your-username/mteam-rss:1.0.0 .

# 4. 推送镜像
docker push your-username/mteam-rss:1.0.0

# 5. 添加latest标签
docker tag your-username/mteam-rss:1.0.0 your-username/mteam-rss:latest
docker push your-username/mteam-rss:latest

# 6. 在Docker Hub查看
echo "访问 https://hub.docker.com/r/your-username/mteam-rss"
```
