<!-- generated-by: gsd-doc-writer -->

# XingRan-Next 企业级权限管理系统

XingRan-Next 是基于若依 (XingRan) 开源管理系统的全新重构版本，采用 **Go + React** 双端分离架构，并集成 **国密 SM2/SM3/SM4 算法** 满足等保合规要求。系统面向企业内部运维、业务、数据可视化的统一管理平台，在保留传统 RBAC 权限模型的基础上扩展了 3D 楼宇可视化、AD 域同步、网络设备纳管、工位/机房运维、RPA 自动化等企业级场景能力。

## 核心特性

- **用户/角色/菜单/部门/岗位/字典/通知/参数**：完整的 RBAC 权限与基础数据治理
- **3D 楼宇/楼层/机房可视化**：基于 Three.js + react-three/fiber 的 CAD 风格平面编辑器与三维场景
- **工位/设备/资产/网段管理**：Excel 批量导入导出、百度地图地理编码、部门/用户自动关联
- **网络设备纳管**：Scrapli (SSH/Telnet) + SNMP + TextFSM 模板解析，支持端口采集、MAC 历史、LLDP 拓扑、端口写命令（shutdown / undo shutdown / description / dot1x 启停）+ 批量配置 + 完整审计（sys_port_write_audit）
- **AD 域集成**：LDAP 连接池、定时组同步、OU 管理、用户密码变更
- **RPA 自动化**：Worker 自动扩缩容 (Docker)、AI 生成脚本与视觉 Agent 降级
- **VDI 桌面云**：深信服 VDI 接口对接、虚拟机生命周期管理
- **工单/排班/值班**：工单轮转分配、定时工单、值班排班与统计
- **知识库/资讯/通知**：Markdown 编辑、定向发布、未读追踪
- **任务调度**：基于 robfig/cron 的可视化任务管理、执行日志
- **监控大盘**：服务器资源、缓存状态、在线用户、操作日志
- **国密安全体系**：SM3 密码哈希、SM2 密钥协商、SM4-CBC 请求/响应体加密、双 Token 认证

## 技术栈

### 后端

| 类别 | 选型 | 版本 |
|------|------|------|
| 语言/运行时 | Go | 1.24 (toolchain go1.24.5) |
| Web 框架 | Gin | v1.10.0 |
| ORM | GORM | v1.30.5 |
| 数据库 | PostgreSQL (主) / SQLite (备) | GORM Driver postgres v1.5.9 / sqlite v1.5.4 |
| 缓存 | Redis | go-redis/v9 v9.7.0 |
| 认证 | JWT 双 Token + 国密 | golang-jwt/jwt v5.2.1、tjfoc/gmsm v1.4.1 |
| 任务调度 | robfig/cron | v3.0.1 |
| 网络设备 | Scrapli + gosnmp + TextFSM | scrapligo v1.3.3、gosnmp v1.35.0 |
| Excel | excelize | v2.10.0 |
| 配置 | Viper | v1.19.0 |
| 日志 | logrus + lumberjack | v1.9.3 / v2.2.1 |
| 文档 | swaggo/swag | v1.16.4 |

### 前端

| 类别 | 选型 | 版本 |
|------|------|------|
| 框架 | React + TypeScript | 19.2 / 5.9 |
| 构建 | Vite | 7.2 |
| UI | Ant Design | 6.1 |
| 样式 | Tailwind CSS | 4.1 |
| 状态管理 | Zustand + TanStack Query | 5.0 / 5.90 |
| 路由 | react-router-dom | 7.10 |
| 3D | three + @react-three/fiber + @react-three/drei | 0.182 / 9.5 / 10.7 |
| 图表 | ECharts | 6.0 |
| 地图 | @uiw/react-baidu-map | 2.7 |
| 国密 | sm-crypto | 0.3.13 |
| 测试 | Vitest | 4.0 |

## 架构概览

```
请求 → Router → 中间件链 (Auth → Permission → 加解密) → Handler → Service → DB/Cache
        ↑                                                              ↓
        └────────────── Response Wrapper (统一 JSON 格式) ←────────────┘
```

**关键设计：**

