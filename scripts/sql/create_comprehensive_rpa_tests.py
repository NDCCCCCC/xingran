#!/usr/bin/env python3
"""
RPA 综合测试任务生成脚本（修复版 v2）

使用正确的字段格式：
- navigate: URL 放在 value 字段
- fill/select: 值放在 value 字段
- 其他参数: 放在 attributes 字段
"""

import argparse
import json
import sys
import os
from typing import List, Dict, Any
import requests


# 配置
BASE_URL = "http://10.62.10.33:9000"
USERNAME = "admin"
PASSWORD = "admin123"


# 设置 Windows 控制台编码
if sys.platform == 'win32':
    import io
    sys.stdout = io.TextIOWrapper(sys.stdout.buffer, encoding='utf-8', errors='replace')
    sys.stderr = io.TextIOWrapper(sys.stderr.buffer, encoding='utf-8', errors='replace')


class RPAClient:
    """RPA API 客户端"""

    def __init__(self, base_url: str):
        self.base_url = base_url
        self.token = None
        self.session = requests.Session()

    def login(self, username: str, password: str) -> bool:
        """登录获取 token"""
        url = f"{self.base_url}/api/v1/system/auth/login"
        response = self.session.post(url, json={
            "username": username,
            "password": password,
            "captchaId": "",
            "captchaCode": ""
        })

        if response.status_code == 200:
            data = response.json()
            if data.get("code") == 0 and "data" in data:
                self.token = data["data"].get("accessToken")
                if self.token:
                    self.session.headers.update({
                        "Authorization": f"Bearer {self.token}"
                    })
                    print(f"[OK] Login successful")
                    return True

        print(f"[FAIL] Login failed: {response.text}")
        return False

    def _post(self, path: str, data: dict) -> dict:
        """发送 POST 请求"""
        url = f"{self.base_url}{path}"
        response = self.session.post(url, json=data)
        result = response.json()

        if result.get("code") != 0:
            raise Exception(f"API Error: {result.get('message', 'Unknown error')}")

        return result.get("data", {})

    def create_task(
        self,
        name: str,
        script: List[Dict[str, Any]],
        target_url: str = "",
        description: str = "",
        timeout: int = 300,
        status: int = 0
    ) -> str:
        """创建任务"""
        data = {
            "name": name,
            "description": description,
            "script": script,
            "targetUrl": target_url,
            "timeout": timeout,
            "maxRetry": 2,
            "priority": 1,
            "status": status
        }

        result = self._post("/api/v1/rpa/tasks", data)
        task_id = result.get("id")
        print(f"[OK] Task created: {task_id}")
        return task_id


# ==================== 测试场景定义 ====================

def create_simple_test() -> tuple[str, str, List[Dict]]:
    """场景1: 简单导航测试"""
    name = "简单导航测试"
    description = "最简单的导航和截图测试"
    target_url = "https://www.baidu.com"

    script = [
        {
            "type": "navigate",
            "selector": "",
            "value": "https://www.baidu.com",
            "attributes": {},
            "timeout": 15000,
            "retry": 2
        },
        {
            "type": "wait",
            "selector": "#kw",
            "value": "",
            "attributes": {"condition": "visible"},
            "timeout": 10000,
            "retry": 0
        },
        {
            "type": "screenshot",
            "selector": "",
            "value": "",
            "attributes": {"name": "测试完成"},
            "timeout": 3000,
            "retry": 0
        }
    ]

    return name, description, script


def create_search_extract_test() -> tuple[str, str, List[Dict]]:
    """场景2: 百度搜索结果提取测试"""
    name = "百度搜索结果提取测试"
    description = "在百度搜索关键词，提取搜索结果标题和链接"
    target_url = "https://www.baidu.com"

    script = [
        {
            "type": "navigate",
            "selector": "",
            "value": "https://www.baidu.com",
            "attributes": {},
            "timeout": 15000,
            "retry": 2
        },
        {
            "type": "fill",
            "selector": "#kw",
            "value": "RPA自动化测试",
            "attributes": {"clear": True},
            "timeout": 5000,
            "retry": 1
        },
        {
            "type": "click",
            "selector": "#su",
            "value": "",
            "attributes": {},
            "timeout": 5000,
            "retry": 1
        },
        {
            "type": "wait",
            "selector": ".result",
            "value": "",
            "attributes": {"condition": "visible", "timeout": 8000},
            "timeout": 10000,
            "retry": 0
        },
        {
            "type": "screenshot",
            "selector": "",
            "value": "",
            "attributes": {"name": "搜索结果页面"},
            "timeout": 3000,
            "retry": 0
        },
        {
            "type": "extract",
            "selector": ".result",
            "value": "",
            "attributes": {
                "extractType": "multiple",
                "fields": [
                    {"name": "title", "selector": "h3", "attribute": "text"},
                    {"name": "link", "selector": "a", "attribute": "href"}
                ],
                "maxItems": 5,
                "outputVariable": "searchResults"
            },
            "timeout": 5000,
            "retry": 0
        },
        {
            "type": "screenshot",
            "selector": "",
            "value": "",
            "attributes": {"name": "任务完成"},
            "timeout": 3000,
            "retry": 0
        }
    ]

    return name, description, script


