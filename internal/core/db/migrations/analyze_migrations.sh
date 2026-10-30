#!/bin/bash
# 迁移文件分析脚本

echo "======================================"
echo "迁移文件分析"
echo "======================================"
echo ""

# 统计总数
TOTAL=$(ls *.sql 2>/dev/null | wc -l)
echo "总文件数: $TOTAL"
echo ""

# 分类统计
echo "文件分类:"
echo "--------------------------------------"

# 修复类 (fix, fix_, constraint, unique, drop_)
FIX_COUNT=$(ls *.sql 2>/dev/null | grep -E "fix_|^fix|constraint|unique|drop_" | wc -l)
echo "修复类: $FIX_COUNT"

# 初始数据类 (menu, dict, insert, init, create_table)
INIT_COUNT=$(ls *.sql 2>/dev/null | grep -E "menu|dict|insert|init|create.*table" | wc -l)
echo "初始数据类: $INIT_COUNT"

# 性能优化类 (index, performance, optimize)
PERF_COUNT=$(ls *.sql 2>/dev/null | grep -E "index|performance|optimize" | wc -l)
echo "性能优化类: $PERF_COUNT"

# 功能新增类 (add_, create_)
FEATURE_COUNT=$(ls *.sql 2>/dev/null | grep -E "^0[0-9]{2}_add_|^0[0-9]{2}_create_" | wc -l)
echo "功能新增类: $FEATURE_COUNT"

echo ""
echo "详细文件列表:"
echo "--------------------------------------"

# 列出修复类文件
echo ""
echo "=== 修复类文件 (可归档) ==="
ls *.sql 2>/dev/null | grep -E "fix_|^fix|constraint|unique|drop_" | sort

# 列出初始数据文件
echo ""
echo "=== 初始数据文件 (需保留) ==="
ls *.sql 2>/dev/null | grep -E "menu|dict|insert|init" | sort

# 列出最新功能迁移 (100+)
echo ""
echo "=== 最新功能迁移 (100+) ==="
ls *.sql 2>/dev/null | grep -E "^[12][0-9]{2}_" | sort

echo ""
echo "======================================"
echo "分析完成"
echo "======================================"
