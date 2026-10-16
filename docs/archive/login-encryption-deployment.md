# 登录端点加密部署指南

## 概述

本指南描述如何为 `/system/auth/login` 端点启用 SM2+SM4 混合请求体加密，实现三层加密保护（HTTPS + SM2+SM4 请求体 + SM2 密码字段加密）。

**目标**: 在生产环境中安全地部署登录端点加密功能，确保向后兼容性，提供完整的回滚程序。

**影响范围**: 登录端点 `/system/auth/login` 的请求/响应加密

**部署时间**: 约 1.5 天（开发环境 0.5 天 + 测试环境 0.5 天 + 生产环境 0.5 天）

---

## 部署前检查清单

### 代码完整性
- [ ] Phase 18 所有计划已完成（18-01 至 18-04）
- [ ] 所有测试通过（单元测试、集成测试、E2E 测试）
- [ ] 性能基准测试达标（加密开销 < 50ms）
- [ ] 安全测试通过（重放攻击、时间戳验证）

### 配置验证
- [ ] 后端配置文件已更新（`configs/config.yaml`）
- [ ] 前端配置已更新（`xingran-react-frontend/src/lib/api.ts`）
- [ ] 必需的排除路径已保留（公钥端点、上传端点、验证码端点）
- [ ] `require_encryption: false` 保持向后兼容

### 运维准备
- [ ] 回滚程序已记录并测试
- [ ] 监控和告警已配置
- [ ] 利益相关者已收到部署通知
- [ ] 部署窗口已批准
- [ ] 值班工程师已就位

---

## 部署架构

### 三层加密保护

```
┌─────────────────────────────────────────────────────┐
│ Layer 1: HTTPS/TLS (传输层)                          │
│ - 算法: TLS 1.2/1.3                                 │
│ - 范围: 完整 HTTP 通信通道                           │
│ - 作用: 防止网络窃听                                  │
└─────────────────────────────────────────────────────┘
                      ↓
┌─────────────────────────────────────────────────────┐
│ Layer 2: SM2+SM4 请求体加密 (应用层)                 │
│ - 算法: SM2 (椭圆曲线) + SM4-CBC (对称)             │
│ - 范围: 完整 HTTP 请求体                             │
│ - 作用: 深度防御，防止 TLS 终止攻击、DPI             │
└─────────────────────────────────────────────────────┘
                      ↓
┌─────────────────────────────────────────────────────┐
│ Layer 3: SM2 密码字段加密 (字段级)                   │
│ - 算法: SM2 (椭圆曲线)                               │
│ - 范围: 密码字段                                     │
│ - 作用: 即使请求体加密被破解，密码仍有 SM2 保护       │
└─────────────────────────────────────────────────────┘
```

### 双层 SM2 加密说明

**密码被加密两次吗？**

是的，这是有意设计的深度防御策略：
1. **内层 SM2 (Layer 3)**: 在应用层加密密码字段
2. **外层 SM2 (Layer 2)**: 加密 SM4 密钥（该密钥加密包含内层 SM2 密码的完整请求体）

**理由**:
- 不同的威胁模型（网络监控 vs 应用级攻击）
- 不同的密钥生命周期（每次请求的 SM4 密钥 vs 持久的 SM2 密钥对）
- 符合 GM/T 0024-2014 双重保护建议

---

## 配置变更说明

### 后端配置变更

**文件**: `configs/config.yaml`, `configs/config.dev.yaml`, `configs/config.prod.yaml`

**变更内容**: 从 `request_encryption.exclude_paths` 移除 `/api/v1/system/auth/login`

```yaml
# 变更前
request_encryption:
  enabled: true
  exclude_paths:
    - "/api/v1/system/auth/public-key"
    - "/api/v1/system/auth/test-sm2"
    - "/api/v1/system/auth/login"        # ← 已移除（启用加密）
    - "/api/v1/upload/*"
    - "/api/v1/captcha/*"
  require_encryption: false  # 向后兼容

# 变更后
request_encryption:
  enabled: true
  exclude_paths:
    - "/api/v1/system/auth/public-key"  # 必须保留（循环依赖）
    - "/api/v1/system/auth/test-sm2"   # 必须保留（测试端点）
    # /api/v1/system/auth/login 已移除   ← 启用加密
    - "/api/v1/upload/*"                # 必须保留（文件上传）
    - "/api/vi/captcha/*"               # 必须保留（图片响应）
  require_encryption: false  # 向后兼容，未来可设为 true
```

