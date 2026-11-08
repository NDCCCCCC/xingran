#!/bin/bash

# ========================================
# VDI 数据获取工具
# 直接从 VDI 系统获取资源组和虚拟机信息
# ========================================

# 配置 VDI 服务器信息
VDI_URL="${VDI_URL:-https://10.62.0.79:6060}"
VDI_USERNAME="${VDI_USERNAME:-admin}"

# 提示输入密码
read -sp "请输入 VDI 密码: " VDI_PASSWORD
echo

if [ -z "$VDI_PASSWORD" ]; then
    echo "错误: 密码不能为空"
    exit 1
fi

echo ""
echo "========================================"
echo "VDI 数据获取工具"
echo "========================================"
echo "服务器: $VDI_URL"
echo "用户: $VDI_USERNAME"
echo "========================================"
echo ""

# 创建临时目录
TEMP_DIR=$(mktemp -d)
trap "rm -rf $TEMP_DIR" EXIT

# ============================================================
# 步骤 1: 认证
# ============================================================
echo "[步骤 1/3] 正在认证..."

# 发送认证请求
AUTH_RESPONSE=$(curl -k -s -X POST "$VDI_URL/v1/auth/tokens" \
    -H "Content-Type: application/json" \
    -d "{\"auth\":{\"name\":\"$VDI_USERNAME\",\"password\":\"$VDI_PASSWORD\"}}")

# 保存原始响应
echo "$AUTH_RESPONSE" > "$TEMP_DIR/auth_response.json"

# 检查认证是否成功
ERROR_CODE=$(echo "$AUTH_RESPONSE" | grep -o '"error_code":[0-9]*' | cut -d':' -f2)

if [ "$ERROR_CODE" != "0" ]; then
    echo "❌ 认证失败"
    echo "$AUTH_RESPONSE"
    exit 1
fi

echo "✅ 认证成功"
echo ""

# 提取 token
TOKEN=$(echo "$AUTH_RESPONSE" | grep -o '"auth_token":"[^"]*"' | cut -d'"' -f4)

if [ -z "$TOKEN" ]; then
    echo "❌ 无法提取 token"
    exit 1
fi

# ============================================================
# 步骤 2: 获取资源组
# ============================================================
echo "[步骤 2/3] 正在获取资源组..."

# 获取资源组
GROUPS_RESPONSE=$(curl -k -s -X GET "$VDI_URL/v1/resources_group" \
    -H "Auth-Token: $TOKEN")

# 保存原始响应
echo "$GROUPS_RESPONSE" > "$TEMP_DIR/groups_response.json"

# 检查请求是否成功
ERROR_CODE=$(echo "$GROUPS_RESPONSE" | grep -o '"error_code":[0-9]*' | cut -d':' -f2)

if [ "$ERROR_CODE" != "0" ]; then
    echo "❌ 获取资源组失败"
    echo "$GROUPS_RESPONSE"
    exit 1
fi

echo "✅ 资源组获取成功"
echo ""

# 显示资源组信息
echo "========================================"
echo "资源组列表"
echo "========================================"

if command -v jq &> /dev/null; then
    # 使用 jq 格式化输出
    echo "$GROUPS_RESPONSE" | jq -r '
        "资源组总数: \(.data | length)",
        "",
        (.data[] |
            "ID: \(.id)\n名称: \(.name)\n描述: \(.note)\n启用: \(.enable)\n"
        )
    '
else
    # 简单文本输出
    echo "$GROUPS_RESPONSE" | python3 -c "
import json, sys
data = json.load(sys.stdin)
groups = data['data']
print(f'资源组总数: {len(groups)}')
print()
for g in groups:
    print(f'ID: {g[\"id\"]}')
    print(f'名称: {g[\"name\"]}')
    print(f'描述: {g.get(\"note\", \"\")}')
    print(f'启用: {g[\"enable\"]}')
    print()
"
fi

echo "========================================"
echo ""

# ============================================================
# 步骤 3: 获取每个资源组的虚拟机
# ============================================================
echo "[步骤 3/3] 正在获取虚拟机信息..."
echo ""

TOTAL_VMS=0

