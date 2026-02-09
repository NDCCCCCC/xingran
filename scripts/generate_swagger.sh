#!/bin/bash

# generate_swagger.sh
# Swagger文档生成脚本（Linux/Mac）

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 项目根目录
PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$PROJECT_ROOT"

echo -e "${GREEN}======================================${NC}"
echo -e "${GREEN}  Swagger 文档生成脚本${NC}"
echo -e "${GREEN}======================================${NC}"
echo ""

# 检查swag是否安装
if ! command -v swag &> /dev/null; then
    echo -e "${YELLOW}swag 未安装，正在安装...${NC}"
    go install github.com/swaggo/swag/cmd/swag@latest

    if [ $? -ne 0 ]; then
        echo -e "${RED}swag 安装失败${NC}"
        exit 1
    fi

    echo -e "${GREEN}swag 安装成功${NC}"
    echo ""
fi

# 显示swag版本
SWAG_VERSION=$(swag --version 2>&1 | head -n1)
echo -e "${GREEN}当前版本: $SWAG_VERSION${NC}"
echo ""

# 生成文档
echo -e "${YELLOW}正在生成Swagger文档...${NC}"
swag init -g cmd/main.go -o docs --parseInternal --parseDependency --parseInternal

if [ $? -eq 0 ]; then
    echo ""
    echo -e "${GREEN}======================================${NC}"
    echo -e "${GREEN}  文档生成成功！${NC}"
    echo -e "${GREEN}======================================${NC}"
    echo ""
    echo -e "生成的文件:"
    echo -e "  - docs/swagger.json"
    echo -e "  - docs/swagger.yaml"
    echo -e "  - docs/docs.go"
    echo ""
    echo -e "访问地址: ${YELLOW}http://localhost:8080/swagger/index.html${NC}"
    echo ""
else
    echo ""
    echo -e "${RED}======================================${NC}"
    echo -e "${RED}  文档生成失败！${NC}"
    echo -e "${RED}======================================${NC}"
    echo ""
    echo -e "请检查:"
    echo -e "  1. 是否在项目根目录"
    echo -e "  2. Handler的Swagger注解是否正确"
    echo -e "  3. 是否有编译错误"
    echo ""
    exit 1
fi