**关键配置说明**:
- `enabled: true`: 加密功能始终启用（生产环境永远不要设为 false）
- `require_encryption: false`: 支持混合客户端环境（新旧客户端共存）
- 未来可设为 `require_encryption: true` 强制所有客户端使用加密

### 前端配置变更

**文件**: `xingran-react-frontend/src/lib/api.ts`

**变更内容**: 从 `ENCRYPTION_BLACKLIST` 移除 `/system/auth/login`

```typescript
// 变更前
const ENCRYPTION_BLACKLIST: string[] = [
  '/system/auth/public-key',
  '/system/auth/test-sm2',
  '/system/auth/login',                  // ← 已移除（启用加密）
  '/system/auth/captcha',
  '/system/auth/encryption-config',
  '/upload',
];

// 变更后
const ENCRYPTION_BLACKLIST: string[] = [
  '/system/auth/public-key',             // 必须保留（循环依赖）
  '/system/auth/captcha',                // 必须保留（图片响应）
  '/system/auth/encryption-config',      // 必须保留（配置同步）
  '/upload',                             // 必须保留（文件上传）
  // /system/auth/login 已移除            ← 启用加密
];
```

**环境变量**:
```bash
# .env.development, .env.production
VITE_ENABLE_REQUEST_ENCRYPTION=true  # 生产环境永远不要设为 false
```

---

## 部署步骤

### 阶段 1: 开发环境验证（0.5 天）

**目标**: 在开发环境中验证加密功能正常工作

**步骤**:

1. **更新后端配置**:
   ```bash
   # 编辑 configs/config.dev.yaml
   # 从 exclude_paths 移除 /api/v1/system/auth/login
   vim configs/config.dev.yaml
   ```

2. **更新前端配置**:
   ```bash
   # 编辑 xingran-react-frontend/src/lib/api.ts
   # 从 ENCRYPTION_BLACKLIST 移除 /system/auth/login
   vim xingran-react-frontend/src/lib/api.ts
   ```

3. **启动本地开发服务器**:
   ```bash
   # 后端
   go run cmd/main.go

   # 前端
   cd xingran-react-frontend
   npm run dev
   ```

4. **验证加密功能**:
   - 打开浏览器 DevTools → Network 标签
   - 登录应用程序
   - 验证请求头包含 `X-Request-Encrypted: true`
   - 验证请求体结构：
     ```json
     {
       "encrypted": true,
       "data": "SM4加密后的请求体（Base64）",
       "sm4Key": "SM2加密后的SM4密钥（Base64）",
       "iv": "SM4初始化向量（Base64）",
       "timestamp": 1735648800,
       "nonce": "随机32字符十六进制"
     }
     ```
   - 检查后端日志显示 "请求解密成功" 消息

5. **测试向后兼容性**:
   ```bash
   # 使用 curl 或 Postman 发送明文登录请求
   curl -X POST http://localhost:9000/api/v1/system/auth/login \
     -H "Content-Type: application/json" \
     -d '{"username":"test","password":"test123"}'

   # 验证登录仍然成功（require_encryption: false）
   ```

6. **运行测试套件**:
   ```bash
   # 后端单元测试
   go test ./internal/api/v1/ -run TestAuth

   # 前端单元测试
   cd xingran-react-frontend
   npm run test

   # 集成测试
   go test ./tests/integration/ -run TestLoginEncryption
   ```

**通过标准**:
- [ ] 所有测试通过
- [ ] 加密请求成功（Network 面板验证）
- [ ] 明文请求仍然成功（向后兼容）
- [ ] 无解密错误日志

---

### 阶段 2: 测试环境部署（0.5 天）

**目标**: 在测试环境中完整验证加密功能

**步骤**:

