# 支持多架构构建
FROM golang:1.25.6-alpine AS builder

WORKDIR /app

# 安装必要的构建工具
RUN apk add --no-cache git gcc musl-dev

# 设置 Go 模块代理和 CGO 编译器
ENV GOPROXY=https://goproxy.cn,direct

# 复制依赖文件
COPY go.mod go.sum ./
RUN go mod download

# 复制源代码
COPY . .

# 编译程序（启用CGO以支持SQLite）
# 使用 TARGETPLATFORM 自动适配目标架构
RUN CGO_ENABLED=1 GOOS=linux GOARCH=$(go env GOARCH) go build -ldflags="-w -s" -o mteam-rss .

# 最终镜像 - 使用多架构基础镜像
FROM alpine:latest

# 安装运行时依赖
RUN apk --no-cache add ca-certificates tzdata sqlite

# 设置时区
ENV TZ=Asia/Shanghai

WORKDIR /app

# 从构建阶段复制可执行文件和模板文件
COPY --from=builder /app/mteam-rss .
COPY --from=builder /app/internal/web/templates ./web/templates
COPY --from=builder /app/config ./config

# 创建必要的目录
RUN mkdir -p /app/torrents /app/data

# 设置卷
VOLUME ["/app/config", "/app/torrents", "/app/data"]

# 暴露端口（可通过环境变量覆盖）
EXPOSE 8080

# 健康检查
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:8080/api/dashboard/stats || exit 1

# 默认启动参数（可通过环境变量或命令行参数覆盖）
# 支持的环境变量：
#   CONFIG_PATH: 配置文件路径 (默认: /app/config/config.yaml)
#   DB_PATH: 数据库路径 (默认: /app/data/downloads.db)
#   SAVE_PATH: 下载保存路径 (默认: /app/torrents)
#   HOST: Web服务器监听地址 (默认: 0.0.0.0)
#   PORT: Web服务器端口 (默认: 8080)
#   GIN_MODE: Gin运行模式 (release/debug)
CMD ["./mteam-rss"]

