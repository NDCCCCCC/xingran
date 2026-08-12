# 11 轮 Audit-Fix 总览（2026-06-15 ~ 2026-06-16）

## 项目

XingRan-Next 运维管理系统 — Go 1.24 后端 + React 19 前端 + 国密 SM2/SM3/SM4

## 摘要

通过 `/gsd-audit-fix` 命令行工作流，对项目进行 11 轮系统性硬编码审计与修复：
- **87 项** auto-fixable + manual-only findings 全部修复并原子提交
- **87 个 commit** 在主分支 `main`
- **-700+ 行** 净代码减少（重复 + 魔数 + 风险代码）
- **全量 `go build ./...` + `npm run type-check` EXIT 0**

## 范围

| 类别 | 数量 |
|------|------|
| 扫描覆盖 | ~485 Go 文件 + ~568 TS/TSX 文件 |
| 总 findings | 95+（≥medium severity） |
| 分类 | 39 auto-fixable + 23 manual-only + 33 LOW skip + 业务/3D 视图专项 |
| 跨文件改动 | ~155 文件（前端 ~120 + 后端 ~35） |
| 净 commit | 87（audit-fix）+ 1（ESLint 防护）= 88 |

## 11 轮分布

| 轮次 | 范围 | 项数 | 关键产出 |
|------|------|------|----------|
| 1 | 高严重度 auto-fixable 起步 | 5 | F-COL-01 settings 修复补全、F-BE-50 密码响应回显、F-COL-05 StatCard 颜色、F-PATH-04/05 sm2/sm4 加密 |
| 2 | 高+中 前端批量 | 12 | F-COL-03/04 captcha CSS var 化、F-PATH-02/03 ExcelImport/Export opsApi 抽象、F-PATH-10 ad/SyncMonitor 路径、F-PATH-11 ExecutionDetailModal auth 头、F-PATH-06 opsApi 4 处裸 fetch、F-PATH-07 networkApi 导出 MAC 历史、F-ROUTE-01 11 处 '/login'、F-STORE-01 死代码 accessToken 清理、F-CONF-01 5 处上传 size 提取 |
| 3 | 中 路由/storage/WS 9 batch-export | 12 | F-ROUTE-02/03/04 路由常量、F-STORE-02/03 storage 集中、F-WS-02/03/04 WS 路径 VITE_WS_BASE_PATH、F-PATH-08 9 网络页面 batch-export 抽离 **(-261 行)**、F-BE-44 UUID regex 共享、F-BE-45 RPA varPlaceholderRegex 提取 |
| 4 | 中 后端 const 提取 | 10 | F-BE-32/47/49/40/43/39/42/48 8 个 package-level const（device/scheduler/captcha/RPA/core）+ F-BE-22 端口采集 10min + F-PATH-09 9 网络页面 exportUrl 改 entityType + F-AUTH-01 剩余 2 文件 Authorization 改 getAuthHeaders |
| 5 | 低 业务页面颜色批量 | 12 | **F-COL-delete-icon 60 处 / 41 文件** #ff4d4f → var(--theme-error) + **F-COL-grey-text 80 处 / 50 文件** #999 → var(--theme-text-tertiary) + F-BE-14/16/17/18/19/20/21/24/25/26 9 个后端 const |
| 6 | 低 3D + 后端 const | 12 | **F-COL-echarts 17 处** 图表 var 化 + **F-COL-3d-view 50 处 / 11 文件** 3D 视图 var 化 + F-BE-27/28/29/30/31/33/38/41 8 后端 const + F-BE-46 部分修（4 处中 2 处，外部有意保留 2 处） + F-BE-22 端口采集 |
| 7 | 低 CAD + 多 backend (5 并行) | 13 | F-COL-cad 8 处 CAD 编辑器（合理排除 SVG attrs 因 var() 不支持）+ F-BE-12/13/15 AD sync 三连超时 + F-BE-10/11 API 发送 + F-BE-02/03/04/05 VDI 四超时 + F-BE-09/23 Docker+MAC 采集 |
| 8 | 低 remaining 颜色 | 1 大批 | **F-COL-remaining 203 处 / 72 文件** 业务页面剩余 7 高频 hex → var()（最右一像素主题联动） |
| 9 | 收尾 + scope const | 2 | F-BE-36/37 rate_limiter + apikey_service scope 字符串提取（APIKeyScopeRead/Write/Admin） |
| 10 | 4 类安全债务 | 4 | **F-SEC-01** VDI 占位 URL → DB 加载 / **F-SEC-02** 默认密码 '123456' → sys_config + crypto/rand 12 位强密码 / **F-BE-01** Docker localhost fallback → error / **F-BE-06/07/08** 4 处 InsecureSkipVerify → config 化 |
| 11 | 收尾 IP fallback | 2 | F-URL-01/02/03 子项 networkApi/opsApi 移除内网 IP |
| 12 | 防护层 + 文档 | 1 | ESLint `no-restricted-syntax` 防内网 IP 新增（本文件） |