1. **部署后端配置变更**:
   ```bash
   # 更新 configs/config.prod.yaml（测试环境使用生产配置）
   vim configs/config.prod.yaml

   # 重新构建
   go build -o xingran-backend ./cmd/main.go

   # 部署到测试服务器
   scp xingran-backend user@test-server:/opt/xingran/
   ssh user@test-server "systemctl restart xingran-backend"
   ```

2. **部署前端变更**:
   ```bash
   cd xingran-react-frontend
   npm run build

   # 部署到测试服务器
   scp -r dist/* user@test-server:/var/www/xingran/

   # 清除 CDN 缓存（如果使用）
   ssh user@test-server "invalidate-cache /var/www/xingran/"
   ```

3. **运行冒烟测试**:
   ```bash
   # E2E 测试
   cd xingran-react-frontend
   npm run test:e2e

   # 集成测试（在测试服务器上）
   ssh user@test-server "cd /opt/xingran && go test ./tests/integration/..."
   ```

4. **手动测试场景**:
   - [ ] 使用有效凭证登录
   - [ ] 使用无效凭证登录（密码错误）
   - [ ] 使用无效凭证登录（用户不存在）
   - [ ] 密码修改流程
   - [ ] 密码重置流程
   - [ ] 验证所有登录场景正常工作

5. **监控指标**:
   - 访问 Prometheus 仪表板
   - 检查解密指标：
     - `request_decryption_duration_ms{endpoint="/system/auth/login"}`
     - P99 延迟应 < 350ms
   - 检查错误率：
     - `request_decryption_failures_total / http_requests_total`
     - 错误率应 < 0.1%
   - 检查 Nonce 存储大小：
     - `nonce_storage_size`
     - 应 < 10000 个条目

6. **性能验证**:
   ```bash
   # 运行性能基准测试
   go test -bench=BenchmarkLoginEncryption ./tests/benchmark/

   # 验证加密开销 < 50ms
   ```

**通过标准**:
- [ ] 所有自动测试通过
- [ ] 所有手动测试场景正常
- [ ] 错误率 < 0.1%
- [ ] P99 延迟 < 350ms
- [ ] 加密开销 < 50ms
- [ ] 利益相关者批准进入生产环境

---

### 阶段 3: 生产环境部署（0.5 天）

**目标**: 安全地将加密功能部署到生产环境

**前提条件**:
- 测试环境验证通过
- 性能指标可接受
- 利益相关者批准
- 部署窗口已确认

**步骤**:

1. **部署前检查**:
   ```bash
   # 验证测试环境结果
   # 确认所有测试通过
   # 确认性能指标可接受
   # 获取生产部署批准
   ```

2. **备份当前配置**:
   ```bash
   # 备份生产配置
   ssh user@prod-server "cp /opt/xingran/configs/config.prod.yaml /opt/xingran/configs/config.prod.yaml.backup"
   ```

3. **部署后端变更**:
   ```bash
   # 更新配置（从 exclude_paths 移除 /api/v1/system/auth/login）
   vim configs/config.prod.yaml

   # 重新构建
   go build -o xingran-backend ./cmd/main.go

   # 部署到生产服务器
   scp xingran-backend user@prod-server:/opt/xingran/

   # 滚动重启后端服务（零停机）
   ssh user@prod-server "./scripts/rolling-restart.sh"
   ```

4. **部署前端变更**:
   ```bash
   cd xingran-react-frontend
   npm run build

   # 部署到生产 CDN
   ./scripts/deploy-frontend.sh

   # 使 CDN 缓存失效
   ./scripts/invalidate-cache.sh
   ```

5. **部署后验证**:
   ```bash
   # 监控错误日志
   ssh user@prod-server "tail -f /var/log/xingran-backend/app.log | grep -i 'decrypt\|error'"

   # 检查指标仪表板
   # 验证错误率 < 0.1%
   # 验证 P99 延迟 < 350ms
   # 验证解密成功率 > 99.9%
   ```

6. **生产环境冒烟测试**:
   - 使用测试账号登录
   - 验证 `X-Request-Encrypted` 请求头存在
   - 检查后端日志显示成功解密
   - 验证错误率无上升
   - 验证用户投诉无增加

7. **监控 30 分钟**:
   - 持续监控错误率
   - 持续监控性能指标
   - 准备回滚（如果出现问题）

