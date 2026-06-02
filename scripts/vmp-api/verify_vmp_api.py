#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
VMP服务器API验证脚本

用于验证深信服VMP服务器的虚拟机分组API功能。
支持获取分组虚拟机列表和实时运行状态。
"""

import os
import sys
import json
import requests
import ssl
from urllib3 import disable_warnings
from urllib3.exceptions import InsecureRequestWarning

# 禁用SSL警告
disable_warnings(InsecureRequestWarning)

# 默认配置
DEFAULT_ENDPOINT = "https://10.62.0.72"
API_PATH = "/vapi/extjs/cluster/vms"


class VMPAPIClient:
    """VMP API客户端"""

    def __init__(self, endpoint, auth_cookie=None):
        """
        初始化VMP API客户端

        Args:
            endpoint: VMP服务器地址，如 https://10.62.0.72
            auth_cookie: 认证Cookie (LoginAuthCookie值)
        """
        self.endpoint = endpoint.rstrip('/')
        self.auth_cookie = auth_cookie
        self.session = requests.Session()

        # 配置更宽松的SSL选项来处理握手失败
        from requests.adapters import HTTPAdapter
        from urllib3.util.ssl_ import create_urllib3_context

        # 创建自定义SSL上下文，支持更旧的协议
        ssl_context = create_urllib3_context()
        ssl_context.check_hostname = False
        ssl_context.verify_mode = ssl.CERT_NONE

        # 设置可选的加密套件（更广泛的兼容性）
        try:
            ssl_context.set_ciphers('DEFAULT:@SECLEVEL=1')
        except:
            pass  # 如果设置失败，使用默认值

        # 创建自定义适配器
        adapter = HTTPAdapter(max_retries=3)
        self.session.mount('https://', adapter)
        self.session.verify = False  # 跳过证书验证

        # 设置默认请求头
        self.session.headers.update({
            'Accept': '*/*',
            'User-Agent': 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36',
            'X-Requested-With': 'XMLHttpRequest',
        })

        # 设置Cookie
        if self.auth_cookie:
            self.session.headers.update({
                'Cookie': f'LoginAuthCookie={self.auth_cookie}'
            })

    def get_grouped_vms(self):
        """
        获取分组的虚拟机列表

        Returns:
            dict: API响应数据
        """
        url = f"{self.endpoint}{API_PATH}"
        params = {
            'group_type': 'group',
            'sort_type': '',
            'desc': '1'
        }

        try:
            print(f"[SEND] Sending request: {url}")
            response = self.session.get(url, params=params, timeout=30)
            response.raise_for_status()
            return response.json()
        except requests.exceptions.RequestException as e:
            print(f"[ERROR] Request failed: {e}")
            sys.exit(1)

    def display_results(self, data):
        """
        美化显示API响应结果

        Args:
            data: API响应数据
        """
        print("\n" + "=" * 60)
        print("[SUMMARY] API Response Summary")
        print("=" * 60)

        success = data.get('success', 0)
        groups = data.get('data', [])

        print(f"Success: {success}")
        print(f"Groups count: {len(groups)}")

        # 统计虚拟机信息
        total_vms = 0
        running_vms = 0
        stopped_vms = 0
        group_stats = []

        for group in groups:
            vms = group.get('data', [])
            total_vms += len(vms)

            group_running = 0
            group_stopped = 0

            for vm in vms:
                status = vm.get('status', 'unknown')
                if status == 'running':
                    group_running += 1
                elif status == 'stopped':
                    group_stopped += 1

            running_vms += group_running
            stopped_vms += group_stopped

            group_stats.append({
                'name': group.get('name', 'Unknown'),
                'id': group.get('id', ''),
                'total': len(vms),
                'running': group_running,
                'stopped': group_stopped
            })

        print(f"Total VMs: {total_vms}")
        print(f"Running: {running_vms}")
        print(f"Stopped: {stopped_vms}")
        print("=" * 60 + "\n")

        # 显示分组详情
        print("=" * 60)
        print("[GROUPS] Virtual Machine Group Details")
        print("=" * 60)

        for i, stats in enumerate(group_stats, 1):
            print(f"\n[Group {i}] {stats['name']} (ID: {stats['id']})")
            print(f"  Total: {stats['total']} | Running: {stats['running']} | Stopped: {stats['stopped']}")

        # 显示每个分组的虚拟机列表（前5个）
        print("\n" + "=" * 60)
        print("[VMS] Virtual Machine List (showing first 5 per group)")
        print("=" * 60)

        for i, group in enumerate(groups, 1):
            group_name = group.get('name', 'Unknown')
            vms = group.get('data', [])

            print(f"\n[{group_name}]")
            display_count = min(5, len(vms))

            for j, vm in enumerate(vms[:display_count], 1):
                vm_name = vm.get('name', 'Unknown')
                status = vm.get('status', 'unknown')
                ip = vm.get('ip', 'N/A')
                cpu_ratio = vm.get('cpu_ratio', 'N/A')
                mem_ratio = vm.get('mem_ratio', 'N/A')
                associated_user = vm.get('associated_user', '')

                print(f"  {j}. {vm_name}")
                print(f"     Status: {status:10} | IP: {ip:15}")

                if cpu_ratio and cpu_ratio != '0':
                    print(f"     CPU: {cpu_ratio}% | Memory: {mem_ratio}%")

                if associated_user:
                    print(f"     User: {associated_user}")

            if len(vms) > display_count:
                print(f"  ... and {len(vms) - display_count} more VMs")

        print("\n" + "=" * 60)
        print("[SUCCESS] Verification completed!")
        print("=" * 60)


def main():
    """主函数"""
    # 从环境变量读取配置
    endpoint = os.getenv('VMP_ENDPOINT', DEFAULT_ENDPOINT)
    auth_cookie = os.getenv('VMP_AUTH_COOKIE', '')

    print("VMP Server API Verification Tool")
    print("=" * 60)
    print(f"Server: {endpoint}")

    if auth_cookie:
        print("Auth: Cookie provided")
    else:
        print("[WARNING] No authentication cookie provided, access may be denied")
        print("Please set VMP_AUTH_COOKIE environment variable")

    print("=" * 60 + "\n")

    # 创建客户端
    client = VMPAPIClient(endpoint, auth_cookie)

    # 获取数据
    try:
        data = client.get_grouped_vms()

        # 显示结果
        client.display_results(data)

        # 可选：保存完整响应到文件
        if os.getenv('VMP_SAVE_RESPONSE', 'false').lower() == 'true':
            import time
            filename = f"vmp_response_{int(time.time())}.json"
            with open(filename, 'w', encoding='utf-8') as f:
                json.dump(data, f, ensure_ascii=False, indent=2)
            print(f"\n[SAVE] Full response saved to: {filename}")

    except Exception as e:
        print(f"[ERROR] Error: {e}")
        sys.exit(1)


if __name__ == '__main__':
    main()
