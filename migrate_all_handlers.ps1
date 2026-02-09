# 批量迁移所有handler文件到统一错误处理
# 运行方式: powershell -ExecutionPolicy Bypass -File migrate_all_handlers.ps1

$ErrorActionPreference = "Stop"

$baseDir = "D:\code\ClaudeCode\ruoyi-go-backend"

# 需要处理的文件列表
$files = @(
    # System 模块
    "internal\api\v1\system\role_handler.go",
    "internal\api\v1\system\department_handler.go",
    "internal\api\v1\system\menu_handler.go",
    "internal\api\v1\system\dict_handler.go",
    "internal\api\v1\system\post_handler.go",
    "internal\api\v1\system\notice_handler.go",
    "internal\api\v1\system\profile_handler.go",
    "internal\api\v1\system\notification_config_handler.go",
    "internal\api\v1\system\notice_user_handler.go",
    "internal\api\v1\system\settings_handler.go",
    "internal\api\v1\system\dashboard_handler.go",
    "internal\api\v1\system\fix_menu_handler.go",
    # Operations 模块
    "internal\api\v1\operations\building_handler.go",
    "internal\api\v1\operations\building_handler_typesafe.go",
    "internal\api\v1\operations\floor_handler.go",
    "internal\api\v1\operations\workstation_handler.go",
    "internal\api\v1\operations\server_room_handler.go",
    "internal\api\v1\operations\room_device_handler.go",
    "internal\api\v1\operations\infopoint_handler.go",
    "internal\api\v1\operations\door_handler.go",
    "internal\api\v1\operations\wall_handler.go",
    "internal\api\v1\operations\dedicated_line_handler.go",
    "internal\api\v1\operations\floor_plan_text_handler.go",
    "internal\api\v1\operations\base_handler.go",
    # Network 模块
    "internal\api\v1\network\device_handler.go",
    "internal\api\v1\network\template_handler.go",
    "internal\api\v1\network\credential_handler.go",
    "internal\api\v1\network\command_handler.go",
    "internal\api\v1\network\backup_handler.go",
    "internal\api\v1\network\execution_handler.go",
    "internal\api\v1\network\discovery_handler.go",
    "internal\api\v1\network\port_handler.go",
    # Monitor 模块
    "internal\api\v1\monitor\cache_enhanced_handler.go",
    "internal\api\v1\monitor\server_handler.go",
    "internal\api\v1\monitor\oper_log_handler.go",
    "internal\api\v1\monitor\login_log_handler.go",
    # 其他
    "internal\api\v1\ws_notice_handler.go"
)

$count = 0
$success = 0

foreach ($file in $files) {
    $filePath = Join-Path $baseDir $file

    if (-not (Test-Path $filePath)) {
        Write-Host "跳过不存在的文件: $file" -ForegroundColor Yellow
        continue
    }

    $count++
    Write-Host "[$count] 处理: $file" -ForegroundColor Cyan

    try {
        # 读取文件内容
        $content = Get-Content $filePath -Raw -Encoding UTF8

        # 1. 移除 net/http 导入
        $content = $content -replace '\s+"net/http"\r?\n', ''

        # 2. 添加 apperrors 导入（如果还没有）
        if ($content -notmatch 'apperrors "github\.com/ruoyi-next/ruoyi-go-backend/pkg/errors"') {
            # 在 response 导入后添加
            $content = $content -replace
                '(github\.com/ruoyi-next/ruoyi-go-backend/pkg/response")\r?\n',
                '$1`r`n`tapperrors "github.com/ruoyi-next/ruoyi-go-backend/pkg/errors"'
        }

        # 3. 替换错误处理模式

        # 请求参数错误
        $content = $content -replace
            'response\.Error\(c, http\.StatusBadRequest, "请求参数错误: \+err\.Error\(\)\)',
            'response.Error(c, apperrors.Wrap(err, apperrors.CodeParamError, "请求参数错误"))'

        $content = $content -replace
            'response\.Error\(c, http\.StatusBadRequest, "请求参数错误"\)',
            'response.Error(c, apperrors.ParamError())'

        # XX不能为空
        $content = $content -replace
            'response\.Error\(c, http\.StatusBadRequest, "([^"]+?)不能为空"\)',
            'response.Error(c, apperrors.ParamMissing("$1"))'

        # ID不能为空
        $content = $content -replace
            'response\.Error\(c, http\.StatusBadRequest, "([^"]+?)ID不能为空"\)',
            'response.Error(c, apperrors.ParamMissing("$1ID"))'

        # 内部服务器错误
        $content = $content -replace
            'response\.Error\(c, http\.StatusInternalServerError, err\.Error\(\)\)',
            'response.Error(c, err)'

        $content = $content -replace
            'response\.Error\(c, http\.StatusInternalServerError, "([^"]+?): \+err\.Error\(\)\)',
            'response.Error(c, apperrors.Wrap(err, apperrors.CodeServerError, "$1"))'

        $content = $content -replace
            'response\.Error\(c, http\.StatusInternalServerError, "([^"]+?)"\)',
            'response.Error(c, apperrors.New(apperrors.CodeServerError, "$1"))'

        # 未找到
        $content = $content -replace
            'response\.Error\(c, http\.StatusNotFound, err\.Error\(\)\)',
            'response.Error(c, err)'

        $content = $content -replace
            'response\.Error\(c, http\.StatusNotFound, "([^"]+?)"\)',
            'response.Error(c, apperrors.RecordNotFoundWithMsg("$1"))'

        # 未授权
        $content = $content -replace
            'response\.Error\(c, http\.StatusUnauthorized, "([^"]+?)"\)',
            'response.Error(c, apperrors.Unauthorized())'

        $content = $content -replace
            'response\.Error\(c, http\.StatusUnauthorized\)',
            'response.Error(c, apperrors.Unauthorized())'

        # 禁止访问
        $content = $content -replace
            'response\.Error\(c, http\.StatusForbidden, "([^"]+?)"\)',
            'response.Error(c, apperrors.PermissionDenied())'

        # 写入文件
        $content | Set-Content $filePath -NoNewline -Encoding UTF8

        $success++
        Write-Host "  ✓ 完成" -ForegroundColor Green
    }
    catch {
        Write-Host "  ✗ 失败: $_" -ForegroundColor Red
    }
}

Write-Host "`n========================================" -ForegroundColor Cyan
Write-Host "处理完成!" -ForegroundColor Cyan
Write-Host "总共处理: $count 个文件" -ForegroundColor White
Write-Host "成功: $success 个文件" -ForegroundColor Green
Write-Host "失败: $($count - $success) 个文件" -ForegroundColor Red
Write-Host "========================================" -ForegroundColor Cyan

# 验证编译
Write-Host "`n验证编译..." -ForegroundColor Yellow
Push-Location $baseDir
try {
    $result = & go build ./internal/api/v1/... 2>&1
    if ($LASTEXITCODE -eq 0) {
        Write-Host "✓ 编译成功!" -ForegroundColor Green
    } else {
        Write-Host "✗ 编译失败，请检查错误信息" -ForegroundColor Red
        Write-Host $result
    }
}
finally {
    Pop-Location
}
