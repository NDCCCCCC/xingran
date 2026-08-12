#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
VMP服务器登录和API验证脚本

功能：
1. 自动登录VMP服务器获取认证Cookie
2. 使用Cookie调用业务API获取虚拟机列表
"""

import os
import sys
import json
import re
import hashlib
import base64
import requests
from urllib3 import disable_warnings
from urllib3.exceptions import InsecureRequestWarning

# 禁用SSL警告
disable_warnings(InsecureRequestWarning)

# 默认配置
DEFAULT_ENDPOINT = "https://10.62.0.72"
LOGIN_API = "/vapi/extjs/access/ticket"
VM_API = "/vapi/extjs/cluster/vms"


class VMPClient:
    """VMP客户端 - 支持登录和API调用"""

    def __init__(self, endpoint):
        """
        初始化VMP客户端

        Args:
            endpoint: VMP服务器地址
        """
        self.endpoint = endpoint.rstrip('/')
        self.session = requests.Session()
        self.auth_cookie = None

        # 配置SSL（跳过验证，处理深信服SSL兼容性问题）
        self._configure_ssl()

    def _configure_ssl(self):
        """配置SSL选项"""
        from requests.adapters import HTTPAdapter
        from urllib3.util.ssl_ import create_urllib3_context
        import ssl

        try:
            ssl_context = create_urllib3_context()
            ssl_context.check_hostname = False
            ssl_context.verify_mode = ssl.CERT_NONE
            ssl_context.set_ciphers('DEFAULT:@SECLEVEL=1')
            adapter = HTTPAdapter(max_retries=3)
            self.session.mount('https://', adapter)
            self.session.verify = False
        except Exception as e:
            print(f"[WARN] SSL配置失败: {e}")
            self.session.verify = False

    def login(self, username, password):
        """
        登录VMP服务器

        Args:
            username: 用户名
            password: 明文密码

        Returns:
            bool: 登录是否成功
        """
        url = f"{self.endpoint}{LOGIN_API}"

        # 准备登录参数
        payload = {
            'username': username,
            'password': self._encrypt_password(password),
            'privacy': '1'
        }

        headers = {
            'Content-Type': 'application/x-www-form-urlencoded; charset=UTF-8',
            'Origin': self.endpoint,
            'Referer': f"{self.endpoint}/login.pl",
            'X-Requested-With': 'XMLHttpRequest'
        }

        try:
            print(f"[LOGIN] Logging in to {url}")
            response = self.session.post(url, data=payload, headers=headers, timeout=30, allow_redirects=False)

            if response.status_code == 200:
                # 从Set-Cookie头提取新的LoginAuthCookie
                set_cookie = response.headers.get('Set-Cookie', '')
                if set_cookie:
                    match = re.search(r'LoginAuthCookie=([^;]+)', set_cookie)
                    if match:
                        self.auth_cookie = match.group(1)
                        print(f"[LOGIN] Success! Got new Cookie (length: {len(self.auth_cookie)})")
                        return True

            print(f"[LOGIN] Failed! Status: {response.status_code}")
            print(f"[LOGIN] Response: {response.text[:200]}")
            return False

        except Exception as e:
            print(f"[ERROR] Login failed: {e}")
            return False

    def _encrypt_password(self, password):
        """
        加密密码（模拟前端加密逻辑）

        注意：这是简化版本，实际加密算法可能更复杂
        深信服可能使用RSA或其他加密方式

        Args:
            password: 明文密码

        Returns:
            str: 加密后的密码
        """
        # TODO: 实现实际的加密算法
        # 目前返回原始密码用于测试
        # 实际需要分析前端JS代码来确定加密方式

        # 临时方案：直接返回密码（某些情况下服务器可能接受明文）
        return password

        # 可能的加密方式（需要进一步分析）：
        # 1. RSA公钥加密
        # 2. AES对称加密
        # 3. 自定义加密算法

    def get_grouped_vms(self):
        """
        获取分组的虚拟机列表

        Returns:
            dict: API响应数据
        """
        if not self.auth_cookie:
            print("[ERROR] Not logged in! Please login first.")
            return None

        url = f"{self.endpoint}{VM_API}"
        params = {
            'group_type': 'group',
            'sort_type': '',
            'desc': '1'
        }

        headers = {
            'Cookie': f'LoginAuthCookie={self.auth_cookie}'
        }

        try:
            print(f"[API] Fetching VM list from {url}")
            response = self.session.get(url, params=params, headers=headers, timeout=30)
            response.raise_for_status()
            return response.json()
        except requests.exceptions.RequestException as e:
            print(f"[ERROR] API call failed: {e}")
            return None

    def display_results(self, data):
        """显示API响应结果"""
        if not data:
            return

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

        for group in groups:
            vms = group.get('data', [])
            total_vms += len(vms)

            for vm in vms:
                status = vm.get('status', 'unknown')
                if status == 'running':
                    running_vms += 1
                elif status == 'stopped':
                    stopped_vms += 1

        print(f"Total VMs: {total_vms}")
        print(f"Running: {running_vms}")
        print(f"Stopped: {stopped_vms}")
        print("=" * 60 + "\n")

        # 显示分组概览
        print("=" * 60)
        print("[GROUPS] Virtual Machine Groups")
        print("=" * 60)

        for i, group in enumerate(groups, 1):
            group_name = group.get('name', 'Unknown')
            group_id = group.get('id', '')
            vms = group.get('data', [])

            running = sum(1 for vm in vms if vm.get('status') == 'running')
            stopped = sum(1 for vm in vms if vm.get('status') == 'stopped')

            print(f"{i}. {group_name} (ID: {group_id})")
            print(f"   Total: {len(vms)} | Running: {running} | Stopped: {stopped}")

        print("\n" + "=" * 60)
        print("[SUCCESS] Verification completed!")
        print("=" * 60)


def main():
    """主函数"""
    # 从环境变量读取配置
    endpoint = os.getenv('VMP_ENDPOINT', DEFAULT_ENDPOINT)
    username = os.getenv('VMP_USERNAME', 'admin')
    password = os.getenv('VMP_PASSWORD', 'sangfor@2020')

    print("=" * 60)
    print("VMP Server Login & API Verification Tool")
    print("=" * 60)
    print(f"Server: {endpoint}")
    print(f"Username: {username}")
    print("=" * 60 + "\n")

    # 创建客户端
    client = VMPClient(endpoint)

    # 步骤1: 登录
    print("[STEP 1] Login to VMP Server")
    print("-" * 60)

    login_success = client.login(username, password)

    if not login_success:
        print("\n[FAILED] Login failed. Please check credentials.")
        print("\nPossible issues:")
        print("1. Wrong username or password")
        print("2. Network connectivity problem")
        print("3. SSL/TLS handshake failure")
        print("4. Account locked or expired")
        sys.exit(1)

    print()

    # 步骤2: 调用API
    print("[STEP 2] Fetch VM List")
    print("-" * 60)

    data = client.get_grouped_vms()

    if data:
        client.display_results(data)

        # 可选：保存响应
        if os.getenv('VMP_SAVE_RESPONSE', 'false').lower() == 'true':
            import time
            filename = f"vmp_response_{int(time.time())}.json"
            with open(filename, 'w', encoding='utf-8') as f:
                json.dump(data, f, ensure_ascii=False, indent=2)
            print(f"\n[SAVE] Full response saved to: {filename}")
    else:
        print("[FAILED] Failed to fetch VM list.")


if __name__ == '__main__':
    main()
