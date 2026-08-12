#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
VMP服务器登录和API验证脚本（增强版）

支持多种密码加密方式:
1. 明文传递（临时方案）
2. RSA-2048加密
3. SM2国密加密

使用前请先运行browser_crypto_analyzer.js获取加密信息
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


class VMPEncryptor:
    """VMP密码加密器"""

    def __init__(self, encryption_method='plain', public_key=None):
        """
        初始化加密器

        Args:
            encryption_method: 加密方式 ('plain', 'rsa', 'sm2')
            public_key: 公钥（PEM格式或十六进制字符串）
        """
        self.encryption_method = encryption_method
        self.public_key = public_key
        self._init_cryptography()

    def _init_cryptography(self):
        """初始化加密库"""
        try:
            from Crypto.PublicKey import RSA
            from Crypto.Cipher import PKCS1_v1_5
            self.crypto_available = True
        except ImportError:
            print("[WARN] pycryptodome未安装，将使用明文传输")
            print("       安装: pip install pycryptodome")
            self.crypto_available = False

    def encrypt(self, password):
        """
        加密密码

        Args:
            password: 明文密码

        Returns:
            str: 加密后的密码
        """
        if self.encryption_method == 'plain':
            return self._encrypt_plain(password)
        elif self.encryption_method == 'rsa':
            return self._encrypt_rsa(password)
        elif self.encryption_method == 'sm2':
            return self._encrypt_sm2(password)
        else:
            return password

    def _encrypt_plain(self, password):
        """明文传递（临时方案）"""
        return password

    def _encrypt_rsa(self, password):
        """RSA加密"""
        if not self.crypto_available or not self.public_key:
            print("[WARN] RSA加密不可用，使用明文")
            return password

        try:
            from Crypto.PublicKey import RSA
            from Crypto.Cipher import PKCS1_v1_5
            import binascii

            # 加载公钥
            rsa_key = RSA.import_key(self.public_key)
            cipher = PKCS1_v1_5.new(rsa_key)

            # 加密
            encrypted = cipher.encrypt(password.encode('utf-8'))

            # 转换为十六进制（与前端JS一致）
            return binascii.hexlify(encrypted).decode('ascii')

        except Exception as e:
            print(f"[ERROR] RSA加密失败: {e}")
            return password

    def _encrypt_sm2(self, password):
        """SM2国密加密"""
        try:
            from gmssl import sm2

            if not self.public_key:
                print("[WARN] SM2公钥未提供")
                return password

            sm2_crypt = sm2.CryptSM2(
                public_key=self.public_key,
                private_key='',
                mode=sm2.EncryptMode
            )

            encrypted = sm2_crypt.encrypt(password.encode('utf-8'))
            return encrypted.hex()

        except ImportError:
            print("[WARN] gmssl未安装，pip install gmssl")
            return password
        except Exception as e:
            print(f"[ERROR] SM2加密失败: {e}")
            return password


class VMPClient:
    """VMP客户端 - 支持登录和API调用"""

    def __init__(self, endpoint, encryption_method='plain', public_key=None):
        """
        初始化VMP客户端

        Args:
            endpoint: VMP服务器地址
            encryption_method: 加密方式 ('plain', 'rsa', 'sm2')
            public_key: 公钥
        """
        self.endpoint = endpoint.rstrip('/')
        self.session = requests.Session()
        self.auth_cookie = None
        self.encryptor = VMPEncryptor(encryption_method, public_key)
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

        # 加密密码
        encrypted_password = self.encryptor.encrypt(password)
        print(f"[LOGIN] 加密方式: {self.encryptor.encryption_method}")
        print(f"[LOGIN] 密文长度: {len(encrypted_password)}")

        # 准备登录参数
        payload = {
            'username': username,
            'password': encrypted_password,
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
            response = self.session.post(url, data=payload, headers=headers,
                                        timeout=30, allow_redirects=False)

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

    def get_grouped_vms(self):
        """
        获取分组的虚拟机列表

        Returns:
            dict: API响应数据
        """
        if not self.auth_cookie:
            print("[ERROR] Not logged in!")
            return None

        url = f"{self.endpoint}{VM_API}"
        params = {'group_type': 'group', 'sort_type': '', 'desc': '1'}
        headers = {'Cookie': f'LoginAuthCookie={self.auth_cookie}'}

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

        total_vms = sum(len(g.get('data', [])) for g in groups)
        running_vms = sum(1 for g in groups for vm in g.get('data', []) if vm.get('status') == 'running')
        stopped_vms = sum(1 for g in groups for vm in g.get('data', []) if vm.get('status') == 'stopped')

        print(f"Total VMs: {total_vms}")
        print(f"Running: {running_vms}")
        print(f"Stopped: {stopped_vms}")
        print("=" * 60 + "\n")

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
    encryption_method = os.getenv('VMP_ENCRYPTION', 'plain')  # 'plain', 'rsa', 'sm2'
    public_key = os.getenv('VMP_PUBLIC_KEY', '')

    print("=" * 60)
    print("VMP Server Login & API Verification Tool (Advanced)")
    print("=" * 60)
    print(f"Server: {endpoint}")
    print(f"Username: {username}")
    print(f"Encryption: {encryption_method}")
    print("=" * 60 + "\n")

    # 创建客户端
    client = VMPClient(endpoint, encryption_method, public_key)

    print("[STEP 1] Login to VMP Server")
    print("-" * 60)

    if not client.login(username, password):
        print("\n[FAILED] Login failed.")
        print("\n提示:")
        print("1. 如果使用明文加密失败，需要先获取公钥")
        print("2. 运行 browser_crypto_analyzer.js 获取加密信息")
        print("3. 设置环境变量: VMP_ENCRYPTION=rsa, VMP_PUBLIC_KEY=<公钥>")
        sys.exit(1)

    print("\n[STEP 2] Fetch VM List")
    print("-" * 60)

    data = client.get_grouped_vms()
    if data:
        client.display_results(data)
    else:
        print("[FAILED] Failed to fetch VM list.")


if __name__ == '__main__':
    main()
