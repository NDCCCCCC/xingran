# 单机全栈部署指南

> 适用场景：在一台物理机 / 虚拟机上部署完整的 XingRan-Next（后端 + 前端 + PostgreSQL + Redis + 可选 RPA Worker）。
> 适用规模：试点 / 小型团队（< 100 用户）首选方案。
> 规模超 1000 用户或开启 RPA 真实 Docker 模式后，建议拆库拆服务并参考 [生产部署指南](deployment.md)。

---

## 1. 与生产部署文档的边界

| 文档 | 场景 | 部署方式 |
|------|------|----------|
| **本文**（single-machine） | 一台设备跑通全套（开发 / 试点 / 小型团队） | 本机进程 + 本机 PG/Redis，或 Docker Compose |
| [deployment.md](deployment.md) | 生产服务器（`/app/szh/`，内网 10.62.10.34 等场景） | systemd + 内网 PG/Redis |
| [secret-management.md](secret-management.md) | 密钥生成、轮转、泄漏处置 | 跨文档通用 |

如果只是"想跑起来看看"，按本文 §4 一条龙走通即可；如果准备上生产，先读完本文 §1-§5，再切到 [deployment.md](deployment.md)。

---

## 2. 硬件规格

### 2.1 选型矩阵

| 角色 | 最低 | 推荐 | 大型/启用 RPA |
|---|---|---|---|
| **CPU** | 4 核 | 8 核+ | 16 核+ |
| **内存** | 16 GB | 32 GB | 64 GB+（每 Worker 容器 1-2 GB） |
| **系统盘** | 60 GB SSD | 100 GB+ SSD | 200 GB SSD |
| **数据盘** | 100 GB | 500 GB+ | 1 TB+ |
| **网络** | 千兆 | 千兆 | 千兆 + 公网带宽 |

> ⚠️ 启用 RPA 真实 Docker 模式（生产 `enable_mock_docker: false`）时，`rpa.scaling.max_workers: 10` 会在主机上拉起 10 个 `rpa-worker` 容器，**每个 1-2 GB 内存**。这是"大型"规格的内存大幅增长主因。

### 2.2 操作系统

| OS | 推荐 | 备注 |
|---|---|---|
| **Ubuntu 22.04 LTS** | ⭐ 首选 | apt 仓库覆盖全部依赖，systemd 完整 |
| **Debian 12** | ⭐ 首选 | 与 Ubuntu 类似，更稳定 |
| **CentOS Stream 9 / RHEL 9** | 可选 | 注意 `firewalld` 与 SELinux 配置 |
| **Windows Server 2019/2022** | 仅开发 | 后端可跑，但 systemd 替代品需用 NSSM / Task Scheduler |
| **macOS** | 仅开发 | 同上 |

### 2.3 必须安装的运行时 / 服务

| 组件 | 版本 | 用途 | 安装方式（Ubuntu/Debian） |
|---|---|---|---|
| **Go** | 1.24+（toolchain go1.24.5） | 后端编译 | `https://go.dev/dl/` 或 `snap install go --classic` |
| **Node.js** | 24+ | 前端构建 | `nvm install 24` 或 `nodesource` 仓库 |
| **npm** | 11+ | 前端依赖 | 随 Node 一同安装 |
| **PostgreSQL** | 18+ | 主库 | `apt install postgresql-18` 或官方 PG 仓库 |
| **Redis** | 7.4+ | L2 缓存 | `apt install redis-server` |
| **OpenSSL** | 3.x | 密钥生成 | 系统自带 |
| **nginx**（推荐） | 1.24+ | 前端托管 + 反向代理 + HTTPS 终止 | `apt install nginx` |
| **Python** | 3.11+ | RPA Worker（可选） | `apt install python3 python3-pip` |
| **Docker** | 24+ | RPA Worker 真实模式（可选） | 官方 docker-ce 仓库 |

---

## 3. 网络与端口规划

### 3.1 端口分配

| 端口 | 用途 | 暴露范围 | 备注 |
|---|---|---|---|
| 9000 | 后端 API（含 Swagger `/swagger/index.html`） | 仅本机 / 内网 | nginx 反代对外 |
| 4000 | 前端开发服（`npm run dev`） | 仅本机 | 仅开发期 |
| 80 / 443 | 生产前端 + API（HTTPS） | 公网 | nginx 托管前端 + 反代 `/api/v1` |
| 5432 | PostgreSQL | 仅本机 / 内网 | 严禁对公网开放 |
| 6379 | Redis | 仅本机 / 内网 | 严禁对公网开放 |
| 2375 | Docker API（RPA Worker 调度） | 仅本机 | `enable_mock_docker: false` 时需要 |
| 22 | SSH（出方向，纳管网络设备） | 出 | — |
| 161/162 | SNMP（出方向，纳管网络设备） | 出 | — |

