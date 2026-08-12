@echo off
setlocal enabledelayedexpansion

REM VDI 诊断工具 - Windows 版本
REM 用于检查 VDI 系统连接和数据

echo =========================================
echo VDI 系统诊断工具
echo =========================================
echo.

REM 设置默认值
if "%VDI_URL%"=="" set VDI_URL=https://10.62.0.79:6060
if "%VDI_USERNAME%"=="" set VDI_USERNAME=admin

if "%VDI_PASSWORD%"=="" (
    echo ❌ 错误: 请设置 VDI_PASSWORD 环境变量
    echo.
    echo 用法:
    echo   set VDI_PASSWORD=your_password
    echo   scripts\vdi_diagnostic_tool.bat
    echo.
    exit /b 1
)

echo VDI 服务器: %VDI_URL%
echo 用户名: %VDI_USERNAME%
echo.

REM 步骤 1: 测试认证
echo 步骤 1: 测试认证...

REM 创建临时请求文件
set AUTH_REQUEST=%TEMP%\vdi_auth_request.json
echo {"auth":{"name":"%VDI_USERNAME%","password":"%VDI_PASSWORD%"}} > %AUTH_REQUEST%

REM 发送认证请求
curl -k -s -X POST "%VDI_URL%/v1/auth/tokens" -H "Content-Type: application/json" -d @%AUTH_REQUEST% > %TEMP%\vdi_auth_response.json

REM 显示响应（前 500 字符）
echo 认证响应:
powershell -Command "$content = Get-Content '%TEMP%\vdi_auth_response.json' -Raw; Write-Host ($content.Substring(0, [Math]::Min(500, $content.Length)))"
echo.

REM 检查是否认证成功
findstr /C:"\"error_code\":0" %TEMP%\vdi_auth_response.json >nul
if errorlevel 1 (
    echo ❌ 认证失败
    type %TEMP%\vdi_auth_response.json
    exit /b 1
)

echo ✅ 认证成功
echo.

REM 步骤 2: 获取资源组
echo 步骤 2: 获取资源组列表...

curl -k -s -X GET "%VDI_URL%/v1/resources_group" -H "Auth-Token: %VDI_TOKEN%" > %TEMP%\vdi_groups_response.json

echo 资源组响应:
powershell -Command "$content = Get-Content '%TEMP%\vdi_groups_response.json' -Raw; Write-Host ($content.Substring(0, [Math]::Min(500, $content.Length)))"
echo.

REM 步骤 3: 检查本地数据库
echo 步骤 3: 检查本地数据库...
echo.
echo 请在数据库中执行以下查询:
echo.
echo   -- 检查虚拟机数据
echo   SELECT COUNT(*) FROM sys_vdi_virtual_machine WHERE deleted_at IS NULL;
echo.
echo   -- 查看虚拟机详情
echo   SELECT vm_id, name, resource_id, status, power_state, ip_address
echo   FROM sys_vdi_virtual_machine
echo   WHERE deleted_at IS NULL
echo   ORDER BY created_at DESC
echo   LIMIT 10;
echo.

echo =========================================
echo 诊断完成
echo =========================================
echo.
echo 临时文件保存在:
echo   %TEMP%\vdi_auth_response.json
echo   %TEMP%\vdi_groups_response.json
echo.

pause
