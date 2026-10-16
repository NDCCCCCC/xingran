#!/bin/bash
# build.sh - Agent 交叉编译脚本

set -e

VERSION=${1:-"1.0.0"}
BUILD_DIR="build"

echo "=========================================="
echo "Building XingRan VDI Agent v$VERSION"
echo "=========================================="

# 清理旧构建
rm -rf "$BUILD_DIR"
mkdir -p "$BUILD_DIR"

echo ""
echo "Building for Linux AMD64..."
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o "$BUILD_DIR/agent-linux-amd64" ./cmd/agent/
echo -e "\033[0;32m✓\033[0m Linux build complete: $BUILD_DIR/agent-linux-amd64"

echo ""
echo "Building for Windows AMD64..."
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -o "$BUILD_DIR/agent-windows-amd64.exe" ./cmd/agent/
echo -e "\033[0;32m✓\033[0m Windows build complete: $BUILD_DIR/agent-windows-amd64.exe"

echo ""
echo "=========================================="
echo "Build summary:"
echo "=========================================="
ls -lh "$BUILD_DIR/"

echo ""
echo "Creating distribution packages..."

cd "$BUILD_DIR"

# Linux 打包
tar czf "agent-linux-amd64-${VERSION}.tar.gz" agent-linux-amd64
echo -e "\033[0;32m✓\033[0m Created: agent-linux-amd64-${VERSION}.tar.gz"

# Windows 打包
if command -v zip &> /dev/null; then
    zip "agent-windows-amd64-${VERSION}.zip" agent-windows-amd64.exe
    echo -e "\033[0;32m✓\033[0m Created: agent-windows-amd64-${VERSION}.zip"
else
    echo -e "\033[1;33m⚠\033[0m zip not found, skipping Windows package"
fi

cd ..

echo ""
echo "=========================================="
echo "Distribution packages:"
echo "=========================================="
ls -lh "$BUILD_DIR/"*.{tar.gz,zip} 2>/dev/null || echo "No packages created"

echo ""
echo -e "\033[0;32mBuild complete!\033[0m"
echo ""
echo "To test locally:"
echo "  ./$BUILD_DIR/agent-linux-amd64 --config=test-config.yaml"
echo ""
echo "To install on target system:"
echo "  Linux:   ./scripts/agent/install-linux.sh <backend_url> <agent_id> <vm_id>"
echo "  Windows: powershell ./scripts/agent/install-windows.ps1 -BackendURL <url> -AgentID <id> -VMID <id>"
echo ""