**通过标准**:
- [ ] 部署无错误
- [ ] 冒烟测试通过
- [ ] 错误率 < 0.1%（30 分钟稳定）
- [ ] P99 延迟 < 350ms
- [ ] 无用户投诉
- [ ] 监控告警无触发

---

## 回滚程序

### 紧急回滚（5 分钟）

**触发条件**: 关键问题阻止用户登录

**步骤**:

1. **回滚后端配置**:
   ```bash
   # 将 /api/v1/system/auth/login 添加回 exclude_paths
   ssh user@prod-server "vim /opt/xingran/configs/config.prod.yaml"

   # 重启后端服务
   ssh user@prod-server "systemctl restart xingran-backend"
   ```

2. **回滚前端变更**:
   ```bash
   # 将 /system/auth/login 添加回 ENCRYPTION_BLACKLIST
   vim xingran-react-frontend/src/lib/api.ts

   # 重新构建和部署
   npm run build
   ./scripts/deploy-frontend.sh

   # 清除 CDN 缓存
   ./scripts/invalidate-cache.sh
   ```

3. **验证回滚**:
   - 测试登录功能
   - 检查错误率恢复到基线
   - 验证用户投诉已解决

**预期结果**: 5 分钟内登录功能恢复正常

---

### 快速回滚（15 分钟）

**触发条件**: 性能下降或非关键问题

**步骤**:

1. **Git 回滚配置变更**:
   ```bash
   # 查找需要回滚的提交
   git log --oneline | grep "18-01"

   # 回滚后端变更
   git revert <commit-hash>
   go build -o xingran-backend ./cmd/main.go
   systemctl restart xingran-backend
   ```

2. **Git 回滚前端变更**:
   ```bash
   # 回滚前端变更
   git revert <commit-hash>
   npm run build
   ./scripts/deploy-frontend.sh
   ```

3. **清除 CDN 缓存**:
   ```bash
   ./scripts/invalidate-cache.sh
   ```

**预期结果**: 15 分钟内完全回滚到部署前状态

---

## 监控和告警

### 关键指标

#### 1. 解密性能

**指标**: `request_decryption_duration_ms{endpoint="/system/auth/login"}`

**目标**:
- P50: < 100ms
- P95: < 250ms
- P99: < 350ms

**Prometheus 查询**:
```promql
# P99 延迟
histogram_quantile(0.99, request_decryption_duration_ms{endpoint="/system/auth/login"})

# 平均延迟
rate(request_decryption_duration_sum{endpoint="/system/auth/login"}[5m]) /
rate(request_decryption_duration_count{endpoint="/system/auth/login"}[5m])
```

#### 2. 解密成功率

**指标**: `request_decryption_failures_total / http_requests_total`

**目标**: 失败率 < 0.1%

**Prometheus 查询**:
```promql
# 失败率
rate(request_decryption_failures_total{endpoint="/system/auth/login"}[5m]) /
rate(http_requests_total{endpoint="/system/auth/login"}[5m])
```

#### 3. Nonce 存储大小

**指标**: `nonce_storage_size`

**告警阈值**: > 10000 个条目

**Prometheus 查询**:
```promql
# 当前 Nonce 数量
nonce_storage_size

# Nonce 增长速率
rate(nonce_storage_size[5m])
```

#### 4. CPU 使用率

**指标**: `process_cpu_usage{service="xingran-backend"}`

**告警阈值**: > 80% 持续

**Prometheus 查询**:
```promql
# CPU 使用率
rate(process_cpu_seconds_total{service="xingran-backend"}[5m]) * 100
```

### 告警规则

**Grafana Alert Rules 配置**:

