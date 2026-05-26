@echo off
setlocal enabledelayedexpansion

REM ========================================
REM VDI 数据获取工具
REM 直接从 VDI 系统获取资源组和虚拟机信息
REM ========================================

REM 配置 VDI 服务器信息
set VDI_URL=https://10.62.0.79:6060
set VDI_USERNAME=admin

REM 提示输入密码
set /p VDI_PASSWORD=请输入 VDI 密码:

if "%VDI_PASSWORD%"=="" (
    echo 错误: 密码不能为空
    pause
    exit /b 1
)

echo.
echo ========================================
echo VDI 数据获取工具
echo ========================================
echo 服务器: %VDI_URL%
echo 用户: %VDI_USERNAME%
echo ========================================
echo.

REM 创建临时目录
set TEMP_DIR=%TEMP%\vdi_data_%RANDOM%
mkdir %TEMP_DIR%

REM ============================================================
REM 步骤 1: 认证
REM ============================================================
echo [步骤 1/3] 正在认证...

REM 创建认证请求文件
echo {> "%TEMP_DIR%\auth_request.json"
echo   "auth": {>> "%TEMP_DIR%\auth_request.json"
echo     "name": "%VDI_USERNAME%",>> "%TEMP_DIR%\auth_request.json"
echo     "password": "%VDI_PASSWORD%">> "%TEMP_DIR%\auth_request.json"
echo   }>> "%TEMP_DIR%\auth_request.json"
echo }>> "%TEMP_DIR%\auth_request.json"

REM 发送认证请求
curl -k -s -X POST "%VDI_URL%/v1/auth/tokens" ^
  -H "Content-Type: application/json" ^
  -d @"%TEMP_DIR%\auth_request.json" ^
  > "%TEMP_DIR%\auth_response.json"

REM 检查认证是否成功
findstr /C:"\"error_code\":0" "%TEMP_DIR%\auth_response.json" >nul
if errorlevel 1 (
    echo ❌ 认证失败
    type "%TEMP_DIR%\auth_response.json"
    pause
    exit /b 1
)

echo ✅ 认证成功
echo.

REM ============================================================
REM 步骤 2: 获取资源组
REM ============================================================
echo [步骤 2/3] 正在获取资源组...

REM 从认证响应中提取 token
for /f "tokens=2 delims=:" %%a in ('findstr /C:"auth_token" "%TEMP_DIR%\auth_response.json"') do (
    set VDI_TOKEN=%%a
    set VDI_TOKEN=!VDI_TOKEN:"=!
    set VDI_TOKEN=!VDI_TOKEN:,=!
    goto :token_found
)
:token_found

REM 获取资源组
curl -k -s -X GET "%VDI_URL%/v1/resources_group" ^
  -H "Auth-Token: !VDI_TOKEN!" ^
  > "%TEMP_DIR%\groups_response.json"

REM 检查资源组请求是否成功
findstr /C:"\"error_code\":0" "%TEMP_DIR%\groups_response.json" >nul
if errorlevel 1 (
    echo ❌ 获取资源组失败
    type "%TEMP_DIR%\groups_response.json"
    pause
    exit /b 1
)

echo ✅ 资源组获取成功
echo.

REM 显示资源组信息
echo ========================================
echo 资源组列表
echo ========================================

REM 使用 PowerShell 解析 JSON 并格式化输出
powershell -Command ^
  "$response = Get-Content '%TEMP_DIR%\groups_response.json' -Raw ^| ConvertFrom-Json; " ^
  "$groups = $response.data; " ^
  "Write-Host '资源组总数:' $groups.Count; " ^
  "Write-Host ''; " ^
  "$groups ^| ForEach-Object { " ^
  "  Write-Host 'ID:' $_.id '- 名称:' $_.name '- 启用:' ($_.enable -eq '1'); " ^
  "}"

echo.
echo ========================================
echo.

REM ============================================================
REM 步骤 3: 获取每个资源组的虚拟机
REM ============================================================
echo [步骤 3/3] 正在获取虚拟机信息...
echo.