### 3.2 防火墙（Linux）

```bash
# 仅放行对外 80/443 + SSH
sudo ufw allow 22/tcp
sudo ufw allow 80/tcp
sudo ufw allow 443/tcp
sudo ufw deny 5432
sudo ufw deny 6379
sudo ufw deny 9000
sudo ufw enable
```

---

## 4. 一条龙部署流程

### 4.1 系统初始化

```bash
# 1. 创建专用账户（避免 root 运行）
sudo useradd -r -s /sbin/nologin xingran
sudo mkdir -p /opt/xingran/{configs,logs,uploads}
sudo chown -R xingran:xingran /opt/xingran
sudo chmod 750 /opt/xingran

# 2. 安装运行时（Ubuntu 示例）
sudo apt update
sudo apt install -y postgresql-18 redis-server nginx openssl curl git
# Go / Node 单独安装（apt 版本通常较旧）

# 3. 初始化 PG
sudo -u postgres createuser xingran -P   # 交互设置密码
sudo -u postgres createdb xingran_next -O xingran

# 4. 配置 Redis 密码
sudo vim /etc/redis/redis.conf
#   requirepass <openssl rand -hex 16>
#   bind 127.0.0.1
sudo systemctl restart redis-server
```

### 4.2 生成密钥

```bash
# 通用密钥
DB_PASSWORD=$(openssl rand -hex 16)
REDIS_PASSWORD=$(openssl rand -hex 16)
JWT_SECRET=$(openssl rand -hex 32)
SM4_KEY=$(openssl rand -base64 16)

# SM2 密钥对（绝不能复用仓库默认值 d8d9a3e6...）
# 方式 A：用 gmssl（推荐）
sudo apt install -y gmssl
gmssl sm2keygen -pass 1234 -out /tmp/priv.pem -pubout /tmp/pub.pem
SM2_PRIV=$(grep -v '^-' /tmp/priv.pem | tr -d '\n' | xxd -p -c 64)
SM2_PUB=$(grep -v '^-' /tmp/pub.pem | tr -d '\n' | xxd -p -c 64)

# 写敏感文件（仅 root 可读）
sudo tee /etc/xingran/secrets.env >/dev/null <<EOF
DB_HOST=127.0.0.1
DB_PORT=5432
DB_USER=xingran
DB_PASSWORD=${DB_PASSWORD}
DB_NAME=xingran_next
DB_SSLMODE=disable

REDIS_HOST=127.0.0.1
REDIS_PORT=6379
REDIS_PASSWORD=${REDIS_PASSWORD}

SERVER_HOST=0.0.0.0
SERVER_PORT=9000
SERVER_MODE=release

JWT_SECRET=${JWT_SECRET}
JWT_SM2_PRIVATE_KEY=${SM2_PRIV}
JWT_SM2_PUBLIC_KEY=${SM2_PUB}

SM4_KEY=${SM4_KEY}

BAIDU_MAP_AK=
EOF

sudo chmod 600 /etc/xingran/secrets.env
sudo chown root:root /etc/xingran/secrets.env
```

### 4.3 后端构建 + 部署

```bash
# 1. 在构建机（或本机）拉代码
git clone <repo-url> /tmp/xingran-src
cd /tmp/xingran-src

# 2. 准备配置（从模板复制，删掉敏感字段）
cp configs/config.prod.example.yaml /opt/xingran/configs/config.yaml
sudo chown xingran:xingran /opt/xingran/configs/config.yaml

# 3. 关键项修改（用 sed 或编辑器）
sudo -E vim /opt/xingran/configs/config.yaml
#   server.mode: release
#   database.host/port/user/password/dbname
#   cache.host/port/password
#   rpa.scaling.enable_mock_docker: false  # 若启用 RPA 真实模式
#   security.request_encryption.require_encryption: true

# 4. 编译（含前端 embed 的版本）
scripts/build/build-embedded.sh
# 产物：xingran-backend-embedded-linux

# 5. 部署
sudo cp xingran-backend-embedded-linux /opt/xingran/xingran-backend
sudo chmod +x /opt/xingran/xingran-backend
sudo chown xingran:xingran /opt/xingran/xingran-backend

# 6. systemd service（参考 deployment.md §4.1）
sudo tee /etc/systemd/system/xingran-backend.service >/dev/null <<'EOF'
[Unit]
Description=XingRan-Next Backend
After=network.target postgresql.service redis-server.service
Wants=postgresql.service redis-server.service

[Service]
Type=simple
User=xingran
Group=xingran
WorkingDirectory=/opt/xingran
ExecStart=/opt/xingran/xingran-backend
EnvironmentFile=/etc/xingran/secrets.env
Restart=always
RestartSec=5
StartLimitBurst=3
StartLimitIntervalSec=60

StandardOutput=journal
StandardError=journal
SyslogIdentifier=xingran-backend

NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/opt/xingran/logs /opt/xingran/uploads

[Install]
WantedBy=multi-user.target
EOF

sudo systemctl daemon-reload
sudo systemctl enable --now xingran-backend
sudo systemctl status xingran-backend
```