```yaml
# slow_decryption.yaml
groups:
  - name: login_encryption
    rules:
      # 告警: 解密速度过慢
      - alert: SlowDecryption
        expr: |
          histogram_quantile(
            0.99,
            request_decryption_duration_ms{endpoint="/system/auth/login"}
          ) > 500
        for: 5m
        labels:
          severity: warning
          component: login_encryption
        annotations:
          summary: "登录端点解密速度过慢"
          description: "P99 解密延迟 {{ $value }}ms 超过阈值 500ms"

      # 告警: 解密失败率过高
      - alert: HighDecryptionFailureRate
        expr: |
          (
            rate(request_decryption_failures_total{endpoint="/system/auth/login"}[5m])
            /
            rate(http_requests_total{endpoint="/system/auth/login"}[5m])
          ) > 0.01
        for: 5m
        labels:
          severity: critical
          component: login_encryption
        annotations:
          summary: "登录端点解密失败率过高"
          description: "失败率 {{ $value | humanizePercentage }} 超过阈值 1%"

      # 告警: Nonce 存储过大
      - alert: LargeNonceStorage
        expr: nonce_storage_size > 10000
        for: 5m
        labels:
          severity: warning
          component: login_encryption
        annotations:
          summary: "Nonce 存储过大"
          description: "当前存储 {{ $value }} 个 nonce，可能存在内存泄漏"

      # 告警: CPU 使用率过高
      - alert: HighCPUUsage
        expr: |
          rate(process_cpu_seconds_total{service="xingran-backend"}[5m]) * 100 > 80
        for: 10m
        labels:
          severity: warning
          component: login_encryption
        annotations:
          summary: "后端服务 CPU 使用率过高"
          description: "CPU 使用率 {{ $value }}% 超过阈值 80%"
```

### Grafana 仪表板配置

**推荐面板**:

1. **解密延迟分布**:
   - 类型: Histogram
   - 查询: `histogram_quantile(0.99, request_decryption_duration_ms{endpoint="/system/auth/login"})`
   - 阈值线: 350ms

2. **解密失败率**:
   - 类型: Stat
   - 查询: `rate(request_decryption_failures_total{endpoint="/system/auth/login"}[5m]) / rate(http_requests_total{endpoint="/system/auth/login"}[5m])`
   - 阈值: 0.1%

3. **Nonce 存储趋势**:
   - 类型: Graph
   - 查询: `nonce_storage_size`
   - 告警线: 10000

4. **CPU 使用率趋势**:
   - 类型: Graph
   - 查询: `rate(process_cpu_seconds_total{service="xingran-backend"}[5m]) * 100`
   - 告警线: 80%

---

## 故障排查

### 问题 1: 登录失败，显示 "解密失败" 错误

**症状**:
- 用户无法登录
- 后端日志显示 "解密失败" 或 "SM2 解密失败"
- 错误率上升

**诊断步骤**:
1. 检查后端日志中的具体错误信息：
   ```bash
   tail -f /var/log/xingran-backend/app.log | grep -i "decrypt\|error"
   ```
2. 验证 SM2 密钥对是否正确配置：
   ```bash
   # 检查公钥端点是否正常
   curl http://localhost:9000/api/v1/system/auth/public-key
   ```
3. 检查时间戳同步（客户端 vs 服务器）：
   ```bash
   # 服务器时间
   date
   # 客户端时间（浏览器）
   # JavaScript: new Date().getTime() / 1000
   ```

**解决方案**:
- **时间戳问题**: 检查客户端 NTP 同步，确保时间差 < 300 秒
- **SM2 密钥问题**: 验证密钥对未被轮换，重启后端服务重新生成密钥
- **Nonce 问题**: 检查 Nonce 存储是否存在内存泄漏，重启服务清除存储

---

### 问题 2: 部署后 CPU 使用率过高

**症状**:
- CPU 使用率 > 80%
- 后端服务响应变慢
- 性能指标下降

**诊断步骤**:
1. 检查 Prometheus CPU 指标：
   ```promql
   rate(process_cpu_seconds_total{service="xingran-backend"}[5m]) * 100
   ```
2. 检查解密延迟指标：
   ```promql
   histogram_quantile(0.99, request_decryption_duration_ms{endpoint="/system/auth/login"})
   ```
3. 检查登录请求速率：
   ```promql
   rate(http_requests_total{endpoint="/system/auth/login"}[5m])
   ```

**解决方案**:
- **DDoS 攻击**: 实施速率限制，封禁恶意 IP
- **加密开销**: 考虑 Web Worker 优化前端加密
- **内存泄漏**: 重启服务并调查 Nonce 存储

---

### 问题 3: 旧客户端无法登录

