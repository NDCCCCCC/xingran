@echo off
REM test-agent.bat - Windows Agent 测试脚本

setlocal enabledelayedexpansion

REM 配置
set AGENT_BIN=.\build\agent-windows-amd64.exe
set CONFIG_FILE=test-agent-config.yaml
set BACKEND_URL=http://localhost:9000
set AGENT_ID=test-agent-001
set VM_ID=test-vm-001

echo ==========================================
echo VM Agent 自动化测试 (Windows)
echo ==========================================
echo Agent: %AGENT_BIN%
echo Config: %CONFIG_FILE%
echo Backend: %BACKEND_URL%
echo.

REM 检查 Agent 是否存在
if not exist "%AGENT_BIN%" (
    echo [ERROR] Agent binary not found: %AGENT_BIN%
    echo 请先运行构建脚本: bash scripts\agent\build.sh
    exit /b 1
)

REM 创建测试配置
echo 创建测试配置...
(
    echo backend_url: "%BACKEND_URL%"
    echo agent_id: "%AGENT_ID%"
    echo vm_id: "%VM_ID%"
    echo listen_addr: ":8443"
    echo heartbeat_interval: 30s
    echo log_level: "debug"
    echo log_path: ".\logs"
    echo platform: "windows"
    echo jwt_secret: "test-secret-key-for-development-only"
) > "%CONFIG_FILE%"

echo [OK] 测试配置创建完成: %CONFIG_FILE%
echo.

REM 创建日志目录
if not exist "logs" mkdir logs

REM 基础测试
echo ==========================================
echo 运行基础测试
echo ==========================================
echo.

REM 测试 1: 配置文件检查
findstr /C:"backend_url" "%CONFIG_FILE%" >nul
if %errorlevel% equ 0 (
    echo 测试: 配置文件加载 ... [OK]
) else (
    echo 测试: 配置文件加载 ... [FAIL]
)

REM 测试 2: Agent 可执行性
if exist "%AGENT_BIN%" (
    echo 测试: Agent 可执行 ... [OK]
) else (
    echo 测试: Agent 可执行 ... [FAIL]
)

REM 测试 3: 端口检查
netstat -an | findstr ":8443" >nul
if %errorlevel% equ 0 (
    echo 测试: 端口 8443 ... [OCCUPIED]
) else (
    echo 测试: 端口 8443 ... [OK]
)

echo.
echo ==========================================
echo 测试完成
echo ==========================================
echo.

REM 询问是否运行功能测试
set /p choice="是否运行功能测试? (需要启动 Agent) [y/N]: "
if /i "%choice%"=="y" (
    echo.
    echo 启动 Agent (在新窗口)...
    start "%AGENT_BIN%" "%AGENT_BIN%" --config="%CONFIG_FILE%"

    echo 等待 Agent 启动...
    timeout /t 3 /nobreak >nul

    echo.
    echo 运行功能测试...
    echo.

    REM 测试健康检查
    echo 测试: 健康检查
    curl -s http://localhost:8443/api/v1/health

    echo.
    echo.
    echo 测试完成。请按 Ctrl+C 停止 Agent 窗口。
)

echo.
echo ==========================================
echo 手动测试命令
echo ==========================================
echo.
echo 运行 Agent:
echo   %AGENT_BIN% --config=%CONFIG_FILE%
echo.
echo 测试 API (PowerShell):
echo   Invoke-WebRequest -Uri "http://localhost:8443/api/v1/health" -Method GET
echo.
echo 测试 API (curl):
echo   curl http://localhost:8443/api/v1/health
echo.

endlocal
