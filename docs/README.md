# docs/ 文档地图

XingRan-Next 项目的全部技术文档索引。最后整理：2026-11-12。

> **产品定义（一页纸）**：[`PRODUCT.md`](PRODUCT.md) — 统一产品名 XingRan-Next、一句话定位（企业 IT 运维统一管理平台）、三类角色、三个可量化成功指标、明确「不做什么」。产品口径以该文件为唯一权威说明。

## 快速导航

| 类别 | 目录 | 内容 | 文件数 |
|------|------|------|--------|
| 产品定义 | [`PRODUCT.md`](PRODUCT.md) | 定位 / 角色 / 成功指标 / 边界（权威） | 1 |
| 架构与设计 | [`architecture/`](architecture/) | 项目总览、数据库、安全国密、启动流程 | 4 |
| 开发规范 | [`standards/`](standards/) | 开发规范、API 响应、权限选择器 | 3 |
| 使用指南 | [`guides/`](guides/) | 缓存、Excel、加密、上传下载、安全改进、错误码/FAQ 索引等 How-to | 10 |
| 部署运维 | [`deployment/`](deployment/) | 生产部署、单机部署、Docker Compose、密钥管理、容量规划、CI/CD | 7 |
| 模块设计 | [`modules/`](modules/) | RPA 子模块（3 份）、Agent 测试（1 份） | 4 |
| 前瞻设计 | [`plans/`](plans/) | 下一版本设计草案 | 2 |
| 外部参考 | [`reference/`](reference/) | 深信服 VDI 接口等第三方资料 | 6 |
| 历史归档 | [`archive/`](archive/) | 已过时/已完成的文档 | 25 |
| 评审记录 | [`reviews/`](reviews/) | 归档理由、前端审查 | 3 |
| 安全专题 | [`security/`](security/) | 登录加密安全设计 | 1 |
| 运维 SQL | [`operations/sql/`](operations/sql/) | 一次性 SQL 变更记录 | 1 |

## 目录详情

### 🎯 PRODUCT.md — 产品定义（权威）

- **`PRODUCT.md`** — 统一产品名、一句话定位、三类角色（系统管理员 / 运维管理员 / 安全管理员）、三个可量化成功指标（管理员独立完成一次运维任务耗时 / 关键数据准确率 / 现场真机功能可用率）、明确「不做什么」边界（含 3 处 Mermaid / 表格化可视化）。

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

### 📖 guides/ — 使用指南（10 份）

- **`cache_usage.md`** — CacheProvider 接口、System 模块新架构、缓存策略与失效（含 Legacy 演进附录）
- **`缓存重试机制使用指南.md`** — L2 Redis 写入失败指数退避重试
- **`gormutil工具说明.md`** — pkg/gormutil（PreloadBuilder / JoinBuilder）解决 N+1
- **`时间工具函数使用说明.md`** — 前端 `formatDateTime/formatDate/formatTime`（对应 `src/utils/datetime.ts`）
- **`EXCEL_IMPORT_GUIDE.md`** — 楼宇/楼层/工位 Excel 批量导入（含工位 dept/user 关联字段）
- **`encryption-config-sync.md`** — 前后端运行时同步加密开关
- **`上传下载功能设计.md`** — FileService 统一文件服务 + Excel 导入导出
- **`SECURITY_IMPROVEMENTS.md`** — 菜单存储 SHA-256 校验 + XSS 清理 + CSP
- **`错误码速查.md`** — 错误码可读说明索引（摘自 `standards/API响应规范.md` §错误响应规范，含 HTTP/业务错误码表 + Mermaid 反查流程；不重写权威文档）
- **`FAQ-索引.md`** — 从 [`管理员使用手册.md`](管理员使用手册.md) §21 摘列的 12 条常见问题索引（按域分组 + Mermaid 排查流程；不重写手册）

### 🚀 deployment/ — 部署与运维（7 份）

按"场景规模"选用：

- **`single-machine-deployment.md`** — 单机全栈部署（硬件规格、OS、运行时、一条龙流程、踩坑清单；开发 / 试点 / 小型团队首选）
- **`docker-compose.md`** — Docker Compose 一键编排（补齐 README 中标注缺失的容器化方案）
- **`capacity-planning.md`** — 容量规划与硬件选型（按用户/工单/模块开关估算 PG/Redis/连接池）
- **`deployment.md`** — 生产 systemd 部署指南（`/app/szh/`、内网 10.62.10.34、systemd 加固；**生产权威**）
- **`secret-management.md`** — 部署期密钥管理（生成、轮转、泄漏处置；跨文档通用）
- **`login-encryption-deployment.md`** — SM2+SM4 登录加密一次性上线 runbook
- **`github-actions-cicd.md`** — GitHub Actions CI/CD 流水线

### 🧩 modules/ — 模块设计（4 份）

