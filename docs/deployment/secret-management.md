# 部署期密钥管理指南

> 适用范围：XingRan-Next 所有环境（开发 / 测试 / 预生产 / 生产）
> 目标：杜绝明文密钥入仓、规范部署期密钥注入流程

---

## 1. 核心原则

### 1.1 配置与代码分离（已落地）

```
configs/
├── config.example.yaml          # 通用脱敏模板（提交到仓库）
├── config.prod.example.yaml     # 生产环境模板（提交到仓库）
├── config.yaml                  # 实际配置（开发者本地，不入库）
├── config.dev.yaml              # 旧版本残留（已 git rm --cached）
├── config.prod.yaml             # 旧版本残留（已 git rm --cached）
├── configbs.yaml                # 服务器下载的旧配置（已 git rm --cached）
├── api_metadata.yaml            # 前端 Dashboard 元数据（无敏感信息）
├── agent-config.yaml            # Agent 配置（无敏感信息）
└── oui-vendors.json             # MAC OUI 厂商库（公开数据）
```

**执行情况**：
- `.gitignore` 已配置 `configs/config*.yaml` 排除（example 除外）
- `git rm --cached` 已将历史已追踪的 `config.yaml` / `config.dev.yaml` / `config.prod.yaml` 从仓库索引移除
- 仓库内现在只保留两份 example 模板 + 公开元数据

### 1.2 敏感字段环境变量注入（已实现）

后端启动时通过 `internal/config/config.go:overrideFromEnv` 从环境变量读取关键密钥，**优先级：环境变量 > 配置文件**。

| 配置文件字段 | 环境变量 | 必填 | 说明 |
|------|------|------|------|
| `database.host` | `DB_HOST` | ✓ | 数据库地址 |
| `database.user` | `DB_USER` | ✓ | 数据库用户 |
| `database.password` | `DB_PASSWORD` | ✓ | **生产绝不能留空** |
| `database.dbname` | `DB_NAME` | ✓ | 数据库名 |
| `database.sslmode` | `DB_SSLMODE` | | 建议生产用 `require` 或 `verify-full` |
| `database.port` | `DB_PORT` | | 整型 |
| `cache.host` | `REDIS_HOST` | ✓ | Redis 地址 |
| `cache.port` | `REDIS_PORT` | | 整型 |
| `cache.password` | `REDIS_PASSWORD` | ✓ | **生产绝不能留空** |
| `jwt.secret_key` | `JWT_SECRET` | ✓ | HS256 备选密钥（`use_sm2: true` 时**不被使用**，仅满足启动校验） |
| `jwt.sm2_private_key` | `XINGRAN_JWT_SM2_PRIVATE_KEY` | ✓ | **真正用于 JWT 签名的密钥**；留空会触发"动态生成"分支，重启后旧 token 全部失效 |
| `jwt.sm2_public_key` | `XINGRAN_JWT_SM2_PUBLIC_KEY` | ✓ | 验签公钥；公钥可对外（嵌入前端、APP） |
| `security.sm4_key` | `SM4_KEY` | ✓ | SM4 主密钥（Base64 编码 16 字节） |
| `baidu.map_ak` | `BAIDU_MAP_AK` | | 百度地图 AK |
| `rpa.ai.generator.api_key` | `RPA_AI_GENERATOR_KEY` | | 文本模型 Key |
| `rpa.ai.generator.base_url` | `RPA_AI_GENERATOR_URL` | | OpenAI 兼容 API |
| `rpa.ai.agent.api_key` | `RPA_AI_AGENT_KEY` | | 视觉模型 Key |
| `rpa.ai.agent.base_url` | `RPA_AI_AGENT_URL` | | Vision API |
| `server.port` | `SERVER_PORT` | | 整型 |

### 1.3 SM2 模式与 HS256 模式（重要）

`config.go` 通过 `use_sm2` 开关决定 JWT 签名算法：