REM 创建 PowerShell 脚本来处理虚拟机数据
echo $response = Get-Content '%TEMP_DIR%\groups_response.json' -Raw ^| ConvertFrom-Json > "%TEMP_DIR%\parse_vms.ps1"
echo $groups = $response.data >> "%TEMP_DIR%\parse_vms.ps1"
echo $token = '%VDI_TOKEN%' >> "%TEMP_DIR%\parse_vms.ps1"
echo $baseUrl = '%VDI_URL%' >> "%TEMP_DIR%\parse_vms.ps1"
echo. >> "%TEMP_DIR%\parse_vms.ps1"
echo $totalVMs = 0 >> "%TEMP_DIR%\parse_vms.ps1"
echo. >> "%TEMP_DIR%\parse_vms.ps1"
echo foreach ($group in $groups) { >> "%TEMP_DIR%\parse_vms.ps1"
echo     if ($group.enable -ne '1') { continue } >> "%TEMP_DIR%\parse_vms.ps1"
echo     Write-Host "资源组: $($group.name) (ID: $($group.id))" >> "%TEMP_DIR%\parse_vms.ps1"
echo. >> "%TEMP_DIR%\parse_vms.ps1"
echo. >> "%TEMP_DIR%\parse_vms.ps1"
echo     $url = "$baseUrl/v1/resource/servers?rcid=$($group.id)^&page=1^&page_size=100" >> "%TEMP_DIR%\parse_vms.ps1"
echo     try { >> "%TEMP_DIR%\parse_vms.ps1"
echo         $vmResponse = Invoke-RestMethod -Uri $url -Method Get -Headers @{'Auth-Token' = $token} -SkipCertificateCheck >> "%TEMP_DIR%\parse_vms.ps1"
echo         $vmCount = $vmResponse.data.data.Count >> "%TEMP_DIR%\parse_vms.ps1"
echo         $totalVMs += $vmCount >> "%TEMP_DIR%\parse_vms.ps1"
echo         Write-Host "  虚拟机数量: $vmCount" >> "%TEMP_DIR%\parse_vms.ps1"
echo. >> "%TEMP_DIR%\parse_vms.ps1"
echo         if ($vmCount -gt 0) { >> "%TEMP_DIR%\parse_vms.ps1"
echo             Write-Host '  前 10 个虚拟机:' >> "%TEMP_DIR%\parse_vms.ps1"
echo             $vmResponse.data.data ^| Select-Object -First 10 ^| ForEach-Object { >> "%TEMP_DIR%\parse_vms.ps1"
echo                 Write-Host "    - ID: $($_._id) 名称: $($_.vm_name) IP: $($_.ip) 状态: $($_.status)" >> "%TEMP_DIR%\parse_vms.ps1"
echo             } >> "%TEMP_DIR%\parse_vms.ps1"
echo         } >> "%TEMP_DIR%\parse_vms.ps1"
echo     } catch { >> "%TEMP_DIR%\parse_vms.ps1"
echo         Write-Host "  ❌ 获取虚拟机失败: $_" >> "%TEMP_DIR%\parse_vms.ps1"
echo     } >> "%TEMP_DIR%\parse_vms.ps1"
echo     Write-Host '' >> "%TEMP_DIR%\parse_vms.ps1"
echo } >> "%TEMP_DIR%\parse_vms.ps1"
echo. >> "%TEMP_DIR%\parse_vms.ps1"
echo Write-Host '========================================' >> "%TEMP_DIR%\parse_vms.ps1"
echo Write-Host "总计: $totalVMs 个虚拟机" >> "%TEMP_DIR%\parse_vms.ps1"
echo Write-Host '========================================' >> "%TEMP_DIR%\parse_vms.ps1"

REM 执行 PowerShell 脚本
powershell -ExecutionPolicy Bypass -File "%TEMP_DIR%\parse_vms.ps1"

echo.
echo.
echo ========================================
echo 详细数据已保存到临时目录
echo 路径: %TEMP_DIR%
echo ========================================
echo.

REM 保存原始 JSON 数据到当前目录
copy "%TEMP_DIR%\groups_response.json" ".\vdi_groups_data.json" >nul
echo ✅ 资源组数据已保存到: .\vdi_groups_data.json

pause
