# 生产环境部署指南

> 适用范围：XingRan-Next 后端服务的生产部署
> 部署路径：`/app/szh/`
> OS：Linux（systemd 服务管理）

---

## 1. 目录结构

```
/app/szh/
├── xingran-backend              # Go 编译产物（构建自 cmd/main.go；TextFSM 模板 + 前端皆已嵌入二进制）
├── xingran-react-frontend/      # 前端构建产物（可选，embed 构建已嵌入二进制）
├── configs/
│   └── config.yaml            # 应用配置（非敏感，由 git 管理）
├── logs/                      # 应用日志（lumberjack 轮转）
├── uploads/                   # 用户上传文件
└── secrets.env → /etc/xingran/secrets.env  # 软链接或独立存放敏感 env
```

> **重要**：`configs/config.yaml` **不含任何密钥**——所有敏感值通过系统环境变量注入（见 §3）。
>
> **重要**：网络设备的 TextFSM 模板（`templates/` 目录下 ~200 个 `.textfsm` 文件 + `lldp/` 子目录）已通过 `go:embed` 嵌入 `xingran-backend` 二进制，**不需要也不应该**单独部署 `templates/` 目录到 `/app/szh/`。如果服务日志出现 `no such file or directory templates/...`，请检查 binary 是否包含模板（`strings /app/szh/xingran-backend | grep huawei_vrp` 应有命中）。

---

## 2. 构建产物

### 2.1 嵌入式构建（前端打包进二进制）

```bash
# 在构建机（Windows）执行
scripts\build-linux.bat

# 产物
xingran-backend-embedded-linux
```

### 2.2 部署到服务器

```bash
# 在生产服务器
sudo useradd -r -s /sbin/nologin szh  # 创建专用账户（如已存在则跳过）
sudo mkdir -p /app/szh/{configs,logs,uploads}
sudo chown -R szh:szh /app/szh

# 上传二进制
scp xingran-backend-embedded-linux szh@10.62.10.34:/app/szh/xingran-backend
# 或 rsync / 堡垒机中转
sudo chmod +x /app/szh/xingran-backend
sudo chown szh:szh /app/szh/xingran-backend
```

---

## 3. 敏感环境变量管理（三种方案）

根据部署场景选择一种。**所有方案都满足"密钥不入 git"**，区别在于加密强度。

### 3.0 决策树

```
你是单机/容器部署?
├── 是 → 3.1 明文 .env(项目内,chmod 600)
└── 否 (systemd 服务)
    ├── 需要审计/合规?  → 3.3 systemd 凭据加密(systemd-creds)
    └── 不需要           → 3.2 systemd EnvironmentFile(明文 + 600 权限) ⭐ 推荐起点
```

### 3.1 方案 A：明文 .env（最简单，单机/容器）

**适用场景**：单机部署、Docker 容器、本地服务器

**做法**：

1. 在项目内创建 `.env` 文件（与 `configs/config.yaml` 同级或项目根）
2. 文件权限 600：
   ```bash
   chmod 600 .env
   chown $USER:$USER .env
   ```
3. 应用启动时 `cmd/main.go:47` 已用 `godotenv.Load()` 自动加载

**示例 `.env`**：

```bash
# /app/szh/.env
# 权限: 600, owner: szh:szh

DB_HOST=10.62.10.34
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=...
DB_NAME=xingran
DB_SSLMODE=disable

REDIS_HOST=10.62.10.34
REDIS_PORT=6379
REDIS_PASSWORD=...
REDIS_DB=0

SERVER_HOST=0.0.0.0
SERVER_PORT=9000
SERVER_MODE=release

JWT_SECRET=...
JWT_SM2_PRIVATE_KEY=...
JWT_SM2_PUBLIC_KEY=...

SM4_KEY=...

BAIDU_MAP_AK=...
RPA_AI_GENERATOR_KEY=
RPA_AI_GENERATOR_URL=
RPA_AI_AGENT_KEY=
RPA_AI_AGENT_URL=
```

**Docker 场景**：

```bash
# 方式 1: --env-file
docker run -d --env-file=/path/to/.env -p 9000:9000 xingran-backend

# 方式 2: docker-compose.yml
services:
  backend:
    image: xingran-backend
    env_file: .env
```

**优点**：
- 零配置，开发者熟悉
- `godotenv` 已内置
- Docker 原生支持

**缺点**：
- 文件明文存磁盘
- 多实例需要分发同一份 .env
- 备份时易遗漏

---

### 3.2 方案 B：systemd EnvironmentFile（推荐起步）⭐

**适用场景**：Linux + systemd 服务（**最常见**）

**做法**：

1. 在 `/etc/xingran/` 创建 secrets 文件：
   ```bash
   sudo mkdir -p /etc/xingran
   sudo vim /etc/xingran/secrets.env
   ```