def create_loop_batch_test() -> tuple[str, str, List[Dict]]:
    """场景3: 循环批量搜索测试"""
    name = "循环批量搜索测试"
    description = "对多个关键词进行批量搜索，测试循环和变量替换功能"
    target_url = "https://www.baidu.com"

    script = [
        {
            "type": "navigate",
            "selector": "",
            "value": "https://www.baidu.com",
            "attributes": {},
            "timeout": 15000,
            "retry": 2
        },
        {
            "type": "screenshot",
            "selector": "",
            "value": "",
            "attributes": {"name": "初始页面"},
            "timeout": 3000,
            "retry": 0
        },
        {
            "type": "loop",
            "selector": "",
            "value": "",
            "attributes": {
                "dataSource": [
                    {"keyword": "RPA自动化"},
                    {"keyword": "Python爬虫"},
                    {"keyword": "Golang开发"}
                ],
                "itemName": "item",
                "actions": [
                    {
                        "type": "fill",
                        "selector": "#kw",
                        "value": "{{item.keyword}}",
                        "attributes": {"clear": True}
                    },
                    {
                        "type": "screenshot",
                        "selector": "",
                        "value": "",
                        "attributes": {"name": "搜索_{{item.keyword}}"}
                    },
                    {
                        "type": "click",
                        "selector": "#su",
                        "value": "",
                        "attributes": {}
                    },
                    {
                        "type": "wait",
                        "selector": ".result",
                        "value": "",
                        "attributes": {"condition": "visible", "timeout": 5000}
                    },
                    {
                        "type": "screenshot",
                        "selector": "",
                        "value": "",
                        "attributes": {"name": "结果_{{item.keyword}}"}
                    }
                ]
            },
            "timeout": 60000,
            "retry": 0
        },
        {
            "type": "screenshot",
            "selector": "",
            "value": "",
            "attributes": {"name": "批量搜索完成"},
            "timeout": 3000,
            "retry": 0
        }
    ]

    return name, description, script


def create_conditional_test() -> tuple[str, str, List[Dict]]:
    """场景4: 条件判断测试"""
    name = "条件判断与分支测试"
    description = "测试条件判断功能，根据页面元素是否存在执行不同分支"
    target_url = "https://www.baidu.com"

    script = [
        {
            "type": "navigate",
            "selector": "",
            "value": "https://www.baidu.com",
            "attributes": {},
            "timeout": 15000,
            "retry": 2
        },
        {
            "type": "screenshot",
            "selector": "",
            "value": "",
            "attributes": {"name": "页面加载完成"},
            "timeout": 3000,
            "retry": 0
        },
        {
            "type": "condition",
            "selector": "#kw",
            "value": "",
            "attributes": {
                "condition": "exists",
                "trueActions": [
                    {
                        "type": "fill",
                        "selector": "#kw",
                        "value": "搜索框存在",
                        "attributes": {}
                    },
                    {
                        "type": "screenshot",
                        "selector": "",
                        "value": "",
                        "attributes": {"name": "搜索框存在分支"}
                    }
                ],
                "falseActions": [
                    {
                        "type": "screenshot",
                        "selector": "",
                        "value": "",
                        "attributes": {"name": "搜索框不存在分支"}
                    }
                ]
            },
            "timeout": 10000,
            "retry": 0
        },
        {
            "type": "screenshot",
            "selector": "",
            "value": "",
            "attributes": {"name": "条件判断测试完成"},
            "timeout": 3000,
            "retry": 0
        }
    ]

    return name, description, script


def create_navigation_test() -> tuple[str, str, List[Dict]]:
    """场景5: 页面导航与滚动测试"""
    name = "页面导航与滚动测试"
    description = "测试页面滚动、导航跳转、悬停等浏览器操作"
    target_url = "https://www.baidu.com"

    script = [
        {
            "type": "navigate",
            "selector": "",
            "value": "https://www.baidu.com",
            "attributes": {},
            "timeout": 15000,
            "retry": 2
        },
        {
            "type": "screenshot",
            "selector": "",
            "value": "",
            "attributes": {"name": "首页"},
            "timeout": 3000,
            "retry": 0
        },
        {
            "type": "fill",
            "selector": "#kw",
            "value": "Python编程",
            "attributes": {},
            "timeout": 5000,
            "retry": 1
        },
        {
            "type": "click",
            "selector": "#su",
            "value": "",
            "attributes": {},
            "timeout": 5000,
            "retry": 1
        },
        {
            "type": "wait",
            "selector": ".result",
            "value": "",
            "attributes": {"condition": "visible"},
            "timeout": 10000,
            "retry": 0
        },
        {
            "type": "screenshot",
            "selector": "",
            "value": "",
            "attributes": {"name": "搜索结果"},
            "timeout": 3000,
            "retry": 0
        },
        {
            "type": "scroll",
            "selector": "",
            "value": "",
            "attributes": {
                "direction": "down",
                "distance": 500
            },
            "timeout": 5000,
            "retry": 0
        },
        {
            "type": "screenshot",
            "selector": "",
            "value": "",
            "attributes": {"name": "滚动后页面"},
            "timeout": 3000,
            "retry": 0
        }
    ]

    return name, description, script