# 解析资源组 ID
if command -v jq &> /dev/null; then
    # 使用 jq 解析
    GROUP_IDS=($(echo "$GROUPS_RESPONSE" | jq -r '.data[].id'))
    GROUP_NAMES=($(echo "$GROUPS_RESPONSE" | jq -r '.data[].name'))
    GROUP_ENABLES=($(echo "$GROUPS_RESPONSE" | jq -r '.data[].enable'))

    i=0
    for group_id in "${GROUP_IDS[@]}"; do
        group_name="${GROUP_NAMES[$i]}"
        group_enable="${GROUP_ENABLES[$i]}"
        i=$((i+1))

        if [ "$group_enable" != "1" ]; then
            echo "⏭️  跳过未启用的资源组: $group_name"
            continue
        fi

        echo "资源组: $group_name (ID: $group_id)"

        # 获取虚拟机列表
        VM_RESPONSE=$(curl -k -s -X GET "$VDI_URL/v1/resource/servers?rcid=$group_id&page=1&page_size=100" \
            -H "Auth-Token: $TOKEN")

        # 保存响应
        echo "$VM_RESPONSE" > "$TEMP_DIR/vms_${group_id}.json"

        # 检查是否成功
        VM_ERROR=$(echo "$VM_RESPONSE" | jq -r '.error_code // "null"')

        if [ "$VM_ERROR" != "0" ] && [ "$VM_ERROR" != "null" ]; then
            echo "  ❌ 获取虚拟机失败: $(echo "$VM_RESPONSE" | jq -r '.error_message')"
            echo ""
            continue
        fi

        # 统计虚拟机数量
        VM_COUNT=$(echo "$VM_RESPONSE" | jq -r '.data.data | length')
        TOTAL_VMS=$((TOTAL_VMS + VM_COUNT))

        echo "  虚拟机数量: $VM_COUNT"

        if [ "$VM_COUNT" -gt 0 ]; then
            echo "  前 10 个虚拟机:"
            echo "$VM_RESPONSE" | jq -r '.data.data[:10] | .[] | "    - ID: \(._id) 名称: \(._id) 名称: \(.vm_name) IP: \(.ip) 状态: \(.status)"'
        fi

        echo ""
    done
else
    # 使用 Python 解析
    python3 << PYTHON_SCRIPT
import json
import subprocess
import urllib.request

with open('$TEMP_DIR/groups_response.json', 'r') as f:
    groups_data = json.load(f)

groups = groups_data['data']
total_vms = 0

token = """ + '"' + "$TOKEN" + '"' + """
base_url = "$VDI_URL"

for group in groups:
    if group['enable'] != '1':
        print(f"⏭️  跳过未启用的资源组: {group['name']}")
        continue

    print(f"资源组: {group['name']} (ID: {group['id']})")

    url = f"{base_url}/v1/resource/servers?rcid={group['id']}&page=1&page_size=100"
    req = urllib.request.Request(url)
    req.add_header('Auth-Token', token)

    try:
        with urllib.request.urlopen(req, context=ssl._create_unverified_context()) as response:
            vm_data = json.load(response)

        if vm_data.get('error_code') != 0:
            print(f"  ❌ 获取虚拟机失败: {vm_data.get('error_message')}")
            continue

        vms = vm_data['data']['data']
        vm_count = len(vms)
        total_vms += vm_count

        print(f"  虚拟机数量: {vm_count}")

        if vm_count > 0:
            print("  前 10 个虚拟机:")
            for vm in vms[:10]:
                print(f"    - ID: {vm.get('_id')} 名称: {vm.get('vm_name')} IP: {vm.get('ip')} 状态: {vm.get('status')}")
    except Exception as e:
        print(f"  ❌ 获取虚拟机失败: {e}")

    print()

print('=' * 40)
print(f"总计: {total_vms} 个虚拟机")
print('=' * 40)
PYTHON_SCRIPT
fi

echo ""
echo "========================================"
echo "数据已保存到临时目录: $TEMP_DIR"
echo "========================================"
echo ""

# 复制原始数据到当前目录
cp "$TEMP_DIR/groups_response.json" "./vdi_groups_data.json"
echo "✅ 资源组数据已保存到: ./vdi_groups_data.json"