2. 填入密钥（**从密码管理器复制，不要 echo 到屏幕**）：
   ```bash
   # /etc/xingran/secrets.env
   # 权限: 600, owner: root:root
   # 修改后: sudo systemctl daemon-reload && sudo systemctl restart szh-backend

   DB_HOST=10.62.10.34
   DB_PORT=5432
   DB_USER=postgres
   DB_PASSWORD=...
   DB_NAME=xingran
   DB_SSLMODE=disable

   REDIS_HOST=10.62.10.34
   REDIS_PORT=6379
   REDIS_PASSWORD=...
   REDIS_DB=0

   SERVER_HOST=0.0.0.0
   SERVER_PORT=9000
   SERVER_MODE=release

   JWT_SECRET=...
   JWT_SM2_PRIVATE_KEY=...
   JWT_SM2_PUBLIC_KEY=...

   SM4_KEY=...

   BAIDU_MAP_AK=...
   RPA_AI_GENERATOR_KEY=
   RPA_AI_GENERATOR_URL=
   RPA_AI_AGENT_KEY=
   RPA_AI_AGENT_URL=
   ```

3. 设权限（仅 root 可读）：
   ```bash
   sudo chmod 600 /etc/xingran/secrets.env
   sudo chown root:root /etc/xingran/secrets.env
   ls -la /etc/xingran/secrets.env
   # 验证: -rw------- 1 root root
   ```

4. service 文件加 `EnvironmentFile=`：
   ```ini
   [Service]
   EnvironmentFile=/etc/xingran/secrets.env
   ExecStart=/app/szh/xingran-backend
   ```

**优点**：
- 零额外依赖
- 与 systemd 深度集成
- 权限管理简单（文件 600）
- 修改后 `systemctl daemon-reload` + restart 立即生效

**缺点**：
- 文件明文存磁盘
- 备份/迁移时易遗忘
- 多服务器需要手动同步

---

### 3.3 方案 C：systemd 凭据加密（最安全）

**适用场景**：合规要求、密钥不能明文落盘、多机密钥管理