### 4.4 前端构建 + nginx 托管

```bash
# 1. 构建前端
cd /tmp/xingran-src/xingran-react-frontend
npm install
# 配置生产 API 地址
cat > .env.production <<EOF
VITE_API_BASE_URL=https://yourdomain.com/api/v1
VITE_ENABLE_REQUEST_ENCRYPTION=true
EOF
npm run build
# 产物：dist/

# 2. 部署到 nginx
sudo mkdir -p /var/www/xingran
sudo cp -r dist/* /var/www/xingran/
sudo chown -R www-data:www-data /var/www/xingran

# 3. nginx 配置
sudo tee /etc/nginx/sites-available/xingran >/dev/null <<'EOF'
server {
    listen 80;
    server_name yourdomain.com;

    # 前端静态资源
    root /var/www/xingran;
    index index.html;

    # SPA history fallback
    location / {
        try_files $uri $uri/ /index.html;
    }

    # 反代后端 API
    location /api/ {
        proxy_pass http://127.0.0.1:9000;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_read_timeout 60s;
    }

    # Swagger
    location /swagger/ {
        proxy_pass http://127.0.0.1:9000/swagger/;
    }

    # 静态上传（避免大文件走 API）
    location /uploads/ {
        alias /opt/xingran/uploads/;
        expires 7d;
    }
}
EOF

sudo ln -sf /etc/nginx/sites-available/xingran /etc/nginx/sites-enabled/
sudo nginx -t && sudo systemctl reload nginx
```

### 4.5 HTTPS（Let's Encrypt）

```bash
sudo apt install -y certbot python3-certbot-nginx
sudo certbot --nginx -d yourdomain.com --non-interactive --agree-tos -m you@example.com
# 自动续期由 certbot timer 完成，无需额外配置
```

### 4.6 可选服务

#### RPA Worker（独立进程模式，最简单）

```bash
# rpa-worker 是独立 Go 服务（见 rpa-worker/ 目录）
cd /tmp/xingran-src/rpa-worker
go build -o /opt/xingran/rpa-worker ./cmd/
sudo tee /etc/systemd/system/xingran-rpa.service >/dev/null <<'EOF'
[Unit]
Description=XingRan RPA Worker
After=xingran-backend.service

[Service]
Type=simple
User=xingran
Group=xingran
WorkingDirectory=/opt/xingran
ExecStart=/opt/xingran/rpa-worker
EnvironmentFile=/etc/xingran/secrets.env
Restart=always

[Install]
WantedBy=multi-user.target
EOF
sudo systemctl enable --now xingran-rpa
```

#### RPA Worker（Docker 真实模式，需 `enable_mock_docker: false`）

```bash
# 1. 启用 Docker API 监听
sudo tee /etc/docker/daemon.json >/dev/null <<'EOF'
{
  "hosts": ["unix:///var/run/docker.sock", "tcp://127.0.0.1:2375"]
}
EOF
sudo systemctl restart docker

# 2. 构建 worker 镜像
cd /tmp/xingran-src/rpa-worker
docker build -t rpa-worker:latest .

# 3. 改配置
#   rpa.scaling.enable_mock_docker: false
#   rpa.scaling.docker.docker_host: 127.0.0.1
#   rpa.scaling.docker.docker_port: 2375

# 4. 重启后端
sudo systemctl restart xingran-backend
```

---

## 5. 部署后验证清单

按顺序逐项确认：

### 5.1 服务状态

