# M-Team RSS下载器

一个基于Go语言开发的多RSS源自动下载器，支持M-Team、PTzone等PT站点，提供Web界面管理，支持Docker部署。

## 功能特性

- **多RSS源支持** - 支持M-Team、PTzone等多个PT站点
- **自动定时抓取** - 可配置的轮询间隔
- **自动下载种子** - 自动下载.torrent文件
- **智能去重** - 基于GUID的智能去重机制
- **Web管理界面** - 实时查看下载状态和管理任务
- **Docker支持** - 完整的容器化部署方案
- **灵活配置** - 支持配置文件、环境变量、命令行参数
- **SQLite数据库** - 轻量级数据存储
- **并发控制** - 可配置的并发下载数
- **错误重试** - 自动重试失败的下载任务

## 快速开始

### Docker部署（推荐）

#### 1. 使用Docker Compose

```bash
# 克隆项目
git clone https://github.com/yourusername/m-team-rss.git
cd m-team-rss

# 启动服务
docker-compose up -d

# 访问Web界面
# http://localhost:8080
```

#### 2. 使用Docker命令

```bash
# 直接运行（使用默认配置）
docker run -d \
  --name mteam-rss \
  -p 8080:8080 \
  -v $(pwd)/config:/app/config \
  -v $(pwd)/torrents:/app/torrents \
  -v $(pwd)/data:/app/data \
  renjianhuashui/mteam-rss:latest

# 使用环境变量配置
docker run -d \
  --name mteam-rss \
  -p 8081:8080 \
  -v $(pwd)/torrents:/app/torrents \
  -v $(pwd)/data:/app/data \
  -e PORT=8080 \
  -e SAVE_PATH=/app/torrents \
  -e DB_PATH=/app/data/downloads.db \
  renjianhuashui/mteam-rss:latest
```

### 本地运行

#### 1. 克隆并构建

```bash
git clone https://github.com/yourusername/mteam-rss.git
cd mteam-rss
go build -o mteam-rss
```

#### 2. 使用配置文件运行

```bash
# 使用默认配置文件 config/config.yaml
./mteam-rss

# 指定配置文件
./mteam-rss -config /path/to/config.yaml
```

#### 3. 使用环境变量运行（不依赖配置文件）

```bash
# Windows PowerShell
$env:DB_PATH="./data/downloads.db"
$env:SAVE_PATH="./torrents"
$env:PORT="8080"
$env:HOST="0.0.0.0"
.\mteam-rss.exe

# Linux/Mac
export DB_PATH="./data/downloads.db"
export SAVE_PATH="./torrents"
export PORT="8080"
export HOST="0.0.0.0"
./mteam-rss
```

#### 4. 使用命令行参数运行

```bash
./mteam-rss \
  -db "./data/downloads.db" \
  -save-path "./torrents" \
  -host "0.0.0.0" \
  -port "8080"
```

## 配置说明

### 配置优先级

**命令行参数 > 环境变量 > 配置文件 > 默认值**

### 配置文件 (config/config.yaml)

```yaml
server:
  host: "0.0.0.0"     # Web服务器监听地址
  port: 8080          # Web服务器端口

database:
  path: "./data/downloads.db"  # 数据库文件路径

download:
  save_path: "./torrents"       # 下载保存路径
  max_concurrent: 3             # 最大并发下载数
  timeout: 60s                  # 下载超时时间
  retry_limit: 3                # 失败重试次数

rss_sources:
  - name: "M-Team Music"
    site_type: "mteam"
    rss_url: "https://rss.m-team.io/api/rss/fetch?..."
    enabled: true
    poll_interval: 15            # 轮询间隔（秒）
    max_items: 50               # 单次抓取最大数量

  - name: "PTzone"
    site_type: "ptzone"
    rss_url: "https://ptzone.xyz/torrentrss.php?..."
    enabled: true
    poll_interval: 15
    max_items: 50

logging:
  level: "info"
  max_size_mb: 100
  max_files: 10
```

### 环境变量