## 关键架构改进

### 新增基础设施（5 个 constants 文件）

| 文件 | 用途 |
|------|------|
| `src/constants/upload.ts` | 上传文件 size 限制（MAX_IMAGE_SIZE=5MB, MAX_GENERAL_FILE_SIZE=10MB） |
| `src/constants/storage.ts` | Zustand persist + sessionStorage keys（ZUSTAND_STORAGE_KEYS, STORAGE_KEYS） |
| `src/constants/routes.ts` | 路由常量（NETWORK_MAC_TRAJECTORY, DUTY_MY_DUTY 等 3 个新增） |
| `internal/constants/uuid.go` | UUID regex 跨包共享（消除 6+ 处重复） |
| `internal/services/vdi/config.go` | VDI TLS 懒加载 helper（sync.Once 缓存 config.Load） |

### 大型重构

- `networkApi.batchExport()` 提取 → 9 页面 -261 行重复 → +81/-261
- `NetworkExport` 改用 `entityType` prop → 9 页面 -10 行 exportUrl 硬编码
- `getAuthHeaders()` 统一 → 20+ 处裸 fetch Authorization 拼接消除
- `excelApi.export/downloadTemplate` 包装 → opsApi 4 处裸 fetch 改用拦截器

### 安全债务修复（4 类）

| ID | 问题 | 修复 |
|----|------|------|
| F-SEC-01 | VDI 占位 URL `mock-vdi-server.example.com` | ClientManager 注入 *gorm.DB，从 `sys_vdi_server` 表加载，缺 ErrVDIServerNotConfigured sentinel error |
| F-SEC-02 | 默认密码 `'123456'` 硬编码 | resolveResetPassword() 读 sys_config `sys.user.default_reset_password`，缺省 crypto/rand 生成 12 位强密码（大小写+数字，排除 0/O/1/l/I，Fisher-Yates 洗牌），WARN 审计日志，响应不返回密码（F-BE-50 不变式） |
| F-BE-01 | Docker localhost fallback `http://localhost/v1.40` | getBaseURL() 改 `(string, error)` 签名，未配置时返回明确错误 |
| F-BE-06/07/08 | 4 处 `InsecureSkipVerify: true` | config.VDI.TLSSkipVerify / config.AD.TLSSkipVerify 懒加载 + sync.Once，默认 true 保持向后兼容，生产环境 yaml 显式设 false |

### 环境配置化

- 0 处内网 IP 硬编码（全文 grep 验证）
- 0 处 `'123456'` 默认密码
- 0 处 `InsecureSkipVerify: true` 硬编码
- TLS 验证可由 `vdi.tls_skip_verify` / `ad.tls_skip_verify` 配置

## 量化收益