```bash
# 后端
sudo systemctl status xingran-backend          # active (running)
sudo journalctl -u xingran-backend -n 50       # 无 [SECURITY WARNING]
ss -tlnp | grep 9000                            # 监听 0.0.0.0:9000

# 前端
curl -I http://yourdomain.com                   # 200
curl -I https://yourdomain.com                  # 200/301

# 数据库
sudo -u postgres psql -c "SELECT 1"             # OK
sudo -u postgres psql xingran_next -c "\dt"     # 表已创建（自动迁移执行）

# Redis
redis-cli -a "$REDIS_PASSWORD" ping             # PONG

# nginx
sudo nginx -t                                    # syntax ok
```

### 5.2 安全校验

```bash
# SM2 不是仓库默认值
sudo systemctl show xingran-backend -p Environment --no-pager | grep SM2_PRIVATE_KEY
# 应显示长随机 hex，不是 d8d9a3e6b356...

# SM4 不是默认 test-secret
echo $SM4_KEY                                    # 随机 base64

# JWT 不是 please-change
echo $JWT_SECRET                                 # 随机 hex

# 服务实际加载的 env
sudo systemctl show xingran-backend | grep -E "EnvironmentFile|Environment="
# 应包含 DB_PASSWORD/REDIS_PASSWORD/JWT_SECRET/SM4_KEY/... 共 15+ 项
```

### 5.3 业务功能

```bash
# 健康检查
curl -s http://127.0.0.1:9000/health | jq .

# 登录（拿 token）
TOKEN=$(curl -s -X POST https://yourdomain.com/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"<初始密码>"}' | jq -r .data.accessToken)
echo "$TOKEN" | cut -d. -f2 | base64 -d 2>/dev/null | jq .   # issuer = XingRan-Next

# 访问受保护接口（菜单/用户/部门）
curl -s https://yourdomain.com/api/v1/system/menus/my-menus \
  -H "Authorization: Bearer $TOKEN" | jq .

# 国密 SM4 解密链路（设备密码 / AD 凭据 / RPA 凭证任选其一）
# 后台 → 网络设备 → 测试连接
# 后台 → AD 域控 → 同步测试
```

---

## 6. 关键配置项速查

完整配置项参考 [`configs/config.prod.example.yaml`](../../configs/config.prod.example.yaml)。以下是单机部署必改项：

| 配置块 | 字段 | 单机推荐值 | 说明 |
|---|---|---|---|
| `server` | `mode` | `release` | 生产必改（关闭 debug 堆栈） |
| `server` | `host` | `0.0.0.0` | 监听所有网卡（nginx 反代需要） |
| `database` | `host` | `127.0.0.1` | 本机 PG |
| `database` | `sslmode` | `disable` | 内网；公网或跨网段改 `require` / `verify-full` |
| `database` | `max_open_conns` | `25-50` | 25=小型；50=中型 |
| `cache` | `pool_size` | `50` | 与并发请求匹配 |
| `cache` | `warm_up_enabled` | `false` | 生产关闭（启动期间拉高 DB） |
| `jwt` | `use_sm2` | `true` | 国密合规强制 |
| `security` | `request_encryption.enabled` | `true` | 生产必开 |
| `security` | `request_encryption.require_encryption` | `true` | 灰度后开启 |
| `security` | `response_encryption.enabled` | `false` | 默认关，参数管理动态开 |
| `rpa.scaling` | `enable_mock_docker` | `false` | 真实模式；开发期可 `true` |
| `rpa.worker` | `min_workers` / `max_workers` | `2` / `10` | 默认值即可 |
| `log` | `level` | `info` | 生产关 debug |
| `ad` | `tls_skip_verify` | `true` | 当前部署 AD 无 TLS；启用 LDAPS 后改 `false` |
| `vdi` | `tls_skip_verify` | `true` | 同上 |

---

## 7. 目录结构（推荐布局）

```
/opt/xingran/
├── xingran-backend                # Go 二进制（含 embed 前端 + TextFSM）
├── rpa-worker                     # 可选：RPA Worker 二进制
├── configs/
│   └── config.yaml                # 应用配置（不含密钥）
├── logs/                          # logrus + lumberjack 轮转
├── uploads/                       # 用户上传 + rpa/screenshots, downloads
└── secrets.env → /etc/xingran/secrets.env   # 软链或独立

/etc/xingran/
└── secrets.env                    # 600 权限，root:root

/etc/systemd/system/
├── xingran-backend.service
└── xingran-rpa.service            # 可选

/var/www/xingran/                  # 前端 dist
/etc/nginx/sites-available/xingran
```

---

## 8. 常见陷阱（踩过的坑）

