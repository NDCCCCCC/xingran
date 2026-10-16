#!/bin/bash

# 手动触发部门到AD同步的脚本
# 用于填充 sys_dept_ou_mapping 表

set -e

# 配置
API_BASE_URL="http://localhost:9000"
USERNAME="admin"
PASSWORD="admin123"

# 颜色输出
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

echo -e "${YELLOW}=== 部门到AD同步触发工具 ===${NC}"
echo "此脚本将触发部门结构同步到AD，填充 sys_dept_ou_mapping 表"
echo ""

# Step 1: 登录获取token
echo -e "${YELLOW}Step 1: 登录系统...${NC}"
LOGIN_RESPONSE=$(curl -s -X POST "${API_BASE_URL}/login" \
  -H "Content-Type: application/json" \
  -d "{\"username\":\"${USERNAME}\",\"password\":\"${PASSWORD}\"}")

TOKEN=$(echo $LOGIN_RESPONSE | grep -o '"token":"[^"]*' | cut -d'"' -f4)

if [ -z "$TOKEN" ]; then
  echo -e "${RED}登录失败，无法获取token${NC}"
  echo "Response: $LOGIN_RESPONSE"
  exit 1
fi

echo -e "${GREEN}登录成功${NC}"
echo ""

# Step 2: 获取AD配置列表
echo -e "${YELLOW}Step 2: 获取AD配置列表...${NC}"
AD_CONFIGS=$(curl -s -X GET "${API_BASE_URL}/system/ad-config/list" \
  -H "Authorization: Bearer ${TOKEN}")

echo "AD配置列表:"
echo "$AD_CONFIGS" | grep -o '"id":"[^"]*","configName":"[^"]*' | sed 's/","id"//g' | sed 's/"configName":"/: /g'
echo ""

# Step 3: 选择AD配置ID（如果有多个，选择第一个启用的）
echo -e "${YELLOW}Step 3: 查找启用的AD配置...${NC}"
AD_CONFIG_ID=$(echo $AD_CONFIGS | grep -o '"id":"[^"]*","enabled":true' | head -1 | cut -d'"' -f4)

if [ -z "$AD_CONFIG_ID" ]; then
  echo -e "${RED}未找到启用的AD配置${NC}"
  exit 1
fi

echo -e "${GREEN}使用AD配置ID: ${AD_CONFIG_ID}${NC}"
echo ""

# Step 4: 触发部门同步
echo -e "${YELLOW}Step 4: 触发部门同步...${NC}"
SYNC_RESPONSE=$(curl -s -X POST "${API_BASE_URL}/system/ad/sync/dept-to-ad" \
  -H "Authorization: Bearer ${TOKEN}" \
  -H "Content-Type: application/json" \
  -d "{\"adConfigId\":\"${AD_CONFIG_ID}\"}")

echo "同步响应:"
echo "$SYNC_RESPONSE" | jq '.' 2>/dev/null || echo "$SYNC_RESPONSE"
echo ""

# Step 5: 等待几秒后检查同步状态
echo -e "${YELLOW}Step 5: 等待5秒后检查同步状态...${NC}"
sleep 5

STATUS_RESPONSE=$(curl -s -X GET "${API_BASE_URL}/system/ad/sync/dept-status/${AD_CONFIG_ID}" \
  -H "Authorization: Bearer ${TOKEN}")

echo "同步状态:"
echo "$STATUS_RESPONSE" | jq '.' 2>/dev/null || echo "$STATUS_RESPONSE"
echo ""

# Step 6: 检查映射表数据
echo -e "${YELLOW}Step 6: 验证映射表数据...${NC}"
echo "请手动连接数据库检查:"
echo "SELECT COUNT(*) FROM sys_dept_ou_mapping;"
echo "SELECT * FROM sys_dept_ou_mapping ORDER BY created_at DESC LIMIT 5;"
echo ""

echo -e "${GREEN}=== 同步触发完成 ===${NC}"
echo "如果同步成功，AD用户登录时应该能找到对应的部门"
echo "如果仍有问题，请检查:"
echo "1. AD连接配置是否正确"
echo "2. 系统中是否存在对应的部门"
echo "3. 查看应用日志获取详细错误信息"