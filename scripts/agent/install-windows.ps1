# install-windows.ps1 - Windows Agent 安装脚本
param(
    [string]$BackendURL = "https://xingran-backend.example.com",
    [string]$AgentID = "",
    [string]$VMID = ""
)

Write-Host "Installing XingRan VDI Agent..." -ForegroundColor Green

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
$agentPath = "C:\Program Files\XingRanAgent"
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
log_path: C:\Program Files\XingRanAgent\logs
platform: windows
"@ | Out-File -FilePath $configPath -Encoding UTF8

Write-Host "Configuration created at: $configPath"

# 创建日志目录
$logPath = "C:\Program Files\XingRanAgent\logs"
New-Item -Path $logPath -ItemType Directory -Force | Out-Null

# Create JEA Session Configuration
function Create-JEAConfiguration {
    param(
        [string]$ConfigPath = "C:\Program Files\WindowsPowerShell\Configuration\sessionConfigs\XingRanAgent.pssc"
    )

    Write-Host "Creating JEA session configuration..."

    # Define role capabilities
    $roleCapabilityPath = "C:\Program Files\WindowsPowerShell\Configuration\RoleCapabilities\XingRanAgentRole.ps1"

    # Create role capabilities file
    @"
@{
    # ID used to uniquely identify this role capability
    GUID = 'a8c7f7e3-6d4a-4f5b-9e8a-2d3c4b5a6e7f'

    # Author of this role capability
    Author = 'XingRan VDI System'

    # Company or vendor of this role capability
    CompanyName = 'XingRan'

    # Description of the functionality provided by these role capabilities
    Description = 'Restricted role for XingRan VDI Agent - user account management only'

    # Cmdlets to make visible when applied to a session
    VisibleCmdlets = @(
        'Microsoft.PowerShell.LocalAccounts\New-LocalUser',
        'Microsoft.PowerShell.LocalAccounts\Remove-LocalUser',
        'Microsoft.PowerShell.LocalAccounts\Enable-LocalUser',
        'Microsoft.PowerShell.LocalAccounts\Disable-LocalUser',
        'Microsoft.PowerShell.LocalAccounts\Set-LocalUser',
        'Microsoft.PowerShell.LocalAccounts\Get-LocalUser',
        'Microsoft.PowerShell.LocalAccounts\Get-LocalGroup',
        'Microsoft.PowerShell.LocalAccounts\Add-LocalGroupMember',
        'Microsoft.PowerShell.LocalAccounts\Remove-LocalGroupMember'
    )

    # VisibleProviders
    VisibleProviders = 'Variable', 'Function'
}
"@ | Out-File -FilePath $roleCapabilityPath -Encoding UTF8

    # Create session configuration file
    $sessionConfigParams = @{
        Path      = $ConfigPath
        SessionType = 'RestrictedRemoteServer'
        RunAsVirtualAccount = $true
        VirtualAccountType = 'LocalAccount'
        RoleDefinitions = @{
            'XingRanAgentUser' = @{ RoleCapabilityFiles = $roleCapabilityPath }
        }
        TranscriptDirectory = 'C:\Program Files\XingRanAgent\transcripts'
    }

    New-PSSessionConfigurationFile @sessionConfigParams -Force

    Write-Host "JEA configuration created at: $ConfigPath"
    Write-Host "Role capabilities at: $roleCapabilityPath"
}

# Create JEA configuration
Create-JEAConfiguration

# Create JEA-restricted service account
$password = ConvertTo-SecureString "XingRanAgent123!" -AsPlainText -Force
try {
    New-LocalUser -Name "XingRanAgentUser" -Password $password -Description "XingRan VDI Agent User" -ErrorAction Stop | Out-Null
    Write-Host "Created user: XingRanAgentUser"
} catch {
    if ($_.Exception.Message -like "*already exists*") {
        Write-Host "User XingRanAgentUser already exists" -ForegroundColor Yellow
    } else {
        Write-Host "WARNING: Failed to create user: $_" -ForegroundColor Yellow
    }
}

# NO LONGER ADD TO ADMINISTRATORS GROUP - JEA provides restricted elevated privileges
# User remains a standard user with JEA virtual account for admin tasks
Write-Host "User configured with JEA restricted privileges (NOT in Administrators group)"

# Register JEA session configuration
Write-Host "Registering JEA session configuration..."
try {
    $psscPath = "C:\Program Files\WindowsPowerShell\Configuration\sessionConfigs\XingRanAgent.pssc"
    if (Test-Path $psscPath) {
        Register-PSSessionConfiguration -Name "XingRanAgentJEA" -Path $psscPath -Force -NoServiceRestart
        Write-Host "JEA session 'XingRanAgentJEA' registered successfully"
    } else {
        Write-Host "WARNING: JEA configuration file not found at $psscPath" -ForegroundColor Yellow
    }
} catch {
    Write-Host "WARNING: Failed to register JEA configuration: $_" -ForegroundColor Yellow
}

# Restart WinRM service to apply JEA configuration
Write-Host "Restarting WinRM service..."
Restart-Service -Name WinRM -Force

# 注册 Windows 服务
$serviceName = "XingRanVMAgent"
$serviceDisplayName = "XingRan VM Agent"

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
Write-Host "XingRan VDI Agent installed successfully!" -ForegroundColor Green
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
