#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
VDI 数据获取工具
直接从 VDI 系统获取资源组和虚拟机信息
"""

import json
import sys
import urllib.request
import ssl
from getpass import getpass

def main():
    # 配置
    VDI_URL = "https://10.62.0.79:6060"
    VDI_USERNAME = "admin"

    print("=" * 50)
    print("VDI 数据获取工具")
    print("=" * 50)
    print(f"服务器: {VDI_URL}")
    print(f"用户: {VDI_USERNAME}")
    print("=" * 50)
    print()

    # 获取密码
    password = getpass("请输入 VDI 密码: ")

    if not password:
        print("❌ 错误: 密码不能为空")
        sys.exit(1)

    # 创建 SSL 上下文（跳过证书验证）
    ssl_context = ssl._create_unverified_context()

    # ============================================================
    # 步骤 1: 认证
    # ============================================================
    print("[步骤 1/3] 正在认证...")

    auth_url = f"{VDI_URL}/v1/auth/tokens"
    auth_data = {
        "auth": {
            "name": VDI_USERNAME,
            "password": password
        }
    }

    try:
        req = urllib.request.Request(
            auth_url,
            data=json.dumps(auth_data).encode('utf-8'),
            headers={'Content-Type': 'application/json'}
        )

        with urllib.request.urlopen(req, context=ssl_context) as response:
            auth_response = json.load(response)

        # 保存原始响应
        with open('vdi_auth_response.json', 'w', encoding='utf-8') as f:
            json.dump(auth_response, f, indent=2, ensure_ascii=False)

        if auth_response.get('error_code') != 0:
            print(f"❌ 认证失败: {auth_response.get('error_message')}")
            sys.exit(1)

        token = auth_response['data']['token']['auth_token']
        print("✅ 认证成功")
        print()

    except Exception as e:
        print(f"❌ 认证请求失败: {e}")
        sys.exit(1)

    # ============================================================
    # 步骤 2: 获取资源组
    # ============================================================
    print("[步骤 2/3] 正在获取资源组...")

    groups_url = f"{VDI_URL}/v1/resources_group"

    try:
        req = urllib.request.Request(
            groups_url,
            headers={'Auth-Token': token}
        )

        with urllib.request.urlopen(req, context=ssl_context) as response:
            groups_response = json.load(response)

        # 保存原始响应
        with open('vdi_groups_response.json', 'w', encoding='utf-8') as f:
            json.dump(groups_response, f, indent=2, ensure_ascii=False)

        if groups_response.get('error_code') != 0:
            print(f"❌ 获取资源组失败: {groups_response.get('error_message')}")
            sys.exit(1)

        groups = groups_response['data']
        print(f"✅ 资源组获取成功")
        print()

        # 显示资源组
        print("=" * 50)
        print("资源组列表")
        print("=" * 50)
        print(f"资源组总数: {len(groups)}")
        print()

        for group in groups:
            print(f"ID: {group['id']}")
            print(f"名称: {group['name']}")
            print(f"描述: {group.get('note', '无')}")
            print(f"启用: {'是' if group['enable'] == '1' else '否'}")
            print("-" * 30)

        print()

    except Exception as e:
        print(f"❌ 获取资源组失败: {e}")
        sys.exit(1)

    # ============================================================
    # 步骤 3: 获取虚拟机
    # ============================================================
    print("[步骤 3/3] 正在获取虚拟机信息...")
    print()

    total_vms = 0
    all_vms = []

    for group in groups:
        if group['enable'] != '1':
            print(f"⏭️  跳过未启用的资源组: {group['name']}")
            continue

        print(f"资源组: {group['name']} (ID: {group['id']})")

        vms_url = f"{VDI_URL}/v1/resource/servers?rcid={group['id']}&page=1&page_size=100"

        try:
            req = urllib.request.Request(
                vms_url,
                headers={'Auth-Token': token}
            )

            with urllib.request.urlopen(req, context=ssl_context) as response:
                vms_response = json.load(response)

            if vms_response.get('error_code') != 0:
                print(f"  ❌ 获取虚拟机失败: {vms_response.get('error_message')}")
                continue

            vms = vms_response['data']['data']
            vm_count = len(vms)
            total_vms += vm_count
            all_vms.extend(vms)

            print(f"  虚拟机数量: {vm_count}")

            if vm_count > 0:
                print("  前 10 个虚拟机:")
                for vm in vms[:10]:
                    vm_id = vm.get('_id', 'N/A')
                    vm_name = vm.get('vm_name', 'N/A')
                    vm_ip = vm.get('ip', 'N/A')
                    vm_status = vm.get('status', 'N/A')
                    print(f"    - ID: {vm_id}, 名称: {vm_name}, IP: {vm_ip}, 状态: {vm_status}")

        except Exception as e:
            print(f"  ❌ 获取虚拟机失败: {e}")

        print()

    # 保存所有虚拟机数据
    if all_vms:
        with open('vdi_all_vms.json', 'w', encoding='utf-8') as f:
            json.dump(all_vms, f, indent=2, ensure_ascii=False)
        print(f"✅ 所有虚拟机数据已保存到: vdi_all_vms.json")

    print()
    print("=" * 50)
    print(f"总计: {total_vms} 个虚拟机")
    print("=" * 50)
    print()
    print("原始 JSON 数据已保存:")
    print("  - vdi_auth_response.json (认证响应)")
    print("  - vdi_groups_response.json (资源组)")
    print("  - vdi_all_vms.json (所有虚拟机)")

if __name__ == "__main__":
    main()
