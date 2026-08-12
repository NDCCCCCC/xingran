# RPA Worker 浏览器缓存配置

## 概述

RPA Worker 使用 `go-rod/rod` 库进行浏览器自动化，它会自动下载 Chromium 浏览器。为了避免每次启动容器都重新下载（约 300MB），我们配置了持久化卷来缓存浏览器文件。

## 持久化卷配置

### Docker Compose 配置

`docker-compose.local.yml` 中配置了以下持久化卷：

```yaml
volumes:
  # Rod 浏览器缓存目录
  - chrome-cache:/home/chrome/.cache

  # 浏览器下载目录
  - chrome-downloads:/home/chrome/Downloads

  # 截图保存目录
  - ./screenshots:/app/screenshots
```

### 环境变量

```yaml
environment:
  # Rod 浏览器缓存根目录
  - XDG_CACHE_HOME=/home/chrome/.cache

  # HOME 目录
  - HOME=/home/chrome
```

### Rod 浏览器下载路径

Chromium 浏览器会被下载到：
```
/home/chrome/.cache/rod/chromium/
```

这个目录通过持久化卷 `chrome-cache` 映射到 Docker 卷，即使容器重启或重新创建，浏览器文件也会保留。

## 首次部署

### 方法 1：使用脚本（推荐）

**Windows:**
```cmd
cd D:\code\ClaudeCode\xingran-go-backend\rpa-worker\deployments
rebuild-local.bat
```

**Linux:**
```bash
cd /path/to/rpa-worker/deployments
chmod +x rebuild-local.sh
./rebuild-local.sh
```

### 方法 2：手动步骤

```bash
# 1. 构建 Go 二进制
cd rpa-worker
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o rpa-worker ./cmd/main.go

# 2. 构建 Docker 镜像
docker build -f deployments/Dockerfile.local -t rpa-worker:latest .

# 3. 启动容器
cd deployments
docker-compose -f docker-compose.local.yml up -d
```

## 首次运行

首次启动时，Rod 会自动下载 Chromium 浏览器（约 300MB）。下载进度会显示在日志中：

```
[launcher.Browser]2026/03/10 05:10:17 Download: https://storage.googleapis.com/...
[launcher.Browser]2026/03/10 05:11:15 Progress: 01%
[launcher.Browser]2026/03/10 05:12:30 Progress: 50%
...
```

**下载时间取决于网络速度**，可能需要 5-30 分钟。

下载完成后，浏览器文件会被缓存到持久化卷中，后续启动不需要重新下载。

## 查看浏览器缓存

```bash
# 查看 Docker 卷
docker volume ls | grep chrome

# 查看卷详细信息
docker volume inspect rpa-worker_chrome-cache

# 查看卷大小
du -sh $(docker volume inspect rpa-worker_chrome-cache -f '{{.Mountpoint}}')
```

## 清理浏览器缓存（如需重新下载）

```bash
# 停止并删除容器
docker-compose -f docker-compose.local.yml down

# 删除持久化卷
docker volume rm rpa-worker_chrome-cache rpa-worker_chrome-downloads

# 重新启动
docker-compose -f docker-compose.local.yml up -d
```

## 验证缓存是否生效

首次运行后，停止并重新启动容器：

```bash
docker-compose -f docker-compose.local.yml restart
```

查看日志，如果**不再看到** `[launcher.Browser] Download:` 消息，说明浏览器缓存已生效。

## 网络加速（可选）

如果从 Google Storage 下载速度较慢，可以考虑：

### 方案 1：使用国内镜像

设置环境变量使用镜像源（需要修改代码支持）。

### 方案 2：预下载浏览器

在宿主机下载 Chromium，然后复制到 Docker 卷中：

```bash
# 1. 下载 Chromium
wget https://storage.googleapis.com/chromium-browser-snapshots/Linux_x64/1321438/chrome-linux.zip

# 2. 创建临时容器并解压
docker run --rm -v rpa-worker_chrome-cache:/cache alpine sh -c \
  "mkdir -p /cache/rod/chromium/1321438 && \
   cd /cache/rod/chromium/1321438 && \
   unzip /chrome-linux.zip"
```

### 方案 3：使用代理

在 docker-compose.yml 中添加代理配置：

```yaml
services:
  rpa-worker:
    environment:
      - HTTP_PROXY=http://proxy.example.com:8080
      - HTTPS_PROXY=http://proxy.example.com:8080
```

## 故障排查

### 问题：每次启动都重新下载浏览器

**原因**：持久化卷没有正确挂载

**检查**：
```bash
docker inspect rpa-worker | grep -A 10 Mounts
```

应该看到 `/home/chrome/.cache` 挂载到 `rpa-worker_chrome-cache` 卷。

### 问题：下载卡住不动

**原因**：网络连接问题或下载速度慢

**解决**：
1. 检查网络连接
2. 使用代理或镜像源
3. 预下载浏览器文件

### 问题：权限错误

**原因**：容器内用户权限问题

**解决**：Dockerfile.local 已设置 `chmod -R 777 /home/chrome`

## 文件结构

```
rpa-worker/
├── deployments/
│   ├── docker-compose.local.yml    # 本地开发配置
│   ├── Dockerfile.local             # 本地构建 Dockerfile
│   ├── rebuild-local.sh             # Linux 构建脚本
│   └── rebuild-local.bat            # Windows 构建脚本
├── configs/
│   └── config.yaml                  # Worker 配置
└── cmd/
    └── main.go                      # 入口文件
```

## 相关文档

- [Rod 官方文档](https://github.com/go-rod/rod)
- [Docker Compose 卷文档](https://docs.docker.com/compose/compose-file/compose-file-v3/#volumes)
- [RPA Worker README](../README.md)