1. **Handler-Service 分层**：所有业务模块采用接口+私有实现+依赖注入构造器模式（如 `NewUserService(db, cache, pwdMgr)`）。
2. **双层缓存架构**：`pkg/cache` 提供 L1(内存)+L2(Redis) 统一接口；`internal/services/system/*_cache_impl.go` 通过 `CacheProvider` 接口解耦，根目录遗留的 `*_cache_service.go` 仍由 `core.Core` 使用以保持兼容。
3. **统一响应格式**：`{code, message, data, timestamp, request_id}`，成功码 `0`。
4. **状态值约定**：通用 `0=正常/启用/可见`，`1=停用/禁用/隐藏`；菜单可见性字段 `visible` 反向（`1=可见`）。
5. **国密请求体加密**：SM4-CBC 加密 body + SM2 加密 SM4 密钥，防重放（300s 时间窗 + nonce）。
6. **双 Token 认证**：Access Token 短期 + Refresh Token 长期，前端 `authStore` 集成自动刷新。

**目录结构：**

```
xingran-go-backend/
├── cmd/                        # 应用入口 (main.go)
├── configs/                    # 配置文件 (config.yaml, config.dev.yaml, config.prod.yaml)
├── internal/
│   ├── api/v1/                 # HTTP Handlers (system/operations/scheduler/workorder/duty/network/knowledge/monitor/vdi/agent/...)
│   ├── services/               # 业务服务层
│   ├── models/                 # GORM 数据模型
│   ├── core/                   # 核心模块 (DB/Cache/JWT/SM4/Device/Scheduler)
│   ├── config/                 # 配置加载
│   ├── device/                 # 网络设备连接 (Scrapli/TextFSM)
│   ├── collectors/             # 数据采集器
│   ├── scheduler/              # Cron 引擎（不要与 api/v1/scheduler 混淆）
│   ├── templates/              # 模板解析
│   ├── utils/                  # 工具函数
│   └── websocket/              # WebSocket 服务
├── pkg/                        # 公共可复用包
│   ├── cache/                  # Redis + 内存缓存接口
│   ├── crypto/                 # SM2/SM4 加解密
│   ├── middleware/             # Auth/CORS/Logging/Encryption
│   ├── permission/             # RBAC 权限定义
│   ├── query/                  # 查询构造器
│   └── response/               # 统一响应包装
├── scripts/                    # 构建与 Swagger 生成脚本
├── docs/                       # 项目文档
└── xingran-react-frontend/       # 前端项目 (React + Vite)
```

## 快速开始

### 环境要求

- Go 1.24+
- PostgreSQL 18+（或 SQLite 备用）
- Redis 7.4+
- Node.js 24+（前端）

### 后端启动

```bash
# 1. 准备配置
cp configs/config.dev.yaml configs/config.yaml
# 按需修改 configs/config.yaml 中的 database/cache/jwt 等项

# 2. 拉取依赖并启动
go mod tidy
go run ./cmd/main.go
```

服务默认监听 `http://0.0.0.0:9000`。

### 前端启动

```bash
cd xingran-react-frontend
npm install
npm run dev      # 开发服，默认 http://localhost:4000
npm run build    # 生产构建
```

### 常用脚本

```bash
# 后端
go build -o xingran-backend.exe ./cmd/main.go
go test ./...
go test ./internal/services/operations/

# 前端
npm run lint         # ESLint 检查
npm run lint:fix     # 自动修复
npm run type-check   # TypeScript 类型检查
npm run test         # Vitest 测试
```

## 配置说明

主要配置项位于 `configs/config.yaml`：

| 配置块 | 关键字段 | 说明 |
|--------|----------|------|
| `server` | `host`, `port`(9000), `mode`(debug/release) | 服务监听与运行模式 |
| `database` | `host`, `port`, `user`, `password`, `dbname`, `sslmode` | 支持 `DB_HOST/DB_PORT/DB_USER/DB_PASSWORD/DB_NAME` 环境变量覆盖 |
| `cache` | `type`(redis/memory), `host`, `port`, `password`, `db`, `l2_writer`, `retry_*` | Redis 连接池、L2 写入 Worker Pool 与重试参数 |
| `jwt` | `secret_key`, `access_key_expire`(7200s), `refresh_key_expire`(604800s), `use_sm2` | JWT 签发与 SM2 开关 |
| `log` | `level`, `log_dir`, `max_size`, `max_backups`, `compress` | 日志轮转 |
| `security.sm4_key` | Base64 编码的 16 字节密钥 | 设备密码、AD 密码、RPA 凭证等敏感字段加密 |
| `security.request_encryption` | `enabled`, `exclude_paths` | 请求体加密开关与排除路径（公钥、上传、验证码、RPA Worker 等；登录接口已不在排除列表中） |
| `security.response_encryption` | `enabled` | 响应体加密（默认禁用，由参数管理 `sys.request.encryption.enabled` 动态控制） |
| `baidu.map_ak` | 百度地图 AK | 工位/楼宇地理编码，可由 `BAIDU_MAP_AK` 环境变量覆盖 |
| `rpa.*` | ai/worker/scaling/storage | RPA Worker、AI 模型、扩缩容与存储 |
| `ad_group_sync` | `enabled`, `cron`, `auto_create_groups` | AD 组定时同步 |

