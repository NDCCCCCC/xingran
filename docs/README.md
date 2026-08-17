# docs/ 文档地图

XingRan-Next 项目的全部技术文档索引。最后整理：2026-08-12。

## 快速导航

| 类别 | 目录 | 内容 |
|------|------|------|
| 架构与设计 | [`architecture/`](architecture/) | 项目总览、数据库、安全国密、启动流程 |
| 开发规范 | [`standards/`](standards/) | 开发规范、API 响应、权限选择器 |
| 使用指南 | [`guides/`](guides/) | 缓存、Excel、加密、上传下载、安全改进等 How-to |
| 部署运维 | [`deployment/`](deployment/) | 生产部署、单机部署、Docker Compose、密钥管理、容量规划 |
| 模块设计 | [`modules/`](modules/) | RPA 子模块、Agent 测试 |
| 前瞻设计 | [`plans/`](plans/) | 下一版本设计草案 |
| 外部参考 | [`reference/`](reference/) | 深信服 VDI 接口等第三方资料 |
| 历史归档 | [`archive/`](archive/) | 已过时/已完成的文档（25 份） |

## 目录详情

### 📐 architecture/ — 架构与设计（4 份）

权威架构类文档，是新成员了解项目的入口。

- **`项目概述和架构设计.md`** — 技术栈、分层架构、模块清单、里程碑总纲
- **`数据库设计.md`** — 表命名规范、状态值枚举、Go/TS 模型对照
- **`安全和认证设计（国密）.md`** — SM2/SM3/SM4、JWT 双 Token、加密脱敏、RBAC
- **`启动流程清单.md`** — 后端启动每一步分类（必跑/一次性/混合幂等）+ SkipSetup 开关

### 📏 standards/ — 开发规范（3 份）

- **`开发规范.md`** — 编码规范总则（状态值、数据库、API、Go/TS、操作日志、测试、部署）
- **`API响应规范.md`** — 统一响应结构、错误码、分页/批量/异步等场景（API 真相源）
- **`跨模块选择器权限处理规范.md`** — 解决跨模块 list/tree 选择器 403 断层

### 📖 guides/ — 使用指南（8 份）

- **`cache_usage.md`** — CacheProvider 接口、System 模块新架构、缓存策略与失效（含 Legacy 演进附录）
- **`缓存重试机制使用指南.md`** — L2 Redis 写入失败指数退避重试
- **`gormutil工具说明.md`** — pkg/gormutil（PreloadBuilder / JoinBuilder）解决 N+1
- **`时间工具函数使用说明.md`** — 前端 `formatDateTime/formatDate/formatTime`（对应 `src/utils/datetime.ts`）
- **`EXCEL_IMPORT_GUIDE.md`** — 楼宇/楼层/工位 Excel 批量导入（含工位 dept/user 关联字段）
- **`encryption-config-sync.md`** — 前后端运行时同步加密开关
- **`上传下载功能设计.md`** — FileService 统一文件服务 + Excel 导入导出
- **`SECURITY_IMPROVEMENTS.md`** — 菜单存储 SHA-256 校验 + XSS 清理 + CSP

### 🚀 deployment/ — 部署与运维（5 份）

按"场景规模"选用：

- **`single-machine-deployment.md`** — 单机全栈部署（硬件规格、OS、运行时、一条龙流程、踩坑清单；开发 / 试点 / 小型团队首选）
- **`docker-compose.md`** — Docker Compose 一键编排（补齐 README 中标注缺失的容器化方案）
- **`capacity-planning.md`** — 容量规划与硬件选型（按用户/工单/模块开关估算 PG/Redis/连接池）
- **`deployment.md`** — 生产 systemd 部署指南（`/app/szh/`、内网 10.62.10.34、systemd 加固；**生产权威**）
- **`secret-management.md`** — 部署期密钥管理（生成、轮转、泄漏处置；跨文档通用）

### 🧩 modules/ — 模块设计（4 份）

- **`agent-test-plan.md`** — VM Agent（VDI 账号）测试方案
- **`rpa/RPA系统设计方案.md`** — 分布式 RPA 总架构（设计层 rod 已替代 Playwright）
- **`rpa/RPA-数据格式规范.md`** — ScriptAction ↔ WorkerAction 字段映射（实现契约）
- **`rpa/RPA-Worker-API认证方案-待办.md`** — 双令牌认证方案（**真实待办**，4 个新建文件未落地）

### 📋 plans/ — 前瞻设计（1 份）

- **`2026-07-09-v1.20.1-design.md`** — v1.20.1 端口写命令设计（status: pending user sign-off）

### 📚 reference/ — 外部参考（5 份）

- **`深信服桌面云开放平台（V1.2）.doc`** — VDI API 原始文档
- **`sangfor_vdi_utf8.txt`** — UTF-8 文本提取版
- **`sangfor_api_{auth,resource,vm}.txt`** — 三段接口摘录

### 📜 archive/ — 历史归档（25 份）

已过时版本、已完成实施计划、一次性上线记录等。详见 [`archive/ARCHIVE-INDEX.md`](archive/ARCHIVE-INDEX.md)。

## 相关索引

- [归档索引](archive/ARCHIVE-INDEX.md)
- 项目根 [CLAUDE.md](../CLAUDE.md) — 完整的开发指南与命令参考
- [README.md](../README.md) — 项目说明

> Swagger UI 文档由 [`internal/docs/`](../internal/docs) 下的 swaggo 产物（`docs.go` / `swagger.json` / `swagger.yaml`）驱动，访问入口：`http://localhost:9000/swagger/index.html`。