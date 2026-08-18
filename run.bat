@echo off
REM run.bat — 启动 xingran-backend 开发服务器(Windows)
REM
REM 解决问题 (log-errors-fix-20260818 P2-15): 旧进程未清理导致
REM   "listen tcp :9000: bind: Only one usage of each socket address"
REM
REM 流程:
REM   1. 查找并杀掉占用 9000 端口的进程(xingran-backend.exe / go.exe 等)
REM   2. 等待 1 秒确保端口释放
REM   3. go run ./cmd/main.go
REM
REM 用法:
REM   run.bat                # 默认启动
REM   SKIP_PORT_CLEANUP=1 run.bat   # 跳过端口清理(快速启动时使用)

setlocal enabledelayedexpansion

REM === 端口清理(可关闭)===
if not "%SKIP_PORT_CLEANUP%"=="1" (
    echo [run] 清理 :9000 端口上的旧进程...

    REM 查找占用 9000 端口的 PID
    for /f "tokens=5" %%a in ('netstat -aon ^| findstr ":9000" ^| findstr "LISTENING"') do (
        set "PID=%%a"
        if not "!PID!"=="" (
            echo [run] 发现占用进程 PID=!PID!,尝试结束...
            taskkill /F /PID !PID! >nul 2>&1
            if !ERRORLEVEL! EQU 0 (
                echo [run] 已结束 PID=!PID!
            ) else (
                echo [run] 结束 PID=!PID! 失败(可能权限不足或已退出)
            )
        )
    )

    REM 兜底: 直接杀 xingran-backend.exe / go.exe(避免遗漏 netstat 未列出的子进程)
    taskkill /F /IM xingran-backend.exe >nul 2>&1
    taskkill /F /IM go.exe /FI "WINDOWTITLE eq go run*" >nul 2>&1

    REM 等待端口释放
    timeout /t 1 /nobreak >nul
)

REM === 启动服务 ===
echo [run] 启动 xingran-backend 开发服务器...
cd /d "%~dp0"
go run ./cmd/main.go

endlocal