**原理**：用 systemd 自带的 [Credential Encryption](https://www.freedesktop.org/software/systemd/man/systemd.exec.html#Credentials) 功能，把密钥加密后存到 `/etc/credstore/` 或 `/var/lib/credstore/`，service 启动时 systemd 用 host key 解密后注入到 `$CREDENTIALS_DIRECTORY/<name>` 文件。

**一次性初始化**（在生产服务器）：

```bash
# 1. 生成 systemd host key(只需一次)
sudo systemd-creds setup
# 输出: "Credential secret file ... not located on encrypted media, using anyway"
#       警告可忽略(明文磁盘也能用,只是建议加密介质)

# 2. 创建凭据存储目录
sudo mkdir -p /etc/credstore

# 3. 加密每个敏感值(交互式输入,不入 history)
#    推荐把所有 env 打包到一个 credential(简化管理)
sudo systemd-creds encrypt --name=szh-secrets - /etc/credstore/szh-secrets.cred <<'EOF'
DB_HOST=10.62.10.34
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=your_db_password
DB_NAME=xingran
DB_SSLMODE=disable

REDIS_HOST=10.62.10.34
REDIS_PORT=6379
REDIS_PASSWORD=your_redis_password
REDIS_DB=0

SERVER_HOST=0.0.0.0
SERVER_PORT=9000
SERVER_MODE=release

JWT_SECRET=your_jwt_secret
JWT_SM2_PRIVATE_KEY=your_sm2_private_key
JWT_SM2_PUBLIC_KEY=your_sm2_public_key

SM4_KEY=your_sm4_key

BAIDU_MAP_AK=your_baidu_ak
RPA_AI_GENERATOR_KEY=
RPA_AI_GENERATOR_URL=
RPA_AI_AGENT_KEY=
RPA_AI_AGENT_URL=
EOF

# 4. 验证加密文件
sudo systemd-creds list
# 应显示: szh-secrets → /etc/credstore/szh-secrets.cred
```

**service 文件**：

```ini
[Service]
# 加密凭据(service 启动时 systemd 自动解密到 $CREDENTIALS_DIRECTORY/szh-secrets)
LoadCredentialEncrypted=szh-secrets:/etc/credstore/szh-secrets.cred

# 启动前把解密后的凭据转成 .env 文件(tmpfs,重启即消失)
ExecStartPre=/bin/sh -c 'cat $CREDENTIALS_DIRECTORY/szh-secrets > /run/szh-secrets.env && chmod 600 /run/szh-secrets.env'

# 读取解密后的 env
EnvironmentFile=/run/szh-secrets.env
# ExecStartPre 等待 /run/szh-secrets.env 准备就绪
ExecStart=/app/szh/xingran-backend
```

**优点**：
- 密钥**加密**存磁盘（不是明文）
- 静态文件可备份到任何介质（仍是密文）
- 多服务器只需同步 host key + 密文
- 重启 `/run/szh-secrets.env` 自动消失

**缺点**：
- 配置稍复杂（systemd-creds 操作）
- host key (`/var/lib/systemd/credential.secret`) 需妥善保管
- 密钥轮转需 `systemd-creds setup --reset` 或重新加密
- ⚠️ 暂未做自动化测试（**需在生产部署时验证完整流程**）

**密钥轮转**（轮换 host key）：

```bash
# 1. 备份当前 host key
sudo cp /var/lib/systemd/credential.secret /root/backup/systemd-host-key-$(date +%Y%m%d).secret

# 2. 重新生成 host key(会失效所有已加密的凭据)
sudo systemd-creds setup --reset

# 3. 重新加密所有凭据(见上面"3. 加密每个敏感值")

# 4. 重启 service
sudo systemctl restart szh-backend
```

---

### 3.4 验证 env 加载（不管哪种方案）

**正确方法**：

```bash
# 1. 启动 service
sudo systemctl start szh-backend

# 2. 看 service 实际加载的 env vars
sudo systemctl show szh-backend | grep -E "EnvironmentFile|Environment="
# 应显示所有 18+ 个 env vars,含 DB_HOST/SM4_KEY/JWT_SECRET 等

# 3. 单个验证
sudo systemctl show szh-backend -p Environment --no-pager | grep SM4_KEY
# 应显示: Environment=SM4_KEY=QmxQNxBXxcY2K8X8by4qBA==
```

**错误方法**（**不要用**）：

```bash
# ❌ 错: 这是 PID 1 的环境,不是 service 的
sudo systemctl show-environment | grep SM4_KEY

# ❌ 错: 会展开到 systemd 上下文,无意义
printenv | grep SM4_KEY  # 普通用户下 SM4_KEY 不存在
```

---

## 4. systemd 服务

### 4.1 service 文件

`/etc/systemd/system/szh-backend.service`：

```ini
[Unit]
Description=SZH XingRan-Next Backend
After=network.target postgresql.service redis-server.service
Wants=postgresql.service redis-server.service

[Service]
Type=simple
User=szh
Group=szh
WorkingDirectory=/app/szh
ExecStart=/app/szh/xingran-backend
EnvironmentFile=/etc/xingran/secrets.env
Restart=always
RestartSec=5
StartLimitBurst=3
StartLimitIntervalSec=60

# 日志接入 journald
StandardOutput=journal
StandardError=journal
SyslogIdentifier=szh-backend

# 安全加固
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/app/szh/logs /app/szh/uploads
ProtectKernelTunables=true
ProtectKernelModules=true
ProtectControlGroups=true

[Install]
WantedBy=multi-user.target
```

### 4.2 启用 + 启动

```bash
sudo systemctl daemon-reload
sudo systemctl enable szh-backend   # 开机自启
sudo systemctl start szh-backend    # 启动
sudo systemctl status szh-backend   # 查看状态
```

### 4.3 常用命令

```bash
# 实时日志
sudo journalctl -u szh-backend -f

# 最近 100 行
sudo journalctl -u szh-backend -n 100 --no-pager

# 重启服务
sudo systemctl restart szh-backend

# 停止服务
sudo systemctl stop szh-backend

# 禁用自启
sudo systemctl disable szh-backend
```

---

## 5. 验证清单

部署后**必须**逐项验证：

- [ ] **服务状态**：`systemctl status szh-backend` 显示 `active (running)`
- [ ] **环境变量加载**：`systemctl show szh-backend | grep -E "EnvironmentFile|Environment="` 显示所有 18+ 个 env vars
- [ ] **无 SM4 警告**：日志中**没有** `[SECURITY WARNING] SM4_KEY 是仓库默认值`（说明用了真 key）
- [ ] **无 SM2 警告**：JWT Manager 初始化成功（无"动态生成"提示）
- [ ] **端口监听**：`ss -tlnp | grep 9000` 显示服务在监听
- [ ] **健康检查**：`curl http://localhost:9000/health` 返回 200
- [ ] **登录测试**：用 admin 账号登录成功，能访问受保护接口
- [ ] **AD 同步测试**（如启用）：管理后台 → AD 同步 → 看到账号列表（证明 SM4 解密成功）
- [ ] **设备连接测试**（如启用）：网络设备 → 选 1 个设备 → 测试连接成功（证明 SM4 解密成功）

---

## 6. 备份与恢复

### 6.1 必须备份的内容

**按部署方案不同，备份内容不同：**

| 内容 | 路径 | 备份方式 |
|------|------|---------|
| **方案 A（明文 .env）** | `/app/szh/.env` | 复制到密码管理器 + 加密 U 盘 |
| **方案 B（EnvironmentFile）** | `/etc/xingran/secrets.env` | 复制到密码管理器 + 加密 U 盘 |
| **方案 C（凭据加密）** | `/etc/credstore/*.cred` (密文) + `/var/lib/systemd/credential.secret` (host key) | host key 单独备份到密码管理器；密文可放任何异地 |
| 应用配置 | `/app/szh/configs/config.yaml` | 已在 git |
| 数据库 | PostgreSQL dump | `pg_dump` 定期 |
| 日志 | `/app/szh/logs/` | journald 永久存储 + 异地 |
| 上传文件 | `/app/szh/uploads/` | rsync 异地 |

### 6.2 密钥丢失应急

**如果 `/etc/xingran/secrets.env` 丢失**：
- ❌ 旧 key 找不到 = SM4 加密的 6 行历史数据**永久不可解密**（用户/设备/AD 凭据）
- 唯一恢复途径：从密码管理器取回
- 如果密码管理器也没存：所有账号需重新输入新密码

**预防措施**：
- `/etc/xingran/secrets.env` 至少 3 处备份：密码管理器 / 加密 U 盘 / 团队共享 vault
- 修改后立即更新所有备份

### 6.3 密钥轮转（SM4 迁移示例）

如果需要更换 SM4_KEY：

1. **生成新 key**：`openssl rand -base64 16`
2. **跑迁移脚本**（本地执行，连接生产 DB）：
   ```bash
   OLD_SM4_KEY="<旧 key>" NEW_SM4_KEY="<新 key>" \
   go run scripts/migrate-sm4-key.go
   ```
3. **更新 secrets.env**：把 `SM4_KEY=` 替换为新值
4. **重启服务**：`sudo systemctl restart szh-backend`
5. **验证**：日志无 [SECURITY WARNING] + 业务功能正常
6. **备份新 key**到所有备份位置

---

## 7. 故障排查

### 7.1 服务启动失败

```bash
# 查看详细错误
sudo journalctl -u szh-backend -n 200 --no-pager

# 常见错误:
# - "SM4_KEY 未配置" → /etc/xingran/secrets.env 中 SM4_KEY= 行为空
# - "数据库连接失败" → DB_HOST/DB_PORT/DB_PASSWORD 不对,或 PG 没启
# - "Redis 连接失败" → REDIS_PASSWORD 不对,或 Redis 没启
# - "JWT secret 是默认值" → JWT_SECRET 未设置或用了仓库默认值
```

### 7.2 性能问题

```bash
# 看资源占用
top -p $(pidof xingran-backend)

# 看连接数
ss -s | grep -i tcp

# 看慢请求
sudo journalctl -u szh-backend --since "5 minutes ago" | grep -i slow
```

### 7.3 SM4 解密失败（业务报错"密码错误"）

可能原因：
1. SM4_KEY 改了但没迁移数据 → 跑 `scripts/migrate-sm4-key.go`
2. 多个实例 SM4_KEY 不一致 → 全部实例用同一 key
3. 数据库被篡改 → 检查 `sys_ad_service_accounts.password_ciphertext` 是否还是密文格式

---

## 8. 升级流程

```bash
# 1. 备份当前版本
sudo cp /app/szh/xingran-backend /app/szh/xingran-backend.bak.$(date +%Y%m%d)

# 2. 备份密钥(理论上 secrets.env 不变)
sudo cp /etc/xingran/secrets.env /etc/xingran/secrets.env.bak.$(date +%Y%m%d)

# 3. 停止服务
sudo systemctl stop szh-backend

# 4. 替换二进制
sudo cp /tmp/xingran-backend-new /app/szh/xingran-backend
sudo chown szh:szh /app/szh/xingran-backend
sudo chmod +x /app/szh/xingran-backend

# 5. 如有 config.yaml 变更,合并
sudo vim /app/szh/configs/config.yaml

# 6. 启动 + 验证
sudo systemctl start szh-backend
sudo systemctl status szh-backend
curl http://localhost:9000/health
```

---

## 9. 相关文档

- [secret-management.md](secret-management.md) — 密钥管理详解
- [架构/安全和认证设计（国密）](../architecture/安全和认证设计（国密）.md) — 国密算法

---

## 10. 变更日志

| 日期 | 变更 | 作者 |
|------|------|------|
| 2026-06-25 | 初版（基于 /app/szh/ 部署路径 + Linux systemd + 环境变量密钥管理） | Claude |
| 2026-06-25 | 修正：删除错误的 `systemctl show-environment` 验证命令；新增三种 env 管理方案（明文 .env / EnvironmentFile / systemd 凭据加密）；加决策树 | Claude |
