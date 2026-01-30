FROM golang:1.25.6-alpine AS builder

WORKDIR /app

# 安装必要的构建工具
RUN apk add --no-cache git gcc musl-dev
# 设置 Go 模块代理和 CGO 编译器
ENV GOPROXY=https://goproxy.cn,direct
# 可选项：在严格网络环境下关闭校验
# ENV GOSUMDB=off
# 复制依赖文件
COPY go.mod go.sum ./
RUN go mod download

# 复制源代码
COPY . .

# 编译程序（启用CGO以支持SQLite）
RUN CGO_ENABLED=1 GOOS=linux go build -ldflags="-w -s" -o mteam-downloader .

# 最终镜像
FROM alpine:latest

# 安装运行时依赖
RUN apk --no-cache add ca-certificates tzdata sqlite

# 设置时区
ENV TZ=Asia/Shanghai

WORKDIR /app

# 从构建阶段复制可执行文件
COPY --from=builder /app/mteam-downloader .

# 复制模板文件
COPY internal/web/templates ./web/templates

# 创建必要的目录
RUN mkdir -p /app/config /app/torrents /app/data

# 设置卷
VOLUME ["/app/config", "/app/torrents", "/app/data"]

# 暴露端口
EXPOSE 8080

# 设置入口点
CMD ["./mteam-downloader", "-config", "/app/config/config.yaml"]
