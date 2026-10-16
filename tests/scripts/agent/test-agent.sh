#!/bin/bash
# test-agent.sh - Agent 自动化测试脚本

set -e

# 颜色定义
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

# 配置
AGENT_BIN="${1:-./build/agent-linux-amd64}"
CONFIG_FILE="${2:-test-agent-config.yaml}"
BACKEND_URL="${3:-http://localhost:9000}"
AGENT_ID="${4:-test-agent-001}"
VM_ID="${5:-test-vm-001}"

echo "=========================================="
echo "VM Agent 自动化测试"
echo "=========================================="
echo "Agent: $AGENT_BIN"
echo "Config: $CONFIG_FILE"
echo "Backend: $BACKEND_URL"
echo ""

# 检查 Agent 是否存在
if [ ! -f "$AGENT_BIN" ]; then
    echo -e "${RED}ERROR: Agent binary not found: $AGENT_BIN${NC}"
    echo "请先运行构建脚本: bash scripts/agent/build.sh"
    exit 1
fi

# 创建测试配置
echo "创建测试配置..."
cat > "$CONFIG_FILE" <<EOF
backend_url: "$BACKEND_URL"
agent_id: "$AGENT_ID"
vm_id: "$VM_ID"
listen_addr: ":18443"
heartbeat_interval: 10s
log_level: "debug"
log_path: "./logs"
platform: "linux"
jwt_secret: "test-secret-key-for-development-only"
EOF

echo -e "${GREEN}✓${NC} 测试配置创建完成: $CONFIG_FILE"
echo ""

# 创建日志目录
mkdir -p logs

# 测试函数
test_case() {
    local name=$1
    local expected=$2

    echo -n "测试: $name ... "

    if [[ "$expected" == "PASS" ]]; then
        echo -e "${GREEN}✓ PASS${NC}"
        return 0
    else
        echo -e "${RED}✗ FAIL${NC}"
        return 1
    fi
}

# 测试套件
run_tests() {
    echo "=========================================="
    echo "运行测试套件"
    echo "=========================================="
    echo ""

    local passed=0
    local failed=0

    # 测试 1: 配置文件加载
    if grep -q "backend_url" "$CONFIG_FILE"; then
        test_case "配置文件加载" "PASS"
        ((passed++))
    else
        test_case "配置文件加载" "FAIL"
        ((failed++))
    fi

    # 测试 2: Agent 可执行性
    if [ -x "$AGENT_BIN" ]; then
        test_case "Agent 可执行" "PASS"
        ((passed++))
    else
        test_case "Agent 可执行" "FAIL"
        ((failed++))
    fi

    # 测试 3: 端口检查
    if netstat -tuln 2>/dev/null | grep -q ":18443"; then
        echo -n "测试: 端口 18443 ... "
        echo -e "${YELLOW}已占用${NC}"
    else
        test_case "端口 18443 可用" "PASS"
        ((passed++))
    fi

    echo ""
    echo "=========================================="
    echo "测试结果"
    echo "=========================================="
    echo -e "通过: ${GREEN}${passed}${NC}"
    echo -e "失败: ${RED}${failed}${NC}"
    echo ""

    if [ $failed -eq 0 ]; then
        return 0
    else
        return 1
    fi
}

# 功能测试
functional_tests() {
    echo "=========================================="
    echo "功能测试 (需要 Agent 运行)"
    echo "=========================================="
    echo ""

    # 启动 Agent
    echo "启动 Agent..."
    "$AGENT_BIN" --config="$CONFIG_FILE" &
    AGENT_PID=$!

    # 等待 Agent 启动
    sleep 3

    # 检查 Agent 是否运行
    if ps -p $AGENT_PID > /dev/null; then
        echo -e "${GREEN}✓ Agent 启动成功 (PID: $AGENT_PID)${NC}"
    else
        echo -e "${RED}✗ Agent 启动失败${NC}"
        return 1
    fi

    echo ""
    echo "运行功能测试..."

    # 测试健康检查
    echo -n "健康检查: "
    if curl -s http://localhost:18443/api/v1/health | grep -q "healthy"; then
        echo -e "${GREEN}✓ PASS${NC}"
    else
        echo -e "${RED}✗ FAIL${NC}"
    fi

    # 测试注册
    echo -n "Agent 注册: "
    if curl -s -X POST http://localhost:18443/api/v1/register \
        -H "Content-Type: application/json" \
        -d "{\"agent_id\":\"$AGENT_ID\",\"vm_id\":\"$VM_ID\"}" \
        | grep -q "success"; then
        echo -e "${GREEN}✓ PASS${NC}"
    else
        echo -e "${YELLOW}⚠ 需要后端服务器${NC}"
    fi

    # 清理
    echo ""
    echo "停止 Agent..."
    kill $AGENT_PID 2>/dev/null || true
    sleep 1

    echo -e "${GREEN}✓ 功能测试完成${NC}"
}

# 主流程
main() {
    # 运行基础测试
    if run_tests; then
        echo -e "${GREEN}✓ 基础测试通过${NC}"
    else
        echo -e "${RED}✗ 基础测试失败${NC}"
        exit 1
    fi

    echo ""
    read -p "是否运行功能测试? (需要启动 Agent) [y/N]: " choice
    if [[ "$choice" =~ ^[Yy]$ ]]; then
        functional_tests
    else
        echo "跳过功能测试"
    fi

    echo ""
    echo "=========================================="
    echo "测试完成"
    echo "=========================================="
    echo ""
    echo "手动运行 Agent:"
    echo "  $AGENT_BIN --config=$CONFIG_FILE"
    echo ""
    echo "测试 API:"
    echo "  curl http://localhost:18443/api/v1/health"
    echo ""
}

# 运行主流程
main
