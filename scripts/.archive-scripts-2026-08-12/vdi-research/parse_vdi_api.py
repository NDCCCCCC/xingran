#!/usr/bin/env python3
# -*- coding: utf-8 -*-
import re
import sys

def extract_auth_api(file_path):
    """提取 VDI API 认证相关接口"""

    with open(file_path, 'r', encoding='utf-8', errors='ignore') as f:
        content = f.read()

    # 按章节标题分割
    sections = re.split(r'(4\.1\.1|4\.1\.2|接口地址|请求方式|请求参数|返回参数|示例)', content)

    results = []
    capture = False
    current_section = ""

    for i, section in enumerate(sections):
        # 查找认证相关章节
        if '操作权限认证' in section or '权限认证' in section:
            capture = True
            current_section = "认证"

        # 查找接口地址
        if '接口地址' in section or 'URL' in section or '/API/' in section:
            capture = True

        if capture:
            # 清理文本
            clean_text = re.sub(r'\s+', ' ', section.strip())
            if len(clean_text) > 10:
                results.append(clean_text)

        if len(results) > 50:  # 限制输出长度
            break

    return results

if __name__ == '__main__':
    file_path = 'docs/sangfor_vdi_utf8.txt'

    print("=" * 80)
    print("VDI API 认证接口信息")
    print("=" * 80)

    results = extract_auth_api(file_path)

    for r in results[:30]:  # 只显示前30个结果
        print(r)
        print("-" * 80)
