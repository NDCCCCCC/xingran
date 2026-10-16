# scripts 归档（2026-08-12 整理）

## 归档原因

2026-08-12 进行 `scripts/` 目录整理时，将顶层散落的 17 个过时/一次性脚本移入此目录统一封存，**并非删除**（目录非 git 仓库，无历史可回溯；移入归档而非 `rm` 以保留可恢复性）。

如需恢复某个脚本，将对应文件 `mv` 回原位即可。

## 分组与清单

### `phase-verification/` — 历史阶段验收脚本（2 个）

阶段已完成验收，留作历史参考。不再随日常开发运行。

| 文件 | 说明 |
|------|------|
| `verify_phase40.sh` | Phase 40（Tech-Debt Cleanup）验收脚本：校验 frontmatter 扫描通过率 + debug sessions 数量 |
| `verify_phase46_r5.sh` | Phase 46 R5（半自动修复）端到端验证脚本 |

### `oneoff-fixes/` — 一次性修复/补丁脚本（4 个）

已经完成其使命的源码 patch 或批量修复脚本。其中 `fix_ip_update.py` 的硬编码路径还是旧仓库名 `xingran-go-backend`（现仓库为 `guoguo`），即便想重跑也已失效。

| 文件 | 说明 |
|------|------|
| `fix_ip_update.py` | 靠行号 hack 修改 `vdi/vm_service_impl.go` 的一次性源码 patch（路径已过期） |
| `fix_debug_frontmatter.py` | Phase 40 用户批准的 `.planning/debug/*.md` frontmatter 批量修复（幂等，已跑完） |
| `validate_debug_frontmatter.sh` | Phase 40 专用 frontmatter 校验脚本，配套上一条 |
| `operlog_e2e_verify.sh` | Phase 34 bash 版操作日志端到端验证；`e2e/operlog_e2e_verify_test.go` 已是 Go 版替代 |

### `vdi-research/` — VDI 内网调研产物（11 个）

一次性逆向/抓数产物，含硬编码内网 IP（`10.62.0.79:6060`）和 admin 凭据。生产系统已通过正常接口对接 VDI，这套调研脚本不再使用。

| 文件 | 说明 |
|------|------|
| `get_vdi_data.bat` / `.py` / `.sh` | VDI 资源组/虚拟机数据获取（三平台同功能） |
| `parse_vdi_api.py` | VDI API 响应解析 |
| `vdi_diagnostic_tool.bat` / `.sh` | VDI 诊断工具 |
| `vdi_groups.json` / `vdi_groups_raw.json` | 一次性抓取的 VDI 数据样本 |
| `check_vdi_local_data.sql` | 本地 VDI 数据校验 SQL |
| `README_VDI_DATA.md` / `VDI_DIAGNOSTIC_GUIDE.md` | 调研配套文档 |

> 注：`vdi/test_standalone/main.go` 是新整理的 VDI 独立测试工具，**不在归档内**，保留在 `vdi/`。

## 与 `.archive-migrate-2026-06-15/` 的关系

- `.archive-migrate-2026-06-15/` —— 归档一次性 DB 迁移脚本（`migrate/` 重命名），所有迁移已在生产 DB 执行过
- `.archive-scripts-2026-08-12/` —— 归档过时/一次性运维脚本（本次整理新增）

两者互不相干，保留各自 README 说明归档背景。