| 症状 | 原因 | 解决 |
|---|---|---|
| 启动报 `SM4_KEY 是仓库默认值` | 用了 `dGVzdC1zZWNyZXQxNiEhIQ==` 默认值 | 用 `openssl rand -base64 16` 重新生成 |
| 启动报 `SM2 私钥是公开值` | 用了仓库默认 `d8d9a3e6...` | 用 gmssl 重新生成 SM2 密钥对 |
| 登录后立刻被踢出 | SM2 私钥每次启动动态生成（重启即换） | 显式配置 `JWT_SM2_PRIVATE_KEY` |
| PG 连接失败 `FATAL: no pg_hba.conf entry` | `sslmode: require` 但服务端未配 TLS | 改 `disable`（内网）或启用服务端 TLS |
| Redis 报 `NOAUTH` | Redis 设了密码但配置 `password: ""` | 通过 `REDIS_PASSWORD` 环境变量注入 |
| `lumberjack` 报权限不足 | systemd `ProtectSystem=strict` | 加 `ReadWritePaths=/opt/xingran/logs` |
| 前端打开空白 | nginx 没配 SPA fallback | `try_files $uri $uri/ /index.html;` |
| `/api/...` 404 | nginx 反代路径不对 | `proxy_pass http://127.0.0.1:9000;`（**注意尾斜杠**） |
| RPA Worker 报 `connection refused :2375` | Docker daemon 没监听 TCP | 配 `/etc/docker/daemon.json` 加 `tcp://127.0.0.1:2375` |
| `templates/...` no such file | 二进制是早期版本未 embed | 改用 `scripts/build/build-embedded.sh` 重新构建 |
| 工位导入报"部门名称不是 UUID" | org_id 传了部门名 | 导入数据必须用 UUID（migration 084 修复历史数据） |

---

## 9. 升级流程

```bash
# 1. 备份
sudo cp /opt/xingran/xingran-backend /opt/xingran/xingran-backend.bak.$(date +%Y%m%d)

# 2. 停止
sudo systemctl stop xingran-backend

# 3. 替换二进制
sudo cp xingran-backend-new /opt/xingran/xingran-backend
sudo chown xingran:xingran /opt/xingran/xingran-backend
sudo chmod +x /opt/xingran/xingran-backend

# 4. 如 config.yaml 变更，手动合并
sudo -E vim /opt/xingran/configs/config.yaml

# 5. 启动 + 验证
sudo systemctl start xingran-backend
sudo systemctl status xingran-backend
curl -s http://127.0.0.1:9000/health
```

前端升级：

```bash
cd xingran-react-frontend
git pull
npm install
npm run build
sudo cp -r dist/* /var/www/xingran/
sudo nginx -s reload
```

---

## 10. 备份与恢复

| 内容 | 路径 | 备份方式 | 频率 |
|---|---|---|---|
| 数据库 | PostgreSQL | `pg_dump xingran_next` | 每日 |
| 上传文件 | `/opt/xingran/uploads/` | rsync 异地 | 每日 |
| 配置 | `/opt/xingran/configs/config.yaml` | git 追踪 | 每次修改 |
| 密钥 | `/etc/xingran/secrets.env` | 密码管理器 + 加密 U 盘 | 每次修改 |
| 二进制 | `/opt/xingran/xingran-backend` | 与 release tag 对应归档 | 每次升级 |
| 日志 | `/opt/xingran/logs/` | journald 永久存储 | 自动 |

应急恢复见 [deployment.md §6.2](deployment.md#62-密钥丢失应急)。

---

## 11. 相关文档

- [deployment.md](deployment.md) — 生产 systemd 部署（密钥、升级、故障排查）
- [secret-management.md](secret-management.md) — 密钥生成、轮转、泄漏处置
- [docker-compose.md](docker-compose.md) — Docker Compose 一键编排（容器化方案）
- [capacity-planning.md](capacity-planning.md) — 容量规划与硬件选型
- [架构/项目概述和架构设计](../architecture/项目概述和架构设计.md) — 系统分层与模块清单
- [架构/安全和认证设计（国密）](../architecture/安全和认证设计（国密）.md) — JWT/SM2/SM3/SM4 设计
- [架构/数据库设计](../architecture/数据库设计.md) — 表结构与迁移机制
- [README.md](../../README.md) — 快速开始与命令参考

---

## 12. 变更日志

| 日期 | 变更 | 作者 |
|---|---|---|
| 2026-08-15 | 初版：补充生产部署指南未覆盖的单机全栈场景、硬件规格、网络规划、踩坑清单 | Claude |