**症状**:
- 部分用户报告无法登录
- 这些用户使用旧版本客户端
- 错误日志显示 "请求体格式错误"

**诊断步骤**:
1. 检查 `require_encryption` 配置：
   ```bash
   grep require_encryption /opt/xingran/configs/config.prod.yaml
   ```
2. 验证前端 CDN 缓存：
   ```bash
   # 检查缓存是否已清除
   curl -I https://cdn.example.com/api.js
   ```

**解决方案**:
- **配置错误**: 设置 `require_encryption: false` 保持向后兼容
- **CDN 缓存**: 清除 CDN 缓存强制前端更新
- **用户通知**: 通知用户清除浏览器缓存

---

## 部署后检查清单

### 功能验证
- [ ] 所有冒烟测试通过
- [ ] 加密请求成功（Network 面板验证）
- [ ] 明文请求仍然成功（向后兼容）
- [ ] 无解密错误日志

### 性能验证
- [ ] 错误率在可接受范围内（< 0.1%）
- [ ] 性能指标达标（P99 < 350ms）
- [ ] 加密开销 < 50ms
- [ ] CPU 使用率正常（< 80%）

### 监控验证
- [ ] 监控和告警正常工作
- [ ] Prometheus 指标正常采集
- [ ] Grafana 仪表板数据正确
- [ ] 无告警触发

### 用户体验
- [ ] 无用户投诉登录问题
- [ ] 支持团队未报告异常
- [ ] 应用商店评论无负面反馈

### 文档验证
- [ ] 文档已更新
- [ ] 回滚程序已测试
- [ ] 利益相关者已通知
- [ ] 部署记录已归档

---

## 相关参考

### 项目文档
- **研究文档**: `.planning/milestones/v1.8-phases/18-login-endpoint-encryption/RESEARCH.md`
- **实现计划**: `.planning/milestones/v1.8-phases/18-login-endpoint-encryption/18-*.md`
- **安全文档**: `docs/security/login-encryption-security.md`
- **架构文档**: `docs/安全和认证设计（国密）.md`

### 配置文件
- **后端配置**: `configs/config.yaml`, `configs/config.prod.yaml`
- **前端配置**: `xingran-react-frontend/src/lib/api.ts`

### 代码实现
- **请求加密**: `pkg/crypto/request_encryption.go`
- **请求解密中间件**: `pkg/middleware/request_decryption.go`
- **登录处理器**: `internal/api/v1/auth.go`

### 测试文件
- **单元测试**: `internal/api/v1/auth_test.go`
- **集成测试**: `tests/integration/login_encryption_test.go`
- **E2E 测试**: `tests/e2e/login-encryption.spec.ts`
- **性能测试**: `tests/benchmark/login_encryption_bench_test.go`

---

## 附录

### A. 时间戳验证窗口

**当前配置**: 300 秒（5 分钟）

**原因**: 容忍客户端和服务器之间的时间偏差

**验证**:
```javascript
// 前端：当前 Unix 时间戳（秒）
const timestamp = Math.floor(Date.now() / 1000);

// 后端：验证时间戳在窗口内
if Math.Abs(serverTime - timestamp) > 300 {
    return error("时间戳超出允许范围")
}
```

### B. Nonce 格式

**格式**: 32 字符十六进制字符串

**生成**:
```javascript
// 前端：生成随机 nonce
const nonce = Array.from(crypto.getRandomValues(new Uint8Array(16)))
    .map(b => b.toString(16).padStart(2, '0'))
    .join('');
```

**验证**:
```go
// 后端：验证 nonce 格式
if len(nonce) != 32 || !isValidHex(nonce) {
    return error("Nonce 格式无效")
}
```

### C. SM2 公钥分发

**端点**: `/api/v1/system/auth/public-key`

**响应格式**:
```json
{
    "code": 0,
    "message": "success",
    "data": {
        "publicKey": "SM2公钥（PEM格式）",
        "timestamp": 1735648800
    },
    "timestamp": 1735648800,
    "request_id": "uuid"
}
```

**缓存策略**: 前端缓存 5 分钟（TTL）

---

**文档版本**: 1.0
**最后更新**: 2026-05-21
**维护者**: XingRan-Next 开发团队