| 模式 | 签发 | 验签 | 真正依赖的密钥 | 适用场景 |
|------|------|------|---------------|---------|
| **SM2 模式**（`use_sm2: true`，默认） | `crypto.GenerateTokenWithSM2(claims, j.sm2PrivateKey)` | `crypto.ValidateTokenWithSM2(token, j.sm2PublicKey)` | **SM2 私钥** | 生产环境（国密合规） |
| HS256 模式（`use_sm2: false`） | `jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(secretKey)` | `jwt.ParseWithClaims(token, ...)` | **secret_key** | 仅作非国密场景备选 |

**生产必须用 SM2 模式**。`jwt.secret_key` 在 SM2 模式下不被使用，但代码仍会校验非空（防止误用默认值）——所以仍需配置。

### 1.4 未覆盖的敏感字段（需手动管理）

| 字段 | 风险 | 建议处理 |
|------|------|---------|
| `vdi.tls_skip_verify` | 生产环境保持 true = 强制 MITM 风险 | 当前部署 VDI 无 TLS 保持 true；启用 TLS 后立即改 false |
| `ad.tls_skip_verify` | 同上 | 当前部署 AD 无 TLS 保持 true；启用 LDAPS 后立即改 false |

---

## 2. 部署期密钥生成（一键脚本）

### 2.1 通用密钥（openssl 即可生成）

```bash
#!/bin/bash
# generate-secrets.sh — 一次性生成所有生产密钥（除 SM2 密钥对外）
# 用法：bash generate-secrets.sh > .env.production

set -euo pipefail

echo "# === XingRan-Next 生产环境密钥 ==="
echo "# 生成时间: $(date -Iseconds)"
echo "# 警告: 此文件包含生产密钥，必须用 ansible-vault / KMS 加密保存"
echo

echo "DB_PASSWORD=$(openssl rand -hex 16)"
echo "REDIS_PASSWORD=$(openssl rand -hex 16)"
echo "JWT_SECRET=$(openssl rand -hex 32)"          # SM2 模式下虽不使用,但需满足启动校验
echo "SM4_KEY=$(openssl rand -base64 16)"
echo "BAIDU_MAP_AK=your_baidu_ak_here"              # 替换为实际申请值
echo "SERVER_PORT=9000"
```

### 2.2 SM2 密钥对（**必须独立工具生成，绝不能复用仓库默认值**）

仓库内 `config.yaml` 的 `sm2_private_key: "d8d9a3e6..."` 是**公开已知值**，绝不能用于生产。

**推荐方式：用项目自带的 Go 工具生成（依赖已在 go.mod）**

`scripts/gen-sm2-keys/main.go`：

```go
package main

import (
	"fmt"
	"github.com/xingran-next/xingran-go-backend/pkg/crypto"
)

func main() {
	priv, pub, err := crypto.GenerateKeyPairSM2()
	if err != nil {
		panic(err)
	}
	privHex, _ := crypto.PrivateKeyToHex(priv)
	pubHex, _ := crypto.PublicKeyToHex(pub)
	fmt.Printf("XINGRAN_JWT_SM2_PRIVATE_KEY=%s\n", privHex)
	fmt.Printf("XINGRAN_JWT_SM2_PUBLIC_KEY=%s\n", pubHex)
}
```

使用：

```bash
go run scripts/gen-sm2-keys/main.go >> .env.production
```

**⚠️ 切勿使用"动态生成"**：如果 `sm2_private_key` 配置留空，`internal/core/security/jwt.go:86-92` 会自动生成 SM2 密钥对——**每次重启密钥都换，旧 token 全部失效，所有用户被强制登出**。生产部署必须显式配置。

---

## 3. 部署期安全检查清单

### 3.1 启动前必查（人工）

