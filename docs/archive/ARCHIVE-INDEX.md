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

以下 5 份 Swagger 死产物原位于 `docs/` 下、与 `internal/docs/` 权威产物重复，已于 Phase 135 / FIX-02 全部删除；下列路径均为历史记录（文件现已不存在）：

- `docs/models.go`（原路径，已删除）— 25 个手写数据模型，全代码库无任何 `docs.XxxType` 限定引用；真实类型在 `internal/api/v1/` 与 `pkg/response/`
- `docs/swagger_templates.go`（原路径，已删除）— 同上，6 个手写响应模板无任何引用
- `docs/docs.go`（原路径，已删除）— swaggo 生成物（33710 行）
- `docs/swagger.json`（原路径，已删除）— 同上
- `docs/swagger.yaml`（原路径，已删除）— 同上

## swaggo 产物位置调整

为分离代码与文档目录，`docs.go` / `swagger.json` / `swagger.yaml` 已从 `docs/` 移到 **`internal/docs/`**（仍是 `package docs`）。生成方式：`swag init -g cmd/main.go -o internal/docs`——两个生成脚本均已按此输出。`cmd/main.go` 已补 `_ ".../internal/docs"` blank import，`internal/docs/docs.go` 的 `init()` → `swag.Register` 生效，Swagger UI 可正常加载接口规范。

> ✅ **生成脚本已对齐（Phase 135 / FIX-02）**：`scripts/build/generate_swagger.bat` 与 `scripts/generate_swagger.sh` 的 `-o` 均已改为 `internal/docs`，产物不再写回 `docs/`；bat 的 `cd /d "%~dp0.."` 层级 bug 也已修正为 `cd /d "%~dp0..\.."`；上列 `docs/` 下 5 份死文件已删除。

> ✅ **blank import 已补（Phase 135 / FIX-02）**：`cmd/main.go` import 块已含 `_ ".../internal/docs"`，`internal/docs/docs.go` 的 `swag.Register` 正常执行，`/swagger/*any` 可取到规范。

---

## 历史记录（已失效，勿再作为待办）

- **operlog nickname 显示**：已由 migration_160/161 + `internal/utils/context_helper.go` 的 `GetNicknamePtrWithDB` + `internal/utils/operlog/operlog.go:254` 落地；原方案文件 `.planning/operlog-nickname-display-plan.md` 已不存在（现行方案文档为 `docs/操作日志-nickname显示实现方案.md`）。
- **RPA Worker Token 认证**：已于 v1.51 DEBT-03（2026-11-01）落地（migration_230 / worker_token_service.go / worker_auth.go / rpa_router.go，见 `docs/modules/rpa/RPA-Worker-API认证方案-待办.md` 顶部落地注记）。
- 本节原登记的两条均已完成，不再是待办；真实开放项的当前结论以里程碑需求清单（`.planning/milestones/v1.53-REQUIREMENTS.md`）为准。

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
│   └── 2026-07-09-v1.20.1-design.md   # v1.20.1 端口写命令设计（✅ 已实施，非 pending）
├── reference/                         # 外部参考资料
│   ├── 深信服桌面云开放平台（V1.2）.doc
│   ├── sangfor_vdi_utf8.txt
│   ├── sangfor_api_auth.txt
│   ├── sangfor_api_resource.txt
│   └── sangfor_api_vm.txt
└── archive/                           # 历史归档（25 份）

# Swagger 产物位于 internal/docs/（swaggo 自动生成，package docs）
```
