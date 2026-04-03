#!/usr/bin/env python3
"""
RPA Worker 任务执行测试脚本

用于测试 RPA Worker 实际执行任务的功能。

使用方法:
    python test_rpa_worker.py --help

环境要求:
    - 后端服务运行中 (http://localhost:9000)
    - RPA Worker 运行中
    - Redis 服务运行中

示例:
    # 创建并执行简单测试任务
    python test_rpa_worker.py create --name "测试任务" --url "https://www.baidu.com"

    # 执行已存在的任务
    python test_rpa_worker.py execute --task-id <task-id>

    # 查看执行状态
    python test_rpa_worker.py status --execution-id <execution-id>

    # 查看所有任务
    python test_rpa_worker.py list
"""

import argparse
import json
import sys
import time
from datetime import datetime
from typing import List, Dict, Any, Optional

import requests


# 配置
BASE_URL = "http://10.62.10.33:9000"  # 后端地址
USERNAME = "admin"
PASSWORD = "admin123"


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
                # 获取 access token
                self.token = data["data"].get("accessToken")
                if self.token:
                    self.session.headers.update({
                        "Authorization": f"Bearer {self.token}"
                    })
                    print(f"✓ 登录成功")
                    return True

        print(f"✗ 登录失败: {response.text}")
        return False

    def _post(self, path: str, data: dict) -> dict:
        """发送 POST 请求"""
        url = f"{self.base_url}{path}"
        response = self.session.post(url, json=data)
        result = response.json()

        if result.get("code") != 0:
            raise Exception(f"API 错误: {result.get('message', 'Unknown error')}")

        return result.get("data", {})

    def _get(self, path: str) -> dict:
        """发送 GET 请求"""
        url = f"{self.base_url}{path}"
        response = self.session.get(url)
        result = response.json()

        if result.get("code") != 0:
            raise Exception(f"API 错误: {result.get('message', 'Unknown error')}")

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
            "maxRetry": 0,
            "priority": 1,
            "status": status
        }

        result = self._post("/api/v1/rpa/tasks/create", data)
        task_id = result.get("id")
        print(f"✓ 任务创建成功: {task_id}")
        return task_id

    def execute_task(
        self,
        task_id: str,
        input_params: Optional[Dict[str, Any]] = None,
        worker_id: Optional[str] = None
    ) -> str:
        """执行任务"""
        data = {
            "taskId": task_id,
            "inputParams": input_params or {},
            "workerId": worker_id or "",
            "priority": 1
        }

        result = self._post("/api/v1/rpa/tasks/execute", data)
        execution_id = result.get("id")
        print(f"✓ 任务执行已启动: {execution_id}")
        return execution_id

    def get_execution_status(self, execution_id: str) -> dict:
        """获取执行状态"""
        return self._get(f"/api/v1/rpa/executions/{execution_id}")

    def list_tasks(self, current: int = 1, page_size: int = 20) -> dict:
        """获取任务列表"""
        return self._post("/api/v1/rpa/tasks/list", {
            "current": current,
            "pageSize": page_size
        })

    def list_executions(self, current: int = 1, page_size: int = 10) -> dict:
        """获取执行记录列表"""
        return self._post("/api/v1/rpa/executions/list", {
            "current": current,
            "pageSize": page_size
        })

    def list_workers(self) -> dict:
        """获取 Worker 列表"""
        return self._post("/api/v1/rpa/workers/list", {
            "current": 1,
            "pageSize": 100
        })


def create_simple_test_script(url: str = "https://www.baidu.com") -> List[Dict[str, Any]]:
    """创建简单测试脚本（导航 + 等待）"""
    return [
        {
            "type": "navigate",
            "selector": "",
            "value": url,
            "attributes": {},
            "timeout": 10000,
            "retry": 0
        },
        {
            "type": "wait",
            "selector": "title",
            "value": "",
            "attributes": {"condition": "visible", "timeout": 5000},
            "timeout": 5000,
            "retry": 0
        },
        {
            "type": "screenshot",
            "selector": "",
            "value": "",
            "attributes": {"name": "page_loaded"},
            "timeout": 5000,
            "retry": 0
        }
    ]


