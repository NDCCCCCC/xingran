@echo off
REM VMP API验证脚本 - Windows批处理启动器

echo ========================================
echo VMP服务器API验证工具
echo ========================================
echo.

REM 检查Python是否安装
python --version >nul 2>&1
if errorlevel 1 (
    echo 错误: 未找到Python，请先安装Python
    pause
    exit /b 1
)

REM 设置脚本目录
set SCRIPT_DIR=%~dp0
cd /d "%SCRIPT_DIR%"

REM 加载.env文件（如果存在）
if exist .env (
    echo 正在加载配置文件 .env ...
    for /f "tokens=*" %%a in ('type .env ^| findstr /v "^#" ^| findstr /v "^$"') do (
        set %%a
    )
    echo 配置已加载
    echo.
)

REM 检查必需的环境变量
if "%VMP_AUTH_COOKIE%"=="" (
    echo 警告: 未设置VMP_AUTH_COOKIE环境变量
    echo.
    echo 请使用以下方式设置Cookie:
    echo   set VMP_AUTH_COOKIE=your_cookie_here
    echo.
    echo 或者复制.env.example为.env并填入实际值
    echo.
    pause
)

REM 运行Python脚本
echo 正在启动API验证...
echo.
python verify_vmp_api.py

echo.
echo ========================================
echo 按任意键退出...
pause >nul
