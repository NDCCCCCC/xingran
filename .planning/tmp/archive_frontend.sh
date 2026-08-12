#!/usr/bin/env bash
# 归档前端目录的历史 md：.planning/ 整体 + 根目录散落报告 + src 旁说明。
# 目标：xingran-react-frontend/.archive/{planning,reports}/，保留 README.md。
set -euo pipefail

cd "$(dirname "$0")/../.."        # 项目根
cd xingran-react-frontend

TS=$(date +%Y%m%d-%H%M%S)
mkdir -p .archive/reports

# 待归档的散落报告（相对前端根）
ROOT_REPORTS=(
  "14-04-SUMMARY.md"
  "14-05b-SUMMARY.md"
  "FRONTEND_OPTIMIZATION_SUMMARY.md"
  "FRONTEND_PERFORMANCE_AUDIT.md"
  "FRONTEND_PERFORMANCE_RE-EVALUATION.md"
)
SRC_REPORTS=(
  "src/components/layout/OPTIMIZATION_SUMMARY.md:layout-OPTIMIZATION_SUMMARY.md"
  "src/pages/operations/floors/REFACTOR.md:floors-REFACTOR.md"
)

echo "### Step 1/4: 打包备份"
tar -czf ".archive/frontend-backup-$TS.tar.gz" \
  .planning "${ROOT_REPORTS[@]}" \
  src/components/layout/OPTIMIZATION_SUMMARY.md \
  src/pages/operations/floors/REFACTOR.md 2>/dev/null
echo "  备份: .archive/frontend-backup-$TS.tar.gz ($(du -h ".archive/frontend-backup-$TS.tar.gz" | cut -f1))"

echo ""
echo "### Step 2/4: 归档 .planning/ -> .archive/planning/"
mv .planning .archive/planning
echo "  前端 .planning 整体移入 .archive/planning/"

echo ""
echo "### Step 3/4: 归档散落报告 -> .archive/reports/"
for f in "${ROOT_REPORTS[@]}"; do
  mv "$f" ".archive/reports/"
  echo "  [根]    $f"
done
for pair in "${SRC_REPORTS[@]}"; do
  src="${pair%%:*}"; dst="${pair##*:}"
  mv "$src" ".archive/reports/$dst"
  echo "  [src]   $src -> reports/$dst"
done

echo ""
echo "### Step 4/4: 生成归档索引"
{
  echo "# 前端文档归档索引"
  echo ""
  echo "- **归档时间**: $TS"
  echo "- **备份包**: \`.archive/frontend-backup-$TS.tar.gz\`"
  echo "- **保留**: 根目录 \`README.md\`"
  echo ""
  echo "## .planning/（前端 GSD 工作流遗留，整体移入 .archive/planning/）"
  echo ""
  echo "- STATE.md / debug/ / quick/（最后活动 2026-06，历史产物）"
  echo ""
  echo "## reports/（散落报告 + 源码旁说明）"
  echo ""
  echo "| 文件 | 来源 |"
  echo "|------|------|"
  for f in "${ROOT_REPORTS[@]}"; do echo "| \`$f\` | 前端根目录 |"; done
  for pair in "${SRC_REPORTS[@]}"; do
    src="${pair%%:*}"; dst="${pair##*:}"
    echo "| \`$dst\` | \`$src\` |"
  done
  echo ""
  echo "> 这些是 Phase 14 (v1.5) 及前端性能/优化历史报告，归档保留备查。"
} > .archive/ARCHIVE-INDEX.md
echo "  索引: .archive/ARCHIVE-INDEX.md"

echo ""
echo "### 完成。"