- **`agent-test-plan.md`** — VM Agent（VDI 账号）测试方案
- **`rpa/RPA系统设计方案.md`** — 分布式 RPA 总架构（设计层 rod 已替代 Playwright）
- **`rpa/RPA-数据格式规范.md`** — ScriptAction ↔ WorkerAction 字段映射（实现契约）
- **`rpa/RPA-Worker-API认证方案-待办.md`** — 双令牌认证方案（✅ 已落地：v1.51 DEBT-03，2026-11-01；migration_230 + worker_token_service.go + pkg/middleware/worker_auth.go + rpa_router.go，详见该文档顶部落地注记）

### 📋 plans/ — 前瞻设计（2 份）

- **`2026-06-24-frontend-vendor-bundle-optimization.md`** — 前端 vendor bundle 体积优化方案
- **`2026-07-09-v1.20.1-design.md`** — v1.20.1 端口写命令设计（✅ 已实施：Phase 56 落地 set_access_vlan / port_binding，v1.51 Phase 124/125 扩展，非 pending）

### 📚 reference/ — 外部参考（6 份）

- **`深信服桌面云开放平台（V1.2）.doc`** — VDI API 原始文档
- **`sangfor_vdi_utf8.txt`** — UTF-8 文本提取版
- **`sangfor_vdi_api.txt`** — VDI API 文本摘录（原 docs 根副本移入）
- **`sangfor_api_{auth,resource,vm}.txt`** — 三段接口摘录

### 📜 archive/ — 历史归档（25 份）

已过时版本、已完成实施计划、一次性上线记录等。详见 [`archive/ARCHIVE-INDEX.md`](archive/ARCHIVE-INDEX.md)。

### 🔍 reviews/ / 🔐 security/ / 🗄 operations/ — 其它专题（5 份）

- **`reviews/`**：`archive-items-rationale.md`、`frontend-review-2026-08-14.md`、`frontend-review-2026-08-14-verification.md`
- **`security/`**：`login-encryption-security.md`
- **`operations/sql/`**：`2026-08-13-drop-idx-api-keys-key.sql`

## 例外与保留声明（重要）

> 2026-11-12 清理 docs 根与子目录同名重复副本时，遵循「先确认引用为零、宁可少删不可误删」：
> - 已删除 29 份**全仓无引用**（或引用仅来自待删集合 / 将改写的 README / 已归档快照）的 docs 根旧副本；`docs/sangfor_*.txt`、`深信服*.doc` 等外部参考已收敛到 `reference/`（同名已有则跳过、不重复落盘）。
> - **以下 16 份根副本因仍被白名单外文件引用而保留为例外**，不删除：`API响应规范.md`、`EXCEL_IMPORT_GUIDE.md`、`RPA-Worker-API认证方案-待办.md`、`RPA系统完成状态报告.md`、`RPA系统设计方案.md`、`agent-test-plan.md`、`cache_usage.md`、`code-review-report.md`、`deployment.md`、`interface-type-safe-migration-guide.md`、`启动流程清单.md`、`安全和认证设计（国密）.md`、`开发规范.md`、`数据库设计.md`、`部署和运维文档.md`、`项目概述和架构设计.md`。
>   引用来源是 `CLAUDE.md`、`CHANGELOG.md`、`scripts/README.md`、`scripts/agent/README.md`、Go 源码注释、`configs/`、`管理员使用手册.md`（归 A1）、`.planning/` 归档等，本任务无所有权、不得改写；这些根副本与子目录权威版并存，待各自维护方统一迁移引用后再收敛。**子目录版本为权威版**，README 与各索引均改指子目录。
> - **`docs/管理员使用手册.md` 绝对不删、不改**（归 A1）。
> - **品牌名历史归档例外**：`.planning/` 下除 `PROJECT.md` 外的文件（STATE / ROADMAP / MILESTONES / notes / reports / research / reviews 等）与 `.github/scripts/check-diff-coverage.sh` 注释中仍保留旧品牌名（RuoYi 系旧称，含 `Nect` 拼写错误变体）的历史字样，归 A4 / 代码维护方所有，本任务不改；这些是历史时点记录与重命名留痕，不代表当前产品名。当前产品名统一为 **XingRan-Next**，见 [`PRODUCT.md`](PRODUCT.md)。

## 相关索引

- [产品定义（权威）](PRODUCT.md)
- [错误码速查](guides/错误码速查.md) / [FAQ 索引](guides/FAQ-索引.md)
- [归档索引](archive/ARCHIVE-INDEX.md)
- 项目根 [CLAUDE.md](../CLAUDE.md) — 完整的开发指南与命令参考
- [README.md](../README.md) — 项目说明

> Swagger UI 文档由 [`internal/docs/`](../internal/docs) 下的 swaggo 产物（`docs.go` / `swagger.json` / `swagger.yaml`）驱动，访问入口：`http://localhost:9000/swagger/index.html`。