**关键环境变量：**

```bash
DB_HOST=localhost
DB_PORT=5432
DB_USER=xingran
DB_PASSWORD=your_password
DB_NAME=xingran_next

REDIS_URL=redis://localhost:6379
REDIS_PASSWORD=

BAIDU_MAP_AK=your_api_key_here
```

## API 文档

集成 Swagger UI：

```bash
# 生成 Swagger 文档
./scripts/generate_swagger.sh     # Linux/Mac (项目当前仅提供此脚本)
```

启动后访问 `http://localhost:9000/swagger/index.html`。

所有响应遵循统一格式：

```json
{
  "code": 0,
  "message": "success",
  "data": {},
  "timestamp": 1766380800,
  "request_id": "uuid-string"
}
```

## 部署

### 生产构建

```bash
# Linux 生产构建
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-s -w" -o xingran-backend-linux ./cmd/main.go

# 使用生产配置
cp configs/config.prod.yaml configs/config.yaml
```

### Docker 部署

仓库根目录当前未提供 `docker-compose.yml`；如需容器化部署，请参考 `docs/deployment/deployment.md` 自行构建镜像。

### 部署前检查清单

- 修改 `jwt.secret_key` 为高熵随机字符串
- 通过 `export SM4_KEY="$(openssl rand -base64 16)"` 重新生成 SM4 密钥
- 数据库、Redis 配置改为生产内网地址与强密码
- 关闭 `server.mode: debug`
- 评估并按需启用响应体加密（默认禁用）

## 详细文档

更多设计与实现细节请参见 `docs/` 目录：

| 文档 | 涵盖内容 |
|------|----------|
| [项目概述和架构设计](docs/architecture/项目概述和架构设计.md) | 系统分层、模块划分、关键设计 |
| [开发规范](docs/standards/开发规范.md) | Go/React 编码规范、状态值约定、API 约定 |
| [API 响应规范](docs/standards/API响应规范.md) | 统一响应格式、错误码、分页 |
| [安全和认证设计（国密）](docs/architecture/安全和认证设计（国密）.md) | JWT 双 Token、SM2/SM3/SM4 加解密、RBAC |
| [数据库设计](docs/architecture/数据库设计.md) | 表结构、命名、迁移管理 |
| [部署/生产部署指南](docs/deployment/deployment.md) | 生产环境 systemd 部署与密钥管理 |
| [EXCEL_IMPORT_GUIDE](docs/guides/EXCEL_IMPORT_GUIDE.md) | Excel 批量导入指南（含工位 dept/user 关联） |
| [cache_usage](docs/guides/cache_usage.md) | 双层缓存设计、Redis 键前缀（含 Legacy 演进附录） |
| [RPA 系统设计方案](docs/modules/rpa/RPA系统设计方案.md) | RPA 分布式架构 |
| [上传下载功能设计](docs/guides/上传下载功能设计.md) | 文件上传下载方案 |
| [文档地图](docs/README.md) | docs/ 全量分类索引 |

## 贡献指南

1. Fork 仓库并创建特性分支：`git checkout -b feature/your-feature`
2. 遵循 `docs/standards/开发规范.md` 编写代码（Handler-Service 模式、接口+私有实现、状态值约定）
3. 提交前本地验证：
   ```bash
   go build ./...      # 后端编译
   go test ./...       # 后端测试
   npm run lint        # 前端检查
   npm run type-check  # 前端类型
   ```
4. 提交信息遵循 Conventional Commits（`feat:`, `fix:`, `docs:`, `chore:` 等）
5. 发起 Pull Request，等待 Code Review

## 许可证

MIT License