def create_interactive_test_script() -> List[Dict[str, Any]]:
    """创建交互测试脚本（搜索 + 点击）"""
    return [
        {
            "type": "navigate",
            "selector": "",
            "value": "https://www.baidu.com",
            "attributes": {},
            "timeout": 10000,
            "retry": 0
        },
        {
            "type": "fill",
            "selector": "#kw",
            "value": "RPA自动化测试",
            "attributes": {"clear": True},
            "timeout": 5000,
            "retry": 0
        },
        {
            "type": "wait",
            "selector": "#kw",
            "value": "",
            "attributes": {"condition": "visible"},
            "timeout": 3000,
            "retry": 0
        },
        {
            "type": "screenshot",
            "selector": "",
            "value": "",
            "attributes": {"name": "before_click"},
            "timeout": 3000,
            "retry": 0
        },
        {
            "type": "click",
            "selector": "#su",
            "value": "",
            "attributes": {},
            "timeout": 5000,
            "retry": 0
        },
        {
            "type": "wait",
            "selector": ".result",
            "value": "",
            "attributes": {"condition": "visible", "timeout": 5000},
            "timeout": 5000,
            "retry": 0
        },
        {
            "type": "screenshot",
            "selector": "",
            "value": "",
            "attributes": {"name": "results"},
            "timeout": 3000,
            "retry": 0
        }
    ]


def wait_for_execution(
    client: RPAClient,
    execution_id: str,
    timeout: int = 120,
    poll_interval: int = 2
) -> dict:
    """等待执行完成"""
    print(f"⏳ 等待执行完成 (超时: {timeout}秒)...")

    start_time = time.time()
    last_status = None

    while time.time() - start_time < timeout:
        try:
            status = client.get_execution_status(execution_id)
            current_status = status.get("status")

            if current_status != last_status:
                print(f"  状态: {current_status}")
                last_status = current_status

            if current_status in ["completed", "failed", "cancelled"]:
                return status

            # 显示进度
            progress = status.get("progress", {})
            if progress:
                current_step = progress.get("currentStep", 0)
                total_steps = progress.get("totalSteps", 0)
                if total_steps > 0:
                    print(f"  进度: {current_step}/{total_steps}")

        except Exception as e:
            print(f"  ⚠ 获取状态失败: {e}")

        time.sleep(poll_interval)

    print(f"✗ 执行超时")
    return client.get_execution_status(execution_id)


def print_execution_result(execution: dict):
    """打印执行结果"""
    status = execution.get("status", "unknown")
    task_name = execution.get("taskName", "")

    print(f"\n{'='*60}")
    print(f"执行结果: {task_name}")
    print(f"{'='*60}")
    print(f"状态: {status}")
    print(f"开始时间: {execution.get('createdAt', '')}")
    print(f"结束时间: {execution.get('finishedAt', '')}")

    if status == "completed":
        print(f"✓ 执行成功")
        output = execution.get("output", {})
        if output:
            print(f"输出数据: {json.dumps(output, indent=2, ensure_ascii=False)}")
    elif status == "failed":
        print(f"✗ 执行失败")
        error = execution.get("error", {})
        if error:
            print(f"错误信息: {error.get('message', 'Unknown error')}")
            print(f"错误详情: {json.dumps(error, indent=2, ensure_ascii=False)}")

    print(f"{'='*60}\n")


def print_workers_summary(client: RPAClient):
    """打印 Worker 摘要"""
    try:
        result = client.list_workers()
        workers = result.get("list", [])

        print(f"\n{'='*60}")
        print(f"Worker 状态")
        print(f"{'='*60}")
        print(f"总数: {len(workers)}")

        for worker in workers:
            worker_id = worker.get("workerId", worker.get("id", ""))
            name = worker.get("workerName", "")
            status = worker.get("status", "unknown")
            current_tasks = worker.get("currentTasks", 0)
            max_concurrency = worker.get("maxConcurrency", 0)

            print(f"  {worker_id} ({name}): {status} - 任务: {current_tasks}/{max_concurrency}")

        print(f"{'='*60}\n")
    except Exception as e:
        print(f"⚠ 无法获取 Worker 状态: {e}")


