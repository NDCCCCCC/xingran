# install-windows.ps1 - Windows Agent 安装脚本
param(
    [string]$BackendURL = "https://ruoyi-backend.example.com",
    [string]$AgentID = "",
    [string]$VMID = ""
)

Write-Host "Installing RuoYi VDI Agent..." -ForegroundColor Green

# 检查管理员权限
$isAdmin = ([Security.Principal.WindowsPrincipal] [Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
if (-not $isAdmin) {
    Write-Host "ERROR: This script must be run as Administrator" -ForegroundColor Red
    exit 1
}

# 验证参数
if ([string]::IsNullOrEmpty($AgentID) -or [string]::IsNullOrEmpty($VMID)) {
    Write-Host "ERROR: AgentID and VMID are required" -ForegroundColor Red
    Write-Host "Usage: .\install-windows.ps1 -BackendURL <url> -AgentID <id> -VMID <id>" -ForegroundColor Yellow
    exit 1
}

# 下载 Agent
$agentUrl = "$BackendURL/api/v1/agent/download/windows"
$agentPath = "C:\Program Files\RuoyiAgent"
$agentExe = "$agentPath\agent.exe"

New-Item -Path $agentPath -ItemType Directory -Force | Out-Null

Write-Host "Downloading agent from: $agentUrl"
try {
    Invoke-WebRequest -Uri $agentUrl -OutFile $agentExe -UseBasicParsing
} catch {
    Write-Host "ERROR: Failed to download agent: $_" -ForegroundColor Red
    exit 1
}

# 创建配置文件
$configPath = "$agentPath\config.yaml"
@"
backend_url: $BackendURL
agent_id: $AgentID
vm_id: $VMID
listen_addr: ":8443"
heartbeat_interval: 30s
log_level: info
log_path: C:\Program Files\RuoyiAgent\logs
platform: windows
"@ | Out-File -FilePath $configPath -Encoding UTF8

Write-Host "Configuration created at: $configPath"

# 创建日志目录
$logPath = "C:\Program Files\RuoyiAgent\logs"
New-Item -Path $logPath -ItemType Directory -Force | Out-Null

# 创建受限管理员账号
$password = ConvertTo-SecureString "RuoyiAgent123!" -AsPlainText -Force
try {
    New-LocalUser -Name "RuoyiAgentUser" -Password $password -Description "RuoYi VDI Agent User" -ErrorAction Stop | Out-Null
    Write-Host "Created user: RuoyiAgentUser"
} catch {
    if ($_.Exception.Message -like "*already exists*") {
        Write-Host "User RuoyiAgentUser already exists" -ForegroundColor Yellow
    } else {
        Write-Host "WARNING: Failed to create user: $_" -ForegroundColor Yellow
    }
}

# 授予用户管理权限（简化版本，生产环境应使用 JEA）
try {
    Add-LocalGroupMember -Group "Administrators" -Member "RuoyiAgentUser" -ErrorAction Stop | Out-Null
    Write-Host "Added RuoyiAgentUser to Administrators group"
} catch {
    Write-Host "WARNING: Failed to add to Administrators group: $_" -ForegroundColor Yellow
}

# 注册 Windows 服务
$serviceName = "RuoyiVMAgent"
$serviceDisplayName = "RuoYi VM Agent"

# 检查服务是否已存在
$existingService = Get-Service -Name $serviceName -ErrorAction SilentlyContinue
if ($existingService) {
    Write-Host "Stopping existing service..." -ForegroundColor Yellow
    Stop-Service -Name $serviceName -Force
    Start-Sleep -Seconds 2

    Write-Host "Removing existing service..."
    sc.exe delete $serviceName | Out-Null
    Start-Sleep -Seconds 2
}

Write-Host "Creating Windows service: $serviceName"
try {
    New-Service -Name $serviceName -BinaryPathName "$agentExe --config=$configPath" -DisplayName $serviceDisplayName -StartupType Automatic -ErrorAction Stop | Out-Null
    Write-Host "Service created successfully"
} catch {
    Write-Host "ERROR: Failed to create service: $_" -ForegroundColor Red
    exit 1
}

# 启动服务
Write-Host "Starting service..."
try {
    Start-Service -Name $serviceName -ErrorAction Stop
    Write-Host "Service started successfully" -ForegroundColor Green
} catch {
    Write-Host "ERROR: Failed to start service: $_" -ForegroundColor Red
    exit 1
}

Write-Host ""
Write-Host "========================================" -ForegroundColor Green
Write-Host "RuoYi VDI Agent installed successfully!" -ForegroundColor Green
Write-Host "========================================" -ForegroundColor Green
Write-Host "Service name: $serviceName"
Write-Host "Config file: $configPath"
Write-Host "Log directory: $logPath"
Write-Host ""
Write-Host "To check service status:"
Write-Host "  Get-Service $serviceName"
Write-Host ""
Write-Host "To view logs:"
Write-Host "  Get-Content $logPath\*.log"
Write-Host ""