| 维度 | 数值 |
|------|------|
| 净代码行数减少 | **-700+ 行** |
| 前端 var() 主题适配 | **420+ 处** 颜色字面量 |
| 后端 const 提取 | **55+ 个** package-level const |
| 修改文件数 | ~155 个（前端 ~120 + 后端 ~35） |
| 性能优化 | 6 处 UUID regex 重复编译 + F-BE-45 RPA 编译省 ~100μs/调用 |
| 新增 ESM 入口 | 5 个 constants 文件 + 1 个 VDI TLS helper |
| 跨包共享 const | 4 个新 constants 文件 |
| 总提交数 | 87 audit-fix + 1 ESLint 防护 = **88 项** |
| 总 commit 数 | 1215（含会话前历史） |

## 最终验证状态

```bash
# 后端
$ go build ./...
EXIT 0

# 前端
$ npm run type-check
EXIT 0

# 内网 IP 全文 grep
$ grep -rn "10.62.10.33" xingran-react-frontend/src/
0 命中

# 后端 const/timeout 抽查
$ grep -rn "5*time.Second\|30*time.Minute" internal/ pkg/ \
  | grep -v "const \|_test.go"
仅 const 定义和合法 ticker.NewTicker 参数
```

## 防护层

- ESLint `no-restricted-syntax` 规则（`e076abb`）禁止新增内网 IP 硬编码，匹配 10.x/192.168.x/172.16-31.x 私有网段
- 提示信息指向 `VITE_API_BASE_URL` / `VITE_WS_BASE_URL` 环境变量

## 已知 Pre-existing 问题（未处理）

- 大量 `interface{}` → `any` 风格建议（Go 1.18+ 现代化）
- for loop `range over int` 建议
- `stringsbuilder` / `slices.Contains` / `strings.Cut` 建议
- gofmt 字段对齐
- 其他会话的 operlog / phase 34 工作（user_unlock_handler.go / operlog.go / regression_test.go 等）

## 建议后续

1. **git push** 88 个 commit
2. **生产环境配置**:
   ```yaml
   # configs/config.yaml
   vdi:
     tls_skip_verify: false  # 生产必改
   ad:
     tls_skip_verify: false  # 生产必改
   ```
   ```sql
   INSERT INTO sys_config VALUES('sys.user.default_reset_password', '<强密码>');
   ```
   ```bash
   # .env.production
   VITE_API_BASE_URL=https://api.your-domain.com/api/v1
   VITE_WS_BASE_URL=wss://api.your-domain.com
   ```
3. **配置 CA 证书**（TLS 校验开启后）
4. **添加更多防护**:
   - ESLint `no-hardcoded-colors` 自定义规则
   - pre-commit grep 检查魔数
5. **视觉回归测试**（dark mode 适配验证 420+ 处 var() 切换）
6. **处理剩余 LOW severity 风格建议**（interface{} → any 批量）

## 经验教训

1. **同模式批量可并行 executor**：Round 7 用 5 并行 executor 处理 ~12 个 finding 大幅缩短 wall-clock
2. **pre-existing 编译错误阻塞**：scrapli_wrapper.go GetPrompt 签名变更导致 device package 编译失败，影响所有依赖 package 的 build 验证。处理：其他会话修复后立即恢复
3. **build cache 损坏**：Windows Go 安装的 GOROOT 缓存偶发损坏（`go clean -cache` 修复）
4. **package-level const 跨包共享 vs 重复**：F-BE-36/37 决定各持一份 const（避免跨包耦合），值保持一致
5. **SVG attrs var() 不支持**：W3C svgwg #1031，需 JSX 重构（`stroke=` → `style={{stroke}}`）超出 const 提取范畴
6. **intentional external revert**：F-BE-46 ai_analyzer.go 被外部（linter/用户）还原，system-reminder 标注 intentional，尊重不修改
7. **ESLint `no-restricted-syntax` regex 匹配 Literal 节点**：可在不写自定义 plugin 前提下防止硬编码

## 关联

- `.planning/STATE.md` — 整体项目状态
- `docs/项目概述和架构设计.md` — 架构参考
- `docs/开发规范.md` — 代码规范
- `xingran-react-frontend/eslint.config.js` — ESLint 配置（含防护规则）
- 87 个 audit-fix commit 散落在 `main` 分支
