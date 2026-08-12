---
slug: build-embedded-linux
description: 创建构建脚本，将前端打包并嵌入到Go后端，生成单个Linux可执行文件
created: 2026-05-21
status: ready
---

# 创建Linux构建脚本（嵌入前端）

## 目标

创建构建脚本，实现：
1. 编译前端React应用（生产构建）
2. 将前端静态文件嵌入到Go后端
3. 生成单个Linux可执行文件

## 技术方案

**前端嵌入策略：**
- 使用Go的`embed`包嵌入前端静态文件
- 前端输出到`xingran-react-frontend/dist`
- Go后端通过embed包提供静态文件服务

**构建流程：**
1. 清理旧的构建产物
2. 构建前端（`npm run build`）
3. 构建Go后端（Linux目标，启用CGO=0）
4. 输出单个可执行文件

---

## 实施步骤

### 步骤1：创建embed文件系统支持

**文件：** `internal/server/embed_fs.go`

```go
package server

import (
	"embed"
	"io/fs"
	"net/http"
	
	"github.com/gin-gonic/gin"
)

//go:embed all:../../xingran-react-frontend/dist
var frontendFS embed.FS

// FrontendFS 返回前端文件系统（去除dist目录前缀）
func FrontendFS() (http.FileSystem, error) {
	sub, err := fs.Sub(frontendFS, "xingran-react-frontend/dist")
	if err != nil {
		return nil, err
	}
	return http.FS(sub), nil
}

// ServeFrontend 提供前端静态文件服务（SPA fallback）
func ServeFrontend(c *gin.Context) {
	frontend, err := FrontendFS()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Frontend not available"})
		return
	}
	
	// 尝试请求的文件
	fileServer := http.FileServer(frontend)
	
	// SPA fallback：如果文件不存在，返回index.html
	if _, err := frontend.Open(c.Request.URL.Path); err != nil {
		c.Request.URL.Path = "/"
	}
	
	fileServer.ServeHTTP(c.Writer, c.Request)
}
```

### 步骤2：更新路由注册

**文件：** `internal/api/router.go`（在`setupRoutes`函数末尾添加）

```go
// 在 setupRoutes 函数末尾添加前端静态文件路由
func setupRoutes(engine *gin.Engine, cfg *config.Config, core *core.Core, allowedOrigins []string) {
	// ... 现有路由配置 ...
	
	// 嵌入式前端静态文件服务
	if cfg.Server.Mode == "release" {
		// 生产模式：使用embed的静态文件
		engine.GET("/assets/*filepath", server.ServeFrontend)
		engine.GET("/index.html", server.ServeFrontend)
		engine.GET("/", server.ServeFrontend)
	}
}
```

### 步骤3：创建Windows构建脚本

**文件：** `scripts/build-embedded.bat`

```batch
@echo off
setlocal enabledelayedexpansion

REM ========================================
REM 构建嵌入式可执行文件（Windows）
REM ========================================

echo ======================================
echo   构建嵌入式可执行文件（Windows）
echo ======================================
echo.

REM 项目根目录
set PROJECT_ROOT=%~dp0..
cd %PROJECT_ROOT%

REM 步骤1：清理前端dist
echo [1/4] 清理前端构建产物...
if exist xingran-react-frontend\dist (
    rmdir /s /q xingran-react-frontend\dist
)
echo 前端dist已清理
echo.

REM 步骤2：构建前端
echo [2/4] 构建前端React应用...
cd xingran-react-frontend
call npm run build
if errorlevel 1 (
    echo 前端构建失败！
    exit /b 1
)
cd ..
echo 前端构建完成
echo.

REM 步骤3：构建Go后端
echo [3/4] 构建Go后端（Windows）...
set GOOS=windows
set GOARCH=amd64
set CGO_ENABLED=0
go build -ldflags="-s -w" -o xingran-backend-embedded.exe ./cmd/main.go
if errorlevel 1 (
    echo Go构建失败！
    exit /b 1
)
echo Go构建完成
echo.

REM 步骤4：完成
echo [4/4] 构建完成！
echo.
echo ======================================
echo   输出文件
echo ======================================
echo   xingran-backend-embedded.exe
echo.
echo 启动服务器:
echo   .\xingran-backend-embedded.exe
echo.

endlocal
```

### 步骤4：创建Linux构建脚本

**文件：** `scripts/build-embedded.sh`

