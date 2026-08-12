#!/usr/bin/env bash
# 归档 .planning 下已完成的 GSD 产物（phases / quick / debug 根）。
# 沿用现有惯例：phases -> milestones/vX.X-phases/；quick -> quick-archive/；debug 根 -> debug/archived/。
# 用法：bash archive_planning.sh
set -euo pipefail

# 切到项目根（脚本位于 .planning/tmp/）
cd "$(dirname "$0")/../.."

TS=$(date +%Y%m%d-%H%M%S)
BACKUP=".archive/planning-backup-$TS.tar.gz"
INDEX=".planning/ARCHIVE-INDEX.md"

# phase 目录名 -> milestone（KEEP=保留；infra=跨版本技术债）
phase_milestone() {
  case "$1" in
    11-*) echo "v1.4";;
    12-*|13-*|14-*|15-*) echo "v1.5";;
    16-*) echo "v1.6";;
    17-*) echo "v1.7";;
    18-*) echo "v1.8";;
    19-*|20-*) echo "v1.9";;
    22*) echo "v1.12";;
    23-*) echo "v1.11";;
    26-*) echo "v1.13";;
    27-*|32-*) echo "v1.14";;
    28-*|39-*) echo "v1.15";;
    40-*|41-*) echo "v1.16";;
    42-*|43-*|44-*|45-*|46-*|47-*) echo "v1.17";;
    48-*|49-*) echo "v1.18";;
    50-*|51-*|52-*|53-*|54-*|55-*) echo "v1.19";;
    56-*) echo "KEEP";;
    *) echo "infra";;
  esac
}

echo "### Step 1/5: 打包备份"
mkdir -p .archive
tar -czf "$BACKUP" .planning/phases .planning/quick .planning/debug 2>/dev/null
echo "  备份: $BACKUP ($(du -h "$BACKUP" | cut -f1))"

# 记录映射用于 INDEX
: > .planning/tmp/_phase_map.txt

echo ""
echo "### Step 2/5: 归档 phases -> milestones/vX.X-phases/"
phases_moved=0
for d in .planning/phases/*/; do
  [ -d "$d" ] || continue
  name=$(basename "$d")
  m=$(phase_milestone "$name")
  if [ "$m" = "KEEP" ]; then
    echo "  [保留] $name (v1.20 最新 milestone)"
    continue
  fi
  dest=".planning/milestones/$m-phases"
  mkdir -p "$dest"
  mv "$d" "$dest/"
  printf '%s\t%s\n' "$name" "$m" >> .planning/tmp/_phase_map.txt
  phases_moved=$((phases_moved+1))
done
echo "  phases 移动: $phases_moved 个目录"

echo ""
echo "### Step 3/5: 归档 quick -> .planning/quick-archive/"
mkdir -p .planning/quick-archive
quick_moved=0
for d in .planning/quick/*/; do
  [ -d "$d" ] || continue
  name=$(basename "$d")
  case "$name" in
    202607*|2607*|default-theme-config) continue;;
    *) mv "$d" ".planning/quick-archive/"; quick_moved=$((quick_moved+1));;
  esac
done
echo "  quick 移动: $quick_moved 个目录"

echo ""
echo "### Step 4/5: 归档 debug 根 -> .planning/debug/archived/"
mkdir -p .planning/debug/archived
debug_moved=0
for f in .planning/debug/*.md; do
  [ -f "$f" ] || continue
  mv "$f" ".planning/debug/archived/"
  debug_moved=$((debug_moved+1))
done
echo "  debug 根移动: $debug_moved 个 md（resolved/ 保持不动）"

echo ""
echo "### Step 5/5: 生成归档索引 $INDEX"
{
  echo "# 规划文档归档索引"
  echo ""
  echo "- **归档时间**: $TS"
  echo "- **备份包**: \`$BACKUP\`（移动前的完整快照，恢复时解包覆盖即可）"
  echo "- **归档范围**: 已完成 GSD 产物（v1.19 及以前 phases、≤2026-06 quick、debug 根历史记录）"
  echo ""
  echo "## Phase → Milestone 映射（$phases_moved 个）"
  echo ""
  echo "| Phase 目录 | 归档到 |"
  echo "|------------|--------|"
  sort -t$'\t' -k2,2 -k1,1 .planning/tmp/_phase_map.txt | while IFS=$'\t' read -r name m; do
    echo "| \`$name\` | \`milestones/$m-phases/$name/\` |"
  done
  echo ""
  echo "## Quick 归档（$quick_moved 个 → \`.planning/quick-archive/\`）"
  echo ""
  echo "保留为活跃（2026-07 起 + 配置类）的 quick 目录仍在 \`.planning/quick/\`。"
  echo ""
  echo "## Debug 归档（$debug_moved 个 → \`.planning/debug/archived/\`）"
  echo ""
  echo "\`.planning/debug/resolved/\`（已解决）保持不动。"
  echo ""
  echo "## 保留（活跃）"
  echo ""
  echo "- **Phase 56 (v1.20)** — \`.planning/phases/56-vlan-...\`（最新 milestone，保留便于回溯）"
  echo "- **Quick** — 2026-07 起的任务 + \`default-theme-config\`"
  echo "- **状态文件** — \`STATE.md / ROADMAP.md / MILESTONES.md / PROJECT.md / HANDOFF.json / RETROSPECTIVE.md\`"
  echo "- **其他规划目录** — \`notes/ reports/ research/ reviews/ seeds/ spikes/ todos/\`"
  echo ""
  echo "> 注：STATE.md/ROADMAP.md 中部分对 \`.planning/phases/XX-...\` 的历史叙述路径已随归档迁移，"
  echo "> 如需查阅请到对应 \`milestones/vX.X-phases/\` 子目录。这些是历史叙述引用，不影响 GSD 运行时状态。"
} > "$INDEX"
rm -f .planning/tmp/_phase_map.txt

echo "  索引已生成"
echo ""
echo "### 完成。"
echo "  备份: $BACKUP"
echo "  索引: $INDEX"
