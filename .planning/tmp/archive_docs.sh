#!/usr/bin/env bash
# 归档 docs/ 下的过程性报告（完成/进度/审查/总结/验证报告 + 冗余简化文档 + 空文件）。
# 保留：核心架构规范、参考指南、RPA 设计/规范、设计计划。
set -uo pipefail
cd "$(dirname "$0")/../.."   # 项目根
cd docs

TS=$(date +%Y%m%d-%H%M%S)
mkdir -p archive

FILES=(
  "CODE_OPTIMIZATION_SUMMARY.md"
  "code-review-report.md"
  "Swagger文档实施总结.md"
  "user_service_N+1优化报告.md"
  "代码简化和优化建议.md"
  "代码简化优化完成报告.md"
  "代码简化优化总结.md"
  "统一错误处理实施总结.md"
  "项目验证报告.md"
  "RPA-Worker-完成报告.md"
  "RPA系统开发进度.md"
  "RPA系统完成状态报告.md"
  "RPA系统完整性检查清单.md"
)

echo "### Step 1/3: 打包备份"
tar -czf "archive/docs-backup-$TS.tar.gz" "${FILES[@]}" "security/login-encryption-security.md" 2>/dev/null
echo "  备份: archive/docs-backup-$TS.tar.gz ($(du -h "archive/docs-backup-$TS.tar.gz" | cut -f1))"

echo ""
echo "### Step 2/3: 移动过程性报告 -> archive/"
for f in "${FILES[@]}"; do
  if [ -f "$f" ]; then mv "$f" "archive/" && echo "  [OK] $f"; else echo "  [跳过-不存在] $f"; fi
done
if [ -f "security/login-encryption-security.md" ]; then
  mv "security/login-encryption-security.md" "archive/login-encryption-security.md"
  echo "  [OK] security/login-encryption-security.md (1行空文件)"
fi

echo ""
echo "### Step 3/3: 生成索引"
{
  echo "# docs 过程性文档归档索引"
  echo ""
  echo "- **归档时间**: $TS"
  echo "- **备份包**: \`archive/docs-backup-$TS.tar.gz\`"
  echo "- **归档原则**: 过程性报告（完成/进度/审查/实施总结/验证/优化报告 + 冗余简化文档）；保留核心架构、规范、参考指南、设计计划"
  echo ""
  echo "## 归档文件（${#FILES[@]} 个报告 + 1 空文件）"
  echo ""
  for f in "${FILES[@]}"; do echo "- \`$f\`"; done
  echo "- \`login-encryption-security.md\`（原 \`security/\`，仅 1 行空文件）"
  echo ""
  echo "## docs/ 保留（核心 + 指南 + 设计，未动）"
  echo ""
  echo "项目概述和架构设计 / 开发规范 / API响应规范 / 安全和认证设计（国密）/ 数据库设计 / 部署和运维文档 / deployment/(2) / secret-management / cache_usage / 缓存架构演进 / 缓存重试机制 / gormutil工具说明 / 时间工具函数 / EXCEL_IMPORT_GUIDE / interface-type-safe-migration-guide / encryption-config-sync / 跨模块选择器权限处理规范 / 上传下载功能设计 / 启动流程清单 / TypeScript代码质量优化方案 / 网络设备连接状态机设计 / code-fix-guide / RPA系统设计方案 / RPA-数据格式规范 / RPA-Worker-API认证方案-待办 / plans/(2) / agent-test-plan"
} > archive/ARCHIVE-INDEX.md
echo "  索引: archive/ARCHIVE-INDEX.md"
echo ""
echo "### 完成。"