```bash
#!/bin/bash

# ========================================
# 构建嵌入式可执行文件（Linux）
# ========================================

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
echo -e "${GREEN}  构建嵌入式可执行文件（Linux）${NC}"
echo -e "${GREEN}======================================${NC}"
echo ""

# 步骤1：清理前端dist
echo -e "[1/5] ${YELLOW}清理前端构建产物...${NC}"
if [ -d "xingran-react-frontend/dist" ]; then
    rm -rf xingran-react-frontend/dist
fi
echo "前端dist已清理"
echo ""

# 步骤2：检查前端依赖
echo -e "[2/5] ${YELLOW}检查前端依赖...${NC}"
cd xingran-react-frontend
if [ ! -d "node_modules" ]; then
    echo "安装前端依赖..."
    npm install
fi
cd ..
echo ""

# 步骤3：构建前端
echo -e "[3/5] ${YELLOW}构建前端React应用...${NC}"
cd xingran-react-frontend
npm run build
if [ $? -ne 0 ]; then
    echo -e "${RED}前端构建失败！${NC}"
    exit 1
fi
cd ..
echo -e "${GREEN}前端构建完成${NC}"
echo ""

# 步骤4：构建Go后端
echo -e "[4/5] ${YELLOW}构建Go后端（Linux）...${NC}"
export GOOS=linux
export GOARCH=amd64
export CGO_ENABLED=0
go build -ldflags="-s -w" -o xingran-backend-embedded-linux ./cmd/main.go
if [ $? -ne 0 ]; then
    echo -e "${RED}Go构建失败！${NC}"
    exit 1
fi
echo -e "${GREEN}Go构建完成${NC}"
echo ""

# 步骤5：设置执行权限
chmod +x xingran-backend-embedded-linux

# 完成
echo -e "[5/5] ${GREEN}构建完成！${NC}"
echo ""
echo -e "${GREEN}======================================${NC}"
echo -e "${GREEN}  输出文件${NC}"
echo -e "${GREEN}======================================${NC}"
echo -e "  ${YELLOW}xingran-backend-embedded-linux${NC}"
echo ""
echo -e "启动服务器:"
echo -e "  ${YELLOW}./xingran-backend-embedded-linux${NC}"
echo ""
```

### 步骤5：更新配置文件

**文件：** `configs/config.yaml`

确保生产模式下关闭CORS或限制允许的来源：

```yaml
server:
  mode: release  # 生产模式
  port: 9000

# 前端配置（可选，用于版本信息）
frontend:
  embedded: true  # 标识使用嵌入式前端
```

---

## 测试验证

### 1. 本地测试（Windows）

```batch
cd scripts
.\build-embedded.bat
```

验证步骤：
1. 检查`xingran-backend-embedded.exe`生成
2. 运行可执行文件：`.\xingran-backend-embedded.exe`
3. 访问：`http://localhost:9000`
4. 验证前端页面正常加载
5. 验证API调用正常（检查Network面板）

### 2. Linux服务器测试

```bash
cd scripts
chmod +x build-embedded.sh
./build-embedded.sh
```

验证步骤：
1. 检查`xingran-backend-embedded-linux`生成
2. 上传到Linux服务器
3. 运行：`./xingran-backend-embedded-linux`
4. 验证前端页面和API功能

### 3. 文件大小检查

```bash
# 检查可执行文件大小
ls -lh xingran-backend-embedded-linux

# 预期大小：~50-80MB（包含所有前端资源）
```

---

## 故障排查

### 问题1：embed失败：`no matching files found`

**原因：** 前端dist目录不存在或为空

**解决：** 确保先执行`npm run build`，且构建成功

### 问题2：前端页面404

**原因：** embed.FS路径配置错误

**解决：** 检查`internal/server/embed_fs.go`中的`//go:embed`路径

### 问题3：API请求被CORS拦截

**原因：** 嵌入式模式下前端和后端同源，但CORS配置可能过严

**解决：** 在`config.yaml`中配置允许的来源：
```yaml
cors:
  allowed_origins:
    - "*"  # 或指定具体域名
```

### 问题4：前端资源加载失败（CSS/JS）

**原因：** 文件系统子目录路径错误

**解决：** 检查`fs.Sub()`调用中的路径是否正确

---

## 文件清单

### 新建文件
- `internal/server/embed_fs.go` - 嵌入式文件系统
- `scripts/build-embedded.bat` - Windows构建脚本
- `scripts/build-embedded.sh` - Linux构建脚本

### 修改文件
- `internal/api/router.go` - 添加前端静态文件路由
- `configs/config.yaml` - 可选：添加前端配置

### 构建产物
- `xingran-backend-embedded.exe` - Windows可执行文件
- `xingran-backend-embedded-linux` - Linux可执行文件

---

## 后续优化

1. **版本信息：** 在构建时注入版本号（使用`ldflags`）
2. **压缩：** 使用upx进一步压缩可执行文件
3. **安装脚本：** 创建systemd服务安装脚本
4. **健康检查：** 添加`/health`端点用于容器编排