```bash
# 1. 配置文件就位
[ -f configs/config.yaml ] || { echo "❌ configs/config.yaml 缺失"; exit 1; }

# 2. 敏感字段留空（确认由 env 注入）
grep -E "^\s*password:\s*\"\"$" configs/config.yaml || { echo "⚠️ 数据库/缓存密码未留空"; }

# 3. 关键环境变量已设置
for v in DB_PASSWORD REDIS_PASSWORD JWT_SECRET SM4_KEY \
         XINGRAN_JWT_SM2_PRIVATE_KEY XINGRAN_JWT_SM2_PUBLIC_KEY; do
  [ -n "${!v:-}" ] || { echo "❌ 环境变量 $v 未设置"; exit 1; }
done

# 4. SM2 私钥不是仓库默认值（公开值,任何能读仓库的人都能伪造 JWT）
[ "$XINGRAN_JWT_SM2_PRIVATE_KEY" != "d8d9a3e6b356cf7538cb0fd6a486055c2a621a63a8a095e60a6362874b26508b" ] \
  || { echo "❌ SM2 私钥是仓库默认值，必须重新生成"; exit 1; }

# 5. SM4 不是默认 test-secret
[ "$SM4_KEY" != "dGVzdC1zZWNyZXQxNiEhIQ==" ] \
  || { echo "❌ SM4 密钥是默认 test-secret，必须重新生成"; exit 1; }

# 6. JWT 不是 please-change
[ "$JWT_SECRET" != "xingran-next-secret-key-please-change-in-production" ] \
  || { echo "❌ JWT secret 是仓库默认值"; exit 1; }

echo "✅ 所有安全检查通过"
```

### 3.2 启动后必查（健康检查）

```bash
# 调用 /actuator/health 或自定义 healthz 端点
curl -s http://localhost:9000/health | jq .

# 验证 JWT 解码出来 issuer 是预期值
TOKEN=$(curl -s -X POST http://localhost:9000/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"..."}' | jq -r .data.accessToken)
echo "$TOKEN" | cut -d. -f2 | base64 -d 2>/dev/null | jq .
```

---

## 4. AD 组同步启用方式（重要）

⚠️ **配置文件的 `ad_group_sync` 块仅作参考默认值，代码不直接读取**。

代码读取的是 `sys_config` 表（`internal/services/addomain/group_config.go`）：

| 配置键 | 说明 |
|--------|------|
| `sys.ad.group.sync.enabled` | 主开关（boolean 字符串 `"true"` / `"false"`） |
| `sys.ad.group.sync.cron` | Cron 表达式（默认 `0 */15 * * * *`） |
| `sys.ad.group.member_ou` | MemberOU 路径 |
| `sys.ad.group.auto_create` | 自动创建组 |
| `sys.ad.group.max_concurrent` | 最大并发数 |
| `sys.ad.group.sync.batch_size` | 批量同步大小 |

### 4.1 通过 API 启用（推荐）

```bash
# 用 admin token 调用更新接口
curl -X POST http://localhost:9000/api/v1/system/ad-domain/group-config/update \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "enabled": true,
    "cron": "0 */15 * * * *",
    "member_ou": "OU=Users,DC=corp,DC=example,DC=com",
    "auto_create_groups": true,
    "max_concurrent": 5,
    "sync_batch_size": 100
  }'
```

### 4.2 通过数据库直接 INSERT（应急）

```sql
INSERT INTO sys_config (config_name, config_key, config_value, config_type, create_by, create_time)
VALUES
  ('AD组同步开关', 'sys.ad.group.sync.enabled', 'true', 'yes', 1, NOW()),
  ('AD组同步Cron', 'sys.ad.group.sync.cron', '0 */15 * * * *', 'yes', 1, NOW()),
  ('AD组MemberOU', 'sys.ad.group.member_ou', '', 'yes', 1, NOW()),
  ('AD组自动创建', 'sys.ad.group.auto_create', 'true', 'yes', 1, NOW()),
  ('AD组最大并发', 'sys.ad.group.max_concurrent', '5', 'yes', 1, NOW()),
  ('AD组同步批大小', 'sys.ad.group.sync.batch_size', '100', 'yes', 1, NOW())
ON CONFLICT (config_key) DO UPDATE SET config_value = EXCLUDED.config_value;
```

