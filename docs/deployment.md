# 生产环境部署指南

> 适用范围：RuoYi-Next 后端服务的生产部署
> 部署路径：`/app/szh/`
> OS：Linux（systemd 服务管理）

---

## 1. 目录结构

```
/app/szh/
├── ruoyi-backend              # Go 编译产物（构建自 cmd/main.go）
├── ruoyi-react-frontend/      # 前端构建产物（可选，embed 构建已嵌入二进制）
├── configs/
│   └── config.yaml            # 应用配置（非敏感，由 git 管理）
├── logs/                      # 应用日志（lumberjack 轮转）
├── uploads/                   # 用户上传文件
└── secrets.env → /etc/ruoyi/secrets.env  # 软链接或独立存放敏感 env
```

> **重要**：`configs/config.yaml` **不含任何密钥**——所有敏感值通过系统环境变量注入（见 §3）。

---

## 2. 构建产物

### 2.1 嵌入式构建（前端打包进二进制）

```bash
# 在构建机（Windows）执行
scripts\build-linux.bat

# 产物
ruoyi-backend-embedded-linux
```

### 2.2 部署到服务器

```bash
# 在生产服务器
sudo useradd -r -s /sbin/nologin szh  # 创建专用账户（如已存在则跳过）
sudo mkdir -p /app/szh/{configs,logs,uploads}
sudo chown -R szh:szh /app/szh

# 上传二进制
scp ruoyi-backend-embedded-linux szh@10.62.10.34:/app/szh/ruoyi-backend
# 或 rsync / 堡垒机中转
sudo chmod +x /app/szh/ruoyi-backend
sudo chown szh:szh /app/szh/ruoyi-backend
```

---

## 3. 敏感环境变量（关键）

### 3.1 创建 secrets 文件

```bash
sudo mkdir -p /etc/ruoyi
sudo vim /etc/ruoyi/secrets.env
```

填入以下内容（**密钥值从密码管理器复制，不要 echo 到屏幕**）：

```bash
# /etc/ruoyi/secrets.env
# 权限: 600, owner: root:root
# 修改后: sudo systemctl daemon-reload && sudo systemctl restart szh-backend

# === 数据库 ===
DB_HOST=10.62.10.34
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=<从密码管理器>
DB_NAME=ruoyi
DB_SSLMODE=disable

# === Redis ===
REDIS_HOST=10.62.10.34
REDIS_PORT=6379
REDIS_PASSWORD=<从密码管理器>
REDIS_DB=0

# === 服务器 ===
SERVER_HOST=0.0.0.0
SERVER_PORT=9000
SERVER_MODE=release

# === JWT ===
JWT_SECRET=<从密码管理器>
JWT_SM2_PRIVATE_KEY=<从密码管理器>
JWT_SM2_PUBLIC_KEY=<从密码管理器>

# === SM4（数据库加密密钥） ===
SM4_KEY=<从密码管理器>

# === 第三方 API ===
BAIDU_MAP_AK=<从百度地图开放平台>
RPA_AI_GENERATOR_KEY=
RPA_AI_GENERATOR_URL=
RPA_AI_AGENT_KEY=
RPA_AI_AGENT_URL=
```

### 3.2 设权限

```bash
sudo chmod 600 /etc/ruoyi/secrets.env
sudo chown root:root /etc/ruoyi/secrets.env
ls -la /etc/ruoyi/secrets.env
# 验证: -rw------- 1 root root
```

---

## 4. systemd 服务

### 4.1 service 文件

`/etc/systemd/system/szh-backend.service`：

```ini
[Unit]
Description=SZH RuoYi-Next Backend
After=network.target postgresql.service redis-server.service
Wants=postgresql.service redis-server.service

[Service]
Type=simple
User=szh
Group=szh
WorkingDirectory=/app/szh
ExecStart=/app/szh/ruoyi-backend
EnvironmentFile=/etc/ruoyi/secrets.env
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
- [ ] **环境变量加载**：`systemctl show szh-backend | grep EnvironmentFile` 显示 `/etc/ruoyi/secrets.env`
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

| 内容 | 路径 | 备份方式 |
|------|------|---------|
| **环境变量** | `/etc/ruoyi/secrets.env` | 复制到密码管理器 + 加密 U 盘 |
| 应用配置 | `/app/szh/configs/config.yaml` | 已在 git |
| 数据库 | PostgreSQL dump | `pg_dump` 定期 |
| 日志 | `/app/szh/logs/` | journald 永久存储 + 异地 |
| 上传文件 | `/app/szh/uploads/` | rsync 异地 |

### 6.2 密钥丢失应急

**如果 `/etc/ruoyi/secrets.env` 丢失**：
- ❌ 旧 key 找不到 = SM4 加密的 6 行历史数据**永久不可解密**（用户/设备/AD 凭据）
- 唯一恢复途径：从密码管理器取回
- 如果密码管理器也没存：所有账号需重新输入新密码

**预防措施**：
- `/etc/ruoyi/secrets.env` 至少 3 处备份：密码管理器 / 加密 U 盘 / 团队共享 vault
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
# - "SM4_KEY 未配置" → /etc/ruoyi/secrets.env 中 SM4_KEY= 行为空
# - "数据库连接失败" → DB_HOST/DB_PORT/DB_PASSWORD 不对,或 PG 没启
# - "Redis 连接失败" → REDIS_PASSWORD 不对,或 Redis 没启
# - "JWT secret 是默认值" → JWT_SECRET 未设置或用了仓库默认值
```

### 7.2 性能问题

```bash
# 看资源占用
top -p $(pidof ruoyi-backend)

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
sudo cp /app/szh/ruoyi-backend /app/szh/ruoyi-backend.bak.$(date +%Y%m%d)

# 2. 备份密钥(理论上 secrets.env 不变)
sudo cp /etc/ruoyi/secrets.env /etc/ruoyi/secrets.env.bak.$(date +%Y%m%d)

# 3. 停止服务
sudo systemctl stop szh-backend

# 4. 替换二进制
sudo cp /tmp/ruoyi-backend-new /app/szh/ruoyi-backend
sudo chown szh:szh /app/szh/ruoyi-backend
sudo chmod +x /app/szh/ruoyi-backend

# 5. 如有 config.yaml 变更,合并
sudo vim /app/szh/configs/config.yaml

# 6. 启动 + 验证
sudo systemctl start szh-backend
sudo systemctl status szh-backend
curl http://localhost:9000/health
```

---

## 9. 相关文档

- [部署和运维文档.md](部署和运维文档.md) — 总览
- [安全和认证设计（国密）.md](安全和认证设计（国密）.md) — 国密算法
- [secret-management.md](deployment/secret-management.md) — 密钥管理详解
- [deployment/login-encryption-deployment.md](deployment/login-encryption-deployment.md) — SM2+SM4 加密部署

---

## 10. 变更日志

| 日期 | 变更 | 作者 |
|------|------|------|
| 2026-06-25 | 初版（基于 /app/szh/ 部署路径 + Linux systemd + 环境变量密钥管理） | Claude |
