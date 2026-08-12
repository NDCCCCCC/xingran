# docs 过程性文档归档索引

- **最近归档**: 2026-08-12（docs 全量重整：过时归档 + 子目录分类）
- **历史归档**: 20260812-141515
- **历史备份包**: `archive/docs-backup-20260812-141515.tar.gz`
- **归档原则**: 过程性报告（完成/进度/审查/实施总结/验证/优化报告）+ 已完成实施计划 + 已过时版本快照 + 一键性 bug 修复 ADR；保留核心架构、规范、参考指南、设计计划

---

## 第一轮归档（20260812-141515）：过程性报告

13 份过程性报告 + 1 个空文件：

- `CODE_OPTIMIZATION_SUMMARY.md`
- `code-review-report.md`
- `Swagger文档实施总结.md`
- `user_service_N+1优化报告.md`
- `代码简化和优化建议.md`
- `代码简化优化完成报告.md`
- `代码简化优化总结.md`
- `统一错误处理实施总结.md`
- `项目验证报告.md`
- `RPA-Worker-完成报告.md`
- `RPA系统开发进度.md`
- `RPA系统完成状态报告.md`
- `RPA系统完整性检查清单.md`
- `login-encryption-security.md`（原 `security/`，仅 1 行空文件）

---

## 第二轮归档（2026-08-12）：docs 全量重整

新增归档 9 份（过时/已完成/一次性文档），并删除 2 个死代码 Go 文件。

### 过时版本快照

- `部署和运维文档.md` — 标 v1.0.0 / Go 1.23 / Vite 6 / Antd 5.21 / 称"若依框架"，全面过时

### 已完成实施计划（功能已 100% 上线）

- `网络设备批量管理功能实施计划.md`
- `code-fix-guide.md`（全部条目 ✅ 完成）
- `TypeScript代码质量优化方案.md`（1355 行已完成工作日志）
- `interface-type-safe-migration-guide.md`（类型安全迁移已完成）
- `2026-06-24-frontend-vendor-bundle-optimization.md`（已核验 echarts 精确集合/react-markdown 死依赖/MarkdownEditor nohighlight 三项全部落地）

### 一次性上线记录 / 一次性 bug 修复

- `login-encryption-deployment.md`（SM2+SM4 登录加密一次性上线 runbook，现已是默认行为）
- `网络设备连接状态机设计文档.md`（scrapligo 连接状态机 bug 修复 ADR）

### 纯重复（已合并入 `cache_usage.md`）

- `缓存架构演进说明.md` — Legacy 缓存移除记录作为附录并入 `docs/guides/cache_usage.md` 后归档

---

## 已清理的死代码（Go 文件，不属于归档范畴）

- `docs/models.go` — 25 个手写数据模型，全代码库无任何 `docs.XxxType` 限定引用；真实类型在 `internal/api/v1/` 与 `pkg/response/`
- `docs/swagger_templates.go` — 同上，6 个手写响应模板无任何引用

## swaggo 产物位置调整

为分离代码与文档目录，`docs.go` / `swagger.json` / `swagger.yaml` 已从 `docs/` 移到 **`internal/docs/`**（仍是 `package docs`）。生成方式：`swag init -g cmd/main.go -o internal/docs`。若未来 `cmd/main.go` 补上 `_ ".../internal/docs"` blank import 即可让 Swagger UI 加载接口规范（当前缺此 import，UI 可能加载不到规范——属代码遗留问题，不在文档清理范围）。

> ⚠️ 代码层面遗留问题（不在文档清理范围）：`cmd/main.go:206` 注册了 Swagger UI，但项目**无 `_ ".../docs"` blank import**，`docs.SwaggerInfo` 运行时可能未初始化——Swagger UI 实际可能加载不到接口规范。建议后续排查 `swag init` 配置与 main.go 导入。

---

## 已移至 `.planning/`（未落地的真实待办，不应归档）

- `.planning/operlog-nickname-display-plan.md` — 操作日志 oper_name 改为 nickname(username) 的方案（2026-06-21 至今**后端未落地**，确认为有效待办）

---

## docs/ 现状（第二轮重整后）

### 保留文档（21 份）按子目录分类

```
docs/
├── README.md                          # 文档地图（本文）
├── architecture/                      # 架构与设计
│   ├── 项目概述和架构设计.md
│   ├── 数据库设计.md
│   ├── 安全和认证设计（国密）.md
│   └── 启动流程清单.md
├── standards/                         # 开发规范
│   ├── 开发规范.md
│   ├── API响应规范.md
│   └── 跨模块选择器权限处理规范.md
├── guides/                            # 使用指南与 How-to
│   ├── cache_usage.md
│   ├── 缓存重试机制使用指南.md
│   ├── gormutil工具说明.md
│   ├── 时间工具函数使用说明.md
│   ├── EXCEL_IMPORT_GUIDE.md
│   ├── encryption-config-sync.md
│   ├── 上传下载功能设计.md
│   └── SECURITY_IMPROVEMENTS.md
├── deployment/                        # 部署与运维
│   ├── deployment.md                  # 生产环境部署指南（当前权威）
│   └── secret-management.md           # 部署期密钥管理
├── modules/                           # 模块设计
│   ├── agent-test-plan.md
│   └── rpa/
│       ├── RPA系统设计方案.md
│       ├── RPA-数据格式规范.md
│       └── RPA-Worker-API认证方案-待办.md
├── plans/                             # 前瞻设计
│   └── 2026-07-09-v1.20.1-design.md   # v1.20.1 端口写命令设计（pending sign-off）
├── reference/                         # 外部参考资料
│   ├── 深信服桌面云开放平台（V1.2）.doc
│   ├── sangfor_vdi_utf8.txt
│   ├── sangfor_api_auth.txt
│   ├── sangfor_api_resource.txt
│   └── sangfor_api_vm.txt
└── archive/                           # 历史归档（25 份）

# Swagger 产物位于 internal/docs/（swaggo 自动生成，package docs）
```
