# 构建阶段
FROM golang:1.21-alpine AS builder

# 安装构建依赖
RUN apk add --no-cache git gcc musl-dev ca-certificates openssh

# 配置Go代理（使用多个备用源）
ENV GOPROXY=https://goproxy.cn,https://goproxy.io,https://goproxy.net,direct
ENV GO111MODULE=on
ENV GOSUMDB=off

# 配置 Git 跳过 SSL 验证（某些网络环境需要）
RUN git config --global http.sslVerify false && \
    git config --global http.postBuffer 524288000

WORKDIR /build

# 复制 go.mod 和 go.sum
COPY go.mod go.sum* ./

# 下载依赖
RUN go mod download

# 复制源代码
COPY . .

# 构建应用
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o rpa-worker ./cmd/main.go

# 运行阶段
FROM alpine:latest

# 安装Chrome浏览器和运行依赖
# 注意：chromium 包已包含大部分依赖，只需额外安装必要的库
RUN apk --no-cache add \
    ca-certificates \
    chromium \
    chromium-chromedriver \
    \
    # Chrome运行需要的额外依赖库
    cups-libs \
    libexif \
    libxkbcommon \
    pango \
    harfbuzz \
    icu-libs \
    ttf-freefont \
    && \
    # 创建Chrome缓存目录
    mkdir -p /home/chrome/.cache && \
    mkdir -p /home/chrome/Downloads && \
    chmod -R 777 /home/chrome

WORKDIR /app

# 从构建阶段复制可执行文件
COPY --from=builder /build/rpa-worker .
COPY --from=builder /build/configs ./configs

# Chrome环境配置
ENV CHROMIUM_PATH=/usr/bin/chromium-browser
ENV CHROMIUM_FLAGS="--no-sandbox --headless=new --disable-gpu --disable-dev-shm-usage --disable-software-rasterizer --disable-extensions"
ENV CHROME_BIN=/usr/bin/chromium-browser
ENV HOME=/home/chrome

# 健康检查端点
EXPOSE 8080

# 健康检查
HEALTHCHECK --interval=30s --timeout=10s --start-period=10s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# 运行应用
CMD ["./rpa-worker", "-config", "configs/config.yaml"]
