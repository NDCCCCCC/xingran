#!/bin/bash

# VDI 诊断工具
# 用于检查 VDI 系统连接和数据

echo "========================================="
echo "VDI 系统诊断工具"
echo "========================================="
echo ""

# 检查环境变量
VDI_URL=${VDI_URL:-"https://10.62.0.79:6060"}
VDI_USERNAME=${VDI_USERNAME:-"admin"}

if [ -z "$VDI_PASSWORD" ]; then
    echo "❌ 错误: 请设置 VDI_PASSWORD 环境变量"
    echo ""
    echo "用法:"
    echo "  export VDI_PASSWORD=your_password"
    echo "  ./scripts/vdi_diagnostic_tool.sh"
    echo ""
    exit 1
fi

echo "VDI 服务器: $VDI_URL"
echo "用户名: $VDI_USERNAME"
echo ""

# 步骤 1: 测试认证
echo "步骤 1: 测试认证..."
AUTH_RESPONSE=$(curl -k -s -X POST "$VDI_URL/v1/auth/tokens" \
    -H "Content-Type: application/json" \
    -d "{\"auth\":{\"name\":\"$VDI_USERNAME\",\"password\":\"$VDI_PASSWORD\"}}")

echo "认证响应: $AUTH_RESPONSE" | head -c 200
echo ""

# 提取 token
TOKEN=$(echo $AUTH_RESPONSE | grep -o '"auth_token":"[^"]*"' | cut -d'"' -f4)

if [ -z "$TOKEN" ]; then
    echo "❌ 认证失败，无法获取 token"
    ERROR_CODE=$(echo $AUTH_RESPONSE | grep -o '"error_code":[0-9]*' | cut -d':' -f2)
    ERROR_MSG=$(echo $AUTH_RESPONSE | grep -o '"error_message":"[^"]*"' | cut -d'"' -f4)
    echo "错误代码: $ERROR_CODE"
    echo "错误信息: $ERROR_MSG"
    exit 1
fi

echo "✅ 认证成功，Token 长度: ${#TOKEN}"
TOKEN_FIRST_20="${TOKEN:0:20}..."
echo "Token (前20字符): $TOKEN_FIRST_20"
echo ""

# 步骤 2: 获取资源组
echo "步骤 2: 获取资源组列表..."
GROUPS_RESPONSE=$(curl -k -s -X GET "$VDI_URL/v1/resources_group" \
    -H "Auth-Token: $TOKEN")

echo "资源组响应: $GROUPS_RESPONSE" | head -c 500
echo ""

# 解析资源组数量
GROUP_COUNT=$(echo $GROUPS_RESPONSE | grep -o '"id"' | wc -l)
echo "找到 $GROUP_COUNT 个资源组"
echo ""

# 步骤 3: 提取资源组 ID 并获取虚拟机
echo "步骤 3: 获取每个资源组的虚拟机..."
echo ""

# 使用 JSON 解析工具（如果有 jq）
if command -v jq &> /dev/null; then
    echo "使用 jq 解析 JSON..."

    # 解析资源组
    GROUP_IDS=$(echo $GROUPS_RESPONSE | jq -r '.data[].id' 2>/dev/null)
    GROUP_NAMES=$(echo $GROUPS_RESPONSE | jq -r '.data[].name' 2>/dev/null)
    GROUP_ENABLES=$(echo $GROUPS_RESPONSE | jq -r '.data[].enable' 2>/dev/null)

    i=1
    for group_id in $GROUP_IDS; do
        # 获取对应的名称和启用状态
        group_name=$(echo $GROUP_NAMES | sed -n "${i}p")
        group_enable=$(echo $GROUP_ENABLES | sed -n "${i}p")

        echo "--- 资源组 $i: $group_name (ID=$group_id, 启用=$group_enable) ---"

        if [ "$group_enable" != "1" ]; then
            echo "⏭️  跳过未启用的资源组"
            echo ""
            i=$((i+1))
            continue
        fi

        # 获取虚拟机列表
        VM_RESPONSE=$(curl -k -s -X GET "$VDI_URL/v1/resource/servers?rcid=$group_id&page=1&page_size=100" \
            -H "Auth-Token: $TOKEN")

        echo "虚拟机响应: $VM_RESPONSE" | head -c 300
        echo ""

        # 解析虚拟机数量
        VM_TOTAL=$(echo $VM_RESPONSE | jq -r '.data.totalCount' 2>/dev/null)
        VM_COUNT=$(echo $VM_RESPONSE | jq -r '.data.data | length' 2>/dev/null)

        echo "总虚拟机数: $VM_TOTAL, 当前页返回: $VM_COUNT"

        if [ "$VM_COUNT" != "0" ] && [ "$VM_COUNT" != "null" ]; then
            echo "前 5 个虚拟机:"
            echo $VM_RESPONSE | jq -r '.data.data[:5] | .[] | "  - \(.vm_name) (ID=\(._id), IP=\(.ip), 状态=\(.status))"' 2>/dev/null
        else
            echo "⚠️  该资源组下没有虚拟机"
        fi

        echo ""
        i=$((i+1))
    done
else
    echo "⚠️  未找到 jq 工具，将显示原始响应"
    echo ""
    echo "建议安装 jq: sudo apt-get install jq (Ubuntu) 或 brew install jq (Mac)"
    echo ""

    # 手动解析第一个资源组
    GROUP_ID=$(echo $GROUPS_RESPONSE | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)

    if [ -n "$GROUP_ID" ]; then
        echo "尝试获取资源组 $GROUP_ID 的虚拟机..."
        VM_RESPONSE=$(curl -k -s -X GET "$VDI_URL/v1/resource/servers?rcid=$GROUP_ID&page=1&page_size=10" \
            -H "Auth-Token: $TOKEN")
        echo "虚拟机响应: $VM_RESPONSE"
    fi
fi

echo "========================================="
echo "诊断完成"
echo "========================================="