def create_comprehensive_test() -> tuple[str, str, List[Dict]]:
    """场景6: 综合端到端测试"""
    name = "综合端到端业务流程测试"
    description = "完整的业务流程测试，包含多种RPA操作"
    target_url = "https://www.baidu.com"

    script = [
        {
            "type": "navigate",
            "selector": "",
            "value": "https://www.baidu.com",
            "attributes": {},
            "timeout": 15000,
            "retry": 2
        },
        {
            "type": "screenshot",
            "selector": "",
            "value": "",
            "attributes": {"name": "01_页面初始化"},
            "timeout": 3000,
            "retry": 0
        },
        {
            "type": "condition",
            "selector": "#kw",
            "value": "",
            "attributes": {
                "condition": "exists",
                "trueActions": [],
                "falseActions": []
            },
            "timeout": 5000,
            "retry": 0
        },
        {
            "type": "loop",
            "selector": "",
            "value": "",
            "attributes": {
                "dataSource": [
                    {"keyword": "RPA", "step": 1},
                    {"keyword": "自动化", "step": 2}
                ],
                "itemName": "searchItem",
                "actions": [
                    {
                        "type": "fill",
                        "selector": "#kw",
                        "value": "{{searchItem.keyword}}",
                        "attributes": {"clear": True}
                    },
                    {
                        "type": "screenshot",
                        "selector": "",
                        "value": "",
                        "attributes": {"name": "02_搜索{{searchItem.step}}"}
                    },
                    {
                        "type": "click",
                        "selector": "#su",
                        "value": "",
                        "attributes": {}
                    },
                    {
                        "type": "wait",
                        "selector": ".result",
                        "value": "",
                        "attributes": {"condition": "visible", "timeout": 6000}
                    },
                    {
                        "type": "screenshot",
                        "selector": "",
                        "value": "",
                        "attributes": {"name": "03_结果{{searchItem.step}}"}
                    }
                ]
            },
            "timeout": 60000,
            "retry": 0
        },
        {
            "type": "screenshot",
            "selector": "",
            "value": "",
            "attributes": {"name": "04_测试完成"},
            "timeout": 3000,
            "retry": 0
        }
    ]

    return name, description, script


# 测试场景集合
TEST_SCENARIOS = {
    "1": create_simple_test,
    "2": create_search_extract_test,
    "3": create_loop_batch_test,
    "4": create_conditional_test,
    "5": create_navigation_test,
    "6": create_comprehensive_test,
}


def main():
    parser = argparse.ArgumentParser(description="RPA 综合测试任务生成脚本")
    parser.add_argument("--url", default=BASE_URL, help="后端服务地址")
    parser.add_argument("--scenario", required=True, help="测试场景编号 (1-6) 或 all")

    args = parser.parse_args()

    # 创建客户端并登录
    client = RPAClient(args.url)
    if not client.login(USERNAME, PASSWORD):
        return 1

    created_tasks = []

    try:
        if args.scenario == "all":
            print("\n开始创建所有测试任务...")
            for key in ["1", "2", "3", "4", "5", "6"]:
                name, description, script = TEST_SCENARIOS[key]()
                task_id = client.create_task(
                    name=name,
                    description=description,
                    script=script,
                    target_url=""
                )
                created_tasks.append((key, name, task_id))
        else:
            for scenario in args.scenario.split(','):
                if scenario in TEST_SCENARIOS:
                    name, description, script = TEST_SCENARIOS[scenario]()
                    task_id = client.create_task(
                        name=name,
                        description=description,
                        script=script,
                        target_url=""
                    )
                    created_tasks.append((scenario, name, task_id))

        # 打印结果
        print(f"\n{'='*80}")
        print(f"{'编号':<6} {'任务名称':<30} {'任务ID':<42}")
        print(f"{'='*80}")
        for scenario, name, task_id in created_tasks:
            print(f"{scenario:<6} {name:<30} {task_id}")
        print(f"{'='*80}")
        print(f"\n[OK] Created {len(created_tasks)} test tasks")

    except Exception as e:
        print(f"[ERROR] {e}")
        import traceback
        traceback.print_exc()
        return 1

    return 0


if __name__ == "__main__":
    sys.exit(main())