def main():
    parser = argparse.ArgumentParser(description="RPA Worker 任务执行测试脚本")
    parser.add_argument("--url", default=BASE_URL, help="后端服务地址")

    subparsers = parser.add_subparsers(dest="command", help="子命令")

    # 创建任务命令
    create_parser = subparsers.add_parser("create", help="创建测试任务")
    create_parser.add_argument("--name", required=True, help="任务名称")
    create_parser.add_argument("--desc", default="", help="任务描述")
    create_parser.add_argument("--target-url", default="https://www.baidu.com", help="目标URL")
    create_parser.add_argument("--interactive", action="store_true", help="使用交互式测试脚本")
    create_parser.add_argument("--no-execute", action="store_true", help="只创建任务，不执行")

    # 执行任务命令
    execute_parser = subparsers.add_parser("execute", help="执行任务")
    execute_parser.add_argument("--task-id", required=True, help="任务ID")
    execute_parser.add_argument("--worker-id", help="指定Worker ID")
    execute_parser.add_argument("--timeout", type=int, default=120, help="等待超时时间（秒）")

    # 状态查询命令
    status_parser = subparsers.add_parser("status", help="查询执行状态")
    status_parser.add_argument("--execution-id", required=True, help="执行ID")
    status_parser.add_argument("--watch", action="store_true", help="持续监控状态")

    # 列表命令
    subparsers.add_parser("list", help="列出所有任务")
    subparsers.add_parser("workers", help="显示Worker状态")

    args = parser.parse_args()

    if not args.command:
        parser.print_help()
        return 1

    # 创建客户端并登录
    client = RPAClient(args.url)
    if not client.login(USERNAME, PASSWORD):
        return 1

    try:
        if args.command == "create":
            # 创建测试任务
            script = create_interactive_test_script() if args.interactive else create_simple_test_script(args.target_url)

            task_id = client.create_task(
                name=args.name,
                description=args.desc,
                script=script,
                target_url=args.target_url
            )

            if not args.no_execute:
                print(f"\n正在执行任务...")
                execution_id = client.execute_task(task_id)
                result = wait_for_execution(client, execution_id)
                print_execution_result(result)
            else:
                print(f"\n任务已创建，ID: {task_id}")
                print(f"使用以下命令执行: python {sys.argv[0]} execute --task-id {task_id}")

        elif args.command == "execute":
            # 执行已有任务
            execution_id = client.execute_task(args.task_id)
            result = wait_for_execution(client, execution_id, timeout=args.timeout)
            print_execution_result(result)

        elif args.command == "status":
            # 查询状态
            status = client.get_execution_status(args.execution_id)

            if args.watch:
                print(f"监控执行状态: {args.execution_id}")
                while True:
                    print(f"\n[{datetime.now().strftime('%H:%M:%S')}] 状态: {status.get('status')}")
                    print_execution_result(status)

                    if status.get("status") in ["completed", "failed", "cancelled"]:
                        break

                    time.sleep(5)
                    status = client.get_execution_status(args.execution_id)
            else:
                print_execution_result(status)

        elif args.command == "list":
            # 列出所有任务
            result = client.list_tasks()
            tasks = result.get("list", [])

            print(f"\n{'='*80}")
            print(f"{'ID':<40} {'名称':<20} {'状态':<10} {'超时':<10}")
            print(f"{'='*80}")

            for task in tasks:
                task_id = task.get("id", "")[:37] + "..."
                name = task.get("taskName", "")[:18]
                status = "启用" if task.get("status") == 0 else "停用"
                timeout = task.get("timeout", 0)

                print(f"{task_id:<40} {name:<20} {status:<10} {timeout:<10}")

            print(f"{'='*80}")
            print(f"总计: {result.get('total', 0)} 条记录\n")

        elif args.command == "workers":
            # 显示 Worker 状态
            print_workers_summary(client)

    except Exception as e:
        print(f"✗ 错误: {e}")
        import traceback
        traceback.print_exc()
        return 1

    return 0


if __name__ == "__main__":
    sys.exit(main())
