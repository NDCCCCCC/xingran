@echo off
REM RPA Worker 构建和部署脚本 (Windows)

setlocal enabledelayedexpansion

REM 颜色设置（Windows 10+）
set "GREEN=[92m"
set "RED=[91m"
set "YELLOW=[93m"
set "NC=[0m"

echo %GREEN%======================================%NC%
echo %GREEN%RPA Worker 构建和部署脚本%NC%
echo %GREEN%======================================%NC%
echo.

REM 获取脚本所在目录
set "SCRIPT_DIR=%~dp0"
set "PROJECT_ROOT=%SCRIPT_DIR%.."

REM 函数：检查Docker
:check_docker
    echo %GREEN%[INFO]%NC% 检查Docker是否安装...
    docker --version >nul 2>&1
    if errorlevel 1 (
        echo %RED%[ERROR]%NC% Docker未安装，请先安装Docker Desktop
        exit /b 1
    )
    echo %GREEN%[INFO]%NC% Docker已安装
    goto :eof

REM 函数：下载依赖
:download_deps
    echo %GREEN%[INFO]%NC% 下载Go依赖...
    cd /d "%PROJECT_ROOT%"
    go mod download
    go mod tidy
    echo %GREEN%[INFO]%NC% 依赖下载完成
    goto :eof

REM 函数：构建镜像
:build_image
    echo %GREEN%[INFO]%NC% 构建Docker镜像...
    cd /d "%PROJECT_ROOT%"
    docker build -f deployments/Dockerfile -t rpa-worker:latest .
    if errorlevel 1 (
        echo %RED%[ERROR]%NC% 镜像构建失败
        exit /b 1
    )
    echo %GREEN%[INFO]%NC% Docker镜像构建完成
    goto :eof

REM 函数：启动服务
:start_services
    echo %GREEN%[INFO]%NC% 启动服务...
    cd /d "%SCRIPT_DIR%"

    if not exist ".env" (
        echo %YELLOW%[WARN]%NC% .env文件不存在，从.env.example复制...
        copy .env.example .env
        echo %YELLOW%[WARN]%NC% 请编辑.env文件配置后重新运行
        goto :eof
    )

    docker-compose up -d
    if errorlevel 1 (
        echo %RED%[ERROR]%NC% 服务启动失败
        exit /b 1
    )
    echo %GREEN%[INFO]%NC% 服务已启动
    goto :eof

REM 函数：停止服务
:stop_services
    echo %GREEN%[INFO]%NC% 停止服务...
    cd /d "%SCRIPT_DIR%"
    docker-compose down
    echo %GREEN%[INFO]%NC% 服务已停止
    goto :eof

REM 函数：查看日志
:logs_services
    cd /d "%SCRIPT_DIR%"
    docker-compose logs -f rpa-worker
    goto :eof

REM 函数：查看状态
:show_status
    cd /d "%SCRIPT_DIR%"
    docker-compose ps
    goto :eof

REM 函数：健康检查
:health_check
    echo %GREEN%[INFO]%NC% 执行健康检查...
    timeout /t 3 /nobreak >nul

    curl -s http://localhost:8080/health
    if errorlevel 1 (
        echo %RED%[ERROR]%NC% 健康检查失败
        exit /b 1
    )
    echo.
    echo %GREEN%[INFO]%NC% 健康检查通过
    goto :eof

REM 函数：完整部署
:deploy
    echo %GREEN%[INFO]%NC% 开始完整部署流程...
    call :check_docker
    call :download_deps
    call :build_image
    call :stop_services
    call :start_services
    timeout /t 5 /nobreak >nul
    call :health_check

    echo.
    echo %GREEN%[INFO]%NC% 部署完成！
    echo.
    echo 健康检查: curl http://localhost:8080/health
    echo 查看日志: %~nx0 logs
    echo 查看状态: %~nx0 status
    goto :eof

REM 函数：显示帮助
:show_help
    echo 用法: %~nx0 [命令]
    echo.
    echo 命令:
    echo   deploy    - 完整部署（默认）
    echo   build     - 仅构建Docker镜像
    echo   start     - 启动服务
    echo   stop      - 停止服务
    echo   restart   - 重启服务
    echo   logs      - 查看日志
    echo   status    - 查看状态
    echo   health    - 健康检查
    echo   help      - 显示此帮助信息
    goto :eof

REM 主程序
set "command=%~1"
if "%command%"=="" set "command=deploy"

if "%command%"=="deploy" (
    call :deploy
) else if "%command%"=="build" (
    call :check_docker
    call :download_deps
    call :build_image
) else if "%command%"=="start" (
    call :start_services
) else if "%command%"=="stop" (
    call :stop_services
) else if "%command%"=="restart" (
    call :stop_services
    call :start_services
) else if "%command%"=="logs" (
    call :logs_services
) else if "%command%"=="status" (
    call :show_status
) else if "%command%"=="health" (
    call :health_check
) else if "%command%"=="help" (
    call :show_help
) else (
    echo %RED%[ERROR]%NC% 未知命令: %command%
    call :show_help
    exit /b 1
)

endlocal
