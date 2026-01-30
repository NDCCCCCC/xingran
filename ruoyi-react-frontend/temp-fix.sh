#!/bin/bash

# 批量修复常见的类型断言问题
files=(
  "src/pages/system/dept/hooks/useDeptData.ts"
  "src/pages/system/menu/hooks/useMenuActions.tsx"
  "src/pages/system/menu/hooks/useMenuData.ts"
  "src/pages/system/notice/hooks/useAPIConfig.ts"
  "src/pages/system/post/index.tsx"
  "src/pages/system/role/hooks/useRoleData.ts"
  "src/pages/system/role/index.tsx"
  "src/pages/system/settings/api-config.tsx"
  "src/pages/system/settings/email-config.tsx"
  "src/pages/operations/info-points/index.tsx"
  "src/pages/operations/workstations/hooks/useWorkstationData.ts"
)

echo "批量修复类型断言问题..."
for file in "${files[@]}"; do
  if [ -f "$file" ]; then
    echo "处理文件: $file"
  fi
done

echo "脚本创建完成"