### 4.3 通过前端页面启用

登录系统 → 系统管理 → 参数管理 → 搜索 `sys.ad.group.*` → 修改值保存

---

## 5. 已泄漏密钥的处置流程

⚠️ **如果发现以下任一情况，按"已泄漏"处理，立即轮转**：

| 泄漏场景 | 处置 |
|---------|------|
| 配置文件被提交到公开仓库 | 立即轮转所有密钥（视为全部失效） |
| 服务器被入侵 | 轮转所有密钥 + 审计访问日志 |
| 团队成员离职 | 轮转其知情的所有密钥 |
| 调试输出含明文 | 轮转被输出的字段 |
| `git log` 历史含敏感值 | 轮转 + `git filter-repo` 清理历史 |

### 5.1 轮转步骤（PostgreSQL）

```sql
-- 1. 生成新密码
-- openssl rand -hex 16

-- 2. 修改 PG 密码
ALTER USER postgres PASSWORD 'NEW_PASSWORD';

-- 3. 修改 Redis 密码（需 CONFIG REWRITE 权限）
CONFIG SET requirepass "NEW_REDIS_PASSWORD";
CONFIG REWRITE;
```

### 5.2 轮转步骤（应用密钥）

```bash
# 1. 生成新 SM4 密钥
NEW_SM4_KEY=$(openssl rand -base64 16)

# 2. 重新加密已加密数据（需要中间层兼容期）
#    - 设备密码
#    - AD 管理员密码
#    - RPA 凭证
#    ⚠️ 直接换 SM4_KEY 会让旧数据无法解密！
#    正确做法：保留旧 key + 新 key 并行，迁移数据后移除旧 key

# 3. 重启应用
systemctl restart xingran-backend
```

### 5.3 历史清理（已 commit 的敏感值）

```bash
# 用 git-filter-repo 清理历史
pip install git-filter-repo
git filter-repo --path configs/config.yaml --invert-paths
git filter-repo --replace-text expressions.txt
# expressions.txt 内容:
#   d8d9a3e6b356cf7538cb0fd6a486055c2a621a63a8a095e60a6362874b26508b==>REDACTED_SM2_PRIVATE_KEY
#   Cpic1234==>REDACTED_DB_PASSWORD
#   )(PO09po==>REDACTED_REDIS_PASSWORD
#   31jIJbkkaQHV6pNXbrQmnOoSrynzalC8==>REDACTED_BAIDU_AK
#   dGVzdC1zZWNyZXQxNiEhIQ====>REDACTED_SM4_KEY
#   xingran-next-secret-key-please-change-in-production==>REDACTED_JWT_SECRET

# 强制推送（破坏性，团队需 rebase）
git push origin --force --all
```

⚠️ 强制推送后所有协作者需要 `git rebase`，且**密钥已泄漏仍需轮转**（git 历史可能被爬取）。

---

## 6. 长期改进方向（P2 路线）

| 阶段 | 方案 | 状态 |
|------|------|------|
| 短期 | 环境变量 + ansible-vault 加密 | ✅ 已落地 |
| 中期 | K8s Secret / Docker Secret | 📋 待评估 |
| 中期 | 阿里云 KMS / 华为 KMS 加密 config | 📋 待评估 |
| 长期 | HashiCorp Vault + Agent Sidecar | 📋 待评估 |
| 长期 | 配置中心 Nacos / Apollo 集中管理 | 📋 待评估 |

---

## 7. 变更日志

| 日期 | 变更 | 提交者 |
|------|------|--------|
| 2026-06-25 | 首次发布：从服务器版 configbs.yaml 倒推整改方案，建立"配置不入仓"流程 | Claude |

---

## 8. 相关文档

- [部署/生产部署指南](deployment.md) — 主部署文档
- [架构/安全和认证设计（国密）](../architecture/安全和认证设计（国密）.md) — 国密算法设计
- `configs/config.example.yaml` — 通用脱敏模板
- `configs/config.prod.example.yaml` — 生产环境模板
