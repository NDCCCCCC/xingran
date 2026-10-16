# Docker Compose 一键编排指南

> 适用场景：用 Docker Compose 在单机（或开发机）一键拉起 XingRan-Next 全栈。
> 适用规模：开发 / 测试 / 试点 / 小型团队（< 100 用户）。
> 生产规模（> 1000 用户）请走 [deployment.md](deployment.md) 的 systemd 方案或 Kubernetes 编排。

---

## 1. 与其它部署方式的边界

| 部署方式 | 文档 | 何时用 |
|---|---|---|
| **本文（Docker Compose）** | 单机一键拉起 | 开发 / 测试 / 试点 / 中小团队；不想运维 PG/Redis |
| [single-machine-deployment.md](single-machine-deployment.md) | 本机进程 + 本机 PG/Redis | 性能最佳，需要调 PG/Redis 参数 |
| [deployment.md](deployment.md) | systemd + 内网 PG/Redis | 生产服务器多机部署 |
| Kubernetes / Helm | 📋 待评估（见 [secret-management.md §6](../deployment/secret-management.md#6-长期改进方向p2-路线)） | 集群化、灰度发布 |

仓库根目录当前未提供 `docker-compose.yml`（README §部署明确说明），本文给出可直接使用的模板。

---

## 2. 架构总览

```
┌──────────────┐     ┌──────────────┐
│   frontend   │     │   backend    │
│  (nginx +    │────▶│  (Go single  │
│  static SPA) │     │   binary)    │
└──────────────┘     └──────┬───────┘
                            │
              ┌─────────────┼─────────────┐
              ▼             ▼             ▼
        ┌──────────┐  ┌──────────┐  ┌──────────┐
        │ postgres │  │  redis   │  │ rpa-     │
        │   18     │  │   7.4    │  │ worker   │
        └──────────┘  └──────────┘  │ (可选)   │
                                    └──────────┘
```

- **backend**：单二进制（含 embed 前端 + TextFSM 模板），监听 9000
- **frontend**：nginx 托管 `dist/`，反代 `/api/` 到 backend
- **postgres**：18 官方镜像
- **redis**：7.4 alpine
- **rpa-worker**：可选，独立服务（与后端通过 HTTP/RPC 通信）

---

## 3. 目录准备

```bash
mkdir -p xingran-deploy/{configs,secrets,nginx,postgres-data,redis-data,uploads,logs}
cd xingran-deploy
```

最终结构：

```
xingran-deploy/
├── docker-compose.yml          # 编排文件
├── .env                         # 非敏感变量（gitignore）
├── secrets.env                  # 敏感变量（gitignore + chmod 600）
├── configs/
│   └── config.yaml              # 后端配置
├── nginx/
│   ├── nginx.conf
│   └── default.conf
├── backend/
│   └── Dockerfile               # 后端镜像构建
├── frontend/
│   └── dist/                    # 前端构建产物（构建后拷贝）
├── postgres-data/               # PG 数据（卷）
├── redis-data/                  # Redis 数据（卷）
├── uploads/                     # 上传文件卷
└── logs/                        # 日志卷
```

---

## 4. docker-compose.yml

```yaml
# xingran-deploy/docker-compose.yml
version: "3.9"

services:
  postgres:
    image: postgres:18-alpine
    container_name: xingran-postgres
    restart: unless-stopped
    environment:
      POSTGRES_USER: ${DB_USER}
      POSTGRES_PASSWORD: ${DB_PASSWORD}
      POSTGRES_DB: ${DB_NAME}
    volumes:
      - ./postgres-data:/var/lib/postgresql/data
      - ./initdb.d:/docker-entrypoint-initdb.d:ro   # 可选：首次启动 SQL
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U ${DB_USER} -d ${DB_NAME}"]
      interval: 10s
      timeout: 5s
      retries: 5
    networks:
      - xingran-net
    # 不暴露端口（仅容器内访问）；如需外部访问 uncomment:
    # ports:
    #   - "127.0.0.1:5432:5432"

  redis:
    image: redis:7.4-alpine
    container_name: xingran-redis
    restart: unless-stopped
    command:
      - redis-server
      - --requirepass
      - ${REDIS_PASSWORD}
      - --appendonly
      - "yes"               # 持久化（AOF）
      - --maxmemory
      - "1gb"               # 按容量规划调整
      - --maxmemory-policy
      - "allkeys-lru"
    volumes:
      - ./redis-data:/data
    healthcheck:
      test: ["CMD", "redis-cli", "-a", "${REDIS_PASSWORD}", "ping"]
      interval: 10s
      timeout: 5s
      retries: 5
    networks:
      - xingran-net
    # ports:
    #   - "127.0.0.1:6379:6379"

  backend:
    build:
      context: ../                      # 仓库根（含 cmd/, internal/, pkg/, configs/）
      dockerfile: backend/Dockerfile
    image: xingran-backend:latest
    container_name: xingran-backend
    restart: unless-stopped
    depends_on:
      postgres:
        condition: service_healthy
      redis:
        condition: service_healthy
    env_file:
      - secrets.env                     # 敏感 env（600 权限）
    environment:
      # 与敏感变量对齐（DB/Redis host 用容器名）
      DB_HOST: postgres
      REDIS_HOST: redis
      SERVER_HOST: 0.0.0.0
      SERVER_PORT: "9000"
      SERVER_MODE: release
      SKIP_AUTOMIGRATE: "false"         # 启动时自动迁移
    volumes:
      - ./uploads:/app/uploads          # 上传文件持久化
      - ./logs:/app/logs                # 日志持久化
      - ./configs/config.yaml:/app/configs/config.yaml:ro
    healthcheck:
      test: ["CMD", "wget", "-q", "-O", "-", "http://localhost:9000/health"]
      interval: 30s
      timeout: 5s
      retries: 3
      start_period: 30s                 # 启动期给足时间做迁移
    networks:
      - xingran-net
    # ports:
    #   - "127.0.0.1:9000:9000"          # 仅调试时暴露，生产由 nginx 反代

  frontend:
    image: nginx:1.27-alpine
    container_name: xingran-frontend
    restart: unless-stopped
    depends_on:
      backend:
        condition: service_healthy
    volumes:
      - ./frontend/dist:/usr/share/nginx/html:ro
      - ./nginx/default.conf:/etc/nginx/conf.d/default.conf:ro
    ports:
      - "80:80"
      - "443:443"
      - ./nginx/certs:/etc/nginx/certs:ro   # 可选：HTTPS 证书
    networks:
      - xingran-net

  # 可选：RPA Worker
  rpa-worker:
    build:
      context: ../rpa-worker
      dockerfile: ../rpa-worker/Dockerfile   # 若 rpa-worker 自带 Dockerfile
    image: rpa-worker:latest
    container_name: xingran-rpa-worker
    restart: unless-stopped
    depends_on:
      - backend
    env_file:
      - secrets.env
    environment:
      BACKEND_URL: http://backend:9000
    networks:
      - xingran-net
    profiles: ["rpa"]                       # 默认不启动，按需 --profile rpa

networks:
  xingran-net:
    driver: bridge
```

> **说明**：
> - `depends_on: condition: service_healthy` 确保 backend 等 PG/Redis 真正就绪后再启动
> - `profiles: ["rpa"]` 让 RPA Worker 默认不启动，需要时 `docker compose --profile rpa up`
> - 数据卷（postgres-data / redis-data / uploads / logs）**必须**挂载到主机，否则容器重建数据丢失

---

## 5. 后端 Dockerfile

仓库根无现成 Dockerfile，需自建 `xingran-deploy/backend/Dockerfile`：

```dockerfile
# xingran-deploy/backend/Dockerfile
# 多阶段构建：先编译，再瘦身运行镜像

# ============ 构建阶段 ============
FROM golang:1.24-alpine AS builder

# 国内构建可加：
# RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories

RUN apk add --no-cache git make

WORKDIR /src

# 复制 go.mod / go.sum 先（缓存命中）
COPY go.mod go.sum ./
RUN go mod download

# 复制源码
COPY . .

# 编译（含前端 embed 的版本；scripts/build/build-embedded.sh 走的是这一路）
ARG EMBED_FRONTEND=true
RUN if [ "$EMBED_FRONTEND" = "true" ]; then \
      sh scripts/build/build-embedded.sh; \
      mv xingran-backend-embedded-linux /out/xingran-backend; \
    else \
      CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /out/xingran-backend ./cmd/main.go; \
    fi

# ============ 运行阶段 ============
FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata curl wget \
    && addgroup -S xingran && adduser -S xingran -G xingran

WORKDIR /app

COPY --from=builder /out/xingran-backend /app/xingran-backend
COPY configs/config.yaml /app/configs/config.yaml

RUN mkdir -p /app/logs /app/uploads \
    && chown -R xingran:xingran /app

USER xingran

EXPOSE 9000

HEALTHCHECK --interval=30s --timeout=5s --start-period=30s --retries=3 \
  CMD wget -q -O - http://localhost:9000/health || exit 1

ENTRYPOINT ["/app/xingran-backend"]
```

> ⚠️ 镜像大小约 80 MB（Alpine + Go 二进制）。若前端未 embed，nginx 容器需要单独提供 `dist/`，见 §8。

---

## 6. 前端构建产物

构建前端后，把产物拷贝到 `xingran-deploy/frontend/dist/`：

```bash
cd <仓库根>/xingran-react-frontend
echo "VITE_API_BASE_URL=/api/v1" > .env.production
echo "VITE_ENABLE_REQUEST_ENCRYPTION=true" >> .env.production
npm install
npm run build
cp -r dist/* <xingran-deploy>/frontend/dist/
```

nginx 用 volume 挂载 `dist/` 后无需重启 container（容器内已是 ro mount，仅初次启动需 up）。

---

## 7. nginx 配置

`xingran-deploy/nginx/default.conf`：

```nginx
server {
    listen 80;
    server_name _;

    # SPA history fallback
    root /usr/share/nginx/html;
    index index.html;

    # 静态资源缓存
    location ~* \.(js|css|png|jpg|svg|woff2?)$ {
        expires 7d;
        add_header Cache-Control "public, immutable";
        try_files $uri =404;
    }

    location / {
        try_files $uri $uri/ /index.html;
    }

    # 反代后端 API
    location /api/ {
        proxy_pass http://backend:9000;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_read_timeout 60s;
        proxy_send_timeout 60s;
    }

    location /swagger/ {
        proxy_pass http://backend:9000/swagger/;
    }

    # 上传文件直出（避免大文件穿后端）
    location /uploads/ {
        proxy_pass http://backend:9000/uploads/;
    }

    # HTTPS（可选；生产推荐）
    # listen 443 ssl;
    # ssl_certificate     /etc/nginx/certs/fullchain.pem;
    # ssl_certificate_key /etc/nginx/certs/privkey.pem;
    # ssl_protocols TLSv1.2 TLSv1.3;
}
```

---

## 8. 敏感配置 secrets.env

按 [secret-management.md §2](../deployment/secret-management.md#2-部署期密钥生成一键脚本) 生成后写入：

```bash
# xingran-deploy/secrets.env（chmod 600，root:root）
DB_PASSWORD=<openssl rand -hex 16>
REDIS_PASSWORD=<openssl rand -hex 16>
JWT_SECRET=<openssl rand -hex 32>
JWT_SM2_PRIVATE_KEY=<hex>
JWT_SM2_PUBLIC_KEY=<hex>
SM4_KEY=<openssl rand -base64 16>

# 可选
BAIDU_MAP_AK=
RPA_AI_GENERATOR_KEY=
RPA_AI_GENERATOR_URL=
RPA_AI_AGENT_KEY=
RPA_AI_AGENT_URL=
```

`.env`（非敏感，可提交）：

```bash
# xingran-deploy/.env
DB_USER=xingran
DB_NAME=xingran_next
```

---

## 9. 启动与运维

### 9.1 启动

```bash
cd xingran-deploy

# 首次：构建镜像
docker compose build

# 启动（基础 4 服务）
docker compose up -d

# 启动 + RPA Worker
docker compose --profile rpa up -d

# 查看状态
docker compose ps
docker compose logs -f backend
```

### 9.2 验证

```bash
# 健康检查
curl -s http://localhost/health                      # 200（前端入口）
curl -s http://localhost/api/v1/health               # 取决于是否注册 health 路由
docker compose exec backend wget -q -O - http://localhost:9000/health

# 数据库就绪
docker compose exec postgres pg_isready -U xingran

# Redis
docker compose exec redis redis-cli -a "$REDIS_PASSWORD" ping
```

### 9.3 常用命令

```bash
# 停止
docker compose down                    # 保留数据卷
docker compose down -v                 # ⚠️ 同时删数据卷（慎用）

# 重启单个服务
docker compose restart backend

# 查看日志
docker compose logs -f --tail 100 backend
docker compose logs -f --since "10m" backend

# 进入容器调试
docker compose exec backend sh
docker compose exec postgres psql -U xingran -d xingran_next

# 升级后端镜像
docker compose build --no-cache backend
docker compose up -d backend
```

---

## 10. 生产化加固（可选）

### 10.1 资源限制

在每个 service 下加：

```yaml
    deploy:
      resources:
        limits:
          cpus: '4'
          memory: 4G
        reservations:
          cpus: '1'
          memory: 1G
```

### 10.2 日志驱动

默认 json-file 无上限会撑爆磁盘。改：

```yaml
services:
  backend:
    logging:
      driver: json-file
      options:
        max-size: "100m"
        max-file: "10"
        tag: "xingran-backend"
```

### 10.3 只读文件系统

```yaml
services:
  backend:
    read_only: true
    tmpfs:
      - /tmp
    volumes:
      - ./uploads:/app/uploads       # 仍需 RW
      - ./logs:/app/logs
```

### 10.4 网络隔离

- frontend 对外暴露 80/443
- backend 仅在 xingran-net 内可达
- postgres/redis 完全不对外
- 调试时临时 `127.0.0.1:port:port` 暴露

---

## 11. 备份与恢复

```bash
# PG 备份
docker compose exec -T postgres pg_dump -U xingran xingran_next > backup-$(date +%Y%m%d).sql

# PG 恢复
cat backup-20260815.sql | docker compose exec -T postgres psql -U xingran -d xingran_next

# Redis 备份（AOF 已在卷内，直接 cp redis-data 即可）
docker compose stop redis
cp -r redis-data redis-data.bak
docker compose start redis

# 上传文件
rsync -av uploads/ backup-uploads-$(date +%Y%m%d)/
```

---

## 12. 故障排查

| 症状 | 原因 | 排查 |
|---|---|---|
| `backend` 启动后立即退出 exit 1 | 缺少 env 或 config.yaml 路径错 | `docker compose logs backend` 看头部 |
| `connection refused` to postgres | PG 还在 init / 网络未通 | `docker compose ps` 看 health 状态；`docker compose logs postgres` |
| `NOAUTH` Redis 错误 | password 与启动命令不一致 | 检查 `secrets.env` 中 `REDIS_PASSWORD` 是否被换行/引号破坏 |
| 前端打开空白 | nginx SPA fallback 未配 | 确认 `try_files $uri $uri/ /index.html;` |
| `/api/...` 404 | nginx 反代路径 | 改成 `proxy_pass http://backend:9000;`（去掉 `/api/` 前缀） |
| 上传文件消失 | `uploads/` 卷未挂载 | `docker compose exec backend ls /app/uploads` |
| `templates/...` no such file | Dockerfile 没走 embed 构建路径 | 检查 `scripts/build/build-embedded.sh` 是否被调用 |
| 自动迁移没跑 | `SKIP_AUTOMIGRATE=true` | 设为 `false` 或移除该环境变量 |

---

## 13. 相关文档

- [single-machine-deployment.md](single-machine-deployment.md) — 本机进程部署（非容器化）
- [deployment.md](deployment.md) — 生产 systemd 部署
- [secret-management.md](secret-management.md) — 密钥生成与轮转
- [capacity-planning.md](capacity-planning.md) — 容器资源限制设置参考
- [Docker 官方文档](https://docs.docker.com/compose/) — compose 语法参考
- [README.md](../../README.md) — 快速开始

---

## 14. 变更日志

| 日期 | 变更 | 作者 |
|---|---|---|
| 2026-08-15 | 初版：补齐 README 中标注缺失的 docker-compose 模板与运维指南 | Claude |