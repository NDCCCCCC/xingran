#!/bin/bash

# ========================================
# Build Embedded Executable (Linux)
# ========================================

set -e

# Color definitions
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Project root directory
PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$PROJECT_ROOT"

echo -e "${GREEN}======================================${NC}"
echo -e "${GREEN}  Build Embedded Executable (Linux)${NC}"
echo -e "${GREEN}======================================${NC}"
echo ""

# Step 1: Clean frontend dist
echo -e "[1/5] ${YELLOW}Cleaning frontend build output...${NC}"
if [ -d "xingran-react-frontend/dist" ]; then
    rm -rf xingran-react-frontend/dist
fi
echo "Frontend dist cleaned"
echo ""

# Step 2: Check frontend dependencies
echo -e "[2/5] ${YELLOW}Checking frontend dependencies...${NC}"
cd xingran-react-frontend
if [ ! -d "node_modules" ]; then
    echo "Installing frontend dependencies..."
    npm install
fi
cd ..
echo ""

# Step 3: Build frontend
echo -e "[3/5] ${YELLOW}Building frontend React app...${NC}"
cd xingran-react-frontend
npm run build
if [ $? -ne 0 ]; then
    echo -e "${RED}Frontend build FAILED!${NC}"
    exit 1
fi
cd ..
echo -e "${GREEN}Frontend build completed${NC}"
echo ""

# Step 4: Copy frontend to embed directory
echo -e "[4/6] ${YELLOW}Copying frontend to embed directory...${NC}"
rm -rf internal/server/xingran-react-frontend
mkdir -p internal/server/xingran-react-frontend
cp -r xingran-react-frontend/dist internal/server/xingran-react-frontend/
echo "Frontend files copied to embed directory"
echo ""

# Step 5: Build Go backend
echo -e "[5/6] ${YELLOW}Building Go backend (Linux)...${NC}"
export GOOS=linux
export GOARCH=amd64
export CGO_ENABLED=0
go build -tags=embed -ldflags="-s -w" -o xingran-backend-embedded-linux ./cmd/main.go
if [ $? -ne 0 ]; then
    echo -e "${RED}Go build FAILED!${NC}"
    exit 1
fi
echo -e "${GREEN}Go build completed${NC}"
echo ""

# Step 6: Set execute permission
chmod +x xingran-backend-embedded-linux

# Complete
echo -e "[6/6] ${GREEN}Build completed!${NC}"
echo ""
echo -e "${GREEN}======================================${NC}"
echo -e "${GREEN}  Output File${NC}"
echo -e "${GREEN}======================================${NC}"
echo -e "  ${YELLOW}xingran-backend-embedded-linux${NC}"
echo ""
echo -e "Start server:"
echo -e "  ${YELLOW}./xingran-backend-embedded-linux${NC}"
echo ""