| 环境变量 | 说明 | 默认值 |
|---------|------|--------|
| `CONFIG_PATH` | 配置文件路径 | `config/config.yaml` |
| `DB_PATH` | 数据库路径 | `./data/downloads.db` |
| `SAVE_PATH` | 下载保存路径 | `./torrents` |
| `HOST` | Web服务器监听地址 | `0.0.0.0` |
| `PORT` | Web服务器端口 | `8080` |
| `GIN_MODE` | Gin运行模式 | `release` |

### 命令行参数

```bash
-config string      # 配置文件路径 (默认: config/config.yaml)
-db string         # 数据库路径
-save-path string  # 下载保存路径
-host string       # Web服务器监听地址
-port int         # Web服务器端口
```

## Docker部署配置说明

### 挂载卷说明

| 宿主机路径 | 容器路径 | 说明 |
|----------|---------|------|
| `./config` | `/app/config` | 配置文件目录（可选） |
| `./torrents` | `/app/torrents` | 下载的种子文件 |
| `./data` | `/app/data` | 数据库文件 |

### 不依赖配置文件的Docker部署

```bash
docker run -d \
  --name mteam-rss \
  -p 8080:8080 \
  -v $(pwd)/torrents:/app/torrents \
  -v $(pwd)/data:/app/data \
  -e DB_PATH=/app/data/downloads.db \
  -e SAVE_PATH=/app/torrents \
  -e PORT=8080 \
  renjianhuashui/mteam-rss:latest
```

然后通过Web界面添加RSS源，无需配置文件。

## Web界面使用

访问 `http://localhost:8080` 后可以：

1. **仪表盘** - 查看系统统计和实时日志
2. **RSS源管理** - 添加、编辑、删除RSS源
3. **任务管理** - 查看下载任务，重试失败任务，清理已完成任务
4. **手动触发** - 手动触发RSS源抓取

## 支持的PT站点

- M-Team (mteam) - 支持sign认证
- PTzone (ptzone) - 支持passkey认证
- 可扩展支持更多站点

## 技术栈

- Go 1.25+
- Gin (Web框架)
- SQLite (数据库)
- gofeed (RSS解析)

