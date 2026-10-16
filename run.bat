@echo off
REM run.bat - Start xingran-backend dev server (Windows)
REM
REM Fixes log-errors-fix-20260818 P2-15: old process not cleaned up causes
REM   "listen tcp :9000: bind: Only one usage of each socket address"
REM
REM Flow:
REM   1. Find and kill processes holding :9000 (xingran-backend.exe / main.exe / go.exe)
REM   2. Wait 1s for port release
REM   3. go run ./cmd/main.go
REM
REM Usage:
REM   run.bat                       # default start
REM   SKIP_PORT_CLEANUP=1 run.bat   # skip port cleanup (fast restart)

setlocal enabledelayedexpansion

REM === Port cleanup (skippable) ===
if not "%SKIP_PORT_CLEANUP%"=="1" (
    echo [run] cleaning :9000 port...

    REM Find PIDs holding :9000
    for /f "tokens=5" %%a in ('netstat -aon ^| findstr ":9000" ^| findstr "LISTENING"') do (
        set "PID=%%a"
        if not "!PID!"=="" (
            echo [run] killing PID=!PID!
            taskkill /F /PID !PID! >nul 2>&1
        )
    )

    REM Defense-in-depth: kill by image name (catches processes netstat may miss).
    REM main.exe is what go run produces on Windows when the package name is "main"
    REM (cmd/main.go is package main, so go run builds it to a temp main.exe).
    taskkill /F /IM xingran-backend.exe >nul 2>&1
    taskkill /F /IM main.exe >nul 2>&1
    taskkill /F /IM go.exe >nul 2>&1

    REM Wait for port release (use ping for sleep — Git Bash's `timeout` is GNU coreutils,
    REM which doesn't accept `/t` and would otherwise break this script when invoked
    REM from Git Bash with `./run.bat`).
    ping -n 2 127.0.0.1 >nul
)

REM === Start server ===
echo [run] starting xingran-backend dev server...
cd /d "%~dp0"
go run ./cmd/main.go

endlocal