# 编辑 config.yaml，填入你的RSS地址
```

配置示例：

```yaml
rss_url: "https://rss.m-team.io/your-rss-feed-url&dl=1"
poll_interval: "5m"
save_path: "./torrents"
max_concurrent: 3
db_path: "./data/downloads.db"
web_port: 8080
web_host: "0.0.0.0"
```

4. 运行

```bash
go run main.go -config config.yaml
```

5. 访问Web界面

打开浏览器访问: http://localhost:8080

### Docker部署

1. 构建镜像

```bash
docker build -t mteam-downloader .
```

2. 创建必要目录

```bash
mkdir -p config torrents data
```

3. 创建配置文件

```bash
cat > config/config.yaml << EOF
rss_url: "https://rss.m-team.io/your-rss-feed-url&dl=1"
poll_interval: "5m"
save_path: "/app/torrents"
max_concurrent: 3
db_path: "/app/data/downloads.db"
web_port: 8080
web_host: "0.0.0.0"
EOF
```

4. 使用docker-compose运行

```bash
docker-compose up -d
```

5. 查看日志

```bash
docker-compose logs -f
```

6. 访问Web界面

打开浏览器访问: http://localhost:8080

## Web界面功能

### 主页面信息

- 系统运行状态（运行中/已停止）
- RSS订阅地址
- 最后检查时间
- 下次检查时间
- 统计信息（总数/已下载/待处理）

### API接口

- `GET /` - Web界面主页
- `GET /api/status` - 获取系统状态
- `GET /api/rss-items` - 获取RSS项目列表
- `GET /api/config` - 获取配置信息
- `POST /api/trigger` - 手动触发立即下载

### 状态API示例

```json
{
  "rss_url": "https://rss.m-team.io/...",
  "status": "running",
  "last_fetch": "2024-01-15T10:30:00Z",
  "next_fetch": "2024-01-15T10:35:00Z",
  "total_items": 25,
  "downloaded": 20,
  "pending": 5
}
```

## 配置说明

| 配置项 | 说明 | 默认值 |
|--------|------|--------|
| rss_url | M-Team RSS订阅地址（需添加&dl=1） | 必填 |
| poll_interval | RSS轮询间隔（支持s/m/h） | 5m |
| save_path | .torrent文件保存路径 | ./torrents |
| max_concurrent | 最大并发下载数 | 3 |
| db_path | SQLite数据库路径 | ./data/downloads.db |
| web_port | Web服务器端口 | 8080 |
| web_host | Web绑定地址 | 0.0.0.0 |

## 目录结构

```
m-team-rss/
├── main.go                      # 主程序入口
├── go.mod                       # Go依赖管理
├── go.sum                       # Go依赖锁定
├── config.yaml                  # 配置文件
├── Dockerfile                   # Docker镜像构建
├── docker-compose.yml           # Docker Compose配置
├── internal/
│   ├── config/                  # 配置模块
│   │   └── config.go
│   ├── database/                # 数据库模块
│   │   └── database.go
│   ├── rss/                     # RSS客户端
│   │   └── client.go
│   ├── downloader/              # 下载器模块
│   │   └── downloader.go
│   ├── scheduler/               # 定时任务调度器
│   │   └── scheduler.go
│   └── web/                     # Web界面模块
│       ├── handler.go
│       └── templates/
│           └── index.html
├── data/                        # 数据目录（数据库）
├── torrents/                    # 下载文件目录
└── config/                      # 配置目录
```

## 数据库结构

### downloads表 - 下载记录

```sql
CREATE TABLE downloads (
    guid TEXT PRIMARY KEY,           -- RSS项目唯一标识
    title TEXT NOT NULL,            -- 标题
    url TEXT NOT NULL,              -- 下载链接
    file_path TEXT,                 -- 本地文件路径
    downloaded_at DATETIME,         -- 下载时间
    pub_date DATETIME,             -- 发布时间
    created_at DATETIME
);
```

### system_status表 - 系统状态

```sql
CREATE TABLE system_status (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    last_fetch DATETIME,           -- 最后抓取时间
    next_fetch DATETIME,           -- 下次抓取时间
    last_error TEXT,               -- 最后错误信息
    updated_at DATETIME
);
```

### rss_cache表 - RSS缓存

```sql
CREATE TABLE rss_cache (
    guid TEXT PRIMARY KEY,          -- RSS项目唯一标识
    title TEXT NOT NULL,           -- 标题
    url TEXT NOT NULL,             -- 下载链接
    pub_date DATETIME,            -- 发布时间
    category TEXT,                -- 分类
    size INTEGER,                 -- 文件大小
    json_data TEXT,               -- 完整JSON数据
    cached_at DATETIME            -- 缓存时间
);
```

## 注意事项

1. **RSS地址配置**: 确保RSS地址包含`&dl=1`参数以获取直接下载链接
2. **并发控制**: 根据网络带宽和服务器性能调整`max_concurrent`
3. **数据备份**: 定期备份`data/`目录下的数据库文件
4. **日志监控**: 建议配置日志收集和监控告警
5. **安全性**: 生产环境建议添加Web认证或使用反向代理

## 故障排除

### 无法连接RSS

- 检查网络连接
- 验证RSS地址是否正确
- 确认是否需要认证

### 下载失败

- 检查`save_path`目录权限
- 查看日志中的错误信息
- 确认下载链接是否有效

### Web界面无法访问

- 检查防火墙设置
- 确认`web_port`未被占用
- 查看容器日志: `docker-compose logs`

### 数据库错误

- 确保`data/`目录存在且有写入权限
- 检查磁盘空间是否充足
- 尝试删除数据库文件重新初始化

## 开发

### 运行测试

```bash
go test ./...
```

### 构建二进制文件

```bash
# Linux
CGO_ENABLED=1 GOOS=linux go build -o mteam-downloader .

# Windows
CGO_ENABLED=1 GOOS=windows go build -o mteam-downloader.exe .

# macOS
CGO_ENABLED=1 GOOS=darwin go build -o mteam-downloader .
```

## 许可证

MIT License

## 贡献

欢迎提交Issue和Pull Request！

## 联系方式

- Issues: https://github.com/yourusername/m-team-rss/issues
- Discussions: https://github.com/yourusername/m-team-rss/discussions

## 更新日志

### v1.0.0 (2024-01-15)

- 初始版本发布
- 支持RSS自动下载
- Web界面展示
- Docker部署支持
